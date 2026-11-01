package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open MySQL: %w", err)
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(2)
	pool.SetConnMaxLifetime(time.Hour)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = pool.PingContext(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect MySQL: %w", err)
	}
	return pool, nil
}
