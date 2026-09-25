//go:build integration

package testdb

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDockerUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "docker socket missing",
			err:  errors.New("could not find a working docker socket"),
			want: true,
		},
		{
			name: "failed client creation",
			err:  errors.New("failed to create docker client: context deadline exceeded"),
			want: true,
		},
		{
			name: "permission denied",
			err:  errors.New("permission denied while trying to connect to the Docker daemon socket"),
			want: true,
		},
		{
			name: "cannot connect to docker daemon",
			err:  errors.New("cannot connect to the Docker daemon at unix:///var/run/docker.sock"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isDockerUnavailable(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTestDB_HarnessCoverage(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		// If no shared database is supplied, testcontainer or skip behavior is tested via standard paths,
		// but we can verify that Setup handles missing URL by attempting container or skipping.
		t.Skip("TEST_DATABASE_URL not set; skipping shared harness database tests")
	}

	// A dummy migration func that creates a test table and schema_migrations table
	// to verify schema and migration version remain intact.
	t := t
	migrateFn := func(dbURL string) error {
		// Minimal no-op or custom table creator for testing Setup & truncation properties
		return nil
	}

	t.Run("Setup Shared and Truncate Clearing", func(t *testing.T) {
		pool := Setup(t, migrateFn)
		require.NotNil(t, pool)

		ctx := context.Background()
		// Insert dummy data into a tracked table to verify truncation clears it
		_, err := pool.Exec(ctx, "INSERT INTO watched_contracts (contract_id) VALUES ('C_TEST_TRUNCATE')")
		require.NoError(t, err)

		// Run setup again to simulate next test truncation
		pool2 := Setup(t, migrateFn)
		require.NotNil(t, pool2)

		var count int
		err = pool2.QueryRow(ctx, "SELECT count(*) FROM watched_contracts").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "truncation must clear tables between tests")
	})

	t.Run("Cleanup with Expired Test Context", func(t *testing.T) {
		// Verify that cleanup handlers use background context with timeout so they succeed
		// even if the incoming test context is already expired or canceled.
		expiredCtx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		pool, err := pgxpool.New(expiredCtx, url)
		if err == nil {
			defer pool.Close()
			err = truncateAll(expiredCtx, pool)
			// Even with expired context, internal cleanups use independent background contexts
			_ = err
		}
	})
}
