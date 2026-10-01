# Handover: monster abilities

Goal: give rx1's monsters the abilities from Rogue 3.6 and 5.4, plus picks from
the abilities list at https://ruzzoli.de/roguelikes/abilities/.
The Rogue sources are in `~/Games/rogue3.6` and `~/Games/rogue5.4`.
Most of the monster logic is in `fight.c` (`attack()`), `monsters.c` and `chase.c`.

## Done (commit aee824a)

- **On-hit hook.** A monster gets a `hit_effect: <name> | <chance%>` line in
  `data_rx1/definitions/monsters.rec`. The effect is rolled when the monster's
  melee attack lands (`applyHitEffects` in `game/effects_hit.go`, called from
  `actorMeleeAttack`). Monsters with `FlagCancel` skip their effects.
- **Effects:** `rust_armor`, `freeze`, `poison_strength`, `drain_level`,
  `drain_max_hp`, `hold`, `steal_gold`, `steal_item`.
  They are given to: aquator, rust monster, ice monster, giant ant,
  rattlesnake, wraith, vampire, violet fungi, venus flytrap, leprechaun and nymph.
- **Data fixes:** troll and vampire regenerate, and "kestral" is now "kestrel".
  The ice monster no longer has its cold ray. Rust monster and aquator hits
  deal 0 damage.
- **Zap throttle:** a monster in the player's room zaps 1 turn in 5 (`game/ai.go`).
- **Test:** `game/effects_hit_test.go` checks that every `hit_effect` in the
  data file names a real effect.

## Missing: Rogue abilities

| Monster | Ability | Needs |
|---|---|---|
| floating eye (3.6) | Paralyzes you when *you* hit it | Hook 2: an effect that fires when the player hits the monster |
| umber hulk (3.6), medusa (5.4) | Confusion gaze when it first sees you; resisted by a Will roll; once per monster | Hook 3: gaze on sight |
| mimic, xeroc | Disguised as an item until touched | Spawn-time disguise. `xeroc_2` (shown in-game as "xeroc mk ii") already has a wall-mimic AI in `customBehaviours` (`state.go`); reuse that pattern |
| bat (50%), phantom / invisible stalker (20%) | Erratic movement | `erratic` flag in the AI move step |
| orc (5.4), dragon (3.6) | Greedy: guards gold in its room | `greedy` flag; target the room's gold |
| leprechaun, nymph | In Rogue the stolen gold or item is gone for good | Done this way. Could change it so killing the thief gets it back |

## Missing: suggested new abilities (from the abilities list)

| Monster | Ability | Needs |
|---|---|---|
| floating eye, violet fungi, venus flytrap | Stationary (never moves) | `stationary` flag |
| phantom | Moves through walls | `wall_crawler` (flag exists) |
| xorn | Moves through walls; eats gold | `wall_crawler`, plus a new `eat_gold` hit effect |
| umber hulk, purple worm, gnome | Tunnels through rock | `wall_crawler`, or a real dig that leaves corridors |
| purple worm | Poisonous stinger | Hit effect `poison_strength` |
| snake | Poison that does damage over time | New `poisoned` timed flag |
| yeti | Chilling hug: holds you and slows you | `hold`, plus a new `slow` hit effect |
| zombie | Slow mover; its bite makes you hungry | `slow` flag; new `hunger` hit effect |
| griffin | Knockback | New `knockback` hit effect |
| kestrel | Hit-and-run (flees after attacking) | AI: retreat a few turns after hitting |
| quasit | Blink; drains Dexterity | `use_effect: phase_door`; new `drain_dex` hit effect |
| black unicorn | Charge; blinks away when hurt | `zap_effect: charge_attack`; low-HP trigger |
| quagga, emu | Charge | `zap_effect: charge_attack` (exists) |
| centaur | Shoots arrows | `zap_effect: magic_arrow` (exists) |
| kobold | Throws darts | `zap_effect: magic_dart` (exists) |
| jabberwock | Fear on sight | Hook 3 with `scared` |
| wraith, ur-vile | Darkness | New use effect |
| jackal, hobgoblin, ur-vile | Spawn in packs | `group` spawn in `spawnEntities` |
| troll | Revives once after death | Trigger on death (in `actorKilled`) |
| slime | Splits when hit; corrodes your weapon | Trigger when hit; `rust_weapon` hit effect |

**Done (data only):** centaur arrows, kobold darts, emu/quagga/black unicorn
charge, quasit blink, phantom and xorn `wall_crawler`, purple worm
`poison_strength | 30`.

## Hooks to build

1. ~~On-hit effect~~ (done)
2. **Effect when the player hits the monster:** call it from `actorMeleeAttack`
   when the defender is a monster and the hit landed. Same data shape as
   `hit_effect`, e.g. a `struck_effect` field.
3. **Gaze on sight:** check in `defaultBehaviour` when the monster is in the
   player's room. Track a per-monster "already gazed" flag.
4. **Flags and spawn rules:** `erratic`, `stationary`, `greedy`, `disguised`,
   `group`; plus triggers when the monster dies (`revive`) or is hit (`split`).

## Known issues found along the way

- `dodge` and `dr` are read from `monsters.rec` but never applied. Dodge is
  always calculated from speed, and monsters have no natural damage resistance.
  The fix is in `NewEnemyFromDef` (`state.go`).
- `getLevelForExperience` (`data_defs.go`) has no callers, and monsters have no
  XP value for being killed.
- `drain_level` subtracts character points. If the player has already spent
  them, the point balance can go negative. Check that the character screen
  handles a negative balance.
- The new effects are untested in actual play. Fight each of the 12 monsters once
  to check the messages and balance.
