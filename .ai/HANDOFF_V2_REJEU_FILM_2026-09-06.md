# Handoff — chantier v2 rejeu/film — état au 2026-09-06 (soir)

> À relire APRÈS compaction de contexte, avant toute action. Source de vérité de l'avancement :
> `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md` (journal du plan, fin de fichier) ; mémoire :
> `memory/project_v2_rejeu_film_chantier.md`. Le user (Guillaume) est présent, quota surveillé,
> contexte de l'assistant presque plein : messages courts, appels d'outils au strict nécessaire.

## 1. Où en est la branche

- `feat/v75` = `5e7704428` (+ éventuels commits ultérieurs : `git log --oneline -5`). TOUT est
  intégré : sept lots (B, F, C, G, E, A, D), correctif du pont d'identité (`d173b1a8c` → schéma 40),
  outil `cmd/replay-diff` + rapport `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`, correctif
  « une piste = une vie » (`48cf4905d` → schéma 41, `UsageSummaryRev` us2). CI verte au niveau
  job jusqu'au merge `b696c7b11` (schéma 41). Notion (Backlog LevelUp, « Séquence à dérouler à
  la release ») : re-cuisson du parc au schéma **41** (à porter à 42 si le point 2 aboutit).
- Verdict du balayage final : le parc peut être re-cuit au schéma 41 sans rien perdre ; trois
  passes, zéro perte nouvelle ; quinze faits résiduels antérieurs au chantier (dont `d9781168`
  −6 portages de crâne d'Oddball).
- Dix worktrees `LevelUp-wt-v2-*` retirés, branches fusionnées supprimées, caches purgés.
  Incident : le retrait a vidé `apps/web/node_modules` du principal (jonctions non retirées) →
  réparé par `npm ci` (370 entrées, typecheck 0). Mémoire durcie
  (`reference_worktree_remove_follows_junctions.md`).

## 2. Agents en cours (identifiants pour SendMessage)

| Agent | ID | Worktree / branche | Livrable attendu |
|---|---|---|---|
| Complément identité des ports de drapeau (`flagCarries`) | `ae58e9a0d3e519fb1` | `LevelUp-wt-v2-flagcarries` / `feat/v2-flagcarries` (base `5e7704428`) | complément dans `replaybuild` (pas dans `analysis/replay`), SchemaVersion 41 → 42, tests par mutation, témoins, journal `.ai/V7.5/v2/FLAGCARRIES_COMPLEMENT_2026-09-06.md`, push, rapport |
| Instruction des quinze faits résiduels | `a77fcfb0cbaa9cb92` | `LevelUp-wt-v2-residus` / `feat/v2-residus` (à créer par l'agent, base `5e7704428`) | verdict par fait, correctifs éventuels (bump 43 si la sortie change), journal `.ai/V7.5/v2/INSTRUCTION_RESIDUS_2026-09-06.md`, push, rapport |

Les agents notifient à la fin ; ne jamais lire leur fichier `.output` (transcript complet).
Un agent muet ou tué par la limite de session se relance par SendMessage sur son ID
(« constate l'état réel du worktree : `git status`, `git log -3`, journal ; reprends à la
première étape non close »).

## 3. Ce qui reste, dans l'ordre

1. À réception du rapport `flagcarries` : vérifier sur pièces (diff stat, tests cités), lancer
   UNE revue adversariale (Opus, contexte frais, lentilles L4 + L6, cuissons témoins ≤ 3, un film
   à la fois, contrat en 6 lignes), corrections par l'exécuteur, ronde 2 sur les corrections
   seulement, puis merge `--no-ff` dans `feat/v75` (résoudre `.ai/thought_log.md` par
   concaténation : `sed -i '/^<<<<<<< /d; /^=======$/d; /^>>>>>>> /d'`), rejouer `go build ./...`,
   `go test` des paquets touchés + `-tags=integration -p 1 ./internal/api/wire/...`,
   `make generate-types` + `git diff --exit-code apps/web/src/lib/api/generated.ts`, push,
   `gh run list --branch feat/v75 --workflow CI --limit 3`. Puis Notion : « schéma 41 » → 42.
2. Idem pour `residus` (revue seulement s'il y a du code ; sinon merge du journal).
3. Facultatif mais recommandé : balayage final au dernier schéma (même méthode, agent frais :
   worktree de balayage, `cmd/replay-diff`, racine de travail à jonctions vers
   `data/cache/film_chunks` et `data/titles` du principal, faits transmis, un film à la fois,
   ~35 min, sortie dans le scratchpad, jamais dans le parc).
4. Nettoyage : retirer les worktrees `flagcarries` et `residus` UNIQUEMENT après avoir supprimé
   les jonctions `node_modules` s'il y en a (ces deux-là n'en ont pas : Go seul ; mais
   `make generate-types` a besoin de `apps/web/node_modules` du PRINCIPAL, qui existe) — vérifier
   `Test-Path` avant `git worktree remove`, ABORTER si une jonction subsiste ; supprimer
   `%LocalAppData%\go-build-v2-flagcarries`, `go-build-v2-residus`, `golangci-v2-*`.
5. Tag v7.5.0 : séquence Notion (prévenir le user avant tout push sur `main` = déploiement prod).
6. Le scratchpad `scratchpad/balayage/` (copies d'artefacts de référence, cuissons `apres*/`)
   peut être supprimé à la fin.

## 4. Règles apprises cette semaine (ne pas réapprendre)

- Un relecteur par worktree (deux se polluent par leurs mutations).
- Conflits sémantiques entre lots parallèles attrapés par le hook `go-vet-cgo` au push et par la
  CI (types `replay` vs `replaydoc`, chemins déplacés, baseline de présence) : rejouer les tests
  du paquet fusionné AVANT de pousser ; entrées de baseline orphelines → les retirer avec
  justification.
- `golangci-lint` : cache GLOBAL → `GOLANGCI_LINT_CACHE` par worktree ; verrou global
  (« parallel golangci-lint is running ») → réessayer.
- Cache `go-build` principal : 69 Go, à vider (`go clean -cache`) quand le disque sature ;
  `GOCACHE` par worktree pour les agents parallèles.
- Jamais deux cuissons en parallèle ; exécuteur borné (3 Gio) ; pic mesuré 0,56 Gio par film.
- Ne jamais écrire « fusionné » au journal avant `git log` (une fusion refusée en silence a
  failli passer pour faite).
- Chaque bump de `SchemaVersion` force la re-cuisson (règle du dépôt) : bumper dès que le
  contenu cuit change.

## 5. Décisions user fermes (05-06/09)

1-6 (05/09) : positions par persist ; document servi séparé maintenant ; `film.replay_artifact`
gouverne la production + `CapReplay` ; R5 web en lignes de code ; déplacement du décodeur sous
`games/halo_infinite/film/` APRÈS v7.5.0 (Notion) ; « heatmap » banni, « lobby » assimilé.
7-11 (06/09) : CTF grave, instruit (fait) ; append-only + grain 20 s confirmés ; transport et
tiroir : ON NE TOUCHE PAS ; table ECS corrigée prudemment (fait, E.9) ; mécanismes de resync
gardés. 12 (06/09) : `flagCarries` à compléter dans `replaybuild` (en cours) ; quinze faits
résiduels à instruire (en cours) ; nettoyage OK (fait).
