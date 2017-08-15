package options

// Build holds options of compose build.
type Build struct {
	NoCache     bool
	ForceRemove bool
	Pull        bool
}

type Pull struct {
	Cached bool
}

// Create holds options of compose create.
type Options struct {
	NoRecreate    bool
	ForceRecreate bool
	NoBuild       bool
	ForceBuild    bool
	Services      []string

	Rollback     bool
	Pull         bool
}

// ImageType defines the type of image (local, all)
type ImageType string

// Valid indicates whether the image type is valid.
func (i ImageType) Valid() bool {
	switch string(i) {
	case "", "local", "all":
		return true
	default:
		return false
	}
}
