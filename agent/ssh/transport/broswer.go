package transport

import (
	"Goauld/agent/config"
	globalcontext "Goauld/agent/context"
	"Goauld/common/log"
	net2 "Goauld/common/net"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

func NewBrowserProxy(canceler *globalcontext.GlobalCanceler) *BrowserProxy {
	var once sync.Once
	return &BrowserProxy{
		WSConnChan:       make(chan net.Conn),
		SocketIOConnChan: make(chan *websocket.Conn),
		PortOk:           make(chan struct{}),
		cancel: func() {
			once.Do(func() {
				canceler.Restart("Browser proxy crashed")
			})
		},
	}
}

type BrowserProxy struct {
	WSConnChan       chan net.Conn
	SocketIOConnChan chan *websocket.Conn
	wsConn           *websocket.Conn
	socketIOConn     *websocket.Conn
	PortOk           chan struct{}
	Port             int
	server           *http.Server
	fakeConn         net.Conn
	cancel           func()
}

func (bp *BrowserProxy) Close() error {
	return bp.server.Close()
}

func pipe(ctx context.Context, src, dst *websocket.Conn) error {
	for {
		msgType, data, err := src.Read(ctx)
		if err != nil {
			return err
		}

		err = dst.Write(ctx, msgType, data)
		if err != nil {
			return err
		}
	}
}

func (bp *BrowserProxy) HandleFake(w http.ResponseWriter, r *http.Request) {
	// r = net2.HTTP10ToHTTP11FakeUpgrader(r)

	// Handle the websocket connection
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Str("Mode", "WSSH").Msg("error initiating websocket connection")

		return
	}
	ctx := context.Background()

	sioConn := <-bp.SocketIOConnChan
	go func() {
		err = pipe(ctx, wsConn, sioConn)
		log.Warn().Err(err).Str("Mode", "WSSH").Msg("error piping websocket connection")

		bp.cancel()
	}()

	go func() {
		_ = pipe(ctx, sioConn, wsConn)
		log.Warn().Err(err).Str("Mode", "WSSH").Msg("error piping websocket connection")

		bp.cancel()
	}()
}

func (bp *BrowserProxy) Serve() error {
	router := http.NewServeMux()
	router.HandleFunc("/", bp.ServeHTTP)
	router.HandleFunc("/wssh/", bp.ServeWS)
	router.HandleFunc("/live/", bp.ServeSocketIO)
	router.HandleFunc("/fake/", bp.HandleFake)
	bp.server = &http.Server{
		Handler: router,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise, poll requests may fail.
		WriteTimeout: 120 * time.Second,
	}

	// serve the HTTP server
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", config.Get().GetBrowserProxyPort()))
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	bp.Port = listener.Addr().(*net.TCPAddr).Port
	bp.PortOk <- struct{}{}
	log.Info().Msgf("Browser proxy: http://127.0.0.1:%d", bp.Port)
	err = bp.server.Serve(listener)
	config.Get().UpdateBrowserProxyPort(bp.Port)
	return err
}

func (bp *BrowserProxy) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	srvUrl := strings.TrimPrefix(strings.TrimPrefix(config.Get().ServerURL(), "http://"), "https://")
	js := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Raw WebSocket Pipe</title>
</head>
<body>
  <h2>Raw WebSocket Pipe</h2>
  <pre id="log"></pre>

  <script>

const logEl = document.getElementById("log");

const log = (...args) => {
    const timestamp = new Date().toLocaleString()
    console.log(timestamp, ...args);
    logEl.textContent += timestamp +" "+ args.join(" ") + "\n";
};

async function startBridge() {
    const ws1 = new WebSocket("ws://%s/wssh/%s");
    const ws2 = new WebSocket("ws://127.0.0.1:%d/wssh/");
    const sio1 = new WebSocket("ws://%s/live/%s/?EIO=4&transport=websocket");
    const sio2 = new WebSocket("ws://127.0.0.1:%d/live/");

    ws1.binaryType = "arraybuffer";
    ws2.binaryType = "arraybuffer";

    const bW1 = [];
    const bW2 = [];
    const bS1 = [];
    const bS2 = [];

    function cloneData(data) {
        if (data instanceof ArrayBuffer) return data.slice(0);
        if (data instanceof Uint8Array) return data.slice(0);
        return data;
    }

    // Flush buffered messages once both sockets are open
    function flushWSSHBuffers() {
        if (!(ws1.readyState === WebSocket.OPEN && ws2.readyState === WebSocket.OPEN)) return;

        while (bW1.length > 0) ws2.send(bW1.shift());
        while (bW2.length > 0) ws1.send(bW2.shift());
        log("WSSH flushed")
    }

    // Flush buffered messages once both sockets are open
    function flushSIOBuffers() {
        if (!(sio1.readyState === WebSocket.OPEN && sio2.readyState === WebSocket.OPEN)) return;
        while (bS1.length > 0) sio2.send(bS1.shift());
        while (bS2.length > 0) sio1.send(bS2.shift());
        log("SIO flushed")
    }

    // Message forwarding
    ws1.onmessage = (e) => (ws2.readyState === WebSocket.OPEN ? ws2.send(e.data) : bW1.push(e.data));
    ws2.onmessage = (e) => (ws1.readyState === WebSocket.OPEN ? ws1.send(e.data) : bW2.push(e.data));
    sio1.onmessage = (e) => {
        const d = cloneData(e.data);
        sio2.readyState === WebSocket.OPEN ? sio2.send(d) : bS1.push(d);
    };
    sio2.onmessage = (e) => {
        const d = cloneData(e.data);
        sio1.readyState === WebSocket.OPEN ? sio1.send(d) : bS2.push(d);
    };

    // Open events
    ws1.onopen =  () => { log("WSSH1 connected"); flushWSSHBuffers(); };
    ws2.onopen =  () => { log("WSSH2 connected"); flushWSSHBuffers(); };
    sio1.onopen =  () => { log("SIO1 connected"); flushSIOBuffers(); };
    sio2.onopen =  () => { log("SIO2 connected"); flushSIOBuffers(); };

    // List of all sockets for error/close handling
    const allSockets = [
        [ws1, "WS1"],
        [ws2, "WS2"],
        [sio1, "SIO1"],
        [sio2, "SIO2"]
    ];


    // Function to wait for all sockets to close
    async function closeAllSockets() {
        await Promise.all(allSockets.map(([sock]) => new Promise(resolve => {
            if (sock.readyState === WebSocket.CLOSED) return resolve();
            sock.onclose = () => resolve();
            if (sock.readyState === WebSocket.OPEN || sock.readyState === WebSocket.CONNECTING) {
                sock.close();
            }
        })));
    }

    async function handleCrash(name, reason) {
        log(name+ " crashed:", reason);
        await closeAllSockets();
        log("All sockets closed. Restarting bridge...");
        setTimeout(startBridge, 1000);
    }

    // Attach handlers
    allSockets.forEach(([sock, name]) => {
        sock.onerror = (e) => handleCrash(name, e);
        sock.onclose = (e) => handleCrash(name, e);
    });
}
startBridge()

  </script>
</body>
</html>

`, srvUrl, config.Get().ID, bp.Port, srvUrl, config.Get().ID, bp.Port)
	w.Write([]byte(js))
}

// ServeWS handle the SSH over Websockets connections.
func (bp *BrowserProxy) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msgf("Receive Websocket connection from browser")
	ctx := context.Background()

	r = net2.HTTP10ToHTTP11FakeUpgrader(r)

	// Handle the websocket connection
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Str("Mode", "WSSH").Msg("error initiating websocket connection")

		return
	}
	bp.wsConn = wsConn

	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	bp.WSConnChan <- conn
	log.Debug().Msgf("Websocket connection from browser forwarded")
}

func (bp *BrowserProxy) ServeSocketIO(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msgf("Receive Socket.IO connection from browser")

	r = net2.HTTP10ToHTTP11FakeUpgrader(r)

	// Handle the websocket connection
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Str("Mode", "WSSH").Msg("error initiating websocket connection")

		return
	}
	bp.socketIOConn = wsConn

	// conn := websocket.NetConn(ctx, wsConn, websocket.MessageText)
	bp.SocketIOConnChan <- wsConn
	log.Debug().Msgf("SOcket.IO connection from browser forwarded")
}
