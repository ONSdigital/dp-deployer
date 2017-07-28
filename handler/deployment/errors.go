package deployment

// AllocationError is an error implementation that includes pending, running and total
// allocation counts.
type AllocationError struct {
	Pending int
	Running int
	Total   int
}

func (e *AllocationError) Error() string {
	return "failed to start all allocations"
}

// AllocationAbortedError is an error implementation that includes the id of the aborted
// allocation.
type AllocationAbortedError struct {
	EvaluationID string
}

func (e *AllocationAbortedError) Error() string {
	return "aborted monitoring allocations for evaluation"
}

// ClientResponseError is an error implementation that includes the body and status
// code of the response.
type ClientResponseError struct {
	Body       string
	StatusCode int
}

func (e *ClientResponseError) Error() string {
	return "unexpected response from client"
}

// EvaluationError is an error implementation that includes the evaluation id of the
// allocations.
type EvaluationError struct {
	ID string
}

func (e *EvaluationError) Error() string {
	return "error occurred for evaluation"
}

// EvaluationAbortedError is an error implementation that includes the id of the
// evaluation.
type EvaluationAbortedError struct {
	ID string
}

func (e *EvaluationAbortedError) Error() string {
	return "aborted monitoring evaluation"
}

// PlanError is an error implementation that includes the errors or warnings
type PlanError struct {
	Errors   string
	Service  string
	Warnings string
}

func (e *PlanError) Error() string {
	return "plan for tasks generated errors or warnings"
}

// TimeoutError is an error implementation that includes the action that timed out.
type TimeoutError struct {
	Action string
}

func (e *TimeoutError) Error() string {
	return "timed out waiting for action to complete"
}
