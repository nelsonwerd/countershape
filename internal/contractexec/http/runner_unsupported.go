//go:build !darwin

package http

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/store"
)

func executeHTTP(context.Context, contractexec.OfficialTarget) (store.ContractExecutionRecord, error) {
	return store.ContractExecutionRecord{}, refuse(CodeUnsupportedProfile, "the standalone HTTP subject runner requires Darwin", nil)
}

func resumeHTTPClassification(context.Context, contractexec.OfficialTarget) (store.ContractExecutionRecord, error) {
	return store.ContractExecutionRecord{}, refuse(CodeUnsupportedProfile, "HTTP classification recovery requires the Darwin target profile", nil)
}
