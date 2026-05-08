package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cms-backend/internal/config"
	"cms-backend/internal/model"
	"cms-backend/internal/repository"

	"github.com/rs/zerolog/log"
)

// --- Schema Service ---

type SchemaService struct {
	db       *repository.DBWrapper
	repo     *repository.SchemaRepository
	schema   *model.DatabaseSchema
	version  int
	lastSync time.Time
}

func NewSchemaService(db *repository.DBWrapper) *SchemaService {
	return &SchemaService{
		db:      db,
		version: 0,
	}
}

// SetRepo sets the repository after construction (avoids circular dep)
func (s *SchemaService) SetRepo(repo *repository.SchemaRepository) {
	s.repo = repo
}

// GetSchema returns the cached schema or discovers it if not cached
func (s *SchemaService) GetSchema(ctx context.Context) (*model.DatabaseSchema, error) {
	if s.schema != nil && time.Since(s.lastSync) < 5*time.Minute {
		return s.schema, nil
	}
	return s.RefreshSchema(ctx)
}

// RefreshSchema forces a fresh schema discovery from the database
func (s *SchemaService) RefreshSchema(ctx context.Context) (*model.DatabaseSchema, error) {
	log.Info().Msg("Refreshing database schema...")

	tables, err := s.repo.DiscoverTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover schema: %w", err)
	}

	s.schema = &model.DatabaseSchema{
		Tables:       tables,
		DiscoveredAt: time.Now().Format(time.RFC3339),
		Version:      s.version + 1,
	}
	s.version++
	s.lastSync = time.Now()

	log.Info().Int("tables", len(tables)).Int("version", s.version).Msg("Schema refreshed successfully")
	return s.schema, nil
}

// GetPrimaryKeyForTable returns the primary key column name for a given table
func (s *SchemaService) GetPrimaryKeyForTable(tableName string) string {
	primaryKeys := s.GetPrimaryKeysForTable(tableName)
	if len(primaryKeys) > 0 {
		return primaryKeys[0]
	}
	return "id"
}

// GetPrimaryKeysForTable returns all primary key columns for a table.
func (s *SchemaService) GetPrimaryKeysForTable(tableName string) []string {
	if s.schema == nil {
		return []string{"id"}
	}
	for _, t := range s.schema.Tables {
		if t.TableName == tableName {
			if len(t.PrimaryKeys) > 0 {
				return append([]string(nil), t.PrimaryKeys...)
			}
			for _, col := range t.Columns {
				if col.IsPrimaryKey {
					return []string{col.ColumnName}
				}
			}
		}
	}
	return []string{"id"}
}

// GetTableSchema returns the schema for a specific table
func (s *SchemaService) GetTableSchema(tableName string) (*model.TableSchema, error) {
	if s.schema == nil {
		return nil, fmt.Errorf("schema not loaded")
	}
	for _, t := range s.schema.Tables {
		if t.TableName == tableName {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("table %s not found in schema", tableName)
}

// --- Auth Service ---

type AuthService struct {
	cfg    *config.Config
	client *http.Client
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// supabaseRequest makes an authenticated request to the Supabase Auth API
func (s *AuthService) supabaseRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/auth/v1%s", s.cfg.SupabaseURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseAnonKey)
	if s.cfg.SupabaseServiceKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseServiceKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Supabase: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(respBody))
	}

	if resp.StatusCode >= 400 {
		errMsg := "authentication failed"
		if msg, ok := result["error_description"].(string); ok && msg != "" {
			errMsg = msg
		} else if msg, ok := result["msg"].(string); ok && msg != "" {
			errMsg = msg
		} else if msg, ok := result["error"].(string); ok && msg != "" {
			errMsg = msg
		}
		return nil, fmt.Errorf("%s (status %d)", errMsg, resp.StatusCode)
	}

	return result, nil
}

func (s *AuthService) supabaseAdminRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	if s.cfg.SupabaseURL == "" || s.cfg.SupabaseServiceKey == "" {
		return nil, fmt.Errorf("Supabase URL and service role key are required")
	}
	return s.supabaseRequestWithAuth(method, path, body, s.cfg.SupabaseServiceKey, s.cfg.SupabaseServiceKey)
}

func (s *AuthService) supabaseUserRequest(method, path string, body interface{}, accessToken string) (map[string]interface{}, error) {
	if s.cfg.SupabaseURL == "" || s.cfg.SupabaseAnonKey == "" {
		return nil, fmt.Errorf("Supabase URL and anon key are required")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	return s.supabaseRequestWithAuth(method, path, body, s.cfg.SupabaseAnonKey, accessToken)
}

func (s *AuthService) supabaseRequestWithAuth(method, path string, body interface{}, apiKey, bearerToken string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/auth/v1%s", s.cfg.SupabaseURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Supabase: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(respBody))
		}
	} else {
		result = map[string]interface{}{}
	}

	if resp.StatusCode >= 400 {
		errMsg := "Supabase request failed"
		if msg, ok := result["error_description"].(string); ok && msg != "" {
			errMsg = msg
		} else if msg, ok := result["msg"].(string); ok && msg != "" {
			errMsg = msg
		} else if msg, ok := result["message"].(string); ok && msg != "" {
			errMsg = msg
		} else if msg, ok := result["error"].(string); ok && msg != "" {
			errMsg = msg
		}
		return nil, fmt.Errorf("%s (status %d)", errMsg, resp.StatusCode)
	}

	return result, nil
}

// Login authenticates a user via Supabase Auth
func (s *AuthService) Login(email, password string) (map[string]interface{}, error) {
	return s.supabaseRequest("POST", "/token?grant_type=password", map[string]interface{}{
		"email":    email,
		"password": password,
	})
}

// Register creates a new user via Supabase Auth
func (s *AuthService) Register(email, password, name string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	if name != "" {
		body["data"] = map[string]interface{}{"full_name": name}
	}
	return s.supabaseRequest("POST", "/signup", body)
}

func (s *AuthService) AdminCreateUser(email, password, name, rollNo, role string) (map[string]interface{}, error) {
	metadata := map[string]interface{}{}
	if name != "" {
		metadata["full_name"] = name
		metadata["name"] = name
	}
	if rollNo != "" {
		metadata["roll_no"] = rollNo
	}
	body := map[string]interface{}{
		"email":         email,
		"password":      password,
		"email_confirm": true,
		"user_metadata": metadata,
		"app_metadata": map[string]interface{}{
			"role": role,
		},
	}
	return s.supabaseAdminRequest("POST", "/admin/users", body)
}

func (s *AuthService) AdminHealthCheck() error {
	_, err := s.supabaseAdminRequest("GET", "/admin/users?page=1&per_page=1", nil)
	if err != nil {
		return fmt.Errorf("Supabase Auth admin API check failed: %w", err)
	}
	return nil
}

func (s *AuthService) AdminUpdateUser(userID, password, role string) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	if password != "" {
		body["password"] = password
	}
	if role != "" {
		body["app_metadata"] = map[string]interface{}{"role": role}
	}
	if len(body) == 0 {
		return map[string]interface{}{}, nil
	}
	result, err := s.supabaseAdminRequest("PUT", "/admin/users/"+userID, body)
	if err == nil {
		return result, nil
	}
	if !strings.Contains(err.Error(), "status 404") {
		return nil, err
	}
	return s.supabaseAdminRequest("PUT", "/admin/user/"+userID, body)
}

func (s *AuthService) UpdateCurrentUserPassword(accessToken, password string) error {
	_, err := s.supabaseUserRequest("PUT", "/user", map[string]interface{}{
		"password": password,
	}, accessToken)
	return err
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(refreshToken string) (map[string]interface{}, error) {
	return s.supabaseRequest("POST", "/token?grant_type=refresh_token", map[string]interface{}{
		"refresh_token": refreshToken,
	})
}

// Logout invalidates the current session
func (s *AuthService) Logout(accessToken string) error {
	url := fmt.Sprintf("%s/auth/v1/logout", s.cfg.SupabaseURL)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("logout failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetUploadSignedURL generates a signed URL for direct upload to Supabase Storage
func (s *AuthService) GetUploadSignedURL(bucket, fileName, contentType string) (string, error) {
	// Use Supabase Storage API to create a signed upload URL
	// Correct endpoint: /storage/v1/object/upload/sign/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/upload/sign/%s/%s", s.cfg.SupabaseURL, bucket, fileName)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseServiceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get signed URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("storage API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Supabase returns { "url": "/storage/v1/object/upload/sign/bucket/file?token=..." }
	if signedPath, ok := result["url"].(string); ok {
		// The returned url is a relative path; prepend the Supabase base URL
		return fmt.Sprintf("%s%s", s.cfg.SupabaseURL, signedPath), nil
	}

	// Legacy field name check
	if signedURL, ok := result["signedUrl"].(string); ok {
		return signedURL, nil
	}

	// Fallback: return a direct upload URL (for public buckets)
	return fmt.Sprintf("%s/storage/v1/object/%s/%s", s.cfg.SupabaseURL, bucket, fileName), nil
}

// --- CRUD Service ---

type CrudService struct{}

func NewCrudService() *CrudService {
	return &CrudService{}
}

// ValidateRecord validates a record against the table schema
func (s *CrudService) ValidateRecord(data map[string]interface{}, tableSchema *model.TableSchema) []string {
	var errors []string

	for _, col := range tableSchema.Columns {
		val, exists := data[col.ColumnName]

		if col.IsNullable == "NO" && col.ColumnDefault == nil && !exists {
			errors = append(errors, fmt.Sprintf("column %s is required", col.ColumnName))
			continue
		}

		if !exists || val == nil {
			continue
		}

		switch col.DataType {
		case "integer", "bigint", "smallint", "numeric", "decimal", "real", "double precision":
			switch val.(type) {
			case float64, float32, int, int64, int32:
				// valid
			case string:
				errors = append(errors, fmt.Sprintf("column %s must be a number", col.ColumnName))
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				errors = append(errors, fmt.Sprintf("column %s must be a boolean", col.ColumnName))
			}
		}
	}

	return errors
}

// PrepareDataForInsert prepares data for database insertion with type coercion
func (s *CrudService) PrepareDataForInsert(data map[string]interface{}, tableSchema *model.TableSchema) map[string]interface{} {
	prepared := make(map[string]interface{})

	for _, col := range tableSchema.Columns {
		val, exists := data[col.ColumnName]
		if !exists {
			continue
		}

		if val == nil || val == "" {
			if col.IsNullable == "YES" {
				prepared[col.ColumnName] = nil
				continue
			}
		}

		switch col.DataType {
		case "boolean":
			switch v := val.(type) {
			case bool:
				prepared[col.ColumnName] = v
			case string:
				prepared[col.ColumnName] = v == "true" || v == "1" || v == "yes"
			case float64:
				prepared[col.ColumnName] = v != 0
			default:
				prepared[col.ColumnName] = val
			}
		case "jsonb", "json":
			prepared[col.ColumnName] = val
		case "timestamp without time zone", "timestamp with time zone":
			if str, ok := val.(string); ok {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					prepared[col.ColumnName] = t
				} else {
					prepared[col.ColumnName] = str
				}
			} else {
				prepared[col.ColumnName] = val
			}
		default:
			prepared[col.ColumnName] = val
		}
	}

	return prepared
}

// FilterDeletedRecords filters out soft-deleted records
func (s *CrudService) FilterDeletedRecords(records []map[string]interface{}, tableSchema *model.TableSchema) []map[string]interface{} {
	if !tableSchema.HasTimestamps {
		return records
	}

	var filtered []map[string]interface{}
	for _, record := range records {
		if deletedAt, exists := record["deleted_at"]; exists && deletedAt != nil {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// BuildPaginatedResponse creates a standardized paginated response
func BuildPaginatedResponse(data []map[string]interface{}, total int64, page, pageSize int) PaginatedResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// GetIDColumn returns the primary key column name for a table
func GetIDColumn(tableSchema *model.TableSchema) string {
	if tableSchema.PrimaryKey != "" {
		return tableSchema.PrimaryKey
	}
	for _, col := range tableSchema.Columns {
		if col.ColumnName == "id" || col.ColumnName == "uuid" || col.ColumnName == "slug" {
			return col.ColumnName
		}
	}
	return "id"
}

// CheckPermission checks if a user role has permission for an action on a table
func CheckPermission(role, action, tableName string, permissions map[string]map[string][]string) bool {
	if role == "admin" {
		return true
	}

	if tablePerms, exists := permissions[tableName]; exists {
		if allowedActions, exists := tablePerms[role]; exists {
			for _, allowed := range allowedActions {
				if allowed == "*" || allowed == action {
					return true
				}
			}
		}
	}

	if role == "viewer" && action == "read" {
		return true
	}

	return false
}
