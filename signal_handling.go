package main

import "os"

// SignalHandler provides the logic for handling various process-ending signals.
type SignalHandler struct {
	Signals chan os.Signal
}

// InitSignalHandler returns a newly created *SignalHandler. This does not call
// signal.Notify from the stdlib. You should do that yourself by passing
// *SignalHandler.Receive as first parameter to signal.Notify.
func InitSignalHandler() *SignalHandler {
	return &SignalHandler{
		Signals: make(chan os.Signal, 1),
	}
}

// SignalProcessor is the function signature for functions that process the
// signals that get received by a SignalHandler
type SignalProcessor func(os.Signal)

// QuitProcessor is the function signature for the functions that handle clean
// up operations when a SignalHandler receives a quit command.
type QuitProcessor func()

// Receive fires up a goroutine that receives signals from SignalHandler.Signals
// and passes them off to the SignalProcessor.
func (s *SignalHandler) Receive(quit chan bool, f SignalProcessor, q QuitProcessor) {
	go func() {
		for {
			select {
			case sig := <-s.Signals:
				f(sig)
			case <-quit:
				q()
				break
			}
		}
	}()
}
