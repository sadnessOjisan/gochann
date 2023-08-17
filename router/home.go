package router

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
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
