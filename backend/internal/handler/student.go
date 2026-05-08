package handler

import (
	"strings"

	"cms-backend/internal/model"
	"cms-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type StudentHandler struct {
	accessRepo *repository.StudentAccessRepository
}

func NewStudentHandler(accessRepo *repository.StudentAccessRepository) *StudentHandler {
	return &StudentHandler{accessRepo: accessRepo}
}

func (h *StudentHandler) Me(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	workspace, err := h.accessRepo.GetWorkspace(c.Context(), profile)
	if err != nil {
		log.Error().Err(err).Str("user_id", profile.ID).Msg("Failed to load student workspace")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load student workspace",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": workspace})
}

func (h *StudentHandler) ListRequests(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	requests, err := h.accessRepo.ListStudentRequests(c.Context(), profile.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load requests",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": requests})
}

func (h *StudentHandler) CreateRequest(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}
	if profile.Role != "student" && profile.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Student access required"})
	}

	var req model.StudentChangeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}
	req.TargetTable = strings.TrimSpace(req.TargetTable)
	req.RequestType = strings.TrimSpace(req.RequestType)

	created, err := h.accessRepo.CreateStudentRequest(c.Context(), profile, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Unable to create request",
			"details": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Request submitted for admin review",
	})
}

// AddAchievementContributor adds a student as a contributor to an achievement
func (h *StudentHandler) AddAchievementContributor(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	var req struct {
		AchievementID string `json:"achievement_id"`
		StudentID     string `json:"student_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.AchievementID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "achievement_id is required"})
	}
	if req.StudentID == "" {
		if profile.StudentID == nil || *profile.StudentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "student_id is required"})
		}
		req.StudentID = *profile.StudentID
	}

	created, err := h.accessRepo.AddAchievementContributor(c.Context(), profile.ID, req.AchievementID, req.StudentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to add contributor",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Contributor request submitted for admin review",
	})
}

// RemoveAchievementContributor removes a student from an achievement
func (h *StudentHandler) RemoveAchievementContributor(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	var req struct {
		AchievementID string `json:"achievement_id"`
		StudentID     string `json:"student_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.AchievementID == "" || req.StudentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "achievement_id and student_id are required"})
	}

	if err := h.accessRepo.RemoveAchievementContributor(c.Context(), profile.ID, req.AchievementID, req.StudentID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to remove contributor",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Contributor removed successfully"})
}

// AddProjectContributor adds a student as a contributor to a project
func (h *StudentHandler) AddProjectContributor(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	var req struct {
		ProjectID string `json:"project_id"`
		StudentID string `json:"student_id"`
		Role      string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.ProjectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}
	if req.StudentID == "" {
		if profile.StudentID == nil || *profile.StudentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "student_id is required"})
		}
		req.StudentID = *profile.StudentID
	}

	created, err := h.accessRepo.AddProjectContributor(c.Context(), profile.ID, req.ProjectID, req.StudentID, req.Role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to add contributor",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    created,
		"message": "Contributor request submitted for admin review",
	})
}

// RemoveProjectContributor removes a student from a project
func (h *StudentHandler) RemoveProjectContributor(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	var req struct {
		ProjectID string `json:"project_id"`
		StudentID string `json:"student_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.ProjectID == "" || req.StudentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id and student_id are required"})
	}

	if err := h.accessRepo.RemoveProjectContributor(c.Context(), profile.ID, req.ProjectID, req.StudentID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to remove contributor",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Contributor removed successfully"})
}

// SearchStudents searches for students by name or roll number
func (h *StudentHandler) SearchStudents(c *fiber.Ctx) error {
	search := c.Query("q", "")
	if search == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "search query 'q' is required"})
	}

	limit := 20
	if l := c.QueryInt("limit"); l > 0 && l <= 50 {
		limit = l
	}

	students, err := h.accessRepo.ListStudentsSearchable(c.Context(), search, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to search students",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": students})
}

// GetContributors gets all contributors for a specific achievement or project
func (h *StudentHandler) GetContributors(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}

	recordType := c.Query("type", "")
	recordID := c.Query("id", "")

	if recordType == "" || recordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type and id query parameters are required"})
	}

	if recordType != "achievement" && recordType != "project" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type must be 'achievement' or 'project'"})
	}

	contributors, err := h.accessRepo.GetStudentContributors(c.Context(), profile.ID, recordType, recordID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to get contributors",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": contributors})
}

type AdminReviewHandler struct {
	accessRepo *repository.StudentAccessRepository
}

func NewAdminReviewHandler(accessRepo *repository.StudentAccessRepository) *AdminReviewHandler {
	return &AdminReviewHandler{accessRepo: accessRepo}
}

func (h *AdminReviewHandler) ListChangeRequests(c *fiber.Ctx) error {
	status := c.Query("status", "pending")
	requests, err := h.accessRepo.ListAdminRequests(c.Context(), status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load change requests",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": requests})
}

func (h *AdminReviewHandler) Approve(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}
	note := reviewNote(c)
	updated, err := h.accessRepo.ApproveRequest(c.Context(), c.Params("id"), profile.ID, note)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to approve request",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": updated, "message": "Request approved"})
}

func (h *AdminReviewHandler) Reject(c *fiber.Ctx) error {
	profile, ok := currentProfile(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Profile not found"})
	}
	note := reviewNote(c)
	updated, err := h.accessRepo.RejectRequest(c.Context(), c.Params("id"), profile.ID, note)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to reject request",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": updated, "message": "Request rejected"})
}

func currentProfile(c *fiber.Ctx) (model.UserProfile, bool) {
	profile, ok := c.Locals("profile").(model.UserProfile)
	return profile, ok
}

func reviewNote(c *fiber.Ctx) string {
	var body struct {
		Note string `json:"note"`
	}
	_ = c.BodyParser(&body)
	return strings.TrimSpace(body.Note)
}
