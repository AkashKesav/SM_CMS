package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cms-backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// DBWrapper wraps either pgxpool.Pool or in-memory storage
type DBWrapper struct {
	pool      *pgxpool.Pool
	inMem     *InMemoryDB
	isInMem   bool
	modeLabel string
}

func (db *DBWrapper) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *DBWrapper) Mode() string {
	return db.modeLabel
}

// IsInMemory returns whether the DB is running in demo/memory mode
func (db *DBWrapper) IsInMemory() bool {
	return db.isInMem
}

// Pool returns the underlying pgxpool.Pool (nil in demo mode)
func (db *DBWrapper) Pool() *pgxpool.Pool {
	return db.pool
}

// InMemory returns the in-memory DB (nil in production mode)
func (db *DBWrapper) InMemory() *InMemoryDB {
	return db.inMem
}

// InMemoryDB provides a demo-mode storage when no database is configured
type InMemoryDB struct {
	mu        sync.RWMutex
	tables    map[string]map[string]map[string]interface{}
	schema    map[string]*model.TableSchema
	idCounter map[string]int
}

func NewInMemoryDB() *DBWrapper {
	db := &InMemoryDB{
		tables:    make(map[string]map[string]map[string]interface{}),
		schema:    make(map[string]*model.TableSchema),
		idCounter: make(map[string]int),
	}

	// Seed with demo data
	db.seedDemoData()

	return &DBWrapper{inMem: db, isInMem: true, modeLabel: "demo"}
}

func (db *InMemoryDB) seedDemoData() {
	log.Info().Msg("Seeding demo data for in-memory mode")

	// Demo: users table
	db.schema["users"] = &model.TableSchema{
		TableName: "users",
		Columns: []model.ColumnInfo{
			{ColumnName: "id", DataType: "integer", IsNullable: "NO", IsPrimaryKey: true, UdtName: "int4", ColumnDefault: ptrStr("autoincrement")},
			{ColumnName: "name", DataType: "character varying", IsNullable: "NO", UdtName: "varchar", CharacterMaximumLength: ptrInt(255)},
			{ColumnName: "email", DataType: "character varying", IsNullable: "NO", UdtName: "varchar", CharacterMaximumLength: ptrInt(255)},
			{ColumnName: "role", DataType: "character varying", IsNullable: "YES", UdtName: "varchar", CharacterMaximumLength: ptrInt(50)},
			{ColumnName: "is_active", DataType: "boolean", IsNullable: "YES", UdtName: "bool"},
			{ColumnName: "bio", DataType: "text", IsNullable: "YES", UdtName: "text"},
			{ColumnName: "metadata", DataType: "jsonb", IsNullable: "YES", UdtName: "jsonb"},
			{ColumnName: "avatar_url", DataType: "character varying", IsNullable: "YES", UdtName: "varchar", CharacterMaximumLength: ptrInt(500)},
			{ColumnName: "created_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
			{ColumnName: "updated_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
		},
		PrimaryKey:    "id",
		PrimaryKeys:   []string{"id"},
		HasTimestamps: true,
	}

	db.tables["users"] = map[string]map[string]interface{}{
		"1": {
			"id": 1, "name": "Alice Johnson", "email": "alice@example.com",
			"role": "admin", "is_active": true, "bio": "System administrator",
			"metadata":   map[string]interface{}{"theme": "dark", "language": "en"},
			"avatar_url": "https://api.dicebear.com/7.x/avataaars/svg?seed=Alice",
			"created_at": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			"updated_at": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		"2": {
			"id": 2, "name": "Bob Smith", "email": "bob@example.com",
			"role": "editor", "is_active": true, "bio": "Content editor and writer",
			"metadata":   map[string]interface{}{"theme": "light", "language": "en"},
			"avatar_url": "https://api.dicebear.com/7.x/avataaars/svg?seed=Bob",
			"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"updated_at": time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		},
		"3": {
			"id": 3, "name": "Carol Williams", "email": "carol@example.com",
			"role": "viewer", "is_active": false, "bio": "",
			"metadata":   map[string]interface{}{},
			"avatar_url": "",
			"created_at": time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
			"updated_at": time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
		},
	}
	db.idCounter["users"] = 3

	// Demo: posts table
	db.schema["posts"] = &model.TableSchema{
		TableName: "posts",
		Columns: []model.ColumnInfo{
			{ColumnName: "id", DataType: "integer", IsNullable: "NO", IsPrimaryKey: true, UdtName: "int4", ColumnDefault: ptrStr("autoincrement")},
			{ColumnName: "title", DataType: "character varying", IsNullable: "NO", UdtName: "varchar", CharacterMaximumLength: ptrInt(500)},
			{ColumnName: "content", DataType: "text", IsNullable: "YES", UdtName: "text"},
			{ColumnName: "status", DataType: "character varying", IsNullable: "YES", UdtName: "varchar", CharacterMaximumLength: ptrInt(50)},
			{ColumnName: "author_id", DataType: "integer", IsNullable: "YES", IsForeignKey: true, UdtName: "int4"},
			{ColumnName: "is_published", DataType: "boolean", IsNullable: "YES", UdtName: "bool"},
			{ColumnName: "tags", DataType: "jsonb", IsNullable: "YES", UdtName: "jsonb"},
			{ColumnName: "published_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
			{ColumnName: "created_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
			{ColumnName: "updated_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
		},
		ForeignKeys: []model.ForeignKeyInfo{
			{TableName: "posts", ColumnName: "author_id", ForeignTableName: "users", ForeignColumnName: "id", ConstraintName: "fk_posts_author"},
		},
		PrimaryKey:    "id",
		PrimaryKeys:   []string{"id"},
		HasTimestamps: true,
	}

	db.tables["posts"] = map[string]map[string]interface{}{
		"1": {
			"id": 1, "title": "Getting Started with Dynamic CMS",
			"content": "This is a comprehensive guide to using our dynamic schema discovery CMS...",
			"status":  "published", "author_id": 1, "is_published": true,
			"tags":         []string{"tutorial", "cms", "dynamic"},
			"published_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"created_at":   time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			"updated_at":   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
		"2": {
			"id": 2, "title": "Advanced Schema Configuration",
			"content": "Learn how to configure complex database schemas with foreign keys...",
			"status":  "draft", "author_id": 2, "is_published": false,
			"tags":         []string{"advanced", "schema"},
			"published_at": nil,
			"created_at":   time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
			"updated_at":   time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
		},
		"3": {
			"id": 3, "title": "Performance Optimization Tips",
			"content": "Optimize your CMS performance with these proven techniques...",
			"status":  "published", "author_id": 1, "is_published": true,
			"tags":         []string{"performance", "optimization"},
			"published_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"created_at":   time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
			"updated_at":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}
	db.idCounter["posts"] = 3

	// Demo: categories table
	db.schema["categories"] = &model.TableSchema{
		TableName: "categories",
		Columns: []model.ColumnInfo{
			{ColumnName: "id", DataType: "integer", IsNullable: "NO", IsPrimaryKey: true, UdtName: "int4", ColumnDefault: ptrStr("autoincrement")},
			{ColumnName: "name", DataType: "character varying", IsNullable: "NO", UdtName: "varchar", CharacterMaximumLength: ptrInt(100)},
			{ColumnName: "slug", DataType: "character varying", IsNullable: "NO", UdtName: "varchar", CharacterMaximumLength: ptrInt(100)},
			{ColumnName: "description", DataType: "text", IsNullable: "YES", UdtName: "text"},
			{ColumnName: "parent_id", DataType: "integer", IsNullable: "YES", IsForeignKey: true, UdtName: "int4"},
			{ColumnName: "sort_order", DataType: "integer", IsNullable: "YES", UdtName: "int4"},
			{ColumnName: "created_at", DataType: "timestamp with time zone", IsNullable: "YES", UdtName: "timestamptz"},
		},
		ForeignKeys: []model.ForeignKeyInfo{
			{TableName: "categories", ColumnName: "parent_id", ForeignTableName: "categories", ForeignColumnName: "id", ConstraintName: "fk_categories_parent"},
		},
		PrimaryKey:    "id",
		PrimaryKeys:   []string{"id"},
		HasTimestamps: true,
	}

	db.tables["categories"] = map[string]map[string]interface{}{
		"1": {
			"id": 1, "name": "Technology", "slug": "technology",
			"description": "All technology related content", "parent_id": nil, "sort_order": 1,
			"created_at": time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		},
		"2": {
			"id": 2, "name": "Programming", "slug": "programming",
			"description": "Programming tutorials and guides", "parent_id": 1, "sort_order": 1,
			"created_at": time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		},
		"3": {
			"id": 3, "name": "Design", "slug": "design",
			"description": "UI/UX design articles", "parent_id": nil, "sort_order": 2,
			"created_at": time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		},
	}
	db.idCounter["categories"] = 3

	log.Info().Int("tables", len(db.schema)).Msg("Demo data seeded successfully")
}

func ptrInt(i int) *int {
	return &i
}

func ptrStr(s string) *string {
	return &s
}

// DiscoverTables returns the in-memory schema
func (db *InMemoryDB) DiscoverTables(ctx context.Context) ([]model.TableSchema, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var tables []model.TableSchema
	for _, schema := range db.schema {
		tables = append(tables, *schema)
	}

	// Sort by table name
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].TableName < tables[j].TableName
	})

	return tables, nil
}

// ListRecords returns records from an in-memory table
func (db *InMemoryDB) ListRecords(ctx context.Context, tableName string, page, pageSize int, sortBy, sortOrder string, filters map[string]string) ([]map[string]interface{}, int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	tableData, exists := db.tables[tableName]
	if !exists {
		return nil, 0, fmt.Errorf("table %s not found", tableName)
	}

	// Convert to slice
	var allRecords []map[string]interface{}
	for _, record := range tableData {
		allRecords = append(allRecords, record)
	}

	// Apply filters
	if len(filters) > 0 {
		var filtered []map[string]interface{}
		for _, record := range allRecords {
			match := true
			for col, val := range filters {
				recordVal, exists := record[col]
				if !exists {
					match = false
					break
				}

				strVal := fmt.Sprintf("%v", recordVal)
				if strings.HasPrefix(val, "like:") {
					search := strings.TrimPrefix(val, "like:")
					if !strings.Contains(strings.ToLower(strVal), strings.ToLower(search)) {
						match = false
						break
					}
				} else if strings.HasPrefix(val, "ne:") {
					if strVal == strings.TrimPrefix(val, "ne:") {
						match = false
						break
					}
				} else {
					if strVal != val {
						match = false
						break
					}
				}
			}
			if match {
				filtered = append(filtered, record)
			}
		}
		allRecords = filtered
	}

	total := int64(len(allRecords))

	// Sort
	if sortBy != "" {
		sort.Slice(allRecords, func(i, j int) bool {
			a := fmt.Sprintf("%v", allRecords[i][sortBy])
			b := fmt.Sprintf("%v", allRecords[j][sortBy])
			if sortOrder == "DESC" {
				return a > b
			}
			return a < b
		})
	}

	// Paginate
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(allRecords) {
		return []map[string]interface{}{}, total, nil
	}
	if end > len(allRecords) {
		end = len(allRecords)
	}

	return allRecords[start:end], total, nil
}

// GetRecord returns a single record
func (db *InMemoryDB) GetRecord(ctx context.Context, tableName string, keyValues map[string]string) (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	tableData, exists := db.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not found", tableName)
	}

	for _, record := range tableData {
		if recordMatchesKeys(record, keyValues) {
			return record, nil
		}
	}

	return nil, fmt.Errorf("record not found")
}

// CreateRecord creates a new record
func (db *InMemoryDB) CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.tables[tableName]; !exists {
		db.tables[tableName] = make(map[string]map[string]interface{})
	}

	db.idCounter[tableName]++
	newID := db.idCounter[tableName]

	// Set auto-fields
	data["id"] = newID
	now := time.Now().Format(time.RFC3339)
	if _, exists := data["created_at"]; !exists {
		data["created_at"] = now
	}
	data["updated_at"] = now

	idStr := fmt.Sprintf("%d", newID)
	db.tables[tableName][idStr] = data

	return data, nil
}

// UpdateRecord updates an existing record
func (db *InMemoryDB) UpdateRecord(ctx context.Context, tableName string, keyValues map[string]string, data map[string]interface{}) (map[string]interface{}, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	tableData, exists := db.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s not found", tableName)
	}

	var record map[string]interface{}
	for _, candidate := range tableData {
		if recordMatchesKeys(candidate, keyValues) {
			record = candidate
			break
		}
	}
	if record == nil {
		return nil, fmt.Errorf("record not found")
	}

	// Update fields
	for k, v := range data {
		if _, isKey := keyValues[k]; k != "id" && !isKey {
			record[k] = v
		}
	}
	record["updated_at"] = time.Now().Format(time.RFC3339)

	return record, nil
}

// DeleteRecord deletes a record
func (db *InMemoryDB) DeleteRecord(ctx context.Context, tableName string, keyValues map[string]string, softDelete bool, timestampColumn string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tableData, exists := db.tables[tableName]
	if !exists {
		return fmt.Errorf("table %s not found", tableName)
	}

	var storageKey string
	var record map[string]interface{}
	for candidateKey, candidate := range tableData {
		if recordMatchesKeys(candidate, keyValues) {
			storageKey = candidateKey
			record = candidate
			break
		}
	}
	if record == nil {
		return fmt.Errorf("record not found")
	}

	if softDelete {
		now := time.Now().Format(time.RFC3339)
		record[timestampColumn] = now
		record["deleted_at"] = now
		return nil
	}

	delete(tableData, storageKey)
	return nil
}

func recordMatchesKeys(record map[string]interface{}, keyValues map[string]string) bool {
	if len(keyValues) == 0 {
		return false
	}

	for key, expected := range keyValues {
		actual, exists := record[key]
		if !exists || fmt.Sprintf("%v", actual) != expected {
			return false
		}
	}
	return true
}
