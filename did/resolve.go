package did

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Resolves handles to DIDs and DIDs to documents.
type Resolver struct {
	didCache    []*documentResolution // Previous DID-to-document resolutions.
	handleCache []*handleResolution   // Previous handle-to-DID resolutions.
	PLCLedger   string                // Where to fetch PLC documents from.
	sinceClean  int                   // How many requests have been made since this was last cleaned?
	untilClean  int                   // How many requests trigger a clean?
	TTL         time.Duration         // How long until a resolution is stale?
}

func DefaultResolver() *Resolver {
	return &Resolver{
		PLCLedger:  "https://plc.directory",
		TTL:        time.Hour * 12,
		untilClean: 1000,
	}
}

type documentResolution struct {
	did      *DID
	doc      *Doc
	resolved time.Time
}

type handleResolution struct {
	did      *DID
	handle   string
	resolved time.Time
}

func (resolver *Resolver) clean() {
	if resolver.sinceClean < resolver.untilClean {
		return
	}
	expirary := time.Now().Add(resolver.TTL)
	var x, y, z, l int
	var handle *handleResolution
	handleCache := resolver.handleCache
	for x, handle = range handleCache {
		if handle.resolved.Compare(expirary) >= 1 {
			l = len(handleCache)
			for y = range handleCache[x:] {
				z = y + 1
				if z == l {
					break
				}
				handleCache[y] = handleCache[z]
			}
			handleCache = handleCache[:l-1]
		}
	}
	resolver.handleCache = handleCache
	var doc *documentResolution
	didCache := resolver.didCache
	for x, doc = range didCache {
		if doc.resolved.Compare(expirary) >= 1 {
			l = len(didCache)
			for y = range didCache {
				z = y + 1
				if z == l {
					break
				}
				didCache[y] = didCache[z]
			}
			didCache = didCache[:l-1]
		}
	}
	resolver.didCache = didCache
	resolver.sinceClean = 0
}

func (resolver *Resolver) Doc(did *DID) (*Doc, error) {
	for _, cached := range resolver.didCache {
		if cached.did == did {
			return cached.doc, nil
		}
	}
	defer resolver.clean()
	switch did.Method {
	case "plc":
		target := fmt.Sprintf("%s/%s", resolver.PLCLedger, did.String())
		response, err := http.Get(target)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != 200 {
			obj := map[string]string{}
			err = json.Unmarshal(bytes, &obj)
			if err != nil {
				return nil, err // TODO: This is fragile.
			}
			return nil, &PLCError{
				DID:        did,
				StatusCode: response.StatusCode,
				Message:    obj["message"],
			}
		}
		result := &Doc{}
		err = json.Unmarshal(bytes, result)
		if err != nil {
			return nil, err
		}
		return result, nil
		// TODO: Implement web method.
	default:
		return nil, fmt.Errorf("invalid method <%s> for did <%s>", did.Method, did.String())
	}
}

func (resolver *Resolver) Handle(handle string) (*DID, error) {
	for _, cached := range resolver.handleCache {
		if cached.handle == handle {
			return cached.did, nil
		}
	}
	defer resolver.clean()
	if !strings.HasPrefix(handle, "_atproto.") {
		handle = fmt.Sprintf("_atproto.%s", handle)
	}
	records, err := net.LookupTXT(handle)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		rawDID, found := strings.CutPrefix(record, "did=")
		if found {
			did, err := Parse(rawDID)
			if err != nil {
				return nil, err
			}
			resolution := &handleResolution{
				did:      did,
				handle:   handle,
				resolved: time.Now(),
			}
			resolver.handleCache = append(resolver.handleCache, resolution)
			return did, nil
		}
	}
	response, err := http.Get(fmt.Sprintf("https://%s/.well-known/at-did", strings.TrimPrefix(handle, "_atproto.")))
	if err != nil {
		return nil, err
	}
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	did, err := Parse(string(responseBytes))
	if err != nil {
		return nil, err
	}
	resolution := &handleResolution{
		did:      did,
		handle:   handle,
		resolved: time.Now(),
	}
	resolver.handleCache = append(resolver.handleCache, resolution)
	return did, nil
}

type PLCError struct {
	DID        *DID   `json:"did"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

func (err PLCError) Error() string {
	return fmt.Sprintf("error resolving did <%s> with code <%d>: %s", err.DID.String(), err.StatusCode, err.Message)
}
