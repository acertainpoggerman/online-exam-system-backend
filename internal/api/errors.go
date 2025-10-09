package api

import (
	"log/slog"
	"net/http"
)

func LogError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	slog.Error(err.Error(), "method", method, "uri", uri)
}

func WriteErrorResponse(w http.ResponseWriter, r *http.Request, statusCode int, msg any) {
	msgWrapped := Envelope{"error": msg}

	if err := WriteJSON(w, statusCode, msgWrapped, nil); err != nil {
		LogError(r, err)
		w.WriteHeader(500)
	}
}

// Premade Errors

func UnauthorizedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "User not Authorized to Access Endpoint."
	WriteErrorResponse(w, r, http.StatusUnauthorized, msg)
}

func InternalServerErrorResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Unexpected Error Occurred."
	WriteErrorResponse(w, r, http.StatusInternalServerError, msg)
}

func BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	WriteErrorResponse(w, r, http.StatusBadRequest, err)
}

func MethodNotImplementedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Method not Implemented."
	WriteErrorResponse(w, r, http.StatusNotImplemented, msg)
}
