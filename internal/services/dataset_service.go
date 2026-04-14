package services

import (
	"context"
	"fmt"
	"os"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/index"
	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
)

type DatasetService struct {
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

func (s *DatasetService) EnsureFresh(ctx context.Context) (bool, []string, error) {
	decision, err := index.ShouldRebuild(s.paths, s.paths.MetadataPath, s.paths.DBPath, s.cfg.Dataset.SchemaVersion)
	if err != nil {
		return false, nil, err
	}
	if !decision.Needed {
		return false, nil, nil
	}
	if !s.cfg.Dataset.AutoRebuildOnCSVChange {
		return false, decision.Reasons, nil
	}

	if _, err := s.Rebuild(ctx); err != nil {
		return false, decision.Reasons, err
	}
	return true, decision.Reasons, nil
}

func (s *DatasetService) NeedsRebuild() (bool, []string, error) {
	decision, err := index.ShouldRebuild(s.paths, s.paths.MetadataPath, s.paths.DBPath, s.cfg.Dataset.SchemaVersion)
	if err != nil {
		return false, nil, err
	}
	return decision.Needed, decision.Reasons, nil
}

func (s *DatasetService) AutoRebuildEnabled() bool {
	return s.cfg.Dataset.AutoRebuildOnCSVChange
}

func (s *DatasetService) Rebuild(ctx context.Context) (index.Metadata, error) {
	return s.RebuildWithProgress(ctx, nil)
}

func (s *DatasetService) RebuildWithProgress(ctx context.Context, onProgress func(DatasetRebuildProgress)) (index.Metadata, error) {
	meta, err := index.BuildWithProgress(ctx, s.paths, s.cfg.Dataset.SchemaVersion, func(p index.Progress) {
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
	if err := index.SaveMetadata(s.paths.MetadataPath, meta); err != nil {
		return index.Metadata{}, err
	}
	return meta, nil
}

func (s *DatasetService) OpenRepository() (*sqlite.Repository, error) {
	return sqlite.Open(s.paths.DBPath)
}

func (s *DatasetService) Status(ctx context.Context) (DatasetStatus, error) {
	decision, err := index.ShouldRebuild(s.paths, s.paths.MetadataPath, s.paths.DBPath, s.cfg.Dataset.SchemaVersion)
	if err != nil {
		return DatasetStatus{}, err
	}
	meta, err := index.LoadMetadata(s.paths.MetadataPath)
	if err != nil {
		return DatasetStatus{}, err
	}

	status := DatasetStatus{
		Paths:            s.paths,
		MetadataPath:     s.paths.MetadataPath,
		SchemaVersion:    s.cfg.Dataset.SchemaVersion,
		RebuildNeeded:    decision.Needed,
		RebuildReasons:   decision.Reasons,
		Metadata:         meta,
		TagCSVPresent:    fileExists(s.paths.TagCSV),
		CharacterPresent: fileExists(s.paths.CharacterCSV),
		HasDatabase:      fileExists(s.paths.DBPath),
	}

	if status.HasDatabase {
		repo, err := sqlite.Open(s.paths.DBPath)
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
