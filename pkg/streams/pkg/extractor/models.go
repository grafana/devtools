package extractor

import (
	"encoding/json"
	"time"
)

type Repo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Org struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

type ExtractedEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Public    bool            `json:"public"`
	CreatedAt time.Time       `json:"created_at"`
	Actor     json.RawMessage `json:"actor"`
	Repo      Repo            `json:"repo"`
	Org       Org             `json:"org"`
	Payload   json.RawMessage `json:"payload"`
}
