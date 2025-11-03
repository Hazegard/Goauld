package wireguard

import (
	"net/netip"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

type Wireguard struct {
	Tun        tun.Device
	Dev        *device.Device
	Net        *Net
	ListenPort int
}

func (w *Wireguard) AddPeer(peer netip.Addr) error {
	return nil
}

func NewWireguard() *Wireguard {
	return &Wireguard{}
}

func (w *Wireguard) Init() error {
	wgTun, tNet, err := CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("100.64.0.1")},
		[]netip.Addr{netip.MustParseAddr("127.0.0.1")}, // We don't use DNS in the WG agent. Yet.
		1420,
	)
	if err != nil {
		return err
	}

	//	l := log.Get().With().Str("From", "wireguard").Logger()
	dev := device.NewDevice(wgTun, conn.NewDefaultBind(), device.NewLogger(2, "wireguard"))
	wgConf := `private_key=a0e68d9fa41f954e6ab7411198923dc2db4e1863057c6d8c278b109325a67a45
listen_port=55555
public_key=1f714810c89a90dfa79140fb165fe106b0e3ce545605eb3788699b8ea9bd4e7e
allowed_ip=100.64.0.2/32
`

	err = dev.IpcSet(wgConf)
	if err != nil {
		return err
	}
	err = dev.Up()
	if err != nil {
		return err
	}

	wgConf, err = dev.IpcGet()
	if err != nil {
		return err
	}
	listenPort := 0
	prefix := "listen_port="
	for _, line := range strings.Split(wgConf, "\n") {
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			listenPort, err = strconv.Atoi(val)
			if err != nil {
				return err
			}
		}
	}

	w.Tun = wgTun
	w.Dev = dev
	w.Net = tNet
	w.ListenPort = listenPort

	return nil
}
