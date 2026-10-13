---
name: debug
domain: dev
description: Reproduce a reported symptom with one command that goes red, find its cause one hypothesis at a time, fix the cause and show the same command go green. Run by /debug with the symptom.
---

# Debug

The person named a symptom when they typed the command. No fix is written until step 3 has a cause.

## 1. Reproduce

Write one command that shows the symptom and can fail on this bug alone: a test, a script, a request. Run it and keep what it printed. If it cannot be made to fail, say so and ask what the person did, rather than fixing what you imagine.

## 2. Read before theorising

Read the whole error, the stack and the log. Read what changed recently near the failure: `git log -p` on the files involved, the current diff.

## 3. Narrow, one hypothesis at a time

- State one hypothesis and the observation that would refute it.
- Add temporary logging at the boundary where the value goes wrong, between components rather than inside one, and run the reproduction again.
- Change one thing per probe. When a hypothesis is refuted, revert everything it motivated.
- After two failed fixes that rest on the same premise, stop fixing and question the premise.

## 4. Fix the cause

Fix where the wrong value is made, not where it is noticed. No guard, nil check or fallback that only silences the symptom. Search for the same pattern elsewhere and fix it there too.

## 5. Prove it

Run the reproduction from step 1: it goes from red to green. Run the covering tests. Remove the temporary logging in the same change.

## Report

The symptom, the reproduction command, the cause with path:line, the fix, and the red and the green output. Mark every claim as measured, inferred or guessed.
