package ttyprint

import (
	"bytes"
	"fmt"
	"github.com/hazegard/togettyc/ttycommon"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	Date             bool      `help:"Show date" optional:"" short:"d" default:"false"`
	NoColor          bool      `help:"Disable colors" optional:"" default:"false"`
	Html             bool      `help:"Display result in HTML" optional:"" short:"H" default:"false"`
	RecordFile       string    `help:"Record file to print" default:"" arg:"" type:"existingfile"`
	StartDate        LocalTime `help:"Show results after the provided date (format:\"YYYY-MM-DD hh:mm:ss\")" optional:"" short:"S" `
	EndDate          LocalTime `help:"Show results before the provided date (format:\"YYYY-MM-DD hh:mm:ss\")" optional:"" short:"E" `
	Tmux             bool      `help:"Clean the output with tmux. It should reduce the noise provoked by garbage terminal manipulation" short:"T" default:"false"`
	InternalTmuxMode bool      `help:"Tmux" default:"false" hidden:"true"`
}

func (c *Config) WriteCli() string {
	args := []string{
		"--date",
	}

	if c.RecordFile != "" {
		args = append(args, strings.ReplaceAll(c.RecordFile, " ", "\\ "))
	}
	return strings.Join(args, " ")
}

type LocalTime time.Time

func (t *LocalTime) UnmarshalText(content []byte) error {
	tt, err := time.ParseInLocation(timeFormat, string(content), time.Local)
	if err != nil {
		return err
	}
	*t = LocalTime(tt)
	return nil
}

func (t *LocalTime) Get() time.Time {
	return time.Time(*t)
}

var (
	timeFormat = "2006-01-02 15:04:05"
)

func (c *Config) Run() error {

	decoder, openedFiles, err := ttycommon.InitDecoder(c.RecordFile)
	if err != nil {
		return fmt.Errorf("could not open record: %v", err)
	}
	for _, file := range openedFiles {
		defer func() {
			err := file.Close()
			if err != nil {
				fmt.Printf("could not close file: %v", err)
			}
		}()
	}
	m := Record{
		Decoder:      decoder,
		Frames:       []frame{},
		CurrentFrame: 0,
	}

	var records []Frame
	if c.InternalTmuxMode || !c.Tmux || !IsInPath("tmux") {
		records = m.ReadAll()
	} else {
		records, err = tmux(*c)
	}

	if err != nil {
		return fmt.Errorf("error reading records: %v\n", err)
	}
	if !c.EndDate.Get().IsZero() || !c.StartDate.Get().IsZero() {
		records = FilterRecordsDate(*c, records)
	}
	if c.NoColor {
		records = StripColor(records)
	}
	if c.Html {
		err = GenerateHtml(*c, records)
		if err != nil {
			return err
		}
	} else {
		err = GenerateConsoleOutput(*c, records)
		if err != nil {
			return err
		}
	}
	return nil
}

func FormatDate(date time.Time) string {
	return date.Format(timeFormat)
}

func FilterRecordsDate(config Config, records []Frame) []Frame {
	var newFrames []Frame

	for _, frame := range records {
		if (config.EndDate.Get().IsZero() || frame.Date.Before(config.EndDate.Get())) && (config.StartDate.Get().IsZero() || frame.Date.After(config.StartDate.Get())) {
			newFrames = append(newFrames, frame)
		}
	}
	return newFrames
}

func tmux(config Config) ([]Frame, error) {
	exePath, err := os.Executable()
	if err != nil {
		return []Frame{}, err
	}
	id := randString(16)
	sessionSocket := fmt.Sprintf("/tmp/tmux_togettyc-%s.sock", id)
	defer func() {
		_ = os.Remove(sessionSocket)
	}()
	paneId := fmt.Sprintf("%s:1.0", id)
	args := []string{
		"-S",
		sessionSocket,
		"new-session",
		"-ds",
		id,
		";",
		"set-option",
		"-g",
		"history-limit",
		"999999999",
		";",
		"send-keys",
		"-t",
		paneId,
		fmt.Sprintf(" %s print --internal-tmux-mode %s; tmux wait -S %s", exePath, config.WriteCli(), id),
		//fmt.Sprintf(" %s --internal-tmux-mode %s; tmux wait -S %s", exePath, config.WriteCli(), id),
		"C-m",
		";",
		"wait",
		id,
		";",
		"capture-pane",
		"-S",
		id,
		"-t",
		paneId,
		"-eCpNJ",
		"-S",
		"-",
		"-E",
		"-",
		";",
		"kill-pane",
		"-t",
		paneId,
		//";",
		//"kill-session",
		//"-t",
		//id,
	}

	env := os.Environ()
	env = append(env, fmt.Sprintf("TMUX_TMPDIR=%s", sessionSocket))
	cmd := exec.Command("tmux", args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	// Run the command.
	if err := cmd.Run(); err != nil {
		return []Frame{}, fmt.Errorf("error running tmux command: %v\n", err)
	}

	// res := out.String()
	newRecords := []Frame{}
	date := time.Time{}
	for _, line := range bytes.Split(out.Bytes(), []byte("\n")) {

		// We try to parse the current line as a date
		dd, err := time.ParseInLocation(timeFormat, string(StripColorBytes(bytes.ReplaceAll(line, []byte("\\033"), []byte("\033")))), time.Local)

		// IF succes, we hold it for the next iteration
		if err == nil {
			date = dd
			continue
		}

		// If the date is 0, skip
		if date.Format("2006") == "0001" {
			continue
		}
		// Remove ANSI escape sequences
		line = bytes.ReplaceAll(line, []byte("\\033"), []byte("\033"))
		//line = bytes.ReplaceAll(line, []byte("\033[2J\033[3J\033[H"), []byte(""))
		//line = bytes.ReplaceAll(line, []byte("\033[?1049h"), []byte(""))
		records := Frame{
			Data: append(line, []byte("\033[0m")...),
			Date: date,
		}
		newRecords = append(newRecords, records)

	}
	return newRecords, nil
}

func IsInPath(cmd string) bool {
	res, err := exec.LookPath(cmd)
	if err != nil {
		return false
	}
	return res != ""
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	// Seed the random number generator.
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
