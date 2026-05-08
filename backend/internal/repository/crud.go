package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CrudRepository struct {
	db *DBWrapper
}

func NewCrudRepository(db *DBWrapper) *CrudRepository {
	return &CrudRepository{db: db}
}

// ListRecords retrieves paginated, sortable, filterable records from any table
func (r *CrudRepository) ListRecords(ctx context.Context, tableName string, page, pageSize int, sortBy, sortOrder string, filters map[string]string) ([]map[string]interface{}, int64, error) {
	if r.db.IsInMemory() {
		return r.db.InMemory().ListRecords(ctx, tableName, page, pageSize, sortBy, sortOrder, filters)
	}

	if !isValidIdentifier(tableName) {
		return nil, 0, fmt.Errorf("invalid table name: %s", tableName)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}
	orderClause := ""
	if sortBy != "" {
		if !isValidIdentifier(sortBy) {
			return nil, 0, fmt.Errorf("invalid sort column: %s", sortBy)
		}
		orderClause = fmt.Sprintf(" ORDER BY %s %s", quoteIdentifier(sortBy), sortOrder)
	}

	whereClause := ""
	whereValues := []interface{}{}
	argIdx := 1

	if len(filters) > 0 {
		var conditions []string
		for col, val := range filters {
			if !isValidIdentifier(col) || val == "" {
				continue
			}
			quotedCol := quoteIdentifier(col)
			if strings.HasPrefix(val, "like:") {
				conditions = append(conditions, fmt.Sprintf("%s::text ILIKE $%d", quotedCol, argIdx))
				whereValues = append(whereValues, "%"+strings.TrimPrefix(val, "like:")+"%")
			} else if strings.HasPrefix(val, "gt:") {
				conditions = append(conditions, fmt.Sprintf("%s > $%d", quotedCol, argIdx))
				whereValues = append(whereValues, strings.TrimPrefix(val, "gt:"))
			} else if strings.HasPrefix(val, "lt:") {
				conditions = append(conditions, fmt.Sprintf("%s < $%d", quotedCol, argIdx))
				whereValues = append(whereValues, strings.TrimPrefix(val, "lt:"))
			} else if strings.HasPrefix(val, "ne:") {
				conditions = append(conditions, fmt.Sprintf("%s::text != $%d", quotedCol, argIdx))
				whereValues = append(whereValues, strings.TrimPrefix(val, "ne:"))
			} else {
				conditions = append(conditions, fmt.Sprintf("%s::text = $%d", quotedCol, argIdx))
				whereValues = append(whereValues, val)
			}
			argIdx++
		}
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	pool := r.db.Pool()

	quotedTable := quoteIdentifier(tableName)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quotedTable, whereClause)
	var total int64
	row := pool.QueryRow(ctx, countQuery, whereValues...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count records: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT * FROM %s%s%s LIMIT $%d OFFSET $%d",
		quotedTable, whereClause, orderClause, argIdx, argIdx+1,
	)
	whereValues = append(whereValues, pageSize, offset)

	rows, err := pool.Query(ctx, query, whereValues...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read row values: %w", err)
		}
		fieldDescriptions := rows.FieldDescriptions()
		record := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			if i < len(values) {
				record[fd.Name] = convertValue(values[i])
			}
		}
		records = append(records, record)
	}

	return records, total, nil
}

// GetRecord retrieves a single record by primary key values from any table
func (r *CrudRepository) GetRecord(ctx context.Context, tableName string, keyValues map[string]string) (map[string]interface{}, error) {
	if r.db.IsInMemory() {
		return r.db.InMemory().GetRecord(ctx, tableName, keyValues)
	}

	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}
	whereClause, values, _, err := buildKeyWhereClause(keyValues, 1)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", quoteIdentifier(tableName), whereClause)
	pool := r.db.Pool()
	rows, err := pool.Query(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("failed to query record: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to read values: %w", err)
		}
		fieldDescriptions := rows.FieldDescriptions()
		result := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			if i < len(values) {
				result[fd.Name] = convertValue(values[i])
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("record not found")
}

// CreateRecord inserts a new record into any table
func (r *CrudRepository) CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	if r.db.IsInMemory() {
		return r.db.InMemory().CreateRecord(ctx, tableName, data)
	}

	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided for insert")
	}

	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	idx := 1
	for col, val := range data {
		if !isValidIdentifier(col) {
			continue
		}
		columns = append(columns, quoteIdentifier(col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		values = append(values, val)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid columns provided for insert")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		quoteIdentifier(tableName),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	pool := r.db.Pool()
	rows, err := pool.Query(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("failed to insert record: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		rowValues, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to read inserted values: %w", err)
		}
		fieldDescriptions := rows.FieldDescriptions()
		record := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			if i < len(rowValues) {
				record[fd.Name] = convertValue(rowValues[i])
			}
		}
		return record, nil
	}

	return nil, fmt.Errorf("no data returned after insert")
}

// UpdateRecord updates an existing record in any table
func (r *CrudRepository) UpdateRecord(ctx context.Context, tableName string, keyValues map[string]string, data map[string]interface{}) (map[string]interface{}, error) {
	if r.db.IsInMemory() {
		return r.db.InMemory().UpdateRecord(ctx, tableName, keyValues, data)
	}

	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided for update")
	}

	for key := range keyValues {
		delete(data, key)
	}

	setClauses := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+1)
	idx := 1

	for col, val := range data {
		if !isValidIdentifier(col) {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), idx))
		values = append(values, val)
		idx++
	}
	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no updateable columns provided")
	}

	whereClause, keyArgs, _, err := buildKeyWhereClause(keyValues, idx)
	if err != nil {
		return nil, err
	}
	values = append(values, keyArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING *",
		quoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		whereClause,
	)

	pool := r.db.Pool()
	rows, err := pool.Query(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		rowValues, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to read updated values: %w", err)
		}
		fieldDescriptions := rows.FieldDescriptions()
		record := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			if i < len(rowValues) {
				record[fd.Name] = convertValue(rowValues[i])
			}
		}
		return record, nil
	}

	return nil, fmt.Errorf("record not found")
}

// DeleteRecord deletes a record from any table
func (r *CrudRepository) DeleteRecord(ctx context.Context, tableName string, keyValues map[string]string, softDelete bool, timestampColumn string) error {
	if r.db.IsInMemory() {
		return r.db.InMemory().DeleteRecord(ctx, tableName, keyValues, softDelete, timestampColumn)
	}

	if !isValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name")
	}
	whereClause, values, _, err := buildKeyWhereClause(keyValues, 1)
	if err != nil {
		return err
	}

	pool := r.db.Pool()
	quotedTable := quoteIdentifier(tableName)

	if softDelete && isValidIdentifier(timestampColumn) {
		query := fmt.Sprintf("UPDATE %s SET %s = NOW() WHERE %s", quotedTable, quoteIdentifier(timestampColumn), whereClause)
		result, err := pool.Exec(ctx, query, values...)
		if err != nil {
			return fmt.Errorf("failed to soft delete record: %w", err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("record not found")
		}
		return nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", quotedTable, whereClause)
	result, err := pool.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

func buildKeyWhereClause(keyValues map[string]string, startArgIdx int) (string, []interface{}, int, error) {
	if len(keyValues) == 0 {
		return "", nil, startArgIdx, fmt.Errorf("no primary key values provided")
	}

	keys := make([]string, 0, len(keyValues))
	for key := range keyValues {
		if !isValidIdentifier(key) {
			return "", nil, startArgIdx, fmt.Errorf("invalid primary key column: %s", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	conditions := make([]string, 0, len(keys))
	values := make([]interface{}, 0, len(keys))
	argIdx := startArgIdx
	for _, key := range keys {
		conditions = append(conditions, fmt.Sprintf("%s::text = $%d", quoteIdentifier(key), argIdx))
		values = append(values, keyValues[key])
		argIdx++
	}

	return strings.Join(conditions, " AND "), values, argIdx, nil
}

func convertValue(val interface{}) interface{} {
	switch v := val.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case [16]byte:
		return formatUUID(v[:])
	case []byte:
		if isValidJSON(v) {
			var jsonData interface{}
			if err := json.Unmarshal(v, &jsonData); err == nil {
				return jsonData
			}
		}
		if len(v) == 16 {
			return formatUUID(v)
		}
		return string(v)
	default:
		return val
	}
}

func formatUUID(b []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
	}
	return true
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func isValidJSON(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if b[0] != '{' && b[0] != '[' {
		return false
	}
	var js json.RawMessage
	return json.Unmarshal(b, &js) == nil
}
