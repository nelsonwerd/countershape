package http

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// Valid seals projection rejection evidence against mutation after creation.
func (r ProjectionRejection) Valid() bool {
	if r.Code == "" || r.Operation == "" || r.Channel == "" || r.Field == "" || !r.digest.Valid() || !r.observationDigest.Valid() || !r.adapterDefinitionDigest.Valid() || !r.definitionDigest.Valid() || len(r.canonicalBytes) == 0 || len(r.transcript) == 0 {
		return false
	}
	type identity struct {
		SchemaVersion, Kind, ObservationDigest, AdapterDefinitionDigest, DefinitionDigest, Code, Operation, Channel, Field, Detail string
		Transcript                                                                                                                 any
	}
	value := identity{
		domain.SchemaVersion,
		"HTTPProjectionRejection",
		r.observationDigest.String(),
		r.adapterDefinitionDigest.String(),
		r.definitionDigest.String(),
		r.Code,
		r.Operation,
		string(r.Channel),
		string(r.Field),
		r.Detail,
		traceIdentities(r.transcript),
	}
	digest, canonicalBytes, err := digestTyped("HTTPProjectionRejection", value)
	return err == nil && digest == r.digest && bytes.Equal(canonicalBytes, r.canonicalBytes)
}
