/******** Peter Winzell (c), 10/30/23 *********************************************/

package mapserver

import (
	"github.com/gorilla/mux"
	"net/http"
)

const (
	ReleaseTag = "server 0.0.1"
)

var pingHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// log.Println("pingHandler called ...")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(" HTTP status(200) OK code returned running:  " + ReleaseTag))
})

func ServeWebViewSite(r *mux.Router) error {

	r.Handle("/version", pingHandler).Methods("GET")
	r.Handle("/", http.FileServer(http.Dir("./static/")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	return http.ListenAndServe(":8085", r)

}
