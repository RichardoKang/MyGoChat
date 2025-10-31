package response

import (
	"time"
)

type GroupResponse struct {
	Uuid        string    `json:"uuid"`
	GroupNumber string    `json:"groupNumber"`
	CreatedAt   time.Time `json:"createAt"`
	Name        string    `json:"name"`
}

type GroupMemberResponse struct {
	UserID   uint   `json:"userid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Muted    bool   `json:"muted"`
}
