package models

import "errors"

var (
	ErrNoData = errors.New("no data")
)

type User struct {
	Login    string `gorm:"varchar(100);index:idx_login,unique" json:"login"`
	Password string `gorm:"varchar(255);not null" json:"-"`
	ID       uint64 `gorm:"primaryKey" json:"id,omitempty"`
}

type UserCredentialsSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
