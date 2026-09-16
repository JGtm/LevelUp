# Rapport de validation — recuisson du parc a la cloture M1 (2026-09-17)

Perimetre : les 76 artefacts de rejeu du parc `halo_infinite`, compares deux a deux entre
`data/cache/replays_schema54_avant_M1_2026-09-17/halo_infinite` (avant) et
`data/cache/replays/halo_infinite` (apres). Source unique : les 76 rapports `replaydiff.Rapport`
de `validation_parc/<id>.json`. Lecture seule sur les deux caches ; aucun decodage, aucun test Go,
aucun fichier du depot touche. Les chiffres ci-dessous sont recalcules a partir des 6 289 lignes
de `differences` des 76 rapports, avec UNE grille unique appliquee au parc entier — les quatre
classements par quart ayant range differemment deux ou trois metriques (section 7).

## 1. Totaux

| Grandeur | Valeur |
|---|---|
| Artefacts | **76 / 76** |
| Schemas | **54 -> 60** sur les 76 (aucune exception) |
| Gains | **4 030** (1 515 `gain` + 2 515 `apparu`) |
| Pertes | **2 235** (2 005 `perte` + 230 `disparu`) |
| Changements | **24** |

Controle de coherence : ces trois totaux egalent exactement la somme des colonnes de
`index.tsv` (gains 4 030, pertes 2 235, changements 24), et la somme des quatre quarts
(538 + 668 + 542 + 487 = 2 235).

Pertes par axe : `pistes` 1 206, `couverture` 969, `ports` 22, `vehicules` 19, `carte` 10,
`armes` 9. **ZERO perte** sur les axes `morts`, `score`, `grenades`, `objectifs`,
`objets-objectif`, `joueurs`, `roster`, `entete`, `horloges`, `equipement`, `autres:identity` —
aucune ligne de difference n'y existe, dans aucun sens. Les 9 pertes de l'axe `armes` sont
toutes `pickups.origin/presents` (famille B) : **aucune ligne `shots.*` ne baisse** sur le parc.

## 2. Pertes par famille — parc entier

| Famille | Pertes | Artefacts |
|---|---|---|
| A — moins de vies et plus longues (lot 1.9.13) | **1 473** | 70 |
| B — poses d'equipement relues par la grammaire (1.9.1) | **475** | 74 |
| C — compteurs de DEFAUT en baisse (perte par convention) | **187** | 75 |
| D — bornes elargies (`bounds.min*`) | **10** | 7 |
| E — vies sans nom rendues visibles | **5** | 4 |
| F — amorcages de bombe | **0** | 0 |
| G — trajet de vehicule arrete a la destruction ecrite | **0** | 0 |
| **H — denominateur de couverture des objectifs restreint** (hors §5, section 4) | **42** | 21 |
| **I — neutralite des socles de drapeau** (hors §5, section 4) | **29** | 4 |
| **RESIDU hors famille apres arbitrage** | **14** | 7 |
| **Total** | **2 235** | 76 |

Par quart (meme grille) :

| Quart | A | B | C | D | E | H | I | Residu | Total |
|---|---|---|---|---|---|---|---|---|---|
| 1 (01e1f945..4577fcc4) | 351 | 121 | 50 | 2 | 0 | 6 | 1 | 7 | 538 |
| 2 (46c3f91d..8a485699) | 451 | 130 | 60 | 2 | 4 | 18 | 3 | 0 | 668 |
| 3 (8bc6074f..b8a44fe8) | 347 | 112 | 37 | 5 | 1 | 10 | 25 | 5 | 542 |
| 4 (bc60b4d9..fccc61cd) | 324 | 112 | 40 | 1 | 0 | 8 | 0 | 2 | 487 |

F et G sont a zero : aucun `coverage.bombArmings.*` n'existe dans le parc, et aucune metrique
`vehicles.rides*` / `coverage.vehicles.aimRideFrames` n'y BAISSE (elles apparaissent ou montent —
sur `0a44c6cc` la chaine vehicule nait de rien : `rides` 0 -> 6, `aimRideFrames` 0 -> 3 006).

## 3. Arbitrages rendus a l'interieur des familles A-G

Quatre motifs comptes HORS par au moins un classement de quart sont verses a une famille du plan,
sur piece :

- `coverage.bridge.directByCreationPropagated` (**70 lignes, 70 artefacts**, effondrement a 0/1
  partout) -> **C**. Le plan la DECLARE lui-meme, a la ligne du gate 1.9.13 du §5 :
  « (B) compteurs de DEFAUT qui tombent — 38 : `bridge.directByCreationPropagated` **13** (le
  record de creation ouvre desormais LA vie du corps, plus le premier de ses sejours : il tombe a
  0 ou 2 partout) ». Le lettrage du gate (A/B/C) n'est pas celui de la consigne (A..G) : ce motif
  est un compteur de defaut, donc C. Controle de non-regression du nommage : `bridge.livesNamed`
  ne baisse que sur 3 artefacts (-2, -2, -1, dans la borne E), `unnamedLives` ne monte que sur 2.
- `coverage.pickups.unknownFamilies` (**14 lignes**) -> **C**. Compteur de defaut en baisse
  (jusqu'a 17 -> 0 sur `94a28b8b`), contrepartie en gain dans le meme rapport (`weaponLabels/n`
  monte, des `pickups/par-family/<arme>` apparaissent). Meme convention que `pickups.originUnknown`,
  deja enumere par C.
- `coverage.grenadeReads.ammoRefused` (**3 lignes** : `0d265ab0`, `94a28b8b`, `a36c8bed`,
  `true` -> absent) -> **C**. Ce booleen est « le VERDICT de la porte du canal munitions »
  (`film_inputs.go:66`), `omitempty` : il disparait parce que le refus n'a plus lieu. Sur
  `0d265ab0` le canal grenade GAGNE au meme moment (`grenadeReads/n` 377 -> 379, `g/n` 1 508 -> 1 516,
  `fromDelta` 188 -> 190).
- `coverage.equipment.camoLives` (**5 lignes**) -> **A**. C'est un compte de VIES : il baisse sur
  les 5 artefacts strictement moins que leur `equipment.tracksTotal` (9->8 pour 81->73 ; 31->28
  pour 81->72 ; 24->23 pour 107->103 ; 19->18 pour 92->90 ; 36->35 pour 113->105). Vies fusionnees.

## 4. Deux familles ABSENTES du §5 — et pourquoi elles sont legitimes

Le corpus gate de cloture (14 temoins, `--base=8f35efb72`) mesurait M1 **base contre tete**. La
recuisson du parc, elle, compare des artefacts cuits AVANT les finitions v7.5 a une tete qui les
contient : l'item **F.4 « Recuisson du parc : NON LANCEE — decision de l'utilisateur »**
(`PLAN_FINITIONS_2026-09-13.md:286`) signifie que deux correctifs livres le 13/09 ne se voyaient
nulle part encore. Ils se voient ici, et NULLE PART AILLEURS que sur leurs propres compteurs.

**H — `coverage.objectives.available` / `.attached` / `.refusedByRoster` (42 lignes, 21 artefacts).**
Lot D10 (journal du 2026-09-13, « pulses d'objectif et denominateur de couverture restreints aux
familles d'objectif ») : `doc.objectives` portait aussi `kills` et `assists`, qui gonflaient le
denominateur. Le journal cite les DEUX chiffres qu'on retrouve ici a l'unite : « `8bc6074f` porte
218 actions dont 119 `kills`/`assists` [...] `coverage.objectives.available` 218 -> 99 » (mesure :
218 -> 99) et « `32d9a94f` (Strongholds) : disponibles 148 -> 55 » (mesure : 148 -> 55). Le meme
journal prevenait : « la PUBLICATION ne perd rien [...] seule la COUVERTURE se restreint » et
« les artefacts deja cuits gardent l'ancien denominateur : le correctif ne se voit qu'a la
recuisson ». Verification independante : sur les 76 artefacts, l'axe `objectifs` et l'axe
`objets-objectif` n'ont **aucune ligne de difference** — les actions d'objectif publiees sont
identiques au bit pres. Ce n'est pas une perte de lecture, c'est un denominateur assaini.

**I — calque de drapeau (29 lignes, 4 artefacts : `bc60b4d9`, `bf5ced1b`, `396cfc92`, `64e8adfa`).**
Lot H des finitions (journal du 2026-09-13) : « neutralite des socles de drapeau en champ
explicite — 8 socles `team_index=-1` sortis du panier neutre, 63 neutres inchanges ». La
decouverte D1 de `RAPPORT_PASSAGES_DRAPEAU_2026-09-11.md` nomme le film : « **`bc60b4d9`
(Illusion) declare TROIS socles de drapeau, dont DEUX pour l'equipe 0** [...] `carrierTeamUnknown = 10`,
le seul film du parc a en porter ». Apres recuisson : `flagCarries/n` **3 -> 2**,
`carrierTeamUnknown` **10 -> disparu**. Controle decisif sur piece : les spans perdus sont
EXCLUSIVEMENT des spans « drapeau au socle » — `bc60b4d9` perd 2 spans et exactement 2
`par-state/home` (75->73, 16->14) ; `bf5ced1b` perd 1 span et exactement 1 `par-state/home`
(27->26, 6->5). **Aucun span de PORTAGE n'est perdu sur le parc** ; ce qui disparait est le
sejour au socle d'un troisieme drapeau qui n'existait pas. `396cfc92` ne porte que la trace
amont (`coverage.flagCarries.spawns` 3 -> 2, aucune perte sur l'axe `ports`).

## 5. Pertes qui restent HORS FAMILLE apres arbitrage — 14 lignes, 7 artefacts

| # | id | metrique | ancien -> nouveau | hypothese |
|---|---|---|---|---|
| 1 | `a6ae19fb` | `coverage.abilityImpulses.reads` | 2 -> **0** | Lectures BRUTES `tag == 1` du canal i57/i59 (`Reads = len(in.reads)`, `document_ability_impulses.go:134`) : ce compteur ne depend NI des vies NI de l'origine. Deux lectures de moins alors que tous les autres canaux du meme film montent (`grenadeReads/n` 333->338, `inventory/n` 167->173, `loadouts/n` 135->140). Aucun mecanisme declare ne l'explique. |
| 2 | `a6ae19fb` | `coverage.abilityImpulses.episodes` | 1 -> **0** | Consequence de la ligne 1 (l'unique episode etait forme par ces lectures) ; il etait `noIdentity` (1 -> 0), donc non publie. |
| 3 | `0d265ab0` | `coverage.abilityImpulses.reads` | 2 -> 1 | Idem ligne 1, amplitude 1. |
| 4 | `0d265ab0` | `coverage.abilityImpulses.episodes` | 2 -> 1 | L'episode perdu etait `noIdentity` (2 -> 0) ; `otherFamily` 0 -> 1 : reclassement partiel. |
| 5 | `1b2d9e08` | `coverage.abilityImpulses.reads` | 3 -> 2 | Idem ligne 1, amplitude 1. |
| 6 | `1b2d9e08` | `coverage.abilityImpulses.episodes` | 3 -> 2 | Episode perdu non publie (`noIdentity` 3 -> 2). |
| 7 | `2cf24f30` | `coverage.abilityImpulses.reads` | 3 -> 2 | Idem ligne 1, amplitude 1. |
| 8 | `2cf24f30` | `coverage.abilityImpulses.episodes` | 2 -> 1 | Episode perdu non publie (`noIdentity` 1 -> 0). |
| 9 | `a396aa7f` | `coverage.abilityImpulses.reads` | 47 -> 46 | Idem ligne 1, amplitude 1. |
| 10 | `a396aa7f` | `coverage.abilityImpulses.episodes` | 24 -> 23 | Episode perdu non publie (`otherFamily` 2 -> 1). |
| 11 | `a396aa7f` | `coverage.abilityImpulses.otherFamily` | 2 -> 1 | Compte l'episode de la ligne 10 ; sa baisse suit, elle ne l'explique pas. |
| 12 | `bfcd1175` | `coverage.abilityImpulses.reads` | 112 -> 111 | Idem ligne 1, amplitude 1, sur le film le plus fourni du canal. |
| 13 | `bfcd1175` | `coverage.abilityImpulses.episodes` | 57 -> 56 | Episode perdu non publie (`noIdentity` 11 -> 9, `published` 39 -> **40**). |
| 14 | `0a44c6cc` | `coverage.vehicles.shotsNoRide` | 0 -> **83** | Compteur de defaut a POLARITE INVERSE, classe « perte » parce qu'il MONTE. Rien n'est perdu : au schema 54 la chaine vehicule n'existait pas du tout sur ce film (`rides` 0 -> 6, `shots` 0 -> 30, `aimRideFrames` 0 -> 3 006). C'est un TROU RENDU VISIBLE, pas une regression — mais le ratio **83 tirs hors trajet contre 30 rattaches** est a instruire. |

Circonstance attenuante commune aux lignes 1-13 : **`coverage.abilityImpulses.published` ne baisse
sur AUCUN artefact du parc** (aucune ligne `*.published` en perte, tous sens confondus) ; les
impulsions publiees MONTENT sur 9 artefacts (jusqu'a 50 -> 62 sur `d1dfbc02`). Ce qui se perd est
confine aux episodes non identifies, donc non publies. Le residu est materiellement mince — 8
lectures brutes sur 6 films de 76 — mais il n'est explique par aucune famille ni par aucune piece
ecrite, et un `reads` qui baisse est une IMPULSION NON LUE dans le film, pas un compteur de defaut.

**Mesure a faire pour clore** : comparer le nombre de `filmdec.AbilityImpulse` rendus par le
balayage sur `a6ae19fb` entre le decodeur de base et le decodeur M1 (2 lectures attendues avant,
0 apres) ; si le canal i57/i59 est desormais refuse pour ce film, le dire et le compter
(`ComponentAbsent` ou un compteur nomme), jamais le laisser en zero muet.

## 6. Changements

24 changements au total. **23 sont dans les changements attendus** :

- 20 voies de nommage `coverage.bridge.namedByNextLife` / `namedByPreviousLife` 0 -> 1 (ou 0 -> 2
  sur `daaa17d6`), sur 17 artefacts ;
- 2 reattributions de piste par xuid sur `64e8adfa` (`tracks/par-xuid/2533274792763167` et
  `tracks/vies-par-xuid/2533274792763167`, 15 -> 14) ;
- 1 reattribution de duree de trajet sur `4f77afc1`
  (`vehicles.rides/duree-totale/par-xuid/2535437483090324`, 1 289 -> 1 157).

**1 changement hors des attendus** :

| id | metrique | ancien -> nouveau | arbitrage |
|---|---|---|---|
| `64e8adfa` | `coverage.flagCarries.homeByObject` | 10 -> 11 | Famille **I** (socles de drapeau, lot H des finitions). Meme artefact : `carrierTeamUnknown` apparait a 13, `ambiguousReturns` 1 -> 2, `closedByHome` 2 -> 1, `ownFlagRefused` 2 -> 1 — mais le calque GAGNE une portee (`flagCarries.spans/n` 200 -> **201**). L'arbitrage par equipe s'abstient plus souvent sur ce film ET publie une portee de plus : abstention comptee, pas donnee perdue. A verser au registre du lot H comme second temoin (le premier etant `bc60b4d9`). |

## 7. Divergences entre les quatre classements, tranchees

| Motif | Quart 1 | Quart 2 | Quart 3 | Quart 4 | Arbitrage de ce rapport |
|---|---|---|---|---|---|
| `bridge.directByCreationPropagated` | HORS | `A*` | HORS | A | **C** (declare au §5, ligne du gate 1.9.13) |
| `pickups.unknownFamilies` | HORS | `C*` | HORS | C | **C** |
| `equipment.camoLives` | HORS | `A*` | HORS | A | **A** |
| `coverage.objectives.*` | HORS (« defaut franc ») | HORS | HORS | HORS | **H** — correctif D10, non defaut |
| `flagCarries.*` (axe `ports`) | n/a | n/a | HORS | HORS | **I** — correctif lot H, non defaut |
| `abilityImpulses.reads` / `.episodes` | HORS | n/a | HORS | HORS | **HORS confirme** (residu, section 5) |
| `vehicles.shotsNoRide` | HORS (« anomalie ») | n/a | n/a | n/a | **HORS confirme** (trou rendu visible) |

Les quatre classements sont d'accord sur l'essentiel : meme total (2 235), memes familles A/B/D/E,
memes axes intacts. Les ecarts portent tous sur des motifs que le plan n'enumerait pas
litteralement, et deux d'entre eux (objectifs, drapeaux) avaient ete remontes comme DEFAUTS FRANCS
par le quart 1 — ils ne le sont pas : ce sont deux correctifs de finitions livres le 13/09 dont la
recuisson etait restee en attente (F.4).

## 8. Verdict

> **14 pertes inexpliquees (7 artefacts), sauvegarde a garder.**

La recuisson est saine sur tout ce qui compte : 0 perte sur les points, les tirs, les kills, les
grenades, les objectifs publies, le score, les projectiles, le roster et les horloges ; 2 221
pertes sur 2 235 sont adossees a un mecanisme ecrit et date. Le residu tient en un seul canal
(impulsions de capacite, 13 lignes sur 6 films) plus un compteur a polarite inverse. Il est
mince, il ne touche aucune donnee publiee — et il n'est pas explique. Le cache
`replays_schema54_avant_M1_2026-09-17` reste le SEUL moyen de refaire la mesure : il se supprime
le jour ou la ligne 1 du tableau de la section 5 est fermee sur piece.
