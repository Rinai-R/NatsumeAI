package middleware

import "net/http"


type CasbinMiddleware struct {
}

func NewCasbinMiddleware() *CasbinMiddleware {
	return &CasbinMiddleware{}
}

func (m *CasbinMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		next(w, r)
	}
}