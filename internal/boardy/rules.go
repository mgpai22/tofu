package boardy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"tofu/internal/rule"
	"tofu/internal/sys"
)

const RulesDirName = "rules"

const (
	RuleFired       EventKind = "rule"
	Waiting         EventKind = "waiting"
	Requested       EventKind = "requested"
	Accepted        EventKind = "accepted"
	RequestDropped  EventKind = "request-dropped"
	Handed          EventKind = "handed"
	Widened         EventKind = "widened"
	ManagersChanged EventKind = "managers"
	Assigned        EventKind = "assigned"
)

type Fired struct {
	rule.Fire
	Checker string `json:"checker"`
}

func (f Fired) Text() string {
	details := make([]string, len(f.Findings))
	for i, finding := range f.Findings {
		details[i] = finding.Detail
	}
	return strings.Join(details, "; ")
}

func (s Store) BoardRules(key string) ([]rule.Rule, error) {
	dir := filepath.Join(s.BoardDir(key), RulesDirName)
	if found, err := sys.Exists(dir); !found || err != nil {
		return nil, err
	}
	rules, err := rule.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		if r.Trigger.On() == rule.EventNone || rule.Builtins()[r.Checker].Subject != rule.SubjectBoardEvent {
			return nil, fmt.Errorf("%s: a board rule declares on: and a board checker, %s or %s", r.File, rule.CheckerCommand, rule.CheckerPersonApproves)
		}
		if to := r.Trigger.To(); to != "" {
			if _, err := ParseStatus(to); err != nil {
				return nil, fmt.Errorf("%s: %w", r.File, err)
			}
		}
	}
	return rules, nil
}

func (f Fired) waits() bool { return f.Blocked && f.Checker == rule.CheckerPersonApproves }

func (s Store) fire(key string, e rule.BoardEvent) ([]Fired, error) {
	rules, err := s.BoardRules(key)
	if err != nil {
		return nil, err
	}
	var fired []Fired
	var refusals []string
	var waits error
	for _, r := range rules {
		if !r.Trigger.FiresOn(e) {
			continue
		}
		fire, err := rule.Run(r, rule.Builtins(), e, e.Ticket, time.Now())
		if err != nil {
			return nil, err
		}
		if len(fire.Findings) == 0 {
			continue
		}
		f := Fired{Fire: fire, Checker: r.Checker}
		fired = append(fired, f)
		switch {
		case f.waits():
			waits = WaitingError{Ticket: e.Ticket, Rule: f.RuleID, Detail: f.Text()}
		case f.Blocked:
			refusals = append(refusals, f.RuleID+": "+f.Text())
		}
	}
	if refusals != nil {
		return fired, RefusedError{Ticket: e.Ticket, Why: strings.Join(refusals, "; ")}
	}
	return fired, waits
}

func (s Store) recordFired(key string, e rule.BoardEvent, fired []Fired) error {
	var err error
	for _, f := range fired {
		kind := RuleFired
		if f.waits() {
			kind = Waiting
		}
		err = errors.Join(err, s.Record(key, Event{Ticket: e.Ticket, Kind: kind, Actor: e.Actor, To: Status(e.To),
			Text: fmt.Sprintf("%s (%s, %s): %s", f.RuleID, f.Mode, e.Event, f.Text())}))
	}
	return err
}

func (m Managed) gate(key string, e rule.BoardEvent) error {
	fired, err := m.Store.fire(key, e)
	return errors.Join(m.Store.recordFired(key, e, fired), err)
}
