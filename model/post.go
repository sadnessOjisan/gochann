package model

import "time"

type Post struct {
	ID        int       `db:"id"`
	Text      string    `db:"text"`
	Title     string    `db:"title"`
	User      User      `db:"user"`
	Comments  []Comment `db:"comments"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
