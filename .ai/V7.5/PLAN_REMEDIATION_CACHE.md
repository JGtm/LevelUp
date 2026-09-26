# Plan — Remediation du cache d'artefacts appauvris

> Cree le 2026-08-25. Branche cible : `wt/remediation-cache` (base `origin/feat/v75` `3cafdfbe8`).
> Contrat d'execution : skill `plan-execution` (defaut). Ce plan ne le remplace pas, il s'y soumet.
> Ce plan OUVRE et TRAITE la dette « le cache DEJA empoisonne » consignee au
> `PLAN_OUVRIER_DISTANT.md` §5ter (lignes 359-366), avant la premiere activation prod de l'ouvrier.

## 1. Objectif et critere de succes

**Probleme.** Le garde de fraicheur post-sync (`internal/sync/replayartifacts/artifacts.go`,
`etatArtefact` + `replaybuild.ArtifactHasPlayerCounters`) ne re-enfile / ne reconstruit un artefact
APPAUVRI (cuit sans les faits de match -> `scoreTimeline.players` vide, objectifs/zones vides,
identite des camps `unresolved`) que pour les matchs INSERES dans le cycle de sync courant. Un
artefact deja appauvri present dans le cache (`data/cache/replays/{slug}/{short8}.json`) n'est donc
reparable par AUCUN chemin : il n'est jamais re-vu. Des la premiere passe d'ouvrier sans faits, le
cache s'empoisonne A DEMEURE.

**Critere de succes.** Une passe operateur explicite qui, pour chaque artefact EXISTANT au schema
COURANT dont `scoreTimeline.players` est vide ALORS QUE la base porte des lignes de match, re-cuit
l'artefact via l'outil blinde existant (un film par processus, plafond memoire), et qui SAUTE :
- l'artefact deja riche (compteurs de joueur presents) ;
- la vacuite legitime (a jour, sans compteurs, mais base SANS joueurs — re-cuire ne donnerait rien) ;
- l'artefact hors schema courant (domaine de la passe `--only-existing` ordinaire d'apres bump).

La commande de remediation est documentee et les gates passent (dont anti-ART). Aucune activation
d'ouvrier n'est faite ici.

## 2. Conception retenue (R0) et ses defauts

**Le mode est une PLANIFICATION PURE du parent — il ne cree AUCUN second chemin de cuisson.**
L'outil `levelup backfill-replay` separe deja PARENT (enumere, trie, saute, `--dry-run`) et ENFANT
(un film, decode, puis meurt sous plafond memoire). Le mode `--repair-impoverished` ne touche QUE la
SELECTION du parent : quels artefacts existants meritent une re-cuisson. L'ENFANT
(`cmd_backfill_replay_child.go`), `Builder.BuildMatch` et le garde anti-regression
`writeArtifactBytes` sont INTACTS — l'enfant lit deja ses faits (`chargerFaitsUnMatch`) et construit
avec, et le garde au point d'ecriture unique interdit deja toute retrogradation. Le mode ne peut donc
QU'AMELIORER un artefact, jamais le degrader.

Pourquoi c'est sur (sur pieces) :
- `replaybuild/replaybuild.go:317` `ArtifactUpToDate` = schema courant seul.
- `replaybuild/replaybuild.go:362` `ArtifactHasPlayerCounters` = `scoreTimeline.players` non vide ;
  son en-tete documente les TROIS vacuites legitimes (film sans entites, appariement ambigu, aucun
  compteur dans la fenetre) et pourquoi le vide est une PRESOMPTION, pas une preuve.
- `replaybuild/artifact_store.go:158` `writeArtifactBytes` : le garde `wouldDowngrade` refuse une
  ecriture qui retrograderait (compteurs sur disque, absents dans le candidat, MEME schema). A schema
  different il se tait (montee de schema toujours ecrite). Quand le disque a 0 joueur, il ne bloque
  jamais : une re-cuisson d'appauvri passe toujours.
- `replayartifacts/artifacts.go:230` `etatArtefact` : la regle canonique post-sync — `complet` =
  `len(facts.Players) == 0 || ArtifactHasPlayerCounters(path)`. Le mode reprend EXACTEMENT ce
  predicat, cote CLI.

**Ce qui debloque la dette.** Le registre disait la remediation de masse « dependante de la bombe RAM
`NamedEventsFrom` ». Le lot blindage (un film = un processus + plafond souple + sentinelle dure +25 %,
defaut 3 GiB) rend justement cette passe executable : le film-bombe `51101d1d` meurt SEUL en
`mortsMemoire` (mesure 2026-08-24 : 7,9 Go en ~2,6 s, isole en ~1 s), la passe continue. Aucune
exclusion du film-bombe n'est codee en dur — le plafond de l'enfant la gere.

## 3. Decisions tranchees (defauts fermes — ne pas re-decider en cours d'execution)

- **D1 — Planification pure.** Le mode ne modifie que la selection parent. Aucun nouveau decodeur,
  aucune boucle de cuisson en un seul process. (Alternative rejetee : boucle dediee = second chemin =
  viole `<= 2 copies` + risque RAM du lot blindage.)
- **D2 — Ou lire les faits : le MEME port que l'ouvrier.** `duckdb.NewReplayFactsRepo(shared)` +
  `FactsForMatch`, en lecture COURTE shared RO relachee AVANT tout decodage. Le comptage
  `len(facts.Players)` est LE MEME que celui que l'enfant obtiendra — zero divergence. (Alternative
  rejetee : un `SELECT count(*)` ad hoc divergerait du filtre `xuid == ""` de `playerFacts`.)
- **D3 — Eviter les vacuites legitimes : comparer au nombre de joueurs en base.** On re-cuit SEULEMENT
  si `ArtifactUpToDate` ET `!ArtifactHasPlayerCounters` ET `len(facts.Players) > 0`. Residu assume et
  documente : un film dont la base a des joueurs mais qui ne peut PAS les produire (vacuite legitime
  a/b/c malgre des lignes) sera RE-TENTE a chaque passe MANUELLE — borne (passe operateur, pas une
  boucle auto), jamais une retrogradation (garde d'ecriture). On NE POSE PAS de marqueur
  `factsApplied` : il forcerait un bump de `replay.SchemaVersion`, explicitement rejete au registre
  (2026-08-24) car il rendrait tout le cache perime, re-cuisson bloquee par la bombe RAM.
- **D4 — Perimetre schema : schema COURANT seul.** Un artefact hors schema courant est compte
  `horsSchemaCourant`, NON re-cuit par ce mode (domaine de `--only-existing` ordinaire d'apres bump).
  Garde le budget de decodage minimal et separe les preoccupations.
- **D5 — `--force` et `--repair-impoverished` s'EXCLUENT** (erreur claire au parse). Le mode EST une
  selection ciblee ; `--force` re-cuirait tout, ce que le mode existe pour eviter.
- **D6 — Blindage RAM herite, pas reinvente.** Un film par enfant, plafond `--mem-limit-gib` (defaut
  3). Le film-bombe meurt seul, la passe continue. Aucune exclusion en dur.
- **D7 — DB indisponible en mode repare = ERREUR DURE** (contrairement a metadata, best-effort). La
  base shared est ESSENTIELLE a la decision (distinguer appauvri-reparable de vacuite legitime).
  Echec explicite (« serveur en ecriture ? reessayer »), coherent avec `registreParShort`.
- **D8 — Validation temoin reel = DRY-RUN read-only uniquement (par defaut).** Compte tenu des 2
  crashs RAM et de l'agent parallele (un seul decodage vivant a la fois), la validation se fait par
  TESTS UNITAIRES (decision pure) + un `--repair-impoverished --dry-run` read-only pointe sur le cache
  reel (aucun decodage, aucune ecriture). Aucun decodage reel lance par l'executant ; jamais le
  film-bombe.

## 4. Fichiers touches (verifies sur pieces le 2026-08-25)

- `apps/go-api/cmd/levelup/cmd_backfill_replay.go` — flag, champ d'option, exclusion `--force`, branche
  de selection, ligne de resume repare. (~330 L -> < 500.)
- `apps/go-api/cmd/levelup/cmd_backfill_replay_repair.go` — NOUVEAU. `etatReparation`,
  `classerReparation` (pur), `selectionnerReparables` (pur, resolveur injecte), `passeReparation`
  (wrapper DB shared RO), `trierEtBornerReplay` (extrait, partage).
- `apps/go-api/cmd/levelup/cmd_backfill_replay_passe.go` — 3 compteurs de rapport + affichage
  conditionnel ; `filtrerEtTrierReplay` reutilise `trierEtBornerReplay`.
- `apps/go-api/cmd/levelup/cmd_backfill_replay_repair_test.go` — NOUVEAU. Tests unitaires.
- `.ai/V7.5/PLAN_OUVRIER_DISTANT.md` — statuer la dette §5ter comme TRAITEE + commande de remediation.
- `.ai/V7.5/README.md` — index : entree du nouveau plan.
- `.ai/thought_log.md` — entree obligatoire.

Aucune couche analysis/service/handler/frontend/adapter/TOML touchee (outil operateur CLI, lecture
seule DB, aucun type canonique nouveau, title-agnostic deja assure par `PathResolver` + `titleSlug`).

## 5. Etapes

### Etape 1 — Le flag et la selection parent (perimetre FERME) — CLOSE 2026-08-25

- [x] `replayBackfillOptions` : `repairImpoverished bool` ajoute.
- [x] `parserOptionsReplay` : `--repair-impoverished` enregistre (aide conforme).
- [x] `runBackfillReplay` : `o.repairImpoverished && o.force` -> erreur claire (D5).
- [x] `cmd_backfill_replay_repair.go` (NOUVEAU, 160 L) : `etatReparation` (5 etats), `classerReparation`
      (pur, resolveur paresseux), `selectionnerReparables` (pur, resolveur injecte), `passeReparation`
      (wrapper shared RO, D7 erreur dure, release AVANT decodage), `trierEtBornerReplay` (extrait).
- [x] `filtrerEtTrierReplay` : tri+limit inline remplaces par `trierEtBornerReplay` (import `sort`
      retire du fichier principal).
- [x] `runBackfillReplay` : branche `passeReparation` vs `filtrerEtTrierReplay` + ligne de resume dediee.
- [x] `cmd_backfill_replay_passe.go` : 3 compteurs (`dejaComplets`, `vacuitesLegitimes`,
      `horsSchemaCourant`) + affichage conditionnel (muet en passe ordinaire).
- [x] En-tete `cmd_backfill_replay.go` : usage + section `--repair-impoverished` (distinction avec
      `--only-existing`).

**Gate etape 1** — PASSE : `go build ./...` exit 0 ; `go vet ./cmd/levelup/...` exit 0. Seuils :
fichiers 348/160/197 L (<= 500), `runBackfillReplay` 47 L, fonctions repair toutes < 30 L (<= 80).

### Etape 2 — Tests unitaires (perimetre FERME) — CLOSE 2026-08-25

- [x] `cmd_backfill_replay_repair_test.go` : helper `ecrireArtefactAvecJoueurs(t, repo, slug, id,
      schema, nbJoueurs)` (JSON `schemaVersion` + `matchId` + `tracks:[1]` + `scoreTimeline.players:[n]`).
- [x] `TestClasserReparation` (table) — les 5 etats :
  - a jour + 0 joueur sur disque + base a des joueurs (resolveur -> 8) => `reparationACuire` ;
  - a jour + 0 joueur sur disque + base SANS joueur (resolveur -> 0) => `reparationVacuiteLegitime` ;
  - a jour + compteurs presents => `reparationDejaComplet` (resolveur NON appele) ;
  - schema anterieur => `reparationHorsSchema` ;
  - artefact absent => `reparationSansArtefact`.
      La table verifie AUSSI la lecture PARESSEUSE (colonne `appelDB`) : le resolveur n'est
      interroge que pour un artefact a jour ET appauvri.
- [x] `TestSelectionnerReparables` (resolveur FAKE, sans DB) : un appauvri-reparable RETENU, un
      legitimement vide SAUTE, un riche INTACT (saute) ; compteurs ventiles corrects ; ordre + `--limit`.
      Livre en DEUX tests : `TestSelectionnerReparables` (ventilation 1/1/1 + resolveur jamais
      interroge pour le riche et le hors-schema) et `TestSelectionnerReparables_OrdreEtLimite`
      (tri par cout croissant, id departage, `--limit 2`). Bonus au meme fichier :
      `TestRunBackfillReplay_RepairEtForceSExcluent` (D5 refuse AVANT tout acces cache/DB).
- [x] Verifier que les tests EXISTANTS de `cmd_backfill_replay_passe_test.go` passent inchanges
      (refactor `trierEtBornerReplay` sans regression) — les 6 tests d'origine PASS, fichier non modifie
      (`git status` : seul `cmd_backfill_replay_passe.go` porte les 3 compteurs).

**Gate etape 2** :
```
cd apps/go-api && go test ./cmd/levelup/... ./internal/replaybuild/... ./internal/sync/replayartifacts/...
```
Exit 0, aucun test skippe pour passer.

**Gate etape 2 — PASSE 2026-08-25** : exit 0 (`ok cmd/levelup 0,203s`, `ok internal/replaybuild 0,708s`,
`ok internal/sync/replayartifacts 0,154s`). Contre-verification `-count=1 -v` sur les tests repair +
les tests existants de la passe : 20 PASS, **0 SKIP**, exit 0.

### Etape 3 — Gates complets, validation dry-run read-only, registre (perimetre FERME)

- [x] `go build ./...` (exit 0) — mesure 2026-08-25 : **exit 0**.
- [x] `go vet ./...` (exit 0) — mesure 2026-08-25 : **exit 0**.
- [x] Tests paquets touches (rappel etape 2) verts — **exit 0** (gate etape 2, + contre-verification
      `-count=1 -v` : 20 PASS, 0 SKIP).
- [x] **Anti-ART OBLIGATOIRE** — **INTEG_EXIT=0**. Le run `./...` a fini en `127` (command-not-found
      environnemental, shell tue) en s'arretant a `internal/api/openapigen`, AVANT persist/sync/ops :
      resultat NON concluant. REJOUE sur le perimetre anti-ART critique `-tags=integration -p 1
      ./internal/replaybuild/... ./internal/sync/... ./internal/persist/... ./internal/ops/...` →
      **exit 0**, tous `ok`, temps REELS (pas de faux vert de cache) : `internal/sync` 138 s,
      `internal/persist` 28 s, `internal/ops` 96 s, `internal/replaybuild` ok. 0 FAIL, 0 panic.
- [x] `make go-api-lint` — **exit 0, 0 issues** (avertissement gosec/plr0913 pre-existant, hors mon code).
- [x] Validation temoin (D8) : `--repair-impoverished --dry-run` pointe read-only sur le cache/DBs
      reels du depot principal (`LEVELUP_REPO_ROOT` + `--cache` sur
      `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/...`). Aucun decodage, aucune ecriture.
      **FAIT le 2026-08-25, exit 0** (detail au journal §8) : **2 appauvris-reparables** sur 951 films
      du cache — 32 deja complets, 0 vacuite legitime, 1 hors schema courant, 0 hors registre, 916 sans
      artefact. La DB shared etait LISIBLE malgre le serveur actif (`OpenReadForQuery`) : l'erreur dure
      D7 ne s'est PAS declenchee, la case est donc `[x]` et non `[!]`. Seule `metadata` etait tenue RW
      par `server.exe` (PID 18092) — degradation best-effort PREVUE (noms EN bruts), sans effet sur la
      selection.
- [x] `PLAN_OUVRIER_DISTANT.md` §5ter : statuer la dette « le cache DEJA empoisonne » comme TRAITEE,
      avec la commande de remediation exacte a lancer AVANT activation ouvrier.
- [x] `.ai/V7.5/README.md` : indexer ce plan.
- [x] `.ai/thought_log.md` : entree ajoutee (date, titre, statut Complete, decision, resultats, suite).

**Gate etape 3** — PASSE : build/vet/tests/anti-ART/lint tous exit 0, dette §5ter statuee TRAITEE
avec commande de remediation, README indexe, thought_log a jour. R3 committe `cache(R3)`.

## 6. Decouvertes (consignees, PAS corrigees hors gate — regle 7 plan-execution)

- 2026-08-25 — `cmd_backfill_replay_repair_test.go` n'etait PAS gofmt (8 octets d'alignement de
  commentaires en trop, aucun CRLF). CORRIGE dans le perimetre : `gofmt` est un gate de livraison du
  lot, pas un fix opportuniste hors perimetre — et le fichier est de ce lot. Gate etape 2 rejoue
  `-count=1` apres reformatage : exit 0. Lecon : un fichier ecrit en fin de session interrompue peut
  n'avoir jamais vu `gofmt` — le verifier a la reprise, `go build`/`go vet` ne l'attrapent pas.
- 2026-08-25 — La suite anti-ART lancee en arriere-plan a ete TUEE par une limite de session sans
  rendre de verdict (elle etait a 174 lignes, 0 FAIL, sur `internal/analysis`). Un resultat partiel
  n'est PAS un exit code : la suite a ete RELANCEE entierement (`-count=1`, `-p 1`) pour obtenir un
  verdict reel. Ne jamais conclure d'une sortie partielle.

## 7. Reprise de session

Source de verite de l'avancement = ce fichier (cases + journal ci-dessous). A la reprise : relire le
contrat `plan-execution`, puis ce plan, reprendre a la premiere case non statuee de l'etape courante.
Decisions §3 = fermes, ne pas re-decider.

## 8. Journal d'execution

- 2026-08-25 — Plan cree (R0 conception terminee sur pieces, R1 redige). Etapes 1-3 non commencees.
- 2026-08-25 — **Etape 1 CLOSE** : code ecrit, `go build ./...` exit 0, `go vet ./cmd/levelup/...` exit 0.
  Seuils : 348/160/197 L (<= 500), `runBackfillReplay` 47 L, fonctions repair < 30 L (<= 80).
- 2026-08-25 — **Etape 2 CLOSE** (reprise apres interruption de session : le fichier de tests etait
  ecrit mais NI gate NI statue ; verifie sur pieces avant de cocher, aucune ligne re-ecrite de memoire).
  Gate : `go test ./cmd/levelup/... ./internal/replaybuild/... ./internal/sync/replayartifacts/...`
  **exit 0** (`ok cmd/levelup 0,203s`, `ok internal/replaybuild 0,708s`,
  `ok internal/sync/replayartifacts 0,154s`). Contre-verification `-count=1 -v` (commande NUE, jamais
  a travers un pipe) : **20 PASS, 0 SKIP, exit 0** — dont les 6 tests d'origine de
  `cmd_backfill_replay_passe_test.go`, fichier NON MODIFIE (le refactor `trierEtBornerReplay` ne
  regresse pas). Ecart au plan, assume : l'item « ordre + `--limit` » est livre dans un test SEPARE
  (`TestSelectionnerReparables_OrdreEtLimite`) plutot que fondu dans `TestSelectionnerReparables` —
  meme couverture, un echec designe sa cause. Bonus hors liste mais dans le perimetre D5 :
  `TestRunBackfillReplay_RepairEtForceSExcluent`.
- 2026-08-25 — **Etape 3, temoin reel (D8)** : dry-run READ-ONLY du worktree pointe sur le cache et les
  DBs du depot principal (binaire compile depuis `wt/remediation-cache` dans le scratchpad,
  `LEVELUP_REPO_ROOT` + `--cache` sur `LevelUp-go-migration`), **exit 0** :
  `artefacts appauvris a reparer : 2 (32 deja complets, 0 vacuites legitimes, 1 hors schema courant,
  0 hors registre, 916 sans artefact)` — 951 films au total, le compte du cache. Les 2 retenus :
  `24dbb67d-a7d8-4359-b319-bdf13959178d` (29 chunks, Recharge - Ranked) et
  `64e8adfa-e3e3-4c26-9f10-7d6a4dc36706` (45 chunks, Catalyst). **74 chunks a decoder, 0 film au-dela
  de 50 chunks : le film-bombe `51101d1d` n'est PAS dans la selection** — la remediation tient en
  quelques minutes. AUCUN decodage, AUCUNE ecriture (le dry-run rend la main avant
  `executerPasseReplay` ; la CLI `levelup` n'installe aucun journal fichier).
  D7 NON DECLENCHE : `OpenReadForQuery` a lu la shared malgre `server.exe` (PID 18092) actif. Seule
  `metadata.duckdb` etait verrouillee — degradation best-effort PREVUE (noms de carte bruts), sans
  effet sur la selection, qui ne depend que de la shared.
- 2026-08-25 — **Etape 3 CLOSE** : anti-ART `INTEG_EXIT=0` (perimetre critique replaybuild/sync/persist/ops
  rejoue apres un `./...` mort en 127 avant persist/sync — temps REELS sync 138 s / persist 28 s /
  ops 96 s, pas de cache), `make go-api-lint` exit 0 (0 issues). Dette §5ter de `PLAN_OUVRIER_DISTANT.md`
  statuee TRAITEE, `README.md` indexe, thought_log a jour. Commits : `cache(R1)` 1d18645a3 (mode + plan),
  `cache(R2)` 257512a3e (tests), `cache(R3)` (docs + cloture plan + thought_log). AUCUN push. Reprise
  apres coupure de session geree : R1/R2 retrouves deja committes byte-identiques, HEAD verifie fige a
  chaque etape avant d'agir.
