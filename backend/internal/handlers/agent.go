package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/Anub1996/orgtraces-backend/internal/database"
	"github.com/Anub1996/orgtraces-backend/internal/models"
	"github.com/Anub1996/orgtraces-backend/internal/ws"
)

type AgentHandler struct {
	Hub *ws.Hub
}

func (h *AgentHandler) Heartbeat(c *fiber.Ctx) error {
	var payload models.HeartbeatPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	orgID := c.Get("X-Org-ID")
	if orgID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing org id"})
	}
	orgUUID, _ := uuid.Parse(orgID)

	ctx := context.Background()

	var employeeID uuid.UUID
	err := database.Pool.QueryRow(ctx,
		`INSERT INTO employees (org_id, machine_id, name, status, last_seen, agent_version)
		 VALUES ($1, $2, $3, $4, NOW(), $5)
		 ON CONFLICT (org_id, machine_id) DO UPDATE
		 SET status = $4, last_seen = NOW(), name = COALESCE(NULLIF($3, ''), employees.name)
		 RETURNING id`,
		orgUUID, payload.MachineID, payload.EmployeeName, payload.Status, c.Get("X-Agent-Version"),
	).Scan(&employeeID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "upsert employee failed"})
	}

	if payload.ActiveApp != "" {
		database.Pool.Exec(ctx,
			`INSERT INTO activities (employee_id, type, app_name, window_title, url, duration_sec, timestamp)
			 VALUES ($1, 'app_switch', $2, $3, $4, $5, $6)`,
			employeeID, payload.ActiveApp, payload.WindowTitle, payload.ActiveURL, payload.IdleSecs, payload.Timestamp,
		)
	}

	h.Hub.Broadcast(orgUUID.String(), ws.Message{
		Type: "heartbeat",
		Data: map[string]interface{}{
			"employee_id": employeeID,
			"machine_id":  payload.MachineID,
			"name":        payload.EmployeeName,
			"status":      payload.Status,
			"active_app":  payload.ActiveApp,
			"title":       payload.WindowTitle,
			"idle_secs":   payload.IdleSecs,
			"timestamp":   payload.Timestamp,
		},
	})

	return c.JSON(fiber.Map{"ok": true, "employee_id": employeeID})
}

func (h *AgentHandler) Screenshot(c *fiber.Ctx) error {
	employeeID, _ := uuid.Parse(c.FormValue("employee_id"))
	sessionID, _ := uuid.Parse(c.FormValue("session_id"))

	file, err := c.FormFile("screenshot")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "no file uploaded"})
	}

	storageKey := fmt.Sprintf("screenshots/%s/%s/%s.jpg",
		employeeID.String(),
		time.Now().Format("2006-01-02"),
		uuid.New().String(),
	)

	// TODO: Upload file to S3 using storageKey
	// For now, save metadata only
	_ = file

	database.Pool.Exec(context.Background(),
		`INSERT INTO screenshots (employee_id, session_id, storage_key, file_size, captured_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		employeeID, sessionID, storageKey, file.Size,
	)

	return c.JSON(fiber.Map{"ok": true, "key": storageKey})
}

func (h *AgentHandler) SessionStart(c *fiber.Ctx) error {
	var body struct {
		EmployeeID string `json:"employee_id"`
		IPAddress  string `json:"ip_address"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}
	empID, _ := uuid.Parse(body.EmployeeID)

	var sessionID uuid.UUID
	err := database.Pool.QueryRow(context.Background(),
		`INSERT INTO sessions (employee_id, ip_address) VALUES ($1, $2) RETURNING id`,
		empID, body.IPAddress,
	).Scan(&sessionID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "session creation failed"})
	}
	return c.JSON(fiber.Map{"session_id": sessionID})
}

func (h *AgentHandler) SessionEnd(c *fiber.Ctx) error {
	sessionID, _ := uuid.Parse(c.Params("id"))

	var body struct {
		ActiveTime int `json:"active_time_sec"`
		IdleTime   int `json:"idle_time_sec"`
		BreakTime  int `json:"break_time_sec"`
		LockCount  int `json:"lock_count"`
	}
	c.BodyParser(&body)

	database.Pool.Exec(context.Background(),
		`UPDATE sessions SET logout_at = NOW(), active_time_sec = $2, idle_time_sec = $3, break_time_sec = $4, lock_count = $5
		 WHERE id = $1`,
		sessionID, body.ActiveTime, body.IdleTime, body.BreakTime, body.LockCount,
	)

	return c.JSON(fiber.Map{"ok": true})
}

func (h *AgentHandler) Alert(c *fiber.Ctx) error {
	var alert models.Alert
	if err := c.BodyParser(&alert); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	database.Pool.Exec(context.Background(),
		`INSERT INTO alerts (org_id, employee_id, type, severity, message, data)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		alert.OrgID, alert.EmployeeID, alert.Type, alert.Severity, alert.Message, alert.Data,
	)

	h.Hub.Broadcast(alert.OrgID.String(), ws.Message{
		Type: "alert",
		Data: map[string]interface{}{
			"employee_id": alert.EmployeeID,
			"type":        alert.Type,
			"severity":    alert.Severity,
			"message":     alert.Message,
		},
	})

	return c.JSON(fiber.Map{"ok": true})
}

func (h *AgentHandler) DLPEvent(c *fiber.Ctx) error {
	var evt models.DLPEvent
	if err := c.BodyParser(&evt); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	database.Pool.Exec(context.Background(),
		`INSERT INTO dlp_events (employee_id, type, details, blocked) VALUES ($1, $2, $3, $4)`,
		evt.EmployeeID, evt.Type, evt.Details, evt.Blocked,
	)

	return c.JSON(fiber.Map{"ok": true})
}
