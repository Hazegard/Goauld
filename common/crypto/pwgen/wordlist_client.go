//go:build !client

package pwgen

import "embed"

//go:embed wordlist_mini.txt.gz
var sources embed.FS

const wl_name = "wordlist_mini.txt.gz"
