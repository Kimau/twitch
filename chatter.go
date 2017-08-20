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
	Nick        IrcNick    `json:"nick"`
	DisplayName string     `json:"display_name"`
	EmoteSets   []EmoteSet `json:"emote_sets"`
	Bits        int        `json:"bits"`

	Mod      bool       `json:"mod"`
	Sub      int        `json:"sub"`
	UserType string     `json:"user_type"`
	Badges   ChatBadges `json:"badges"`
	Color    string     `json:"color"`

	TimeInChannel time.Duration `json:"time_in_channel"`
	LastActive    time.Time     `json:"last_active"`

	id ID // not cannonical data
}

func (ch *Chatter) updateTime() time.Duration {
	newTime := time.Now()

	timeSince := newTime.Sub(ch.LastActive)
	ch.TimeInChannel += timeSince
	ch.LastActive = newTime

	return timeSince
}

func (ch *Chatter) updateChatterFromTags(m *irc.Message) *Chatter {

	for tagName, tagVal := range m.Tags {
		switch tagName {

		// ----- Do Nothing -----
		case TwitchTagUniqueID:
		case TwitchTagRoomID:
		case TwitchTagMsgParamMonths: // Do nothing by itself
		case TwitchTagBits: // Do nothing with bits value getting from Badge
		case TwitchTagSystemMsg:
		case TwitchTagMsgEmotes:
		case TwitchTagEmoteOnly: // Msg only contains emotes
		case TwitchTagMsgTime:
		case TwitchTagMsgTimeTmi:
		case TwitchTagThreadID:
		case TwitchTagWhisperID:
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

				ch.Sub = mVal

			case TwitchUserNoticeCharity:
			// TODO :: Handle Charity Bits
			default:
				log.Printf("We didn't handle [%s:%s]", tagName, tagVal)

			}

		case TwitchTagUserID:
			ch.id = ID(tagVal)

		case TwitchTagLogin:
			ch.Nick = IrcNick(tagVal)

		case TwitchTagUserTurbo:
			if ch.Badges == nil {
				ch.Badges = make(ChatBadges)
			}
			ch.Badges[TwitchTagUserTurbo] = "1"

		case TwitchTagUserBadge:
			ch.Badges = make(ChatBadges)
			if len(tagVal) < 1 {
				continue
			}
			ch.Badges = ChatBadgesFromString(string(tagVal))

		case TwitchTagUserColor:
			ch.Color = string(tagVal)

		case TwitchTagUserDisplayName:
			ch.DisplayName = string(tagVal)
			if ch.Nick == "" {
				ch.Nick = IrcNick(tagVal)
			}
		case TwitchTagUserEmoteSet:
			emoteStrings := strings.Split(string(tagVal), ",")
			ch.EmoteSets = []EmoteSet{}
			for _, v := range emoteStrings {
				ch.EmoteSets = append(ch.EmoteSets, EmoteSet(v))
			}
		case TwitchTagUserMod:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				ch.Mod = (intVal > 0)
			}

		case TwitchTagUserSub:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				if intVal < 1 || intVal > ch.Sub {
					ch.Sub = intVal
				}

			}

		case TwitchTagUserType:
			ch.UserType = string(tagVal)

			switch ch.UserType {
			case TwitchTypeEmpty:
				ch.Mod = false
			case TwitchTypeMod:
				ch.Mod = true
			case TwitchTypeGlobalMod:
				ch.Mod = true
			case TwitchTypeAdmin:
				ch.Mod = true
			case TwitchTypeStaff:
				ch.Mod = true
			}

		default:
			fmt.Printf("Didn't deal with tag [%s:%s]\n", tagName, tagVal)

		}
	}

	return ch
}
