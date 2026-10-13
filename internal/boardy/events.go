package boardy

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type EventKind string

const (
	Created   EventKind = "created"
	Moved     EventKind = "moved"
	Commented EventKind = "commented"
	Spawned   EventKind = "spawned"
	Reported  EventKind = "reported"
)

func (EventKind) Enum() []string {
	return enum(Created, Moved, Commented, RuleFired, Waiting, Requested, Accepted, RequestDropped, Handed, Widened, ManagersChanged, Assigned, Asked, Answered, Spawned, Reported)
}

type Event struct {
	At      time.Time `json:"at"`
	Ticket  string    `json:"ticket"`
	Kind    EventKind `json:"kind"`
	Actor   string    `json:"actor,omitempty"`
	From    Status    `json:"from,omitempty"`
	To      Status    `json:"to,omitempty"`
	Text    string    `json:"text,omitempty"`
	Session string    `json:"session,omitempty"`
	Agent   string    `json:"agent,omitempty"`
	Call    string    `json:"call,omitempty"`
	Files   []string  `json:"files,omitempty"`
}

func (s Store) RecordTicket(event Event) error {
	ref, err := SplitID(event.Ticket)
	if err != nil {
		return err
	}
	return s.Record(ref.Key, event)
}

func (s Store) Thread(id string) ([]Event, error) {
	ref, err := SplitID(id)
	if err != nil {
		return nil, err
	}
	all, err := s.Events(ref.Key, 0)
	thread := []Event{}
	for _, event := range all {
		if event.Ticket == id {
			thread = append(thread, event)
		}
	}
	return thread, err
}

func (s Store) Record(key string, event Event) error {
	event.At = cmp.Or(event.At, time.Now().UTC().Truncate(time.Second))
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.locked(key, "events", func() error {
		file, err := os.OpenFile(filepath.Join(s.BoardDir(key), EventsFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, err = file.Write(append(line, '\n'))
		return errors.Join(err, file.Close())
	})
}

func (s Store) Events(key string, from int) ([]Event, error) {
	file, err := os.Open(filepath.Join(s.BoardDir(key), EventsFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	var events []Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 1<<24)
	for line := 1; scanner.Scan(); line++ {
		if line <= from {
			continue
		}
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return events, fmt.Errorf("%s line %d: %w", EventsFileName, line, err)
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}
