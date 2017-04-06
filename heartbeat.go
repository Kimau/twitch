package twitch

import (
	"fmt"
	"log"
	"strings"
	"sync"
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
	beats     []HeartbeatData
	heartLock sync.RWMutex

	prevFollowCount int
	followers       []ChannelFollow

	internalBeat *time.Ticker
	client       *Client
}

// StartBeat - Blocking Loop Which
func (heart *Heartbeat) StartBeat() {
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
			err := heart.client.DumpViewers()
			if err != nil {
				fmt.Printf("DUMP ERROR: %s", err)
			}

		}
	}
}

// GetBeat - Get Sepecific Beat
func (heart *Heartbeat) GetBeat(beatNum int) HeartbeatData {
	heart.heartLock.RLock()
	defer heart.heartLock.RUnlock()
	return heart.beats[beatNum]
}

// GetAllBeats - Get All the Beats
func (heart *Heartbeat) GetAllBeats() []HeartbeatData {
	heart.heartLock.RLock()
	defer heart.heartLock.RUnlock()

	log.Printf("Beats Source %d", len(heart.beats))
	return heart.beats[0:]
}

func (heart *Heartbeat) beat(t time.Time) {
	heart.heartLock.Lock()
	defer heart.heartLock.Unlock()

	fmt.Println("-- BEAT --")

	var prevDataPoint *HeartbeatData
	if len(heart.beats) > 0 {
		prevDataPoint = &heart.beats[len(heart.beats)-1]
	}

	// Stream Viewer Count
	sb, err := heart.client.Stream.GetStreamByUser(heart.client.RoomID)
	if err != nil || sb == nil || (sb.AverageFPS == 0) {
		if prevDataPoint == nil || prevDataPoint.IsLive {
			heart.beats = append(heart.beats, HeartbeatData{Time: t})
			prevDataPoint = &heart.beats[len(heart.beats)-1]
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
	heart.client.updateFollowerCache(fList)

	// Check for new followers
	adjustedTotal := num
	for _, f := range fList {
		fTime, ok := heart.client.FollowerCache[f.User.ID]

		if ok && fTime.After(prevDataPoint.Time) {
			// New Follow
			heart.client.Alerts.Post(f.User.Name, AlertFollow, 0)
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
			heart.client.Alerts.Post(IrcNick(h.HostLogin), AlertHost, 0)

			newHostNames = append(newHostNames, h.HostLogin)
		}
	}

	if len(newHostNames) > 0 {
		fmt.Printf("HOST STARTED: %s\n", strings.Join(newHostNames, ", "))
	}

	// Update Data Points
	prevDataPoint = &hbd
	if len(heart.beats) < heartBeatLimit {
		heart.beats = append(heart.beats, hbd)
	} else {
		heart.beats = append(heart.beats[1:], hbd)
	}

	heart.client.Alerts.Post(heart.client.RoomName, AlertNone, len(heart.beats)-1)
}
