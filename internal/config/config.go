package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var (
	ErrReadConfig = errors.New("error reading config")
	ErrNotFound   = errors.New("config file not found at path")
)

type Config struct {
	Server   ServerCfg   `yaml:"server"`
	Database DatabaseCfg `yaml:"database"`
	Logger   LoggerCfg   `yaml:"logger"`
}

type ServerCfg struct {
	Port    int           `yaml:"port" env-required:"true"`
	Host    string        `yaml:"host" env-default:"localhost"`
	Timeout time.Duration `yaml:"timeout" env-default:"5s"`
}

type DatabaseCfg struct {
	Who  string `yaml:"who" env-default:"sqlite"`
	Host string `yaml:"host" env-required:"true"`
}

// TODO type AppCfg struct{}

type LoggerCfg struct {
	Env string `yaml:"env" env-required:"true"`
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

	return "", ErrNotFound
}

func MustLoad() *Config {
	var cfg Config

	path, err := getPathOfConfig()
	if err != nil {
		panic(err)
	}
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic(fmt.Errorf("%w: %v", ErrReadConfig, err))
	}
	return &cfg
}
