//go:build ignore

package handler

import (
	"net/http"

	"cms-backend/internal/config"
	"cms-backend/internal/handler"
	"cms-backend/internal/middleware"
	"cms-backend/internal/repository"
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var app *fiber.App

func init() {
	cfg, err := config.Load()
	if err != nil {
		return
	}

	db, _ := repository.NewPostgresDB(cfg.DatabaseURL)
	if db == nil {
		db = repository.NewInMemoryDB()
	}

	schemaService := service.NewSchemaService(db)
	crudService := service.NewCrudService()
	authService := service.NewAuthService(cfg)
	accessRepo := repository.NewStudentAccessRepository(db)
	crudRepo := repository.NewCrudRepository(db)
	schemaRepo := repository.NewSchemaRepository(db)
	schemaService.SetRepo(schemaRepo)

	schemaHandler := handler.NewSchemaHandler(schemaService)
	crudHandler := handler.NewCrudHandler(crudService, crudRepo, schemaService)
	authHandler := handler.NewAuthHandler(authService, accessRepo)
	studentHandler := handler.NewStudentHandler(accessRepo)
	adminReviewHandler := handler.NewAdminReviewHandler(accessRepo)
	accessHandler := handler.NewAccessHandler(accessRepo, authService)

	app = fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true // Allow all origins in serverless mode
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))

	// Health check
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "message": "CMS Backend is running", "mode": db.Mode()})
	})

	// Auth routes (public)
	api := app.Group("/api")
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/logout", authHandler.Logout)

	// Authenticated routes
	jwtMid := middleware.JWTAuth(cfg.JWTSecret)
	profileMid := middleware.LoadProfile(accessRepo)
	authenticated := api.Group("", jwtMid, profileMid)

	authenticated.Get("/me", authHandler.Me)
	authenticated.Post("/password", authHandler.ChangePassword)
	authenticated.Get("/schema", middleware.RequireAdmin(), schemaHandler.GetSchema)
	authenticated.Post("/schema/refresh", middleware.RequireAdmin(), schemaHandler.RefreshSchema)

	// Student routes
	student := authenticated.Group("/student")
	student.Get("/me", studentHandler.Me)
	student.Get("/requests", studentHandler.ListRequests)
	student.Post("/requests", studentHandler.CreateRequest)
	student.Post("/achievements/contributors", studentHandler.AddAchievementContributor)
	student.Delete("/achievements/contributors", studentHandler.RemoveAchievementContributor)
	student.Post("/projects/contributors", studentHandler.AddProjectContributor)
	student.Delete("/projects/contributors", studentHandler.RemoveProjectContributor)
	student.Get("/contributors", studentHandler.GetContributors)
	student.Get("/search", studentHandler.SearchStudents)

	// Admin routes
	adminMid := middleware.RequireAdmin()
	admin := authenticated.Group("/admin", adminMid)
	admin.Get("/change-requests", adminReviewHandler.ListChangeRequests)
	admin.Post("/change-requests/:id/approve", adminReviewHandler.Approve)
	admin.Post("/change-requests/:id/reject", adminReviewHandler.Reject)
	admin.Get("/users", accessHandler.ListUsers)
	admin.Post("/users/provision", accessHandler.ProvisionStudents)
	admin.Post("/users/:id/role", accessHandler.SetRole)
	admin.Post("/users/:id/password", accessHandler.ResetPassword)

	// Generic CRUD routes (admin only)
	tables := authenticated.Group("/tables", adminMid)
	tables.Get("/:tableName", crudHandler.ListRecords)
	tables.Get("/:tableName/:id", crudHandler.GetRecord)
	tables.Post("/:tableName", crudHandler.CreateRecord)
	tables.Put("/:tableName/:id", crudHandler.UpdateRecord)
	tables.Patch("/:tableName/:id", crudHandler.UpdateRecord)
	tables.Delete("/:tableName/:id", crudHandler.DeleteRecord)

	// File upload
	authenticated.Post("/upload/sign", authHandler.GetUploadSignedURL)
}

// Handler is the Vercel serverless entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	adaptor.FiberApp(app)(w, r)
}
