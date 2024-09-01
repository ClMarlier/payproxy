package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type payload struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/yo/man", func(w http.ResponseWriter, r *http.Request) {
		var p payload
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		fmt.Println(p)
		w.Write([]byte("success"))
	})
	http.ListenAndServe(":9998", mux)
}
