package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrOrderHasBeenProcessedByUser = errors.New("this order already been processed by user")
var ErrOrderHasBeenProcessedByAnotherUser = errors.New("this order already been processed by another user")
var ErrUserHasNoItems = errors.New("no items found")

type Status string

const (
	NEW        Status = "NEW"
	REGISTERED Status = "REGISTERED"
	PROCESSING Status = "PROCESSING"
	INVALID    Status = "INVALID"
	PROCESSED  Status = "PROCESSED"
)

func (s *Status) Scan(value interface{}) error {
	sv, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal Status value: ", value))
	}

	*s = Status(sv)
	return nil
}

func (s Status) Value() (driver.Value, error) {
	return string(s), nil
}

type OrderTime time.Time

func (ot OrderTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", time.Time(ot).Format(time.RFC3339))
	return []byte(formatted), nil
}

type Order struct {
	UploadedAt OrderTime `gorm:"default:now()" json:"uploaded_at"`
	Number     string    `gorm:"primaryKey" json:"number"`
	Status     Status    `sql:"type:order_status" json:"status"`
	User       User      `json:"-"`
	UserID     uint64    `json:"-"`
	Accrual    float64   `json:"accrual,omitempty"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	o.Status = NEW
	return nil
}

func (o *Order) AfterUpdate(tx *gorm.DB) (err error) {
	if o.Status == PROCESSED && o.Accrual > 0 {
		result := tx.Model(&User{}).Where("id = ?", o.UserID).Update("balance", gorm.Expr("balance + ?", o.Accrual))
		return result.Error
	}
	return
}
