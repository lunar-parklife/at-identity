package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/lunar-parklife/at-identity/did"
)

var resolver *did.Resolver = did.DefaultResolver()

type errorMessage struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func sendError(w http.ResponseWriter, message *errorMessage) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Add("Content-Type", "text/json")
	messageBytes, _ := json.Marshal(message)
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(messageBytes)))
	_, _ = w.Write(messageBytes)
}

func resolveDID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	query := r.URL.Query()
	if !query.Has("did") {
		message := errorMessage{
			Error:   "InvalidRequest",
			Message: "A DID is required.",
		}
		sendError(w, &message)
		return
	}
	id, err := did.Parse(query.Get("did"))
	if err != nil {
		message := errorMessage{
			Error:   "InvalidRequest",
			Message: err.Error(),
		}
		sendError(w, &message)
		return
	}
	doc, err := resolver.Doc(id)
	if err != nil {
		plcerr, ok := err.(did.PLCError)
		if ok {
			var errType string
			switch plcerr.StatusCode {
			case 404:
				errType = "DidNotFound"
			case 410:
				errType = "DidDeactivated"
			default:
				errType = "InvalidRequest"
			}
			message := errorMessage{
				Error:   errType,
				Message: plcerr.Error(),
			}
			sendError(w, &message)
		}
		return
	}
	docBytes, err := json.Marshal(&doc)
	_, _ = w.Write(docBytes)
}

func resolveHandle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	query := r.URL.Query()
	if !query.Has("handle") {
		http.Error(w, "a handle to resolve is required", http.StatusBadRequest)
		return
	}
	handle := query.Get("handle")
	did, err := resolver.Handle(handle)
	if err != nil {
		slog.Error("error resolving handle", "request", r, "err", err)
		http.Error(w, "error resolving handle", http.StatusInternalServerError)
		return
	}
	result := map[string]string{"did": did.String()}
	bytes, err := json.Marshal(&result)
	if err != nil {
		return
	}
	_, _ = w.Write(bytes)
}

func main() {
	http.HandleFunc("/xrpc/com.atproto.identity.resolveDid/", resolveDID)
	http.HandleFunc("/xrpc/com.atproto.identity.resolveHandle/", resolveHandle)
}
