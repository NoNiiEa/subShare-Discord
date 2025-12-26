package models

import "time"

type Slip struct {
	ID int `json:"id"`
	TransRef string `json:"trans_ref"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}