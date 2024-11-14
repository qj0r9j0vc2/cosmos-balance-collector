package main

import (
	"github.com/pkg/errors"
	log "github.com/xlab/suplog"
	"time"
)

func calculateTargetTimeAndHeight(chain string, targetTime, latestBlockTime time.Time, blockInterval time.Duration, latestBlockHeight int64) (int64, error) {
	expectedBlockInterval := blockInterval

	elapsed := latestBlockTime.Sub(targetTime)

	blocksPassed := int64(elapsed / expectedBlockInterval)
	height := latestBlockHeight - blocksPassed

	blockTimestamp, err := GetBlockTime(chain, height)
	if err != nil {
		return 0, errors.Wrap(err, "Error while fetching block time")
	}

	timeDifference := blockTimestamp.Sub(targetTime)
	if timeDifference > 10*time.Second || timeDifference < -10*time.Second {
		adjustment := timeDifference / time.Duration(float64(blocksPassed))
		if timeDifference > 0 {
			expectedBlockInterval -= adjustment
		} else {
			expectedBlockInterval += adjustment
		}

		blocksPassed = int64(elapsed / expectedBlockInterval)
		height = latestBlockHeight - blocksPassed
	}

	return height, nil
}

func calculateDailyHeights(chain string, startedAt, endedAt time.Time, blockInterval time.Duration, latestBlockHeight int64) map[time.Time]int64 {
	dailyHeights := make(map[time.Time]int64)
	loopDate := endedAt.Truncate(24 * time.Hour) // Truncate to the start of the day
	expectedBlockInterval := blockInterval
	var failedCnt = 0

	for !loopDate.Before(startedAt.Truncate(24 * time.Hour)) { // Truncate startedAt to day as well
		if failedCnt > 10 {
			log.Println("Too many consecutive failures, returning nil.")
			return nil
		}

		elapsed := endedAt.Sub(loopDate)
		log.Printf("expectedBlockInterval: %s", expectedBlockInterval)
		blocksPassed := int64(elapsed / expectedBlockInterval)
		height := latestBlockHeight - blocksPassed

		log.Infoln("origin height: ", height)
		blockTimestamp, err := GetBlockTime(chain, height)
		if err != nil {
			log.Println("Error fetching block time:", err)
			failedCnt++
			continue
		}

		// Reset failed count on success
		failedCnt = 0

		timeDifference := blockTimestamp.Sub(loopDate)
		if timeDifference > 10*time.Second || timeDifference < -10*time.Second {
			log.Printf("Time discrepancy detected on %s: expected %s, but got %s. Adjusting...\n", loopDate.Format("2006-01-02"), loopDate, blockTimestamp)

			if blocksPassed != 0 {
				adjustment := timeDifference.Abs() / time.Duration(float64(blocksPassed))
				log.Printf("adjustment: %s / blocksPassed: %d / time difference: %s", adjustment, blocksPassed, timeDifference)

				if timeDifference > 0 {
					expectedBlockInterval -= adjustment
				} else {
					expectedBlockInterval += adjustment
				}

				blocksPassed = int64(elapsed / expectedBlockInterval)
				height = latestBlockHeight - blocksPassed
				log.Printf("adjusted height: %d", height)
			}
		}

		// Store the result using the truncated date
		dailyHeights[loopDate] = height

		// Move to the previous day
		loopDate = loopDate.Add(-24 * time.Hour)
	}

	return dailyHeights
}
