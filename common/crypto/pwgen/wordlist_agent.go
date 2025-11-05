//go:build client

package pwgen

import "embed"

//go:embed wordlist.txt.gz
var sources embed.FS

const wl_name = "wordlist.txt.gz"
