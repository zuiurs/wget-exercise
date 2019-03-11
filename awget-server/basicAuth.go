package main

import (
	"fmt"
	"net/http"
)

const (
	user     = "hello"
	password = "world"
)

func basicAuthHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || !auth(u, p) {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"SECRET AREA\"")
			http.Error(w, fmt.Sprintf("Unauthorized user: %s, %s, %t", u, p, ok), http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func auth(u, p string) bool {
	return user == u && password == p
}
