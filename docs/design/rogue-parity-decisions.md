# rx1 vs Rogue: parity decisions

Baseline where 3.6 and 5.4 disagree: **5.4**. Source of truth for differences:
`rogue-compare-*.md` in this folder. Everything rx1 added on purpose stays
(town, persistent levels, arriving on stairs, light sources, themes, fatigue,
sneak attacks, extra weapons, special rooms, timed curses, extra traps).

## Match 5.4 (to implement)
- Monsters placed in rooms at level creation (80% gold rooms / 25% others).
- Wanderers: 70-turn cooldown, may spawn while player is in a corridor, never carry items.
- Hunger: 5.4 stages (stomach 2150, hungry/weak/faint/starve), starvation death, fruit/mango not needed unless in 5.4 food table.
- Healing potions: 5.4 formulas (healing roll(lvl,4), extra roll(lvl,8), raise max HP when over, cure blindness).
- Strength: 5.4 tables, max 31. Potion of gain strength also does restore strength.
- New items: potion of gain strength; rings of stealth and searching.
- Traps: Rogue's trap set and rules, plus rx1's extra types.
- Score: 5.4 formula (uses the `worth` fields).
- Monster stats: exact 5.4 table (exp, levels, mean flags), haste past level 29.
- Scroll/wand rules: 5.4 sleep, aggravate, scare monster (floor repel + dust), empty wands stay, charges 3-7 (light 10-19).
- XP: 5.4 table and formula.
- Pack group sizes as Rogue; equipping takes a turn.
- Magic mapping reveals secret doors.

## Deliberate differences (keep)
- Status effect durations are short, see "Durations" below.
- Curses are timed (100-399 equipped turns), no donning delay.
- Ring bonuses apply to every attack, periodic +1 HP regen,
  80x23 map (rooms max 6 tall).
- Missing items kept out: poison, light, protect armor, blank, wand of nothing,
  sustain strength, adornment, aggravate, maintain armor.

## Monster abilities and AI (decided, implemented)
Matched 5.4: rust armor exemptions, dragon fire breath (6d6, save, bounce), venus flytrap hold,
floating eye (3.6 style: moves, paralyses when it hits), hunt across the level, diagonal/doorway
rule, newest-first order, only mean monsters wake, hits always wake, fliers' extra move, monsters
pick up items.
Kept as rx1 (deliberate): ice monster freeze (Stun), wraith level drain, vampire drain and heal,
disguise reveal does not cost the strike, shortest-path (Dijkstra) chasing.

## Durations (proposed, implemented)
Unit: player actions (rx1 counters tick once per action; Rogue's are rounds, 850 is too harsh).

| Effect | Rogue | rx1 now | Rationale |
|---|---|---|---|
| Blindness | 850 | 25-40 | long enough to hurt, short enough to wait out |
| Hallucination | 850 | 100-150 | harmless, so longer; only the disguise is annoying |
| See invisible | 850 | 100-200 | a benefit, worth a long run |
| Levitation | 30 (5.4) | 25-35 | 5.4 value, spread |
| Haste self | 4-7 rounds (3.6) | 20-30 | 5.4/3.6 value is too short to use; scaled |
| Slow (wand) | n/a | 20-30 | same as haste |
| Confusion | 20-27 (3.6) | 20-27 | 3.6 value (was 18-28) |

Unchanged at 8-15: monster detection, trap detection, invisibility, cancellation.

Also changed (5.4 parity): enchant armor/weapon pick-an-item kept, no +8 cap, uncurses;
ring of teleportation 2% per turn; fire/lightning/cold wands 6d6 and a save vs magic
negates the hit (bounce chances and bolt range unchanged); daggers 2-5 per group;
5.4 `worth` for wands; gold carry via `gold_chance` (rx1's old carry chance), item carry is one 5.4 roll.
