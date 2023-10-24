package config

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	RunAddr         string `json:"server_address" env:"SERVER_ADDRESS"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `json:"database_dsn" env:"DATABASE_DSN"`
	Config          string `json:"-" env:"CONFIG"`
	TLSCertPath     string `json:"tls_cert_path" env:"TLS_CERT_PATH"`
	TLSKeyPath      string `json:"tls_key_path" env:"TLS_KEY_PATH"`
	LogLevel        string `env:"LOG_LEVEL" envDefault:"debug"`
	EnableHTTPS     bool   `json:"enable_https" env:"ENABLE_HTTPS"`
}

var config ServerConfig

func ParseFlags() (*ServerConfig, error) {
	flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	flag.BoolVar(&config.EnableHTTPS, "s", true, "enable https")
	flag.StringVar(&config.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", "", "Data Source Name (DSN)")
	flag.StringVar(&config.Config, "c", "", "Config json file path")
	flag.StringVar(&config.TLSCertPath, "l", "./certs/cert.pem", "path to tls cert file")
	flag.StringVar(&config.TLSKeyPath, "k", "./certs/private.pem", "path to tls key file")
	flag.Parse()

	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("error parsing env variables: %w", err)
	}

	if config.Config != "" {
		data, err := os.ReadFile(config.Config)
		if err != nil {
			return nil, fmt.Errorf("error opening config file: %w", err)
		}

		var configFromFile ServerConfig
		if err := json.NewDecoder(bytes.NewReader(data)).Decode(&configFromFile); err != nil {
			return nil, fmt.Errorf("error parsing json file config: %w", err)
		}

		if err := mergo.Merge(&config, configFromFile); err != nil {
			return nil, fmt.Errorf("cannot merge configs: %w", err)
		}
	}

	return &config, nil
}
