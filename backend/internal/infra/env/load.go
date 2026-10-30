package env

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
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
	EnvLogLevel           = "LOG_LEVEL"
	EnvLogFormat          = "LOG_FORMAT"
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
	DefaultLogLevel           = slog.LevelInfo
	DefaultLogFormat          = logging.FormatJSON
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
	LogLevel           *slog.LevelVar
	LogFormat          logging.Format
}

func Load() (*Vars, error) {
	var errs []error

	dbConnectionString := os.Getenv(EnvDBConnectionString)
	if dbConnectionString == "" {
		errs = append(errs, fmt.Errorf("get env %s: not set", EnvDBConnectionString))
	}

	dbURL, err := url.Parse(dbConnectionString)
	if err == nil && dbURL.Scheme != "postgres" && dbURL.Scheme != "postgresql" {
		err = fmt.Errorf("want scheme 'postgres://', got scheme %q", dbURL.Scheme)
	}
	if err != nil {
		errs = append(errs, fmt.Errorf("parse env %s: %w", EnvDBConnectionString, err))
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

	logLevel, err := getEnvLevel(EnvLogLevel, DefaultLogLevel)
	errs = append(errs, err)

	logFormat, err := getEnvFormat(EnvLogFormat, DefaultLogFormat)
	errs = append(errs, err)

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

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
		LogLevel:           logLevel,
		LogFormat:          logFormat,
	}, nil
}

// LogAttrs returns the loaded values as log attrs, with the database password
// redacted.
func (v *Vars) LogAttrs() []slog.Attr {
	dbConnectionString := v.DBConnectionString
	if dbURL, err := url.Parse(dbConnectionString); err == nil {
		dbConnectionString = dbURL.Redacted()
	}
	return []slog.Attr{
		slog.String(EnvDBConnectionString, dbConnectionString),
		slog.Int(EnvDBMaxOpenConns, v.DBMaxOpenConns),
		slog.Int(EnvDBMaxIdleConns, v.DBMaxIdleConns),
		slog.Duration(EnvDBConnMaxLifetime, v.DBConnMaxLifetime),
		slog.Duration(EnvDBConnMaxIdleTime, v.DBConnMaxIdleTime),
		slog.String(EnvAppPort, v.AppPort),
		slog.Duration(EnvAppReadTimeout, v.AppReadTimeout),
		slog.Duration(EnvAppWriteTimeout, v.AppWriteTimeout),
		slog.Duration(EnvAppIdleTimeout, v.AppIdleTimeout),
		slog.Duration(EnvAppShutdownTimeout, v.AppShutdownTimeout),
		slog.String(EnvLogLevel, v.LogLevel.Level().String()),
		slog.String(EnvLogFormat, v.LogFormat.String()),
	}
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

func getEnvLevel(key string, defaultValue slog.Level) (*slog.LevelVar, error) {
	var value slog.LevelVar
	valueStr := os.Getenv(key)
	if valueStr == "" {
		value.Set(defaultValue)
		return &value, nil
	}
	if err := value.UnmarshalText([]byte(valueStr)); err != nil {
		return nil, fmt.Errorf("parse %s: %w", key, err)
	}
	return &value, nil
}

func getEnvFormat(key string, defaultValue logging.Format) (logging.Format, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	var value logging.Format
	if err := value.UnmarshalText([]byte(valueStr)); err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}
