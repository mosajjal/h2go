package h2go

import (
	"fmt"
	"net/http"
)

// HTTP status codes used by the proxy protocol.
const (
	HeadError    = 500
	HeadOK       = 200
	HeadData     = 201
	HeadHeart    = 202
	HeadQuit     = 203
	HeadNotFound = 404
)

// WriteHTTPError writes an HTTP error response with status 500.
func WriteHTTPError(w http.ResponseWriter, message string) {
	w.WriteHeader(HeadError)
	fmt.Fprintf(w, "%s", message)
}

// WriteNotFoundError writes an HTTP not found response with status 404.
func WriteNotFoundError(w http.ResponseWriter, message string) {
	w.WriteHeader(HeadNotFound)
	fmt.Fprintf(w, "%s", message)
}

// WriteHTTPOK writes an HTTP success response with status 200.
func WriteHTTPOK(w http.ResponseWriter, data string) {
	w.WriteHeader(HeadOK)
	fmt.Fprintf(w, "%s", data)
}

// WriteHTTPData writes an HTTP data response with status 201.
func WriteHTTPData(w http.ResponseWriter, data []byte) {
	w.WriteHeader(HeadData)
	w.Write(data)
}

// WriteHTTPQuit writes an HTTP quit response with status 203.
func WriteHTTPQuit(w http.ResponseWriter, data string) {
	w.WriteHeader(HeadQuit)
	fmt.Fprintf(w, "%s", data)
}

// WriteHTTPHeart writes an HTTP heartbeat response with status 202.
func WriteHTTPHeart(w http.ResponseWriter, data string) {
	w.WriteHeader(HeadHeart)
	fmt.Fprintf(w, "%s", data)
}
