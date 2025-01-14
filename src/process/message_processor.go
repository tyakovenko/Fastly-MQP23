package process

import (
	"fmt"
	"mira/analyze"
	"mira/common"
	"mira/config"
	"time"
)

// Constantly read messages from channel, build up windows with frequency maps, calling analysis on windows when full
func ProcessBGPMessages(msgChannel chan common.Message, config *config.Configuration) error {

	// Each bucket is 60s
	parseDuration := 60 * time.Second
	maximumTimespan := time.Duration(config.MaxBuckets*60) * time.Second

	// Stores windows that will be used for analysis
	windows := make(map[string]*common.Window)

	// Read messages from channel
	for msg := range msgChannel {
		// Get current window from map if it exists
		currWindow, exists := windows[msg.Filter]

		// If the filter is not already a window, create a new window for it
		if !exists {
			window := common.Window{
				Filter:    msg.Filter,
				BucketMap: make(map[time.Time][]common.BGPMessage),
			}

			// Add window to the map
			windows[msg.Filter] = &window

			// Update currWindow to point to the newly added window
			currWindow = &window
		}

		// Round timestamp to size of parseDuration to place in correct bucket
		messageBucket := msg.BGPMessage.Timestamp.Truncate(parseDuration)

		// If a bucket with this timestamp doesn't exist, need to create one
		if _, ok := currWindow.BucketMap[messageBucket]; !ok {
			fmt.Println("----------------------------------------------------------------------------------")

			// If there's at least one timestamp, may have to fill in missing zeroes in map
			if len(currWindow.BucketMap) > 1 {
				// Get minimum timestamp from map
				minTimestamp := getBucketMapMin(currWindow.BucketMap)

				// Walk through map and see if any zeroes are missing
				for tempTimestamp := minTimestamp; tempTimestamp.Before(messageBucket); tempTimestamp = tempTimestamp.Add(time.Minute) {
					if _, ok := currWindow.BucketMap[tempTimestamp]; !ok {
						fmt.Println("Appended a 0 to map at ", tempTimestamp)
						currWindow.BucketMap[tempTimestamp] = make([]common.BGPMessage, 0)
					}
				}
			}

			// if there are enough buckets to fill the window, then perform analysis
			if len(currWindow.BucketMap) >= config.WindowSize {
				fmt.Printf("len(currWindow.BucketMap): %d maximumBuckets: %d", len(currWindow.BucketMap), config.WindowSize)

				// First want to remove timestamps out of scope so len(bucketMap) == maximumBuckets
				minimumTimestamp := messageBucket.Add(-maximumTimespan)
				for timestamp := range currWindow.BucketMap {
					if timestamp.Before(minimumTimestamp) {
						fmt.Println("Expired bucket: ", timestamp)
						delete(currWindow.BucketMap, timestamp)
					}
				}

				// Now window is ready for analysis
				analyze.AnalyzeBGPMessages(*currWindow, config)
			}

			// Create new bucket for new timestamp
			fmt.Println("Creating bucket: ", messageBucket)
			currWindow.BucketMap[messageBucket] = make([]common.BGPMessage, 0)
		}

		// Append the message to the corresponding bucket
		currWindow.BucketMap[messageBucket] = append(currWindow.BucketMap[messageBucket], msg.BGPMessage)
	}

	//check if the return statements are ok here?
	return nil
}

// Returns the minimum key value of a bucketMap
func getBucketMapMin(bucketMap map[time.Time][]common.BGPMessage) time.Time {
	var minTimestamp time.Time

	for timestamp := range bucketMap {
		minTimestamp = timestamp
		break
	}

	for timestamp := range bucketMap {
		if timestamp.Before(minTimestamp) {
			minTimestamp = timestamp
		}
	}

	return minTimestamp
}
