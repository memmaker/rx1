# Monsters: Rogue 3.6 vs Rogue 5.4 vs rx1

Scope: everything about monsters. Sources read in full: `/Users/felix/Games/rogue3.6/{init,monsters,chase,fight,daemons,rooms,move,main,command,newlevel}.c`,
`/Users/felix/Games/rogue5.4/{extern,monsters,chase,fight,daemons,rooms,new_level,move,misc,sticks,command,rogue.h}`, and the rx1 working tree
(uncommitted changes included): `game/{data_monsters,spawn_rogue,state,ai,abilities,actor,actions,effects_hit,effects_zap,deamons,special_rooms}.go`, `rpg/rogue.go`,
`data_rx1/definitions/monsters.rec`.

Verdict key: SAME = identical rule/number, DIFFERENT = rule exists in rx1 but differs, rx1-only = rx1 invented it, MISSING = Rogue has it, rx1 does not.
"3.6 vs 5.4" is flagged where the two Rogues disagree. `rnd(n)` = 0..n-1 in all three. `roll(n,s)` = n dice of s sides.

A note on which "3.6" this is: it is the Berkeley 3.6 source, which has 26 monsters (`init.c: monsters[26]`), no dragon breath, no treasure rooms, no monster level scaling,
no `exp_add`, no ISFLY. Its damage strings use `NdS`; 5.4 uses `NxS` (`roll_em` splits on `x`).

---

## 1. Roster: which monster exists in which game

| In | Monsters |
|---|---|
| All three (13) | bat, centaur, dragon, hobgoblin, leprechaun, nymph, orc, snake, troll, vampire, wraith, yeti, zombie |
| 3.6 + rx1 only (13) | giant ant, floating eye, violet fungi, gnome, invisible stalker, jackal, kobold, mimic, purple worm, quasit, rust monster, umber hulk, xorn |
| 5.4 + rx1 only (13) | aquator, emu, venus flytrap, griffin, ice monster, jabberwock, kestrel, medusa, phantom, quagga, rattlesnake, black unicorn, xeroc |
| rx1-only (3) | slime, ur-vile, xeroc mk ii (wizard-menu test monster, never spawns: `rogueRandMonster` skips `xeroc_2`) |

rx1 therefore has 42 definitions (39 from the union of both Rogues + 3 own). Rogue has 26 per game, one per letter A..Z. rx1 is keyed by `internal_name`, not by letter
(several monsters share a letter: e/i/s/r/f/x/U/g...).

## 2. Side-by-side monster table

Columns for Rogue: `letter; level; base exp; armor class (lower = better); damage per attack ("/" separates attacks); carry %; flags`.
Rogue hp is always `roll(level, 8)` (no hp column). Rogue `exp` here is the table value; the bonus formula is in section 4.
rx1 adds `dlvl` (depth slot, section 3), `gold` dice and its own `carry` %. The verdict compares level / exp / armor / damage / mean-flag against the
Rogue version(s) that have the monster. Carry % and gold differ for every monster (rx1 re-tuned all of them, see 5.9) and are not repeated in the verdict.

| Monster | Rogue 3.6 | Rogue 5.4 | rx1 (`monsters.rec`) | Verdict |
|---|---|---|---|---|
| bat | B; L1; x1; AC3; 1d2; carry 0%; no flags | B; L1; x1; AC3; 1d2; carry 0%; fly | b; L1; x1; AC3; 1d2; dlvl 3; gold 0; carry 1%; flags: flying chase erratic | SAME stats |
| emu | - | E; L1; x2; AC7; 1d2; carry 0%; mean | e; L1; x2; AC7; 1d2; dlvl 2; gold 0; carry 1%; flags: mean chase | SAME stats |
| hobgoblin | H; L1; x3; AC5; 1d8; carry 0%; mean | H; L1; x3; AC5; 1d8; carry 0%; mean | h; L1; x3; AC5; 1d8; dlvl 5; gold 5d4; carry 35%; flags: mean group | SAME stats |
| kestrel | - | K; L1; x1; AC7; 1d4; carry 0%; mean,fly | k; L1; x1; AC7; 1d4; dlvl 1; gold 0; carry 2%; flags: chase | DIFF: flag mean: Rogue 5.4 has it, rx1 not |
| jackal | J; L1; x2; AC7; 1d2; carry 0%; mean | - | j; L1; x2; AC7; 1d2; dlvl 2; gold 0; carry 10%; flags: chase group | DIFF: flag mean: Rogue 3.6 has it, rx1 not |
| floating eye | E; L1; x5; AC9; 0d0; carry 0%; no flags | - | e; L1; x5; AC9; 0d0; dlvl 6; gold 0; carry 25%; flags: stationary | SAME stats |
| ice monster | - | I; L1; x5; AC9; 0d0; carry 0%; no flags | i; L1; x5; AC9; 0d0; dlvl 6; gold 1d6; carry 20%; flags: mean | DIFF: flag mean: rx1 only |
| slime | - | - | s; L2; x10; AC8; 1d3; dlvl 6; gold 3d6; carry 20%; flags: - | rx1-only |
| snake | S; L1; x3; AC5; 1d3; carry 0%; mean | S; L1; x2; AC5; 1d3; carry 0%; mean | s; L1; x3; AC5; 1d3; dlvl 4; gold 0; carry 10%; flags: - | DIFF: exp: rx1 3, 3.6 3, 5.4 2; flag mean: Rogue 3.6/5.4 has it, rx1 not |
| gnome | G; L1; x8; AC5; 1d6; carry 10%; no flags | - | g; L1; x8; AC5; 1d6; dlvl 10; gold 5d5; carry 35%; flags: tunnel | SAME stats |
| orc | O; L1; x5; AC6; 1d8; carry 15%; block | O; L1; x5; AC6; 1d8; carry 15%; greedy | o; L1; x5; AC6; 1d8; dlvl 8; gold 1d6; carry 35%; flags: greedy | SAME stats |
| rattlesnake | - | R; L2; x9; AC3; 1d6; carry 0%; mean | r; L2; x9; AC3; 1d6; dlvl 7; gold 0; carry 3%; flags: - | DIFF: flag mean: Rogue 5.4 has it, rx1 not |
| invisible stalker | I; L8; x120; AC3; 4d4; carry 0%; invis | - | i; L8; x120; AC3; 4d4; dlvl 20; gold 1d8; carry 25%; flags: chase invisible erratic | SAME stats |
| rust monster | R; L5; x25; AC2; 0d0/0d0; carry 0%; mean | - | r; L5; x20; AC2; 0d0/0d0; dlvl 13; gold 2d20; carry 65%; flags: - | DIFF: exp: rx1 20, 3.6 25; flag mean: Rogue 3.6 has it, rx1 not |
| zombie | Z; L2; x7; AC8; 1d8; carry 0%; mean | Z; L2; x6; AC8; 1d8; carry 0%; mean | z; L2; x7; AC8; 1d8; dlvl 9; gold 1d100; carry 85%; flags: chase slow | DIFF: exp: rx1 7, 3.6 7, 5.4 6; flag mean: Rogue 3.6/5.4 has it, rx1 not |
| giant ant | A; L2; x10; AC3; 1d6; carry 0%; mean | - | a; L2; x10; AC3; 1d6; dlvl 7; gold 2d4; carry 10%; flags: - | DIFF: flag mean: Rogue 3.6 has it, rx1 not |
| leprechaun | L; L3; x10; AC8; 1d1; carry 0%; no flags | L; L3; x10; AC8; 1d1; carry 0%; no flags | l; L3; x10; AC8; 1d1; dlvl 10; gold 5d10; carry 50%; flags: chase | SAME stats |
| centaur | C; L4; x15; AC4; 1d6/1d6; carry 15%; no flags | C; L4; x17; AC4; 1d2/1d5/1d5; carry 15%; no flags | c; L4; x15; AC4; 1d6/1d6; dlvl 11; gold 1d10; carry 15%; flags: - | DIFF: exp: rx1 15, 3.6 15, 5.4 17; damage: rx1 1d6/1d6 = 3.6 (3.6 1d6/1d6, 5.4 1d2/1d5/1d5) |
| kobold | K; L1; x1; AC7; 1d4; carry 0%; mean | - | k; L1; x1; AC7; 1d4; dlvl 1; gold 2d8; carry 55%; flags: - | DIFF: flag mean: Rogue 3.6 has it, rx1 not |
| aquator | - | A; L5; x20; AC2; 0d0/0d0; carry 0%; mean | A; L5; x20; AC2; 0d0/0d0; dlvl 13; gold 2d12; carry 50%; flags: mean | SAME stats |
| black unicorn | - | U; L7; x190; AC-2; 1d9/1d9/2d9; carry 0%; mean | U; L7; x190; AC-2; 1d9/1d9/2d9; dlvl 12; gold 1d1000; carry 90%; flags: - | DIFF: flag mean: Rogue 5.4 has it, rx1 not |
| quagga | - | Q; L3; x15; AC3; 1d5/1d5; carry 0%; mean | q; L3; x15; AC3; 1d5/1d5; dlvl 12; gold 0; carry 3%; flags: - | DIFF: flag mean: Rogue 5.4 has it, rx1 not |
| nymph | N; L3; x40; AC9; 0d0; carry 100%; no flags | N; L3; x37; AC9; 0d0; carry 100%; no flags | n; L3; x37; AC9; 0d0; dlvl 14; gold 2d20; carry 60%; flags: chase | DIFF: exp: rx1 37, 3.6 40, 5.4 37 |
| mimic | M; L7; x140; AC7; 3d4; carry 30%; no flags | - | m; L7; x100; AC7; 3d4; dlvl 23; gold 4d4; carry 25%; flags: disguised | DIFF: exp: rx1 100, 3.6 140 |
| yeti | Y; L4; x50; AC6; 1d6/1d6; carry 30%; no flags | Y; L4; x50; AC6; 1d6/1d6; carry 30%; no flags | Y; L4; x50; AC6; 1d6/1d6; dlvl 15; gold 4d20; carry 35%; flags: haste | SAME stats |
| troll | T; L6; x55; AC4; 1d8/1d8/2d6; carry 50%; regen,mean | T; L6; x120; AC4; 1d8/1d8/2d6; carry 50%; regen,mean | T; L6; x120; AC4; 1d8/1d8/2d6; dlvl 17; gold 2d10; carry 55%; flags: regenerating revive | DIFF: exp: rx1 120, 3.6 55, 5.4 120; flag mean: Rogue 3.6/5.4 has it, rx1 not |
| violet fungi | F; L8; x85; AC3; 000d0; carry 0%; mean | - | f; L3; x85; AC3; 0d0; dlvl 19; gold 3d10; carry 55%; flags: stationary | DIFF: level: rx1 3, 3.6 8; flag mean: Rogue 3.6 has it, rx1 not |
| wraith | W; L5; x55; AC4; 1d6; carry 0%; no flags | W; L5; x55; AC4; 1d6; carry 0%; no flags | w; L5; x55; AC4; 1d6; dlvl 18; gold 0; carry 55%; flags: - | SAME stats |
| venus flytrap | - | F; L8; x80; AC3; %%%x0; carry 0%; mean | f; L8; x80; AC3; 0d0; dlvl 16; gold 2d20; carry 60%; flags: mean stationary | DIFF: damage: rx1 0d0, 5.4 0x0 |
| phantom | - | P; L8; x120; AC3; 4d4; carry 0%; invis | p; L8; x120; AC3; 4d4; dlvl 19; gold 0; carry 20%; flags: invisible chase wall_crawler erratic | SAME stats |
| purple worm | P; L15; x7000; AC6; 2d12/2d4; carry 70%; no flags | - | P; L15; x4000; AC6; 2d12/2d4; dlvl 26; gold 2d80; carry 90%; flags: tunnel | DIFF: exp: rx1 4000, 3.6 7000 |
| ur-vile | - | - | u; L7; x190; AC-2; 1d9/1d9/2d9; dlvl 21; gold 0; carry 45%; flags: group | rx1-only |
| umber hulk | U; L8; x130; AC2; 3d4/3d4/2d5; carry 40%; mean | - | U; L8; x200; AC2; 3d4/3d4/2d5; dlvl 22; gold 2d10; carry 65%; flags: tunnel | DIFF: exp: rx1 200, 3.6 130; flag mean: Rogue 3.6 has it, rx1 not |
| griffin | - | G; L13; x2000; AC2; 4d3/3d5; carry 20%; mean,fly,regen | g; L13; x2000; AC2; 4d3/3d5; dlvl 24; gold 3d10; carry 45%; flags: mean flying regenerating | SAME stats |
| medusa | - | M; L8; x200; AC2; 3d4/3d4/2d5; carry 40%; mean | M; L8; x200; AC2; 3d4/3d4/2d5; dlvl 22; gold 1d20; carry 40%; flags: - | DIFF: flag mean: Rogue 5.4 has it, rx1 not |
| quasit | Q; L3; x35; AC2; 1d2/1d2/1d4; carry 30%; mean | - | Q; L3; x32; AC2; 1d2/1d2/1d4; dlvl 14; gold 2d60; carry 75%; flags: - | DIFF: exp: rx1 32, 3.6 35; flag mean: Rogue 3.6 has it, rx1 not |
| xeroc | - | X; L7; x100; AC7; 4d4; carry 30%; no flags | x; L7; x100; AC7; 4d4; dlvl 20; gold 2d8; carry 25%; flags: disguised | SAME stats |
| xeroc mk ii | - | - | x; L7; x100; AC7; 4d4; dlvl 20; gold 2d8; carry 20%; flags: invisible wall_crawler | rx1-only |
| xorn | X; L7; x120; AC-2; 1d3/1d3/1d3/4d6; carry 0%; mean | - | X; L7; x120; AC-2; 1d3/1d3/1d3/4d6; dlvl 21; gold 0; carry 65%; flags: wall_crawler | DIFF: flag mean: Rogue 3.6 has it, rx1 not |
| vampire | V; L8; x380; AC1; 1d10; carry 20%; regen,mean | V; L8; x350; AC1; 1d10; carry 20%; regen,mean | V; L8; x350; AC1; 1d10; dlvl 23; gold 0; carry 75%; flags: chase regenerating | DIFF: exp: rx1 350, 3.6 380, 5.4 350; flag mean: Rogue 3.6/5.4 has it, rx1 not |
| jabberwock | - | J; L15; x3000; AC6; 2d12/2d4; carry 70%; no flags | J; L15; x3000; AC6; 2d12/2d4; dlvl 25; gold 2d100; carry 75%; flags: chase | SAME stats |
| dragon | D; L10; x9000; AC-1; 1d8/1d8/3d10; carry 100%; greedy | D; L10; x5000; AC-1; 1d8/1d8/3d10; carry 100%; mean | D; L10; x6800; AC-1; 1d8/1d8/3d10; dlvl 26; gold 3d100; carry 95%; flags: mean flying regenerating greedy | DIFF: exp: rx1 6800, 3.6 9000, 5.4 5000 |
Observations on the table:
- Stats identical in all games that have the monster: bat, emu, hobgoblin, floating eye, ice monster, gnome, orc, invisible stalker, leprechaun, aquator, yeti, wraith, phantom, griffin, xeroc, xorn (stats), jabberwock, black unicorn, quagga.
- exp differs: snake (3.6 3, 5.4 2, rx1 3 = 3.6), zombie (7/6/7 = 3.6), nymph (40/37/37 = 5.4), troll (55/120/120 = 5.4), vampire (380/350/350 = 5.4), centaur (15/17/15 = 3.6), rust monster (3.6 25, rx1 20), mimic (3.6 140, rx1 100), purple worm (3.6 7000, rx1 4000), umber hulk (3.6 130, rx1 200), quasit (3.6 35, rx1 32), dragon (3.6 9000, 5.4 5000, rx1 6800; neither).
  The 3.6 vs 5.4 differences themselves (snake, zombie, centaur exp and damage, nymph, troll, vampire, dragon) are Rogue-internal changes.
- Level differs: violet fungi is level 8 in 3.6 (hp 8d8), level 3 in rx1 (hp 3d8). (5.4 flytrap, the successor, is 8.)
- Venus flytrap and violet fungi damage is not a plain dice string in Rogue (`"%%%x0"` placeholder in 5.4 `extern.c`, `"000d0"` in 3.6). Real damage is generated at hit time (see 6.5).
- Mean flag: Rogue `ISMEAN` monsters wake easily (section 5.5). rx1 `mean` is set only on: hobgoblin, emu, ice monster (not mean in Rogue), aquator, venus flytrap, griffin, dragon. It is missing from 18 monsters that are mean in Rogue (kestrel, jackal, snake, rattlesnake, rust monster, zombie, giant ant, kobold, black unicorn, quagga, troll, violet fungi, medusa, quasit, xorn, vampire, umber hulk, and 3.6 only-ones). rx1 compensates for some with `chase`.
- The 3.6 ISBLOCK on orc has no code effect in 3.6 (grep: flag never tested). 5.4 replaced it with ISGREEDY. rx1 follows 5.4 (greedy).
- ISREGEN: 3.6 troll/vampire; 5.4 troll/vampire/griffin (dead flag, never read in 5.4); rx1 troll/vampire/griffin/dragon (dragon is rx1's addition). rx1 effect is +1 hp per player turn (section 7.4).
- rx1 `flying` is on bat, griffin, dragon only (kestrel has ISFLY in 5.4; rx1 lacks it). rx1 `haste` on yeti and `slow` on zombie are rx1's own; neither Rogue has them (the zombie is not slow in Rogue, the yeti not fast).
- rx1 `revive` (troll) and `group` (hobgoblin, jackal, ur-vile) are rx1-only. `group` is parsed but no code reads it (section 5.2).
- rx1 nymph, leprechaun, xeroc, mimic, ice monster etc. carry gold dice and large carry%, unlike Rogue where only a handful carry (section 5.9).

## 3. Spawn depth tables

### 3.1 Rogue tables

3.6 `monsters.c`: `lvl_mons = "KJBSHEAOZGLCRQNYTWFIXUMVDP"`, `wand_mons = "KJBSH AOZG CRQ Y W IXU V  "` (blanks never wander).
5.4 `monsters.c`: `lvl_mons = "KEBSHIROZLCQANYFTWPXUMVGJD"`, `wand_mons = "KEBSH ROZ CQ Y W P UMV GJ "` with blanks at positions of I, L, N, F, X, D (positions 6, 10, 14, 16, 20, 26).

Both: `randmonster(wander)`:
```
d = level + (rnd(10) - 5)      // 3.6, 1-based loop via mons[--d]
d = level + (rnd(10) - 6)      // 5.4, 0-based
if d < 1 (3.6) / d < 0 (5.4)  -> rnd(5) (+1 in 3.6)
if d > 26 (3.6) / d > 25 (5.4) -> rnd(5) + 22 (3.6) / rnd(5)+21 (5.4)
3.6: do { mons = wander ? wand_mons : lvl_mons; } while (mons[--d] == ' ')   -- on a blank, keeps decrementing (shifts to the next lower letter)
5.4: same with `wander` blank loop
```
Both give a window of 10 slots centred 1 below the level: slots level-5 .. level+4 (3.6, 1-based) which is the same as 5.4's 0-based `level-6..level+3` + 1. Both equal.
Blank handling: in both games a blank slot is skipped by decrementing d until a letter is found, so the monster before a blank gets double weight for wanderers.

### 3.2 rx1

`game/spawn_rogue.go: rogueRandMonster(random, level, wander)`: `d = level + rnd(10) - 5` (1-based); `d < 1 -> rnd(5)+1`; `d > 26 -> rnd(5)+22`; then picks uniformly among all
monster defs whose `dlvl == d` (monsters.rec field). Excludes `xeroc_2`, genocided, and (if wander) the `nonWanderers` set
{ice_monster, leprechaun, nymph, venus_flytrap, xeroc, dragon, floating_eye, violet_fungi, mimic, purple_worm}. If the slot yields no wandering candidate rx1 re-picks (it does not
do Rogue's decrement-to-next-letter).

Verdicts: window formula SAME. Slot assignment DIFFERENT (rx1 merges both games into one table so some slots hold two monsters, each then getting about half the weight).
Wanderer blank set DIFFERENT: rx1 is the union of 3.6 and 5.4 blanks (troll wanders in rx1 as in 5.4's table; in 3.6 it is blank). Blank handling DIFFERENT (rx1 re-pick vs. decrement).

### 3.3 Slot table (slot index 1..26)

| Slot | 3.6 monster (wander?) | 5.4 monster (wander?) | rx1 `dlvl` occupants |
|---|---|---|---|
| 1 | kobold (W) | kestrel (W) | kestrel, kobold |
| 2 | jackal (W) | emu (W) | emu, jackal |
| 3 | bat | bat | bat |
| 4 | snake | snake | snake |
| 5 | hobgoblin | hobgoblin | hobgoblin |
| 6 | floating eye (blank) | ice monster (blank) | floating eye, ice monster, slime (only slime wanders: rx1-only) |
| 7 | giant ant | rattlesnake | rattlesnake, giant ant |
| 8 | orc | orc | orc |
| 9 | zombie | zombie | zombie |
| 10 | gnome (W) | leprechaun (blank) | gnome, leprechaun (only gnome wanders) |
| 11 | leprechaun (blank) | centaur | centaur |
| 12 | centaur | quagga | black unicorn, quagga |
| 13 | rust monster | aquator | rust monster, aquator |
| 14 | quasit | nymph (blank) | nymph, quasit (only quasit wanders) |
| 15 | nymph (blank) | yeti | yeti |
| 16 | yeti | venus flytrap (blank) | venus flytrap (never wanders: slot empty for wanderers, re-pick) |
| 17 | troll (blank) | troll | troll |
| 18 | wraith | wraith | wraith |
| 19 | violet fungi (blank) | phantom | violet fungi, phantom (only phantom wanders) |
| 20 | invisible stalker | xeroc (blank) | invisible stalker, xeroc (only stalker wanders) |
| 21 | xorn | black unicorn | ur-vile, xorn |
| 22 | umber hulk | medusa | umber hulk, medusa |
| 23 | mimic (blank) | vampire | mimic, vampire |
| 24 | vampire | griffin | griffin (vampire is at 23 in rx1) |
| 25 | dragon (blank) | jabberwock | jabberwock |
| 26 | purple worm (blank) | dragon (blank) | purple worm, dragon (neither wanders; wanderers on slot 26 never exist) |

So the rx1 depth of a shared monster follows 5.4 (zombie, orc, troll, wraith, yeti 15 [5.4] vs 16 [3.6], centaur 11 [5.4] vs 12 [3.6], nymph 14 [5.4] vs 15, leprechaun 10 [5.4] vs 11,
vampire 23 [5.4] vs 24, dragon 26 [5.4] vs 25) and 3.6-only monsters follow 3.6 position (jackal 2, kobold 1, ...). Result: the same monster appears one level different from one of the two Rogues for
every shared monster that moved between versions; also total weight per depth is not uniform (e.g. slot 16 has one monster that cannot wander, slot 26 two that cannot wander).
Unlike Rogue, rx1 has no exception for the stalker and the other 3.6/5.4 split in a given slot; the sets were simply merged.

## 4. HP, exp, level scaling

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| HP | `monsters.c: new_monster`: `hp = max_hp = roll(lvl, 8)` | same, `roll(lvl, 8)` after `lvl += lev_add` | `state.go: NewEnemyFromDef`: `max(1, Dice(lvl,8).Roll())` | SAME (rx1 floors at 1; Rogue min is lvl) |
| Level scaling past Amulet | none | `lev_add = max(0, level - AMULETLEVEL(26))`; `lvl += lev_add`; `arm -= lev_add` | `levAdd = max(0, dungeonLevel-26)`; `lvl += levAdd`; `Arm = def.Armor - levAdd` | SAME (3.6 has none: DIFFERENT 3.6 vs 5.4) |
| Exp bonus | none: `s_exp` as in table | `exp = base + lev_add*10 + exp_add(tp)` | `Exp = def.Exp + levAdd*10 + rpg.ExpAdd(lvl, hp)` | SAME as 5.4; DIFFERENT from 3.6 |
| exp_add | n/a | `mod = (lvl==1) ? maxhp/8 : maxhp/6; if lvl>9 mod*=20; else if lvl>6 mod*=4` | `rpg/rogue.go: ExpAdd` identical | SAME as 5.4 |
| Haste past level 29 | none | `if (level > 29) turn_on(*tp, ISHASTE)` | none | MISSING in rx1 |
| Strength field | `s_str = 10` | `s_str = 10` | `Str = 10` | SAME |
| Exp awarded on kill | `fight.c: killed()` adds `mp->t_stats.s_exp` | same, plus none | `state.go: actorKilled` awards only if killer is the player | SAME. Note HANDOVER.md claim "monsters give no XP" is stale |
| Level-up table | `e_levels` 10,20,40,80,... | same | `rpg.ExpLevels` same | SAME |
| Stat reset on level change | n/a | n/a | n/a | n/a |

## 5. Population, wake/sleep, AI

### 5.1 Initial population and treasure rooms

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Monster in a room at level creation | `rooms.c: do_rooms`: `if (rnd(100) < (goldval>0 ? 80 : 25))` place `new_monster(randmonster(FALSE))` in a random spot; hero's start room included (corrected/verified: `do_rooms` runs before the hero is placed, and the hero is then put on a floor cell with no monster; monster then not asleep: see 5.3) | same rule | None. `spawn_rogue.go` comment: "do_rooms: gold only; monsters spawn via wandering daemon during play" | MISSING: a new rx1 level starts empty (only the wanderer daemon and special rooms add monsters) |
| Treasure room | none | `new_level.c: treas_room`: 1/TREAS_ROOM(20) chance per level; room gets 2..10 items (MINTREAS 2, MAXTREAS 10) and `nm = max(rnd(spots)+2, numItems+2)` monsters from `level+1`, each `ISMEAN` and with `give_pack` | `rogueTreasureRoom`: 1/20, 2..10 items, `nm = max(rnd(spots)+2, numItems+2)`, monsters from `level+1`, flags Sleep + Mean | SAME as 5.4 (3.6 has none) |
| Ring-fenced vault monsters | none | none | `special_rooms.go` has other special rooms (not Rogue) | rx1-only |
| Visited level persistence | levels discarded on leaving (monsters freed in `new_level`) | same | `dungeon.go`: visited levels persist with monsters (commit 3febb97) | rx1-only |

### 5.2 Groups
Neither Rogue has monster packs. rx1 `monsters.rec` marks hobgoblin, jackal, ur-vile as `group`. `foundation/mapflags.go` parses it, but no code reads it: no packs are generated. HANDOVER.md says "packs of 1..3 extra"; not implemented. MISSING relative to the doc, rx1-only relative to Rogue.

### 5.3 Wake and sleep

Rogue has no "asleep" flag. A monster whose `ISRUN` is clear simply does nothing. `ISRUN` is set by `runto()`, `wake_monster()`, aggravate, being hit.

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Initial state | `new_monster`: not `ISRUN` | same | monsters are spawned with Sleep for non-wanderers; wanderers spawn aware (`SetAware`) | DIFFERENT in model, same effect |
| `wake_monster` condition | `monsters.c`: `ISMEAN && !ISRUN && !ISHELD && !stealth ring` and `rnd(100) > 33` (about 2/3) | `!ISRUN && rnd(3) != 0 && ISMEAN && !ISHELD && !stealth && !levitating` -> `runto` | `ai.go: aiAct` sleeping branch: if same room, a Mean monster with `CanPerceivePlayer` (2/3) wakes; a non-mean wakes with 2/3 * 1/10 | DIFFERENT: Rogue wakes only mean monsters; rx1 lets non-mean wake slowly. SAME 2/3 chance for mean |
| Who triggers it | `look()` (adjacent squares each move), and `light()` / entering a lit room: every monster in the room | same | any turn while player and monster share a room (no adjacency requirement) | DIFFERENT (rx1 wakes whole room continuously; Rogue only on those calls) |
| Stealth ring | stops waking by `wake_monster` | same, plus levitation | no ring of stealth in rx1 | MISSING |
| Aggravate | scroll aggravate: `aggravate()` sets all `runto` | same, also ring of aggravate | `effects_use.go: aggroMonsters` only clears Sleep | DIFFERENT (no ring; no chase set, just wakes) |
| Damage wakes | any hit via `runto` | same | `damageActorWithFollowUp`: 90% wake chance | DIFFERENT (Rogue 100%) |
| Hit bonus vs. sleeping | none | `roll_em`: +4 vs defender not `ISRUN` (so, quirk, monsters get +4 vs the player, who normally lacks ISRUN until first paralysis recovery) | `rollAttack`: +4 vs a sleeping or held non-player defender | DIFFERENT |
| Sneak attack | none | none | `sneakMultiplier`: x3 damage (dagger x5) vs unaware, always hits | rx1-only |
| Medusa gaze wake | n/a | `wake_monster`: M needs `ISRUN`, not `ISFOUND`, not `ISCANC`; hero not blind/hallucinating; lit room or dist<3; confuses `spread(20)` unless save | `ai.go: aiGaze` once; `Save(playerLevel, VsMagic)` fails -> confuse `spread(20)`; scare -> Stun | DIFFERENT detail (rx1 has no ISFOUND/lighting test) |
| Umber hulk gaze | 3.6 `wake_monster`: U gazes (confuses) | none (hulk absent) | umber hulk has no gaze effect in rx1; medusa has | MISSING |
| Greedy monsters | 3.6: `ISGREEDY` (dragon) goes for gold in the hero room | 5.4: `wake_monster`: greedy target room gold else hero; orc, dragon-era | `abilities.go: aiGoForGold` while unaware | SAME in spirit |

### 5.4 Wandering-monster daemon

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Daemon | `daemons.c: rollwand`: `if (++between >= 4) { if (roll(1,6) == 4) { wanderer(); kill_daemon(rollwand); fuse(swander, WANDERTIME, AFTER);} between = 0 }` | same | `state.go: wanderingMonsterTick`: every 4 turns, `rand.Intn(6)==4` -> spawn | SAME roll; rx1 has no quiet period |
| Quiet period | `WANDERTIME 70` (flat); `swander` restarts rollwand | `spread(70)` (67..73); `spread(nm) = nm - nm/20 + rnd(nm/10)` | none: it can fire again 4 turns later | DIFFERENT. Rogue mean interval ~ 70 + 4*6 = ~94 turns; rx1 ~ 24 turns (4x faster) |
| Initial fuse | `main.c`: starts with `fuse(swander, 70, AFTER)` so no wanderer in first ~70 turns | same | none | MISSING |
| Spawn location | `wanderer()`: random room not the hero's (`do ... while (hr == player room)`), `rnd_pos`, then `runto` | `find_floor(... monst=TRUE)`, never hero room, then `runto` | `wanderingMonster`: requires player in a room; picks random other room (10 tries) and floor tile (10 tries); `SetAware`; message "You sense a %s stirring in the dungeon" | SAME idea; rx1 adds a message, requires player to be in a room |
| Monster selection | `randmonster(TRUE)` | `randmonster(TRUE)` | `rogueRandMonster(..., wander=true)` | SAME (see 3) |
| Pack on spawn (corrected) | `wanderer()` never gives a pack (carry roll exists only in `do_rooms`, 3.6 `new_monster` does not roll carry) | `wanderer()` never calls `give_pack` (only `do_rooms` and `treas_room` do): wanderers carry nothing | carry rule in `NewEnemyFromDef` (5.9) | DIFFERENT (corrected: rx1 rolls carry for wanderers too, Rogue never) |

### 5.5 Mean, chase, and who is "awake"

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Awake model | `ISRUN` | `ISRUN` | `Sleep` flag + `Aware` | DIFFERENT model |
| After waking | monster chases forever (door-seeking, tracks via `ch_ret`/`find_dest`) and loses interest only via 3.6 `ISHELD`/etc. | same; in `chase()` loses nothing; fixed: `runto` always | `ai.go`: `wantToChase = sameRoom || FlagChase`: a monster without `chase` stops pursuing outside the player's room (goes idle/loiters) | DIFFERENT (major): in Rogue every awake monster hunts the hero across the level |
| Awareness test | none (ISRUN or not) | none | `sameRoom && LOS && (Mean || 2/3)`; `enemyCanSpotPlayer`: dist<=4, or <=10 with LOS if player emits light | rx1-only |
| `ISMEAN` effect | only in `wake_monster` and treasure room | same | only `ai.go` sleeping/awareness branch | SAME role |

### 5.6 Movement and chase AI

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Order of actions | `daemons: runners()` (AFTER daemon) iterates the monster list once per hero move; `attach()` prepends so newest monster acts first | `move_monst`/`runners` same; newest first | `deamons.go: enemyMovement` iterates `Actors()` in insertion order (oldest first) | DIFFERENT (order reversed) |
| Speed | `ISHASTE`: 2 moves; `ISSLOW`: acts on alternate turns via `t_turn` | same; plus `ISFLY` extra move if dist²>=3 (the bat/kestrel/griffin close faster) | energy system: speed 10 normal, 20 haste (cost 100/speed), 5 slow; level <=2 halves the player-turn time so monsters are at half speed | DIFFERENT: rx1 lacks ISFLY bonus; adds slow zombie and fast yeti; the level<=2 slowdown is rx1-only |
| Stationary | only letter F (violet fungi) never moves in `do_chase` | flytrap F stationary (flag-free letter test) | `stationary` flag on floating eye, violet fungi, venus flytrap (`abilities.go aiSpecialMove`) | DIFFERENT: rx1 also pins the floating eye (in Rogue the eye E does move) |
| Pathing | greedy step reducing distance to goal, picks adjacent cell nearest by squared distance; if target is in another room seeks the nearest door by squared distance to the goal | 5.4: same, ties broken randomly | Dijkstra map to the player (true shortest path with distance cost) | DIFFERENT: rx1 monsters are perfect trackers, Rogue's can get stuck |
| Diagonal rule | `diag_ok`: both orthogonal neighbours must pass `step_ok`; doors block diagonals | same | none: `geometry.Neighbors.All`, free diagonal movement, no corner/doorway restriction | DIFFERENT |
| Attack range | adjacent only, via `diag_ok` too (cannot attack diagonally around corners/through doors) | same | `distance <= 1`, no corner rule | DIFFERENT |
| Confusion | 80% random step (`rnd(10) < 8`), 5% recover | 80% (`rnd(5) != 0`), 5% | `doesActConfused` (duration `spread(20)`) | DIFFERENT detail, same idea |
| Erratic | bat 'B' 50% random; invisible stalker 'I' 20% | bat 50%; phantom 'P' 20% | `abilities.go: erraticChance`: flying 50%, otherwise 20% | SAME numbers (rx1 gives non-flying erratic 20%: stalker, phantom) |
| Scare monster | monsters will not step on a scare scroll square | same | none on floor | MISSING |
| Xeroc/disguise | 3.6 mimic 'M' disguised as item (switch `rnd(level>25 ? 9 : 8)` over GOLD, POTION, SCROLL, STAIRS, WEAPON, ARMOR, RING, STICK, AMULET) | 5.4 xeroc 'X' disguised via `rnd_thing()` (incl. FOOD) | `disguised` flag (mimic, xeroc); `aiAct` returns early while disguised; `revealDisguised` | SAME idea, different set |
| Reveal costs the strike | "Wait! That's a mimic!" the hit is lost | same for xeroc | `actorMeleeAttackMult`: reveal then attack proceeds | DIFFERENT (rx1 keeps the strike) |
| Door-seeking | tests squared distance of every door in room to goal | same | covered by Dijkstra | n/a |
| Ranged use | none | dragon flame only (below) | `ai.go defaultBehaviour`: zap/use effects at 1/5 when same room and not adjacent (breath, dart, arrow, lightning, charge) | rx1-only (except flame) |
| Invisible | 'I' stalker `ISINVIS`: seen only with see invisible | phantom 'P' | `invisible` flag | SAME |

### 5.7 Item pickup by monsters

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Monsters picking up floor items | none | `chase.c: find_dest`: monsters with carry>0 head for items in their room (prob = carry%) and pick them up | none; only greedy `aiGoForGold` | MISSING (5.4 only) |

### 5.8 To-hit
| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Swing | `fight.c: swing`: `rnd(20)+1+wplus >= 21 - lvl - arm` | `rnd(20)`, need `(20-lvl)-arm` | `rpg.Swing`: `rand(1..20)+wplus >= 21-lvl-arm` | SAME probability |
| Monster Str | 10 | 10 | 10 | SAME |

### 5.9 Carrying (corrected: renumbered from 5.12)
| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Rule (corrected) | `do_rooms` only (3.6 `new_monster` itself does not roll): `rnd(100) < m_carry` -> `new_thing()` attached to pack (single item, any type, gold not included); wanderers and generated treasure never carry in 3.6 | `give_pack` (called from `do_rooms` and `treas_room` only, not `wanderer`): `level >= max_level && rnd(100) < m_carry` -> `new_thing()`; leprechaun death makes gold | `NewEnemyFromDef`: `rand(100) < carry`; then again `rand(100) < carry` -> `rogueNewThing` item else gold dice. P(item) = c^2, P(gold) = c(1-c) | DIFFERENT |
| Table values | mostly 0; 100 nymph/dragon; 70 purple worm; 50 troll; 40 umber hulk; 30 mimic/yeti/quasit | similar (see table) | re-tuned for every monster (3% to 95%) with gold dice | DIFFERENT |
| Drop on death | pack dropped on the floor | same | `dropInventory` drops gold and items | SAME |
| Gold | not carried by monsters (except leprechaun, steal) | leprechaun | any monster with gold dice | rx1-only |
| First-visit only | no | `level >= max_level` (not when revisiting) | levels persist, so no repeat | DIFFERENT |

## 6. Special abilities

### 6.1 Rust, freeze, strength, level and HP drain

| Ability | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Rust armor (corrected) | R rust monster, each hit: if worn armor `ac < 9` it rusts `ac++`; leather not exempt; no ISPROT (3.6 has no protect armor scroll or ring) | A aquator: same + not leather; skipped for leather and `o_arm >= 9`; blocked by `ISPROT` or ring `R_SUSTARM` (maintain armor) | `effects_hit.go rust_armor`: first armor with protection>0, no leather/protect exemptions; aquator + rust monster | DIFFERENT: no leather exemption, no ISPROT/sustain checks (5.4); same as 3.6 |
| Freeze | none (eye is a different thing) | I ice monster: `no_command += rnd(2)+2`, death if >50 turns | `freeze` sets Stun | DIFFERENT |
| Floating eye / freeze | E floating eye (`fight.c: attack`): hit paralyses hero, `no_command += rnd(2)+2` | n/a (absent; ice monster I does the same) | `floating_eye` (0d0, stationary) has `struck_effect: freeze | 100`: the hero is Stunned when HE hits it (NetHack style, `applyStruckEffects`), its own attack does nothing; `freeze` is a Stun | DIFFERENT (corrected: was MISSING; trigger is inverted, and the eye does move in 3.6 but is pinned in rx1) |
| Strength drain | A giant ant: `!save(VS_POISON)` -> `chg_str(-1)` | R rattlesnake: same | `poison_strength`: `Save(VsPoison)`; `ChangeStrength(-1)` clamped 3..18 | SAME (clamp is rx1) |
| Level drain | W wraith 15%: exp = `e_levels[lvl-1]+1`; lose `roll(1,10)` hp and max hp; death at level 1 | same | `drain_level`: `DrainLevel` sets exp to the bottom of the new level; lose 1d10; `max(1,...)` instead of death | DIFFERENT: exp reset value and no death |
| Vampire | none in 3.6 | V 30%: lose `roll(1,3)` max hp and hp | `drain_max_hp` 30%: -1 max hp; vampire heals +1 | DIFFERENT |
| Hold | F violet fungi: `ISHELD`; damage "N d1" with `++fung_hit`; on miss still deals `fung_hit` | F flytrap: `ISHELD`; "N x1" with `++vf_hit`; -1 hp on each hit | `hold` effect (`holdAndSqueeze`): first hit grabs, later hits deal count-1 | DIFFERENT detail |
| Gold steal | L leprechaun: `GOLDCALC` (= rnd(50+10*level)+2), +4x if `!save(VS_MAGIC)`, then vanishes | same | `steal_gold`: same GOLDCALC, +4x if save fails, vanishes | SAME |
| Item steal | N nymph: random non-equipped magic item; both disappear | same | `steal_item`: random unequipped IsMagic item; amulet excluded | SAME |
| Dragon breath | none | `do_chase`: 1-in-5 when in a straight line, dist²<=36, any range, `fire_bolt`: 6d6, `save(VS_MAGIC)` to avoid, bounces | `effects_zap.go fireBreath`: flat 5 dmg to all along Bresenham path, no save, any angle in same room, 1/5 when not adjacent | DIFFERENT |
| Regeneration | ISREGEN: 33% +1 hp per `attack()` call | dead flag (never read) | `removeDeadAndApplyRegeneration`: +1 hp/turn for FlagRegenerating | DIFFERENT |
| Invisible | I stalker | P phantom | `invisible` flag | SAME |
| Revive | none | none | `tryRevive` (troll) | rx1-only |
| Hit-and-run, split, knockback, poison, slow, hunger, eat_gold, rust_weapon, lightning, magic dart/arrow, charge | none | none | `effects_hit.go` / `effects_zap.go` (slime splits, etc.) | rx1-only |
| Cancellation | wand of cancel: removes abilities | same | effects suppressed if owner has FlagCancel | SAME |

## 7. Other differences

| Topic | Rogue | rx1 |
|---|---|---|
| Disguise table | 3.6: rnd(8/9) item types; 5.4: rnd_thing (POTION SCROLL RING STICK FOOD WEAPON ARMOR STAIRS GOLD AMULET) | `disguised` mimics a fixed glyph |
| Ice monster mean | no | mean |
| Slime | none | slime splits |
| Level 1-2 slowdown | none | monsters at half speed (rx1-only) |

## 8. Summary of main differences

1. Roster: rx1 contains both games' 26 (39 distinct) plus slime, ur-vile, xeroc mk ii. Stats match Rogue except 12 exp values, violet fungi level (3 vs 8), centaur damage (3.6 variant).
2. Mean flags: rx1 dropped `mean` from 17 monsters and added it to the ice monster.
3. Depth: same randmonster formula; merged slot table puts two monsters on 14 slots and leaves some slots with no wanderers.
4. No initial population: rx1 levels start empty; Rogue fills rooms 80%/25%. Treasure rooms match 5.4.
5. Wanderer daemon: same roll, but no ~70-turn quiet period and no initial fuse (about 4x as frequent).
6. Hp, exp_add, and post-26 scaling match 5.4; haste past 29 is missing.
7. AI: Dijkstra tracking, free diagonals, `sameRoom || chase` pursuit rule, reverse action order, no scare monster, no stealth, no ISFLY bonus.
8. Abilities: gold/item steal and strength drain match; rust, freeze, level drain, vampire, dragon breath, regeneration, and reveal behaviour differ (see 6.1).
9. Carrying: c^2 item / c(1-c) gold rule with re-tuned carry%, versus Rogue's single roll.
