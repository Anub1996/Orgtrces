package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/Anub1996/orgtraces-backend/internal/database"
	"github.com/Anub1996/orgtraces-backend/internal/middleware"
	"github.com/Anub1996/orgtraces-backend/internal/models"
	"github.com/google/uuid"
)

type AuthHandler struct {
	JWTExpiry time.Duration
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	var mgr models.Manager
	var passwordHash string
	err := database.Pool.QueryRow(context.Background(),
		`SELECT id, org_id, email, name, password_hash, role
		 FROM managers WHERE email = $1 AND is_active = true`,
		req.Email,
	).Scan(&mgr.ID, &mgr.OrgID, &mgr.Email, &mgr.Name, &passwordHash, &mgr.Role)

	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	var valid bool
	database.Pool.QueryRow(context.Background(),
		`SELECT $1 = crypt($2, $1)`, passwordHash, req.Password,
	).Scan(&valid)

	if !valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, err := middleware.GenerateToken(mgr.ID, mgr.OrgID, mgr.Role, h.JWTExpiry)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "token generation failed"})
	}

	database.Pool.Exec(context.Background(),
		`UPDATE managers SET last_login = NOW() WHERE id = $1`, mgr.ID)

	return c.JSON(models.LoginResponse{Token: token, Manager: mgr})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	managerID := c.Locals("manager_id").(uuid.UUID)

	var mgr models.Manager
	err := database.Pool.QueryRow(context.Background(),
		`SELECT id, org_id, email, name, role, created_at
		 FROM managers WHERE id = $1`,
		managerID,
	).Scan(&mgr.ID, &mgr.OrgID, &mgr.Email, &mgr.Name, &mgr.Role, &mgr.CreatedAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "manager not found"})
	}
	return c.JSON(mgr)
}
