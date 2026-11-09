# Plan : briefing de l'Explorateur, cartes qui disparaissent selon les filtres

> Rédigé le 2026-10-26 après relevé sur pièces (code + copie des bases de JGtm, 1 181 matchs).
> Recommandations validées par le user le 2026-10-26 (« ok avec tes recos »).
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, statut de chaque
> item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture, zéro fix hors périmètre).
> Branche : `feat/briefing-seuils`, créée depuis `release/v8.0.7` dans un worktree dédié (la copie
> principale porte le travail en cours de `fix/medailles-vide-legitime` : ne pas changer de branche).
> Effort : moyen (Go service + contrat + composants du briefing ; aucune requête SQL nouvelle).
> Exécution : agent Sonnet dans `../LevelUp-wt-briefing-seuils`, piloté ; aucun commit sans
> accord du user (CLAUDE.md règle 16, prime sur la règle 8 du contrat) ; cache Go dédié
> `%LOCALAPPDATA%\go-build-briefing-seuils` (une autre session compile en parallèle).

## Constat (résumé du relevé)

Aucune absence n'est un bug : chacune suit une règle écrite. Mais les seuils fixes à 10 matchs
vident le bandeau dès qu'un filtre resserre :

1. « Par carte » : un groupe n'entre qu'à 10 matchs. CTF (138 matchs, 37 cartes), Bases (66),
   Roi de la colline (39), 30 derniers jours (22) : aucune carte qualifiée, la carte saute.
2. « Par sélection » : 97 % des matchs en Partie rapide ; 10 filtres sur 20 la réduisent à une
   seule ligne « Quick Play ».
3. « Par contexte » : exige 10 matchs en solo ET 10 en escouade.
4. Sous 10 matchs (`LowSample`), le service sort avant de calculer Classement, Arme, Séries,
   Moments forts, et le front masque Durée, Séries, Pic rang, Pic MMR. « PVP classé » (8 matchs)
   n'a donc aucune progression de rang.
5. Omission pure, contraire à la règle du user du 2026-10-17 (une section qui disparaît faute de
   données se lit comme un bug : bloc affiché + état vide en pointillé).

Matrice complète : artefact https://claude.ai/artifact/QNTajuSHMeAwyG7hxWABke

## Objectif et critère de succès

Le bandeau n'escamote plus une carte faute de matchs : il montre une comparaison quand elle est
possible, un état vide expliqué quand le filtre est trop étroit, et rien seulement quand l'axe est
celui du filtre (ou n'a qu'une valeur). Sous 10 matchs, les blocs qui ne comparent rien restent.

Critère : les 8 filtres du gate visuel (E4) affichent exactement l'état attendu du tableau E4.

## Décisions (tranchées)

- **D1 — Seuil par groupe des cartes « Par… » : 10 → 5 matchs** (`MinDimensionGroupMatches`).
  Mesuré : CTF 0 → 14 cartes éligibles, Assassin en équipe 2 → 27, année 2025 2 → 26.
- **D2 — Seuil de « Par contexte » aligné sur D1 : 5 de chaque côté** (`minContextSplitMatches`,
  déjà documenté comme « aligné MinDimensionGroupMatches »). Confirmé par le user le 2026-10-26.
- **D3 — Une carte « Par… » compare au moins deux groupes.** Au moins 2 valeurs dans le filtre
  mais moins de 2 groupes à D1 matchs → carte affichée en ÉTAT VIDE (titre + pointillé), plus de
  carte à une ligne.
- **D4 — Absence légitime conservée.** Moins de 2 valeurs dans le filtre (axe filtré, ou une seule
  valeur dans les données) → carte absente. « Par contexte » : un côté à 0 match → absente.
- **D5 — « Par contexte » sous le seuil** (les deux côtés présents, l'un < D2) → état vide qui
  donne les deux comptes (« 3 en solo, 63 en escouade : il en faut 5 de chaque côté »).
- **D6 — Échantillon faible (< 10, seuil inchangé) : seul ce qui compare est masqué.** Restent :
  socle complet (Durée, Séries, Pic rang, Pic MMR), Classement, Arme favorite, Moments forts.
  Masqués : écarts « vs habituel » (Baseline), cartes « Par… », « Par contexte » — sans état vide
  (la mention « Échantillon faible (N matchs) » existante reste l'explication unique).
- **D7 — Seuil des chaînes du Classement inchangé (10).** Il est aujourd'hui couplé à D1
  (`MinRankedChainMatches = MinDimensionGroupMatches`) : découplé en littéral nommé propre.

## Étapes

### E0 — Mise en place
- [x] Worktree dédié sur `feat/briefing-seuils` depuis `release/v8.0.7` ; `git log --oneline -3`
- [x] Ligne de base verte (env CGO : `C:\msys64\ucrt64\bin` dans le PATH, `CGO_ENABLED=1`,
      `CC=C:\msys64\ucrt64\bin\gcc.exe`) :
      `go test ./internal/service/ -run 'Briefing'` et
      `npx vitest run src/features/explorer` (depuis `apps/web`)
- Gate : les deux commandes vertes, résultat noté dans « Mesures ».

### E1 — Service Go et types du briefing
- [x] `match_history_service_briefing.go` fait 494 lignes : extraire AVANT toute modification
      `buildBriefingDimensions`, `buildDimension`, `selectTopFlop`, `mapKeyed`, `modeKeyed`,
      `playlistKeyed`, `rawRowsToBreakdownRows`, `rawOutcomeToCanonical` vers
      `match_history_service_briefing_dimensions.go` (déplacement pur, sans autre changement)
- [x] Constantes : `MinDimensionGroupMatches = 5` ; `MinRankedChainMatches = 10` en littéral
      nommé avec son propre commentaire (D7) ; `minContextSplitMatches = MinDimensionGroupMatches`
      (D2) ; commentaires des trois mis à jour (plus de « aligné sur 10 »)
- [x] `domain/explorer_briefing.go` :
      `ExplorerBriefingDimension.Insufficient bool json:"insufficient,omitempty"` ;
      `ExplorerBriefingContextSplit.Insufficient bool json:"insufficient,omitempty"` ;
      `ExplorerBriefing.MinGroupMatches int json:"min_group_matches,omitempty"` (seuil servi au
      front pour ses libellés, posé hors échantillon faible) ; doc du paquet et des champs
      réécrite (supprimer « dégradation par omission, jamais de N/A affiché » ; décrire D3-D6)
- [x] `buildDimension` : `distinct < 2` → nil (D4) ; moins de 2 groupes qualifiés →
      `{Dimension, Entries: []…{}, Insufficient: true}` (slice vide, jamais nil : `entries` n'a
      pas `omitempty`) ; sinon inchangé (tri P-8 en plein historique conservé)
- [x] `buildBriefingContextSplit` : un côté à 0 → nil (D4) ; un côté < seuil →
      `Insufficient: true` avec `Solo.Matches` / `Squad.Matches` remplis (D5) ; sinon inchangé
- [x] `buildExplorerBriefing` : plus de retour anticipé sur `LowSample`. Toujours calculés :
      Ranked (si `rankedCapable`), Weapons, Streaks, Dominance. Seulement hors `LowSample` :
      Baseline, Dimensions, ContextSplit, `MinGroupMatches` (D6). Commentaire de la fonction mis
      à jour
- [x] Tests service (fichiers `match_history_service_briefing*_test.go`, aucun nouveau fichier
      sauf si un fichier dépasse 500 lignes) :
  - [x] dimension : 2 groupes à 5 → entrées ; groupes à 4 et 5 → `Insufficient` ; aucun groupe
        qualifié → `Insufficient` ; une seule valeur → nil ; `entries` sérialisé `[]`
  - [x] contexte : 5/5 → normal ; 3/63 → `Insufficient` avec comptes ; 0/10 → nil
  - [x] échantillon faible (8 matchs notés, avec frags d'arme et un drapeau de domination) :
        Ranked, Weapons, Streaks, Dominance non nil ; Baseline, Dimensions, ContextSplit nil ;
        `MinGroupMatches` = 0
  - [x] Classement : une chaîne à 7 matchs à côté d'une à 12 → seule celle à 12 (D7 tient)
  - [x] tests existants adaptés au seuil 5 un par un (chaque assertion modifiée citée dans le
        message de commit ; aucune suppression de test sans `[!]` justifié)
- Gate : `go test ./internal/service/ ./internal/domain/` vert ;
  `wc -l internal/service/match_history_service_briefing*.go` : aucun fichier non-test > 500.

### E2 — Contrat HTTP
- [x] `make openapi-gen` puis `make generate-types` ; diff de `openapi.yaml` limité aux trois
      champs ajoutés
- [x] `internal/api/handlers/explorer_briefing_test.go` : un cas qui sert un briefing avec une
      dimension `insufficient` et vérifie la présence de `insufficient` et `min_group_matches`
      dans le JSON
- Gate : `go test ./internal/api/handlers/ -run Explorer` vert ; `make openapi-check` vert ;
  `make check-types` vert.

### E3 — Web
- [x] i18n `apps/web/src/lib/i18n/manifests/explorer.toml`, FR + EN, sans anglicisme :
  - `explorer.briefing.dim_insufficient_map` — fr « Moins de deux cartes comptent au moins {n}
    matchs sur ce filtre : pas de comparaison possible. »
  - `explorer.briefing.dim_insufficient_mode` — même phrase, « modes »
  - `explorer.briefing.dim_insufficient_playlist` — même phrase, « sélections »
  - `explorer.briefing.context_insufficient` — fr « {solo} en solo, {squad} en escouade : il faut
    au moins {n} matchs de chaque côté pour comparer. »
  - EN rédigé en parallèle ; puis `node apps/web/scripts/build_i18n_manifests.mjs`
- [x] `ExplorerBriefingModules.tsx` : `DimensionCard` et `ContextSplitCard` rendent, quand
      `insufficient`, la `BriefingSectionCard` (même titre + (i)) avec `EmptyStateNotice`
      (description seule, aucune action) DEDANS ; `{n}` = `briefing.min_group_matches`
- [x] `favoriteWeaponSlots` : une carte en état vide compte pour un nombre de lignes nommé
      (constante `INSUFFICIENT_CARD_LINES = 2`, commentée) ; test unitaire ajouté dans
      `ExplorerBriefing.logic.test.ts`
- [x] `ExplorerBriefingStrip.tsx` : en échantillon faible, Durée et tuiles conditionnelles rendues
      (seuls les écarts « vs habituel » disparaissent, déjà conditionnés à `baseline`) ;
      `ExplorerBriefingModules` rendu aussi en échantillon faible, la mention « Échantillon
      faible (N matchs) » conservée au-dessus des modules
- [x] En-têtes de fichier de `ExplorerBriefingStrip.tsx` et `ExplorerBriefingModules.tsx` réécrits
      (ils décrivent l'omission et « en low_sample : seules Matchs / Taux / FDA / Perf »)
- [x] Tests vitest (`ExplorerBriefingStrip.test.tsx`) :
  - [x] le test « omet Durée totale en low_sample » devient « rend Durée totale en échantillon
        faible » (renommage = retrait de l'ancien nom de la baseline JSONL, cf. mémoire)
  - [x] échantillon faible : Classement, Arme favorite, Moments forts rendus ; aucune carte
        « Par… » ; mention présente
  - [x] dimension `insufficient` : titre « Par carte » + texte d'état vide avec le seuil
  - [x] contexte `insufficient` : texte avec les deux comptes
- Gate (depuis `apps/web`) : `npx vitest run src/features/explorer src/lib/api` vert ;
  `make check-types` vert ; `npx eslint src/features/explorer` sans nouvelle erreur.

### E4 — Gate visuel par le user
- [~] Redémarrer le serveur air local (Start-Process détaché, port 8000) depuis le worktree — sans objet : vérifié sur les serveurs déjà lancés par une autre session sur `release/v8.1.0` (contient 1b0154a8c), en lecture seule
- [x] Remettre la main au user avec la liste ci-dessous (page Explorer de JGtm, mode Matchs) ; — à la demande du user (2026-10-26), vérification faite par le pilote dans Chrome MCP après connexion du user ; relevé dans « Mesures »
      attendus calculés sur la copie du 2026-10-26 avec D1-D6 (± 1 match possible) :

| Filtre | Par carte | Par sélection | Par mode | Par contexte | Autres |
|---|---|---|---|---|---|
| Aucun | 6 lignes | 4 lignes | 6 lignes | affichée | tout |
| Mode CTF | 6 lignes | état vide | absente | affichée | tout |
| Mode Bases | 2 lignes | état vide | absente | état vide (3 / 63) | tout |
| Mode Roi de la colline | état vide | état vide | absente | affichée | tout |
| 30 derniers jours | état vide | état vide | 2 lignes | affichée (5 / 17) | tout |
| Sélection Big Team Battle (11) | état vide | absente | état vide | affichée (6 / 5) | tout |
| Carte Illusion | absente | absente | 4 lignes | affichée | tout |
| PVP classé (8) | rangée absente, mention « Échantillon faible (8 matchs) » | | | | Durée, Séries, Pic rang, Classement, Arme, Moments forts présents |

- Gate : retour du user ; tout écart est soit corrigé dans E3 (formule ou rendu), soit consigné
  `[!]` avec sa décision. Point d'attention à lui signaler : « Par sélection » sera souvent en
  état vide (Partie rapide quasi exclusive) ; s'il le juge bruyant, décision neuve à prendre,
  pas d'ajustement silencieux.

### E5 — Clôture
- [~] `make go-api-lint` (nouveau code 0 issue) — couvert par `make gate-push` (golangci `--new-from-merge-base`, 0 issue)
- [x] `make gate-push` — vert sur 1b0154a8c (golangci 0 issue, typecheck, lint 0 erreur, baseline 21 703 tests) ; CI 40370415086 verte
- [x] `.ai/thought_log.md` (entrée « Complété hors gate visuel ») ; aucun `[!]` à reporter
- [x] Rebaser sur `main` : v8.0.7 est fusionnée par squash (`main` 413971bcc, 2026-10-26) et
      `release/v8.0.7` est close ; la cible devient `release/v8.0.8` (créée depuis `main`).
      Après commit sur `feat/briefing-seuils` :
      `git rebase --onto main 06531b561 feat/briefing-seuils`, puis regates E1-E3
- [x] Commit et fusion demandés par le user le 2026-10-26 : « merge ton travail dans release/v8.1.0, je ferais mes vérifications là » (gate visuel E4 sur cette branche)
- [x] `powershell -File scripts/disk-hygiene.ps1 -Lot briefing-seuils -Apply` après fusion — worktree et branche supprimés ; cache golangci supprimé ; `go-build-briefing-seuils` refusé (écrit < 10 min), laissé à la tâche planifiée (48 h)

## Mesures

Relevé du 2026-10-26 (copie de `stats.duckdb` JGtm + `shared_matches_v2.duckdb`, règles du code
rejouées en SQL ; libellés de mode normalisés à la main) :

| Filtre | Matchs | Solo / escouade | Cartes (≥10 / ≥5) | Modes (≥10 / ≥5) | Sélections (≥10 / ≥5) |
|---|---|---|---|---|---|
| Aucun | 1 181 | 560 / 621 | 71 (45 / 50) | 22 (6 / 8) | 9 (2 / 4) |
| Année 2025 | 249 | 57 / 192 | 61 (2 / 26) | 15 (5 / 6) | 4 (1 / 2) |
| Assassin en équipe | 247 | 33 / 214 | 47 (2 / 27) | 1 | 1 |
| CTF | 138 | 29 / 109 | 37 (0 / 14) | 1 | 2 (1 / 1) |
| Bases | 66 | 3 / 63 | 28 (0 / 2) | 1 | 2 (1 / 1) |
| Illusion | 56 | 34 / 22 | 1 | 5 (1 / 4) | 1 |
| Roi de la colline | 39 | 23 / 16 | 27 (0 / 0) | 1 | 2 (1 / 1) |
| 30 derniers jours | 22 | 5 / 17 | 20 (0 / 0) | 4 (1 / 2) | 2 (1 / 1) |
| Big Team Battle | 11 | 6 / 5 | 9 (0 / 0) | 7 (0 / 0) | 1 |
| PVP classé | 8 | 0 / 8 | 5 | 2 | 2 |

E0 (2026-10-26) : ligne de base verte avant changement — `go test ./internal/service/ -run Briefing`
ok ; vitest `src/features/explorer` 31 fichiers, 262 tests.

Clôture E1-E3 (agent Sonnet, gates relancés par le pilote) : `go test ./internal/service/
./internal/domain/` ok ; `go test ./internal/api/handlers/ -run Explorer` ok ; `make openapi-check`
et `make check-types` verts ; vitest `src/features/explorer src/lib/i18n` 41 fichiers, 335 tests.
Écarts au plan, statués :
- [~] tests de seuils dans deux NOUVEAUX fichiers (`match_history_service_briefing_thresholds_test.go`,
  `ExplorerBriefingStrip.thresholds.test.tsx`) : les fichiers d'origine dépassaient 500 lignes.
- [~] état vide : titre commun `explorer.briefing.insufficient_title` (« Pas assez de matchs pour
  comparer ») + description par dimension ; `EmptyStateNotice` (title requis) non modifié.
- [~] aucun test renommé ne figurait dans une baseline (grep du dépôt) : rien à retirer.
- golangci : aucune issue sur les fichiers touchés ; `--new-from-merge-base` non lancé (autre
  golangci actif) — repris en E5 par `make gate-push`.

E4 (2026-10-26, Chrome MCP, JGtm, serveurs sur `release/v8.1.0` incluant 1b0154a8c) — réponse de
`matches-query` (`include_briefing`) par filtre, puis rendu vérifié à l'écran sur Bases et PVP classé :

| Filtre | Matchs | Par carte | Par sélection | Par mode | Par contexte | Autres | Écart |
|---|---|---|---|---|---|---|---|
| Aucun | 1 181 | 6 | 4 | 6 | 560 / 621 | tout | aucun |
| Capture du drapeau | 134 | 6 | état vide | absente | 24 / 110 | tout | aucun (138 attendu : libellés de mode approchés) |
| Bases | 67 | 2 | état vide | absente | état vide 3 / 64 | tout | aucun |
| Roi de la colline | 39 | état vide | état vide | absente | 23 / 16 | tout | aucun |
| 30 derniers jours | 22 | état vide | état vide | 2 | 5 / 17 | tout | aucun |
| Grand combat en équipe | 11 | état vide | absente | état vide | 6 / 5 | tout | aucun |
| Illusion | 56 | absente | absente | 5 | 34 / 22 | tout | 5 lignes au lieu de 4 (libellés de mode approchés dans la mesure) |
| PVP classé | 8 | rangée absente + « Échantillon faible (8 matchs) » | | | | Durée, Séries, Pic rang, Pic MMR, Classement, Arme, Moments forts | aucun |

Rendu : états vides dans leur carte (titre « Pas assez de matchs pour comparer » + texte), « Par
contexte » vide empilé avec l'arme favorite à hauteur des voisines (pas de réglage de
`favoriteWeaponSlots` nécessaire).

## Découvertes (hors périmètre, non traitées)

- `favoriteWeaponSlots` ne compte pas la hauteur d'un « Par contexte » en état vide empilé avec
  l'arme favorite : jugé en E4, hauteur correcte, rien à régler.
- `ExplorerBriefingStrip.test.tsx` est au plafond `max-lines` : tout nouveau test du bandeau va
  dans un fichier voisin.
- Socle à 8 tuiles sur un écran d'environ 1 250 px : « Pic MMR » passe seul à la ligne et s'étire
  sur toute la largeur (vu sur Bases, déjà le cas avant ce lot ; désormais visible aussi en
  échantillon faible). Décision de mise en page à prendre avec le user.
- Classement en « PVP classé » (8 matchs de placement sur deux saisons) : « Placement → Placement
  (4 matchs restants) · ±0.0 pt/match ». Donnée exacte mais peu parlante ; non traité.

- `PLAN_CARTES_VARIANTES_STATS_2026-10-26.md` (E2) modifie aussi `buildBriefingDimensions`
  (regroupement des variantes avant `breakdown.ByMap`). Pas de conflit fonctionnel ; si les deux
  branches avancent ensemble, la seconde fusionnée rebase sur le fichier extrait en E1.
- Les écarts « vs habituel » des lignes de dimension et du socle restent masqués en échantillon
  faible (D6) ; non rouvert.

## Amendement du 2026-10-26 (retours du user après E4)

Branche `feat/briefing-socle-equilibre`, agent Sonnet piloté, vérifié dans Chrome.
- [x] Socle équilibré : grille à colonnes égales par container queries (150 px mini, gap 8) ; une
      ligne si tout tient, sinon deux lignes équilibrées (8 → 4+4, 7 → 4+3, 6 → 3+3, 5 → 3+2),
      plancher 2 colonnes. Table littérale `SOCLE_GRID_CLASSES` recalculée par un test. Vérifié :
      1 600 px → 8 sur une ligne (socle plafonné à 1 272 px) ; 1 250 px → 4+4 ; 500 px → 2 colonnes.
- [x] « Séries marquantes » → « Séries » (intitulé sur deux lignes à 152 px) ; l'infobulle nomme
      les séries marquantes (demande du user).
- [x] D6 révisée par le user : en échantillon faible, plus rien n'est retiré (écarts, « Par… »,
      « Par contexte » calculés comme au-delà de 10) ; la ligne « Échantillon faible » disparaît au
      profit d'un triangle d'avertissement + infobulle sur la tuile Matchs. Vérifié côté page
      (icône, libellé, pas de ligne à part) ; le rendu des cartes en échantillon faible attend un
      serveur reconstruit sur cette branche (tests Go : mêmes règles D3-D5 qu'au-delà du seuil).
- Gates : `go test ./internal/service/ ./internal/domain/ ./internal/api/handlers/` ok ;
  `make openapi-check` ok (contrat inchangé) ; vitest `src/features/explorer src/lib/i18n` 350 tests ;
  `tsc -b --force` et eslint sans erreur.

## Reprise de session

Lire ce fichier (cases cochées), `git log --oneline -10` sur `feat/briefing-seuils` dans le
worktree, puis reprendre à la première étape non close.
