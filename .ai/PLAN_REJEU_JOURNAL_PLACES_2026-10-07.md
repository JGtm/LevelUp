# Plan — Rejeu : le journal des places ne concerne que le document publié — 2026-10-07

Branche `feat/rejeu-journal-places` (worktree `LevelUp-wt-rejeu-journal-places`, base `origin/feat/v75`
`7e9c72eaf`). Superviseur : session levelup-dc. Découverte source : D16 de
`.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md` (cause établie par la session RI levelup-57).

Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'une action faisable, chaque
item statué `[x]` / `[~]` / `[!]`). Périmètre : `film/replay` (code, tests, commentaires) ; aucun
décodage de film, aucune écriture dans `data/` ; `SchemaVersion` et révisions de couche inchangées (aucune
sortie publiée ne change). Pas de fusion dans `feat/v75`.

## Le défaut

`levelup backfill-killsource` (re-cuisson du 2026-10-07) : 82 matchs journalisent en ERROR « rejeu :
entree(s) du roster presente(s) SANS EQUIPE lue », 73 pour toutes leurs entrées (Oddball surtout, puis
Assault et VIP). Les 126 artefacts publiés sont à 0 entrée sans équipe.

## Étapes

### J0 — Cause sur pièces

- [x] J0.1 Le message et son appel : `replay/sieges.go:289-294` (`journaliserLesPlaces`, ERROR si
      `cov.SansEquipe > 0`), appelé par `replay/build_pistes.go:144` (`poserLesEquipesEtLeRoster`) dans
      tout assemblage `BuildFromPositions` (base `7e9c72eaf`).
- [x] J0.2 Le chemin interne : `replay.PortagesAuSync` (`replay/porteurs_au_sync.go:115-131`) assemble
      par `BuildFromPositions` (l. 126) un document qu'il ne publie pas (`portagesDuDocument` ne relit que
      les quatre calques de porteur et le calage) ; `grammar.ScanPlayerTeams` n'y est appelé que dans la
      branche `g.Drapeau` (l. 169-177). Hors CTF, seul un bot dont la déclaration BOT_METADATA porte une
      équipe en reçoit une (`replay/occupants.go`) : toute autre entrée présente, tout humain, compte dans
      `sansEquipe` (`replay/sieges_tirs.go:194-197`, indépendant de la table du film).
      Reproduit par le test J1.1, rouge avant la correction (ERROR « SANS EQUIPE » sur 4 entrées).
- [x] J0.3 Les données sont justes : le placement des vies range par camp depuis la base
      (`sync/killcollector/placement_des_vies.go:108`, `Equipes: equipesNumeriques(ids.Equipes)`), jamais
      depuis ce document.
- [x] J0.4 Choix de la correction, justifié (cf. Décisions).

### J1 — Tests d'abord

- [x] J1.1 Test du chemin interne (`replay/journal_des_places_test.go`,
      `TestPortagesAuSync_LeDocumentInterneNeJournalisePasLesPlaces`, JP-INTERNE) : `PortagesAuSync` sur un
      Oddball (garde du crâne, aucune lecture des équipes) n'émet aucune ligne ERROR « SANS EQUIPE »
      (journal capturé). Témoin non vide : les mêmes entrées sous `optionsDuRegistre` donnent 4 entrées
      présentes sans équipe. ROUGE avant la correction (l'ERROR sort).
- [x] J1.2 Test du chemin publié (`TestBuildFromFacts_LeDocumentPublieJournaliseLEntreeSansEquipe`,
      JP-PUBLIE) : les mêmes entrées republiées depuis les faits (`BuildFromFacts`, `Options{}`) émettent
      l'ERROR et comptent `coverage.seats.sansEquipe` = 4 — vert avant et après.

### J2 — Correction

- [x] J2.1 `Options.documentInterne` (non exporté, `replay/options.go`), posé par `PortagesAuSync` juste
      avant son `BuildFromPositions` ; l'appel à `journaliserLesPlaces` est gardé dans
      `poserLesEquipesEtLeRoster` (`if !a.opt.documentInterne`).
- [x] J2.2 Commentaires au présent : le réglage (`options.go`), l'appel gardé (`build_pistes.go`), la doc
      de `journaliserLesPlaces` et de `SeatCoverage.SansEquipe` (`sieges.go`), le commentaire de
      `PortagesAuSync`. Aucune sortie publiée ne change : `SchemaVersion`, révisions de couche et
      `PlacementRev` inchangés.
- [x] J2.3 J1.1 et J1.2 verts, ainsi que les tests voisins (`TestPortagesAuSync_*`, `TestAssemblage*`,
      `TestBot*`, `TestHumainSansPlaceNEstJamaisEcarte`).

### J3 — Mutations

- [x] J3.1 Garde retirée (`if true`) : J1.1 ROUGE.
- [x] J3.2 Réglage non posé par `PortagesAuSync` : J1.1 ROUGE.
- [x] J3.3 Garde inversée (`if a.opt.documentInterne`) : J1.2 ROUGE (J1.1 aussi).
- [x] J3.4 Réglage posé dans `BuildFromFacts` (chemin publié réduit au silence) : J1.2 ROUGE.
      Chaque fichier restauré depuis sa copie (scratchpad), diff final contrôlé.

### J4 — Gate

- [x] J4.1 `gofmt -l` muet, `go vet` du paquet 0 ; `go test -count=1 ./internal/games/halo_infinite/film/replay/...`
      ok (32 s) ; `go test -count=1 ./internal/sync/killcollector/...` ok ; en plus, appelant du chemin
      interne : `go test -tags=integration -p 1 ./internal/sync/killcollector/` ok (32 s, témoin sur films
      réels sauté faute de données, voulu). En série, CGO, `GOCACHE` dédié.
- [x] J4.2 `go test -count=1 ./internal/archlint/` ok (43 s).
- [x] J4.3 `golangci-lint run --new-from-rev=7e9c72eaf ./internal/games/halo_infinite/film/replay/...` :
      0 issue (v2.12.2, cache isolé).

### J5 — Clôture

- [x] J5.1 D16 notée corrigée dans `.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md` (Découvertes), avec la
      réponse à sa question « à vérifier » (le placement range par camp depuis la base).
- [x] J5.2 Entrée `.ai/thought_log.md` (statut « En cours » jusqu'à la revue et la CI).
- [x] J5.3 Revue adversariale, ronde 1 (un relecteur frais, premier plan, lecture seule, contrat écrit,
      lentilles L6 et L3, diff `7e9c72eaf..6e5eb7689`) : AUCUN constat recevable, 22 conditions vérifiées
      qui tiennent. Une nuance retenue, corrigée dans les textes du lot (J0.2, D16, thought_log, en-tête
      du test) : hors CTF, un bot dont la déclaration BOT_METADATA porte l'équipe en reçoit une, donc
      « toute entrée présente sans équipe » ne vaut que pour les humains. Pas de ronde 2 (aucun P0 ni P1).
- [ ] J5.4 `delivery-checklist` ; commit(s) `fix(rejeu):` ; push ; CI suivie au premier plan, verte.

## Décisions

- DJ1 (J0.4) : un réglage NON EXPORTÉ de `Options`, `documentInterne`, posé par `PortagesAuSync`, garde
  l'appel au journal des places dans l'assemblage. Sa valeur zéro est celle de toute cuisson publiée, et
  le compilateur interdit à tout appelant hors de `replay` de le poser : le chemin publié ne peut pas
  être réduit au silence de l'extérieur, et le journal garde un seul site d'appel.
- DJ2 (alternatives écartées) : poser les équipes du film sur ce chemin n'est pas « à bas coût » —
  `grammar.ScanPlayerTeams` est l'une des lectures les plus chères du lot V0, payée sur un CTF seulement
  (`TestPortagesAuSync_ChaqueFamilleNePaieQueSesLectures`), pour un roster que personne ne relit.
  Déplacer l'appel du journal dans les chemins de publication (`BuildFromFilm`, `BuildFromFacts`) en
  ferait deux sites à tenir d'accord.
- DJ3 : la garde couvre TOUT `journaliserLesPlaces` (l'ERROR « sans équipe », l'avertissement des places,
  le compteur expvar et l'ERROR des bots sans place) : le journal entier dit le roster publié. Sur le
  chemin interne, seule l'ERROR « sans équipe » pouvait sortir : sans table du film (`FilmTable` n'y est
  pas passée), la pose des places ne chaîne aucun arrivant, n'écarte aucun bot et ne mesure aucun
  dépassement.

## Découvertes (notées, non traitées)

- DJ-a : le même document interne émet aussi, à chaque synchronisation d'un mode à porteur, des
  AVERTISSEMENTS propres à une publication, que le journal capturé de J1.1 montre et dont les conditions
  sont structurelles sur ce chemin : « table du film NON EMPLOYEE » (`FilmTable` jamais passée,
  `identity_registry_film_table.go`), « origine d'horloge non établie » puis « aucune origine établie »
  (`FilmClockOriginUS` jamais passé, `origin.go`), « équipes NON LUES dans le film » hors CTF
  (`player_teams.go`), « impulsions de capacité NON BALAYÉES » et « charges d'équipement NON BALAYÉES »
  (`build_inventaire.go`). Même nature que D16, niveau WARN ; non traité (périmètre : le journal des
  places). `Options.documentInterne` est le point d'appui naturel d'un lot qui les taira.
- DJ-b : `backfill-killsource` n'assemble aucun document publié (aucun appel à `BuildFromFilm`,
  `BuildFromFacts` ni `replaybuild` dans `cmd/levelup/cmd_backfill_killsource*.go` ni dans
  `sync/killcollector`) : les journaux « rejeu : » de l'assemblage qu'il émet viennent du document
  interne de `PortagesAuSync`. L'ERROR de D15 (« l'entité ti=9 d'un bot et son entrée BOT_METADATA disent
  deux équipes », `occupants.go`, `journaliserLesEquipesDeclarees`, avec son compteur expvar) exige des
  entités ti=9 lues, ce que ce chemin ne fait que sous la garde du drapeau : ses 10 matchs sont des CTF
  vus par le document interne. L'écart lu reste une observation du film, à instruire avec D15 ; son
  journal, lui, a la même nature que D16. Non traité.

## Journal

- 2026-10-07 : plan écrit ; lecture du brief, de CLAUDE.md, des skills `plan-execution` et
  `arch-rules`, du code cité.
- 2026-10-07 : J0 à J4 clos (cause sur pièces, tests d'abord, correction, quatre mutations rouges, gate
  vert) ; `go build ./...` et `go vet ./...` 0 en plus du gate (skill `delivery-checklist`). Découvertes
  DJ-a et DJ-b notées, non traitées. Commit `6e5eb7689`.
- 2026-10-07 : revue adversariale ronde 1 sans constat recevable ; nuance « humains » reportée dans les
  textes du lot.
