package turn

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tofu/internal/konst"
	"tofu/internal/llm"
)

type ranTool struct {
	name string
	ran  *[]string
}

func (t ranTool) Name() string         { return t.name }
func (t ranTool) Definition() llm.Tool { return llm.Tool{Name: t.name} }
func (t ranTool) Run(_ context.Context, raw json.RawMessage) (Result, error) {
	*t.ran = append(*t.ran, string(raw))
	return Result{}, nil
}

func TestShellWriteToSourceIsRefusedOnlyWhenTheCommandCarriesTheSource(t *testing.T) {
	var ran []string
	shell := WithSourceBudget([]Tool{ranTool{name: "bash", ran: &ran}}, nil)[0]
	for _, c := range []struct {
		command string
		refused bool
	}{
		{"sed -i s/a/b/ src/app.ts", true},
		{"echo 'package x' > x.go", true},
		{"cat > main.go <<'EOF'\npackage main\nEOF", true},
		{"printf 'x' | tee y.go", true},
		{"echo x >> NOTES.md", false},
		{"tofu serve --schema > apps/desk/schema/tofu.host.json", false},
		{"go run ./gen > types_gen.go", false},
		{"stringer -type=Kind > kind_string.go", false},
	} {
		raw, _ := json.Marshal(map[string]string{"command": c.command})
		_, err := shell.Run(context.Background(), raw)
		var refusal SourceBudgetError
		if errors.As(err, &refusal) != c.refused {
			t.Errorf("%q: refused %v, want %v, error %v", c.command, err != nil, c.refused, err)
		}
	}
	if len(ran) != 4 {
		t.Errorf("ran %d commands, want the four that carry no source: %v", len(ran), ran)
	}
}

func TestTheLeadWritesOneSliceItselfAndSpawnsForASecond(t *testing.T) {
	var ran []string
	budgeted := WithSourceBudget([]Tool{ranTool{name: "edit", ran: &ran}, ranTool{name: "write", ran: &ran}}, nil)
	edit, write := budgeted[0], budgeted[1]
	call := func(tool Tool, path, text string) error {
		raw, _ := json.Marshal(map[string]string{"path": path, "content": text, "old_string": text, "new_string": text})
		_, err := tool.Run(context.Background(), raw)
		return err
	}
	lines := func(n int) string { return strings.Repeat("x\n", n) }
	budget := konst.OrchestratorSourceLinesPerTurn
	quarter := budget / 4
	for _, c := range []struct {
		tool    Tool
		path    string
		lines   int
		refused bool
	}{
		{edit, "store.go", quarter, false},
		{write, "./export.go", quarter, false},
		{edit, "store_test.go", quarter, false},
		{write, "ANSWER.txt", 10 * budget, false},
		{edit, "cmd/main.go", 1, true},
		{write, "notes/plan.md", 10 * budget, false},
		{edit, "page.go", budget, true},
		{edit, "page.go", 1, false},
	} {
		err := call(c.tool, c.path, lines(c.lines))
		var refusal SourceBudgetError
		if errors.As(err, &refusal) != c.refused {
			t.Errorf("%s %s with %d lines: refused %v, want %v, error %v", c.tool.Name(), c.path, c.lines, err != nil, c.refused, err)
		}
	}
	if len(ran) != 6 {
		t.Errorf("ran %d writes, want the six inside the slice and the budget", len(ran))
	}
}
