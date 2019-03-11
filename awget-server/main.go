package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr    string
	listenTLSAddr string
	healthy       int32
)

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "server listen address")
	flag.StringVar(&listenTLSAddr, "listen-tls-addr", ":8081", "server listen tls address")
	flag.Parse()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go httpServer(wg, listenAddr)
	go httpsServer(wg, listenTLSAddr)
	wg.Wait()
}

func httpServer(wg *sync.WaitGroup, addr string) {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/hello", contentText("Hello World!"))
	router.Handle("/secret/hello", basicAuthHandler(contentText("[Secret] Hello World!")))
	router.Handle("/timeout/hello", contentCapriceText("[Timeout] Hello World!"))
	router.Handle("/large1", contentBinary("/home/zuiurs/bin1"))
	router.Handle("/large2", contentBinary("/home/zuiurs/bin2"))
	router.Handle("/healthz", healthz())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:     listenAddr,
		Handler:  tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog: logger,
		//		ReadTimeout:  5 * time.Second,
		//		WriteTimeout: 10 * time.Second,
		//		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		defer wg.Done()

		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Server stopped")
}

func httpsServer(wg *sync.WaitGroup, addr string) {
	logger := log.New(os.Stdout, "https: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/hello", contentText("Hello World!"))
	router.Handle("/secret/hello", basicAuthHandler(contentText("[Secret] Hello World!")))
	router.Handle("/timeout/hello", contentCapriceText("[Timeout] Hello World!"))
	router.Handle("/large1", contentBinary("/home/zuiurs/bin1"))
	router.Handle("/large2", contentBinary("/home/zuiurs/bin2"))
	router.Handle("/healthz", healthz())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:     listenTLSAddr,
		Handler:  tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog: logger,
		//		ReadTimeout:  5 * time.Second,
		//		WriteTimeout: 10 * time.Second,
		//		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		defer wg.Done()

		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenTLSAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenTLSAddr, err)
	}

	<-done
	logger.Println("Server stopped")
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
