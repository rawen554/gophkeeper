package models

type User struct {
	Login     string  `gorm:"varchar(100);index:idx_login,unique" json:"login"`
	Password  string  `gorm:"varchar(255);not null"`
	ID        uint64  `gorm:"primaryKey" json:"id,omitempty"`
	Balance   float64 `gorm:"default:0" json:"-"`
	Withdrawn float64 `gorm:"default:0" json:"-"`
}

type UserCredentialsSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserBalanceShema struct {
	Balance   float64 `gorm:"default:0" json:"current"`
	Withdrawn float64 `gorm:"default:0" json:"withdrawn"`
}
