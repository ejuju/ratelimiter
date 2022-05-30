package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMaxRequests(t *testing.T) {
	// init test server with ratelimiter middleware
	var router http.Handler = http.NewServeMux()
	limiter := NewDefaultLimiter(1, time.Second, time.Minute)
	router = New(limiter)(router) // register rate limiter middleware

	// send valid request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resrec := httptest.NewRecorder()
	router.ServeHTTP(resrec, req)

	// send invalid request
	resrec = httptest.NewRecorder()
	router.ServeHTTP(resrec, req)

	// expect ban from middlewares
	want := http.StatusTooManyRequests
	got := resrec.Result().StatusCode
	if want != got {
		t.Fatalf("want status %v but got %v", want, got)
	}
}
