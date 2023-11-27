package process

import (
	"BGPAlert/analyze"
	"BGPAlert/common"
	"BGPAlert/config"
	"fmt"
	"strconv"
	"time"
)

// Constantly read messages from channel, build up windows with frequency maps, calling analysis on windows when full
func ProcessBGPMessages(msgChannel chan common.Message, config *config.Configuration) error {
	// Parse windowSize from config
	var maximumBuckets int
	// If windowSize can't be parsed to an int -> just default to 30
	if windowSize, err := strconv.ParseInt(config.WindowSize, 10, 64); err != nil {
		fmt.Println("Inputted config size is not a number, defaulting to 30")
		maximumBuckets = 30
	} else {
		maximumBuckets = int(windowSize)
	}

	// Each bucket is 60s
	parseDuration := 60 * time.Second
	maximumTimespan := time.Duration(maximumBuckets*60) * time.Second

	// Stores windows that will be used for analysis
	var windows []common.Window

	// Read messages from channel
	for msg := range msgChannel {
		// Make sure we select window with correct filter
		var currWindow *common.Window

		// Go through current windows and see if filter is already there
		for _, window := range windows {
			if window.Filter == msg.Filter {
				currWindow = &window
				break
			}
		}

		// If that filter is not already a window, make a new window for it
		if currWindow == nil {
			// Declare and assign window using :=
			window := common.Window{
				Filter:    msg.Filter,
				BucketMap: make(map[time.Time][]common.BGPMessage),
			}

			// Use append correctly to update the windows slice
			windows = append(windows, window)

			// Update currWindow to point to the newly added window
			currWindow = &windows[len(windows)-1]
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

			// If we at least maximumBuckets in bucketMap we can perform analysis
			if len(currWindow.BucketMap) >= maximumBuckets {
				fmt.Println(fmt.Sprintf("len(currWindow.BucketMap): %d maximumBuckets: %d", len(currWindow.BucketMap), maximumBuckets))

				// First want to remove timestamps out of scope so len(bucketMap) == maximumBuckets
				minimumTimestamp := messageBucket.Add(-maximumTimespan)
				for timestamp := range currWindow.BucketMap {
					if timestamp.Before(minimumTimestamp) {
						fmt.Println("Expired bucket: ", timestamp)
						delete(currWindow.BucketMap, timestamp)
					}
				}

				// Now window is ready for analysis
				analyze.AnalyzeBGPMessages(*currWindow)
			}

			// Create new bucket for new timestamp
			fmt.Println("Creating bucket: ", messageBucket)
			currWindow.BucketMap[messageBucket] = make([]common.BGPMessage, 0)
		}

		// Append the message to the corresponding bucket
		currWindow.BucketMap[messageBucket] = append(currWindow.BucketMap[messageBucket], msg.BGPMessage)
	}

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
