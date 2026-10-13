---
id: interface-craft
domain: dev
document: framework-neutral craft for a change that draws or moves something: numbers not adjectives, how often decides, durations, springs, amplitude, interruption, the four states, feedback once per action
---

# Interface craft

These hold in a browser, a native toolkit or a GPU-drawn interface. A framework's own reference says how to get them out of that framework.

## Numbers, not adjectives

Every motion names its property, its from and to, and either a spring (stiffness, damping, mass) or a duration and a curve. "Smooth", "snappy" and "subtle" are not specifications; a reviewer cannot check them and the next change cannot keep them.

## How often it is seen decides

- Hundreds of times a day, a shortcut, a palette, a tab switch: no entrance and no exit. The press still answers at once.
- Dozens of times a day, hover, list navigation, a toggle: the least motion that reads, or none.
- Occasionally, a dialog, a drawer, a toast: a standard short motion.
- Once, a first run or a first success: room for more.

## Durations and curves

- Hover and press 100 to 150 ms. A toggle or small state change about 200 ms. A panel or dialog 200 to 300 ms. A whole sequence 400 to 600 ms.
- Press is faster than release. Exit is faster than enter, about 0.6 to 0.7 of it.
- One ease-out curve for anything not on a spring: `cubic-bezier(0.23, 1, 0.32, 1)` or `cubic-bezier(0.16, 1, 0.3, 1)`. Linear only for constant motion such as a spinner.

## Springs

- A spring keeps its velocity when its target changes, so it suits anything a person can trigger again or grab mid-flight. A duration suits a fade or a motion that must end at a known time.
- Starting points: stiffness 500, damping 25, mass 1 for position; 550 and 30 for scale; 600 and 25 for a snappy control; damping at `2 * sqrt(stiffness * mass)` when it must not overshoot.
- From a visual duration `vd` seconds with no bounce: `w = 2 * pi / (1.2 * vd)`, stiffness `w * w`, damping `2 * w`.
- Bounce is spent on a drag or a flick, never on a routine menu.

## Amplitude

- Hover scale about 1.02, press 0.96 to 0.98. Offsets 4 to 16 px for a small element, 20 to 40 px for a large one.
- An icon that changes crossfades with a small offset or scale; it never cuts.
- Frequent things move least. Rare things may move more.

## Interruption

- A motion triggered again before it ends turns around from where it is, carrying its velocity. It never restarts from its origin and never stacks a second copy on the first.
- Something that follows the pointer between items is one object moving, not one fade per item.

## What to animate

- Transform and opacity. Never a property that re-runs layout or reshapes text inside a moving frame.
- Nothing per streamed token, per tool call or per log line. The stream is its own feedback; one settle when it ends.
- Nothing draws when nothing moves.

## The four states

Every view that loads data has an empty, a loading, an error and a filled state, and each is designed rather than left to fall out of the code. A control has rest, hover, press, focus and disabled, and focus is visible from the keyboard.

## Feedback

- One response per action, from the innermost target. Two nested handlers do not both animate.
- It starts on the frame of the event, never after a debounce.
- A repeat within about 70 ms plays smaller and shorter; past about 220 ms it plays in full.
- Blocking the person, such as a permission prompt, earns the one strongest signal; nothing else competes with it.

## Reduced motion

When the platform reports it, position, size and scale go to their target at once; opacity and colour that say something changed still fade; decorative loops stop. Nothing is left in a half-state.

## The report

Each motion as property, from and to, and spring or duration and curve, with where the numbers came from; then measured frame numbers on an optimized build in a focused window. A motion with no numbers beside it was not judged.
