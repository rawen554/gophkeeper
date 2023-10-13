package models

type Withdraw struct {
	ProcessedAt OrderTime `gorm:"default:now()" json:"processed_at"`
	OrderNum    string    `json:"order"`
	User        User      `json:"-"`
	UserID      uint64    `gorm:"column:user_id" json:"-"`
	Sum         float64   `json:"sum"`
}

func (w *Withdraw) TableName() string {
	return "withdrawals"
}

type BalanceWithdrawShema struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
