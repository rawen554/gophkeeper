package utils

import (
	"fmt"
	"os"
	"path"
)

func CreateUsersDir(username string) error {
	if err := os.MkdirAll(path.Join(".", username), 0750); err != nil {
		return fmt.Errorf("error creating user's dir: %v", err)
	}

	return nil
}
