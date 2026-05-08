package handler

import (
	"strings"

	"cms-backend/internal/model"
	"cms-backend/internal/repository"
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

const defaultStudentPassword = "user123"

type AccessHandler struct {
	accessRepo  *repository.StudentAccessRepository
	authService *service.AuthService
}

func NewAccessHandler(accessRepo *repository.StudentAccessRepository, authService *service.AuthService) *AccessHandler {
	return &AccessHandler{accessRepo: accessRepo, authService: authService}
}

func (h *AccessHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.accessRepo.ListAccessUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load access users",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": users})
}

func (h *AccessHandler) ProvisionStudents(c *fiber.Ctx) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if strings.TrimSpace(req.Password) == "" {
		req.Password = defaultStudentPassword
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 6 characters"})
	}

	if err := h.authService.AdminHealthCheck(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Supabase Auth admin key is not valid for this project",
			"details": err.Error(),
		})
	}

	users, err := h.accessRepo.ListAccessUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load students",
			"details": err.Error(),
		})
	}

	result := model.ProvisionResult{
		Users: make([]model.ProvisionUserResult, 0, len(users)),
	}

	for _, user := range users {
		provisioned := h.provisionOne(c, user, req.Password)
		switch provisioned.Status {
		case "created":
			result.Created++
		case "updated":
			result.Updated++
		default:
			result.Failed++
		}
		result.Users = append(result.Users, provisioned)
	}

	if result.Failed == 0 {
		if err := h.accessRepo.NormalizeProfileRoles(c.Context()); err != nil {
			log.Warn().Err(err).Msg("Failed to normalize student profile roles")
		}
	}

	return c.JSON(fiber.Map{"success": true, "data": result})
}

func (h *AccessHandler) provisionOne(c *fiber.Ctx, user model.AccessUser, password string) model.ProvisionUserResult {
	role := repository.InitialRoleForRollNo(user.RollNo)
	email := strings.TrimSpace(user.LoginEmail)
	if email == "" {
		email = repository.GeneratedStudentEmail(user.RollNo)
	}
	result := model.ProvisionUserResult{
		StudentID:  user.StudentID,
		RollNo:     user.RollNo,
		LoginEmail: email,
		Role:       role,
		UserID:     user.UserID,
	}

	if email == "" || user.StudentID == "" {
		result.Status = "failed"
		result.Message = "student is missing roll number or id"
		return result
	}

	if user.UserID != "" {
		if _, err := h.authService.AdminUpdateUser(user.UserID, password, role); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
			return result
		}
		if err := h.accessRepo.SetProfileForStudent(c.Context(), user.UserID, role, user.StudentID); err != nil {
			result.Status = "failed"
			result.Message = err.Error()
			return result
		}
		result.Status = "updated"
		result.Message = "password and role refreshed"
		return result
	}

	created, err := h.authService.AdminCreateUser(email, password, user.Name, user.RollNo, role)
	if err != nil {
		userID, lookupErr := h.accessRepo.GetAuthUserIDByEmail(c.Context(), email)
		if lookupErr != nil || userID == "" {
			result.Status = "failed"
			result.Message = err.Error()
			return result
		}
		if _, updateErr := h.authService.AdminUpdateUser(userID, password, role); updateErr != nil {
			result.Status = "failed"
			result.Message = updateErr.Error()
			return result
		}
		if profileErr := h.accessRepo.SetProfileForStudent(c.Context(), userID, role, user.StudentID); profileErr != nil {
			result.Status = "failed"
			result.Message = profileErr.Error()
			return result
		}
		result.UserID = userID
		result.Status = "updated"
		result.Message = "existing auth user linked"
		return result
	}

	userID := extractSupabaseUserID(created)
	if userID == "" {
		userID, _ = h.accessRepo.GetAuthUserIDByEmail(c.Context(), email)
	}
	if userID == "" {
		result.Status = "failed"
		result.Message = "auth user was created but no user id was returned"
		return result
	}
	if err := h.accessRepo.SetProfileForStudent(c.Context(), userID, role, user.StudentID); err != nil {
		result.Status = "failed"
		result.Message = err.Error()
		return result
	}

	result.UserID = userID
	result.Status = "created"
	result.Message = "auth user created"
	return result
}

func (h *AccessHandler) SetRole(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role != "admin" && req.Role != "student" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role must be admin or student"})
	}

	if _, err := h.authService.AdminUpdateUser(userID, "", req.Role); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to update Supabase app role")
	}
	if err := h.accessRepo.SetUserRole(c.Context(), userID, req.Role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to update role",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Role updated"})
}

func (h *AccessHandler) ResetPassword(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if strings.TrimSpace(req.Password) == "" {
		req.Password = defaultStudentPassword
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 6 characters"})
	}

	if _, err := h.authService.AdminUpdateUser(userID, req.Password, ""); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to reset password",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password reset"})
}

func extractSupabaseUserID(result map[string]interface{}) string {
	if result == nil {
		return ""
	}
	if id, ok := result["id"].(string); ok {
		return id
	}
	if user, ok := result["user"].(map[string]interface{}); ok {
		if id, ok := user["id"].(string); ok {
			return id
		}
	}
	return ""
}
