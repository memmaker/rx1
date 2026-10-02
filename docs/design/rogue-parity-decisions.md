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
- Status effect durations are short (8-15 turns), values to be re-tuned (open).
- Curses are timed (100-399 equipped turns), no donning delay.
- Ring bonuses apply to every attack, enchant cap, periodic +1 HP regen,
  80x23 map (rooms max 6 tall).
- Missing items kept out: poison, light, protect armor, blank, wand of nothing,
  sustain strength, adornment, aggravate, maintain armor.

## Open (to review row by row)
- Monster special abilities (rust, freeze, level drain, vampire, dragon breath, regeneration, disguise reveal).
- Monster AI (shortest path, diagonals, action order, scare avoidance, non-chasers).
- Status effect duration tuning.
