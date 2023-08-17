package router

import (
	"net/http"
	"time"
)

func SignoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:    "token",
		Expires: time.Now(),
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
