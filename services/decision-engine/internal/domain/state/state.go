package state

type State string

const (
	StateNew                  State = "new"
	StateWaitingForCategory   State = "waiting_for_category"
	StateWaitingClarification State = "waiting_clarification"
	StateSolutionOffered      State = "solution_offered"
	StateEscalatedToOperator  State = "escalated_to_operator"
	StateClosed               State = "closed"
)

