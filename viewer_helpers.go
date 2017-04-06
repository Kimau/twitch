package twitch

import (
	"fmt"
	"math/rand"
	"time"
)

// GetRandomFollowers - Returns Random Follow for Channel
func (ah *Client) GetRandomFollowers(numFollowers int) []*Viewer {
	mapLength := len(ah.FollowerCache)
	listOfOffset := rand.Perm(mapLength)

	if numFollowers < mapLength {
		listOfOffset = listOfOffset[:numFollowers]
	}

	c := 0
	vRes := make([]*Viewer, numFollowers, numFollowers)

	offset := 0
	for i := range ah.FollowerCache {
		for _, x := range listOfOffset {
			if x == offset {
				vRes[c] = ah.viewers[i]
				c++
				break
			}
		}

		offset++
	}

	for i := 0; i < len(vRes); i++ {
		if vRes[i] == nil {
			fmt.Println(mapLength)
			fmt.Println(listOfOffset)
			fmt.Println(vRes)
			panic("Failed to Gen Random")
		}
	}

	return vRes
}

// MostUpToDateViewer - Get Viewer based on User recent update useful for judging how stale data is
func (ah *Client) MostUpToDateViewer(numFollowers int) (*Viewer, time.Time) {
	var mostRecentViewer *Viewer
	oldTime := time.Unix(0, 0)

	for _, v := range ah.viewers {
		if v.User == nil {
			continue
		}

		updatedTime := v.User.UpdatedAt()
		if updatedTime.After(oldTime) {
			oldTime = updatedTime
			mostRecentViewer = v
		}
	}

	return mostRecentViewer, oldTime
}
