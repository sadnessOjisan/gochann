package router

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
)

// どんな結果だろうと必ずクッキーを消すようにする。early return しない。
func SignoutHandler(w http.ResponseWriter, r *http.Request) {
	dsn := os.Getenv("dbdsn")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("ERROR: db open err: %v", err)
	}
	defer db.Close()

	token, err := r.Cookie("token")
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	ins, err := db.Prepare("delete from session where token =?")
	if err != nil {
		log.Printf("ERROR: prepare token delete err: %v", err)
	}
	_, err = ins.Exec(token.Value)
	if err != nil {
		log.Printf("ERROR: exec token delete err: %v", err)
	}

	cookie := &http.Cookie{
		Name:    "token",
		Expires: time.Now(),
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
