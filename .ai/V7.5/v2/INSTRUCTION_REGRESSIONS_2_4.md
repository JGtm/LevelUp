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
