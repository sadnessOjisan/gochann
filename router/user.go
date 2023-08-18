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
		fmt.Printf("error")
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
		password := r.FormValue("password")
		salt := pseudo_uuid()
		password_added_salt := password + salt
		password_byte := []byte(password_added_salt)
		hasher := sha256.New()
		hasher.Write([]byte(password_byte))
		hashedPasswordString := hex.EncodeToString(hasher.Sum(nil))

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()
		if err != nil {
			log.Printf("ERROR: db open err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		exsist_user_row := db.QueryRow("select count(*) from users where name = ? and password = ? limit 1", name, hashedPasswordString)
		var count int
		exsist_user_row.Scan(&count)

		// アカウント情報が存在するユーザーならクッキー発行してログインさせる
		if count == 1 {
			uuid := pseudo_uuid()
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
		print(salt)
		// 入力情報に一致するユーザ情報がない場合はアカウントを新規作成してログイン
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
		user_id, err := res.LastInsertId()
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

	// GET /posts
	if r.Method == http.MethodGet {

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()
		if err != nil {
			fmt.Printf("error")
		}
		rows, err := db.Query("select * from users")

		var users []model.User
		for rows.Next() {
			u := &model.User{}
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
				log.Printf("ERROR: db scan user err: %v", err)
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
