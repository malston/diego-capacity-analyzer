# Demo Materials

Presentation and reference materials for demoing the Diego Capacity Analyzer.

## Quick Start

```bash
# Run the demo (starts backend + frontend, serve presentation slides on port 8888, opens browser)
./run-demo.sh
```

## Files

| File                                   | Purpose                                            |
| -------------------------------------- | -------------------------------------------------- |
| `run-demo.sh`                          | One-command demo launcher                          |
| `demo-plan.md`                         | Full 25-minute presentation plan with timing       |
| `demo-script.md`                       | One-page cheat sheet for presenters                |
| `demo-slides.html`                     | Reveal.js slide deck (press `S` for speaker notes) |
| `demo-slides.md`                       | Slide deck outline (markdown)                      |
| `feature-walkthrough.html`             | Detailed feature walkthrough presentation          |
| `feature-walkthrough-screenshots.html` | Feature walkthrough with embedded screenshots      |
| `formula-cheatsheet.md`                | Mathematical formulas and thresholds reference     |

## Viewing Presentations

The HTML slide decks use [Reveal.js](https://revealjs.com/) and need to be served via HTTP:

```bash
# Option 1: Use the demo script
./run-demo.sh
# Then open http://localhost:8888/demo/demo-slides.html

# Option 2: Serve manually from project root
cd ../..  # go to project root
python3 -m http.server 8888
# Then open http://localhost:8888/docs/demo/demo-slides.html
```

**Keyboard shortcuts in presentations:**

- `→` / `←` — Navigate slides
- `S` — Open speaker notes (separate window)
- `O` — Overview mode
- `F` — Fullscreen

## Reference Documents

- **FAQ**: [`docs/FAQ.md`](../FAQ.md) — Common questions and answers
- **UI Guide**: [`docs/UI-GUIDE.md`](../UI-GUIDE.md) — Full dashboard documentation
- **Formula Cheatsheet**: [`formula-cheatsheet.md`](./formula-cheatsheet.md) — All calculations explained

## Sample Data

The app includes 9 pre-built sample scenarios in `frontend/public/samples/`:

| Scenario                 | Description                                    |
| ------------------------ | ---------------------------------------------- |
| `small-dev.json`         | 3 hosts, 6 cells — small dev environment       |
| `medium-production.json` | 8 hosts, 50 cells — typical production         |
| `large-enterprise.json`  | 15 hosts, 200 cells — enterprise scale         |
| `overcommitted.json`     | Shows what happens with high overcommit        |
| `constrained.json`       | Near capacity limits                           |
| ...                      | See `frontend/public/samples/` for all options |

Load these via the dashboard's "Load Sample" dropdown — no credentials needed.
