package handler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"cms-backend/internal/model"
	"cms-backend/internal/repository"
	"cms-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type CrudHandler struct {
	crudService   *service.CrudService
	crudRepo      *repository.CrudRepository
	schemaService *service.SchemaService
}

func NewCrudHandler(crudService *service.CrudService, crudRepo *repository.CrudRepository, schemaService *service.SchemaService) *CrudHandler {
	return &CrudHandler{
		crudService:   crudService,
		crudRepo:      crudRepo,
		schemaService: schemaService,
	}
}

// ListRecords returns paginated, sortable, filterable records from a table
func (h *CrudHandler) ListRecords(c *fiber.Ctx) error {
	tableName := c.Params("tableName")
	if tableName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table_name is required",
		})
	}
	if _, err := h.schemaService.GetTableSchema(tableName); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Table not found",
			"details": err.Error(),
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))
	sortBy := c.Query("sort_by", "")
	sortOrder := c.Query("sort_order", "DESC")

	filters := make(map[string]string)
	for key, values := range c.Queries() {
		if key == "page" || key == "page_size" || key == "sort_by" || key == "sort_order" {
			continue
		}
		filters[key] = values
	}

	records, total, err := h.crudRepo.ListRecords(c.Context(), tableName, page, pageSize, sortBy, sortOrder, filters)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Msg("Failed to list records")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch records",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    records,
		"meta":    service.BuildPaginatedResponse(records, total, page, pageSize),
	})
}

// GetRecord returns a single record by ID
func (h *CrudHandler) GetRecord(c *fiber.Ctx) error {
	tableName := c.Params("tableName")
	id := c.Params("id")

	if tableName == "" || id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table_name and id are required",
		})
	}

	tableSchema, err := h.schemaService.GetTableSchema(tableName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Table not found",
			"details": err.Error(),
		})
	}
	keyValues, err := parseRecordKey(tableSchema, id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid record identifier",
			"details": err.Error(),
		})
	}

	record, err := h.crudRepo.GetRecord(c.Context(), tableName, keyValues)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Str("id", id).Msg("Failed to get record")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Record not found",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    record,
	})
}

// CreateRecord creates a new record in a table
func (h *CrudHandler) CreateRecord(c *fiber.Ctx) error {
	tableName := c.Params("tableName")
	if tableName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table_name is required",
		})
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	tableSchema, err := h.schemaService.GetTableSchema(tableName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Table not found",
			"details": err.Error(),
		})
	}
	if errors := h.crudService.ValidateRecord(data, tableSchema); len(errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": errors,
		})
	}
	data = h.crudService.PrepareDataForInsert(data, tableSchema)

	record, err := h.crudRepo.CreateRecord(c.Context(), tableName, data)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Msg("Failed to create record")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create record",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    record,
		"message": "Record created successfully",
	})
}

// UpdateRecord updates an existing record
func (h *CrudHandler) UpdateRecord(c *fiber.Ctx) error {
	tableName := c.Params("tableName")
	id := c.Params("id")

	if tableName == "" || id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table_name and id are required",
		})
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	tableSchema, err := h.schemaService.GetTableSchema(tableName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Table not found",
			"details": err.Error(),
		})
	}
	keyValues, err := parseRecordKey(tableSchema, id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid record identifier",
			"details": err.Error(),
		})
	}
	if errors := h.crudService.ValidateRecord(data, tableSchema); len(errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": errors,
		})
	}
	data = h.crudService.PrepareDataForInsert(data, tableSchema)

	record, err := h.crudRepo.UpdateRecord(c.Context(), tableName, keyValues, data)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Str("id", id).Msg("Failed to update record")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to update record",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    record,
		"message": "Record updated successfully",
	})
}

// DeleteRecord deletes a record
func (h *CrudHandler) DeleteRecord(c *fiber.Ctx) error {
	tableName := c.Params("tableName")
	id := c.Params("id")

	if tableName == "" || id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table_name and id are required",
		})
	}

	softDelete := c.Query("soft", "true") == "true"
	timestampColumn := c.Query("timestamp_column", "deleted_at")
	tableSchema, err := h.schemaService.GetTableSchema(tableName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Table not found",
			"details": err.Error(),
		})
	}
	if softDelete && !tableHasColumn(tableSchema, timestampColumn) {
		softDelete = false
	}
	keyValues, err := parseRecordKey(tableSchema, id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid record identifier",
			"details": err.Error(),
		})
	}

	err = h.crudRepo.DeleteRecord(c.Context(), tableName, keyValues, softDelete, timestampColumn)
	if err != nil {
		log.Error().Err(err).Str("table", tableName).Str("id", id).Msg("Failed to delete record")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to delete record",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Record deleted successfully",
	})
}

func parseRecordKey(tableSchema *model.TableSchema, rawID string) (map[string]string, error) {
	primaryKeys := tableSchema.PrimaryKeys
	if len(primaryKeys) == 0 && tableSchema.PrimaryKey != "" {
		primaryKeys = []string{tableSchema.PrimaryKey}
	}
	if len(primaryKeys) == 0 {
		primaryKeys = []string{"id"}
	}

	if len(primaryKeys) == 1 {
		return map[string]string{primaryKeys[0]: rawID}, nil
	}

	decodedID, err := url.QueryUnescape(rawID)
	if err != nil {
		decodedID = rawID
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(decodedID), &parsed); err != nil {
		return nil, fmt.Errorf("composite key must be URL-encoded JSON: %w", err)
	}

	keyValues := make(map[string]string, len(primaryKeys))
	for _, key := range primaryKeys {
		value, exists := parsed[key]
		if !exists || value == nil {
			return nil, fmt.Errorf("missing primary key value for %s", key)
		}
		keyValues[key] = fmt.Sprintf("%v", value)
	}

	return keyValues, nil
}

func tableHasColumn(tableSchema *model.TableSchema, columnName string) bool {
	for _, column := range tableSchema.Columns {
		if column.ColumnName == columnName {
			return true
		}
	}
	return false
}
