package types

// Workflow enums. These are stored as short strings in the database.

// NodeStatus is the lifecycle status of a workflow node.
type NodeStatus string

const (
	NodeDraft     NodeStatus = "draft"     // entry / amendment node
	NodeRunning   NodeStatus = "running"   // an approval step
	NodeCompleted NodeStatus = "completed" // terminal success
	NodeRejected  NodeStatus = "rejected"  // terminal rejection
)

// NodeType distinguishes ordinary approval nodes from routing nodes.
type NodeType string

const (
	NodeTypeNode NodeType = "node"
)

// QuorumMode controls how many operators of a node must agree to advance.
type QuorumMode string

const (
	QuorumAny QuorumMode = "any" // one approval advances
	QuorumAll QuorumMode = "all" // every assigned operator must approve
)

// TransitionAction is the kind of move a transition represents.
type TransitionAction string

const (
	ActAgree    TransitionAction = "agree"
	ActReject   TransitionAction = "reject"
	ActBack     TransitionAction = "back"
	ActTransfer TransitionAction = "transfer"
)

// TaskStatus is the status of a per-user workflow task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskCompleted TaskStatus = "completed"
	TaskSkipped   TaskStatus = "skipped"
)

// EventAction labels an entry in a workflow instance's history.
type EventAction string

const (
	EventInitiate EventAction = "initiate"
	EventAgree    EventAction = "agree"
	EventReject   EventAction = "reject"
	EventTransfer EventAction = "transfer"
	EventBack     EventAction = "back"
	EventComplete EventAction = "complete"
)

// AmendmentMode controls who may correct a soft-rejected document.
type AmendmentMode string

const (
	AmendCreator AmendmentMode = "creator" // only the original initiator
	AmendPool    AmendmentMode = "pool"    // any user in the initiator pool
)

// NotifyEvent is a process-level informational (FYI) mail, not an inbox task.
type NotifyEvent string

const (
	NotifySubmit   NotifyEvent = "submit"   // document entered approval
	NotifyComplete NotifyEvent = "complete" // last step agreed — fully approved
	NotifyReject   NotifyEvent = "reject"   // hard / terminal rejection
)
