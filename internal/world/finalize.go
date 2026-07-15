package world

import (
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

func finalizePhysical(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	materialization gitobj.MaterializationReceipt,
	tool resolvedTool,
	logicalArgv []string,
	environment []string,
	physical physicalProcessResult,
) (Result, error) {
	return finalizePhysicalWithLineage(
		allocated, attempt, materialization, tool, logicalArgv, environment, physical, processCLILineage{},
	)
}

func finalizePhysicalWithLineage(
	allocated allocatedAttempt,
	attempt domain.Attempt,
	materialization gitobj.MaterializationReceipt,
	tool resolvedTool,
	logicalArgv []string,
	environment []string,
	physical physicalProcessResult,
	cli processCLILineage,
) (Result, error) {
	var err error
	if physical.primary != "" {
		attempt, err = attempt.Fail(physical.primary)
	} else {
		attempt, err = attempt.Advance(domain.AttemptTearingDown)
	}
	if err != nil {
		return Result{}, err
	}
	if physical.teardownError {
		attempt, err = attempt.RecordTeardownControl(domain.ControlTeardownError)
		if err != nil {
			return Result{}, err
		}
	}
	if physical.orphanRisk {
		attempt, err = attempt.RecordTeardownControl(domain.ControlOrphanRisk)
		if err != nil {
			return Result{}, err
		}
	}
	attempt, err = attempt.Advance(domain.AttemptFinalized)
	if err != nil {
		return Result{}, err
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		return Result{}, err
	}
	receipt, err := buildProcessReceipt(receiptInput{
		allocated: allocated, materialization: materialization, attempt: attempt,
		tool: tool, logicalArgv: append([]string(nil), logicalArgv...),
		environment: environment, physical: physical, primary: physical.primary, cli: cli,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		world: allocated.world, finalized: finalized, states: attempt.StateHistory(), roots: allocated.roots,
		materialization: cloneMaterializationReceipt(materialization), process: receipt,
	}, nil
}
