package clock

import (
	"context"
	"fmt"
	"time"
)

type TimeoutRepository interface {
	FindExpired(context.Context, time.Time) ([]string, error)
	RecordTimeout(context.Context, string, time.Time) ([][]byte, error)
}

func RunTimeoutWorker(ctx context.Context, interval time.Duration, repository TimeoutRepository, publish func(string, []byte)) error {
	if interval <= 0 || repository == nil {
		return fmt.Errorf("timeout worker requires a positive interval and repository")
	}
	if err := checkExpired(ctx, repository, publish); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := checkExpired(ctx, repository, publish); err != nil {
				return err
			}
		}
	}
}

func checkExpired(ctx context.Context, repository TimeoutRepository, publish func(string, []byte)) error {
	now := time.Now().UTC()
	matchIDs, err := repository.FindExpired(ctx, now)
	if err != nil {
		return err
	}
	for _, matchID := range matchIDs {
		events, err := repository.RecordTimeout(ctx, matchID, now)
		if err != nil {
			return err
		}
		if publish != nil {
			for _, event := range events {
				publish(matchID, event)
			}
		}
	}
	return nil
}
