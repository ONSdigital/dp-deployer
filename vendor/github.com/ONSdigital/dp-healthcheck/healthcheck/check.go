package healthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// A list of possible check statuses
const (
	StatusOK       = "OK"
	StatusWarning  = "WARNING"
	StatusCritical = "CRITICAL"
)

// Checker represents the interface all checker functions abide to
type Checker func(context.Context, *CheckState) error

// CheckState represents the health status returned by a checker
type CheckState struct {
	name        string
	status      string
	statusCode  int
	message     string
	lastChecked *time.Time
	lastSuccess *time.Time
	lastFailure *time.Time
	mutex       *sync.RWMutex
}

// checkStateJSON represents the health status struct for use with json marshal/unmarshal (to deal with unexported fields)
type checkStateJSON struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	StatusCode  int        `json:"status_code,omitempty"`
	Message     string     `json:"message"`
	LastChecked *time.Time `json:"last_checked"`
	LastSuccess *time.Time `json:"last_success"`
	LastFailure *time.Time `json:"last_failure"`
}

// Check represents a check performed by the health check
type Check struct {
	state   *CheckState
	checker Checker
}

// Name gets the check name
func (s *CheckState) Name() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.name
}

// Status gets the check status
func (s *CheckState) Status() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.status
}

// StatusCode gets the check status code
func (s *CheckState) StatusCode() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.statusCode
}

// Message gets the check message
func (s *CheckState) Message() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.message
}

// LastChecked gets the last checked time of the check
func (s *CheckState) LastChecked() *time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastChecked == nil {
		return nil
	}

	t := *s.lastChecked
	return &t
}

// LastSuccess gets the time of the last successful check
func (s *CheckState) LastSuccess() *time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastSuccess == nil {
		return nil
	}

	t := *s.lastSuccess
	return &t
}

// LastFailure gets the time of the last failed check
func (s *CheckState) LastFailure() *time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastFailure == nil {
		return nil
	}

	t := *s.lastFailure
	return &t
}

// Update updates the relevant state fields based on the status provided
// status of the check, must be one of healthcheck.StatusOK, healthcheck.StatusWarning or healthcheck.StatusCritical
// message briefly describing the check state
// statusCode returned if the check was an HTTP check (optional, provide 0 if not relevant)
func (s *CheckState) Update(status, message string, statusCode int) error {
	now := time.Now().UTC()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch status {
	case StatusOK:
		s.lastSuccess = &now
	case StatusWarning, StatusCritical:
		s.lastFailure = &now
	default:
		return fmt.Errorf("invalid check status, must be one of %s, %s or %s", StatusOK, StatusWarning, StatusCritical)
	}

	s.status = status
	s.message = message
	s.statusCode = statusCode
	s.lastChecked = &now

	return nil
}

// hasRun returns true if the check has been run and has state
func (c *Check) hasRun() bool {
	if c.state.LastChecked() == nil {
		return false
	}
	return true
}

// NewCheck returns a pointer to a new instantiated Check with
// the provided checker function
func NewCheck(name string, checker Checker) (*Check, error) {
	if checker == nil {
		return nil, errors.New("expected checker but none provided")
	}

	return &Check{
		state:   NewCheckState(name),
		checker: checker,
	}, nil
}

// NewCheckState returns a pointer to a new instantiated CheckState
func NewCheckState(name string) *CheckState {
	return &CheckState{
		name:  name,
		mutex: &sync.RWMutex{},
	}
}

// MarshalJSON returns the json representation of the check as a byte array
func (c *Check) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.state)
}

// MarshalJSON returns the json representation of the check state as a byte array
func (s *CheckState) MarshalJSON() ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return json.Marshal(checkStateJSON{
		Name:        s.name,
		Status:      s.status,
		StatusCode:  s.statusCode,
		Message:     s.message,
		LastChecked: s.lastChecked,
		LastSuccess: s.lastSuccess,
		LastFailure: s.lastFailure,
	})
}

// UnmarshalJSON takes the json representation of a check as a byte array and populates the Check object
func (c *Check) UnmarshalJSON(b []byte) error {
	if c.state == nil {
		c.state = NewCheckState("")
	}
	return json.Unmarshal(b, c.state)
}

// UnmarshalJSON takes the json representation of a check state as a byte array and populates the CheckState object
func (s *CheckState) UnmarshalJSON(b []byte) error {
	if s.mutex == nil {
		*s = *NewCheckState("")
	}

	temp := &checkStateJSON{}
	err := json.Unmarshal(b, temp)
	if err == nil {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		s.name = temp.Name
		s.status = temp.Status
		s.statusCode = temp.StatusCode
		s.message = temp.Message
		s.lastChecked = temp.LastChecked
		s.lastSuccess = temp.LastSuccess
		s.lastFailure = temp.LastFailure
	}
	return err
}
