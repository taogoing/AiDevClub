package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

const defaultRankingStopTimeout = 10 * time.Second

type rankingRecalculator interface {
	RecalculateArticleHotRanking(context.Context) error
	RecalculateSkillHotRanking(context.Context) error
	RecalculateMcpServerHotRanking(context.Context) error
}

type RankingScheduler struct {
	rankingSvc  rankingRecalculator
	interval    time.Duration
	stopTimeout time.Duration
	stopCh      chan struct{}
	cancel      context.CancelFunc
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewRankingScheduler(rankingSvc rankingRecalculator, interval time.Duration) *RankingScheduler {
	return &RankingScheduler{
		rankingSvc:  rankingSvc,
		interval:    interval,
		stopTimeout: defaultRankingStopTimeout,
		stopCh:      make(chan struct{}),
	}
}

func (s *RankingScheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				s.recalculate(ctx)
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *RankingScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		if s.cancel != nil {
			s.cancel()
		}

		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()
		timer := time.NewTimer(s.stopTimeout)
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
			log.Printf("Timed out stopping ranking scheduler after %s", s.stopTimeout)
		}
	})
}

func (s *RankingScheduler) recalculate(ctx context.Context) {
	if err := s.rankingSvc.RecalculateArticleHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate article hot ranking: %v", err)
	}
	if ctx.Err() != nil {
		return
	}

	if err := s.rankingSvc.RecalculateSkillHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate skill hot ranking: %v", err)
	}
	if ctx.Err() != nil {
		return
	}

	if err := s.rankingSvc.RecalculateMcpServerHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate mcp server hot ranking: %v", err)
	}
}
