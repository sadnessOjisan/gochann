package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
		println("POST")
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

		w.Header().Add("set-cookie", "token=uuid; Max-Age=86400; SameSite=Strict; Secure; HttpOnly")
		cookie := &http.Cookie{
			Name:     "token",
			Value:    uuid,
			MaxAge:   86400,
			SameSite: http.SameSiteStrictMode,
			HttpOnly: true,
			Secure:   true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/posts", http.StatusTemporaryRedirect)
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

type newUserHandler struct {
	count int
}

func (h *newUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Printf("method not allowed")
		return
	}
	t := template.Must(template.ParseFiles("./template/users-new.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, nil); err != nil {
		panic(err.Error())
	}
}

type postsHandler struct {
	count int
}

type Post struct {
	ID        int       `db:"id"`
	Text      string    `db:"text"`
	UserId    int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (h *postsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
		}
		text := r.FormValue("text")
		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		defer db.Close()

		row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
		var userID int
		if err := row.Scan(&userID); err != nil {
			log.Fatalf("user_id getRows rows.Scan error err:%v", err)
		}

		if err != nil {
			log.Println(err)
		}
		ins, err := db.Prepare("insert into posts(text, user_id) value (?, ?)")
		if err != nil {
			fmt.Printf("error")
			return
		}
		res, err := ins.Exec(text, userID)
		post_id, err := res.LastInsertId()
		http.Redirect(w, r, fmt.Sprintf("posts/%d", post_id), http.StatusTemporaryRedirect)
		return
	}
	if r.Method == http.MethodGet {
		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		if err != nil {
			fmt.Printf("error")
		}
		rows, err := db.Query("select id, text, user_id, created_at, updated_at from posts")
		defer db.Close()

		var posts []Post
		for rows.Next() {
			p := &Post{}
			if err := rows.Scan(&p.ID, &p.Text, &p.UserId, &p.CreatedAt, &p.UpdatedAt); err != nil {
				log.Fatalf("getRows rows.Scan error err:%v", err)
			}
			posts = append(posts, *p)
		}

		t := template.Must(template.ParseFiles("./template/posts.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, posts); err != nil {
			panic(err.Error())
		}
	}
}

func postsNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Printf("method not allowed")
		return
	}
	t := template.Must(template.ParseFiles("./template/posts-new.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, nil); err != nil {
		panic(err.Error())
	}
}

func main() {
	http.Handle("/count", new(countHandler))
	http.Handle("/users", new(getUsersHandler))
	// for /users/:id
	http.Handle("/users/", new(getUserHandler))
	http.Handle("/users/new", new(newUserHandler))
	http.Handle("/posts", new(postsHandler))
	http.HandleFunc("/posts/new", postsNewHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
