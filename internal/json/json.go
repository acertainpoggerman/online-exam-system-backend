package json

import (
	"encoding/json"
	"net/http"
)

type Wrapper map[string]any

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

// Pass reference of out ( i.e. ReadRequestBody(r, &out) )
func ReadRequestBody(r *http.Request, out any) error {

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(out)
}

// Pass reference of out ( i.e. Unmarshal(data, &out) )
func Unmarshal(data []byte, out any) error {
	return json.Unmarshal(data, out)
}

func Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}
