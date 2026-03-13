//nolint:revive
package common

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Appname application name.
const Appname = "Goauld"

// JVersion holds the version information.
type JVersion struct {
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Version string `json:"version"`
}

var (
	// Commit the commit hash.
	Commit = "none"
	// Date the date of the commit.
	Date = "2006-01-02T15:04:05Z"
	// Version  the latest tag when compiled.
	Version = "dev"
)

// JSONVersion creates a JVersion.
func JSONVersion() JVersion {
	return JVersion{
		Commit:  Commit,
		Date:    Date,
		Version: Version,
	}
}

// Compare return whether two versions are equals.
func (j JVersion) Compare(jj JVersion) int {
	return strings.Compare(j.Version, jj.Version)
}

// String returns the version encoded as string JSON.
func (j JVersion) String() string {
	return fmt.Sprintf("%s-%.8s (%s)", j.Version, j.Commit, j.Date)
}

// AppName return the sanitized App Name.
func AppName() string {
	out := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}

		return -1
	}, Appname)

	return out
}

// GetVersion return the version.
func GetVersion() string {
	return fmt.Sprintf("%s-%.8s (%s)", Version, Commit, Date)
}

// Title holds the title printed in help.
func Title(_type string) string {
	var prettyDate string
	d, err := time.Parse(`2006-01-02T15:04:05Z`, Date)
	if err != nil {
		//nolint:forbidigo
		fmt.Println(err)
		prettyDate = Date
	} else {
		prettyDate = d.Format("2006-01-02 15:04:05")
	}

	return fmt.Sprintf("%s - %s (%s-%.8s %s)", Appname, _type, Version, Commit, prettyDate)
}
