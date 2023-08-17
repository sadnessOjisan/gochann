package router

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"learn-go-server/model"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// see: https://stackoverflow.com/questions/15130321/is-there-a-method-to-generate-a-uuid-with-go-language
func pseudo_uuid() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

func UsersDetailHandler(w http.ResponseWriter, r *http.Request) {
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

	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
		log.Fatalf("getRows rows.Scan error err:%v", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(u); err != nil {
		log.Println(err)
	}
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		password := []byte(r.FormValue("password"))
		hasher := sha256.New()
		hasher.Write([]byte(password))
		hashedPasswordString := hex.EncodeToString(hasher.Sum(nil))

		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		defer db.Close()
		if err != nil {
			fmt.Printf("error")
		}
		ins, err := db.Prepare("insert into users(name, password) value (?, ?)")
		if err != nil {
			fmt.Printf("error")
			return
		}
		res, err := ins.Exec(name, hashedPasswordString)
		user_id, err := res.LastInsertId()
		uuid := pseudo_uuid()

		session_insert, err := db.Prepare("insert into session(user_id, token) value (?, ?)")
		session_insert_res, err := session_insert.Exec(user_id, uuid)
		if err != nil {
			log.Println(err)
		}
		print(session_insert_res.LastInsertId())

		cookie := &http.Cookie{
			Name:     "token",
			Value:    uuid,
			Expires:  time.Now().AddDate(0, 0, 1),
			SameSite: http.SameSiteStrictMode,
			HttpOnly: true,
			Secure:   true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		if err != nil {
			fmt.Printf("error")
		}
		rows, err := db.Query("select * from users")

		defer db.Close()

		var users []model.User
		for rows.Next() {
			u := &model.User{}
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
