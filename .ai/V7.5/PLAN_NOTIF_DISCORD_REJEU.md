# PLAN — Notification Discord « rejeu 2D prêt », groupée anti-spam (LOT B, pt 5)

> Contrat d'exécution : skill `plan-execution` (ordre strict, aucune étape différée,
> chaque item statué `[x]` / `[~]` / `[!]`, zéro fix opportuniste hors périmètre).
> Plan parent : `.ai/V7.5/PLAN_REPLAY2D_NOTION_2026-08-25.md`, Lot B.
> Worktree : `LevelUp-wt-notif-rejeu`, branche `wt/notif-rejeu` (base `c42624dd5`).
> Exécuteur : ne touche NI `thought_log` NI `REGISTRE_REPORTS` NI Notion ; jamais
> `git add -A` ; jamais de push. Verdicts de gates : commandes NUES.
>
> Etat : B0 LIVRÉ et VALIDÉ (superviseur, 3 arbitrages tranchés au §5) — B1 et B2 LIVRÉS,
> gates verts (§7). Reste : relecture / fusion par le superviseur.

Besoin produit (encadré Notion « REPLAY 2D », point 5) : « Quand un replay a été
généré/récupéré par le local ou le worker, envoyer une notif Discord, attention au spam,
faudrait peut-être grouper. »

---

## 1. B0 — Constats vérifiés sur pièces

### 1.1 Le point d'ancrage serveur UNIQUE

`writeArtifactBytes` — `apps/go-api/internal/replaybuild/artifact_store.go:158`.

C'est le SEUL point d'écriture d'un artefact de rejeu du dépôt, et son en-tête le dit déjà
explicitement (`artifact_store.go:140-146` : « Quatre écrivains canoniques visent ce même
fichier […] tous les quatre finissent ICI »). Les deux chaînes exigées, vérifiées ligne à
ligne :

**Chaîne A — génération LOCALE (fil de l'eau post-sync, in-process serveur)**

| # | Fichier:ligne | Geste |
|---|---|---|
| 1 | `internal/sync/replayartifacts/artifacts.go:244` | `Run` — étape post-sync 1.58, placement `local` |
| 2 | `internal/sync/replayartifacts/artifacts.go:276` puis `:390` | `buildAll` -> `builder.BuildMatch(...)` |
| 3 | `internal/replaybuild/replaybuild.go:182` puis `:221` | `BuildMatch` -> `writeArtifact(outPath, doc)` |
| 4 | `internal/replaybuild/replaybuild.go:418-423` | `writeArtifact` -> `writeArtifactBytes(outPath, blob)` |
| 5 | `internal/replaybuild/artifact_store.go:158` | **ancrage** |

**Chaîne B — livraison par l'OUVRIER distant (dépôt HTTP, in-process serveur)**

| # | Fichier:ligne | Geste |
|---|---|---|
| 1 | `cmd/replay-worker/job.go:238-250` | l'ouvrier construit chez lui, relit le fichier, `sendArtifact` |
| 2 | `cmd/replay-worker/protocol.go:86-88` | `POST /build-queue/artifact?job_id=…&worker_id=…` |
| 3 | `internal/api/handlers/build_worker_artifact.go:61` puis `:85` | route Huma -> `h.storeArtifact(ctx, jobID, workerID, in.RawBody)` |
| 4 | `internal/api/wire/server_build_worker.go:28` | `WithArtifactStore(reg.StoreBuildArtifact)` (le câblage qui ferme la chaîne) |
| 5 | `internal/api/wire/registry_build_queue.go:207` puis `:220` | `StoreBuildArtifact` -> `replaybuild.StoreArtifact(...)` |
| 6 | `internal/replaybuild/artifact_store.go:62` puis `:71` | `StoreArtifact` -> `writeArtifactBytes(outPath, blob)` |
| 7 | `internal/replaybuild/artifact_store.go:158` | **ancrage** |

**Les deux autres écrivains du même point** (cités par l'en-tête, vérifiés) :

- action admin `RunReplayBuild` — `internal/api/wire/registry_replay_build.go:66` ->
  `BuildMatch` -> même chaîne A à partir de l'étape 3. **IN-PROCESS SERVEUR** : il sera
  donc notifié lui aussi, et c'est VOULU (un artefact construit à la main est un rejeu
  qui devient disponible, exactement comme les deux autres).
- CLI de rattrapage `levelup backfill-replay` (enfant) —
  `cmd/levelup/cmd_backfill_replay_child.go:53`. **AUTRE PROCESS** : il ne verra jamais
  le puits de notification du serveur, donc un backfill de masse ne produira AUCUN
  message. Ce n'est pas un trou, c'est la propriété qu'on veut (un backfill = des
  centaines d'artefacts, jamais une nouvelle à annoncer).

**Corollaire important** : l'ouvrier appelle lui aussi `writeArtifactBytes` dans SON
process (étape B1 ci-dessus). Quand il tourne sur le poste de dev avec `--work` pointant
le cache du dépôt, le MÊME fichier est écrit deux fois (une fois par l'ouvrier, une fois
par le serveur au dépôt). Comme seul le process serveur câble le puits, il n'y a qu'UNE
notification. À ne pas casser en câblant le puits ailleurs qu'au boot du serveur.

### 1.2 Ce que l'ancrage NE doit PAS notifier

`writeArtifactBytes` a deux sorties non-écrivantes qu'il faut distinguer :

- `artifact_store.go:159-168` : garde anti-régression — l'artefact est REFUSÉ (il
  rétrograderait celui en place), un WARN est journalisé, `replay_artifact_downgrade_refused_total`
  monte, et la fonction rend le digest de CE QUI RESTE sur le disque, **sans erreur**. Un
  puits naïf placé sur le `return` final ne le verrait pas — il faut publier
  UNIQUEMENT après l'écriture réelle (`atomicfile.WriteFile` ok, `artifact_store.go:172`).
- `artifact_store.go:169-174` : erreurs `MkdirAll` / `WriteFile` — rien n'est écrit, rien
  ne doit être notifié.

### 1.3 L'infrastructure Discord EXISTANTE — ce qu'on réutilise

| Brique | Fichier:ligne | Réutilisation |
|---|---|---|
| Client webhook failsafe | `internal/notify/discord.go:226` (`SendWebhook`) / `:235` (`SendWebhookCtx`) | TEL QUEL. Expurge le secret des logs (`sanitizeSendError`, `:209`), jamais de panic. |
| Types d'embed | `internal/notify/discord.go:31-62` | TEL QUEL (`Embed`, `EmbedField`, `WebhookPayload`). |
| Résolution du webhook env-ou-store | `internal/notify/discord.go:158-170` + `internal/config/config_settings.go:62` (`DiscordWebhookURLFromEnv`) | TEL QUEL via `LoadNotifyConfig` / `LoadNotifyConfigForTitle`. Précédence : `LEVELUP_DISCORD_WEBHOOK_URL` > `DISCORD_WEBHOOK_URL` > `discord_webhook_url` du store, avec contrôle de préfixe. |
| Indicateur `discord_webhook_present` | `internal/platform/settings/store.go:494` (`DiscordWebhookURLPresent`) ; test `internal/api/handlers/settings_media_test.go:381` | LU seulement (rien à modifier) : c'est ce champ qui prouve que la source « env OU store » est déjà canonique. |
| Gate global | `internal/notify/discord.go:154` (`discord_notifications_enabled`) | TEL QUEL. |
| i18n FR/EN inline | `internal/notify/discord.go:283-397` (`discordStrings`) + `:401` (`T`) | ON AJOUTE 4 clés (§3.4). Exemption emojis documentée `discord.go:278-282` (contenu de message, pas décoration de code). |
| Libellés title-aware | `internal/notify/labels_resolver.go:28` (`LabelsForSlug`) + `internal/notify/labels.go` (`discordFooterText`) | TEL QUEL — title-agnostic sans le moindre `slug ==`. |
| Patron « politique pure + boucle + envoi » | `internal/ops/disk_watch.go:70` (`ShouldNotifyDisk(state, current, now)`) + `internal/api/wire/registry_monitoring_diskwatch.go:44` (`RunDiskWatchLoop`) + `internal/notify/disk.go:26` (`NotifyDiskAlert`) | PATRON COPIÉ : décision PURE avec horloge en paramètre, boucle dans `wire`, embed dans `notify`. |
| Patron « puits process câblé au boot » | `internal/notify/labels_resolver.go:13-23` (`SetDefaultLabelsResolver`, RWMutex, nil = failsafe) | PATRON COPIÉ pour le puits d'artefact (§2.1). |
| Câblage d'une boucle au boot | `cmd/server/main.go:1157-1161` (`RunDiskWatchLoop` sur `schedulerCtx`/`schedulerWG`) | PATRON COPIÉ. |

### 1.4 Ce qui MANQUE (à écrire en B1)

1. Aucune publication d'événement au point d'écriture : `writeArtifactBytes` n'expose rien
   (seulement `observability.IncCounter` sur le refus, `artifact_store.go:166`).
2. Aucun groupeur par fenêtre à horloge injectable. La seule coalescence existante est
   celle des notifications IN-APP (`notifications.Emitter.EmitCoalesced`,
   `internal/notifications/emitter.go:24`) — voir §1.5, elle n'est PAS réutilisable ici.
3. Aucun embed « rejeux prêts » ni clé i18n correspondante (grep `Replay|rejeu` dans
   `internal/notify/` et `internal/notifications/` : 0 occurrence).

### 1.5 L'infra notifications IN-APP : examinée, NON retenue (justifié)

`internal/notifications` (service per-joueur persisté + `EmitCoalesced` avec fenêtre) et
son relais externe `internal/notifications/external` ont été examinés. Non retenus :

- le relais externe est **coach-only et opt-in strict** : `dispatcher.go:84` exige
  `ncfg.NotifyCoach` (défaut FALSE, décision vie privée assumée, `discord.go:92-98`), et
  `categories.go:15` liste exclusivement les catégories du coach, sous garde-rail
  `categories_guardrail_test.go` qui ÉCHOUE si une catégorie non-coach y entre. Y faire
  entrer « rejeu prêt » violerait un invariant explicite et livrerait la feature OFF par
  défaut — interdit par CLAUDE.md règle 11 ;
- les notifications in-app sont **per-joueur** (préférences, xuid, route cible) alors
  qu'un artefact de rejeu n'appartient à aucun joueur en particulier ;
- `EmitCoalesced` fusionne **après coup** (première notif émise tout de suite, les
  suivantes sommées dedans) : le premier message partirait immédiatement, ce qui est
  exactement le comportement anti-spam qu'on veut éviter.

Le chemin retenu est celui de l'alerte disque : `notify` + politique pure + boucle.
(Une notification in-app « rejeu prêt » reste possible plus tard ; hors périmètre — §7.)

---

## 2. Architecture cible

```
writeArtifactBytes (ancrage, artifact_store.go)
    -- écriture RÉELLE seulement -->  replaybuild.publishArtifactStored(ev)
                                            |  (puits process, nil par défaut)
                                            v
wire (boot) : replaybuild.SetArtifactStoredSink(closure) --> replaynotify.Grouper.Add(now, ev)
                                            |
wire : RunReplayNotifyLoop(ctx)  --tick-->  Grouper.Due(now) --> []Batch
                                            |
                                            v
                       wire : résolution des liens (1 lecture shared courte)
                                            |
                                            v
                       notify.NotifyReplayBatch(cfg, batch)  --> SendWebhookCtx
```

Fichiers **NEUFS** — 5 de production (+ 4 de test, §7) :

| Fichier | Rôle | Livré |
|---|---|---|
| `internal/replaybuild/artifact_events.go` | `ArtifactStored` + `SetArtifactStoredSink` + `publishArtifactStored` (RWMutex, nil = no-op, panique récupérée) | 89 L |
| `internal/replaynotify/group.go` | groupeur PUR (aucun timer, aucune goroutine, `now` en paramètre) | 181 L |
| `internal/notify/replay.go` | `NotifyReplayBatch` + `buildReplayEmbed` | 101 L |
| `internal/api/wire/registry_replay_notify.go` | câblage du puits + boucle + résolution des liens | 226 L |
| `internal/domain/replay_link.go` | `ReplayLinkTarget` (arbitrage A1) | 24 L |

Fichiers **TOUCHÉS** (9) : `internal/replaybuild/artifact_store.go` (publication + params
d'identité), `internal/replaybuild/replaybuild.go` (passage de l'identité à
`writeArtifact`), `internal/replaybuild/artifact_store_test.go` (2 appels mis à jour),
`internal/notify/discord.go` (4 clés i18n + champ `NotifyReplay` + sa lecture),
`internal/port/services.go` (`ReplayLinkRepo`), `internal/platform/duckdb/replay_facts_repo.go`
(`LinkTargetsForMatches`), `internal/api/wire/registry_notifications.go` (bascule sur
`publicBaseURL()`), `cmd/server/main.go` (câblage + boucle), `app_settings.example.json` +
`docs/CONFIGURATION.md` + `docs/FR/CONFIGURATION.md`.

Respect des règles transverses : logique **hors handler HTTP** (le handler
`build_worker_artifact.go` n'est pas modifié) ; `slog.InfoContext/ErrorContext`
systématiques ; aucune erreur avalée (chaque échec = log + compteur expvar) ; aucun
`slug ==` (le titre est une donnée portée par l'événement, les libellés passent par
`LabelsForSlug`) ; aucun flag qui laisse la feature OFF (§3.6).

### 2.1 Le puits (pourquoi un enregistrement process, et pas 3 appels)

`writeArtifactBytes` est appelé depuis 4 écrivains, dans 3 binaires différents. Threader
un notifieur jusqu'à lui imposerait 3 sites d'émission distincts (post-sync, dépôt
ouvrier, action admin) — soit la 3e copie d'un même geste, interdite par CLAUDE.md
règle 6, et la garantie que les trois divergeront.

Le puits process est le patron DÉJÀ en place dans ce dépôt pour exactement ce besoin :
`notify.SetDefaultLabelsResolver` (`labels_resolver.go:19`, RWMutex, nil = repli failsafe)
et `observability.IncCounter`, appelé à la ligne 166 du fichier d'ancrage lui-même.

Garde-rail obligatoire (B2) : un test grep interdit plus d'UN appelant de production de
`SetArtifactStoredSink` (le boot serveur). Sans lui, un second câblage ferait double
notification.

---

## 3. Le groupeur — contrat

### 3.1 Comportement (cadre imposé)

- Le **1er événement d'un titre arme la fenêtre** de ce titre. Défaut : **10 minutes**
  (constante nommée `DefaultWindow`, pas de nombre magique).
- À échéance : **UN seul message** « N rejeux prêts » + la liste des matchs (avec liens
  quand ils sont résolvables).
- Après un flush, la fenêtre du titre est DÉSARMÉE : l'événement suivant en réarme une
  neuve (fenêtre glissante interdite — elle repousserait le message indéfiniment sous un
  flux continu).
- Groupement **par titre** : un lot par `title_slug`, chacun avec sa propre fenêtre, sa
  langue et ses libellés (`LabelsForSlug`). C'est ce qui garde le message title-agnostic.

### 3.2 Invariants (tous testés en B2)

1. **Horloge injectée** : `Add(now, ev)` / `Due(now)`. Le paquet n'importe ni `time.Now`
   ni `time.Ticker` — la boucle `wire` fournit l'instant. Tests 100 % déterministes.
2. **Déduplication** par `(title_slug, match_id)` dans la fenêtre : une re-cuisson du même
   match ne compte qu'une fois, et n'avance PAS l'échéance (l'instant d'armement reste
   celui du 1er événement).
3. **Plafond de lot** `MaxListed` (20) : au-delà, la liste est tronquée et le message dit
   « … et N autres » (limites Discord : 4096 car. de description, 1024 par champ).
4. **Plafond mémoire** `MaxPending` (200 par titre) : au-delà, seul le compteur monte, les
   éléments excédentaires ne sont pas mémorisés + WARN une fois par fenêtre + compteur
   `replay_notify_pending_overflow_total`.
5. **Le flush draine toujours**, même si le webhook n'est pas configuré (sinon la mémoire
   grossit sans fin sur une instance sans Discord).
6. **Aucune erreur avalée** : envoi refusé par Discord -> WARN + `replay_notify_failed_total`,
   et le lot est ABANDONNÉ (pas de file de retransmission : un rejeu reste disponible dans
   l'app, il ne vaut pas une mécanique de retry).

### 3.3 Perte du groupe en cours au redémarrage — ACCEPTÉE

Un redémarrage du serveur (déploiement, crash) pendant une fenêtre armée perd les
événements en attente : aucun message ne partira pour ces artefacts.

C'est un choix, pas un oubli, et il diverge délibérément de `disk_watch` (qui PERSISTE son
état, `registry_monitoring_diskwatch.go:44-58`) :

- l'état disque persisté sert à ne PAS re-notifier (anti-rafale au boot) ; ici le risque
  symétrique n'existe pas : au pire on notifie MOINS ;
- le coût d'une perte est un message manqué. Les artefacts, eux, sont sur le disque et
  restent servis par l'app — rien n'est perdu côté produit ;
- persister imposerait un `adminstate.FileStore` de plus, sa réhydratation, son fichier
  corrompu à gérer, pour un gain nul.

À écrire tel quel dans l'en-tête de `internal/replaynotify/group.go` (le lecteur suivant
doit trouver la raison sur place, pas dans ce plan).

### 3.4 Le message (i18n FR + EN, parité obligatoire)

4 clés à ajouter dans `discordStrings` (`internal/notify/discord.go:283-397`) :

| Clé | FR | EN |
|---|---|---|
| `discord_replay_ready_title` | `Rejeux 2D prêts` | `2D replays ready` |
| `discord_replay_ready_desc_one` | `**1 rejeu** est prêt à être visionné.` | `**1 replay** is ready to watch.` |
| `discord_replay_ready_desc_many` | `**{count} rejeux** sont prêts à être visionnés.` | `**{count} replays** are ready to watch.` |
| `discord_replay_ready_more` | `… et {count} autre(s)` | `… and {count} more` |

FR sans anglicismes (« rejeu », jamais « replay » dans le texte affiché). Le nom du champ
de liste réutilise la convention existante ; footer = `discordFooterText(cfg.Labels)`
(`labels.go`), couleur = constante nommée locale au fichier (patron `disk.go:14-18`).

### 3.5 Les liens vers les matchs

La route front d'un rejeu est **joueur-scopée** :
`apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx`.
Il n'existe aucune route de match sans `playerSlug` (vérifié : `find *match*` sous
`apps/web/src/routes/`). Construction canonique côté Go :
`notifications.PlayerTargetRoute(titleSlug, playerSlug, "matches/"+matchID+"/replay")`
(`internal/notifications/routes.go:34` — point unique de vérité, garde-rail
`routes_guard_test.go`), préfixée par la base publique `LEVELUP_PUBLIC_BASE_URL`
(déjà lue à `internal/api/wire/registry_notifications.go:75` — B1 factorise les deux
lectures dans un helper non exporté de `wire` plutôt que d'en faire une 2e copie).

Résolution du `playerSlug` : **au flush, dans `wire`, une seule lecture shared courte pour
tout le lot** (au plus une par fenêtre de 10 min) — `match_participants` donne les xuid du
match (table confirmée : `internal/platform/duckdb/replay_facts_repo.go`, en-tête + 
`playerFacts`), croisés avec les joueurs connus de l'instance (même source que le
scheduler, `domain.SyncablePlayers`, `internal/scheduler/auto_sync_run.go:109` — l'accesseur
exact côté `wire` est à REVÉRIFIER SUR PIÈCES en B1). La même requête ramène `map_name`
depuis `match_registry` pour enrichir la ligne.

Dégradations, toutes journalisées, jamais fatales : base publique absente, aucun joueur
connu dans le match, ou lecture en échec -> la ligne affiche le match_id court seul,
DEBUG + `replay_notify_links_unresolved_total`. Le message part quand même.

### 3.6 Le réglage `discord_notify_replay` — défaut TRUE

Ajout d'une ligne dans `notifyConfigFromMap` (`discord.go:172-180`), sur le modèle exact
de `discord_notify_disk` : `cfg.NotifyReplay = boolValDefault(s, "discord_notify_replay", true)`.

Ce n'est PAS un flag interdit par CLAUDE.md règle 11 : le défaut est ACTIF, la feature est
livrée allumée. C'est une préférence utilisateur de catégorie, comme les 6 autres
(`sync`, `backfill`, `friends`, `version`, `reauth`, `disk`) — et c'est précisément la
réponse au « attention au spam » du besoin : pouvoir couper CETTE catégorie sans couper
Discord. Aucun impact front (aucun de ces réglages n'est exposé dans `apps/web` — vérifié :
`discord_notify_disk` n'apparaît que dans les docs, le Go et `app_settings.example.json`).

---

## 4. Décisions tranchées (ne pas re-débattre en B1)

1. Ancrage = `writeArtifactBytes`, publication APRÈS écriture réelle seulement.
2. Pas de champ « origine » (local / ouvrier / admin) dans l'événement : l'ancrage ne la
   connaît pas, et les trois appelants journalisent déjà la leur
   (`artifacts.go:403`, `registry_build_queue.go:231`, `registry_replay_build.go:70`).
   Le message produit n'en a pas besoin.
3. Signature élargie : `writeArtifactBytes(outPath, titleSlug string, blob []byte)` et
   `writeArtifact(outPath, titleSlug string, doc)` — 3 paramètres, sous le seuil. Les deux
   appelants connaissent le titre (`StoreArtifact` le reçoit, `Builder` porte `b.titleSlug`).
   Le `match_id` vient du digest déjà calculé (`digestFromBytes`, `replaybuild.go:390`).
4. Tick de la boucle : 1 minute (constante nommée). Conséquence assumée et documentée :
   le message part entre T+10 et T+11 min.
5. Fenêtre désarmée après flush (pas de fenêtre glissante).
6. Aucune persistance de l'état du groupeur (§3.3).
7. Aucun retry d'envoi (§3.2 invariant 6).

## 5. Arbitrages — TRANCHÉS par le superviseur (2026-08-26)

- **A1 — les liens (§3.5) : RETENU**, avec une CONDITION d'architecture ajoutée par le
  superviseur — la résolution `playerSlug` + `map_name` passe par une MÉTHODE DE REPO
  (chemin de lecture canonique `OpenReadForQuery`), **jamais** de SQL inline dans `wire`.
  Appliqué : `port.ReplayLinkRepo` + `duckdb.ReplayFactsRepo.LinkTargetsForMatches`.
- **A2 — `discord_notify_replay` défaut TRUE : RETENU**, docs bilingues dans le même commit.
- **A3 — l'action admin notifie aussi : RETENU** tel quel.

---

## 6. B1 — implémentation (LIVRÉE)

- [x] B1.1 `internal/replaybuild/artifact_events.go` (NEUF, 89 L) : `ArtifactStored`
  (`TitleSlug`, `MatchID`, `Path`, `Bytes`, `Tracks`, `SchemaVersion`),
  `SetArtifactStoredSink` (`:62`) + `publishArtifactStored` (`:74`) sous RWMutex, no-op si
  nil, panique du puits RÉCUPÉRÉE et journalisée (une notification cassée ne fait jamais
  échouer une écriture d'artefact).
- [x] B1.2 `artifact_store.go` : `writeArtifactBytes(outPath, titleSlug, matchID, blob)`
  (`:168`), publication APRÈS `atomicfile.WriteFile` (`:186`) et nulle part ailleurs ;
  `StoreArtifact` (`:71`) et `replaybuild.go` `writeArtifact` (`:418`) / `BuildMatch`
  (`:221`) mis à jour. En-têtes des deux fonctions actualisés dans le même commit.
  ÉCART ASSUMÉ vs plan : 4 paramètres au lieu de 3 — `matchID` est passé par l'appelant
  au lieu d'être lu dans le digest, parce que `doc.MatchID` peut porter la forme COURTE
  côté ouvrier (`validateArtifact` ne compare que les formes courtes) alors que
  `match_registry` est indexé par la forme COMPLÈTE : la résolution du lien échouerait.
- [x] B1.3 `internal/replaynotify/group.go` (NEUF, 181 L) : `New` / `Add(now, ev)` /
  `Due(now)` / `Pending()`, constantes `DefaultWindow` (10 min), `MaxListed` (20),
  `MaxPending` (200). Aucune goroutine, aucun `time.Now`. En-tête portant §3.3.
- [x] B1.4 `internal/notify/replay.go` (NEUF, 101 L) : `ReplayReadyItem`,
  `NotifyReplayBatch` + `buildReplayEmbed` (extrait pour être exerçable sans réseau, comme
  `BuildCoachEmbed`) ; 4 clés i18n FR/EN dans `discordStrings` (`discord.go:354`) ; champ
  `NotifyReplay` (`discord.go:96`) lu `boolValDefault(s, "discord_notify_replay", true)`
  (`discord.go:184`).
- [x] B1.5 `internal/api/wire/registry_replay_notify.go` (NEUF, 226 L) :
  `InstallReplayNotify` (`:56`), `RunReplayNotifyLoop` (`:82`), `flushReplayNotify` (`:96`),
  `sendReplayBatch` (`:108`), `replayReadyItems` (`:141`), `replayLinkTargets` (`:176`),
  `publicBaseURL` (`:224`). Compteurs : `replay_notify_events_total`,
  `replay_notify_events_invalid_total`, `replay_notify_batches_sent_total`,
  `replay_notify_artifacts_total`, `replay_notify_failed_total`,
  `replay_notify_links_unresolved_total`, `replay_notify_link_read_failed_total` ; jauges
  `replay_notify_pending_titles` / `_artifacts`.
  A1 (condition du superviseur) : `internal/domain/replay_link.go` (NEUF, 24 L),
  `port.ReplayLinkRepo` (`services.go:265`), `duckdb.ReplayFactsRepo.LinkTargetsForMatches`
  (`replay_facts_repo.go:135`) — zéro SQL dans `wire`, ouverture via
  `duckdb.OpenReadForQuery`.
  `registry_notifications.go:75` bascule sur `publicBaseURL()` (centralisation : la lecture
  d'env était sur le point de passer à 2 copies).
- [x] B1.6 `cmd/server/main.go` : `reg.InstallReplayNotify()` (`:1123`, dans le bloc de
  câblage du registry, avant le montage des routes ouvrier) + boucle sur
  `schedulerCtx`/`schedulerWG` (`:1180`, juste après `RunDiskWatchLoop`).
- [x] B1.7 Docs : `app_settings.example.json:21`, `docs/CONFIGURATION.md:297` et
  `docs/FR/CONFIGURATION.md:300` (EN + FR dans le même commit — le hook `docs-fr-sync` du
  pre-commit l'exige et l'a vérifié).

## 7. B2 — tests et gates (LIVRÉS, verts)

- [x] B2.1 `internal/replaynotify/group_test.go` (NEUF, 8 tests) : armement et non-sortie
  avant échéance (jusqu'à T+10-1ns) ; un seul lot pour toute la fenêtre, ordre d'arrivée
  conservé ; dédup sans décalage d'échéance ; fenêtre désarmée après flush et réarmée par
  l'événement suivant ; deux titres = deux fenêtres + ordre de sortie stable ; troncature
  `MaxListed` avec reste compté ; débordement `MaxPending` compté ; groupeur vide et
  événements sans identité.
- [x] B2.2 `internal/replaybuild/artifact_events_test.go` (NEUF, 5 tests) : chaîne ouvrier
  (`StoreArtifact`) et chaîne locale (`writeArtifact`) publient exactement 1 événement aux
  champs exacts ; refus anti-régression = 0 événement ; erreur d'écriture = 0 événement ;
  puits nil et puits en PANIQUE n'empêchent pas l'artefact d'atterrir.
  MORDANT PROUVÉ : `publishArtifactStored` neutralisé -> 2 tests FAIL, restauré -> verts.
- [x] B2.3 `internal/archlint/no_second_artifact_sink_test.go` (NEUF) — ÉCART vs plan :
  placé dans `archlint` (convention du dépôt pour les greps repo-wide) et non dans
  `replaybuild`. Allowlist datée à 2 entrées (le setter lui-même + le câblage de boot).
  MORDANT PROUVÉ : un appel ajouté dans `internal/ops/` -> FAIL nommant le fichier et la
  ligne ; mutation retirée.
- [x] B2.4 `internal/notify/replay_test.go` (NEUF, 5 tests) — ÉCART vs plan : fichier neuf
  plutôt qu'extension, car AUCUN test de parité sur `discordStrings` n'existait (vérifié
  sur pièces). Couvre : parité FR/EN des 4 clés + FR sans anglicisme (« rejeu », jamais
  « replay ») ; rendu liste/liens/reste omis ; singulier FR et EN ; les 3 portes de no-op ;
  défaut `NotifyReplay` ACTIF et coupure respectée.
- [x] B2.5 Gates, commandes NUES depuis `apps/go-api` (exit codes réels) :

  | Commande | Exit |
  |---|---|
  | `go test ./internal/replaynotify/` | `EXIT_1_REPLAYNOTIFY=0` |
  | `go test ./internal/replaybuild/` | `EXIT_2_REPLAYBUILD=0` |
  | `go test ./internal/notify/` | `EXIT_3_NOTIFY=0` |
  | `go test ./internal/platform/duckdb/` | `EXIT_4_DUCKDB=0` (37,4 s) |
  | `go test ./internal/api/wire/` | `EXIT_5_WIRE=0` |
  | `go test ./internal/archlint/` | `EXIT_6_ARCHLINT=0` |
  | `go test ./internal/port/ ./internal/domain/` | `EXIT_7_PORT_DOMAIN=0` |
  | `go vet ./...` (module COMPLET, typecheck des tests inclus) | `EXIT_8_VET_COMPLET=0` |
  | `gofmt -l` (arbres touchés) | `EXIT_9_GOFMT=0` (aucun fichier listé) |
  | `make go-api-lint` (racine) | `EXIT_10_GOLANGCI=0` — « 0 issues » |

  Pas de `-race` (incompatible DuckDB ici). CGO msys64. `go vet ./...` complet retenu en
  plus du plan : c'est le gate qui attrape les appelants de test périmés après un
  changement de signature (leçon du hotfix `3177a57a2`), et j'en ai changé une.

## 8. Risques et anti-régressions

| Risque | Parade |
|---|---|
| Double notification (puits câblé deux fois) | Garde-rail B2.3 + câblage unique au boot serveur |
| Notification sur un artefact NON écrit (refus anti-régression) | Publication placée après `atomicfile.WriteFile`, test B2.2(c) |
| Spam par backfill de masse | Le CLI de backfill est un AUTRE process : jamais de puits (§1.1) |
| Fuite mémoire si Discord n'est pas configuré | Le flush draine toujours + `MaxPending` (§3.2 invariants 4-5) |
| Message tronqué / rejeté par Discord | `MaxListed` = 20 + « et N autres » |
| Régression du contrat d'écriture | Aucun changement de comportement d'écriture : seuls la signature et une publication s'ajoutent ; `artifact_store_test.go` reste vert sans modification |

## 9. Découvertes (hors périmètre — consigner, NE PAS traiter)

- Une notification IN-APP « rejeu prêt » (`internal/notifications`) serait cohérente avec
  le reste du produit, mais impose une catégorie, des préférences et de l'i18n front.
  Hors périmètre du point 5 (qui dit « notif Discord »).
- `internal/notifications/external/dispatcher.go:83` relit `app_settings.json` à CHAQUE
  relais ; `notify.LoadNotifyConfig` fait de même à chaque appel. Sans conséquence à la
  fréquence d'un flush toutes les 10 min — noté, non traité.
- **SQL inline dans `wire`** : `registry_notifications.go:141` (`loadRecentMediaMatchIDs`)
  et `:169` (`loadParticipantXUIDs`) portent des requêtes en dur dans la couche de
  câblage — exactement ce que la condition A1 interdit désormais pour le neuf. Dette
  antérieure, NON traitée (hors périmètre du lot B).
- **Construction de placeholders SQL dupliquée** : `strings.Repeat("?,", n)` apparaît en
  15+ exemplaires (surtout sous `cmd/`, plus `internal/sync/enrichments.go:249` et
  `internal/sync/replayartifacts/artifacts.go:448`). Ma copie est un helper LOCAL à un
  fichier (`replay_facts_repo.go:184`, 2 usages dans la même fonction). Centraliser
  l'ensemble déborde très largement du lot — noté, non traité.
- `internal/notify` n'avait AUCUN test de parité FR/EN sur `discordStrings` avant ce lot :
  les ~45 clés préexistantes ne sont donc couvertes par rien de systématique. Mon test ne
  couvre QUE les 4 clés du lot (périmètre fermé) ; généraliser serait un lot à soi seul.

## 10. Journal d'exécution

- **2026-08-25 — B0 `[x]`** : plan écrit, ancrage `writeArtifactBytes`
  (`artifact_store.go:158`) PROUVÉ sur les deux chaînes (locale et ouvrier) avec
  fichier:ligne, infra Discord existante inventoriée (réutilisée / manquante), relais
  in-app examiné et écarté avec justification citée, groupeur spécifié (fenêtre 10 min,
  horloge injectée, perte au redémarrage acceptée et argumentée), B1/B2 découpés avec
  gates nus. Aucun code écrit. STOP — validation superviseur requise (3 arbitrages §5).
- **2026-08-26 — sync `[x]`** : `git fetch` + `git merge origin/feat/v75` (lot A UI, lot C0
  plafonds, schéma 19) — merge `efa1fdf53`, sans conflit. Vérification sur pièces APRÈS
  merge : `git diff --stat` de mes fichiers Go cibles entre le commit B0 et HEAD = VIDE ;
  `writeArtifactBytes` et `writeArtifact` relus, identiques au constat B0. Le schéma
  courant est bien passé à 19 (visible dans les logs de test), sans effet sur ce lot.
- **2026-08-26 — B1 `[x]` (B1.1 → B1.7)** : voir §6. Un point du plan CONTREDIT sur pièces
  et corrigé : l'accesseur des joueurs connus n'est PAS `domain.SyncablePlayers` (qui est
  un FILTRE d'activité de sync appliqué à une liste, `auto_sync_run.go:109`) mais
  `cfg.LoadPlayers(titleSlug)` — la source canonique de `db_profiles.json`, déjà utilisée
  pour le même besoin (xuid -> player_slug) par `xuidsToSlugs`
  (`registry_notifications.go:215`). Le filtre d'activité n'est délibérément PAS appliqué :
  un joueur dont le sync est en pause reste un propriétaire de lien parfaitement valide.
- **2026-08-26 — B2 `[x]` (B2.1 → B2.5)** : voir §7. Deux mordants prouvés par mutation
  (puits neutralisé ; second câblage ajouté), les deux mutations retirées. 10 gates nus à
  l'exit 0, dont `go vet ./...` complet et `make go-api-lint` (« 0 issues »).
- **RESTE** : relecture adversariale et fusion par le superviseur. Non exécuté ici (hors
  mandat de l'exécuteur) : entrée `thought_log`, `REGISTRE_REPORTS`, push, Notion.
