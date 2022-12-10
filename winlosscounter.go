package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/r35krag0th/win-loss-rux/numericsapp"
	"github.com/r35krag0th/win-loss-rux/version"
	"github.com/sirupsen/logrus"
)

var (
	envName         = getenv("APP_ENV", "dev")
	consulKeyPrefix = fmt.Sprintf("win-loss-api/%s/counters", envName)
)

// WinLossCounter represents a counter and is used to persist data in the storage backend.
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

// NewWinLossCounter creates a new WinLossCounter with default values.
// Default values for Wins, Losses, and Draws are zero.
// Additionally, a "pretty name" will be set, which is the counter name in slug format.
// The slug format essentially replaces all non-alpha-numerics with dashes.
func NewWinLossCounter(name string) *WinLossCounter {
	logger := logrus.WithFields(logrus.Fields{
		"name":    name,
		"version": version.Version,
	})
	logger.Debug("Creating new WinLossCounter")

	prettyName := strings.ReplaceAll(name, "-", " ")
	logger.Debugf("Transforming name '%s' into pretty name '%s'", name, prettyName)
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

// ListAll returns a list of all known counter names.
func (w WinLossCounter) ListAll() []string {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "ListAll",
		"version": version.Version,
	})
	// Consul

	logger.Debug("Creating Consul KV Client")
	kv := w.consulClient.KV()

	logger.Debugf("Listing keys with prefix: %s", consulKeyPrefix)
	matchedKeys, _, err := kv.List(consulKeyPrefix, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to list keys")
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

// SetConsulClient will set the Hashicorp Consul client this counter will use to access the storage backend.
func (w *WinLossCounter) SetConsulClient(c *api.Client) {
	w.consulClient = c
}

// ValidateAndFix ensures that Wins, Losses, and Draws are greater than or equal to zero.
func (w *WinLossCounter) ValidateAndFix() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "ValidateAndFix",
		"wins":    w.Wins,
		"losses":  w.Losses,
		"draws":   w.Draws,
		"version": version.Version,
	})

	if w.Wins < 0 {
		logger.WithFields(logrus.Fields{
			"original_value": w.Wins,
			"key":            "wins",
		}).Warn("Wins was less than 0.  Setting to 0.")
		w.Wins = 0
	}

	if w.Losses < 0 {
		logger.WithFields(logrus.Fields{
			"original_value": w.Losses,
			"key":            "losses",
		}).Warn("Losses was less than 0.  Setting to 0.")
		w.Losses = 0
	}

	if w.Draws < 0 {
		logger.WithFields(logrus.Fields{
			"original_value": w.Draws,
			"key":            "draws",
		}).Warn("Draws was less than 0.  Setting to 0.")
		w.Draws = 0
	}
}

// AddWin will to increment the current value of Wins by 1.
// If the counter doesn't exist it will automatically be created.
func (w *WinLossCounter) AddWin() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "AddWin",
		"version": version.Version,
	})

	w.Wins += 1
	logger.Infof("Incrementing Wins to %d", w.Wins)
	w.Save()
}

// RemoveWin will attempt to decrement the current value of Wins by 1.
// If the new value is less than zero it will be set to zero.
func (w *WinLossCounter) RemoveWin() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "RemoveWin",
		"version": version.Version,
	})

	w.Wins -= 1
	logger.Infof("Decrementing Wins to %d", w.Wins)
	w.Save()
}

// AddLoss will to increment the current value of Losses by 1.
// If the counter doesn't exist it will automatically be created.
func (w *WinLossCounter) AddLoss() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "AddLoss",
		"version": version.Version,
	})
	w.Losses += 1
	logger.Infof("Incrementing Losses to %d", w.Losses)
	w.Save()
}

// RemoveLoss will attempt to decrement the current value of Losses by 1.
// If the new value is less than zero it will be set to zero.
func (w *WinLossCounter) RemoveLoss() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "RemoveLoss",
		"version": version.Version,
	})
	w.Losses -= 1
	logger.Infof("Decrementing Losses to %d", w.Losses)
	w.Save()
}

// AddDraw will to increment the current value of Draws by 1.
// If the counter doesn't exist it will automatically be created.
func (w *WinLossCounter) AddDraw() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "AddDraw",
		"version": version.Version,
	})
	w.Draws += 1
	logger.Infof("Incrementing Draws to %d", w.Draws)
	w.Save()
}

// RemoveDraw will attempt to decrement the current value of Draws by 1.
// If the new value is less than zero it will be set to zero.
func (w *WinLossCounter) RemoveDraw() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "RemoveDraw",
		"version": version.Version,
	})
	w.Draws -= 1
	logger.Infof("Decrementing Draws to %d", w.Draws)
	w.Save()
}

// Reset will reset the current counter's values to zero and persist the changes
// in the storage backend (Consul).
func (w *WinLossCounter) Reset() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "Reset",
		"version": version.Version,
	})
	w.Wins = 0
	w.Losses = 0
	w.Draws = 0
	logger.Info("Counter has been reset")
	w.Save()
}

// Destroy will delete the counter, by name, from the storage backend (Consul).
func (w *WinLossCounter) Destroy() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "Destroy",
		"version": version.Version,
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

// FromJson parses the given JSON string to load the counter values.
func (w *WinLossCounter) FromJson(v string) error {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "FromJson",
		"version": version.Version,
	})
	logger.WithFields(logrus.Fields{
		"json_input": v,
	}).Debug("attempting to unmarshal json input")
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

// ToJson returns the JSON string representation of the current counter and it's values.
func (w WinLossCounter) ToJson() (error, string) {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "ToJson",
		"version": version.Version,
	})
	b, err := json.Marshal(w)
	if err != nil {
		logger.WithError(err).Error("Failed to marshall this counter into JSON")
		return err, ""
	}

	logger.Debug("This counter was successfully marshalled into JSON")
	return nil, string(b)
}

// Load will hydrate the counter data from the storage backend (Consul).
func (w *WinLossCounter) Load() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "Load",
		"version": version.Version,
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

// Save persists the current counter in the storage backend (Consul).
func (w *WinLossCounter) Save() {
	logger := logrus.WithFields(logrus.Fields{
		"name":    w.Name,
		"func":    "Save",
		"version": version.Version,
	})
	w.ValidateAndFix()

	logger.Debug("Creating state JSON")
	err, stateJson := w.ToJson()
	if err != nil {
		logger.WithError(err).Error("Failed to JSON-ify Counter")
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

func (w WinLossCounter) valueToNumericsCounter(value int, postfix string, color string) *numericsapp.CounterWidgetResponse {
	return &numericsapp.CounterWidgetResponse{
		WidgetResponse: numericsapp.WidgetResponse{
			Postfix: postfix,
			Color:   color,
		},
		Data: numericsapp.NewNDataInt(value),
	}
}

// WinsToNumericsCounter returns the Wins value with color for the Numerics iOS Application
func (w WinLossCounter) WinsToNumericsCounter(color string) *numericsapp.CounterWidgetResponse {
	return w.valueToNumericsCounter(w.Wins, "Wins", color)
}

// LossesToNumericsCounter returns the Losses value with color for the Numerics iOS Application
func (w WinLossCounter) LossesToNumericsCounter(color string) *numericsapp.CounterWidgetResponse {
	return w.valueToNumericsCounter(w.Losses, "Losses", color)
}

// DrawsToNumericsCounter returns the Draws value with color for the Numerics iOS Application
func (w WinLossCounter) DrawsToNumericsCounter(color string) *numericsapp.CounterWidgetResponse {
	return w.valueToNumericsCounter(w.Draws, "Draws", color)
}
