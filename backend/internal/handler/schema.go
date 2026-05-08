package handler

import (
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type SchemaHandler struct {
	schemaService *service.SchemaService
}

func NewSchemaHandler(schemaService *service.SchemaService) *SchemaHandler {
	return &SchemaHandler{
		schemaService: schemaService,
	}
}

// GetSchema returns the complete discovered database schema
func (h *SchemaHandler) GetSchema(c *fiber.Ctx) error {
	schema, err := h.schemaService.GetSchema(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to get schema")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to discover schema",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    schema,
	})
}

// RefreshSchema forces a fresh schema discovery
func (h *SchemaHandler) RefreshSchema(c *fiber.Ctx) error {
	schema, err := h.schemaService.RefreshSchema(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to refresh schema")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to refresh schema",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    schema,
		"message": "Schema refreshed successfully",
	})
}
