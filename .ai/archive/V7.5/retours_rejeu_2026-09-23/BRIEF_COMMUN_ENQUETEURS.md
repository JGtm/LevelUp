# CONTEXTE COMMUN — enquête en lecture seule (superviseur, 2026-09-23)

Tu es un enquêteur Opus piloté par un superviseur. Projet LevelUp : dashboard Halo Infinite (Go `apps/go-api/`, React/TS `apps/web/`, DuckDB). Dépôt principal : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration` (bash : `/c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`), branche `feat/v75`, commit de référence `43a01721e`. CE CHECKOUT EST PARTAGÉ avec d'autres sessions : tu n'y écris RIEN (aucune modification de fichier, aucun `git add/commit/stash/checkout/merge/reset`, aucune écriture dans `.ai/`, `data/`, `apps/`). Lecture seule : Read / Grep / Glob, `git log / show / blame / diff`.

L'utilisateur (Guillaume, francophone ; il a fait lui-même la rétro-ingénierie du film Theater et a AUTORITÉ sur la mécanique de jeu) a remonté des défauts du REJEU 2D et de l'UI. Ta mission : établir SUR PIÈCES la cause racine de TON point et proposer une solution planifiable. AUCUNE implémentation.

## Espace de travail
Scratchpad du superviseur (tu peux y écrire) :
- Windows : `C:\Users\GUILLA~1\AppData\Local\Temp\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration\5e1d6f3f-b6be-4233-b361-e8a896dd728d\scratchpad`
- bash : `/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/5e1d6f3f-b6be-4233-b361-e8a896dd728d/scratchpad` (ci-dessous `SP`)
- Rapport final : `SP/rapports/<ton_lot>.md` (en français).
- Scripts d'analyse : Node uniquement (`node x.mjs`) dans `SP/sondes/<ton_lot>/`. JAMAIS de Python.
- Documents de rejeu publiés (schéma 68, JSON, 111 fichiers) : `<repo>/data/cache/replays/halo_infinite/<id8>.json`. Lecture séquentielle autorisée (un document à la fois en mémoire).
- Bases DuckDB : UNIQUEMENT les COPIES `SP/db/shared_matches_v2.duckdb` et `SP/db/metadata.duckdb`, via `SP/diag_q.exe <db> "<sql>"` (lecture seule, sortie TSV). Jamais les bases vivantes de `data/titles/`. Tables append-only : lire les vues `_latest` (ex. `match_kill_events_latest`).
- Faits de film persistés : `<repo>/data/cache/film_facts/halo_infinite/<id8>.filmfacts.bin` (lus par `replay.BuildFromFacts`, `apps/go-api/internal/games/halo_infinite/film/replay/build_from_facts.go:48`, et `filmfacts*.go` ; paquet sans DuckDB). Exemples de tests de recherche : `film/replay/*_research_test.go` (`//go:build research`).
- Scripts Node du superviseur réutilisables : `SP/veh.mjs`, `SP/vsamples.mjs`, `SP/outliers.mjs`, `SP/sweep_vshots.mjs`, `SP/sweep_vw.mjs`, `SP/unk.mjs`, `SP/teams.mjs`, `SP/ctf.mjs`, `SP/ident.mjs`, `SP/ghost.mjs`.
- `.ai/thought_log.md` fait 112 000 lignes : Grep uniquement, jamais de lecture intégrale.

## Horloge affichée du rejeu (vérifiée : `apps/web/src/features/match-replay/model/replayWindow.ts:111-146`)
`affiché_ms = t × frameIntervalMs − (t0FilmMs − originMs)` (plancher 0), donc `t = (affiché_ms + t0FilmMs − originMs) / frameIntervalMs`.
81c02726 et b1ad85eb : décalage 22 700 ms → `t = affiché_s × 10 + 227`. ab526724 : décalage 400 ms → `t = affiché_s × 10 + 4`.

## Interdits (non négociables — sinistres vécus sur cette machine)
1. Aucun serveur (air, :8000), aucun Vite, aucun navigateur (outils MCP chrome-devtools interdits).
2. Aucun décodage de film, aucun `replay-build` / `backfill-*` / `replay-worker`, aucune écriture d'artefact, aucune boucle sur des films ou des faits (bombe RAM : un film BTB = 7,9 Go).
3. Commandes `go` : seulement si INDISPENSABLE à ta preuve, et alors :
   a. dans un worktree DÉTACHÉ à toi : `git -C /c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration worktree add --detach /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-inv-<lot> 43a01721e` — jamais dans le checkout principal ;
   b. `GOCACHE` privé : `GOCACHE=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-inv-<lot>/.gocache` ;
   c. sous le VERROU machine : `bash SP/voie.sh acquire <lot>` AVANT toute commande go (il attend si un autre enquêteur tient la voie ; s'il répond TIMEOUT, relance-le), puis `bash SP/voie.sh release <lot>` APRÈS, même en cas d'échec. Une seule commande go à la fois sur toute la machine ;
   d. sondes = tests `//go:build research` qui lisent AU PLUS 3 fichiers de faits persistés (jamais un film), en avant-plan : `go test -tags research -count=1 -run <TonTest> ./internal/games/halo_infinite/film/replay/` depuis `apps/go-api` du worktree ;
   e. si CGO est exigé : `PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1 CC=C:/msys64/ucrt64/bin/gcc.exe`.
   Laisse le worktree en place à la fin (le superviseur le retire) et donne son chemin dans ton rapport.
4. Aucune commande en arrière-plan (`run_in_background` interdit) : tout en avant-plan.
5. Pas de Workflow, pas de sous-agents, pas d'écriture en mémoire.

## Doctrines du projet que la SOLUTION proposée doit respecter
- La GRAMMAIRE du film prime sur les heuristiques : un seuil empirique n'est acceptable que comme « repli nommé » et compté, jamais comme porte principale ; un négatif doit être MESURÉ (instrument étalonné), jamais supposé. Le film est AUTOPORTANT (jamais de profil par build). Équipe = le film seul (ADR 0034). « Le rejeu se tait plutôt que de deviner », mais l'utilisateur veut les effets quand le film les porte.
- Title-agnostic (capabilities, jamais `slug ==`), pas de feature flag OFF, toute string UI en FR + EN, couleurs = jetons sémantiques, fichiers ≤ 500 L / fonctions ≤ 80 L, tout correctif a un test ROUGE avant / VERT après, toute centralisation a son garde-rail (test grep). Lecteurs = vues `_latest`.
- Toute montée de schéma du document de rejeu implique une republication des artefacts : à SIGNALER comme décision utilisateur, jamais à lancer.

## Méthode
Distingue toujours MESURÉ (fichier:ligne, requête, sortie chiffrée) / DÉDUIT / HYPOTHÈSE. Challenge sur pièces les conclusions des lots antérieurs (thought_log, `.ai/`) : le code a pu bouger, les documents `.ai/` rotent. Si une question exige de la rétro-ingénierie lourde (Ghidra, décodage de film), NE LA LANCE PAS : écris la question, la sonde proposée et son coût. Arrête-toi dès que la cause racine est prouvée ou que les pièces disponibles sont épuisées.

## Format du rapport `SP/rapports/<ton_lot>.md`
1. Constat reproduit sur pièces (chiffres, fichiers, instants affichés ↔ frames).
2. Cause racine prouvée — ou causes candidates classées, avec la sonde qui tranche.
3. Historique (si applicable) : ce que les lots antérieurs ont fait sur ce sujet et pourquoi ça n'a pas suffi.
4. Solution proposée : 2 options max, recommandation, fichiers touchés, montée de schéma oui/non, republication oui/non, tests + garde-rails, gate de vérification SUR DOCUMENTS RÉELS, risques, taille (S/M/L).
5. Questions pour l'utilisateur (seulement ce qui relève de SA décision : mécanique de jeu, rendu, arbitrage produit), formulées pour qu'il réponde en une ligne.
6. Hors périmètre découvert (noté, non traité).

Ta réponse finale (message de fin) : 10 lignes max — chemin du rapport, cause racine en une phrase, recommandation en une phrase, worktree éventuel à retirer.
