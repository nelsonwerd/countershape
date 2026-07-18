package model

import (
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestClosedRunAlgebraExhaustiveCrossProduct(t *testing.T) {
	bundle := testBundle(t)
	allowed := bundle.Predicate().AllowedTuples()[0]
	projected, err := NewProjectedObservation(
		testRef(t, EvidenceCapturedObservation, 'd'),
		testRef(t, EvidenceProjectionResult, 'e'),
		allowed,
	)
	if err != nil {
		t.Fatal(err)
	}
	unprojected, err := NewCapturedUnprojected(testRef(t, EvidenceCapturedObservation, 'd'))
	if err != nil {
		t.Fatal(err)
	}
	observations := []CaptureObservation{NewNoCapture(), unprojected, projected}
	primaries := []domain.ControlReason{
		"",
		domain.ControlMaterializationError,
		domain.ControlStartError,
		domain.ControlReadinessError,
		domain.ControlProbeTransportError,
		domain.ControlTimeout,
		domain.ControlCancelled,
		domain.ControlOutputLimit,
		domain.ControlProjectionRejected,
	}
	scopes := make([]StandaloneScope, 243)
	for index := range scopes {
		scopes[index] = testScope(t, scopeStates(index))
	}
	privateManifest := testPrivateManifest(t)
	accepted := 0
	rejected := 0
	for _, primary := range primaries {
		for teardownBit := 0; teardownBit < 2; teardownBit++ {
			for orphanBit := 0; orphanBit < 2; orphanBit++ {
				teardown := teardownBit == 1
				orphan := orphanBit == 1
				controlled := primary != "" || teardown || orphan
				for spawnIndex := 0; spawnIndex < 2; spawnIndex++ {
					started := spawnIndex == 1
					var spawn SpawnObservation
					if started {
						spawn, _ = NewChildPIDObservation(4242)
					} else {
						spawn, _ = NewStartErrorObservation("OS_START_ERROR")
					}
					var process ProcessClosure
					if controlled {
						process, err = NewControlledProcessClosure(primary, teardown, orphan, testProcessEvidence(t, started))
					} else {
						process, err = NewCleanProcessClosure(testProcessEvidence(t, started))
					}
					if err != nil {
						t.Fatalf("process construction failed: primary=%q teardown=%t orphan=%t started=%t: %v", primary, teardown, orphan, started, err)
					}
					for _, observation := range observations {
						for _, scope := range scopes {
							witness, witnessErr := NewClosedRunWitness(spawn, process, observation, scope, privateManifest)
							wantValid := true
							if !started {
								wantValid = primary == domain.ControlStartError && observation.state == CaptureNone && scope.state != ScopeComplete
							} else if primary == domain.ControlStartError {
								wantValid = false
							}
							if process.state == ProcessClean && observation.state != CaptureProjected {
								wantValid = false
							}
							if observation.state != CaptureProjected && primary == "" {
								wantValid = false
							}
							if (witnessErr == nil) != wantValid {
								t.Fatalf(
									"witness validity mismatch: primary=%q teardown=%t orphan=%t started=%t capture=%q scope=%q err=%v",
									primary, teardown, orphan, started, observation.state, scope.state, witnessErr,
								)
							}
							if witnessErr != nil {
								rejected++
								continue
							}
							accepted++
							wantDisposition := deriveDisposition(process.state, scope.state)
							if !witness.Valid() || witness.Disposition() != wantDisposition {
								t.Fatalf("derived disposition = %q, want %q", witness.Disposition(), wantDisposition)
							}
						}
					}
				}
			}
		}
	}
	if accepted == 0 || rejected == 0 || accepted+rejected != len(primaries)*2*2*2*len(observations)*len(scopes) {
		t.Fatalf("cross-product accounting = accepted %d rejected %d", accepted, rejected)
	}
}

func scopeStates(value int) [5]ScopeCheckState {
	states := [5]ScopeCheckState{}
	options := [...]ScopeCheckState{ScopeCheckClean, ScopeCheckMissing, ScopeCheckViolated}
	for index := range states {
		states[index] = options[value%3]
		value /= 3
	}
	return states
}

func TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	allowed := bundle.Predicate().AllowedTuples()[0]
	different := differentBytesTuple(t, allowed)
	tests := []struct {
		name        string
		primary     domain.ControlReason
		scopeStates [5]ScopeCheckState
		tuple       emitmodel.ExactTuple
		want        ExecutionResult
	}{
		{
			name:        "eligible member",
			scopeStates: [5]ScopeCheckState{ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean},
			tuple:       allowed,
			want:        ResultConforms,
		},
		{
			name:        "eligible nonmember",
			scopeStates: [5]ScopeCheckState{ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean},
			tuple:       different,
			want:        ResultContradicts,
		},
		{
			name:        "process controlled",
			primary:     domain.ControlTimeout,
			scopeStates: [5]ScopeCheckState{ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean},
			tuple:       allowed,
			want:        ResultIneligible,
		},
		{
			name:        "scope partial",
			scopeStates: [5]ScopeCheckState{ScopeCheckMissing, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean},
			tuple:       allowed,
			want:        ResultIneligible,
		},
		{
			name:        "scope violation outranks missing",
			primary:     domain.ControlTimeout,
			scopeStates: [5]ScopeCheckState{ScopeCheckMissing, ScopeCheckViolated, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean},
			tuple:       different,
			want:        ResultIneligible,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spawn, _ := NewChildPIDObservation(4242)
			var process ProcessClosure
			var err error
			if test.primary == "" {
				process, err = NewCleanProcessClosure(testProcessEvidence(t, true))
			} else {
				process, err = NewControlledProcessClosure(test.primary, false, false, testProcessEvidence(t, true))
			}
			if err != nil {
				t.Fatal(err)
			}
			observation, err := NewProjectedObservation(
				testRef(t, EvidenceCapturedObservation, 'd'), testRef(t, EvidenceProjectionResult, 'e'), test.tuple,
			)
			if err != nil {
				t.Fatal(err)
			}
			witness, err := NewClosedRunWitness(spawn, process, observation, testScope(t, test.scopeStates), testPrivateManifest(t))
			if err != nil {
				t.Fatal(err)
			}
			run, err := NewFinalizedContractRun(target, testDigest(byte('1'+index)), witness)
			if err != nil {
				t.Fatal(err)
			}
			execution, err := DeriveContractExecution(bundle, target, run)
			if err != nil || execution.Result() != test.want {
				t.Fatalf("result = %q, want %q, err=%v", execution.Result(), test.want, err)
			}
		})
	}
}

func differentBytesTuple(t *testing.T, original emitmodel.ExactTuple) emitmodel.ExactTuple {
	t.Helper()
	fields := original.Fields()
	if len(fields) != 1 || fields[0].Value().Tag() != portablevalue.TagBytes {
		t.Fatal("fixture no longer has one bytes field")
	}
	portable, err := portablevalue.Bytes([]byte("different\n"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := emitmodel.NewExactValue(portable)
	if err != nil {
		t.Fatal(err)
	}
	fields[0], err = emitmodel.NewExactField(fields[0].FieldID(), value)
	if err != nil {
		t.Fatal(err)
	}
	tuple, err := emitmodel.NewExactTuple(fields)
	if err != nil {
		t.Fatal(err)
	}
	return tuple
}
