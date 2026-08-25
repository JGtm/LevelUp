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

*Cout assume de ce choix, dit plutot que taise* : quand un artefact EST a jour, il est
desormais lu DEUX fois (une par predicat, ~2 Mo chacune). Le surcout ne porte que sur les
matchs deja construits d'un cycle post-sync — negligeable devant la lecture DuckDB et la
resolution reseau du manifeste qui l'entourent. Le fusionner en un seul passage
(`ArtifactFreshness(path) (aJour, avecFaits bool)`) est possible mais ajouterait une
troisieme fonction publique la ou deux predicats se lisent mieux ; a reconsiderer si un
backfill de masse en fait un poste de cout mesure.

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

- [x] 1.1 Cree `internal/domain/match_facts.go` : `MatchFacts`, `MatchPlayerFact` et `Empty()`
      deplaces avec leurs commentaires d'origine, enrichis de la mesure du §2.
- [x] 1.2 `internal/port/services.go` : les deux declarations remplacees par des ALIAS
      (`type MatchFacts = domain.MatchFacts`), `ReplayFactsRepo` inchange.
- [x] 1.3 Verifie : `git diff --stat` ne montre que `port/services.go` (16 insertions,
      53 suppressions) plus le nouveau `domain/match_facts.go`. Les ~30 sites d'appel de
      `port.MatchFacts` n'ont pas bouge — c'est la propriete de l'alias.

Gate 1 : PASSE — `go build ./...` (CGO) exit 0, `go vet ./...` exit 0, `go test` OK sur
replaybuild + port + platform/duckdb + domain. Commit `907c0deda`.

### Etape 2 — Les faits voyagent dans le payload du job

- [x] 2.1 `domain/build_queue.go` : `Facts *MatchFacts` ajoute a `BuildQueuePayload`, avec le
      raisonnement (transport vs re-derivation vs rattrapage) et le cout mesure.
- [x] 2.2 `wire/registry_build_queue.go` : `EnqueueReplayBuild` resout les faits via
      `replayMatchFacts` sous les memes handles que l'identite du match, avant `closeAll()`.
      Faits vides => `WarnContext` explicite, pointeur laisse nil, mise en file poursuivie.
- [x] 2.3 `cmd/replay-worker/job.go` : consomme `p.Facts`. L'avertissement de degradation est
      CORRIGE — il nomme desormais les zones de mode et les socles de drapeau, que la mesure
      du §2 montre perdus et que la phrase d'origine passait sous silence.
- [x] 2.4 Tests : `TestBuildQueue_LesFaitsSurviventALaFile` (aller-retour complet par
      `payload_json` en base : variante, mapId, `TeamScores` pointeur, triplet d'appariement)
      et `TestBuildQueue_SansFaitsResteUnJobValide` (match hors registre : pointeur nil, pas
      d'objet vide invente).
      - `[~]` Le sous-cas « `EnqueueReplayBuild` pose les faits lus EN BASE » n'a pas de test
        direct : la fonction exige des tokens Halo (resolution du manifeste), ce que les deux
        E2E existants contournent deja en fabriquant le payload a la main
        (`build_queue_worker_binary_integration_test.go:191-192`). Ses deux moities sont
        couvertes separement — `FactsForMatch` par `platform/duckdb/replay_facts_repo_test.go`,
        le transport par le test ci-dessus — et la couture (3 lignes) est sous `build` + `vet`.

Gate 2 : PASSE — `go build ./...` exit 0, `go vet ./...` exit 0, 7 tests de file verts
(`internal/ops`), wire + domain OK. Commit `8c4791e85`.

### Etape 3 — La fraicheur cesse de figer un artefact appauvri

- [x] 3.1 `internal/replaybuild/replaybuild.go` : predicat PUR `ArtifactHasMatchFacts(path)`
      ajoute a cote d'`ArtifactUpToDate`. Le signal (`scoreTimeline.players`) est adosse au
      CONTRAT du document, pas seulement a la mesure : `document_score.go:147-148` dit deja
      « Players porte les joueurs dont le slot d'entite a ete apparie a une ligne de match.
      Vide quand l'appelant n'a pas fourni les lignes ».
- [x] 3.2 `sync/replayartifacts/artifacts.go` (`enqueueAll`) : un artefact a jour au schema
      MAIS sans faits, pour un match dont la base connait des participants, repart en file.
      Log INFO explicite + compteur `postsync_replay_artifacts_factless_requeued_total`.
- [x] 3.3 Tests : `TestEnqueueAll_ReEnfileUnArtefactAppauvri`,
      `TestEnqueueAll_ArtefactCompletResteSaute`, `TestEnqueueAll_SansFaitsConnus_NeReEnfilePas`
      (le garde-fou contre la re-cuisson perpetuelle) et `TestArtifactHasMatchFacts` (5 cas).
- [!] 3.4 `cmd/levelup/cmd_backfill_replay.go` NON TOUCHE — le lot « blindage backfill
      parent/enfant » le tient (consigne du superviseur). Sa reprise continue donc d'ignorer
      les faits. Le predicat lui est offert tel quel ; point de couture ecrit au §7. **C'est
      la seule dette ouverte de ce lot.**

Gate 3 : PASSE — `go build ./...` exit 0, `go vet ./...` exit 0, 5 tests replaybuild +
5 tests replayartifacts verts. Commit `6661a33af`.

### Etape 4 — Preuve de bout en bout : l'ouvrier rend l'artefact COMPLET

- [x] 4.1 `build_queue_worker_binary_integration_test.go` — c'est CE test-la qu'il fallait
      etendre (et non le transport voisin, qui fabrique son artefact a la main et ne fait donc
      jamais tourner le constructeur) : le job porte desormais les faits REELS du film temoin
      (`faitsDuTemoin()`, valeurs du releve versionne), et l'artefact livre est verifie COMPLET
      — `scoreTimeline.players` non vide et `coverage.score.teamIdentity != "unresolved"`.
      Pas d'assertion sur `objectives`/`zoneStates` : le temoin `000d5950` est un Slayer, il
      n'en a legitimement aucun. Leur transport est prouve au niveau de la file
      (`TestBuildQueue_LesFaitsSurviventALaFile`).
- [~] 4.2 Comparaison structurelle ouvrier vs cuisson locale : **deja faite, et c'est elle qui
      a produit le tableau du §2**. Les deux temoins ont ete cuits par le MEME
      `replaybuild.Builder` que celui de l'ouvrier, via `cmd/replay-build`, avec puis sans
      faits. L'ouvrier ne differe que par la PROVENANCE des faits (le job au lieu de la base),
      et ce maillon-la est teste separement. Refaire la comparaison en test exigerait un
      second decodage de film — non necessaire, et ecarte pour ne pas concurrencer le
      chantier `filmdec` en cours (consigne du superviseur).

Gate 4 : `go vet -tags="integration cgo" ./internal/api/wire/...` exit 0. Le test lui-meme
SKIPPE dans ce worktree (`000d5950` n'est pas dans son cache film : le cache vit dans le depot
principal) — verifie : `--- SKIP: TestOuvrierReel_ConstruitEtLivre`. Il s'executera sur la
machine qui porte le cache. **Aucun film n'a ete decode par ce lot.**

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

## 5bis. Ronde 1 de revue adversariale (2026-08-25) — ce qui a ete corrige, ce qui reste

Verdict d'entree : **0 P0, 4 P1, 4 P2**, et 12 conditions verifiees qui tiennent (alias
`domain` sans effet de bord, lectures placees avant `closeAll`, aucun elargissement de la
surface de l'ouvrier, payload borne, seuils de taille).

| # | Constat | Traitement |
|---|---|---|
| P1-1 | `openapi.yaml` non regenere — drift PROUVE (1 025 o). Pas cosmetique : `BuildQueuePayload` est `additionalProperties: false`, un validateur strict REFUSERAIT le payload | **corrige** `3a1edbe77` — chaine officielle, +47 l. au contrat, +18 a `generated.ts`, `make openapi-check` exit 0 sur ses deux moities, `tsc -b` exit 0 |
| P1-2 | Le garde testait `facts.Empty()` : un match au registre SANS participants re-cuisait un artefact identique, verdict qui ne bascule jamais | **corrige** `f269d6bca` — le verdict porte sur `len(Players)` ; test discriminant ajoute |
| P1-3 | Le predicat prenait « players => lines » pour « lines => players ». Trois vacuites legitimes MALGRE des faits | **corrige** `0a623a414` — renomme `ArtifactHasPlayerCounters` (il dit ce qu'il constate), les deux sens de l'implication ecrits, residu borne |
| P1-4 | Rien n'empechait un artefact COMPLET d'etre ecrase par un appauvri (jobs deja en file a la bascule) — perte DEFINITIVE | **corrige** `ed2d8c5b5` — `keepRicherArtifact` au rangement, 3 tests (regression refusee, enrichissement accepte, premier depot accepte) |
| P1-5 | *(trouve en traitant P2-1)* `buildAll` figeait aussi, sur le chemin qui a les faits en main | **corrige** `ed2d8c5b5`+`R1-5` — regle centralisee `etatArtefact`, lue par les deux chemins post-sync |
| P2-2 | Compteur `appauvris` incremente AVANT la mise en file ; resume muet si tout a ete refuse | **corrige** dans `f269d6bca` (comptage apres succes, resume des qu'il y a du travail, compte des refus publie) |
| P2-3 | Le commentaire nil/non-nil de `Facts` ne decrivait pas la regle reelle | **corrige** dans `0a623a414` |
| P2-4 | Cas discriminant manquant ; critere de succes E2E derriere un skip | **corrige** (test ajoute) / **consigne** : la couverture CI reelle de ce lot est l'aller-retour du payload et les gardes unitaires, PAS le E2E ouvrier (il SKIPPE sans cache film) |
| P2-1 | Portee du predicat plus etroite que sa doc | **partiellement corrige** (les DEUX chemins post-sync sont couverts) + **dette ecrite ci-dessous** |

**Portee reelle, sur pieces** — quatre appelants d'`ArtifactUpToDate` :

1. `replayartifacts.enqueueAll` — conscient des faits ;
2. `replayartifacts.buildAll` — conscient des faits (R1-5) ;
3. `wire.requireArtifactBeforeSuccess:261` — **schema seul, A DESSEIN** : c'est une verification
   de PRESENCE avant de valider un `complete`. La rendre consciente des faits ferait echouer un
   job dont l'artefact est legitimement appauvri (match hors registre) — un faux refus ;
4. `cmd/levelup/cmd_backfill_replay.go:145` — **schema seul**, fichier tenu par le lot blindage.

**DETTE OUVERTE — le cache DEJA empoisonne.** Aucun chemin ne repare un artefact appauvri qui
existe deja et dont le match ne sera plus jamais insere : la selection post-sync ne voit que les
matchs INSERES d'un cycle. Aujourd'hui c'est sans objet (l'ouvrier n'a jamais tourne en prod, et
les 34 artefacts du cache ont ete cuits localement), mais ca le deviendra des la premiere passe
d'ouvrier. **Condition de reprise** : une passe de rattrapage explicite — `levelup
backfill-replay --only-existing` etendu au critere « a jour MAIS sans compteurs de joueur alors
que la base a des lignes » — qui depend elle-meme de la bombe RAM `NamedEventsFrom` (§6.1) pour
etre executable en masse. A ouvrir AVANT la premiere activation prod de l'ouvrier, pas apres.

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
