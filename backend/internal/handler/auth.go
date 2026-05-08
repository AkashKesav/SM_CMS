package handler

import (
	"fmt"
	"os"
	"strings"

	"cms-backend/internal/model"
	"cms-backend/internal/repository"
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type AuthHandler struct {
	authService *service.AuthService
	accessRepo  *repository.StudentAccessRepository
	demoMode    bool
}

func NewAuthHandler(authService *service.AuthService, accessRepo *repository.StudentAccessRepository) *AuthHandler {
	demoMode := os.Getenv("DATABASE_URL") == "" || os.Getenv("SUPABASE_URL") == ""
	return &AuthHandler{
		authService: authService,
		accessRepo:  accessRepo,
		demoMode:    demoMode,
	}
}

// isDemoMode returns true when no Supabase URL is configured
func (h *AuthHandler) isDemo() bool {
	return h.demoMode || os.Getenv("SUPABASE_URL") == ""
}

func demoAuthEnabled() bool {
	switch strings.ToLower(os.Getenv("ALLOW_DEMO_AUTH")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// mockToken generates a simple mock JWT-compatible token for demo mode
func mockToken(email string) string {
	// Simple mock token — the JWT middleware in demo mode accepts any non-empty token
	return fmt.Sprintf("demo-token-%s", email)
}

// Login handles user login via Supabase Auth (or demo mock)
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Login identifier and password are required",
		})
	}

	// Demo mode: accept any credentials
	if h.isDemo() {
		log.Info().Str("email", req.Email).Msg("Login (demo mode)")
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"access_token":  mockToken(req.Email),
				"refresh_token": "demo-refresh-token",
				"user": fiber.Map{
					"id":    "demo-user-id",
					"email": req.Email,
					"role":  "admin",
					"name":  "",
				},
			},
		})
	}

	loginEmail := req.Email
	if h.accessRepo != nil {
		resolved, err := h.accessRepo.ResolveLoginIdentifier(c.Context(), req.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Unable to resolve login identifier",
				"details": err.Error(),
			})
		}
		loginEmail = resolved
	}

	// Production: proxy to Supabase
	result, err := h.authService.Login(loginEmail, req.Password)
	if err != nil {
		log.Warn().Err(err).Str("identifier", req.Email).Msg("Login failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid credentials",
			"details": err.Error(),
		})
	}

	userData, _ := result["user"].(map[string]interface{})
	accessToken, _ := result["access_token"].(string)
	refreshToken, _ := result["refresh_token"].(string)
	user := normalizeUser(userData)
	user = h.enrichUser(c, user)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user":          user,
		},
	})
}

// Register handles user registration via Supabase Auth (or demo mock)
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name,omitempty"`
		RollNo   string `json:"roll_no,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 6 characters",
		})
	}

	if strings.TrimSpace(req.RollNo) == "" && !h.isDemo() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Roll number is required for student registration",
		})
	}

	// Demo mode
	if h.isDemo() {
		log.Info().Str("email", req.Email).Msg("Registration (demo mode)")
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"access_token":  mockToken(req.Email),
				"refresh_token": "demo-refresh-token",
				"user": fiber.Map{
					"id":    "demo-user-id",
					"email": req.Email,
					"role":  "admin",
					"name":  req.Name,
				},
			},
		})
	}

	result, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		log.Warn().Err(err).Str("email", req.Email).Msg("Registration failed")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Registration failed",
			"details": err.Error(),
		})
	}

	userData, _ := result["user"].(map[string]interface{})
	accessToken, _ := result["access_token"].(string)
	refreshToken, _ := result["refresh_token"].(string)
	user := normalizeUser(userData)
	user["role"] = "student"

	if h.accessRepo != nil {
		userID, _ := user["id"].(string)
		if userID != "" {
			if _, err := h.accessRepo.CreateRegistrationRequest(c.Context(), userID, req.Email, req.Name, req.RollNo); err != nil {
				log.Warn().Err(err).Str("email", req.Email).Msg("Failed to create student registration request")
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "Registration created, but student claim failed",
					"details": err.Error(),
				})
			}
		}
	}

	if accessToken == "" {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"message": "Registration successful. Please check your email to verify your account. Your student details are pending admin approval.",
			},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user":          user,
		},
	})
}

// Me returns the normalized current user from JWT plus database profile.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	email, _ := c.Locals("user_email").(string)
	role, _ := c.Locals("user_role").(string)
	if role == "" {
		role = "student"
	}

	user := fiber.Map{
		"id":    userID,
		"email": email,
		"role":  role,
	}
	if profile, ok := c.Locals("profile").(model.UserProfile); ok {
		user["role"] = profile.Role
		if profile.StudentID != nil {
			user["student_id"] = *profile.StudentID
		}
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"user": user}})
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 6 characters",
		})
	}
	if h.isDemo() {
		return c.JSON(fiber.Map{"success": true, "message": "Password updated"})
	}

	token := bearerToken(c.Get("Authorization"))
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization token"})
	}
	if err := h.authService.UpdateCurrentUserPassword(token, req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to update password",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password updated"})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	if h.isDemo() {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"access_token":  "demo-refreshed-token",
				"refresh_token": "demo-refresh-token",
			},
		})
	}

	result, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Token refresh failed",
			"details": err.Error(),
		})
	}

	accessToken, _ := result["access_token"].(string)
	refreshToken, _ := result["refresh_token"].(string)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	if !h.isDemo() {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			_ = h.authService.Logout(authHeader[7:])
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}

// GetUploadSignedURL generates a signed URL for file upload
func (h *AuthHandler) GetUploadSignedURL(c *fiber.Ctx) error {
	var req struct {
		Bucket      string `json:"bucket"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Bucket == "" || req.FileName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bucket and file_name are required",
		})
	}

	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}

	// Demo mode: return a placeholder URL
	if h.isDemo() {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"signed_url": fmt.Sprintf("/storage/v1/object/sign/%s/%s", req.Bucket, req.FileName),
				"file_name":  req.FileName,
			},
		})
	}

	signedURL, err := h.authService.GetUploadSignedURL(req.Bucket, req.FileName, req.ContentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to generate signed URL",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"signed_url": signedURL,
			"file_name":  req.FileName,
		},
	})
}

// normalizeUser extracts consistent user fields from Supabase user object
func normalizeUser(userData map[string]interface{}) fiber.Map {
	if userData == nil {
		return fiber.Map{
			"id":    "",
			"email": "",
			"role":  "authenticated",
		}
	}

	userID, _ := userData["id"].(string)
	email, _ := userData["email"].(string)

	role := "authenticated"
	if meta, ok := userData["app_metadata"].(map[string]interface{}); ok {
		if r, ok := meta["role"].(string); ok && r != "" {
			role = r
		}
	}
	if userMeta, ok := userData["user_metadata"].(map[string]interface{}); ok {
		if r, ok := userMeta["role"].(string); ok && r != "" {
			role = r
		}
	}

	name := ""
	if userMeta, ok := userData["user_metadata"].(map[string]interface{}); ok {
		if n, ok := userMeta["full_name"].(string); ok {
			name = n
		}
		if n, ok := userMeta["name"].(string); ok && name == "" {
			name = n
		}
	}

	return fiber.Map{
		"id":    userID,
		"email": email,
		"role":  role,
		"name":  name,
	}
}

func (h *AuthHandler) enrichUser(c *fiber.Ctx, user fiber.Map) fiber.Map {
	if h.accessRepo == nil {
		return user
	}
	userID, _ := user["id"].(string)
	email, _ := user["email"].(string)
	if userID == "" {
		return user
	}
	profile, err := h.accessRepo.GetProfile(c.Context(), userID, email)
	if err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to enrich user profile")
		return user
	}
	user["role"] = profile.Role
	if profile.StudentID != nil {
		user["student_id"] = *profile.StudentID
	}
	return user
}

func bearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
