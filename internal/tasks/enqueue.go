package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

type FetchPricePayload struct {
	UserID  string        `json:"user_id"`
	Symbols []interface{} `json:"symbols"`
}

func NewFetchPriceTask(userID string, symbols []interface{}) (*asynq.Task, error) {
	payload, err := json.Marshal(FetchPricePayload{UserID: userID, Symbols: symbols})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeFetchPrices, payload), nil
}
