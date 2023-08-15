package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// カウンターを持つ HTTP リクエストハンドラー
type countHandler struct {
	count int
}

func (h *countHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.count++
	fmt.Fprintf(w, "Count: %d\n", h.count)
}

type showUserHandler struct {
	count int
}

type User struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (h *showUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
	if err != nil {
		fmt.Printf("error")
	}
	rows, err := db.Query("select * from users")

	defer db.Close()

	var users []User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			log.Fatalf("getRows rows.Scan error err:%v", err)
		}
		users = append(users, *u)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Println(err)
	}
}

func main() {
	http.Handle("/count", new(countHandler))
	http.Handle("/users", new(showUserHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
