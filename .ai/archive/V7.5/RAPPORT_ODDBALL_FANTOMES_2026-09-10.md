# Portages Oddball « fantomes » — mesure, cause, correctif (lot 6.2, 2026-09-10)

> Worktree `LevelUp-wt-oddball-fantomes`, branche `wt/oddball-fantomes` (depuis `feat/v75`
> a `d00799a1e`). Aucune base DuckDB ouverte : un serveur de developpement les tient
> (`netstat` : `127.0.0.1:8000` LISTENING au demarrage du lot). Aucune recuisson du parc.

## 0. Verdict en trois lignes

1. **Le defaut est FERME au HEAD, et la fermeture est ATTRIBUEE.** Sur les **quatre** films
   Oddball du parc (94 portages), **0 portage tombe hors de la presence bipede de son
   porteur**. Le defaut EXISTAIT bien au schema 48 — il est reproduit ici a l'identique
   (6 portages sur 36 pour `d9781168`, memes bornes, memes vies) — et il a ete ferme par
   `b6b0ee4c5` (« E2/I2-I3-I4 : le lien direct corps -> joueur entre au registre »), qui
   nomme les vies bipedes qui manquaient. **La cause n'etait pas l'attribution xuid des
   portages** : celle-ci est IDENTIQUE entre le schema 48 et le HEAD (36 portages, memes
   bornes) ; ce qui manquait etait le denominateur, c'est-a-dire les VIES.
2. **La regle demandee par le report existe deja en production** (`carrierPresence.gate`,
   `skull_carries.go`), avec son compteur publie (`coverage.skullCarries.carrierAbsent`) et
   son `slog`. Elle rend `carrierAbsent = 0` sur les quatre films : il n'y a rien a rejeter.
   **La reimplementer n'aurait rien corrige et aurait regresse** — les seuls portages qu'un
   rejet ou un rognage plus strict toucherait sont les 4 portages « debordants » du §4, que
   l'oracle API prouve REELS et deja SOUS-mesures.
3. **Le correctif livre est ailleurs, et il est mesure** : le pont d'identite du crane
   n'etait pas COMPLETE comme celui du drapeau — 6 trains de tics sur 4 films partaient en
   `noBridge`, dont les 62,3 s du plus gros porteur de `43716616` et 25,8 s sur `c88ec007`.
   `SkullInput` recoit desormais le pont complet de la couche d'assemblage, exactement comme
   `FlagInput` depuis le schema 42. Resultat : **94 -> 98 portages, `noBridge` 6 -> 2**,
   +41,0 s de portage publie, **78,9 % -> 82,2 % de l'oracle API**, aucun joueur ne depasse
   son oracle, et **toujours 0 portage hors presence**.

**Bump de schema : NON.** Aucune forme ne change (ni champ ajoute, ni champ retire) — seules
les VALEURS bougent, sur les seuls films Oddball. Rien a mettre sous `omitempty`.

---

## 1. Methode — et pourquoi elle n'ouvre aucune base

### 1.1 Le parc d'artefacts ne contient AUCUN Oddball

L'instruction du lot disait « identifie les films Oddball par la presence de `skullCarries`
dans les 64 artefacts de `data/cache/replays/halo_infinite/` ». **C'est impossible : aucun
des 64 artefacts ne porte `skullCarries`** (verification : `grep -l skullCarries *.json` rend
zero fichier ; les 64 sont au schema 51). Ni `d9781168` ni `51ebbc0f` n'ont d'artefact au
parc. Le parc Oddball vit dans le **cache de films** (`data/cache/film_chunks/`, 1 127 films),
pas dans le parc d'artefacts.

### 1.2 Les quatre films Oddball, identifies sans base

Le recensement des modes passe normalement par la base partagee. Il existe une copie
**versionnee** de ce recensement : `.ai/V7.5/replay2d/registre_film/oracle_lotA_bis.tsv`
(467 matchs, export du registre commit dans le depot). Croisee avec le cache de films :

| film | mode | carte | duree | chunks au cache |
|---|---|---|---|---|
| `d9781168` | Oddball:Arena | Dredge | 716 s | 39 |
| `51ebbc0f` | Oddball:Arena | Banished Narrows | 500 s | 28 |
| `c88ec007` | Oddball:Arena | Live Fire | 666 s | 36 |
| `43716616` | Oddball:Arena | Smallhalla | 336 s | 20 |
| `24dbb67d` | Ranked:Oddball | Recharge - Ranked | 519 s | **0 (purge)** |
| `60ae07c4` | Ranked:Oddball | Live Fire - Ranked | 810 s | **0 (purge)** |
| `92f18088` | Ranked:Oddball | Lattice - Ranked | 610 s | **0 (purge)** |

**Le corpus mesurable est donc de QUATRE films** — deux de plus que la mesure du 2026-09-08
(`.ai/V7.5/v2/RESTES_R4_R5_2026-09-08.md` §5.4, qui n'en connaissait que deux). Les trois
`Ranked:Oddball` ne sont plus au cache : non mesurables, non recuisables, sans consequence.

### 1.3 Cuisson hors ligne, faits de match versionnes

`replay-build` n'ouvre aucune base : il consomme un `<short8>.facts.json`. Deux sources, aucune
n'est une base :

- `d9781168` : le fichier **deja commis** `internal/analysis/replay/testdata/equivalence/d9781168.facts.json` ;
- les trois autres : engendres par `awk` depuis `oracle_lotA_bis.tsv` +
  `oracle_lotA_bis_participants.tsv` (versionnes). **Controle du generateur** : rejoue sur
  `d9781168`, il rend exactement les valeurs du fichier commis (8 joueurs, memes triplets,
  memes camps, scores 196/191).

Racine de travail : une copie physique dans le scratchpad (`config/titles/halo_infinite`,
`data/titles/halo_infinite/reference` depuis le worktree ; chunks + manifestes depuis le parc),
comme `cmd/replay-corpus-gate/staging.go`. **Le parc n'est lu qu'en lecture.**

```bash
CGO_ENABLED=0 LEVELUP_REPO_ROOT=<scratch>/wr go run ./cmd/replay-build \
  --map Dredge --title halo_infinite --facts <...>/d9781168.facts.json \
  d9781168-5fd6-4b00-a862-56e3a0a1f956
```

### 1.4 Le critere de mesure, et le seul qui compte pour le rendu

Un portage est **hors presence** quand aucune vie bipede NOMMEE de son porteur ne recouvre,
meme partiellement, son intervalle `[t0, t1]` — c'est la formulation du report du 2026-08-28.
Un portage est **debordant** quand des vies le recouvrent mais ne le couvrent pas entierement.

Le critere de RENDU est plus precis, et il est lu dans le code du client :
`posOfPlayerAt` (`apps/web/src/features/match-replay/model/livesPosition.ts`) rend une position
si l'image tombe dans la fenetre `[startFrame, endFrame]` d'une vie (avec INTERPOLATION entre
points — un point exact n'est pas requis), ou dans les 15 images qui suivent la fin d'une vie
(`KILLPOS_WINDOW_MS` = 1 500 ms / pas de 100 ms). Sans position, `skullCarrierLayer` ne
dessine rien, et `skullPresenceAt` donne la precedence a `carried` : **le crane devient
invisible**. C'est la grandeur « images muettes » ci-dessous.

---

## 2. Phase A — la mesure

### 2.1 Au HEAD (schema 51, avant le correctif de ce lot)

| film | portages | **hors presence** | debordants | images portees | images muettes |
|---|---|---|---|---|---|
| `d9781168` | 36 | **0** | 3 | 3 312 | 613 (18,5 %) |
| `51ebbc0f` | 19 | **0** | 1 | 2 380 | 81 (3,4 %) |
| `43716616` | 12 | **0** | 0 | 1 243 | 0 |
| `c88ec007` | 27 | **0** | 0 | 3 018 | 0 |
| **total** | **94** | **0 (0,0 %)** | **4 (4,3 %)** | **9 953** | **694 (7,0 %)** |

Couvertures publiees (`coverage.skullCarries`), au HEAD avant correctif :
`carrierAbsent = 0` sur les quatre films ; `noBridge` = 1 / 0 / 2 / 3.

### 2.2 Au schema 48 — le defaut est REPRODUIT a l'identique

Cuisson du meme film, meme fichier de faits, avec le code de `1947a83f8^` = `b9e1064ad`
(dernier etat au schema 48, la revision de la mesure du 2026-09-08) :

```
d9781168 schema=48 vies=174 dont sans xuid=19 | portages=36 horsPresence=6 debordants=4
         images portees=3313 muettes=1106 (33,4 %)
```

**6 sur 36 = 16,7 %** : la valeur exacte du registre. Les deux temoins de l'utilisateur du
2026-08-28 sortent au bit pres — `2535435655459376` porte `[326..426]` alors que sa premiere
vie commence a **868**, et `[6944..6974]` alors que sa derniere s'acheve a **6505** ; ses
14 vies sont exactement celles listees au §5.4 de `RESTES_R4_R5_2026-09-08.md`. **Le harnais
hors ligne de ce lot reproduit donc la mesure de reference.**

### 2.3 La cause exacte — ni pont aplati, ni horloge, ni mauvais xuid

Les trois hypotheses de l'instruction sont **ecartees sur pieces**, par une seule observation :
entre le schema 48 et le HEAD, **les 36 portages sont les MEMES** — meme nombre, memes xuid,
memes bornes `[t0, t1]`, meme `noBridge = 1`, meme `carrierAbsent = 0`. Ce qui a change, ce
sont **les vies** :

| | schema 48 (`b9e1064ad`) | HEAD |
|---|---|---|
| vies publiees | 174 | 174 |
| dont sans xuid | **19** | **0** |
| vies de `2535435655459376` | 14, la 1re `[868..984]`, la derniere `[6421..6505]` | 17, la 1re **`[0..984]`**, la derniere **`[6605..6977]`** |

Le portage `[326..426]` n'a jamais ete mal attribue : **la vie qui le porte n'etait pas
publiee sous le nom de son porteur.** Le calque n'avait donc pas de position a lire — le
symptome vu par l'utilisateur (« icone de crane absente au premier portage ») — et la mesure
de couverture, qui compare des portages a des vies, comptait le portage comme fantome.

### 2.4 Attribution de la fermeture (bissection, meme film, memes faits)

| revision | schema | vies sans xuid | hors presence | debordants | images muettes |
|---|---|---|---|---|---|
| `b9e1064ad` (`1947a83f8^`) | 48 | 19 | **6** | 4 | 1 106 (33,4 %) |
| `fb953ee94` (P2-bis, exclusion temporelle) | 50 | 15 | **3** | 5 | 876 (26,4 %) |
| `b6b0ee4c5` (E2/I2-I3-I4, lien direct corps -> joueur) | 50 | **0** | **0** | 3 | 613 (18,5 %) |
| `HEAD` (`d00799a1e`) | 51 | 0 | 0 | 3 | 613 (18,5 %) |

**Le portage fantome est un effet de bord ferme par le registre d'identite** (`b6b0ee4c5`),
en deux temps : l'exclusion temporelle du roster en retire la moitie, le lien direct corps ->
joueur le reste. Aucune ligne de `skull_carries.go` n'a ete touchee entre-temps.

### 2.5 Ce qui reste apres la fermeture : 4 portages qui DEBORDENT un trou de replication

| film | porteur | portage | images sans position |
|---|---|---|---|
| `d9781168` | 2533274974091007 | `[492..1023]` | 406 / 532 |
| `d9781168` | 2535455182553276 | `[4480..4771]` | 47 / 292 |
| `d9781168` | 2533274974091007 | `[5796..6187]` | 160 / 392 |
| `51ebbc0f` | 2533274920533076 | `[3035..3455]` | 81 / 421 |

Exemple : `2533274974091007` a les vies `[227..534] [920..922] [974..1033]` ; son portage
`[492..1023]` traverse un trou de 385 images (38,5 s) ou son bipede n'est pas replique.
`unionOverlap` (regle de l'UNION, schema 45) conserve donc les bornes, et le crane disparait
pendant le trou.

**Ces quatre portages sont VRAIS, et il ne faut surtout pas les rejeter.** Preuve
independante, l'oracle API (`match_objective_stats`, `D10_oracle_objective_stats.json`,
versionne) : pour ce joueur, `longest_time_as_skull_carrier_seconds` = **54,9 s** contre
**53,1 s** reconstruits. La reconstruction est **plus courte** que la verite, jamais plus
longue — sur les quatre films elle rend **78,9 % du temps de portage de l'oracle**, aucun
joueur au-dessus du sien. Un rejet ou un rognage « plus strict » retirerait du temps de
portage REEL et **eloignerait** le calque de la feuille de match : c'est exactement la
regression mesuree le 2026-09-06 (60,1 s / 147,4 s publies contre 191 s / 196 s a la feuille),
et l'en-tete de `carrierPresence` la documente deja.

---

## 3. Phase B — le correctif livre

### 3.1 Ce qui n'a PAS ete fait, et pourquoi

L'instruction prescrivait « exiger que l'intervalle porte tombe DANS la presence bipede du
xuid, comme le drapeau ; sinon marquer le portage non ponte ou le rejeter, et le COMPTER ».

**Cette regle est deja en production** — `carrierPresence.gate` (`skull_carries.go`), pose le
2026-08-30, durci le 2026-09-06 (l'ignorance passe avant le rognage) et le 2026-09-07 (une
identite deduite ajoute une presence, jamais n'en retire une). Elle compte ses rejets
(`SkullCarriesCoverage.CarrierAbsent`, publie au contrat et journalise). Elle rend **0** sur
les quatre films. **Il n'y a donc rien a ajouter, et rien a durcir sans casser le §2.5.**

### 3.2 Ce qui a ete fait : le pont d'identite du crane, comme celui du drapeau

Le report du registre le prevoyait deja (ligne « Les calques VIP et CRANE subissent le MEME
plafond que le drapeau avant le schema 42 », 2026-09-06) : le calque du crane resolvait son
pont slot -> xuid par les **seuls instants de mort**, qui exigent TROIS progressions
coincidentes (`deathInstantMin` = 3). Un joueur qui meurt moins de trois fois dans la manche
lui echappe **par construction** — et en Oddball c'est souvent le porteur, que son equipe
protege. Son train de tics partait en `noBridge`, sans aucun intervalle publie.

Trois modifications, aucun changement de forme :

| fichier | changement |
|---|---|
| `internal/analysis/replay/skull_carries.go` | `SkullInput` gagne `Identity objectiveevents.RoundIdentity` ; `skullIdentityOf(in, opt)` prefere le pont de l'appelant, sinon resout localement (copie exacte de `flagIdentityOf`) |
| `internal/replaybuild/matchfacts.go` | `skullInput(recs, isSkull, pont)` pose `pont.identite()` — le MEME resolveur paresseux que les actions d'objectif et le drapeau, donc **zero resolution supplementaire** (il en economise une) |
| `internal/analysis/replay/testdata/equivalence/d9781168.tsv` | digest de l'etape `skull` re-fige (cf. §5) |

La garde de mode ne bouge pas : hors Oddball, `skullInput` rend une entree vide **et ne
demande pas le pont** (un test l'exige : `pont.resolu` doit rester faux). Precision, pour ne pas
sur-vendre la garde : dans `readFilmStats`, `statborgIdentity` reveille de toute facon le
resolveur, tous modes confondus — le cablage de ce lot **ne coute donc aucune resolution
supplementaire, il en economise une** (celle que `attachSkullCarries` refaisait pour son compte).
La garde vaut pour la fonction elle-meme, le jour ou un autre appelant assemblera une entree de
crane sans avoir cette raison-la.

### 3.3 Le gain, mesure film par film contre l'oracle API

`noBridge` **6 -> 2**, portages **94 -> 98** :

| film | portages avant -> apres | `noBridge` avant -> apres | joueur recupere | avant | apres | oracle |
|---|---|---|---|---|---|---|
| `d9781168` | 36 -> **37** | 1 -> **0** | 2533274977810162 | 25,0 s | **48,0 s** | 51,1 s |
| `51ebbc0f` | 19 -> 19 | 0 -> 0 | — | — | — | — |
| `43716616` | 12 -> 12 | 2 -> **2** | *(non recupere, cf. §6)* | 0,0 s | 0,0 s | 62,3 s |
| `c88ec007` | 27 -> **30** | 3 -> **0** | 2535448682342105 | 15,0 s | **33,0 s** | 40,8 s |

Total du parc Oddball : **985,9 s -> 1 026,9 s** publiees pour **1 249,0 s** a l'oracle, soit
**78,9 % -> 82,2 %**. Controles :

- **aucun joueur ne depasse son oracle** apres correctif (le controle est dans l'instrument) ;
- **seuls 4 joueurs sur 33 voient leur valeur bouger**, tous vers leur propre oracle ;
- **0 portage hors presence** avant comme apres ;
- images muettes **694 -> 694** (le correctif n'ajoute aucune image sans position).

### 3.4 Tests — rouges d'abord, mutation prouvee

`internal/analysis/replay/skull_carries_identity_test.go` (4 tests, calques sur
`flag_carries_identity_test.go`) :

- `TestSkullIdentityOfResoutLocalementSansPontFourni` — hors ligne, le paquet resout seul ;
- `TestSkullIdentityOfPrefereLePontDeLAppelant` — le pont fourni nomme le slot a 2 morts ;
- `TestSkullIdentityOfRespecteUnPontMuet` — un pont fourni VIDE est une reponse, pas un
  silence (il ne fait pas retomber sur la resolution locale) ;
- `TestSkullCarriesPontFourniPublieLePortage` — **la mutation** : le meme film rend
  `0 portage / noBridge = 1` avec le pont par morts, et `1 portage / noBridge = 0` avec le
  pont complet. Le premier cas EST l'etat d'avant ce lot.

`internal/replaybuild/skullidentity_test.go` (2 tests) :

- `TestSkullInputPoseLePontCompleteSurUnFilmOddball` — le porteur a 2 morts est nomme, le slot
  deja nomme garde son nom, le slot agrege reste muet ; **mutation** : sans lignes de match, le
  meme appel laisse le slot sans nom (le test ne prouverait rien si le pont par morts savait
  deja le nommer) ;
- `TestSkullInputHorsOddballNeResoutRien` — la garde de mode : hors Oddball, `pont.resolu`
  reste faux.

Les 6 tests ont ete ecrits AVANT le code et ont echoue a la compilation
(`undefined: skullIdentityOf`, `unknown field Identity`) avant de passer.

---

## 4. Artefacts a recuire

**Quatre**, et eux seuls — ce sont les seuls films Oddball du cache, et le calque du crane est
le seul touche :

```
d9781168   43716616   51ebbc0f   c88ec007
```

Aucun n'a d'artefact au parc aujourd'hui (§1.1) : la « recuisson » est en fait une **premiere**
cuisson pour les quatre. Les trois `Ranked:Oddball` (`24dbb67d`, `60ae07c4`, `92f18088`) n'ont
plus de chunks au cache et ne peuvent pas etre cuits. **Aucun film non-Oddball n'est concerne** :
hors Oddball, `skullInput` rend la meme entree vide qu'avant, a l'octet.

Recuisson par le superviseur (le lot n'ouvre aucune base) :
`levelup backfill-replay --one <matchId>`.

---

## 5. Goldens

- **Aucun golden de CI ne change**, et la suite complete des trois paquets le confirme :
  `go test ./internal/analysis/replay/... ./internal/replaybuild/... ./internal/service/replayview/...`
  passe sans `-update`. Les goldens de CI portent sur `000d5950` (mini-bobine Slayer), ou
  `isSkullVariant` est faux : l'entree du calque est vide comme avant.
- **Un digest du corpus LOCAL change** : `testdata/equivalence/d9781168.tsv`, etape `skull`,
  `f154063c…` -> `37f52dc5…` (37 portages au lieu de 36). Regenere par
  `LEVELUP_REPO_ROOT=<wr> go run ./cmd/replay-equiv -films d9781168 -update`, puis **reporte
  ligne a ligne** : le harnais compare 50 etapes et n'en signale qu'UNE seule modifiee par ce
  lot — `skull`.
- **Decouverte, non traitee** : ce meme fichier est PERIME sur trois autres lignes,
  independamment de ce lot — `flag` (`68393539…` mesure `4d5c10eb…`), l'etape
  `bipedCreations` (160 valeurs) **absente du fichier**, et `artifact` (2 586 160 octets fige
  contre 2 663 057 mesures **avant** ce lot). Ces trois ecarts precedent le lot 6.2 : les
  re-figer ici ratifierait en silence des changements que ce lot n'a pas mesures. Ils
  demandent un lot de re-figeage declare du corpus local (§6).

---

## 6. Decouvertes consignees, NON traitees

1. **`43716616` : 62,3 s de portage toujours perdues** — 2 trains restent `noBridge` malgre le
   pont complet. Cause identifiee : les quatre films Oddball sont **multi-manche** (2 ou 3
   manches), donc `CompletedByLines` (garde MONO-MANCHE : le triplet apparie des totaux de
   match) ne s'applique a aucun d'eux. Le gain de ce lot vient entierement de
   `CompletedByElimination`, qui ne resout pas ces deux slots-la. C'est le plus gros porteur du
   match (2533274978052136, 62,3 s a l'oracle) : le calque le montre a 0. Reprise = une voie de
   completion par manche pour un slot dont l'elimination ne tranche pas.
2. **694 images de portage sans position (7,0 % du parc Oddball, 18,5 % de `d9781168`)** — les
   4 portages du §2.5 traversent un trou de replication du bipede. Le portage est vrai, la
   position manque : le crane disparait a l'ecran. **Ce n'est pas un defaut de reconstruction**,
   et le stopgap est cote RENDU, deja ecrit au registre : passer les positions bipedes a
   `skullPresence` et rendre le crane LIBRE quand le porteur n'a pas de position a l'image.
   Lot web, hors perimetre 6.2.
3. **Le pont complet n'est toujours PAS pose sur la couronne VIP** (`vipInput`,
   `internal/replaybuild/matchfacts.go`) — meme plafond, meme correctif d'une ligne. Non traite
   ici : aucun film VIP au parc (recensement du 2026-09-08), donc **rien a mesurer**, et le
   depot interdit de livrer un correctif non mesure. **A la 3e copie du patron
   `xIdentityOf`** (drapeau, crane, puis VIP), la regle n° 6 du depot exige une centralisation
   + garde-rail : c'est a faire DANS le lot qui cablera le VIP, pas avant.
4. **La reconstruction sous-estime systematiquement le portage de 1,5 a 2 s par periode** (un
   train est borne par son PREMIER et son DERNIER tic ; la seconde d'amorce et celle de chute
   ne sont pas comptees). Mesure : 82,2 % de l'oracle apres correctif, avec un ecart par joueur
   toujours negatif. Corrigeable par une demi-fenetre de tic aux deux bornes, mais c'est un
   changement de DEFINITION du portage : decision produit, pas un correctif.
5. **Trois films Oddball du registre n'ont plus de chunks au cache** (`24dbb67d`, `60ae07c4`,
   `92f18088`) alors que `60ae07c4` est nomme au corpus d'equivalence
   (`testdata/equivalence/CORPUS.txt`, avec son `.facts.json` et son `.tsv` commis) : le
   harnais local le rend « illisible » sans que rien ne l'annonce. Idem `1c4c63c2` et
   `a349fea8`. Purge de cache, pas une regression.
6. **`config/replay_corpus.toml`, famille `oddball`** : la raison du temoin dit encore
   « `d9781168` perd 6 portages de crane (36 -> 30, schema 23), residu non instruit ». Le
   residu de 30 n'existe plus depuis le 2026-09-08 et les 6 fantomes sont fermes depuis
   `b6b0ee4c5` : le texte est a reecrire au prochain passage sur ce manifeste (doc perimee,
   pas doc inversee — le temoin reste pertinent).

---

## 7. Gates joues

| gate | commande | verdict |
|---|---|---|
| build | `CGO_ENABLED=1 go build ./...` | **OK** |
| vet | `go vet ./internal/analysis/replay/... ./internal/replaybuild/... ./internal/service/replayview/...` | **OK** |
| tests | `go test` sur les 3 paquets | **OK** (`replay` 8,3 s, `replaybuild` 0,7 s, `replayview` 0,3 s) |
| lint | `golangci-lint run --new-from-merge-base=feat/v75 ./...` | **0 issue** (code de sortie 0) |
| corpus | `replay-corpus-gate --reference=base --base feat/v75` | **0 perte**, couverture PARTIELLE — cf. §8 |

Note sur le lint : l'unique ligne de sortie est un avertissement PREEXISTANT du filtre `nolint`
(« Found unknown linters in //nolint directives »), qui vient de directives d'autres lots — ce
diff n'ajoute aucun `//nolint`.

### 8. Le gate corpus : ce qui a pu etre joue, et ce qui ne l'a pas pu

`cmd/replay-corpus-gate` exporte les faits de chaque temoin en invoquant
`levelup replay-facts-export`, qui **ouvre la base partagee du titre**. Le serveur de
developpement la tient en ecriture (`127.0.0.1:8000` LISTENING) : une ouverture RO d'un second
processus sur le meme fichier est interdite par le modele mono-writer (ADR 0013 / 0016), et
l'instruction du lot l'interdit explicitement.

Le gate a donc ete joue avec sa propre echappatoire documentee : un `--parc-root` qui ne porte
que les **films** (aucune base a ouvrir — l'export echoue, `slog.Warn`, sans faire echouer les
autres temoins), les faits **pre-poses** dans `<work-root>/facts` pour les temoins dont une
source versionnee existe, et `--allow-missing`.

```bash
CGO_ENABLED=0 go run ./cmd/replay-corpus-gate --reference=base --base feat/v75 \
  --parc-root <scratch>/parc-films   # films SEULS : aucune base sous data/titles/
  --source-root <worktree> --work-root <scratch>/gate --allow-missing
```

### 8.1 Verdict, ligne par ligne (code de sortie **0**)

| temoin | famille | base `feat/v75` | HEAD | gains | **pertes** | duree | statut |
|---|---|---|---|---|---|---|---|
| `bcb6d393` | ctf_mono_manche | — | — | — | — | — | **ABSENT** (faits non exportes) |
| `fb1a1a72` | ctf_multi_manche | — | — | — | — | — | **ABSENT** (faits non exportes) |
| `d9781168` | **oddball** | 51 | 51 | **11** | **0** | 30,2 s | **ok** |
| `c75f33b8` | assaut_bombe | — | — | — | — | — | **ABSENT** (faits non exportes) |
| `bf15f7ab` | slayer | — | — | — | — | — | **ABSENT** (faits non exportes) |
| `51ebbc0f` | **deux_manches** (Oddball) | 51 | 51 | 0 | **0** | 19,8 s | **ok** |
| `084a804d` | vehicules | 51 | 51 | 0 | **0** | 2 min 31 | **ok** |

Lecture :

- **`d9781168` : 11 gains, 0 perte.** C'est le portage supplementaire (36 -> 37) et les
  grandeurs qui en decoulent sur les axes de `replay-diff` (comptes et sommes de durees du
  calque `skullCarries`). Le signal du gate etant BINAIRE sur les pertes, **0 perte = vert**.
- **`51ebbc0f` : 0 gain, 0 perte** — le second Oddball ne bouge pas (son pont par morts nommait
  deja tous ses slots, `noBridge = 0` avant comme apres). Le correctif ne touche donc pas ce
  qui allait bien.
- **`084a804d` : 0 gain, 0 perte** — le temoin NON-Oddball (BTB Heavies:CTF, 57 chunks, le plus
  dense du parc) est **identique** : preuve qu'aucun effet collateral ne sort du mode Oddball.
- Les deux schemas sont egaux partout (51 = 51) : **aucun bump**, confirme par l'outil.

Quatre des sept temoins (`bcb6d393`, `fb1a1a72`, `c75f33b8`, `bf15f7ab`) n'ont **aucune source
de faits hors base** : ni fichier commis dans `testdata/equivalence/`, ni ligne dans
`oracle_lotA_bis.tsv`. Ils sortent **ABSENT** — pas en echec. Les trois temoins couverts
comprennent **les deux seuls Oddball du manifeste**, c'est-a-dire toute la population que ce
diff peut modifier, plus un controle hors mode. **Le gate corpus COMPLET reste a jouer par le
superviseur, serveur arrete** — c'est le seul point de ce lot qui ne peut pas etre clos ici.
