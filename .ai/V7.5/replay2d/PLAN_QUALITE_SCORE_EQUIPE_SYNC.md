# PLAN — Qualité de données : le « score d'équipe » de l'API n'est pas le score affiché

Date : 2026-08-24. Origine : REGISTRE_REPORTS ligne « L'API Teams[].Stats.CoreStats.Score
n'est pas le score affiché » (lot A phases 0-bis/0-ter, mesures dans
`.ai/V7.5/replay2d/registre_film/LOTA_PHASE0.md`) + backlog Notion « Rejeu 2D — bilan du
18/08/2026 », § À savoir. Go utilisateur : 2026-08-24.

Branche : `wt/qualite-score` (worktree dédié, base feat/v75). Exécution sous le contrat du
skill `plan-execution` (ordre strict, gates, statuts `[x]`/`[~]`/`[!]`, zéro fix hors
périmètre — les découvertes se consignent en §Découvertes, jamais ne se corrigent).

## Objectif et critère de succès

Établir, preuves à l'appui, D'OÙ peut venir un « score d'équipe » AFFICHABLE pour les modes
où `match_registry.team_0_score/team_1_score` ne le porte pas (Strongholds = ticks,
KOTH:Arena = tantôt secondes tantôt collines), et livrer un VERDICT + OPTIONS chiffrées au
superviseur. Ce lot S'ARRÊTE au verdict : AUCUNE écriture DB, AUCUN changement de sync ni
d'affichage sans arbitrage explicite du superviseur (phase 2 = gate d'arbitrage).

Succès = les trois questions du §Phase 1 ont chacune une réponse mesurée et sourcée, et le
rapport `RAPPORT_QUALITE_SCORE_EQUIPE.md` propose 2-3 options d'implémentation avec leur
coût (colonne nouvelle + backfill / correction à l'affichage / statu quo documenté).

## Faits établis (ne pas re-mesurer, ne pas re-discuter)

- `7344d24f` (Strongholds:Arena, Vagabond) : le jeu affiche 200/126, l'API donne 193/112
  (= émissions de ticks - 1 ; 200 = plafond = victoire). Détail : LOTA_PHASE0.md §A.0.1.
- `606d9844` et `8076f97f` (KOTH) : le film porte 0/3 et 3/0 (collines/manches), l'API
  105/8 et 78/105 (secondes de garde) — la MÊME colonne change de sémantique selon le match.
- Le film Theater porte le score affiché (`coverage.score.oracle = "displayed"`, schéma 12+),
  mais seuls les matchs avec artefact de rejeu en disposent (39 artefacts aujourd'hui).
- Règle du registre : ne JAMAIS rebaisser le seuil d'un gate sur l'oracle `team_score`.

## Hors périmètre (fermé)

- Toute écriture DuckDB (colonne, backfill, migration) — phase d'arbitrage d'abord.
- Le rejeu 2D et ses artefacts (le score du rejeu est déjà juste, oracle « displayed »).
- La refonte des KPI dérivés (ADR 0006) ; les modes PvE ; Halo 5 (mesurer Infinite d'abord,
  noter en §Découvertes si la même colonne H5 semble suspecte).

## Phase 0 — Recensement des surfaces qui lisent team_0/1_score

- [x] 0.1 Recenser TOUTES les lectures Go de `team_0_score`/`team_1_score` (grep sur
      `apps/go-api/`, colonnes SQL et champs struct qui en dérivent) : fichier:ligne, ce que
      la surface affiche (en-tête de match, media, dominance, historique, autre).
      -> 14 chaînes L1..L14 dans le rapport §0.1, plus 4 familles de faux positifs écartées.
- [x] 0.2 Recenser les surfaces web qui affichent ces valeurs (chaîne handler -> composant).
      -> 7 entrées W1..W7 dans le rapport §0.2.
- [x] 0.3 Dire pour chaque surface si l'anomalie est VISIBLE par l'utilisateur (ex. un
      Strongholds affiché « 193-112 » au lieu de « 200-126 ») ou masquée (outcome seul).
      -> rapport §0.3 : 5 surfaces visibles (W1..W5), Explorer aggravée par un tri
      inter-modes ; rejeu 2D et dominance Infinite non impactés.

Gate 0 : tableau exhaustif dans le rapport (aucun « etc. ») ; commande de contrôle fournie
dans le rapport (le grep exact rejouable). Clore la phase avant d'ouvrir la phase 1.

## Phase 1 — Diagnostic données : existe-t-il une source du score affiché ?

- [x] 1.1 JSON brut API : re-télécharger les stats des 3 matchs fautifs (`7344d24f`,
      `606d9844`, `8076f97f`) via le wrapper API existant (Grunt/SPNKr ; tokens via
      `MultiUserTokenStore`, JAMAIS de re-capture — un RT valide se rafraîchit) et inspecter
      `Teams[].Stats` EN ENTIER : y a-t-il un autre champ (round scores, objective stats,
      score par manche) qui porte 200/126, 3/2, 105 ? Dumper les JSON dans le dossier du lot.
      -> OUI : `Teams[].Stats.CoreStats.Score` porte 200/126, 3/0, 0/3 — le score AFFICHÉ, et
      c'est le champ que la sync lit déjà. La base contient l'autre champ,
      `ZonesStats.StrongholdScoringTicks` (193/112, 105/8, 78/105). Dumps dans
      `registre_film/api_dumps/`. **La prémisse « la même colonne API change de sémantique »
      est RÉFUTÉE** : c'est la base qui mélange deux champs, pas l'API.
- [x] 1.2 Ampleur : sur `match_registry` (lecture seule), mesurer par mode (pair_name /
      famille de modes, jamais un slug de titre) la population concernée : combien de
      Strongholds (score = ticks), combien de KOTH, et pour KOTH la proportion
      secondes vs collines (heuristique à documenter : un score de KOTH affiché est <= 5
      en Arena ; > 20 = secondes). Chiffres exacts, requêtes jointes au rapport.
      -> heuristique DEVENUE SANS OBJET après 1.1 (et impossible en Strongholds : ticks et
      score dans la même plage). Remplacée par une mesure EXACTE : re-fetch des 1 934 matchs
      portant un score, confrontation base/API. **80 faux (4,1 %), tous synchronisés avant le
      2026-04-06 ; 396/396 justes depuis le 2026-05-06.** Strongholds 51/83, Total Control
      16/124, Oddball 6/26, KOTH 3/56, CTF 2/353, Slayer et autres 2/1 248. Liste nominative :
      `registre_film/score_equipe_ecarts_2026-08-24.tsv`.
- [x] 1.3 Oracle croisé : pour les matchs qui ONT un artefact de rejeu (39), confronter
      colonne API vs score affiché du film — proportion de matchs où l'écart existe, par mode.
      -> 35 artefacts sur disque (pas 39), 20 confrontables (identité des deux camps résolue) :
      **18/20 film = API**, les 2 écarts sont des films tronqués qui sous-comptent. Sur les 2
      artefacts où base et API divergent (`7344d24f`, `af13e2b2`), **le film tranche pour l'API**.
- [x] 1.4 Verdict : une des trois issues, écrite et argumentée —
      (a) l'API porte le score affiché ailleurs (nommer le champ, couverture mesurée) ;
      (b) l'API ne le porte pas, mais il est CALCULABLE par règle de mode (donner la règle
          et son taux de réussite sur l'échantillon) ;
      (c) ni l'un ni l'autre — seul le film le porte (statu quo documenté).
      -> **ISSUE (a)** : l'API porte le score affiché dans `Teams[].Stats.CoreStats.Score`,
      couverture 1 933/1 934 matchs (le seul manquant est un FFA sans TeamId 0/1), concordance
      100 % avec la base depuis le 2026-05-06 et 18/20 avec le film. (b) et (c) réfutées.
      Le défaut est CLOS À LA SOURCE (correction API 343 entre avril et mai 2026) ; ce qui
      reste est un résidu de 80 lignes qu'un re-sync ne répare PAS (`persistMatchRegistry` est
      un INSERT nu, sans ON CONFLICT). Options chiffrées dans le rapport.

Gate 1 : les 3 matchs fautifs expliqués par le verdict ; toute règle proposée testée sur
l'échantillon 1.2 avec taux chiffré. STOP : rendre le CR au superviseur. La phase 2
(implémentation) n'existe pas dans ce lot — elle sera un lot séparé après arbitrage.

-> Gate 1 PASSÉ le 2026-08-24, CR rendu et vérifié sur pièces par le superviseur.

## Phase 2 (arbitrée le 2026-08-24) : CLI de backfill, SANS EXÉCUTION

Arbitrage utilisateur du 2026-08-24, sur la base du verdict (a) et des options chiffrées :
**option 1 (backfill ciblé), CODE SEULEMENT.** La CLI s'écrit maintenant ; elle ne s'exécute
pas — ni `--apply`, ni `--dry-run`, ni contre une copie. L'exécution est gatée « avant le tag
v7.5.0 » et suivie côté superviseur.

Raison de la barrière : le `--dry-run` lui-même fait 80 appels API et lit la shared DB, et une
autre session tient des fichiers du dépôt principal. Le lot livre donc un correctif **prêt à
jouer**, pas un correctif joué.

- [x] 2.1 Mutualiser l'extraction, ZÉRO copie : exporter `extractTeamScoresByID`
      (`internal/sync/transforms_helpers.go:160`) en `ExtractTeamScoresByID` et migrer ses
      appelants. La CLI DOIT appeler cette fonction, jamais une réimplémentation — c'est la
      seule façon de garantir que le backfill et la sync lisent le MÊME champ pour toujours.
      Contrainte : ne PAS créer de fichier à la racine de `internal/sync/` (ratchet
      `archlint/sync_root_freeze_test.go`, baseline gelée à 80).
- [x] 2.2 CLI `apps/go-api/cmd/backfill-team-scores` — entrée et lecture.
      `--ids-file` (défaut : le TSV versionné `registre_film/score_equipe_ecarts_2026-08-24.tsv`,
      dont SEULE la colonne `match_id` est lue) ; `--match` pour un id unique. Le TSV ne sert
      QUE de liste : les valeurs à écrire sont re-fetchées à l'exécution, jamais lues du TSV.
- [x] 2.3 CLI — décision et écriture. Re-fetch `GetMatchStats`, extraction via 2.1,
      comparaison à `match_registry.team_0_score/team_1_score`. Écriture UNIQUEMENT si
      différent, par `UPDATE … WHERE match_id = ?` row-by-row sérialisé (JAMAIS
      `UPDATE … FROM (VALUES …)`). **Le ratchet `no_art_patterns_test.go` exclut `cmd/` de
      ses scans : il ne protège PAS cet outil.** La protection est posée localement au
      paquet (`no_bulk_update_test.go`), et l'unicité du writer vient du verrou fichier
      DuckDB, pas du lease `dblease` (mutex intra-process, gardé pour la discipline).
      `--dry-run` par DÉFAUT ; `--apply` explicite pour écrire.
      Gardes de vraisemblance : jamais de NULL, jamais de négatif, jamais hors bornes du
      `SMALLINT` de la colonne ; un match sans `TeamId` 0/1 (FFA) = skip loggé.
- [x] 2.4 Tests unitaires table-driven, SANS réseau ni DB réelle, sur la décision de
      correction (extraction + comparaison + gardes) et sur le chargement de la liste.
- [x] 2.5 Gates : `gofmt -l`, `go vet`, tests des packages touchés, et — puisque 2.1 touche
      `internal/` — le garde-rail anti-ART et le gel du god-package sync.
- [x] 2.6 Documenter dans le rapport une section « Correctif prêt » : commandes exactes du
      jour J (dry-run puis apply, serveur arrêté) et note prod (vérifier si le résidu existe
      sur le VPS avant d'y rejouer la même passe).

Gate 2 : la CLI compile, ses tests passent, les ratchets `no_art_patterns` et
`sync_root_freeze` sont verts, et **la CLI n'a été exécutée contre AUCUNE base** — ni réelle,
ni copie. Toute sortie de mesure dans ce rapport doit provenir de la phase 1, jamais d'un run
de la phase 2.

-> Gate 2 PASSÉ le 2026-08-24 : `go build ./cmd/backfill-team-scores/` exit 0 ;
`go test ./cmd/backfill-team-scores/ -count=1` ok (24 cas, dont le chargement du TSV
versionné → 80 ids) ; `gofmt -l` vide ; `go vet` exit 0 ;
`go test ./internal/sync/ -run "NoArt|Pattern" -count=1` ok 13,245 s ;
`go test ./internal/archlint/ -count=1` ok 15,489 s (gel du god-package sync tenu — aucun
fichier ajouté à sa racine) ; `go test ./internal/sync/... -count=1` ok sur les 12 paquets
(non-régression du renommage). Seuils : 332 / 127 / 91 L par fichier, aucune fonction > 80 L.
**Aucune exécution de la CLI, contre aucune base.** Détail : rapport §Correctif prêt et
§Gates rejoués.

## Garde-rails d'exécution

- DuckDB : lecture SEULE, via `OpenReadForQuery` / CLI existants (`cmd/diag_q`) — le serveur
  air local peut tenir les DB en RW ; ne jamais ouvrir en RW, ne jamais forcer OpenReadOnly.
- Les données vivent dans le dépôt PRINCIPAL (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/`) ;
  le worktree n'a pas de `data/`.
- Commandes `go` : UNE à la fois (jamais en parallèle), avec un GOCACHE PRIVÉ au lot
  (`$env:GOCACHE` vers un dossier dédié) — le cache partagé se corrompt sous concurrence.
- Réseau : uniquement l'API Halo officielle pour 1.1 (c'est la source de la sync) ; pas de
  re-capture de token ; si l'auth échoue, diagnostiquer et CONSIGNER, ne pas « réparer ».
- Logging Go : slog structuré si du code instrumenté est écrit ; instruments jetables gatés
  par env var (patron `I22_FILM`), jamais actifs en CI.

## Découvertes

Détail et justification dans `RAPPORT_QUALITE_SCORE_EQUIPE.md` §Découvertes. Résumé :

1. 11 des 80 lignes fausses ne s'expliquent pas par les ticks de zone : 7 inversions exactes
   des deux camps (6 Oddball + 1 BTB:Sentry Defense), 1 inversion + ticks, 2 One Flag CTF à
   2 au lieu de 3, et `f395b462` (Attrition) à **1 950** contre 0 à l'API. Toutes du lot
   2026-02-14, 10/11 par `first_sync_by = Madina97294`. Mécanisme non identifié.
2. `match_participants.team_id` est SAIN sur ces mêmes matchs (les `Outcome` par camp
   concordent avec l'API) : l'inversion ne touche que les deux colonnes de score.
3. Aucun contrôle de vraisemblance à l'écriture du registre (1 950 passe le `SMALLINT`).
4. La colonne « Score » de l'Explorer est triable sur des unités hétérogènes entre modes —
   question de conception, pas de donnée.
5. Le plan annonce 39 artefacts de rejeu ; il y en a **35** sur disque.
6. `LOTA_PHASE0.md` §0b.3 est à amender : `7344d24f` « non expliqué » l'est désormais —
   l'oracle était faux, pas le film. Fichier NON modifié.
7. `sync/comeback.go:141-143` : kill-switch daté dans sa justification mais sans date cible
   de retrait ni critère mesurable (modèle CLAUDE.md non suivi). Non traité.

## CR attendu (à rendre au superviseur)

Rapport `.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md` : tableaux phase 0 et 1,
verdict 1.4, options d'implémentation chiffrées (avec contraintes anti-ART : INSERT-only,
vues `_latest`, migration déployée élargie = step au nom neuf), gates rejoués (commandes +
sorties). Statut de CHAQUE item ci-dessus. Commits atomiques dans le worktree, sujets
`score-sync(pN): ...`, JAMAIS `git add -A`.
