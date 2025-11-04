package wireguard

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	"Goauld/common/wireguard"
	"fmt"
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

func (w *Wireguard) AddPeer(peerConf wireguard.WGConfig) error {
	pub, err := peerConf.PublicKey.Hex()
	if err != nil {
		return err
	}
	/*wgConf, err := w.Dev.IpcGet()
	if err != nil {
		return err
	}*/
	peerConfig := fmt.Sprintf("public_key=%s\nallowed_ip=%s/32\n", pub, peerConf.IP)

	err = w.Dev.IpcSet(peerConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set peer config")

		return err
	}

	return nil
}

func NewWireguard() *Wireguard {
	return &Wireguard{}
}

func (w *Wireguard) Init(wgCfg wireguard.WGConfig) error {
	wgTun, tNet, err := CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(config.Get().Wireguard.IP)},
		[]netip.Addr{netip.MustParseAddr("127.0.0.1")}, // We don't use DNS in the WG agent. Yet.
		1420,
	)
	if err != nil {
		return err
	}

	//	l := log.Get().With().Str("From", "wireguard").Logger()
	dev := device.NewDevice(wgTun, conn.NewDefaultBind(), device.NewLogger(2, "wireguard"))
	privKey, err := wgCfg.PrivateKey.Hex()
	if err != nil {
		log.Error().Err(err).Msg("Error generating private key")

		return err
	}
	wgConf := fmt.Sprintf("private_key=%s\nlisten_port=0", privKey)
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
