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

type Resolver struct {
	didCache    []*documentResolution
	handleCache []*handleResolution
	PLCLedger   string
	TTL         time.Duration
}

func DefaultResolver() *Resolver {
	return &Resolver{
		PLCLedger: "https://plc.directory",
		TTL:       time.Hour * 12,
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

func (resolver *Resolver) Clean() {
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
}

func (resolver *Resolver) Doc(did *DID) (*Doc, error) {
	for _, cached := range resolver.didCache {
		if cached.did == did {
			return cached.doc, nil
		}
	}
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
			return nil, fmt.Errorf("error resolving DID <%s> with status code <%d>: %s", did.String(), response.StatusCode, obj["message"])
		}
		result := &Doc{}
		err = json.Unmarshal(bytes, result)
		if err != nil {
			return nil, err
		}
		return result, nil
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
			did, err := parse(rawDID)
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
	did, err := parse(string(responseBytes))
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
