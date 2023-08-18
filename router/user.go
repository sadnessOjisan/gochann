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
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
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
		log.Printf("ERROR: user id is not found err")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dsn := os.Getenv("dbdsn")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("ERROR: db open err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	row := db.QueryRow("select * from users where id = ? limit 1", id)
	defer db.Close()

	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
		log.Printf("ERROR: db scan user err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(u); err != nil {
		log.Println(err)
	}
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	// POST users
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		if !(utf8.RuneCountInString(name) >= 1 && utf8.RuneCountInString(name) <= 32) {
			log.Printf("ERROR: name length is not invalid name: %s, utf8.RuneCountInString(name): %d", name, utf8.RuneCountInString(name))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()
		if err != nil {
			log.Printf("ERROR: db open err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		exsist_user_row := db.QueryRow("select id, password, salt from users where name = ? limit 1", name)
		var user_id int
		var current_user_salt string
		var current_user_hashed_password string
		exsist_user_row.Scan(&user_id, &current_user_hashed_password, &current_user_salt)

		password := r.FormValue("password")
		if !(utf8.RuneCountInString(password) >= 1 && utf8.RuneCountInString(password) <= 100) {
			log.Printf("ERROR: title length is not invalid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if user_id == 0 {
			// アカウント情報が存在しないなら登録してクッキーを発行する
			salt := pseudo_uuid()
			password_added_salt := password + salt
			password_byte := []byte(password_added_salt)
			hasher := sha256.New()
			hasher.Write([]byte(password_byte))
			hashedPasswordString := hex.EncodeToString(hasher.Sum(nil))

			ins, err := db.Prepare("insert into users(name, password, salt) value (?, ?, ?)")
			if err != nil {
				log.Printf("ERROR: prepare users insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := ins.Exec(name, hashedPasswordString, salt)
			if err != nil {
				log.Printf("ERROR: exec user insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			added_user_id, err := res.LastInsertId()
			if err != nil {
				log.Printf("ERROR: get last user id err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			uuid := pseudo_uuid()

			session_insert, err := db.Prepare("insert into session(user_id, token) value (?, ?)")
			if err != nil {
				log.Printf("ERROR: prepare session insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_, err = session_insert.Exec(added_user_id, uuid)
			if err != nil {
				log.Printf("ERROR: exec session insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

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
		} else {
			// アカウント情報が存在するユーザーなら、入力されたパスワードと正しいか確認してから、クッキー発行してログインさせる
			password_added_salt := password + current_user_salt
			password_byte := []byte(password_added_salt)
			hasher := sha256.New()
			hasher.Write([]byte(password_byte))
			hashedPasswordString := hex.EncodeToString(hasher.Sum(nil))

			if current_user_hashed_password != hashedPasswordString {
				log.Printf("ERROR: user input password mismatch")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			uuid := pseudo_uuid()
			session_insert, err := db.Prepare("insert into session(user_id, token) value (?, ?)")
			if err != nil {
				log.Printf("ERROR: prepare session insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_, err = session_insert.Exec(user_id, uuid)
			if err != nil {
				log.Printf("ERROR: exec session insert err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
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

	}

	// GET /posts
	if r.Method == http.MethodGet {

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()
		if err != nil {
			log.Printf("ERROR: db open err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rows, err := db.Query("select * from users")

		var users []model.User
		for rows.Next() {
			u := &model.User{}
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
				log.Printf("ERROR: db scan users err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
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
