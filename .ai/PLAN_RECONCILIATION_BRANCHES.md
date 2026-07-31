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

- [ ] B1. `apps/go-api/internal/analysis/replay/` — le paquet d'assemblage du rejeu.
- [ ] B2. `apps/web/src/features/match-replay/` + la route `.../replay.tsx` + les types
      `ReplayDocument` dans `lib/api/types.ts` *(attention : ce dernier fichier est partagé,
      n'ajouter que nos blocs)*.
- [ ] B3. `data/titles/halo_infinite/reference/` — bornes de carte, structures, objectifs.
      **N'existe que sur notre branche**, et `replay-build` échoue sans.
- [ ] B4. Les documents `.ai/` du chantier. **Conflit attendu sur `thought_log.md`** : les deux
      branches y ont écrit. Fusionner par ordre chronologique, ne rien perdre.
- [ ] B5. `cmd/replay-build/`, `cmd/mapstruct-build/`, `cmd/mapquant-build/`, `cmd/mapobj-build/`.

### Phase C — les 16 fichiers `filmdec`  *(le vrai travail)*

- [ ] C1. Pour chacun, **diff des deux versions** et décision explicite. Les deux lignées
      descendent de la même recherche : une bonne part des écarts sera **additive**, pas
      contradictoire.
- [ ] C2. **La règle de départage** : quand les deux versions décodent la même chose
      différemment, celle qui a un **test** ou une **mesure publiée** gagne. À défaut, garder les
      deux chemins et mesurer.
- [ ] C3. **Ne jamais supprimer un fichier propre à l'autre lignée.** `killhealth.go` sert au
      killfeed, `fire_events.go` au rejeu — les deux restent.
- [ ] C4. Le point le plus délicat est `traverse.go` (144/105) : c'est le dispatch des ~200
      grammaires de composants. Les deux lignées y ont ajouté des branches différentes. **Les
      réunir, pas en choisir une.**

### Phase D — les commits

- [ ] D1. **Pas un seul commit géant.** Trois ou quatre commits thématiques : le décodage, le
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

- [ ] `go test ./internal/games/halo_infinite/film/killsource/...` — le paquet a des
      `golden_*_test.go`, et un second test qui tourne **sans fixture** pour interdire que les
      golden dégénèrent en nombres nus. C'est le meilleur garde-fou des deux côtés.

### Côté app

- [ ] `go build ./...` · `make check-types` · `npx vitest run` · les ratchets de pré-push
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

### Découvertes (hors périmètre, non traitées)

_(rien pour l'instant)_
