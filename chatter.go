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
	emoteSets   []EmoteSet
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

				cu.sub = mVal

			case TwitchUserNoticeCharity:
			// TODO :: Handle Charity Bits
			default:
				log.Printf("We didn't handle [%s:%s]", tagName, tagVal)

			}

		case TwitchTagUserID:
			cu.id = ID(tagVal)

		case TwitchTagLogin:
			cu.nick = IrcNick(tagVal)

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
				if t[0] == TwitchBadgeBits {
					cu.bits = iVal
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
			cu.emoteSets = []EmoteSet{}
			for _, v := range emoteStrings {
				cu.emoteSets = append(cu.emoteSets, EmoteSet(v))
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
				if intVal < 1 || intVal > cu.sub {
					cu.sub = intVal
				}

			}

		case TwitchTagUserType:
			cu.userType = string(tagVal)

			switch cu.userType {
			case TwitchTypeEmpty:
				cu.mod = false
			case TwitchTypeMod:
				cu.mod = true
			case TwitchTypeGlobalMod:
				cu.mod = true
			case TwitchTypeAdmin:
				cu.mod = true
			case TwitchTypeStaff:
				cu.mod = true
			}

		default:
			fmt.Printf("Didn't deal with tag [%s:%s]\n", tagName, tagVal)

		}
	}

	return cu
}

func (cu *chatter) NameWithBadge() string {
	// Only add one of these badges
	r := "."
	for _, v := range [][]string{
		{TwitchBadgeBroadcaster, "B️"},
		{TwitchBadgeStaff, "S"},
		{TwitchBadgeGlobalMod, "G"},
		{TwitchBadgeMod, "M"},
		{TwitchBadgeSub, ""},
		{TwitchBadgeTurbo, "T"},
	} {
		_, ok := cu.badges[v[0]]
		if ok {
			r = v[1]
			break
		}
	}

	// Special Badge Logic & to catch unknown badge types
	for n, v := range cu.badges {
		switch n {
		case TwitchBadgeStaff:
		case TwitchBadgeTurbo:
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
	b, ok := cu.badges[TwitchBadgeBits]
	if ok {
		return fmt.Sprintf("%s %s B%d", r, cu.nick, b)
	}

	return fmt.Sprintf("%s %s", r, cu.nick)
}
