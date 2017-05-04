package twitch

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{}
)

// WebsocketHandler - Handler will be called when a new socket connection is Created
type WebsocketHandler func(WebsocketConn)

// WebsocketConn - Useful Wrapper for a single Connection. Note each connection will be on it's own go routine
type WebsocketConn struct {
	CmdChan chan string
	ws      *websocket.Conn
}

// WebsocketHelper - Helper Class to Wrap a Websocket
type WebsocketHelper struct {
	handler WebsocketHandler
}

// CreateWebsocketHelper - Creates a Web Socket Helper
func CreateWebsocketHelper(newHandler WebsocketHandler) *WebsocketHelper {
	return &WebsocketHelper{
		handler: newHandler,
	}
}

func (wh *WebsocketHelper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	wc := WebsocketConn{}

	// Setup Socket
	wc.ws, err = upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer wc.ws.Close()

	// Create Reader for Channel
	wc.CmdChan = make(chan string)
	go wc.pumpCommands()

	wh.handler(wc)
}

func (wc *WebsocketConn) pumpCommands() {
	defer close(wc.CmdChan)

	for {
		mType, msg, err := wc.ws.ReadMessage()
		if err != nil {
			wc.ErrorClose("Error in CMD Scanner", err)
			return
		}

		if mType == websocket.TextMessage {
			cmdStr := string(msg)
			cmdStr = strings.Trim(cmdStr, " \n\t")
			wc.CmdChan <- cmdStr
		} else if mType == websocket.BinaryMessage {
			wc.CmdChan <- string("[DATA]" + base64.StdEncoding.EncodeToString(msg))
		}
	}
}

////////////////////////////////////////////////////////

// ErrorClose - Close the Socket and log and Error
func (wc *WebsocketConn) ErrorClose(explain string, err error) {
	log.Printf("Error Websocket: %s\n%s", explain, err)
	if wc.ws != nil {
		wc.ws.Close()
	}
}

// WriteFormatted - Write a Fomatted String to Socket
func (wc *WebsocketConn) WriteFormatted(formatStr string, args ...interface{}) error {
	msg := fmt.Sprintf(formatStr, args...)
	err := wc.ws.WriteMessage(websocket.TextMessage, []byte(msg))
	return err
}

// WriteString - Write a String to Socket
func (wc *WebsocketConn) WriteString(msg string) error {
	err := wc.ws.WriteMessage(websocket.TextMessage, []byte(msg))
	return err
}

// WriteJSON - Write JSON as text to Socket
func (wc *WebsocketConn) WriteJSON(data interface{}) error {
	return wc.ws.WriteJSON(data)
}

// WritePrefixJSON - Write JSON with a prefix string
func (wc *WebsocketConn) WritePrefixJSON(prefix string, data interface{}) error {
	marshalData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msgData := append([]byte(prefix), marshalData...)
	err = wc.ws.WriteMessage(websocket.TextMessage, msgData)
	return err
}
