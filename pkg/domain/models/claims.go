package models

import "time"

type Claims struct {
	UserId         int
	Email          string
	SessionId      int
	ExpirationTime time.Time
}
