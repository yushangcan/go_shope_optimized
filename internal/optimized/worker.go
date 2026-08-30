package optimized

import (
	"context"
	"errors"

	"go_shope/internal/mq"
	"go_shope/internal/observability"
	"go_shope/internal/redisstore"
)

type Worker struct {
	Service  *Service
	Store    *redisstore.Store
	Consumer *mq.Consumer
}

func (w *Worker) Run(ctx context.Context) error {
	return w.Consumer.Run(ctx, func(ctx context.Context, message mq.OrderEvent) error {
		event := redisstore.OrderEvent{RequestID: message.RequestID, UserID: message.UserID, ActivityID: message.ActivityID}
		if err := w.Store.Mark(ctx, event.RequestID, RequestProcessing, nil); err != nil {
			return err
		}
		if err := w.Service.ProcessEvent(ctx, event); err != nil {
			observability.WorkerEvents.WithLabelValues("failed").Inc()
			_ = w.Store.Compensate(ctx, event, err.Error())
			_ = w.Store.Mark(ctx, event.RequestID, RequestFailed, map[string]any{"reason": err.Error()})
			return errors.New("order processing failed")
		}
		observability.WorkerEvents.WithLabelValues("succeeded").Inc()
		return nil
	})
}
