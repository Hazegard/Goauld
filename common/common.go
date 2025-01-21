package common

import (
	"fmt"
	"time"
)

const APP_NAME = "Goa'uld"

var (
	commit  = "none"
	date    = ""
	version = "dev"
)

func Title(_type string) string {
	prettyDate := ""
	d, err := time.Parse(`2006-01-02T15:04:05Z`, date)
	if err != nil {
		fmt.Println(err)
		prettyDate = date
	} else {
		prettyDate = d.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s - %s (%s-%.8s %s)", APP_NAME, _type, version, commit, prettyDate)
}
