package server

import (
	_ "embed"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/choice"
)

const seedChoicepointDigest = "sha256:493bc5cfc6b8c993945339f03e5bca4eadaccad510fe51a123deadb9c9fcda1c"

// seedChoicepointBytes is an exact synthetic U7C portable Choicepoint. It is
// checked in so the U8 browser can be exercised without rerunning a physical
// study and without depending on a private evidence root at runtime.
//
//go:embed seed/cli-choicepoint.json
var seedChoicepointBytes []byte

func loadSeedChoicepoint() (choice.ChoicepointRecord, error) {
	record, err := choice.ParseChoicepointRecord(seedChoicepointBytes)
	if err != nil || !record.Valid() {
		return choice.ChoicepointRecord{}, fmt.Errorf("STUDIO_SEED_CHOICEPOINT_REFUSED")
	}
	if record.Digest().String() != seedChoicepointDigest {
		return choice.ChoicepointRecord{}, fmt.Errorf("STUDIO_SEED_CHOICEPOINT_IDENTITY_REFUSED")
	}
	return record, nil
}
