package database

import (
	"context"
	"fmt"
	"log/slog"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jmoiron/sqlx"
	pgxslog "github.com/mcosta74/pgx-slog"
)

func NewPostgresConnection(ctx context.Context, config Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.DBName, config.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	poolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   pgxslog.NewLogger(slog.Default()),
		LogLevel: tracelog.LogLevelInfo,
		Config:   tracelog.DefaultTraceLogConfig(),
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	stdDB := stdlib.OpenDBFromPool(pool)

	db := sqlx.NewDb(stdDB, "pgx")
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
