package services

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/index"
	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
)

type DatasetService struct {
	mu    sync.RWMutex
	cfg   config.Config
	paths config.DatasetPaths
}

type DatasetRebuildProgress struct {
	Stage   string
	Current int
	Total   int
	Detail  string
}

type DatasetStatus struct {
	Paths            config.DatasetPaths
	MetadataPath     string
	SchemaVersion    int
	RebuildNeeded    bool
	RebuildReasons   []string
	Metadata         *index.Metadata
	Counts           sqlite.DatasetCounts
	HasDatabase      bool
	TagCSVPresent    bool
	CharacterPresent bool
}

func NewDatasetService(cfg config.Config) *DatasetService {
	return &DatasetService{
		cfg:   cfg,
		paths: cfg.DatasetPaths(),
	}
}

func (s *DatasetService) UpdateConfig(cfg config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.paths = cfg.DatasetPaths()
	s.mu.Unlock()
}

func (s *DatasetService) EnsureFresh(ctx context.Context) (bool, []string, error) {
	s.mu.RLock()
	paths := s.paths
	schemaVersion := s.cfg.Dataset.SchemaVersion
	autoRebuild := s.cfg.Dataset.AutoRebuildOnCSVChange
	s.mu.RUnlock()

	decision, err := index.ShouldRebuild(paths, paths.MetadataPath, paths.DBPath, schemaVersion)
	if err != nil {
		return false, nil, err
	}
	if !decision.Needed {
		return false, nil, nil
	}
	if !autoRebuild {
		return false, decision.Reasons, nil
	}

	if _, err := s.Rebuild(ctx); err != nil {
		return false, decision.Reasons, err
	}
	return true, decision.Reasons, nil
}

func (s *DatasetService) NeedsRebuild() (bool, []string, error) {
	s.mu.RLock()
	paths := s.paths
	schemaVersion := s.cfg.Dataset.SchemaVersion
	s.mu.RUnlock()

	decision, err := index.ShouldRebuild(paths, paths.MetadataPath, paths.DBPath, schemaVersion)
	if err != nil {
		return false, nil, err
	}
	return decision.Needed, decision.Reasons, nil
}

func (s *DatasetService) AutoRebuildEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Dataset.AutoRebuildOnCSVChange
}

func (s *DatasetService) Rebuild(ctx context.Context) (index.Metadata, error) {
	return s.RebuildWithProgress(ctx, nil)
}

func (s *DatasetService) RebuildWithProgress(ctx context.Context, onProgress func(DatasetRebuildProgress)) (index.Metadata, error) {
	s.mu.RLock()
	paths := s.paths
	schemaVersion := s.cfg.Dataset.SchemaVersion
	s.mu.RUnlock()

	meta, err := index.BuildWithProgress(ctx, paths, schemaVersion, func(p index.Progress) {
		if onProgress == nil {
			return
		}
		onProgress(DatasetRebuildProgress{
			Stage:   p.Stage,
			Current: p.Current,
			Total:   p.Total,
			Detail:  p.Detail,
		})
	})
	if err != nil {
		return index.Metadata{}, err
	}
	if err := index.SaveMetadata(paths.MetadataPath, meta); err != nil {
		return index.Metadata{}, err
	}
	return meta, nil
}

func (s *DatasetService) OpenRepository() (*sqlite.Repository, error) {
	s.mu.RLock()
	dbPath := s.paths.DBPath
	s.mu.RUnlock()
	return sqlite.Open(dbPath)
}

func (s *DatasetService) Status(ctx context.Context) (DatasetStatus, error) {
	s.mu.RLock()
	paths := s.paths
	schemaVersion := s.cfg.Dataset.SchemaVersion
	s.mu.RUnlock()

	decision, err := index.ShouldRebuild(paths, paths.MetadataPath, paths.DBPath, schemaVersion)
	if err != nil {
		return DatasetStatus{}, err
	}
	meta, err := index.LoadMetadata(paths.MetadataPath)
	if err != nil {
		return DatasetStatus{}, err
	}

	status := DatasetStatus{
		Paths:            paths,
		MetadataPath:     paths.MetadataPath,
		SchemaVersion:    schemaVersion,
		RebuildNeeded:    decision.Needed,
		RebuildReasons:   decision.Reasons,
		Metadata:         meta,
		TagCSVPresent:    fileExists(paths.TagCSV),
		CharacterPresent: fileExists(paths.CharacterCSV),
		HasDatabase:      fileExists(paths.DBPath),
	}

	if status.HasDatabase {
		repo, err := sqlite.Open(paths.DBPath)
		if err != nil {
			return DatasetStatus{}, fmt.Errorf("open sqlite: %w", err)
		}
		defer repo.Close()
		counts, err := repo.Counts(ctx)
		if err != nil {
			return DatasetStatus{}, err
		}
		status.Counts = counts
	}

	return status, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
