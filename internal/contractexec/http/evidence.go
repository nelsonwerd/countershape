package http

import contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"

// This alias confines the target model family name to the manifest-admitted
// surface while the implementation uses the shorter package-local name.
type targetModel = contractmodel.ContractExecutionTarget
