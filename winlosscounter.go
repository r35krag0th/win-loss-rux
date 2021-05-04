package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

var (
	envName         = getenv("APP_ENV", "dev")
	consulKeyPrefix = fmt.Sprintf("win-loss-api/%s/counters", envName)
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

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
	logger := logrus.WithFields(logrus.Fields{
		"name": name,
	})
	logger.Debugf("Creating new WinLossCounter")

	prettyName := strings.ReplaceAll(name, "-", " ")
	logger.Debug("Transforming name '%s' into pretty name '%s'", name, prettyName)
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
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "ListAll",
	})
	// Consul

	logger.Debug("Creating Consul KV Client")
	kv := w.consulClient.KV()

	logger.Debugf("Listing keys with prefix: %s", consulKeyPrefix)
	matchedKeys, _, err := kv.List(consulKeyPrefix, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to list keys", w.Name)
	}

	var returnedKeys []string
	for _, k := range matchedKeys {
		// fmt.Printf("[ListAll] k=%s\n", k)
		splitBySlash := strings.SplitN(k.Key, "/", 3)

		if len(splitBySlash) < 3 || splitBySlash[2] == "" {
			logger.Warn("--- Too few segments or last segment was empy.  SKIP.")
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
	logger := logrus.WithFields(logrus.Fields{
		"name":   w.Name,
		"func":   "ValidateAndFix",
		"wins":   w.Wins,
		"losses": w.Losses,
		"draws":  w.Draws,
	})

	if w.Wins < 0 {
		logger.Warnf("Wins was less than 0.  Setting to 0.")
		w.Wins = 0
	}

	if w.Losses < 0 {
		logger.Warnf("Losses was less than 0.  Setting to 0.")
		w.Losses = 0
	}

	if w.Draws < 0 {
		logger.Warnf("Draws was less than 0.  Setting to 0.")
		w.Draws = 0
	}
}

func (w *WinLossCounter) AddWin() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "AddWin",
	})

	w.Wins += 1
	logger.Infof("Incrementing Wins to %d", w.Wins)
	w.Save()
}

func (w *WinLossCounter) RemoveWin() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "RemoveWin",
	})

	w.Wins -= 1
	logger.Info("Decremting Wins to %d", w.Wins)
	w.Save()
}

func (w *WinLossCounter) AddLoss() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "AddLoss",
	})
	w.Losses += 1
	logger.Info("Incrementing Losses to %d", w.Losses)
	w.Save()
}

func (w *WinLossCounter) RemoveLoss() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "RemoveLoss",
	})
	w.Losses -= 1
	logger.Info("Decremting Losses to %d", w.Losses)
	w.Save()
}

func (w *WinLossCounter) AddDraw() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "AddDraw",
	})
	w.Draws += 1
	logger.Info("Incrementing Draws to %d", w.Draws)
	w.Save()
}

func (w *WinLossCounter) RemoveDraw() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "RemoveDraw",
	})
	w.Draws -= 1
	logger.Info("Decremting Draws to %d", w.Draws)
	w.Save()
}

func (w *WinLossCounter) Reset() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "Reset",
	})
	w.Wins = 0
	w.Losses = 0
	w.Draws = 0
	logger.Info("Counter has been reset")
	w.Save()
}

func (w *WinLossCounter) Destroy() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "Destroy",
	})

	logger.Debug("Getting Consul Client")
	kv := w.consulClient.KV()

	logger.Debugf("Deleting the key '%s'", w.consulKey())
	_, err := kv.Delete(w.consulKey(), nil)
	if err != nil {
		logger.WithError(err).Error("Destroying the counter failed")
		return
	}

	logger.Info("The counter has been destroyed")
}

func (w *WinLossCounter) FromJson(v string) error {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "FromJson",
	})
	logger.Debug(v)
	var tmp WinLossCounter
	err := json.Unmarshal([]byte(v), &tmp)
	if err != nil {
		logger.WithError(err).Error("Failed to unmarshall into a WinLossCounter")
		return err
	}

	logger.Debug("JSON has been unmarshalled into a WinLossCounter")
	w.Wins = tmp.Wins
	w.Losses = tmp.Losses
	w.Draws = tmp.Draws

	w.ValidateAndFix()

	return nil
}

func (w WinLossCounter) ToJson() (error, string) {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "ToJson",
	})
	b, err := json.Marshal(w)
	if err != nil {
		logger.WithError(err).Error("Failed to marshall this counter into JSON")
		return err, ""
	}

	logger.Debug("This counter was successfully marshalled into JSON")
	return nil, string(b)
}

func (w *WinLossCounter) Load() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "Load",
	})

	if w.Name == "" {
		logger.Debug("Bypassing Load because this counter has no name")
		return
	}

	logger.Debugf("consulClient is -> %+v", w.consulClient)

	// === Consul
	logger.Debug("(Before) Getting KV Client")
	kv := w.consulClient.KV()

	logger.Debugf("(Before) kv.Get(%s, nil)", w.consulKey())
	p, _, err := kv.Get(w.consulKey(), nil)

	logger.Debug("(Before) err != nil check")
	if err != nil {
		logger.WithError(err).Errorf("Key Not Found: %s", w.consulKey())
		return
	}
	if p == nil {
		logger.Errorf("Key did not have a value: %s", w.consulKey())
		return
	}

	err = w.FromJson(string(p.Value))
	if err != nil {
		logger.WithError(err).Error("Failed to Load")
		return
	}
}

func (w *WinLossCounter) Save() {
	logger := logrus.WithFields(logrus.Fields{
		"name": w.Name,
		"func": "Save",
	})
	w.ValidateAndFix()

	logger.Debug("Creating state JSON")
	err, stateJson := w.ToJson()
	if err != nil {
		logger.WithError(err).Error("Failed to JSONify Counter")
		return
	}

	// Consul
	logger.Debug("Creating Consul KV Client")
	kv := w.consulClient.KV()

	logger.Debugf("Creating KV Pair for %s with JSON Data: %s", w.consulKey(), stateJson)
	wp := &api.KVPair{Key: w.consulKey(), Value: []byte(stateJson)}
	_, err = kv.Put(wp, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to write new state to Consul")
	}
}

func (w WinLossCounter) valueToNumericsCounter(value int, postfix string, color string) *NumericsCounterWidgetResponse {

	return &NumericsCounterWidgetResponse{
		NumericsWidgetResponse: NumericsWidgetResponse{
			Postfix: postfix,
			Color:   color,
		},
		Data: struct {
			Value int `json:"value"`
		}{
			Value: value,
		},
	}
}

func (w WinLossCounter) WinsToNumericsCounter(color string) *NumericsCounterWidgetResponse {
	return w.valueToNumericsCounter(w.Wins, "Wins", color)
}

func (w WinLossCounter) LossesToNumericsCounter(color string) *NumericsCounterWidgetResponse {
	return w.valueToNumericsCounter(w.Losses, "Losses", color)
}

func (w WinLossCounter) DrawsToNumericsCounter(color string) *NumericsCounterWidgetResponse {
	return w.valueToNumericsCounter(w.Draws, "Draws", color)
}
