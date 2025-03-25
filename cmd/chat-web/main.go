package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	wsServer := os.Getenv("WEBSOCKET_SERVER")
	if wsServer == "" {
		wsServer = "localhost:80"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.FileServer(http.Dir("public")).ServeHTTP(w, r)
			return
		}

		tmpl, err := template.ParseFiles("public/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			WSServer string
		}{
			WSServer: wsServer,
		}

		tmpl.Execute(w, data)
	})

	log.Println("chat web running on port:8080 ...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
