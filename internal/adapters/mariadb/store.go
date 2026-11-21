// Package mariadb provides SQL-backed metadata stores for the container profile.
package mariadb

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/ports"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const databaseName = "metadata"

type Config struct {
	Address           string
	RuntimeUser       string
	RuntimePassword   string
	MigrationUser     string
	MigrationPassword string
	ConnectTimeout    time.Duration
	ReconnectInterval time.Duration
}

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, config Config) (*Store, error) {
	if config.Address == "" || config.RuntimeUser == "" || config.RuntimePassword == "" ||
		config.MigrationUser == "" || config.MigrationPassword == "" {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "mariadb.configure", "database address and separate runtime and migration credentials are required")
	}
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = 5 * time.Second
	}
	if config.ReconnectInterval <= 0 {
		config.ReconnectInterval = time.Second
	}

	migrationDB, err := openDB(config.Address, databaseName, config.MigrationUser, config.MigrationPassword, config.ConnectTimeout)
	if err != nil {
		return nil, domain.NewFailure(domain.FailureUnavailable, "mariadb.connect", "metadata database is unavailable")
	}
	defer migrationDB.Close()
	if err := waitReady(ctx, migrationDB, config.ReconnectInterval); err != nil {
		return nil, err
	}
	if err := migrate(ctx, migrationDB); err != nil {
		return nil, err
	}

	runtimeDB, err := openDB(config.Address, databaseName, config.RuntimeUser, config.RuntimePassword, config.ConnectTimeout)
	if err != nil {
		return nil, domain.NewFailure(domain.FailureUnavailable, "mariadb.connect", "metadata database is unavailable")
	}
	if err := waitReady(ctx, runtimeDB, config.ReconnectInterval); err != nil {
		_ = runtimeDB.Close()
		return nil, err
	}
	runtimeDB.SetMaxOpenConns(8)
	runtimeDB.SetMaxIdleConns(4)
	runtimeDB.SetConnMaxLifetime(5 * time.Minute)
	return &Store{db: runtimeDB}, nil
}

func openDB(address, database, user, password string, timeout time.Duration) (*sql.DB, error) {
	config := mysql.NewConfig()
	config.User = user
	config.Passwd = password
	config.Net = "tcp"
	config.Addr = address
	config.DBName = database
	config.ParseTime = true
	config.Loc = time.UTC
	config.Timeout = timeout
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	config.Collation = "utf8mb4_bin"
	return sql.Open("mysql", config.FormatDSN())
}

func waitReady(ctx context.Context, db *sql.DB, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return domain.NewFailure(domain.FailureUnavailable, "mariadb.connect", "metadata database did not become ready")
		}
		select {
		case <-ctx.Done():
			return domain.NewFailure(domain.FailureUnavailable, "mariadb.connect", "metadata database did not become ready")
		case <-ticker.C:
		}
	}
}

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metadata_schema_migrations (
		version INT NOT NULL PRIMARY KEY,
		applied_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB`); err != nil {
		return databaseFailure("mariadb.migrate", err)
	}
	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM metadata_schema_migrations`).Scan(&current); err != nil {
		return databaseFailure("mariadb.migrate", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return domain.NewFailure(domain.FailureInternal, "mariadb.migrate", "embedded metadata migrations are unavailable")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	maxVersion := 0
	for _, entry := range entries {
		version, ok := migrationVersion(entry.Name())
		if !ok {
			return domain.NewFailure(domain.FailureInternal, "mariadb.migrate", "embedded metadata migration name is invalid")
		}
		if version > maxVersion {
			maxVersion = version
		}
		if version <= current {
			continue
		}
		raw, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return domain.NewFailure(domain.FailureInternal, "mariadb.migrate", "embedded metadata migration cannot be read")
		}
		for _, statement := range strings.Split(string(raw), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return databaseFailure("mariadb.migrate", err)
			}
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO metadata_schema_migrations (version) VALUES (?)`, version); err != nil {
			return databaseFailure("mariadb.migrate", err)
		}
	}
	if current > maxVersion {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "mariadb.migrate", "metadata database schema is newer than this backend")
	}
	return nil
}

func migrationVersion(name string) (int, bool) {
	var version int
	if _, err := fmt.Sscanf(name, "%03d_", &version); err != nil || version < 1 {
		return 0, false
	}
	return version, true
}

func databaseFailure(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return domain.ContextFailure(operation, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.ContextFailure(operation, context.DeadlineExceeded)
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1205, 1213:
			failure := domain.NewFailure(domain.FailureConflict, operation, "metadata database transaction conflicted")
			failure.Retryable = true
			return failure
		}
	}
	return domain.NewFailure(domain.FailureUnavailable, operation, "metadata database operation is unavailable")
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return domain.NewFailure(domain.FailureUnavailable, "mariadb.ready", "metadata database is unavailable")
	}
	if err := s.db.PingContext(ctx); err != nil {
		return databaseFailure("mariadb.ready", err)
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Capabilities() ports.CapabilitySet {
	return ports.CapabilitySet{
		ContractVersion: ports.MetadataRepositoryVersion,
		Operations:      []string{"append_draft", "load_draft", "list_revisions", "load_active", "conditional_activate"},
		Unsupported:     []string{"automatic_file_migration", "proposal_queries"},
		Concurrency:     8,
		MaximumBytes:    1 << 20,
		Classifications: []string{"public", "internal"},
	}
}
