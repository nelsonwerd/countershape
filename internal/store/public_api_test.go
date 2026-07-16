package store_test

import (
	"reflect"
	"testing"

	"github.com/nelsonwerd/countershape/internal/store"
)

func TestRawHeadAdvanceCannotMintFreshConfirmationAuthority(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	if _, exposed := objectStore.MethodByName("AdvanceHead"); exposed {
		t.Fatal("generic raw head advancement is exported")
	}
	method, present := objectStore.MethodByName("AdvanceConfirmation")
	if !present || method.Type.NumIn() != 4 ||
		method.Type.In(3).PkgPath() != "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication" {
		t.Fatalf("confirmation transition does not require its opaque internal publication authority: %v", method.Type)
	}
}

func TestRawHeadAdvanceCannotPublishRefineRuling(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	if _, exposed := objectStore.MethodByName("AdvanceHead"); exposed {
		t.Fatal("generic raw head advancement is exported")
	}
	method, present := objectStore.MethodByName("AdvanceRuling")
	if !present || method.Type.NumIn() != 4 ||
		method.Type.In(3).PkgPath() != "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication" {
		t.Fatalf("ruling transition does not require its promotion-issued authority: %v", method.Type)
	}
}

func TestStudyHeadExportedTransitionsAreTypedAndClosed(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	expected := map[string]string{
		"CreateStudy":         "github.com/nelsonwerd/countershape/internal/domain",
		"AdvanceBaseline":     "github.com/nelsonwerd/countershape/internal/compare",
		"AdvanceDivergence":   "github.com/nelsonwerd/countershape/internal/compare",
		"AdvanceReduction":    "github.com/nelsonwerd/countershape/internal/reduce",
		"AdvanceConfirmation": "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication",
		"AdvanceChoicepoint":  "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication",
		"AdvanceRuling":       "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication",
	}
	for name, authorityPackage := range expected {
		method, present := objectStore.MethodByName(name)
		if !present || method.Type.NumIn() != 4 || method.Type.In(3).PkgPath() != authorityPackage {
			t.Fatalf("%s input authority = %v, want package %s", name, method.Type, authorityPackage)
		}
	}
	for _, forbidden := range []string{"AdvanceHead", "AdvanceResidue", "ReplaceHead", "ForceHead"} {
		if _, exposed := objectStore.MethodByName(forbidden); exposed {
			t.Fatalf("unsafe head mutation surface %s is exported", forbidden)
		}
	}
}
