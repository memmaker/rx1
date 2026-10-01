# Combat design: two visions

Status: draft for discussion. Pick one direction (or a deliberate blend) before
balancing monsters.

## Where rx1 stands today

- **Resolution:** GURPS-style. 3d6 ≤ skill to hit, then 3d6 ≤ Dodge to avoid,
  armor reduces damage (DR), criticals on both sides.
  *Bug: the defense roll never succeeds (`rpg/rpg.go:164`), so every successful
  attack hits.*
- **Progression:** character points. The player gets 10 per new dungeon level
  (`newLevelReached`) and spends them on stats and skills. Kills give nothing.
- **Score:** gold.
- **Stealth already exists:** sleeping monsters, `mean` monsters waking up more
  easily, and `CanPerceivePlayer` rolled against the player's Stealth skill.
- **Monsters:** Rogue's roster with most of its abilities, plus picks from the
  abilities list (see `HANDOVER.md`).

## Shared pressure (both visions)

Keep Rogue's clock: hunger, slow natural healing, wandering monsters and
permadeath. Both visions need a reason to keep moving down; they differ in what
the player does when something is in the way.

---

## Vision 1: Rogue, but fairer, with more combat tools

**Pitch:** combat is the main activity, and the player wins by making good
tactical choices. Deaths come from decisions, not from a streak of bad dice.

### Pillars

1. **Less swing.** The bell curve does most of the work already; keep it and
   remove the extremes.
2. **Readable odds.** The player can see roughly what a fight will cost before
   committing.
3. **Many verbs.** Every fight offers more than one option.

### Mechanics

- **Fix the defense roll** and use Dodge as the main defensive stat.
- **Cap criticals.** Drop the ×3 results; use a critical success to guarantee
  max damage, and a critical failure to lose a turn rather than hurt yourself.
- **Show the odds.** Look/target displays the hit chance in both directions
  ("You hit 62%, it hits you 40%") and the monster's damage dice.
- **Combat verbs:**
  - *All-out attack*: +4 to hit, no defense roll this turn.
  - *All-out defense*: +3 Dodge, no attack. Lets you hold a corridor or wait for
    help.
  - *Shove*: push a monster back one tile (reuse `knockback`); into a trap or
    out of a doorway.
  - *Retreat step*: move back with +2 Dodge (GURPS retreat).
  - *Aimed throws*: thrown weapons and darts as real options, not leftovers.
- **Recovery between fights.** A short rest (several turns, costs food) restores
  fatigue points; HP stays scarce.
- **Kills give character points**, scaled by monster level, so fighting is
  rewarded. Depth bonus stays.
- **Monster abilities are telegraphed.** "The ice monster's touch freezes",
  "The rust monster eyes your armor": the first time you meet one, a one-line
  hint appears. Nasty effects become problems to plan for, not surprises.

### What it costs

- More UI: odds display, combat-mode keys.
- Balance work: kill rewards and depth rewards must not make the player outpace
  the dungeon.
- Risk: fights get longer if both sides dodge more. Counter with higher damage
  or fewer HP.

---

## Vision 2: OSR-style Rogue — avoid fights, think around problems

**Pitch:** monsters are obstacles, not loot. Fighting fair is how you die.
The player wins by sneaking, tricking, bribing, running and using the
environment.

### Pillars

1. **No reward for killing.** Experience comes from treasure and depth.
2. **Combat is dangerous and quick** for both sides.
3. **Information beats stats.** Knowing a monster's habits matters more than
   your sword skill.
4. **Every problem has a non-combat answer.**

### Mechanics

- **XP for gold.** Character points come from gold brought up the stairs, or
  from gold found, plus the existing depth bonus. Kills give nothing.
- **Lethal combat.** Keep the critical table, lower HP on both sides, and make
  armor heavy (DR, but slower movement and worse stealth).
- **Stealth as a core skill:**
  - Noise: running, fighting and opening doors make noise that wakes monsters in
    nearby rooms.
  - Light: lit rooms expose you, dark rooms and corridors hide you (ties in with
    the `darkness` ability).
  - Monsters that wake don't automatically know where you are; they search.
  - Backstab: attacking a sleeping or unaware monster gets a large bonus, so
    killing is possible, but only on your terms.
- **Monster reactions.** Many monsters are not hostile on sight. A reaction roll
  (2d6 in OSR) decides: hostile, wary, neutral or open to a bribe.
  - Leprechauns can be paid off; orcs and dragons are `greedy` and can be
    distracted with thrown gold.
  - Animals can be distracted with food.
- **Out-of-the-box tools:**
  - Throwing items to make noise elsewhere.
  - Closing and spiking doors.
  - Luring monsters into traps (trap objects already exist).
  - Using monster abilities against each other: a confused or scared monster
    attacks or flees from its neighbours.
  - Scrolls and potions as utility (scare, sleep, teleport) rather than damage.
- **Morale.** Monsters flee when hurt or when their pack leader dies (reuse
  `scared`). A fight you start badly can still end without a corpse.
- **Lore as power.** The lore documents (`lore/monsters/*.txt`) hint at
  weaknesses and habits; reading them matters.

### What it costs

- New systems: noise, reaction rolls, morale, bribes.
- Monster AI needs states (asleep, unaware, searching, hostile, fleeing).
- Risk: if stealth is too strong the game becomes a walking simulator; if too
  weak, players fight anyway and die. Hunger and wandering monsters must keep the
  pressure on.

---

## Comparison

| | Vision 1: Fair fighter | Vision 2: OSR |
|---|---|---|
| Main activity | Fighting well | Avoiding and outwitting |
| Kills | Give character points | Give nothing |
| Character points from | Kills + depth | Gold + depth |
| Combat feel | Longer, readable, tactical | Short, deadly, avoidable |
| Criticals | Capped | Full table |
| Stealth | Nice to have | Core skill |
| Monster abilities | Telegraphed threats | Puzzles with non-combat answers |
| New systems | Combat verbs, odds display, rest | Noise, reactions, morale, bribes |
| Fits current code | Defense fix, `knockback`, kill rewards | Stealth checks, depth points, `greedy`, `scared`, lore files |

## Decisions needed

1. Which vision, or which blend? (A possible blend: Vision 2 progression and
   stealth with Vision 1's odds display and fixed defense.)
2. What gives character points: kills, gold, depth, or a mix?
3. How lethal should a "fair" fight be at equal level: about 50/50, or clearly
   in the player's favour?

## Light sources (decided, built)

Angband-style light, with tiers and visuals copied from heavenandhell. This
supports both visions, and Vision 2 most of all: darkness is both a danger and
a place to hide.

### Rules

- **Slot:** a dedicated `light_source` equip slot. The player starts with a
  torch equipped.
- **Tiers** (radius, fuel, color, flicker):

  | Item | Radius | Fuel (turns) | Color | Flicker |
  |---|---|---|---|---|
  | Torch | 2 | 2500 | 255 223 117 | fire, 0.125s |
  | Brass Lantern | 3 | 5000 | 255 223 117 | fire, 0.125s |
  | The Star-Glass | 4 | infinite | 5 250 255 | smooth, 0.125s |
  | The Morning Star | 12 | infinite | 255 248 207 | smooth, 0.5s |

- **Fuel:** burns 1 per player turn, only while the light is equipped and the
  player is not standing in a lit room. No burning in town.
- **Burnt out:** the light stays equipped with radius 0. There are no warnings
  and no refuelling; find a new light.
- **Darkness:** with no working light in a dark area the player sees nothing
  but their own tile (no free "adjacent tiles" rule). Lit rooms are seen as before.
- **With a light:** the player sees every tile within the light radius that is
  in line of sight.
- **Memory:** seen tiles are remembered and drawn dim. Lit rooms out of view at
  ×0.5 brightness, dark tiles at ×0.16.
- **Visuals:** brightness falls off as `1 − EaseInExpo(d / (r + 1))`, between
  0.16 and 1.0, tinted by the light's color and its flicker frame. Each tile's
  flicker frame is offset by `x·7 + y·13`, so the edge shimmers instead of
  pulsing as one.
- **Stealth cost:** monsters notice the player within 4 tiles regardless of
  light. A player carrying a working light, or standing in a lit room, can also
  be noticed from 10 tiles with line of sight.
- **Finding lights:** torches and lanterns appear in the item tables and town
  stores; the Star-Glass and Morning Star are deep, rare finds.

### Open

- The flicker (done: 100ms redraw ticker) rx1 used to only redraw on input and
  animations.
- Whether wands of light or the `darkness` ability should also light/unlight
  rooms permanently, as in Rogue.

## First steps either way

- Fix the defense roll (`rpg/rpg.go:164`). Both visions need it.
- Show monster hit chances in the look screen; useful for tuning in both.
