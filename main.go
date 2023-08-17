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

type Comment struct {
	ID        int       `db:"id"`
	Text      string    `db:"text"`
	User      User      `db:"user"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Post struct {
	ID        int       `db:"id"`
	Text      string    `db:"text"`
	User      User      `db:"user"`
	Comments  []Comment `db:"comments"`
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
		rows, err := db.Query(`
		  select
		    p.id, p.text, p.created_at, p.updated_at,
			u.id as user_id, u.name as user_name
		  from
		    posts p
		  inner join
		    users u
		  on
		    user_id = u.id
		`)
		defer db.Close()
		if err != nil {
			println("rows scan fail")
			panic(err.Error())
		}
		var posts []Post
		for rows.Next() {
			p := &Post{}
			u := &User{}
			if err := rows.Scan(&p.ID, &p.Text, &p.CreatedAt, &p.UpdatedAt, &u.ID, &u.Name); err != nil {
				log.Fatalf("getRows rows.Scan error err:%v", err)
			}
			p.User = *u
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

func postsDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sub := strings.TrimPrefix(r.URL.Path, "/posts")
		_, id := filepath.Split(sub)
		if id == "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Printf("id is not found")
			return
		}
		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		defer db.Close()
		if err != nil {
			fmt.Printf("error")
		}
		query := `
		  select
		    p.id, p.text, p.created_at, p.updated_at,
			post_user.id, post_user.name,
			c.id as comment_id, c.text as comment_text, c.created_at as comment_created_at, c.updated_at as comment_updated_at,
			comment_user.id, comment_user.name
		  from posts p
		  inner join comments c
		  on p.id = c.post_id
		  inner join users post_user
		  on p.user_id = post_user.id
		  inner join users comment_user
		  on c.user_id = comment_user.id
		  where p.id = ?
		  order by c.id
		`
		rows, err := db.Query(query, id)
		if err != nil {
			println("db query error")
			panic(err.Error())
		}
		println("rows: ", rows)

		post := &Post{}
		for rows.Next() {
			comment := &Comment{}
			post_user := &User{}
			comment_user := &User{}
			err = rows.Scan(
				&post.ID, &post.Text, &post.CreatedAt, &post.UpdatedAt,
				&post_user.ID, &post_user.Name,
				&comment.ID, &comment.Text, &comment.CreatedAt, &comment.UpdatedAt,
				&comment_user.ID, &comment_user.Name,
			)
			if err != nil {
				log.Fatalf("%v", *comment)
				log.Fatalf("%v", err)
				return
			}
			post.User = *post_user
			comment.User = *comment_user
			post.Comments = append(post.Comments, *comment)
		}

		t := template.Must(template.ParseFiles("./template/post-detail.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := t.Execute(w, post); err != nil {
			panic(err.Error())
		}
		return
	}

	if r.Method == http.MethodPost {
		text := r.FormValue("text")
		segments := strings.Split(r.URL.Path, "/")
		if len(segments) != 4 || segments[2] == "" || segments[3] != "comments" {
			http.NotFound(w, r)
			return
		}
		post_id := segments[2]
		db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
		if err != nil {
			log.Fatalf("open db error err:%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer db.Close()

		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
		var user_id int
		if err := row.Scan(&user_id); err != nil {
			log.Fatalf("user_id getRows rows.Scan error err:%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ins, err := db.Prepare("insert into comments(text, post_id, user_id) value (?, ?, ?)")
		if err != nil {
			log.Fatalf("prepare error err:%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = ins.Exec(text, post_id, user_id)

		http.Redirect(w, r, fmt.Sprintf("/posts/%s", post_id), http.StatusSeeOther)
		return
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("token")
	// cookie に token がないなら home ページを表示
	if err != nil {
		t := template.Must(template.ParseFiles("./template/home.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, nil); err != nil {
			log.Fatalf("tempalte engine rendering error err:%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	db, err := sql.Open("mysql", "ojisan:ojisan@(127.0.0.1:3306)/micro_post?parseTime=true")
	defer db.Close()
	if err != nil {
		log.Fatalf("open db error err:%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
	var user_id int
	if err := row.Scan(&user_id); err != nil {
		log.Fatalf("user_id getRows rows.Scan error err:%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// cookie の情報が session になかった場合
	if user_id == 0 {
		t := template.Must(template.ParseFiles("./template/home.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, nil); err != nil {
			log.Fatalf("tempalte engine rendering error err:%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// user 情報が見つかった時
	http.Redirect(w, r, "/posts", http.StatusSeeOther)
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.Handle("/count", new(countHandler))
	http.Handle("/users", new(getUsersHandler))
	// for /users/:id
	http.Handle("/users/", new(getUserHandler))
	http.Handle("/users/new", new(newUserHandler))
	http.HandleFunc("/posts/", postsDetailHandler)
	http.Handle("/posts", new(postsHandler))
	http.HandleFunc("/posts/new", postsNewHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
