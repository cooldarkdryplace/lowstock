package lowstock

import (
	"context"
	"log"
	"time"
)

var (
	pollPeriod   = 20 * time.Second
	nUpdHandlers = 10
)

type Worker struct {
	ls      *LowStock
	ticker  *time.Ticker
	updChan chan Update
}

func NewWorker(l *LowStock) *Worker {
	return &Worker{
		ls:      l,
		ticker:  time.NewTicker(pollPeriod),
		updChan: make(chan Update, 1000), // TODO: replace with PubSub
	}
}

func (w *Worker) handleUpdates(ctx context.Context) {
	for {
		select {
		case update := <-w.updChan:
			if err := w.ls.HandleEtsyUpdate(ctx, update); err != nil {
				log.Printf("Failed to handle update: %s", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) etsyUpdates(ctx context.Context) {
	updates, err := w.ls.Updates(ctx)
	if err != nil {
		log.Printf("Failed to handle updates: %s", err)
		return
	}

	for _, upd := range updates {
		w.updChan <- upd
	}
}

func (w *Worker) Run(ctx context.Context) {
	log.Println("Starting Etsy Update workers...")
	for i := 0; i < nUpdHandlers; i++ {
		go w.handleUpdates(ctx)
	}

	w.etsyUpdates(ctx)

	for {
		select {
		case <-w.ticker.C:
			w.etsyUpdates(ctx)
		case <-ctx.Done():
			log.Println("Stopping worker...")
			return
		}
	}
}
