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

The app includes pre-built sample scenarios in `frontend/public/samples/`:

| Scenario                        | Scale                           | HA   | Use Case                        |
| ------------------------------- | ------------------------------- | ---- | ------------------------------- |
| `small-foundation.json`         | 4 hosts, 10 cells               | None | Dev/test environment            |
| `medium-foundation.json`        | 8 hosts, 50 cells               | N-2  | Staging environment             |
| `large-foundation.json`         | 16 hosts, 500 cells, 2 clusters | N-2  | Production                      |
| `multi-cluster-enterprise.json` | 36 hosts, 1000 cells, 3 AZs     | N-3  | Enterprise production           |
| `cpu-constrained.json`          | 4 hosts, 128 cells              | N-1  | CPU bottleneck (8:1 vCPU ratio) |
| `memory-constrained.json`       | 4 hosts, 40 cells               | N-1  | Memory bottleneck               |
| `diego-benchmark-50k.json`      | 20 hosts, 250 cells             | None | CF benchmark: 62.5K instances   |
| `diego-benchmark-250k.json`     | 36 hosts, 1000 cells, 3 AZs     | N-3  | CF benchmark: 250K instances    |

Load these via the dashboard's "Load Sample" dropdown — no credentials needed.
