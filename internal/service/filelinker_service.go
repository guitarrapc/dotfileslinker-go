package service

import (
	"fmt"
	"path/filepath"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/internal/util"
)

// FileLinkerService coordinates planning and applying dotfile links.
type FileLinkerService struct {
	logger   Logger
	builder  *linkPlanBuilder
	executor *linkPlanExecutor
}

// NewFileLinkerService creates a new instance of FileLinkerService.
func NewFileLinkerService(fs infrastructure.FileSystem, logger Logger) *FileLinkerService {
	if logger == nil {
		logger = NewNullLogger()
	}
	return &FileLinkerService{
		logger:   logger,
		builder:  newLinkPlanBuilder(fs, logger),
		executor: newLinkPlanExecutor(fs, logger),
	}
}

// LinkDotfiles links dotfiles from the specified repository to the user's home directory or system root.
func (s *FileLinkerService) LinkDotfiles(repoRoot, userHome, ignoreFileName string, overwrite, dryRun bool) error {
	absoluteRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve repository root %q: %w", repoRoot, err)
	}
	absoluteUserHome, err := filepath.Abs(userHome)
	if err != nil {
		return fmt.Errorf("failed to resolve user home %q: %w", userHome, err)
	}
	repoRoot = absoluteRepoRoot
	userHome = absoluteUserHome
	if util.IsSameOrDescendant(userHome, repoRoot) {
		return fmt.Errorf("user home %q must not be the repository root or one of its descendants", userHome)
	}

	if dryRun {
		s.logger.Info("DRY RUN MODE: No files will be actually linked")
	}
	s.logger.Infof("Starting to link dotfiles from %s to %s", repoRoot, userHome)
	s.logger.Infof("Using ignore file: %s", ignoreFileName)

	plan, err := s.builder.build(repoRoot, userHome, ignoreFileName)
	if err != nil {
		return err
	}
	if err := s.executor.execute(plan, overwrite, dryRun); err != nil {
		return err
	}

	if dryRun {
		s.logger.Info("DRY RUN COMPLETED: No files were actually linked")
	} else {
		s.logger.Info("Dotfiles linking completed")
	}
	return nil
}
