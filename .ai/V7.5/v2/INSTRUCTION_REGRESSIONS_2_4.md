# Instruction des regressions 2, 3 et 4 du balayage du parc — 2026-09-06

Branche `feat/v2-regressions`, worktree `LevelUp-wt-v2-regressions`, base `9e73368e8` (= `feat/v75`
avec les sept lots et le correctif CTF, schema 40).

Source : `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md` sections 6.2, 6.4 et 7. La candidate 1 (actions
d'objectif de CTF) etait deja instruite et corrigee (`RoundIdentity.CompletedByLines`, schema 40) ;
elle est verifiee close ici en passant (voir la note de fin).

---

## Verdict en une page

| # | Candidate | Verdict | Cause |
|---|---|---|---|
| **4** | Un joueur perd toutes ses vies nommees (3 matchs cites, **18** au parc) | **REGRESSION**, corrigee | `nameClosedLives` re-devinait la vie a nommer au lieu d'utiliser celle que la fermeture avait DESIGNEE |
| **2** | Tractions de grappin −10 a −40 % (**16** matchs, **54** tractions) | **REGRESSION**, corrigee | `buildGrappleLines` bornait chaque traction a la DERNIERE vie du slot |
| **3** | Episodes camo / surbouclier −1 a −2 (**11** matchs, **17** episodes) | **REGRESSION** pour 10 matchs, corrigee · **1 cas est un GAIN documente** | `trackFrameWindows` n'indexait qu'UNE fenetre par slot ; le cas restant est un episode ancre sur un point de trajectoire aberrant |

**Les trois candidates ont UNE SEULE cause racine** : le commit `48cf4905d` (2026-09-02,
schema 36, « une track = une vie ») a decoupe les pistes a `lifeGapUS`, et **trois consommateurs
ont continue de supposer qu'un slot ne porte qu'une piste**. Chacun ne retenait que la DERNIERE
et jetait en silence ce qui appartenait aux vies anterieures. Le message du commit annonce
pourtant « les fermetures nomment la vie qu'elles closent » : l'intention etait juste, le code ne
la realisait pas — le meme ecart entre le message et la mesure que pour la candidate 1
(`d173b1a8c` : « neutralite mono-manche prouvee par construction », refutee sur 14 films).

Schema bumpe **40 -> 41** : la sortie de cuisson change, donc un artefact 36 a 40 doit se lire
« a re-cuire ».

---

## Methode et environnement

Cuisson par le chemin de production (`cmd/replay-build` = `replaybuild.NewBuilder` + `BuildMatch`,
verrou `filmproc.AcquireSolo`, plafond 3 Gio, priorite basse), **un film a la fois**, racine de
travail dediee sous le scratchpad ou `data/cache/film_chunks`, `film_manifests`, `mvar` et
`data/titles` sont des jonctions en lecture vers le checkout principal et `data/cache/replays` un
vrai dossier. Les FAITS du match sont passes a chaque cuisson (`--facts`) et leur lecture est
verifiee film par film (`0` occurrence de « faits du match illisibles », `coverage.score` non vide) :
sans eux la courbe de score et les actions d'objectif tombent, et toute comparaison serait fausse.
Pic memoire maximum observe : **0,55 Gio** (`4f77afc1`, 64 chunks).

Comparaison par `cmd/replay-diff` (lecture generique, worktree du balayage) et par lecture directe
des artefacts. Le parc du checkout principal n'a recu aucune ecriture.

---

## Candidate 4 — les vies nommees

### 4.1 Re-mesure (temoin `145908d1`, BTB:CTF, Breaker, 14 chunks)

Artefact de reference du parc au schema 20 contre la cuisson au HEAD (schema 40) :

| | reference s20 | HEAD s40 |
|---|---|---|
| pistes publiees | 67 | 91 |
| **pistes nommees** | **53** | **51** |
| identites distinctes | 24 | **23** |
| `coverage.bridge.slots` | **53** | **53** |
| slots dont le document publie les TIRS mais dont aucune piste n'est nommee | **0** | **2** (29 tirs) |

Joueur par joueur, les deux vies perdues :

| slot | reference s20 | HEAD s40 |
|---|---|---|
| 562 | une piste `[1439..2080]`, nommee `2533274905069348` | deux pistes `[1439..1452]` et `[1578..2080]`, **aucune nommee** |
| 570 | une piste `[1760..2010]`, nommee `2535406820545228` | deux pistes `[1760..1798]` et `[2009..2010]`, **aucune nommee** |

### 4.2 Verite terrain

Le pont est **identique** dans les deux etats (`coverage.bridge` : 53 slots, 46 par lecture,
7 fermetures par reapparition, 0 collision) : le film dit exactement la meme chose, seule la
PUBLICATION a change. Et le document publie **17 tirs sur le slot 562 et 12 sur le slot 570** —
la porte des tirs exige le pont, donc le document affirme connaitre le proprietaire de ces slots
tout en laissant leurs pistes anonymes. Ce n'est pas une abstention prudente, c'est une
incoherence interne.

### 4.3 Bissection

Meme film, meme outil, trois etats cuits :

| commit | schema | pistes nommees | slots orphelins |
|---|---|---|---|
| `48cf4905d^` (`4c569cbb0`) | 35 | **53** | 0 |
| **`48cf4905d`** | 36 | **51** | 1 |
| HEAD `9e73368e8` | 40 | 51 | 2 (le second slot n'a de tirs publies que depuis le schema 39) |

**Ligne** : `apps/go-api/internal/analysis/replay/owners.go`, `nameClosedLives` — « pose l'identite
d'une fermeture sur l'UNIQUE vie anonyme du slot ferme », avec `anon = -2 // plusieurs vies
anonymes : on ne tranche pas`.

### 4.4 Cause, sur pieces

Les deux fermetures (`closures.go`) raisonnent sur **une vie precise** : A designe « l'unique corps
libre dont l'intervalle couvre l'instant du tir », B « la vie qui commence une reapparition apres
cette mort ». Mais `closeBridge` ne rendait que la table `slot -> index`, et cette designation
etait **jetee**. Le nommage devait alors la re-deviner, avec un critere plus faible (« l'unique vie
anonyme »), et s'abstenait des qu'un slot en portait deux.

C'est le meme motif que la candidate 1 : une information disponible est jetee, puis remplacee par
un canal moins couvrant.

### 4.5 Correctif

`closureReport.closedLife` (slot -> **indice de la vie designee**, -1 quand deux vies distinctes
ont ete designees) est renseigne par les deux fermetures au moment exact de l'attribution
(`noteLife`), et `nameClosedLives` pose l'identite sur CETTE vie. `freeLives` et `livesCoveringAt`
travaillent desormais sur des indices et non sur des copies. **Aucune decision du pont n'est
modifiee** : les gardes de contestation, la corroboration et les compteurs sont inchanges — verifie
sur le temoin (`slots=53 viesNommees=46 viesTotal=93` avant et apres).

`closures.go` franchissait les 500 lignes : la fermeture B est sortie dans `closures_respawn.go`,
deplacement pur, sans changement de logique.

### 4.6 Resultat mesure

| etat | pistes nommees | identites distinctes |
|---|---|---|
| reference s20 / cuisson s35 | 53 | 24 |
| schemas 36 a 40 | 51 | 23 |
| **avec le correctif** | **53** | **24** |

Sur les deux autres matchs cites par le balayage : `11de8353` 17 -> **19** identites (18 a la
reference : le nommage par vie en couvre desormais plus que le nommage par slot), `4f77afc1`
23 -> **24** (24 a la reference).

### 4.7 Ce que le correctif ne fait pas, volontairement

Sur le slot 562, la vie nommee est `[1439..1452]` (celle que la reapparition designe) ; la seconde,
`[1578..2080]`, reste anonyme bien qu'elle commence **a 4 cm** de la fin de la premiere — le corps
est immobile a son apparition, le film ne retransmet rien pendant 12,6 s, et `lifeGapUS` (5 s)
coupe la vie en deux. Nommer la seconde par contiguite serait l'heritage de slot que le schema 36 a
precisement supprime. Le cas est porte au registre des reports comme decouverte.

### 4.8 Tests

`closures_test.go` — trois tests, chacun **prouve par mutation** :

- `TestFermetureBNommeLaVieDesigneeQuandLeSlotEnPorteDeux` (rouge si la designation est jetee :
  « la fermeture devait DESIGNER la vie d'indice 1 ») ;
- `TestFermetureANeTranchePasEntreDeuxViesDuMemeSlot` (rouge si la garde du depouillement des tirs
  est retiree : « la designation doit valoir -1, obtenu 1 ») ;
- `TestFermetureBNeTranchePasQuandDeuxMortsDesignentLeMemeSlot` (rouge si la garde de `noteLife`
  est retiree : « obtenu 2 ») ;
- `TestNameClosedLivesNeReecritJamaisUneVieLue` (la lecture prime sur la deduction).

---

## Candidate 2 — les tractions de grappin

### 2.1 Re-mesure (temoin `879a4dba`, Fortitude, 24 chunks)

| | reference s34 | HEAD s39 |
|---|---|---|
| `coverage.grapple.lightReads` (tirs lus) | 32 | **32** |
| `coverage.grapple.heavyReads` (accroches lues) | **23** | **23** |
| `grappleLines` publiees | **23** | **15** |
| `pullLives` | 14 | 11 |

### 2.2 Verite terrain — tractions PERDUES, pas faux positifs elimines

**Les 23 accroches sont lues dans le film dans les deux etats** (`heavyReads` identique) : le
decodage n'a rien perdu, c'est la publication qui jette. La question posee par le balayage — faux
positifs elimines ou tractions perdues ? — est donc tranchee : **perdues**.

Les 8 tractions disparues, confrontees a la decoupe des vies :

| slot | vies au HEAD | tractions perdues | traction publiee |
|---|---|---|---|
| 543 | `[757..878]` `[1719..1732]` | 836-850 | — |
| 584 | `[1797..2080]` `[2161..2705]` | 1870-1888, 1912-1925, 1983-1996, 2026-2042 | — |
| 587 | `[1833..2287]` `[2414..2438]` | 2072-2086 | — |
| 631 | `[2821..3163]` `[3213..3215]` `[3307..3381]` `[3460..3661]` | 3046-3061, 3123-3139 | 3642-3654 |

**Chaque traction perdue appartient a une vie qui n'est pas la derniere de son slot** ; les slots
dont la derniere vie couvre la traction (520, 565, 578) n'ont rien perdu.

### 2.3 Bissection

Cuisson du meme film au commit pivot : `48cf4905d` (schema 36) rend **15** tractions
(`pulls=15 pullLives=11`), contre 23 a la reference s34 qui le precede. Le chantier « usages
d'equipement » du schema 38, soupconne par le balayage, est **hors de cause**.

**Ligne** : `apps/go-api/internal/analysis/replay/grapple_lines.go` —
`byTrack[tracks[i].Slot] = &tracks[i]` (une seule piste retenue par slot), puis dans `grappleLine`
le clamp `t0 < track.StartFrame` / `t1 > track.EndFrame` suivi de `if t1 <= t0 { return ..., false }`.

### 2.4 Correctif

`byTrack` porte **toutes** les vies du slot ; `grappleLine` choisit celle qui couvre l'ACCROCHE
(a defaut, celle qui couvre le tir), via `lifeCovering`. `coverage.grapple.pullLives` compte
desormais des vies (cle slot + image de debut) et non des slots — le nom du compteur disait deja
ce qu'il devait mesurer.

### 2.5 Resultat mesure

| match | reference | schemas 36-40 | avec le correctif |
|---|---|---|---|
| `879a4dba` | 23 tractions / 14 vies | 15 / 11 | **23 / 15** |
| `084a804d` | 71 / 33 | 61 / 29 | **71 / 34** |

Dans les deux cas, **tractions publiees = accroches lues** (23/23 et 71/71).

### 2.6 Test

`TestBuildGrappleLines_UneTractionDUneVieAnterieureEstPubliee` — deux vies d'un meme slot, une
accroche dans chacune ; **prouve par mutation** (retour a la derniere vie du slot : « 1 traction(s),
attendu 2 »).

---

## Candidate 3 — les episodes de camouflage et de surbouclier

### 3.1 Re-mesure

| match | axe | reference s20 | HEAD s39 |
|---|---|---|---|
| `82f29378` (Oasis) | surbouclier | 1 episode / 1 vie | **0 / 0** |
| `13d92593` (Dredge) | surbouclier | 1 episode / 1 vie | **0 / 0** |
| `084a804d` (Fortitude Heavies) | camouflage | 15 episodes / 9 vies | **13 / 8** |

### 3.2 Cause, sur pieces

`equipment_episodes.go`, `trackFrameWindows` : `out[t.Slot] = [2]int{...}` — **une seule fenetre par
slot**, ecrasee par la derniere vie. `episodeAccum.close` borne ensuite l'episode a cette fenetre,
et `if t1 < t0 { return }` le supprime des qu'il appartient a une vie anterieure. Meme cause,
meme commit `48cf4905d` que les deux autres candidates.

### 3.3 Correctif

`trackFrameWindows` rend **toutes** les fenetres du slot ; `windowFor` choisit celle qui recouvre le
plus l'episode (recouvrement, pas appartenance : une lecture peut tomber dans un trou de
replication) ; `finish` ferme l'episode sur la vie de son OUVERTURE, jamais sur la derniere du slot.

### 3.4 Resultat mesure

| match | reference | HEAD | avec le correctif |
|---|---|---|---|
| `82f29378` | 4 episodes (3 camo + 1 surbouclier) | 3 | **4** |
| `084a804d` | 15 camo / 9 vies, 6 surbouclier / 5 vies | 13 / 8, 6 / 5 | **15 / 9, 6 / 5** |
| `13d92593` | 1 surbouclier | 0 | **0** — voir ci-dessous |

### 3.5 `13d92593` n'est PAS une regression : c'est un gain deja documente

L'episode perdu vaut `{"slot":538,"fam":"overshield","t0":3603,"t1":3603}` — **duree nulle**. La
piste du slot 538 s'etendait au schema 20 de la frame 1270 a **3603**, mais son dernier point est
a **267 unites** du precedent, 194,6 s plus tard : c'est lui, et lui seul, qui donnait au document
les bornes de scene `minX -227,27 · minY -140,23 · minZ 33,26`, contre `-18,57 · -16,99 · 64,68`
aujourd'hui. L'assainissement des trajectoires (section 6.3 du balayage, « les points supprimes
etaient aberrants, et les bornes le prouvent ») a retire ce point ; l'episode qui s'y ancrait est
parti avec lui. **Rien a corriger** — et le correctif ci-dessus ne le fait pas revenir, ce qui est
verifie sur pieces (cuisson : `episodesSurbouclier=0`).

### 3.6 Tests

`TestEpisodeDUneVieAnterieureEstPublie` et
`TestEpisodeOuvertEnFinDeVieAnterieureSeFermeSurSaPropreVie` — **prouves par mutation** (une seule
fenetre par slot : « 1 episode(s), attendu 2 » et « 0 episode(s), attendu 1 »).

---

## Perimetre : aucun quatrieme calque

Recensement des indexations par slot du paquet `analysis/replay` : `flag_carries`, `skull_carries`,
`neutral_deaths`, `objectives`, `zone_attribution`, `zone_states_hill`, `vehicle_shots` et
`published_tracks` indexent par **xuid**, par **frame** ou accumulent (`append`) — aucun n'ecrase une
entree slot -> piste. Les deux seuls consommateurs qui supposaient « un slot = une vie » sont ceux
corriges ici.

## Impact sur le parc

Releve sur les 161 paires du balayage :

| candidate | matchs distincts | volume perdu |
|---|---|---|
| 2 — grappin | **16** | 54 tractions |
| 3 — episodes | **11** (dont 1 justifie : `13d92593`) | 17 episodes |
| 4 — vies nommees | **18** (le balayage n'en citait que 3, sur le seul critere « xuid entierement perdu ») | 2 identites entierement perdues, 26 vies nommees |

Tout artefact cuit entre le **2026-09-02** (`48cf4905d`) et ce correctif est concerne des lors que
son film porte un slot recycle — c'est-a-dire la quasi-totalite des matchs de plus de quelques
minutes. La propagation passe par la re-cuisson de release (`backfill-replay`), que le bump de
schema 40 -> 41 rend obligatoire.

## Schema

`SchemaVersion` **40 -> 41**, chronique ecrite dans `document.go` et raison dans le ratchet
`structure_test.go`. Le golden d'assemblage a ete regenere : son **unique** ecart est la ligne de
version (`schema 40` -> `schema 41`, 1 ligne sur 606, verifie par diff avant regeneration) — les
correctifs ne changent rien sur le film de reference `000d5950`. Le contrat OpenAPI declare
`schemaVersion` comme un `integer` sans `enum`, `const`, `default` ni `example` : un bump ne le
deplace pas.

## Gates (tous verts, 2026-09-06)

```
go test -count=1 ./internal/analysis/filmdec/... ./internal/analysis/replay/...
        ./internal/replaybuild/... ./internal/archlint/...            ok (x5)
go test -tags=integration -p 1 -count=1 ./internal/api/wire/...       ok 20,5 s
go build ./...                                                        exit 0
golangci-lint run --new-from-merge-base=origin/main ./...              0 issues
```

Les temoins inconditionnels (`TestGoldenMiniBobine`, `TestEquivalenceMiniFilm`, goldens des
familles) sont dans ces suites et restent verts.

## Note : la candidate 1 est verifiee close

La cuisson de `145908d1` au HEAD publie **7 actions d'objectif**, exactement comme son artefact du
parc au schema 20 — le balayage en mesurait **0** (« perd la totalite »). L'axe `objectifs` de la
comparaison reference/HEAD ne porte plus **aucun** ecart sur ce match. Le correctif
`RoundIdentity.CompletedByLines` fait ce qu'il annonce.

## Decouvertes, notees et NON traitees (registre des reports)

1. **Une vie coupee par un trou de replication a corps immobile.** `145908d1` slot 562 : deux vies
   separees de 12,6 s dont la seconde commence a 4 cm de la fin de la premiere, sans mort entre les
   deux. `lifeGapUS` (5 s) coupe une vie que rien ne separe ; la seconde reste anonyme et porte
   pourtant 17 tirs publies. Fusionner sur preuve de continuite (spatiale ET absence de mort)
   fermerait le dernier cas d'incoherence interne — decision produit, hors mandat.
2. **Le pont se laisse ecraser quand deux morts designent deux vies du meme slot.** `closeByRespawn`
   ecrit `owner[slot]` deux fois de suite, le dernier gagne, sans contestation comptee. Le correctif
   ci-dessus refuse au moins de nommer une vie dans ce cas (`closedLife = -1`), mais la faiblesse du
   pont est anterieure et n'est pas traitee : elle demande de decider si un slot peut legitimement
   changer de proprietaire en cours de match.

---

## Corrections apres la revue adversariale REG-R1 (2026-09-06)

La revue confirme le diagnostic sur pieces (les trois lignes fautives relues a la base),
l'additivite element par element sur quatre films cuits, les sept mutations, et 22 conditions
verifiees. Cinq constats sont traites ci-dessous, **chacun prouve rouge puis vert**.

### C1 — une traction publiee par la base etait jetee, meme sur un slot MONO-VIE

**Le constat, et il refute une de mes conditions.** Quand ni l'image de l'accroche ni celle du
tir ne tombe dans une fenetre publiee du slot, `lifeCovering` rendait nil et la traction
disparaissait — alors que la base la publiait. Deux configurations, atteignables avec **une seule
vie** (donc sans rapport avec le decoupage) : la vie entierement comprise entre le tir et
l'accroche, et un couple tir/accroche anterieur a la vie de moins de `grapplePullCapUS` (2,5 s).
Ma condition « sur un film ou chaque slot n'a qu'une piste, sortie identique octet pour octet »
etait donc **fausse en general** ; elle ne tenait que sur les deux films ou elle a ete mesuree.

**Correctif** : a defaut de fenetre couvrante, la traction se rattache a la vie la **plus proche**
de l'accroche (`lifeNearest`) et reste bornee exactement comme avant — clamp aux frames de la vie,
refus si la fenetre est vide. Ce calque ne publie donc jamais moins qu'avant.

**Preuve** : `TestBuildGrappleLines_AucuneFenetreCouvranteRattacheALaVieLaPlusProche`, les deux
scenarios du verdict en sous-tests. Rouge avant correctif (« 0 traction(s), attendu 1 » sur les
deux), vert apres, aux memes intervalles que la base (`[102..103]` et `[110..120]`).

### C2 — `camoLives` / `overshieldLives` comptaient des SLOTS

Defaut symetrique de celui que j'avais corrige pour `pullLives`, dans le fichier meme que je
modifiais : `equipmentCoverage` indexait par `e.Slot` sous un commentaire parlant de vies. Le cas
n'etait pas atteignable avant ce chantier (seuls les episodes de la derniere vie survivaient) —
c'est mon correctif qui le rend possible, a lui de compter juste.

**Correctif** : la cle devient (slot, image de debut de la vie qui recouvre l'episode), via
`windowFor`. **Preuve** : `TestCouvertureDesEpisodesCompteDesViesPasDesSlots` (rouge : « 2
episodes / 1 vies »), et le test livre precedemment assure desormais aussi `camoLives = 2`.

**Sur le chiffre du journal** : la revue lisait « 15 camo sur 9 vies » (section 3.4) comme un
compte de slots. Verifie en recomptant les vies porteuses sur l'artefact cuit de `084a804d` :
**9 vies ET 9 slots** — aucun slot n'y porte deux episodes de camouflage sur deux vies
differentes. Le chiffre etait donc juste, mais par coincidence : c'est le COMPTEUR qui mesurait
des slots.

### C3 — la scission n'etait pas un deplacement pur

`git diff -M` ne detecte aucun renommage, et `closures_respawn.go:75` porte
`rep.noteLife(slot, vies[0])` — la ligne qui fait fonctionner le correctif pour la fermeture B.
L'en-tete du fichier annoncait « sans changement de logique » : un relecteur qui la croyait
sautait le coeur du changement, ce qui est l'anti-pattern « doc inversee » que ce meme journal
reproche a `48cf4905d`.

**Correctif** : l'en-tete enumere desormais ce qui a change (`claims` en indices de vie, lecture
de `lives[i].from`, slot relu par `lives[vies[0]].slot`, ajout de `noteLife`, signature de
`sortedVictims`) et ce qui est identique a l'octet pres (`respawnWindow`, `victimsInWindow`,
`overlapsNamedLife`, `containsXUID`) — liste etablie par diff fonction par fonction contre
`9e73368e8`. **La section 4.5 ci-dessus doit se lire avec cette correction** : la scission
s'accompagne du correctif de la fermeture B ; seule la logique de DECISION est inchangee.

### C4 — le commentaire du calque des episodes annoncait un gain que le correctif ne produit pas

`equipment_episodes.go` citait `13d92593` parmi les films dont l'episode revient, quand la
chronique de `document.go` et la mesure disent l'inverse. **Correctif** : le commentaire dit
maintenant que cet episode ne revient pas et **pourquoi** (duree nulle, ancre sur le point
aberrant qui faussait les bornes de scene), avec la cuisson de controle a l'appui.

### C5 — un quatrieme consommateur, qui n'a jamais JETE mais qui ATTRIBUE a tort

`usage_summary.go` resolvait le proprietaire d'un geste par un agregat **slot -> joueur, dernier
gagnant** : sur un slot recycle, les tractions, episodes et poses du PREMIER occupant etaient
credites au SECOND. La faiblesse est anterieure a ce chantier, mais il l'**elargit** — les gestes
des vies non dernieres n'existaient pas avant pour etre mal attribues. Le cas existe au parc :
`879a4dba` porte `coverage.bridge.slotCollisions = 1` (slot 610).

**Correctif** : `usageOwners.at(slot, frame)` resout par la VIE qui couvre l'instant du geste,
avec repli sur le dernier occupant quand aucune ne le couvre. Les trois sites d'appel ont deja
leur instant (`GrappleLine.T0`, `EquipmentEpisode.T0`, `EquipmentPlacement.T0`) : il n'y avait
rien a deviner. **`UsageSummaryRev` passe de `us1` a `us2`** — la doc de cette constante l'exige
(« changer une regle d'attribution ici DOIT incrementer cette revision »), et c'est ce qui fera
re-resumer les matchs deja projetes.

**Preuve** : `TestUsageSummary_SlotRecycleCrediteChaqueOccupant` — un slot a deux identites, un
geste de chaque famille sur chaque vie. Rouge avant (« 111 : 0/0/0 » et « 222 : 2/2/2 »), vert
apres (1/1/1 chacun). Le test d'attribution existant reste vert, et son commentaire — qui
justifiait le « dernier gagnant » comme une dette web assumee — est corrige.

**Consequence a assumer** : le web (`rosterLogic.ts`, `indexBySlot`) garde sa regle par slot, donc
Go et web peuvent differer sur un slot a deux identites. C'est l'ecart JUSTE contre l'ecart
copie ; il est inscrit au registre des reports.

### Recensement corrige

« Aucun quatrieme calque » etait vrai des calques qui JETTENT de la donnee — la revue l'a
reverifie sur 31 consommateurs de `[]Track` et 20 de `[]lifeSpan`. Il etait faux comme enonce
absolu : **deux compteurs indexaient par slot une grandeur devenue par vie**
(`equipmentCoverage`, C2) et **un consommateur ATTRIBUE par slot** (`usageSlotOwners`, C5).
Les trois sont corriges ici.

### Gates apres corrections (tous verts, 2026-09-06)

```
go test -count=1 ./internal/analysis/replay/... ./internal/replaybuild/... ./internal/archlint/...  ok (x4)
go test -tags=integration -p 1 -count=1 ./internal/api/wire/...                                     ok 20,4 s
go build ./...                                                                                      exit 0
golangci-lint run --new-from-merge-base=origin/main ./...                                           0 issues
goldens inconditionnels (mini-bobine familles, equivalence mini-film, assemblage + 3 satellites,
  incertitude du golden, distribution des durees de camo)                                           PASS
```

**Cuisson de controle** : `879a4dba` re-cuit seul (pic 0,23 Gio, faits lus, 23 tractions pour
23 accroches).

CE QUE J'AI COMPARE, ET CE QUE CELA NE PROUVAIT PAS. Ma premiere comparaison opposait cette
cuisson a `apres/879a4dba_fix.json`, cuit AVANT le bump de schema : son unique ecart
(`schemaVersion` 40 -> 41) etait donc attendu par construction et ne disait rien des cinq
corrections. La comparaison juste est celle contre l'artefact cuit au commit PRECEDENT
(`13c0336b6^` = `79bf2e6d2`), deja au schema 41 ; elle a ete faite par la seconde ronde REG-R2 et
rend **zero ecart** : md5 egaux (`ec083886d1bc98d7947fcc6d67ed53d4`) une fois neutralise le seul
champ qui differe, `matchId` — la cuisson de reference avait reçu l'UUID complet en argument, la
mienne l'identifiant court, artefact d'INVOCATION et non de code (`cmp` : premier ecart au
caractere 253 851, dans ce champ). `coverage.grapple` et `coverage.equipment` identiques des deux
cotes.

Les cinq corrections ne changent donc la sortie d'aucune cuisson de ce film : C1 ne s'y declenche
pas (les 23 tractions ont toutes un `t0` a l'interieur d'une fenetre publiee), C2 y compte le meme
nombre (9 vies = 9 slots), et C5 ne touche pas l'artefact mais le resume d'usage.

---

## Corrections apres la seconde ronde REG-R2 (2026-09-06)

La seconde ronde ferme C1, C2, C3 et C4, et laisse **C5 PARTIEL** : la correction que j'avais
livree pour lui ne couvrait pas le canal des POSES. Quatre defauts nouveaux, tous a l'interieur
du perimetre de C1 et C5, dont **un seul touche la donnee**. Traites ci-dessous.

### N-3 — C5 restait OUVERT pour les poses, sur le chemin MAJORITAIRE du canal

**Le constat.** Un objet lache a la mort porte `t0 = finVie + 1` — le poseur n'occupe deja plus le
slot —, et rien cote Go ne borne `EquipmentPlacement.T0` a une fenetre publiee. Mon `at` retombait
donc sur le repli « dernier occupant du match » et creditait le lacher au joueur SUIVANT sur un
slot repris : **exactement la regle que mon commit declarait avoir supprimee**. Et ce n'est pas un
residu, contrairement a ce que disait mon commentaire : la revue a mesure la part des poses dont
le `T0` tombe hors de toute fenetre publiee — **153/351 (44 %), 443/466 (95 %), 34/105 (32 %)** sur
trois films.

**Correctif** : `usageOwners.atOrJustBefore`, jumeau d'`ownerAtFrameOrLast` (`rosterLogic.ts`) —
la vie qui couvre l'instant, sinon celle qui vient de s'y achever. Seules les poses l'empruntent ;
les tractions et les episodes gardent `at`, dont la mesure montre que le repli n'y joue jamais
(23/23 et 31/31 tractions, 7/7 et 15/15 episodes ont leur `T0` dans une fenetre). Quand AUCUNE vie
ne precede — une pose datee avant la premiere vie publiee du slot —, c'est la premiere vie qui
repond : sur un slot mono-identite c'est le meme joueur qu'avant, sur un slot recycle c'est son
premier occupant, jamais le dernier ; ce canal ne perd donc aucune ligne au passage.

**Preuve** : `TestUsageSummary_LacherALaMortRevientAuLacheur` — slot a deux identites, un objet
lache a `finVie + 1` de chacune. Rouge avant (« 111 : 0 lacher(s), attendu 1 » et « 222 : 2 »),
vert apres (1 chacun). Le commentaire qui presentait le repli comme un residu est corrige, et
l'en-tete du fichier distingue desormais les deux resolveurs.

### N-4 — un joueur dont l'unique vie est sur un slot repris perdait TOUS ses lancers

`usageFilmIndexOwners` construisait sa garde « au moins une vie publiee » depuis `dernier` — le
seul dernier occupant de chaque slot. Un joueur dont toutes les vies sont sur des slots repris
ensuite n'y figurait pas et ne recevait aucune grenade, alors que l'en-tete de la fonction lui en
promet. Defaut **anterieur** au chantier (byte-identique a l'ancienne map), mais ma correction
avait touche cette ligne en conservant le vieux critere, trois lignes sous son jumeau corrige.

**Correctif** : la garde se lit sur `parVie`, c'est-a-dire sur toutes les vies.
**Preuve** : `TestUsageSummary_UneVieSurUnSlotReprisNePerdPasSesLancers`, rouge avant
(« 111 : 0 lancer(s), attendu 2 »), vert apres.

### N-1 et N-2 — deux commentaires qui affirmaient plus que ce qui est garanti

- **N-1** : « ce calque ne doit JAMAIS publier moins qu'avant » est faux, et la revue l'a mesure :
  une accroche tombant sur la DERNIERE image d'une vie a bien une vie couvrante, dont la fenetre
  est vide (`t1 <= t0`), et la traction est refusee la ou l'ancien code la posait sur la vie
  SUIVANTE (base 1, HEAD 0). **Le comportement du HEAD est le bon** — dater sur sa vie suivante la
  traction d'un joueur qui vient de mourir serait faux —, c'est l'invariant absolu qui etait trop
  large. Le commentaire dit maintenant ce qui est garanti : quand aucune vie ne couvre l'accroche,
  la traction est publiee comme avant ; et il nomme l'exception.
- **N-2** : « cela ne depend d'aucun ordre d'iteration » est faux pour `lifeNearest` — deux vies a
  egale distance de l'accroche (accroche au milieu du trou) donnent un resultat qui s'inverse avec
  l'ordre. Aucun impact donnee : l'ordre de production EST chronologique (`decimateTracks` emet
  les vies closes puis la courante, rien ne retrie `doc.Tracks` avant ce calque), donc « la
  premiere » vaut « la precedente », ce que la regle veut dire. Le commentaire enonce desormais la
  dependance a un ordre GARANTI plutot qu'une independance fausse, et signale qu'un futur tri de
  `doc.Tracks` inverserait ce cas.

Aucune ligne de code touchee pour ces deux points.

### N-5 — la condition de reprise du registre decrivait du travail deja fait

`buildSlotOwnership` / `ownerAtFrame` / `ownerAtFrameOrLast` existent deja
(`lib/replay/rosterLogic.ts:184-230`) : il ne reste pas a « porter `at` cote TS » mais a basculer
**deux appelants** encore sur `indexBySlot` (`equipmentUsageLogic.ts:258`,
`equipmentKillBadges.ts:56`). L'entree du registre est corrigee en ce sens, avec la precision que
les poses prennent `ownerAtFrameOrLast` et le reste `ownerAtFrame`.

### Ce qui NE change pas

`UsageSummaryRev` reste **`us2`** : une seule regle d'attribution a change dans ce chantier, et la
montee la couvre entierement — N-3 et N-4 la precisent, ils n'en ouvrent pas une seconde.

### Gates apres la seconde ronde (tous verts, 2026-09-06)

```
go test -count=1 ./internal/analysis/replay/... ./internal/replaybuild/... ./internal/archlint/...  ok (x4)
go test -tags=integration -p 1 -count=1 ./internal/api/wire/... ./internal/sync/replayartifacts/... ok
go build ./...                                                                                      exit 0
golangci-lint run --new-from-merge-base=origin/main ./...                                           0 issues
```

Aucune cuisson n'etait requise : `BuildUsageSummary` ne participe pas a l'artefact de rejeu (il le
PROJETTE apres coup), et les deux commentaires de `grapple_lines.go` ne touchent aucune ligne de
code. La cuisson de controle de la ronde precedente reste la piece.
