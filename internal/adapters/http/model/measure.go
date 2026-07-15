package model

// HTTPStimulusMeasure is a well-founded description of only the reducible
// parts of one HTTP stimulus. U4 provides no reducer; U5 may consume this
// immutable tuple without changing stimulus identity.
type HTTPStimulusMeasure struct {
	queryEntries    int
	queryBytes      int
	headerEntries   int
	headerBytes     int
	bodyPresence    int
	bodyBytes       int
	seedFiles       int
	seedPathBytes   int
	seedContentByte int
}

type measureIdentity struct {
	QueryEntries    int `json:"query_entries"`
	QueryBytes      int `json:"query_bytes"`
	HeaderEntries   int `json:"header_entries"`
	HeaderBytes     int `json:"header_bytes"`
	BodyPresence    int `json:"body_presence_units"`
	BodyBytes       int `json:"body_bytes"`
	SeedFiles       int `json:"seed_files"`
	SeedPathBytes   int `json:"seed_path_bytes"`
	SeedContentByte int `json:"seed_content_bytes"`
}

func measureOf(query []HTTPQueryEntry, headers []HTTPRequestHeader, body HTTPBody, seeds []HTTPSeedFile) HTTPStimulusMeasure {
	result := HTTPStimulusMeasure{
		queryEntries: len(query), headerEntries: len(headers), bodyBytes: len(body.bytes), seedFiles: len(seeds),
	}
	if body.presence == PresencePresent {
		result.bodyPresence = 1
	}
	for _, entry := range query {
		result.queryBytes += len(entry.name) + len(entry.value)
	}
	for _, header := range headers {
		result.headerBytes += len(header.name) + len(header.value)
	}
	for _, seed := range seeds {
		result.seedPathBytes += len(seed.path)
		result.seedContentByte += len(seed.contents)
	}
	return result
}

func (m HTTPStimulusMeasure) identity() measureIdentity {
	return measureIdentity{
		QueryEntries: m.queryEntries, QueryBytes: m.queryBytes,
		HeaderEntries: m.headerEntries, HeaderBytes: m.headerBytes,
		BodyPresence: m.bodyPresence, BodyBytes: m.bodyBytes,
		SeedFiles: m.seedFiles, SeedPathBytes: m.seedPathBytes, SeedContentByte: m.seedContentByte,
	}
}

func (m HTTPStimulusMeasure) QueryEntries() int      { return m.queryEntries }
func (m HTTPStimulusMeasure) QueryBytes() int        { return m.queryBytes }
func (m HTTPStimulusMeasure) HeaderEntries() int     { return m.headerEntries }
func (m HTTPStimulusMeasure) HeaderBytes() int       { return m.headerBytes }
func (m HTTPStimulusMeasure) BodyPresenceUnits() int { return m.bodyPresence }
func (m HTTPStimulusMeasure) BodyBytes() int         { return m.bodyBytes }
func (m HTTPStimulusMeasure) SeedFiles() int         { return m.seedFiles }
func (m HTTPStimulusMeasure) SeedPathBytes() int     { return m.seedPathBytes }
func (m HTTPStimulusMeasure) SeedContentBytes() int  { return m.seedContentByte }

func (m HTTPStimulusMeasure) WellFoundedTuple() []uint64 {
	return []uint64{
		uint64(m.queryEntries + m.headerEntries + m.bodyPresence + m.seedFiles),
		uint64(m.queryBytes + m.headerBytes + m.bodyBytes + m.seedContentByte),
		uint64(m.seedPathBytes), uint64(m.seedFiles), uint64(m.headerEntries),
		uint64(m.queryEntries), uint64(m.bodyPresence),
	}
}
