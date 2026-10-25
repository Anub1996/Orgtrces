package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberws "github.com/gofiber/contrib/websocket"

	"github.com/Anub1996/orgtraces-backend/internal/config"
	"github.com/Anub1996/orgtraces-backend/internal/database"
	"github.com/Anub1996/orgtraces-backend/internal/handlers"
	"github.com/Anub1996/orgtraces-backend/internal/middleware"
	"github.com/Anub1996/orgtraces-backend/internal/ws"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg.DSN()); err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer database.Close()

	middleware.SetJWTSecret(cfg.JWT.Secret)

	hub := ws.NewHub()

	app := fiber.New(fiber.Config{
		BodyLimit:    10 * 1024 * 1024, // 10MB for screenshot uploads
		ServerHeader: "Orgtraces",
		AppName:      "Orgtraces API v1",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Authorization, X-Agent-Key, X-Org-ID, X-Agent-Version",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":     "ok",
			"service":    "orgtraces-api",
			"ws_clients": hub.ClientCount(),
		})
	})

	// Auth routes
	authH := &handlers.AuthHandler{JWTExpiry: cfg.JWT.Expiry}
	auth := app.Group("/api/auth")
	auth.Post("/login", authH.Login)
	auth.Get("/me", middleware.AuthRequired(), authH.Me)

	// Agent routes (authenticated by API key)
	agentH := &handlers.AgentHandler{Hub: hub}
	agent := app.Group("/api/agent", middleware.AgentAuth(cfg.AgentKey))
	agent.Post("/heartbeat", agentH.Heartbeat)
	agent.Post("/screenshot", agentH.Screenshot)
	agent.Post("/session/start", agentH.SessionStart)
	agent.Put("/session/:id/end", agentH.SessionEnd)
	agent.Post("/alert", agentH.Alert)
	agent.Post("/dlp-event", agentH.DLPEvent)

	// Dashboard routes (authenticated by JWT)
	dashH := &handlers.DashboardHandler{}
	dash := app.Group("/api/dashboard", middleware.AuthRequired())
	dash.Get("/stats", dashH.Stats)
	dash.Get("/employees", dashH.Employees)
	dash.Get("/employees/:id/activity", dashH.EmployeeActivity)
	dash.Get("/employees/:id/screenshots", dashH.EmployeeScreenshots)
	dash.Get("/alerts", dashH.Alerts)
	dash.Put("/alerts/:id/resolve", dashH.ResolveAlert)

	// WebSocket for real-time dashboard updates
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/dashboard", fiberws.New(func(c *fiberws.Conn) {
		orgID := c.Query("org_id")
		if orgID == "" {
			c.Close()
			return
		}
		hub.Register(c, orgID)
	}))

	// Serve dashboard static files (production)
	app.Static("/dashboard", "./dashboard/dist")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down...")
		app.Shutdown()
	}()

	log.Printf("Orgtraces API starting on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
