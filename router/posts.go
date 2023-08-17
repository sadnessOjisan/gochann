package router

import (
	"database/sql"
	"fmt"
	"html/template"
	"learn-go-server/model"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func PostsNewHandler(w http.ResponseWriter, r *http.Request) {
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

func PostsDetailHandler(w http.ResponseWriter, r *http.Request) {
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

		post := &model.Post{}
		for rows.Next() {
			comment := &model.Comment{}
			post_user := &model.User{}
			comment_user := &model.User{}
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

func PostsHandler(w http.ResponseWriter, r *http.Request) {
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
		var posts []model.Post
		for rows.Next() {
			p := &model.Post{}
			u := &model.User{}
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
