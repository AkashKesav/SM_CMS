package model

type UserProfile struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Name      string  `json:"name,omitempty"`
	StudentID *string `json:"student_id,omitempty"`
}

type StudentChangeRequest struct {
	ID              string                 `json:"id"`
	RequesterUserID string                 `json:"requester_user_id"`
	RequesterName   string                 `json:"requester_name,omitempty"`
	RequesterRollNo string                 `json:"requester_roll_no,omitempty"`
	StudentID       *string                `json:"student_id,omitempty"`
	RequestType     string                 `json:"request_type"`
	TargetTable     string                 `json:"target_table"`
	TargetKey       map[string]interface{} `json:"target_key,omitempty"`
	CurrentData     map[string]interface{} `json:"current_data,omitempty"`
	ProposedData    map[string]interface{} `json:"proposed_data"`
	Status          string                 `json:"status"`
	AdminNote       string                 `json:"admin_note,omitempty"`
	ReviewedBy      *string                `json:"reviewed_by,omitempty"`
	ReviewedAt      *string                `json:"reviewed_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type StudentWorkspace struct {
	Profile       UserProfile                         `json:"profile"`
	Student       map[string]interface{}              `json:"student,omitempty"`
	LinkedRecords map[string][]map[string]interface{} `json:"linked_records"`
	References    map[string][]map[string]interface{} `json:"references"`
	Requests      []StudentChangeRequest              `json:"requests"`
}

type AccessUser struct {
	UserID     string `json:"user_id,omitempty"`
	StudentID  string `json:"student_id"`
	RollNo     string `json:"roll_no"`
	Name       string `json:"name"`
	LoginEmail string `json:"login_email"`
	Role       string `json:"role"`
	HasAccount bool   `json:"has_account"`
}

type ProvisionUserResult struct {
	StudentID  string `json:"student_id"`
	RollNo     string `json:"roll_no"`
	LoginEmail string `json:"login_email"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	UserID     string `json:"user_id,omitempty"`
}

type ProvisionResult struct {
	Created int                   `json:"created"`
	Updated int                   `json:"updated"`
	Failed  int                   `json:"failed"`
	Users   []ProvisionUserResult `json:"users"`
}
