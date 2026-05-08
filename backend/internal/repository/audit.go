package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Action    string      `json:"action"`
	TableName string      `json:"table_name"`
	RecordID  string      `json:"record_id"`
	OldValues interface{} `json:"old_values,omitempty"`
	NewValues interface{} `json:"new_values,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// AuditLogger provides async audit logging
type AuditLogger struct {
	db    *DBWrapper
	ch    chan AuditEntry
	quit  chan struct{}
	wg    sync.WaitGroup
}

// NewAuditLogger creates a new async audit logger
func NewAuditLogger(db *DBWrapper) *AuditLogger {
	al := &AuditLogger{
		db:   db,
		ch:   make(chan AuditEntry, 1000),
		quit: make(chan struct{}),
	}

	// Start the background writer
	al.wg.Add(1)
	go al.processEntries()

	return al
}

// Log queues an audit entry for async writing
func (al *AuditLogger) Log(entry AuditEntry) {
	entry.CreatedAt = time.Now()

	// For in-memory mode, just log it
	if al.db.IsInMemory() {
		log.Debug().
			Str("action", entry.Action).
			Str("table", entry.TableName).
			Str("record_id", entry.RecordID).
			Str("user_id", entry.UserID).
			Msg("Audit log")
		return
	}

	select {
	case al.ch <- entry:
	default:
		log.Warn().Msg("Audit log channel full — dropping entry")
	}
}

// Close stops the audit logger and flushes pending entries
func (al *AuditLogger) Close() {
	close(al.quit)
	al.wg.Wait()
}

func (al *AuditLogger) processEntries() {
	defer al.wg.Done()

	for {
		select {
		case entry := <-al.ch:
			al.writeEntry(entry)
		case <-al.quit:
			// Drain remaining entries
			for {
				select {
				case entry := <-al.ch:
					al.writeEntry(entry)
				default:
					return
				}
			}
		}
	}
}

func (al *AuditLogger) writeEntry(entry AuditEntry) {
	pool := al.db.Pool()
	if pool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var oldJSON, newJSON interface{}
	if entry.OldValues != nil {
		if b, err := json.Marshal(entry.OldValues); err == nil {
			oldJSON = string(b)
		}
	}
	if entry.NewValues != nil {
		if b, err := json.Marshal(entry.NewValues); err == nil {
			newJSON = string(b)
		}
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO audit_log (user_id, action, table_name, record_id, old_values, new_values, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)`,
		entry.UserID, entry.Action, entry.TableName, entry.RecordID, oldJSON, newJSON, entry.CreatedAt,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write audit log entry")
	}
}

// EnsureAuditTable creates the audit_log table if it doesn't exist
func EnsureAuditTable(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
		user_id TEXT,
		action TEXT NOT NULL,
		table_name TEXT NOT NULL,
		record_id TEXT,
		old_values JSONB,
		new_values JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_audit_log_table ON audit_log(table_name);
	CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
	`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create audit_log table: %w", err)
	}

	log.Info().Msg("Audit log table ensured")
	return nil
}
