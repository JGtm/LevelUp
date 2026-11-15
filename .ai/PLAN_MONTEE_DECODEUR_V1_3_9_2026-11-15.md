# Plan — montée du décodeur de film v1.3.9 dans la 8.2.0 (2026-11-15)

Décision du user (2026-11-15, « option 1 ») : la 8.2.0 embarque la dernière version du décodeur, pour la
table des 157 médailles du jeu (décodeur #26, 5a71271), avec tout ce que main du décodeur porte depuis v1.3.8.
Montée confiée à levelup-58 par levelup-dd (pilote 8.2.0). Point de gel convenu avec levelup-1f (phase C) :
main du décodeur après la fusion de `feat/c2-manches` (schéma 122). C3, C4, C7, C8 partent dans la version
suivante. Contrat : skill `plan-execution`.

Lieux : worktree `LevelUp-wt-decodeur139` (branche `feat/monte-decodeur-v139`, depuis
`release/v8.2.0@010e65bbc4`, aucune jonction vers `data/`) ; clone du décodeur `hifd-medailles`.
GOWORK local par `scripts/briques-workspace.sh` uniquement (jamais de `go.work`, rien de versionné).

## E1 — Côté LevelUp sur main du décodeur (GOWORK), avant le gel

- [x] Clone du décodeur à jour de `origin/main` (5a71271, détaché) ; `levelup-briques.work` engendré ; `GOWORK` posé.
- [x] Miroirs des schémas 117 à 121 (122 et 123 restent à E2) : `domain/replaydoc`, `service/replayview`,
      `make openapi-gen`, `make generate-types` ; gardes de parité vertes.
- [x] Fixtures de contrat (`REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run ContractFixtures -update`
      du décodeur, racine de données = le worktree de montée, voir Découvertes) puis `go run ./cmd/film-web-snapshot`.
- [x] Correctif `replayidentity.BotIdentities` (spécification de levelup-1f) : bots non épinglés NOMMÉS
      passés au décodeur avec leurs déclarations, bots sans nom toujours refusés, compteur
      `killsource_bots_non_epingles` en deux parts (passés / refusés). Témoins 4f77afc1 (« 343 Byrontron »
      nommé) et copie décalée « 43 Ritzy » (reste sans nom). Mesure parc : vies de bot nommées en plus,
      zéro nom humain changé (mesure faite, voir Journal). Relecture par levelup-1f : `[!]` à faire par levelup-1f,
      hors de cette session.
- [!] Gate : `go build ./... && go vet ./...`, `go test ./...` (apps/go-api), vitest du rejeu, `make check-types`.
      Tout vert sauf `archlint` `TestToutSiteDuRegistreExiste` : ancre périmée dans le registre des replis du
      décodeur (`internal/facts/fallback/registre_objectifs.go:302` porte `unSi(auDernierNumero)`, le code dit
      `unSi(tf.auDernierNumero)`). Correction demandée à levelup-1f dans c2-manches (gel) ; gate rejoué en E2.

## E2 — Gel et tag du décodeur

- [ ] `feat/c2-manches` fusionnée (signal de levelup-1f) ET règle des croix des morts sans type de
      levelup-d0 fusionnée (« lâcher de commande OU compte à rebours de réapparition », accord du user du
      15/11 relayé par levelup-dd) ; aucune autre fusion jusqu'au tag.
- [ ] Rebase E1 sur le gel ; miroir du schéma 122 ; fixtures de contrat refigées.
- [ ] `go test ./...` du décodeur avec `DECODEUR_FILM_RACINE_DONNEES` = dépôt principal, sur le commit à taguer.
- [ ] `replay-equiv` : écarts attribués, `-update`, seconde passe sans `-update` 20/20 identiques ;
      références commitées dans le décodeur (PR, CI verte, squash).
- [ ] Gate du corpus : aucune PERTE ; tout FAUX nouveau instruit (repli neuf = FAUX) ; décision du user
      inscrite dans `[[decision]]` de `config/replay_corpus.toml` le cas échéant.
- [ ] Tag `v1.3.9` + release GitHub (`--generate-notes --verify-tag`, latest), corps qui dit chaque
      changement de sortie depuis v1.3.8 (117 à 122, médailles).

## E3 — LevelUp sur v1.3.9 publiée

- [ ] `go.mod` / `go.sum` sur `v1.3.9` (jamais une pseudo-version), GOWORK retiré.
- [ ] Gates SANS GOWORK : suite Go, `replay-equiv` (20/20), intégration du rejeu, vitest du rejeu, `make gate-push`.
- [ ] Notes de version et changelog FR/EN (ligne « Briques publiées »), `.ai/thought_log.md`.
- [ ] Rebase sur la tête de `release/v8.2.0`, poussée sur CI verte ; SHA à levelup-dd.
- [ ] Prévenir levelup-d0 (croix des morts sans type) ; artefact des vérifications (lignes du rejeu + médailles).

## E4 — Parc (gestes à accord du user)

- [ ] Régénération du parc, UNE fois, sur accord explicite du user ; serveur local arrêté ; compte de films
      avant et après (1 614).
- [ ] `.ai/annexes_v8_1_2/gestes_prod_8_2_0.md` G6 et G7 mis à jour (dossier d'attente du VPS à renvoyer :
      geste du user, rien vers le VPS sans lui).

## Découvertes

- (levelup-1f, 2026-11-15) `cmd/replay-equiv`, étape `score` : ne couvre ni `scoreTimeline.players[]` ni
  `players[].stats` ; un défaut d'association ou de manches du tableau ne se voit que dans l'empreinte opaque
  `artifact`. À traiter hors de cette montée : ajouter une étape (ou étendre `score`) avec l'empreinte des
  joueurs du tableau et de leurs stats. Non traité ici (règle 7) ; à porter au backlog à la clôture.
- Le test de fixtures de contrat du décodeur écrit sous `<racine de données>/apps/web/...` : avec la racine
  prescrite (le dépôt principal) il aurait écrit dans le dépôt principal. Il a été lancé avec la racine = le
  worktree de montée (catalogues versionnés, aucune donnée du parc lue) ; le clone du décodeur n'a rien reçu.
  À répéter tel quel en E2.
- `scoreTimeline.players[].stats` (schéma 118) : ses tableaux `rounds` / `total` ne sont pas comblés par
  `normalizeScoreTimeline` ; aucun consommateur web aujourd'hui. À combler avec la première UI qui les lit.
- Les films `a349fea8` et `50247b26` sortent sans aucune piste nommée, avant comme après le correctif des bots
  (aucun nom lu) : cas préexistant, hors périmètre.
- Aucune UI ajoutée pour `medals`, `scoreActions`, `roundClock`, `respawnCountdowns`, `rules`.

## Journal

- 2026-11-15 : gel déplacé, accord levelup-1f et levelup-d0 : main du décodeur après c2-manches (122) puis
  le lot de d0 (123 : `endReason` released|deleted, `releaseFrame` par piste). La part web de d0 (croix des
  morts sans type, `livesMarks.ts` / `killFeedLogic.ts`) passe juste après cette montée, poussée par d0 sur
  `release/v8.2.0`. Elle ne demande pas de régénération en plus.
- 2026-11-15, E1 : miroirs 117 à 121 (`medals`, `scoreActions`, `roundClock`, `respawnCountdowns`, `rules`,
  `coverage.{medals,scoreActions,clocks}`, `coverage.seats.presencesListees`, `coverage.score.statsWithheld`,
  `scoreTimeline.players[].stats`), contrat OpenAPI et types web régénérés, clés racine du schéma web et
  frontière de nullabilité, polarité des 42 feuilles de couverture neuves (compte : échec 237, succès 212,
  neutre 186). Fixtures de contrat 121 (8 fichiers) et instantané `film-source` régénérés.
- 2026-11-15, E1, bots : `BotIdentities` passe les bots non épinglés nommés ; compteur en deux parts
  `killsource_bots_non_epingles_passes` / `_refuses`. Mesure avant/après (binaires `replay-build` des deux
  états, racines de données jetables, films lus en lecture seule) sur 21 films (18 témoins du corpus, plus
  `4f77afc1`, `ca684191` et `4db2574e`) : `4f77afc1` gagne 2 vies nommées (« 343 Byrontron », siège 1), zéro nom
  humain changé sur les 21 films, aucun autre film ne bouge.
