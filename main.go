package main

import (
	"fmt"
	"log"
	"net/http"
)

// カウンターを持つ HTTP リクエストハンドラー
type countHandler struct {
	count int
}

func (h *countHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.count++
	fmt.Fprintf(w, "Count: %d\n", h.count)
}
func main() {
	http.Handle("/count", new(countHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
