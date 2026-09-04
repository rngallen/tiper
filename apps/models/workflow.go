package models

import (
	"dfms/pkg/types"
	"dfms/utils"
	"time"

	"gorm.io/gorm"
)

// Process is an approval workflow definition. One process is seeded per
// workflow-backed content type (receipt, GLR, billing run, …). Nodes are the
// states; Transitions are the moves between them.
type Process struct {
	ID                 uint                `gorm:"primaryKey" json:"-"`
	UID                string              `gorm:"type:varchar(26);uniqueIndex:idx_uniqueProcessUID;not null" json:"id"`
	ContentType        types.ContentType   `gorm:"default:12;not null;check:ContentType=12" json:"-"`
	Code               string              `gorm:"type:varchar(100);uniqueIndex:idx_uniqueProcessCode;not null;check:chk_process_code,[Code] <> ''" json:"code"`
	Name               string              `gorm:"type:varchar(255);not null;check:chk_process_name,[Name] <> ''" json:"name"`
	DocContentType     types.ContentType   `gorm:"uniqueIndex:idx_uniqueProcessDocCT;not null" json:"docContentType"`
	AmendmentMode      types.AmendmentMode `gorm:"type:varchar(16);default:'creator';not null;check:AmendmentMode IN ('creator','pool')" json:"amendmentMode"`
	RejectReturnNodeID *uint               `gorm:"index:idx_processRejectReturn" json:"-"` // no FK (soft pointer into Node)
	Note               string              `gorm:"type:nvarchar(max)" json:"note"`
	CreatedAt          time.Time           `json:"-"`
	UpdatedAt          time.Time           `json:"-"`
	Nodes              []Node              `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"nodes,omitempty"`
	Transitions        []Transition        `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"transitions,omitempty"`
}

// BeforeCreate assigns a ULID public identifier.
func (p *Process) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	p.UID = uid
	return nil
}

// Node is a state inside a Process. OperatorUsers / OperatorRoles define who is
// assigned tasks when an instance enters the node.
type Node struct {
	ID            uint              `gorm:"primaryKey" json:"-"`
	UID           string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueNodeUID;not null" json:"id"`
	ContentType   types.ContentType `gorm:"default:13;not null;check:ContentType=13" json:"-"`
	ProcessID     uint              `gorm:"not null;uniqueIndex:idx_uniqueNodeSequence,priority:1;index:idx_nodeProcessStatus,priority:1" json:"-"`
	Name          string            `gorm:"type:varchar(255);not null" json:"name"`
	Status        types.NodeStatus  `gorm:"type:varchar(16);not null;index:idx_nodeStatus;index:idx_nodeProcessStatus,priority:2;check:Status IN ('draft','running','completed','rejected')" json:"status"`
	NodeType      types.NodeType    `gorm:"type:varchar(16);default:'node';not null;check:NodeType IN ('node')" json:"nodeType"`
	QuorumMode    types.QuorumMode  `gorm:"type:varchar(16);default:'any';not null;check:QuorumMode IN ('any','all')" json:"quorumMode"`
	Sequence      int               `gorm:"default:0;not null;uniqueIndex:idx_uniqueNodeSequence,priority:2" json:"sequence"`
	Note          string            `gorm:"type:nvarchar(max)" json:"note"`
	CreatedAt     time.Time         `json:"-"`
	UpdatedAt     time.Time         `json:"-"`
	Process       *Process          `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"process,omitempty"`
	OperatorUsers []User            `gorm:"many2many:NodeOperatorUser;constraint:OnDelete:NO ACTION" json:"operatorUsers,omitempty"`
	OperatorRoles []Role            `gorm:"many2many:NodeOperatorRole;constraint:OnDelete:NO ACTION" json:"operatorRoles,omitempty"`
}

// NodeOperatorRole is the Node ↔ Role join created by Node.OperatorRoles.
// PK (NodeID, RoleID) serves "who operates this step". idx_nodeOpRoleByRole
// (RoleID, NodeID) serves role-delete ("is this role attached to a step?").
type NodeOperatorRole struct {
	NodeID uint `gorm:"primaryKey;index:idx_nodeOpRoleByRole,priority:2"`
	RoleID uint `gorm:"primaryKey;index:idx_nodeOpRoleByRole,priority:1"`
}

func (NodeOperatorRole) TableName() string { return "NodeOperatorRole" }

// NodeOperatorUser is the Node ↔ User join created by Node.OperatorUsers.
// idx_nodeOpUserByUser (UserID, NodeID) covers direct-operator lookups.
type NodeOperatorUser struct {
	NodeID uint `gorm:"primaryKey;index:idx_nodeOpUserByUser,priority:2"`
	UserID uint `gorm:"primaryKey;index:idx_nodeOpUserByUser,priority:1"`
}

func (NodeOperatorUser) TableName() string { return "NodeOperatorUser" }

// BeforeCreate assigns a ULID public identifier.
func (n *Node) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	n.UID = uid
	return nil
}

// Transition connects an input node to an output node for a given action.
// Input/Output node IDs are soft pointers (nullable for start/end edges).
type Transition struct {
	ID            uint                   `gorm:"primaryKey" json:"-"`
	UID           string                 `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTransitionUID;not null" json:"id"`
	ContentType   types.ContentType      `gorm:"default:14;not null;check:ContentType=14" json:"-"`
	ProcessID     uint                   `gorm:"not null;uniqueIndex:idx_uniqueTransitionCode,priority:1" json:"-"`
	Name          string                 `gorm:"type:varchar(100);not null" json:"name"`
	Code          string                 `gorm:"type:varchar(100);not null;uniqueIndex:idx_uniqueTransitionCode,priority:2" json:"code"`
	ActType       types.TransitionAction `gorm:"type:varchar(16);index:idx_transitionActType;not null;check:ActType IN ('agree','reject','back','transfer')" json:"actType"`
	InputNodeID   *uint                  `gorm:"index:idx_transitionInput" json:"-"`
	OutputNodeID  *uint                  `gorm:"index:idx_transitionOutput" json:"-"`
	InputNodeUID  string                 `gorm:"-" json:"inputNodeId,omitempty"`
	OutputNodeUID string                 `gorm:"-" json:"outputNodeId,omitempty"`
	IsActive      bool                   `gorm:"default:1;not null" json:"isActive"`
	CreatedAt     time.Time              `json:"-"`
	UpdatedAt     time.Time              `json:"-"`
	Process       *Process               `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"-"`
}

// BeforeCreate assigns a ULID public identifier.
func (t *Transition) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	t.UID = uid
	return nil
}

// ProcessInstance is a running workflow for one document (ObjectID +
// DocContentType). CurNodeID is the node it currently sits at.
// At most one active row (Status running or draft) per document is enforced
// by filtered unique index idx_uniqueActiveInstanceDoc in pkg/migrate.
type ProcessInstance struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueInstanceUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:15;not null;check:ContentType=15" json:"-"`
	// No is the operator-facing document number (GLR, receipt, billing run),
	// not the instance UID. Used in search, emails, and audit text.
	// chk_instance_no rejects empty strings on a greenfield migrate.
	No             string            `gorm:"type:varchar(100);index:idx_instanceNo;not null;check:chk_instance_no,[No] <> ''" json:"no"`
	ProcessID      uint              `gorm:"index:idx_instanceProcess;not null" json:"-"`
	DocContentType types.ContentType `gorm:"not null;index:idx_instanceDocObject,priority:1;index:idx_instanceDocStatus,priority:1" json:"docContentType"`
	ObjectID       uint              `gorm:"not null;index:idx_instanceDocObject,priority:2" json:"-"`
	CreatedByID    *uint             `gorm:"index:idx_instanceCreatedBy" json:"-"`
	CreatedBy      *User             `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION" json:"-"`
	CurNodeID      *uint             `gorm:"index:idx_instanceCurNode" json:"-"` // soft pointer into Node
	Status         types.NodeStatus  `gorm:"type:varchar(16);index:idx_instanceStatus;index:idx_instanceDocStatus,priority:2;not null;check:Status IN ('draft','running','completed','rejected')" json:"status"`
	Summary        string            `gorm:"type:varchar(500)" json:"summary"`
	Version        int               `gorm:"default:0;not null" json:"version"`
	// ResumeNodeID is set on soft-reject to the node that rejected, so resubmit
	// returns to that step instead of restarting the whole approval chain.
	ResumeNodeID *uint      `gorm:"index:idx_instanceResumeNode" json:"-"`
	CreatedAt    time.Time  `json:"createdAt"`
	SubmittedAt  *time.Time `json:"submittedAt,omitempty"`
	EndedAt      *time.Time `json:"endedAt,omitempty"`
	Process      *Process   `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"-"`
}

// BeforeCreate assigns a ULID public identifier.
func (p *ProcessInstance) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	p.UID = uid
	return nil
}

// Task is a unit of work assigned to a user at a node within an instance.
// idx_taskStatusUser (Status, UserID, CreatedAt) matches the inbox seek
// WHERE Status = pending AND UserID = :me.
// At most one pending task per assignee per node per instance is enforced by
// filtered unique idx_uniquePendingTaskUser in pkg/migrate.
type Task struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueTaskUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:16;not null;check:ContentType=16" json:"-"`
	InstanceID  uint              `gorm:"not null;index:idx_taskPending,priority:1" json:"-"`
	NodeID      uint              `gorm:"not null;index:idx_taskPending,priority:2" json:"-"`
	UserID      *uint             `gorm:"index:idx_taskPending,priority:3;index:idx_taskUserStatus,priority:1;index:idx_taskStatusUser,priority:2" json:"-"`
	Status      types.TaskStatus  `gorm:"type:varchar(32);not null;index:idx_taskPending,priority:4;index:idx_taskUserStatus,priority:2;index:idx_taskStatusCreated,priority:1;index:idx_taskStatusUser,priority:1;check:Status IN ('pending','completed','skipped')" json:"status"`
	Comment     string            `gorm:"type:varchar(500)" json:"comment"`
	ActedAt     *time.Time        `json:"actedAt,omitempty"`
	CreatedAt   time.Time         `gorm:"index:idx_taskUserStatus,priority:3;index:idx_taskStatusCreated,priority:2;index:idx_taskStatusUser,priority:3" json:"createdAt"`
	Instance    *ProcessInstance  `gorm:"foreignKey:InstanceID;constraint:OnDelete:NO ACTION" json:"-"`
	Node        *Node             `gorm:"foreignKey:NodeID;constraint:OnDelete:NO ACTION" json:"-"`
	User        *User             `gorm:"foreignKey:UserID;constraint:OnDelete:NO ACTION" json:"-"`
}

// BeforeCreate assigns a ULID public identifier.
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	t.UID = uid
	return nil
}

// Event is a history record of a step in an instance.
type Event struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueEventUID;not null" json:"id"`
	ContentType types.ContentType `gorm:"default:17;not null;check:ContentType=17" json:"-"`
	// idx_eventInstance (InstanceID): SQL Server appends clustered ID, matching Progress ORDER BY ID.
	InstanceID uint  `gorm:"index:idx_eventInstance;not null" json:"-"`
	UserID     *uint `gorm:"index:idx_eventUser;index:idx_eventUserActCreated,priority:1" json:"-"`
	// OnBehalfOfUserID is set when a substitute acts for the task owner.
	OnBehalfOfUserID *uint             `gorm:"index:idx_eventOnBehalf" json:"-"`
	OnBehalfOf       *User             `gorm:"foreignKey:OnBehalfOfUserID;constraint:OnDelete:NO ACTION" json:"-"`
	ActType          types.EventAction `gorm:"type:varchar(32);index:idx_eventActType;index:idx_eventUserActCreated,priority:2;not null;check:ActType IN ('initiate','agree','reject','transfer','back','complete')" json:"actType"`
	// ActName is the operator-facing verb (Approved, Returned, Resubmitted, …).
	ActName        string           `gorm:"type:varchar(255);not null;check:chk_event_act_name,[ActName] <> ''" json:"actName"`
	OldNodeID      *uint            `gorm:"index:idx_eventOldNode" json:"-"`
	NewNodeID      *uint            `gorm:"index:idx_eventNewNode" json:"-"`
	Comment        string           `gorm:"type:nvarchar(max)" json:"comment"`
	IPAddress      string           `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`
	TotalRejection bool             `gorm:"default:0;not null" json:"totalRejection"`
	CreatedAt      time.Time        `gorm:"index:idx_eventUserActCreated,priority:3" json:"createdAt"`
	Instance       *ProcessInstance `gorm:"foreignKey:InstanceID;constraint:OnDelete:NO ACTION" json:"-"`
}

// BeforeCreate assigns a ULID public identifier.
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	e.UID = uid
	return nil
}

// InitiatorPool is the set of users who may amend a soft-rejected document when
// the process AmendmentMode is "pool".
type InitiatorPool struct {
	ID          uint              `gorm:"primaryKey" json:"-"`
	ContentType types.ContentType `gorm:"default:18;not null;check:ContentType=18" json:"-"`
	ProcessID   uint              `gorm:"uniqueIndex:idx_uniqueInitiatorPoolProcess;not null" json:"-"`
	Process     Process           `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"-"`
	Users       []User            `gorm:"many2many:WorkflowInitiatorPoolUsers;constraint:OnDelete:NO ACTION" json:"users,omitempty"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// WorkflowInitiatorPoolUser is the initiator-pool ↔ user join.
// PK (InitiatorPoolID, UserID) serves pool membership. idx_initPoolUserByUser
// covers "is this user in any initiator pool?" (user delete / substitute).
type WorkflowInitiatorPoolUser struct {
	InitiatorPoolID uint `gorm:"primaryKey;index:idx_initPoolUserByUser,priority:2"`
	UserID          uint `gorm:"primaryKey;index:idx_initPoolUserByUser,priority:1"`
}

func (WorkflowInitiatorPoolUser) TableName() string { return "WorkflowInitiatorPoolUsers" }

// WorkflowNotifyUser is a process-level FYI watcher. They get mail on the
// event but no inbox task and no Approve/Reject action.
type WorkflowNotifyUser struct {
	ProcessID uint              `gorm:"primaryKey;index:idx_notifyUserByUser,priority:2"`
	Event     types.NotifyEvent `gorm:"primaryKey;type:varchar(16);not null;check:Event IN ('submit','complete','reject')"`
	UserID    uint              `gorm:"primaryKey;index:idx_notifyUserByUser,priority:1"`
}

func (WorkflowNotifyUser) TableName() string { return "WorkflowNotifyUsers" }

// ApprovalSubstitute authorizes a delegate to act on the principal's pending
// approval tasks for one process.
//
// Constraints: unique (PrincipalUserID, ProcessID) so each person has at most
// one substitute per workflow. chk_substitute_not_self blocks self-delegation.
// idx_substituteDelegateProcess (DelegateUserID, ProcessID) serves the inbox
// EXISTS used by MyTasks. NodeID nil = every running step of that process.
//
// Tasks stay on the principal. The engine emails/SMS the delegate when a
// covered operator receives a new task (subject "Substitute · …").
type ApprovalSubstitute struct {
	ID              uint              `gorm:"primaryKey" json:"-"`
	UID             string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueSubstituteUID;not null" json:"id"`
	ContentType     types.ContentType `gorm:"default:19;not null;check:ContentType=19" json:"-"`
	PrincipalUserID uint              `gorm:"not null;uniqueIndex:idx_uniqueSubstitutePrincipalProcess,priority:1;check:chk_substitute_not_self,[PrincipalUserID] <> [DelegateUserID]" json:"-"`
	ProcessID       uint              `gorm:"not null;uniqueIndex:idx_uniqueSubstitutePrincipalProcess,priority:2;index:idx_substituteProcess;index:idx_substituteDelegateProcess,priority:2" json:"-"`
	DelegateUserID  uint              `gorm:"not null;index:idx_substituteDelegateProcess,priority:1" json:"-"`
	NodeID          *uint             `gorm:"index:idx_substituteNode" json:"-"`
	StartsAt        time.Time         `gorm:"not null" json:"startsAt"`
	EndsAt          *time.Time        `json:"endsAt,omitempty"`
	CreatedAt       time.Time         `json:"-"`
	UpdatedAt       time.Time         `json:"-"`
	Principal       *User             `gorm:"foreignKey:PrincipalUserID;constraint:OnDelete:NO ACTION" json:"principal,omitempty"`
	Delegate        *User             `gorm:"foreignKey:DelegateUserID;constraint:OnDelete:NO ACTION" json:"delegate,omitempty"`
	Process         *Process          `gorm:"foreignKey:ProcessID;constraint:OnDelete:NO ACTION" json:"process,omitempty"`
	Node            *Node             `gorm:"foreignKey:NodeID;constraint:OnDelete:NO ACTION" json:"node,omitempty"`
}

// BeforeCreate assigns a ULID public identifier.
func (s *ApprovalSubstitute) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	s.UID = uid
	return nil
}
