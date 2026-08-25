package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestRankingSchedulerStop(t *testing.T) {
	scheduler := NewRankingScheduler(nil, time.Hour)
	scheduler.Start(context.Background())
	scheduler.Stop()
	scheduler.Stop()
}

func TestRankingSchedulerStopCancelsInProgressRecalculation(t *testing.T) {
	recalculator := &cancelAwareRecalculator{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	scheduler := NewRankingScheduler(recalculator, time.Millisecond)
	scheduler.Start(context.Background())
	requireSignal(t, recalculator.started, time.Second, "ranking recalculation did not start")

	stopped := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(stopped)
	}()
	requireSignal(t, stopped, time.Second, "scheduler stop did not return after cancellation")
	requireSignal(t, recalculator.canceled, time.Second, "in-progress recalculation was not canceled")
}

func TestRankingSchedulerStopHasBoundedDeadline(t *testing.T) {
	recalculator := &stuckRecalculator{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	scheduler := NewRankingScheduler(recalculator, time.Millisecond)
	t.Cleanup(func() {
		close(recalculator.release)
		scheduler.wg.Wait()
	})
	scheduler.stopTimeout = 20 * time.Millisecond
	scheduler.Start(context.Background())
	requireSignal(t, recalculator.started, time.Second, "ranking recalculation did not start")

	stopped := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(stopped)
	}()
	requireSignal(t, stopped, 250*time.Millisecond, "scheduler stop exceeded its deadline")
}

type cancelAwareRecalculator struct {
	started  chan struct{}
	canceled chan struct{}
}

func (r *cancelAwareRecalculator) RecalculateArticleHotRanking(ctx context.Context) error {
	close(r.started)
	<-ctx.Done()
	close(r.canceled)
	return ctx.Err()
}

func (r *cancelAwareRecalculator) RecalculateSkillHotRanking(ctx context.Context) error {
	return ctx.Err()
}

func (r *cancelAwareRecalculator) RecalculateMcpServerHotRanking(ctx context.Context) error {
	return ctx.Err()
}

func (r *cancelAwareRecalculator) RecalculateDownloadRanking(ctx context.Context) error {
	return ctx.Err()
}

type stuckRecalculator struct {
	started chan struct{}
	release chan struct{}
}

func (r *stuckRecalculator) RecalculateArticleHotRanking(context.Context) error {
	close(r.started)
	<-r.release
	return nil
}

func (r *stuckRecalculator) RecalculateSkillHotRanking(context.Context) error {
	return nil
}

func (r *stuckRecalculator) RecalculateMcpServerHotRanking(context.Context) error {
	return nil
}

func (r *stuckRecalculator) RecalculateDownloadRanking(context.Context) error {
	return nil
}

func requireSignal(t *testing.T, signal <-chan struct{}, timeout time.Duration, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(timeout):
		t.Fatal(message)
	}
}
