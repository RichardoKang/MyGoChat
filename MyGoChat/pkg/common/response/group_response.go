package response

import "time"

type GroupResponse struct {
	Uuid        string    `json:"uuid"`
	GroupNumber string    `json:"groupNumber"`
	CreatedAt   time.Time `json:"createAt"`
	Name        string    `json:"name"`
}
