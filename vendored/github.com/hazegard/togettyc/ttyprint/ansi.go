package ttyprint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
)

const ansi2 = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
const ansi = `\x1B\[[0-9;]*[a-zA-Z]`

var re = regexp.MustCompile(ansi)
var re2 = regexp.MustCompile(ansi2)

func StripAllAnsi(str []byte) []byte {
	return re2.ReplaceAll(str, []byte{})
}

func StripColor(records []Frame) []Frame {
	var strippedFrames []Frame
	for _, record := range records {
		strippedData := []byte(StripColorString(string(record.Data)))

		strippedFrame := Frame{
			Date: record.Date,
			Data: strippedData,
		}
		strippedFrames = append(strippedFrames, strippedFrame)
	}
	return strippedFrames
}

func StripColorBytes(str []byte) []byte {
	return re.ReplaceAll(str, []byte{})
}

func StripColorString(str string) string {
	return re.ReplaceAllString(str, "")
}

func GenerateConsoleOutput(config Config, records []Frame) error {
	var result [][]byte
	sep := ": "
	if config.InternalTmuxMode {
		sep = "\n"
	}

	for _, frame := range records {
		if config.Date {
			content := append([]byte("\033[0m"+FormatDate(frame.Date)+sep), frame.Data...)
			result = append(result, content)
		} else {
			result = append(result, frame.Data)
		}
	}
	_, err := io.WriteString(os.Stdout, string(bytes.Join(result, []byte("\n"))))
	if err != nil {
		return fmt.Errorf("writing to stdout: %v\n", err)
	}
	return nil
}
