package twitch

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-irc/irc"
)

const (
	emoteSmallURL = "http://static-cdn.jtvnw.net/emoticons/v1/%d/1.0"
	emoteBigURL   = "http://static-cdn.jtvnw.net/emoticons/v1/%d/1.0"
)

// EmoteSet -
type EmoteSet string

// EmoteID -
type EmoteID int

// Emote - Emote match string and internal ID
type Emote struct {
	MatchString string  `json:"code,omitempty"`
	ID          EmoteID `json:"id,omitempty"`
}

// EmoteSetMap - Group of Emotes
type EmoteSetMap struct {
	SetMap map[EmoteSet][]Emote `json:"emoticon_sets,omitempty"`
}

// EmoteReplace - Characters to replace in a string
type EmoteReplace struct {
	ID    EmoteID
	Start int
	End   int
}

// EmoteReplaceListFromBack - Emote List for reverse sorting so stuff is in the order
type EmoteReplaceListFromBack []EmoteReplace

func (a EmoteReplaceListFromBack) Len() int           { return len(a) }
func (a EmoteReplaceListFromBack) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EmoteReplaceListFromBack) Less(i, j int) bool { return a[i].End > a[j].End }
func (a EmoteReplaceListFromBack) String() string {
	r := ""
	for i, v := range a {
		if i > 0 {
			r += "|"
		}

		r += fmt.Sprintf("%d,%d,%d", v.ID, v.Start, v.End)
	}

	return r
}

// EmoteURL - Returns the URL of the Emote
func EmoteURL(id EmoteID, isBig bool) string {
	if isBig {
		return fmt.Sprintf(emoteBigURL, id)
	}
	return fmt.Sprintf(emoteSmallURL, id)
}

// StringToEmoteID - String to EmoteID conversion
func StringToEmoteID(s string) (EmoteID, error) {
	i, e := strconv.Atoi(s)
	if e != nil {
		return EmoteID(-1), e
	}
	return EmoteID(i), nil
}

func emoteTagToList(val irc.TagValue) (EmoteReplaceListFromBack, error) {

	if len(val) <= 0 {
		return EmoteReplaceListFromBack{}, nil
	}

	erList := EmoteReplaceListFromBack{}
	emoteGroup := strings.Split(string(val), "/")

	for _, eg := range emoteGroup {
		egs := strings.Split(eg, ":")
		egID, err := StringToEmoteID(egs[0])
		if err != nil {
			return nil, fmt.Errorf("Unable to StringToEmoteID %s - %s", egs[0], err.Error())
		}

		egReplaceSets := strings.Split(egs[1], ",")
		for _, rs := range egReplaceSets {
			if len(rs) < 2 {
				return nil, fmt.Errorf("Unable to Split %s - %s", rs, err.Error())
			}
			rsSplit := strings.Split(rs, "-")

			rsStart := rsSplit[0]
			rsEnd := rsSplit[1]
			rsStartVal, err := strconv.Atoi(rsStart)
			if err != nil {
				return nil, fmt.Errorf("Failed to conv %s - %s", rsStart, err.Error())
			}
			rsEndVal, err := strconv.Atoi(rsEnd)
			if err != nil {
				return nil, fmt.Errorf("Failed to conv %s - %s", rsEnd, err.Error())
			}

			erList = append(erList, EmoteReplace{
				ID:    egID,
				Start: rsStartVal,
				End:   rsEndVal,
			})
		}
	}

	sort.Sort(erList)
	return erList, nil

}