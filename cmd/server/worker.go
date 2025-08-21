package main

import (
	"context"
	"encoding/json"
	"log"
	"portfolio-zen-backend/internal/services"
	"portfolio-zen-backend/internal/tasks"

	"github.com/hibiken/asynq"
)

func worker() {

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "127.0.0.1:6379"},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeFetchPrices, func(c context.Context, t *asynq.Task) error {
		var p tasks.FetchPricePayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		// Call your price-fetching logic
		return services.FetchAndUpdatePrices(c, p)
	})

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
