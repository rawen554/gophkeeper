package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

type DataType string

const (
	PASS DataType = "PASS"
	TEXT DataType = "TEXT"
	BIN  DataType = "BIN"
	CARD DataType = "CARD"
)

func (s *DataType) Scan(value interface{}) error {
	sv, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal DataType value: ", value))
	}

	*s = DataType(sv)
	return nil
}

func (s DataType) Value() (driver.Value, error) {
	return string(s), nil
}

type DataRecord struct {
	UploadedAt time.Time `gorm:"default:now()" json:"uploaded_at"`
	Type       DataType  `sql:"type:data_type" json:"type"`
	Checksum   string    `gorm:"checksum" json:"checksum"`
	Data       string    `gorm:"data" json:"data"`
	User       User      `gorm:"not null;" json:"-"`
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `json:"-"`
	Blocked    bool      `gorm:"blocked" json:"blocked"`
}
