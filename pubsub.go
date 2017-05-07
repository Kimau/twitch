package twitch

/*
import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	psWebSockAddr = "wss://PubSubConn-edge.twitch.tv"

	psPingType  = "PING"
	psPongType  = "PONG"
	psReconnect = "RECONNECT"

	psChanBits       PubSubSubject = "channel-bits-events-v1"      // append channel id
	psChanSubs       PubSubSubject = "channel-subscribe-events-v1" // append channel id
	psVideoPlayback  PubSubSubject = "video-playback"              // append channel id
	psChatModActions PubSubSubject = "chat_moderator_actions"      // append channel id
	psUserWhispers   PubSubSubject = "whispers"                    // append user id

	psBitsMsg = "bits_event"
)

//////////////////////////////////////////////////////////
// Bit PubSubConn Data

type psBitsBadgeEntitlement struct {
	New      int `json:"new_version"`      // "new_version": 25000,
	Previous int `json:"previous_version"` // "previous_version": 10000
}

type psBitsMsgData struct {
	UserName      IrcNick                `json:"user_name"`       // "user_name": "dallasnchains",
	ChannelName   IrcNick                `json:"channel_name"`    // "channel_name": "dallas",
	UserID        ID                     `json:"user_id"`         // "user_id": "129454141",
	ChannelID     ID                     `json:"channel_id"`      // "channel_id": "44322889",
	Time          string                 `json:"time"`            // "time": "2017-02-09T13:23:58.168Z",
	ChatMessage   string                 `json:"chat_message"`    // "chat_message": "cheer10000 New badge hype!",
	BitsUsed      int                    `json:"bits_used"`       // "bits_used": 10000,
	TotalBitsUsed int                    `json:"total_bits_used"` // "total_bits_used": 25000,
	Context       string                 `json:"context"`         // "context": "cheer",
	BadgeEntitled psBitsBadgeEntitlement `json:"badge_entitlement"`
}

//////////////////////////////////////////////////////////
// Subscribe PubSubConn Data
/*
struct psSubMsgData {

}

{
   "type": "MESSAGE",
   "data": {
      "topic": "channel-subscribe-events-v1.44322889",
      "message": {
         "user_name": "dallas",
         "display_name": "dallas",
         "channel_name": "twitch",
         "user_id": "44322889",
         "channel_id": "12826",
         "time": "2015-12-19T16:39:57-08:00",
         "sub_plan": "Prime"/"1000"/"2000"/"3000",
         "sub_plan_name": "Mr_Woodchuck - Channel Subscription (mr_woodchuck)",
         "months": 9,
         "context": "sub"/"resub",
         "sub_message": {
            "message": "A Twitch baby is born! KappaHD",
            "emotes": [
            {
               "start": 23,
               "end": 7,
               "id": 2867
            }]
         }
     }
   }
}

//////////////////////////////////////////////////////////
// Whisper PubSubConn Data
{
   "type":"MESSAGE",
   "data":{
      "topic":"whispers.44322889",
      "message":{
         "type":"whisper_received",
         "data":{
            "id":41
         },
         "thread_id":"129454141_44322889",
         "body":"hello",
         "sent_ts":1479160009,
         "from_id":39141793,
         "tags":{
            "login":"dallas",
            "display_name":"dallas",
            "color":"#8A2BE2",
            "emotes":[

            ],
            "badges":[
               {
                  "id":"staff",
                  "version":"1"
               }
            ]
         },
         "recipient":{
            "id":129454141,
            "username":"dallasnchains",
            "display_name":"dallasnchains",
            "color":"",
            "badges":[]
         },
         "nonce":"6GVBTfBXNj7d71BULYKjpiKapegDI1"
      },
      "data_object":{
         "id":41,
         "thread_id":"129454141_44322889",
         "body":"hello",
         "sent_ts":1479160009,
         "from_id":44322889,
         "tags":{
            "login":"dallas",
            "display_name":"dallas",
            "color":"#8A2BE2",
            "emotes":[],
            "badges":[
               {
                  "id":"staff",
                  "version":"1"
               }
            ]
         },
         "recipient":{
            "id":129454141,
            "username":"dallasnchains",
            "display_name":"dallasnchains",
            "color":"",
            "badges":[]
         },
         "nonce":"6GVBTfBXNj7d71BULYKjpiKapegDI1"
      }
   }
}
*

//////////////////////////////////////////////////////////

// PubSubSubject - Type of Subject you can Sub to
type PubSubSubject string

// PubSubTopic - Requested Sub to Topic
type PubSubTopic struct {
	Target  ID
	Subject PubSubSubject
}

func (pss PubSubTopic) String() string {
	return fmt.Sprintf("%s.%s", pss.Subject, pss.Target)
}

// PubSubConn - Subsciption to Published Topic
type PubSubConn struct {
	NewSub chan PubSubTopic
	DelSub chan PubSubTopic

	activeTopics []PubSubTopic

	pingTicket  *time.Timer
	pongTimeOut *time.Timer

	isConnected   bool
	socket        *websocket.Conn
	weakClientRef *Client
}

// CreatePubSub - Create Subsciption to Published Topics
func CreatePubSub(ah *Client, topics []PubSubTopic) (*PubSubConn, error) {

	ps := PubSubConn{
		NewSub: make(chan PubSubTopic),
		DelSub: make(chan PubSubTopic),

		activeTopics:  topics,
		weakClientRef: ah,
	}

	go ps.runningLoop()

	return &ps, nil
}

func (ps *PubSubConn) runningLoop() {

	for {
		// Connect to Twitch
		err := ps.reconnect()
		if err != nil {
			log.Printf("")
		}

		// Setup Input Scanner
		scanLineChan := make(chan string, 10)
		go func() {
			scanner := bufio.NewScanner(ps.socket)

			for ps.isConnected && scanner.Scan() {
				scanLineChan <- scanner.Text()
			}

			close(scanLineChan)
		}()

		// Handle Inputs
		for ps.isConnected {
			select {
			// Socket Activity
			case l, ok := <-scanLineChan:
				if !ok {
					ps.isConnected = false
					continue
				}

				log.Println("PUBSUB [DEBUG]: %s", l)

			// New Subs
			case ns, ok := <-ps.NewSub:
				if !ok {
					ps.isConnected = false
					log.Println("PUBSUB: Closed Running Loop - New Sub Channel Closed")
					return
				}

			// Kill Subs
			case ks, ok := <-ps.DelSub:
				if !ok {
					ps.isConnected = false
					log.Println("PUBSUB: Closed Running Loop - Del Sub Channel Closed")
					return
				}
			}
		}

	}
}

func (ps *PubSubConn) sendListen() error {
	websocket.JSON
}

func (ps *PubSubConn) sendUnlisten() error {

}

func (ps *PubSubConn) reconnect() error {
	if ps.socket != nil {
		ps.socket.Close()
	}

	ws, err := websocket.Dial(psWebSockAddr, "", fmt.Sprintf("http://%s", ps.weakClientRef.domain))
	if err != nil {
		return err
	}
	ps.socket = ws

	// Setup Ping Ticker
	ps.PingPong()

	ps.isConnected = true
	return nil
}

// PingPong - You must ping once every 5min and if no response in 10 seconds you must reconnect
func (ps *PubSubConn) PingPong() {

	n, err := fmt.Fprint(ws, psPingType)
	if err != nil {
		log.Printf("PubSubConn: Ping failed to write")
		ps.Reconnect()
		return
	}

	if ps.pongTimeOut == nil {
		ps.pongTimeOut = time.NewTimer(time.Second * 10)
	} else {
		ps.pongTimeOut.Reset(time.Second * 10)
	}
}
*/
