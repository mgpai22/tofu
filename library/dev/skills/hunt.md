---
name: hunt
domain: dev
description: Look for bugs with three separate sub-agents, a hunter, a skeptic and a referee, passing only structured findings between them, and keep only what can be shown now. Run by /hunt, with an optional path or area.
---

# Hunt

You run the hunt and judge nothing yourself. Each stage is a sub-agent with its own context, and only findings pass between stages, never your reasoning or the session.

## 1. Scope

Take the path or area the person named. With none, take the files the current diff touches (`git diff --name-only HEAD`), or on a clean tree the last commit's. Rank them by risk before any reading: size, how often they changed (`git log --format= --name-only`), and whether they parse input, hold state or run concurrently. Split a large scope into shards of a few files each.

## 2. Hunter

Spawn the `hunter` sub-agent, one per shard and in parallel when there are several, each given its shard and nothing else. Wait until every hunter has reported.

## 3. Skeptic

Spawn the `skeptic` sub-agent with the hunters' findings as they wrote them and nothing else: no session, no opinion of yours. Wait until it has reported.

## 4. Referee

Spawn the `referee` sub-agent only after the hunter and the skeptic have both reported. Give it both reports. It keeps a finding only when it can show it now: a failing test, a reproduction, or an input and the path it takes.

## 5. Report

The referee's confirmed findings, each with path:line, severity and the evidence that shows it; then the count of findings dismissed and why. Fixing is a separate step the person asks for: change no code in a hunt.
