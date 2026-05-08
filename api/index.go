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
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))

	api := app.Group("/api")
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/logout", authHandler.Logout)

	api.Get("/schema", schemaHandler.GetSchema)
	
	// CRUD
	jwtMid := middleware.JWTAuth(cfg.JWTSecret)
	api.Get("/tables/:table", jwtMid, crudHandler.List)
	api.Get("/tables/:table/:id", jwtMid, crudHandler.Get)
	api.Post("/tables/:table", jwtMid, crudHandler.Create)
	api.Put("/tables/:table/:id", jwtMid, crudHandler.Update)
	api.Delete("/tables/:table/:id", jwtMid, crudHandler.Delete)

	// Student Workspace
	api.Get("/student/me", jwtMid, studentHandler.GetWorkspace)
	api.Get("/student/requests", jwtMid, studentHandler.ListRequests)
	api.Post("/student/requests", jwtMid, studentHandler.CreateRequest)

	// Admin Review
	adminMid := middleware.RequireAdmin()
	api.Get("/admin/change-requests", jwtMid, adminMid, adminReviewHandler.ListRequests)
	api.Post("/admin/change-requests/:id/approve", jwtMid, adminMid, adminReviewHandler.ApproveRequest)
	api.Post("/admin/change-requests/:id/reject", jwtMid, adminMid, adminReviewHandler.RejectRequest)
	api.Get("/admin/users", jwtMid, adminMid, accessHandler.ListUsers)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	adaptor.FiberApp(app)(w, r)
}
