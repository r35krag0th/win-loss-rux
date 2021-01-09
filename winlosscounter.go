package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"strings"
)

var ctx = context.Background()

type WinLossCounter struct {
	rdb *redis.Client
	Name string `json:"name"`
	Wins int `json:"wins"`
	Losses int `json:"losses"`
	Draws int `json:"draws"`
}

func NewWinLossCounter(name string) *WinLossCounter {
	tmp := &WinLossCounter{
		Name: name,
		Wins: 0,
		Losses: 0,
		Draws: 0,
	}

	tmp.Connect()
	return tmp
}

func (w WinLossCounter) redisKey() string {
	return fmt.Sprintf("WinLossCounter-%s", w.Name)
}

func (w WinLossCounter) ListAll() []string {
	cmdResponse := w.rdb.Keys(ctx, w.redisKey())
	foundKeys, err := cmdResponse.Result()
	if err != nil {
		fmt.Println("[ListAll] Error: ", err)
		return []string{}
	}

	var returnedKeys []string
	for _, k := range foundKeys {
		returnedKeys = append(returnedKeys, strings.SplitN(k, "-", 2)[1])
	}
	return returnedKeys
}

func (w *WinLossCounter) Connect() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: password,
		DB: 0,
	})
	w.rdb = client
}

func (w *WinLossCounter) ValidateAndFix() {
	if w.Wins < 0 {
		w.Wins = 0
	}

	if w.Losses < 0 {
		w.Losses = 0
	}

	if w.Draws < 0 {
		w.Draws = 0
	}
}

func (w *WinLossCounter) AddWin() {
	w.Wins += 1
	w.Save()
}

func (w *WinLossCounter) RemoveWin() {
	w.Wins -= 1
	w.Save()
}

func (w *WinLossCounter) AddLoss() {
	w.Losses += 1
	w.Save()
}

func (w *WinLossCounter) RemoveLoss() {
	w.Losses -= 1
	w.Save()
}

func (w *WinLossCounter) AddDraw() {
	w.Draws += 1
	w.Save()
}

func (w *WinLossCounter) RemoveDraw() {
	w.Draws -= 1
	w.Save()
}

func (w *WinLossCounter) Reset() {
	w.Wins = 0
	w.Losses = 0
	w.Draws = 0
	w.Save()
}

func (w *WinLossCounter) Destroy() {
	cmdResponse := w.rdb.Del(ctx, w.redisKey())
	_, err := cmdResponse.Result()
	if err != nil {
		fmt.Printf("[Destroy](%s) failed: %v+", w.Name, err)
	}
}

func (w *WinLossCounter) FromJson(v string) error {
	var tmp WinLossCounter
	err := json.Unmarshal([]byte(v), &tmp)
	if err != nil {
		return err
	}

	w.Wins = tmp.Wins
	w.Losses = tmp.Losses
	w.Draws = tmp.Draws

	w.ValidateAndFix()

	return nil
}

func (w WinLossCounter) ToJson() (error, string) {
	b, err := json.Marshal(w)
	if err != nil {
		return err, ""
	}

	return nil, string(b)
}

func (w *WinLossCounter) Load() {
	rawValue, redisErr := w.rdb.Get(ctx, w.redisKey()).Result()
	if redisErr == redis.Nil {
		fmt.Printf("[Load] Key (%s) does not exist\n", w.Name)
		return
	}

	if redisErr != nil {
		fmt.Println("[Load] Error getting state: ", w.Name, redisErr)
		return
	}

	err := w.FromJson(rawValue)
	if err != nil {
		fmt.Println("[Load] Error: ", w.Name, err)
	}
}

func (w *WinLossCounter) Save() {
	w.ValidateAndFix()

	err, stateJson := w.ToJson()
	if err != nil {
		fmt.Println("[Save] Error getting JSON: ", w.Name, err)
		return
	}
	redisErr := w.rdb.Set(ctx, w.redisKey(), stateJson, 0).Err()
	if redisErr != nil {
		fmt.Println("[Save] Error saving state: ", w.Name, err)
	}
}