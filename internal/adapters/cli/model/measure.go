package model

// CLIStimulusMeasure is reducer-neutral, well-founded measure data. It counts
// only reducible per-stimulus inputs; fixed executable/base argv are excluded.
// U5 may define neighbor ordering over this tuple without changing U3 identity.
type CLIStimulusMeasure struct {
	argvItems             int
	argvBytes             int
	stdinPresenceUnits    int
	stdinBytes            int
	environmentEntries    int
	environmentValueBytes int
	fixtureFiles          int
	fixturePathBytes      int
	fixtureContentBytes   int
}

type measureIdentity struct {
	ArgvItems             int `json:"argv_items"`
	ArgvBytes             int `json:"argv_bytes"`
	StdinPresenceUnits    int `json:"stdin_presence_units"`
	StdinBytes            int `json:"stdin_bytes"`
	EnvironmentEntries    int `json:"environment_entries"`
	EnvironmentValueBytes int `json:"environment_value_bytes"`
	FixtureFiles          int `json:"fixture_files"`
	FixturePathBytes      int `json:"fixture_path_bytes"`
	FixtureContentBytes   int `json:"fixture_content_bytes"`
}

func measureOf(argv []string, stdin CLIStdin, environment []CLIEnvironmentBinding, fixtures []CLIFixtureFile) CLIStimulusMeasure {
	result := CLIStimulusMeasure{argvItems: len(argv), stdinBytes: len(stdin.bytes), environmentEntries: len(environment), fixtureFiles: len(fixtures)}
	if stdin.presence == PresencePresent {
		result.stdinPresenceUnits = 1
	}
	for _, argument := range argv {
		result.argvBytes += len(argument)
	}
	for _, binding := range environment {
		result.environmentValueBytes += len(binding.value)
	}
	for _, file := range fixtures {
		result.fixturePathBytes += len(file.path)
		result.fixtureContentBytes += len(file.contents)
	}
	return result
}

func (m CLIStimulusMeasure) identity() measureIdentity {
	return measureIdentity{
		ArgvItems: m.argvItems, ArgvBytes: m.argvBytes,
		StdinPresenceUnits: m.stdinPresenceUnits, StdinBytes: m.stdinBytes,
		EnvironmentEntries: m.environmentEntries, EnvironmentValueBytes: m.environmentValueBytes,
		FixtureFiles: m.fixtureFiles, FixturePathBytes: m.fixturePathBytes, FixtureContentBytes: m.fixtureContentBytes,
	}
}

func (m CLIStimulusMeasure) ArgvItems() int             { return m.argvItems }
func (m CLIStimulusMeasure) ArgvBytes() int             { return m.argvBytes }
func (m CLIStimulusMeasure) StdinPresenceUnits() int    { return m.stdinPresenceUnits }
func (m CLIStimulusMeasure) StdinBytes() int            { return m.stdinBytes }
func (m CLIStimulusMeasure) EnvironmentEntries() int    { return m.environmentEntries }
func (m CLIStimulusMeasure) EnvironmentValueBytes() int { return m.environmentValueBytes }
func (m CLIStimulusMeasure) FixtureFiles() int          { return m.fixtureFiles }
func (m CLIStimulusMeasure) FixturePathBytes() int      { return m.fixturePathBytes }
func (m CLIStimulusMeasure) FixtureContentBytes() int   { return m.fixtureContentBytes }

// WellFoundedTuple returns an immutable coarse-to-fine tuple. It is data, not
// a reducer or proof that any proposed neighbor is semantically valid.
func (m CLIStimulusMeasure) WellFoundedTuple() []uint64 {
	return []uint64{
		uint64(m.fixtureFiles + m.environmentEntries + m.argvItems + m.stdinPresenceUnits),
		uint64(m.fixtureContentBytes + m.environmentValueBytes + m.argvBytes + m.stdinBytes),
		uint64(m.fixturePathBytes), uint64(m.fixtureFiles), uint64(m.environmentEntries),
		uint64(m.argvItems), uint64(m.stdinPresenceUnits),
	}
}
