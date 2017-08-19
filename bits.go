package twitch

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

const (
	bitActionBackgroundDark  = "dark"
	bitActionBackgroundLight = "light"
)

// BitActionSizes - Bit Action Size
type BitActionSizes struct {
	Num1 string `json:"1"`
	Num2 string `json:"2"`
	Num3 string `json:"3"`
	Num4 string `json:"4"`
	One5 string `json:"1.5"`
}

// BitActionTypes - Bit Action Type
type BitActionTypes struct {
	Animated BitActionSizes `json:"animated"`
	Static   BitActionSizes `json:"static"`
}

// BitTier - Bit Tier of an Action like Cheer100
type BitTier struct {
	Color   string                    `json:"color"`
	ID      string                    `json:"id"`
	Images  map[string]BitActionTypes `json:"images"`
	MinBits int                       `json:"min_bits"`
}

// BitActions - List of Cheer Actions
type BitActions struct {
	Actions []struct {
		Backgrounds []string  `json:"backgrounds"`
		Prefix      string    `json:"prefix"`
		Scales      []string  `json:"scales"`
		States      []string  `json:"states"`
		Tiers       []BitTier `json:"tiers"`
	} `json:"actions"`
}

// CheerReplace - Cheer Replace
type CheerReplace struct {
	Bit   BitTier `json:"bit"`
	Start int     `json:"start"`
	End   int     `json:"end"`
}

type CheerReplaceList []CheerReplace

func (a CheerReplaceList) Len() int           { return len(a) }
func (a CheerReplaceList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CheerReplaceList) Less(i, j int) bool { return a[i].End > a[j].End }
func (a CheerReplaceList) String() string {
	r := ""
	for i, v := range a {
		if i > 0 {
			r += "|"
		}

		r += fmt.Sprintf("%s,%d,%d", v.Bit, v.Start, v.End)
	}

	return r
}

func getBitActions(ah *Client) (*BitActions, error) {
	ba := BitActions{}

	_, err := ah.Get(ah.AdminAuth,
		fmt.Sprintf("bits/actions?channel_id=%s", ah.RoomID), &ba)
	if err != nil {
		return nil, err
	}

	return &ba, nil
}

func (ba *BitActions) getCheerTiers(input string, bitsUsed int) []CheerReplace {
	var result CheerReplaceList
	totalAmount := 0

	inputLength := len(input)

	for _, a := range ba.Actions {
		offset := 0

		actRe := regexp.MustCompile(a.Prefix + "([0-9]+)")

		indexList := actRe.FindAllStringSubmatchIndex(input, -1)
		for i, matches := range indexList {
			am, err := strconv.Atoi(input[matches[2]:matches[3]])
			if err != nil {
				panic(err)
			}

			totalAmount += am

			// Find Tier that matches
			var tier *BitTier
			for _, t := range a.Tiers {
				if am > t.MinBits && (tier == nil || tier.MinBits < t.MinBits) {
					tier = &t
				}

				// Add result
				if tier != nil {
					result = append(result, CheerReplace{
						*tier,
						matches[0],
						matches[3],
					})
				}
			}
		}
	}

	sort.Sort(result)
	return result
}
