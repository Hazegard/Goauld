package ttyplay

import (
	"fmt"
	"github.com/hazegard/togettyc/ttycommon"
	"maze.io/x/ttyrec"
	"os"
	"time"
)

type Config struct {
	Speed      float64 `help:"Modify the speed" optional:"" short:"s" default:"1.0"`
	NoWait     bool    `help:"No wait mode" optional:"" short:"n" default:"false"`
	RecordFile string  `help:"Record file to replay" default:"" arg:"" type:"existingfile"`
}

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

	frames, stop := decoder.DecodeStream()
	defer stop()

	var previous *ttyrec.Frame
	for frame := range frames {
		if previous != nil {
			delay := frame.Time.Sub(previous.Time)
			if !c.NoWait {
				time.Sleep(time.Duration(float64(delay) / c.Speed))
			}
		}
		if _, err := os.Stdout.Write(frame.Data); err != nil {
			fmt.Printf("\nerror writing frame: %v\n", err)
			return err
		}

		previous = frame
	}

	return nil
}
