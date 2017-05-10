package twitch

import (
	"fmt"
	"log"
	"sync"
)

// Alert - The main method to find out when stuff has happened
type Alert struct {
	Name   AlertName   `json:"name"`
	Source IrcNick     `json:"source"`
	Data   interface{} `json:"data"`
}

func (a Alert) String() string {
	return fmt.Sprintf("%s: %s - %s", a.NameString(), a.Source, a.Data)
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
	case AlertWhisper:
		return "Whisper"
	}

	return "UNKNOWN"
}

type subToAlertPump struct {
	Name string
	C    chan Alert
}

// AlertPump - Managers the Subscription to Alerts
type AlertPump struct {
	subbedToAlerts []subToAlertPump
	newSubs        chan subToAlertPump
	killSubs       chan subToAlertPump
	newAlerts      chan Alert
	recentAlerts   []Alert
	recentLock     sync.Mutex

	client *Client
}

// StartAlertPump - Start the internal go routine and create the pump
func StartAlertPump(clientRef *Client) *AlertPump {

	pump := AlertPump{
		subbedToAlerts: []subToAlertPump{},
		newSubs:        make(chan subToAlertPump, 10),
		killSubs:       make(chan subToAlertPump, 10),
		newAlerts:      make(chan Alert, 10),
		recentAlerts:   []Alert{},

		client: clientRef,
	}

	go pump.run()

	return &pump
}

func (pump *AlertPump) run() {

pumpLoop:
	for {
		select {
		case deadSub, ok := <-pump.killSubs:
			if !ok {
				break pumpLoop
			}

			subs := []subToAlertPump{}
			for _, sub := range pump.subbedToAlerts {
				if sub.C == deadSub.C {
					close(deadSub.C)
				} else {
					subs = append(subs, sub)
				}
			}
			pump.subbedToAlerts = subs

		case newSub, ok := <-pump.newSubs:
			if !ok {
				break pumpLoop
			}

			pump.subbedToAlerts = append(pump.subbedToAlerts, newSub)

		case newAlert := <-pump.newAlerts:
			err := pump.postInternal(newAlert)
			if err != nil {
				log.Printf("Failed to post alert [%s]\n%s", newAlert, err)
			}
		}
	}

	// Close all alert Channels
	for _, sub := range pump.subbedToAlerts {
		close(sub.C)
	}
}

// Sub - Create a Subsciption to Alerts
func (pump *AlertPump) Sub(subName string) chan Alert {
	newSub := make(chan Alert, 10)
	pump.newSubs <- subToAlertPump{
		Name: subName,
		C:    newSub,
	}

	return newSub
}

// Unsub - Kill a Subscription Channel
func (pump *AlertPump) Unsub(deadChannel chan Alert) {
	pump.killSubs <- subToAlertPump{
		Name: "dead",
		C:    deadChannel,
	}
}

// Post - Post Alert to Listeners
func (pump *AlertPump) Post(source IrcNick, name AlertName, extraData interface{}) {
	pump.newAlerts <- Alert{
		Name:   name,
		Source: source,
		Data:   extraData,
	}
}

// CopyRecentAlerts - Make a copy of recent alerts and returns them
func (pump *AlertPump) CopyRecentAlerts() (retList []Alert) {
	pump.recentLock.Lock()
	copy(retList, pump.recentAlerts)
	pump.recentLock.Unlock()
	return retList
}

func (pump *AlertPump) postInternal(newAlert Alert) error {
	pump.recentLock.Lock()
	defer pump.recentLock.Unlock()

	// Exception for None just forward it
	if newAlert.Name != AlertNone {

		// Sanity Check to avoid doubling of Alerts
		for _, a := range pump.recentAlerts {
			if (a.Name == newAlert.Name) && (a.Source == newAlert.Source) {
				return fmt.Errorf("Doubling of Prev Alert: %s", a)
			}
		}

		// Add to Recent Alerts
		if len(pump.recentAlerts) > 10 {
			pump.recentAlerts = append(pump.recentAlerts[:9], newAlert)
		} else {
			pump.recentAlerts = append(pump.recentAlerts, newAlert)
		}
	}

	// Forward Alert
	for _, subs := range pump.subbedToAlerts {
		select {
		case subs.C <- newAlert:
		default: // Non blocking
		}
	}

	return nil
}
