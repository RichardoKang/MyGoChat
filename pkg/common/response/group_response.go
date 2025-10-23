package response

import "time"

type GroupResponse struct {
	Uuid      string    `json:"uuid"`
	CreatedAt time.Time `json:"createAt"`
	Name      string    `json:"name"`
}
