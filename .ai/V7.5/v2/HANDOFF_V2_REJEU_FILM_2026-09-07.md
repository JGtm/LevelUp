# Handoff — chantier v2 rejeu/film — état au 2026-09-07 (matin)

> À relire APRÈS compaction, avant toute action. Complète (ne remplace pas) `HANDOFF_V2_REJEU_FILM_2026-09-06.md`.
> Mémoire détaillée (IDs d'agents, hashs, leçons) : `memory/project_v2_rejeu_film_chantier.md`.
> DÉCISION USER 07/09 : PÉRIMÈTRE GELÉ — on termine ce qui est en vol, aucun nouveau lot sans accord explicite.
> Consignes de rythme (user) : 2 agents max en parallèle, Sonnet pour rondes 2 et tâches mécaniques, Opus seulement
> pour instruire un algorithme, comptes rendus courts (ce qui a changé, ce qui bloque).

## 1. Où en est l'intégration

- `feat/v75` = `a059caefc` (schéma 43), BLOQUÉ : `.ai/thought_log.md` du worktree principal porte ~97 lignes NON COMMITÉES
  d'une autre session du user (frise point de vue, worktree `LevelUp-wt-frise-pov`). Ne pas toucher, ne pas `stash`.
  Question posée au user (commit par l'autre session recommandé) — sans réponse à ce jour.
- Intégration DÉPORTÉE : worktree `LevelUp-wt-v2-integ`, branche `feat/v2-integ` (CI sur `feat/**`), agent `a10b801adef5284fd`.
  Merges faits, gates complets verts après chacun : manches `90ca609a0` (44) → durées `6af8f6db8` (45) → corpus `0d862af0a`
  → web-vies `7cdb0e56f` → vies-anonymes `eb7a3dfbd` (47). HEAD integ = `eb7a3dfbd`.
- Restent à fusionner dans integ : `feat/v2-drapeaux` (46 ; HEAD `9ed9dea3a` + test W1 en cours par `aa38defbd4d87e2b8`),
  `feat/v2-pont-muet` (48 ; HEAD `b4c0d58c2`, ronde PONT-R2 en cours par `afa9719b7095526dd`).
  Revue VIES-ALIGN (alignement vies/durées `74f7af7fa`) en cours par `a676dc6d036cdf9ef` — si constat P0/P1 : corriger sur
  `feat/v2-vies-anonymes` puis re-merger dans integ.
- Règles de conflit à l'intégration : `.ai/*.md` concaténation `sed -i '/^<<<<<<< /d; /^=======$/d; /^>>>>>>> /d'` ;
  `document.go` → SchemaVersion = le plus grand, chronique = toutes les entrées ordonnées ; `structure_test.go` un seul bloc
  de contrôle ; goldens régénérés (`-run GoldenAssembly -update`) ; `openapi.yaml`/`generated.ts` régénérés par les cibles ;
  `.ai/baselines/` JAMAIS régénéré ; tout autre `.go` en conflit → abort, faire aligner la branche par son exécuteur.

- **Mise à jour (07/09, fin de matinée)** : intégration integ TERMINÉE, sept lots fusionnés.
  Hashs réels finaux : manches `90ca609a0` (44), durées `6af8f6db8` (45), corpus `0d862af0a`,
  web-vies `7cdb0e56f`, vies-anonymes `eb7a3dfbd` (47), pont-muet `ee4084c14` (48, aligné depuis
  `273e94f11`), drapeaux `1b32fc775` (46, aligné depuis `3f0f81ed8` — les hashs `9ed9dea3a` et
  `b4c0d58c2` ci-dessus sont périmés, remplacés après alignement de chaque branche sur l'état
  courant de `feat/v2-integ`). HEAD `feat/v2-integ` = `1b32fc775`, SchemaVersion 48, gates
  verts, poussé sur `origin`. Intégration integ terminée, reste le ff (fast-forward de
  `feat/v2-integ` dans `feat/v75` dès que le blocage du principal est levé).

## 2. Séquence de fin (dans l'ordre)

1. Attendre PONT-R2, VIES-ALIGN, W1 drapeaux. Corrections éventuelles par les exécuteurs (Sonnet pour ronde 2).
2. Intégrateur : merge drapeaux (46) puis pont (48) dans integ, gates complets, entrée plan + thought_log + handoff §10,
   push `feat/v2-integ`, CI par `gh run view` (JAMAIS `gh run watch` long : sature le disque temporaire).
3. Quand le journal de l'autre session est commité : dans le principal `git merge --ff-only feat/v2-integ`, push `feat/v75`, CI.
4. Notion « Backlog LevelUp » (page 39a7ae87-e7a3-809e-8e03-e4ffedcf5086), « Séquence à dérouler à la release » :
   « Re-cuisson du parc au schéma 41 » → 48 ; ajouter : recompter la jointure des bots après re-cuisson.
5. Lot final mécanique (Sonnet) : extraction des chroniques (`document.go` 1417 L, `usage_summary.go` 525 L, `replaybuild.go`
   572 L > 500) — déplacement pur, goldens inchangés.
6. Nettoyage : worktrees `LevelUp-wt-v2-*` (vérifier `Test-Path` des jonctions et `node_modules` avant `git worktree remove` ;
   ABORTER si une jonction subsiste), caches `%LocalAppData%\go-build-v2-*`, `golangci-v2-*`, scratchpad `balayage/`, `manches/`,
   `durees/`, `vies/`, `drapeaux/`, `pont/`, `corpus/` (racines à JONCTIONS : retirer les jonctions par `cmd //c rmdir`, jamais
   `rm -rf` à travers). Mémoire : `reference_worktree_remove_follows_junctions`, `reference_msys_ln_s_copies_use_mklink_junction`.
7. Tag v7.5.0 : séquence Notion, PRÉVENIR le user avant tout push sur `main` (déploiement prod).

## 3. Décisions user (07/09)

- Les vies anonymes n'existent pas : une vie = humain ou bot ; nommer à la source, jamais « inconnu » (mémoire
  `feedback_no_anonymous_lives_every_life_is_named`).
- Un joueur ne porte jamais son propre drapeau ; CTF neutre = un seul drapeau (`reference_ctf_flag_lua_mechanics`).
- Fin de manche = replacement au point de départ SANS mort → vies terminées sans mort sur tous les slots
  (`reference_round_end_respawn_lives_without_death`). Hypothèse user à vérifier : drapeaux, crânes et bombes ne « meurent »
  pas non plus entre deux manches (objets replacés sans événement de lâcher/retour daté) — cause probable des états
  `enJeu`/`sol` périmés et des recouvrements sur films multi-manche.
- Périmètre gelé ; les restes du registre = PROCHAIN CHANTIER (à planifier sous plan-review avec accord user).

## 4. Prochain chantier (restes au registre, NE PAS LANCER sans accord)

Actions d'objectif écartées par le pont statborg (`deathInstantMin = 3` ; `3372e7eb` 35 actions, 2 joueurs) · vies sur un slot
jamais mort (`d9781168` 19 ; = joueur à 0 mort de la feuille ; table d'index joueur du film, `ti=5`, `ManagedPropertyFilmIndex`
jamais branchées) · écart K/D/A résiduel `51ebbc0f` (69) · machine à états des drapeaux à partager (`assembleFlagLives`,
`64e8adfa` closedOverlaps 10) + reset des objets aux frontières de manche (hypothèse ci-dessus) · CTF multi-manche et calques
VIP à re-vérifier au schéma 48 · 5 P2 de l'audit · `drawnSwapAt` web · jointure bots (égalité de chaîne) à recompter.

## 5. Pièges appris (07/09)

`ln -s` COPIE sous MSYS (29 Go) → `mklink //J` · dans un worktree `MERGE_HEAD` est sous `<repo>/.git/worktrees/<nom>/` ·
ne jamais renommer un test gelé dans `.ai/baselines/tests_pre_migration.jsonl` · une défense en profondeur masque la mutation
du composant qu'elle protège (tester le composant seul) · deux lots revus séparément peuvent se casser à l'intersection
(recuire les témoins après alignement) · un gate doit prouver qu'il ne peut pas passer vert à vide (plancher de couverture).
