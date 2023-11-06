package httpio

import (
	"encoding/json"
	"io"
	"net/http"
)

func Decode(r io.Reader, data any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(&data)
}

func WriteJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func BadRequestResponse(w http.ResponseWriter, msg string) error {
	return WriteJSON(w, http.StatusBadRequest, map[string]string{
		"error": msg,
	})
}

func InternalServerErrorResponse(w http.ResponseWriter, msg string) error {
	return WriteJSON(w, http.StatusInternalServerError, map[string]string{
		"error": msg,
	})
}
