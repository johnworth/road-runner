package main

import (
	"os"
	"testing"
	"time"
)

func TestInitSignalHandler(t *testing.T) {
	t.Run("not nil", func(t *testing.T) {
		if actual := InitSignalHandler(); actual == nil {
			t.Errorf("returned nil instead of pointer")
		}
	})

	t.Run("signals channel created", func(t *testing.T) {
		if handler := InitSignalHandler(); handler.Signals == nil {
			t.Error("channl was nil")
		}
	})
}

func TestReceive(t *testing.T) {
	workdone := make(chan bool)
	quitrecv := make(chan bool)

	processor := func(s os.Signal) {
		workdone <- true
	}

	quitprocessor := func() {
		quitrecv <- true
	}

	t.Run("interrupt received", func(t *testing.T) {
		var handler *SignalHandler
		q := make(chan bool)

		if handler = InitSignalHandler(); handler == nil {
			t.Fatal("nil SignalHandler")
		}

		handler.Receive(q, processor, quitprocessor)
		handler.Signals <- os.Interrupt
		select {
		case <-workdone:
		case <-time.After(time.Second * 3):
			t.Error("signal wasn't handled")
		}
	})

	t.Run("quit received", func(t *testing.T) {
		var handler *SignalHandler
		q := make(chan bool)

		if handler = InitSignalHandler(); handler == nil {
			t.Fatal("nil SignalHandler")
		}

		handler.Receive(q, processor, quitprocessor)
		q <- true
		select {
		case <-quitrecv:
		case <-time.After(time.Second * 3):
			t.Error("quit wasn't handled")
		}
	})
}
