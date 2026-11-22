# Plan : correctif 8.2.2, rattrapage de l'historique par la synchronisation automatique

Date : 2026-11-22. Contrat d'exécution : skill `plan-execution` (avec `levelup-plan`).
Branche d'intégration : `release/v8.2.2`, créée depuis `origin/main` (fee334a039, v8.2.1).
Worktree d'intégration : `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-822`.
Origine : issue GitHub #213 (joueur sur l'application Windows 8.2.1, ~7 000 matchs, historique bloqué).

## Objectif

Toute synchronisation automatique (cycle périodique, lancement, fermeture du jeu, « Synchroniser
maintenant » du menu de la zone de notification) rattrape l'historique d'un joueur jusqu'à son premier
match, Halo Infinite comme Halo 5, puis redevient un delta. Critère observable : un test de bout en bout
sur un faux historique de 1 000 matchs montre que des cycles successifs du VRAI chemin automatique
rangent les 1 000 matchs, rangent la fin d'historique, puis que le cycle suivant ne parcourt plus qu'une
page.

## Constat (vérifié sur pièces le 2026-11-22)

- Le rattrapage livré en 8.1.2 (`sync_meta.history_complete_at`, `internal/sync/engine.go:486`,
  `internal/sync/writes.go:85-128`) ne vit que dans le moteur V1 (`SyncEngine.run`).
- Le cycle automatique passe par le pipeline V2 (`internal/scheduler/auto_sync.go` `runOnceV2`,
  ADR 0027). Sa découverte `matchListProviderV2.ListUnknownMatches`
  (`internal/sync/v2/fetchers.go:91`) s'arrête au premier match connu, au plus 20 pages, et ignore
  l'état de l'historique. Le tick manuel du menu (`cmd/server/boot_shell.go:170`,
  `RunOnceTrigger`) et le rattrapage au lancement (`internal/wakeup`) prennent le même chemin.
- Halo 5 : `livesync.Runner.RunDelta` (`internal/games/halo_5/livesync/runner.go:113`) s'arrête au
  premier connu ; le parcours complet (`livesync/backfill.go`) n'existe qu'en commande.
- Les tests du rattrapage 8.1.2 ne passaient que par le moteur V1 : aucun test ne jouait le chemin
  que l'ordonnanceur appelle vraiment. C'est le trou que ce plan ferme (étape 5).

## Décisions

- **D-1** Version 8.2.2 (correctif), depuis `main`. Après le squash de la 8.2.2 par l'utilisateur,
  `main` est fusionnée dans `release/v8.3.0` (geste du pilote, sur accord).
- **D-2** Une seule règle, une seule source : un paquet `internal/sync/historystate` porte la clé
  `history_complete_at` (nom inchangé : les joueurs déjà marqués par V1 restent marqués), sa lecture
  (`Complete(ctx, db) bool`, lecture en échec = complet, journalisé, comme aujourd'hui), son écriture
  (`MarkComplete`), le budget par passe et la borne de sûreté. V1, V2 et Halo 5 l'appellent ; plus
  aucune copie de la clé ni de la logique ailleurs (garde-rail grep, D-9).
- **D-3** Budget par passe de rattrapage : 200 matchs inconnus, `domain.DefaultSyncOptions().MaxMatches`
  (valeur unique, pas de nouvelle constante). Borne de sûreté de profondeur : 50 000 matchs (constante
  `maxHistoryMatchesPerPass` de V1, déplacée dans `historystate`).
- **D-4** Halo Infinite, V2 : quand l'historique d'un joueur n'est pas complet, la découverte ne
  s'arrête plus au premier connu : elle saute les connus, garde les inconnus dans l'ordre de l'API
  jusqu'au budget, et continue jusqu'à la page vide ou partielle (fin d'historique vue) ou jusqu'au
  budget atteint (fin non vue). Historique complet : comportement actuel inchangé (arrêt au premier
  connu, 20 pages).
- **D-5** La fin d'historique se range seulement si, pour CE joueur : la découverte a vu la fin, aucune
  erreur de découverte, la persistance du cycle a réussi, et chaque match inconnu listé pour lui a été
  persisté ou rendu « introuvable » par l'API (404/410, compté et journalisé). Toute autre erreur de
  récupération laisse la clé absente : la passe suivante reprend (les connus sont sautés). L'écriture de
  `sync_meta` se fait sous le bail d'écriture du joueur (`AcquirePlayerWriterTimeout`, règle ART n°4).
- **D-6** Halo 5 : `Runner.RunDelta`, historique incomplet = passe de rattrapage par la mécanique de
  `livesync/backfill.go` (connus sautés, persistance page par page), bornée au budget D-3 en matchs
  insérés, fin rangée à la page vide sans erreur (règle D-5). La commande `h5-backfill` garde son
  comportement (sans budget).
- **D-7** Cadence : tant qu'au moins un joueur du cycle a un historique incomplet, l'ordonnanceur
  replanifie le cycle suivant après 10 minutes (constante nommée `catchUpCycleInterval`) au lieu de
  l'intervalle réglé ; dès que tous sont complets, retour à l'intervalle réglé. Tous les profils de
  déploiement. Le résultat de cycle porte l'information par joueur (`HistoryPending`). Journal :
  une ligne par cycle qui dit combien de joueurs rattrapent et l'intervalle choisi.
- **D-8** Le tick du suivi en direct (watcher, 25 matchs) reste tel quel : il passe par V1, qui suit
  déjà la règle (via `historystate` après migration).
- **D-9** Garde-rail : un test `archlint` interdit le littéral `"history_complete_at"` hors de
  `internal/sync/historystate`.
- **D-10** (révisée le 22/11 par l'utilisateur) Bouton « Synchroniser maintenant » dans Réglages ›
  Données, visible par tout joueur, DANS la 8.2.2 : il lance le même cycle que l'automatique
  (`RunOnceTrigger`), donc le rattrapage compris (étape 4bis). L'état de l'historique à l'écran :
  décision de l'utilisateur en attente (D-15). L'aide est corrigée (étape 6) et renvoie à ce bouton.
- **D-12** (utilisateur, 22/11) Tout l'historique, sans limite de date ni de nombre. Le choix d'une
  limite est une ligne du backlog.
- **D-13** (utilisateur, 22/11) Marque-page au lieu du rebalayage : `historystate` range, par joueur,
  la position où la passe de rattrapage s'est arrêtée (décalage dans la liste de l'API et identifiant du
  dernier match traité, clé `sync_meta` à côté de `history_complete_at`). La passe suivante : (1) delta
  du dessus jusqu'au premier connu, qui compte N nouveaux matchs ; (2) reprise au marque-page + N moins
  une page de recouvrement ; (3) la page de recouvrement doit contenir l'identifiant rangé, sinon
  reprise depuis le début (lente, sûre), journalisée et comptée. Un match en échec ramène le marque-page
  avant lui. Fin vue = les deux clés : fin rangée, marque-page effacé. Halo 5 suit la même règle si sa
  liste se pagine par décalage ; sinon écart écrit dans le rapport, rebalayage gardé pour Halo 5 seul.
- **D-14** Avant de figer le budget (200) et la cadence (10 min) : mesure d'UNE passe de rattrapage de
  200 matchs sur une copie de base (durée, appels API, pic mémoire, disque des films), étape 5bis. Si
  la passe dépasse 5 minutes ou 1,5 Go de pic, le budget ou la cadence se règlent sur la mesure (le
  pilote propose, l'utilisateur tranche).
- **D-16** (pilote, 22/11) Un match qui échoue à 3 passes de rattrapage (`maxCatchUpAttempts`) est
  abandonné par le rattrapage : journalisé, compté (`sync_v2_history_abandoned_total`), il ne compte
  plus dans le budget ni ne bloque la fin. Sans cela, un seul match cassé laisserait le joueur en
  rattrapage pour toujours (cycles de 10 minutes à vie). Liste des échecs dans le marque-page, bornée
  à 200 entrées.
- **D-15** (utilisateur, 22/11) État de l'historique à l'écran, version simple, élégante, aux normes UI
  de l'app (étape 4ter) : dans l'onglet « Données & Médias », à côté du bouton, par titre suivi : état
  (complet / rattrapage en cours), matchs enregistrés, date du plus ancien, dernière passe (heure,
  matchs ajoutés, erreur éventuelle), prochaine passe. Texte seul, pas de graphique. Mise à jour
  automatique tant qu'un rattrapage tourne. Lecture seule côté serveur.
- **D-16** (pilote, 22/11) Un match qui échoue toujours ne garde pas le joueur en rattrapage à vie. Le marque-page
  (`history_catchup_cursor`) porte aussi la liste ordonnée des matchs en échec de rattrapage avec leur nombre de
  passes (`failed: [{id, attempts}]`, du plus ancien au plus récent, au plus 200 entrées, les plus anciennes sortent ;
  une liste ordonnée plutôt qu'un objet, pour savoir qui est le plus ancien). Un match en échec à `maxCatchUpAttempts`
  (3) passes est abandonné : il n'est plus retenté par le rattrapage, ne compte plus dans le budget, ne retient ni la
  fin ni le marque-page ; journalisé (`slog.Warn`, identifiant et dernière erreur) et compté
  (`sync_v2_history_abandoned_total`) au moment de l'abandon, dans la passe du dernier échec. La fin se range quand la
  fin est vue et que les seuls inconnus restants sont abandonnés ; le journal de fin dit combien. Un 404/410 reste
  « introuvable » : jamais compté comme échec. Un match abandonné qui sort de la liste (borne de 200) redevient un inconnu
  ordinaire.
- **D-11** Rien ne change dans la base : aucune migration, aucune table. Clé `sync_meta` existante.
- Réservé à l'utilisateur : squash dans `main`, tag `v8.2.2`, publication de la release et de la mise à
  jour de l'application, réponse publique sur #213 (texte validé par lui le 22/11).

## Pistes et modèles

Chemin critique : étape 1 (paquet partagé), puis pistes A, B, C en parallèle, puis intégration.

| Étape | Piste | Modèle | Worktree / branche | Effort |
|---|---|---|---|---|
| 1 | socle | Sonnet | `LevelUp-wt-822-socle` / `fix/822-socle` | rapide |
| 2 | A Infinite V2 | Sonnet | `LevelUp-wt-822-a` / `fix/822-a` | moyen |
| 3 | B Halo 5 | Sonnet | `LevelUp-wt-822-b` / `fix/822-b` | moyen |
| 4 | C cadence | Sonnet | `LevelUp-wt-822-c` / `fix/822-c` | rapide |
| 4bis + 4ter | E bouton et état de l'historique | Sonnet | `LevelUp-wt-822-e` / `fix/822-e` | moyen |
| 5bis | mesure d'une passe | Sonnet | copie de base dans le scratchpad | rapide |
| 5 | convergence bout en bout | Sonnet | `LevelUp-wt-822` (intégration) | moyen |
| 6 | aide, version, notes | Haiku | `LevelUp-wt-822-d` / `fix/822-d` | rapide |
| 7 | revue adversariale du diff cumulé | Fable | lecture seule | moyen |
| 8 | gates complets, journal, PR | pilote | `LevelUp-wt-822` | moyen |

Caches : chaque agent pose `GOCACHE=%LOCALAPPDATA%\go-build-822<piste>` et
`GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-822<piste>` ; clôture par
`powershell -File scripts/disk-hygiene.ps1 -Lot 822<piste> -Apply`.

## Étapes

### Étape 1 — paquet `internal/sync/historystate` (socle)

- [x] 1.1 Paquet `historystate` : clé, `Complete`, `MarkComplete` (sous bail joueur tenu par l'appelant,
  documenté dans le paquet), `PassBudget()` (= D-3), `MaxDepth` (= D-3). `MarkComplete` rend un booléen
  (clé rangée ou non). Lecture et écriture de `sync_meta` déplacées dans le paquet `internal/sync/syncmeta`
  (évite le cycle `sync` -> `historystate` -> `sync` ; `sync.GetSyncMeta` / `SetSyncMeta` supprimées, appelants migrés).
- [x] 1.2 V1 migré : `engine.go` et `writes.go` appellent `historystate` ; anciennes fonctions
  `historyComplete` / `markHistoryComplete` / `SyncMetaHistoryCompleteAt` supprimées avec leurs usages.
- [x] 1.3 Champ `HistoryPending bool` sur `domain.SyncResult` (V1 et H5) et sur l'issue par joueur du
  cycle V2 (`PlayerOutcome`), sans encore le remplir hors V1 ; V1 le remplit (incomplet en fin de passe).
- [x] 1.4 Garde-rail D-9 dans `internal/archlint/` (`history_key_single_source_test.go`, littéraux lus dans l'AST).
- [x] 1.5 Tests unitaires `historystate` sur DuckDB `:memory:` ; tests V1 existants du rattrapage verts
  sans modification de leur intention.
- Gate : `cd apps/go-api && go build ./... && go vet ./internal/sync/... ./internal/archlint/... &&
  go test ./internal/sync/... ./internal/archlint/... ./internal/domain/...`.

### Étape 2 — piste A : découverte V2 (Halo Infinite)

- [x] 2.0 Marque-page (D-13) ajouté à `historystate` (`LoadCursor`, `SaveCursor`, `ClearCursor`, type `Cursor`, clé `history_catchup_cursor`), tests. Commit `beb37a3eb8` (fix/822-a), à fusionner tôt : la piste Halo 5 en dépend.
- [x] 2.1 `KnownLoader.LoadKnown` rend `KnownState` (ensemble connu + `HistoryState{Incomplete, Cursor}`, lus sur la base joueur déjà ouverte).
- [x] 2.2 `ListUnknownMatches` : deux modes (D-4, D-13), rend `Listing{Unknown, EndSeen, Proposed}` (`v2/fetchers_history.go`).
- [x] 2.3 `cycle.go` + `v2/history.go` : après une persistance réussie, range la fin ou le marque-page pour chaque
  joueur qui remplit D-5 ; remplit `HistoryPending` (sorties anticipées comprises) ; introuvables reconnus par
  `sync.IsNotFound` (verdict typé de la brique halo-api, 404/410) ; « persisté » prouvé par relecture de l'ensemble connu.
- [x] 2.4 Câblage `cmd/server/sync_v2_wiring.go` : `WithHistoryWriter(playerDBOpenerRW)`, l'ouverture sous bail joueur du post-sync
  (`postSyncPlayerOpener`) ; ratchet `sync_v2_wiring_history_test.go`.
- [x] 2.6 D-16 : échecs de rattrapage comptés dans le marque-page, abandon à 3 passes (`v2/history_settle.go`, `historystate.LoadCatchUp` / `SaveFailures`), tests (abandon à la 3e passe puis fin rangée, échec puis succès, borne de 200, 404 jamais un échec).
- [x] 2.5 Tests : découverte (incomplet : saute les connus, budget, fin vue, page partielle, borne de
  sûreté ; complet : inchangé) ; cycle (fin rangée seulement sous D-5 : cas erreur de découverte, erreur
  de récupération, introuvable, persistance en échec).
- Gate : `go test ./internal/sync/v2/... ./cmd/server/...` puis
  `go test -tags=integration ./internal/sync/... ./internal/persist/...`.

### Étape 3 — piste B : Halo 5

- [ ] 3.1 `Runner.RunDelta` : historique incomplet = passe bornée par la mécanique de `backfill.go`
  (option de budget ajoutée à `Backfill`, sans dupliquer la boucle).
- [ ] 3.2 Fin rangée sous D-5 ; `HistoryPending` rempli.
- [ ] 3.3 Tests : passe bornée, reprise sur trois passes jusqu'à la fin, erreur de page = pas de fin
  rangée, historique complet = delta inchangé, `h5-backfill` inchangé.
- Gate : `go test ./internal/games/halo_5/...`.

### Étape 4 — piste C : cadence de l'ordonnanceur

- [ ] 4.1 `catchUpCycleInterval` = 10 min ; après chaque cycle (V2 et filet `syncPlayer`), le ticker
  est recalé sur 10 min si un joueur a `HistoryPending`, sinon sur l'intervalle réglé (D-7).
- [ ] 4.2 Ligne de journal par cycle ; auto-sync désactivée = aucun cycle, comme aujourd'hui.
- [ ] 4.3 Tests avec horloge ou ticker injecté : bascule 6 h -> 10 min -> 6 h.
- Gate : `go test ./internal/scheduler/...`.

### Étape 4bis — bouton « Synchroniser maintenant » (Réglages › Données)

- [ ] 4b.1 Route serveur qui lance le cycle de l'ordonnanceur (`RunOnceTrigger`, raison `manual`),
  refus propre si un cycle tourne déjà (409 + message i18n), auto-sync désactivée = le bouton lance
  quand même un cycle (décision : un clic explicite passe outre le réglage). Pas de logique dans le
  handler (port vers l'ordonnanceur).
- [ ] 4b.2 Bouton dans l'onglet `data` des réglages, visible pour tout joueur et tout profil de
  déploiement sauf démo ; textes FR/EN ; état « en cours » tant que le cycle tourne.
- [ ] 4b.3 Tests : handler `httptest`, composant (vitest).
- Gate : `go test ./internal/api/... ./internal/scheduler/...`, `make generate-types` sans diff
  résiduel, `npx vitest run src/features/settings --maxWorkers=2`, `npx tsc --noEmit -p .`.

### Étape 4ter — état de l'historique (D-15)

- [ ] 4t.1 Port + lecture `platform/duckdb` : nombre de matchs et date du plus ancien par joueur et
  titre (fuseau canonique, fragment partagé `sqlfrag`), état et marque-page par `historystate`.
- [ ] 4t.2 Dernière passe et prochaine passe lues de l'ordonnanceur (issue par joueur déjà tenue,
  heure du prochain cycle), derrière un port.
- [ ] 4t.3 Route de lecture (handler sans logique), réponse localisée sans paire `_fr`/`_en`,
  `make generate-types`.
- [ ] 4t.4 Encadré dans l'onglet « Données & Médias », avec le bouton de l'étape 4bis : skills
  `frontend-patterns`, `color-tokens`, `foundations-usage` ; query key dans `lib/query/keys.ts` ;
  rafraîchissement périodique seulement pendant un rattrapage ; état vide et état d'erreur ; FR/EN.
- [ ] 4t.5 Tests : repo sur DuckDB `:memory:`, handler `httptest`, composant vitest.
- Gate : celui de l'étape 4bis.

### Étape 5bis — mesure d'une passe (D-14)

- [ ] 5b.1 Sur une COPIE de la base locale (règle : neutraliser les chemins des réglages avant de lancer
  un serveur sur copie), clé `history_complete_at` retirée pour un joueur de plus de 1 000 matchs, une
  passe : durée, appels API par type, pic mémoire, octets écrits sous l'archive des films. Résultat
  écrit dans ce plan.

### Étape 5 — convergence de bout en bout (le test qui manquait)

- [ ] 5.1 Test qui construit l'orchestrateur V2 par la même fabrique que la production (ou le plus près
  possible, écart justifié par écrit) avec un faux client Halo de 1 000 matchs et une vraie DuckDB
  temporaire : cycles successifs, assertion à chaque cycle (200 de plus, nouveaux matchs du dessus
  pris en premier, fin rangée au 5e ou 6e cycle, cycle suivant = une page et zéro rattrapage).
- [ ] 5.2 Même scénario pour Halo 5 par `Runner.RunDelta`.
- [ ] 5.3 Même scénario par le moteur V1 (watcher) : le contrat est le même pour les trois chemins.
- Gate : `go test -tags=integration -run Convergence ./internal/...`.

### Étape 6 — aide, version, notes (Haiku)

- [x] 6.1 Aide FR/EN (`features/help/i18n/{fr,en}.ts`) : le texte qui annonce un bouton de
  synchronisation manuelle dans les réglages décrit le comportement réel (synchronisation automatique,
  historique rattrapé de 200 matchs par passe, toutes les 10 minutes tant qu'il n'est pas complet ;
  parties personnalisées non synchronisées). Toutes les langues de `config/locales.toml`.
  Constat : le bouton existe (`settings/SyncTab.tsx`, « Synchroniser tous les joueurs ») ; ce qui
  était faux, c'était « une synchronisation complète peut être forcée depuis les Paramètres ». Réécrit
  dans les deux langues (entrée « Sync » / « Synchronisation », puce manuelle en FR).
- [x] 6.2 Version 8.2.2 : `apps/desktop/src-tauri/{Cargo.toml,Cargo.lock,tauri.conf.json}` (trois
  lignes faites). `apps/web/package.json` et `package-lock.json` NON modifiés : leur version est
  `0.0.0`, la 8.2.1 ne les a jamais touchés (`git show fee334a039` ne liste que les trois fichiers desktop).
- [x] 6.3 `docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md`, `docs/RELEASE_NOTES.md`,
  `docs/FR/RELEASE_NOTES.md` : section 8.2.2 (sans date, « Unreleased » / « Non publié » comme
  le fichier le fait ailleurs). Issue #213 citée dans les deux CHANGELOG (les sections précédentes en citent).
- Gate : `cd apps/web && npx vitest run src/features/help --maxWorkers=2 && npx tsc --noEmit -p .`.

### Étape 7 — revue adversariale (Fable)

- [ ] 7.1 Skills `adversarial-review` + `levelup-revue` sur `git diff origin/main...release/v8.2.2`,
  une ronde ; P0/P1 corrigés, P2 au backlog.

### Étape 8 — gates complets, journal, PR

- [ ] 8.1 `go test ./...`, `go test -tags=integration ./internal/sync/... ./internal/persist/...
  ./internal/games/...`, `make go-api-lint`, web : `npx vitest run` complet une fois, `make check-types`.
- [ ] 8.2 ADR 0027 : amendement court (EN) « history catch-up applies to every sync path ».
- [ ] 8.3 Entrée `.ai/thought_log.md`.
- [ ] 8.4 Poussée de `release/v8.2.2`, CI verte (`gh run list --branch release/v8.2.2 --workflow CI
  --limit 1`), PR vers `main` (squash par l'utilisateur).
- [ ] 8.5 Texte de réponse à #213 prêt (validé par l'utilisateur le 22/11), publié après la PR.

## Statuts, ordre, reprise

`[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification). Aucune case vide à la
clôture. Une étape est close quand son gate passe et que son commit est sur sa branche. Reprise : lire ce
fichier, `git log --oneline -10` dans chaque worktree, `ListAgents`.

## Découvertes

(Les correctifs hors périmètre sont interdits : les noter ici.)

- Étape 1 : `sync/match_recomputer.go` lit `sync_meta` par sa propre requête (clé `scoring_revision`),
  copie de `syncmeta.Get` ; à aligner plus tard, hors périmètre.
- Étape 1 : en mode file asynchrone, si la réouverture de la base joueur échoue après le drain, V1 ne
  peut pas ranger la fin (base `nil`) : `HistoryPending` reste vrai, la passe suivante rattrape.
- Étape 1 : une passe V1 qui sort en erreur avant la fin (chargement des connus, bail) rend
  `HistoryPending` faux (valeur par défaut) ; l'ordonnanceur (étape 4) lit aussi l'erreur du cycle.
- Étape 2 : `GetMatchStats` signale un 404/410 par un verdict TYPÉ (`infinite.IsNotFound`, `errors.Is(err, ErrFilmGone)`
  sur `*HTTPError`) ; ajouté `haloadapter.IsNotFound` et son ré-export `sync.IsNotFound` (même motif qu'`IsAuthError`).
  Les erreurs de récupération de V2 conservent la chaîne `%w`, donc le verdict traverse `RunFetchShared`.
- Étape 2 : le risque d'un match qui échoue toujours (parse, persistance, 500 permanent) est traité par D-16 :
  abandon à la 3e passe en échec, fin rangée ensuite. Les échecs se comptent par passe où le match était listé ;
  un orphelin de l'ensemble connu abandonné reste redemandé par la rotation des orphelins (`Set.Recover`) sans
  jamais bloquer.
- Étape 2 : l'entrée « introuvable » du dessus de liste (404 jamais persisté) est recomptée comme nouveau match à chaque
  passe ; elle décale la reprise du marque-page de ce nombre ; au-delà d'une page (25) la page de recouvrement ne
  contient plus le dernier match rangé et la lecture repart du début (journalisé, compté).
- Étape 2 : la page de recouvrement est lue deux fois (vérification puis lecture ordinaire) : un appel d'API de plus par
  passe, laissé tel quel.
- Étape 2 : joueur dont `LoadKnown` échoue (état de l'historique inconnu) : `HistoryPending` reste faux ; l'ordonnanceur
  (étape 4) lit l'erreur du cycle. Sorties anticipées du cycle : seules les issues par joueur déjà créées portent
  `HistoryPending`.
- Étape 2 : `ClearCursor` range une valeur vide (aucun DELETE sur `sync_meta`, règle anti-ART) ; le type s'appelle
  `Cursor` et la lecture `LoadCursor` (un type et une fonction ne peuvent pas porter le même nom dans un paquet).
