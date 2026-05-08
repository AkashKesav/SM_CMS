package model

// ColumnInfo represents metadata about a database column
type ColumnInfo struct {
	ColumnName             string  `json:"column_name"`
	DataType               string  `json:"data_type"`
	IsNullable             string  `json:"is_nullable"`
	ColumnDefault          *string `json:"column_default"`
	CharacterMaximumLength *int    `json:"character_maximum_length"`
	NumericPrecision       *int    `json:"numeric_precision"`
	UdtName                string  `json:"udt_name"`
	IsPrimaryKey           bool    `json:"is_primary_key"`
	IsForeignKey           bool    `json:"is_foreign_key"`
}

// ForeignKeyInfo represents a foreign key relationship
type ForeignKeyInfo struct {
	TableName         string `json:"table_name"`
	ColumnName        string `json:"column_name"`
	ForeignTableName  string `json:"foreign_table_name"`
	ForeignColumnName string `json:"foreign_column_name"`
	ConstraintName    string `json:"constraint_name"`
}

// TableSchema represents the complete schema for a single table
type TableSchema struct {
	TableName     string           `json:"table_name"`
	Columns       []ColumnInfo     `json:"columns"`
	ForeignKeys   []ForeignKeyInfo `json:"foreign_keys"`
	PrimaryKey    string           `json:"primary_key"`
	PrimaryKeys   []string         `json:"primary_keys"`
	HasTimestamps bool             `json:"has_timestamps"`
}

// DatabaseSchema represents the complete discovered database schema
type DatabaseSchema struct {
	Tables       []TableSchema `json:"tables"`
	DiscoveredAt string        `json:"discovered_at"`
	Version      int           `json:"version"`
}

// ColumnType represents the UI type for rendering form fields
type ColumnType string

const (
	TypeText        ColumnType = "text"
	TypeNumber      ColumnType = "number"
	TypeBoolean     ColumnType = "boolean"
	TypeDate        ColumnType = "date"
	TypeDateTime    ColumnType = "datetime"
	TypeJSON        ColumnType = "json"
	TypeUUID        ColumnType = "uuid"
	TypeEmail       ColumnType = "email"
	TypeURL         ColumnType = "url"
	TypeTextArea    ColumnType = "textarea"
	TypeImage       ColumnType = "image"
	TypeSelect      ColumnType = "select"
	TypeMultiSelect ColumnType = "multiselect"
	TypeCode        ColumnType = "code"
)
