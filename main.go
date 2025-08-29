package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lunar-parklife/at-identity/did"
)

var resolver *did.Resolver = &did.Resolver{
	PLCLedger: "https://plc.directory",
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
	http.HandleFunc("/xrpc/com.atproto.identity.resolveHandle/", resolveHandle)
}
