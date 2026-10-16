# PLAN — Quatre citations Halo 5 portées sur Halo Infinite, dont trois lues du film (2026-10-16)

> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, statut de
> chaque item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture, zéro fix hors
> périmètre — les découvertes vont au §Découvertes).
> Branche : `feat/citations-film` depuis `feat/v75`, worktree dédié. Fusion dans `feat/v75`
> = geste à confirmer avec l'utilisateur. Push `main` : jamais dans ce plan.
> Reprise de session : lire ce fichier (cases cochées), puis `git log --oneline -10` de la branche.

## Objectif et critère de succès

Quatre citations nouvelles, actives, calculées pour TOUT joueur suivi ayant une base, au
post-sync et par rattrapage sur les matchs déjà synchronisés :

| Norm | FR | EN | Définition (Halo 5, Halopedia) | Source | Paliers |
|---|---|---|---|---|---|
| `tick_tick_boom` | Tick Tick Boom | Tick Tick Boom | Gagner des parties d'Assaut | API (outcome + variante) | D4 |
| `first_strike` | Premier coup | First Strike | Faire le premier frag du match | `highlight_events` (déjà dans `ctx.Events`), règle « Premier sang » existante | D4 |
| `wonderful_toys` | Jouets éducatifs | Wonderful Toys | Prendre au moins 3 armes sur socle dans un match Assassin | film (`match_usage_players_latest.pad_pickups`) | D4 |
| `almost_doesnt_count` | Presque, ça ne compte pas | Almost Doesn't Count | Tuer un Spartan bouclier à 0 % et survivre | film (bouclier `Sh` du tueur) — **sous condition E1** | D4 |

Succès = sur la base locale, après `levelup seed citation-mappings` puis
`levelup backfill --citations-recompute-all --all`, chaque citation livrée a une progression non
nulle chez au moins un joueur suivi, ET un match dont le film est dérivé APRÈS l'étape citations
du post-sync voit ses citations film recalculées sans intervention.

## Faits vérifiés sur pièces (2026-10-16)

1. `match_citations_latest` garde, PAR MATCH, la seule génération la plus récente
   (`steps_player_append_only_match_citations.go:40`, `PARTITION BY match_id`). Écrire les seules
   citations film dans une génération neuve MASQUERAIT les autres : tout recalcul est un recalcul
   COMPLET du match (`BackfillMatchCitations` sur ce match).
2. Le post-sync choisit les matchs SANS ligne dans `match_citations` (`selectMatchesForCitations`,
   `citations_backfill.go:157`). Un match déjà calculé ne repasse jamais en `force=false`.
3. Les dérivations du film (`Deriver`, `replayartifacts/derivations.go:114`) tournent APRÈS l'étape
   1.6 citations, et en production chez l'ouvrier, en asynchrone. Sans déclencheur dédié, les
   citations film seraient calculées à 0 puis figées.
4. Bonus sur socle : anonymes par construction (`powerup_pickups_json`, grain match). Armes sur
   socle : attribuées (`pad_pickups`, `XUID` nul possible). Bonus pris par joueur : candidat
   `taken_json`, familles `powerup_*` — à mesurer (E1).
5. Santé : `TrackPoint.Hp` (couverture ~0,6 %) et `Sh` (~16 %) vivent dans l'artefact seulement,
   rien en base.
6. Les `compute_wins_*` (`games/halo_infinite/citations_custom.go:76-130`) testent la variante par
   `strings.Contains` — quatre copies du même patron.

## Décisions produit (à trancher AVANT exécution)

- **D1 — « Presque, ça ne compte pas »** (utilisateur, 2026-10-16) : compte un frag dont le tueur a
  un bouclier à 0 % à l'instant du frag. L'instant = dernière mesure `Sh` de la piste du tueur
  avant le frag (le film ne réplique le bouclier qu'à son changement, `document_aim.go:97`).
  Calcul à la dérivation, sur le document déjà en mémoire (une recherche par frag dans la piste
  du tueur), jamais une relecture d'artefact au calcul des citations. Livrée si E1 montre une
  mesure de bouclier disponible avant au moins 90 % des frags ; sinon `[!]` et compte rendu.
  Points à statuer à E1, avec la mesure : fraîcheur maximale de la dernière mesure (la recharge
  n'est pas lisible : un 0 ancien peut être périmé) ; « survivre » = proposition « le tueur n'est
  pas mort dans la seconde qui suit le frag ».
- **D2 — « Jouets éducatifs »** (utilisateur, 2026-10-16) : armes prises sur socle seulement,
  colonne existante `pad_pickups` (déjà exploitée par les stats d'usage), vaut 1 par match
  Assassin où elle vaut au moins 3. Aucun calcul nouveau des prises.
- **D3 — « Premier coup »** (utilisateur, 2026-10-16) : la règle « Premier sang » existante
  (`analysis/match_impact.go:109`, `firstByTime` sur les frags de `highlight_events`), exposée par
  un helper exporté et réutilisée telle quelle — aucun calcul nouveau. N'a pas besoin du film.
- **D4 — Paliers** (utilisateur, 2026-10-16) : Tick Tick Boom `5,10,15,25,50` ; Premier coup
  `10,25,50,100,250` ; Jouets éducatifs `5,10,20,35,60` ; Presque `10,20,30,50,100`, puis
  `50,100,150,250,500` (utilisateur, 2026-10-16, au vu de E5 : quatre joueurs sur cinq au plafond).
- **D7 — Matchs sans artefact sur disque** (utilisateur, 2026-10-16) : pas de recuisson ; « Presque,
  ça ne compte pas » progresse avec les matchs dont le film est (ou sera) dérivé en `us7`.
- **D5 — Visuels** (utilisateur, 2026-10-16) : blanc avec ombre légère, le style de « Défenseur
  du drapeau », pour Premier coup, Presque et Tick Tick Boom (versions `*_BLANC` soumises à
  l'utilisateur), sous `static/commendations/halo_infinite/HI_citation_<Nom>.png`, les natives
  intactes ; Jouets éducatifs garde la native. VALIDÉES par l utilisateur le 2026-10-16 et déjà
  au dépôt : `HI_citation_Premier_coup.png`, `HI_citation_Presque_ca_ne_compte_pas.png`,
  `HI_citation_Tick_Tick_Boom.png`. En cas de conflit de fusion sur une image de citation, les
  versions remaniées de cette session prévalent (utilisateur, 2026-10-16).
- **D6 — Catégories** : Tick Tick Boom et Jouets éducatifs = Mode de jeu ; Premier coup et Presque
  = Multijoueur.

## Étapes

### E0 — Mise en place
- [x] Worktree `feat/citations-film` depuis `feat/v75` (`../LevelUp-wt-citations`) ; cache Go dédié `go-build-citations`.
- [x] Gate : `git branch --show-current` = `feat/citations-film`.

### E1 — Mesure (recherche, aucun code de production)
- [x] Sur les artefacts du parc : part des frags dont la piste du tueur porte une mesure `Sh`
      avant le frag ; distribution de l'âge de cette dernière mesure ; part des frags à 0 %.
- [x] Même mesure, au plafond de fraîcheur proposé, pour statuer D1 (fraîcheur et « survivre »).
- [x] Coût : temps ajouté à la dérivation d un artefact par la recherche des frags (cible :
      négligeable devant la dérivation actuelle).
- [x] Gate : compte rendu chiffré dans le thought_log ; D1 statuée (`[x]` livrée telle que
      décidée, ou `[!]` avec la mesure) et validée par l'utilisateur avant E3.

### E2 — Victoires d'Assaut, Premier coup, helper de mode
- [x] Prédicats de mode uniques (`containsAny`, `isSlayerMode`, `isCTFMode`, ...) dans `citations_custom_modes.go`, les quatre
      `compute_wins_*` et Bulldozer migrées dessus, garde-rail `TestCitationsCustom_ModesParPredicat` interdisant `ctx.Playlist` / `ctx.GameVariant` hors
      du helper (règle des 3 copies).
- [x] `compute_wins_assault` : outcome gagné et variante `assault` ou `bomb` (variante, jamais la
      playlist — « Assault » est aussi un conteneur de playlist, skill `halo-modes`).
- [x] `compute_first_strike` : 1 si le « Premier sang » du match (`analysis.FirstBlood`, nouveau fichier `first_blood.go`, appelé aussi par
      `ComputeMatchImpactFull` : même règle, tests d impact inchangés et verts) est le joueur.
- [x] Seed `tick_tick_boom` et `first_strike` (libellés et descriptions EN, images D5, catégories D6).
- [x] Gate : `go test ./internal/analysis/ ./internal/games/halo_infinite/ ./internal/ops/` vert ; golangci-lint sur ces paquets : aucun constat nouveau.

### E3 — Citations du film
- [x] `loadFilmStats(sharedDB, matchID, xuid)` (`internal/sync/citations_film_stats.go`) sur le patron
      de `loadObjectiveStats` : lit `pad_pickups` et `shieldless_kills` de `match_usage_players_latest`.
      Vue absente = DEBUG et clés absentes ; autre erreur SQL = WARN ; colonne NULL = clé absente.
- [~] Porte capability : la PRODUCTION de la colonne est gatée par `film.usage_summary` (fil de l'eau
      `persisterResumesUsage` et `levelup backfill-usage-summary`, inchangés) ; la LECTURE côté
      citations est best-effort comme `loadObjectiveStats` (Halo 5 n'a pas la vue : DEBUG, clés
      absentes, citations à 0). Pas de nouvelle clé de capability.
- [x] `compute_wonderful_toys` : 1 si match Assassin et `pad_pickups >= 3` (constante nommée).
- [x] D1 livrée : `replay.AttachShieldlessKills` (`usage_summary_shieldless.go`, pur) posé après
      `BuildUsageSummary` ; frags lus en base avant tout segment d'écriture (`usage_frags.go`,
      `preparerFragsDuResume` dans `Deriver`, et `fragsDuFilDesMorts` dans le backfill) ; migration
      `shared_match_usage_players_shieldless_v1` (colonne nullable, vue `_latest` recréée) ;
      persister INSERT-only ; `UsageSummaryRev` us6 -> us7 (chronique). Citation en `stat` sur
      `shieldless_kills`.
- [x] Seed `wonderful_toys` et `almost_doesnt_count` (libellés et descriptions EN, images D5).
- [x] Gate : tests purs (customs, `AttachShieldlessKills`, `loadFilmStats`), intégration
      `TestResumeUsage_FragsSansBouclier` ; `go test` vert sur `sync`, `ops`, `games/...`, `persist`,
      `migration`, `sync/replayartifacts`, `cmd/levelup` ; `-tags=integration` vert sur `persist`,
      `sync/replayartifacts`, `cmd/levelup`. Lint : `DispatchCustom` passait à 16 de complexité,
      routage converti en table (`customFunctions`) ; aucun autre constat nouveau.

### E4 — Déclencheur après les dérivations du film
- [~] Jeton `_film` : remplacé par une comparaison d'horodatages (item suivant). Un jeton aurait été
      une ligne de plus dans `match_citations_latest`, lue par les contrôles V1-V4 (`value <= 0` hors
      `_processed`) et par les sommes par citation ; l'horodatage `written_at` existe déjà des deux
      côtés.
- [x] Sélection `selectMatchesFilmPlusRecent` (`internal/sync/citations_film_selection.go`) : matchs
      du joueur dont la génération de citations courante est plus ancienne que la dernière dérivation
      de leur film (`match_usage_films_latest.written_at`). Titre sans film : vue absente, rien rendu.
      Coût mesuré sur une copie de la base partagée : 5 ms (1 352 matchs filmés).
- [x] Rattrapage du résumé d'usage depuis les artefacts existants : `levelup backfill-usage-summary`
      (révision `us7`, sans recuisson) — lancement en lot à DEMANDER à l'utilisateur (E5).
- [~] Appel après `Deriver` : couvert par l'appel au cycle post-sync suivant. Le dépôt d'ouvrier
      dérive hors de tout contexte joueur (aucune base joueur ouverte) ; la sélection, appelée dans
      `runPostSyncCitations` de chaque joueur, recalcule au cycle suivant tout match dont le film a
      été dérivé après ses citations (cuisson locale, dépôt d'ouvrier, rattrapage, backfill).
      Recalcul COMPLET du match (fait 1).
- [x] Gate : `TestCitationsFilm_RecalculApresDerivation` (tag `integration`) — match calculé sans
      film, film dérivé ensuite, sélection = ce match, citation du film posée, `bulltrue` inchangée,
      sélection vide ensuite ; `TestCitationsFilm_TitreSansFilm`. `go test -tags=integration
      ./internal/sync/` vert ; lint sans constat nouveau.

### E5 — Rattrapage et vérification sur la base locale
- [x] Serveur arrêté (coordination avec les cinq autres sessions ; levelup-66 tenait le port pour une
      recette, attendue) : sauvegarde des bases dans le scratchpad, puis, binaire de la branche contre
      le repo principal : `levelup seed citation-mappings` (4 insérées, 106 mises à jour),
      `levelup backfill-usage-summary` (237 écrits, 0 échec, 11 s), `levelup backfill
      --citations-recompute-all --all` (9 joueurs, 0 échec, checks V1-V4 verts, 6 min). Air et Vite
      relancés.
- [x] Contrôle DuckDB : progression non nulle pour les quatre citations (tableau du journal) ; les
      citations EXISTANTES sont identiques, joueur par joueur et citation par citation, à un recalcul
      complet par le binaire de `feat/v75` sur une copie des mêmes données (0 écart). Les écarts
      constatés contre la sauvegarde d'avant viennent du recalcul complet lui-même (frags par arme
      venus depuis de la source de dégât, citation « Chasse au rapatrieur » désactivée remise à 0) :
      le binaire de `feat/v75` produit les mêmes.
- [x] Gate : `go test ./...` vert ; `go test -tags=integration ./internal/sync/... ./internal/persist/...`
      vert ; `golangci-lint --new-from-merge-base=origin/main` (cible `make go-api-lint`) : 0 constat.

### E6 — Docs et clôture
- [x] `docs/COMMENDATIONS_REFERENCE.md` (110 citations, tables Mode de jeu et Multijoueur, section « Film-derived sources ») + `docs/FR/CITATIONS_REFERENCE.md` (renvoi, compte mis à jour).
- [x] Notes de version FR/EN (v7.5, bloc « Quatre nouvelles citations ») ; thought_log ; `.ai/REGISTRE_REPORTS.md` (D7 et les trois découvertes).
- [ ] Skill `delivery-checklist` ; commit, push de la branche, CI verte.

## Découvertes (hors périmètre, non traitées)

- `computeFKFD` (`progression/profile/queries.go:289-310`) lit un type `first_kill` qu'Infinite ne
  stocke pas : vaut 0 sur Infinite.
- `citations.engine = "not_exposed"` dans le `capabilities.toml` Infinite alors que les citations y
  tournent.
- Image native Infinite du Fusil Storm présente, aucune citation de maîtrise ne l'utilise.

## Journal

### E1 — mesure du bouclier (2026-10-16)

Instrument : `internal/sync/replayartifacts/citations_bouclier_research_test.go` (tag `research`),
233 artefacts du parc croisés aux frags de `match_kill_events_latest` (copie de la base partagée).

- 22 725 frags à tueur nommé. Mesure de bouclier dans la vie du tueur avant le frag : 16 513
  (72,7 %). Vie trouvée SANS aucune mesure avant le frag : 5 340 (23,5 %). Vie du tueur
  introuvable : 872 (3,8 %).
- Une vie sans mesure est une vie dont le bouclier N'A JAMAIS CHANGÉ (le film ne le réplique qu'au
  changement) : bouclier plein, donc pas un frag à 0 %. Témoin : 94,5 % des vies de VICTIMES
  (21 345 / 22 577), qui ont par définition pris des dégâts, portent une mesure. L'état du
  bouclier est donc connu pour 96,2 % des frags (>= 90 % : D1 LIVRÉE).
- Âge de la dernière mesure : médiane 0 ms, p75 0 ms, p90 9 200 ms. Bouclier à 0 % : 4 402 frags,
  dont 4 396 sur une mesure de moins d'une seconde. Plafond de fraîcheur retenu : 5 s (constante
  nommée ; écarte les 6 zéros anciens, que la recharge a pu effacer sans réplication).
- Survie (« le tueur n'est pas mort dans la seconde qui suit », accord utilisateur du 2026-10-16) :
  3 480 des 4 396 frags (79 %). Environ 15 par match sur le parc.
- Contrôle d'horloge : fin de vie de la victime moins instant du frag, médiane 0 ms (p10 -100, p90 0).
- Coût : 81 ms pour 233 artefacts, 0,35 ms par artefact (négligeable).
- Conséquence pour E3 : les frags ne sont PAS dans le document de rejeu ; la colonne se calcule à la
  dérivation, en croisant le document en mémoire aux frags de `match_kill_events_latest` du match
  (même base, même passe que le résumé d'usage).

### E2 — Tick Tick Boom et Premier coup (2026-10-16)

`compute_wins_assault` (variante `assault`/`bomb`, jamais la playlist) et `compute_first_strike`
(règle « Premier sang » de l'impact du match, extraite en `analysis.FirstBlood` dans un fichier
neuf : `match_impact.go` est déjà au-dessus du seuil de 500 lignes, il ne grossit pas). Les quatre
victoires de mode et Bulldozer passent par des prédicats uniques ; le garde-rail interdit de relire
la playlist ou la variante hors de `citations_custom_modes.go`. Premier coup ne dépend pas du film :
il se calcule au post-sync normal, avec les événements highlight déjà chargés.

### E3 — Jouets éducatifs et Presque, ça ne compte pas (2026-10-16)

Les frags ne sont pas dans le document de rejeu : la colonne `shieldless_kills` se calcule à la
dérivation du résumé d'usage en croisant le document au fil des morts du match, lu par un segment de
LECTURE avant tout segment d'écriture (même règle que les niveaux d'armes et les véhicules). Une
lecture en échec n'arrête pas le résumé (ses autres grandeurs ne lisent pas la base) : les frags
restent NULL et c'est journalisé en ERROR. La révision `us7` fait reprendre le parc par
`levelup backfill-usage-summary` sans recuisson.

### E5 — rattrapage local (2026-10-16)

Totaux après recalcul complet (somme des valeurs, plafonnée au dernier palier) :

| Joueur | Tick Tick Boom | Premier coup | Jouets éducatifs | Presque, ça ne compte pas |
|---|---|---|---|---|
| Chocoboflor | 0 | 55 | 37 | 100 (plafond) |
| JGtm | 2 | 101 | 36 | 100 (plafond) |
| Madina97294 | 0 | 126 | 60 | 100 (plafond) |
| Nuzzles | 42 | 250 | 17 | 100 (plafond) |
| XxDaemonGamerxX | 0 | 2 | 6 | 3 |

Deux constats signalés à l'utilisateur :

1. **Les paliers de « Presque, ça ne compte pas » (D4 : 10-100) sont atteints par quatre joueurs sur
   cinq.** Comptes bruts du résumé d'usage : 124 à 233 frags sans bouclier sur 87 à 127 matchs
   mesurés, soit environ deux par match mesuré.
2. **Les frags sans bouclier ne sont mesurés que sur les matchs dont l'artefact est encore sur
   disque** : le rattrapage a re-projeté 237 artefacts ; 8 993 matchs du registre n'en ont pas. JGtm a
   un résumé d'usage sur 926 matchs mais un compte mesuré sur 127 seulement (les autres résumés datent
   de `us6` et gardent la colonne NULL). Les mesurer exigerait de recuire leurs films.

### Paliers de « Presque, ça ne compte pas » relevés (2026-10-16)

D4 révisé par l'utilisateur au vu de E5 : `50,100,150,250,500`. Second arrêt court du serveur local
(accord des sessions), re-seed (110 mises à jour) et recalcul complet (9 joueurs, V1-V4 verts). Les
valeurs ne sont plus plafonnées : Chocoboflor 124, JGtm 175, Madina97294 202, Nuzzles 233,
XxDaemonGamerxX 3 (égales aux comptes bruts du résumé d'usage).
