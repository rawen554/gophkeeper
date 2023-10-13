package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	RunAddr     string `env:"RUN_ADDRESS" envDefault:":8080"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI string `env:"DATABASE_URI"`
	Key         string `env:"KEY" envDefault:"b4952c3809196592c026529df00774e46bfb5be0"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"debug"`
}

var config ServerConfig

func ParseFlags() (*ServerConfig, error) {
	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("error parsing env variables: %w", err)
	}

	flag.StringVar(&config.RunAddr, "a", config.RunAddr, "address and port to run server")
	flag.StringVar(&config.AccrualAddr, "r", config.AccrualAddr, "Accrual System Address")
	flag.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "Data Source Name (DSN)")
	flag.StringVar(&config.Key, "k", config.Key, "key is used to sign JWT tokens")
	flag.StringVar(&config.LogLevel, "l", config.LogLevel, "debug | info | warn | error")
	flag.Parse()

	return &config, nil
}

func GetDummy() *ServerConfig {
	return &ServerConfig{
		RunAddr:  ":8080",
		Key:      "b4952c3809196592c026529df00774e46bfb5be0",
		LogLevel: "debug",
	}
}
