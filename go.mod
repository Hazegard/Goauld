module Goauld

go 1.23.0

toolchain go1.24.1

require (
	filippo.io/age v1.2.1
	github.com/alecthomas/kong v1.10.0
	github.com/aus/proxyplease v0.1.0
	github.com/aymanbagabas/go-pty v0.2.2
	github.com/caddyserver/certmagic v0.22.2
	github.com/cenkalti/backoff/v5 v5.0.2
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/elazarl/goproxy v1.7.2
	github.com/evertras/bubble-table v0.17.1
	github.com/glebarez/sqlite v1.11.0
	github.com/gliderlabs/ssh v0.3.8
	github.com/goccy/go-yaml v1.17.1
	github.com/jellydator/ttlcache/v2 v2.11.1
	github.com/karagenc/socket.io-go v0.1.0
	github.com/maruel/natural v1.1.1
	github.com/pkg/sftp v1.13.9
	github.com/qdm12/dns/v2 v2.0.0-rc8
	github.com/rs/zerolog v1.34.0
	github.com/things-go/go-socks5 v0.0.5
	github.com/urfave/negroni v1.0.0
	github.com/xtaci/kcp-go/v5 v5.6.18
	github.com/xtaci/smux v1.5.34
	golang.org/x/crypto v0.36.0
	gorm.io/gorm v1.25.12
	nhooyr.io/websocket v1.8.11
	www.bamsoftware.com/git/champa.git v0.20220703.0
	www.bamsoftware.com/git/dnstt.git v1.20241021.0
)

require github.com/u-root/u-root v0.11.0 // indirect

require (
	github.com/alexbrainman/sspi v0.0.0-20231016080023-1a75b4708caa // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/caddyserver/zerossl v0.1.3 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/creack/pty v1.1.24 // indirect
	github.com/deckarep/golang-set/v2 v2.7.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/color v1.18.0
	github.com/fatih/structs v1.1.0 // indirect
	github.com/git-lfs/go-ntlm v0.0.0-20190401175752-c5056e7fa066 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/karagenc/yeast v0.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/libdns/libdns v0.2.3 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14
	github.com/mattn/go-isatty v0.0.20
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mholt/acmez/v3 v3.1.1 // indirect
	github.com/miekg/dns v1.1.63
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/onsi/ginkgo/v2 v2.23.0 // indirect
	github.com/petermattis/goid v0.0.0-20250303134427-723919f7f203 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/qdm12/gosettings v0.4.3 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	// Cannot upgrade quic-go as it breaks socket.io
	github.com/quic-go/quic-go v0.47.0
	github.com/quic-go/webtransport-go v0.8.0
	github.com/rapid7/go-get-proxied v0.0.0-20250207205329-09112877ac70 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sasha-s/go-deadlock v0.3.5 // indirect
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/xiegeo/coloredgoroutine v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.uber.org/mock v0.5.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sync v0.12.0
	golang.org/x/sys v0.31.0
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	h12.io/socks v1.0.3 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
	modernc.org/sqlite v1.36.2 // indirect
)
