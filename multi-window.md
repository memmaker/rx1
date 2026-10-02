# Multi-window mode

rx1 normally draws everything in one terminal grid: the message line, the map, the inventory and visible-monster
panels on the right, and the status bar. In multi-window mode those side windows are moved out of the grid. The
terminal then shows only the map, and the map gets all of the space.

There are two multi-window front ends:

| Front end | How to start it | Where the side windows go |
|---|---|---|
| Terminal | `rx1 -p DIR` (can be combined: `rx1 -p DIR -n Name`) | one ANSI text file per window in `DIR` |
| Web | always on in `fx-games/site/rx1/play/` | RVIP-WM windows next to the map |

Nothing changes without `-p` on the desktop. Classic mode is still the code path when no pane sink is set.

## The seam: `console.Panes`

All the work happens in one small interface in [console/panes.go](console/panes.go):

```go
type Panes interface { Set(name string, p Pane) }
type Pane struct { Text string; Fg, Bg color.RGBA } // cview-tagged text + the colours "-" resets to
```

- `UI.SetPanes(p)` turns the mode on. It must be called before the dungeon UI is built. `main` does this in
  `prepareUI`.
- The five places that filled the side `TextView`s now call `u.setPane(view, name, text)`. With no sink, that is
  the old `SetText`/`setColoredText`. With a sink, the text goes to `Panes.Set` instead.
- With a sink, `InitDungeonUI` builds a grid that holds only the map. The classic grid now lives in
  `addClassicPanels`, unchanged. The resize handler leaves the grid alone and skips the minimum-size screen,
  because the map scrolls.
- Two layout guesses that depended on the terminal width are fixed for panes: the inventory always shows full
  names, and the status bar always uses its two-line form.

Pane names:

| Name | Content |
|---|---|
| `messages` | the whole message log (faded like the classic log) |
| `prompt` | the one-off info line (`Print`: look mode, "No enemies in sight", …) |
| `inventory` | the inventory list |
| `visible` | visible monsters with health bars |
| `status` | the two-line status bar |

`Pane.Lines()` parses the cview tags rx1 writes (`#rrggbb` colours, `-` resets, `r` reverse) into styled spans and
trims trailing blank text. `Pane.ANSI()` and `Pane.HTML()` render those spans. Both front ends use them, so
neither one parses tags itself. The check is in [console/panes_test.go](console/panes_test.go).

The console package knows nothing about files or browsers. The front ends live in `main`, behind the existing
build tags.

## Terminal front end ([startup_desktop.go](startup_desktop.go))

`rx1 -p DIR` creates `DIR` and writes `DIR/<pane>.ans` on every change: `messages.ans`, `prompt.ans`,
`inventory.ans`, `visible.ans`, `status.ans`. Each file is replaced whole: it is written to `.tmp` and then
renamed, so a reader never sees half a file. The content is 24-bit ANSI colour, one line per row.

To watch the side windows, open other terminals (or tmux/iTerm splits) next to the game:

```bash
watch -tc -n 0.2 cat /tmp/rx1/inventory.ans
```

The game terminal shows only the map, using the whole window.

## Web front end ([startup_js.go](startup_js.go) + `fx-games/site/rx1/play/index.html`)

- If the page defines `rvipPane`, `prepareUI` installs `webPanes`. It calls
  `rvipPane(name, html, plainText)` for every pane. The page puts the HTML into the window's `<pre>`, and the
  `prompt` pane goes to `RvipWM.prompt` over the map.
- The page has five RVIP-WM windows: Map, Messages, Inventory, Visible and Status. You can drag, resize, rename,
  close and reopen them through the **Windows ▾** menu, and change each one's text size with A−/A+.
- The Map window holds the tcell terminal. `fitMap()` measures one cell at the map's font size and calls
  `rvipResize(cols, rows)`, which Go exports and which calls `screen.SetSize`. tcell's grid is therefore always
  exactly what fits the Map window, and resizing never scales text (RVIP rule 3). The map below 20×8 cells stops
  shrinking.
- The window layout is saved in IndexedDB (database `rx1`, store `kv`, key `layout`). The page uses no
  localStorage.
- `build_web.sh` copies `~/Games/rvip-tools/web/rvip-wm.js` next to the page. rx1 is hosted on ruzzoli.de, not
  under `/roguelikes/`, so the shared `../rvip-wm.js` isn't there.

## Menus inside a window

`UI.drawToPane(box, pane, w, h, draw, restore)` makes a modal draw into a pane instead of over the map. Each frame,
`draw` runs on an off-screen `tcell.SimulationScreen` of `w() × h()` cells, and the cells become that pane's spans.
The modal keeps its cview panel, so it still has the keyboard and all of its key handling. When the map is the
front panel again (`Panels.SetChangedFunc`), the `restore` functions put each pane's normal content back.
`sendPane` sends a pane only when its content changed, because menus redraw every frame.

The inventory menu (`i`, and the item pickers built on `openInventory`) uses this. It opens inside the Inventory
window: on the web and in `inventory.ans`.

## Known limits

- Other modals (Space actions, vendor, character sheet, help) still open over the map's terminal grid. Each one can
  move into a window with one `drawToPane` call. A modal wider than a small Map window is clipped.
- You can't click in a menu that is in a side window; the keys work.
- The page has no RvipApp: saves still live in the in-memory file system, so a reload loses the game, as before.
- The terminal front end has no layout of its own. Placing the files in windows is up to your terminal or tmux.
