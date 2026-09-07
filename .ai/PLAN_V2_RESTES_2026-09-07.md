# Plan — Restes du chantier v2 rejeu/film (après gel du 2026-09-07)

> Périmètre : les faits laissés au registre à la clôture du chantier v2 (`.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`,
> handoffs `.ai/HANDOFF_V2_REJEU_FILM_2026-09-06.md` et `_2026-09-07.md`). Ce plan s'exécute sous le contrat du
> skill `plan-execution` (ordre strict, aucun report d'une action exécutable, chaque item statué `[x]`/`[~]`/`[!]`,
> zéro fix hors périmètre : toute découverte va en « Découvertes », pas dans le diff).
> Base de départ : `feat/v75` une fois `feat/v2-integ` intégré (schéma 48). Vérifier avec `git log --oneline -3`
> que la chronique de `internal/analysis/replay/document.go` porte bien 44/45/46/47/48 avant de commencer.
> Branche : `feat/v2-restes` (préfixe `feat/` obligatoire : la CI ne se déclenche pas sur `wt/`). Worktree dédié
> `LevelUp-wt-v2-restes` (le principal est partagé : jamais `git add -A`, jamais `git stash`).

## 0. Doctrine (décisions user, fermes, non rediscutables)

1. Une vie est un humain ou un bot. Une piste sans xuid est un DÉFAUT DE NOMMAGE à réparer à la source
   (`analysis/replay` lives/identity/replaybuild), jamais un état « inconnu » affiché ni dessiné.
2. Une lecture vraie du film n'est JAMAIS jetée ni rognée parce qu'un nom manque ; une identité déduite AJOUTE une
   présence, jamais n'en retire une, et n'est jamais lue comme une mort. Rien n'est inventé : ce qui ne se résout
   pas est publié « non résolu » et COMPTÉ dans la couverture (compteur servi jusqu'au contrat), avec `slog.Warn`.
3. CTF à deux drapeaux : un joueur ne porte jamais son propre drapeau ; captures = score ; une capture est un fait
   daté qui tranche sur tout recouvrement. CTF neutre : un seul drapeau, l'étiquette d'équipe n'est pas un propriétaire.
4. Fin de manche = replacement de tous les joueurs au point de départ SANS mort (vies terminées sans mort sur tous
   les slots). Hypothèse user à vérifier en R3 : les objets d'objectif (drapeaux, crânes, bombes) sont eux aussi
   replacés à la frontière de manche sans événement de lâcher/retour daté.
5. Tout changement du contenu cuit bumpe `SchemaVersion` (un seul bump par branche : 49 pour ce plan).
6. Le décodeur (`filmdec`, `himap`) ne bouge pas avant v7.5.0 : un défaut du décodeur = diagnostic + registre.
7. L'INDEX EST L'INDEX (user, 07/09) : partout où le film porte un identifiant direct (index de joueur du pied de film
   et de `PlayerIndexTable` ↔ xuid, `bid(N.0)` pour les bots, et tout lien direct index ↔ slot de statborg ou de
   bipède que le film contient), il est utilisé à 100 %, jamais remplacé par une déduction. Le pont par morts n'est
   qu'un REPLI pour les liens que le film ne donne pas directement (aujourd'hui : slot de bipède ↔ index, faute de
   champ d'identité dans `BipedPosition`), et une VÉRIFICATION des liens directs. UNE table d'identité par film,
   calculée une fois dans `replaybuild`, publiée dans l'artefact (section `identity` : index ↔ xuid, slot de
   statborg ↔ index, slot de bipède ↔ index dans le temps, avec la source et la couverture de chaque lien) ; tous
   les calques la consomment ; aucun calque ne reconstruit son propre pont (garde-rail).

## 1. Méthode commune à tous les lots

- Rouvrir chaque `fichier:ligne` cité AVANT de coder (le code a bougé) ; skill `go-features` avant tout algo neuf.
- Cuisson : `cmd/replay-build` (exécuteur borné 3 Gio), UN film à la fois, racine de travail dans le scratchpad à
  JONCTIONS vers `data/cache/film_chunks` et `data/titles` du principal, créées par
  `cmd //c mklink //J "$(cygpath -w LIEN)" "$(cygpath -w CIBLE)"` — JAMAIS `ln -s` (copie 29 Go sous MSYS) ;
  retirer les jonctions par `cmd //c rmdir` avant tout `rm -rf`. Faits `facts/<short8>.facts.json` transmis
  (technique : `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`). Jamais d'écriture sous le `data/` du principal.
- Comparaison : `cmd/replay-diff` (axe des durées inclus) élément par élément ; toute ligne « perte » est instruite
  une par une (compteur d'échec qui baisse = gain ; durée qui baisse = à expliquer par une prise datée ou une borne
  vraie, sinon P0). Gate de non-régression : `make replay-corpus-gate` (mode base par défaut : compare la cuisson
  du HEAD à celle de la révision de base ; exit 1 à la première perte ; `--reference=parc` = balayage informatif).
- Oracles indépendants, dans l'ordre : feuille de match (`facts` : morts/frags/assistances/score par joueur, score
  d'équipe = temps de portage en Oddball), calque des actions aux mêmes instants (`flag_grabs`, `flag_captures`),
  positions des pistes (le drapeau suit le porteur pendant le span).
- Tests : chaque règle ajoutée a un test qui rougit si on la retire (mutation jouée, sortie collée au journal) ; une
  défense en profondeur masque la mutation du composant qu'elle protège → tester le composant seul. Ne jamais
  renommer un test gelé dans `.ai/baselines/tests_pre_migration.jsonl`. `t.Skip` interdit.
- Gates de clôture d'un lot (tous verts, en série, préfixe `CGO_ENABLED=1`, GOCACHE dédié au worktree) :
  `go test -count=1 ./...` · `go test -tags=integration -p 1 -count=1 ./internal/api/wire/...` · `go build ./...` ·
  `CGO_ENABLED=0 go vet ./internal/domain/... ./internal/analysis/...` ·
  `golangci-lint run --new-from-merge-base=origin/main ./...` (0 issues) · cliquet de parité
  `internal/service/replayview/parity_test.go` · `make generate-types` +
  `git diff --exit-code apps/web/src/lib/api/generated.ts` · goldens (`-run GoldenAssembly -update`, écart = version
  + champs annoncés) · `make replay-corpus-gate` (0 perte). Flakes connus (registre) :
  `TestStartImport_HappyPathReturns202WithJobID` (handlers, Windows, isolé PASS ; suite `-p 1` verte) et
  `TestWorker_Run_PersistsAndACKs` (persist, runner chargé).
- Revue adversariale (skill `adversarial-review`) en fin de CHAQUE lot : contrat en 6 lignes, lentilles L4 (faux,
  perdu OU INVENTÉ servi = P0) + L6 + L3, un relecteur par worktree, 2 rondes max, corrections par l'exécuteur,
  ronde 2 sur les corrections seules ; le nombre de P0+P1 doit décroître.
- Journal : `.ai/V7.5/v2/RESTES_<lot>_<date>.md` par lot (diagnostic chiffré, cause, correctif, mutations, témoins
  avant/après, découvertes NON traitées) + entrée `.ai/thought_log.md` + registre `.ai/V7.5/REGISTRE_REPORTS.md`
  mis à jour (fermer l'entrée traitée, ouvrir les découvertes avec condition de reprise). Commit par lot, préfixe
  `fix(restes/<lot>):`, push, CI de branche par `gh run view` (jamais `gh run watch` long : sature le disque temporaire).
- Seuils : fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres ; `slog` structuré, aucune erreur avalée ; title-agnostic
  (capabilities `film.replay_artifact`, jamais de slug) ; FR sans anglicisme côté UI, parité FR/EN par typage.

## 2. Lots, dans l'ordre (chaque lot clos — gate + revue + journal — avant le suivant)

### R0 — Dette mécanique (aucun changement de sortie ; 0 bump) — [x] CLOS 2026-09-07 (lot Q6, worktree `LevelUp-wt-q6-r0`, branche `feat/v2-restes-r0`)
- [x] `internal/analysis/replay/document.go` (1 510 L sur pièces, pas 1 417 — le code a bougé depuis la rédaction du
      plan) : chronique des schémas (v2..v48) extraite dans `document_chronicle.go` (déplacement pur, `sort | diff`
      vide) ; `document.go` 553 L, `document_chronicle.go` 959 L. `usage_summary.go` (525 L) : chronique `us2`/`us3`
      extraite dans `usage_summary_chronicle.go` (déplacement pur, `sort | diff` vide) ; `usage_summary.go` 513 L,
      `usage_summary_chronicle.go` 14 L. `internal/replaybuild/replaybuild.go` (572 L, 573 L sur pièces) :
      construction de `replay.Options` extraite dans `options.go` via `buildReplayOptions` (glue de fonction Go
      incompressible — signature/return/site d'appel — mais chaque ligne de champ bit-à-bit inchangée) ;
      `replaybuild.go` 545 L, `options.go` 45 L. Effet de bord nécessaire (pas hors périmètre) :
      `film_stats_cables_guard_test.go` relisait le littéral dans `replaybuild.go` par regex — adapté pour lire
      `options.go`. `document.go`/`usage_summary.go`/`replaybuild.go` restent > 500 L après cette extraction unique
      — **statué `[!]`** : le plan ne prescrit qu'une extraction et interdit d'en faire davantage ; consigné au
      registre (`.ai/V7.5/REGISTRE_REPORTS.md`) pour un lot dédié.
- [x] `flag_carries_test.go` (575 L) scindé par responsabilité, aucun test renommé : `flag_carries_test.go` (aides
      partagées, 75 L), `flag_carries_guards_test.go` (5 tests de garde, 117 L), `flag_carries_assignment_test.go`
      (6 tests de machine à états, 197 L), `flag_carries_anon_lives_test.go` (5 tests vies anonymes/slot partagé,
      207 L — nommé ainsi et non `flag_carries_identity_test.go` : ce nom existait DÉJÀ au HEAD pour un fichier sans
      rapport, écrasé par erreur puis restauré depuis `git show HEAD:...` avant tout commit, intercepté par
      `git status` affichant `M` au lieu de `??`). Déplacement pur reconfirmé après renommage (`sort | diff` vide).
      `equipment_episodes_test.go:371-372` : bornes réelles vérifiées sur pièces (`StartFrame: 40`/`EndFrame: 60`,
      union mesurée `[40..260]`) — commentaire faux corrigé `[45..60]`/`[45..250]` → `[40..60]`/`[40..260]`.
- [x] Gate : goldens INCHANGÉS (`git diff --exit-code` sur `testdata/` des 3 paquets, EXIT=0), `SchemaVersion` = 48
      inchangé, `gofmt -l` vide, `go build ./...` (module entier) EXIT=0, `go vet` EXIT=0, `go test -count=1` vert
      sur `internal/analysis/replay/...`, `internal/replaybuild/...`, `internal/service/replayview/...` (parité
      incluse), `golangci-lint run --new-from-merge-base=origin/main` → 0 issues. Preuve de « déplacement pur » :
      concaténation triée des lignes avant/après identique — vide pour 1/2/4 ; pour 3, vide hors la glue de
      fonction Go incompressible (détail : `.ai/thought_log.md` 2026-09-07 « lot Q6 = R0 »).

### P — Changement de paradigme : registre d'identité des entités du film (bump 49 ; englobe R1, R2, R3)
Décision user (07/09) : le principe « l'index c'est l'index » vaut pour TOUTES les entités du film, pas seulement les
joueurs : objets d'objectif (drapeaux, crânes, bombes, zones), véhicules, et tous les assets de partie ou de mode
(armes au sol, équipements, socles). Chaque calque doit consommer un registre unique d'entités identifiées par
leurs identifiants du film, au lieu de reconstruire des liens par heuristique (socle le plus proche, premier
occupant d'un slot, recouvrement maximal, seuil de morts). Gros lot, mené en phases closes, chacune avec sa revue.
Impact décodeur, borné en deux temps : P2 à P5 se font UNIQUEMENT avec ce que le décodeur expose déjà (identifiants
d'objets, index de joueur, pied de film, `bid`, morts, positions) — aucun changement de `filmdec`/`himap` ; les liens
qui restent déduits sont publiés comme tels (provenance). Les flux non décodés (`ti=5`, `ManagedPropertyFilmIndex`,
états de mode, roster, horloge) font l'objet d'un plan DÉCODEUR séparé après v7.5.0 et le déplacement sous
`games/halo_infinite/film/` : un item par flux, additif (nouveau type d'enregistrement lu, jamais une réécriture),
mesuré par la part de liens directs dans la provenance, sous les garde-rails existants (corpus `gamefiles`, goldens).
- [ ] P1 — Inventaire (journal, avant tout code) : pour chaque entité que le décodeur expose déjà (index de joueur,
      slot de bipède, objet d'objectif, véhicule, arme au sol, équipement, socle/zone) : son identifiant dans le film,
      sa durée de vie (création/replacement/destruction, frontières de manche), les liens DIRECTS que le film donne
      (porteur ↔ objet, occupant ↔ véhicule, objet ↔ équipe propriétaire, objet ↔ socle) et les liens que seul un
      repli reconstruit aujourd'hui, avec le calque et la ligne de code qui porte chaque repli. Deux colonnes de
      faisabilité : « disponible dans le décodeur actuel » / « exige du travail décodeur » (après v7.5.0, §0.6).
      Axes à couvrir explicitement par l'inventaire, au-delà des joueurs et des objets (liens aujourd'hui DÉDUITS
      alors qu'une source directe peut exister) : (a) le TEMPS — origine de la frise et grille de frames (aujourd'hui
      dérivées du calage du fil des morts : `resolveOriginMs`, `originResolved`, grille plus courte que les
      enregistrements) : quelle horloge directe le film porte (états de partie, tampons des enregistrements) ;
      (b) les MANCHES — bornes aujourd'hui calculées par consensus sur les trains de score (`RoundBounds`) : quel flux
      d'état de mode donne les frontières directement ; (c) les ÉQUIPES — équipe par index de joueur telle que le film
      la porte (joueurs entrés en cours, changements d'équipe), la feuille en vérification ; (d) le ROSTER — entrées et
      sorties de joueurs (index, instant) comme source directe de l'occupation des slots, au lieu des trous de pistes ;
      (e) les BOTS — identifiant stable `bid(N.0)` porté par le registre et par le document servi, la jointure web se
      faisant sur l'identifiant et non sur le nom nu (égalité de chaîne) ; (f) les CATALOGUES DE CARTE — socles,
      spawns, zones référencés par identifiant d'asset (`himap`, Forge) et non par géométrie ; (g) la PROVENANCE —
      chaque lien du registre porte sa source (`direct` / `catalogue` / `déduit` / `non résolu`) publiée dans
      l'artefact, pour que l'UI puisse l'afficher et que le gate corpus refuse toute régression d'un lien direct vers
      un lien déduit ; (h) le MULTI-TITRE — types d'entités canoniques (`internal/games/canonical`) pour qu'un autre
      titre puisse alimenter le même registre par son adapter.
- [ ] P2 — Registre des joueurs (= R1 + R2) : table d'identité unique (index ↔ xuid ↔ slots de statborg et de bipède
      dans le temps), publiée (`identity`), source et couverture par lien, lien direct à 100 %, pont par morts en
      repli et vérification, élimination sur le roster, rien de jeté. Migration des lecteurs joueurs (actions,
      portages, épisodes, zones, fermetures, usage de session) vers la table ; garde-rail contre tout pont maison.
- [ ] P3 — Registre des objets d'objectif (= R3) : chaque drapeau/crâne/bombe/zone identifié par son objet du film,
      avec équipe propriétaire et socle tels que le film ou le catalogue les donnent (jamais « le socle le plus
      proche » quand l'objet est connu), cycle de vie par manche (§0.4 : replacement sans événement daté → fermeture
      `roundEnd`), une seule machine à états partagée par les trois calques ; les liens porteur ↔ objet viennent des
      enregistrements d'attachement du film quand ils existent, la géométrie n'étant qu'une vérification.
- [ ] P4 — Registre des véhicules et assets : véhicule identifié par son objet (pas par le premier occupant d'un slot),
      occupant ↔ véhicule dans le temps, armes au sol et équipements par objet ; migration des calques `vehicle_rides`,
      ramassages, épisodes d'équipement ; `slotCollisions` cesse d'être un motif d'ambiguïté quand l'objet est connu.
- [ ] P5 — Clôture du paradigme : `make replay-corpus-gate` (0 perte), balayage informatif du parc, oracle feuille de
      match sur tous les témoins, chronique 49, garde-rail « aucun calque ne construit de lien d'identité hors du
      registre » (allowlist datée), doc `docs/` (FR et EN) de la section `identity` de l'artefact.
Les lots R1, R2, R3 ci-dessous restent la description détaillée des faits que P2 et P3 doivent fermer ; ils ne
s'exécutent pas séparément si P est retenu. S'il est différé, R1 → R3 s'exécutent tels quels.

### R1 — Actions d'objectif écartées par le pont statborg (bump 49 ; ⊂ P2)
Fait : `3372e7eb`, 35 actions d'objectif sur 76 restent `unpublished` ; elles appartiennent aux 2 joueurs à 0 mort
(roster désormais 8/8 grâce à `Options.RosterXUIDs`, mais le pont des actions exige `deathInstantMin = 3`).
Principe (user) : UNE ACTION EST UNE ACTION, que son auteur meure ou non. Le seuil de morts n'est pas une propriété
des actions : c'est un artefact de la façon dont le pont d'identité a été construit (les instants de mort sont le
seul point commun exploité entre la feuille, qui nomme les joueurs, et le film, qui ne connaît que des index de
joueur). Un joueur avec trop peu de morts ne peut pas être apparié par cette voie, il n'est donc pas nommé, et
l'action, qui exige un nom pour être publiée, est jetée. Le défaut est dans l'identité, jamais dans l'action :
l'action doit toujours être publiée (sous l'identité résolue, sinon « non résolue » et comptée), et l'identité
doit venir d'autres voies quand les morts manquent (élimination sur le roster, table d'index joueur du film).
C'est la même racine que le premier lot du chantier (ports de drapeau des porteurs à moins de trois morts) : R1 est
le dernier lecteur encore cadencé sur le pont par morts pour son identité.
- [ ] Mesure : sur les 7 témoins du manifeste `config/replay_corpus.toml` + `3372e7eb` + `c0a82e88`, relever
      `coverage.objectives.{available,attached,unpublished,noSlot}` ; identifier la fonction qui pose le seuil (grep
      `deathInstantMin`) et son appelant dans `replaybuild` (`identifiedEvents`, `pontParManche`).
- [ ] Inventaire des identifiants (avant toute conception, journal) : pour chaque flux du film consommé par le rejeu
      (actions d'objectif/statborg, positions de bipède, morts, ramassages, véhicules, équipement), QUELLE clé il
      porte (index de joueur, slot de statborg, slot de bipède, xuid) et QUELS liens directs le film fournit entre
      ces clés (pied de film, `PlayerIndexTable`, `bid(N.0)`, `ManagedPropertyFilmIndex`, `ti=5`, en-têtes de roster :
      `.ai/V7.5/README.md` et notes de rétro-ingénierie). Tableau clé → source directe → lien manquant.
- [ ] Conception (doctrine §0.7) : une table d'identité unique par film dans `replaybuild`, publiée dans l'artefact
      (section `identity`, servie jusqu'au contrat, avec source et couverture par lien) ; les actions sont
      nommées par le lien DIRECT index ↔ xuid à 100 % ; le pont par morts (`deathInstantMin`) ne sert plus qu'aux
      liens sans source directe, et vérifie les liens directs (désaccord → `slog.Warn` + compteur, jamais un nom
      inventé) ; un index sans lien direct ni repli se résout par élimination sur le roster (`resolvedByElimination`)
      ou reste « non résolu » ET PUBLIÉ (l'action n'est jamais jetée). Garde-rail : test qui interdit à un calque de
      reconstruire un pont (allowlist datée des seuls producteurs de la table).
- [ ] Tests par mutation : élimination retirée → rouge ; deux candidats pour un index → reste non publié (rouge si on
      en choisit un) ; film mono-manche entièrement nommé → identique hors numéro.
- [ ] Témoins : `3372e7eb` unpublished 35 → 0 (ou résidu expliqué joueur par joueur), actions par joueur = feuille
      (`flag_captures`, `flag_steals`, zones) ; `c0a82e88` `noSlot` 69 → ? (chaque baisse = une action nommée, vérifiée
      par la feuille) ; `fb1a1a72` (3 manches) sans perte ; `bf15f7ab` (Slayer) identique hors numéro.
- [ ] Bump 49 + chronique + ratchet + golden ; gates ; revue ; journal ; registre (entrée `3372e7eb` fermée).

### R2 — Vies sur un slot que nulle mort ne termine (même bump 49 ; ⊂ P2)
Fait : `d9781168` (Oddball, à manches) : 19 vies sans nom, toutes sur UN slot sans aucune vie nommée = un joueur qui
ne meurt jamais de tout le match (doctrine §0.4 : les vies se terminent aux frontières de manche sans mort).
- [ ] Vérification (1 h, avant tout code) : croiser le slot sans nom avec la feuille : le joueur à 0 mort est-il
      unique ? Ses frags/assistances de la feuille se retrouvent-ils sur les kills attribués à ce slot (calque des
      fermetures) ? Résultat au journal.
- [ ] Sources d'identité sans mort, par ordre de coût : (a) élimination sur le roster (un seul xuid sans slot, un seul
      slot sans nom) ; (b) `PlayerIndexTable` / `ManagedPropertyFilmIndex` / `ti=5` (jamais branchés : voir
      `.ai/V7.5/README.md` et les notes de rétro-ingénierie ; diagnostic seul si cela exige de toucher `filmdec`).
- [ ] Implémentation dans la passe de nommage (`unnamed_lives.go`) : nouvelle cause `byElimination` comptée et
      journalisée ; jamais si deux candidats ; les vies ainsi nommées portent `deduced = true` (pas une mort).
- [ ] Tests par mutation ; témoins : `d9781168` 19 → 0 (ou résidu expliqué), temps de portage par équipe ne baisse
      pas (172,5 / 158,8 s minimum ; feuille 191 / 196), `51ebbc0f` 8 → ?, un film entièrement nommé identique.
- [ ] Gates ; revue ; journal ; registre.

### R3 — Reset des objets aux frontières de manche (même bump 49 ; ⊂ P3)
Fait : `64e8adfa` (CTF 2 manches) : `closedOverlaps = 10`, états `enJeu`/`sol` périmés (un `flag_returns` invisible
d'`assignFlags`), 7 fautes d'attribution avant l'invariant, machine à états des drapeaux dupliquée
(`assembleFlagLives` vs `flag_assign.go`).
- [ ] Vérification de l'hypothèse §0.4 sur pièces : positions des objets drapeau/crâne/bombe à la première frame de
      chaque manche (retour au socle/spawn sans événement daté) sur `64e8adfa`, `fb1a1a72`, `51ebbc0f`, `d9781168`.
- [ ] Conception : à chaque borne de manche (`objectiveevents.RoundBounds`, source unique) : fermer les portages
      ouverts avec une raison DATÉE `roundEnd` (nouvelle raison servie, jamais `dropped`), remettre `enJeu`/`sol`
      de chaque objet à son état de spawn, recommencer la machine à états. UNE machine à états partagée par les
      trois calques (drapeau, crâne, bombe) : extraire, migrer les copies, poser un garde-rail (test grep) contre
      une seconde copie (règle « ≤ 2 copies d'un même pattern »).
- [ ] Tests par mutation (reset retiré → recouvrement ; raison `roundEnd` publiée `dropped` → rouge).
- [ ] Témoins : `64e8adfa` closedOverlaps 10 → 0, `unresolved` 0, 0 portage sur son propre drapeau, captures = score,
      aucune durée par joueur en baisse hors fermeture datée à la borne de manche ; `fb1a1a72` (3 manches) ;
      `bcb6d393` (mono-manche) identique hors numéro ; `d9781168` (crâne) temps de portage ne baisse pas.
- [ ] Gates ; revue ; journal ; registre (entrées `64e8adfa`, machine à états, `enJeu` fermées).

### R4 — Écart résiduel aux compteurs de la feuille sur `51ebbc0f` (diagnostic ; bump seulement si correctif)
Fait : après le lot pont (schéma 48), écart cumulé K/D/A 69 (avant 94), les 8 joueurs sous la feuille.
- [ ] Par joueur et par manche : frags/morts/assistances du document contre la feuille ; localiser les événements
      manquants (après la dernière frame de la grille ? hors fenêtre de manche ? morts non appariées : lire
      `coverage.bridge.deathOffsetMatched` et le nombre de morts du fil).
- [ ] Verdict : source (film incomplet : registre + `slog`) ou lecteur (correctif, tests, témoins, bump 49).

### R5 — Re-vérification au schéma 48/49 : CTF multi-manche, calques VIP et crâne
- [ ] `make replay-corpus-gate --reference=parc` (informatif) + contrôle par la feuille sur `fb1a1a72` (3 manches),
      `51ebbc0f`, `64e8adfa` : captures/vols par joueur = feuille ; VIP (`vip_crown.go`) et crâne (`skull_carries.go`)
      sur un film de chaque variante : 0 portage perdu, durées ≥ parc, aucune identité inventée.
- [ ] Fermer ou rouvrir (avec chiffres) les entrées « CTF multi-manche » et « calques VIP/crâne » du registre.

### R6 — Constats P2 de l'audit et web — CLOS le 2026-09-07 (lot `feat/v2-restes-r6`, M2 du plan d'orchestration)
- [x] Les 5 P2 de `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md` (section « Constats retenus », gravité P2) :
      traiter un par un (correctif + mutation) ou fermer avec preuve de non-lieu ; colonne « Décision/état ».
      P2-1/P2-2 : confirmés sur pièces, racine dans `filmdec` (décodeur gelé §0.6) — correctif hors périmètre,
      diagnostic + registre (`.ai/V7.5/REGISTRE_REPORTS.md`). P2-3 : non-lieu, déjà corrigé par `f1b4f4ee5`
      (`fix(manches/MANCHES-R1)`, 2026-09-07 00:05, déjà sur `feat/v75`). P2-4, P2-5 : corrigés, mutations jouées.
      Détail : `.ai/V7.5/v2/RESTES_R6_2026-09-07.md`.
- [x] Web : `apps/web/src/features/match-replay/model/equippedLogic.ts` `drawnSwapAt` (lecture non bornée à la
      vie en cours, le code a bougé depuis la rédaction du plan — la fonction est côté `match-replay`, pas
      `lib/replay`) : bornée via `currentLifeOf` + `trackWindow` (même patron que `loadoutAt`/`abilityAt`,
      correctif P0-2) ; tests vitest par mutation (`equippedLogic.test.ts`) ; gates `tsc -b`, eslint,
      `vitest run --pool=forks` : 0 erreur, 653 fichiers / 6972 tests verts.

### R7 — Calage du fil des morts : budget de candidats (registre D3 du lot pont) — [x] CLOS 2026-09-07 (lot M3, worktree `LevelUp-wt-m3-calage`, branche `feat/v2-restes-r7`)
Fait figé par `TestUnAmasPlusGrosQueLeVraiCalageALARMEAuLieuDeSeTaire` (`pont_marge_test.go`) : un amas de morts
distinctes plus nombreux que le vrai calage remplit le budget de 3 candidats ; l'alarme se déclenche, le calage rendu
est faux.
- [x] Correctif : budget adaptatif (candidats jusqu'à ce que le meilleur compte réel dépasse le meilleur vote) ou
      dédoublonnage par fin de vie en plus de par mort ; le test de documentation doit être RETOURNÉ (il rougit
      quand le correctif fait mieux : le mettre à jour pour exiger le vrai calage).
      **DÉDOUBLONNAGE PAR FIN DE VIE retenu** : le diagnostic (journal `.ai/V7.5/v2/RESTES_R7_2026-09-07.md` §1)
      montre que le vote comptait les MORTS appariables quand l'affinage apparie 1:1 (chaque fin de vie servie une
      seule fois, `countDeathMatches`) — un amas de 20 morts distinctes déposait donc 20 voix sur CHAQUE fin de vie
      isolée du vrai calage pour UNE seule paire réalisable, et ces paniers fantômes remplissaient le budget. La voix
      d'un panier devient `min(morts distinctes, fins distinctes)` (borne de Hall/König, majorant EXACT de
      l'appariement que l'affinage mesure) via le nouvel helper `paniersParPivot` de `lives.go`, appelé une fois par
      côté. **Aucun seuil nouveau** : `deathOffsetCandidats` reste 3, `deathOffsetMargeMin` 2, `deathMatchWindowMS`
      150. Le budget adaptatif est ÉCARTÉ : il aurait laissé le classement faux et exigé d'affiner les 15 paniers
      fantômes avant d'atteindre le vrai calage. Fixture adversariale : `[216300 233625 -190050]` / `off=216350 n=2`
      → `[199950 216300 233625]` / `off=200000 n=15 second=2`. Test RETOURNÉ et renommé
      `TestUnAmasPlusGrosQueLeVraiCalageNEmportePasLeBudget` (exige le vrai calage, teste `voteDeathOffsets` SEUL,
      vérifie que l'alarme se TAIT) ; mutation jouée, sortie au journal §3. **Baseline INCHANGÉE** : vérifié sur
      pièces, le test renommé n'est pas dans `.ai/baselines/tests_pre_migration.jsonl` (datée 2026-06-26, le test
      date du 2026-09-07). Effet du ratchet lint corrigé à la source : `unparam` a sorti « `k` always receives
      `deathOffsetCandidats` » sur `voteDeathOffsets` — attribution VÉRIFIÉE (le même lint sur un worktree détaché
      à la base rend 0 issues), le paramètre de budget est donc retiré (4 appelants, tous sur la même constante).
- [x] Neutralité : `d9781168` et `51ebbc0f` identiques hors numéro (marges inchangées) ; bump 49.
      **AUCUN GOLDEN NE BOUGE, DONC PAS DE BUMP** : `go test ./internal/analysis/replay/ -run Golden -update` puis
      `git diff --exit-code -- internal/analysis/replay/testdata/` → EXIT=0 ; `SchemaVersion` reste **48** et aucune
      entrée n'est ajoutée à `document_chronicle.go`. C'est le résultat attendu : là où le vote localisait déjà le
      vrai calage, `min(morts, fins)` vaut le compte des morts. Le 49 reste donc disponible pour le premier lot qui
      changera réellement le contenu cuit (P2). **`make replay-corpus-gate` et les deux témoins sont à jouer par le
      SUPERVISEUR** (ce worktree n'a aucun film) — commandes exactes au §6 du journal.

### R8 — Flakes CI hors rejeu (doctrine : tout rouge se répare, même préexistant)
- [ ] `internal/api/handlers` `TestStartImport_HappyPathReturns202WithJobID` : le job d'import asynchrone survit au
      retour HTTP (`jobs.Store` écrit après la fin du test → `TempDir RemoveAll`) : attendre la fin du job dans le
      test (ou fermer le store) ; preuve : `-count=20` vert sous charge (`-p 8`).
- [ ] `internal/persist` `TestWorker_Run_PersistsAndACKs` : assertion de système de fichiers sur runner chargé
      (51,9 s en CI contre 0,08 s en local) : identifier l'attente implicite, la remplacer par une synchronisation.

### R9 — Release (séquence Notion « Backlog LevelUp », dans l'ordre ; prévenir le user avant tout push sur `main`)
- [ ] Re-cuisson du parc au dernier schéma (`backfill-replay`, un film à la fois, verrou `filmproc`), puis recompter
      la jointure des bots film ↔ tableau (nom nu, égalité de chaîne : `scratchpad/review/BOT_SANS_EQUIPE.md`).
- [ ] `backfill-medailles-feed` ; puis déplacement du décodeur sous `games/halo_infinite/film/` APRÈS v7.5.0.

## 3. Découvertes (à consigner ici, ne pas traiter dans le lot courant)

- **[R7, 2026-09-07] `lives.go` était DÉJÀ à 509 L (au-dessus du seuil de 500) et passe à 543 L**
  avec le helper `paniersParPivot`. Dette accrue de 34 L, non créée ; aucune scission n'est
  prescrite par R7 et la règle 7 du contrat interdit le fix hors périmètre. Rattaché à l'entrée
  de registre ouverte par R0 sur les fichiers > 500 L du paquet, qui liste désormais `lives.go`.
- **[R7, 2026-09-07] Le coût du vote du calage double** (deux passes `|morts| × |fins|` au lieu
  d'une). Ordre de grandeur négligeable devant l'affinage, mais **non mesuré sur un film réel** :
  aucun film dans le worktree M3. À relever au prochain lot qui cuit un film BTB (P2) si la
  durée de cuisson bouge.
- **[R7, 2026-09-07] Chemin périmé au registre** : l'entrée D3 citait le test dans
  `pont_muet_test.go` alors que PONT-R2 l'avait déplacé dans `pont_marge_test.go`. Corrigé en
  fermant l'entrée ; aucune autre occurrence.

## 4. Reprise de session

Relire ce plan et `plan-execution` ; `git log --oneline -5` sur `feat/v2-restes` ; reprendre à la première case non
statuée du lot courant ; ne jamais commencer R(n+1) avant la clôture de R(n) (gate + revue + journal + registre).
