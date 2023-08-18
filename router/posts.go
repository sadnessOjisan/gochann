package router

import (
	"database/sql"
	"fmt"
	"html/template"
	"learn-go-server/model"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func PostsNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("ERROR: invalid method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := r.Cookie("token")
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dsn := os.Getenv("dbdsn")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("ERROR: db open err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer db.Close()
	signin_user_query := `
		  select
		    users.id, users.name
		  from
		    session
		  inner join
		    users
		  on
		    users.id = session.user_id
		  where
		    token = ?
		`
	row := db.QueryRow(signin_user_query, token.Value)
	u := &model.User{}
	if err := row.Scan(&u.ID, &u.Name); err != nil {
		// token に紐づくユーザーがないので認証エラー。token リセットしてホームに戻す。
		cookie := &http.Cookie{
			Name:    "token",
			Expires: time.Now(),
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	t := template.Must(template.ParseFiles("./template/posts-new.html", "./template/_header.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, u); err != nil {
		log.Printf("ERROR: exec templating err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func PostsDetailHandler(w http.ResponseWriter, r *http.Request) {
	// GET /posts/:id
	if r.Method == http.MethodGet {
		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
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
		signin_user_query := `
		  select
		    users.id, users.name
		  from
		    session
		  inner join
		    users
		  on
		    users.id = session.user_id
		  where
		    token = ?
		`
		row := db.QueryRow(signin_user_query, token.Value)
		u := &model.User{}
		if err := row.Scan(&u.ID, &u.Name); err != nil {
			// token に紐づくユーザーがないので認証エラー。token リセットしてホームに戻す。
			cookie := &http.Cookie{
				Name:    "token",
				Expires: time.Now(),
			}

			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		sub := strings.TrimPrefix(r.URL.Path, "/posts")
		_, id := filepath.Split(sub)
		if id == "" {
			log.Printf("ERROR: post id not found err: %v", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := `
		  select
		    p.id, p.title, p.text, p.created_at, p.updated_at,
			post_user.id, post_user.name,
			c.id as comment_id, c.text as comment_text, c.created_at as comment_created_at, c.updated_at as comment_updated_at,
			comment_user.id, comment_user.name
		  from
		    posts p
		  join
		    users post_user
		  on
		    p.user_id = post_user.id
		  left join
		    comments c
		  on
		    p.id = c.post_id
		  left join
		    users comment_user
		  on
		    c.user_id = comment_user.id
		  where
		    p.id = ?
		`
		rows, err := db.Query(query, id)
		if err != nil {
			log.Printf("ERROR: exec posts query err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		post := &model.Post{}
		for rows.Next() {
			fmt.Println("rows scan")
			post_user := &model.User{}
			comment_dto := &struct {
				ID        sql.NullInt16
				Text      sql.NullString
				CreatedAt sql.NullTime
				UpdatedAt sql.NullTime
			}{}
			user_dto := &struct {
				ID   sql.NullInt16
				Name sql.NullString
			}{}
			err = rows.Scan(
				&post.ID, &post.Title, &post.Text, &post.CreatedAt, &post.UpdatedAt,
				&post_user.ID, &post_user.Name,
				&comment_dto.ID, &comment_dto.Text, &comment_dto.CreatedAt, &comment_dto.UpdatedAt,
				&user_dto.ID, &user_dto.Name,
			)
			if err != nil {
				log.Printf("ERROR: posts db scan err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			post.User = *post_user
			if comment_dto.ID.Int16 != 0 {
				post.Comments = append(post.Comments, model.Comment{
					ID:        int(comment_dto.ID.Int16),
					Text:      comment_dto.Text.String,
					CreatedAt: comment_dto.CreatedAt.Time,
					UpdatedAt: comment_dto.UpdatedAt.Time,
					User: model.User{
						ID:   int(user_dto.ID.Int16),
						Name: user_dto.Name.String,
					},
				})
			}

		}

		t := template.Must(template.ParseFiles("./template/post-detail.html", "./template/_header.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Println(post.Comments)

		if err := t.Execute(w, struct {
			model.Post
			model.User
		}{Post: *post, User: *u}); err != nil {
			log.Printf("ERROR: exec templating err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	// POST /posts/:id/comments
	if r.Method == http.MethodPost {
		text := r.FormValue("text")
		segments := strings.Split(r.URL.Path, "/")
		if len(segments) != 4 || segments[2] == "" || segments[3] != "comments" {
			http.NotFound(w, r)
			return
		}
		post_id := segments[2]

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()
		if err != nil {
			log.Printf("ERROR: db open err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
		var user_id int
		if err := row.Scan(&user_id); err != nil {
			log.Printf("ERROR: db scan user err: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ins, err := db.Prepare("insert into comments(text, post_id, user_id) value (?, ?, ?)")
		if err != nil {
			log.Printf("ERROR: prepare comment insert err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = ins.Exec(text, post_id, user_id)

		http.Redirect(w, r, fmt.Sprintf("/posts/%s", post_id), http.StatusSeeOther)
		return
	}
}

func PostsHandler(w http.ResponseWriter, r *http.Request) {
	// POST /posts
	if r.Method == http.MethodPost {
		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
		}
		title := r.FormValue("title")
		text := r.FormValue("text")

		dsn := os.Getenv("dbdsn")
		db, err := sql.Open("mysql", dsn)
		defer db.Close()

		row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
		var userID int
		if err := row.Scan(&userID); err != nil {
			log.Printf("ERROR: db scan user err: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ins, err := db.Prepare("insert into posts(title, text, user_id) value (?, ?, ?)")
		if err != nil {
			fmt.Printf("error")
			return
		}
		res, err := ins.Exec(title, text, userID)
		post_id, err := res.LastInsertId()
		http.Redirect(w, r, fmt.Sprintf("posts/%d", post_id), http.StatusSeeOther)
		return
	}

	// GET /posts
	if r.Method == http.MethodGet {
		token, err := r.Cookie("token")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
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
		signin_user_query := `
		  select
		    users.id, users.name
		  from
		    session
		  inner join
		    users
		  on
		    users.id = session.user_id
		  where
		    token = ?
		`
		row := db.QueryRow(signin_user_query, token.Value)
		u := &model.User{}
		if err := row.Scan(&u.ID, &u.Name); err != nil {
			log.Printf("ERROR: db scan user err: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
		  select
		    p.id, p.title, p.text, p.created_at, p.updated_at,
			u.id as user_id, u.name as user_name
		  from
		    posts p
		  inner join
		    users u
		  on
		    user_id = u.id
		`)
		defer db.Close()

		var posts []model.Post
		for rows.Next() {
			p := &model.Post{}
			u := &model.User{}
			if err := rows.Scan(&p.ID, &p.Title, &p.Text, &p.CreatedAt, &p.UpdatedAt, &u.ID, &u.Name); err != nil {
				log.Printf("ERROR: db scan post err: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			p.User = *u
			posts = append(posts, *p)
		}

		t := template.Must(template.ParseFiles("./template/posts.html", "./template/_header.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, struct {
			Posts []model.Post
			model.User
		}{Posts: posts, User: *u}); err != nil {
			log.Printf("ERROR: exec templating err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
