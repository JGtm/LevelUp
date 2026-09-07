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

### R0 — Dette mécanique (aucun changement de sortie ; 0 bump)
- [ ] `internal/analysis/replay/document.go` (1 417 L) : extraire la chronique des schémas dans `document_chronicle.go`
      (déplacement pur, commentaires intacts) ; `usage_summary.go` (525 L) : idem chronique `us1..us3` ;
      `internal/replaybuild/replaybuild.go` (572 L) : extraire la construction de `replay.Options` dans `options.go`.
- [ ] `flag_carries_test.go` (575 L) → scinder par responsabilité (identité / gardes / assignation) ;
      `equipment_episodes_test.go:371-372` : commentaire faux (`[45..60]`/`[45..250]` → `[40..60]`/`[40..260]`).
- [ ] Gate : goldens INCHANGÉS (`git diff --exit-code` sur `testdata/`), suite complète verte, lint 0.
      Preuve de « déplacement pur » : concaténation triée des lignes avant/après identique.

### R1 — Actions d'objectif écartées par le pont statborg (bump 49)
Fait : `3372e7eb`, 35 actions d'objectif sur 76 restent `unpublished` ; elles appartiennent aux 2 joueurs à 0 mort
(roster désormais 8/8 grâce à `Options.RosterXUIDs`, mais le pont des actions exige `deathInstantMin = 3`).
- [ ] Mesure : sur les 7 témoins du manifeste `config/replay_corpus.toml` + `3372e7eb` + `c0a82e88`, relever
      `coverage.objectives.{available,attached,unpublished,noSlot}` ; identifier la fonction qui pose le seuil (grep
      `deathInstantMin`) et son appelant dans `replaybuild` (`identifiedEvents`, `pontParManche`).
- [ ] Conception : nommer l'index joueur d'une action par le pont canonique DANS LE TEMPS (`ResolveSlotXUID`,
      `RoundIdentity.At`) et par le roster de la feuille (`RosterXUIDs`) : un index dont le xuid est le seul du
      roster sans slot connu se résout par élimination (compté `resolvedByElimination`) ; sinon reste `unpublished`
      et compté (jamais inventé). Le seuil de 3 morts devient une PRÉFÉRENCE d'ordre, pas une condition.
- [ ] Tests par mutation : élimination retirée → rouge ; deux candidats pour un index → reste non publié (rouge si on
      en choisit un) ; film mono-manche entièrement nommé → identique hors numéro.
- [ ] Témoins : `3372e7eb` unpublished 35 → 0 (ou résidu expliqué joueur par joueur), actions par joueur = feuille
      (`flag_captures`, `flag_steals`, zones) ; `c0a82e88` `noSlot` 69 → ? (chaque baisse = une action nommée, vérifiée
      par la feuille) ; `fb1a1a72` (3 manches) sans perte ; `bf15f7ab` (Slayer) identique hors numéro.
- [ ] Bump 49 + chronique + ratchet + golden ; gates ; revue ; journal ; registre (entrée `3372e7eb` fermée).

### R2 — Vies sur un slot que nulle mort ne termine (même bump 49)
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

### R3 — Reset des objets aux frontières de manche (même bump 49)
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

### R6 — Constats P2 de l'audit et web
- [ ] Les 5 P2 de `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md` (section « Constats retenus », gravité P2) :
      traiter un par un (correctif + mutation) ou fermer avec preuve de non-lieu ; colonne « Décision/état ».
- [ ] Web : `apps/web/src/lib/replay/…` `drawnSwapAt` (lecture non bornée à la vie en cours) : borner via
      `currentLifeOf`, test vitest par mutation ; gates `tsc --noEmit`, eslint, vitest.

### R7 — Calage du fil des morts : budget de candidats (registre D3 du lot pont)
Fait figé par `TestUnAmasPlusGrosQueLeVraiCalageALARMEAuLieuDeSeTaire` (`pont_marge_test.go`) : un amas de morts
distinctes plus nombreux que le vrai calage remplit le budget de 3 candidats ; l'alarme se déclenche, le calage rendu
est faux.
- [ ] Correctif : budget adaptatif (candidats jusqu'à ce que le meilleur compte réel dépasse le meilleur vote) ou
      dédoublonnage par fin de vie en plus de par mort ; le test de documentation doit être RETOURNÉ (il rougit
      quand le correctif fait mieux : le mettre à jour pour exiger le vrai calage).
- [ ] Neutralité : `d9781168` et `51ebbc0f` identiques hors numéro (marges inchangées) ; bump 49.

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

(vide au départ)

## 4. Reprise de session

Relire ce plan et `plan-execution` ; `git log --oneline -5` sur `feat/v2-restes` ; reprendre à la première case non
statuée du lot courant ; ne jamais commencer R(n+1) avant la clôture de R(n) (gate + revue + journal + registre).
