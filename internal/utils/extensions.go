package utils

import "github.com/rawen554/goph-keeper/internal/models"

func GetExtension(dataType models.DataType) string {
	switch dataType {
	case models.PASS:
		return ".json"
	case models.TEXT:
		return ".json"
	default:
		return ".json"
	}
}
