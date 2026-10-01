# Handover: monster abilities

Goal: give rx1's monsters the abilities from Rogue 3.6 and 5.4, plus picks from
the abilities list at https://ruzzoli.de/roguelikes/abilities/.
The Rogue sources are in `~/Games/rogue3.6` and `~/Games/rogue5.4`.

## How abilities are defined (`data_rx1/definitions/monsters.rec`)

| Field | Fires | Code |
|---|---|---|
| `hit_effect: <name> \| <chance%>` | When the monster's melee attack lands | `game/effects_hit.go` |
| `struck_effect: <name> \| <chance%>` | When something hits the monster; the effect targets the attacker | same registry |
| `gaze_effect: confuse \| scare` | Once, when the monster first sees the player; a Will roll resists it | `aiGaze` in `game/abilities.go` |
| `use_effect`, `zap_effect` | 1 turn in 5 while in the player's room | `game/ai.go` |
| `flags:` | see below | `foundation/mapflags.go`, `game/abilities.go` |

New flags: `stationary`, `erratic` (50% for flyers, otherwise 20%), `greedy`
(walks to gold in its room and picks it up while unaware of you), `disguised`
(drawn as a random item and inactive until it is attacked), `group` (spawns 1–3
extra of its kind), `revive` (gets up once at full HP), `tunnel` (digs
corridors), `poisoned` (player status; 1 damage per turn).
`wall_crawler` now moves the monster through rock as well as spawning it in walls.

`dodge` and `dr` from the data file are now applied.

## Who has what

- **Rogue:** floating eye (freezes whoever hits it; stationary), umber hulk and
  medusa (confusion gaze), mimic and xeroc (disguised), bat, phantom and invisible
  stalker (erratic), orc and dragon (greedy), plus the hit effects from commit aee824a.
- **From the abilities list:** violet fungi and venus flytrap (stationary),
  phantom and xorn (through walls), xorn (eats gold), umber hulk, purple worm and
  gnome (tunnel), purple worm (poison sting), snake (poison over time), yeti
  (hold, slow), zombie (slow, hunger), griffin (knockback), kestrel (hit and
  run), quasit (blink, Dexterity drain), black unicorn, quagga and emu (charge),
  black unicorn (blink), centaur (arrows), kobold (darts), jabberwock (fear
  gaze), wraith and ur-vile (darkness), jackal, hobgoblin and ur-vile (packs),
  troll (revives), slime (splits, corrodes weapons).

## Simplifications

- Wall-crawlers and tunnelers step in a straight line toward you. Another actor in
  the way blocks them.
- Greedy monsters take a greedy step toward the gold. That works in open rooms.
- A disguised monster still shows its real name in look and info screens.
- Kestrel "hit and run" sets `scared`. It stays scared until it leaves your room.
- Jabberwock's fear is a stun.
- Black unicorn blinks at random (its `use_effect`), not specifically when hurt.
- Thieves (leprechaun, nymph) vanish with the loot, as in Rogue.

## Still open

- Monsters give no XP. `getLevelForExperience` (`data_defs.go`) has no callers.
  This is a design decision for rx1's character-point system.
- `drain_level` can push character points negative. Check the character screen.
- Nothing here has been tried in actual play yet. Spawn each monster from the
  wizard menu and check messages and balance. Splitting slimes, packs and
  tunneling are the most likely to need tuning.
