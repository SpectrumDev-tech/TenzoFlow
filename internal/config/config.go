package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"tenzoflow/internal/logger"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var validSSLModes = []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

var (
	ErrReadConfig    = errors.New("error reading config")
	ErrPathNotSet    = errors.New("config path is not set (use --config flag or CONFIG_PATH)")
	ErrInvalidConfig = errors.New("invalid config")
)

type Config struct {
	Server   ServerCfg   `yaml:"server"`
	Database DatabaseCfg `yaml:"database"`
	Logger   LoggerCfg   `yaml:"logger"`
}

type ServerCfg struct {
	Port            int           `yaml:"port" env-required:"true"`
	Host            string        `yaml:"host" env-default:"localhost"`
	Timeout         time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"30s"`
}

func (s *ServerCfg) Addr() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

type DatabaseCfg struct {
	Driver   string      `yaml:"driver" env-default:"sqlite"`
	SQLite   SQLite      `yaml:"sqlite"`
	Postgres PostgresCfg `yaml:"postgres"`
}
type SQLite struct {
	Path string `yaml:"path"`
}
type PostgresCfg struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`

	MaxOpenConns    int           `yaml:"max_open_conns" env-default:"10"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env-default:"10m"`
}

func (p PostgresCfg) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:     "/" + p.Name,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}
	return u.String()
}

func (p PostgresCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("host", p.Host),
		slog.Int("port", p.Port),
		slog.String("user", p.User),
		slog.String("name", p.Name),
		slog.String("sslmode", p.SSLMode),
		slog.Int("max_open_conns", p.MaxOpenConns),
	)
}

func (d DatabaseCfg) LogValue() slog.Value {
	attrs := []slog.Attr{slog.String("driver", d.Driver)}
	switch d.Driver {
	case DriverSQLite:
		attrs = append(attrs, slog.String("path", d.SQLite.Path))
	case DriverPostgres:
		attrs = append(attrs, slog.Any("postgres", d.Postgres))
	}
	return slog.GroupValue(attrs...)
}

type LoggerCfg struct {
	Env   string `yaml:"env" env-required:"true"`
	Level string `yaml:"level"`
}

func (c *Config) Validate() error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		add("server.port must be 1...65535, got %d", c.Server.Port)
	}

	if c.Logger.Level != "" {
		if _, err := logger.ParseLevel(c.Logger.Level); err != nil {
			add("logger.level %v", err)
		}
	}

	switch c.Logger.Env {
	case logger.EnvLocal, logger.EnvProd:
	default:
		add("logger.env must be %q or %q, got %q", logger.EnvLocal, logger.EnvProd, c.Logger.Env)
	}
	switch c.Database.Driver {
	case DriverSQLite:
		p := c.Database.SQLite.Path
		switch {
		case p == "":
			add("database.sqlite.path is required")
		case strings.HasPrefix(p, "~"):
			// Go never expands "~": it would silently create a directory literally named "~".
			add("database.sqlite.path %q: '~' is not expanded, use ./data/app.db or an absolute path", p)
		}
	case DriverPostgres:
		pg := c.Database.Postgres
		if pg.Host == "" {
			add("database.postgres.host is required")
		}
		if pg.Port < 1 || pg.Port > 65535 {
			add("database.postgres.port must be in 1..65535, got %d", pg.Port)
		}
		if pg.User == "" {
			add("database.postgres.user is required")
		}
		if pg.Name == "" {
			add("database.postgres.name is required")
		}
		if pg.Password == "" {
			add("database.postgres.password is required (set DB_PG_PASSWORD)")
		}
		if !slices.Contains(validSSLModes, pg.SSLMode) {
			add("database.postgres.sslmode must be one of %v, got %q", validSSLModes, pg.SSLMode)
		}

	default:
		add("database.driver must be %q or %q, got %q", DriverSQLite, DriverPostgres, c.Database.Driver)
	}

	return errors.Join(errs...)
}

func getPathOfConfig() (string, error) {
	var path string

	flag.StringVar(&path, "config", "", "config file path")
	flag.Parse()

	if path != "" {
		return path, nil
	}
	if path = os.Getenv("CONFIG_PATH"); path != "" {
		return path, nil
	}

	return "", ErrPathNotSet
}

func MustLoad() *Config {
	path, err := getPathOfConfig()
	if err != nil {
		panic(err)
	}
	return MustLoadFrom(path)
}

func MustLoadFrom(path string) *Config {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic(fmt.Errorf("%w: %w", ErrReadConfig, err))
	}
	if err := cfg.Validate(); err != nil {
		panic(fmt.Errorf("%w: %w", ErrInvalidConfig, err))
	}

	return &cfg
}
