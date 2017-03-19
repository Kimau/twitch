package twitch

import (
	"fmt"
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
	Time      time.Time
	IsLive    bool
	ViewCount int
	HostList  []ID
}

// Heartbeat - A beat which captures some data
// * Live Status
// * Viewer Count
// * host notifications
type Heartbeat struct {
	Beats []HeartbeatData

	internalBeat *time.Ticker
	client       *Client
}

// StartBeat - Blocking Loop Which
func (heart *Heartbeat) StartBeat() {

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

func (heart *Heartbeat) beat(t time.Time) {
	hbd := HeartbeatData{
		Time: t,
	}

	var prevDataPoint *HeartbeatData
	if len(heart.Beats) > 0 {
		prevDataPoint = &heart.Beats[len(heart.Beats)-1]
	}

	// Stream Viewer Count
	sb, err := heart.client.Stream.GetStreamByUser(heart.client.RoomID)
	if err != nil || sb == nil || (sb.AverageFPS > 0) {
		hbd.IsLive = false
	} else {
		hbd.IsLive = true
		hbd.ViewCount = sb.Viewers
		heart.client.RoomStream = sb
	}

	// List of Hosts
	hostList, err := heart.client.Stream.GetHostsByUser(heart.client.RoomID)
	if err != nil {
		fmt.Printf("Host Check failed: %s\n", err.Error())
	} else {

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
				newHostNames = append(newHostNames, h.HostLogin)
			}
		}

		if len(newHostNames) > 0 {
			fmt.Printf("HOST STARTED: %s", strings.Join(newHostNames, ", "))
		}

		prevDataPoint = &hbd
		if len(heart.Beats) < heartBeatLimit {
			heart.Beats = append(heart.Beats, hbd)
		} else {
			heart.Beats = append(heart.Beats[1:], hbd)
		}

	}
}
