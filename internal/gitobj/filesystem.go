package gitobj

type filesystemIdentity struct {
	Device string `json:"device"`
	Inode  string `json:"inode"`
}

func (i filesystemIdentity) Valid() bool { return i.Device != "" && i.Inode != "" }

func (i filesystemIdentity) Equal(other filesystemIdentity) bool {
	return i.Valid() && i == other
}

func (i filesystemIdentity) display() string { return "dev=" + i.Device + ",ino=" + i.Inode }
