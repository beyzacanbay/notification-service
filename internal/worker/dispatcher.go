package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/beyzacanbay/notification-service/internal/queue"
)

type Dispatcher struct {
	consumer    *queue.Consumer
	processor   *Processor
	concurrency int
	logger      *slog.Logger
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

func NewDispatcher(consumer *queue.Consumer, processor *Processor, concurrency int, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		consumer:    consumer,
		processor:   processor,
		concurrency: concurrency,
		logger:      logger,
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	ctx, d.cancel = context.WithCancel(ctx)

	for i := 0; i < d.concurrency; i++ {
		d.wg.Add(1)
		go d.work(ctx, i)
	}

	d.logger.Info("worker dispatcher started", "concurrency", d.concurrency)
}

func (d *Dispatcher) Stop() {
	d.logger.Info("stopping worker dispatcher...")
	d.cancel()
	d.wg.Wait()
	d.logger.Info("worker dispatcher stopped")
}

func (d *Dispatcher) work(ctx context.Context, workerID int) {
	defer d.wg.Done()

	log := d.logger.With("worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Info("worker shutting down")
			return
		default:
			notificationID, err := d.consumer.Dequeue(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error("dequeue error", "error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if notificationID == "" {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if err := d.processor.Process(ctx, notificationID); err != nil {
				log.Error("process error", "notification_id", notificationID, "error", err)
			}
		}
	}
}
