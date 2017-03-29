package main

import (
	"errors"
	"time"
)

// TimeTracker tracks when road-runner should exit.
type TimeTracker struct {
	Timer   *time.Timer
	EndDate time.Time
}

// NewTimeTracker returns a new *TimeTracker.
func NewTimeTracker(d time.Duration, exitFunc func()) *TimeTracker {
	endDate := time.Now().Add(d)
	exitTimer := time.AfterFunc(d, exitFunc)
	return &TimeTracker{
		EndDate: endDate,
		Timer:   exitTimer,
	}
}

// ApplyDelta generates a new end date and modifies the time with the passed-in
// duration.
func (t *TimeTracker) ApplyDelta(deltaDuration time.Duration) error {
	//apply the new duration to the current end date.
	newEndDate := t.EndDate.Add(deltaDuration)

	//create a new duration that is the difference between the new end date and now.
	newDuration := t.EndDate.Sub(time.Now())

	//modify the Timer to use the new duration.
	wasActive := t.Timer.Reset(newDuration)

	//set the new enddate
	t.EndDate = newEndDate

	if !wasActive {
		return errors.New("Timer was not active")
	}
	return nil
}
