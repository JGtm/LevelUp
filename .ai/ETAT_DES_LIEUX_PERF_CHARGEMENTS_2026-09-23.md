# Etat des lieux : lenteur des chargements (Escouade en tete) — 2026-09-23

> Investigation sur preuves, sans modification de code. Branche `feat/v75` (43a01721e),
> serveur de dev local (air + Vite), joueur JGtm (1 160 matchs), composition
> XxDaemonGamerxX + Madina97294 + Chocoboflor, poste 16 coeurs / 32 Go.
> Trois enquetes Opus en lecture seule (front, backend Escouade, socle et acces DuckDB)
> + mesures en direct (Chrome DevTools MCP, `logs/http.log` en DEBUG, echantillonnage CPU)
> + chronometrage des requetes SQL sur une COPIE des bases (outil temporaire, supprime).

## 1. Ce qui a ete mesure

### 1.1 Pages, temps serveur (`duration_ms` du middleware HTTP, fichier `logs/http.log`)

| Page | Requete principale | Temps serveur | Reste de la page |
|---|---|---|---|
| Escouade, premier passage (localStorage vide) | POST `/pages/teammates` x7 | 6,0 s ; 44,8 s (502) ; 41,4 s (502) ; 24,4 s ; 53,8 s (502) ; 54,1 s (502) ; 25,3 s | 5 POST `/filters/resolve` de 0,4 a 0,6 s ; **194 s** avant l'etat stable |
| Escouade, rechargement (etat persiste, session de 7 matchs) | POST `/pages/teammates` x1 | **26,3 s** | 3 `/filters/resolve` (0,3 a 0,6 s) |
| Escouade, clic « session precedente » (11 matchs) | POST `/pages/teammates` x2 en parallele | 8,2 s (intermediaire, 0 match) + **26,8 s** | |
| Synthese | POST `/pages/synthesis` | **6,2 s** | 2 `/filters/resolve` 0,4 s |
| Sessions | POST `/pages/sessions/detail` | **6,2 s** | 2 `/filters/resolve` 0,3 a 0,45 s |
| Series temporelles | POST `/pages/timeseries` | **9,5 s** | 2 `/filters/resolve` 0,4 a 0,5 s |
| Carriere | GET `/pages/career` puis 4 GET en cascade | 2,8 s puis `highlight-matches` 2,5 s | 5,6 s jusqu'au dernier appel |
| Accueil (juste apres un boot) | GET `/pages/home`, `season-pass` | 4,2 s et 2,6 s | |
| Explorer | POST `matches-query` | 0,9 s | |

Socle present sur toutes les pages : `/bootstrap` 0,2 a 0,34 s, `/titles/.../field-mappings`
0,35 a 0,41 s **meme en 304**, `/filters/resolve` 0,17 a 0,62 s, deux ou trois fois par page.
Le serveur Vite sert 246 a 411 modules non empaquetes par page (0,3 a 0,5 s en local).

CPU du process `server.exe` pendant une requete : 1,3 a 1,9 coeur (DuckDB limite a 2 threads +
Go) ; 2,8 a 3,0 coeurs quand deux requetes Escouade tournent en parallele. Le travail est
CPU, pas de l'attente disque.

### 1.2 Sequence reelle du premier passage sur Escouade (10:34:13 a 10:37:27)

1. T1 : `pages/teammates` SANS coequipier (la composition n'est pas encore connue, elle
   arrive par `GET /friends`) : 6,0 s, 92 Ko.
2. T2 : composition + tout l'historique : 44,8 s. Le serveur ecrit 4 096 octets puis la
   connexion est coupee par `WriteTimeout: 30 s` (`cmd/server/main.go:1468`) ; Vite renvoie
   **502** ; le front rejoue (retry sur 5xx, `apps/web/src/app/queryClient.ts:15-21`) :
   41,4 s (502 encore), puis 24,4 s (200, 203 Ko). Le calcul serveur des deux requetes
   coupees va jusqu'au bout : le contexte n'est pas annule.
3. Re-ancrage sur la derniere session de la composition (`SquadLayout.tsx:365-412`) :
   DEUX requetes partent en parallele, l'une avec `filters.sessions` seul, l'autre avec
   `picked_squad_session_labels` en plus (deux sources de verite, cle de requete differente) :
   53,8 s et 54,1 s, toutes deux en 502, puis retry : 25,3 s (200, 29 Ko).
4. Etat stable au bout de 194 s. Travail serveur cumule : environ 250 s pour une page.

Pendant tout ce temps le selecteur de session n'existe pas : `SquadFilterBar.tsx:200` ne le
rend que quand `compositionSessions` est arrive, et cette liste vient UNIQUEMENT de la
reponse teammates. C'est le « bloque sur toutes les sessions ».

### 1.3 Ou passent les 26 s d'une requete Escouade (rechargement, 7 matchs)

Horodatage des journaux DEBUG de `GetPage` (`logs/general.log`, 10:44:01 a 10:44:27) :

| Tranche | Duree | Contenu |
|---|---|---|
| debut → `squad_t0_applied` (teammates.07) | 7,5 s | top coequipiers, historique du main, matchs communs x3, allies, map stats, heatmap, matrice d'impact |
| → `teammates.13` | 2,9 s | par minute, radar, intensite |
| → `frag_distribution_built` | 3,0 s | series de performance, frags par arme |
| → `first_blood` | 2,9 s | precision, mecaniques, premier frag |
| → `teammates_echange` + `isolement_nuage` | **8,8 s** | paires d'assistance + journal des morts sur TOUT l'historique, deux fois |
| → `portee_profils` | 1,1 s | roles de portee |
| → reponse | 0,3 s | bandeau, sessions de composition, usage, formes |

Le cout ne depend presque pas du perimetre : 0 match = 8,2 s, 7 matchs = 26,3 s, 11 matchs =
26,8 s, 38 matchs = 24,4 s (requete seule). Le filtre de session ne reduit rien.

### 1.4 Chronometrage SQL sur copie des bases (DuckDB 2 threads / 512 Mo, comme le serveur)

Tailles : `match_kill_events` 3,95 M lignes, `match_lives` 1,72 M, `highlight_events` 1,31 M,
`killer_victim_pairs` 0,54 M, `medals_earned` 0,33 M, `match_participants` 96 636,
`match_registry` 9 170, `v_gamertag_lookup` 53 796 lignes.

| Requete | 2 threads / 512 Mo | 16 threads / 8 Go |
|---|---|---|
| `SELECT count(*) FROM v_gamertag_lookup` (materialisation de la vue) | **3,1 a 3,3 s** | 0,8 a 1,4 s |
| `v_gamertag_lookup WHERE xuid = ?` (un seul xuid : rien n'est pousse) | 3,0 s | 2,0 a 2,2 s |
| Q29 top coequipiers AVEC `LEFT JOIN v_gamertag_lookup` | **3,0 a 3,3 s** | 2,4 a 3,1 s |
| Q29 SANS la jointure | **32 ms** | 67 ms |
| Q32 evenements d'impact, 38 matchs, AVEC jointure | 2,3 a 2,9 s | 2,9 a 3,1 s |
| Q32 SANS la jointure (8 299 lignes) | 8 a 55 ms | 6 a 8 ms |
| Q32b equipe alliee, 38 matchs, AVEC jointure | 2,6 a 2,9 s | 3,2 a 3,5 s |
| `match_kill_events_latest` (count, vue QUALIFY sur 3,95 M) | 0,85 s | 0,6 s |
| Univers Tactique du main (EXISTS par match sur la vue) | 0,7 s | 1,1 s |
| Journal des morts du main, tout l'historique (100 978 lignes) | 0,5 s | |
| Tally frags par arme, tout l'historique | 0,4 s | |
| `highlight_events` tout l'historique (296 746 lignes) | 80 a 90 ms | |
| Base canonique `LoadPlayerMatches` (1 160 lignes, 50 colonnes) | 14 a 53 ms | 21 a 29 ms |
| `player_match_enrichment_latest IN (1 160)` (base joueur) | 14 a 18 ms | |
| `match_skill_rank_latest IN (1 160)` (base joueur) | 14 a 16 ms | |
| `match_csrs_latest IN (1 160)` | 5 a 6 ms | |
| Q4 `mv_player_matches` (filtres, 1 160 lignes) | 9 a 43 ms | 6 a 13 ms |
| Q30 matchs communs avec un coequipier (579 lignes) | 15 a 16 ms | 30 ms |

Lecture : le socle « charger l'historique » est bon marche (environ 100 ms pour les quatre
requetes de `LoadPlayerMatches`). Ce qui coute, c'est la vue `v_gamertag_lookup`, evaluee EN
ENTIER a chaque jointure (elle agrege `match_participants`, deux passes sur
`match_kill_events_latest` et deux sur `killer_victim_pairs` en FULL OUTER JOIN ; aucun filtre
n'est pousse), et les lectures des grandes tables sur tout l'historique. Passer a 16 threads
n'aide PAS les jointures : le probleme est le plan, pas le parallelisme. Le depot avait deja
mesure ce defaut sur un autre lecteur (`sync/killcollector/credit_annuaire.go:12-25` :
77 ms contre 0,9 ms par match, x84).

## 2. Causes, par ordre d'impact

### C1. Escouade : six evaluations de `v_gamertag_lookup` par requete, environ 18 s fixes

Q29 (`LoadTopTeammates`), Q32b (`loadMainTeamAllies`) et Q32 (`LoadImpactEvents`, appele
QUATRE fois sur les memes match_ids : matrice d'impact, intensite, series de performance,
premier frag) portent chacune `LEFT JOIN v_gamertag_lookup` (`platform/duckdb/queries_squad.go`
:29-53, :175-187, :294-309). Six evaluations a 3 s : c'est le socle des 8,2 s « a vide » et de
la partie fixe des 26 s. La page n'a besoin que des xuids et des noms des quatre joueurs
selectionnes, deja connus.

### C2. Escouade : `WriteTimeout` 30 s + retry front = travail triple et pages en 502

`cmd/server/main.go:1467-1469` (`ReadTimeout` 15 s, `WriteTimeout` 30 s). Toute reponse au-dela
de 30 s est tronquee (`response_bytes: 4096` dans `http.log`, puis « superfluous
WriteHeader »), Vite repond 502, le front rejoue deux fois (`queryClient.ts:15-21`), et le
calcul serveur de la requete coupee continue jusqu'au bout. Quatre des sept requetes du premier
passage ont fini ainsi. Aucun `AbortSignal` cote client (`lib/api/client.ts:254-266`) : une
requete devenue inutile n'est jamais annulee.

### C3. Escouade, front : l'ancrage et le selecteur dependent de la reponse la plus lourde

- Le selecteur de sessions n'est rendu qu'avec `compositionSessions` (`SquadFilterBar.tsx:200`),
  qui ne vient que de POST `/pages/teammates` (`SquadLayout.tsx:268`).
- L'ancrage sur la derniere session est decide APRES la reponse (`SquadLayout.tsx:365-412`,
  `decideCompositionReanchor`) : la premiere requete part donc sur tout l'historique, puis le
  snap relance tout.
- Deux sources de verite pour la session (etat local `pickedSquadSessionLabels` + store
  `filterContext.sessions.picked_sessions`, toutes deux dans la cle de requete,
  `lib/query/keys.ts:172-173`) : chaque snap ou clic du rail produit UNE requete intermediaire
  jamais affichee (mesure : 8,2 s + 26,8 s par clic).
- A froid, la composition arrive par `GET /friends` (`SquadLayout.tsx:245-252`) : une premiere
  requete teammates part sans coequipier (6 s pour rien).
- Cote serveur, `sessionMatchIDs` n'est construit que depuis `picked_squad_session_labels`
  (`teammates_service.go:286-291`) : la requete intermediaire (avec `filters.sessions` seul)
  calcule toutes les sections sur 38 matchs au lieu de 7.

Bilan front a froid : 4 POST teammates (2 jamais affiches) et 5 `/filters/resolve`. A chaud :
1 teammates + 3 resolve, mais 26 s quand meme.

### C4. Escouade, backend : 155 a 170 requetes SQL par page, presque toutes sur tout l'historique

Modele de cout etabli sur pieces (agent backend) : 68 requetes fixes + 29 par coequipier suivi.
Pour 3 coequipiers : 21 appels `LoadFor` de l'historique COMPLET d'un membre (heatmap, par
minute, radar, intensite, series de perf, bandeau ; 1+N d'entre eux servent seulement a lire
un xuid deja connu), 4 `LoadImpactEvents` identiques, journal des morts du main sur tout
l'historique deux fois (`teammates_squad_echange.go:101`, `teammates_squad_isolement.go:78`,
alors que `TacticalQuery.Matchs` accepte une liste blanche), blocs usage et formes qui relisent
trois fois les memes tables et rechargent deux TOML par requete, environ 80 relectures de
`db_profiles.json`. Aucun cache : `CachedPlayerMatchesRepo` (TTL 5 min + singleflight,
`platform/duckdb/player_matches_cache.go`) n'est instancie nulle part (grep vide).

### C5. Pages solo : deux blocs communs font l'essentiel des 6 a 9 s

- Synthese (6,2 s) : `LoadPlayerMatches` termine a +0,15 s, la distribution des frags a
  +0,6 s, puis **5,6 s** dans le bloc « records de distance par arme » (`WeaponRangeRepo`,
  deux « side read » a +2,6 s et +6,2 s).
- Sessions (6,1 s) : bloc « coordination » **3,9 s** (journal des morts via
  `TacticalRepository.KillEvents`, `service/coordination_block.go:80`), bloc portee 1,6 s.
- Series temporelles (9,5 s) : portee par arme 0,5 s, coordination 3,2 s, puis 5 s sans
  journal (sections restantes).
- Carriere : `highlight-matches` (2,5 s) charge DEUX fois tout l'historique
  (`handlers/career.go:184-188`) pour enrichir 15 lignes, et le chargement est en cascade de
  trois niveaux cote front.
- Accueil : `pages/home` 4,2 s et `season-pass` 2,6 s (re-mint de token synchrone apres un
  boot, `wire/registry_auth.go:159-172`).

### C6. Socle : `/filters/resolve` x2-3 par page, catalogue de saisons en 403 a chaque appel

- `/filters/resolve` recharge tout l'historique a chaque appel, sans cache serveur
  (`filters_service.go:111`, `filters_repo.go:33-71`). Il est monte par PlayerLayout sur TOUTES
  les pages joueur alors que seules les pages Stats et l'Escouade le lisent, plus un aperçu
  (`useFiltersPreview`) et, sur Escouade, un troisieme pour le store escouade.
- Le catalogue des saisons est vide en base : chaque appel (resolve, `field-mappings`,
  `highlight-matches`) tente un GET Waypoint qui repond 403, non memorise
  (`service/seasons_catalog.go:154-195`) : 38 echecs dans `logs/service.log` sur la seule
  matinee, environ 100 ms a chaque fois.
- `field-mappings` reconstruit tout le DTO AVANT de comparer l'ETag : un 304 coute 0,4 s
  (`handlers/field_mappings.go:170-205`).
- `/bootstrap` bloque le rendu du shell et attend un appel live de confidentialite borne a 2 s,
  mis en cache seulement en cas de succes (`bootstrap_service.go:186, 334-358`).

### C7. Synchronisation : rafales de bascules RO/RW et cache DuckDB froid

Le post-sync LUSR v2 prend un ecrivain partage tous les 3 candidats et ne teste le « deja vu »
qu'apres (`sync/skill/skill_v2_shadow.go:165-183, 270-279`) : **1 243 bascules RO→RW→RO entre
10:32:33 et 10:33:48** aujourd'hui, sans rien ecrire (`general.log`), avec un provider passe
en « error state » a 10:33:44. Chaque bascule ferme le handle RO et vide le cache DuckDB ;
un lecteur HTTP attend jusqu'a 3 s par acces (`middleware/read_budget.go:14`) puis recoit un
500 (pas un 503) et le front rejoue. Toutes les 15 minutes, sur 4 a 5 profils, meme sans
nouveau match (`consecutive_zero_inserts: 1600`).

### C8. Reglages et environnement

- DuckDB `threads=2`, `memory_limit=512MB` (`platform/duckdb/db.go:38-41`) : calibre pour le VPS,
  applique aussi en local. La mesure montre que cela ne change rien aux jointures fautives ;
  ce n'est pas la cause premiere.
- Base joueur : UNE seule connexion, partagee avec le sync (`db.go:286-297`) : toutes les
  lectures d'un joueur se serialisent.
- Session HTTP : reecriture du fichier de session sous verrou global apres CHAQUE requete
  (`middleware/session.go:59-63`, `platform/session/store.go:149-183`), pollings compris.
- Dev : StrictMode double les effets (deux flux de connexion Xbox demarres, 1,1 s chacun) ;
  HTTP/1.1, six connexions ; ajouter un fichier `.go` n'importe ou relance air (le serveur a
  redemarre a 10:48 quand l'outil de mesure a ete ecrit sous `cmd/`).

### C9. Observabilite : rien ne permettait de voir tout cela

Les reponses 2xx ne sont journalisees qu'en DEBUG (`middleware/slog_logger.go:52-56`) et le
niveau fichier par defaut est INFO : aucune duree de requete reussie dans les logs avant cette
session (les seules durees connues etaient celles des 500). Aucun `time.Since` dans les services
Escouade, filtres et repos ; pas de journal des requetes lentes dans `db_query.go` ; pas de
pprof ; `/debug/vars` seulement (pool DuckDB, provider).

## 3. Ce qui N'EST PAS la cause

- Le chargement de l'historique canonique (`LoadPlayerMatches`) : environ 100 ms de SQL.
- Le volume de donnees en soi : 1 160 matchs, reponses de 10 a 200 Ko.
- Le disque ou la memoire : CPU sature sur 2 threads, 300 a 770 Mo de working set.
- Le nombre de threads DuckDB : x2 a x4 sur les agregations, rien sur les jointures fautives.
- Le rendu React : pas mesure comme dominant ici (des defauts existent, cf. rapport front :
  abonnement au store entier, `SessionMultiSelect` quadratique, cinq graphes redessines a
  chaque rendu, echarts charge apres les donnees), mais ils pesent des dizaines de ms face a
  des dizaines de secondes serveur.

## 4. Plan d'attaque propose (par rapport gain / effort)

1. **Instrumenter d'abord** : journaliser toute requete > 1 s en INFO avec `duration_ms` et
   route ; chrono par section dans `GetPage` et dans les pages solo ; compteur de requetes SQL
   par requete HTTP. Une demi-journee, prerequis pour verifier les gains suivants.
2. **Escouade backend, gains surs** : retirer `LEFT JOIN v_gamertag_lookup` de Q29/Q32/Q32b
   (les noms viennent d'un annuaire lu une fois, ou du roster deja connu) : environ -18 s par
   requete ; charger `LoadImpactEvents` une fois ; un `LoadFor` par membre ; passer le perimetre
   au journal des morts (`TacticalQuery.Matchs`) ; construire `sessionMatchIDs` aussi depuis
   `filters.sessions`. Cible : requete a chaud sous 5 s.
3. **Timeout et retry** : `WriteTimeout` route par route (ou 120 s pour les pages lourdes),
   pas de retry automatique sur les POST de page, `AbortSignal` de TanStack passe a `fetch`
   et contexte annule cote Go.
4. **Escouade front** : decider la session AVANT la requete lourde (endpoint leger qui rend
   `composition_sessions` + derniere session, ou reponse en deux temps), une seule source de
   verite pour la session pickee, aucune requete teammates tant que la composition initiale
   n'est pas connue, `keepPreviousData` conserve.
5. **Pages solo** : perimetre pousse en SQL pour les blocs coordination et portee ; cache par
   (xuid, titre) invalide par le post-sync pour `LoadPlayerMatches` et `filters/resolve`
   (le decorateur existe deja) ; memoriser l'echec du catalogue de saisons ; comparer l'ETag
   de `field-mappings` avant de calculer ; `highlight-matches` charge par identifiants.
6. **Sync** : zero ecrivain quand rien n'est nouveau (filigrane lu AVANT de prendre l'ecrivain),
   une rafale par joueur ; 503 plutot que 500 sur `ErrSwapTimeout`.
7. **Reglages locaux** : `LEVELUP_DUCKDB_THREADS` / `LEVELUP_DUCKDB_MEMORY_LIMIT` dans
   `.env.local` (gain modeste mais gratuit), session touchee au plus toutes les N minutes.

## 5. Limites de la mesure

- Serveur de dev local (Vite en proxy, air) ; la prod VPS (nginx, binaire, Linux) n'a pas ete
  mesuree ; les temps serveur y sont probablement du meme ordre ou pires (CPU plus petit).
- Un seul joueur, une seule composition, une matinee ; pas de mesure pendant un cycle de sync.
- Les chronometrages SQL viennent d'une copie des bases lue avec les memes reglages, cache
  DuckDB froid puis chaud (deux executions) ; pas de mesure de la base metadata (verrouillee
  par le serveur pendant la copie).
- Les rapports complets des trois enquetes (cascade front, modele de cout backend, socle et
  DuckDB) sont dans le journal de la session ; les preuves `fichier:ligne` citees ici en sont
  extraites et ont ete verifiees sur pieces pour les points structurants (cache jamais cable,
  reglages DuckDB, rafale de bascules, echecs du catalogue de saisons, ecriture de session,
  timeout serveur, jointure sur la vue).

## 6. Resultats de la campagne (mesure de cloture du 2026-09-23, 17:19-17:22 et 17:45, meme protocole que §1)

Serveur `air` du worktree d'integration (`feat/perf-chargements`), `LEVELUP_REPO_ROOT` sur le
checkout principal (donnees reelles), `LEVELUP_LOGS_FILE_LEVEL=debug`, Vite du worktree,
instance Chrome du MCP avec la session de l'utilisateur, DuckDB `threads=2` / `512MB` inchange.
Durees client = jusqu'a la derniere reponse d'API de la page ; durees serveur = `duration_ms`
du middleware, sections = ligne `http_timings` (lot L1).

| Page / geste | Avant (matin) | Apres (cloture) | Ce qui reste |
|---|---|---|---|
| Escouade, premier passage (localStorage vide) | 194 s, 7 POST teammates dont 4 en 502 | 2,6 s : lecture legere 197 ms puis UNE requete lourde 1 256 ms, deja sur la bonne session | squad_members 276 ms, echange 186, range_profiles 159 |
| Escouade, rechargement | 26,3 s | 1,6 s (legere 133 ms + lourde 858 ms) | |
| Escouade, clic « session precedente » (11 matchs) | 8,2 s a vide + 26,8 s | 0,24 s (une requete de 236 ms) | |
| Synthese, toutes les periodes | 6,2 s | 2,4 s (synthesis 2 098 ms) | weapon_records 1 640 ms (historique complet : plan structurel) |
| Sessions | 6,1 s | 0,7 s (sessions/detail 284 ms) | |
| Series temporelles | 9,5 s | 0,9 s (timeseries 499 ms) | |
| Carriere (page + matchs marquants) | 2,8 s + 2,5 s | career 122-275 ms, highlight-matches 940-1 139 ms | rencontres / rivaux : voir ci-dessous |
| Carriere, rencontres et rivaux (hors sync, 19:47-19:50, apres L9-go, trois passages) | 10,7 s et 10,1 s (non captes le matin) | page complete en 3,0 / 7,1 / 4,6 s ; top-encounters 1 574 / 4 493 / 3 387 ms, rivals 2 105 / 6 472 / 4 090 ms ; annuaire 21-60 ms, plus aucune lecture par ami (amis suivis resolus par le registre, section `career_friends` absente) | la fenetre `_latest` du kill-feed (Q26, Q27 x2) sur tout l'historique : 0,8-2,2 s par lecture sur copie a vide, 3-6 s sous la concurrence de la page a 2 threads / 512 Mo (plan structurel + reglage local C.4) |
| Carriere, memes lectures, serveur relance avec `LEVELUP_DUCKDB_THREADS=8` et `LEVELUP_DUCKDB_MEMORY_LIMIT=4GB` dans l'environnement du processus (reglage local C.4, deux passages 19:57-19:58) | idem | page complete en 1,8 / 2,6 s ; top-encounters 805 / 2 021 ms, rivals 1 118 / 2 031 ms | le meme cout SQL, 2 a 4 fois moins long avec 8 threads et 4 Go : le defaut 2 threads / 512 Mo (calibre pour le VPS) etait la moitie du temps restant sur le poste de dev |
| Accueil (hors sync, 18:34) | 4,2 s + 2,6 s | 2,1 s : pages/home 1 642 ms (310 Ko), season-pass 298 ms ; sections paresseuses ensuite (prestige, series, citations, medias) 1-50 ms | pages/home 1,6 s et 310 Ko de charge utile (historique complet : plan structurel) ; 2,5 s pendant un cycle de sync |
| Socle : /filters/resolve, field-mappings, /bootstrap | 0,2-0,6 s a chaque appel, x2-3 par page | 1 a 8 ms apres le premier appel du process (caches L5b) | |
| Cycle d'auto-sync (post-sync LUSR) | 1 243 bascules RO/RW en 75 s | 11 bascules sur le cycle de 16 h (4 joueurs « rien de nouveau », 1 rafale) | |
| Reponses tronquees (502) et rejeux | 4 sur 7 requetes Escouade | 0 ; requetes abandonnees en 499 (5 sur la fenetre) | |

Chronos SQL sur copie (2 threads / 512 Mo) : Q29 1,7-2,4 s -> 21 ms ; Q32 (38 matchs) 2,1-2,4 s
-> 11-17 ms ; Q32b 2,3 s -> 6-7 ms ; KillEvents 6 matchs 1,79 s -> 59 ms ; MortsAvecContexte
3,26 s -> 80 ms ; LoadWeaponRange 1,32 s -> 64 ms ; rencontres (Q26) 2,6-3,7 s -> 0,8-1,3 s ;
rivaux (Q27 x2) 6,6-7,0 s -> 1,9-2,2 s ; Q10 et Comparer 2,3-2,6 s -> 9-30 ms ; lecture
legere des sessions de composition 45-197 ms.

Verification : 12 lots (L1, L3, L4a, L6, L5a, L5b, L2, L4b, L7, L8, L9-go, L9-web) et le lot de
cloture C4, chacun avec tests de parite et mutations ; gates complets sur l'arbre fusionne le
2026-09-23 au soir (gofmt vide, build, vet, `go test ./...` 190 paquets sans echec,
integration `-p 1` sync/persist/duckdb/migration 18 paquets sans echec en 13 min, golangci-lint 0 issue nouvelle par
lot, `tsc -b --force` 0, eslint 0 erreur, vitest 791 fichiers / 8 503 tests) ; revue
adversariale a quatre lentilles sur le diff cumule (1 P0, 3 P1, 12 P2, tous corriges par L9-go
et L9-web ou consignes au plan) ; CI de branche lancee au push de cloture, verdict consigne au §12 du plan.

Caveats : mesure sur le serveur de dev (Vite HTTP/1.1, StrictMode : la premiere requete de
chaque page est annulee a ~20 ms puis rejouee, propre au dev) ; un cycle d'auto-sync a tourne
pendant la premiere mesure de Carriere et Accueil (attente du lecteur partage pendant la phase
d'ecriture : 5,7 s hors sections sur top-encounters) ; la prod (VPS, moins de coeurs) reste a
mesurer apres deploiement, les lignes `http_timings` le permettent.
