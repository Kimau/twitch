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

func (cu *Chatter) updateChatterFromTags(m *irc.Message) *Chatter {

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
				cu.Badges = make(ChatBadges)
			}
			cu.Badges[TwitchTagUserTurbo] = "1"

		case TwitchTagUserBadge:
			cu.Badges = make(ChatBadges)
			if len(tagVal) < 1 {
				continue
			}
			cu.Badges = ChatBadgesFromString(string(tagVal))

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
