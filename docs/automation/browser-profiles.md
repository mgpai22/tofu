---
title: Browser profiles
description: A Chrome profile of tofu's own for agents, signed in once and kept, with or without a window.
order: 4
updated: 2026-10-13
---

A browser profile is a Chrome profile that belongs to tofu rather than to you.
With `browserProfile` set, tofu starts the Chrome you have installed on that
profile's own folder and drives it directly, with no extension. Sign a site in
once, a Google account included, and every later run finds it signed in.

Use one when an agent should browse as its own user, in the background, while
your own Chrome stays yours. With `browserProfile` empty, tofu drives your own
Chrome through the [extension](/docs/automation/browser-extension), as before.

## Make one and sign in

```text
tofu browser profile new work
tofu browser profile login work
tofu settings set browserProfile work
```

`login` opens a Chrome window on the profile at the Google sign-in page, or
at the address you give it, with no debugger attached. Sign in, then close
the window: an agent cannot start the profile while that window is open.

Each profile is its own Chrome user data folder under
`~/.tofu/browser/profiles/<name>`. tofu never starts Chrome on your own
profile folder, which Chrome refuses to debug anyway.

## Headless or not

`browserHeadless` is on by default, so the Chrome tofu starts has no window.
`tofu settings set browserHeadless false` shows it. The setting is read when
that Chrome starts; a Chrome already running keeps its mode until it stops.

## How it runs

The first browser call starts Chrome with the profile folder, a debugging
port Chrome picks, and `--headless=new` when headless, then reads the port
from the `DevToolsActivePort` file Chrome writes there. Later calls, from any
tofu, attach to that same Chrome, so its tabs and cookies carry from one
command to the next. It keeps running until you stop it:

```text
tofu browser profile stop work
```

Every tab in a profile's Chrome is tofu's own, so the browser tools read and
act in all of them. tofu finds Chrome where it installs on each system:
under Program Files or your local app data on Windows, in `/Applications` or
`~/Applications` on macOS, and as `google-chrome`, `google-chrome-stable`,
`chromium` or `chromium-browser` on your path on Linux.

## Commands

| Command | What it does |
|---|---|
| `tofu browser profile list` | each profile, `running` or `stopped`, and its folder |
| `tofu browser profile new <name>` | makes an empty profile; a name is lowercase letters, digits and hyphens |
| `tofu browser profile login <name> [<url>]` | opens a window on it to sign in |
| `tofu browser profile stop <name>` | closes the Chrome tofu started on it |
| `tofu browser profile remove <name>` | stops it and deletes its folder |
| `tofu browser read [--tab <id>]` | prints a tab's title and address |

```text
Browser profiles · 2 profiles · ~/.tofu/browser/profiles      ✓ listed

  ● work   running  ~/.tofu/browser/profiles/work
  ○ other  stopped  ~/.tofu/browser/profiles/other
```

> [!WARNING]
> A profile folder holds the sessions you signed in with. Treat it as a
> password, and remove the profile when you no longer need it.
