package utils

import (
	"time"
)

type TimeProvider interface {
	Now() time.Time
}

type RealTimeProvider struct{}

func (RealTimeProvider) Now() time.Time {
	return time.Now()
}

type EpochTicker struct {
	Epoch        *EpochTimingConfig
	timeProvider TimeProvider

	// C is the channel on which the epoch index is sent
	C <-chan int64
}

// NewEpochTicker creates a ticker that sends the epoch index on the channel C
// at the start of the epoch
func NewEpochTicker(epoch *EpochTimingConfig) *EpochTicker {
	c := make(chan int64)
	ticker := &EpochTicker{
		Epoch:        epoch,
		timeProvider: RealTimeProvider{},
		C:            c,
	}
	ticker.start(c)
	return ticker
}

func (t *EpochTicker) start(c chan int64) {
	go func() {
		now := t.timeProvider.Now()
		currentEpoch := t.Epoch.EpochIndex(now)

		epoch := currentEpoch + 1
		epochStart := t.Epoch.StartTime(epoch)

		for {
			<-time.NewTimer(epochStart.Sub(now)).C
			c <- epoch
			epoch += 1
			epochStart = t.Epoch.StartTime(epoch)
			now = t.timeProvider.Now()
		}
	}()
}
