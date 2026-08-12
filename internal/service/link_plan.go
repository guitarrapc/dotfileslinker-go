package service

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guitarrapc/dotfileslinker-go/internal/util"
)

type linkPlan []linkPlanEntry

type linkPlanEntry struct {
	source       string
	target       string
	ensureParent bool
}

func (p linkPlan) validate(repoRoot string) error {
	seenTargets := make(map[string]string, len(p))
	for i, entry := range p {
		if util.PathEquals(entry.source, entry.target) {
			return fmt.Errorf("source and destination resolve to the same path: %q", entry.source)
		}
		if util.PathsOverlap(repoRoot, entry.target) {
			return fmt.Errorf("destination %q overlaps dotfiles repository %q", entry.target, repoRoot)
		}

		key := filepath.Clean(entry.target)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if previousSource, exists := seenTargets[key]; exists {
			return fmt.Errorf("multiple sources map to destination %q: %q and %q", entry.target, previousSource, entry.source)
		}
		for previousIndex := 0; previousIndex < i; previousIndex++ {
			previousTarget := p[previousIndex].target
			if util.PathsOverlap(previousTarget, entry.target) {
				return fmt.Errorf("destinations %q and %q overlap", previousTarget, entry.target)
			}
		}
		seenTargets[key] = entry.source
	}
	return nil
}

// Kept as a small compatibility helper for focused plan-validation tests.
func validateLinkPlan(repoRoot string, plan []linkPlanEntry) error {
	return linkPlan(plan).validate(repoRoot)
}
