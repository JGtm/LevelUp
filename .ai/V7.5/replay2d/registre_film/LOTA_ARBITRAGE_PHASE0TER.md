# Lot A — arbitrage superviseur apres la phase 0-ter (2026-08-18) : GATE 0 PASSE sur l'oracle redefini, phase 1 ouverte

> Lu sur pieces : `LOTA_PHASE0.md` (sections 0-bis et 0-ter), commits `24256e317` et `f41ec9274`,
> TSV de manches sous `lotA/`.

## Ce que la phase 0 (trois passes) etablit

1. **L'en-tete de 5 bits de chaque valeur du statborg est le NUMERO DE MANCHE** (0 = manche 1,
   1 = manche 2, ...) ; la grammaire d'ancrage de production (`objectiveevents/statborg.go`)
   l'imposait a zero et REJETAIT donc toute manche apres la premiere. Corrige dans l'instrument :
   Oddball 4/4 exact (dont un match a TROIS manches, 80+33+42 / 32+80+80 = oracle 192/155),
   frags 385/391, controle negatif propre (2,5 % d'en-tetes non nuls isoles sur 9 films sans manches,
   0 sur `530820e5`). Portee : TOUS les consommateurs de `StatRecords` (score, `NamedEvents`,
   `SlotIdentity`, `CrossCheckNamedEvents`, `Extract` -> `match_objective_events`) ne voyaient
   que la manche 1 d'un match a manches multiples. C'est une correction de PRODUCTION a livrer.
2. **La forme DENSE de liste de composants (`gate=1`, masque R(64)) est ignoree par l'ancrage** :
   33 records perdus sur `24dbb67d`. Seconde correction de production a livrer.
3. **Le film porte le SCORE AFFICHE, l'API `Teams[].Stats.CoreStats.Score` porte AUTRE CHOSE dans
   deux modes** : Strongholds = le nombre de TICKS (emissions - 1 : 193/174/132 exact la ou le film
   dit 200 = plafond = victoire, `outcome`=2), KOTH = des secondes de colline sur 2 films (collines
   sur les 2 autres). Sur les 16 films ou l'API est le score affiche : 16/16 exact, 5 modes sur 5.
4. Identite des equipes 94,1 %, compteurs joueurs 98,5 % (4 modes/5, Oddball 98,5 % au frag pres),
   volume 8,9 Ko median, cout 0,6-2,4 s / <= 21 Mo par film — sauf `1b1e380f` (3,3 Go, tue par le
   plafond) : le decodage doit etre GARDE en production.

## Arbitrage

Le GATE 0 tel qu'ecrit (accord >= 90 % avec `team_0/1_score` sur >= 4 modes) n'est PAS atteint au
sens strict (76,2 %, 3 modes) et aucun seuil n'a bouge. Mais la condition mesurait la JUSTESSE DU
DECODEUR au travers d'un oracle suppose etre le score affiche ; la phase 0-ter prouve, mesure a
l'appui (rampes a la cadence du tick, plafond = victoire, `emissions - 1` = valeur API), que l'API
n'est pas cet oracle en Strongholds et sur 2 KOTH. Sur l'oracle qui est bien le score affiche : 16/16,
5/5. **Decision : GATE 0 PASSE, phase 1 ouverte.** L'ecart avec l'API devient une ligne de QUALITE DE
DONNEES au registre (hors perimetre de ce plan : la sync stocke des ticks/secondes la ou l'app affiche
« score »).

## Phase 1 — contenu tranche (le plan §Lot A phase 1 s'applique, avec ces precisions)

- A.1.0 **Corrections de production dans `objectiveevents/statborg.go`** (source unique D1) : (a)
  en-tete de 5 bits = index de manche, `StatRecord` porte `Round` (0-based) ; (b) forme dense
  `gate=1` lue ; (c) GARDE memoire/volume : plafond de records par film (valeur mesuree x 4 sur le
  corpus, ecrite) au-dela duquel `StatRecords` s'arrete, journalise `slog.WarnContext` et rend ce
  qu'il a (aucune publication partielle silencieuse : `coverage.score.truncated = true`) ; tests
  unitaires sur vecteurs reels (manche 1/2 de `24dbb67d`, un record dense) ; les consommateurs
  existants (`NamedEvents`, `SlotIdentity`, `ScoreCurve`) sont re-verifies : leurs tests verts, et
  UNE ligne de journal dit ce qui change pour eux (les evenements des manches > 1 apparaissent).
- A.1.1 `ScoreTimeline` : par equipe, `rounds: [{round, points: [{t, v}]}]` + `total: [{t, v}]`
  (somme des manches finalisees + courante, monotone), `teamId` par D3 (mesure : `comp 2 A` du slot
  d'equipe = somme des frags du camp, acquis 0-bis) ; par joueur : score personnel, frags, morts,
  assistances (par manche + total) ; `coverage.score = {teamIdentity, rounds, modeSupported,
  truncated, oracle: "displayed"}`. Modes : les 5 sont publies (Oddball inclus).
- A.1.2 `Options.Objectives` en production + retrait d'`originMs` dans `buildObjectiveActions`
  (report `:123`), `PlayerLine` fournies par l'appelant (worker/CLI/admin) — inchange.
- A.1.3 suppression de `filmdec/statborg.go` + `TestParseStatborgRecord_V8` (D1 confirme : la chaine
  ne cadre pas ti=6, 841/841 lectures fausses).
- A.1.4 contrat : `SchemaVersion` 11 -> **12** (la branche integre desormais l'item 6 phase 3 =
  schema 11), `wantReplayDocumentFields` + les nouveaux champs avec leur ligne, OpenAPI
  (`make generate-types`), `generated.ts`, `NULLABLE_ARRAYS`, goldens, temoins re-cuits (`000d5950`
  Slayer, `530820e5` CTF, `24dbb67d` Oddball 2 manches).
- A.1.5 controle apres cuisson : les temoins portent `scoreTimeline` egale a l'oracle affiche
  (16/16 attendu sur les films de la phase 0 ou l'API est le score affiche ; Oddball par manches) ;
  `objectives[]` non vide sur le temoin CTF ; accord >= 95 % avec `match_objective_events` (ms).

Rien de la phase 2 (web) dans ce lot d'execution : gate visuel utilisateur ensuite.
