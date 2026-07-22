package importservice

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

func sortEntries(entries []sourceEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}

func fingerprintEntries(kind, sourceMeta string, entries []sourceEntry) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s\x00%s\x00", kind, sourceMeta)
	for _, entry := range entries {
		fmt.Fprintf(hash, "%s\x00%s\x00%d\x00%d\x00%d\x00%d\x00",
			entry.Path, entry.Kind, entry.Size, entry.modifiedNanos, entry.compressedSize, entry.checksum)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func checkAndAppend(entries *[]sourceEntry, seen map[string]string, entry sourceEntry, total *int64, limits limits) error {
	key := collisionKey(entry.Path)
	if _, exists := seen[key]; exists {
		return sourceError("source-path-collision", "normalized source paths collide")
	}
	if len(*entries) >= limits.maxEntries {
		return sourceError("source-limit-exceeded", "more than %d entries", limits.maxEntries)
	}
	if entry.Kind == "file" {
		if entry.Size > limits.maxEntryBytes {
			return sourceError("source-limit-exceeded", "entry exceeds %d bytes", limits.maxEntryBytes)
		}
		if entry.Size < 0 || *total > limits.maxTotalBytes-entry.Size {
			return sourceError("source-limit-exceeded", "content exceeds %d bytes", limits.maxTotalBytes)
		}
		*total += entry.Size
	}
	seen[key] = entry.Path
	*entries = append(*entries, entry)
	return nil
}
