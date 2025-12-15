package main

import (
	"Goauld/client/api"
	"Goauld/client/types"
	wireguard2 "Goauld/client/wireguard"
	"Goauld/common/cmd"
	"Goauld/common/log"
	net2 "Goauld/common/net"
	"Goauld/common/utils"
	"Goauld/common/wireguard"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/mattn/go-isatty"
)

type Wireguard struct {
	Generate Generate `cmd:"" name:"generate" yaml:"generate"  help:"Generate Wireguard configuration file."`
	Start    Start    `cmd:"" name:"start" yaml:"start" help:"Start Wireguard tunnel."`
}

func (wg *Wireguard) Run(_ *api.API, _ ClientConfig) error {
	return nil
}

type Generate struct{}

func (gen *Generate) Run(_ *api.API, _ ClientConfig) error {
	pri, pub, err := wireguard.GenerateWireGuardKeyPair()
	if err != nil {
		return err
	}
	ip := wireguard2.RandomCarrierGradeNATIP()
	conf := wireguard.WGConfig{
		PublicKey:  pub,
		PrivateKey: pri,
		IP:         ip.String(),
	}

	res, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	if isatty.IsTerminal(os.Stdout.Fd()) {
		//nolint:forbidigo
		fmt.Println("Append the following configuration to your configuration file")
		//nolint:forbidigo
		fmt.Println("```")
		//nolint:forbidigo
		fmt.Println(string(res))
		//nolint:forbidigo
		fmt.Println("```")
	} else {
		//nolint:forbidigo
		fmt.Println()
		//nolint:forbidigo
		fmt.Println(string(res))
	}

	return nil
}

type Start struct {
	Target     string             `arg:"" name:"agent" yaml:"agent" help:"Target agent for which to start the WireGuard service."`
	Port       int                `default:"${_wg_port}" yaml:"port" name:"port" help:"Port number to listen on for WireGuard connections."`
	Ranges     string             `default:"" name:"range" yaml:"ranges" optional:"" help:"IP ranges to route through the WireGuard VPN (comma-separated)."`
	AutoRanges bool               `default:"false" name:"auto-range" yaml:"auto-ranges" optional:"" help:"Set by default the ranges of the agent"`
	Loopback   bool               `name:"loopback" yaml:"loopback" help:"Use the loopback interface with the 240.0.0.0/8 address range."`
	Exec       bool               `name:"exec" yaml:"exec" default:"true" help:"Execute WireGuard commands directly with elevated privileges."`
	WGConf     wireguard.WGConfig `name:"conf" yaml:"conf" embed:"" help:"WireGuard configuration options."`
}

func (s *Start) Validate() error {
	ranges := strings.Split(s.Ranges, ",")
	var newRange []string
	var invalid []string
	for _, r := range ranges {
		if net2.IsValidCIDR(r) {
			newRange = append(newRange, r)

			continue
		}
		if net2.IsValidIP(r) {
			newRange = append(newRange, r+"/32")

			continue
		}
		invalid = append(invalid, r)
	}
	if len(invalid) > 0 {
		return fmt.Errorf("invalid ranges: %s", strings.Join(invalid, ","))
	}
	if s.Loopback {
		newRange = append(newRange, "240.0.0.0/8")
	}
	s.Ranges = strings.Join(newRange, ",")

	return nil
}

func (s *Start) Run(clientAPI *api.API, cfg ClientConfig) error {
	notFound := cmd.CheckCommands([]string{wireguard2.WGCommand})
	if len(notFound) > 0 {
		return fmt.Errorf("command not found: %s", strings.Join(notFound, ","))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agent, err := clientAPI.GetAgentByName(cfg.Wireguard.Start.Target)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.Wireguard.Start.Target).Str("Target", cfg.Wireguard.Start.Target).Msg("Failed to get agent")

		return err
	}
	if s.AutoRanges {
		s.Ranges = s.Ranges + "," + agent.IPs
	}
	conf := s.GenerateWGConf(cfg, agent)
	dir := GetCacheDir()
	wgName := truncateString(strings.ReplaceAll(cfg.Wireguard.Start.Target, "@", "_"), 15)
	p := filepath.Join(dir, wgName+".conf")
	log.Info().Str("Path", p).Msg("Wireguard configuration generated and saved:")
	if cfg.Verbose > 0 {
		//nolint:forbidigo
		fmt.Println(conf)
	}
	err = utils.WriteToFile(conf, p)
	if err != nil {
		log.Error().Err(err).Str("Path", p).Msg("Failed to save Wireguard configuration")

		return err
	}
	err = os.Chmod(p, 0o600)
	if err != nil {
		log.Warn().Err(err).Str("Path", p).Msg("Failed to change file permissions")
	}

	defer func() {
		strCmd := wireguard2.DownCmdString(p)
		if s.Exec {
			err := cmdEnd(p)
			if err != nil {
				log.Debug().Err(err).Str("Path", p).Msg("Failed to end Wireguard agent")
				log.Debug().Str("Cmd", strCmd).Msg("Please manually run the command in the directory")
			}
		} else {
			log.Info().Str("Command", strCmd).Msg("Execute the command to stop the Wireguard agent")
		}
	}()

	tun := ""
	if s.Exec {
		content, err := cmdStart(p)
		if cfg.Verbose > 0 {
			//nolint:forbidigo
			fmt.Println(string(content))
		}
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "[+] Interface for") {
				words := strings.Fields(line)
				tun = words[len(words)-1]
			}
		}

		log.Debug().Str("Tun", tun).Msg("Wireguard configuration started")

		if err != nil {
			log.Error().Err(err).Str("Path", p).Msg("Failed to start Wireguard agent")

			return err
		}
	} else {
		log.Info().Str("Command", wireguard2.UpCmdString(p)).Msg("Execute the command to start the Wireguard agent")
	}

	wgConf := wireguard.WGConfig{
		PublicKey:  cfg.Wireguard.Start.WGConf.PublicKey,
		PrivateKey: "",
		IP:         cfg.Wireguard.Start.WGConf.IP,
	}

	err = clientAPI.AddWGPeer(agent.ID, wgConf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send Wireguard configuration to the agent")

		return err
	}

	localListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Wireguard.Start.Port))
	if err != nil {
		log.Error().Err(err).Str("Addr", fmt.Sprintf("127.0.0.1:%d", cfg.Wireguard.Start.Port)).Msg("Failed to listen on local port")

		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := localListener.Accept()
				if err != nil {
					log.Warn().Err(err).Str("Addr", fmt.Sprintf("127.0.0.1:%d", cfg.Wireguard.Start.Port)).Msg("Failed to accept connection")

					continue
				}

				proxy, err := proxyCommand(
					agent.Name,
					GenerateServerPassword(cfg.GetStaticPassword(), agent.OneTimePassword),
					cfg.GetSshdHost(),
					cfg.GetSshdPort(),
				)
				if err != nil {
					log.Error().Err(err).Str("Host", cfg.GetSshdHost()).Str("Port", cfg.GetSshdPort()).Msg("Failed to connect to sshd server")
					if proxy != nil {
						proxy.Close()
					}

					return
				}

				rConn, err := proxy.DialContext(ctx, "tcp", "127.0.0.1:"+agent.GetWGPort())
				if err != nil {
					log.Warn().Err(err).Str("Remote", "127.0.0.1:"+agent.GetWGPort()).Msg("Failed to connect to remote")
					proxy.Close()

					continue
				}
				pipe(conn, rConn)
				proxy.Close()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Info().Msg("Starting TCP to UDP forwarder")
				dialer := net.Dialer{}
				tcpCon, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Wireguard.Start.Port))
				if err != nil {
					log.Warn().Err(err).Str("target", agent.Name).Msg("Failed to connect to SSH server")

					continue
				}
				udpListener, err := wireguard2.ListenUDP(tcpCon, cfg.Wireguard.Start.Port)
				if err != nil {
					log.Warn().Err(err).Str("target", agent.Name).Msg("Failed to listen on UDP")

					continue
				}
				err = udpListener.Run(ctx)
				if err != nil {
					log.Warn().Err(err).Str("target", agent.Name).Msg("Failed to listen")
				}
			}
		}
	}()

	if s.Exec {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Second)
					res, err := cmdHandshakes(tun)
					if err != nil {
						log.Debug().Err(err).Str("Target", agent.Name).Msg("Failed to get handshake tunnel")
					}
					words := strings.Fields(string(res))
					timestamp := words[len(words)-1]
					if timestamp == "0" {
						continue
					}
					log.Debug().Msg("Handshake received")
					log.Run().Str("Target", agent.Name).Str("Interface", tun).Msg("Wireguard tunnel established")

					return
				}
			}
		}()
	} else {
		log.Run().Str("Target", agent.Name).Msg("Wireguard tunnel started")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	log.Info().Msg("Wireguard tunnel running, CTRL-C to stop")
	sig := <-c
	cancel()
	log.Info().Str("signal", sig.String()).Msg("received signal")
	log.Info().Msg("Stopping...")

	return nil
}

func cmdHandshakes(file string) ([]byte, error) {
	latestHSCmd := wireguard2.LatestHandshakes(file)
	//nolint:gosec
	c := exec.Command(latestHSCmd[0], latestHSCmd[1:]...)
	c.Stdin = os.Stdin

	return c.CombinedOutput()
}

func cmdStart(file string) ([]byte, error) {
	wgCmd := wireguard2.UpCmd(file)
	//nolint:gosec
	c := exec.Command(wgCmd[0], wgCmd[1:]...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	c.Stdin = os.Stdin
	log.Info().Msgf("Running command: %s", wireguard2.UpCmdString(file))

	return c.CombinedOutput()
}

func cmdEnd(file string) error {
	wgCmd := wireguard2.DownCmd(file)
	//nolint:gosec
	c := exec.Command(wgCmd[0], wgCmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	log.Info().Msgf("Running command: %s", wireguard2.DownCmdString(file))

	return c.Run()
}

func (s *Start) GenerateWGConf(cfg ClientConfig, agent types.Agent) string {
	if agent.WireguardIP == "" {
		agent.WireguardIP = "100.64.0.1"
	}
	ranges := []string{
		agent.WireguardIP + "/32",
	}
	if cfg.Wireguard.Start.Ranges != "" {
		ranges = append(ranges, cfg.Wireguard.Start.Ranges)
	}
	conf := fmt.Sprintf(`[Interface]
PrivateKey=%s
Address=%s/24

[Peer]
Endpoint=127.0.0.1:%d
AllowedIps=%s
PublicKey=%s
Persistentkeepalive=25`, cfg.Wireguard.Start.WGConf.PrivateKey, cfg.Wireguard.Start.WGConf.IP, cfg.Wireguard.Start.Port, strings.Join(ranges, ","), agent.WireguardPublicKey)

	return conf
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}

	return s[:length]
}

func pipe(localConn, remoteConn net.Conn) {
	defer localConn.Close()
	defer remoteConn.Close()

	go io.Copy(remoteConn, localConn)
	io.Copy(localConn, remoteConn)
}
