package api

import (
	"encoding/json"
	"net/http"
)

type Envelope map[string]any

func WriteJSON(w http.ResponseWriter, statusCode int, data any, headers http.Header) error {

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dataJSON = append(dataJSON, '\n')

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(dataJSON)

	return nil
}
