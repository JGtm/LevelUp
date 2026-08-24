# Plan — Ouvrier distant : les FAITS, et ce qui reste avant l'activation

> Ecrit le 2026-08-24. Branche `wt/ouvrier-distant`, base `b16ba17e5`. Execution sous le
> contrat du skill `plan-execution` (ordre strict, items statues, gates rejoues). PAS de push.
>
> **Ce plan n'ouvre pas le chantier « ouvrier distant » : il le TERMINE.** L'inventaire sur
> pieces du 2026-08-24 etablit que le transport, la file durable, le heartbeat, la reprise de
> bail et la verification d'integrite sont LIVRES ET PROUVES (`PLAN_TRANSPORT_ARTEFACT.md`,
> tous items `[x]`, gate final avec un vrai binaire le 2026-08-14). Ce qui reste tient en une
> phrase : **l'ouvrier construit un artefact APPAUVRI parce que le payload du job ne porte pas
> les faits du match, et `ArtifactUpToDate` fige cet artefact appauvri comme « a jour ».**

## Critere de succes

Un ouvrier sans base produit, pour un temoin donne, un artefact STRUCTURELLEMENT IDENTIQUE a
celui que produit la cuisson locale avec faits. Tant que ce n'est pas vrai, activer l'ouvrier
DEGRADE le rejeu en production de facon PERMANENTE (cf. le piege de fraicheur, D3).

---

## 1. Ce que l'inventaire a etabli (phase O0) — carte de l'existant

| Brique | Etat | Preuve (fichier:ligne) |
|---|---|---|
| Binaire ouvrier (claim / decode / push / complete) | **LIVRE** | `cmd/replay-worker/{main,job,protocol}.go` |
| File DURABLE (pas `jobs.json`) | **LIVRE** | `internal/migration/build_queue_schema.go:31-110` — DuckDB `data/global/monitoring.duckdb`, tables d'evenements `build_job_events` / `build_worker_events` + vues `build_jobs_latest` / `build_workers_latest` (append-only, conforme ADR 0026) |
| Heartbeat + bail + reprise + max tentatives | **LIVRE** | `internal/ops/build_queue.go:156-169` (pose du bail), `:291-315` (prolongation paresseuse), `:338-381` (reclaim), `domain/build_queue.go:104/109/113` |
| Balayeur de bails expires | **LIVRE** | `wire/registry_build_queue.go:277-289`, appele a chaque claim (`ops/build_queue.go:138-140`) et toutes les 60 s (`wire/registry_monitoring_store.go:108`) |
| Transport de l'artefact + integrite | **LIVRE** | `handlers/build_worker_artifact.go`, validation `replaybuild/artifact_store.go:66-99` (schema, matchId, presence de trajectoires), ecriture atomique `:101-109` |
| Securite du jeton | **LIVRE** | `handlers/build_worker.go:254-271` — `subtle.ConstantTimeCompare`, middleware pose AVANT Huma (`:88-89`) |
| Point de decision de placement | **LIVRE** | `internal/replaybuild/placement.go:63-88` — `local` interdit en prod, `worker` sans jeton degrade en `off` |
| Mise en file au fil de l'eau | **LIVRE** | `sync/replayartifacts/artifacts.go:241-244` -> `wire/registry_build_queue.go:45-85` |
| **Les FAITS dans le payload du job** | **MANQUANT** | `cmd/replay-worker/job.go:137` — `port.MatchFacts{}` litteral vide |
| **`ArtifactUpToDate` conscient des faits** | **MANQUANT** | `internal/replaybuild/replaybuild.go:307` — compare le SEUL `schemaVersion` |

`jobs.json` (`data/cache/jobs.json`) n'est PAS la file : c'est le `JobStore` des jobs
d'administration asynchrones. Le raisonnement est ecrit en tete du DDL
(`internal/migration/build_queue_schema.go:4-9`). **Aucune file a construire : elle existe.**

---

## 2. La mesure AVEC / SANS faits — chiffree, champ par champ

Protocole : meme film, meme carte, meme binaire (`cmd/replay-build`, hors ligne, sans CGO),
un seul parametre change (`--facts`). Temoin **`7344d24f`** (Strongholds:Arena, carte
Vagabond, module `fo08_wetland`, 122 trajectoires, schema 18). Faits reels versionnes :
`.ai/V7.5/replay2d/registre_film/lotCbis/faits_7344d24f.json`.

| Champ de l'artefact | AVEC faits | SANS faits | Perte |
|---|---|---|---|
| `objectives` | **246** | **0** | TOTALE |
| `coverage.objectives.available` / `.attached` | 246 / 246 | 0 / 0 | TOTALE |
| `zoneStates` | **3** | **0** | TOTALE |
| `coverage.zones` | `{"method":"captures+geometry",...}` | `null` | TOTALE |
| `scoreTimeline.players` | **8** | **0** | TOTALE |
| `coverage.score.teamIdentity` | `b` | `unresolved` | camps perdus |
| `coverage.score.points` | **1706** | **612** | -64 % |
| `tracks` / `roster` / `shots` / `grenades` | 122 / 8 / 2836 / 186 | 122 / 8 / 2836 / 186 | intactes |
| Taille | 2 238 996 o | 2 161 766 o | -77 230 o (-3,4 %) |

**Lecture.** Le film seul porte les trajectoires, les tirs, les grenades et le roster : ces
calques ne bougent pas. Ce que les faits apportent, et qu'ils sont SEULS a apporter :

1. **Les actions d'objectif** (`identifiedEvents`, `matchfacts.go:118-131`) — sans
   `facts.Players` il n'y a aucun slot a apparier, et sans `facts.GameVariantName` aucune
   famille d'objectif : refus explicite, 246 -> 0.
2. **Les zones du mode** (`replaybuild.go:179`, `matchZones(matchID, facts.MapID, facts.GameVariantName)`)
   — le catalogue de zones est indexe par l'asset UGC de la carte. Sans `MapID`, **tout le
   travail KOTH/Strongholds de la v7.5 disparait de l'artefact**.
3. **Les socles de drapeau** (`replaybuild.go:175`, `flagSpawns(matchID, facts.MapID)`) —
   meme cle, meme perte.
4. **L'identite des camps et les compteurs** (`ScoreInput.Lines` / `TeamByXUID` /
   `TeamScores`) — c'est exactement le bandeau 0-0 diagnostique le 2026-08-24
   (`.ai/thought_log.md:21-25`).

**L'avertissement du code SOUS-ESTIME la perte.** `cmd/replay-worker/job.go:131-136` annonce
« sans compteurs de joueur ni actions d'objectif » ; la mesure ajoute **les zones et les
socles de drapeau**, que ce commentaire ne mentionne pas. A corriger dans le meme lot.

**Le calque du DRAPEAU VIVANT, lui, SURVIT** : `flagInput` ne prend aucun fait, par
construction (`matchfacts.go:102-110`, « c'est ce qui rend le calque publiable hors ligne »).
Seuls ses SOCLES (issus du catalogue de carte via `MapID`) tombent. La dette est donc bornee :
elle porte sur ce que la BASE sait, jamais sur ce que le film sait.

**Second temoin — `530820e5` (CTF:Arena, carte Catalyst), qui CONFIRME cette borne :**

| Champ | AVEC faits | SANS faits |
|---|---|---|
| `objectives` | 183 | **0** |
| `scoreTimeline.players` | 8 | **0** |
| `coverage.score.teamIdentity` | `b` | **`unresolved`** |
| `coverage.score.points` | 782 | **6** |
| `flagCarries` | 1 | 1 (intact) |
| `coverage.flagCarries.carries` / `.captures` / `.objectLives` | 30 / 3 / 41 | **30 / 3 / 41 (intacts)** |
| Taille | 1 616 733 o | 1 587 866 o |

La vie des drapeaux traverse la frontiere SANS PERTE — 30 portages, 3 captures et 41 vies
d'objet identiques des deux cotes. C'est la mesure qui BORNE la dette : seuls les quatre
calques nommes ci-dessus dependent des faits.

---

## 3. Decisions TRANCHEES (defauts proposes)

**D1 — Transport des faits : EMBARQUES DANS LE JOB.** Mesure : les faits d'un match pesent
**713 a 756 octets** minifies (3 temoins) ; le payload porte deja ~20 Ko d'URL pre-signees.
Le cout est de ~3 %. Les deux alternatives sont ecartees sur piece : *re-deriver* est
impossible (l'ouvrier n'a AUCUNE base, c'est sa propriete de securite) ; *omettre puis
rattraper au retour* imposerait au VPS web de re-cuire l'artefact — exactement ce que la
regle « le VPS web ne decode JAMAIS » interdit.

**D2 — `MatchFacts` descend dans `domain`, `port` garde un ALIAS.** Contrainte de couche
verifiee : `port` importe `domain` (`port/services.go:14`), `domain` n'importe JAMAIS `port`.
Mettre `port.MatchFacts` dans `domain.BuildQueuePayload` creerait un cycle. `MatchFacts` est
un type de DOMAINE (ce que la base sait d'un match), pas un contrat de port. Un alias Go
(`type MatchFacts = domain.MatchFacts`) laisse les ~30 sites d'appel existants INCHANGES.

**D3 — `ArtifactUpToDate` : la fraicheur devient consciente des faits, SANS bump de schema.**
C'est le piege le plus couteux du chantier : un artefact cuit sans faits porte le bon
`schemaVersion`, donc `ArtifactUpToDate` le declare « a jour » et **plus rien ne le re-cuira
jamais**. Un ouvrier active sans faits EMPOISONNE le cache de rejeu de facon permanente.
Signal retenu : `scoreTimeline.players` vide alors que la base connait des participants.
Il est direct (mesure : 8 AVEC / 0 SANS), il ne coute aucun champ nouveau, et il est JUSTE
dans le cas limite (un match sans participants en base a legitimement un artefact sans
joueurs). **Alternative ecartee** : un marqueur `factsApplied` dans le document forcerait un
bump 18 -> 19, donc la re-cuisson des 34 artefacts — bloquee aujourd'hui par la bombe RAM
`NamedEventsFrom`/`incrementTimes` (OOM ~26 Go) consignee au registre le 2026-08-24.
Le predicat pur vit dans `replaybuild` ; la COMBINAISON avec la base vit chez l'appelant.

**D4 — File durable, heartbeat, reprise, transport, integrite, jeton : RIEN A FAIRE.**
Livres et prouves (tableau §1). Toute reecriture serait du travail detruit.

**D5 — Levee du garde `LocalOnlyReplay` : HORS DE PORTEE DE CE PLAN, arbitrage utilisateur.**
Voir §5 : la premisse du chantier ne resiste pas a la lecture du garde.

---

## 4. Phases d'execution

Regle d'ordre : une etape n'est ouverte que lorsque la precedente est close ET son gate
rejoue. Statuts : `[x]` fait, `[~]` couvert ailleurs (avec reference), `[!]` non traite
(avec justification ecrite). **Aucune case vide a la cloture.** Zero fix hors perimetre :
les decouvertes vont au §6.

### Etape 1 — `MatchFacts` descend dans `domain` (refactor pur, zero changement de comportement)

- [ ] 1.1 Creer `internal/domain/match_facts.go` : y DEPLACER `MatchFacts`, `MatchPlayerFact`
      et la methode `Empty()`, avec leurs commentaires d'origine (ils portent le raisonnement
      du triplet d'appariement — ne rien reecrire).
- [ ] 1.2 Dans `internal/port/services.go` : remplacer les deux declarations par des ALIAS
      (`type MatchFacts = domain.MatchFacts`, `type MatchPlayerFact = domain.MatchPlayerFact`),
      en conservant `ReplayFactsRepo` tel quel.
- [ ] 1.3 Verifier qu'AUCUN autre fichier ne change (l'alias est transparent).

Gate 1 : `go build ./...` && `go vet ./...` && `go test ./internal/replaybuild/... ./internal/port/... ./internal/platform/duckdb/...`
et `git diff --stat` ne montre QUE `domain/match_facts.go` et `port/services.go`.

### Etape 2 — Les faits voyagent dans le payload du job

- [ ] 2.1 `domain/build_queue.go` : ajouter `Facts *MatchFacts \`json:"facts,omitempty"\`` a
      `BuildQueuePayload`, avec le commentaire qui dit POURQUOI (l'ouvrier n'a pas de base) et
      ce que coute son absence (renvoi a la mesure du §2).
- [ ] 2.2 `wire/registry_build_queue.go` : dans `EnqueueReplayBuild`, resoudre les faits via
      `replayMatchFacts(ctx, sharedSQL, fullID)` — la fonction EXISTE deja (`registry_replay_build.go:143`)
      — et les poser dans le payload. Des faits vides restent une mise en file LEGITIME
      (match hors registre) : journaliser, jamais echouer.
- [ ] 2.3 `cmd/replay-worker/job.go` : consommer `p.Facts` au lieu du litteral vide ligne 137.
      Faits absents => conserver l'avertissement, **corrige** pour nommer aussi les zones et
      les socles de drapeau (la mesure du §2 prouve que la phrase actuelle sous-estime).
- [ ] 2.4 Tests : (a) `EnqueueReplayBuild` pose les faits lus en base dans le payload ;
      (b) un payload serialise puis deserialise conserve les faits a l'identique
      (aller-retour JSON, c'est le vrai chemin : le payload est stocke en `payload_json`) ;
      (c) un match hors registre met en file avec des faits vides sans erreur.

Gate 2 : `go build ./...` && `go vet ./...` && `go test ./internal/api/wire/... ./internal/domain/... ./cmd/replay-worker/...`

### Etape 3 — La fraicheur cesse de figer un artefact appauvri

- [ ] 3.1 `internal/replaybuild/artifact_store.go` (ou `replaybuild.go`) : ajouter le predicat
      PUR `ArtifactHasMatchFacts(path string) bool` — vrai si `scoreTimeline.players` est non
      vide. Documenter le signal et pourquoi il n'y a pas de champ dedie (D3).
- [ ] 3.2 Cabler la fraicheur facts-aware chez l'appelant qui a la base :
      `sync/replayartifacts/artifacts.go` (`enqueueAll`, ligne ~290) — un artefact a jour au
      schema MAIS sans faits, pour un match dont la base connait des participants, doit etre
      REMIS EN FILE. Journaliser le motif (INFO), jamais silencieusement.
- [ ] 3.3 Tests : artefact au bon schema sans faits + base avec participants => re-enfile ;
      artefact avec faits => saute ; match sans participants => saute (pas de re-cuisson
      perpetuelle).
- [ ] 3.4 `[!]` **Ne PAS toucher `cmd/levelup/cmd_backfill_replay.go`** (lot « blindage
      backfill parent/enfant » en cours cote session utilisateur). Le point de couture est
      ecrit au §7.

Gate 3 : `go build ./...` && `go vet ./...` && `go test ./internal/replaybuild/... ./internal/sync/replayartifacts/...`

### Etape 4 — Preuve de bout en bout : l'ouvrier rend l'artefact COMPLET

- [ ] 4.1 Etendre la preuve E2E existante (`internal/api/wire/build_queue_transport_e2e_cgo_test.go`,
      qui prouve deja l'octet-a-octet du transport) pour que le job enfilé porte des faits et
      que l'artefact recu les PORTE : `objectives` non vide, `zoneStates` non vide,
      `scoreTimeline.players` non vide.
- [ ] 4.2 Comparaison STRUCTURELLE ouvrier vs cuisson locale sur le meme temoin : memes
      compteurs sur les champs du tableau §2. C'est le critere de succes du plan.

Gate 4 : `go test -tags=integration -p 1 -run 'TestOuvrier|TestBuildQueue' ./internal/api/wire/`

### Gates de cloture (rejoues integralement)

`go build ./...` · `go vet ./...` · tests des paquets touches ·
`go test -tags=integration -p 1 ./...` SI persist/sync touche (anti-ART) · `make go-api-lint` ·
entree `.ai/thought_log.md` · registre `.ai/V7.5/REGISTRE_REPORTS.md` mis a jour (ligne 259).

---

## 5. Le garde local — la premisse du chantier ne resiste pas au code

La consigne d'ouverture pose que « la condition historique du rejeu public est tombee
(mystere CTF solde le 11/08 : aucun impact sur les tirs fatals, 99,55 % acquis) » et que la
levee du garde est « une MARCHE DE CE CHANTIER ». **Le fichier du garde dit autre chose, et
c'est lui qui fait foi.**

`internal/api/handlers/replay_local_gate.go:53-54` — le critere n'est pas « les tirs fatals
sont acquis », c'est :

> CRITERE MESURABLE   couverture des tirs >= 88 % et `coverage.verdict.bridge` nominal sur
> TOUS les films du corpus nomme ci-dessous, sans collision de trace.

Et son etat, ligne 26 :

> **LE CRITERE CI-DESSOUS N'EST PLUS SATISFAIT : `64e8adfa` tombe a 87,39 %, sous le plancher
> de 88 %.**

Ligne 28-32 :

> LE GARDE RESTE DONC EN PLACE, et c'est le sens dans lequel il doit echouer : un critere qui
> n'est plus atteint interdit le retrait, il ne se renegocie pas. [...] L'arbitrage « justesse
> contre seuil d'activation » appartient a l'utilisateur ; tant qu'il n'est pas rendu, le
> corpus passe 6 sur 7.

Les 99,55 % du mystere CTF portent sur **les tirs FATALS** (qui / quand / quelle arme) ;
le garde mesure **la couverture de TOUS les tirs par film**. Deux metriques differentes : la
premiere ne satisfait pas la seconde. Le registre ajoute (`REGISTRE_REPORTS.md:80`) que le
seuil de 88 % lui-meme doit etre **RE-STATUE**, parce qu'il avait ete calibre sur une
couverture GONFLEE par des attributions depuis retirees — « comparer 87,39 % juste a 88 %
gonfle = pommes/oranges ».

**Conclusion, et elle est tranchee dans le sens de la prudence :** la levee du garde n'est pas
une tache d'ingenierie de ce plan, c'est un ARBITRAGE UTILISATEUR que le garde lui-meme
reserve explicitement. Ce plan ne la programme pas. Deux options a soumettre :

- **(a)** re-statuer le seuil sur le corpus re-cuit AVEC faits, puis decider — c'est la voie
  que le registre prescrit, et elle depend de la bombe RAM (§6, decouverte 1) ;
- **(b)** decision utilisateur explicite d'activer malgre `64e8adfa` a 87,39 %, ecrite dans
  le garde avec sa date et son motif.

**Ordre non negociable dans les deux cas : les FAITS d'abord.** Activer l'ouvrier avant
l'etape 3 empoisonnerait durablement le cache d'artefacts (D3).

---

## 6. Decouvertes non traitees (consignees, PAS corrigees)

1. **Bombe RAM a la cuisson avec faits** — `NamedEventsFrom`/`incrementTimes`, OOM ~26 Go,
   consignee au registre le 2026-08-24. Elle bloque toute re-cuisson de MASSE, donc l'option
   (a) du §5 et tout bump de schema. Hors perimetre de ce plan, mais c'est le vrai chemin
   critique de l'activation.
2. **Les URL pre-signees vieillissent DANS la file.** Le payload est resolu a la mise en file
   et reporte d'evenement en evenement sans etre rafraichi (`ops/build_queue.go:165`, `:312`,
   `:365`) ; il n'est deserialise qu'a la reponse de claim (`:180-182`). Un job qui attend un
   ouvrier plus longtemps que la validite du CDN Azure echouera au telechargement, puis
   consommera ses 3 tentatives. Non mesure (validite Azure inconnue). A instruire avant de
   promettre une file profonde.
3. **Le reclaim n'est pas rejoue a l'arret** — `wire/registry_monitoring_store.go:98-101` :
   la branche `ctx.Done()` appelle `flushDetections` et `sweepMonitoringRetention`, pas
   `reclaimBuildQueue`. Consequence bornee (le prochain claim le fera).
4. **Aucun test ne fait tourner DEUX ouvriers simultanement** (registre ligne 134). La
   mecanique le permet ; ne pas le promettre.
5. **L'audit de portabilite du 20/08 n'existe pas** dans `.ai/` : le lot L9 de
   `PLAN_SUPERVISION_2026-08-20.md:52` attend toujours un rapport E3 jamais versionne. La
   contrainte « aucune dependance au jeu installe cote ouvrier » reste NEANMOINS etablie, par
   un autre document : `.ai/V7.5/replay2d/CLE_USB_REJEU_2D.md:200-202` (« Le pipeline du rejeu
   ne consomme AUCUNE capture memoire ») et le gate final du plan transport, ou l'ouvrier a
   tourne avec « un depot A LUI (copie des references, 1,5 Mo) ».

---

## 7. Point de couture avec le lot « blindage backfill parent/enfant »

`cmd/levelup/cmd_backfill_replay.go` est TENU par une autre session et n'est pas touche ici.
Ce plan y laisse une seule dette, cote appelant :

- Le CLI lit deja les faits (`cmd_backfill_replay.go:369-372`, via `duckdb.NewReplayFactsRepo`)
  — il n'est donc PAS concerne par le trou de l'etape 2.
- Il est concerne par l'etape 3 : sa reprise se fonde sur `replaybuild.ArtifactUpToDate`, qui
  continuera d'ignorer les faits. **Interface offerte** : le predicat pur
  `replaybuild.ArtifactHasMatchFacts(path) bool` (etape 3.1). Le porteur du lot backfill n'a
  qu'a combiner `ArtifactUpToDate(p) && (facts.Empty() || ArtifactHasMatchFacts(p))` la ou il
  decide de sauter un match. Aucune modification de signature existante.

## 8. Reprise de session

Avancement = les cases de ce fichier. Reprendre a la premiere etape non close, apres avoir
rejoue son gate (le code a pu bouger : re-ouvrir les fichiers cibles avant de coder).
