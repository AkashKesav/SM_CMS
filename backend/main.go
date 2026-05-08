package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"cms-backend/internal/config"
	"cms-backend/internal/handler"
	"cms-backend/internal/middleware"
	"cms-backend/internal/repository"
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database connection (optional for dev mode)
	var db *repository.DBWrapper
	if cfg.DatabaseURL != "" {
		db, err = repository.NewPostgresDB(cfg.DatabaseURL)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to database — starting in demo mode")
			db = repository.NewInMemoryDB()
		}
	} else {
		log.Info().Msg("No DATABASE_URL configured — starting in demo mode")
		db = repository.NewInMemoryDB()
	}
	defer db.Close()

	// Ensure audit_log table exists in production mode
	if !db.IsInMemory() {
		if err := repository.EnsureAuditTable(db.Pool()); err != nil {
			log.Warn().Err(err).Msg("Failed to create audit_log table — audit logging disabled")
		}
		if err := repository.EnsureStudentAccessSchema(db.Pool()); err != nil {
			log.Warn().Err(err).Msg("Failed to ensure student access schema")
		}
	}

	// Initialize audit logger
	auditLogger := repository.NewAuditLogger(db)
	defer auditLogger.Close()

	// Initialize services
	schemaService := service.NewSchemaService(db)
	crudService := service.NewCrudService()
	authService := service.NewAuthService(cfg)

	// Initialize repositories
	schemaRepo := repository.NewSchemaRepository(db)
	crudRepo := repository.NewCrudRepository(db)
	accessRepo := repository.NewStudentAccessRepository(db)

	// Wire schema service with its repo
	schemaService.SetRepo(schemaRepo)

	// Pre-load schema
	if _, err := schemaService.GetSchema(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to pre-load schema — will load on first request")
	}

	// Initialize handlers
	schemaHandler := handler.NewSchemaHandler(schemaService)
	crudHandler := handler.NewCrudHandler(crudService, crudRepo, schemaService)
	authHandler := handler.NewAuthHandler(authService, accessRepo)
	studentHandler := handler.NewStudentHandler(accessRepo)
	adminReviewHandler := handler.NewAdminReviewHandler(accessRepo)
	accessHandler := handler.NewAccessHandler(accessRepo, authService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit:    50 * 1024 * 1024, // 50MB for file uploads
		ErrorHandler: middleware.ErrorHandler,
	})

	// Global middleware
	app.Use(logger.New(logger.Config{
		Format: "${pid} | ${status} | ${latency} | ${method} | ${path}\n",
	}))
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: allowLocalOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	// Rate limiting on auth endpoints
	authLimiter := limiter.New(limiter.Config{
		Max:               10,
		Expiration:        60 * 1e9, // 1 minute
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		},
	})

	// General API rate limiting
	apiLimiter := limiter.New(limiter.Config{
		Max:               100,
		Expiration:        60 * 1e9, // 1 minute
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		},
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "message": "CMS Backend is running", "mode": db.Mode()})
	})

	// Auth routes (public, rate limited)
	auth := app.Group("/api/auth", authLimiter)
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/logout", authHandler.Logout)

	// Schema discovery route (authenticated)
	api := app.Group("/api", middleware.JWTAuth(cfg.JWTSecret), middleware.LoadProfile(accessRepo), apiLimiter)
	api.Get("/me", authHandler.Me)
	api.Post("/password", authHandler.ChangePassword)
	api.Get("/schema", middleware.RequireAdmin(), schemaHandler.GetSchema)
	api.Post("/schema/refresh", middleware.RequireAdmin(), schemaHandler.RefreshSchema)

	student := api.Group("/student")
	student.Get("/me", studentHandler.Me)
	student.Get("/requests", studentHandler.ListRequests)
	student.Post("/requests", studentHandler.CreateRequest)
	// Contributor management
	student.Post("/achievements/contributors", studentHandler.AddAchievementContributor)
	student.Delete("/achievements/contributors", studentHandler.RemoveAchievementContributor)
	student.Post("/projects/contributors", studentHandler.AddProjectContributor)
	student.Delete("/projects/contributors", studentHandler.RemoveProjectContributor)
	student.Get("/contributors", studentHandler.GetContributors)
	student.Get("/search", studentHandler.SearchStudents)

	admin := api.Group("/admin", middleware.RequireAdmin())
	admin.Get("/change-requests", adminReviewHandler.ListChangeRequests)
	admin.Post("/change-requests/:id/approve", adminReviewHandler.Approve)
	admin.Post("/change-requests/:id/reject", adminReviewHandler.Reject)
	admin.Get("/users", accessHandler.ListUsers)
	admin.Post("/users/provision", accessHandler.ProvisionStudents)
	admin.Post("/users/:id/role", accessHandler.SetRole)
	admin.Post("/users/:id/password", accessHandler.ResetPassword)

	// Generic CRUD routes (authenticated)
	tables := api.Group("/tables", middleware.RequireAdmin())
	tables.Get("/:tableName", crudHandler.ListRecords)
	tables.Get("/:tableName/:id", crudHandler.GetRecord)
	tables.Post("/:tableName", crudHandler.CreateRecord)
	tables.Put("/:tableName/:id", crudHandler.UpdateRecord)
	tables.Patch("/:tableName/:id", crudHandler.UpdateRecord)
	tables.Delete("/:tableName/:id", crudHandler.DeleteRecord)

	// File upload routes
	api.Post("/upload/sign", authHandler.GetUploadSignedURL)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info().Msg("Shutting down server...")

		if err := app.ShutdownWithTimeout(10 * 1e9); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	}()

	log.Info().Str("port", cfg.BackendPort).Str("mode", db.Mode()).Msg("Starting CMS Backend")
	if err := app.Listen(":" + cfg.BackendPort); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}

	log.Info().Msg("Server stopped gracefully")
}

func allowLocalOrigins(origin string) bool {
	if origin == "" {
		return true
	}
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		origin == "http://tauri.localhost" ||
		origin == "https://tauri.localhost" ||
		origin == "tauri://localhost"
}
