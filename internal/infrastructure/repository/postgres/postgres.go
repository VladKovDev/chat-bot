package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
	logger logger.Logger
}

func NewPool(ctx context.Context, cfg *Config, logger logger.Logger) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(getPostgresDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("unable to parse db config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		return true
	}

	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second

	logger.Info("connecting to database",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.Name),
		logger.Int32("max_conns", poolConfig.MaxConns),
		logger.Int32("min_conns", poolConfig.MinConns),
	)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Info("successfully connected to database")

	return &Pool{
		Pool:   pool,
		logger: logger,
	}, nil
}

func (p *Pool) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		p.logger.Error("PostgreSQL health check failed", p.logger.Err(err))
		return err
	}

	return nil
}

func (p *Pool) Close() {
	p.Pool.Close()
	p.logger.Info("PostgreSQL connection pool closed")
}

func (p *Pool) Shutdown(ctx context.Context) error {
	p.Pool.Close()
	p.logger.Info("PostgreSQL connection pool closed")
	return nil
}

func getPostgresDSN(cfg *Config) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		url.QueryEscape(cfg.Password),
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}
