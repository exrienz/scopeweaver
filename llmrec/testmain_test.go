package llmrec

import (
	"database/sql"
	"os"
	"testing"

	"github.com/Autumn-27/artex/internal/dbtest"
)

// Coordinate schema initialization and cleanup with every other DB-backed suite.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestDatabaseSuiteOwnsSharedLock(t *testing.T) {
	conn, err := sql.Open("pgx", dbtest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c, err := conn.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var acquired bool
	if err := c.QueryRowContext(t.Context(), "SELECT pg_try_advisory_lock($1)", int64(7337741002)).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if acquired {
		_, _ = c.ExecContext(t.Context(), "SELECT pg_advisory_unlock($1)", int64(7337741002))
		t.Fatal("llmrec suite does not hold the shared database advisory lock")
	}
}
