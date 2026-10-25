package models

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	Plan      string    `json:"plan"`
	MaxUsers  int       `json:"max_users"`
	CreatedAt time.Time `json:"created_at"`
}

type Manager struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"`
	Role      string    `json:"role"` // admin, manager, viewer
	CreatedAt time.Time `json:"created_at"`
}

type Employee struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	MachineID    string     `json:"machine_id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Department   string     `json:"department"`
	AgentVersion string     `json:"agent_version"`
	Status       string     `json:"status"` // online, idle, offline, locked
	LastSeen     *time.Time `json:"last_seen"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Session struct {
	ID         uuid.UUID  `json:"id"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	LoginAt    time.Time  `json:"login_at"`
	LogoutAt   *time.Time `json:"logout_at"`
	ActiveTime int        `json:"active_time_sec"`
	IdleTime   int        `json:"idle_time_sec"`
	BreakTime  int        `json:"break_time_sec"`
	LockCount  int        `json:"lock_count"`
	IPAddress  string     `json:"ip_address"`
}

type Activity struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	SessionID  uuid.UUID `json:"session_id"`
	Type       string    `json:"type"` // app_switch, url_visit, file_access, keystroke_count, mouse_activity
	AppName    string    `json:"app_name,omitempty"`
	WindowTitle string   `json:"window_title,omitempty"`
	URL        string    `json:"url,omitempty"`
	Duration   int       `json:"duration_sec"`
	Timestamp  time.Time `json:"timestamp"`
}

type Screenshot struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	SessionID  uuid.UUID `json:"session_id"`
	StorageKey string    `json:"storage_key"`
	FileSize   int64     `json:"file_size"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	CapturedAt time.Time `json:"captured_at"`
}

type Alert struct {
	ID         uuid.UUID `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Type       string    `json:"type"` // idle_exceeded, no_person, multiple_persons, phone_detected, dlp_violation, unauthorized_app
	Severity   string    `json:"severity"` // low, medium, high, critical
	Message    string    `json:"message"`
	Data       string    `json:"data,omitempty"` // JSON metadata
	Resolved   bool      `json:"resolved"`
	CreatedAt  time.Time `json:"created_at"`
}

type WellnessLog struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Type       string    `json:"type"` // break_reminder, break_taken, break_skipped
	WorkedMins int       `json:"worked_mins"`
	BreakMins  int       `json:"break_mins"`
	Timestamp  time.Time `json:"timestamp"`
}

type DLPEvent struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Type       string    `json:"type"` // clipboard_blocked, printscreen_blocked, file_transfer_blocked
	Details    string    `json:"details"`
	Blocked    bool      `json:"blocked"`
	Timestamp  time.Time `json:"timestamp"`
}

// --- API request/response types ---

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string  `json:"token"`
	Manager Manager `json:"manager"`
}

type HeartbeatPayload struct {
	MachineID    string     `json:"machine_id"`
	EmployeeName string     `json:"employee_name"`
	Status       string     `json:"status"`
	ActiveApp    string     `json:"active_app"`
	WindowTitle  string     `json:"window_title"`
	ActiveURL    string     `json:"active_url,omitempty"`
	IdleSecs     int        `json:"idle_secs"`
	CPUUsage     float64    `json:"cpu_usage"`
	RAMUsage     float64    `json:"ram_usage"`
	Timestamp    time.Time  `json:"timestamp"`
}

type DashboardStats struct {
	TotalEmployees  int     `json:"total_employees"`
	OnlineNow       int     `json:"online_now"`
	IdleNow         int     `json:"idle_now"`
	AvgProductivity float64 `json:"avg_productivity"`
	AlertsToday     int     `json:"alerts_today"`
	ComplianceRate  float64 `json:"compliance_rate"`
}

type EmployeeDetail struct {
	Employee
	CurrentApp    string  `json:"current_app"`
	CurrentTitle  string  `json:"current_title"`
	TodayActive   int     `json:"today_active_min"`
	TodayIdle     int     `json:"today_idle_min"`
	TodayBreaks   int     `json:"today_breaks"`
	Productivity  float64 `json:"productivity_pct"`
}
