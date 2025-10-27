package ttyrec

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hazegard/togettyc/ttyencoder"

	"github.com/runletapp/go-console"
	"golang.org/x/term"
)

type Config struct {
	Append   bool     `help:"Append to the existing file" short:"a"`
	Compress bool     `help:"Compress the result (zstd)" short:"Z"`
	Output   string   `help:"Output file name" short:"f"`
	Shell    string   `help:"Shell to use, using current one by default" short:"S"`
	Exe      string   `help:"Command to execute" arg:"" optional:""`
	Args     []string `arg:"" help:"arguments" optional:""`
}

func (cfg *Config) Run() error {
	if cfg.Output == "" {
		cfg.Output = fmt.Sprintf("togettyc-%s.log", time.Now().Format("20060102150405"))
	}
	if cfg.Compress && !strings.HasSuffix(cfg.Output, ".zst") {
		cfg.Output += ".zst"
	}
	return run(*cfg)
}

func run(config Config) error {

	if config.Shell == "" {
		config.Shell = getShellCmd(append([]string{detectShell()}, shells...))
	}

	config.Args = formatArgs(config.Args)
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	proc, err := console.New(w, h)
	if err != nil {
		return err
	}
	err = proc.SetENV(os.Environ())
	if err != nil {
		return err
	}
	err = proc.Start(append([]string{config.Shell}, config.Args...))
	if err != nil {
		return fmt.Errorf("error starting command in pty: %v", err)
	}

	go handleResize(proc)

	defer func() { _ = proc.Close() }()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("error creating terminal state: %v", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	errChan := make(chan error)
	doneChan := make(chan struct{})
	go func() {
		_, err = io.Copy(proc, os.Stdin)
		if err != nil {
			errChan <- fmt.Errorf("error writing to pty: %v", err)
		}
	}()

	/*f, err := openEncoder(config)
	if err != nil {
		return fmt.Errorf("error opening recorder: %v", err)
	}
	defer f.Close()
	e := ttycommon.NewEncoder(f)*/

	e, err := ttyencoder.NewEncoder().
		WithAppend(config.Append).
		WithCompress(config.Compress).
		WithOutput(config.Output).
		Open()
	defer e.Close()
	go func() {
		_, err = io.Copy(io.MultiWriter(e, os.Stdout), proc)
		if err != nil {
			errChan <- fmt.Errorf("error writing to recorder: %v", err)
		}
	}()

	go func() {
		if _, err := proc.Wait(); err != nil {
			errChan <- fmt.Errorf("error running command: %v", err)
			return
		}
		doneChan <- struct{}{}

	}()

	select {
	case err := <-errChan:
		return err
	case <-doneChan:
		proc.Close()
		return nil
	}
}
