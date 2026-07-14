package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const MaxBodyBytes = 1 << 20

type Reporter struct {
	Logger *slog.Logger
}

func NewReporter(logger *slog.Logger) Reporter {
	if logger == nil {
		logger = slog.Default()
	}
	return Reporter{Logger: logger}
}

func (r Reporter) Problem(w http.ResponseWriter, status int, code, message, operation string, entityID int64, err error) {
	args := []any{"error_code", code, "operation", operation}
	if entityID != 0 {
		args = append(args, "entity_id", entityID)
	}
	if err != nil {
		args = append(args, "error", err)
	}
	if status >= 500 {
		r.Logger.Error("request failed", args...)
	} else {
		r.Logger.Warn("request rejected", args...)
	}
	WriteError(w, status, code, message, nil)
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	WriteJSON(w, status, map[string]any{"error": map[string]any{
		"code": code, "message": message, "details": details,
	}})
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fmt.Errorf("request body is too large: %w", err)
		}
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func DecodeOrProblem(reporter Reporter, w http.ResponseWriter, r *http.Request, dst any, operation string) bool {
	if err := DecodeJSON(r, dst); err != nil {
		status := http.StatusBadRequest
		code := "INVALID_JSON"
		message := "Request body must be valid JSON"
		if strings.Contains(err.Error(), "too large") {
			status, code, message = http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large"
		}
		reporter.Problem(w, status, code, message, operation, 0, err)
		return false
	}
	return true
}

func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}

func NewRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(value[:])
}

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (w *ResponseWriter) WriteHeader(status int) {
	if w.Status != 0 {
		return
	}
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *ResponseWriter) Write(body []byte) (int, error) {
	if w.Status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *ResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := NewRequestID()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		}
		rw := &ResponseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("http panic", "request_id", requestID, "error", fmt.Sprint(recovered),
					"error_code", "INTERNAL_ERROR", "operation", "http_request")
				if rw.Status == 0 {
					WriteError(rw, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
				}
			}
			status := rw.Status
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request", "request_id", requestID, "method", r.Method,
				"path", r.URL.Path, "status", status, "duration_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(rw, r)
	})
}
