# Plan — Rejeu : trois derniers correctifs (bot à deux équipes, avertissements du document interne, `--dry-run`) — 2026-10-07

Branche `feat/rejeu-journal-places` (worktree `LevelUp-wt-rejeu-journal-places`, tête de départ
`8ee3f36d0`). Superviseur : session levelup-dc. Trois corrections décidées par l'utilisateur, relayées
par le superviseur.

Contrat d'exécution : skill `plan-execution`. Périmètre : `film/replay`, `sync/killcollector`,
`cmd/levelup`. Aucune écriture dans `data/` ; mesures sur une COPIE de la base partagée et des films lus
en lecture seule, un décodage à la fois. `SchemaVersion`, `SchemaDesFaits` et révisions de couche
inchangés tant qu'aucune sortie publiée ne change. Pas de fusion dans `feat/v75`.

## Étapes

### K1 — Bot à deux équipes : l'API arbitre

- [x] K1.1 Cause sur pièces : `replay/occupants.go` `equipeDe` (base `8ee3f36d0`) publiait le désignateur
      de l'entité et comptait l'écart (`rejeu_bots_equipe_contre_declaration`, ERROR sans détail).
- [x] K1.2 Correspondance `team_id` -> désignateur : IDENTITÉ, convention du contrôle
      (`player_teams.go` `controler` : `base == lue`). Preuve sur les 126 artefacts du parc (lecture
      seule) : `coverage.teams` 1 123 accords, 0 contradiction, 4 silences ; bots du roster publié
      joints par `bid` à `match_participants` (copie de la base) : 68 accords, 0 contradiction,
      4 bots absents de la base.
- [x] K1.3 `occupants_equipe_arbitree.go` : `arbitrerParLaBase` (feuille par `RosterEntry.Bid`, équipe
      négative = inconnue), journal par bot (WARN si tranché, ERROR + `rejeu_bots_equipe_sans_arbitre`
      sinon ; `rejeu_bots_equipe_contre_declaration` compte toute contradiction). Câblage :
      `build_pistes.go` (`base: a.opt.ScoreboardTeams`) ; sync : `EntreePorteursAuSync.Equipes` <-
      `ids.Equipes` (`killcollector/placement_des_vies.go`), posé en `ScoreboardTeams` par
      `optionsDuRegistre`. Commentaires devenus faux corrigés (`options.go`, `player_teams.go`,
      `build_pistes.go`, `replaybuild/options.go`, `replaybuild/flagidentity_test.go`).
- [x] K1.4 Tests `occupants_equipe_arbitree_test.go` : A-FEUILLE, A-INCONNU (5 cas), A-ACCORD,
      A-JOURNAL (niveaux + deux compteurs), A-ASSEMBLAGE (`BuildFromPositions`, Byrontron), A-SYNC ;
      couture du collecteur dans `placement_des_vies_test.go`. Mutations rouges : arbitrage retiré,
      câblage retiré, garde `t >= 0` retirée, niveaux du journal inversés, compteur sans arbitre retiré.
- [x] K1.5 Mesure (cuisson en processus, un film à la fois, binaire de la branche, racine de scratch,
      faits tirés d'une COPIE de la base par `diag_q`) sur 4 des 10 matchs : `c3256313` (bid(33.0)),
      `bcb6d393` (bid(59.0)), `4db2574e` (bid(4.0)), `06a883f7` (bid(19.0), bid(56.0)). 5 bots sur 5 :
      entité 1, déclaration 0, base 1 — LA BASE CONFIRME L'ENTITÉ ; l'équipe publiée ne change pas
      sur ces films. Les 10 matchs ont tous leurs bots dans la base.
- [x] K1.6 Aucun des 10 matchs n'a d'artefact dans `data/cache/replays/halo_infinite` (126 artefacts,
      lecture seule) ; la re-cuisson du parc (`etape1_backfill_replay.log`) n'a émis aucune
      contradiction entité/déclaration : aucune sortie publiée ne change, `SchemaVersion`,
      `SchemaDesFaits` et révisions inchangés.

### K2 — Avertissements propres à une publication tus sur le document interne

- [x] K2.1 Les six : `identity_registry_film_table.go` (table NON EMPLOYEE), `origin.go` (origine non
      établie ; calques non recalés), `player_teams.go` (équipes NON LUES), `build_inventaire.go`
      (impulsions, charges NON BALAYEES).
- [x] K2.2 `journal_de_publication.go` : `niveauDePublication(documentInterne)` = WARN ou Debug ;
      `IdentityInput.documentInterne` (non exporté, faux = collecteur et cuisson gardent WARN) ;
      `resolveOriginMs` prend `temoinDuFil` + niveau (5 paramètres). Ajouts au périmètre : « la base
      CONTREDIT le film » (devenu atteignable en interne par K1.3) et « origine LUE contredite » suivent
      le même niveau.
- [x] K2.3 Tests `journal_de_publication_test.go` : JDP-INTERNE (aucun des six en WARN), JDP-PUBLIE
      (les six en WARN, `BuildFromFacts`), JDP-NIVEAU (les deux sans témoin). Mutations rouges :
      toujours WARN, toujours Debug, registre non câblé, niveau forcé sur les deux sans témoin.

### K3 — `backfill-killsource --dry-run` : une seule sélection

- [x] K3.1 Cause : `passeDesFilms` rendait le plan sur `filmsACollecter` (avec la borne) avant de
      construire le collecteur ; la passe retirait ensuite les matchs sans carte. Log de la passe du
      2026-10-07 (`etape2bis`) : 95 écartés faute de carte, 10 à décoder ; le plan en annonçait 105.
      Les 4 de 10 non écrits sont des verdicts du décodage (3 sans kill-feed, 1 clé de profil
      inconnue), pas de la sélection.
- [x] K3.2 `candidatsDeLaPasse` / `idsDeLaPasseEnLigne` (`cmd_backfill_killsource_carte.go`), appelées
      une fois par `passeDesFilms` / `passeDesFilmsEnLigne`, plan compris ; `selectionSansBorne` sans
      exception `--dry-run`. Le plan en ligne construit la capture et un collecteur de sélection sans
      source (aucun réseau).
- [x] K3.3 Tests : `cmd_backfill_killsource_carte_integration_test.go` (cas de l'écart : premier film
      sans carte, plan = passe sous `--limit 0` et `1`, hors ligne et en ligne) ;
      `cmd_backfill_killsource_selection_unique_test.go` (garde-rail AST : `filmsACollecter` et
      `matchsSansPasseDeFilm` n'ont qu'un appelant). Mutations rouges : borne au plan, filtre retiré
      (hors ligne, en ligne), plan rendu sur `filmsACollecter` dans `passeDesFilms`.

### K4 — Livraison

- [x] K4.1 Gates : `go test -tags=integration -p 1` (replay, killcollector, cmd/levelup, replaybuild),
      `go test` (archlint, film/..., replaybuild, replayverite, replaydiff), `go vet`, golangci-lint
      `--new-from-merge-base=origin/main` (0, et 0 sous tag `integration` sur cmd/levelup).
- [x] K4.2 Revue adversariale (un relecteur, lentilles L3 + L6) : 0 P0/P1, 4 P2 (trois angles morts de
      test du lot, deux commentaires de `replaybuild` rendus faux par le lot) — traités dans le lot,
      parce qu'ils portent sur ses propres tests et sur une doc qu'il invalide.
- [x] K4.3 Commit `fed3e54be`, poussé ; CI `37634967331` verte (gitleaks `37634967364`, Deploy Pre-Check
      `37634967300` verts).
- [x] K4.4 Entrée `.ai/thought_log.md`.

## Décisions

- Correspondance d'équipe base -> film : identité (K1.2), sans table codée.
- La base arbitre aussi le document interne du sync (même registre, mêmes équipes que la cuisson).
- Avertissements internes rétrogradés en Debug (pas supprimés).

## Découvertes (non traitées)

- D1 Les 4 films mesurés (build `HI_1_12_0`) journalisent `killsource: equipe de bot(s) NON LUE dans
  BOT_METADATA ... hors_grammaire=1`, et leurs 5 bots contredits ont tous une déclaration 0 contre
  entité et base 1 : la lecture de l'équipe BOT_METADATA est suspecte sur ce build.
- D2 `06a883f7` : la cuisson compte 2 bots contredits, le document interne du sync 1.
- D3 golangci-lint sous `-tags=integration` : 5 constats dans des fichiers de `killcollector` non
  touchés (`postsync_backlog_integration_test.go` goimports, `postsync_travail.go:119` unparam,
  `emprise_cablage_partage_test.go:28-30` unused). La CI lint sans ce tag.
- D4 Pas de `--dry-run` réel mesuré : `metadata.duckdb` est verrouillée par le serveur, la copie est
  refusée ; l'égalité plan = passe est tenue par les tests K3.3.

## Journal

- 2026-10-07 : K1-K3 exécutés, gates verts, revue adversariale passée.
- 2026-10-07 : `fed3e54be` poussé, CI 37634967331 verte ; plan clos.
