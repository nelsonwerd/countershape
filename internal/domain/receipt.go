package domain

// ReceiptReference is deliberately opaque. In particular GradeVerbatim is not
// parsed, normalized, ranked, or translated into a Countershape state.
type ReceiptReference struct {
	authority     string
	gradeVerbatim string
	commitOID     string
	commandDigest Digest
}

func NewDidrunReceipt(gradeVerbatim, commitOID string, commandDigest Digest) (ReceiptReference, error) {
	if gradeVerbatim == "" || commitOID == "" || !commandDigest.Valid() {
		return ReceiptReference{}, refuse(ErrReceiptAuthority, "incomplete didrun reference")
	}
	return ReceiptReference{
		authority:     "didrun",
		gradeVerbatim: gradeVerbatim,
		commitOID:     commitOID,
		commandDigest: commandDigest,
	}, nil
}

func Unreceipted() string { return "UNRECEIPTED" }

func (r ReceiptReference) Authority() string { return r.authority }

func (r ReceiptReference) GradeVerbatim() string { return r.gradeVerbatim }

func (r ReceiptReference) CommitOID() string { return r.commitOID }

func (r ReceiptReference) CommandDigest() Digest { return r.commandDigest }

type ReceiptWire struct {
	Authority     string `json:"authority"`
	GradeVerbatim string `json:"grade_verbatim"`
	CommitOID     string `json:"commit_oid"`
	CommandDigest string `json:"command_digest"`
}

func (r ReceiptReference) Wire() ReceiptWire {
	return ReceiptWire{
		Authority:     r.authority,
		GradeVerbatim: r.gradeVerbatim,
		CommitOID:     r.commitOID,
		CommandDigest: r.commandDigest.String(),
	}
}

func ReceiptFromWire(wire ReceiptWire) (ReceiptReference, error) {
	if wire.Authority != "didrun" {
		return ReceiptReference{}, refuse(ErrReceiptAuthority, wire.Authority)
	}
	digest, err := ParseDigest(wire.CommandDigest)
	if err != nil {
		return ReceiptReference{}, err
	}
	return NewDidrunReceipt(wire.GradeVerbatim, wire.CommitOID, digest)
}
