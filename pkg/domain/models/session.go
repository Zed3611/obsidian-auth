package models

import "time"

type Session struct {
	Id               int
	UserId           int
	RefreshTokenHash string
	Ip               string
	UserAgent        string
	RefreshCount     int
	ActiveTil        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
