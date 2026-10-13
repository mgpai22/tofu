package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func render() error {
	files, err := filepath.Glob(filepath.Join(benchDir, "passes", "*.json"))
	if err != nil {
		return err
	}
	var passes []pass
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		var p pass
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		passes = append(passes, p)
	}
	slices.SortFunc(passes, func(a, b pass) int { return a.Recorded.Compare(b.Recorded) })
	base := slices.IndexFunc(passes, func(p pass) bool { return p.Name == "v0" })
	if base < 0 {
		return fmt.Errorf("no v0 pass under %s", filepath.Join(benchDir, "passes"))
	}
	var page strings.Builder
	page.WriteString("# CLI passes\n\n")
	page.WriteString("Written by `go run ./bench/cli -pass <name> -note <what changed>` from the repository root, with `-modfile` to build an earlier dependency set. Each pass builds `./cmd/tofu`, then runs every command below in a home and working directory made fresh for each run, with GOMAXPROCS=2, stdin empty and HTTP(S)_PROXY pointed at a dead port so no verb reaches the network. Init is the sum of `GODEBUG=inittrace=1` over `tofu version`. Every figure is the median of the pass's runs, wall also at p90. The ratio is v0 CPU over the pass's CPU.\n\n")
	page.WriteString("## Init\n\n| pass | commit | recorded | platform | runs | init clock ms | init allocations MB | note |\n|---|---|---|---|---|---|---|---|\n")
	var methods []string
	for _, p := range passes {
		hash, dirty, _ := strings.Cut(p.Commit, "-")
		fmt.Fprintf(&page, "| %s | %s %s | %s | %s | %d | %.1f | %.2f | %s |\n", p.Name, hash[:min(8, len(hash))], dirty, p.Recorded.Format(time.DateTime), p.Platform, p.Runs, p.InitMs, p.InitMB, p.Note)
		methods = append(methods, p.Method)
	}
	slices.Sort(methods)
	page.WriteString("\n")
	for _, method := range slices.Compact(methods) {
		fmt.Fprintf(&page, "%s.\n", method)
	}
	baseline := map[string]row{}
	for _, r := range passes[base].Rows {
		baseline[r.Command] = r
	}
	for _, p := range passes[base+1:] {
		fmt.Fprintf(&page, "\n## %s against v0\n\n| command | v0 CPU ms | %s CPU ms | ratio | v0 wall ms | %s wall ms | %s wall p90 ms | v0 RSS MB | %s RSS MB |\n|---|---|---|---|---|---|---|---|---|\n", p.Name, p.Name, p.Name, p.Name, p.Name)
		for _, r := range p.Rows {
			v0 := baseline[r.Command]
			fmt.Fprintf(&page, "| `%s` | %.1f | %.1f | %.2fx | %.1f | %.1f | %.1f | %.1f | %.1f |\n", r.Command, v0.CPUMs, r.CPUMs, v0.CPUMs/r.CPUMs, v0.WallMs, r.WallMs, r.WallP90Ms, v0.RSSMB, r.RSSMB)
		}
	}
	measured, err := os.ReadFile(filepath.Join(benchDir, "measured.json"))
	if err != nil {
		return err
	}
	var measures []measure
	if err := json.Unmarshal(measured, &measures); err != nil {
		return fmt.Errorf("measured.json: %w", err)
	}
	page.WriteString("\n## Outside the verbs\n\nChanges a verb timing cannot see, each measured by the command named in its row, before and after, on the same machine. The ratio is before over after where less is better and after over before where more is, so above 1x is a gain and below it a cost.\n\n| pass | change | measure | before | after | ratio | commit | measured by |\n|---|---|---|---|---|---|---|---|\n")
	for _, m := range measures {
		gain := m.Before / m.After
		if m.MoreIsBetter {
			gain = 1 / gain
		}
		fmt.Fprintf(&page, "| %s | %s | %s | %s | %s | %.2fx | %s | `%s` |\n", m.Pass, m.Change, m.Measure,
			strconv.FormatFloat(m.Before, 'f', -1, 64), strconv.FormatFloat(m.After, 'f', -1, 64), gain, m.Commit, m.Command)
	}
	page.WriteString("\n## v0\n\n| command | CPU ms | wall ms | wall p90 ms | RSS MB |\n|---|---|---|---|---|\n")
	for _, r := range passes[base].Rows {
		fmt.Fprintf(&page, "| `%s` | %.1f | %.1f | %.1f | %.1f |\n", r.Command, r.CPUMs, r.WallMs, r.WallP90Ms, r.RSSMB)
	}
	return os.WriteFile(filepath.Join(benchDir, "passes.md"), []byte(page.String()), 0o644)
}

type measure struct {
	Pass, Change, Measure, Commit, Command string
	Before, After                          float64
	MoreIsBetter                           bool
}
