package optimized

import (
	"context"
	"errors"
	"time"

	"go_shope/internal/observability"
	"go_shope/internal/redisstore"
)

type Worker struct {
	Service *Service
	Store   *redisstore.Store
	Group   string
	Name    string
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.Store.EnsureGroup(ctx, w.Group); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		events, err := w.Store.Read(ctx, w.Group, w.Name, 10, 2*time.Second)
		if err != nil {
			if errors.Is(err, redisstore.ErrUnavailable) {
				time.Sleep(time.Second)
			}
			continue
		}
		for _, event := range events {
			if err := w.Store.Mark(ctx, event.RequestID, RequestProcessing, nil); err != nil {
				continue
			}
			if err := w.Service.ProcessEvent(ctx, event); err != nil {
				observability.WorkerEvents.WithLabelValues("failed").Inc()
				_ = w.Store.Compensate(ctx, event, err.Error())
				_ = w.Store.Ack(ctx, w.Group, event.StreamID)
				continue
			}
			observability.WorkerEvents.WithLabelValues("succeeded").Inc()
			_ = w.Store.Ack(ctx, w.Group, event.StreamID)
		}
	}
}
