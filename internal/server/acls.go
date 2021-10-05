package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

type ctxKeyType string

type AuthTypeValue string

const (
	AuthType ctxKeyType = "AUTH_TYPE"
	Username ctxKeyType = "USERNAME"

	AuthTypeWebSession AuthTypeValue = "WEB_SESSION"
	AuthTypeBearer     AuthTypeValue = "BEARER"
)

func getACLMiddleware() []mux.MiddlewareFunc {
	// see adamlouis/goq for auth checks
	return []mux.MiddlewareFunc{
		// auth for web session
		// func(next http.Handler) http.Handler {
		// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 		if p, _ := sessionManager.Get(w, r); p != nil {
		// 			if p.Authenticated {
		// 				ctx := context.WithValue(
		// 					context.WithValue(
		// 						r.Context(),
		// 						AuthType,
		// 						AuthTypeWebSession),
		// 					Username,
		// 					p.Username,
		// 				)
		// 				r = r.WithContext(ctx)
		// 			}
		// 		}
		// 		next.ServeHTTP(w, r)
		// 	})
		// },
		// // auth for api bearer token
		// func(next http.Handler) http.Handler {
		// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 		az := r.Header["Authorization"]
		// 		if len(az) == 1 && strings.HasPrefix(az[0], _bearerPrefix) {
		// 			hv := az[0]
		// 			token := hv[len(_bearerPrefix):]
		// 			if apiKeyChecker.Check(token) {
		// 				ctx := context.WithValue(r.Context(), AuthType, AuthTypeBearer)
		// 				r = r.WithContext(ctx)
		// 			}
		// 		}
		// 		next.ServeHTTP(w, r)
		// 	})
		// },
		// authz for all routes
		// could make this more declarative
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// by default, DO NOT allow ... unless some condition is met
				allow := false

				// allow all gets
				if r.Method == http.MethodGet {
					allow = true
				}

				// TODO: allow posts ... for now, I just want to ship a read-only demo

				// allow if some condition is met
				if allow {
					next.ServeHTTP(w, r)
					return
				}

				http.Error(w, "forbidden", http.StatusForbidden)
			})
		},
	}

}
