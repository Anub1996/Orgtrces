package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	ManagerID uuid.UUID `json:"manager_id"`
	OrgID     uuid.UUID `json:"org_id"`
	Role      string    `json:"role"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

func GenerateToken(managerID, orgID uuid.UUID, role string, expiry time.Duration) (string, error) {
	claims := Claims{
		ManagerID: managerID,
		OrgID:     orgID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization"})
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("manager_id", claims.ManagerID)
		c.Locals("org_id", claims.OrgID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}

func AgentAuth(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-Agent-Key")
		if key == "" {
			key = c.Query("key")
		}
		if key != apiKey {
			return c.Status(401).JSON(fiber.Map{"error": "invalid agent key"})
		}
		return c.Next()
	}
}

func OrgID(c *fiber.Ctx) uuid.UUID {
	return c.Locals("org_id").(uuid.UUID)
}
