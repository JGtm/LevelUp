# HANDOFF — Campagne perf des chargements (23-24 septembre 2026)

> À lire en premier à la reprise de tout travail de performance ou de tout chantier qui touche
> les pages Escouade, Carrière, Synthèse, Sessions, Séries temporelles, Accueil, le sync LUSR v2
> ou les caches de lecture. Rédigé le 2026-09-24 à 00:10 par la session de pilotage de la
> campagne. Sources de vérité : `.ai/PLAN_PERF_CHARGEMENTS_2026-09-23.md` (plan, journaux des
> lots, §10 clôture, §11 découvertes, §12 journal) et
> `.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` (§1 mesures du matin, C1-C9 causes,
> §6 tableau avant / après). Les entrées `.ai/thought_log.md` des 23 et 24 septembre détaillent
> chaque lot et chaque correctif post-fusion.

## 1. État au 2026-09-24, 00:10

| Quoi | État |
|---|---|
| Branche `feat/v75` | HEAD `9d335ea43` (docs d'épilogue), poussée. Contient la campagne (fusion `4693062d5`) et trois correctifs post-fusion (`6d8644181`, `46aec098c`, voir §5). |
| CI de `feat/v75` | VERTE sur `46aec098c` : run push `35922061028` (9 jobs, E2E ignoré par conception) et run pull request `35922071886` (E2E React Playwright `success`) ; gitleaks, Deploy Pre-Check et gate ADR 0021 verts. |
| `main` / prod | PAS touchés. Le merge vers `main` (= déploiement prod automatique) reste le geste de l'utilisateur, dans le cadre de la préparation du déploiement v7.5. |
| Branche de campagne | `feat/perf-chargements` : locale à `1acc33099`, distante à `433375983`, entièrement fusionnée. À supprimer en `-D` (git refuse `-d` à cause du commit de docs non poussé). Branches de lots `feat/perf-l*` et `feat/perf-c4` supprimées localement, jamais poussées. |
| Worktrees | Tous retirés (`LevelUp-wt-perf`, lots, relecteurs, C4). Jonctions `node_modules` supprimées avant chaque retrait, `node_modules` du checkout principal intact. |
| Serveurs / navigateur | Serveurs de mesure et de reproduction E2E arrêtés (ports 8000 et 5173 libres). Instance Chrome du MCP : un onglet `about:blank`, session utilisateur conservée dans le profil. |
| `.env.local` (checkout principal, non versionné) | `LEVELUP_DUCKDB_THREADS=8` et `LEVELUP_DUCKDB_MEMORY_LIMIT=4GB` ajoutés (plan §10 C.4). Effectifs depuis le lot C4 : avant lui ces variables étaient lues à l'init du paquet, avant le chargement de `.env.local`. Prod inchangée (défauts 2 threads / 512 Mo). |
| Scratchpad de la session | `e2e/levelup-api.exe`, `e2e/levelup.exe`, `e2e/demo/` (fixture démo synthétique), journaux de gates : jetables. |

## 2. Résultats mesurés (dev local, JGtm 1 160 matchs, DuckDB 2 threads / 512 Mo, même protocole matin et soir)

| Page / geste | Avant (23/09 matin) | Après (23/09 soir) | Ce qui reste |
|---|---|---|---|
| Escouade, premier passage | 194 s, 4 réponses coupées en 502 puis rejouées | 2,6 s : lecture légère 197 ms puis UNE requête lourde 1,3 s, déjà sur la bonne session | squad_members 276 ms, échange 186, range_profiles 159 |
| Escouade, rechargement / clic du rail | 26,3 s / 8,2 s à vide + 26,8 s | 1,6 s / 0,24 s | |
| Synthèse | 6,2 s | 2,4 s | weapon_records 1,6 s (historique complet) |
| Sessions / Séries temporelles | 6,1 s / 9,5 s | 0,7 s / 0,9 s | |
| Carrière, rencontres et rivaux | 10,7 s et 10,1 s | 1,6 à 4,5 s et 2,1 à 6,5 s (trois passages) ; 0,8 à 2,0 s avec 8 threads / 4 Go | fenêtre `_latest` du kill-feed (Q26, Q27 x2) sur tout l'historique, très sensible à la concurrence des six lectures de la page |
| Accueil | 4,2 s + 2,6 s | 2,1 s | `pages/home` 1,6 s pour 310 Ko |
| Socle (`/filters/resolve`, field-mappings, `/bootstrap`) | 0,2 à 0,6 s, x2-3 par page | 1 à 8 ms après le premier appel | |
| Cycle d'auto-sync (post-sync LUSR) | 1 243 bascules RO/RW en 75 s | 11 bascules par cycle | |

Chronos SQL sur copie : Q29 2 s vers 21 ms ; Q32 2,2 s vers 11-17 ms ; KillEvents 1,8 s vers 59 ms ; rencontres 2,6-3,7 s vers 0,8-1,3 s ; rivaux 6,6-7 s vers 1,9-2,2 s ; top-encounters de bout en bout (résolution des amis) 6,6-19,3 s vers 0,8-1,0 s.

## 3. Ce qui a été fait — 12 lots + C4, un exécuteur Opus par lot, un worktree chacun

| Lot | Mécanisme livré | Garde-rail |
|---|---|---|
| L1 instrumentation | sections de durée par requête (`internal/observability/timing`), ligne `http_timings` (general.log, DEBUG), `LEVELUP_SLOW_REQUEST_MS` | |
| L3 timeouts / annulation | `WriteTimeout` 30 s vers 120 s, 499 `client_closed` et 503 `db_busy` dans `mapServiceError`, `signal` transmis à `fetch` par les hooks, pas de rejeu client sur 502/504, `Touch` de session throttlé | tests handlers + client |
| L4a front Escouade | session pickée = store seul (plus de segment de clé redondant), `enabled` sur la requête lourde, aperçu seulement filtres en attente, résolution solo seulement sous la barre solo (D4.4), tiroir d'assets fermé = aucune requête | tests SquadLayout / route joueur |
| L6 sync LUSR v2 | filigrane lu sur le lecteur avant tout écrivain, zéro écrivain en régime stationnaire | `lusr_watermark_guardrail_test.go` |
| L5a blocs des pages solo | listes de périmètre liées SOUS les vues `_latest` (tactical, weapon_range) | `tactical_repo_fenetres_test.go`, `weapon_range_repo_fenetres_test.go` (EXPLAIN ANALYZE) |
| L5b socle et caches | catalogue de saisons mémoïsé, field-mappings par version, caches filtres et historique (`player_read_cache.go`), instantané `db_profiles` par mtime, highlight-matches par ids, privacy | `archlint/player_read_cache_invalidation_test.go` |
| L2 Escouade backend | annuaire des gamertags sans `v_gamertag_lookup` (`squad_repo_annuaire.go`), un chargement partagé par requête (`pourLaRequete`), `LoadImpactEvents` x1, journal des morts restreint, `sessionMatchIDs` depuis `filters.sessions` | ratchet des lectures de la vue |
| L4b lecture légère | `GET /players/{slug}/pages/teammates/sessions` (Q29, Q30, Q32b), ancrage décidé AVANT la requête lourde | parité avec GetPage (16 scénarios) |
| L7 Carrière | rencontres, rivaux, Q10, Comparer sans la vue des noms (annuaire) | `annuaire_ratchet_test.go` |
| L8 départages | `ORDER BY` totaux Q29 et Q32b, `MapCapabilityError` sur POST teammates | tests d'intégration ex aequo |
| L9-go revue | cache jamais alimenté par un chargement dégradé (P0), rafales LUSR bornées 50 matchs / 2 s, fins de contexte ni mémorisées ni en ERROR (`observability.LevelUnlessCanceled`), matrice d'impact par xuid, lecture légère en erreur sur Q32b illisible, amis de la Carrière par le registre puis UNE lecture, session piquée sans match = aucun match, ordres totaux Q32 / Q32c / Q10, ratchet de la vue sur tout `internal/`, clé du cache testée par réflexion | tests + mutations par item |
| L9-web revue | lien profond gardé au premier ancrage, fraîcheur de la légère à trois conditions, sans coéquipier la lourde attend la légère si session pickée, 503 léger rejoué une fois, `AbortError` hors capture globale, StrictMode expliqué | 27 tests, témoin StrictMode |
| C4 | bornes DuckDB relues à l'ouverture de chaque connexion (`.env.local` honoré) | `TestResourceLimits_EnvSetAfterPackageInitIsHonored` |

Revue adversariale du diff cumulé (quatre relecteurs Opus, lentilles parité / annulation-ART / front / SQL-périmètre) : 1 P0, 3 P1, 12 P2, tous traités par L9-go et L9-web ou consignés au §11.

## 4. Ce qui reste — structurel, pas dans cette campagne

Les coûts restants ont tous la même forme : une lecture qui rejoue TOUT l'historique à chaque requête.

1. Fenêtre `_latest` du kill-feed (`match_kill_events_latest`, ~3,95 M lignes) évaluée par lecture, sur tout l'historique : Carrière (Q26, Q27 x2), 0,8 à 2,2 s par lecture sur copie, 3 à 6 s sous la concurrence de la page à 2 threads.
2. Vue match : la vue des noms évaluée 4 à 5 fois par ouverture (~10 s) ; Relations : 2 lectures. Le pattern annuaire (L2 / L7) s'applique tel quel.
3. Pages à historique complet sans cache : `pages/home` 310 Ko en 1,6 s, weapon_records de la Synthèse 1,6 s.
4. Lectures restantes de `v_gamertag_lookup` figées par le ratchet (Explorer, Médias, classement mondial, killcollector) : coût connu, à retirer avec leur lecture.

Découvertes consignées au §11 du plan (à trier, PAS des lots par défaut) : `logDBError` journalise un `context canceled` en ERROR ; `friends_xp` ouvre en RW la DB d'amis non suivis à chaque bootstrap ; `LEVELUP_REPLAY_PUBLIC` lu à l'init du paquet (même défaut que C4) ; crons de boot (noms d'assets, expansion des playlists) pendant les premières requêtes ; `go test ./...` écrit `data/titles/halo_5/warehouse/metadata.duckdb` et des rasters sous `data/cache/replays` ; badge Touriste non déterministe ; ex aequo Q32 / Q10 ; `FirstSeen` des rencontres jamais posé ; bots dans Q10 ; clé LUSR dupliquée Chocoboflor (`repair_msr_index`).

Hors campagne mais à faire par l'utilisateur : Chocoboflor est en `reauth_required` au boot (refresh token mort), d'où « joueur en échec » à chaque cycle de sync — reconnexion SSO, jamais de re-capture (ADR 0023).

## 5. Les trois rouges post-fusion et ce qu'ils enseignent

1. `archlint.TestAucuneCibleDeRepliNeNommeUnLotClos` lisait `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` à la racine, déplacé sous `.ai/V7.5/` par l'archivage `fe2106f4b` (CI de `feat/v75` déjà rouge avant la fusion). Corrigé `6d8644181`. Leçon : un archivage de docs doit grepper les chemins codés en dur dans les tests.
2. Job Frontend : échec TLS du runner au checkout. Relance, vert. Pas un signal.
3. **E2E Playwright de la PR rouge : régression du lot L4a.** `PlayerLayout` s'abonnait au routeur (`useRouterState`) tout en rendant `<Navigate params={{...}}>` sur un slug inconnu ; `Navigate` re-navigue à chaque changement d'identité de ses props et la navigation re-rend l'abonné : boucle, onglet figé. Reproduit en local sur la démo synthétique de la CI, corrigé `46aec098c` (hooks abonnés dans un enfant `SoloFiltersSync` monté sur slug valide, garde-fou « aucun `useRouterState` sur slug inconnu », mutation rouge). **Piège** : l'E2E ne tourne QUE sur les PR vers `main` (`if: github.event_name == 'pull_request'`), jamais au push `feat/**` — une branche de campagne peut être verte au niveau job avec une régression E2E. Protocole local, à jouer avant toute fusion d'un chantier web :

```
cd apps/go-api && go build -o <scratch>/levelup-api.exe ./cmd/server/ && go build -o <scratch>/levelup.exe ./cmd/levelup/
<scratch>/levelup.exe seed-demo --synthetic --out <scratch>/demo        # LEVELUP_REPO_ROOT = checkout
LEVELUP_DEMO_MODE=true LEVELUP_DEMO_LOCALE=fr LEVELUP_API_PORT=8000 LEVELUP_DEMO_FIXTURES_DIR=<scratch>/demo <scratch>/levelup-api.exe
cd apps/web && VITE_API_PROXY_TARGET=http://127.0.0.1:8000 npm run dev
cd apps/web && CI=true PLAYWRIGHT_BASE_URL=http://localhost:5173 E2E_DEMO_MODE=1 E2E_SYNTHETIC_DEMO=1 npx playwright test --project=chromium --reporter=list --timeout=30000
```

Quatrième leçon, vue à la première CI de la branche : un test Go renommé ou supprimé exige le retrait de sa paire (Package, Test) dans `.ai/baselines/tests_pre_migration.jsonl` dans le même commit, avec un paragraphe daté dans `scripts/check_test_baseline.sh` (le lot L5b l'avait raté ; remède appliqué en `433375983`).

## 6. Recommandation pour la suite (donnée à l'utilisateur le 2026-09-24, en attente de sa décision)

1. Livrer v7.5 telle quelle et mesurer la prod (VPS 2 vCPU) un ou deux jours avec `http_timings` et `LEVELUP_SLOW_REQUEST_MS` avant tout lot structurel.
2. Écrire tout de suite l'ADR « lectures par périmètre » (EN-only, `docs/adr/0036-...`) : toute lecture de page porte un périmètre lié sous les vues `_latest` ou passe par un cache invalidé au sync ; jamais `v_gamertag_lookup` ni une fenêtre `_latest` sur toute une table dans le chemin d'une requête ; un chargement par requête, jamais N. Les garde-rails existants (ratchet de la vue, fenêtres bornées, invalidation des caches, filigrane LUSR) deviennent ses invariants nommés.
3. Un petit plan pour Opus après la mesure prod, dans cet ordre : lot A vue match + Relations par le pattern annuaire (10 s vers moins d'une seconde, risque faible ; le seul lot acceptable AVANT la mesure prod) ; lot B pages à historique complet (périmètre du joueur sous la fenêtre du kill-feed + cache de lecture joueur invalidé au sync pour Carrière / Synthèse, sections paresseuses de l'accueil) ; lot C compaction des tables append-only (recette ADR 0026), seulement sur preuve prod, car il touche aux invariants anti-ART.

## 7. Décisions utilisateur en attente

- Go pour l'ADR (rédaction immédiate) et pour le lot A avant ou après la mesure prod.
- Périmètre du plan structurel (lots B et C), après la mesure prod.
- Reconnexion SSO de Chocoboflor.
- Suppression de la branche locale `feat/perf-chargements` (et de la distante, non demandée).
- Triage des découvertes du §11 (par défaut : pas des lots).

## 8. Règles de pilotage qui ont tenu (à reconduire)

- Un exécuteur Opus explicite par lot, un worktree temporaire chacun (`feat/perf-<lot>` depuis la branche de campagne), GOCACHE privé, jonction `node_modules` seulement si le lot touche `apps/web` ; agents sans serveur, navigateur, push, `git add -A`, `--no-verify`, `git stash`.
- Le superviseur fusionne (`--no-ff`), rejoue les gates sur l'arbre fusionné, écrit les entrées thought_log à partir du texte rendu par l'agent, tient seul le §12 du plan, et vérifie SUR PIÈCES chaque rapport avant de le relayer (deux rapports contenaient des écarts à accepter explicitement).
- Tests de parité et mutations rouges exigés par lot ; revue adversariale à quatre lentilles en fin de campagne seulement.
- Mesure : serveur `air` du worktree d'intégration avec `LEVELUP_REPO_ROOT` sur le checkout principal, `LEVELUP_LOGS_FILE_LEVEL=debug`, instance Chrome du MCP avec la session de l'utilisateur, hors cycle d'auto-sync (toutes les 15 min, post-sync ~60 s), première requête abandonnée par StrictMode exclue des comptes (D9w.6). Détail : plan §8 bis.
- Pièges d'outillage rencontrés : `git show rev:chemin` sous MSYS convertit `:` en `;` (utiliser `MSYS_NO_PATHCONV=1`) ; `git merge -F -` refuse stdin (écrire le message dans un fichier) ; un retrait de worktree suit les jonctions (les supprimer avant, jamais `--force`) ; `go test ./internal/api/...` crée `metadata.duckdb` sous `data/` du worktree.
