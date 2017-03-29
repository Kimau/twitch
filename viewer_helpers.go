package twitch

import "math/rand"
import "fmt"

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
				vRes[c] = ah.Viewers[i]
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
