package main

import (
	"log"
	"net/http"
	"github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./w3cDemo/")))
    log.Fatal(http.ListenAndServe(":7100", r))
}