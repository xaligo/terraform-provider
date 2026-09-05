package local

type Paths struct {
	SourceDir  string
	OutputPath string
}

type WriteOptions struct {
	Overwrite              bool
	ExpectedPreviousDigest string
}

type Inspection struct {
	Digest string
	Exists bool
}

type DeleteResult struct {
	Deleted bool
	Warning string
}
