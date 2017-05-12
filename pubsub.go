package twitch

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"encoding/json"
	"strconv"
)

const (
	psWebSockAddr = "wss://pubsub-edge.twitch.tv"

	psPingType  = `{"type":"PING"}`
	psPongType  = `{"type":"PONG"}`
	psReconnect = `{"type":"RECONNECT"}`

	psChanBits       = "channel-bits-events-v1"      // append channel id
	psChanSubs       = "channel-subscribe-events-v1" // append channel id
	psVideoPlayback  = "video-playback"              // append channel id
	psChatModActions = "chat_moderator_actions"      // append channel id
	psUserWhispers   = "whispers"                    // append user id

	psBitsMsg = "bits_event"
)

var (
	regexPSTopic = regexp.MustCompile("([0-9a-zA-Z]+).([0-9a-zA-Z]+)")
)

//////////////////////////////////////////////////////////

// PubSubBase - PS Message Base
type PubSubBase struct {
	Type  string `json:"type"`
	Nonce string `json:"nonce,omitempty"`
	Error string `json:"error,omitempty"`
	Data  struct {
		Topic   PubSubTopic `json:"topic"`
		DataStr string      `json:"message"`
	} `json:"data,omitempty"`
}

type psWrapper struct {
	Type    string `json:"type"`
	DataStr string `json:"data"`
}

type psBitsMsgData struct {
	UserName      IrcNick `json:"user_name"`       // "user_name": "dallasnchains",
	ChannelName   IrcNick `json:"channel_name"`    // "channel_name": "dallas",
	UserID        ID      `json:"user_id"`         // "user_id": "129454141",
	ChannelID     ID      `json:"channel_id"`      // "channel_id": "44322889",
	Time          string  `json:"time"`            // "time": "2017-02-09T13:23:58.168Z",
	ChatMessage   string  `json:"chat_message"`    // "chat_message": "cheer10000 New badge hype!",
	BitsUsed      int     `json:"bits_used"`       // "bits_used": 10000,
	TotalBitsUsed int     `json:"total_bits_used"` // "total_bits_used": 25000,
	Context       string  `json:"context"`         // "context": "cheer",
	BadgeEntitled struct {
		New      int `json:"new_version"`      // "new_version": 25000,
		Previous int `json:"previous_version"` // "previous_version": 10000
	} `json:"badge_entitlement"`
}

type psSubMsgData struct {
	UserName    IrcNick `json:"user_name"`    // "user_name": "dallas",
	DisplayName string  `json:"display_name"` // "display_name": "dallas",
	ChannelName IrcNick `json:"channel_name"` // "channel_name": "twitch",
	UserID      ID      `json:"user_id"`      // "user_id": "44322889",
	ChannelID   ID      `json:"channel_id"`   // "channel_id": "12826",
	Time        string  `json:"time"`         // "time": "2017-02-09T13:23:58.168Z",

	SubPlan     string `json:"sub_plan"`      // "sub_plan": "Prime"/"1000"/"2000"/"3000",
	SubPlanName string `json:"sub_plan_name"` // "sub_plan_name": "Mr_Woodchuck - Channel Subscription (mr_woodchuck)",
	Months      int    `json:"months"`        // "months": 9,
	Context     string `json:"context"`       // "context": "sub"/"resub",

	SubMessage struct {
		Message string                   `json:"message"` // "message": "A Twitch baby is born! KappaHD"
		Emotes  EmoteReplaceListFromBack `json:"emotes"`  // Emote List
	} `json:"sub_message"`
}

type psWhispMsgData struct {
	Type string `json:"type"` // "type":"whisper_received",
	Data struct {
		DataID int `json:"id"`
	} `json:"data,omitempty"`

	ThreadID string `json:"thread_id"` //        "thread_id":"129454141_44322889",
	Body     string `json:"body"`      //        "body":"hello",
	SentTS   int    `json:"sent_ts"`   //        "sent_ts":1479160009,
	FromID   int    `json:"from_id"`   //        "from_id":39141793,
	Tags     struct {
		Login       IrcNick                  `json:"login"`        // "login":"dallas",
		DisplayName string                   `json:"display_name"` // "display_name":"dallas",
		Emotes      EmoteReplaceListFromBack `json:"emotes"`       // Emote List
		Color       string                   `json:"color"`        // "color":"#8A2BE2",
		Badges      []Badge                  `json:"badges"`
	} `json:"tags"`
	Recipient struct {
		RecpID      int     `json:"id"`           // "id":129454141,
		Nick        IrcNick `json:"username"`     // "username":"dallasnchains",
		DisplayName string  `json:"display_name"` //  "display_name":"dallasnchains",
		Color       string  `json:"color"`
		Badges      []Badge `json:"badges"`
	} `json:"recipient"`
}

//////////////////////////////////////////////////////////

// PubSubTopic - Requested Sub to Topic
type PubSubTopic struct {
	Target  ID
	Subject string
}

func (pss PubSubTopic) String() string {
	return fmt.Sprintf("%s.%s", pss.Subject, pss.Target)
}

// UnmarshalJSON - JSON Helper
func (pss *PubSubTopic) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	subStr := regexPSTopic.FindStringSubmatch(s)
	if len(subStr) != 3 {
		return fmt.Errorf("Invalid length from: %d %s", len(subStr), s)
	}
	pss.Subject = subStr[1]
	pss.Target = ID(subStr[2])
	return nil
}

// MarshalJSON - JSON Helper
func (pss PubSubTopic) MarshalJSON() ([]byte, error) {
	pStr := fmt.Sprintf("%s.%s", pss.Subject, pss.Target)
	return []byte(pStr), nil
}

// PubSubTopicList - List of Topics
type PubSubTopicList []PubSubTopic

func (psList PubSubTopicList) String() string {
	topicListStr := ""
	for _, t := range psList {
		topicListStr += t.String() + ", "
	}
	topicListStr = strings.Trim(topicListStr, ", ")
	return topicListStr
}

// PubSubConn - Subsciption to Published Topic
type PubSubConn struct {
	NewSub chan PubSubTopic
	DelSub chan PubSubTopic

	activeTopics []PubSubTopic

	pingTicker  *time.Ticker
	pongTimeOut *time.Timer

	ws            *WebsocketConn
	weakClientRef *Client
}

// CreatePubSub - Create Subsciption to Published Topics
func CreatePubSub(ah *Client, topics PubSubTopicList) (*PubSubConn, error) {

	ps := PubSubConn{
		NewSub: make(chan PubSubTopic),
		DelSub: make(chan PubSubTopic),

		activeTopics:  topics,
		weakClientRef: ah,
	}

	return &ps, nil
}

func (ps *PubSubConn) closePubSub(pubString string) {
	ps.ws.ErrorClose(pubString, fmt.Errorf(pubString))
	ps.ws = nil
}

func (ps *PubSubConn) runningLoop() {
	var err error
	wshelper := CreateWebsocketHelper(nil)

	for {
		// Connect to Twitch
		ps.ws, err = wshelper.ClientDial(psWebSockAddr)
		if err != nil {
			log.Printf("PUBSUB: %s", err.Error())
			continue
		}

		// Setup
		ps.PingPong()
		err = ps.sendListen(ps.activeTopics)
		if err != nil {
			ps.closePubSub(err.Error())
		}

		ps.pingTicker = time.NewTicker(time.Minute * 4)

		// Handle Inputs
		for ps.ws != nil {
			select {
			// Socket Activity
			case l, ok := <-ps.ws.CmdChan:
				if !ok {
					ps.closePubSub("PUBSUB: Cmd Channel Closed")
					continue
				}
				ps.handleCmdResponse([]byte(l))

				// Ping Ticker
			case _, ok := <-ps.pingTicker.C:
				if ok {
					ps.PingPong()
				}

				// Pong Timeout
			case _, ok := <-ps.pongTimeOut.C:
				if ok {
					log.Printf("PUBSUB - PONG TIMEOUT")
				}

			// New Subs
			case ns, ok := <-ps.NewSub:
				if !ok {
					ps.closePubSub("PUBSUB: Closed Running Loop - New Sub Channel Closed")
					return
				}

				for _, oldSub := range ps.activeTopics {
					if oldSub == ns {
						continue
					}
				}

				err = ps.sendListen(PubSubTopicList{ns})
				if err != nil {
					ps.closePubSub(err.Error())
				}
				ps.activeTopics = append(ps.activeTopics, ns)

			// Kill Subs
			case ks, ok := <-ps.DelSub:
				if !ok {
					ps.closePubSub("PUBSUB: Closed Running Loop - Del Sub Channel Closed")
					return
				}

				filterList := []PubSubTopic{}
				for _, oldSub := range ps.activeTopics {
					if oldSub == ks {
						err = ps.sendUnlisten(PubSubTopicList{ks})
						if err != nil {
							ps.closePubSub(err.Error())
						}

					} else {
						filterList = append(filterList, oldSub)
					}
				}
				ps.activeTopics = filterList
			}
		}

	}
}

func (ps *PubSubConn) handleMessageResponse(msg *PubSubBase) error {
	var err error

	subList := PubSubTopicList{}
	for _, c := range ps.activeTopics {
		if c == msg.Data.Topic {
			subList = append(subList, c)
		}
	}

	if len(subList) < 1 {
		return fmt.Errorf("No-one subbed to %s why are we getting it", msg.Data.Topic)
	}

	switch msg.Data.Topic.Subject {
	case psChanBits:
		bitData := psBitsMsgData{}
		err = json.Unmarshal([]byte(msg.Data.DataStr), &bitData)

	case psChanSubs:
		subData := psSubMsgData{}
		err = json.Unmarshal([]byte(msg.Data.DataStr), &subData)

	case psVideoPlayback:
		panic("TODO")

	case psChatModActions:
		panic("TODO")

	case psUserWhispers:
		wrapper := psWrapper{}
		err = json.Unmarshal([]byte(msg.Data.DataStr), &wrapper)
		if err != nil {
			return err
		}

		whispData := psWhispMsgData{}
		err = json.Unmarshal([]byte(wrapper.DataStr), &whispData)
		if err != nil {
			return err
		}

		// TODO :: consolidate with chat whispers to avoid double alert
		llp := MakeLogLineMsg(LogCatWhisper,
			LogLineParsedMsg{
				UserID:  ID(strconv.Itoa(whispData.FromID)),
				Nick:    whispData.Tags.Login,
				Bits:    0,
				Badge:   "",
				Content: whispData.Body,
				Emotes:  whispData.Tags.Emotes,
			})

		ps.weakClientRef.Alerts.Post(whispData.Tags.Login, AlertWhisper, llp)

	default:
		panic("UNKNOWN")
	}

	return err
}

func (ps *PubSubConn) handleCmdResponse(inputData []byte) {
	psm := PubSubBase{}
	err := json.Unmarshal(inputData, &psm)
	if err != nil {
		log.Printf("PUBSUB [ERROR]: %s", err.Error())
	}

	// log.Printf("PUBSUB DEBUG: %s", inputData)

	switch psm.Type {
	case "PONG":
		ps.pongTimeOut.Stop()

	case "RESPONSE":
		if len(psm.Error) > 3 {
			log.Printf("PUBSUB RESPONSE: %s", psm.Error)
		}

	case "MESSAGE":
		err := ps.handleMessageResponse(&psm)
		if err != nil {
			log.Printf("PUBSUB [ERROR]: %s", err.Error())
		}

	default:
		log.Printf("PUBSUB [DEBUG]: %s", inputData)
	}
}

func (ps *PubSubConn) sendListen(topicList PubSubTopicList) error {
	nonce := GenerateRandomString(16)
	authToken := ps.weakClientRef.GetAuth()

	// LISTEN
	listenJSON := fmt.Sprintf(`{ "type": "LISTEN", "nonce": "%s", 
		"data": { "topics": ["%s"], "auth_token": "%s" }}`,
		nonce, topicList, authToken)

	log.Printf("PUBSUB: LISTEN [%s]", topicList)

	return ps.ws.WriteString(listenJSON)
}

func (ps *PubSubConn) sendUnlisten(topicList PubSubTopicList) error {
	nonce := GenerateRandomString(16)
	authToken := ps.weakClientRef.GetAuth()

	// LISTEN
	unlistenJSON := fmt.Sprintf(`{ "type": "UNLISTEN", "nonce": "%s", 
		"data": { "topics": ["%s"], "auth_token": "%s" }}`,
		nonce, topicList, authToken)

	return ps.ws.WriteString(unlistenJSON)
}

// PingPong - You must ping once every 5min and if no response in 10 seconds you must reconnect
func (ps *PubSubConn) PingPong() {

	err := ps.ws.WriteString(psPingType)
	if err != nil {
		ps.closePubSub("PubSubConn: Ping failed to write")
		return
	}

	if ps.pongTimeOut == nil {
		ps.pongTimeOut = time.NewTimer(time.Second * 10)
	} else {
		ps.pongTimeOut.Reset(time.Second * 10)
	}
}
