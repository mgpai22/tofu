package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var hiddenParts = []string{"bug", "feature", "rename", "search", "newfile"}

var rubricChecks = []string{"visible tests pass", "gofmt clean", "go vet clean", "old name gone", "no stray files"}

var expectedFiles = []string{"page.go", "inventory.go", "cart.go", "discount.go", "report.go", "cart_test.go", "export.go", "ANSWER.txt"}

var harnessDirs = []string{".tofu/", ".omp/", ".claude/", ".codex/", ".agents/"}

type score struct {
	Hidden       map[string]bool `json:"hidden"`
	Rubric       map[string]bool `json:"rubric"`
	FilesChanged int             `json:"files_changed"`
	Added        int             `json:"lines_added"`
	Deleted      int             `json:"lines_deleted"`
	Stray        []string        `json:"stray,omitempty"`
	HarnessFiles []string        `json:"harness_files,omitempty"`
	Failures     []string        `json:"failures,omitempty"`
}

func (s score) hiddenPassed() int {
	return countTrue(s.Hidden)
}

func (s score) rubricPassed() int {
	return countTrue(s.Rubric)
}

func countTrue(checks map[string]bool) int {
	n := 0
	for _, passed := range checks {
		if passed {
			n++
		}
	}
	return n
}

func (s score) text() string {
	var b strings.Builder
	for _, part := range hiddenParts {
		fmt.Fprintf(&b, "  hidden %-8s %s\n", part, verdict(s.Hidden[part]))
	}
	for _, check := range rubricChecks {
		fmt.Fprintf(&b, "  rubric %-18s %s\n", check, verdict(s.Rubric[check]))
	}
	fmt.Fprintf(&b, "  hidden %d of %d, rubric %d of %d\n", s.hiddenPassed(), len(hiddenParts), s.rubricPassed(), len(rubricChecks))
	for _, failure := range s.Failures {
		fmt.Fprintf(&b, "  %s\n", failure)
	}
	return b.String()
}

func verdict(passed bool) string {
	if passed {
		return "pass"
	}
	return "FAIL"
}

func scoreRepo(repo string, env []string) score {
	s := score{Hidden: map[string]bool{}, Rubric: map[string]bool{}}
	run := func(name string, args ...string) (string, bool) {
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = repo, env
		out, err := cmd.CombinedOutput()
		return string(out), err == nil
	}
	_, _ = run("git", "add", "-A")
	numstat, _ := run("git", "diff", "--cached", "--numstat")
	for _, line := range strings.Split(strings.TrimSpace(numstat), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		s.FilesChanged++
		if isHarness(fields[2]) {
			s.HarnessFiles = append(s.HarnessFiles, fields[2])
			continue
		}
		added, _ := strconv.Atoi(fields[0])
		deleted, _ := strconv.Atoi(fields[1])
		s.Added, s.Deleted = s.Added+added, s.Deleted+deleted
		if !slices.Contains(expectedFiles, fields[2]) && !strings.HasSuffix(fields[2], "_test.go") {
			s.Stray = append(s.Stray, fields[2])
		}
	}
	out, passed := run("go", "test", "-count=1", "./...")
	s.Rubric["visible tests pass"] = passed
	keepFailure(&s, "visible tests", out, passed)
	unformatted, _ := run("gofmt", "-l", ".")
	s.Rubric["gofmt clean"] = strings.TrimSpace(unformatted) == ""
	out, passed = run("go", "vet", "./...")
	s.Rubric["go vet clean"] = passed
	keepFailure(&s, "go vet", out, passed)
	s.Rubric["old name gone"] = !anyGoFileContains(repo, "CalcTotal")
	s.Rubric["no stray files"] = len(s.Stray) == 0
	for _, part := range hiddenParts {
		body, err := os.ReadFile(filepath.Join(here(), "testdata", "hidden", part+".go.txt"))
		if err != nil {
			s.Failures = append(s.Failures, err.Error())
			continue
		}
		placed := filepath.Join(repo, "hidden_"+part+"_test.go")
		if err := os.WriteFile(placed, body, 0o644); err != nil {
			s.Failures = append(s.Failures, err.Error())
			continue
		}
		out, passed := run("go", "test", "-count=1", "-run", "TestHidden", ".")
		s.Hidden[part] = passed
		keepFailure(&s, "hidden "+part, out, passed)
		_ = os.Remove(placed)
	}
	return s
}

func keepFailure(s *score, label, out string, passed bool) {
	if passed {
		return
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "FAIL") && !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "ok") && !strings.HasPrefix(line, "#") {
			s.Failures = append(s.Failures, label+": "+line)
			return
		}
	}
	s.Failures = append(s.Failures, label+": failed")
}

func isHarness(path string) bool {
	return slices.ContainsFunc(harnessDirs, func(dir string) bool { return strings.HasPrefix(path, dir) })
}

func anyGoFileContains(repo, needle string) bool {
	matches, _ := filepath.Glob(filepath.Join(repo, "*.go"))
	return slices.ContainsFunc(matches, func(path string) bool {
		body, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(body), needle)
	})
}
