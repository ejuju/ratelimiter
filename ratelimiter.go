package ratelimiter

import (
	"fmt"
	"net/http"
	"time"
)

//
type RateLimiter interface {
	Identify(r *http.Request) string // returns the identifier of the user from the http.Request
	StoreNewRequest(id string)       // adds a new request to the user's history and returns false if the user should be banned
	Allow(id string) bool            // returns true if user is allowed to make this request, if false, user should be banned
	Ban(id string)                   // bans the user
	IsBanned(id string) bool         // returns true if the user is banned
}

// History stores the last requests an IP address has made and the last time it was banned, as well as a mutex to sync read/write operations
type History struct {
	lastBan  time.Time
	requests []time.Time
}

// Users ...
type Users map[string]*History // maps the user identifier to its history

// DefaultLimiter implements the RateLimiter interface using the IP address as the user identifier
type DefaultLimiter struct {
	users       Users
	window      time.Duration
	banDuration time.Duration
	maxRequests int
}

// Identify returns the IP addr for the current request
func (l *DefaultLimiter) Identify(r *http.Request) string {
	return r.RemoteAddr
}

//
func (l *DefaultLimiter) Allow(id string) bool {
	h, ok := l.users[id]
	if ok == false || h == nil || len(h.requests) == 0 {
		return true
	}
	return isValidRequest(h.requests, l.window, l.maxRequests)
}

//
func (l *DefaultLimiter) IsBanned(id string) bool {
	h, ok := l.users[id]
	if ok == false {
		return false
	}
	return h.lastBan.Add(l.window).Sub(time.Now()) > 0
}

//
func (l *DefaultLimiter) Ban(id string) {
	l.users[id].lastBan = time.Now()
}

//
func (l *DefaultLimiter) StoreNewRequest(id string) {
	h, ok := l.users[id]
	if ok == false {
		l.users[id] = &History{
			requests: []time.Time{time.Now()},
		}
		return
	}
	if len(h.requests) == 0 {
		h.requests = []time.Time{time.Now()}
		return
	}
	h.requests = append(h.requests, time.Now())
}

// NewDefaultLimiter ...
// maxRequests represents the maximum number of requests a given IP address is allowed to make in the given time duration window
// window represents the duration over which the requests are counted
// banDuration represents the duration for how long the IP addr gets banned if it exceed the max. requests in the given time window
func NewDefaultLimiter(maxRequests int, window, banDuration time.Duration) *DefaultLimiter {
	// set default max requests to 100 if invalid input
	if maxRequests <= 0 {
		maxRequests = 100
	}
	// set default window to 1 minute if invalid input
	if window <= 0 {
		window = 1 * time.Minute
	}
	// set default ban duration to 30 minutes if invalid input
	if banDuration <= 0 {
		banDuration = 30 * time.Minute
	}

	users := make(map[string]*History)
	return &DefaultLimiter{
		users:       users,
		maxRequests: maxRequests,
		window:      window,
		banDuration: banDuration,
	}
}

// New initiates and return the middleware function that will limit the incoming requests
func New(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get identifier
			id := limiter.Identify(r)
			fmt.Println(id, limiter.IsBanned(id))
			// check if user is banned, if yes, abort request
			banned := limiter.IsBanned(id)
			if banned {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			// save new request to user history
			limiter.StoreNewRequest(id)

			// check if request should be allowed
			ok := limiter.Allow(id)
			if ok == false {
				limiter.Ban(id)
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Returns true if last requests respected the limit
func isValidRequest(reqs []time.Time, window time.Duration, maxReqs int) bool {
	reqsLen := len(reqs)

	if reqsLen < maxReqs {
		return true
	}

	oldest := reqs[reqsLen-maxReqs]
	latest := reqs[reqsLen-1]
	if latest.Sub(oldest) > window {
		return true
	}

	return false
}
