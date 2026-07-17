// Command p07b-a2-compiler-probe emits one deterministic compiler product for
// fresh-process recovery tests and checked-in examples. Its two-line stdout is
// exactly the typed bundle digest followed by the canonical body in base64.
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/compiler"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

func main() {
	bundle, err := compileFixture()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, bundle.Digest())
	fmt.Fprintln(os.Stdout, base64.StdEncoding.EncodeToString(bundle.CanonicalBytes()))
}

func compileFixture() (model.ContractBundle, error) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		return model.ContractBundle{}, err
	}
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		return model.ContractBundle{}, err
	}
	value, err := portablevalue.Bytes([]byte("ok\n"))
	if err != nil {
		return model.ContractBundle{}, err
	}
	exact, err := model.NewExactValue(value)
	if err != nil {
		return model.ContractBundle{}, err
	}
	field, err := model.NewExactField("cli.stdout.bytes", exact)
	if err != nil {
		return model.ContractBundle{}, err
	}
	tuple, err := model.NewExactTuple([]model.ExactField{field})
	if err != nil {
		return model.ContractBundle{}, err
	}
	predicate, err := model.NewPredicate(
		source.StimulusDigest(), source.Profile(), []string{"cli.stdout.bytes"}, []model.ExactTuple{tuple},
	)
	if err != nil {
		return model.ContractBundle{}, err
	}
	input, err := compilation.New(
		fixedDigest("d"), fixedDigest("c"), compilation.ActionAllowObserved,
		source, profile, predicate,
	)
	if err != nil {
		return model.ContractBundle{}, err
	}
	return compiler.Compile(input)
}

func fixedDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}
