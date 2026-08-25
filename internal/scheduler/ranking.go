package scheduler

import (
	"context"
	"log"
	"time"

	"aidevclub/internal/service"
)

type RankingScheduler struct {
	rankingSvc *service.RankingService
	interval   time.Duration
	stopCh     chan struct{}
}

func NewRankingScheduler(rankingSvc *service.RankingService, interval time.Duration) *RankingScheduler {
	return &RankingScheduler{
		rankingSvc: rankingSvc,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

func (s *RankingScheduler) Start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.recalculate()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *RankingScheduler) Stop() {
	close(s.stopCh)
}

func (s *RankingScheduler) recalculate() {
	ctx := context.Background()

	if err := s.rankingSvc.RecalculateArticleHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate article hot ranking: %v", err)
	}

	if err := s.rankingSvc.RecalculateSkillHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate skill hot ranking: %v", err)
	}

	if err := s.rankingSvc.RecalculateMcpServerHotRanking(ctx); err != nil {
		log.Printf("Failed to recalculate mcp server hot ranking: %v", err)
	}

	if err := s.rankingSvc.RecalculateDownloadRanking(ctx); err != nil {
		log.Printf("Failed to recalculate download ranking: %v", err)
	}

	log.Println("Ranking recalculation completed")
}
