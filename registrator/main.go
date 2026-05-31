package main

import (
	"log"
	"net/http"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("[registrator] ")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
