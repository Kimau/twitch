package twitch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	badgeGlobalAddr = "https://badges.twitch.tv/v1/badges/global/display?language=en"
	badgeChanAddr   = "https://badges.twitch.tv/v1/badges/channels/%s/display?language=en"
)

// BadgeData - Url and Details
type BadgeData struct {
	ImageURL1x  string `json:"image_url_1x"` //  "https://static-cdn.jtvnw.net/badges/v1/e4d5291f-1eee-411a-9431-5370f462a327/1",
	ImageURL2x  string `json:"image_url_2x"` //  "https://static-cdn.jtvnw.net/badges/v1/e4d5291f-1eee-411a-9431-5370f462a327/2",
	ImageURL4x  string `json:"image_url_4x"` //  "https://static-cdn.jtvnw.net/badges/v1/e4d5291f-1eee-411a-9431-5370f462a327/3",
	Description string `json:"description"`  //  "cheer 800000",
	Title       string `json:"title"`        //  "cheer 800000",
	ClickAction string `json:"click_action"` //  "visit_url",
	ClickURL    string `json:"click_url"`    //  "https://blog.twitch.tv/introducing-cheering-celebrate-together-da62af41fac6"
}

// BadgeDataWrap - Badge Wrapper for safe passing around
type BadgeDataWrap struct {
	Badge   string       `json:"badge"`
	Version BadgeVersion `json:"version"`
	Data    BadgeData    `json:"data"`
}

// BadgeVersion - The numerical version of the Badge like 3 for 3 month sub
type BadgeVersion string

// BadgeVersionList - Map of Badges by Version
type BadgeVersionList map[BadgeVersion]BadgeData

type badgeSetInteralJSON struct {
	Versions map[string]BadgeData `json:"versions"`
}

// BadgeMethod - The functions for Channels
type BadgeMethod struct {
	client *Client

	m           sync.Mutex
	RoomBadge   map[string]BadgeVersionList
	GlobalBadge map[string]BadgeVersionList
}

// ChatBadges - List of Chat Badges
type ChatBadges map[string]BadgeVersion

// ChatBadgesFromString - Chat Badges from String
func ChatBadgesFromString(src string) ChatBadges {
	cb := make(ChatBadges)

	bList := strings.Split(src, ",")
	for _, v := range bList {
		subB := strings.Split(v, "/")
		cb[subB[0]] = BadgeVersion(subB[1])
	}

	return cb
}

func (cb ChatBadges) String() string {
	if len(cb) == 0 {
		return ""
	}

	reStr := ""
	for k, v := range cb {
		reStr += fmt.Sprintf("%s/%s,", k, v)
	}

	return reStr[0 : len(reStr)-1]
}

// CreateBadgeMethod - Creates the Wrapper for Badges and fetches the maps
func CreateBadgeMethod(ah *Client) *BadgeMethod {
	bm := BadgeMethod{
		client: ah,

		RoomBadge:   make(map[string]BadgeVersionList),
		GlobalBadge: make(map[string]BadgeVersionList),
	}

	// Get Global Set
	bm.GlobalBadge = bm.fetchBadgeList(badgeGlobalAddr)

	// Get Room Set
	roomURL := fmt.Sprintf(badgeChanAddr, bm.client.RoomID)
	bm.RoomBadge = bm.fetchBadgeList(roomURL)

	return &bm
}

func (bm *BadgeMethod) fetchBadgeList(url string) map[string]BadgeVersionList {
	bset := struct {
		BadgeSets map[string]badgeSetInteralJSON `json:"badge_sets"`
	}{
		BadgeSets: make(map[string]badgeSetInteralJSON),
	}

	// Make Web Request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	resp, err := bm.client.httpClient.Do(req)

	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		bodyStr, _ := ioutil.ReadAll(resp.Body)
		panic(fmt.Sprintf("api error, response code: %d \n%s\n %s \n %#v", resp.StatusCode, url, bodyStr, resp))
	}

	err = json.NewDecoder(resp.Body).Decode(&bset)
	if err != nil {
		panic(err)
	}

	resp.Body.Close()

	//

	finalSet := make(map[string]BadgeVersionList)

	for badgeID, vList := range bset.BadgeSets {
		bvl := make(BadgeVersionList)
		for versionStr, data := range vList.Versions {
			bvl[BadgeVersion(versionStr)] = data
		}

		finalSet[badgeID] = bvl
	}

	return finalSet
}

// GetBadgeSafe - Get Badge Data Wrapper
func (bm *BadgeMethod) GetBadgeSafe(badgeID string, ver BadgeVersion) (BadgeDataWrap, error) {

	bData := bm.Badge(badgeID, ver)
	if bData == nil {
		return BadgeDataWrap{badgeID, ver, BadgeData{}}, fmt.Errorf("Unable to find Badge")
	}

	return BadgeDataWrap{
		badgeID,
		ver,
		*bData,
	}, nil
}

// Badge - Get Badge Data
func (bm *BadgeMethod) Badge(badgeID string, ver BadgeVersion) *BadgeData {
	bm.m.Lock()
	defer bm.m.Unlock()

	bVal, ok := bm.RoomBadge[badgeID]
	if ok {
		bVer, ok := bVal[ver]
		if ok {
			return &bVer
		}

	}

	bVal, ok = bm.GlobalBadge[badgeID]
	if ok {
		bVer, ok := bVal[ver]
		if ok {
			return &bVer
		}
	}

	return nil
}

// BadgeHTML - Get Badge HTML
func (bm *BadgeMethod) BadgeHTML(badgeID string, ver BadgeVersion) string {
	bVer := bm.Badge(badgeID, ver)

	if bVer != nil {
		return fmt.Sprintf(`<span class="badge %s" clickDest="%s" badge="%s" version="%s" style="background-image: url(%s);">%s</span>`,
			badgeID, bVer.ClickURL, bVer.Title,
			ver, bVer.ImageURL1x, bVer.Title)
	}

	return fmt.Sprintf(`<span class="badge %s">%s:%s</span>`, badgeID, badgeID, ver)
}
