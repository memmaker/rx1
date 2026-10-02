# Level generation and world structure: Rogue 3.6 vs Rogue 5.4 vs rx1

Sources read (working tree, including uncommitted rx1 work):

- Rogue 3.6: `/Users/felix/Games/rogue3.6/` (`rooms.c`, `passages.c`, `newlevel.c`, `monsters.c`, `daemons.c`, `command.c`, `move.c`, `rogue.h`, `init.c`, `rip.c`, `things.c`).
- Rogue 5.4: `/Users/felix/Games/rogue5.4/` (`rooms.c`, `passages.c`, `new_level.c`, `monsters.c`, `daemons.c`, `command.c`, `move.c`, `rogue.h`, `extern.c`, `misc.c`, `scrolls.c`, `things.c`, `rip.c`).
- rx1: `dungen/rogue.go`, `dungen/base.go`, `dungen/style.go`, `dungen/room.go`, `game/dungeon.go`, `game/spawn_rogue.go`, `game/special_rooms.go`, `game/town.go`, `game/state.go`, `game/actions.go`, `game/object.go`, `game/deamons.go`, `foundation/objects.go`, `config.rec`.

Verdicts: SAME = same rule and numbers. DIFFERENT = rule or numbers differ. rx1-only = rx1 has it, Rogue does not. "3.6 vs 5.4" marks where the two Rogues differ from each other.

rx1 picks a generator style per level (see section 0). Everything below about "rx1" means the Rogue-style generator (`dungen.RogueGenerator`) unless said otherwise. `game.spawnEntities` populates every style the same way.

---

## 0. Level styles (rx1-only structure)

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Generators | one (do_rooms + do_passages) | one (same, plus maze rooms) | five: Rogue grid, NetHack rooms, NetHack cave, NetHack maze, Brogue (`dungen/style.go`) | rx1-only |
| Style per level | n/a | n/a | `PlanLevelStyles`: levels 1 and 2 always Rogue; the rest by weight Rogue 30, Rooms 25, Cave 15, Maze 10, Brogue 20; any style that did not come up is given to a level of a style that came up more than once (so a full 26-level run shows all five) | rx1-only |
| Wizard override | n/a | n/a | wizard menu can force a style for one generation | rx1-only |
| Lava | n/a | n/a | cave and Brogue levels from level 13 (`LavaFromLevel`), keeps all walkable tiles connected (`dungen/lava.go`) | rx1-only |
| Lit rooms in other styles | n/a | n/a | NetHack `lit()` = `rn2(1+level)+1 < 11 && rn2(77) != 0`; Brogue: never lit; mega dungeon: 1 in 4 lit | rx1-only |
| Secret doors in other styles | n/a | n/a | NetHack: none on levels 1-2, then 1 in 8 per door (`maybeSDoor`), secret corridor squares 1 in 100, secret niches; Brogue: `min(max((level-1)*67/25,0),67)` percent of doors | rx1-only |
| Secret level | n/a | n/a | one hidden "stairs down" under a random corridor (or floor) tile of a level 7..12 (`secretLevelDepth = 7 + rand.Intn(6)`); found by walking over it or by searching; leads to a Hauberk mega dungeon 2x width by 3x height (about 220 rooms), 3 guaranteed treasure rooms, objects and traps scaled by rooms/9; its only stairs lead back (`gotoLevel`, `hideSecretStairs`) | rx1-only |

---

## 1. Room grid, size, gone rooms, dark rooms, mazes

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Grid | 3x3, MAXROOMS 9 (`rogue.h`) | 3x3, MAXROOMS 9 | 3x3, 9 rooms (`doRooms`) | SAME |
| Cell size `bsze` | `COLS/3`, `LINES/3` of the terminal (`rooms.c:do_rooms`); 80x24 gives 26x8; a bigger terminal gives bigger rooms | `NUMCOLS/3`, `NUMLINES/3` fixed 80x24: 26x8 | `(mapWidth-1)/3`, `mapHeight/3`; `config.rec` MapWidth 80, MapHeight 23 gives 26x7 (the map has no message or status line, but the "row 0 is not used" rule below is kept) | DIFFERENT (3.6 vs 5.4: terminal-dependent vs fixed). rx1 cell is 7 tall, so rooms are at most 6 tall and rows 21-22 stay empty |
| Cell origin | `top.x = (i%3)*bsze.x + 1`, `top.y = (i/3)*bsze.y` | same | same | SAME |
| Room size (walls included) | `max.x = rnd(bsze.x-4)+4` (4..25), `max.y = rnd(bsze.y-4)+4` (4..7); floor interior is max-2 | same | same formula; with bsze.y 7 gives `max.y` 4..6 (floor height 2..4 instead of 2..5) | SAME formula, DIFFERENT result because of the 7-row cell |
| Room position | `pos = top + rnd(bsze - max)`, redo until `pos.y != 0` | same | same, redo until `pos.Y != 0` | SAME |
| Gone rooms | `left_out = rnd(4)` draws (0..3), each marks `rnd_room()` gone (a repeat can pick the same room again, so 0..3 gone rooms); never all 9 | same | same (`leftOut := r.rnd(4)`) | SAME |
| Gone room position | random point in cell `top + rnd(bsze-2)+1`, `pos.y` in 1..LINES-2; no walls, only a `#` junction where corridors meet | same (`putpass`) | same; the junction is a corridor tile | SAME |
| Dark room chance | `rnd(10) < level-1` per room: 0% on level 1, 10% on 2, +10% per level, 100% from level 11 (`rooms.c:do_rooms`) | same | `r.rnd(10) < r.level-1`, same | SAME |
| Maze rooms | none | `if (rnd(10) < level-1) { ISDARK; if (rnd(15)==0) r_flags = ISMAZE; }`: only a dark room can be a maze, 1 in 15 of them, so up to 6.7% of rooms from level 11. The assignment `= ISMAZE` replaces the dark flag. Maze fills the cell: `max = bsze-1`, `pos = top` (x 1 becomes 0; if y 0 then y+1 and max.y-1) (`do_maze`, `dig`) | same dig algorithm (reservoir pick among the 4 neighbours two steps away, depth-first), `max = bsze-1`; cell 25x6; the maze stays flagged dark/unlit (`rp.dark` is not cleared) | SAME as 5.4. 3.6 has no mazes. Small difference: 5.4 maze rooms are not flagged dark but behave as passages (`roomin` gives the passage because F_PASS is set); rx1 makes a real unlit "room" of the maze tiles, so room based rules (hidden object scan, wandering spawn room, special rooms, treasure room) can pick it |
| Lit state in play | dark room: only adjacent squares seen | same | `room.SetLit(!dark)`; lit room tiles are lit permanently once explored | SAME in intent |
| Room walls | drawn `-` `|` | same | wall glyph system (themes) | cosmetic |

## 2. Passages, connections, doors, secret doors

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Connection graph | `rdes[].conn` table: orthogonal neighbours in the 3x3 grid | same | `isConnectable` computes the same table | SAME |
| Spanning tree | random start room; repeatedly pick a random unconnected neighbour of the current room (reservoir `rnd(++j)==0`); if none, jump to a random room already in the graph; until 9 rooms (`passages.c:do_passages`) | same | same loop (`doPassages`) | SAME |
| Extra connections | `rnd(5)` attempts (0..4); each picks a random room and one random adjacent not-yet-connected room; silently does nothing if none | same | same (`r.rnd(5)`) | SAME |
| Corridor shape (`conn`) | direction 'r' or 'd' from the room numbers; door at random wall point (x for down, y for right) via `rnd(max-2)+1`; one turn at `turn_spot = rnd(distance-1)+1`; gone rooms use `#` | same, plus for maze rooms re-pick the door point until it is a maze passage (infinite loop in C) | same; maze retry capped at 1000 tries; corridor squares are never placed over a Room or Door tile | SAME (cap and no-overwrite are safety changes) |
| Warning on bad connection | "Warning, connectivity problem on this level." | "warning, connectivity problem..." | none (unit test `TestRogueGeneratorConnected` checks all tiles reachable) | rx1 omits message |
| Doors | symbol `+`, no open/closed state | same (F_LOCKED defined, unused) | door tile; shown closed/open by theme, always walkable | SAME |
| Secret door chance (`door`) | `rnd(10) < level-1 && rnd(100) < 20`: P = clamp(level-1,0,10)/10 x 1/5. Level 1: 0%, 2: 2%, 6: 10%, 11+: 20% | `rnd(10)+1 < level && rnd(5)==0`: identical probability | `r.rnd(10)+1 < r.level && r.rnd(5) == 0`, per door | SAME in all three |
| Secret corridor squares (`putpass`) | none | each corridor square: `rnd(10)+1 < level && rnd(40)==0` is invisible until found (P = clamp(level-1,0,10)/10 x 1/40, max 2.5% per square) | same formula (`putPass`), kept in `secretPassages` | SAME as 5.4. DIFFERENT from 3.6 (3.6 vs 5.4 differ) |
| Maze room doors | n/a | no door is placed on a maze room (`door` returns early) | same | SAME |
| Door/exit bookkeeping | `r_exit[]` | `r_exit[]` + `passnum` passage numbering | not needed | n/a |

## 3. Gold

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Chance per room | `rnd(100) < 50` and `(!amulet \|\| level >= max_level)` | `rnd(2)==0` and the same condition | `rnd(2)==0 && (!hasAmulet \|\| level >= deepest)`; gold is per room, so non-Rogue styles with more rooms get more gold | SAME |
| Amount | `GOLDCALC = rnd(50 + 10*level) + 2` | same | `rnd(50+10*level)+2` | SAME |
| Position | `rnd_pos` anywhere in the room (can be a spot where a monster or later object also goes) | `find_floor(rp, ..., FALSE, FALSE)`: a FLOOR (or PASSAGE in a maze) square | `findFloor(room, 0, isFree)`: empty non-special floor | SAME as 5.4 (3.6 vs 5.4: tiny placement difference) |
| Room remembers gold | `r_goldval`, `r_gold`; used for monster chance and greedy monsters | same | not stored; gold is just an item | DIFFERENT (affects monster chance, see section 6) |
| Gold in monster packs | no (only items via `new_thing`) | no | monsters may carry gold (`def.Gold`) or an item, see section 6 | rx1-only |

## 4. Objects per level

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Attempts | `MAXOBJ = 9` attempts, each `rnd(100) < 35` (mean 3.15 objects) (`newlevel.c:put_things`) | `MAXOBJ = 9`, each `rnd(100) < 36` (mean 3.24) (`new_level.c:put_things`) | `rogueMaxObj*scale` attempts, each `rnd(100) < 36`; scale = max(1, rooms/9), so 1 on Rogue-grid levels | SAME as 5.4. DIFFERENT from 3.6 (35 vs 36). 3.6 vs 5.4 differ |
| Placement | random room (`rnd_room`) and random point until FLOOR | `find_floor(NULL, ..., FALSE, FALSE)`: random existing room, FLOOR square | random room, random floor tile, `isFree` | SAME |
| Item type table (`things[]`, `new_thing`) | potion 27, scroll 27, food 18, weapon 9, armor 9, ring 5, stick 5 (sum 100) (`init.c`) | potion 26, scroll 36, food 16, weapon 7, armor 7, ring 4, stick 4 (sum 100) (`extern.c`) | `rogueNewThing` weights 26/36/16/7/7/4/4 (the file previously had scroll 33; working tree now has 36) | SAME as 5.4. DIFFERENT from 3.6 |
| Forced food | `no_food` counts levels since food; `no_food++` in every `new_level`; `no_food > 3` forces food; reset to 0 when food is made | same | `levelsWithoutFood++` in `gotoLevel` (only for newly generated levels, not revisits); `<= 3` rolls normally else food; reset on food | SAME (revisits are not regenerated in rx1) |
| Ration vs mango | `rnd(10) != 0` ration | same | by item chance in `food.rec` | not compared here |
| Curses and blessings | by item type in `new_thing` (weapon 10% cursed -1..-3, 5% +1..+3; armor 20%/8%; etc.) | same | `rogueNewThing` copies the same rates (weapon 10/5, armor 20/8, stat rings, teleport ring), cursed means stuck | SAME |
| Treasure room | none | `rnd(TREAS_ROOM=20) == 0` per level (5%): a random room gets `rnd(spots)+2` items where `spots = min((h-2)*(w-2)-2, 8)` (so 2..9 items), then guard monsters from level+1: `max(rnd(spots)+2, items+2)`, capped by room squares, `ISMEAN` ("no sloughers") (`treas_room`) | same numbers in `rogueTreasureRoom` (items 2..9, the code comment says 2-10 but the math gives 2..9 as in C); monsters from `level+1`, asleep, `FlagMean`; secret level always 3 treasure rooms | SAME as 5.4. 3.6 has none (3.6 vs 5.4 differ) |
| No new loot after amulet | `if (amulet && level < max_level) return` | same | `noNewStuff = hasAmulet && level < deepest` (checked on generation only; visited levels persist unchanged) | SAME |
| Lights | no lights | no lights | `rollLights` once per new deepest level: torch 30%, lantern or brass lantern 21% from level 5, star glass (once, level > 8, 10%), morning star (once, level > 10, 5%) | rx1-only |
| Lore documents | none | none | 1-2 unused lore documents per level | rx1-only |
| Item stacking of shop/town | none | none | town vendors (section 12) | rx1-only |

## 5. Stairs and the player's arrival

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Down stairs | one `%` on a random FLOOR square of a random existing room, placed after objects and traps (`new_level`) | `find_floor(NULL, &stairs, FALSE, FALSE)` same place in order; `seenstairs = FALSE` | `placeStairs`: random room, random floor, placed during generation (before objects; objects avoid the tile via `isFree`) | SAME (effect) |
| Up stairs | none: the same `%` tile is used with `<` | same | a second, separate "stairs up" tile on another random floor tile (never the same tile as down) | rx1-only |
| Taking stairs down | `d_level`: `winat == STAIRS`, `level++`, `new_level()` | `d_level`: `chat == STAIRS`, `levit_check()` first (levitating blocks), `level++` | `PlayerTryDescend` on a "stairs down" tile; no levitation check | DIFFERENT from 5.4 (no levitation rule). 3.6 vs 5.4 differ |
| Taking stairs up | works only with the Amulet; else "I see no way up." (3.6) / "your way is magically blocked" (5.4); message "wrenching sensation in your gut" on success | same, plus `levit_check` | up stairs always work; no Amulet needed; level 1 up goes to town | DIFFERENT |
| Arrival position | hero is put at a random FLOOR square of a random room (`new_level`), never on the stairs | `find_floor(NULL,&hero,FALSE,TRUE)`: random free floor square | arrives standing on the matching stairs (down arrives on up stairs, up arrives on down stairs); a monster on the stairs moves the player next to them; trap door, chasm and teleport use a random spawn position | DIFFERENT (rx1-only stair arrival) |
| Dark-room start | `light(&hero)` | `enter_room` | normal FOV/light | n/a |
| Stairs under traps | traps need FLOOR so no overlap | trap chosen first, then stairs may be placed on a trap's floor square (hidden trap data stays in the flags) | traps and stairs never share a tile | minor DIFFERENT |
| Mimic disguise | `M` can look like stairs (`mch = STAIRS`) or amulet at level > 25 | xeroc `X` disguises as `rnd_thing()` (no stairs) | mimic/xeroc disguised as a random item category | not generation, noted |

## 6. Monsters at generation

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Monster per room at generation | in every non-gone room after the gold: `rnd(100) < (goldval > 0 ? 80 : 25)`, a monster from `randmonster(FALSE)` at a FLOOR square (`rooms.c:do_rooms`) | same, `find_floor(rp,&mp,FALSE,TRUE)` (free, steppable square) | NONE. `spawnEntities` comment: "monsters spawn via wandering daemon during play". Only treasure room, special room and zoo guards exist at generation | DIFFERENT (major): a new rx1 level starts without room monsters |
| Pack for placed monster | `rnd(100) < m_carry` gives `new_thing()` always | `give_pack`: only if `level >= max_level` and `rnd(100) < m_carry` | `NewEnemyFromDef` for every monster (also wanderers, summons): `Intn(100) < CarryChance` rolled twice in a row (so chance squared), then a second branch gives either a `rogueNewThing` item or gold from `def.Gold` | DIFFERENT |
| Monster choice (`randmonster`) | `d = level + rnd(10) - 5`; `d < 1` gives `rnd(5)+1`; `d > 26` gives `rnd(5)+22`; table `lvl_mons = "KJBSHEAOZGLCRQNYTWFIXUMVDP"` (1-based slot) | `d = level + rnd(10) - 6` (0-based, same range level-5..level+4 in 1-based terms); `d<0` gives `rnd(5)`; `d>25` gives `rnd(5)+21`; table `K E B S H I R O Z L C Q A N Y F T W P X U M V G J D` | `d = level + rnd(10) - 5`, same clamps (1..26, `d>26` gives `rnd(5)+22`); then a random monster among ALL monsters whose `dlvl` field equals d (union of both rosters, 41 monsters, one `dlvl` each; `xeroc_2` excluded); retry up to 100 times if the slot is empty | DIFFERENT roster/table, SAME distribution of the target slot. Slots with several monsters (e.g. dlvl 6: floating eye, ice monster, slime) pick uniformly among them, so rarer than in Rogue where each slot has exactly one monster |
| Wanderer table | `wand_mons = "KJBSH AOZG CRQ Y W IXU V  "`: blanks are E (floating eye), L, N, T (troll), F, M, D, P | `wand_mons`: blanks at I (ice monster), L, N, F (venus flytrap), X, D | `nonWanderers` = ice_monster, leprechaun, nymph, venus_flytrap, xeroc, dragon, floating_eye, violet_fungi, mimic, purple_worm (union of both blank sets, except the troll which wanders as in 5.4); an excluded slot is retried | DIFFERENT (3.6 vs 5.4 also differ: troll is non-wandering in 3.6) |
| Level scaling past 26 | none (3.6 has no `lev_add`) | `lev_add = level - 26` (min 0): +lev_add level, -lev_add armor, +10 exp per level; `level > 29` makes monsters `ISHASTE` | `levAdd = max(0, level-26)` for level, armor and exp (`NewEnemyFromDef`); no haste at level > 29 | SAME as 5.4 except missing haste. DIFFERENT from 3.6 |
| Hit points | `roll(lvl, 8)` | `roll(lvl, 8)` (also `s_maxhp`) | `NewDice(lvl,8,0).Roll()` | SAME |
| Monster sleep | non-ISRUN monsters sleep until woken; no special rule | same; `wake_monster` wakes mean monsters 2 of 3 times when seen | placed treasure/special room monsters get `FlagSleep` | SAME in intent |
| Greedy monsters | `ISGREED` guard `r_gold` | same (`proom->r_goldval`) | greedy move toward gold (handover notes) | n/a (monster domain) |
| Level 1-2 monster speed | none | none | `enemyMovement`: on dungeon levels 1 and 2 monsters get half the energy (`playerTimeSpent /= 2`) | rx1-only |

## 7. Wandering monsters

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Start | `fuse(swander, 0, WANDERTIME=70, AFTER)` once at game start (`main.c`) | `WANDERTIME = spread(70)` = 67..73 (`spread(nm) = nm - nm/20 + rnd(nm/10)`) | no initial delay | DIFFERENT (3.6 vs 5.4: fixed 70 vs 67..73) |
| Roll | after the fuse `swander` starts daemon `rollwand`: every 4th turn `roll(1,6) == 4` (1 in 6) (`daemons.c:rollwand`) | same | every 4th player turn `rand.Intn(6) == 4` (1 in 6) (`wanderingMonsterTick`) | SAME roll |
| After a spawn | `kill_daemon(rollwand); fuse(swander, 0, WANDERTIME, BEFORE)`: no new roll until another 70 turns pass. Average gap about 70 + 24 = 94 turns | same | no cooldown: the 1-in-6-per-4-turns roll never stops; average gap about 24 turns | DIFFERENT (about 4x more wanderers in rx1) |
| The timer carries across levels | yes (not reset by `new_level`) | yes | the 4-turn counter is global and also carried | SAME |
| Confusion side effect | `unconfuse` restarts `rollwand` | same | n/a | n/a |
| Where | random existing room different from the hero's room (`rnd_room`, `rnd_pos`, `step_ok`); if hero is in a corridor any room qualifies | `find_floor(NULL, &cp, FALSE, TRUE)` until not in `proom` (or a monster is there); gives up after 500 tries | a random room that is not the player's (10 tries), a free floor tile of it (10 tries); nothing spawns when the player is not inside a room (corridors, town) | DIFFERENT (rx1 spawns nothing while the player is in a corridor) |
| What | `randmonster(TRUE)` of current level | same | `rogueRandMonster(level, wander=true)` | SAME rule, different roster |
| Behaviour | `ISRUN`, `t_dest = &hero` (chases at once) | `runto` | `SetAware()` | SAME |
| Message | none (wizard only) | none (wizard only) | "You sense a X stirring in the dungeon" | rx1-only |
| Wanderer pack | none | none (only `give_pack` at room generation) | yes, see section 6 | DIFFERENT |

## 8. Traps

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Chance a level has traps | `rnd(10) < level`: level 1: 10%, level 10 and deeper: 100% (`newlevel.c`) | same | `rnd(10) < level` | SAME |
| Count | `ntraps = rnd(level/4) + 1`, max `MAXTRAPS = 10` | same | `min(rnd(level/4)+1, 10) * scale` (scale 1 on Rogue-grid levels, larger on big levels) | SAME |
| Placement | random existing room, random square, must be FLOOR with no monster or object (`winat == FLOOR`) | `find_floor(NULL,...)` retry while the square is not FLOOR and real; so a trap never lands on an object, stairs not yet placed; not in mazes | `findFloor(nil,0,isFree)`: free floor tile in a random room | SAME (effect) |
| Types | 6 equally likely (`rnd(6)`): trap door, bear trap, sleeping gas, arrow, teleport, poison dart | 8 equally likely (`rnd(NTRAPS=8)`): trap door, arrow, sleep, bear, teleport, dart, rust (water gush rusts armor), mystery (11 random flavour messages) | 7 equally likely: exploding, slow, teleport, dart, arrow, descend (trap door), bear (`GetAllTrapCategories`) | DIFFERENT. No sleep, rust or mystery trap; extra exploding and slow traps |
| Hidden by default | placed with `^` on the hidden layer; `tr_flags = 0` (no ISFOUND) | `F_REAL` cleared: looks like floor | `SetHidden(true)` always; drawn dimmer when known by detection | SAME concept |
| Triggered by stepping on it | yes, hidden or not; becomes visible (`ISFOUND`) and stays | yes, becomes visible (`F_SEEN`); levitating hero is never trapped (`ISLEVIT`) | yes; but the trap fires once and is then removed (`isAlive=false`, removed at end of turn); no levitation check (`FlagFly` ignored) | DIFFERENT: Rogue traps are permanent |
| Monsters and traps | monsters ignore traps | ignore | monsters trigger traps too (`triggerTileEffectsAfterMovement` for any actor; a descend trap kills the monster) | rx1-only |
| Trap door | `level++; new_level()`, arrive at random place | same | `force_descend_target` then `descendToRandomLocation`; the level below is generated or reloaded and the player lands on a random spawn tile | SAME effect |
| Bear trap | `no_move += BEARTIME(3)` | same | hold for `rand.Intn(10)+5` (5..14) turns | DIFFERENT numbers |
| Sleep trap | `no_command += SLEEPTIME(5)` | same | none | DIFFERENT |
| Arrow trap | `swing(lvl-1, AC, 1)` to hit, 1d6 damage, else the arrow falls to the floor | same | `magic_arrow`: always a hit on whatever stands there, damage is the arrow's thrown damage `1d1`, an arrow item is dropped | DIFFERENT |
| Dart trap | `swing(lvl+1, AC, 1)`, 1d4, lose 1 Str unless sustain strength (5.4 also allows a poison save) | same | `magic_dart`: thrown damage of dart `1d3`, no hit roll, no Str loss | DIFFERENT |
| Teleport trap | `teleport()` random | same | teleport away | SAME |
| Exploding trap | none | none | `explode`: 5 damage in radius 3 | rx1-only |
| Slow trap | none | none | `slow_target` | rx1-only |
| Passive discovery | none | none | entering a lit room that was last seen more than 50 turns ago: each hidden trap has 1 in 20 chance to be found, and 1 in 3 per hidden trap gives "you feel like something is wrong with this room" (`checkTilesForHiddenObjects`) | rx1-only |
| Detection | none (no trap detection item; P_TFIND is magic detection) | none | scroll of trap detection (`detect_traps`, 8..15 turns `FlagSeeTraps`) | rx1-only |

## 9. Searching and secrets

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Command | `s` searches the 8 neighbours (and the hero's square) once (`command.c:search`); ring of searching searches every turn | `s`, same; hero square skipped; ring of searching every turn | `Wait` (`.`) always searches (no separate rest/search); no ring of searching | DIFFERENT (no ring; rest = search) |
| Secret door | `rnd(100) < 20` (1 in 5) per neighbour | `rnd(5 + probinc) == 0` (1 in 5) | `rand.Intn(5+probinc) == 0` | SAME |
| Hidden trap | `rnd(100) > 50` gives up, so found with 51% | `rnd(2 + probinc) == 0` (50%) | `rand.Intn(2+probinc) == 0` | SAME as 5.4 (3.6 is 51%) |
| Secret corridor square | none (no secret passages in 3.6) | `rnd(3 + probinc) == 0` (1 in 3) | `Intn(3+probinc) == 0` | SAME as 5.4 |
| Blind / hallucinating | blind hero cannot search at all (`if (on(player, ISBLIND)) return`); no hallucination effect | `probinc = 3` when hallucinating, `+2` when blind (harder, not impossible); hallucination also gives a random trap name | same `probinc` (+3 / +2) | SAME as 5.4. 3.6 vs 5.4 differ |
| Messages | trap name only; secret door silent | "a secret door", "you found <trap>" | "You found a secret door / passage / <trap>" | cosmetic |
| Stepping into a secret door | secret door is impassable until found | same | impassable (shows wall) until found | SAME |
| Secret stairs | none | none | hidden stairs on the secret-level depth are found by walking over the tile or by searching next to it (odds 3, like a passage) | rx1-only |
| Magic mapping | 3.6 scroll maps the level and turns `SECRETDOOR` into doors | 5.4 turns secret doors into doors and secret passage squares into passages | `reveal_map` explores the whole map but leaves secrets (doors, passages, hidden stairs) hidden; hidden traps stay hidden | DIFFERENT |

## 10. Amulet level, depth, winning

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Amulet appears | `level > 25 && !amulet` (`put_things`): one amulet on every generated level 26+ until picked up | `level >= AMULETLEVEL (26) && !amulet` | `level >= 26 && !hasAmulet` | SAME |
| Amulet flag | `amulet` set at pickup, never cleared | same | derived from the inventory (`HasItemWithName("amulet_of_yendor")`); dropping it clears the effect | minor DIFFERENT |
| Maximum depth | none; 26+ is allowed, monsters clamp to slots 22..26 | none | none (`maximumDungeonLevel = 26` only sizes the style plan and wizard menu); levels beyond 26 are Rogue-grid style | SAME |
| Going back up | with the amulet only, each level is regenerated (`level--; new_level()`), no new objects when `level < max_level` | same | free; levels persist | DIFFERENT (see section 11) |
| Win condition | `u_level` at level 1 with the amulet: `level--` gives 0, `total_winner()` | same (`level == 0`) | `PlayerTryAscend` on level 1 with the amulet calls `gameWon()` ("ESCAPED the dungeon") instead of going to town | SAME trigger |
| Win score | `total_winner` adds the value of each pack item (weapons, armor, rings, wands, scrolls, potions, amulet 1000) to the purse | same idea, using `oi_worth` tables | score is gold only (`calculateTotalNetWorth`) | DIFFERENT |
| Starting depth | level 1 | level 1 | town (level 0), stairs down to level 1 | rx1-only |

## 11. Level persistence, revisits, falling

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Persistence | none: every `new_level` frees the old objects and monsters and builds a new level, going up or down | same | a visited dungeon level is kept as it was left: map, monsters, items, explored area, secrets, glow (`leaveLevel`, `visitedLevel`, `returnToLevel`), so no stair scumming; keyed by level number and secret flag | DIFFERENT (rx1-only persistence) |
| What revisit does | n/a | n/a | restores; the player is put on the matching stairs, or beside them if a monster stands there; `levelsWithoutFood` is not advanced; wanderers and deaths during an absence are not simulated | rx1-only |
| Loot regeneration with amulet | `amulet && level < max_level` gives no new objects | same | irrelevant for kept levels; a never-visited skipped level gets `noNewStuff` | SAME for fresh levels |
| Fall (trap door) | one level, random square | same | `descendToRandomLocation`; also Brogue chasm tiles drop one level; monsters falling are killed | SAME for the player; chasms rx1-only |
| Teleportation | random square on the same level | same | random spawn position | SAME |
| Level feelings | none | none | none. The only flavour messages: "You feel a cold draft from below." on entering the level with the hidden stairs, "you feel like something is wrong with this room" | rx1-only messages |

## 12. Town (rx1-only)

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Town | none | none | level 0 loaded from `data_rx1/prefabs/town.txt` (80x23 text map, mountain border, buildings `#`, vendors `1`..`4`, stairs `>`, start `@`); everything explored and lit; no monsters and no wandering monsters (they need a room); reloaded every visit | rx1-only |
| Vendors | none | none | `1` curator (buys/sells lore documents), `2` blacksmith (rusty -1 weapons and armor: dagger, mace, spear, long sword, short bow, leather armor, ring mail, scale mail), `3` general store (torch, lantern, food ration, arrows, bolts, darts, potion of life), `4` home (stash, 99 slots, kept for the whole game) | rx1-only |
| Prices | n/a | n/a | flat per kind in `itemPrice` (armor 80, weapon 50, document 40, light 30, food 10, missile 2, potion of life 500, other 25); sells at half | rx1-only |
| Stairs | n/a | n/a | level 1 up stairs lead to town and arrive on the town's stairs down; level 1 up stairs with the amulet win the game | rx1-only |

## 13. Special rooms and vaults (rx1-only)

`game/special_rooms.go` ports NetHack `mkroom.c`. After the level is populated, one room at most becomes special (`makeSpecialRoom`). The chain is tried in order and the first hit ends it, even if no room fits:

| Kind | Needs level > | Chance | Contents |
|---|---|---|---|
| Leprechaun hall | 5 | 1 in 8 | leprechauns, plus gold |
| Zoo | 6 | 1 in 7 | random monsters (`rogueRandMonster`), plus gold `10 + rnd(5*level)` per monster |
| Morgue | 11 | 1 in 6 | zombie, wraith (40%), vampire (if `rn2(level) > 8` and roll > 85) |
| Ant hole | 12 | 1 in 8 | ants |

- Room choice (`pickSpecialRoom`): a walled room (not a maze or cave room), not the room holding the player, never one with the up stairs, a room with the down stairs only 1 time in 3, preferring rooms with exactly one door (else a 1-in-5 pass).
- Stocking: each floor tile 50%, never within 1 tile of a door, at most 25 monsters, all asleep.
- Not ported: court, beehive, barracks, swamp, cockatrice nest, temple, shops.
- Rogue 5.4's only special room is the treasure room (section 4), and rx1 also has it. Rogue 3.6 has no special room.
- The test prefab `vault_test.txt` and `line_room.txt` are wizard-only test maps.

---

## 14. Differences between Rogue 3.6 and 5.4 in this domain

1. Room cell size: 3.6 uses the terminal's `COLS/3` x `LINES/3`; 5.4 fixes 80x24 (26x8).
2. Maze rooms (5.4 only): dark room with 1 in 15 chance (`do_maze`, `dig`).
3. Secret corridor squares (5.4 only): `rnd(10)+1 < level && rnd(40)==0` per square. 3.6 has only secret doors (same 1-in-5 probability in both).
4. Treasure rooms (5.4 only): 1 in 20 levels.
5. Object attempts succeed 35% (3.6) vs 36% (5.4); item type weights differ (3.6: 27/27/18/9/9/5/5, 5.4: 26/36/16/7/7/4/4).
6. Trap types: 3.6 has 6 (trap door, bear, sleep, arrow, teleport, dart); 5.4 has 8 (adds rust and mystery) and levitation avoids traps. Dart trap save vs poison is 5.4 only.
7. Search: 3.6 trap 51%, blind cannot search, no secret passages; 5.4 trap 50%, `probinc` +2 blind and +3 hallucinating, secret passages 1 in 3, hero square skipped.
8. `WANDERTIME`: 3.6 fixed 70, 5.4 `spread(70)` (67..73). Wanderer placement: 3.6 any room but the hero's; 5.4 `find_floor` loop with a 500-try cap.
9. Monster tables (letters and order), wanderer blanks, and 5.4 level scaling past level 26 plus haste past 29.
10. Monster packs: 3.6 attaches `new_thing()` on the `m_carry` roll; 5.4 requires `level >= max_level` (`give_pack`).
11. Gold placement: 3.6 `rnd_pos`; 5.4 `find_floor`.
12. Stairs up: 3.6 `winat`, no levitation check; 5.4 `chat` plus `levit_check` and a different "magically blocked" message.
13. 3.6 mimics can pretend to be stairs or the amulet; 5.4 xeroc pretends to be a random item.
14. 5.4 numbers passages (`passnum`) for vision; 3.6 does not.

## 15. Summary of the main differences between rx1 and Rogue

- rx1 builds the level, doors, secret doors, gold, object counts, treasure room, gone and dark rooms, extra passages and secret passage squares exactly like 5.4's formulas (SAME), and the search odds (door 1/5, passage 1/3, trap 1/2) too.
- rx1 places no monsters in rooms at level creation (Rogue: 80% of gold rooms and 25% of the others). Population comes from wanderers, treasure and special rooms only.
- Wandering monsters have no 70-turn cooldown after a spawn and no start delay: roughly one every 24 turns instead of every 94. They never spawn while the player is in a corridor and they announce themselves.
- Levels persist, with separate up and down stairs, arrival on the stairs, free ascent, and a town at level 0; Rogue regenerates every level and goes up only with the amulet, arriving at a random spot.
- Level styles: Rogue grid on levels 1-2, then a weighted mix of NetHack rooms, caves, mazes and Brogue levels (all five appear in a run); rooms sizes in the Rogue style use a 7-row cell on the 80x23 map.
- Traps: 7 types (no sleep, rust or mystery; extra exploding and slow); they fire once and vanish; monsters trigger them; no levitation check; damage and durations differ from Rogue.
- Monster choice uses the same `level+rnd(10)-5` curve but over a merged roster of both Rogues by `dlvl` slot; wander exclusions are the union of both blank sets; no haste past level 29; monster packs use a squared carry roll and may hold gold.
- Win: going up from level 1 with the amulet ends the game (same trigger), but the score is gold only.
- rx1-only extras: secret level (depth 7-12) with mega dungeon, special rooms from NetHack, lights, lore documents, scroll of trap detection, passive trap hints in lit rooms, monsters at half speed on levels 1-2; no ring of searching; magic mapping does not reveal secrets.
