package ttyprint

import (
	"bytes"
	"github.com/hazegard/togettyc/ttycommon"
	"time"
)

type frame struct {
	Data []byte
	Date time.Time
}

type Frame struct {
	Data []byte
	Date time.Time
}

type Record struct {
	Decoder      *ttycommon.Decoder
	Frames       []frame
	CurrentFrame int
}

func (m *Record) ReadAll() []Frame {
	frames, stop := m.Decoder.DecodeStream()
	defer stop()
	var timestamps []time.Time
	var datas [][]byte

	var currentData []byte
	currentTs := time.Time{}

	for f := range frames {
		bb := bytes.Split(f.Data, []byte("\n"))

		i0 := 0
		// Handle if the previous frame does not end with a newline
		// To concatenate the peace of frames in on timestamp
		if currentData != nil {
			// If the last frame has remaining element
			// append the first line of the current frame to the last frame element
			currentData = append(currentData, bb[0]...)
			// If we have more than one line in the current frame,
			// The first line of the current frame is not the last.
			// So we can append the concatenation to the full data,
			// as well as the current timestamp (which is the timestamp of the previous frame).
			// Then we set the initialization value to 1 to skep the first frame element
			// Finally, we reset the buffers
			if len(bb) > 1 {
				datas = append(datas, currentData)
				timestamps = append(timestamps, currentTs)
				i0 = 1
				currentData = []byte{}
				currentTs = time.Time{}
			} else {
				// If there is only one element, we replace the first element of the frame with the
				// concatenated one, and we proceed with the remaining elements
				bb[0] = currentData
			}
		}

		for i := i0; i < len(bb)-1; i++ {
			// Skip empty elements
			if len(bb[i]) == 0 {
				continue
			}
			datas = append(datas, bb[i])
			timestamps = append(timestamps, time.Unix(int64(f.Time.Seconds), int64(f.Time.MicroSeconds)*1000))
		}
		currentData = bb[len(bb)-1]
		currentTs = time.Unix(int64(f.Time.Seconds), int64(f.Time.MicroSeconds)*1000)
	}

	// var res []string

	var newFrames []Frame
	for i := 0; i < len(datas); i++ {
		if timestamps[i].Format("2006") == "0001" {
			continue
		}
		data := bytes.TrimRight(datas[i], " ")

		data = bytes.ReplaceAll(data, []byte("\033[2J\033[3J\033[H"), []byte(""))
		data = bytes.ReplaceAll(data, []byte("\033[?1049h"), []byte(""))

		f := Frame{
			Data: data,
			Date: timestamps[i],
		}

		newFrames = append(newFrames, f)

		/*if cfg.Date {
			f.Data = fmt.Sprintf("%20s: %s\n",  string(data))
		} else {
			f.Data = fmt.Sprintf("%s\n", string(data))
		}*/
	}
	return newFrames
}
