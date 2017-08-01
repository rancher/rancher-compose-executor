package lookup

import (
	"fmt"
	"strings"
)

type MemoryResourceLookup struct {
	Content map[string][]byte
}

func (m *MemoryResourceLookup) Lookup(file, relativeTo string) ([]byte, string, error) {
	file = strings.TrimPrefix(file, "/")
	relativeTo = strings.TrimSuffix(relativeTo, "/")
	finalFile := relativeTo + "/" + file

	if relativeTo == "." {
		finalFile = file
	}

	if content, ok := m.Content[finalFile]; ok {
		return content, finalFile, nil
	}

	return nil, finalFile, fmt.Errorf("not found: %s", finalFile)
}
