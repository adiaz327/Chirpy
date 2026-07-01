package main

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"encoding/json"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) getFileserverHits() int32 {
	return cfg.fileserverHits.Load()
}

func (cfg *apiConfig) resetFileserverHits() {
	cfg.fileserverHits.Store(0)
}

func respondWithError(w http.ResponseWriter, code int, msg string){
	w.WriteHeader(code)
	// THink we have to alter this to {"error": "Something went wrong"}
	io.WriteString(w, msg)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}){

}

func main() {
	cfg := apiConfig{}
	mux := http.NewServeMux()
	fileHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", fileHandler)))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, "200 OK")
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		io.WriteString(w, fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.getFileserverHits()))
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		cfg.resetFileserverHits()
		w.WriteHeader(200)
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, req *http.Request) {
		type paramaters struct{
			Body string `json:"body"`
		}

		decoder := json.NewDecoder(req.Body)
		params := paramaters{}
		err := decoder.Decode(&params)
		if err != nil{
			respondWithError(w, 400, "Something went wrong")
			return
		}
		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return 
		}

		type response struct {
			Valid bool `json:"valid"`
		}

		resp := response{Valid: true}
		dat, err := json.Marshal(resp)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		req.Header.Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()

}
