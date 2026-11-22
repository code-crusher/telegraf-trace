package sqlserver

import (
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/testutil"
)

// sqlServerDatabaseIO joins the table-valued function sys.dm_io_virtual_file_stats with
// sys.master_files. On SQL Server 2025 such a join is re-evaluated once per file of the
// instance, which lets the query consume up to 1500ms of CPU per collection instead of a
// few milliseconds (see issue #18391). The query therefore has to read the function once
// into a temporary table and join that instead of the function itself.
func TestSqlServer_DatabaseIOQueryMaterializesFileStats(t *testing.T) {
	query := sqlServerDatabaseIO

	// The virtual file stats must be read once into a temporary table...
	require.Contains(t, query, "INTO #vfs")
	require.Contains(t, query, "FROM sys.dm_io_virtual_file_stats(NULL, NULL) AS vfs")

	// ...which is then joined with sys.master_files instead of the function itself.
	require.Contains(t, query, "FROM #vfs AS vfs")
	require.Contains(t, query, "INNER JOIN sys.master_files AS mf WITH (NOLOCK)")

	// The temporary table must be created inside the dynamic batch, so that it is always
	// dropped when the batch finishes and never leaks onto a pooled connection.
	require.Contains(t, query, "DROP TABLE #vfs;';")

	// The output columns must be unchanged.
	for _, column := range []string{
		"[measurement]",
		"[sql_instance]",
		"[database_name]",
		"[physical_filename]",
		"[logical_filename]",
		"[file_type]",
		"[read_latency_ms]",
		"[reads]",
		"[read_bytes]",
		"[write_latency_ms]",
		"[writes]",
		"[write_bytes]",
		"[rg_read_stall_ms]",
		"[rg_write_stall_ms]",
	} {
		require.Contains(t, query, column)
	}

	// The statistics must be materialized before they are joined, otherwise the
	// function would still be re-evaluated for every row of the join.
	var into, join = -1, -1
	for i, line := range strings.Split(query, "\n") {
		line = strings.TrimSpace(line)
		if line == "INTO #vfs" {
			into = i
		}
		if strings.HasPrefix(line, "INNER JOIN sys.master_files") {
			join = i
		}
	}
	require.GreaterOrEqual(t, into, 0, "the query must materialize the file stats with SELECT ... INTO")
	require.GreaterOrEqual(t, join, 0, "the query must join sys.master_files")
	require.Greater(t, join, into, "sys.master_files must be joined after the file stats are materialized")

	// The function must only be read by the materializing statement, never by the
	// statement that is joined with sys.master_files.
	require.Equal(t, 1, strings.Count(query, "sys.dm_io_virtual_file_stats"),
		"sys.dm_io_virtual_file_stats must be referenced exactly once")
}

// The batch sent to the server starts with a SELECT ... INTO statement that does not
// return a rowset, followed by the SELECT whose rows Telegraf consumes. This checks that
// the plugin handles such a batch and still emits the expected metric, using a mocked
// database connection that replies to the query the plugin actually sends.
func TestSqlServer_DatabaseIOQueryBatch(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	s := &SQLServer{
		DatabaseType: "SQLServer",
		IncludeQuery: []string{"SQLServerDatabaseIO"},
		AuthMethod:   "connection_string",
		Log:          testutil.Logger{},
	}
	require.NoError(t, s.initQueries())

	rows := sqlmock.
		NewRows([]string{
			"measurement", "sql_instance", "database_name", "physical_filename",
			"logical_filename", "file_type", "read_latency_ms", "reads", "read_bytes",
			"write_latency_ms", "writes", "write_bytes", "volume_mount_point",
			"rg_read_stall_ms", "rg_write_stall_ms",
		}).
		AddRow("sqlserver_database_io", "server:instance", "master", `C:\master.mdf`,
			"master", "ROWS", int64(11), int64(22), int64(33), int64(44), int64(55),
			int64(66), `C:\`, int64(77), int64(88))
	mock.ExpectQuery(s.queries["SQLServerDatabaseIO"].Script).WillReturnRows(rows)

	var acc testutil.Accumulator
	require.NoError(t, s.gatherServer(db, s.queries["SQLServerDatabaseIO"], &acc, "Server=fake;Database=fakedb;"))
	require.NoError(t, mock.ExpectationsWereMet())

	require.True(t, acc.HasMeasurement("sqlserver_database_io"))
	require.Equal(t, uint64(1), acc.NMetrics())
	metric, ok := acc.Get("sqlserver_database_io")
	require.True(t, ok)
	require.Equal(t, map[string]string{
		"sql_instance":        "server:instance",
		"database_name":       "master",
		"physical_filename":   `C:\master.mdf`,
		"logical_filename":    "master",
		"file_type":           "ROWS",
		"volume_mount_point":  `C:\`,
		"measurement_db_type": "SQLServer",
	}, metric.Tags)
	require.Equal(t, map[string]any{
		"read_latency_ms":   int64(11),
		"reads":             int64(22),
		"read_bytes":        int64(33),
		"write_latency_ms":  int64(44),
		"writes":            int64(55),
		"write_bytes":       int64(66),
		"rg_read_stall_ms":  int64(77),
		"rg_write_stall_ms": int64(88),
	}, metric.Fields)
}

// Checks that the optimized query works against a real SQL Server and that the returned
// metrics are unchanged compared to the previous implementation.
func TestSqlServerIntegration_DatabaseIOQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("MSSQL_CONNECTION_STRING") == "" {
		t.Skip("Missing environment variable MSSQL_CONNECTION_STRING")
	}

	sl := config.NewSecret([]byte(os.Getenv("MSSQL_CONNECTION_STRING")))
	server := &SQLServer{
		Servers:      []*config.Secret{&sl},
		DatabaseType: "SQLServer",
		IncludeQuery: []string{"SQLServerDatabaseIO"},
		AuthMethod:   "connection_string",
		Log:          testutil.Logger{},
	}

	var acc testutil.Accumulator
	require.NoError(t, server.Start(&acc))
	defer server.Stop()
	require.NoError(t, server.Gather(&acc))

	require.True(t, acc.HasMeasurement("sqlserver_database_io"))
	require.True(t, acc.HasTag("sqlserver_database_io", "sql_instance"))
	require.True(t, acc.HasTag("sqlserver_database_io", "database_name"))
	require.True(t, acc.HasTag("sqlserver_database_io", "physical_filename"))
	require.True(t, acc.HasTag("sqlserver_database_io", "logical_filename"))
	require.True(t, acc.HasTag("sqlserver_database_io", "file_type"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "reads"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "read_bytes"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "read_latency_ms"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "write_latency_ms"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "writes"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "write_bytes"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "rg_read_stall_ms"))
	require.True(t, acc.HasInt64Field("sqlserver_database_io", "rg_write_stall_ms"))
	require.True(t, acc.HasTag("sqlserver_database_io", "volume_mount_point"))
}
