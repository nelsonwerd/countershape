//go:build !darwin

package runner

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/store"
)

func executeCLI(context.Context, contractexec.OfficialTarget) (store.ContractExecutionRecord, error) {
	return store.ContractExecutionRecord{}, refuse(CodeUnsupportedProfile, "the physical contract runner requires Darwin", nil)
}

func resumeCLIClassification(context.Context, contractexec.OfficialTarget) (store.ContractExecutionRecord, error) {
	return store.ContractExecutionRecord{}, refuse(CodeUnsupportedProfile, "contract-run recovery requires the Darwin target profile", nil)
}
