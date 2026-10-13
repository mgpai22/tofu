# bench profile: what tofu's memory and CPU are made of

**tofu itself is 43.5 MB of the 387 MB tree peak on the bench/rivals task (11%) and 2.38 s of its 55.1 s CPU (4.3%).** The rest is the model's own commands: `go vet` compiling the standard library cold is 318 MB of the peak and 30 s of the CPU, and `go test` is 21 s. Inside tofu the Go heap in use never passes 17 MB; its working set is half private (22.6 MB) and half shared image pages (21.7 MB).

**One sub-agent adds 1.3 MB of tree peak and 0.117 s of CPU.** tofu's own process is 1.2 MB and 0.103 s of that, the undo snapshot's `git` 0.017 s. Half of what each sub-agent allocates is the project registry read from disk and decoded again.

Runs on 2026-10-13, 18:06 to 18:17, windows/amd64 with 32 logical CPUs, one at a time behind the shared rivalslot. Live rows on the owner's claude-sub subscription (claude-sonnet-5-5), cassette rows on a fresh home. Build 0.5.13-fix1+dev at f360965a, go1.27.1. `PROFILE_WORK=<scratch> PROFILE_SLOT=<slot dir> bash bench/profile/run.sh`, or `PROFILE_LIVE=no` for the cassette rows only; `go run ./bench/profile report` prints every row from `runs/`.

## How it is measured

- A Windows job object holds the whole tree and posts each new member to a completion port; every member is opened at once and kept, so its final CPU and peak working set are read even after it exits. 421 of 421 processes seen on the rivals task, CPU per process summing to the job's.
- Tree RSS is the live members' working sets summed every 100 ms (bench/rivals and bench/scale sample every 250 ms).
- A process's kind comes from its ancestry: under `go.exe` it is `go <verb>` of the outermost one, under `gofmt.exe` it is gofmt, under a shell it is shells, tofu's direct children keep their image name.
- tofu profiles itself when `TOFU_PPROF=<dir>` is set: `cpu.pprof`, `heap-peak.pprof` each time the heap in use grows by 1 MB, `heap-end.pprof` and `memstats.json`, written before `os.Exit`. `profile sample -pprof` sets it; the stock rows run the same binary without it. The rows above the TOFU-1319 section were taken with the `go build -overlay` hook that came before it. Profiling costs 8.6 MB and 0.09 s; every size below comes from the stock rows, profiles only attribute.
- Go's CPU profiler on Windows counts threads blocked in a system call; every CPU top drops waiting samples, and what is left is within 10% of the process's own CPU time.

## The bench/rivals tofu-sonnet run, split

73.4 s wall, job CPU 55.14 s, peak tree RSS 387.0 MB, peak commit 533.4 MB.

| kind | processes | MB at the tree peak | largest one process MB | CPU s |
|---|---|---|---|---|
| go vet | 209 | 318.0 | 263.7 | 29.95 |
| go test | 167 | 0.0 | 144.7 | 21.30 |
| tofu | 1 | 43.5 | 44.3 | 2.38 |
| shells | 21 | 25.5 | 11.4 | 0.50 |
| git (undo snapshot) | 18 | 0.0 | 8.2 | 0.72 |
| go version (tofu's toolchain probe) | 2 | 0.0 | 13.2 | 0.19 |
| rtk (the owner's hook) | 1 | 0.0 | 19.0 | 0.06 |
| gofmt | 2 | 0.0 | 8.1 | 0.05 |

The spread between runs (376 to 412 MB) is in `go vet`, not in tofu (42.1 to 43.5 MB).

**tofu's heap**: in use peaks at 17.0 MB; 164.1 MB allocated over the run in 32 collections, at most 37 goroutines. `sys.readProjectRegistry` is 61.2 MB of the 144.6 MB allocated (41%): every ledger row (48.7 MB), the shell sift's scorer (29.2 MB) and the gate (23.2 MB) read and decode the registry again.

**tofu's CPU** is system calls, not Go code: stat and open 0.46 s, a thread walk 0.19 s (`shell.resumeSuspended` walking every thread on the machine), process creation 0.11 s. By function: `turn.gatedCall.run` 0.65 s, `main.appendRow` 0.48 s, `main.(*toolGate).Decide` 0.34 s, `boundClassifier` reopening the credential store 0.16 s. Garbage collection 0.04 s.

## One sub-agent added, on the bench/scale tiny task

| agents | wall s | job CPU s | peak tree MB | tofu own peak MB | tofu CPU s | git processes |
|---|---|---|---|---|---|---|
| 0 | 3.9 | 1.00 | 41.8 | 37.7 | 0.47 | 12 |
| 1 | 8.7 | 1.31 | 44.5 | 38.7 | 0.67 | 16 |
| 10 | 30.4 | 2.17 | 54.9 | 49.5 | 1.50 | 23 |

Per sub-agent: 1.31 MB of tree peak (bench/scale fitted 0.4 MB over seven rungs; one pair here, so neither is settled), 0.56 MB of heap, 4.7 goroutines, 0.117 s CPU, 8.4 MB allocated of which 45% is the registry. `SpawnTool.background` is 55 ms per agent: 17 ms re-reading and saving the session header, 13 ms re-reading settings for the spawn's limits, 12 ms reopening the credential store. On a fresh home `gh api user` (39 MB, 0.2 s), the once-a-day memory identity probe, is the tree peak.

## Where tofu can get smaller and faster, ranked by the saving measured

1. **Undo snapshot without four `git` processes a step** (`internal/snapshot`): up to 0.7 s CPU a task, 47% of the tiny task's tree CPU, 8 MB at a coincident peak.
2. **The `gh` identity probe off the startup path** (`internal/memory/author.go`): 38.6 MB at the peak and up to 0.5 s of startup, once a day per home.
3. **Resume a suspended shell with NtResumeProcess** instead of walking every thread (`shell.resumeSuspended`): 0.21 s CPU a task, about 50 ms per bash command.
4. **Gate credentials and the ledger's directory resolved once per process**: about 0.3 s CPU a task, 12 ms per sub-agent.
5. **A sub-agent's settings and session header kept in memory**: about 30 ms of the 103 ms a sub-agent costs tofu.
6. **The project registry cached per process, keyed by modification time**: 41% of all allocation, 2.1 MB per sub-agent; small in time (about 20 ms).
7. **The Go toolchain probed once per process** (`internal/turn/tool_bash.go`): 0.1 s CPU and one 13 MB process a task.
8. **Git for Windows' real bash, not its launcher**: one 6.1 MB process and about 20 ms less per command, Windows only.
9. **The HTML sanitizer's init deferred**: about 5 ms and 0.75 MB per process start.

Not ranked: 21.7 MB of tofu's working set is shared pages from the 49.7 MB binary and system DLLs; `tofu version` alone peaks at 18.7 MB.

## After TOFU-1319

Items 1, 3 and 8 are in, item 6 in part, and `TOFU_PPROF` replaced the overlay. **tofu's own CPU on the rivals task went from 2.38 and 2.14 s to 1.28 and 1.14 s, its allocation from 144.6 and 176.3 MB to 66.9 MB, and its git processes from 18 to 7.** Runs on 2026-10-13, 19:00 to 19:35, same machine, same slot, before and after binaries built from one tree; `runs/` holds the final after rows (no suffix), the rows of a first after build that cached the registry in a package variable (`after-…-r2` to `-r4`, withdrawn because this project holds no global state), and the second before round (`before-…-r2`). The first before round is the table above.

- The undo snapshot writes its tree in process from the undo index (format 2 and 3; anything else, a split or sparse index, or a sha256 store falls back to `git write-tree`), so a step is `ls-files` and `add`, two processes instead of three. The tree ids match git's: a driven turn recorded the seed commit's tree as its start and the same id `git write-tree` prints over the undo index as its end.
- Prune runs at most once an hour per store, and never on a store younger than an hour, because nothing in it can be older than the one hour grace: the stamp is `undo.git/objects/info/pruned`. Before, every session start ran one `write-tree` per session folder, then `update-ref` and `prune`.
- A suspended shell resumes with one `NtResumeProcess` call instead of a snapshot of every thread on the machine.
- Finding a registered project streams the registry and decodes only each entry's folder and paths: 194 KB a lookup instead of 957 KB over a 1100 entry registry, about the same CPU (3.3 against 2.9 ms). An unregistered project, a registry edit and `ProjectGlob` still decode it whole. Decoding it once per process needs a value the callers share, which they do not have yet.
- Git for Windows' `bin\bash.exe` launcher is replaced by `usr\bin\bash.exe` with the environment the launcher sets (`MSYSTEM=MINGW64`, `EXEPATH`, `PLINK_PROTOCOL` when unset, `HOME` when unset, and `mingw64\bin`, `usr\bin` and `$HOME\bin` before `PATH`), checked by diffing `env` under both: one process fewer per command. Only when that layout exists; every other shell, and every other OS, runs as before.
- The key redactor skips its regular expression when no key name occurs in the text.

| row | tofu CPU s | job CPU s | git processes | tool calls | allocated MB |
|---|---|---|---|---|---|
| rivals before, TOFU-1317 | 2.38 | 55.14 | 18 | 25 | 144.6 (profiled row, 16 calls) |
| rivals before, r2 | 2.14 | 52.16 | 18 | 14 | 176.3 (profiled row, 35 calls) |
| rivals after | 1.28 | 50.16 | 7 | 19 | |
| rivals after, profiled row | 1.14 | 48.41 | 7 | 24 | 66.9 |

Per sub-agent, job CPU from the live tiny task, (10 agents less 0) / 10: before 0.117 and 0.142 s; after 0.104 s, and 0.061, 0.103, 0.070 and 0.098 s with the withdrawn cache. On the cassette, where the turns are fixed, 0.091 and 0.100 s before against 0.098 s after (0.076 and 0.071 s with the cache). What is left per sub-agent is outside these files: the credential store opened by `LoadKeyRedactor` in `turn.Run` (8 ms) and by `cred.Open` in the sub-agent opener (7 ms), and settings re-read through `settingsPaths` (about 9 ms), which are items 4 and 5.

Not done here: item 2 needs `internal/memory/store.go`, where `Block` asks for the author whether or not a user-local shelf exists; item 7 is already once per process, and the two `go version` processes are the go command and the toolchain it switches to.

## What these numbers cannot say

- One run per row; the stock rivals peak moved 25 MB between two runs on the same binary, all in `go vet`.
- Live rows ran on the owner's real home for the login, so his hooks (rtk) and his project registry applied; the registry's size sets item 6's cost.
- Linux and macOS are not measured: the tree split uses Windows job objects, and `profile sample` refuses elsewhere.
- A live row moves with what the model chose to do: the before rivals rows made 25 and 14 tool calls and the after rows 19 and 22, and a 10 sub-agent lead takes two or three turns, each of which snapshots twice.
