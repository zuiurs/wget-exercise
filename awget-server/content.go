package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func contentText(s string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, s)
	})
}

func contentCapriceText(s string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n := rand.Intn(9); n < 6 {
			time.Sleep(3 * time.Second)
		}
		contentText(s).ServeHTTP(w, r)
	})
}

func contentBinary(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "File Open Error", http.StatusInternalServerError)
			return
		}

		fi, err := f.Stat()
		if err != nil {
			http.Error(w, "Get file stat error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(int(fi.Size())))
		w.WriteHeader(http.StatusOK)

		_, err = io.Copy(w, f)
		if err != nil {
			http.Error(w, "Write Error", http.StatusInternalServerError)
			return
		}
	})
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}
