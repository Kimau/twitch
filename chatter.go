package twitch

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-irc/irc"
)

// Chatter - The IRC Chatter Data
type Chatter struct {
	Nick        IrcNick
	DisplayName string
	EmoteSets   []EmoteSet
	Bits        int

	Mod      bool
	Sub      int
	UserType string
	Badges   map[string]int
	Color    string

	TimeInChannel time.Duration
	LastActive    time.Time

	id ID // not cannonical data
}

func createChatter(nick IrcNick, m *irc.Message) Chatter {
	if strings.Contains(string(nick), ".") {
		// This is likely a system message
		if m == nil {
			log.Fatalf("createChatter shouldn't process system message: %s\n %s", nick, m)
		}

		newNick, b := m.GetTag(TwitchTagUserDisplayName)
		if b == false {
			log.Fatalf("createChatter couldn't recover from sys message\n %s", m)
		}
		nick = IrcNick(newNick)
	}

	cu := Chatter{
		Nick:        nick,
		DisplayName: string(nick),
		Bits:        0,

		Mod:      false,
		Sub:      0,
		UserType: TwitchTypeEmpty,
		Color:    "#000000",

		TimeInChannel: 0,
		LastActive:    time.Now(),
	}

	if m != nil {
		cu.updateChatterFromTags(m)
	}

	return cu
}

func (cu *Chatter) updateActive() {
	newTime := time.Now()
	timeSince := newTime.Sub(cu.LastActive)

	cu.TimeInChannel += timeSince
	cu.LastActive = newTime
}

func (cu *Chatter) updateChatterFromTags(m *irc.Message) *Chatter {
	cu.updateActive()

	for tagName, tagVal := range m.Tags {
		switch tagName {

		// ----- Do Nothing -----
		case TwitchTagUniqueID:
		case TwitchTagRoomID:
		case TwitchTagMsgParamMonths: // Do nothing by itself
		case TwitchTagBits: // Do nothing with bits value getting from Badge
		case TwitchTagSystemMsg:
		case TwitchTagMsgEmotes:
		case TwitchTagMsgTime:
		case TwitchTagMsgTimeTmi:
		// ----- End of Do Nothing -----

		case TwitchTagMsgID:
			switch tagVal {
			case TwitchUserNoticeReSub:
				months, ok := m.Tags[TwitchTagMsgParamMonths]
				if !ok {
					log.Println("Error processing Resub: Missing Months Tag")
					continue
				}
				mVal, err := strconv.Atoi(string(months))
				if err != nil {
					log.Println("Error processing Resub:", err)
					continue
				}

				cu.Sub = mVal

			case TwitchUserNoticeCharity:
			// TODO :: Handle Charity Bits
			default:
				log.Printf("We didn't handle [%s:%s]", tagName, tagVal)

			}

		case TwitchTagUserID:
			cu.id = ID(tagVal)

		case TwitchTagLogin:
			cu.Nick = IrcNick(tagVal)

		case TwitchTagUserTurbo:
			if cu.Badges == nil {
				cu.Badges = make(map[string]int)
			}
			cu.Badges[TwitchTagUserTurbo] = 1

		case TwitchTagUserBadge:
			cu.Badges = make(map[string]int)
			if len(tagVal) < 1 {
				continue
			}

			for _, badgeStr := range strings.Split(string(tagVal), ",") {
				iVal := 0
				// fmt.Printf("BADGE [%s]", badgeStr)
				t := strings.Split(badgeStr, "/")
				testVal, err := strconv.Atoi(t[1])
				if err != nil {
					log.Println(tagName, badgeStr, err)
				} else {
					iVal = testVal
				}
				if t[0] == TwitchBadgeBits {
					cu.Bits = iVal
				}
				cu.Badges[t[0]] = iVal
			}

		case TwitchTagUserColor:
			cu.Color = string(tagVal)

		case TwitchTagUserDisplayName:
			cu.DisplayName = string(tagVal)
			if cu.Nick == "" {
				cu.Nick = IrcNick(tagVal)
			}
		case TwitchTagUserEmoteSet:
			emoteStrings := strings.Split(string(tagVal), ",")
			cu.EmoteSets = []EmoteSet{}
			for _, v := range emoteStrings {
				cu.EmoteSets = append(cu.EmoteSets, EmoteSet(v))
			}
		case TwitchTagUserMod:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				cu.Mod = (intVal > 0)
			}

		case TwitchTagUserSub:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				if intVal < 1 || intVal > cu.Sub {
					cu.Sub = intVal
				}

			}

		case TwitchTagUserType:
			cu.UserType = string(tagVal)

			switch cu.UserType {
			case TwitchTypeEmpty:
				cu.Mod = false
			case TwitchTypeMod:
				cu.Mod = true
			case TwitchTypeGlobalMod:
				cu.Mod = true
			case TwitchTypeAdmin:
				cu.Mod = true
			case TwitchTypeStaff:
				cu.Mod = true
			}

		default:
			fmt.Printf("Didn't deal with tag [%s:%s]\n", tagName, tagVal)

		}
	}

	return cu
}

// SingleBadge - Outputs Badge to Display based on priority
// https://badges.twitch.tv/v1/badges/global/display?language=en
// https://badges.twitch.tv/v1/badges/channels/x/display?language=en
func (cu *Chatter) SingleBadge() string {
	// Only add one of these badges
	r := "."
	for _, v := range [][]string{
		{TwitchBadgeBroadcaster, "C"},
		{TwitchBadgeStaff, "X"},
		{TwitchBadgeGlobalMod, "G"},
		{TwitchBadgeMod, "M"},
		{TwitchBadgeSub, ""},
		{TwitchBadgePrime, "P"},
		{TwitchBadgeTurbo, "T"},
	} {
		_, ok := cu.Badges[v[0]]
		if ok {
			r = v[1]
			break
		}
	}

	// Special Badge Logic & to catch unknown badge types
	for n, v := range cu.Badges {
		switch n {
		case TwitchBadgeStaff:
		case TwitchBadgeTurbo:
		case TwitchBadgePrime:
		case TwitchBadgeSub:
			r += fmt.Sprintf("S%d", v)
		case TwitchBadgeMod:
		case TwitchBadgeGlobalMod:
		case TwitchBadgeBroadcaster:
		case TwitchBadgeBits:
		default:
			r += fmt.Sprintf("(%s%d)", n, v)
		}

	}

	// Badge Bits
	b, ok := cu.Badges[TwitchBadgeBits]
	if ok {
		return fmt.Sprintf("%sB%d", r, b)
	}

	return r
}
