---
title: Boardy managers and rules
description: Who may create, move and close a ticket on a boardy board, how other sessions ask through triage, how a ticket moves to another board, and board rules that stop a move.
order: 19
updated: 2026-10-13
---

A boardy board has managers. A manager creates, assigns, accepts and closes
tickets. Every other session asks: its `new` becomes a request in the
board's triage, and a manager accepts it, which gives it a number, or drops
it. Board rules decide what a move needs, and when the person has to approve.

## Why

**Two sessions never write one ticket at once.** Numbers come only from a
manager's create or accept, under the board's lock, so two sessions cannot
both write `DEMO-12`.

**An agent works like an engineer.** It moves its own ticket from doing to
review, with its evidence in the Log, and appends to its own Log. It never
changes owns and never closes.

**Owns are checked on every change.** A create, a widen or a move to doing
is refused when its owns overlap a doing ticket's owns on any board of the
project, and the refusal names that ticket. Only doing holds owns, unless
`holds` in `board.toml` names more statuses.

**The person is asked only when a rule says so.** Without a rule, a manager
closes a ticket. With one, the close waits for the person.

## How to

Name the managers. A manager is a session id or the person's name:

```
tofu boardy manager add s-7f2c
tofu boardy manager add luiz --as luiz
tofu boardy manager list
```

A board with no managers is run by the person. Only the person or a manager
changes the list.

Accept or drop what other sessions asked for:

```
tofu boardy triage list
tofu boardy triage accept T1
tofu boardy triage drop T2 --reason duplicate of DEMO-4
```

Assign, widen owns, and hand a ticket to another board:

```
tofu boardy assign DEMO-3 s-7f2c
tofu boardy widen DEMO-3 "internal/rule/**"
tofu boardy hand DEMO-3 OPS
tofu boardy hand DEMO-3 OPS --project ../other-repo
```

`hand` gives the ticket a new number on the other board with its whole Log,
and `show DEMO-3` says `handed to OPS-1`.

## Board rules

A board rule is a rule file, the same shape as a library rule, in
`~/.tofu/projects/<project>/boards/<KEY>/rules/`. It fires on one event:
`create`, `move`, `widen` or `close`, and `to:` narrows a move or a close
to one status. A close is a move to done or dropped.

The person approves every done:

```yaml
id: person_closes
domain: general
kind: structural
concern: process_discipline
checker: person_approves
mode: enforced
text: the person approves every done
on: close
to: done
```

A command decides. A non-zero exit refuses the move with the rule's text:

```yaml
id: test_line
domain: general
kind: structural
concern: process_discipline
checker: command
command: grep -q "test:" {ticket}
mode: enforced
text: a ticket reaches review with a test line in its Log
on: move
to: review
```

`{ticket}` is the ticket's file. The command also gets `TOFU_BOARD_EVENT`,
`TOFU_TICKET`, `TOFU_TICKET_ID`, `TOFU_FROM`, `TOFU_TO` and `TOFU_ACTOR`. A
rule in `shadow` mode never stops a move; what it found goes in
`events.jsonl`.

Two board rules need no file. Set them in `board.toml`:

```
log_to_review = 200
scope_cites = ["docs/research/", "no source"]
```

The first refuses a move to review while the Log has fewer than 200
characters; the second refuses a create or a widen whose Scope names none of
the listed sources. Unset, a review needs any Log and a Scope may say
anything.

An outside hook runs every check the real command runs, role, owns,
Acceptance, Log, references and rules, and writes nothing:

```
tofu boardy check move DEMO-7 --to doing
```

It exits 1 with the same refusal the move would print, such as
`DEMO-7 refused: cmd/tofu/boardy*.go overlaps cmd/tofu/**, held by OPS-1 (doing)`.

## Who is acting

Inside a tofu session, `TOFU_SESSION` names the session, and `--as` is
ignored. The lead session acts for the person: it manages a board that
lists no managers or lists `person`, so it spawns for a ticket on a fresh
board without being added. A sub-agent, named `<session>/<ticket>`, is an
agent and never acts as the person. Outside a session the command line is
the person, named by `--as`.

## Reference

- Roles: manager (named in `board.toml`), the person and the lead session acting for them, and every sub-agent or other session as an agent.
- Moves an agent may make: its own ticket, doing to review, with a Log.
- A move to doing, review or done needs an Acceptance; review also needs a Log, of at least `log_to_review` characters when the board sets it.
- Triage requests live in `boards/<KEY>/triage/T<n>.json` until accepted or dropped.
- Every accept, drop, hand, widen, assign and rule that fired is a line in `events.jsonl`.
