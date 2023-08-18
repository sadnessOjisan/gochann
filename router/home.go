package router

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("token")
	// cookie に token がないなら home ページを表示
	if err != nil {
		t := template.Must(template.ParseFiles("./template/home.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, nil); err != nil {
			log.Printf("ERROR: exec templating err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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

	row := db.QueryRow("select user_id from session where token = ? limit 1", token.Value)
	var user_id int
	if err := row.Scan(&user_id); err != nil {
		// token に紐づくユーザーがないので認証エラー。token リセットしてホームに戻す。
		cookie := &http.Cookie{
			Name:    "token",
			Expires: time.Now(),
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// cookie の情報が session になかった場合
	if user_id == 0 {
		t := template.Must(template.ParseFiles("./template/home.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, nil); err != nil {
			log.Printf("ERROR: exec templating err: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// user 情報が見つかった時
	http.Redirect(w, r, "/posts", http.StatusSeeOther)
}
