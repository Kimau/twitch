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
	sub      int
	userType string
	badges   map[string]int
	color    string

	lastActive time.Time
	id         ID // not cannonical data
}

func (cu *chatter) UpdateChatterFromTags(m *irc.Message) *chatter {

	cu.lastActive = time.Now()

	for tagName, tagVal := range m.Tags {
		switch tagName {

		case TwitchTagUserID:
			cu.id = ID(tagVal)

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
			if cu.nick == "" {
				cu.nick = IrcNick(tagVal)
			}
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
				cu.sub = intVal
			}
		case TwitchTagUserType:
			cu.userType = string(tagVal)
		}
	}

	return cu
}

func (cu *chatter) NameWithBadge() string {
	// Only add one of these badges
	r := " "
	for _, v := range [][]string{
		{TwitchBadgeBroadcaster, "B️"},
		{TwitchBadgeStaff, "S"},
		{TwitchBadgeGlobalMod, "G"},
		{TwitchBadgeMod, "M"},
		{TwitchBadgeTurbo, "T"},
	} {
		_, ok := cu.badges[v[0]]
		if ok {
			r = v[1]
			break
		}
	}

	s, ok := cu.badges[TwitchBadgeSub]
	if ok {
		r += fmt.Sprintf("[%2d]", s)
	}

	// to catch unknown badge types
	for n, v := range cu.badges {
		switch n {
		case TwitchBadgeStaff:
		case TwitchBadgeTurbo:
		case TwitchBadgeSub:
		case TwitchBadgeMod:
		case TwitchBadgeGlobalMod:
		case TwitchBadgeBroadcaster:
		case TwitchBadgeBits:
		default:
			r += fmt.Sprintf("(%s%d)", n, v)
		}

	}

	b, ok := cu.badges[TwitchBadgeBits]
	if ok {
		return fmt.Sprintf("%s %s B%d", r, cu.nick, b)
	}

	return fmt.Sprintf("%s %s", r, cu.nick)
}
