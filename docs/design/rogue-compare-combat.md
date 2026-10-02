# Rogue 3.6 vs Rogue 5.4 vs rx1: player stats and combat

Sources: `/Users/felix/Games/rogue3.6`, `/Users/felix/Games/rogue5.4` (C), rx1 working tree (`game/`, `rpg/`, `data_rx1/definitions/*.rec`, uncommitted changes included).
Verdict: SAME = all three agree; DIFFERENT = rx1 differs from at least one Rogue; rx1-only = no Rogue counterpart; 3.6/5.4 = the two Rogues differ from each other (noted in the cells).
Rogue AC is lower-is-better; rx1 shows protection as a positive number but computes the same thing (`10 - protection`).

## 1. Starting state

| Topic | Rogue 3.6 | Rogue 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Stats | lvl 1, exp 0, hp 12, Str 16 (1% chance of 18/xx), dmg 1d4, AC 10 (init.c:init_player) | INIT_STATS: Str 16, exp 0, lvl 1, AC 10, hp 12, "1x4" (extern.c, init.c:init_player). Always 16 | Str 16, lvl 1, HP 12, Arm 10, "1d4" (game/actor.go:NewPlayer), plus FP (fatigue) 3 | SAME (Str 16 flat in 5.4/rx1; 3.6 has the 18/xx chance; FP rx1-only) |
| Weapon | mace +1,+1 wielded | mace +1,+1 wielded | +1,+1 mace (game/state.go:init) | SAME |
| Armor | ring mail, AC = a_class-1 = 6, known | same | ring mail, plus 1 (protection 3+1 = AC 6) | SAME |
| Ranged | bow +1,+0 (hplus 1), arrows 25+rnd(15) | short bow +1,+0, arrows 25+rnd(15) | +1 short bow, 25-39 arrows | SAME |
| Food | 1 ration | 1 ration | 1 ration | SAME |
| Extras | none | none | torch (light slot); all items pre-identified | rx1-only |
| food_left | HUNGERTIME 1300 | HUNGERTIME 1300 | TurnsSinceEating=0 | see hunger |

## 2. Strength tables

| Topic | 3.6 (fight.c) | 5.4 (fight.c tables) | rx1 (rpg/rogue.go) | Verdict |
|---|---|---|---|---|
| Strength range | 3..18 plus 18/01..18/100 exceptional; chg_str adds 1-100 to exceptional | single int 3..31 (add_str clamps) | int 3..18 (ChangeStrength clamp), MaxStr tracked | DIFFERENT |
| to-hit add (str_plus) | 18/100:+3, 18/>50:+2, >=17:+1, >6:0, else str-7 | 3..6: -4..-1; 7..16: 0; 17..20: +1; 21..30: +2; 31: +3 | StrPlus: >=17:+1, >6:0, else str-7 | SAME as both for 3..17 (no 18/xx or >20 range) |
| damage add (add_dam) | 18/100:+6, >90:+5, >75:+4, any 18/xx:+3, 18:+2, >15:+1, >6:0, else str-7 | 3..6: -4..-1; 7..15: 0; 16,17: +1; 18: +2; 19,20: +3; 21:+4; 22..30:+5; 31:+6 | AddDam: <6:-1, <16:0, <18:+1, else +2 | DIFFERENT: low end flat -1 (str 3..5) instead of -4..-1 and str 6 gives 0 (Rogue -1); 16..17 +1 and 18 +2 match; no 18/xx |
| Who gets it | player only; monsters use their own s_str 10 -> str_plus 0 | same | monsters Str 10 (rollAttack); player incl. ring bonus | SAME |
| Strength loss | ant bite chg_str(-1) unless save(VS_POISON)/sustain ring (3.6) | rattlesnake, same rule (5.4); dart trap also | poison_strength: Save(VsPoison) then ChangeStrength(-1) (effects_hit.go); no sustain-strength ring in rx1 | DIFFERENT (no sustain; monsters differ) |
| Restore strength | potion restores to max (3.6 keeps max via ring-less table) | restore corrects for ring bonus (max_stats) | RestoreStrength: Str = max(Str, MaxStr) | SAME in effect |

## 3. Level, experience, HP

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| e_levels | 10,20,40,80,160,320,640,1280,2560,5120,10240,20480,40920,81920,163840,327680,655360,1310720,2621440 (19, max lvl 20) fight.c | 10,20,40,80,160,320,640,1300,2600,5200,13000,26000,50000,100000,200000,400000,800000,2000000,4000000,8000000 (20, max lvl 21) extern.c | ExpLevels = 3.6 table | 3.6/5.4 DIFFERENT; rx1 = 3.6 |
| Exp per kill | static s_exp from the monster table (fight.c:killed) | s_exp = m.exp + lev_add*10 + exp_add(tp): mod = maxhp/8 (lvl 1) else maxhp/6; x20 if lvl>9, x4 if lvl>6 (monsters.c:new_monster) | NewEnemyFromDef: def.Exp + levAdd*10 + ExpAdd (5.4 formula) | rx1 = 5.4 formula on 3.6 table (DIFFERENT from each) |
| Level-up HP | check_level: add = roll(newlvl-oldlvl, 10); max_hp += add; hp += add, capped at max_hp | same roll; hp += add, no cap | AddExperience: HP and MaxHP += roll(lvl-old,10) | SAME roll; 3.6 caps hp, 5.4/rx1 do not |
| Level drain | wraith 15%: lose level, hp/max_hp -= roll(1,10); death if exp==0 | wraith 15% same; vampire 30% max-HP drain roll(1,3) | drain_level always applies when the effect fires; Exp set to ExpLevels[lvl-2]+1; HP loss roll(1,10); drain_max_hp costs 1 and heals the monster 1 | DIFFERENT (rx1 numbers, no vampire roll(1,3)) |
| Eating unknown/yuk | 30% "Yuk" fruit exp+1 | same | none (no fruit) | rx1 absent |

## 4. Regeneration

| Topic | 3.6 (daemons.c:doctor) | 5.4 (daemons.c:doctor) | rx1 (game/deamons.go) | Verdict |
|---|---|---|---|---|
| Level < 8 | +1 HP when quiet > 20 - 2*lvl (turns since last fight) | +1 HP when quiet + 2*lvl > 20 (same) | +1 HP every max(3, 20-2*level) turns when not hungry and no visible enemy | DIFFERENT in trigger (periodic, "quiet" is no-visible-enemy) |
| Level >= 8 | when quiet >= 3, heal rnd(lvl-7)+1 | same | max(3,...) = 3 turns, always +1 | DIFFERENT (no rnd(lvl-7)+1 bonus) |
| Regen ring | +1 HP per ring per turn | +1 per ring | Regenerating flag: +1 HP per turn, ring has charges 10d3 that run out | DIFFERENT (charges) |
| While hungry | unchanged | unchanged | regen stops; fatigue drops every 3*healInterval | rx1-only |
| Fatigue | none | none | FP regenerates when not healing | rx1-only |
| Resting | ' ' (space) = rest, 's' search | '.' rest, 's' search | Wait() rests and searches together | DIFFERENT |

## 5. Hunger

| Topic | 3.6 (daemons.c:stomach, misc.c:eat) | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Start/max | 1300 / STOMACHSIZE 2000 | same | n/a | |
| Digestion | per turn: 1 + ring_eat(L)+ring_eat(R) - amulet | same | TurnsSinceEating++ (slow digestion halves) | DIFFERENT |
| Hungry | food_left < 300 (2*MORETIME) | same | every 300 turns FlagHunger += 1 | DIFFERENT |
| Weak | < 150 (MORETIME) | same | no weak state | rx1 missing |
| Faint | food_left <= 0: if not already out, rnd(100)>20 returns -> ~21%; no_command = rnd(8)+4 | `rnd(5)!=0` returns -> 20%; no_command += rnd(8)+4 | none | DIFFERENT (3.6/5.4 differ by 1%) |
| Starve | no starvation death (only faint; death 'E' only via eye paralysis > 100) | food_left < -850 (STARVETIME) death('s') | none | 3.6/5.4 DIFFERENT; rx1 missing |
| Ration | food_left += 1300 + rnd(400) - 200, cap 2000 | same, but negative food resets to 0 first | satiate_fully clears all hunger | DIFFERENT |
| Fruit/mango | food type fruit (10%) same food value | same (rnd(10)!=0 = 90% ration) | only rations | rx1 missing |
| Forced food | no_food>3 levels forces food (things) | same | same rule (spawn_rogue.go) | SAME |
| Hunger-state change | message | message and stops running | message | SAME-ish |
| Ring hunger | ring_eat: regen 2, sustain str 1, search 33% random, digest -(50%) | uses[] table: protection 1, add str 1, search -3, regen 2, digest -2, ... | none | rx1 missing |

## 6. To-hit and damage

| Topic | 3.6 (fight.c) | 5.4 (fight.c) | rx1 | Verdict |
|---|---|---|---|---|
| swing | `rnd(20)+1; need = (21 - at_lvl) - op_arm; res+wplus >= need` | `rnd(20); need = (20 - at_lvl) - op_arm` (arithmetic identical) | rpg.Swing: Intn(20)+1+wplus >= 21-atLvl-defArm | SAME |
| wplus | weapon hplus + str_plus + ring add-hit | same | GetMelee: weapon plusses + StrPlus + ring hit bonuses | SAME (ring applies to all attacks in rx1) |
| Sleeping/held target | no bonus | +4 to hit when defender not ISRUN | +4 to hit on asleep/held defender (not the player) (actions.go:rollAttack) | 3.6/5.4 DIFFERENT; rx1 = 5.4 |
| Damage | roll(weapon dice) + dplus + add_dam, floor 0 | same | RollAttacks: roll + dplus, floor 0 | SAME |
| Dice format | "2d4" | "2x4" | "2d4" | cosmetic |
| Multi-attack | dmg string split on '/', one swing each | same | RollAttacks splits on '/', one swing each | SAME |
| Ring add-dam/hit | applied only when `weap == cur_weapon`; with no weapon (NULL) the monster's attacks also receive the player's ring bonus (3.6 bug) | NULL weapon gets no ring bonus | rings apply to all of the player's attacks including bare hands/thrown | 3.6/5.4 DIFFERENT; rx1 DIFFERENT |
| Bare hands | stats.s_dmg 1d4 | same | stats.Dmg | SAME |
| Armor on damage? | no | no | no | SAME |
| Sneak attack | none | none | x3 vs asleep/held/unaware/lunged (x5 dagger), also never misses (+100 hit) | rx1-only |
| Weapon patterns | none | none | spear pierces two, axe sweeps, rapier lunge, whip reach | rx1-only |
| Speed of heavy weapons | none | none | mace/hammer ("club" type) attack at half speed | rx1-only |
| Vorpal | none | none | +4/+4 vs named enemy else +1/+1 | rx1-only |

## 7. Armor class

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Base | 10 | 10 | 10 | SAME |
| Armor table a_class | leather 8, ring 7, studded 7, scale 6, chain 5, banded 4, splint 4, plate 3 (init.c) | same (extern.c) | protection 2,3,3,4,5,6,6,7 = 10 - a_class (armor.rec). Names: banded "lorica segmentata", splint "steel breastplate" | SAME |
| Generation prob | cumulative 20,35,50,63,75,85,95,100 | 20,15,15,13,12,10,10,5 | chain 12, scale 13 (armor.rec lists chain before scale; per-armor chance identical to 3.6/5.4: leather 20, ring 15, studded 15, scale 13, chain 12, splint 10, banded 10, plate 5) | SAME (corrected: no swap) |
| Computation | s_arm = a_class(worn) - enchant; minus ring of protection | same | GetArmorClass: for each piece `min(ac,10) - protection(incl. plus)`, then minus StatArmor ring modifier | SAME |
| Protection rings | only ONE counted (left else right) | both subtracted | rings summed | 3.6/5.4 DIFFERENT; rx1 = 5.4 |
| Cursed armor | 20%: ac += rnd(3)+1, cursed | same | 20% stuck, plus -(rnd(3)+1) | SAME |
| Blessed armor | 8% more: ac -= rnd(3)+1 | same | 8% | SAME |
| Rust | rust monster: o_ac++ if < 9 | aquator: skipped for leather, AC>=9, protect-armor, maintain-armor ring | rust_armor lowers plus by 1 down to protection 0; no leather exemption | DIFFERENT |
| Trap AC | arrow/dart traps use pstats.s_arm (10, worn armor not counted) | same | traps always hit, no swing | DIFFERENT |
| Donning time | wear() and take_off call waste_time | same | none | DIFFERENT |
| Cursed removal | dropcheck: "You can't. It appears to be cursed." permanent | same | stuckTurns 100..399 equipped turns | DIFFERENT |

## 8. Thrown weapons, ammo, launchers

| Topic | 3.6 (fight.c:roll_em, weapons.c) | 5.4 | rx1 (actor.go:GetThrowing) | Verdict |
|---|---|---|---|---|
| Launched ammo | launcher matches (arrow+bow): o_hurldmg + launcher hplus/dplus | same | damageDice (launched) + launcher plusses | SAME |
| Missile w/o launcher | ISMISL uses o_damage (e.g. dagger/spear thrown at melee dice) | `o_launch<0` -> hurldmg, else o_damage | thrown_damage; non-weapon uses its thrown dice | DIFFERENT between Rogues, rx1 uses thrown dice |
| Non-missile | hurldmg | hurldmg | thrown dice | SAME |
| Stack | ISMANY count rnd(8)+8, throw splits one | same | quiver stacks | similar |
| Dagger | 1d6/1d4 MISL | ISMISL, count rnd(4)+2 | 1d6/1d4 | SAME |
| Weapons list | mace 2d4/1d3, long sword 1d10/1d2, bow, arrow 1d1/1d6, dagger, rock 1d2/1d4, 2h 3d6/1d2, sling, dart, crossbow, bolt 1d2/1d10, spear 1d8/1d6 | mace 2x4, long 3x4/1x2, 2h 4x4/1x2, short bow, arrow 1x1/2x3, dagger, dart, shuriken 1x2/2x4, spear 2x3/1x6; no rock/sling/crossbow | weapons.rec: 5.4 numbers for long sword 3d4, 2h 4d4, spear 2d3/1d6, arrow 2d3 launched/1d1 thrown; keeps 3.6 crossbow/bolt; adds main gauche, club, axe, rapier, whip, overkill 5d8+1 | 3.6/5.4 DIFFERENT; rx1 mostly 5.4 + extras |
| Weapon pick | uniform rnd(MAXWEAPONS) | weighted pick_one | weights per .rec (matching 5.4) | rx1 = 5.4 |
| Curses on weapons | 10% cursed hplus -= rnd(3)+1; 5% hplus += rnd(3)+1 | same | 10% stuck and -rnd(3)+1; 5% blessed | SAME |
| Vanishing | fall(): message only | "vanishes as it hits the ground" | recoverable | DIFFERENT |

## 9. Monster attacks and special hits

| Monster effect | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Rust | rust monster R | aquator A | rust_armor effect | DIFFERENT (see AC) |
| Paralysis gaze | floating eye E: no_command += rnd(2)+2 on hit | ice monster I: freeze rnd(2)+2, death 'h' if >50 | freeze sets FlagStun; Save each turn to shake off; floating eye is a struck_effect | DIFFERENT |
| Str poison | ant A | rattlesnake R | poison_strength (Save(VsPoison)) | SAME mechanism |
| Level drain | wraith W 15% | wraith W 15% | drain_level | SAME |
| Hold | violet fungi F: ISHELD, fung_hit++, damage grows | venus flytrap F: hit immediately hp-- plus growing vf_hit | hold: FlagHeld, squeeze count -1; killing holder frees | DIFFERENT (3.6 fungus applies growth on a miss) |
| Gold theft | leprechaun L: GOLDCALC, failed VS_MAGIC = 4x; monster vanishes | same | steal_gold same 4x | SAME |
| Item theft | nymph N random unequipped magic item | same | steal_item | SAME |
| Regenerating monsters | ISREGEN +1 at 33% per attack | same | Regenerating flag +1/turn | DIFFERENT |
| Umber hulk gaze | confuses (3.6) | not present | CanConfuse flag on attacks | DIFFERENT |
| rx1 extras | | | slow, hunger, knockback, poison over time (5-10 turns), hit_and_run, split, rust_weapon, eat_gold; struck effects fire back | rx1-only |
| Save | `14 + which - lvl/2` vs roll(1,20); VS_POISON 0, VS_MAGIC 3, VS_BREATH 2 | same, with protection ring bonus for VS_MAGIC | Save: Intn(20)+1 >= 14+which-lvl/2; no VsBreath | SAME |

## 10. Status effects and rules

| Topic | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Sleeping monsters | wake_monster: mean monsters chase on rnd(100)>33 | rnd(3)!=0 and not levitated/stealthy | wake only if in same room and perceive (2/3); others 1/10 after perception | DIFFERENT |
| Damage wakes | yes | yes | sleeping victims wake with 90% on damage | DIFFERENT |
| Player paralysis | no_command; sleep trap SLEEPTIME 5; paralysis potion HOLDTIME 2 | spread(5); no paralysis potion | FlagStun with per-turn Save(level+stun-1, VsMagic) | DIFFERENT |
| Held player | only 'F' attackable; moving fails | same | each turn Save(level+StrPlus, VsPoison) to break free; bear trap hold_target FlagHeld rand(10)+5 | DIFFERENT |
| Bear trap | no_move += 3 | spread(3) | hold 5..14 turns | DIFFERENT |
| Confused movement | 80% random (rnd(100)>80 gets direction) | `rnd(5)!=0` 80% | 80% random | SAME |
| Confused throw/zap | get_dir redirect 20% | `rnd(5)==0` 20% | not examined | - |
| Confusion duration | HUHDURATION 20+rnd(8) | spread(20) | 8..15 | DIFFERENT |
| Blind | SEEDURATION 850 (3.6 cannot search) | 850 (can search, probinc +2) | 8..15; blind may search | DIFFERENT |
| Haste | potion rnd(4)+4 turns (extra ntimes); drinking again faints rnd(8) | same, no_command rnd(8) and clears haste | speed 20; duration rand(speed/2..speed) ; haste cancels slow | DIFFERENT |
| Monster haste/slow | ISSLOW acts every other turn (t_turn toggle), ISHASTE twice | same; ISFLY moves again when dist>=3; levels >29 hasted | energy system 100/speed; slow 5, haste 20 | DIFFERENT mechanism, similar effect |
| Early-level slowdown | none | none | levels <=2: enemy time /2 | rx1-only |
| Levitation | potion | HEALTIME 30; avoids traps | fly 8..15 turns | DIFFERENT |
| Invisibility / see invis | 3.6 see invisible 850 | same | invisible 8..15, see invisible 8..15 | DIFFERENT |
| Fight-to-flee | none | none (F/f fight to death only) | FlagScared flee after hit_and_run | rx1-only |
| Fight to death | 'f' is run prefix only | 'f'/'F' fight until death; kamikaze | not examined | 5.4 only |
| Monster aggro | ISRUN on wake | same | notice by LOS in room | DIFFERENT |
| Monster conf | 80% random, small unconfuse | `rnd(5)!=0`, 1/20 unconfuse | 4/5 random or -2 attack, 1/6 clear | DIFFERENT |
| Dragon breath | 1 in 5 (DRAGONSHOT) | same | 1 in 5 for monsters with abilities | SAME |
| Searching | 's'; secret door rnd(100)<20; trap about 50%; ring of searching each turn | secret door 1/(5+probinc); trap 1/(2+probinc); passage 1/(3+probinc); probinc +3 halluc, +2 blind | same as 5.4 | rx1 = 5.4 (3.6 differs) |
| Teleport ring | rnd(100)<2 | rnd(50)==0 | 5% per move | DIFFERENT |
| Traps | arrow: swing(lvl-1, s_arm,1) roll(1,6); dart: swing(lvl+1) roll(1,4), chg_str(-1) unless sustain | same, plus mystery and rust traps | arrow/dart traps spawn projectile, always hit, no str loss; slow/teleport/descend/exploding (5 dmg radius 3) | DIFFERENT |
| Trap placement | rnd(10)<level; count | same | rnd(10)<level; min(rnd(level/4)+1,10) | SAME |

## 11. Healing items

| Item | 3.6 | 5.4 | rx1 | Verdict |
|---|---|---|---|---|
| Healing | roll(lvl,4); above max -> hp = ++max_hp | roll(lvl,4); max_hp bonus +1/+2 | heal = maxHP/2 | DIFFERENT |
| Extra healing | roll(lvl,8) | same | full maxHP | DIFFERENT |
| Max HP | none | none | gain_max_hp (potion of life) | rx1-only |
| Raise level | raise_level | same | RaiseLevel | SAME |

## 12. Death and score

| Topic | 3.6 (rip.c) | 5.4 (rip.c) | rx1 (game/state.go) | Verdict |
|---|---|---|---|---|
| Gold on death | purse -= purse/10 | same | none (raw gold) | DIFFERENT |
| Pack value | total_winner worth tables (weapons base*(1+10*(hplus+dplus)), armor etc.) on win | same | no pack valuation | DIFFERENT |
| Killer codes | monster letter, 'E' | 's' starvation, 'h' hypothermia | n/a | |
| Ranking | purse | purse | escaped first, then max level, then gold | DIFFERENT |

## 13. Rings (combat-relevant)

| Ring | Rogue (3.6/5.4) | rx1 | Verdict |
|---|---|---|---|
| Protection | AC bonus (3.6 one ring, 5.4 both) | summed | rx1 = 5.4 |
| Add strength | adds to s_str | strength ring bonus "1d3-1" | similar |
| Increase damage / dexterity | add_dam / add_hit only when weapon == cur_weapon | apply to all attacks | DIFFERENT |
| Sustain strength | present | absent | missing |
| Aggravate, maintain armor | present (maintain armor 5.4) | absent | missing |
| Bonus gen | rnd(3), 0 becomes -1 cursed; aggravate/teleport cursed | "1d3-1"; 0 becomes -1 and stuck; teleportation stuck | SAME |

## Where 3.6 and 5.4 differ from each other

- exp table (high levels), per-kill exp formula (static vs exp_add)
- +4 to hit sleeping/held (5.4 only)
- Ring bonuses applied to unarmed monsters (3.6 bug, fixed in 5.4)
- Protection rings: one (3.6) vs both (5.4)
- Strength model (18/xx vs int 3..31)
- Starvation death (5.4 only); faint 21% vs 20%; negative food reset on eat
- Rust: rust monster vs aquator (leather/AC>=9/maintain exemptions)
- Fungus vs flytrap hold damage
- Search chances, teleport ring chance, durations (fixed vs spread), weapon table, weighted vs uniform weapon pick, paralysis potion (3.6 only), f/F fight-to-death (5.4)
- Level-up hp cap (3.6 caps)

## Main differences: rx1 vs both Rogues

1. Time and speed: energy system, levels <=2 enemies at half rate; no t_turn toggling.
2. Hunger: staged "hungry" every 300 turns only; no weak, faint or starvation; no fruit; rations satiate fully.
3. Regeneration: periodic +1 (max(3, 20-2*lvl)); no level>=8 bonus; ring has charges; fatigue (FP) system added.
4. Strength: capped at 18, no 18/xx; AddDam low-end flatter (str 6 gives 0).
5. Rings of damage/to-hit apply to every attack including bare hands and throws.
6. Curses are timed (stuckTurns 100..399), no donning delay, no remove-curse permanence.
7. Healing potions: heal = half maxHP, extra heal = full, instead of roll(lvl,4)/roll(lvl,8).
8. Status effects are short generic 8..15 turn timers; held/stun use per-turn Saves; traps are magic projectiles that always hit.
9. Added combat layer: sneak attacks x3/x5, weapon patterns, half-speed maces, vorpal, scared/flee monsters, extra hit effects.
10. exp: 3.6 level table with 5.4 per-kill formula; score has no 10% death penalty or pack valuation.

Not examined in detail: confused throw/zap in rx1, scroll effects (enchant/remove curse/protect armor), sticks.c wand effects, fruit/mango (absent), armor probabilities were verified (corrected): armor.rec chain 12 / scale 13 equals 3.6 and 5.4, no swap.
