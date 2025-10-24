package pwgen

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"
)

// extracted from https://github.com/martinhoefling/goxkcdpwgen
// We needed to extract this as importing the function makes garble
// explodes and takes a lot ot time to compile (20min vs 20 seconds)
// as the library uses a hardcoded string list

// Generator encapsulates the password generator configuration
type Generator struct {
	wordlist   []string
	numwords   int
	delimiter  string
	capitalize bool
}

// NewGenerator returns a new password generator with default values set
func NewGenerator() *Generator {
	return &Generator{wordlist: []string{}, numwords: 4, delimiter: " ", capitalize: false}
}

// GeneratePassword creates a randomized password returned as byte slice
func (g *Generator) GeneratePassword() []byte {
	return []byte(g.GeneratePasswordString())
}

// GeneratePasswordString creates a randomized password returned as string
func (g *Generator) GeneratePasswordString() string {
	var words = make([]string, g.numwords)
	for i := 0; i < g.numwords; i++ {
		if g.capitalize {
			words[i] = strings.Title(randomWord(g.wordlist))
		} else {
			words[i] = randomWord(g.wordlist)
		}
	}
	return strings.Join(words, g.delimiter)
}

// SetNumWords sets the word count for the generator
func (g *Generator) SetNumWords(count int) {
	g.numwords = count
}

// SetDelimiter sets the delimiter string. Can also be set to an empty string.
func (g *Generator) SetDelimiter(delimiter string) {
	g.delimiter = delimiter
}

// UseCustomWordlist sets the wordlist to the wl provided one
func (g *Generator) UseCustomWordlist(wl []string) {
	g.wordlist = wl
}

// SetCapitalize turns on/off capitalization of the first character
func (g *Generator) SetCapitalize(capitalize bool) {
	g.capitalize = capitalize
}

func randomWord(list []string) string {
	return list[randomInteger(len(list))]
}

func init() {
	// seed math/rand in case we have to fall back to using it
	rand.Seed(time.Now().Unix() + int64(os.Getpid()+os.Getppid()))
}

func randomInteger(max int) int {
	i, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err == nil {
		return int(i.Int64())
	}
	fmt.Println("WARNING: No crypto/rand available. Falling back to PRNG")
	return rand.Intn(max)
}
