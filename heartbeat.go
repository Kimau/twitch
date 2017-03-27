package twitch

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	heartBeatRate  = time.Minute
	heartBeatLimit = 100
	heartDumpEvery = time.Minute * 30
)

var ()

// HeartbeatData - Single Beat of the Heart
type HeartbeatData struct {
	Time      time.Time `json:"time"`
	IsLive    bool      `json:"live"`
	ViewCount int       `json:"view"`
	HostList  []ID      `json:"hosts"`
}

// Heartbeat - A beat which captures some data
// * Live Status
// * Viewer Count
// * host notifications
type Heartbeat struct {
	Beats []HeartbeatData

	subbedToAlerts []chan Alert
	recentAlerts   []Alert

	prevFollowCount int

	internalBeat *time.Ticker
	client       *Client
}

// StartBeat - Blocking Loop Which
func (heart *Heartbeat) StartBeat() {

	heart.subbedToAlerts = []chan Alert{}
	heart.prevFollowCount = -1

	// First Beat
	heart.beat(time.Now())

	timeSinceDump := time.Duration(0)
	heart.internalBeat = time.NewTicker(heartBeatRate)

	// Beat every X minutes
	for ts := range heart.internalBeat.C {
		heart.beat(ts)

		// Dumping to File
		timeSinceDump += heartBeatRate
		if timeSinceDump > heartDumpEvery {
			timeSinceDump = 0
			err := heart.client.DumpState()
			if err != nil {
				fmt.Printf("DUMP ERROR: %s", err)
			}

		}
	}
}

// SubToAlerts - Call to get a sub channel to the alert
func (heart *Heartbeat) SubToAlerts() chan Alert {
	newSub := make(chan Alert, 10)
	heart.subbedToAlerts = append(heart.subbedToAlerts, newSub)

	return newSub
}

// PostAlert - Post Alert to Listeners
func (heart *Heartbeat) PostAlert(newAlert Alert) {
	// Sanity Check to avoid doubling of Alerts
	for _, a := range heart.recentAlerts {
		if (a.Name == newAlert.Name) && (a.Source == newAlert.Source) {
			log.Printf("Doubling of Alerts: \n%s\n%s", a, newAlert)
			return
		}
	}

	if len(heart.recentAlerts) > 10 {
		heart.recentAlerts = append(heart.recentAlerts[:9], newAlert)
	} else {
		heart.recentAlerts = append(heart.recentAlerts, newAlert)
	}

	//
	for _, c := range heart.subbedToAlerts {
		select {
		case c <- newAlert:
		default: // Non blocking
		}
	}
}

func (heart *Heartbeat) beat(t time.Time) {

	var prevDataPoint *HeartbeatData
	if len(heart.Beats) > 0 {
		prevDataPoint = &heart.Beats[len(heart.Beats)-1]
	}

	// Stream Viewer Count
	sb, err := heart.client.Stream.GetStreamByUser(heart.client.RoomID)
	if err != nil || sb == nil || (sb.AverageFPS == 0) {
		if prevDataPoint == nil || prevDataPoint.IsLive {
			heart.Beats = append(heart.Beats, HeartbeatData{Time: t})
			prevDataPoint = &heart.Beats[len(heart.Beats)-1]
		}
		return
	}
	heart.client.RoomStream = sb

	hbd := HeartbeatData{
		Time:      t,
		IsLive:    true,
		ViewCount: sb.Viewers,
	}
	if prevDataPoint == nil {
		prevDataPoint = &hbd
	}

	// Get Channel Followers
	fList, num, err := heart.client.Channel.GetFollowers(heart.client.RoomID, 10, true)
	// Check for new followers
	adjustedTotal := num
	for _, f := range fList {
		whenFollow, err := time.Parse(time.RFC3339, f.CreatedAtString)
		if err != nil {
			log.Printf("ERROR - Parsing Timestamp: %s", f.CreatedAtString)
		}

		if whenFollow.After(prevDataPoint.Time) {
			// New Follow
			heart.PostAlert(Alert{AlertFollow, f.User.Name, 0})
			adjustedTotal--
		}
	}

	if heart.prevFollowCount > 0 && adjustedTotal < heart.prevFollowCount {
		// Lost Followers
		// TODO :: Hunt for the lost follow
		log.Printf("Lost a follow but TODO that updating of status")
	}

	heart.prevFollowCount = num

	// List of Hosts
	hostList, err := heart.client.Stream.GetHostsByUser(heart.client.RoomID)
	if err != nil {
		fmt.Printf("Host Check failed: %s\n", err.Error())
		return
	}

	newHostNames := []string{}

	for _, h := range hostList {
		srcID := IDFromInt(h.HostID)
		hbd.HostList = append(hbd.HostList, srcID)

		if prevDataPoint != nil {
			for _, h := range prevDataPoint.HostList {
				if h == srcID {
					continue
				}
			}
		} else {
			heart.PostAlert(Alert{
				Name:   AlertHost,
				Source: IrcNick(h.HostLogin),
				Extra:  0})
			newHostNames = append(newHostNames, h.HostLogin)
		}
	}

	if len(newHostNames) > 0 {
		fmt.Printf("HOST STARTED: %s\n", strings.Join(newHostNames, ", "))
	}

	// Update Data Points
	prevDataPoint = &hbd
	if len(heart.Beats) < heartBeatLimit {
		heart.Beats = append(heart.Beats, hbd)
	} else {
		heart.Beats = append(heart.Beats[1:], hbd)
	}
}
