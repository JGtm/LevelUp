# RECETTE — décodage kill-weapon pur-film (Halo Infinite Theater)

> Consolidation de la connaissance STABLE. But du chantier : pour chaque kill, sortir
> tueur/victime/arme/catégorie/assist/dégâts, PUR FILM (aucun CE au runtime), avec une
> couverture ≥ 85 %. Statut au 2026-07-10 : firearm résolu, couverture instable (dépend du
> marqueur), grenade/mêlée cadrés. Branche : `feat/filmdec-continuation`.
>
> **JOURNAL RE À JOUR (2026-07-11) : lire `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md`** — voie déser déterministe,
> trace CE live (base=24, locator validé 87%), leçon anti-cheat, chemin de reprise. Il consolide
> les avancées juillet et supersède l'index maître `../../README_KILLWEAPON_INDEX.md` (ère juin warp/dead-state).

## 1. Structure d'un film

- Fichiers `data/cache/film_chunks/<film>/chunk_XX.bin` (zlib). `chunk_27` = KILL FEED
  autoritatif (tueur/victime/temps, tous les kills du match, champ Gamertag présent).
- Les autres chunks = flux de FRAMES (paquets type-0). Chaque paquet type-0 commence par un
  octet MARQUEUR (`payload[0]`). Le décodage lit des champs bit-packés MSB-first.

## 2. Carte des MARQUEURS (le point clé du chantier)

| Marqueur | Rôle | État décodeur |
|---|---|---|
| `0xE6` | KILL-EVENT : `[readOpt5 victime][readOpt5 tueur][R32][R1][readOpt5 assist][R32]`. Position VARIABLE (locator `locateKillEventCursor`). Déser `FUN_14104bd08`. **AUCUN champ arme/cause.** | ✅ locator OK ; assist OK |
| `0xd2` | RECORD DE DÉGÂT firearm. Préambule base=24. Porte : attaquant (readOpt5 ~bit34), FAMILLE d'arme (`+0x10`, readOpt 1+32 = LA CAUSE), variant (const `0x42C9679F`), + TABLEAU DE HITS (montant de dégât). Déser `FUN_14080c1f8`. | ✅ famille + attaquant ; montant partiel |
| `0xC0 0xC2 0xC3 0xCA 0xD3 0xE9` | **FRÈRES** : frames ECS où le kill-event est un composant à position VARIABLE, souvent l'arme APRÈS le curseur. Curseurs dans la bande `[90,115)`. | ✅ locator OK depuis **keFloor=80** (Phase 1) ; ⚠️ arme parfois `cause-?` (arme après curseur → wRel=false) |
| `0x534(hit)/0x535(miss)` | ÉVÉNEMENT DE MÊLÉE (11 bits, bit-unaligned). `anchor=bp+3` ; type@`anchor+76` ∈ {0x42 normal, 0x47 marteau, 0x60 épée} ; arme high32 @`anchor+86/88/101`. | ✅ arme + variante |
| `0x4c0c00` | LANCER DE GRENADE (24 bits). id32 arme @`+24` (Frag `0xB0171062` / Plasma `0xC0E34C44` / Shock `0x3B2567D4` / Spike `0x9212E428`) ; lanceur 5b @`+103`. | ✅ arme + lanceur (LANCER, pas kill) |
| `0xD3` (aussi) | état d'ENTITÉ-ARME (id64 complet). Sert à nommer la variante mêlée (Diminisher/Rushdown). ATTENTION : un id64 d'arme à feu ici = arme TENUE (held), pas un kill. | ✅ pour mêlée pure |
| `0xA0` | flux de deltas ECS (bulk). Pas un record de kill. | — |

## 3. LE DRIVER DE COUVERTURE (pourquoi elle varie 54–99 %)

**La structure est identique entre films. Ce qui varie = SOUS QUEL MARQUEUR le film sérialise
ses kills.** Mesuré (mode `pipeline`, kills localisés par marqueur) :
- `0014603f` : 62/66 kills sous `0xd2` (propre) → couverture 81 %.
- `000d5950` : 12/52 sous `0xd2`, 40/52 sous frères (`0xC0/C2/C3/E9`) → couverture 56 %.

=> Le décodeur était MÛR sur `0xd2` mais le LOCATOR jetait les frères. **Le non-couvert
= décodable pas encore décodé** (bug locator), PAS des suicides/trahisons (~5 %).

**PHASE 1 RÉSOLUE (2026-07-10)** : la cause exacte = `keFloor=140` (plancher de scan du
kill-event) jetait les curseurs des frères, qui tombent dans `[90,115)`. Fix = `keFloor 140→80`
(un seul constant, `deserlen.go`). Résultat mesuré (pairmatrix, production, 5 films) :
couverture 000d5950 54→**78.5 %** (+24.7), 00502e52 58→**71.6 %**, 0215fe6b 69→**73.2 %** ;
accuracy MONTE partout (96.4–100 %). Non-régression : 0014603f inchangé (81.5 %/100 %), oracle
CE 129/134 identique. Couverture globale désormais **71.6–81.5 %**. Reste ~20 % = kills sans
kill-event localisable (mécanisme différent) + suicides/environnement légitimes.

## 4. ARCHITECTURE (contre-vérifiée Ghidra) — jointure MÊME-HORLOGE

Le film ne sérialise PAS de « code de cause » par-kill (les champs DamageEffect /
DamageReportingModifier sont TÉLÉMÉTRIE Xbox only). La catégorie d'un kill = **la source du
dégât FATAL** à son horloge (c'est ainsi que le jeu affiche le kill-feed au replay) :
- firearm = famille du `0xd2` (ou frère) fatal.
- mêlée = event `0x534/535` fatal.
- grenade = record de dégât d'EXPLOSION de grenade (À LOCALISER — Phase 2 ; PAS le lancer,
  PAS une corrélation temporelle). Le lancer `0x4c0c00` ≠ le kill.

## 5. DÉGÂTS (Phase 3)

- Le record `0xd2` porte le montant par hit dans le TABLEAU DE HITS (R32, film = brut quantifié).
  Déquant LIVE via table de plage `DAT_1451f98d0`.
- EHP plein Halo Infinite = **225** SELON LA CONSTANTE DU CODE (`games.DefaultEffectiveHpToKill`),
  NON encore vérifié depuis le film. Vérif film incontestable = décoder correctement le montant
  (la déquant `DAT_1451f98d0`) puis un kill one-shot / le max de dégât cumulé sur une victime.
  Mon décode `top16/256` cape à ~150 = MAL CALIBRÉ (à corriger en Phase 3).

## 6. Outils

`cmd/tmp_kwval` : modes `pipeline` (firearm + diag), `killfeed` (attribution unifiée +
assist ; `csv` pour capture), `pairmatrix` (accuracy vs chunk_27), `keallscan`, `famscan`.
`cmd/tmp_acurtis` : `melee`/`gren`/`struct`/`feed` (events mêlée/grenade).
Roster : `solveRoster` (index frame → XUID, permutation-invariant vs chunk_27). Gamertags :
`chunk27Gamertags` (champ Gamertag de chunk_27).
