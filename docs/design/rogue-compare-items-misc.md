# Rogue 3.6 vs Rogue 5.4 vs rx1: weapons, armor, inventory, interface

Sources read: `/Users/felix/Games/rogue3.6` (an RVIP port: adds `x` explore, `>`/`<` walk-to-stairs, Enter menu, inventory menu, cursor-list get_item), `/Users/felix/Games/rogue5.4` (same RVIP additions), and the rx1 working tree at `/Users/felix/Projects/rx1` (uncommitted changes included).
Verdicts: SAME / DIFFERENT / rx1-only. "3.6 vs 5.4" differences are flagged in the verdict column as `3.6!=5.4`.

Conventions. AC in Rogue: lower is better, start 10. rx1 stores `protection` (= 10 - AC, higher is better, as 5.4 displays `[protection N]`) plus `plus`. 3.6 dice are `NdM`, 5.4 `NxM`.

Provenance of rx1: combat math (`rpg/rogue.go`) is 3.6-style; level population (`game/spawn_rogue.go`) and item weights are 5.4; a +4 to-hit vs sleeping/held is 5.4.

---

## 1. Weapon table

### 1.1 Rogue 3.6 (`weapons.c: init_dam, w_names`; `rip.c: total_winner` for worth)

12 weapons, no per-weapon probability: `new_thing` picks `rnd(MAXWEAPONS)`, so each is 1/12.

| Weapon | Melee | Thrown | Launcher | Flags | Worth |
|---|---|---|---|---|---|
| mace | 2d4 | 1d3 | - | - | 8 |
| long sword | 1d10 | 1d2 | - | - | 15 |
| long bow | 1d1 | 1d1 | - | - | 75 |
| arrow | 1d1 | 1d6 | BOW | ISMANY, ISMISL | 1 |
| dagger | 1d6 | 1d4 | - | ISMISL | 2 |
| rock | 1d2 | 1d4 | SLING | ISMANY, ISMISL | 1 |
| two handed sword | 3d6 | 1d2 | - | - | 30 |
| sling | 0d0 | 0d0 | - | - | 1 |
| dart | 1d1 | 1d3 | - | ISMANY, ISMISL | 1 |
| crossbow | 1d1 | 1d1 | - | - | 15 |
| crossbow bolt | 1d2 | 1d10 | CROSSBOW | ISMANY, ISMISL | 1 |
| spear | 1d8 | 1d6 | - | ISMISL | 2 |

### 1.2 Rogue 5.4 (`extern.c: weap_info`, `weapons.c: init_dam`)

9 weapons (plus a fake 10th, dragon breath). Probabilities sum to 100, chosen by `pick_one`.

| Weapon | Melee | Thrown | Launcher | Flags | Prob | Worth |
|---|---|---|---|---|---|---|
| mace | 2x4 | 1x3 | - | - | 11 | 8 |
| long sword | 3x4 | 1x2 | - | - | 11 | 15 |
| short bow | 1x1 | 1x1 | - | - | 12 | 15 |
| arrow | 1x1 | 2x3 | BOW | ISMANY, ISMISL | 12 | 1 |
| dagger | 1x6 | 1x4 | - | ISMISL | 8 | 3 |
| two handed sword | 4x4 | 1x2 | - | - | 10 | 75 |
| dart | 1x1 | 1x3 | - | ISMANY, ISMISL | 12 | 2 |
| shuriken | 1x2 | 2x4 | - | ISMANY, ISMISL | 12 | 5 |
| spear | 2x3 | 1x6 | - | ISMISL | 12 | 5 |

### 1.3 rx1 (`data_rx1/definitions/weapons.rec`; selection `game/spawn_rogue.go: pickWeighted`)

16 weapons. `chance` is a weight (pickWeighted does not assume sum 100). No worth field, no group-size field (missiles are individual items stacked in inventory).

| Name | Slot | Chance | Melee | Thrown | Launcher |
|---|---|---|---|---|---|
| main gauche | one-handed | 8 | 1d4 | 1d3 | - |
| mace | one-handed | 11 | 2d4 | 1d3 | - |
| long sword | one-handed | 11 | 3d4 | 1d2 | - |
| short bow | launcher | 12 | 1d1 | 1d1 | - |
| arrow | quiver | 12 | 2d3 | 1d1 | bow |
| battle axe (internal id `dagger`) | one-handed | 8 | 1d6 | 1d4 | - |
| two handed sword | two-handed | 10 | 4d4 | 1d2 | - |
| dart | quiver | 12 | 1d1 | 1d3 | - |
| crossbow | launcher | 12 | 1d1 | 1d1 | - |
| crossbow bolt | quiver | 12 | 1d10 | 1d2 | crossbow |
| spear | one-handed | 12 | 2d3 | 1d6 | - |
| overkill | one-handed | 1 | 5d8+1 | 1d6 | - |
| club | one-handed | 11 | 1d6 | 1d3 | - |
| axe | one-handed | 10 | 1d8 | 1d3 | - |
| rapier | one-handed | 8 | 1d6 | 1d2 | - |
| whip | one-handed | 8 | 1d4 | 1d1 | - |

### 1.4 Weapon comparison by topic

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Count | 12 | 9 | 16 | DIFFERENT (all three) |
| Mace | 2d4 / 1d3 | 2x4 / 1x3 | 2d4 / 1d3, chance 11 | SAME |
| Long sword | 1d10 | 3x4 | 3d4, chance 11 | rx1 = 5.4; 3.6!=5.4 |
| Two-handed sword | 3d6 | 4x4 | 4d4, chance 10 | rx1 = 5.4; 3.6!=5.4 |
| Bow | "long bow" | "short bow" 15 worth | short bow, chance 12 | rx1 = 5.4 |
| Arrow | 1d1 melee, 1d6 thrown | 1x1 melee, 2x3 thrown | 2d3 (shot dmg) as melee/launched; 1d1 hand-thrown | rx1 = 5.4 values when fired; naming of fields differs |
| Dagger | 1d6 / 1d4 | 1x6 / 1x4 | 1d6/1d4 exists only as "battle axe" (id `dagger`) | DIFFERENT name, chance 8 = 5.4 |
| Spear | 1d8 / 1d6 | 2x3 / 1x6 | 2d3 / 1d6, chance 12 | rx1 = 5.4; 3.6!=5.4 |
| Dart | 1d1 / 1d3 | 1x1 / 1x3 | 1d1 / 1d3, chance 12 | SAME |
| Sling, rock | yes | no | no | rx1 = 5.4 |
| Crossbow, bolt | yes (1d1, bolt 1d2/1d10) | no | yes (bolt 1d10/1d2) | rx1 = 3.6 |
| Shuriken | no | yes 1x2/2x4 | no | rx1 = 3.6 |
| Added by rx1 | - | - | main gauche, overkill, club, axe, rapier, whip | rx1-only |
| Probability | uniform 1/12 | table, sum 100 | weights (sum is not 100) | rx1 uses 5.4-style weighted pick |
| Worth | per table above | per table above | none | rx1-only absence |
| Group sizes | ISMANY: `rnd(8)+8`, `newgrp()`; dagger not grouped | ISMANY `rnd(8)+8`; dagger `rnd(4)+2` grouped | no group concept; stacking only (`Item.CanStackWith`) | DIFFERENT |
| Flags | ISMANY/ISMISL | same | `two_handed`, slot types; missile = quiver slot | DIFFERENT |
| Weapon worth formula | `base*(1+10*hplus+10*dplus)*count` | `base*(3*(hplus+dplus)+count)` | none | 3.6!=5.4; rx1 none |

### 1.5 Thrown/fired damage selection (`fight.c: roll_em`; rx1 `game/actor.go: GetMelee, GetThrowing`)

| Case | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Missile with matching wielded launcher | `o_hurldmg` + launcher hplus/dplus | same | missile melee dice (e.g. arrow 2d3) + launcher plusses | SAME in effect |
| Hand-thrown ISMISL (dagger, dart, spear) | uses melee dice `o_damage` (`cp = ISMISL ? o_damage : o_hurldmg`) | uses thrown dice `o_hurldmg` | uses `thrownDamage` | rx1 = 5.4; 3.6!=5.4 |
| Hand-thrown non-missile (mace) | thrown dice | thrown dice (o_launch<0) | thrown dice | SAME |
| Arrow thrown by hand | 1d1 (melee dice) | 1x1... uses `o_hurldmg` only if `o_launch<0`; arrow has launcher so 1x1 | 1d1 | SAME |
| To-hit | `rnd(20)+1+wplus >= (21-lvl)-arm` | `rnd(20)+wplus >= (20-lvl)-arm` (equivalent) | `rand.Intn(20)+1+wplus >= 21-lvl-arm` (`rpg/rogue.go: Swing`) | SAME |
| +4 vs sleeping/held | no | yes (defender not ISRUN) | yes (`game/actions.go: rollAttack`) | rx1 = 5.4 |
| Ring bonuses to hit/dmg | else-if per hand (only one ring of protection counted) | both rings apply | rings add to all attacks, both rings | rx1 = 5.4 |
| `str_plus`/`add_dam` | functions with 18/xx handling | tables, 18/xx mapped to 19..31 | `rpg.StrPlus/AddDam`, strength clamped 3..18, no 18/xx; AddDam <6:-1, <16:0, <18:+1, else +2 | DIFFERENT (rx1 differs from both for mid strengths) |
| Missile breakage | none (lands via `fall`) | none | none, placed on map at target (`addItemToMap`) | SAME |

---

## 2. Armor table

| Armor | AC 3.6 | AC 5.4 | rx1 protection (AC) | Prob 3.6 (cumulative) | Prob 5.4 | rx1 chance | Worth 3.6 | Worth 5.4 | rx1 worth |
|---|---|---|---|---|---|---|---|---|---|
| leather armor | 8 | 8 | 2 (8) | 20 (20) | 20 | 20 | 5 | 20 | - |
| ring mail | 7 | 7 | 3 (7) | 15 (35) | 15 | 15 | 30 | 25 | - |
| studded leather armor | 7 | 7 | 3 (7) | 15 (50) | 15 | 15 | 15 | 20 | - |
| scale mail / scale armor | 6 | 6 | 4 (6) | 13 (63) | 13 | 13 | 3 | 30 | - |
| chain mail | 5 | 5 | 5 (5) | 12 (75) | 12 | 12 | 75 | 75 | - |
| splint mail (rx1: steel breastplate) | 4 | 4 | 6 (4) | 10 (85) | 10 | 10 | 80 | 80 | - |
| banded mail (rx1: lorica segmentata) | 4 | 4 | 6 (4) | 10 (95) | 10 | 10 | 90 | 90 | - |
| plate mail | 3 | 3 | 7 (3) | 5 (100) | 5 | 5 | 400 | 150 | - |

(corrected: re-verified against init.c, extern.c arm_info and armor.rec; AC and probabilities of every armor, including scale 13 / chain 12, are identical in all three; armor.rec merely lists chain before scale.)

Sources: 3.6 `init.c: a_class, a_names, a_chances`; 5.4 `extern.c: arm_info`; rx1 `data_rx1/definitions/armor.rec`.

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| AC values and probabilities | table above | identical | identical (as protection = 10-AC) | SAME |
| Worth | `base*(1+10*(a_class-o_ac))`, base values above | `base + (9-o_arm)*100 + 10*(a_class-o_arm)` | none | 3.6!=5.4; rx1 has no worth |
| Display | `a_class - o_ac` as plus | "+N name [protection M]", `M = 10-o_arm` | "+N name" and `[prot]` | rx1 = 5.4 |
| Slots | one body armor | one | all armor in `torso`; head/hands/feet/back slots exist in `foundation/slots.go` but no data uses them | rx1-only (unused) |
| Armor class | `AC = o_ac` of worn armor, rings subtract | same | `GetArmorClass = min(ac,10) - protection` per piece minus ring modifier (`game/actor.go`) | SAME |
| Starting armor | ring mail, `o_ac = a_class-1` (AC 6) | same | ring mail +1 (protection 3 + plus 1 = AC 6) | SAME |

---

## 3. Bonus generation (cursed / magical chance)

| Category | 3.6 (`things.c: new_thing`) | 5.4 (`things.c: new_thing`) | rx1 (`game/spawn_rogue.go`) | Verdict |
|---|---|---|---|---|
| Weapon | `k=rnd(100)`: `<10` cursed, `hplus -= rnd(3)+1`; `<15` `hplus += rnd(3)+1`. No dplus on generation | same | `r=rnd(100)`: `<10` stuck, `hitPlus -= rnd(3)+1`; `<15` `hitPlus += rnd(3)+1`. Missiles same but never stuck | SAME (10% cursed / 5% good) |
| Armor | `k<20` cursed, `ac += rnd(3)+1`; `<28` `ac -= rnd(3)+1` | same (o_arm) | `r<20` stuck, `plus -= rnd(3)+1`; `<28` `plus += rnd(3)+1` | SAME (20% / 8%) |
| Rings: str/prot/hit/dam | `o_ac = rnd(3)`; 0 becomes -1 and cursed | same | `stat_bonus 1d3-1`; 0 becomes -1 and stuck | SAME |
| Ring aggravate / teleport | cursed | cursed | teleportation stuck (aggravate not in rx1) | SAME / rx1 lacks aggravate |
| Missing rx1 rings | - | maintain armor, sustain str, damage rings exist (14 rings) | rings.rec header lists missing: sustain strength, aggravate, maintain armor | rx1 subset |
| Wand/staff charges (`fix_stick`) | `3+rnd(5)`; light `10+rnd(10)`; striking 3,3 d8 | light `rnd(10)+10`, others `rnd(5)+3`; staff 2x3, wand 1x1 | per-item `charges` dice in data | 3.6!=5.4; rx1 data-driven |

---

## 4. Enchant limits and rust

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Enchant armor scroll | `o_ac--`, uncurse; no cap | `o_arm--`, uncurse; no cap | `AddEnchantment` (+1 plus); allowed only if `plus <= 7` (`ArmorInfo.IsEnchantable`, `game/item.go`); player picks armor from menu | DIFFERENT (rx1 has cap at +8; menu pick) |
| Enchant weapon scroll | uncurse; `rnd(100)>50` hplus++ else dplus++ | same 50/50, uncurse | 50/50 hit/damage, only if `max(hitPlus,damagePlus) <= 7` | DIFFERENT (cap) |
| Remove curse | uncurses armor, weapon, both rings | `uncurse` same set | `Item.RemoveCurse` clears stuck, negative bonus to 0, 4% chance of +1..+3 blessing; effect via `removeCurse` (`game/effects_use.go`) acting on chosen/unknown items | DIFFERENT (blessing chance, rx1-only) |
| Protect armor scroll | none | yes (sets ISPROT) | none found | rx1 = 3.6 |
| Rust source | rust monster 'R'; no traps | aquator 'A' and rust trap (`move.c: rust_armor`) | `rust_armor` hit effect (`game/effects_hit.go`); `rust_weapon` struck effect corrodes weapon (`WeaponInfo.Corrode`, damagePlus down to -3) | rx1 adds weapon corrosion (rx1-only) |
| Rust rule | if `o_ac < 9`, `o_ac++`; leather can rust | `o_arm++` unless `o_arm >= 9`; leather immune; ISPROT or ring of maintain armor negates | `ArmorInfo.Rust` lowers plus while `GetProtection() > 0` | DIFFERENT; 3.6!=5.4 |
| Vorpal | none | none | `vorpalize` scroll: +4/+4 vs named enemy, +1/+1 otherwise (Brogue-style) | rx1-only |

---

## 5. Pack, stacking, letters, weight, drop/throw/pickup

| Topic | 3.6 (`pack.c`) | 5.4 (`pack.c`) | rx1 | Verdict |
|---|---|---|---|---|
| Max pack | `MAXPACK 23`; blocks at `inpack == MAXPACK-1`, effective 22 | `pack_room`: `++inpack > MAXPACK`, 23 | `NewInventory(23)`, counted in stacks (`IsFull`: stacks == 23) | rx1 = 5.4 count, but stack-based; 3.6!=5.4 |
| Stack counting | ISMULT (potion/scroll/food) stacks by `which`, each unit increments `inpack`; groups (`o_group`) share a slot | same, per unit for ISMULT, per slot for groups | any number of identical items in one slot | DIFFERENT (unlimited stack counts) |
| Weight | none | none | none | SAME |
| Letters | positional, shift when items removed | stable per item (`o_packch`, `pack_used[]`) | letters via inventory menu (not verified in detail) | 3.6!=5.4 |
| Pickup | automatic on walking | automatic; `,` picks up; `m <dir>` moves without picking up; blocked while levitating | automatic (`AutoPickup: true` in `config.rec`) and `,` command; gold goes straight to purse; message "You cannot carry any more items" when full | rx1 = 5.4 (`,`); no `m` |
| Scare monster pickup | second pickup turns to dust | same | scare via scroll only (not verified) | not verified |
| Stack rule | `which` equal, ISMULT; weapon groups | same | `CanStackWith`: same name+category, not non-missile weapons/armor/rings, same effects and charges, missiles equal plusses | DIFFERENT detail |
| Drop | non-weapon with count>=2 drops one; weapon stack drops entirely | `leave_pack(obj,TRUE,!ISMULT)`: whole weapon/armor stack, one potion/scroll/food | drop costs a turn (`endPlayerTurn`); equipped stuck item refused | SAME-ish |
| Throw | `missile()`: one WEAPON-type item | `leave_pack(obj,TRUE,FALSE)` one item | any item throwable (`IsThrowable` always true); `v` throw; `f` launch; `h` quick shot at nearest; `g` aim | DIFFERENT (rx1 throws anything; launching commands rx1-only) |
| Food position | kept at front of pack | no | not applicable | 3.6 only |

---

## 6. Wield, wear, take off, rings

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Wield | cursed current weapon blocks switch | same | `CanEquip/CanUnequip = !IsStuck()`, message "You cannot remove this item" | SAME |
| Turn cost | wield: free? `wear`/`take_off` use `waste_time` | same; wear picks armor first then "already wearing some" | equip/unequip does NOT call `endPlayerTurn` (no turn cost) | DIFFERENT (rx1 free) |
| Delay | `waste_time` (one turn plus daemons) | same | none | DIFFERENT |
| Ring slots | two hands L/R; `ring_on` asks hand if both free | same | two ring slots, left then right (`RingLeft`, `RingRight`) | SAME |
| Cursed stickiness | cannot remove cursed weapon/armor/ring ("You can't. It appears to be cursed.") until remove curse | same | stuck for 100-399 equipped turns (`Item.stuckTurns`, `makeStuck`; countdown `AfterEquippedTurn`), also lifted by remove curse | DIFFERENT (timed curse, rx1-only) |
| IsCursed | cursed flag | cursed flag | stuck or `statBonus < 0` | DIFFERENT |
| Armor identified on wear | ISKNOW set | similar | `isKnown` set on equip | SAME |
| Ring identified | by effect | by effect | `always_id_on_use: true` | DIFFERENT |
| Ring food use | `ring_eat` | `ring_eat` table (PROTECT 1, ADDSTR 1, SUSTSTR 1, SEARCH -3, SEEINVIS -5, ADDHIT -3, ADDDAM -3, REGEN 2, DIGEST -2, STEALTH 1, SUSTARM 1; negative = 1-in-N) | slow digestion halves hunger; regeneration/see invisible rings have `charges` and burn out | DIFFERENT (rx1-only burnout) |
| Ring teleport | `rnd(100) < 2` per turn | `rnd(50) == 0` (2%) | teleportation ring exists (mechanics not verified) | 3.6!=5.4 (equal rate) |
| Slots in rx1 | - | - | main hand, off hand, two-handed, launcher, quiver, light source, amulet, torso (and unused head/hands/feet/back) | rx1-only |

---

## 7. `new_thing` category probabilities and food

| Category | 3.6 | 5.4 | rx1 (`rogueNewThing`) | Verdict |
|---|---|---|---|---|
| potion | 27 | 26 | 26 | rx1 = 5.4 |
| scroll | 27 | 36 | 36 | rx1 = 5.4 |
| food | 18 | 16 | 16 | rx1 = 5.4 |
| weapon | 9 | 7 | 7 | rx1 = 5.4 |
| armor | 9 | 7 | 7 | rx1 = 5.4 |
| ring | 5 | 4 | 4 | rx1 = 5.4 |
| stick | 5 | 4 | 4 (wands) | rx1 = 5.4 |
| amulet | placed if `level > 25 && !amulet` | `level >= 26` | `level >= 26` | rx1 = 5.4 |

Food guarantee (`no_food`): 3.6 `no_food++` in `new_level` (newlevel.c:39), forced if `no_food > 3`, reset on food draw. 5.4 same. rx1: `levelsWithoutFood > 3` forces food, resets on a food draw. SAME.
Food type: 3.6 `rnd(100) > 10` ration else fruit; 5.4 `rnd(10) != 0` ration else fruit (the same 10% fruit, different RNG call); rx1 only "ration of food" (`satiate_fully`), no fruit: DIFFERENT.

`put_things` / level items: 3.6 MAXOBJ 9 attempts at `rnd(100) < 35`; 5.4 9 attempts at `rnd(100) < 36`; rx1 `rogueMaxObj*scale` attempts at `rnd(100) < 36` (scale = room-count factor): rx1 = 5.4; 3.6!=5.4.
Treasure room: 3.6 none; 5.4 `rnd(20)==0`, MINTREAS 2, MAXTREAS 10, monsters from next level; rx1 same constants, secret level gets 3: rx1 = 5.4.
Gold: 3.6 `rnd(100) < 50 && (!amulet || level >= max_level)`, `rnd(50+10*level)+2`; 5.4 `rnd(2)==0` same conditions; rx1 same as 5.4: SAME.
Traps: rx1 `rnd(10) < level`, count `min(rnd(level/4)+1,10)*scale`. rx1-only extras: lore documents (1-2 per level), 3.6 and 5.4 none.

---

## 8. Light sources

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Light items | none (lit rooms; wand of light only) | none | `lights.rec`: torch (fuel 800, radius 1), lantern (1000-1200, 2), brass lantern (1000-1200, 3), the Star-Glass (infinite, 4), the Morning Star (infinite, 12) | rx1-only |
| Generation | - | - | `rollLights` per new level: torch 30%; lantern from level 5, 21%; Star-Glass level >8, 10%, once per run; Morning Star level >10, 5%, once per run | rx1-only |
| Fuel | - | - | burns 1/turn (`burnPlayerLight`), not in lit rooms or the town; player starts with a torch | rx1-only |
| Vision | rooms lit/dark; corridor sight 1 | same | radius governs FoV and `enemyCanSpotPlayer` (within 4 always, up to 10 with LOS if light shown) | rx1-only |
| Wand/scroll of light | wand `light` charges `10+rnd(10)` | same | `light` effect | SAME |

---

## 9. Hunger and eating

| Topic | 3.6 (`misc.c: eat`, `daemons.c: stomach`) | 5.4 | rx1 (`game/deamons.go`) | Verdict |
|---|---|---|---|---|
| Food value | `food_left += HUNGERTIME(1300) + rnd(400) - 200`, cap STOMACHSIZE 2000 | `food_left += HUNGERTIME-200+rnd(400)`, first raised to 0 if negative; cap 2000 | `satiate_fully` resets hunger counter | DIFFERENT |
| Hungry | below 300 | below 300 | hunger increments every 300 turns since eating (`hungerInterval 300`), message "You are hungry." | DIFFERENT |
| Weak / Faint | weak <150; faint <=0, about 21%, `rnd(8)+4` turns | faint `rnd(5)==0` (20%), `rnd(8)+4` | none found | rx1 lacks stages |
| Starvation | no death | death at `food_left < -STARVETIME (850)` | none found | 3.6!=5.4; rx1 none |
| "Yuk" exp | `rnd(100)>70` gives exp++ | similar for fruit | not found | not verified |
| Effects of hunger | weaker | weaker | blocks natural healing, drains fatigue (rx1 FP) | rx1-only |
| Natural heal | level-based interval | level-based | `max(3, 20 - 2*level)` turns when no enemy visible | DIFFERENT |

---

## 10. Starting kit, stats, level-up

| Topic | 3.6 (`main.c, init.c`) | 5.4 (`init.c: init_player`) | rx1 (`game/actor.go: NewPlayer`, `game/state.go: init`) | Verdict |
|---|---|---|---|---|
| Weapon | mace +1,+1 wielded | same | mace +1,+1 | SAME |
| Armor | ring mail, `a_class-1`, worn | same | ring mail +1 | SAME |
| Missile | long bow +1,+0; arrows `25+rnd(15)` | short bow +1,+0; arrows `rnd(15)+25` | short bow +1 hit; arrows `25+rand.Intn(15)` | rx1 = 5.4 name |
| Food | 1 | 1 | 1 food ration | SAME |
| Extra | - | - | torch; player knows `potion_life` (bought in town) | rx1-only |
| Str | 16 (1%: 18/rnd(100)+1) | 16 fixed | 16 | rx1 = 5.4 |
| HP / AC / unarmed | 12 / 10 / 1d4 | 12 / 10 / 1x4 | 12 / 10 / 1d4 | SAME |
| Extra stat | - | - | fatigue points (FP 3) | rx1-only |
| XP table | 10, 20, 40, 80, ... 20480, 40920, 81920, ..., 2621440 | 10,20,40,80,160,320,640,1300,2600,5200,13000,26000,50000,100000,200000,400000,800000,2000000,4000000,8000000 | `rpg.ExpLevels` = 3.6 table (including 40920) | rx1 = 3.6; 3.6!=5.4 |
| Level-up HP | `roll(n,10)`, capped at max_hp | `roll(n,10)` added to max and current, no cap | `AddExperience`: `roll(n,10)` on gain, no cap | rx1 = 5.4 |
| Wraith drain | 15% | similar | `DrainLevel` | SAME |
| Saving throw | `14 + which - lvl/2` | same | `d20 >= 14 + which - lvl/2` (`rpg.Save`) | SAME |
| Strength limits | 3..31 w/ 18/xx | same | `ChangeStrength` clamps 3..18 | DIFFERENT |
| Level-up message | "welcome to level N" | same | not verified | not verified |
| Floating eye death | `no_command > 100 && food_left <= 0` | - | hold effects only | not verified |

---

## 11. Commands

rx1 keymap: `data_rx1/keymaps/default.txt` (WASD) and `data_rx1/keymaps/rogue.txt` (vi keys, Rogue letters).

| Action | 3.6 / 5.4 | rx1 default | rx1 rogue keymap | Verdict |
|---|---|---|---|---|
| Move | hjklyubn | wasd + numpad | hjklyubn + numpad | rogue keymap SAME |
| Run | HJKLYUBN (also 3.6 `f`+dir; 5.4 Ctrl-dir, `F` fight) | WASD, `5` run direction | HJKLYUBN, `g` run direction | SAME / different keys |
| Stairs | `>` `<` (5.4 adds `<` only with amulet at 1 in original; RVIP walk to stairs) | `>` `<`, Enter | `>` `<`, Enter | DIFFERENT (rx1 `<` works any time: level 1 up goes to town; with amulet wins) |
| Throw | `t` | `v` | `t` | SAME in rogue keymap |
| Eat / quaff / read | `e` / `q` / `r` | `e` use (all items), `r` read | `e` eat, `q` quaff, `r` read | SAME in rogue keymap |
| Zap | `z` (3.6 `p` = zap in direction) | `e` use / `g` aim | `z` zap, `s` use, `a` apply | DIFFERENT |
| Wield/wear/off | `w` `W` `T` | via `i` inventory menu | `w` `W` `T` | SAME in rogue keymap |
| Rings | `P` `R` | via inventory | `P` `R` | SAME in rogue keymap |
| Drop | `d` | inventory menu | `d` | SAME |
| Inventory | `i`, `I` (3.6 `I` = single item) | `i`; `I` = items in view | `i` | DIFFERENT |
| Search | `s`; space rests (3.6); `.` rest (5.4) | no search command; `.` wait also searches | `.` wait (`s` is `use`) | DIFFERENT |
| Pickup | automatic (3.6); `,` and `m` (5.4) | `,` + auto | `,` + auto | rx1 = 5.4 partly |
| Look / identify | `/` | `x` look (cursor) | `/` look | DIFFERENT mechanism |
| Character | - | `c` | `C` | rx1-only |
| Messages log | Ctrl-P (5.4) | `l` | `o` | DIFFERENT |
| Discoveries | `D` (5.4) | none found | none found | rx1 lacks |
| Call / name item | `c`, 5.4 `c` | none found | none found | rx1 lacks |
| Repeat | `a` (5.4), counts (3.6 and 5.4) | none found | none found | rx1 lacks |
| Fight to death | 5.4 `F`/`f` | none | none | rx1 lacks |
| Version / help | `v`, `?` | `?` and F2 manual, `=` key bindings | same | SAME-ish |
| Save | `S` | none found | none found | rx1 lacks (not verified) |
| Quit | `Q` | `Q` | `Q` | SAME |
| Redraw | Ctrl-L | none | none | rx1 lacks |
| Explore | RVIP `x` | `o` auto explore | none listed | rx1-only |
| Command menu | RVIP Enter | Tab; Enter = stairs/map interaction | Enter | DIFFERENT |
| Other rx1 | | `m` monsters, `l` log, `t` tactics, `h` quick shot, `f` launch, `g` aim, `U`/`O` overlays, Ctrl-T themes, F3/F4 gamma, F5/F6 wiz level, F10 wizard | similar plus `;` `:` `(` `)` | rx1-only |

---

## 12. Options, messages

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Options | terse, flush, jump, step (slow_invent), askme, name, fruit, file | terse, flush, jump, seefloor, passgo, tombstone, inven, name, fruit, file | `config.rec`: AnimationDelay, animation toggles, MapWidth 80, MapHeight 23, DiagonalMovementEnabled, AutoPickup, WallSlide, PlayerName, DataRootDir, KeyMapFile, Theme | DIFFERENT |
| Option UI | `o` | `o` (env ROGUEOPTS) | file only, no in-game option screen found | DIFFERENT |
| Message handling | one-line, `--More--` | same | scrolling message log (`l`) | DIFFERENT |
| Environment | - | SEED, ROGUEOPTS | none | - |

---

## 13. Search

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Secret door | `rnd(100) < 20` (20%) | 1/5 | 1/5 | SAME |
| Trap | `rnd(100) > 50` (about 49%) | 1/2 | 1/2 | SAME |
| Secret passage | none | 1/3 | 1/3 (also secret stairs) | rx1 = 5.4 |
| Blind/hallucinating | blind cannot search | penalty via `probinc` | `probinc` +3 hallucinating, +2 blind | rx1 = 5.4 |
| Command | `s` | `s` | no key; `Wait()` also searches | DIFFERENT |

---

## 14. Scoring, tombstone, winning

| Topic | 3.6 (`rip.c`) | 5.4 (`rip.c`) | rx1 (`game/state.go: gameWon, gameOver, writePlayerScore`) | Verdict |
|---|---|---|---|---|
| Death score | `purse -= purse/10`, top 10, encrypted v36 score file | same 10% penalty; variable `numscores` (default 10); one score per user unless `allscore` | score is gold only (`calculateTotalNetWorth`), no 10% penalty, no item worth; gob-encoded `scores.bin`, 15 entries | DIFFERENT |
| Sort | by score | by score | escaped first (by gold), then max level, then gold | rx1-only |
| Win worth | `total_winner`: amulet 1000, food 2*count, scroll/potion `mi_worth*count`, ring base +20*o_ac if >0 else 50, stick +20*charges | halved if unidentified, ring +o_arm*100 else 10, floor 0 | none | DIFFERENT |
| Flags | 0 killed, 1 quit, 2 winner | + 3 killed with amulet | escaped / died / max level | DIFFERENT |
| Win condition | carry amulet to level 0 via `<` | same | `<` on level 1 with amulet calls `gameWon`; without amulet goes to town | rx1 adds town |
| Tombstone | always | `tombstone` option | not verified | not verified |
| Wizard games | not scored | not scored | score not tied to wizard mode | DIFFERENT |
| Max level | 26 (amulet) | 26 | `maximumDungeonLevel: 26` | SAME |

---

## 15. Save, wizard/debug

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Save | `S` saves and exits; file unlinked on restore | same | not verified (no save key in keymaps) | not verified |
| Wizard entry | Ctrl-P, password `mTBellIQOsLNA` | `+`, same password | F10 wizard menu, no password | DIFFERENT |
| Wizard commands | `C` create_obj, Ctrl-I level objects, Ctrl-W whatis, Ctrl-D/U level change, Ctrl-F map, Ctrl-X monsters, Ctrl-T teleport, Ctrl-E food, Ctrl-A inpack, Ctrl-N charge, Ctrl-H sword+plate mail, `@` position | `|` position, `C`, `$` inpack, Ctrl-G, Ctrl-W, Ctrl-D/A, Ctrl-F, Ctrl-T, Ctrl-E, Ctrl-Q add_pass, Ctrl-X turn_see, `~` charge, Ctrl-I sword+plate, `*` pr_list | `OpenWizardMenu`: toggle show map, teleport (town, test map, secret level, depth, new level per style), raise level, curse all equipment, create item, create monster, create trap; F5/F6 wiz ascend/descend | DIFFERENT (menu based) |

---

## 16. rx1-only features

- Town (`game/town.go`, `data_rx1/prefabs/town.txt`): stash/home, shop, potion of life; level 1 up-stairs lead there.
- Visited levels persist (`levels` map): no stair scumming. Rogue regenerates each level.
- Secret level at depth 7-12.
- Level generators: Brogue-, NetHack-, megadungeon-, cave-, maze-style (`dungen/`).
- Themes (`data_rx1/themes/*.rec`: amber, ascii, cp437, fancy, green, hack) and Ctrl-T switch.
- Light sources, light radius and vision (section 8).
- Auto-explore (`o`) and look cursor (`x`), item/monster overlays and lists.
- Fatigue points (FP) with tactics menu (charge attack, heroic charge, sprint).
- Brogue-style weapon patterns (`game/actions.go`): spear hits the one behind, axe sweeps adjacent, rapier lunges two steps (never misses, triple damage), whip reaches 5 tiles, club/mace half speed, dagger sneak attack x5 (x3 others).
- Timed stuck curses (100-399 turns), blessing chance on remove curse, enchant cap +8.
- Rust of weapons; ring burnout by charges; lore documents; created monsters/traps via wizard menu.
- Speed energy system (normal 10, hasted 20, slowed 5); levels 1-2 give enemies half time.
- Wandering monsters every 4 turns with 1-in-6 chance (rx1 specific).
- Web build (`startup_js.go`), Stack-count unlimited per slot, 15-entry score table.

## 17. Summary of main differences

1. Weapons: 3.6 has 12 weapons chosen uniformly with 1d10 sword / 3d6 two-hander; 5.4 has 9 with weighted table and heavier dice. rx1 follows 5.4 dice/weights (mace 2d4, long sword 3d4, two-hander 4d4, spear 2d3), keeps 3.6's crossbow, and adds 7 weapons (main gauche, overkill, club, axe, rapier, whip, battle axe replacing dagger).
2. Armor: AC values and probabilities are identical in 3.6, 5.4 and rx1 (rx1 stores protection = 10-AC). Worth differs between 3.6 and 5.4; rx1 has none. rx1 uses only the torso slot.
3. Bonus generation is identical in all three (weapon 10%/5%, armor 20%/8%, ring `rnd(3)` with 0 -> -1 cursed). rx1 adds timed stuck curses, an enchant cap (+8) and a remove-curse blessing chance.
4. Rust: 3.6 rust monster can rust leather and has no protection; 5.4 aquator/rust trap, leather immune, protect armor scroll and maintain armor ring; rx1 rust effects reduce protection and also corrode weapons.
5. Pack: 3.6 effective 22 slots with shifting letters; 5.4 23 slots and stable letters; rx1 23 stacks with unlimited counts per stack and any item throwable.
6. Item category weights, `put_things` 36% and treasure rooms: rx1 follows 5.4 (3.6 uses 27/27/18/9/9/5/5 and 35%). Food forcing (`no_food > 3`) is the same in all three; rx1 has no fruit.
7. Combat: rx1 uses 3.6 to-hit and XP tables but 5.4 +4 vs sleeping and both-ring bonuses, and thrown-dagger logic as in 5.4; strength bonuses differ from both (no 18/xx).
8. Equip: Rogue wear/take-off costs turns; rx1 equip is free. Hunger in rx1 is a simple 300-turn counter without weak/faint/starve.
9. Interface: rx1 has two keymaps (WASD default, Rogue-style `rogue.txt`), menus, targeting cursors and auto explore; no save, discoveries, call, repeat or count commands were found. Options are a config file, not an in-game screen.
10. Score/wizard: rx1 scores gold only in a 15-entry gob table; wizard is a password-free F10 menu.

Items marked "not verified" were not confirmed in rx1 source (save/load, tombstone, scare-monster dust, yuk exp, inventory letters, ring teleport mechanics).
