package main

import (
	"github.com/r35krag0th/win-loss-rux/version"
	"github.com/sirupsen/logrus"
)

// CounterPage is the data structure handed off to the templates to render counters.
type CounterPage struct {
	Name       string
	Title      string
	Wins       int
	Losses     int
	Draws      int
	PrettyName string
}

// NewCounterPageFromWinLossCounter creates a CounterPage from a WinLossCounter that is usually
// created from backend data.
func NewCounterPageFromWinLossCounter(counter *WinLossCounter) *CounterPage {
	logger := logrus.WithFields(logrus.Fields{
		"wins":        counter.Wins,
		"losses":      counter.Losses,
		"draws":       counter.Draws,
		"pretty_name": counter.PrettyName,
		"name":        counter.Name,
		"version":     version.Version,
	})
	logger.Debug("Creating CounterPage")
	return &CounterPage{
		Title:      "",
		Wins:       counter.Wins,
		Losses:     counter.Losses,
		Draws:      counter.Draws,
		PrettyName: counter.PrettyName,
		Name:       counter.Name,
	}
}
