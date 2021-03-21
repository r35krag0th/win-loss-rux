package main

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"strings"
)

const consulKeyPrefix = "win-loss-api/counters"

type WinLossCounter struct {
	consulClient *api.Client
	Name         string `json:"name"`
	PrettyName   string `json:"pretty_name,omitempty"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	Draws        int    `json:"draws"`
	Urls         struct {
		Html string `json:"html"`
		Api  string `json:"api"`
	}
}

func NewWinLossCounter(name string) *WinLossCounter {
	prettyName := strings.ReplaceAll(name, "-", " ")
	fmt.Printf("Transforming '%s' into '%s'", name, prettyName)
	tmp := &WinLossCounter{
		Name: name,
		// PrettyName: strings.ReplaceAll(name, "-", " "),
		PrettyName: prettyName,
		Wins:       0,
		Losses:     0,
		Draws:      0,
	}
	return tmp
}

func (w WinLossCounter) consulKey() string {
	return fmt.Sprintf(strings.Join([]string{consulKeyPrefix, "%s"}, "/"), w.Name)
}

func (w WinLossCounter) ListAll() []string {
	// Consul
	kv := w.consulClient.KV()
	matchedKeys, _, err := kv.List(consulKeyPrefix, nil)
	if err != nil {
		fmt.Printf("[ListAll] Failed for Key (%s) -- %v", w.Name, err)
	}

	var returnedKeys []string
	for _, k := range matchedKeys {
		// fmt.Printf("[ListAll] k=%s\n", k)
		splitBySlash := strings.SplitN(k.Key, "/", 3)

		if len(splitBySlash) < 3 || splitBySlash[2] == "" {
			fmt.Printf("--- Too few segments or last segment was empy.  SKIP.")
			continue
		}

		returnedKeys = append(
			returnedKeys,
			strings.SplitN(k.Key, "/", 3)[2],
		)
	}
	return returnedKeys
}

func (w *WinLossCounter) SetConsulClient(c *api.Client) {
	w.consulClient = c
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
	kv := w.consulClient.KV()
	_, err := kv.Delete(w.consulKey(), nil)
	if err != nil {
		fmt.Printf("[Destroy](%s) failed: %v+\n", w.Name, err)
		return
	}

	fmt.Printf("[Destroy](%s) OK\n", w.Name)
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
	fmt.Printf("[Load] consulClient is -> %+v\n", w.consulClient)

	// === Consul
	fmt.Println("[Load] Pre Getting KV Client")
	kv := w.consulClient.KV()

	fmt.Printf("[Load] Pre kv.Get(%s, nil)\n", w.consulKey())
	p, _, err := kv.Get(w.consulKey(), nil)

	fmt.Println("[Load] Pre err != nil check")
	if err != nil {
		fmt.Printf("[Load] Key (%s) does not exist -- %s\n", w.Name, err)
		return
	}
	if p == nil {
		fmt.Printf("[Load] Key (%s) variable 'p' from Consul was nil...\n", w.Name)
		return
	}

	err = w.FromJson(string(p.Value))
	if err != nil {
		fmt.Println("[Load] Error: ", w.Name, err)
		return
	}
}

func (w *WinLossCounter) Save() {
	w.ValidateAndFix()

	err, stateJson := w.ToJson()
	if err != nil {
		fmt.Println("[Save] Error getting JSON: ", w.Name, err)
		return
	}

	// Consul
	kv := w.consulClient.KV()
	wp := &api.KVPair{Key: w.consulKey(), Value: []byte(stateJson)}
	_, err = kv.Put(wp, nil)
	if err != nil {
		fmt.Println("[Save->Consul] Failed to write Wins", err)
	}
}
