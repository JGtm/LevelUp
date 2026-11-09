# Recoupements entre J12 et la campagne de grammaire (2026-10-01, soir)

> Source : réponse de la session du chantier de suite d'audit (PC principal de l'utilisateur, Remote
> Control), reçue le 2026-10-01 au soir, lecture seule de son côté. La branche J12 n'étant pas poussée,
> ces faits ne sont pas vérifiés par diff ici : à reconfirmer après `git fetch` quand elle le sera.
>
> **Mise à jour du 2026-10-02 (critique de complétude n° 2, N20)** : la branche J12 est désormais
> poussée (`origin/feat/suite-audit-decodeur-j12` = `bc0e2511a`), et ses faits ont été vérifiés par
> diff (section « Vérifié par diff » ci-dessous). La phrase ci-dessus et la mention « NON poussée »
> de la section suivante sont PÉRIMÉES sur ce point ; elles sont gardées telles que reçues.

## Faits transmis

> **Périmée sur la poussée** (2026-10-02) : « NON poussée » ci-dessous était vrai à la réception, le
> 2026-10-01 au soir. La branche a été poussée depuis et vérifiée par diff le 2026-10-02 (dernière
> section). Le reste se lit avec cette section-là, qui fait foi.

- Branche J12 : `feat/suite-audit-decodeur-j12`, tête `bc0e2511a`, NON poussée ; elle contient
  `8b894a677` (fusion J11 dans `feat/v75`, CI verte, poussée le 2026-10-01 vers 17 h).
- Base de la campagne : `69564ef7d` = commit local de documentation posé sur `8b894a677` (vérifié ici :
  `git merge-base --is-ancestor 8b894a677 69564ef7d` vrai). La campagne a donc J1 à J11.
- Ampleur de J12 : 1 130 fichiers changés sur le module (7 121 +, 6 199 −), surtout mécanique :
  `go fix` Go 1.26 (637 fichiers, dont 500 de test), 188 tris `sort.Slice*` → `slices.*` à comparateur
  total, `ctx` propagé et `slog` → variantes `Context`, plus de `slog` dans `grammar`/`facts`
  (diagnostics typés `film/internal/constat`), tag `//go:build research` sur ~380 instruments, chemins
  `.ai/` et commentaires `film/filmdec/` corrigés.
- Fichiers de la campagne touchés par J12 : `grammar/object_deaths_march.go` (+3/−4, tri),
  `grammar/movement_states.go` (+5/−10, tri + `go fix`), `grammar/testdata/grammar_rev.golden`
  (empreinte régénérée à révision constante `grammar-2026-09-27.3`), `dispatch_biped.go`,
  `dispatch_item.go`, `dispatch_player.go`, `keyframe_closure.go`, `vehicle_occupancy_march.go`,
  `movement_states_jump.go`.
- INCHANGÉS par J12 : `facts/killsource/walk.go`, `grammar/testdata/ecs_table.tsv`,
  `grammar/testdata/frame_closure.golden` ; `marchLocateStrict` non touché à sa connaissance (grep à
  refaire après fetch).
- Vague J11.4 : en cours sur le PC principal depuis 17 h 47 (serveur arrêté) ; `backfill-replay` vers
  1 000 / 1 292 films, puis `usage-summary`, `pad-tiers --force`, `killsource` (3 ouvriers),
  `bomb-stats --force`, `vehicle-takes --force`, `healthcheck` ; fin estimée vers 5-6 h le 2026-10-02.
  Sorties : `data/cache/replays/halo_infinite/<short8>.json` du PC principal (`coverage.fallbacks`),
  lignes killsource en base `shared_matches_v2` (`decoder_rev = killsource-2026-09-27`). L'autre PC n'a
  pas le même jeu de données.
- Fusion de J12 dans `feat/v75` visée en fin de matinée du 2026-10-02 (preuve « zéro différence »
  sur 20 films, ~1 h, puis CI), suivie de la revue adversariale finale du chantier, qui peut encore
  toucher `grammar`.

## Conséquences pour la campagne

1. Les lots de MARCHE (naissances : `movement_states.go`, `object_deaths_march.go`) et les lots de
   COMPOSANTS qui passent par `dispatch_*.go` auront des conflits textuels avec J12 (tris, `go fix`) :
   mécaniques, à résoudre en fusionnant `feat/v75` post-J12 dans `feat/campagne-grammaire`
   (`feat/v75` a raison en cas de conflit). Précisé le 2026-10-02 (N9) : « mécaniques » est attendu
   par lecture du diff, pas prouvé par un essai de fusion. J12.3 (`film_context.go`, sans `slog`) et
   J12.4 (`registry.go`) sont des contraintes de STRUCTURE pour L6a et L8. Un essai `git merge-tree`
   est prévu avant tout développement de phase 2 (PLAN §6.0).
2. `grammar_rev.golden` : les deux branches le régénèrent à révision constante ; à re-régénérer par la
   commande du dépôt après la fusion.
3. J12.7 pose un ratchet de tag `research` : voir la section « Vérifié par diff » ci-dessous —
   `grammar/frame_closure_detail*.go` sont du code de production (non tagué, appelé par l'instrument
   seul, comme `FrameClosure`) et ne peuvent pas porter le tag ; les instruments de la campagne le
   portent déjà.
4. Une montée de `grammar.Rev` par un lot de la campagne périme le parc recuit par J11.4 : la recuisson
   qui suivra se décide avec l'utilisateur, une seule fois, après la fusion des lots. Précisé le
   2026-10-02 (PLAN §6.0 et §6.3, D7) : une recuisson par VAGUE fusionnée ; deux vagues sont
   recommandées.

## Vérifié par diff (2026-10-02, après `git fetch` de `origin/feat/suite-audit-decodeur-j12` = `bc0e2511a`)

- `bc0e2511a` contient `origin/feat/v75` ; `git diff --stat origin/feat/v75...origin/feat/suite-audit-decodeur-j12 -- …/film/` = 1 130 fichiers, 7 121 +, 6 199 − (identique à l'annonce).
- Fichiers MODIFIÉS par la campagne à ce jour et touchés par J12 : trois seulement —
  `grammar/testdata/grammar_rev.golden` (à re-régénérer), `research/cmd_fermeture/main.go`
  (`strings.Split` → `strings.SplitSeq`) et `research/cmd_fermeture/gb1_research_test.go`
  (`os.IsNotExist` → `errors.Is(err, fs.ErrNotExist)`) : conflits mécaniques.
- Aucun fichier NEUF de la campagne n'existe dans J12.
- Ratchet `archlint/research_tag_test.go` de J12.7 : tout `*_research_test.go` porte `//go:build research`
  en ligne 1 (ou `research && X`) — tous les fichiers de la campagne le portent ; un fichier de
  production ne peut PAS porter le tag : `frame_closure_detail*.go` restent donc non tagués, au même
  statut que `FrameClosure` (production, appelée par l'instrument seul).
- À la fusion de `feat/v75` post-J12 : jouer `go test ./internal/archlint/` et conformer le code de la
  campagne aux ratchets de J12 (tris à comparateur total, `errors.Is`, `strings.SplitSeq`, `slog`
  absent de `grammar`) — plusieurs instruments de la campagne emploient encore `sort.Slice`,
  `os.IsNotExist` ou `strings.Split`.
