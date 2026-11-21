//go:build integration

package mariadb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
)

func TestMariaDBRepositoryIntegrationDraftAndReportImmutability(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()

	draft := draftDocument("draft-immutable", "orders.jsonl")
	if err := store.AppendDraft(ctx, draft, "integration-user"); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendDraft(ctx, draft, "integration-user"); err != nil {
		t.Fatalf("idempotent draft append failed: %v", err)
	}
	if err := store.AppendDraft(ctx, draftDocument("draft-immutable", "changed.jsonl"), "integration-user"); failureCategory(err) != domain.FailureConflict {
		t.Fatalf("mutated draft append error=%v, want conflict", err)
	}
	loadedDraft, err := store.LoadDraft(ctx, "draft-immutable")
	if err != nil || string(loadedDraft) != string(draft) {
		t.Fatalf("loaded draft=%s err=%v", loadedDraft, err)
	}

	report := metadata.Validate(draft)
	if err := store.Append(ctx, report); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, report); err != nil {
		t.Fatalf("idempotent report append failed: %v", err)
	}
	mutatedReport := report
	mutatedReport.Valid = false
	mutatedReport.CompatibilityDecision = "rejected"
	mutatedReport.Diagnostics = []metadata.SchemaDiagnostic{{
		Code: "metadata_invalid", Path: "/", Message: "metadata was rejected by the v1 validator",
	}}
	if err := store.Append(ctx, mutatedReport); failureCategory(err) != domain.FailureConflict {
		t.Fatalf("mutated report append error=%v, want conflict", err)
	}
	loadedReport, err := store.Load(ctx, report.InputDigest)
	if err != nil || !reflect.DeepEqual(loadedReport, report) {
		t.Fatalf("loaded report=%+v err=%v", loadedReport, err)
	}
}

func TestMariaDBRepositoryIntegrationCompetingExpectedActiveUpdates(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	initial := activeDocument("initial")
	if _, err := store.Activate(ctx, initial, ""); err != nil {
		t.Fatal(err)
	}

	candidates := [][]byte{activeDocument("candidate-a"), activeDocument("candidate-b")}
	start := make(chan struct{})
	results := make(chan error, len(candidates))
	var wait sync.WaitGroup
	for _, candidate := range candidates {
		wait.Add(1)
		go func(candidate []byte) {
			defer wait.Done()
			<-start
			_, err := store.Activate(ctx, candidate, "initial")
			results <- err
		}(candidate)
	}
	close(start)
	wait.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if failureCategory(err) == domain.FailureConflict {
			conflicts++
		} else {
			t.Fatalf("unexpected activation error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("activation successes=%d conflicts=%d, want one of each", successes, conflicts)
	}
	active, err := store.LoadActive(ctx)
	if err != nil || (string(active) != string(candidates[0]) && string(active) != string(candidates[1])) {
		t.Fatalf("active metadata=%s err=%v", active, err)
	}
	var historyCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata_activation_history`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 {
		t.Fatalf("activation history rows=%d, want initial and winning activation", historyCount)
	}
}

func TestMariaDBRepositoryIntegrationActivationPointerAndHistoryAreAtomic(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	initial := activeDocument("atomic-initial")
	if _, err := store.Activate(ctx, initial, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `CREATE TRIGGER mimir_test_reject_activation_history
		BEFORE INSERT ON metadata_activation_history FOR EACH ROW
		SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected test failure'`); err != nil {
		t.Fatalf("create test trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS mimir_test_reject_activation_history`)
	})

	candidate := activeDocument("atomic-candidate")
	if _, err := store.Activate(ctx, candidate, "atomic-initial"); err == nil {
		t.Fatal("activation unexpectedly succeeded when history insert failed")
	}
	active, err := store.LoadActive(ctx)
	if err != nil || string(active) != string(initial) {
		t.Fatalf("active pointer changed after failed activation: active=%s err=%v", active, err)
	}
	var revisionCount, historyCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata_revisions
		WHERE revision_id = 'atomic-candidate' AND revision_status = 'active'`).Scan(&revisionCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata_activation_history`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if revisionCount != 0 || historyCount != 1 {
		t.Fatalf("rolled-back revision rows=%d history rows=%d, want 0 and 1", revisionCount, historyCount)
	}
}

func TestMariaDBRepositoryIntegrationReconcilesUncertainCommits(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()

	uncertainDB := sql.OpenDB(uncertainCommitConnector{dsn: integrationDSN(t)})
	defer uncertainDB.Close()
	uncertainStore := &Store{db: uncertainDB}

	draft := draftDocument("uncertain-draft", "orders.jsonl")
	if err := uncertainStore.AppendDraft(ctx, draft, "integration-user"); failureCategory(err) != domain.FailureAmbiguousCommit {
		t.Fatalf("uncertain draft commit error=%v, want ambiguous commit", err)
	}
	if err := store.AppendDraft(ctx, draft, "integration-user"); err != nil {
		t.Fatalf("draft retry did not reconcile committed write: %v", err)
	}
	if _, err := store.LoadDraft(ctx, "uncertain-draft"); err != nil {
		t.Fatalf("committed draft is unavailable after reconciliation: %v", err)
	}

	active := activeDocument("uncertain-active")
	if _, err := uncertainStore.Activate(ctx, active, ""); failureCategory(err) != domain.FailureAmbiguousCommit {
		t.Fatalf("uncertain activation commit error=%v, want ambiguous commit", err)
	}
	if loaded, err := store.LoadActive(ctx); err != nil || string(loaded) != string(active) {
		t.Fatalf("committed active metadata=%s err=%v", loaded, err)
	}
	if previous, err := store.Activate(ctx, active, ""); err != nil || previous != "" {
		t.Fatalf("activation retry previous=%q err=%v", previous, err)
	}
	var historyCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata_activation_history`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 1 {
		t.Fatalf("uncertain activation history rows=%d, want exactly one", historyCount)
	}
}

func TestMariaDBRepositoryIntegrationMigrationAndErrorHandling(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	var migrationCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata_schema_migrations WHERE version = 1`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 1 {
		t.Fatalf("initial migration rows=%d, want one", migrationCount)
	}

	config := integrationConfig(t)
	wrongCredentials := config
	wrongCredentials.RuntimePassword += "-incorrect"
	wrongCredentials.MigrationPassword += "-incorrect"
	openCtx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
	_, openErr := Open(openCtx, wrongCredentials)
	cancel()
	if failureCategory(openErr) != domain.FailureUnavailable {
		t.Fatalf("open with invalid credentials error=%v, want unavailable", openErr)
	}

	cleanupDB, err := openDB(config.Address, databaseName, config.MigrationUser, config.MigrationPassword, config.ConnectTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = cleanupDB.ExecContext(context.Background(), `DELETE FROM metadata_schema_migrations WHERE version = 999`)
		_ = cleanupDB.Close()
	})
	if _, err := cleanupDB.ExecContext(ctx, `INSERT INTO metadata_schema_migrations (version) VALUES (999)`); err != nil {
		t.Fatal(err)
	}
	store.Close()
	openCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
	_, openErr = Open(openCtx, config)
	cancel()
	if failureCategory(openErr) != domain.FailureUnsupportedCapability {
		t.Fatalf("open with newer schema error=%v, want unsupported capability", openErr)
	}
}

func TestMariaDBRepositoryIntegrationInjectedStoreAuditRecovery(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	stateDir := t.TempDir()
	if err := os.Chmod(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	auditPath := filepath.Join(stateDir, "audit")
	if err := os.WriteFile(auditPath, []byte("blocking audit directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	draft := draftDocument("audit-recovery", "orders.jsonl")

	_, authorErr := local.AuthorBytesWithStores(ctx, draft, stateDir, "local-user", true, store, store)
	if failureCategory(authorErr) != domain.FailureUnavailable {
		t.Fatalf("authoring without writable audit storage error=%v, want unavailable", authorErr)
	}
	if _, err := store.LoadDraft(ctx, "audit-recovery"); err != nil {
		t.Fatalf("database draft was not committed before audit recovery: author error=%v load error=%v", authorErr, err)
	}
	if err := os.Remove(auditPath); err != nil {
		t.Fatal(err)
	}
	if _, err := local.AuthorBytesWithStores(ctx, draft, stateDir, "local-user", true, store, store); err != nil {
		t.Fatalf("author retry did not reconcile the required audit event: %v", err)
	}
	audit, err := os.ReadFile(filepath.Join(stateDir, "audit", "events.jsonl"))
	if err != nil || !strings.Contains(string(audit), `"operation":"metadata.authored"`) {
		t.Fatalf("authored audit event missing: err=%v", err)
	}
}

func integrationStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), integrationConfig(t))
	if err != nil {
		t.Fatalf("open integration MariaDB: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	resetIntegrationStore(t, store)
	return store
}

func integrationConfig(t *testing.T) Config {
	t.Helper()
	address := os.Getenv("MIMIR_MARIADB_TEST_ADDRESS")
	user := os.Getenv("MIMIR_MARIADB_TEST_USER")
	password := os.Getenv("MIMIR_MARIADB_TEST_PASSWORD")
	if address == "" || user == "" || password == "" {
		t.Skip("MariaDB integration credentials are not configured")
	}
	return Config{
		Address: address, RuntimeUser: user, RuntimePassword: password,
		MigrationUser: user, MigrationPassword: password,
		ConnectTimeout: 2 * time.Second, ReconnectInterval: 100 * time.Millisecond,
	}
}

func resetIntegrationStore(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `DROP TRIGGER IF EXISTS mimir_test_reject_activation_history`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DELETE FROM metadata_activation_history`,
		`UPDATE metadata_active SET revision_id = '', metadata_digest = '', activated_at = NULL WHERE singleton_id = 1`,
		`DELETE FROM metadata_revisions`,
		`DELETE FROM metadata_validation_reports`,
		`DELETE FROM metadata_discovery_proposals`,
	} {
		if _, err := store.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
}

func integrationDSN(t *testing.T) string {
	t.Helper()
	config := integrationConfig(t)
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = config.RuntimeUser
	mysqlConfig.Passwd = config.RuntimePassword
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = config.Address
	mysqlConfig.DBName = databaseName
	mysqlConfig.ParseTime = true
	mysqlConfig.Loc = time.UTC
	mysqlConfig.Collation = "utf8mb4_bin"
	return mysqlConfig.FormatDSN()
}

func draftDocument(revision, path string) []byte {
	return []byte(`{"schemaVersion":"mimir.metadata/v1","revision":"` + revision + `","status":"draft","sources":[{"id":"orders","classification":"internal","objects":[{"id":"orders","path":"` + path + `"}]}]}`)
}

func activeDocument(revision string) []byte {
	return []byte(strings.Replace(string(draftDocument(revision, "orders.jsonl")), `"status":"draft"`, `"status":"active"`, 1))
}

func failureCategory(err error) domain.FailureCategory {
	var failure domain.Failure
	if errors.As(err, &failure) {
		return failure.Category
	}
	return ""
}

type uncertainCommitConnector struct{ dsn string }

func (c uncertainCommitConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := (&mysql.MySQLDriver{}).Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return uncertainCommitConn{Conn: conn}, nil
}

func (c uncertainCommitConnector) Driver() driver.Driver {
	return uncertainCommitDriver{connector: c}
}

type uncertainCommitDriver struct{ connector uncertainCommitConnector }

func (d uncertainCommitDriver) Open(string) (driver.Conn, error) {
	return d.connector.Connect(context.Background())
}

type uncertainCommitConn struct{ driver.Conn }

func (c uncertainCommitConn) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	beginner, ok := c.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, driver.ErrSkip
	}
	tx, err := beginner.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return uncertainCommitTx{Tx: tx}, nil
}

type uncertainCommitTx struct{ driver.Tx }

func (tx uncertainCommitTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	return errors.New("injected lost commit acknowledgement")
}
