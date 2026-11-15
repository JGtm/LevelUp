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
- [x] Gate : `go build ./... && go vet ./...`, `go test ./...` (apps/go-api), vitest du rejeu, `make check-types`.
      (rejoué vert en E2, ancre corrigée dans #27 ; texte d'origine : tout vert sauf `archlint` `TestToutSiteDuRegistreExiste` : ancre périmée dans le registre des replis du
      décodeur (`internal/facts/fallback/registre_objectifs.go:302` porte `unSi(auDernierNumero)`, le code dit
      `unSi(tf.auDernierNumero)`). Correction acceptée par levelup-1f dans le commit de c2-manches (gel) ; gate rejoué en E2.)
      Sièges du roster de `4f77afc1` changés par le correctif des bots : vérifiés sur pièces par levelup-1f,
      ATTENDU (sans les bots, Narotlcs prenait à tort la place 1 ; avec eux il ouvre son siège listé 26).
      À dire dans les notes (E3) : places des arrivants corrigées quand des bots non épinglés sont déclarés.

## E2 — Gel et tag du décodeur

- [x] (GEL 2026-11-15 : main du décodeur 52a72a8, #27 puis #28 ; levelup-1f ne fusionne plus rien avant le tag)
      `feat/c2-manches` fusionnée (signal de levelup-1f) ET règle des croix des morts sans type de
      levelup-d0 fusionnée (« lâcher de commande OU compte à rebours de réapparition », accord du user du
      15/11 relayé par levelup-dd) ; aucune autre fusion jusqu'au tag.
- [x] Rebase E1 sur le gel (E1 commité sur le worktree, le clone est au gel 0c3cc96, schéma 124) ; miroirs des schémas 122, 123 et 124 ;
      fixtures de contrat 123 refigées (racine = le worktree), `film-web-snapshot` régénéré.
- [x] `go test ./...` du décodeur avec `DECODEUR_FILM_RACINE_DONNEES` = dépôt principal (lecture seule), sur ed09770 (0c3cc96 plus
      les références refigées) : 31 paquets ok ; seuls échecs : les quatre `TestContractFixtures*` (attendu sous cette racine).
- [x] `replay-equiv`, racine jetable `gate_work_equiv`. Sur 0c3cc96 : 20 différents sur 20, une seule étape (`artifact`) sur 79, voir Journal ;
      `-update` dans le clone (branche `chore/equiv-v139`, commit local ed09770, non poussé), seconde passe sans `-update` : 20 identiques sur 20.
      Références commitées dans le décodeur : PR, CI verte et squash restent à ta charge.
- [!] Gate du corpus : décisions du user inscrites (21 `[[decision]]`, voir Journal) ; 0 FAUX non accepté, 0 décision sans effet ;
      RESTE 3 témoins en PERTE (`bcb6d393`, `c75f33b8`, `4f77afc1`) pour des mesures que le journal ne classe pas « à accepter » :
      à instruire. `f75e7053` : purgé du registre et jamais cuit dans le parc, ne peut pas entrer.
- [ ] Tag `v1.3.9` + release GitHub (`--generate-notes --verify-tag`, latest), corps qui dit chaque
      changement de sortie depuis v1.3.8 (117 à 122, médailles).

## E3 — LevelUp sur v1.3.9 publiée

- [ ] `go.mod` / `go.sum` sur `v1.3.9` (jamais une pseudo-version), GOWORK retiré.
- [ ] Gates SANS GOWORK : suite Go, `replay-equiv` (20/20), intégration du rejeu, vitest du rejeu, `make gate-push`.
- [ ] Notes de version et changelog FR/EN (ligne « Briques publiées »), `.ai/thought_log.md`.
- [ ] Rebase sur la tête de `release/v8.2.0`, poussée sur CI verte ; SHA à levelup-dd.
- [ ] Prévenir levelup-d0 (croix des morts sans type) ; artefact des vérifications (lignes du rejeu + médailles).

## E4 — Parc (gestes à accord du user)

- [ ] (ACCORD DU USER 2026-11-15 : « tu as mon accord pour regen les rejeux » ; à lancer après E3, binaire sur
      v1.3.9, sessions paires prévenues) Régénération du parc, UNE fois, sur accord explicite du user ; serveur local arrêté ; compte de films
      avant et après (1 614).
- [ ] `.ai/annexes_v8_1_2/gestes_prod_8_2_0.md` G6 et G7 mis à jour (dossier d'attente du VPS à renvoyer :
      geste du user, rien vers le VPS sans lui).

## Découvertes

- Le gate du corpus retire GOWORK pour compiler HEAD et base (`compilerReplayBuild`, `environnementSansGOWORK`) : sous GOWORK il
  juge le décodeur PUBLIÉ, jamais le clone. Pour juger le clone, le mode `--reference=parc` avec `GOFLAGS=-modfile=<copie de go.mod
  hors dépôt portant un replace vers le clone>` (le `go.mod` versionné n'est pas touché). Le mode base, lui, compile aussi la base
  avec ce GOFLAGS : inutilisable ainsi. À régler avant E3 (le gate se lance alors sans GOWORK sur v1.3.9 publiée).
- Le gate compile l'outil `replay-verite` de la base avec le GOWORK hérité : « registre des replis d'avant INCONNU », donc tout
  repli est jugé nouveau (FAUX). Le gate sous GOWORK ne peut pas rendre un verdict R-1 exact.
- Un gate qui lit un parc jetable : la racine `gate_work_parc` a perdu son dossier `film_chunks` en cours de gate (disparu sans cause
  établie ; le principal est intact, 1 614 films) ; les sept films concernés ont été recopiés et rejoués.

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

- 2026-11-15, gate du corpus instruit par levelup-1f sur pièces (gate_corpus3) :
  - a521164d : DÉFAUT du lot 122 (build 1.4.1, bande statborg non prouvée, compteurs nommés faux). La correction
    (règle générale) arrive en PR sur main du décodeur AVANT le tag ; le gel est rouvert pour elle seule. Relancer
    le gate ensuite.
  - 50247b26 et bfecd02b, et le R-1 de 4f77afc1 : replis nommés des lots 118 et 122, jugés neufs faute de
    registre de base. À accepter.
  - 4f77afc1 : O-S3 FP 100, écart de l'API (1 frag et un score de 0, joueur partant), à accepter.
    `refusedByRoster` 12 -> 40 : le lot 118 lit tous les sièges ; rien n'est publié.
  - 11de8353 : compteurs neufs des lots 120, 121 et 123, et toMax du lot 119, à accepter. pickups.origin 15 -> 14 :
    artefact du gate (racine de parc jetable sans `reference/map_weapon_pads.json`, donc map_absent). Copier
    `reference/` dans la racine au prochain passage.
  - Second passage (gate_corpus2), instruit par levelup-1f :
    - a349fea8 : même cas que 50247b26 (rien de publié), à accepter.
    - 084a804d : DÉFAUT, rattaché au point 1 (assistances au double, 8 au lieu de 4, comme a521164d 6 au lieu de 3).
      La correction couvre les deux films, mesurée sur les 18 témoins.
    - `repli_emission_hors_domaine_jetee` sur bcb6d393, c75f33b8, 111fa685, 60ae07c4 et 396cfc92 : rangé sous le
      point 3, aucun FP, gains nets.
    - Les deux vies de « Truly Elusive » rendues au bot (4f77afc1) : ce xuid est absent de match_participants. Gain.
  - Accord du user relayé par levelup-d0 (« Oui ok je suis d'accord avec sa recommandation, mais ajouter au backlog
    d'investiguer sur ces écarts restants… ») : CONFIRMÉ par le user dans cette session le 2026-11-15 (« Oui ok »).
    Portée : les écarts classés « à accepter » ci-dessus ; les deux défauts (a521164d, 084a804d) restent à
    corriger. Décisions inscrites dans `[[decision]]` APRÈS le passage du gate avec la correction, sous les noms
    de mesure exacts du rapport. La ligne de backlog est tenue par levelup-d0.

- 2026-11-15 : 122 fusionné (décodeur a36bf67, #27, ancre du registre corrigée dedans). Sortie : joueur de chaque
  ligne du tableau lu par le composant 57 et les listes de session, `stats` publié sur les films à plusieurs
  manches, `coverage.score.roundsUnplayed` neuf, dernière mort d'un partant conservée. Miroir LevelUp à faire :
  `coverage.score.roundsUnplayed` (int[] facultatif) et version 122. Goldens d'assemblage refigés avec les données
  de l'hôte : `go test ./...` du décodeur avec `DECODEUR_FILM_RACINE_DONNEES` obligatoire sur le commit à taguer.
  Miroirs 122 et 123 faits ensemble après la fusion du 123 (une seule régénération des fixtures).

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
- 2026-11-15, E2 (gel 52a72a8 : #27 schéma 122, #28 schéma 123) : miroirs 122 (`coverage.score.roundsUnplayed`) et 123
  (`tracks[].endReason`, `tracks[].releaseFrame`, `coverage.tracks.lifeEnds` = `LifeEndCoverage`) ; noms Go alignés sur le décodeur ;
  contrat OpenAPI, types web, clé de couverture du schéma web, polarité des 8 feuilles neuves (compte : échec 240, succès 214, neutre 189),
  `contracttest` (type `LifeEndCoverage`), plafond de surface 329 ; fixtures de contrat 123 et instantané `film-source` régénérés.
  Le clone du décodeur et le dépôt principal n'ont rien reçu.
- 2026-11-15, E2, décodeur : `go test ./...` sur 52a72a8 (racine = dépôt principal en lecture) : seuls échecs les quatre `TestContractFixtures*`.
  `replay-equiv` (racine jetable) : 20 identiques sur 20.
- 2026-11-15, E2, gate du corpus. Run 1 (`--reference=base`, base 010e65bbc4) : invalide pour le décodeur (HEAD compilé sans GOWORK = v1.3.7) ;
  il juge le seul correctif des bots sur v1.3.7. Runs 2 et 3 (`--reference=parc`, `GOFLAGS=-modfile` vers une copie hors dépôt de `go.mod`
  portant un replace vers le clone, parc et verrou jetables) : HEAD au schéma 123 sur 18 témoins sur 19 ; `f75e7053` absent (match absent
  du registre). Banc de vérité : 7 ok (`fb1a1a72`, `d9781168`, `bf15f7ab`, `51ebbc0f`, `0797ce72`, `e5adf7b2`, `11de8353`), 11 FAUX, 0 MANQUE.
  FAUX : repli `repli_emission_hors_domaine_jetee` 0 vers N sur `bcb6d393` (29), `c75f33b8` (17), `111fa685` (16), `4f77afc1` (27),
  `60ae07c4`, `bfecd02b`, `396cfc92` (1 chacun) ; repli `repli_bande_statborg_par_defaut` 0 vers 1 sur `a349fea8`, `a521164d`, `50247b26` ;
  O-K3 faux positifs 0 vers 4 sur `084a804d` (manques 72 vers 0) et 0 vers 3 sur `a521164d` ; O-S3 faux positifs 0 vers 100 sur `4f77afc1`
  (manques 21 735 vers 850). Registre des replis d'avant inconnu : tout repli est jugé neuf. Le repli de bande vient du 118 ; l'émission hors
  domaine monte avec la lecture du tableau de tous les sièges (118), le repli existe depuis v8.1.0. PERTES du filet `replaydiff` : les leaves
  de couverture neuves (`coverage.clocks.*`, `scoreActions.*`, `lifeEnds.*`) classées échec passent de absent à N ; `roster.presence.toMax`
  disparu (119, présence exacte) ; `seats.relaisBornes` en baisse ; `4f77afc1` : deux vies de « Truly Elusive » rendues au bot « 343
  Byrontron » (119 plus correctif des bots), `pickups.origin` 15 vers 14 sur `11de8353`. Aucune décision inscrite.
- 2026-11-15, E2, gate LevelUp sous GOWORK : `go build`, `go vet` verts ; `go test ./...` : tout vert, `archlint` compris
  (`TestToutSiteDuRegistreExiste` vert), après ajout de `commande_*.bin.gz` à `binairesCouverts` ; vitest du rejeu 350 fichiers et
  4 794 tests verts (un premier passage a échoué sur des délais de 5 s et des démarrages de worker sous charge, rejoué avec 3 workers) ;
  `tsc -b` vert. Jonction `apps/web/node_modules` retirée.
- 2026-11-15, E2 bis (gel rouvert, #29 0c3cc96, schéma 124, `objectives-2026-11-15.2`) : `replay-equiv` sur 0c3cc96, racine jetable :
  20 différents sur 20, une seule étape sur 79, `artifact`, jamais les 78 autres (dont `score`). Écart par film (octets) : +12 sur
  `000d5950`, `01e1f945`, `64e8adfa`, `7344d24f`, `696a9d7c`, `53ce4390`, `51101d1d`, `a349fea8`, `50247b26`, `11de8353`, `111fa685`,
  `e5adf7b2`, `bcb6d393`, `fb1a1a72` (révision `objectives-2026-11-15.2` dans `coverage.decoder` et les entrées de `layers`, ligne de schéma 123
  vers 124 de même longueur) ; +14 sur `d9781168` et `60ae07c4`, +20 sur `9f57c612` (davantage d'entrées de `layers`) ; -24 sur `084a804d`
  et -52 sur `1c4c63c2` (assistances écartées, un repli de moins) ; -768 sur `a521164d` (`scoreTimeline.players` retiré, film sans deux
  équipes lues). Vérifié sur les artefacts des témoins du corpus (123 contre 124) : aucun autre champ ne change. `-update`, commit
  local `ed09770` dans le clone (branche `chore/equiv-v139`), seconde passe 20 sur 20 identiques. Miroir 124 : fixtures de contrat 124,
  `film-web-snapshot`, aucune forme neuve.
- 2026-11-15, E2 bis, gate du corpus (parc jetable, `--reference=parc`, replace vers le clone par `-modfile` hors dépôt, surcharge
  `reference/generated` du parc posée dans la racine de travail) : 18 témoins sur 19 cuits au schéma 124. Les défauts du lot 122 sont corrigés
  (`a521164d` et `084a804d` : plus de faux positifs O-K3). `11de8353` retrouve `spawnPointsState` established et 54 points, plus de perte
  d'origine de ramassage. `f75e7053` : absent du registre actif (2 043 matchs après la purge du 2026-10-26, 10 026 avant ; la sauvegarde
  d'avant purge le porte) et aucun artefact de parc ; hors gate. Registre des replis d'avant : l'outil du banc compilé à la base (`git archive` de
  010e65bbc4, v1.3.7 publiée, GOWORK=off) donne 130 replis ; `replay-verite -registre-avant` sur les 18 artefacts : seuls les deux replis
  jugés neufs le restent (`repli_bande_statborg_par_defaut` absent du registre ; `repli_emission_hors_domaine_jetee` présent mais de 0 à N, donc
  FAUX), identique au gate. Décisions : 21 `[[decision]]` sous les noms exacts du rapport (2 replis R-1, O-S3 de `4f77afc1`, 15 feuilles de
  couverture des lots 120, 121 et 123, `toMax` du lot 119, 2 mesures de pistes de `4f77afc1` sans xuid (`tracks.xuid/presents`, `tracks/xuids-distincts`) ; la mesure `tracks/par-xuid/<xuid>` porte un
  vrai xuid : NON inscrite, à trancher) avec la phrase du user ; les 3 décisions v1.3.7
  devenues sans effet et 3 des miennes (`refusedByRoster`, `tracks/vies-par-xuid/...`, `tracks/vies-nommees`, sans constat à requalifier)
  retirées. Passage final (avant retrait de la mesure à xuid, sans effet sur ces statuts) : 0 FAUX non accepté, 0 décision sans effet, 15 témoins ok, 3 PERTE : `coverage.seats.relaisBornes` (`bcb6d393`
  4 vers 0, `c75f33b8` 3 vers 0, `4f77afc1` 4 vers 1), les scores de deux joueurs (485 vers 200 et 220 vers 100)
  (`bcb6d393`, le banc les range en gain : publié 485 contre officiel 200, maintenant exact), `coverage.seats.sansPresence` 1 vers 2 et
  `coverage.teams.unread` 0 vers 1 (`4f77afc1`) : non classées « à accepter », rien d'inscrit.
