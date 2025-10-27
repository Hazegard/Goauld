package ttytime

import (
	"fmt"
	"github.com/hazegard/togettyc/ttycommon"
	"maze.io/x/ttyrec"
	"time"
)

type Config struct {
	HumanReadable bool     `help:"Print human readable time" optional:"" short:"H" default:"false"`
	RecordFile    []string `help:"Record file to replay" default:"" arg:"" type:"string"`
}

func (c *Config) Run() error {
	for _, f := range c.RecordFile {
		duration, err := RunOne(f)
		if err != nil {
			return err
		}
		if c.HumanReadable {
			fmt.Printf("%16s %s\n", duration, f)
		} else {
			fmt.Printf("%8d %s\n", duration/time.Second, f)

		}
	}
	return nil
}

func RunOne(recordFile string) (time.Duration, error) {
	decoder, openedFiles, err := ttycommon.InitDecoder(recordFile)
	if err != nil {
		return time.Duration(0), fmt.Errorf("could not open record: %v", err)
	}
	for _, file := range openedFiles {
		defer func() {
			err := file.Close()
			if err != nil {
				fmt.Printf("could not close file: %v", err)
			}
		}()
	}
	frames, stop := decoder.DecodeStream()
	defer stop()
	var first *ttyrec.Frame
	var d time.Duration

	for frame := range frames {
		if first == nil {
			first = frame
		} else {
			d = frame.Time.Sub(first.Time)
		}
	}
	return d, nil
}
