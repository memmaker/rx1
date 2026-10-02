# Rogue 3.6 / 5.4 vs rx1: magic items

Sources: `/Users/felix/Games/rogue3.6` (init.c, potions.c, scrolls.c, sticks.c, rings.c, things.c, misc.c, pack.c, rip.c, rogue.h), `/Users/felix/Games/rogue5.4` (extern.c, potions.c, scrolls.c, sticks.c, rings.c, things.c, misc.c, pack.c, rip.c, rogue.h, wizard.c), and the rx1 working tree (`game/`, `data_rx1/definitions/*.rec`). Verdicts: SAME / DIFFERENT / rx1-only.

Notation: Rogue tables are cumulative over `rnd(100)` and sum to 100. rx1 uses `pickWeighted` with relative `chance:` weights, so percentages are `chance / sum`.

## 1. Generation

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Category weights (potion/scroll/food/weapon/armor/ring/stick) | 27/27/18/9/9/5/5 (init.c `things`) | 26/36/16/7/7/4/4 (extern.c `things`) | 26/36/16/7/7/4/4 (spawn_rogue.go `rogueNewThing`) | rx1 = 5.4; 3.6 vs 5.4 DIFFERENT |
| Per-level object chance | `rnd(100) < 35`, MAXOBJ 9 (newlevel.c `put_things`) | `rnd(100) < 36`, MAXOBJ 9 | `rnd(100) < 36`, `rogueMaxObj 9 x scale` | rx1 = 5.4 |
| Treasure room | none | `rnd(TREAS_ROOM)==0` | 1 in 20 (3 in secret level) | rx1 = 5.4 |
| Forced food | `no_food > 3` after `no_food++` per level | same | `levelsWithoutFood > 3`, incremented in dungeon.go:272, reset on food | SAME |
| Fruit | 10% (`rnd(100) > 10` picks ration) | 10% (`rnd(10) != 0`) | none, only `food_ration` | DIFFERENT (rx1 has no fruit) |
| Amulet | level > 25 and not yet held | `level >= 26 && !amulet` | level >= 26 if not held | SAME |
| Weapon curse | | | `r<10` stuck and hit -(rnd(3)+1); `r<15` blessed +(rnd(3)+1); missiles never stick | rx1 mirrors Rogue's 10%/5% |
| Armor curse | 20%/8% | same | `r<20` stuck, plus -(rnd(3)+1); `r<28` plus +(rnd(3)+1) | SAME |
| Ring bonus (prot/str/dex/dmg) | `o_ac = rnd(3)`; 0 becomes -1 and cursed => {-1,1,2}, 1/3 cursed | same | stat dice `1d3-1`; 0 becomes -1 and stuck | SAME |
| Aggravate / teleportation rings | always cursed | always cursed | teleportation always stuck; no aggravate ring | partly rx1-only gap |
| What "cursed" means | ISCURSED: cannot remove ("You can't. It appears to be cursed.") until remove curse | same | `makeStuck`: `stuckTurns` 100..399 equipped turns (state.go:1333), then removable. Not permanent | DIFFERENT |
| Stick charges | `3+rnd(5)` = 3..7; light `10+rnd(10)` | `rnd(5)+3`; light `rnd(10)+10` | `1d8` = 1..8; light `3d10` = 3..30 (wands.rec) | DIFFERENT |
| Wand vs staff | `rnd(100) > 50` metal "wand", else wood "staff"; staff 2d3 / wand 1d1 damage (cosmetic melee) | `rnd(2)==0`; staff "2x3", wand "1x1" | all sticks are "wands"; no staff, no melee damage difference | DIFFERENT (rx1 drops the distinction) |

## 2. Potions

Probabilities as % of all potions (Rogue sums to 100). rx1 sum is 66 (potion_life has chance 0, town store only).

| Potion | 3.6 prob / worth | 5.4 prob / worth | rx1 chance (%) |
|---|---|---|---|
| confusion | 8 / 50 | 7 / 5 | 7 (10.6) |
| paralysis | 10 / 50 | absent | absent |
| poison | 8 / 50 | 8 / 5 | absent |
| hallucination | absent | 8 / 5 | 8 (12.1) |
| gain strength | 15 / 150 | 13 / 150 | absent |
| see invisible | 2 / 170 | 3 / 100 | 3 (4.5) |
| healing | 15 / 130 | 13 / 130 | 13 (19.7) |
| monster detection | 6 / 120 | 6 / 130 | 6 (9.1) |
| magic detection | 6 / 105 | 6 / 105 | 6 (9.1) |
| raise level | 2 / 220 | 2 / 250 | 2 (3.0) |
| extra healing | 5 / 180 | 5 / 200 | 5 (7.6) |
| haste self | 4 / 200 | 5 / 190 | 5 (7.6) |
| restore strength | 14 / 120 | 13 / 130 | absent |
| blindness | 4 / 50 | 5 / 5 | 5 (7.6) |
| thirst quenching | 1 / 50 | absent | absent |
| levitation | absent | 6 / 75 | 6 (9.1) |
| potion of life | absent | absent | 0 (town store only; rx1-only) |

Verdict: the list follows 5.4's chances for the types rx1 kept. Missing in rx1: poison, gain strength, restore strength (no strength-loss mechanic at all), paralysis/thirst (3.6). Because removed types are not renormalised, rx1 healing is about 19.7% versus 13% in Rogue. `worth` in the .rec files is not parsed (`NewItemDefFromRecord` has no worth case).

### Effects and durations

Rogue duration units are command rounds (fuses tick once per round after the `ntimes` loop, so hasted hero actions share a tick). rx1 counters decrement once per player action (`decrementStatusEffects` via `endPlayerTurn`). rx1 `Set` writes 1 then `Increase` adds, so effective duration is value + 1, and re-quaffing resets rather than extends (confusion on the player only `Increase`s, so it stacks).

| Effect | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Healing | `roll(lvl,4)`; if over max, `++max_hp`; `sight()` cures blindness (potions.c `quaff`) | same | heals `maxHP/2`; no max HP gain; does not cure blindness (effects_use.go) | DIFFERENT |
| Extra healing | `roll(lvl,8)`; over max => `++max_hp` | `roll(lvl,8)`; `++max_hp` extra; `come_down` ends hallucination | heals to full `maxHP`; no max HP gain | DIFFERENT |
| Confusion | `rnd(8)+20` = 20..27, lengthens | `spread(20)` = 19..20, lengthens | `Increase(rand.Intn(8)+confuseDuration())`, `confuseDuration = spread(20)` (rx1 spread is +-10%: 18..21), so 18..28; stacks | DIFFERENT (close to 3.6 range) |
| Hallucination | absent | `spread(850)` (808..892) | 8..15 player actions | DIFFERENT (duration far shorter) |
| Blindness | `fuse(sight, 850)` if not blind | `spread(850)`, lengthens | 8..15 actions (+1) | DIFFERENT (850 vs ~12) |
| See invisible | `CANSEE`, `SEEDURATION` 850, not auto-known | `spread(850)`, not auto-known | 8..15 actions; not auto-identified | DIFFERENT duration, SAME ID rule |
| Haste self | `fuse(nohaste, rnd(4)+4)` = 4..7 rounds; if already hasted: faint, `no_command += rnd(8)` | same, also clears ISRUN | 5..9 (+1) actions, speed 10; no faint penalty | DIFFERENT |
| Levitation | absent | `HEALTIME` -> `spread(30)` = 29..31 | 8..15 actions (`FlagFly`) | DIFFERENT |
| Monster detection | shows monsters once; known only if any exist | `SEEMONST` for 20 | `FlagSeeMonsters` 8..15 actions, never auto-ID | DIFFERENT |
| Magic detection | one-time display; `p_know` set whenever level has objects (3.6) | known only if something magic shown | `FlagSeeMagic` until level change; never auto-ID | DIFFERENT |
| Raise level | `exp = e_levels[lvl-1]+1`, `check_level` | same | `RaiseLevel` via `AddExperience`; rx1 ExpLevels 10,20,40,80... vs Rogue 10,20,40,80,160,320,640,1280,2560,5120,10000,20000... in 3.6/5.4 | DIFFERENT table scale (outside scope) |
| Poison | `chg_str(-(rnd(3)+1))` | same + `come_down` | absent | rx1 missing |
| Gain/restore strength | +1 / restore | same (5.4 accounts for rings) | absent | rx1 missing |
| Paralysis | `no_command = HOLDTIME` (2) | absent | absent | 3.6 only |
| Thirst quenching | nothing | absent | absent | 3.6 only |
| Potion of life | absent | absent | max HP +1 and `Heal(1)`; pre-identified | rx1-only |
| Auto-identify | confusion, paralysis, poison, strength, healing, raise, extra healing, haste, blindness known on use; see invis, restore not | similar; confusion not known if hallucinating; monster det. not set | `alwaysIDOnUse`: haste, blindness, healing, extra healing, raise level, hallucination. Never: confusion, detection, levitation, see invisible | DIFFERENT |

## 3. Scrolls

rx1 sum is 108.

| Scroll | 3.6 prob / worth | 5.4 prob / worth | rx1 chance |
|---|---|---|---|
| monster confusion | 8 / 170 | 7 / 140 | 8 |
| magic mapping | 5 / 180 | 4 / 150 | 5 |
| light | 10 / 100 | absent | absent |
| hold monster | 2 / 200 | 2 / 180 | 3 |
| sleep | 5 / 50 | 3 / 5 | 5 |
| enchant armor | 8 / 130 | 7 / 160 | 8 |
| identify | 21 / 100 | potion 10, scroll 10, weapon 6, armor 7, ring/wand/staff 10 = 43 / 80-115 | one scroll, 27 |
| scare monster | 4 / 180 | 3 / 200 | 4 |
| gold detection | 4 / 110 | absent | absent |
| food detection | absent | 2 / 60 | 4 |
| teleportation | 7 / 175 | 5 / 165 | 7 |
| enchant weapon | 10 / 150 | 8 / 150 | 10 |
| create monster | 5 / 75 | 4 / 75 | 5 |
| remove curse | 8 / 105 | 7 / 105 | 8 |
| aggravate monsters | 1 / 60 | 3 / 20 | 4 |
| blank paper | 1 / 50 | absent | absent |
| genocide | 1 / 200 | absent | 1 |
| protect armor | absent | 2 / 250 | absent |
| trap detection, vorpalize weapon (1), fire wall (4) | absent | absent | rx1-only |

Verdict: rx1 chances mostly follow 3.6 (confusion 8, mapping 5, teleport 7, enchant 10/8, create 5, remove curse 8, sleep 5, scare 4, genocide 1). Genocide is 3.6 only (5.4 removed it). Missing: light, gold detection, blank, protect armor, identify variants.

### Effects

| Scroll | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Monster confusion | sets CANHUH on hero; next hit confuses; not known | same (cancel clears it) | `FlagCanConfuse`; next melee hit confuses and clears (actions.go:354); monsters lose it 1/4 per action; not auto-ID | SAME in spirit |
| Magic mapping | maps level; known | also reveals traps; known | maps; always ID | SAME |
| Hold monster | all monsters within +-2 box held (ISHELD, no ISRUN); not known | RUNNING monsters only; known if >= 1 | all visible monsters `FlagHeld`; each has 10% per action to break free (ai.go); ID via witness | DIFFERENT |
| Sleep | hero sleeps `no_command += 4+rnd(5)` (4..8); known | `rnd(5)+4`, clears ISRUN; known | puts all visible monsters to sleep (inverted: a benefit, not a penalty) | DIFFERENT |
| Enchant armor | `o_ac--` (better), uncurses; not known | same | player picks any armor; `plus++` if `plus <= 7` (cap +8); does not uncurse; always ID | DIFFERENT |
| Enchant weapon | uncurses; `rnd(100) > 50` hit else dmg; no weapon: "strange sense of loss" | `rnd(2)==0` hit else dmg; needs a WEAPON wielded | player picks any weapon; coin flip hit/dmg if `max(hit,dmg) <= 7`; no uncurse | DIFFERENT |
| Identify | single scroll, any item type | five typed scrolls | one scroll, identifies one chosen magic item | DIFFERENT from 5.4, SAME shape as 3.6 |
| Scare monster read | laughter only | laughter only | all visible monsters `FlagScared` | DIFFERENT |
| Scare monster on floor | monsters will not step on it; first pickup sets ISFOUND, second pickup turns to dust | ISFOUND set for every object at end of `add_pack`; dust on 2nd pickup | no floor-repel, no dust | DIFFERENT (mechanic missing) |
| Teleportation | random room; known if room changed | same | `phase_door` via `RandomSpawnPosition`; hero confused 1/5; always ID | DIFFERENT |
| Create monster | random adjacent open cell, `randmonster(FALSE)` | avoids scare scroll tile | random free adjacent cell, `rogueRandMonster(level)` | SAME |
| Remove curse | clears ISCURSED on armor, weapon, both rings; not known | same via `uncurse` | player picks one cursed item; clears `stuckTurns`, negative `statBonus` set to 0 (4% chance +1..+3) | DIFFERENT |
| Aggravate | `runto` on all monsters | same | only unsets `FlagSleep` on all monsters; always ID | DIFFERENT |
| Genocide | letter prompt; kills all, removes from spawn tables | absent | monster-type menu, all levels, `g.genocided` | SAME as 3.6 |
| Protect armor | absent | sets ISPROT (blocks rust) | absent | 5.4 only |
| Light / gold detection / food detection | light room; gold det.; absent | absent; absent; food det. | rx1: food detection `FlagSeeFood` until level change; no light scroll | partial |
| Blank | nothing | absent | absent | 3.6 only |
| Naming | "titled '<syllables>'" | same | 33 fixed Latin-like title strings | DIFFERENT (cosmetic) |

## 4. Wands and staffs

rx1 sum is 99; all are "wands". `worth` shown for completeness (rx1 does not use it).

| Stick | 3.6 prob / worth | 5.4 prob / worth | rx1 chance |
|---|---|---|---|
| light | 12 / 120 | 12 / 250 | 12 |
| striking | 9 / 115 | absent | absent |
| invisibility | absent | 6 / 5 | 6 |
| lightning | 3 / 200 | 3 / 330 | 3 |
| fire | 3 / 200 | 3 / 330 | 3 |
| cold | 3 / 200 | 3 / 330 | 3 |
| polymorph | 15 / 210 | 15 / 310 | 15 |
| magic missile | 10 / 170 | 10 / 170 | 10 |
| haste monster | 9 / 50 | 10 / 5 | 10 |
| slow monster | 11 / 220 | 11 / 350 | 11 |
| drain life | 9 / 210 | 9 / 300 | 9 |
| nothing | 1 / 70 | 1 / 5 | absent |
| teleport away | 5 / 140 | 6 / 340 | 6 |
| teleport to | 5 / 60 | 6 / 50 | 6 |
| cancellation | 5 / 130 | 5 / 280 | 5 |

rx1 follows 5.4 (invisibility, no striking, no nothing).

### Effects

| Stick | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Zero charges | "Nothing happens", item stays | same | item is destroyed at 0 charges (`hasPaidWithCharge`, state.go:883) | DIFFERENT |
| No direction given | random direction | random | n/a | n/a |
| Light | lights room | same | lights room | SAME |
| Magic missile | bolt `1d4`, hplus 100, dplus 1 => 2..5; monster `save_throw(VS_MAGIC)` avoids; always known | same, plus wielded weapon's plusses (`o_launch`) | `damageLocation(..., 0)` = 0 damage (likely bug) | DIFFERENT, likely defect |
| Fire / lightning / cold | 6d6, BOLT_LENGTH 6, bounces, monster save, can hit hero (`save(VS_MAGIC)`) | same (`fire_bolt`, dragon bounces fire) | cold `Spread(8,0.35)` 6..10, target Held (frozen), no save, 1/7 bounce. Fire `Spread(10,0.8)` 2..17, 1/20 bounce. Lightning `Spread(7,0.5)` 4..10, chains 50%, 1/20 bounce | DIFFERENT |
| Polymorph | random monster letter, keeps hp ratio | same | `RandomMonsterDef`, no pack retention | similar |
| Teleport away | random room floor | `find_floor`, not hero's position | `phaseDoor` (target also confused) | DIFFERENT detail |
| Teleport to | monster placed next to hero | same | random free cell near zapper | SAME |
| Cancellation | ISCANC, clears ISINVIS | also clears CANHUH, resets disguise | `FlagCancel` (timed for player) | DIFFERENT detail |
| Haste monster | clears ISSLOW if slow, else sets ISHASTE | same | `haste()` unsets `FlagHaste` instead of `FlagSlow` when target is slowed (bug) | DIFFERENT, defect |
| Slow monster | converse, then `runto` | same | correct | SAME |
| Drain life | needs hp >= 2; `cnt = hp/num` before halving => total = full HP (3.6); | halves hero HP first, then `cnt = hp/num` => total = half | user loses `max(1, hp/2)`; each monster in room (or adjacent in corridor) takes `max(1, dmg/n)`; no "too weak" check, can kill the user at 1 HP | 5.4 idea, DIFFERENT safety |
| Striking | adjacent only; 1/20 3d8+9 else 1d8+3 | absent | absent | 3.6 only |
| Invisibility | absent | ISINVIS | `makeInvisible` | SAME as 5.4 |
| Nothing | no effect | no effect | absent | rx1 missing |
| Hold (rx1 zap) | n/a | n/a | held `rand.Intn(10)+5` | rx1-only |
| Auto-ID | MM, fire, cold, lightning known; light | similar | MM, light, cold, fire, lightning `alwaysIDOnUse`; others if effect witnessed | SAME in spirit |
| Charges shown | only when ISKNOW | same | no unknown-charge display rules | DIFFERENT |
| Material names | 3.6: wood=staff, metal=wand | unique materials | 33 woods + 22 metals, all "X wand" | DIFFERENT |

## 5. Rings

| Ring | 3.6 prob / worth | 5.4 prob / worth | rx1 chance |
|---|---|---|---|
| protection | 9 / 200 | 9 / 400 | 9 |
| add strength | 9 / 200 | 9 / 400 | 9 |
| sustain strength | 5 / 180 | 5 / 280 | absent |
| searching | 10 / 200 | 10 / 420 | absent |
| see invisible | 10 / 175 | 10 / 310 | 10 |
| adornment | 1 / 100 | 1 / 10 | absent |
| aggravate monster | 11 / 100 | 10 / 10 | absent |
| dexterity | 8 / 220 | 8 / 440 | 8 |
| increase damage | 8 / 220 | 8 / 400 | 8 |
| regeneration | 4 / 260 | 4 / 460 | 4 (charges `10d3`) |
| slow digestion | 9 / 240 | 9 / 240 | 9 |
| teleportation | 9 / 100 | 5 / 30 | 5 |
| stealth | 7 / 100 | 7 / 470 | absent |
| maintain armor | absent | 5 / 380 | absent |

rx1 chances follow 5.4. 5.4 worth also adds a per-stone value at init.

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Protection / defense | only one ring counts in `roll_em` (else-if LEFT/RIGHT, a bug); no effect on saves | both rings count; also improves `save(VS_MAGIC)` | both rings (`ac - GetStatModifier(StatArmor)`); no save effect | rx1 = 5.4 for armor |
| To-hit/damage ring bonuses | apply when `weap == cur_weapon`, so bare-handed counts | `weap == NULL` skips | apply to all melee including bare-handed and thrown | rx1 closer to 3.6, more generous for throws |
| Strength | `chg_str(o_ac)`, 18/xx supported | cap 3..31, no 18/xx | `GetStrength` adds ring, cap 3..18 | DIFFERENT |
| Teleportation | `rnd(100) < 2` per turn (2%) | `rnd(50)==0` (2%) | 5% per player move (movement.go:106) | DIFFERENT |
| Regeneration | `doctor` faster heal, hunger 2 | same | +1 HP/turn (`FlagRegenerating`, non-stacking) for only 10..30 equipped turns, ring then goes inert | DIFFERENT, rx1-only expiry |
| Slow digestion | `-(rnd(100)<50)` | -2 table entry | `TurnsSinceEating` increments only every other turn | SAME in spirit |
| Searching / stealth / sustain / adornment / aggravate / maintain | present | present | absent | rx1 missing |
| See invisible | sets CANSEE while worn | same | flag while equipped | SAME |
| Hunger from rings | `ring_eat`: regen 2, sustain 1, searching 1/3, slow dig -1/2, others 0 | table: prot 1, str 1, sustain 1, searching 1/3 -3, see inv 1/5 -5, hit/dmg 1/3, regen 2, slow dig -2, stealth 1, sustain armor 1 | no ring cost except slow digestion | DIFFERENT |
| Permanent rings | n/a | n/a | `charges: -1`; counters decrement via `AfterEquippedTurn`, at 0 `GetEquipFlag` returns none | rx1-only |
| Auto-ID | not on put on; "call it" prompt | same | all rings identified on equip (`alwaysIDOnUse`); no "call it" | DIFFERENT |
| Bonus display | only if ISKNOW | same | n/a | |

## 6. Identification, naming, worth, score

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Flavors | scroll syllables, potion colors, ring stones, stick materials | same | 27 potion colors, 26 ring stones, 33 scroll titles (fixed list), 33 woods + 22 metals | DIFFERENT counts |
| Guess / call it | yes | yes (`call_it`, oi_guess) | none | rx1 missing |
| Identify | `whatis()` any type | `whatis(insist,type)`, `set_know` | one-scroll, any one magic item; weapons/armor `isKnown` per stack | see scrolls |
| Score | `total_winner`: potions/scrolls `worth x count` (all set known, no halving); rings `worth + o_ac*20` (stat rings, > 0 else 50); sticks `worth + 20*charges`; amulet 1000; food `2*count` | same style but halves worth for unidentified; rings `worth + o_arm*100` (else 10); weapons `worth x (3*(hplus+dplus)+count)`; armor `base + (9-o_arm)*100 + 10*(a_class-o_arm)` | score is gold only (`calculateTotalNetWorth`); `worth` in .rec is not parsed; town prices flat in town.go `itemPrice`: potion_life 500, missile 2, armor 80, weapon 50, document 40, light 30, food 10, other 25; sell for half | DIFFERENT |
| Enchant limits | none enforced beyond int | none | armor +8 max, weapon 8 max per stat via `<= 7` check | rx1-only |

## 7. Food, hunger, amulet

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Stomach | `food_left`, HUNGERTIME 1300, STOMACHSIZE 2000, MORETIME 150 | same | `TurnsSinceEating`; `FlagHunger` +1 per 300 turns; no stomach value | DIFFERENT |
| Eating a ration | `+= 1300 + rnd(400) - 200` (1100..1499), cap 2000; 30% "awful" gives exp++ | same, resets negative `food_left` to 0 first | `Satiate()` clears hunger fully; no cap, no exp | DIFFERENT |
| Hungry/weak/faint | HUNGERTIME warn, MORETIME weak, 0 faint (`rnd(100) > 20` returns, so ~21% faint, `no_command = rnd(8)+4`) | `rnd(5) != 0` returns (20% faint), `no_command += rnd(8)+4` | hungry blocks natural regen and costs fatigue every 3x`healInterval`; no weak or faint state | DIFFERENT |
| Starvation | none in 3.6 | death at `food_left < -STARVETIME` (850) | none | 3.6 = rx1 (no death) |
| Digestion | `-= ring_eat(L)+ring_eat(R)+1-amulet` | same | 1 per turn, 0.5 with slow digestion | DIFFERENT |
| Amulet | worth 1000; picking up sets `amulet = TRUE`, never reset; hunger reduced by 1 | dropping resets `amulet`; hunger offset | spawns level >= 26; win by climbing from level 1 with it (actions.go:285); no hunger effect | win condition SAME; hunger effect DIFFERENT |

## 8. 3.6 vs 5.4 differences (where it matters here)

- Categories 27/27/18/9/9/5/5 vs 26/36/16/7/7/4/4; object chance 35% vs 36%; no treasure rooms in 3.6.
- Potions: 3.6 has paralysis and thirst quenching; 5.4 has hallucination and levitation. Durations: confusion `rnd(8)+20` vs `spread(20)`; blindness/see-invisible 850 vs `spread(850)`.
- Scrolls: 3.6 has light, gold detection, blank, genocide, one identify (21); 5.4 has protect armor, food detection, five identify variants (43).
- Sticks: 3.6 has striking and nothing weight 1 each with different probs; 5.4 has invisibility and a different material scheme. Drain life: full HP vs half HP total.
- Rings: 3.6 lacks maintain armor; ring of protection only works for one hand and not for saves; worth values 5.4 roughly doubled; ring aggravate/teleport weights 11/9 vs 10/5.
- Healing: 5.4 `check_level` no longer caps HP; extra healing max-HP bump differs slightly.
- Starvation death exists only in 5.4.

## 9. Main rx1 differences and probable defects

1. Heal potions restore a fixed fraction (1/2 and full) instead of `roll(lvl,4/8)`, and never raise max HP or cure blindness.
2. Magic missile wand deals 0 damage (`damageLocation(..., 0)`), versus 2..5 plus save.
3. `haste()` on a slowed target unsets the wrong flag; slowed monsters stay slowed.
4. Drain life has no "too weak" guard and can kill a 1 HP user.
5. Empty wands are destroyed; charges are 1..8 (light 3..30) instead of 3..7 (10..19).
6. Sleep scroll, scare scroll and aggravate scroll have inverted or reduced semantics; no scare-scroll floor mechanic.
7. Curses are timed (`stuckTurns` 100..399) and remove curse / enchant behave differently from Rogue's flag-clearing.
8. Many item types are absent: poison, strength, restore strength, light scroll, gold detection, protect armor, blank, wand of nothing, rings of searching/stealth/sustain/adornment/aggravate/maintain armor.
9. Chances are not renormalised after removals, so surviving items are more common (healing about 20%, identify 25%).
10. Ring of regeneration, and all non-permanent rings, expire after equipped turns (rx1-only); the ring teleportation rate is 5% against Rogue's 2%.
11. Durations for blindness, hallucination, see invisible, levitation are 8..15 actions against 850/30 rounds in Rogue.
12. `worth` fields are ignored and score is gold only; no hunger/stomach model, fainting, or amulet hunger effect.
13. rx1-only items: potion of life, scrolls of trap detection, vorpalize weapon, fire wall; cold ray freezes the target.
