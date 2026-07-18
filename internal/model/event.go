package model

const EventSchema = "elefante.events/v1"

type EventType string

const (
	EventStarted          EventType = "started"
	EventFact             EventType = "fact"
	EventDiagnostic       EventType = "diagnostic"
	EventPlan             EventType = "plan"
	EventApprovalRequired EventType = "approval_required"
	EventProgress         EventType = "progress"
	EventStdout           EventType = "stdout"
	EventStderr           EventType = "stderr"
	EventResult           EventType = "result"
	EventError            EventType = "error"
	EventCompleted        EventType = "completed"
)

type Event struct {
	Schema   string    `json:"schema"`
	Sequence uint64    `json:"sequence"`
	Command  string    `json:"command"`
	Type     EventType `json:"type"`
	Payload  any       `json:"payload"`
}

type ExitOrigin string

const (
	ExitOriginElefante ExitOrigin = "elefante"
	ExitOriginChild    ExitOrigin = "child"
)

type Exit struct {
	Origin ExitOrigin `json:"origin"`
	Code   int        `json:"code"`
}

type CompletedPayload struct {
	Exit Exit `json:"exit"`
}

type ApprovalRequiredPayload struct {
	PlanDigest string             `json:"plan_digest"`
	Effects    []EffectClass      `json:"effects"`
	Trust      []TrustRequirement `json:"trust,omitempty"`
}
