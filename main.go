package main

import (
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func main() {
	mux := http.NewServeMux()
	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()

}
