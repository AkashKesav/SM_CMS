package repository

import (
	"context"
	"fmt"
	"time"

	"cms-backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type SchemaRepository struct {
	db *DBWrapper
}

func NewSchemaRepository(db *DBWrapper) *SchemaRepository {
	return &SchemaRepository{db: db}
}

// DiscoverTables queries PostgreSQL system catalogs or returns in-memory schema
func (r *SchemaRepository) DiscoverTables(ctx context.Context) ([]model.TableSchema, error) {
	if r.db.IsInMemory() {
		return r.db.InMemory().DiscoverTables(ctx)
	}

	pool := r.db.Pool()
	var tables []model.TableSchema

	// Step 1: Discover all user tables
	tableQuery := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_type = 'BASE TABLE'
			AND table_name NOT LIKE 'pg_%'
			AND table_name != 'information_schema'
			AND table_name != 'audit_log'
		ORDER BY table_name;
	`

	rows, err := pool.Query(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tableNames = append(tableNames, tableName)
	}

	// Step 2: For each table, discover columns and constraints
	for _, tableName := range tableNames {
		tableSchema, err := r.discoverTableDetails(ctx, pool, tableName)
		if err != nil {
			log.Warn().Err(err).Str("table", tableName).Msg("Failed to discover table details")
			continue
		}
		tables = append(tables, *tableSchema)
	}

	return tables, nil
}

// discoverTableDetails discovers columns, primary keys, and foreign keys for a specific table
func (r *SchemaRepository) discoverTableDetails(ctx context.Context, pool *pgxpool.Pool, tableName string) (*model.TableSchema, error) {
	schema := &model.TableSchema{
		TableName: tableName,
	}

	// Discover columns
	columnQuery := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.udt_name,
			COALESCE(
				(SELECT true
				 FROM information_schema.table_constraints tc
				 JOIN information_schema.key_column_usage kcu
				 ON tc.constraint_name = kcu.constraint_name
				 AND tc.table_schema = kcu.table_schema
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				 AND tc.table_name = c.table_name
				 AND kcu.column_name = c.column_name),
				false
			) as is_primary_key,
			COALESCE(
				(SELECT true
				 FROM information_schema.table_constraints tc
				 JOIN information_schema.key_column_usage kcu
				 ON tc.constraint_name = kcu.constraint_name
				 AND tc.table_schema = kcu.table_schema
				 WHERE tc.constraint_type = 'FOREIGN KEY'
				 AND tc.table_name = c.table_name
				 AND kcu.column_name = c.column_name),
				false
			) as is_foreign_key
		FROM information_schema.columns c
		WHERE c.table_name = $1
			AND c.table_schema = 'public'
		ORDER BY c.ordinal_position;
	`

	rows, err := pool.Query(ctx, columnQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to discover columns for %s: %w", tableName, err)
	}
	defer rows.Close()

	var hasCreatedAt, hasUpdatedAt bool
	for rows.Next() {
		var col model.ColumnInfo
		err := rows.Scan(
			&col.ColumnName,
			&col.DataType,
			&col.IsNullable,
			&col.ColumnDefault,
			&col.CharacterMaximumLength,
			&col.NumericPrecision,
			&col.UdtName,
			&col.IsPrimaryKey,
			&col.IsForeignKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		schema.Columns = append(schema.Columns, col)

		if col.IsPrimaryKey {
			if schema.PrimaryKey == "" {
				schema.PrimaryKey = col.ColumnName
			}
			schema.PrimaryKeys = append(schema.PrimaryKeys, col.ColumnName)
		}
		if col.ColumnName == "created_at" {
			hasCreatedAt = true
		}
		if col.ColumnName == "updated_at" {
			hasUpdatedAt = true
		}
	}

	schema.HasTimestamps = hasCreatedAt && hasUpdatedAt

	// Discover foreign keys
	fkQuery := `
		SELECT
			tc.table_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1;
	`

	fkRows, err := pool.Query(ctx, fkQuery, tableName)
	if err != nil {
		log.Warn().Err(err).Str("table", tableName).Msg("Failed to discover foreign keys")
	} else {
		defer fkRows.Close()
		for fkRows.Next() {
			var fk model.ForeignKeyInfo
			if err := fkRows.Scan(
				&fk.TableName,
				&fk.ColumnName,
				&fk.ForeignTableName,
				&fk.ForeignColumnName,
				&fk.ConstraintName,
			); err != nil {
				log.Warn().Err(err).Str("table", tableName).Msg("Failed to scan foreign key")
				continue
			}
			schema.ForeignKeys = append(schema.ForeignKeys, fk)
		}
	}

	return schema, nil
}

// NewPostgresDB creates a new connection pool to PostgreSQL and returns a DBWrapper
func NewPostgresDB(databaseURL string) (*DBWrapper, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Info().Msg("Successfully connected to PostgreSQL")
	return &DBWrapper{pool: pool, isInMem: false, modeLabel: "production"}, nil
}
