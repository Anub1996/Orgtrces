package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/Anub1996/orgtraces-backend/internal/database"
	"github.com/Anub1996/orgtraces-backend/internal/middleware"
	"github.com/Anub1996/orgtraces-backend/internal/models"
)

type DashboardHandler struct{}

func (h *DashboardHandler) Stats(c *fiber.Ctx) error {
	orgID := middleware.OrgID(c)
	ctx := context.Background()

	var stats models.DashboardStats

	database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM employees WHERE org_id = $1`, orgID,
	).Scan(&stats.TotalEmployees)

	database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM employees WHERE org_id = $1 AND status = 'online'`, orgID,
	).Scan(&stats.OnlineNow)

	database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM employees WHERE org_id = $1 AND status = 'idle'`, orgID,
	).Scan(&stats.IdleNow)

	database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts WHERE org_id = $1 AND created_at >= CURRENT_DATE AND NOT resolved`, orgID,
	).Scan(&stats.AlertsToday)

	database.Pool.QueryRow(ctx,
		`SELECT COALESCE(
			ROUND(100.0 * COUNT(*) FILTER (WHERE status IN ('online','idle')) / NULLIF(COUNT(*), 0), 1),
		0) FROM employees WHERE org_id = $1`, orgID,
	).Scan(&stats.ComplianceRate)

	database.Pool.QueryRow(ctx,
		`SELECT COALESCE(
			ROUND(AVG(active_time_sec)::numeric / NULLIF(AVG(active_time_sec + idle_time_sec), 0) * 100, 1),
		0) FROM sessions s
		JOIN employees e ON e.id = s.employee_id
		WHERE e.org_id = $1 AND s.login_at >= CURRENT_DATE`, orgID,
	).Scan(&stats.AvgProductivity)

	return c.JSON(stats)
}

func (h *DashboardHandler) Employees(c *fiber.Ctx) error {
	orgID := middleware.OrgID(c)
	dept := c.Query("department")
	status := c.Query("status")
	search := c.Query("search")

	query := `
		SELECT e.id, e.org_id, e.machine_id, e.name, e.email, e.department,
			   e.agent_version, e.status, e.last_seen, e.created_at,
			   COALESCE(a.app_name, '') as current_app,
			   COALESCE(a.window_title, '') as current_title,
			   COALESCE(s.today_active, 0) as today_active_min,
			   COALESCE(s.today_idle, 0) as today_idle_min
		FROM employees e
		LEFT JOIN LATERAL (
			SELECT app_name, window_title FROM activities
			WHERE employee_id = e.id ORDER BY timestamp DESC LIMIT 1
		) a ON true
		LEFT JOIN LATERAL (
			SELECT SUM(active_time_sec)/60 as today_active, SUM(idle_time_sec)/60 as today_idle
			FROM sessions WHERE employee_id = e.id AND login_at >= CURRENT_DATE
		) s ON true
		WHERE e.org_id = $1`

	args := []interface{}{orgID}
	argN := 2

	if dept != "" {
		query += fmt.Sprintf(" AND e.department = $%d", argN)
		args = append(args, dept)
		argN++
	}
	if status != "" {
		query += fmt.Sprintf(" AND e.status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if search != "" {
		query += fmt.Sprintf(" AND (e.name ILIKE $%d OR e.email ILIKE $%d)", argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}

	query += " ORDER BY e.status = 'online' DESC, e.last_seen DESC NULLS LAST LIMIT 200"

	rows, err := database.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	employees := []models.EmployeeDetail{}
	for rows.Next() {
		var emp models.EmployeeDetail
		rows.Scan(
			&emp.ID, &emp.OrgID, &emp.MachineID, &emp.Name, &emp.Email, &emp.Department,
			&emp.AgentVersion, &emp.Status, &emp.LastSeen, &emp.CreatedAt,
			&emp.CurrentApp, &emp.CurrentTitle,
			&emp.TodayActive, &emp.TodayIdle,
		)
		if emp.TodayActive+emp.TodayIdle > 0 {
			emp.Productivity = float64(emp.TodayActive) / float64(emp.TodayActive+emp.TodayIdle) * 100
		}
		employees = append(employees, emp)
	}
	return c.JSON(employees)
}

func (h *DashboardHandler) EmployeeActivity(c *fiber.Ctx) error {
	employeeID, _ := uuid.Parse(c.Params("id"))
	date := c.Query("date", "today")

	dateFilter := "CURRENT_DATE"
	if date != "today" {
		dateFilter = "'" + date + "'::date"
	}

	rows, err := database.Pool.Query(context.Background(),
		`SELECT id, employee_id, session_id, type, app_name, window_title, url, duration_sec, timestamp
		 FROM activities
		 WHERE employee_id = $1 AND timestamp >= `+dateFilter+`
		 ORDER BY timestamp DESC LIMIT 500`,
		employeeID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	activities := []models.Activity{}
	for rows.Next() {
		var a models.Activity
		rows.Scan(&a.ID, &a.EmployeeID, &a.SessionID, &a.Type, &a.AppName, &a.WindowTitle, &a.URL, &a.Duration, &a.Timestamp)
		activities = append(activities, a)
	}
	return c.JSON(activities)
}

func (h *DashboardHandler) EmployeeScreenshots(c *fiber.Ctx) error {
	employeeID, _ := uuid.Parse(c.Params("id"))

	rows, err := database.Pool.Query(context.Background(),
		`SELECT id, employee_id, session_id, storage_key, file_size, width, height, captured_at
		 FROM screenshots
		 WHERE employee_id = $1 AND captured_at >= CURRENT_DATE
		 ORDER BY captured_at DESC LIMIT 100`,
		employeeID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	screenshots := []models.Screenshot{}
	for rows.Next() {
		var s models.Screenshot
		rows.Scan(&s.ID, &s.EmployeeID, &s.SessionID, &s.StorageKey, &s.FileSize, &s.Width, &s.Height, &s.CapturedAt)
		screenshots = append(screenshots, s)
	}
	return c.JSON(screenshots)
}

func (h *DashboardHandler) Alerts(c *fiber.Ctx) error {
	orgID := middleware.OrgID(c)
	resolved := c.Query("resolved", "false")

	rows, err := database.Pool.Query(context.Background(),
		`SELECT id, org_id, employee_id, type, severity, message, data, resolved, created_at
		 FROM alerts WHERE org_id = $1 AND resolved = $2
		 ORDER BY created_at DESC LIMIT 100`,
		orgID, resolved == "true",
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "query failed"})
	}
	defer rows.Close()

	alerts := []models.Alert{}
	for rows.Next() {
		var a models.Alert
		rows.Scan(&a.ID, &a.OrgID, &a.EmployeeID, &a.Type, &a.Severity, &a.Message, &a.Data, &a.Resolved, &a.CreatedAt)
		alerts = append(alerts, a)
	}
	return c.JSON(alerts)
}

func (h *DashboardHandler) ResolveAlert(c *fiber.Ctx) error {
	alertID, _ := uuid.Parse(c.Params("id"))
	managerID := c.Locals("manager_id").(uuid.UUID)

	database.Pool.Exec(context.Background(),
		`UPDATE alerts SET resolved = true, resolved_by = $2, resolved_at = NOW() WHERE id = $1`,
		alertID, managerID,
	)
	return c.JSON(fiber.Map{"ok": true})
}
