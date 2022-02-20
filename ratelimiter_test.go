package ratelimiter

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestEndToEnd(t *testing.T) {
	maxRequests := 10

	// init test server with ratelimiter middleware
	router := mux.NewRouter()
	limiter := NewDefaultLimiter(maxRequests, time.Second, time.Minute)
	router.Use(New(limiter)) // register rate limiter middleware
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	server := http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    8000,
	}

	// listen and serve server
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(1 * time.Second) // wait for server to start

	// test clients
	nbReqs := maxRequests + 1
	reqInterval := 100 * time.Millisecond

	go func() {
		for i := 0; i < nbReqs; i++ {
			_, err := http.Get("http://localhost:8080/")
			if err != nil {
				return
			}
			// if res.StatusCode == http.StatusTooManyRequests {
			// 	if i+1 != maxRequests {
			// 		t.Error(fmt.Errorf("should have gotten banned after %v requests, got banned after %v instead", maxRequests, i+1))
			// 		return
			// 	}
			// }

			// time.Sleep(reqInterval)
		}
	}()

	for i := 0; i < nbReqs; i++ {
		res, err := http.Get("http://localhost:8080/")
		if err != nil {
			t.Error(err)
			return
		}
		if res.StatusCode == http.StatusTooManyRequests {
			if i+1 != maxRequests {
				t.Error(fmt.Errorf("should have gotten banned after %v requests, got banned after %v instead", maxRequests, i+1))
				return
			}
		}

		time.Sleep(reqInterval)
	}
}
