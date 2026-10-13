package tui

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"tofu/interface/tui/fixture"
	"tofu/interface/tui/subagent"
	"tofu/internal/host"
	roster "tofu/internal/subagent"
)

type steppedClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *steppedClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *steppedClock) pass(by time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(by)
}

func stuckLead(t *testing.T) (*App, *steppedClock, <-chan struct{}) {
	t.Helper()
	clock := &steppedClock{at: fixture.Opened()}
	cancelled, stuck := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(stuck) })
	playing, _ := host.New(host.Config{Now: clock.now, Play: func(ctx context.Context, _ Pick, _ string, live host.Live) {
		live.Emit(Event{Kind: EventRequesting})
		live.Emit(Event{Kind: EventText, Text: "three pieces, three sub-agents"})
		live.Emit(Event{Kind: EventSubAgent, SubAgents: runningSubAgent()})
		go func() {
			select {
			case <-ctx.Done():
				close(cancelled)
			case <-stuck:
			}
		}()
		<-stuck
	}})
	t.Cleanup(playing.Close)
	app := newTestApp(Options{Repo: testRepo, Branch: "develop", Now: clock.now, Wires: anthropicAlone, Host: playing})
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	typeAndSend(app, "hand the three pieces to sub-agents")
	pumpUntil(t, app, EventSubAgent)
	return app, clock, cancelled
}

func drawn(t *testing.T, app *App, when string) {
	t.Helper()
	t.Logf("%s, at 80x24:\n%s", when, ansi.Strip(app.View().Content))
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, quit := cmd().(tea.QuitMsg)
	return quit
}

func TestStopEscFewSecondsIntoAStopTheTurnIgnoresStopsEverything(t *testing.T) {
	app, clock, cancelled := stuckLead(t)
	esc := tea.KeyPressMsg{Code: tea.KeyEscape}

	app.Update(esc)
	clock.pass(time.Second)
	app.Update(esc)
	select {
	case <-cancelled:
		t.Fatal("an esc one second into the lead's stop stopped everything")
	default:
	}
	drawn(t, app, "esc, then esc one second later: the lead is stopping and the turn has not ended")
	clock.pass(3 * time.Second)
	app.Update(esc)
	drawn(t, app, "esc four seconds into the stop")
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("an esc four seconds into a stop the turn ignored did not stop everything")
	}
}

func TestLinkARestoredSubAgentNeverRelabelsANewSpawn(t *testing.T) {
	restored := subagent.Row{Name: "sub-1", Doing: "the first note", State: roster.Finished, Started: fixedClock()().Add(-time.Hour)}
	started := subagent.Row{Name: "sub-2", Doing: "the second note", State: roster.Working}
	app := newTestApp(Options{Repo: testRepo, Branch: "develop", Now: fixedClock(), Wires: bothWires})
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for _, event := range []Event{
		{Kind: EventSubAgent, SubAgents: []subagent.Row{restored}},
		{Kind: EventTurnStarted},
		{Kind: EventRequesting},
		{Kind: EventToolCall, ID: "call-9", Tool: "spawn", Text: "the second note", Promote: true},
		{Kind: EventSubAgent, SubAgents: []subagent.Row{restored, started}},
		{Kind: EventToolResult, ID: "call-9", Text: "sub-2 is running", SubAgents: []subagent.Row{started}},
	} {
		app.Update(event)
	}
	drawn(t, app, "a restored sub-1, then a spawn that started sub-2")
	if spawned := rowHolding(t, app, "[&sub-"); strings.Count(spawned, "[&sub-2]") != 1 || strings.Contains(spawned, "[&sub-1]") {
		t.Fatalf("the new spawn's chip reads %q, want sub-2 and never the restored sub-1", spawned)
	}
}

func TestStopCtrlCTwiceAfterAStopTheTurnIgnoresQuits(t *testing.T) {
	app, clock, cancelled := stuckLead(t)
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	clock.pass(4 * time.Second)
	if quits(app.update(ctrlC)) {
		t.Fatal("the first ctrl+c after a stop the turn ignored quit, want it to stop everything first")
	}
	drawn(t, app, "esc, then ctrl+c four seconds later")
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("the first ctrl+c four seconds into a stop the turn ignored did not stop everything")
	}
	if !quits(app.update(ctrlC)) {
		drawn(t, app, "the second ctrl+c")
		t.Fatal("the second ctrl+c with the turn still running after stop everything did not quit")
	}
}
