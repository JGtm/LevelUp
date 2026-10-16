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
  `10,25,50,100,250` ; Jouets éducatifs `5,10,20,35,60` ; Presque `10,20,30,50,100`.
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
- [ ] Helper unique `variantMatches(ctx, motifs...)` dans `citations_custom.go`, les quatre
      `compute_wins_*` migrées dessus, garde-rail grep (test) interdisant `strings.Contains(gv,` hors
      du helper (règle des 3 copies).
- [ ] `compute_wins_assault` : outcome gagné et variante `assault` ou `bomb` (variante, jamais la
      playlist — « Assault » est aussi un conteneur de playlist, skill `halo-modes`).
- [ ] `compute_first_strike` : 1 si le « Premier sang » du match (helper exporté depuis
      `analysis/match_impact.go`, sans changer le calcul ni ses tests) est le joueur.
- [ ] Seed `tick_tick_boom` et `first_strike` (libellés et descriptions EN, images D5, catégories D6).
- [ ] Gate : `go test ./internal/analysis/ ./internal/games/halo_infinite/ ./internal/ops/`.

### E3 — Citations du film
- [ ] `loadFilmStats(sharedDB, matchID, xuid)` sur le patron de `loadObjectiveStats`
      (`citations.go:520`) : lit `pad_pickups` (et la colonne D1 si livrée) de
      `match_usage_players_latest`. Vue absente = log DEBUG et clés absentes ; erreur SQL = log
      WARN, jamais avalée.
- [ ] Porte capability : la lecture est gatée par `film.usage_summary` (existante) ; Halo 5 ne la
      déclare pas, donc clés absentes et citations à 0, sans erreur.
- [ ] `compute_wonderful_toys` : 1 si match Assassin et `pad_pickups >= 3`.
- [ ] Si D1 livrée : colonne `shieldless_kills` ajoutée au résumé d'usage (`replay.BuildUsageSummary`,
      migration ALTER de `match_usage_players`, persister INSERT-only, vue `_latest` recréée, ADR 0026),
      `compute_almost_doesnt_count` = sa valeur.
- [ ] Seed `wonderful_toys` (+ `almost_doesnt_count` si D1 livrée).
- [ ] Gate : tests purs des customs (Assassin / non-Assassin, seuil 2 / 3, colonne absente) +
      `go test ./internal/sync/ ./internal/ops/ ./internal/games/... ./internal/persist/`.

### E4 — Déclencheur après les dérivations du film
- [ ] Jeton `_film` dans la génération écrite quand le film du match est dérivé au moment du calcul.
- [ ] Sélection `selectMatchesForFilmCitations` : matchs du joueur ayant une ligne
      `match_usage_players_latest` et dont la génération courante ne porte pas `_film`.
- [ ] Si D1 livrée : rattrapage du résumé d usage depuis les artefacts existants (commande
      `usage-summary`, pas de recuisson du film) — lancement en lot DEMANDÉ à l utilisateur avant.
- [ ] Appel après `Deriver` (local et dépôt de l'ouvrier) ET au cycle post-sync suivant (rattrapage
      d'un dépôt manqué), recalcul COMPLET du match (fait 1).
- [ ] Gate : test d'intégration (tag `integration`) — match calculé sans film, film dérivé ensuite,
      citations film > 0 et citations API inchangées après le déclencheur.

### E5 — Rattrapage et vérification sur la base locale
- [ ] Serveur arrêté : `levelup seed citation-mappings`, puis
      `levelup backfill --citations-recompute-all --all` ; checks V1-V4 verts.
- [ ] Contrôle DuckDB : progression non nulle par citation livrée ; aucun écart des citations
      existantes avant/après (somme par citation).
- [ ] Gate : `go test ./...` + `go test -tags=integration ./internal/sync/... ./internal/persist/...`
      + `make go-api-lint`.

### E6 — Docs et clôture
- [ ] `docs/COMMENDATIONS_REFERENCE.md` + `docs/FR/CITATIONS_REFERENCE.md` (inventaire, sources film).
- [ ] Notes de version FR/EN si le format l'exige ; thought_log ; `.ai/REGISTRE_REPORTS.md`.
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
