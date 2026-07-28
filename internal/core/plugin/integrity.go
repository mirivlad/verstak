package plugin

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChecksumsFile is what a packaged plugin carries to say what left the build.
const ChecksumsFile = "checksums.txt"

// VerifyPackage compares a plugin directory against the checksums its package
// was built with.
//
// A package travels through a build, an install script, a .deb and whatever the
// user's own copy did to it, and this project has already shipped a build whose
// installed plugins were a version behind their source. A file that arrives
// changed, missing or extra is now named at discovery instead of showing up as
// a plugin behaving oddly.
//
// It is a checksum, not a signature: it detects damage and staleness, not
// somebody who meant it. Anyone able to change a plugin's files can change this
// file too. Saying otherwise would be worse than not checking.
//
// A directory with no checksums file is not an error -- plugins under
// development have none -- and reports no findings.
func VerifyPackage(pluginDir string) ([]string, error) {
	expected, err := readChecksums(filepath.Join(pluginDir, ChecksumsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	actual, err := hashTree(pluginDir)
	if err != nil {
		return nil, err
	}

	var findings []string
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		got, present := actual[name]
		if !present {
			findings = append(findings, fmt.Sprintf("%s is missing", name))
			continue
		}
		if got != expected[name] {
			findings = append(findings, fmt.Sprintf("%s does not match the package", name))
		}
	}

	extra := make([]string, 0)
	for name := range actual {
		if _, listed := expected[name]; !listed {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		findings = append(findings, fmt.Sprintf("%s is not part of the package", name))
	}
	return findings, nil
}

func readChecksums(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sums := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		// "<hash>  <path>", the shape sha256sum writes.
		parts := strings.SplitN(text, "  ", 2)
		if len(parts) != 2 || len(parts[0]) != 64 || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("%s line %d is not a checksum", ChecksumsFile, line)
		}
		sums[filepath.ToSlash(strings.TrimSpace(parts[1]))] = strings.ToLower(parts[0])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sums, nil
}

func hashTree(root string) (map[string]string, error) {
	sums := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == ChecksumsFile {
			return nil
		}
		sum, err := hashFile(path)
		if err != nil {
			return err
		}
		sums[relative] = sum
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sums, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
