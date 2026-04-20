package revproxy

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: rec}
	lrw.WriteHeader(http.StatusNotFound)

	if lrw.status != http.StatusNotFound {
		t.Errorf("lrw.status = %d, want %d", lrw.status, http.StatusNotFound)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("recorder code = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLoggingResponseWriter_Write_DefaultsStatusTo200(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: rec}

	n, err := lrw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("written bytes = %d, want 5", n)
	}
	if lrw.status != http.StatusOK {
		t.Errorf("lrw.status = %d, want %d", lrw.status, http.StatusOK)
	}
	if lrw.bytes != 5 {
		t.Errorf("lrw.bytes = %d, want 5", lrw.bytes)
	}
}

func TestLoggingResponseWriter_Write_AfterWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: rec}
	lrw.WriteHeader(http.StatusCreated)
	lrw.Write([]byte("data"))

	if lrw.status != http.StatusCreated {
		t.Errorf("lrw.status = %d, want %d", lrw.status, http.StatusCreated)
	}
	if lrw.bytes != 4 {
		t.Errorf("lrw.bytes = %d, want 4", lrw.bytes)
	}
}

func TestLoggingResponseWriter_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: rec}

	lrw.Write([]byte("abc"))
	lrw.Write([]byte("de"))
	lrw.Write([]byte("f"))

	if lrw.bytes != 6 {
		t.Errorf("lrw.bytes = %d, want 6", lrw.bytes)
	}
}

func TestAccessLog_PassThrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := accessLog(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

type flushingRecorder struct {
	*httptest.ResponseRecorder
	flushed atomic.Bool
}

func (f *flushingRecorder) Flush() {
	f.flushed.Store(true)
	f.ResponseRecorder.Flush()
}

func TestLoggingResponseWriter_Flush_WithFlusher(t *testing.T) {
	rec := &flushingRecorder{ResponseRecorder: httptest.NewRecorder()}
	lrw := &loggingResponseWriter{ResponseWriter: rec}
	lrw.Flush()
	if !rec.flushed.Load() {
		t.Error("Flush() did not call underlying Flusher")
	}
}

func TestLoggingResponseWriter_Flush_WithoutFlusher(t *testing.T) {
	// httptest.ResponseRecorder implements Flusher, so use a plain struct
	lrw := &loggingResponseWriter{ResponseWriter: struct{ http.ResponseWriter }{httptest.NewRecorder()}}
	// Should not panic when underlying writer does not implement Flusher
	lrw.Flush()
}

func TestAccessLog_StatusDefault200(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("no explicit WriteHeader"))
	})

	wrapped := accessLog(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
