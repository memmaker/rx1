# Monster Lore — research draft

Proposed lore per monster, for review. Nothing in data_rx1/ changed yet.

# Monster lore, part 1

Version notes below come from the monster tables in `~/Games/rogue3.6` and `~/Games/rogue5.4` (`extern.c`), as I remember them. Check them against the source before using them. The existing white flavor lines are the PC/Epyx-manual-style blurbs. Keep them.

NetHackWiki says in summary: in early Rogue the aquator was the "rust monster" and the ice monster was the "floating eye". The names were changed for the commercial MS-DOS version to avoid D&D copyright problems. The DOS version also replaced S (snake) with a slime that can divide. Source: https://nethackwiki.com/wiki/Rogue_(game)

## Bat
**Existing:** complete (flavor, Origin, Rogue Trivia, NetHack History). No work needed.
**Flavor:** keep the existing line.
**Rogue history / meta:** 'B' in every version. It moves erratically on purpose: 50% random moves in the 3.6 code.
**Original source:** a real animal. The OD&D/Monster Manual "giant bat" is a likely model (unverified).
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "bat or bird": a comic passage about a bat's aimless flight (P. G. Wodehouse).
**Sources:** existing file; data.base above.

## Emu
**Existing:** flavor line only.
**Flavor:** keep the existing line ("Yendor's bizarre sense of humor").
**Rogue history / meta:** 'E' in Rogue 5.x. It took the slot of the 3.6 floating eye. It is one of several real animals (emu, kestrel, quagga) that fill letters in 5.x. Many people think these were chosen to fill the alphabet with something not taken from D&D (unverified). Weak early monster.
**Original source:** a real animal. *Dromaius novaehollandiae* is a large flightless Australian bird known for a powerful kick. It has no D&D source.
**Local roguelike texts:** none found.
**Sources:** rogue5.4/extern.c; nethackwiki Rogue_(game).

## Hobgoblin
**Existing:** flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** 'H' in every version. It is the first "real" fighter on dungeon level 1. Its stats follow D&D: hit dice 1+1, AC 5 (unverified, compare with extern.c).
**Original source:** the OD&D Monsters & Treasure (1974) hobgoblin, a larger and tougher goblin. The word comes from English folklore, where Puck was called "Hobgoblin".
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "hobgoblin": the folklore meaning, Puritan use of the word for wicked spirits, and Puck in *A Midsummer Night's Dream*. Hobgoblins are "good-humoured", brownie-type spirits.
**Sources:** data.base; nethackwiki.

## Kestrel
**Existing:** flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** 'K' in Rogue 5.x. It replaced the 3.6 kobold (D&D), most likely to drop D&D names. It is a real small falcon.
**Original source:** a real animal (*Falco tinnunculus*, known for hovering). It has no D&D source.
**Local roguelike texts:** none found. "Kestrel" in `port/nethack/monsters.txt` is only an artist's name.
**Sources:** nethackwiki Rogue_(game).

## Jackal
**Existing:** empty.
**Flavor (proposed):** "Small, cowardly scavengers that hunt in packs on the upper levels, nipping at anything weaker than themselves."
**Rogue history / meta:** 'J' in Rogue 3.6, a weak level-1 animal. In 5.x the letter went to the jabberwock (Lewis Carroll). This kept the jackal's role as Hack/NetHack's first monster.
**Original source:** a real animal. The folklore below is a good Origin section.
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "jackal": in Asian folktales the jackal flushes game for the lion and eats the leftovers. Its reputation for cowardice comes from this. In Hausa tales it is the wise judge, "O Learned One of the Forest". From Funk & Wagnalls *Standard Dictionary of Folklore*.
**Sources:** data.base; nethackwiki.

## Floating eye
**Existing:** empty.
**Flavor (proposed):** "A great eyeball drifting through the halls. Harmless in itself, but meet its gaze in battle and you'll stand frozen while its neighbours feed."
**Rogue history / meta:** 'E' in Rogue 3.6 and early versions. Hitting it can paralyze you. Per NetHackWiki, its role (paralysis) moved to the 'I' ice monster in the DOS/5.x versions to avoid D&D names. NetHack kept it, and "Elbereth" and "hit a floating eye" deaths are famous YASDs.
**Original source:** D&D Greyhawk supplement (1975) / Monster Manual (1977) floating eye. It is a small fish-like creature whose gaze paralyzes.
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "floating eye": huge drifting eyeballs whose paralyzing gaze lets other monsters nibble you to death. `~/Games/angband-4.2.6/lib/gamedata/monster.txt` line ~708: "A disembodied eye, floating a few feet above the ground."
**Sources:** both paths above; nethackwiki Rogue_(game).

## Ice monster
**Existing:** flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** 'I' in Rogue 5.x. It replaced the 3.6 invisible stalker on that letter and took over the floating eye's freezing/paralysis role. NetHackWiki says it "causes just as many YASDs". The renaming is linked to the commercial DOS port avoiding D&D names.
**Original source:** an original Rogue invention, as far as known (unverified). It reworks the D&D floating eye.
**Local roguelike texts:** none.
**Sources:** nethackwiki Rogue_(game).

## Slime
**Existing:** flavor line only ("garbage of the dungeon").
**Flavor:** keep the existing line.
**Rogue history / meta:** per NetHackWiki, the DOS (Epyx/A.I. Design) version replaced 'S' snake with a slime that divides when hit. This is the source of the rx1 slime.
**Original source:** probably the D&D slimes and jellies: green slime and ochre jelly. The ochre jelly splits when hit (unverified as the direct model).
**Local roguelike texts:** none found. Possibly in `~/Games/roguepc/src` (check monster table).
**Sources:** nethackwiki Rogue_(game).

## Snake
**Existing:** empty.
**Flavor (proposed):** "An ordinary serpent that slid into the dungeon long ago. Its bite is weak, but it is quick to strike."
**Rogue history / meta:** 'S' in Rogue 3.6 and 5.x, a level-1 animal. It was replaced by the slime in the DOS version.
**Original source:** a real animal. The OD&D/Monster Manual "giant snake" types are a likely model (unverified).
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "*snake": Genesis 3, the serpent "more subtle than any beast of the field".
**Sources:** data.base.

## Gnome
**Existing:** empty.
**Flavor (proposed):** "Small earth-dwellers who know the tunnels better than you. They seldom pick a fight they can't win."
**Rogue history / meta:** 'G' in Rogue 3.6. In 5.x the letter went to the griffin, a stronger monster (NetHackWiki calls the griffin a "higher level monster replacing Gnome").
**Original source:** OD&D Monsters & Treasure (1974) gnome. Paracelsus's elemental earth spirits come before that.
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "gnome*": the Alaraba passage from Lord Dunsany, *The Charwoman's Shadow*. A frightened gnome in a hat is "three times as large as an imp".
**Sources:** data.base; nethackwiki.

## Orc
**Existing:** flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** 'O' in every version. In some versions orcs (and leprechauns) go for gold (unverified for orcs). Fits the "hired guards" blurb.
**Original source:** Tolkien (*The Hobbit*, *LotR*), then OD&D Monsters & Treasure (1974).
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "orc*": stocky, boar-tusked humanoids related to goblins. They live in tribes and dislike sunlight. They can't make their own gear, so they keep raiding for it ("het Boek van de Regels"). The "orcrist" entry quotes *The Hobbit* on goblins.
**Sources:** data.base.

## Rattlesnake
**Existing:** flavor line only ("do not bark").
**Flavor:** keep the existing line.
**Rogue history / meta:** 'R' in Rogue 5.3+/5.4. It took the letter after the rust monster became the aquator. Its bite lowers strength (poison) (unverified, compare with fight.c).
**Original source:** a real animal (*Crotalus*). It has no D&D source.
**Local roguelike texts:** data.base "*snake" (see Snake).
**Sources:** nethackwiki Rogue_(game).

## Invisible stalker
**Existing:** empty.
**Flavor (proposed):** "You hear it before you see it. Actually, you never see it."
**Rogue history / meta:** 'I' in Rogue 3.6. It is invisible, and the 3.6 source comment says stalkers are "slightly confused all of the time" (20% random moves, quoted in bat.txt). In 5.x the letter went to the ice monster.
**Original source:** OD&D Monsters & Treasure (1974) invisible stalker, an extra-planar hunter that wizards summon to track a target.
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "*stalker": passage from H. G. Wells, *The Invisible Man* (the unwrapping scene).
**Sources:** data.base; rogue3.6 chase.c comment.

## Rust monster
**Existing:** complete. No work needed.
**Local roguelike texts:** `~/Games/nethack50/dat/data.base` "rust monster" uses the same blurb as the existing flavor line.
# Monster lore, part 2

Shared local sources: `~/Games/roguelikes-index/trivia-rogue-family/index.html` (sourced Rogue-family trivia, cites file:line), `~/Games/nethack50/dat/data.base`, `~/Games/roguepc/src/extern.c`.
Shared web fact (NetHackWiki, "Rogue (game)"): several names were changed around the 1983-84 commercial IBM PC release, said to avoid D&D copyright trouble: rust monster→aquator, floating eye→ice monster, mimic→xeroc, xorn→ur-vile (PC) / black unicorn (5.4), violet fungi→venus flytrap, giant ant→rattlesnake.

## Zombie
**Existing:** flavor line only (fallen adventurers reanimated).
**Flavor:** keep existing.
**Rogue history / meta:** One of the few monsters in Rogue 3.6's A-Z alphabet ('Z') that survived every version unchanged in name and role: a slow, plain hitter with no special attack. A generic undead filler for its letter (interpretation).
**Original source:** OD&D *Monsters & Treasure* (1974) and *Monster Manual* (1977): mindless animated corpses made by evil clerics/magic-users. The idea comes from Haitian Vodou folklore, popularised by *White Zombie* (1932) and Romero's *Night of the Living Dead* (1968).
**Local roguelike texts:** `nethack50/dat/data.base` has a zombie entry (Haitian zombi lore). 3.6 roster in `roguelikes-index/.../index.html` ("Monster alphabet A-Z").
**Sources:** rogue3.6/init.c:78-106; https://nethackwiki.com/wiki/Rogue_(game)

## Giant ant
**Existing:** no file.
**Flavor:** "This giant variety of the ordinary ant will fight just as fiercely as its small, distant cousin." (NetHack data.base)
**Rogue history / meta:** 'A' in Rogue 3.6. Its sting saps strength: "You feel a sting in your arm and now feel weaker" (-1 Str unless saved). In 5.x the letter A went to the aquator and the strength-sapping bite moved to the new rattlesnake (R): "you feel a bite in your leg and now feel weaker". Super-Rogue and Advanced Rogue kept it (AR 5.8s even has ant-hill special rooms).
**Original source:** OD&D *Monsters & Treasure* (1974) "Ants, Giant"; *Monster Manual* (1977) lists giant ant workers, warriors and queen. Clearly influenced by 1950s B-movies like *Them!* (1954) (interpretation).
**Local roguelike texts:** `nethack50/dat/data.base` "* ant" entry; `roguelikes-index/trivia-rogue-family/index.html` "Giant ant poison, rattlesnake bite" (rogue3.6/fight.c:156-170).
**Sources:** as above; https://nethackwiki.com/wiki/Rogue_(game)

## Leprechaun
**Existing:** full (flavor, Description, Rogue Trivia with code, NetHack History). Good, no redo needed.
**Flavor:** keep.
**Rogue history / meta:** Possible addition: killing one adds GOLDCALC gold to the room (+4 more portions if your save succeeds) (rogue3.6/fight.c:207-218, 697-713).
**Original source:** D&D *Greyhawk* supplement (1975) (unverified) and *Monster Manual* (1977): invisible-at-will, illusionist and thief of gold.
**Local roguelike texts:** `nethack50/dat/data.base` leprechaun entry; trivia index "Leprechaun gold".
**Sources:** rogue3.6/fight.c.

## Centaur
**Existing:** flavor line only.
**Flavor:** keep.
**Rogue history / meta:** 'C' from 3.6 onward in every version. In 5.4 it attacks 1x2/1x5/1x5 (kick, weapon, weapon). Unusual among Rogue's mostly hostile roster in its flavor text as a "peaceful" creature turned fierce.
**Original source:** Greek myth: the Kentauroi of Thessaly, wild and lustful (battle with the Lapiths) yet including the wise Chiron, teacher of Achilles. In D&D: OD&D *Monsters & Treasure* (1974), *Monster Manual* (1977), neutral/good woodland dwellers.
**Local roguelike texts:** `nethack50/dat/data.base` "*centaur" (from *Larousse Encyclopaedia of Mythology*: monstrous yet "moral", citing Pholos and Cheiron).
**Sources:** rogue5.4/extern.c:190-221.

## Kobold
**Existing:** empty file (only colour tag).
**Flavor:** "Small, dog-faced and spiteful, the kobold is usually the first thing to greet a new adventurer in the Dungeons of Doom." (new)
**Rogue history / meta:** First letter of 3.6's depth string "KJBSH..." and of 5.4's "KEBSH...": the kobold is literally the weakest, shallowest monster in Rogue, the classic first kill. Present in every version.
**Original source:** German folklore: the Kobold is a household or mine sprite; miners blamed kobolds for the worthless ore later named cobalt. D&D (OD&D 1974; *Monster Manual* 1977) turned it into a small, evil, dog-like humanoid hating gnomes/elves.
**Local roguelike texts:** `nethack50/dat/data.base` "*kobold*": "an artificial creation of a master wizard", about 3' tall, "vaguely dog-like face", hate Elves.
**Sources:** rogue3.6/monsters.c:19-45; rogue5.4/monsters.c:21-45.

## Aquator
**Existing:** flavor + one-line trivia (was rust monster).
**Flavor:** keep.
**Rogue history / meta:** Introduced in Rogue 5.x as the renamed rust monster (A taking giant ant's letter). Behaviour change: leather armor became immune, and a ring of maintain armor or scroll of protect armor stops it ("the rust vanishes instantly"). Damage "0d0/0d0": it can't hurt you directly. A water-spraying creature is a neat in-world justification for "rust" without D&D's name (interpretation). Same in Rogue PC (`roguepc/src/extern.c:427`).
**Original source:** D&D rust monster (*Greyhawk*, 1975; see rust_monster.txt). The name itself appears to be an invention of Rogue.
**Local roguelike texts:** trivia index "Rust monster vs aquator" (rogue5.4/move.c:400-424); `roguepc/web/dist/help.html` tip: "its touch rusts your armour".
**Sources:** https://nethackwiki.com/wiki/Rogue_(game)

## Black unicorn
**Existing:** Rogue Trivia only (xorn/ur-vile history, code); flavor line empty.
**Flavor:** "Not every unicorn is a symbol of purity. This one is black as pitch and twice as mean." (new)
**Rogue history / meta:** Existing trivia covers it well. The PC name "ur-vile" is often linked to Tolkien, but no source was found confirming that (unverified).
**Original source:** Unicorns: medieval bestiaries (Ctesias, Physiologus). AD&D *Monster Manual* (1977) unicorns are good; black unicorns appear in later fantasy (Terry Brooks, *The Black Unicorn*, 1987, postdates Rogue). The xorn it replaced is from *Greyhawk* (1975).
**Local roguelike texts:** arogue5.8 / urogue monster tables (already in file).
**Sources:** rogue5.4/extern.c.

## Quagga
**Existing:** flavor line only.
**Flavor:** keep (fix "horrors"→? optional).
**Rogue history / meta:** Replaced the quasit ('Q') in Rogue 5.x; the quasit is a D&D demon, so likely another copyright-motivated swap (unverified). Stats in Rogue PC: level 3, AC 2, 1d2/1d2/1d4, 32 xp (`roguepc/src/extern.c:445`). A real-world animal is an odd and memorable pick for 'Q'.
**Original source:** Real zoology: *Equus quagga quagga*, a half-striped plains zebra from South Africa, hunted to extinction; the last one died in Amsterdam's Artis zoo in 1883. Not a D&D monster (unverified).
**Local roguelike texts:** only Rogue PC/5.4 tables; no lore text found.
**Sources:** https://en.wikipedia.org/wiki/Quagga

## Nymph
**Existing:** flavor, mythology, D&D, NetHack history. Good.
**Flavor:** keep.
**Rogue history / meta:** Add: in Rogue she steals a random unequipped magic item ("She stole %s!") then vanishes; 3.6 had a "Nymph bug fix" excluding worn rings. Super-Rogue's nymph steals your most valuable item instead.
**Original source:** covered.
**Local roguelike texts:** trivia index "Nymph steals a magic item", "Nymph steals your best item" (srogue/fight.c:216-256).
**Sources:** rogue3.6/fight.c:224-266.

## Mimic
**Existing:** empty file.
**Flavor:** "The treasure you were about to pick up just grew teeth."  (new) / or data.base: waits patiently "for its meals to come in search of it."
**Rogue history / meta:** 'M' in 3.6, disguised as gold, potion, scroll, stairs, weapon, armor, ring, stick, or (below level 25) even the Amulet. Attacking reveals it: "Wait! That's a mimic!" In 5.x/PC renamed xeroc ('X'), likely a pun on Xerox copiers (unverified); 'M' became medusa.
**Original source:** D&D *Greyhawk* supplement (1975) and *Monster Manual* (1977): a creature posing as chests, doors or stonework, with adhesive skin.
**Local roguelike texts:** `nethack50/dat/data.base` "*mimic": "ancestors of the modern day chameleon"; trivia index "Mimic reveal"; arogue7.7 "Wait! That's a %s!".
**Sources:** rogue3.6/monsters.c:77-94; roguepc/src/fight.c:38.

## Yeti
**Existing:** flavor line only (typo "peks" → "peaks").
**Flavor:** keep, fix typo.
**Rogue history / meta:** 'Y' in every Rogue version, a mid-depth brawler (claw/claw). Shows Rogue's habit of filling awkward letters with folklore beasts (interpretation).
**Original source:** Himalayan folklore; "Abominable Snowman" coined 1921 from a mistranslated report of the Everest reconnaissance expedition. In D&D: *Monster Manual* (1977), cold-dwelling, paralysing gaze.
**Local roguelike texts:** `nethack50/dat/data.base` "yeti" (from Daniel Cohen, *A Modern Look at Monsters*): neither "particularly abominable, nor does it necessarily live in the snows".
**Sources:** https://en.wikipedia.org/wiki/Yeti

## Troll
**Existing:** flavor line only.
**Flavor:** keep.
**Rogue history / meta:** 'T' in every version, one of the deep heavy hitters (1d8/1d8/2d6 in 3.6). XP rose from 55 (3.6) to 120 (5.4). In Rogue it regenerates (ISREGEN), echoing D&D.
**Original source:** Norse/Scandinavian folklore. The D&D regenerating, rubbery green troll is taken from Poul Anderson's *Three Hearts and Three Lions* (1961), which Gygax cited in Appendix N; OD&D 1974 / *Monster Manual* 1977.
**Local roguelike texts:** `nethack50/dat/data.base` "*troll" quotes Larry Niven's *The Magic Goes Away* (severed hand crawling "like a huge green spider").
**Sources:** trivia index "Damage list with slashes"; https://en.wikipedia.org/wiki/Troll_(Dungeons_%26_Dragons)

## Violet fungi
**Existing:** flavor, Rogue Trivia (rename to venus flytrap, code); Origin empty; NetHack History heading empty.
**Flavor:** keep.
**Rogue history / meta:** Add: holds you and its damage grows each hit (1d1, 2d1, 3d1...); even a miss costs you; Super-Rogue's counter is shared across all fungi, so each is deadlier after any hit.
**Original source (fill Origin):** D&D *Greyhawk* supplement (1975) and *Monster Manual* (1977), companion to the shrieker; its tendrils rot flesh.
**Local roguelike texts:** `nethack50/dat/data.base` "*fung*" (encyclopedia entry on fungi); trivia index "Violet fungus damage grows by itself", "Violet fungi counter" (srogue/fight.c:200-203).
**Sources:** rogue3.6/fight.c:197-201.

## Wraith
**Existing:** flavor line only.
**Flavor:** keep.
**Rogue history / meta:** 'W' in all versions and the only true level drainer: 15% per hit, "You suddenly feel weaker", lose a level and 1d10 max HP; with zero experience it kills (Super-Rogue). 5.4 adds the vampire's max-HP drain alongside.
**Original source:** Scots word for a ghost/apparition; Tolkien's Ringwraiths. D&D OD&D (1974) wraith drains life levels, like the wight, a direct Tolkien echo (barrow-wights).
**Local roguelike texts:** `nethack50/dat/data.base` "wraith" shares the Nazgûl passage from *The Fellowship of the Ring* (Weathertop); trivia index "Wraith drains levels", "Wraith can kill outright".
**Sources:** rogue3.6/fight.c:174-199; srogue/fight.c:188-199.
# Monster lore, part 3

Background that applies to every section below: Rogue 3.6 (Unix, around 1980) uses D&D monster names, one per letter A to Z. Later versions (5.x and the A.I. Design/Epyx PC port) renamed the letters that clashed with D&D. A Usenet post (rec.games.roguelike.rogue, "Monster names") says this was done to avoid legal trouble with TSR once Rogue went commercial. The renames by letter, comparing the `rogue3.6/init.c` and `rogue5.4/extern.c` tables in ~/Games:
F violet fungi → venus flytrap, G gnome → griffin, J jackal → jabberwock, M mimic → medusa, P purple worm → phantom, Q quasit → quagga, U umber hulk → black unicorn (5.4) / ur-vile (PC), X xorn → xeroc.
The existing one-line flavor texts appear to come from the Epyx/PC Rogue manual (unverified).

## Venus flytrap
**Existing:** a flavor line only ("prehistoric ancestors…"). There is no Origin or Trivia section.
**Flavor:** keep the existing line.
**Rogue history / meta:** The flytrap took the F slot from Rogue 3.6's violet fungi. Like the fungus, it does not move and it holds you in place: you cannot walk away until it dies. Its hold also does growing damage, as the ~/Games/roguepc help says: "holds you until it dies". The plant got a more exotic and "real" name than the D&D fungus.
**Original source:** The violet fungi comes from the D&D Monster Manual (1977). The Venus flytrap (Dionaea muscipula) is a real carnivorous plant from the Carolinas. Darwin called it "one of the most wonderful plants in the world".
**Local roguelike texts:** nethack50/dat/data.base has no flytrap entry. Its violet fungus entry covers the 3.6 ancestor.
**Sources:** ~/Games/rogue3.6/init.c, ~/Games/rogue5.4/extern.c, ~/Games/roguepc/web/dist/help.html

## Phantom
**Existing:** a flavor line only (it has a typo: "finds" should be "fiends").
**Flavor:** These shadowy fiends "live" deep in the Dungeons of Doom. Watch out for them, if you can.
**Rogue history / meta:** The phantom took the P slot from the purple worm. It is permanently invisible, so it fills the role of Rogue 3.6's invisible stalker (I), which became the ice monster. It is one of the generic names chosen after the D&D cleanup.
**Original source:** A general folklore ghost. It has no single D&D source.
**Local roguelike texts:** nethack50/dat/data.base mentions a "phantom chase" in its Wild Hunt entry (Encyclopedia Mythica). It has no phantom entry of its own.
**Sources:** ~/Games/rogue5.4/extern.c, ~/Games/nethack50/dat/data.base (line ~1207)

## Purple worm
**Existing:** empty file.
**Flavor:** A gargantuan cousin of the humble rain-worm, it can swallow an adventurer whole.
**Rogue history / meta:** This is the P monster of Rogue 3.6. It was dropped in 5.x for the phantom as part of the D&D-name cleanup. In 3.6 it is a deep, slow-to-appear heavy hitter.
**Original source:** OD&D (1974) and the Monster Manual (1977). It is a huge burrowing worm with a poisonous tail sting, and on a good bite roll it swallows its victim whole. It became one of D&D's signature "dungeon dread" monsters.
**Local roguelike texts:** nethack50/dat/data.base, "*purple worm": a gargantuan version of the rain-worm that swallows and digests its victims within minutes, and that senses vibrations or wakes to a shriek. Angband (lib/gamedata/monster.txt): a massive worm whose maw drips acid and poison.
**Sources:** ~/Games/nethack50/dat/data.base:4449, ~/Games/angband-4.2.6/lib/gamedata/monster.txt:6928, ~/Games/rogue3.6/init.c

## Ur-vile
**Existing:** a flavor line only ("brought in from another dimension by Yendor…").
**Flavor:** keep the existing line.
**Rogue history / meta:** The ur-vile replaced the umber hulk (U) in the PC/Epyx port. Unix 5.4 uses the black unicorn there instead. Usenet accounts say the designers mainly wanted another monster starting with U.
**Original source:** The name comes from Stephen R. Donaldson's *Chronicles of Thomas Covenant* (1977) and **not** from Tolkien. In Donaldson's books, ur-viles are eyeless black creatures with a strong sense of smell, bred by Lord Foul. They are loremasters who fight in wedge formations.
**Local roguelike texts:** None found. Angband's "uruk" is Tolkien and unrelated.
**Sources:** https://groups.google.com/g/rec.games.roguelike.rogue/c/lkkab2FEYJM , https://en.wikipedia.org/wiki/Rogue_(video_game)

## Umber hulk
**Existing:** empty file.
**Flavor:** Looking into its glaring eyes leaves you confused, and it digs through rock as if it were sand.
**Rogue history / meta:** This is the U monster of Rogue 3.6. Its gaze confuses you. In 5.x that confusing attack moved to the medusa, while the U slot went to the black unicorn (Unix) or the ur-vile (PC).
**Original source:** The Greyhawk supplement (1975) and Monster Manual (1977). It is a huge insectoid burrower whose eyes cause confusion, and it tunnels through solid stone.
**Local roguelike texts:** Angband: "This bizarre creature has glaring eyes and large mandibles capable of slicing through rock." NetHack data.base has no entry for it.
**Sources:** ~/Games/angband-4.2.6/lib/gamedata/monster.txt:4208, ~/Games/rogue3.6/init.c

## Griffin
**Existing:** a flavor line only ("Gryphons are mythical"). It has a lowercase "his" and is otherwise fine.
**Flavor:** keep the existing line.
**Rogue history / meta:** The griffin took the G slot from Rogue 3.6's gnome and turned a weak early monster into a fast, late, flying killer. The ~/Games/roguepc help lists it among the deep monsters that "hit very hard".
**Original source:** Classical myth. It has the body of a lion and the head and wings of an eagle. Herodotus places griffins guarding gold in the far north. The griffin is also in OD&D/MM, but the name is generic.
**Local roguelike texts:** Angband (Gryphon): "It is half lion, half eagle. It flies menacingly towards you."
**Sources:** ~/Games/angband-4.2.6/lib/gamedata/monster.txt:4110, ~/Games/roguepc/web/dist/help.html

## Medusa
**Existing:** a flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** The medusa took the M slot from the mimic. Its mimic disguise went to the xeroc, and it inherited the umber hulk's confusing gaze. It is a deep monster.
**Original source:** Greek myth. She is the mortal Gorgon whose gaze turns onlookers to stone, and Perseus beheaded her using a mirrored shield. In D&D (MM 1977) the medusa also petrifies with her gaze.
**Local roguelike texts:** nethack50/dat/data.base, "medusa/perseus": Medusa is the only one of the three Gorgons in mortal form. It retells the myth in which Minerva changes her hair into serpents (Bulfinch).
**Sources:** ~/Games/nethack50/dat/data.base:3577

## Quasit
**Existing:** empty file.
**Flavor:** A small, evil cousin of the imp, quick to claw and quicker to vanish.
**Rogue history / meta:** This is the Q monster of Rogue 3.6. It was replaced in 5.x by the quagga, a real extinct zebra, which was a deliberately harmless-sounding swap away from D&D.
**Original source:** The Greyhawk supplement and Monster Manual (1977). It is a minor demon related to imps, with claws that sap Dexterity, and it can turn invisible and regenerate.
**Local roguelike texts:** nethack50/dat/data.base: "Quasits are small, evil creatures, related to imps," with poisonous talons. Angband: "A demon of small stature with an annoying bite."
**Sources:** ~/Games/nethack50/dat/data.base, ~/Games/angband-4.2.6/lib/gamedata/monster.txt:4306

## Xeroc
**Existing:** a flavor line only (it has typos: "creatured", "aleady").
**Flavor:** Another creature rumored to have been brought by Yendor from another dimension. As you already know, they can disguise themselves as almost anything.
**Rogue history / meta:** Rogue 5.x introduced it in the X slot that had belonged to the xorn, and it took over the mimic's job of posing as an item. rogue5.4/chase.c notes that a monster in the way "can also be a Xeroc, which we shouldn't step on". The name is a made-up pun on Xerox, the photocopier company, because it copies the look of objects.
**Original source:** None in D&D. Its behavior is the D&D mimic (MM 1977).
**Local roguelike texts:** Only code references (rogue5.4/chase.c and others). No lore text exists.
**Sources:** https://groups.google.com/g/rec.games.roguelike.rogue/c/lkkab2FEYJM , ~/Games/rogue5.4/chase.c:394

## Xeroc mk ii
**Existing:** no file found in the list. It looks like a monster invented for rx1.
**Flavor (suggestion):** Yendor's improved copier: it copies faster, sharper, and with teeth.
**Rogue history / meta:** It is not in any classic Rogue version (unverified across all variants). It extends the Xerox joke with a "Mark II" model number.
**Original source:** Same as the xeroc.
**Local roguelike texts:** None.
**Sources:** none

## Xorn
**Existing:** empty file.
**Flavor:** A three-legged, three-armed lump of living stone that walks through walls looking for its next meal of gems and metal.
**Rogue history / meta:** This is the X monster of Rogue 3.6. It was replaced in 5.x by the xeroc. Either way, X was always a hard letter to fill.
**Original source:** The Greyhawk supplement and Monster Manual (1977), from the Elemental Plane of Earth. It has radial symmetry, a mouth on top of its head, eats minerals, and phases through stone.
**Local roguelike texts:** nethack50/dat/data.base: "A distant cousin of the earth elemental," and it can make its body porous enough to pass through any obstacle. Angband: a huge creature of elemental Earth with four arms.
**Sources:** ~/Games/nethack50/dat/data.base, ~/Games/angband-4.2.6/lib/gamedata/monster.txt:8685

## Vampire
**Existing:** a flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** The vampire is the V monster in every version from 3.6 onward. It is one of the few D&D/folklore names that survived the cleanup, because the name is public domain. It drains hit points and experience levels, and it regenerates.
**Original source:** Slavic and Balkan folklore, made famous by Bram Stoker's *Dracula* (1897). In D&D (OD&D/MM) it drains levels.
**Local roguelike texts:** nethack50/dat/data.base: the passage from *Dracula* where Van Helsing lists the vampire's shapes (wolf, bat, mist, elemental dust).
**Sources:** ~/Games/nethack50/dat/data.base:5760

## Jabberwock
**Existing:** a flavor line only (it has typos: "larget", "dissapointed").
**Flavor:** Yendor was renowned for having the world's largest collection of these pernicious beasts. He was disappointed when nobody came to see them.
**Rogue history / meta:** The jabberwock took the J slot from Rogue 3.6's jackal and turned a level-1 pest into one of the deepest monsters. It is a literary choice that dodges D&D entirely.
**Original source:** Lewis Carroll's poem "Jabberwocky," in *Through the Looking-Glass* (1871). Its key line is "The jaws that bite, the claws that catch!" A boy kills the beast with a vorpal sword, and D&D borrowed the "vorpal" name from the poem.
**Local roguelike texts:** nethack50/dat/data.base, "jabberwock/vorpal*": it quotes the full poem, so cite it rather than reproduce it.
**Sources:** ~/Games/nethack50/dat/data.base:2831

## Dragon
**Existing:** a flavor line only.
**Flavor:** keep the existing line.
**Rogue history / meta:** The dragon is the D monster in every version and the toughest regular monster. Its breath is a flame bolt.
**Original source:** Myth everywhere. In the West it is the enemy of man (Saint George, Beowulf). In D&D it comes from OD&D, with Tolkien's Smaug in the background.
**Local roguelike texts:** nethack50/dat/data.base, "*dragon": a Western dragon leaves destruction and disease and attacks with sulphurous breath and a deadly tail (Deirdre Headon, *Mythical Beasts*).
**Sources:** ~/Games/nethack50/dat/data.base:1465
