module Goauld

go 1.25.6

require (
	filippo.io/age v1.3.1
	github.com/alecthomas/kong v1.14.0
	github.com/alexbrainman/sspi v0.0.0-20250919150558-7d374ff0d59e
	github.com/aus/proxyplease v0.1.0
	github.com/aymanbagabas/go-pty v0.2.2
	github.com/caddyserver/certmagic v0.25.1
	github.com/cenkalti/backoff/v5 v5.0.3
	github.com/charmbracelet/bubbles v1.0.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/charmbracelet/ssh v0.0.0-20250826160808-ebfa259c7309
	github.com/charmbracelet/x/term v0.2.2
	github.com/coder/websocket v1.8.14
	github.com/elazarl/goproxy v1.8.2
	github.com/evertras/bubble-table v0.19.2
	github.com/fatih/color v1.18.0
	github.com/glebarez/sqlite v1.11.0
	github.com/goccy/go-yaml v1.19.2
	github.com/gokrazy/rsync v0.3.3
	github.com/google/uuid v1.6.0
	github.com/gopasspw/clipboard v0.0.4
	github.com/hazegard/socket.io-go v0.1.1
	github.com/hazegard/togettyc v0.3.6
	github.com/iamacarpet/go-winpty v1.0.4
	github.com/jellydator/ttlcache/v2 v2.11.1
	github.com/kevinburke/ssh_config v1.6.0
	github.com/keygen-sh/machineid v1.1.2
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/mattn/go-colorable v0.1.14
	github.com/mattn/go-isatty v0.0.20
	github.com/miekg/dns v1.1.72
	github.com/pkg/sftp v1.13.10
	github.com/prometheus-community/pro-bing v0.8.0
	github.com/quic-go/quic-go v0.59.0
	github.com/rs/zerolog v1.34.0
	github.com/things-go/go-socks5 v0.1.0
	github.com/urfave/negroni v1.0.0
	github.com/xtaci/kcp-go/v5 v5.6.70
	github.com/xtaci/smux v1.5.56
	golang.org/x/crypto v0.48.0
	golang.org/x/net v0.50.0
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.41.0
	golang.org/x/term v0.40.0
	golang.org/x/text v0.34.0
	golang.org/x/time v0.14.0
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
	gorm.io/gorm v1.31.1
	// go get gvisor.dev/gvisor@go pour utiliser la bonne version de gvisor
	gvisor.dev/gvisor v0.0.0-20260213200628-444f74c20034
	www.bamsoftware.com/git/champa.git v0.20250620.0
	www.bamsoftware.com/git/dnstt.git v1.20241021.0
)

//nolint:gomoddirectives
replace github.com/hazegard/socket.io-go => ./vendored/github.com/hazegard/socket.io-go

replace github.com/aus/proxyplease => ./vendored/github.com/aus/proxyplease@v0.1.0

replace github.com/hazegard/togettyc => ./vendored/github.com/hazegard/togettyc

require github.com/sirupsen/logrus v1.9.3

require (
	filippo.io/hpke v0.4.0 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/caddyserver/zerossl v0.1.5 // indirect
	github.com/charmbracelet/colorprofile v0.4.2 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/conpty v0.2.0 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/creack/pty v1.1.24 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/flynn/noise v1.0.1-0.20220214164934-d803f5c4b0f4 // indirect
	github.com/git-lfs/go-ntlm v0.0.0-20190401175752-c5056e7fa066 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/renameio/v2 v2.0.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/karagenc/yeast v0.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/reedsolomon v1.13.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/landlock-lsm/go-landlock v0.7.0 // indirect
	github.com/libdns/libdns v1.1.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mholt/acmez/v3 v3.1.4 // indirect
	github.com/mmcloughlin/md4 v0.1.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/petermattis/goid v0.0.0-20260113132338-7c7de50cc741 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/webtransport-go v0.10.0 // indirect
	github.com/rapid7/go-get-proxied v0.0.0-20250207205329-09112877ac70 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sasha-s/go-deadlock v0.3.6 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/u-root/u-root v0.15.0 // indirect
	github.com/xiegeo/coloredgoroutine v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20260212183809-81e46e3db34a // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	h12.io/socks v1.0.3 // indirect
	kernel.org/pub/linux/libs/security/libcap/psx v1.2.77 // indirect
	maze.io/x/ttyrec v1.0.0 // indirect
	modernc.org/libc v1.67.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.45.0 // indirect
)
