package boardy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func ticketIn(t *testing.T, m Managed, status Status, owns ...string) Ticket {
	t.Helper()
	ticket := ticketWith(string(status), owns...)
	ticket.Status, ticket.Log = status, "- evidence"
	created, err := m.Local.Create("DEMO", ticket, "test")
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func overlaps(t *testing.T, m Managed) []string {
	t.Helper()
	boards, err := m.Store.Boards()
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, finding := range m.Store.Lint(boards) {
		if strings.Contains(finding.What, "owns") {
			found = append(found, finding.What)
		}
	}
	return found
}

func TestOwnsHeldOnlyWhileDoing(t *testing.T) {
	m := managedBoard(t, "DEMO")
	first := ticketIn(t, m, Review, "x/**")
	second := ticketIn(t, m, Review, "x/**")
	if found := overlaps(t, m); len(found) != 0 {
		t.Fatalf("two review tickets owning x/** lint %q, want clean", found)
	}
	if _, err := m.Move(first.ID, Doing, "", manager); err != nil {
		t.Fatalf("the first into doing: %v", err)
	}
	_, err := m.Move(second.ID, Doing, "", manager)
	refused(t, err, first.ID)

	holder := ticketIn(t, m, Doing, "y/**")
	if _, err := m.Widen(second.ID, []string{"y/a.go"}, manager); err != nil {
		t.Errorf("widening a review ticket over a doing ticket's owns: %v", err)
	}
	_, err = m.Widen(first.ID, []string{"y/a.go"}, manager)
	refused(t, err, holder.ID)

	doing := ticketWith("doing", "x/a.go")
	doing.Status = Doing
	_, err = m.Create("DEMO", doing, manager)
	refused(t, err, first.ID)
	backlog := ticketWith("backlog", "x/a.go")
	if _, err := m.Create("DEMO", backlog, manager); err != nil {
		t.Errorf("a backlog ticket owning a doing ticket's path: %v", err)
	}
	if found := overlaps(t, m); len(found) != 0 {
		t.Errorf("one doing ticket per path lints %q, want clean", found)
	}
}

func TestOwnsAbsoluteOutsideTheProject(t *testing.T) {
	m := managedBoard(t, "DEMO")
	whole := ticketIn(t, m, Doing, "F:/localhost/boji-convert/**")
	ticketIn(t, m, Doing, "f/localhost/boji-convert/**", "/home/someone/repo/**")
	if found := overlaps(t, m); len(found) != 0 {
		t.Fatalf("absolute owns lint %q, want clean", found)
	}
	inside := ticketIn(t, m, Review, `F:\localhost\boji-convert\main.go`)
	_, err := m.Move(inside.ID, Doing, "", manager)
	refused(t, err, whole.ID)

	folded := ticketIn(t, m, Review, "f:/LOCALHOST/Boji-Convert/write.go")
	_, err = m.Move(folded.ID, Doing, "", manager)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		refused(t, err, whole.ID)
	} else if err != nil {
		t.Errorf("a case-different path on %s: %v", runtime.GOOS, err)
	}
}

func TestOwnsHoldingStatusesFromBoardFile(t *testing.T) {
	m := managedBoard(t, "DEMO")
	path := filepath.Join(m.Store.BoardDir("DEMO"), BoardFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `holds = ["doing"]`) {
		t.Errorf("a new board.toml does not name its holding statuses:\n%s", raw)
	}
	var older []string
	for line := range strings.Lines(string(raw)) {
		if !strings.HasPrefix(line, "holds") {
			older = append(older, line)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(older, "")), 0o644); err != nil {
		t.Fatal(err)
	}
	if board, err := m.Store.Board("DEMO"); err != nil || !board.Holding(Doing) || board.Holding(Review) {
		t.Fatalf("a board.toml with no holds line: holds %v, err %v; want doing only", board.Holds, err)
	}
	ticketIn(t, m, Review, "z/**")
	ticketIn(t, m, Review, "z/**")
	if found := overlaps(t, m); len(found) != 0 {
		t.Fatalf("review tickets on a doing-only board lint %q", found)
	}
	if err := os.WriteFile(path, append([]byte(strings.Join(older, "")), `holds = ["doing", "review"]`+"\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if found := overlaps(t, m); len(found) != 1 {
		t.Errorf("two review tickets on a board holding review lint %q, want one overlap", found)
	}
	if err := os.WriteFile(path, append([]byte(strings.Join(older, "")), `holds = ["doing", "reviewed"]`+"\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Store.Board("DEMO"); err == nil || !strings.Contains(err.Error(), "reviewed") {
		t.Errorf("an unknown holding status: %v, want an error naming it", err)
	}
}
