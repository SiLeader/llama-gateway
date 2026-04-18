package route

import (
	"net/http"
	"strings"
)

type Match struct {
	endpoints []Endpoint
}

type Endpoint struct {
	method string
	prefix *string
	exact  *string
}

func NewMatch(endpoint ...Endpoint) *Match {
	return &Match{endpoints: endpoint}
}

func (r *Match) IsMatch(q *http.Request) bool {
	for _, e := range r.endpoints {
		if e.IsMatch(q) {
			return true
		}
	}
	return false
}

func (e Endpoint) IsMatch(q *http.Request) bool {
	if e.method != q.Method {
		return false
	}
	if e.exact != nil && *e.exact == q.URL.Path {
		return true
	}
	if e.prefix != nil && strings.HasPrefix(q.URL.Path, *e.prefix) {
		return true
	}
	return false
}

func GetExact(exact string) Endpoint {
	return Endpoint{
		method: "GET",
		exact:  &exact,
	}
}

func PostExact(exact string) Endpoint {
	return Endpoint{
		method: "POST",
		exact:  &exact,
	}
}

func GetPrefix(prefix string) Endpoint {
	return Endpoint{
		method: "GET",
		prefix: &prefix,
	}
}

func PostPrefix(prefix string) Endpoint {
	return Endpoint{
		method: "POST",
		prefix: &prefix,
	}
}
