package main

import (
	"log"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	log.Println("chat web running on http://localhost:12346 ...")
	if err := http.ListenAndServe(":12346", nil); err != nil {
		log.Fatal(err)
	}
}
