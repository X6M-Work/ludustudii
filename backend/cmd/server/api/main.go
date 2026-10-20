package main

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/spa"
)

func main() {
	http.Handle("/", spa.Handler())
	http.ListenAndServe(":8080", nil)
}
