package did

import (
	"encoding/json"
)

// An identity document on the network.
type Doc struct {
	AlsoKnownAs        []string              // All the handles of this identity.
	ID                 *DID                  // The DID of this identity.
	Service            []*Service            // What services this identity uses.
	VerificationMethod []*VerificationMethod // The verification methods.
}

type docMarshal struct {
	AlsoKnownAs        []string              `json:"alsoKnownAs"`
	Id                 string                `json:"id"`
	Service            []*Service            `json:"service"`
	VerificationMethod []*VerificationMethod `json:"verificationMethod"`
}

func (doc *Doc) MarshalJSON() ([]byte, error) {
	model := docMarshal{
		AlsoKnownAs:        doc.AlsoKnownAs,
		Id:                 doc.ID.String(),
		Service:            doc.Service,
		VerificationMethod: doc.VerificationMethod,
	}
	return json.Marshal(&model)
}

func (doc *Doc) UnmarshalJSON(from []byte) error {
	var model docMarshal
	err := json.Unmarshal(from, &model)
	if err != nil {
		return err
	}
	id, err := Parse(model.Id)
	if err != nil {
		return err
	}
	doc.AlsoKnownAs = model.AlsoKnownAs
	doc.ID = id
	doc.Service = model.Service
	doc.VerificationMethod = model.VerificationMethod
	return nil
}

func (doc *Doc) Verify(handle string) bool {
	for _, alias := range doc.AlsoKnownAs {
		if alias == handle {
			return true
		}
	}
	return false
}

type Service struct {
	ID              string `json:"id"`
	ServiceEndpoint string `json:"serviceEndpoint"`
	Type            string `json:"type"`
}

type VerificationMethod struct {
	Controller         string `json:"controller"`
	ID                 string `json:"id"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
	Type               string `json:"type"`
}
