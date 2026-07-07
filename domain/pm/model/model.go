package model

// Row is a dynamic query result row, mirroring the Java service's Map<String,Object>
// responses for the many ad-hoc list/detail/aggregate queries in this module.
// Declared as an alias (not a distinct named type) so that []model.Row is
// identical to []map[string]interface{}, which GORM's Scan special-cases for
// raw query results — a distinct named type falls through to reflection-based
// struct scanning and fails with "destination not a pointer".
type Row = map[string]interface{}

type TaskRequest struct {
	Title         string  `json:"title" binding:"required"`
	Description   *string `json:"description"`
	ProjectID     *int64  `json:"project_id"`
	AssignedTo    *string `json:"assigned_to"`
	Deadline      *string `json:"deadline"`
	Labels        *string `json:"labels"`
	Points        *int    `json:"points"`
	Status        *string `json:"status"`
	StatusID      *int64  `json:"status_id"`
	PriorityID    *int64  `json:"priority_id"`
	StartDate     *string `json:"start_date"`
	ParentTaskID  *int64  `json:"parent_task_id"`
	Collaborators *string `json:"collaborators"`
	SortOrder     *int    `json:"sort_order"`
	Context       *string `json:"context"`
}

type MoveTaskStatusRequest struct {
	StatusID int64 `json:"status_id" binding:"required"`
}

type MoveTaskKeyRequest struct {
	StatusKey string `json:"status_key" binding:"required"`
}

type TicketRequest struct {
	Title               string  `json:"title" binding:"required"`
	ClientID            *int64  `json:"client_id"`
	ProjectID           *int64  `json:"project_id"`
	TicketTypeID        *int64  `json:"ticket_type_id"`
	RequestedBy         *string `json:"requested_by"`
	Status              *string `json:"status"`
	AssignedTo          *string `json:"assigned_to"`
	CreatorName         *string `json:"creator_name"`
	CreatorEmail        *string `json:"creator_email"`
	Labels              *string `json:"labels"`
	TaskID              *int64  `json:"task_id"`
	CcContactsAndEmails *string `json:"cc_contacts_and_emails"`
}

type TicketCommentRequest struct {
	Description string  `json:"description" binding:"required"`
	Files       *string `json:"files"`
	IsNote      bool    `json:"is_note"`
}
