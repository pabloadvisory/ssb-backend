package httpx

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

func WriteCacheableJSON(writer http.ResponseWriter, request *http.Request, status int, value any, maxAge time.Duration) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(body)
	etag := fmt.Sprintf(`"%x"`, digest)
	writer.Header().Set("ETag", etag)
	writer.Header().Set("Cache-Control", "public, max-age="+strconv.FormatInt(int64(maxAge/time.Second), 10)+", stale-while-revalidate=30")
	for _, candidate := range strings.Split(request.Header.Get("If-None-Match"), ",") {
		candidate = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(candidate), "W/"))
		if candidate == "*" || candidate == etag {
			writer.WriteHeader(http.StatusNotModified)
			return nil
		}
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, err = writer.Write(append(body, '\n'))
	return err
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func WriteError(writer http.ResponseWriter, request *http.Request, status int, code, message string) {
	WriteJSON(writer, status, ErrorEnvelope{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFromContext(request.Context()),
	}})
}

func DecodeJSON(writer http.ResponseWriter, request *http.Request, destination any, maxBytes int64) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("malformed JSON at byte %d", syntaxError.Offset)
		case errors.As(err, &typeError):
			return fmt.Errorf("invalid value for %s", typeError.Field)
		case errors.Is(err, io.EOF):
			return errors.New("request body is empty")
		case errors.Is(err, http.ErrBodyReadAfterClose):
			return errors.New("request body could not be read")
		default:
			return err
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}
