package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	// THink we have to alter this to {"error": "Something went wrong"}
	io.WriteString(w, msg)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {

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
		type paramaters struct {
			Cleaned_body string `json:"body"`
		}

		decoder := json.NewDecoder(req.Body)
		params := paramaters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}
		if len(params.Cleaned_body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		// Cleanup here
		badWords := []string{"kerfuffle", "sharbert", "fornax"}
		wordsInBody := strings.Split(params.Cleaned_body, " ")
		fmt.Println(params.Cleaned_body)
		fmt.Println(wordsInBody)
		resultWords := make([]string, len(wordsInBody))
		for _, bodyWord := range wordsInBody {
			if slices.Contains(badWords, bodyWord) {
				resultWords = append(resultWords, "****")
			} else {
				resultWords = append(resultWords, bodyWord)
			}
		}
		params.Cleaned_body = strings.Join(resultWords, "")
		fmt.Println(params.Cleaned_body)
		fmt.Println(resultWords)

		type response struct {
			Cleaned_body string `json:"cleaned_body"`
		}

		resp := response{Cleaned_body: params.Cleaned_body}
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
