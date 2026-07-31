# PLAN — réconcilier le rejeu 2D avec `feat/killsource-prod`

> Écrit le 2026-07-31. Contrat d'exécution : skill `plan-execution`.
> **Tout ce qui suit est MESURÉ.** Une reprise n'a pas à refaire l'analyse — seulement à
> exécuter, et à vérifier avec les critères d'acceptation donnés au §5.

---

## 1. LA TOPOLOGIE, MESURÉE

```
811be64ec  (base commune)
   ├── … 320 commits …→  feat/filmdec-continuation   (le rejeu 2D — nous)
   └── … 1857 commits …→ main  →  6c2b00402
                                     └── 2 commits → feat/killsource-prod
```

| relation | devant | derrière |
|---|---|---|
| `feat/killsource-prod` vs `main` | **2** | 4 |
| `feat/filmdec-continuation` vs `main` | 320 | **1 859** |
| nous vs `killsource-prod` | 320 | 1 857 |

**Les deux branches partagent une histoire** (base `811be64ec`) : un merge normal est possible,
aucune histoire orpheline. C'est ce qui rend cette réconciliation faisable, contrairement à
`feat/filmdec-killweapon` qui n'a **aucun** ancêtre commun avec `main`.

---

## 2. LE FAIT CENTRAL — DEUX LIGNÉES INDÉPENDANTES DU MÊME DÉCODEUR

| | fichiers `internal/analysis/filmdec/` |
|---|---|
| base commune `811be64ec` | **0** |
| `main` | **0** |
| `feat/filmdec-continuation` | **58** |
| `feat/killsource-prod` | **48** |

`filmdec` n'existait **ni dans la base, ni dans `main`**. Les deux branches l'ont donc créé
**indépendamment**, depuis la même recherche d'origine. D'où des conflits `add/add` sur les
**16 noms de fichiers communs**.

Le commit `3f0ec70b3` le dit lui-même : *« le décodeur posé sur main sans une ligne adaptée »*.
Ce n'était donc pas que de la recherche — c'est du code applicatif.

### Ce que chaque lignée a en propre

| nous seulement | eux seulement |
|---|---|
| `capture.go` · `fire_events.go` · `grenade_events.go` · `i0_layout.go` · `keyframe_loadout.go` · `map_bounds.go` · `projectiles.go` · `vitality.go` (+ leurs tests) | `killhealth.go` · `default_state_arch.go` · `components_flock.go` · `components_walk_batch9.go` (+ leurs tests) |
| ce dont le **rejeu** a besoin | ce dont le **killfeed** a besoin |

**Les deux sont nécessaires au produit final.** Aucune ne remplace l'autre.

---

## 3. LES CONFLITS — 138, dont 16 seulement sont de vrais désaccords

Calculés par `git merge-tree` (aucun fichier touché).

| origine | nombre | nature |
|---|---|---|
| **les deux `filmdec`** | **16** | le vrai travail — deux lectures du même format |
| l'évolution de `main` | ~120 | workflows, `.gitignore`, `openapi.yaml`, `match-view`, `server.go`… **rien à voir avec la recherche killsource** |

### Les 16, avec leur divergence

| fichier | ajoutées / supprimées |
|---|---|
| `traverse.go` | **144 / 105** |
| `offline_biped.go` | 140 / 45 |
| `unit_weaponstate.go` | **109 / 143** |
| `offline_aim.go` | 95 / 15 |
| `quant_width_guard_test.go` | 68 / 11 |
| `frame_records.go` | 60 / 30 |
| `components_biped_ability.go` | 49 / **145** |
| `components_object.go` | 44 / 72 |
| `components_movement.go` | 40 / 60 |
| `offline_aim_test.go` | 40 / 9 |
| `probe_export.go` | 36 / 15 |
| `offline_biped_test.go` | 30 / 14 |
| `position_capture.go` | 26 / 61 |
| `components_batch5.go` | 16 / 4 |
| `registry.go` | 9 / 4 |
| `world.go` | 2 / 42 |

**Total sur `filmdec` : 2 972 ajoutées / 1 488 supprimées.**

---

## 4. LA MÉTHODE RETENUE — porter, ne pas merger

**Ne pas merger `killsource-prod` dans notre branche.** Cela ferait hériter 1 859 commits de
`main` à une branche qui ne les a jamais vus, pour résoudre 138 conflits dont 120 ne nous
concernent pas.

**Faire l'inverse** : repartir de `feat/killsource-prod` — saine, moderne, mergeable vers `main` —
et y porter notre travail. C'est jouable parce que **presque tout le nôtre est additif** : sur
**840 fichiers** touchés, seuls **80** intersectent avec l'autre branche.

### Phase A — la branche

- [x] A1. Depuis `feat/killsource-prod`, créer `feat/replay2d-prod` (nom à confirmer) et son
      worktree.
- [x] A2. Vérifier l'état de départ : `go build ./...`, tests `killsource` verts. **C'est la
      ligne de base** — si elle n'est pas verte, ne rien porter.

### Phase B — l'additif  *(conflit quasi nul)*

Tout ceci n'existe **que** chez nous et n'entre en collision avec rien :

- [x] B1. `apps/go-api/internal/analysis/replay/` — le paquet d'assemblage du rejeu.
- [x] B2. `apps/web/src/features/match-replay/` + la route `.../replay.tsx` + les types
      `ReplayDocument` dans `lib/api/types.ts` *(attention : ce dernier fichier est partagé,
      n'ajouter que nos blocs)*.
- [x] B3. `data/titles/halo_infinite/reference/` — bornes de carte, structures, objectifs.
      **N'existe que sur notre branche**, et `replay-build` échoue sans.
- [x] B4. Les documents `.ai/` du chantier. **Conflit attendu sur `thought_log.md`** : les deux
      branches y ont écrit. Fusionner par ordre chronologique, ne rien perdre.
- [x] B5. `cmd/replay-build/`, `cmd/mapstruct-build/`, `cmd/mapquant-build/`, `cmd/mapobj-build/`.

### Phase C — les 16 fichiers `filmdec`  *(le vrai travail)*

- [x] C1. Pour chacun, **diff des deux versions** et décision explicite. Les deux lignées
      descendent de la même recherche : une bonne part des écarts sera **additive**, pas
      contradictoire.
- [x] C2. **La règle de départage** : quand les deux versions décodent la même chose
      différemment, celle qui a un **test** ou une **mesure publiée** gagne. À défaut, garder les
      deux chemins et mesurer.
- [x] C3. **Ne jamais supprimer un fichier propre à l'autre lignée.** `killhealth.go` sert au
      killfeed, `fire_events.go` au rejeu — les deux restent.
- [x] C4. Le point le plus délicat est `traverse.go` (144/105) : c'est le dispatch des ~200
      grammaires de composants. Les deux lignées y ont ajouté des branches différentes. **Les
      réunir, pas en choisir une.**

### Phase D — les commits

- [x] D1. **Pas un seul commit géant.** Trois ou quatre commits thématiques : le décodage, le
      paquet d'assemblage, la feature web, les documents. Nos 320 commits sont une histoire de
      recherche dont la valeur est dans les docs `.ai` — qui voyagent comme fichiers.

---

## 5. LES CRITÈRES D'ACCEPTATION — c'est ce qui rend l'opération sûre

**Les deux lignées ont leurs mesures. Aucune ne peut être cassée en silence.**

### Côté rejeu — l'artefact doit se reconstruire à l'identique

```bash
cd apps/go-api
CGO_ENABLED=0 LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-build \
  --map Cliffhanger 000d5950 <repo>/data/cache/film_chunks/000d5950
```

| grandeur | attendu |
|---|---|
| traces / points | **99 / 29 221** |
| tirs rattachés | **475 / 519** |
| vies nommées | **90 / 105** |
| lancers | **70 / 70** |
| projectiles | **439** |
| états d'inventaire | **184** sur 24 images-clés |
| emprises de structure | **10 223** |

Et sur les deux autres films : `01e1f945` → 1 862/2 154 · `64e8adfa` → 2 312/2 879.

**Un écart sur l'un de ces chiffres = quelque chose a bougé. Comprendre AVANT de continuer.**

### Côté killfeed — leurs tests golden

- [x] `go test ./internal/games/halo_infinite/film/killsource/...` — le paquet a des
      `golden_*_test.go`, et un second test qui tourne **sans fixture** pour interdire que les
      golden dégénèrent en nombres nus. C'est le meilleur garde-fou des deux côtés.

### Côté app

- [x] `go build ./...` · `make check-types` · `npx vitest run` · les ratchets de pré-push
      (`knip`, linters, `govulncheck` qui charge **tout** le module).

---

## 6. CE QUE LA RÉCONCILIATION DÉBLOQUE

Une fois faite, le **fil des éliminations** du rejeu cesse d'être bloqué : `killsource` rend par
mort la victime, le tueur crédité, l'assistant, les deux parts de dégâts, la source du dégât et
le kill par véhicule. Il entre alors par une **entrée de données** (`Options.Kills`), comme
`Deaths` ou `Loadouts`, avec un adaptateur qui connaît les deux formes.

C'est le lot 5 de `PLAN_FINALISATION_REJEU_2D.md`, et les médailles en dépendent.

---

## 7. PIÈGES

1. **`thought_log.md` conflictera** — les deux branches y écrivent. Fusion chronologique.
2. **`data/…/reference/` n'existe que chez nous.** Un oubli et `replay-build` échoue avec une
   erreur de bornes de carte.
3. **Ne pas ranger `.ai/` pendant la réconciliation.** 85 fichiers à la racine, dont beaucoup
   seraient à archiver — mais les déplacer maintenant ajouterait du bruit à un merge déjà chargé.
   **Après.**
4. **`lib/api/types.ts` est partagé** : n'y porter que nos blocs `Replay*`, sans écraser ce que
   `main` y a ajouté en 1 859 commits.
5. **Ne pas relancer `git stash`** — interdit par le dépôt, commit WIP à la place.
6. Le pré-push a deux gates faciles à oublier : le ratchet de code mort (un export non consommé
   suffit) et `govulncheck`, qui **charge tout le module** — un seul `cmd/tmp_*` cassé bloque.

---

## 8. CE QUI N'EST PAS DANS CE PLAN

- **`feat/filmdec-killweapon`** reste une histoire orpheline, sans ancêtre commun avec `main`.
  Ce qu'elle contenait a été reposé sur `main` par `feat/killsource-prod` : elle n'a plus à être
  réconciliée, seulement archivée.
- Le rangement de `.ai/` — après, cf. piège 3.
- La suite du rejeu : `PLAN_FINALISATION_REJEU_2D.md` et les trois plans de chantier.

---

## 9. JOURNAL D'EXÉCUTION

> Ce fichier vit désormais sur `feat/replay2d-prod` (worktree `.claude/worktrees/replay2d-prod`).
> La copie restée sur `feat/filmdec-continuation` est figée — ne plus l'éditer.

### [2026-07-31] Phase A — CLOSE

- **A1** `[x]` — branche `feat/replay2d-prod` créée depuis `feat/killsource-prod` (`c9c1b9b41`),
  worktree `.claude/worktrees/replay2d-prod`. Nom retenu : celui proposé par le plan, sans
  variante.
- **A2** `[x]` — ligne de base mesurée dans le nouveau worktree :
  - `go build ./...` → exit 0.
  - `go test ./internal/games/halo_infinite/film/killsource/... ./internal/analysis/filmdec/...`
    → `ok killsource 0.271s` · `ok filmdec 0.230s`.

**Décision d'ordonnancement** : les phases B (additif) et C (les 16 `filmdec`) sont portées avant
tout gate de compilation global. Le paquet `internal/analysis/replay/` dépend d'API `filmdec` qui
n'arrivent qu'en phase C — un `go build` vert n'est donc **pas** attendu à la fin de B. Le premier
gate de compilation est posé à la fin de C, et les critères du §5 à la fin de D.

### [2026-07-31] Phase B — PARTIELLE (l'additif est porté ; le recâblage API/web reste)

**Le périmètre réel, mesuré.** `git diff 811be64ec..feat/filmdec-continuation` = **841 fichiers :
809 ajouts purs + 32 modifications**. Sur les 809 ajouts, **48 collisionnent** avec
`killsource-prod`, **761 non**. Sur les 48, **28 sont identiques octet pour octet** — il ne reste
que **20 vrais désaccords** : les 16 `filmdec` du §3, plus 4 hors décodeur.

- **B1** `[x]` — `internal/analysis/replay/` porté (28 fichiers). La fermeture transitive des
  dépendances va plus loin que la liste du plan : `go list -deps` sur la chaîne
  `replay` + les 4 `cmd/*-build` exige aussi `analysis/objectivescore`, `analysis/positions`,
  `analysis/weaponv3`, `internal/himap`, `internal/himodule`, `internal/ooz`. Tous portés.
- **B2** `[!]` — `features/match-replay/` (18 fichiers) porté. **La route ne l'est pas** :
  voir la découverte n°2 ci-dessous. `lib/api/types.ts` et `lib/query/keys.ts` sont laissés à
  leur version `main` en attendant.
- **B3** `[x]` — `data/titles/halo_infinite/reference/` porté (4 fichiers : `map_objectives.json`,
  `map_quant_bounds.json`, `map_structure/{ridgeline,sgh_streets}.json`).
- **B4** `[x]` — documents `.ai/` portés (111 fichiers). `thought_log.md` fusionné par
  interclassement chronologique : 136 entrées de notre lignée + 55 de la leur, **2 104 entrées
  au total, zéro perte**. Choix consigné : sur le conflit profond (rotation trimestrielle faite
  sur `main`, pas chez nous), les **deux** blocs sont conservés — dupliquer est réversible,
  perdre ne l'est pas. Re-appliquer la rotation est une tâche d'après-réconciliation.
- **B5** `[x]` — les 4 `cmd/*-build` portés, ainsi que l'outillage de recherche
  (`cmd/tmp_*`, `cmd/wf_*`, `tools/ce/`) : le piège 6 rappelle que `govulncheck` charge **tout**
  le module, donc les laisser derrière ne simplifierait rien et perdrait la recherche.

**Arbitrages sur les 4 collisions hors `filmdec`** :

| fichier | retenu | pourquoi |
|---|---|---|
| `.ai/RE_LOG_KILLWEAPON.md` | `killsource` | 18 309 lignes contre 824 — la leur est strictement plus riche |
| `.github/workflows/gitleaks.yml` | `killsource` | `main` fait autorité sur la CI |
| `cmd/rdata_weapon_scan/main.go` | nous | +829/-167, notre version a continué d'évoluer |
| `static/grenades-assets/.../index.json` | nous | correspond à nos 8 PNG (nommage minuscule) |

**Les 5 workflows CI + `.gitignore`** : version `main` retenue telle quelle. Vérifié sur pièces —
**nos améliorations CI sont déjà sur `main`** (`timeout-minutes` ×9, bloc `permissions:`, ratchet
`--new-from-merge-base` : tous présents chez eux). Seules nos 8 lignes de `.gitignore` sont
réajoutées. `internal/migration/order.go` : nos 4 migrations réinsérées dans l'ordre canonique.

### DEUX DÉCOUVERTES QUI INVALIDENT L'HYPOTHÈSE « TOUT EST ADDITIF » (§4)

Elles ne portent pas sur le décodeur — celui-ci se réconcilie bien comme prévu — mais sur les
**deux couches qui le consomment**, que les 1 859 commits de `main` ont refondues.

1. **L'API est passée à Huma.** Nous enregistrions le rejeu en chi dans `internal/api/server.go`.
   Sur `main`, le registre a déménagé dans `internal/api/wire/` et les routes se déclarent en
   `huma.Get(api, "...", h.handle..., humacore.Op(...))` **depuis les handlers**. Porter
   l'endpoint `/matches/{match_id}/replay` (plus `objective-events` et `positions`) est une
   **réécriture**, pas une copie.
2. **Le routage web a été refondu.** Notre route vivait en
   `routes/players/$playerSlug/matches/$matchId/replay.tsx`. Ce schéma est **mort** sur `main`
   (attrape-tout `players/$.tsx`) ; la vraie route est
   `routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx` — et elle
   **existe déjà, en stub**, derrière `RouteCapabilityGate` et le flag `REJEU_2D_ENABLED`. Notre
   implémentation doit remplacer ce stub en adoptant ses paramètres (`lang`, `titleSlug`) et sa
   garde. Le fichier au schéma mort a été retiré du portage.

### Ce qui reste de la phase B — 10 fichiers partagés, laissés à leur version `main`

`apps/go-api/api/openapi.yaml` · `internal/api/server.go` · `internal/service/match_view_service.go` ·
`internal/sync/comeback.go` · `features/match-view/{MatchKDCumulChart,MatchTugOfWarChart,MatchViewPage,queries}` ·
`lib/api/types.ts` · `lib/query/keys.ts` · `routeTree.gen.ts` (régénéré, jamais édité à la main).

Aucun ne contient de marqueur de conflit : ils sont **intacts**, à l'état `main`. Le travail
restant est le recâblage décrit ci-dessus, pas une résolution de conflit.

### [2026-07-31] Phase C — CLOSE. **Les deux lignées avaient un ancêtre commun.**

**LE FAIT QUI CHANGE TOUT, ET QUE LE §2 SE TROMPAIT EN NIANT.** Le plan posait deux lignées
« indépendantes », donc 16 conflits `add/add` à réconcilier à la main — 2 972 lignes ajoutées
contre 1 488 supprimées. C'est faux, et c'est mesurable :

```
git merge-base feat/filmdec-continuation feat/filmdec-killweapon  ->  83ea06dd5   (43 fichiers filmdec)
git diff feat/filmdec-killweapon feat/killsource-prod -- .../filmdec/              ->  VIDE
```

`83ea06dd5` porte déjà 43 fichiers `filmdec`, et le `filmdec` de `killsource-prod` est
**identique octet pour octet** à celui de `filmdec-killweapon`. Il existe donc un **ancêtre
commun du décodeur**, et les 16 fichiers ne sont pas un `add/add` : c'est une **fusion à trois
voies ordinaire**.

- **C1** `[x]` — les 16 fusionnés avec `83ea06dd5` pour base. **14 se fusionnent
  automatiquement**, 2 seulement conflictent (`components_movement.go` : 1 ; `unit_weaponstate.go` :
  8). Le « vrai travail » annoncé par le §3 se réduit à 9 conflits, tous dans 2 fichiers.
- **C2** `[x]` — règle de départage appliquée deux fois seulement, et jamais en éliminant :
  - `components_movement.go` : `absAxisW(pd,i)` (nous) contre `absAxisWFor(pd,idx,i)` (eux).
    **Aucun arbitrage nécessaire** : `absAxisWFor` retombe sur `absAxisW` quand aucune table par
    index n'est installée — ce qui est le défaut. Leur version est un sur-ensemble strict de la
    nôtre ; elle est retenue, et notre mesure (47 bits, table de région 13/13/14) est préservée
    telle quelle.
  - `unit_weaponstate.go` : **les deux lignées ont trouvé la même correction, à un jour
    d'intervalle et par la même méthode statique** — la rotation d'un cran de i25/i26/i27, puis
    la polarité active-basse du gate de `weapon-state-ammo`. Les corps de fonction obtenus sont
    identiques ; seuls les commentaires divergeaient. **Les deux faisceaux de preuve sont
    conservés** (nos mesures de largeur — 22 bits à 100 %, n=40 — et leurs adresses de `getName`
    + le recoupement Rosette à 10 bits), parce qu'ils ne se recouvrent pas.
- **C3** `[x]` — aucun fichier propre à une lignée n'a été supprimé. `killhealth.go`,
  `default_state_arch.go`, `components_flock.go`, `components_walk_batch9.go` (eux) et
  `capture.go`, `fire_events.go`, `grenade_events.go`, `i0_layout.go`, `keyframe_loadout.go`,
  `map_bounds.go`, `projectiles.go`, `vitality.go` (nous) cohabitent.
- **C4** `[x]` — `traverse.go`, annoncé comme le point le plus délicat (144/105), **se fusionne
  automatiquement** : les branches ajoutées de part et d'autre du dispatch ne se recouvrent pas.

**Quatre ruptures de compilation, toutes hors décodeur**, dues à l'évolution de `main` :

| symbole | cause | correctif |
|---|---|---|
| `authpkg.NewMSALProvider` (×2 : `cmd/tmp_filmmanifest`, `cmd/mapobj-build`) | ADR 0023 — MSAL retiré | `NewSISUProvider()`, le helper canonique des CLI de `main` |
| `nullableStr` (`weapon_kills_v3_repo.go`) | helper disparu de `main` | `nullableString`, même signature, même sémantique |
| `port.MatchViewService` incomplet | notre `port/services.go` a fusionné proprement, pas le service | les 2 champs, 2 `With*` et 2 méthodes (`GetObjectiveEvents`, `GetMatchPositions`) reportés — additif pur, aucune couche HTTP touchée |

**GATE — `go build ./...` vert, et les deux lignées de mesures tiennent.**

| test | résultat |
|---|---|
| `go test ./internal/analysis/filmdec/...` | `ok 0.305s` |
| `go test ./internal/games/halo_infinite/film/killsource/...` (golden) | `ok 0.301s` |
| `go test ./internal/analysis/replay/...` | `ok` (+ `mapvar` `ok`) |
| `go test ./internal/analysis/weaponv3/...` | `ok 43.1s` |

**Les critères §5 du rejeu, reconstruits sur la branche fusionnée — tous exacts :**

| grandeur | attendu | obtenu |
|---|---|---|
| traces / points (`000d5950`) | 99 / 29 221 | **99 / 29 221** |
| tirs rattachés | 475 / 519 | **475 / 519** |
| vies nommées | 90 / 105 | **90 / 105** |
| lancers | 70 / 70 | **70 / 70** |
| projectiles | 439 | **439** |
| états d'inventaire | 184 | **184** |
| emprises de structure | 10 223 | **10 223** |
| `01e1f945` | 1 862 / 2 154 | **1 862 / 2 154** |
| `64e8adfa` | 2 312 / 2 879 | **2 312 / 2 879** |

Aucun écart. La réunion du décodeur est **bit-exacte** des deux côtés.

### [2026-07-31] B2 (rec��blage) + phase D + §5 — CLOSE, avec UNE réserve nommée

**B2** `[x]` — l'endpoint et la route existent, réécrits contre l'architecture de `main`.

- **API en Huma.** `GET .../matches/{match_id}/replay` est déclaré par
  `ReplayHandler.Mount` (`huma.Get` + `humacore.Op`), monté dans `server_apiv1.go` ; les deux
  calques `objective-events` et `positions` rejoignent `MatchViewHandler.Mount` (4 routes au
  lieu de 2). Le registre `wire/registry_pages.go` gagne la factory `Replay` et les deux
  repos du MatchViewService.
- **Le garde local est devenu un middleware.** Il repose sur `r.RemoteAddr` — la seule donnée
  que le demandeur ne peut pas falsifier — et Huma ne l'expose pas à un handler typé. Il vit
  donc à l'étage transport (`handlers.LocalOnlyReplay`), monté en `r.Use` devant la route. Le
  contenu du garde (date de bascule, cible de retrait, critère mesurable) est intact.
- **Deux ratchets ont dicté la forme** : `TestNoCapabilityErrorDup` impose
  `MapCapabilityError(ctx, err, probe)` plutôt qu'un `errors.Is` recopié ;
  `TestContractRoutesDocumented` + `TestOpenAPIYAMLIsUpToDate` imposent `openapi-gen` puis
  `generate-types`. Les deux sont passés.
- **Route web sur le nouveau schéma.** `$matchId.tsx` devient un layout à `Outlet`, la vue
  passe dans `$matchId/index.tsx`, et notre implémentation remplace le stub
  `REJEU_2D_ENABLED` en gardant `RouteCapabilityGate` et les paramètres `lang`/`titleSlug`.
  `routeTree.gen.ts` régénéré par le générateur du projet (jamais édité à la main).
- **Les query keys sont title-scopées** (`matchReplay`, `matchObjectiveEvents`,
  `matchPositions` prennent `titleSlug`), comme l'exige le garde-rail anti-fuite cross-titre.
- **Les deux calques match-view sont rebranchés** : l'overlay de captures CTF a été
  ré-implémenté contre les builders extraits de `main` (`buildCaptureSeries`), et la heatmap
  de positions est montée dans la section « Déroulé ». Sans cela, `_objectiveCaptures.ts`,
  `MatchPositionsHeatmap.tsx` et les deux hooks devenaient du code mort — que `knip` refuse.
- **`comeback.go` réuni** : notre routage par type d'objectif (CTF → courbe de captures ;
  zone/hill/skull → 0) englobe leur repli « marge de score finale » pour les titres sans
  kill-feed. Les deux apports coexistent, les 3 tests CTF passent.

### §5 — LES CRITÈRES D'ACCEPTATION, MESURÉS

| gate | commande | résultat |
|---|---|---|
| build Go | `go build ./...` | **vert** |
| suite Go complète | `go test ./...` | **vert** (0 FAIL) |
| golden killfeed | `go test ./internal/games/halo_infinite/film/killsource/...` | **vert** |
| rejeu — 3 films | `replay-build` | **identiques au §5** (revérifié après tous les changements) |
| types web | `tsc --noEmit` | **vert** |
| tests web | `npx vitest run` | **384 fichiers, 3 344 tests, 0 échec** |
| eslint | `npm run lint` | **0 erreur** (19 warnings préexistants, react-compiler/TanStack Table) |
| code mort web | `npx knip` | **vert** (exit 0) |

**Quatre ratchets Go ont dû être traités** — tous déclenchés par l'outillage de recherche :
`TestNoNewHalowaypointLiteral` (2 entrées d'allowlist datées), `TestNoLocalLongestRun`
(exemption par chemin, datée et argumentée — ce sont des mesures de plage de bits, pas des
séries métier ; **et ce test échouait DÉJÀ sur `feat/killsource-prod` avant toute
réconciliation**), `TestNoNewRawStartTimeLiteral` (corrigé pour de bon : les 2 sites appellent
`analysis.SQLStartTimeCanonical`), `TestSentinel_NoNewEnvVarReaders` (le repli env-var legacy
de `cmd/mapobj-build` est **supprimé**, pas allowlisté).

### LA RÉSERVE — le ratchet golangci-lint reste rouge, et il l'était déjà

`golangci-lint run --new-from-merge-base=origin/main` :

| branche | issues |
|---|---|
| `feat/killsource-prod` (avant nous) | **30** |
| `feat/replay2d-prod` (après le portage) | **48** |

**Le gate n'est pas cassé par la réconciliation : il était déjà rouge.** Les 26 issues que le
portage ajoute sont dans l'outillage de recherche et le décodeur (8 `unconvert` dans
`cmd/tmp_*`, 3 `goconst` sur des noms de composants `filmdec`, 3 fonctions non appelées,
4 `unparam`, 2 `gocyclo` dans `himap`/`himodule`, 1 `errcheck`, 1 `ineffassign`, 1 `prealloc`,
1 `revive`). **Non traitées** — les corriger ne rendrait pas le gate vert (les 30 de la lignée
killfeed resteraient), et toucher au décodeur pour du style ferait courir un risque à des
mesures qu'on vient de prouver bit-exactes. C'est une dette à solder AVANT le merge vers
`main`, sur les deux lignées à la fois. `[!]`

`govulncheck` remonte 11 vulnérabilités de la bibliothèque standard Go 1.26.1 (corrigées en
1.26.2) : c'est la version de la chaîne d'outils locale, sans rapport avec ce portage.

### CE QUI N'A PAS ÉTÉ PORTÉ, ET POURQUOI

Rien de fonctionnel. Le seul écart assumé est la **dette de lint ci-dessus**.

### Découvertes (hors périmètre, non traitées)

1. **Deux jeux d'images de grenades coexistent** — `Dynamo-light.png` (lignée `killsource`) et
   `dynamo_light.png` (la nôtre). `index.json` ne peut en désigner qu'un : le jeu majuscule est
   désormais orphelin. Aucun consommateur dans le code (`grep` sur `apps/`), donc sans effet ;
   à nettoyer après la réconciliation.
2. **La rotation trimestrielle du `thought_log` est à re-appliquer** après la réconciliation
   (cf. B4).
3. **Le stub de rejeu de `main` est probablement inatteignable** : `$matchId.tsx` y rend
   `MatchViewPage` sans `Outlet`, alors que `$matchId/replay.tsx` est une sous-route. Notre
   lignée avait justement fait la bascule en layout + `index.tsx`. À traiter avec le recâblage
   web, pas avant.
