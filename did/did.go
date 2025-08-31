package did

import (
	"errors"
	"fmt"
	"strings"
)

type DID struct {
	Method string
	Value  string
}

func (did *DID) String() string {
	return fmt.Sprintf("did:%s:%s", did.Method, did.Value)
}

func Parse(from string) (*DID, error) {
	split := strings.Split(strings.TrimPrefix(from, "at://"), ":")
	if len(split) < 3 {
		return nil, errors.New("possible DID not valid (not enough segments)")
	}
	if split[1] != "plc" && split[1] != "web" {
		return nil, errors.New("possible DID uses wrong method")
	}
	return &DID{
		Method: split[1],
		Value:  split[2],
	}, nil
}
