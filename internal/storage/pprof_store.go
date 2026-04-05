package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

const maxRetainedFiles = 3

// PprofFileStore saves raw pprof binary files with a retention policy.
type PprofFileStore struct {
	dataDir string
}

// NewPprofFileStore creates a store rooted at dataDir.
func NewPprofFileStore(dataDir string) *PprofFileStore {
	return &PprofFileStore{dataDir: dataDir}
}

// Save writes the raw pprof bytes to disk and returns the file path.
// Files are organised as: {dataDir}/pprof/{endpointID}/{profileType}/YYYYMMDD_HHMMSS.pb.gz
func (s *PprofFileStore) Save(endpointID string, profileType domain.ProfileType, data []byte) (string, error) {
	dir := s.profileDir(endpointID, profileType)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create pprof dir: %w", err)
	}

	filename := time.Now().UTC().Format("20060102_150405") + ".pb.gz"
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write pprof file: %w", err)
	}
	return path, nil
}

// ApplyRetentionPolicy deletes the oldest files beyond maxRetainedFiles.
func (s *PprofFileStore) ApplyRetentionPolicy(endpointID string, profileType domain.ProfileType) error {
	dir := s.profileDir(endpointID, profileType)

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read pprof dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".gz" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	// Sort ascending (oldest first)
	sort.Strings(files)

	for len(files) > maxRetainedFiles {
		if err := os.Remove(files[0]); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove old pprof file: %w", err)
		}
		files = files[1:]
	}
	return nil
}

func (s *PprofFileStore) profileDir(endpointID string, profileType domain.ProfileType) string {
	return filepath.Join(s.dataDir, "pprof", endpointID, string(profileType))
}
