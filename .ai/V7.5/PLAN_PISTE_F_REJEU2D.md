# Plan piste F — rejeu 2D : fonds de carte + kill feed

> Worktree `LevelUp-wt-replay2d`, branche `feat/v75` (mode branche unique). JAMAIS de merge
> vers `main`. Le rejeu reste OFF EN PROD : le garde `handlers/replay_local_gate.go` n'est PAS
> touche par ce lot.

## Perimetre ferme

| item | statut |
|---|---|
| F1 — fond de carte servi par l'API et dessine sous le rejeu | `[x]` |
| F2 — kill feed dans la page replay, synchronise avec l'horloge du rejeu | `[x]` |
| F3 — lien header -> replay : verifier, ne pas dupliquer | `[x]` |

Tout le reste (completion des cartes manquantes, defaut des toits, attribution fine des tirs)
est HORS PERIMETRE — les decouvertes vont en section « Decouvertes », pas dans le code.

## Etat de l'art verifie sur pieces (2026-08-11)

1. **Les assets existent** : 21 PNG + 21 sidecars sous
   `data/titles/halo_infinite/reference/map_backgrounds/`. Aucun consommateur ne les lit
   (seul `cmd/mapfond-build` les ecrit) — verifie par grep des resolveurs
   `PathResolver.MapBackground{Dir,Path,MetaPath}`.
2. **Le calage est absolu et compatible avec le rejeu.** Le document de rejeu est
   dequantifie avec l'AABB monde du BSP (`filmdec.MapQuantEntry`), donc ses coordonnees
   SONT des metres monde. Controle : `000d5950` (Cliffhanger) a
   `bounds` X[-7,02 ; 43,79] Y[-25,14 ; 30,35], entierement contenu dans le cadre de
   `ridgeline.png` (X[-57,30 ; 92,93], Y[-70,81 ; 78,87]). La phrase du doc de paquet
   « l'echelle/offset absolus ne sont PAS garantis » date d'avant `map_bounds.go` : elle
   est corrigee dans le meme lot.
3. **La chaine nom affiche -> module existe deja** et n'est declaree qu'a un endroit :
   `filmdec.NormalizeMapName` + `map_quant_bounds.json` (15 cartes, `cliffhanger ->
   ridgeline`). C'est elle qu'on reutilise, on n'en ecrit pas une seconde.
4. **Le document de rejeu ne porte AUCUNE identite de carte.** Le fond se resout donc au
   SERVICE (match -> carte -> module), pas au build : rebatir les artefacts existants pour
   y injecter un champ couterait un decodage de film complet et rendrait la feature
   invisible sur les 3 artefacts deja produits.
5. **Le kill feed n'a besoin d'aucun nouvel appel** : la page replay charge deja
   `useMatchView`, dont `combat_tab.highlight_events` porte les kills AVEC l'arme resolue
   (`weapon_label`, `weapon_image_url`, `weapon_image_tinted`) et `team_tab.scoreboard`
   porte les equipes. Aucun artefact a reconstruire.
6. **PIEGE MESURE — les deux horloges ne sont pas la meme.** `highlight_events` est servi
   T0-CORRIGE par la Match View (`correctMatchViewEventsT0`), tandis que le film part du
   DEBUT DU MATCH, countdown compris. Mesure sur `000d5950` (T0 = 18 465 ms) : en
   rapprochant les 91 fins de vie exploitables du rejeu des 93 morts du registre, l'ecart
   median vaut **-629 ms a offset 0** contre 3,1 s a +T0 et 4,2 s a -T0. Le rejeu suit donc
   l'horloge BRUTE. Recalage retenu : `msRejeu = event_time_ms + t0_ms`, ce qui exige
   d'exposer `t0_ms` au contrat (il ne l'etait pas).

## Etapes

### Etape F1a — backend : servir le fond de carte

- `[x]` `internal/analysis/replay` : resolution pure `module -> fond` (chargement du
  sidecar deja ecrit, pas de nouvelle formule de calage).
- `[x]` `internal/platform/duckdb` : resolution `match_id -> nom de carte` (registre +
  cascade EN par `asset_translations` quand `map_name` est un UUID).
- `[x]` `internal/service/replay_service.go` : `MapBackground` (calage) et
  `MapBackgroundImage` (octets PNG), degradation gracieuse en `ErrReplayNotAvailable`.
- `[x]` `internal/api/handlers/replay.go` : `GET .../replay/background` (Huma, JSON) et
  `GET .../replay/background.png` (chi, octets), sous le MEME groupe que le rejeu — donc
  derriere le garde local inchange.
- `[x]` Contrat regenere (openapi + types web) dans le meme commit.
- Gate : `go test ./internal/analysis/replay/... ./internal/service/... ./internal/api/...`

### Etape F1b — front : dessiner le fond sous le rejeu

- `[x]` `mapBackground.ts` : convention monde<->pixel (miroir exact du sidecar) + rectangle
  de dessin, pur et teste.
- `[x]` `queries.ts` + `lib/query/keys.ts` : chargement du calage et de l'image.
- `[x]` `ReplayCanvas` : le fond passe SOUS tout le reste ; repli sur le sol structure /
  les props Forge quand la carte n'a pas de PNG.
- `[x]` Strings FR + EN.
- Gate : `make check-types` + `make test-web`.

### Etape F2a — backend : exposer `t0_ms`

- `[x]` `MatchViewHeader.t0_ms` (offset countdown, deja en base) + contrat regenere.
- Gate : `go test ./internal/service/...`

### Etape F2b — front : le kill feed dans la page replay

- `[x]` Extraire de `_momentum.ts` la collecte des kills (sans binning) — source unique
  partagee avec le kill feed de la Match View.
- `[x]` Extraire la cascade de couleur d'equipe de `MatchKillFeed.tsx` vers
  `match-view/teamColor.ts` (et non `colors.ts`, qui porte un autre sujet : la palette des
  JOUEURS) — une seule cascade, deux consommateurs.
- `[x]` BONUS IMPOSE PAR LA REGLE n6 : la cascade « allie = meme camp que moi » existait
  deja en DEUX copies (`MatchTugOfWarChart`, `MatchKDCumulChart`) et une troisieme
  variante (`MatchCadenceChart`). Centralisee dans `match-view/xuidMeta.ts`, les trois
  copies migrees, garde-rail `xuidMeta.guard.test.ts` pose (il a d'ailleurs DEBUSQUE la
  troisieme copie, que le grep initial avait manquee).
- `[x]` `ReplayKillFeed.tsx` : les kills du moment, recales `+ t0_ms`, icone d'arme et
  couleur d'equipe reutilisees telles quelles.
- `[x]` Branchement dans la page replay, strings FR + EN.
- Gate : `make check-types` + `make test-web`.

### Etape F3 — le lien header

- `[x]` Verifier `MatchHeader.replayLink.tsx` sur pieces (route, params, condition
  d'affichage). Ne rien dupliquer.

### Cloture

- `[x]` `golangci-lint --new-from-merge-base=origin/main` : **0 issue**.
- `[x]` `make check-types`, `npm run lint` (0 erreur, 19 avertissements — 20 avant ce lot),
  `lint:fields`, `lint-no-hardcoded-colors`, `lint-contract-ratchet`,
  `check-generated-types-fresh`, `knip-ratchet` (0/0/0), `npm run build`.
- `[x]` `lint-cross-feature-imports` : 7 / plafond 7. Le plafond n'a PAS ete releve —
  `match-replay=>match-view` est entree a l'allowlist avec sa justification ecrite.
- `[x]` Tests Go cibles : `service`, `api/...`, `port`, `analysis/replay`, `domain/...`,
  `archlint`, `platform/duckdb` (+ `-tags=integration -run TestReplayMapRepo`),
  `make go-api-test`. Tout vert. `himap` NON joue en local (>60 min) — la CI tranche.
- `[x]` `make test-web` : suite complete.
- `[x]` thought_log + plan a jour.
- `[!]` Push `feat/v75` + CI de branche verte AU NIVEAU JOB — a faire, l'accord de commit
  appartient a l'utilisateur (regle du depot).
- `[!]` Gate visuel utilisateur sur `000d5950` — le rendu change (fond de carte + kill
  feed), donc le verdict appartient a l'utilisateur, et les temoins sont NOMMES PAR LUI.
  PIEGE D'ENVIRONNEMENT a lui signaler : le worktree `LevelUp-wt-replay2d` n'a PAS les
  bases (son `shared_matches_v2.duckdb` fait 12 Kio) — la page a besoin de la Match View
  pour le kill feed, donc le serveur doit tourner sur la racine de donnees du depot
  principal (`LEVELUP_REPO_ROOT`), pas sur celle du worktree.

## Decouvertes (non traitees dans ce lot)

1. **La documentation de paquet du rejeu etait FAUSSE** : `analysis/replay/document.go`
   affirmait que « l'echelle/offset absolus ne sont PAS garantis ». C'etait vrai avant
   `filmdec/map_bounds.go` (toutes les cartes dequantifiees avec les bornes de
   Cliffhanger). Corrige dans ce lot, avec sa mesure — sans quoi la piste F entiere
   paraissait impossible.
2. **`MatchCadenceChart` portait une TROISIEME variante** de la cascade d'appartenance
   d'equipe, avec une regle de repli legerement differente. Migree (comportement
   preserve), mais elle dit que le grep manuel ne suffit pas : c'est le garde-rail qui
   l'a trouvee.
3. **6 des 21 fonds de carte cuits ne sont atteignables par AUCUN rejeu** : le catalogue
   de bornes ne connait que 15 cartes, et un artefact de rejeu ne se produit pas sans
   bornes. Les fonds de `recharge`, `prism`, `deadlock`, `oasis`, `scarr` et `corpo`
   attendent donc une entree au catalogue. Hors perimetre — a verser au registre des
   reports.
4. **La suite web a un test instable** : `PalmaresRelationsPage.test.tsx` (« badge
   cross-jeu ») a expire a 5 s sous charge de suite complete, puis passe en 5 s isole.
   Aucun rapport avec ce lot.

## Journal

- 2026-08-11 — etat de l'art verifie sur pieces, plan ecrit.
- 2026-08-12 — F1, F2, F3 livrees et verifiees. Reste : accord de commit/push utilisateur,
  CI de branche, gate visuel.
