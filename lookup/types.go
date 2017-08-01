package lookup

// ResourceLookup defines methods to provides file loading.
type ResourceLookup interface {
	Lookup(file, relativeTo string) ([]byte, string, error)
}
