package projectiontranslate

import (
	"testing"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestNewTupleEnforcesProfileTagAndAbsencePolicy(t *testing.T) {
	resolved := resolveHTTPFixed(t)
	valid, err := resolved.Translate(emittedHTTPProjection(t, 200, nil))
	if err != nil {
		t.Fatal(err)
	}
	validFields := valid.Fields()

	presentString, err := portablevalue.String("200")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		value portablevalue.Value
	}{
		{name: "wrong present tag", value: presentString},
		{name: "missing disallowed", value: portablevalue.Missing()},
		{name: "null disallowed", value: portablevalue.Null()},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := append([]Field(nil), validFields...)
			fields[0] = Field{id: string(counterhttp.HTTPFieldStatus), value: test.value}
			if _, err := newTuple(resolved.Profile(), fields); !IsCode(err, CodeInvalidTag) {
				t.Fatalf("newTuple() error = %v, want %s", err, CodeInvalidTag)
			}
		})
	}

	if _, err := newTuple(resolved.Profile(), validFields); err != nil {
		t.Fatalf("newTuple(valid profile-authorized missing) error = %v", err)
	}
}
