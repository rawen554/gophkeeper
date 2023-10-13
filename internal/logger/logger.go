package logger

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return logger.Sugar(), nil
}
