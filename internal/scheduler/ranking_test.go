package scheduler

import (
	"testing"
	"time"
)

func TestRankingSchedulerStop(t *testing.T) {
	scheduler := NewRankingScheduler(nil, time.Hour)
	scheduler.Start()
	scheduler.Stop()
	scheduler.Stop()
}
