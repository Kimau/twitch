package twitch

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-irc/irc"
)

type chatter struct {
	nick        IrcNick
	displayName string
	emoteSets   map[int]int
	bits        int

	mod      bool
	sub      bool
	userType string
	badges   map[string]int
	color    string

	lastActive time.Time
}

func (cu *chatter) UpdateChatterFromTags(m *irc.Message) *chatter {

	/*
		id, ok := m.Tags[TwitchTagUserID]
		if ok {
			vwr := c.viewers.GetViewer(ID(id))

			if vwr.Chatter == nil {
				vwr.Chatter = &chatter{
					nick: chatterName,
				}
			}
			cu = vwr.Chatter
		}
	*/
	cu.lastActive = time.Now()

	for tagName, tagVal := range m.Tags {
		switch tagName {

		case TwitchTagUserTurbo:
			if cu.badges == nil {
				cu.badges = make(map[string]int)
			}
			cu.badges[TwitchTagUserTurbo] = 1

		case TwitchTagUserBadge:
			cu.badges = make(map[string]int)
			for _, badgeStr := range strings.Split(string(tagVal), ",") {
				iVal := 0
				t := strings.Split(badgeStr, "/")
				testVal, err := strconv.Atoi(t[1])
				if err != nil {
					log.Println(tagName, badgeStr, err)
				} else {
					iVal = testVal
				}
				cu.badges[t[0]] = iVal
			}

		case TwitchTagUserColor:
			cu.color = string(tagVal)
		case TwitchTagUserDisplayName:
			cu.displayName = string(tagVal)
		case TwitchTagUserEmoteSet:
			emoteStrings := strings.Split(string(tagVal), ",")
			cu.emoteSets = make(map[int]int)
			for _, v := range emoteStrings {
				vInt, err := strconv.Atoi(v)
				if err != nil {
					log.Println(tagName, tagVal, err)
				} else {
					cu.emoteSets[vInt] = 1
				}
			}
		case TwitchTagUserMod:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				cu.mod = (intVal > 0)
			}
		case TwitchTagUserSub:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				cu.sub = (intVal > 0)
			}
		case TwitchTagUserType:
			cu.userType = string(tagVal)
		}
	}

	return cu
}

func (cu *chatter) NameWithBadge() string {
	r := ""
	for n, v := range cu.badges {
		r += fmt.Sprintf("%s%d", n[0:1], v)
	}
	r += string(cu.nick)
	return r
}
