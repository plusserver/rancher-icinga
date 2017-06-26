// A small utility that can be used to test the REGISTER_CHANGES feature.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type IcingaEvent struct {
	Operation  string                 `json:"operation"`
	Name       string                 `json:"name"`
	IcingaType string                 `json:"type"`
	Vars       map[string]interface{} `json:"vars"`
}

func main() {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/event", registerEvent).Methods("POST")

	log.Println("Starting to listen")
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(":8765", loggedRouter))
}

func registerEvent(w http.ResponseWriter, r *http.Request) {
	var event IcingaEvent
	fmt.Printf("%+v\n", r)
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	fmt.Println(string(body))
	if err := json.Unmarshal(body, &event); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	} else {

		fmt.Printf("%q\n", event)

		w.WriteHeader(http.StatusCreated)
	}
}
