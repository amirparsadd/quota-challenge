package obslog

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

const maxSQLLog = 400

// DBTracer logs SQL at debug; failed queries at warn.
type DBTracer struct{}

func (DBTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	slog.DebugContext(ctx, "db query start",
		"sql", truncateSQL(data.SQL),
		"args_count", len(data.Args),
	)
	return ctx
}

func (DBTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	tag := data.CommandTag.String()
	if data.Err != nil {
		slog.WarnContext(ctx, "db query end", "err", data.Err.Error(), "command_tag", tag)
		return
	}
	slog.DebugContext(ctx, "db query end", "command_tag", tag)
}

func truncateSQL(sql string) string {
	sql = strings.Join(strings.Fields(sql), " ")
	if len(sql) <= maxSQLLog {
		return sql
	}
	return sql[:maxSQLLog] + "…"
}

// AttachPoolTracer enables query tracing on a connection config.
func AttachPoolTracer(cfg *pgx.ConnConfig) {
	if cfg == nil {
		return
	}
	cfg.Tracer = DBTracer{}
}