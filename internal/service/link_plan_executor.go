package service

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
	"github.com/guitarrapc/dotfileslinker-go/internal/util"
)

type linkDisposition uint8

const (
	linkDispositionCreate linkDisposition = iota
	linkDispositionSkip
	linkDispositionReplace
)

type linkOperation struct {
	entry             linkPlanEntry
	disposition       linkDisposition
	sourceIsDirectory bool
	validationError   error
}

type appliedLinkOperation struct {
	operation  linkOperation
	backupPath string
}

type linkPlanExecutor struct {
	fs     infrastructure.FileSystem
	logger Logger
}

func newLinkPlanExecutor(fs infrastructure.FileSystem, logger Logger) *linkPlanExecutor {
	return &linkPlanExecutor{fs: fs, logger: logger}
}

func (e *linkPlanExecutor) execute(plan linkPlan, overwrite, dryRun bool) error {
	if dryRun {
		var failures []error
		for _, entry := range plan {
			operation := e.inspect(entry, overwrite)
			e.logDryRunOperation(operation)
			if operation.validationError != nil {
				failures = append(failures, operation.validationError)
			}
		}
		return errors.Join(failures...)
	}

	applied := make([]appliedLinkOperation, 0, len(plan))
	var operationErrors []error
	for _, entry := range plan {
		operation := e.inspect(entry, overwrite)
		if operation.validationError != nil {
			e.logger.Error(fmt.Sprintf("Cannot link %s to %s: %s", entry.source, entry.target, operation.validationError))
			operationErrors = append(operationErrors, operation.validationError)
			continue
		}
		if operation.disposition != linkDispositionSkip && entry.ensureParent {
			parent := filepath.Dir(entry.target)
			e.logger.Verbosef("Ensuring directory exists: %s", parent)
			if err := e.fs.EnsureDirectory(parent); err != nil {
				operationErr := fmt.Errorf("failed to create directory %s for %s: %w", parent, entry.target, err)
				e.logger.Error(operationErr.Error())
				operationErrors = append(operationErrors, operationErr)
				continue
			}
		}
		e.logger.Verbosef("Linking %s to %s", entry.source, entry.target)
		appliedOperation, err := e.applyLink(operation)
		if err != nil {
			operationErrors = append(operationErrors, err)
			continue
		}
		if appliedOperation != nil {
			applied = append(applied, *appliedOperation)
		}
	}

	var cleanupErrors []error
	for _, operation := range applied {
		if operation.backupPath == "" {
			continue
		}
		if err := e.fs.DeleteBackup(operation.backupPath, operation.operation.entry.target); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf(
				"links are committed, but failed to remove replacement backup %s; cleanup may be incomplete: %w",
				operation.backupPath, err))
		}
	}
	for _, operation := range applied {
		e.logger.Success(fmt.Sprintf("Creating symbolic link: %s -> %s", operation.operation.entry.target, operation.operation.entry.source))
	}
	return errors.Join(errors.Join(operationErrors...), errors.Join(cleanupErrors...))
}

func (e *linkPlanExecutor) inspect(entry linkPlanEntry, overwrite bool) linkOperation {
	operation := linkOperation{entry: entry, disposition: linkDispositionCreate, sourceIsDirectory: e.fs.DirectoryExists(entry.source)}
	exists, err := e.fs.PathExists(entry.target)
	if err != nil {
		operation.validationError = fmt.Errorf("failed to inspect target %s: %w", entry.target, err)
		return operation
	}
	if !exists {
		return operation
	}
	currentLinkTarget := e.fs.GetLinkTarget(entry.target)
	switch {
	case util.LinkTargetEquals(entry.target, currentLinkTarget, entry.source):
		operation.disposition = linkDispositionSkip
	case overwrite:
		operation.disposition = linkDispositionReplace
	default:
		operation.validationError = fmt.Errorf("'%s' already exists; use --force to overwrite", entry.target)
	}
	return operation
}

func (e *linkPlanExecutor) logDryRunOperation(operation linkOperation) {
	entry := operation.entry
	e.logger.Verbosef("Linking %s to %s", entry.source, entry.target)
	if operation.validationError != nil {
		e.logger.Error(fmt.Sprintf("[DRY-RUN] Cannot link %s to %s: %s", entry.source, entry.target, operation.validationError))
		return
	}
	if operation.disposition == linkDispositionSkip {
		e.logger.Success(fmt.Sprintf("[DRY-RUN] Would skip already linked: %s -> %s", entry.target, entry.source))
		return
	}
	if operation.disposition == linkDispositionReplace {
		e.logger.Verbosef("[DRY-RUN] Would replace existing target: %s", entry.target)
	}
	kind := "file"
	if operation.sourceIsDirectory {
		kind = "directory"
	}
	e.logger.Success(fmt.Sprintf("[DRY-RUN] Would create %s symlink: %s -> %s", kind, entry.target, entry.source))
}

func (e *linkPlanExecutor) linkFile(source, target string, overwrite, dryRun bool) error {
	return e.execute(linkPlan{{source: source, target: target}}, overwrite, dryRun)
}

func (e *linkPlanExecutor) applyLink(operation linkOperation) (*appliedLinkOperation, error) {
	entry := operation.entry
	if operation.disposition == linkDispositionSkip {
		e.logger.Success(fmt.Sprintf("Skipping already linked: %s -> %s", entry.target, entry.source))
		return nil, nil
	}
	backupPath := ""
	if operation.disposition == linkDispositionReplace {
		var err error
		backupPath, err = e.moveTargetAside(entry.target)
		if err != nil {
			return nil, err
		}
	}
	var linkErr error
	if operation.sourceIsDirectory {
		linkErr = e.fs.CreateDirectorySymlink(entry.target, entry.source)
	} else {
		linkErr = e.fs.CreateFileSymlink(entry.target, entry.source)
	}
	if linkErr != nil {
		if backupPath != "" {
			linkErr = errors.Join(linkErr, e.restoreMovedTarget(entry.target, backupPath))
		}
		e.logger.Error(fmt.Sprintf("Failed to create symlink from %s to %s: %s", entry.source, entry.target, linkErr))
		return nil, fmt.Errorf("failed to create symlink from %s to %s: %w", entry.source, entry.target, linkErr)
	}
	return &appliedLinkOperation{operation: operation, backupPath: backupPath}, nil
}

func (e *linkPlanExecutor) moveTargetAside(target string) (string, error) {
	backupPath := target + ".dotfileslinker-backup"
	for suffix := 1; ; suffix++ {
		exists, err := e.fs.PathExists(backupPath)
		if err != nil {
			return "", fmt.Errorf("failed to inspect backup path for %s: %w", target, err)
		}
		if !exists {
			break
		}
		backupPath = fmt.Sprintf("%s.dotfileslinker-backup.%d", target, suffix)
	}
	e.logger.Verbosef("Temporarily moving existing target: %s -> %s", target, backupPath)
	if err := e.fs.Move(target, backupPath); err != nil {
		return "", fmt.Errorf("failed to move existing target aside: %w", err)
	}
	return backupPath, nil
}

func (e *linkPlanExecutor) restoreMovedTarget(target, backupPath string) error {
	var rollbackErrors []error
	exists, err := e.fs.PathExists(target)
	if err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to inspect replacement during rollback: %w", err))
	} else if exists {
		if err := e.fs.Delete(target); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to remove replacement during rollback: %w", err))
		}
	}
	if len(rollbackErrors) == 0 {
		if err := e.fs.Move(backupPath, target); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to restore original target: %w", err))
		}
	}
	return errors.Join(rollbackErrors...)
}
