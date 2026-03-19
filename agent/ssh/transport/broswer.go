package transport

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	net2 "Goauld/common/net"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func NewBrowserProxy() *BrowserProxy {
	return &BrowserProxy{
		WSConnChan:       make(chan net.Conn),
		SocketIOConnChan: make(chan *websocket.Conn),
		PortOk:           make(chan struct{}),
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
		_ = pipe(ctx, wsConn, sioConn)
	}()

	go func() {
		_ = pipe(ctx, sioConn, wsConn)
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
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	bp.Port = listener.Addr().(*net.TCPAddr).Port
	bp.PortOk <- struct{}{}
	log.Info().Msgf("Browser proxy: http://127.0.0.1:%d", bp.Port)
	err = bp.server.Serve(listener)

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
      console.log(...args);
      logEl.textContent += args.join(" ") + "\n";
    };

    // Replace with your endpoints
    const WS1_URL = "ws://%s/wssh/%s";
    const WS2_URL = "ws://127.0.0.1:%d/wssh/";

    const SIO1_URL = "ws://%s/live/%s/?EIO=4&transport=websocket";
    const SIO2_URL = "ws://127.0.0.1:%d/live/";

    const ws1 = new WebSocket(WS1_URL);
    const ws2 = new WebSocket(WS2_URL);

    const sio1 = new WebSocket(SIO1_URL);
    const sio2 = new WebSocket(SIO2_URL);

    // Optional: support binary data
    ws1.binaryType = "arraybuffer";
    ws2.binaryType = "arraybuffer";
    //sio1.binaryType = "arraybuffer";
    //sio2.binaryType = "arraybuffer";

    let ws1Open = false;
    let ws2Open = false;

    let sio1Open = false;
    let sio2Open = false;

    // Buffers for messages arriving before both sockets are ready
    const buffer1 = [];
    const buffer2 = [];

    const bufferSIO1 = [];
    const bufferSIO2 = [];

function cloneData(data) {
  if (data instanceof ArrayBuffer) return data.slice(0);
  if (data instanceof Uint8Array) return data.slice(0);
  return data; // strings are fine
}

    // Flush buffered messages once both sockets are open
    function flushWSSHBuffers() {
      if (!ws1Open || !ws2Open) return;

      while (buffer1.length > 0) ws2.send(buffer1.shift());
      while (buffer2.length > 0) ws1.send(buffer2.shift());
log("WSSH flushed")
    }

    // Flush buffered messages once both sockets are open
    function flushSIOBuffers() {
      if (!sio1Open || !sio2Open) return;

      while (bufferSIO1.length > 0) sio2.send(bufferSIO1.shift());
      while (bufferSIO2.length > 0) sio1.send(bufferSIO2.shift());
log("SIO flushed")
    }


    // WS1 → WS2 (immediate forwarding)
    ws1.onmessage = (event) => {
		//log("WS1 " + event.data);
      if (ws2.readyState === WebSocket.OPEN) {
        ws2.send(event.data);
      } else {
        buffer1.push(event.data);
      }
    };

    // WS2 → WS1 (immediate forwarding)
    ws2.onmessage = (event) => {
		//log("WS2 " + event.data);
      if (ws1.readyState === WebSocket.OPEN) {
        ws1.send(event.data);
      } else {
        buffer2.push(event.data);
      }
    };

    // SIO1 → SIO2 (immediate forwarding)
    sio1.onmessage = (event) => {
		//log("SIO1 " + event.data);
const toSend = cloneData(event.data);
      if (sio2.readyState === WebSocket.OPEN) {
console.log(sio2);
        sio2.send(
toSend
);
      } else {
        bufferSIO1.push(toSend);
      }
    };

    // SIO2 → SIO1 (immediate forwarding)
    sio2.onmessage = (event) => {
const toSend = cloneData(event.data);
		//log("SIO2 " + toSend);
      if (sio1.readyState === WebSocket.OPEN) {
        sio1.send(toSend);
      } else {
        bufferSIO2.push(toSend);
      }
    };

    ws1.onopen = () => {
      ws1Open = true;
      log("WS1 connected");
      flushWSSHBuffers();
    };

    ws2.onopen = () => {
      ws2Open = true;
      log("WS2 connected");
      flushWSSHBuffers();
    };


    sio1.onopen = () => {
      log("SIO1 connected");
      sio1Open = true;
      flushSIOBuffers();
    };

    sio2.onopen = () => {
      log("SIO2 connected");
      sio2Open = true;
      flushSIOBuffers();
    };
	ws1.onerror = (e) => {
		console.log("WS1 error", e);
		closeBoth("WS1 error");
	  };

	ws2.onerror = (e) => {
	console.log("WS2 error", e);
	closeBoth("WS2 error");
	};

	sio1.onerror = (e) => {
		console.log("SIO1 error", e);
		closeBoth("SIO1 error");
	  };

	sio2.onerror = (e) => {
		console.log("SIO2 error", e);
		closeBoth("SIO2 error");
	};

	ws1.onclose = (event) => {
	  log("WS1 closed",
		  "code:", event.code,
		  "reason:", event.reason,
		  "wasClean:", event.wasClean);
	};
	
	ws2.onclose = (event) => {
	  log("WS2 closed",
		  "code:", event.code,
		  "reason:", event.reason,
		  "wasClean:", event.wasClean);
	};

	sio1.onclose = (event) => {
	  log("SIO1 closed",
		  "code:", event.code,
		  "reason:", event.reason,
		  "wasClean:", event.wasClean);
	};
	
	sio2.onclose = (event) => {
	  log("SIO2 closed",
		  "code:", event.code,
		  "reason:", event.reason,
		  "wasClean:", event.wasClean);
	};

    ws1.onerror = (e) => log("WS1 error", e);
    ws2.onerror = (e) => log("WS2 error", e);

    sio1.onerror = (e) => log("SIO1 error", e);
    sio2.onerror = (e) => log("SIO2 error", e);


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
