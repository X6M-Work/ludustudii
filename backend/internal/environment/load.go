package environment

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

const (
	EnvDBConnectionString = "DB_CONNECTION"
	EnvDBMaxOpenConns     = "DB_MAX_OPEN_CONNS"
	EnvDBMaxIdleConns     = "DB_MAX_IDLE_CONNS"
	EnvDBConnMaxLifetime  = "DB_CONN_MAX_LIFETIME"
	EnvDBConnMaxIdleTime  = "DB_CONN_MAX_IDLE_TIME"
	EnvAppPort            = "APP_PORT"
	EnvAppReadTimeout     = "APP_READ_TIMEOUT"
	EnvAppWriteTimeout    = "APP_WRITE_TIMEOUT"
	EnvAppIdleTimeout     = "APP_IDLE_TIMEOUT"
	EnvAppShutdownTimeout = "APP_SHUTDOWN_TIMEOUT"
)

const (
	DefaultDBMaxOpenConns     = 10
	DefaultDBMaxIdleConns     = 5
	DefaultDBConnMaxLifetime  = 30 * time.Minute
	DefaultDBConnMaxIdleTime  = 10 * time.Minute
	DefaultAppReadTimeout     = 5 * time.Second
	DefaultAppWriteTimeout    = 10 * time.Second
	DefaultAppIdleTimeout     = 90 * time.Second
	DefaultAppShutdownTimeout = 15 * time.Second
)

type Vars struct {
	DBConnectionString string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
	DBConnMaxIdleTime  time.Duration
	AppPort            string
	AppReadTimeout     time.Duration
	AppWriteTimeout    time.Duration
	AppIdleTimeout     time.Duration
	AppShutdownTimeout time.Duration
}

func Load() (*Vars, error) {
	var errs []error

	dbConnectionString := os.Getenv(EnvDBConnectionString)
	if dbConnectionString == "" {
		errs = append(errs, fmt.Errorf("get env %s: not set", EnvDBConnectionString))
	}

	dbMaxOpenConns, err := getEnvInt(EnvDBMaxOpenConns, DefaultDBMaxOpenConns)
	errs = append(errs, err)

	dbMaxIdleConns, err := getEnvInt(EnvDBMaxIdleConns, DefaultDBMaxIdleConns)
	errs = append(errs, err)

	dbConnMaxLifetime, err := getEnvDuration(EnvDBConnMaxLifetime, DefaultDBConnMaxLifetime)
	errs = append(errs, err)

	dbConnMaxIdleTime, err := getEnvDuration(EnvDBConnMaxIdleTime, DefaultDBConnMaxIdleTime)
	errs = append(errs, err)

	appPort := os.Getenv(EnvAppPort)
	if appPort == "" {
		errs = append(errs, fmt.Errorf("get env %s: not set", EnvAppPort))
	}

	appReadTimeout, err := getEnvDuration(EnvAppReadTimeout, DefaultAppReadTimeout)
	errs = append(errs, err)

	appWriteTimeout, err := getEnvDuration(EnvAppWriteTimeout, DefaultAppWriteTimeout)
	errs = append(errs, err)

	appIdleTimeout, err := getEnvDuration(EnvAppIdleTimeout, DefaultAppIdleTimeout)
	errs = append(errs, err)

	appShutdownTimeout, err := getEnvDuration(EnvAppShutdownTimeout, DefaultAppShutdownTimeout)
	errs = append(errs, err)

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	slog.Info("Environment variables loaded",
		slog.String("DB_CONNECTION", dbConnectionString),
		slog.Int("DB_MAX_OPEN_CONNS", dbMaxOpenConns),
		slog.Int("DB_MAX_IDLE_CONNS", dbMaxIdleConns),
		slog.Duration("DB_CONN_MAX_LIFETIME", dbConnMaxLifetime),
		slog.Duration("DB_CONN_MAX_IDLE_TIME", dbConnMaxIdleTime),
		slog.String("APP_PORT", appPort),
		slog.Duration("APP_READ_TIMEOUT", appReadTimeout),
		slog.Duration("APP_WRITE_TIMEOUT", appWriteTimeout),
		slog.Duration("APP_IDLE_TIMEOUT", appIdleTimeout),
		slog.Duration("APP_SHUTDOWN_TIMEOUT", appShutdownTimeout),
	)

	return &Vars{
		DBConnectionString: dbConnectionString,
		DBMaxOpenConns:     dbMaxOpenConns,
		DBMaxIdleConns:     dbMaxIdleConns,
		DBConnMaxLifetime:  dbConnMaxLifetime,
		DBConnMaxIdleTime:  dbConnMaxIdleTime,
		AppPort:            appPort,
		AppReadTimeout:     appReadTimeout,
		AppWriteTimeout:    appWriteTimeout,
		AppIdleTimeout:     appIdleTimeout,
		AppShutdownTimeout: appShutdownTimeout,
	}, nil
}

func getEnvInt(key string, defaultValue int) (int, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func getEnvDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}
