# PLAN — Notification Discord « rejeu 2D prêt », groupée anti-spam (LOT B, pt 5)

> Contrat d'exécution : skill `plan-execution` (ordre strict, aucune étape différée,
> chaque item statué `[x]` / `[~]` / `[!]`, zéro fix opportuniste hors périmètre).
> Plan parent : `.ai/V7.5/PLAN_REPLAY2D_NOTION_2026-08-25.md`, Lot B.
> Worktree : `LevelUp-wt-notif-rejeu`, branche `wt/notif-rejeu` (base `c42624dd5`).
> Exécuteur : ne touche NI `thought_log` NI `REGISTRE_REPORTS` NI Notion ; jamais
> `git add -A` ; jamais de push. Verdicts de gates : commandes NUES.
>
> Etat : B0 (ce document) LIVRÉ — B1 et B2 sont BLOQUÉS jusqu'à validation superviseur.

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

Fichiers **NEUFS** (4) :

| Fichier | Rôle | Taille visée |
|---|---|---|
| `internal/replaybuild/artifact_events.go` | type `ArtifactStored` + `SetArtifactStoredSink` + `publishArtifactStored` (RWMutex, nil = no-op) | ~70 L |
| `internal/replaynotify/group.go` | groupeur PUR (aucun timer, aucune goroutine, `now` en paramètre) | ~150 L |
| `internal/notify/replay.go` | `NotifyReplayBatch(cfg, batch) bool` — construction de l'embed | ~90 L |
| `internal/api/wire/registry_replay_notify.go` | câblage du puits + `RunReplayNotifyLoop` + résolution des liens | ~150 L |

Fichiers **TOUCHÉS** (5) : `internal/replaybuild/artifact_store.go` (publication + 1 param
`titleSlug`), `internal/replaybuild/replaybuild.go` (passage du `titleSlug` à
`writeArtifact`), `internal/notify/discord.go` (4 clés i18n + champ `NotifyReplay`),
`cmd/server/main.go` (lancement de la boucle, sur le modèle des lignes 1157-1161),
`app_settings.example.json` + `docs/CONFIGURATION.md` + `docs/FR/CONFIGURATION.md`
(réglage `discord_notify_replay`).

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

## 5. Arbitrages demandés au superviseur (avant B1)

- **A1 — les liens (§3.5).** Retenu : lecture shared courte au flush pour résoudre
  `playerSlug` + `map_name`. C'est la SEULE partie de B1 qui touche la base. Repli
  possible si le superviseur veut un B1 plus serré : message sans lien profond
  (match_id court + rien d'autre), zéro accès base, ~40 L et 1 test de moins.
- **A2 — le réglage `discord_notify_replay` (§3.6).** Retenu : ajouté, défaut TRUE, docs
  bilingues mises à jour. Alternative : s'en passer et ne dépendre que de
  `discord_notifications_enabled` (moins de fichiers touchés, mais plus aucun moyen de
  couper cette seule catégorie).
- **A3 — l'action admin notifie aussi.** Retenu (§1.1) : un artefact construit à la main
  déclenche un message groupé comme les autres. Si le superviseur préfère l'exclure, il
  faudrait un critère au point d'ancrage — que l'ancrage n'a pas — donc ce serait un
  retour à l'émission par appelant, avec ses 3 copies. Recommandation : garder tel quel.

---

## 6. B1 — implémentation (BLOQUÉ jusqu'au go du superviseur)

- [ ] B1.1 `internal/replaybuild/artifact_events.go` : type `ArtifactStored`
  (`TitleSlug`, `MatchID`, `Path`, `Bytes`, `Tracks`, `SchemaVersion`),
  `SetArtifactStoredSink(fn)` + `publishArtifactStored(ev)` sous RWMutex, no-op si nil.
  En-tête expliquant POURQUOI un puits process (§2.1) avec le renvoi à `labels_resolver.go`.
- [ ] B1.2 `artifact_store.go` : `writeArtifactBytes` prend `titleSlug`, publie APRÈS
  `atomicfile.WriteFile` réussi, et **jamais** sur le chemin de refus anti-régression ni
  sur erreur. `replaybuild.go` : `writeArtifact` et son appelant `BuildMatch` passent
  `b.titleSlug`. Commentaires du fichier d'ancrage mis à jour dans le MÊME commit.
- [ ] B1.3 `internal/replaynotify/group.go` : groupeur pur (§3.1/§3.2) — `Add(now, ev)`,
  `Due(now) []Batch`, constantes `DefaultWindow` (10 min), `MaxListed` (20),
  `MaxPending` (200). Aucune goroutine, aucun `time.Now`. En-tête portant la décision
  §3.3 (perte au redémarrage ACCEPTÉE, et pourquoi elle diverge de `disk_watch`).
- [ ] B1.4 `internal/notify/replay.go` + 4 clés i18n FR/EN dans `discordStrings` + champ
  `NotifyReplay` dans `NotifyConfig` et sa lecture `boolValDefault(..., true)`.
  `NotifyReplayBatch` : no-op si webhook vide ou `NotifyReplay` false, `SendWebhookCtx`
  sinon, retour bool.
- [ ] B1.5 `internal/api/wire/registry_replay_notify.go` : câblage du puits au boot,
  `RunReplayNotifyLoop(ctx)` (ticker 1 min, sortie sur `ctx.Done()`, patron
  `RunDiskWatchLoop`), résolution des liens (§3.5, arbitrage A1), envoi, compteurs
  expvar `replay_notify_batches_sent_total` / `replay_notify_artifacts_total` /
  `replay_notify_failed_total` / `replay_notify_links_unresolved_total` /
  `replay_notify_pending_overflow_total`, logs `slog.InfoContext` à message STABLE.
- [ ] B1.6 `cmd/server/main.go` : lancement de la boucle sur `schedulerCtx`/`schedulerWG`
  (modèle lignes 1157-1161) + câblage du puits AVANT le montage des routes ouvrier.
- [ ] B1.7 Docs : `app_settings.example.json`, `docs/CONFIGURATION.md` et
  `docs/FR/CONFIGURATION.md` (parité FR/EN dans le même commit).

## 7. B2 — tests et gates (BLOQUÉ jusqu'au go)

- [ ] B2.1 `internal/replaynotify/group_test.go` — unitaires PURS, horloge injectée :
  (a) 1er événement arme, rien avant T+10 ; (b) à T+10 un lot unique de N ; (c) dédup
  `(title, match)` sans décaler l'échéance ; (d) après flush, la fenêtre est désarmée et
  l'événement suivant en réarme une neuve ; (e) deux titres = deux fenêtres indépendantes ;
  (f) `MaxListed` -> troncature + reste compté ; (g) `MaxPending` -> débordement compté,
  pas de fuite mémoire ; (h) groupeur vide -> aucun lot.
- [ ] B2.2 `internal/replaybuild/artifact_events_test.go` — **test du point d'ancrage** :
  (a) `StoreArtifact` (chaîne ouvrier) publie exactement 1 événement, champs exacts ;
  (b) `writeArtifact` (chaîne locale) publie exactement 1 événement ; (c) un dépôt
  RÉTROGRADANT (le cas de `TestStoreArtifact_RefuseLaRegression`,
  `artifact_store_test.go:149`) ne publie RIEN ; (d) puits nil = aucun panic.
- [ ] B2.3 Garde-rail : un seul appelant de production de `SetArtifactStoredSink`
  (test grep sur les sources hors `_test.go`, patron `internal/archlint/`).
- [ ] B2.4 `internal/notify` : parité FR/EN des 4 clés (le paquet a déjà ses tests de
  libellés — étendre plutôt que dupliquer) + no-op quand `NotifyReplay` est false.
- [ ] B2.5 Gates, commandes NUES (jamais de pipe qui masque l'exit), depuis
  `apps/go-api` :
  ```
  go test ./internal/replaynotify/...
  go test ./internal/replaybuild/...
  go test ./internal/notify/...
  go test ./internal/api/wire/...
  go test ./internal/archlint/...
  go vet ./internal/replaynotify/... ./internal/replaybuild/... ./internal/notify/... ./internal/api/wire/...
  ```
  puis, à la racine : `make go-api-lint`.
  PAS de `-race` (incompatible DuckDB dans ce dépôt). CGO requis (msys64) pour les paquets
  qui tirent le driver.

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

## 10. Journal d'exécution

- **2026-08-25 — B0 `[x]`** : plan écrit, ancrage `writeArtifactBytes`
  (`artifact_store.go:158`) PROUVÉ sur les deux chaînes (locale et ouvrier) avec
  fichier:ligne, infra Discord existante inventoriée (réutilisée / manquante), relais
  in-app examiné et écarté avec justification citée, groupeur spécifié (fenêtre 10 min,
  horloge injectée, perte au redémarrage acceptée et argumentée), B1/B2 découpés avec
  gates nus. Aucun code écrit. STOP — validation superviseur requise (3 arbitrages §5).
- B1 `[ ]` — bloqué.
- B2 `[ ]` — bloqué.
