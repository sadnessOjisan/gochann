package model

import "time"

type Comment struct {
	ID        int       `db:"id"`
	Text      string    `db:"text"`
	User      User      `db:"user"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
