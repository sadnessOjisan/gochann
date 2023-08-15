package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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

type getUsersHandler struct {
	count int
}

type User struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (h *getUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		password := []byte(r.FormValue("password"))
		hashedPassword := sha256.Sum256(password)

		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		if err != nil {
			fmt.Printf("error")
		}
		ins, err := db.Prepare("insert into users(name, password) value (?, ?)")
		if err != nil {
			fmt.Printf("error")
			return
		}
		defer db.Close()
		ins.Exec(name, hashedPassword)
		return
	}
	if r.Method == http.MethodGet {
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

}

type getUserHandler struct {
	count int
}

func (h *getUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Printf("method not allowed")
		return
	}
	sub := strings.TrimPrefix(r.URL.Path, "/users")
	_, id := filepath.Split(sub)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Printf("id is not found")
		return
	}
	db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
	if err != nil {
		fmt.Printf("error")
	}
	row := db.QueryRow("select * from users where id = ? limit 1", id)

	defer db.Close()

	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
		log.Fatalf("getRows rows.Scan error err:%v", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(u); err != nil {
		log.Println(err)
	}
}

func main() {
	http.Handle("/count", new(countHandler))
	http.Handle("/users", new(getUsersHandler))
	// for /users/:id
	http.Handle("/users/", new(getUserHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
