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

// Alert - The main method to find out when stuff has happened
type Alert struct {
	Name   AlertName `json:"name"`
	Source IrcNick   `json:"source"`
	Extra  int       `json:"extra"`
}

func (a Alert) String() string {
	switch a.Name {
	case AlertNone:
		return fmt.Sprintf("None: %s %d", a.Source, a.Extra)
	case AlertHost:
		return fmt.Sprintf("Host: %s %d", a.Source, a.Extra)
	case AlertSub:
		return fmt.Sprintf("Sub: %s %d", a.Source, a.Extra)
	case AlertFollow:
		return fmt.Sprintf("Follow: %s %d", a.Source, a.Extra)
	case AlertBits:
		return fmt.Sprintf("Bits: %s %d", a.Source, a.Extra)
	}

	return fmt.Sprintf("BROKEN ALERT: %s %d", a.Source, a.Extra)
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
	followers       []ChannelFollow

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
	// If there are more then 30 follows in a SECOND who cares
	fList, num, err := heart.client.Channel.GetFollowers(heart.client.RoomID, 30, true)

	// Check for new followers
	adjustedTotal := num
	for _, f := range fList {
		err := heart.client.updateFollowerCache(f)
		if err != nil {
			log.Printf("ERROR - Updating Follow Cache: %s\n%s", f.CreatedAtString, err)
		}

		fTime, ok := heart.client.FollowerCache[f.User.ID]

		if ok && fTime.After(prevDataPoint.Time) {
			// New Follow
			heart.PostAlert(Alert{AlertFollow, f.User.Name, 0})
			adjustedTotal--
		} else {
			// Avoid 99% of the work
			break
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

	heart.PostAlert(Alert{
		Name:   AlertNone,
		Source: heart.client.RoomName,
		Extra:  len(heart.Beats) - 1,
	})
}
