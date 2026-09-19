package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/guitarrapc/dotfileslinker-go/internal/infrastructure"
)

type directoryLinkFixture struct {
	repo, home, source, target string
	service                    *FileLinkerService
	logger                     *MockLogger
}

func setupDirectoryLink(t *testing.T, patterns *string, ignore string) directoryLinkFixture {
	t.Helper()
	root := t.TempDir()
	f := directoryLinkFixture{repo: filepath.Join(root, "repo"), home: filepath.Join(root, "home"), logger: NewMockLogger()}
	f.source = filepath.Join(f.repo, "HOME", ".agents", "skills", "demo")
	f.target = filepath.Join(f.home, ".agents", "skills", "demo")
	if err := os.MkdirAll(filepath.Join(f.source, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	writeDirectoryTestFile(t, filepath.Join(f.source, "SKILL.md"), "skill")
	writeDirectoryTestFile(t, filepath.Join(f.source, "references", "guide.md"), "guide")
	writeDirectoryTestFile(t, filepath.Join(f.repo, "HOME", ".agents", "settings.json"), "{}")
	writeDirectoryTestFile(t, filepath.Join(f.repo, "dotfiles_ignore"), ignore)
	if patterns != nil {
		writeDirectoryTestFile(t, filepath.Join(f.repo, "dotfiles_link_dirs"), *patterns)
	}
	f.service = NewFileLinkerService(infrastructure.NewDefaultFileSystem(), f.logger)
	return f
}
func writeDirectoryTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func assertDirectoryTestLink(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.Readlink(path)
	if want == "" {
		if err == nil {
			t.Fatalf("unexpected symlink %s -> %s", path, got)
		}
		return
	}
	if err != nil || got != want {
		t.Fatalf("Readlink(%s) = %q, %v; want %q", path, got, err, want)
	}
}

func TestSelectedDirectoriesBecomeLinks(t *testing.T) {
	for _, tc := range []struct {
		name, patterns, ignore          string
		missing, directoryLink, skipped bool
	}{
		{name: "missing", missing: true},
		{name: "empty"},
		{name: "exact", patterns: "HOME/.agents/skills/demo", directoryLink: true},
		{name: "wildcard", patterns: "# Skills\r\n\r\nHOME/.agents/skills/*/\r\n", directoryLink: true},
		{name: "negated", patterns: "HOME/.agents/skills/*\n!HOME/.agents/skills/demo"},
		{name: "unmatched", patterns: "HOME/.agents/skills/other"},
		{name: "file", patterns: "HOME/.agents/skills/demo/SKILL.md"},
		{name: "ignored", patterns: "HOME/.agents/skills/*", ignore: "HOME/.agents/skills/demo/", skipped: true},
		{name: "ignored-parent", patterns: "HOME/.agents/skills/*", ignore: "HOME/.agents/skills/", skipped: true},
		{name: "whole-subtree", patterns: "HOME/.agents/skills/*", ignore: "*.md", directoryLink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patterns := &tc.patterns
			if tc.missing {
				patterns = nil
			}
			f := setupDirectoryLink(t, patterns, tc.ignore)
			if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
				t.Fatal(err)
			}
			_, err := os.Stat(f.target)
			if tc.skipped {
				if !os.IsNotExist(err) {
					t.Fatalf("expected absent directory: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				want := ""
				if tc.directoryLink {
					want = f.source
				}
				assertDirectoryTestLink(t, f.target, want)
				want = ""
				if !tc.directoryLink {
					want = filepath.Join(f.source, "SKILL.md")
				}
				assertDirectoryTestLink(t, filepath.Join(f.target, "SKILL.md"), want)
				data, err := os.ReadFile(filepath.Join(f.target, "references", "guide.md"))
				if err != nil || string(data) != "guide" {
					t.Fatalf("reference: %q %v", data, err)
				}
			}
			assertDirectoryTestLink(t, filepath.Join(f.home, ".agents", "settings.json"), filepath.Join(f.repo, "HOME", ".agents", "settings.json"))
			if _, err := os.Stat(filepath.Join(f.home, "dotfiles_link_dirs")); !os.IsNotExist(err) {
				t.Fatalf("config should not be linked: %v", err)
			}
		})
	}
}

func TestDirectoryLinkRerunAndNewFiles(t *testing.T) {
	patterns := "HOME/.agents/skills/*"
	f := setupDirectoryLink(t, &patterns, "")
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
		t.Fatal(err)
	}
	writeDirectoryTestFile(t, filepath.Join(f.source, "new.md"), "new")
	if data, err := os.ReadFile(filepath.Join(f.target, "new.md")); err != nil || string(data) != "new" {
		t.Fatalf("new source file not visible: %q %v", data, err)
	}
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
		t.Fatal(err)
	}
	assertDirectoryTestLink(t, f.target, f.source)
	assertDirectoryTestLink(t, filepath.Join(f.target, "SKILL.md"), "")
}

func TestDirectoryLinkMigration(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, dryRun := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "no-force", true: "force"}[force], map[bool]string{false: "apply", true: "dry-run"}[dryRun]}, "-"), func(t *testing.T) {
				f := setupDirectoryLink(t, nil, "")
				if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
					t.Fatal(err)
				}
				writeDirectoryTestFile(t, filepath.Join(f.repo, "dotfiles_link_dirs"), "HOME/.agents/skills/*")
				err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", force, dryRun)
				if (err != nil) == force {
					t.Fatalf("force=%v dryRun=%v error=%v", force, dryRun, err)
				}
				want := ""
				if force && !dryRun {
					want = f.source
				}
				assertDirectoryTestLink(t, f.target, want)
				for _, path := range []string{filepath.Join(f.source, "SKILL.md"), filepath.Join(f.target, "SKILL.md")} {
					data, err := os.ReadFile(path)
					if err != nil || string(data) != "skill" {
						t.Fatalf("skill content: %q %v", data, err)
					}
				}
				if _, err := os.Lstat(f.target + ".dotfileslinker-backup"); !os.IsNotExist(err) {
					t.Fatalf("unexpected backup: %v", err)
				}
			})
		}
	}
}

func TestDirectoryLinkDryRun(t *testing.T) {
	patterns := "HOME/.agents/skills/*"
	f := setupDirectoryLink(t, &patterns, "")
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(f.logger.SuccessLogs, "\n"), "Would create directory symlink") {
		t.Fatal("directory not reported")
	}
	if _, err := os.Stat(f.home); !os.IsNotExist(err) {
		t.Fatalf("dry-run created home: %v", err)
	}
}

func TestDirectoryLinkEmptyDirectory(t *testing.T) {
	patterns := "HOME/.agents/skills/*"
	f := setupDirectoryLink(t, &patterns, "")
	empty := filepath.Join(filepath.Dir(f.source), "empty")
	if err := os.Mkdir(empty, 0755); err != nil {
		t.Fatal(err)
	}
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
		t.Fatal(err)
	}
	assertDirectoryTestLink(t, filepath.Join(filepath.Dir(f.target), "empty"), empty)
}

func TestDirectoryLinkUnreadableConfigFailsBeforeMutation(t *testing.T) {
	f := setupDirectoryLink(t, nil, "")
	if err := os.Mkdir(filepath.Join(f.repo, "dotfiles_link_dirs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err == nil {
		t.Fatal("expected configuration read failure")
	}
	if _, err := os.Stat(f.home); !os.IsNotExist(err) {
		t.Fatalf("created home before validation: %v", err)
	}
}

func TestRootDirectoryLinkDryRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ROOT is Unix-only")
	}
	patterns := "ROOT/opt/dotfileslinker-directory-test"
	f := setupDirectoryLink(t, &patterns, "")
	if err := os.MkdirAll(filepath.Join(f.repo, "ROOT", "opt", "dotfileslinker-directory-test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(f.logger.SuccessLogs, "\n"), "Would create directory symlink: /opt/dotfileslinker-directory-test") {
		t.Fatal("ROOT directory not reported")
	}
}

func TestSelectedParentStopsTraversal(t *testing.T) {
	patterns := "HOME/.agents/skills/demo\n!HOME/.agents/skills/demo/references"
	f := setupDirectoryLink(t, &patterns, "")
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
		t.Fatal(err)
	}
	assertDirectoryTestLink(t, f.target, f.source)
	assertDirectoryTestLink(t, filepath.Join(f.target, "references", "guide.md"), "")
	plan, err := f.service.builder.build(f.repo, f.home, "dotfiles_ignore")
	if err != nil || len(plan) != 2 {
		t.Fatalf("expected two link operations, got %v: %v", plan, err)
	}
}

func TestDefaultExcludedDirectoryIsNotLinked(t *testing.T) {
	patterns := "HOME/.agents/skills/*"
	f := setupDirectoryLink(t, &patterns, "")
	if err := os.Mkdir(filepath.Join(filepath.Dir(f.source), ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := f.service.LinkDotfiles(f.repo, f.home, "dotfiles_ignore", false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(f.target), ".git")); !os.IsNotExist(err) {
		t.Fatalf("linked default exclusion: %v", err)
	}
}
