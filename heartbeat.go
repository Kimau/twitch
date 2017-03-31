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
	return fmt.Sprintf("%s: %s %d", a.NameString(), a.Source, a.Extra)
}

// NameString - Gives Label for Type
func (a Alert) NameString() string {
	switch a.Name {
	case AlertNone:
		return "None"
	case AlertHost:
		return "Host"
	case AlertSub:
		return "Sub:"
	case AlertFollow:
		return "Follow"
	case AlertBits:
		return "Bits"
	}

	return "UNKNOWN"
}

// Heartbeat - A beat which captures some data
// * Live Status
// * Viewer Count
// * host notifications
type Heartbeat struct {
	Beats []HeartbeatData

	subbedToAlerts []chan Alert
	AlertsRecent   []Alert

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
func (heart *Heartbeat) PostAlert(newAlert Alert) error {

	// Exception for None just forward it
	if newAlert.Name != AlertNone {

		// Sanity Check to avoid doubling of Alerts
		for _, a := range heart.AlertsRecent {
			if (a.Name == newAlert.Name) && (a.Source == newAlert.Source) {
				return fmt.Errorf("Doubling of Prev Alert: %s", a)
			}
		}

		// Add to Recent Alerts
		if len(heart.AlertsRecent) > 10 {
			heart.AlertsRecent = append(heart.AlertsRecent[:9], newAlert)
		} else {
			heart.AlertsRecent = append(heart.AlertsRecent, newAlert)
		}
	}

	// Forward Alert
	for _, c := range heart.subbedToAlerts {
		select {
		case c <- newAlert:
		default: // Non blocking
		}
	}

	return nil
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
			err := heart.PostAlert(Alert{AlertFollow, f.User.Name, 0})
			if err != nil {
				log.Printf("Double Follow Error: %s %s %s\n %s",
					prevDataPoint.Time, fTime, f.CreatedAtString, err)
			}
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
			err := heart.PostAlert(Alert{
				Name:   AlertHost,
				Source: IrcNick(h.HostLogin),
				Extra:  0})
			if err != nil {
				log.Printf("ERROR - Host Doubled: %s", err)
			}
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

	err = heart.PostAlert(Alert{
		Name:   AlertNone,
		Source: heart.client.RoomName,
		Extra:  len(heart.Beats) - 1,
	})

	if err != nil {
		log.Printf("ERROR - Heartbeat alert failed: %s", err)
	}
}
