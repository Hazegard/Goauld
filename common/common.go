package common

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

const App_Name = "Goa'uld"

var (
	commit  = "none"
	date    = "2006-01-02T15:04:05Z"
	version = "dev"
)

func AppName() string {
	out := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, App_Name)
	return string(out)
}

func Title(_type string) string {
	prettyDate := ""
	d, err := time.Parse(`2006-01-02T15:04:05Z`, date)
	if err != nil {
		fmt.Println(err)
		prettyDate = date
	} else {
		prettyDate = d.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s - %s (%s-%.8s %s)", App_Name, _type, version, commit, prettyDate)
}
