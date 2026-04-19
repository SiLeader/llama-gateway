package revproxy

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	// WriteHeader が明示的に呼ばれていない場合は 200
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.bytes += n
	return n, err
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{
			ResponseWriter: w,
		}

		next.ServeHTTP(lrw, r)

		if lrw.status == 0 {
			lrw.status = http.StatusOK
		}

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		// nginx の combined log に近い形式
		// $remote_addr - - [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
		msg := fmt.Sprintf(
			`%s - - [%s] "%s %s %s" %d %d %q %q %v`,
			host,
			start.Format(time.RFC3339),
			r.Method,
			r.URL.RequestURI(),
			r.Proto,
			lrw.status,
			lrw.bytes,
			r.Referer(),
			r.UserAgent(),
			time.Since(start),
		)
		slog.InfoContext(r.Context(), msg)
	})
}
