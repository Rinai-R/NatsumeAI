// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import "net/http"

type CasbinMiddleware struct {
}

func NewCasbinMiddleware() *CasbinMiddleware {
	return &CasbinMiddleware{}
}

func (m *CasbinMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO generate middleware implement function, delete after code implementation

		// Passthrough to next handler if need
		next(w, r)
	}
}
