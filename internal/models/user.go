package models

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

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

type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

func (u *User) GetUserFolder() ([]fs.DirEntry, error) {
	return os.ReadDir(fmt.Sprintf("./userdata/%s-%d", u.Login, u.ID))
}
