# Plan de remediation durable — Contention sync <-> service (pool tokens + verrou DB partagee)

Statut : PLANIFIE (diagnostic termine, non implemente)
Date : 2026-07-01
Methode : diagnostic multi-agents (5 axes) + verification directe du code. Tous les points ancres file:line.
Contexte declencheur : l'utilisateur signale "le switch de titre bug (des fois il ne change pas)" + lenteur generale.
Directive utilisateur : PAS de rustine, du solide et perenne. Mieux gerer la rafale ET decoupler l'evaluation
sain/malsain des tokens du chemin de requete (sonde de sante background).

## 1. Cause racine unifiee

Les symptomes (switch qui echoue, pages lentes, 429 en cascade) ne sont PAS des bugs independants. Ce sont
5 manifestations d'UNE faute de conception : **absence d'isolation des ressources partagees entre le trafic
background (sync) et le trafic user-facing (pages, switch de titre)**.

Mecanisme central : le sync tient le writer RW du SharedProvider (le seul handle dont dependent AUSSI toutes
les lectures — contrainte mono-process DuckDB) pendant TOUTE sa boucle reseau. `acquireSharedWriter` est pris
en tete de `run()` (engine.go:259) et relache par defer en fin de run (engine.go:264), alors que ~95% du temps
est du reseau (fetch API Halo). Donc la fenetre RW ~= duree du fetch (dizaines de secondes) au lieu de ~= duree
des INSERT (sub-seconde). Pendant cette fenetre, tout `provider.Get(ctx)` lecteur est parque jusqu'a
readyTimeout=30s (provider.go:16,174-238) -> pages 500/timeout, et le `GetMatchCount` 5s du bootstrap
(bootstrap_repo.go:44) expire -> switch fait un rollback silencieux (appShellStore.ts:191-194).

Deux couplages aggravants distincts :
- Le bootRepo est FIGE sur le provider Infinite (main.go:623, titleSlug=DefaultSlug main.go:358). Le count est
  title-agnostic (port/repository.go:50). Donc un switch vers Halo 5 attend un provider INFINITE que le sync
  Infinite tient — alors que les bases sont isolees par fichier (data/titles/{slug}/). Bug de cablage pur.
- Le pool de tokens a la meme faute cote auth : aucun semaphore ne separe la concurrence sync (fan-out non
  borne engine.go:422) de la reserve user, et un seul 429 met les 7 tokens en cooldown global (pool.go:457-461).

**En une phrase** : le sync consomme sans plafond des ressources (writer DB, slots de tokens, exchanges XBL)
dont le chemin de service a besoin, et il les tient pendant ses I/O reseau au lieu de ses seules ecritures.

## 2. Reponse a la directive utilisateur : le modele sain/malsain est REACTIF, pas proactif

Verifie directement (pool.go:391-548) :
- Il existe un `refresherLoop` (ticker 10s, pool.go:477-548) MAIS ce n'est PAS une sonde de sante proactive.
  Il ne fait que RE-RAFRAICHIR (re-jouer la chaine d'exchange XBL) les slots deja marques `healthy=false`. Il
  ne TESTE jamais activement un token contre l'API Halo avant qu'une vraie requete en ait besoin.
- Un slot est `healthy=true` par defaut ; il ne devient `false` QUE via une VRAIE requete qui echoue :
  OnHTTPError (429/503, pool.go:395) ou MarkUnhealthy (401/403). Donc la sante est DECOUVERTE par l'echec
  d'une requete reelle — exactement le probleme que l'utilisateur pointe.
- Pire : un 429 isole marque LES 7 slots malsains d'un coup (scorched-earth, pool.go:457-461), puis le
  refresher lance jusqu'a 7 `go Refresh` en parallele (pool.go:514) -> 7 exchanges XBL concurrents -> 429 XBL
  (user.auth.xboxlive.com currentRequests). Le backoff exponentiel existe (globalCooldown<<consecutive429) mais
  est NEUTRALISE des qu'un Retry-After est present (pool.go:408-411 reset consecutive429=0) -> thrash loop 1s.
- Un token fraichement "healthy=true" (re-exchange) peut reprendre un 429 a la requete suivante, car le
  probleme est un quota par-compte, pas un token expire. Le refresh ne valide donc pas la vraie usabilite.

=> Le modele voulu par l'utilisateur (sonde background qui maintient en permanence l'ensemble "sain", requetes
ne tapant que du connu-bon) N'EXISTE PAS. Il y a un refresher reactif, pas un health-prober proactif. C'est
une cible explicite du fix (Chantiers 4 et 6).

## 3. Principe directeur

> Le sync ne doit JAMAIS tenir une ressource partagee pendant une operation reseau, et ne doit jamais pouvoir
> epuiser un budget dont depend le chemin de service.

Trois invariants :
- A. **Fenetre d'exclusivite = fenetre d'ecriture.** Le writer RW DuckDB n'est detenu QUE pendant les INSERT,
  jamais pendant fetch/discovery (RO ou hors-lease). C'est deja le modele Collect->Persist (worker
  combined_persister.go:83-91, fenetre sub-seconde) — a generaliser au cycle de vie de l'engine.
- B. **Budgets separes sync / user-facing.** Tout pool borne (tokens, exchanges XBL, concurrence DB) reserve
  une part garantie a l'interactif ; le background pioche dans un sous-budget qui ne mord jamais sur la reserve.
- C. **Le chemin critique du service ne depend d'aucune lecture lourde contendue.** Il tape la ressource de son
  PROPRE titre, avec budget d'attente court + fail-fast + degradation, jamais un blocage de 30s.

## 4. Chantiers ordonnes

### CRITIQUE pour le symptome utilisateur (switch echoue + lenteur)

**Chantier 1 — Resserrer la fenetre RW du sync (invariant A). LE FIX CENTRAL.**
- Quoi : ne plus tenir le writer RW shared pendant discovery+fetch. Discovery/known-set en lecture RO
  (provider.Get), chaque batch ecrit par le worker async (deja court), RW acquis uniquement pour insert/post-sync.
  Cible mesurable : shared_provider_rw_window_max_ms < 1500 (compteur Phase 0 deja livre).
- Pourquoi : racine commune des axes 2,3,4,5. Passer la fenetre RW de ~80s a quelques centaines de ms -> les
  pages ne stallent plus, le GetMatchCount 5s du bootstrap ne tombe plus en plein swap. Un chantier eteint 4/5 axes.
- Fichiers : engine.go:259 (deplacer acquireSharedWriter hors boucle), :264 (defer), :508-512 (release avant
  Drain, modele deja present en async), :341-483 (boucle fetch a passer en RO).
- A REUTILISER : le pipeline V2 / CycleOrchestrator (ADR 0027) implemente DEJA ce modele exact
  Discovery(RO)->Fetch(RO, aucun writer lease)->Persist(single writer). Cable en prod (main.go:1130) mais gate
  OFF par LEVELUP_SYNC_PIPELINE=v2 (auto_sync.go:433-439), avec concurrence explicite (cycle.go:76-81
  FetchSharedParallelism=8/FetchPlayerParallelism=4). Recommandation : ACTIVER V2 apres soak, plutot que
  reecrire la V1. Regle projet "pas de flags qui laissent une feature OFF" -> retirer le gate une fois valide.
- Effort : moyen. Risque : modere mais maitrise. ZERO risque ART (aucune modif du SQL d'ecriture, refactor
  strictement cote cycle-de-handle — certifie par PLAN_BSWAP). Verifier que le discovery RO ne re-fetch pas des
  dupes du meme cycle (non-fatal via pre-check idempotence submitMatchAsBatch:123).

**Chantier 2 — Bootstrap title-aware + non bloquant (invariant C). CRITIQUE pour le switch, LIVRABLE EN PREMIER.**
- Quoi : (a) resoudre le provider du titre COURANT via cfg.SharedManager.For(SharedDBPath(currentTitleSlug)) au
  lieu du bootRepo fige Infinite ; (b) executer GetMatchCount en best-effort borne (goroutine + select timeout
  1-2s, defaut profile_ready_no_sync) sur le modele exact de fetchPrivacyNonBlocking (bootstrap_service.go:296).
- Pourquoi : meme apres le Chantier 1, le couplage bootRepo->mauvais titre est un bug STRUCTUREL distinct. (a)
  l'elimine seul (le count H5 ne tape plus le provider Infinite sature). (b) garantit qu'un provider pathologique
  (StateError, retry loop provider_writer.go:322) ne fait jamais deborder le switch.
- Fichiers : main.go:623 (bootRepo fige), :358 (titleSlug=DefaultSlug), :535 (SharedManager deja injecte, pret),
  bootstrap_repo.go:44 (timeout 5s), bootstrap_service.go:296 (modele non bloquant a copier), :416-419 (catch
  degradation deja en place), manager.go:33 (Manager.For deduplique par path).
- Effort : rapide. Risque : faible (Manager.For deduplique, cas DefaultSlug = meme provider Infinite qu'aujourd'hui,
  byte-identique en mono-titre). Etendre les tests TestResolveSetupState_* (bootstrap_service_test.go:112-150).

> Chantiers 1 + 2 = eradication de la classe "switch qui echoue". Le 2 est indispensable meme si le 1 est parfait
> (couplage de cablage independant de la contention).

**Chantier 3 — Budget court fail-fast pour les lectures user-facing (invariant C).**
- Quoi : differencier le budget d'attente du reader par classe de requete. Lectures user (home, explorer, career)
  passent un maxWait court (~2-3s) au lieu de defaultReadyTimeout=30s ; au depassement -> 503 Retry-After (le
  handler home sait deja le faire pour isHandleClosedOrInvalidated, handlers/home.go:140-146) au lieu d'un 500 a
  31s. Le moins invasif : context.WithTimeout court dans un middleware "budget user-facing" sur le groupe
  /players/{slug}/pages/* — Get() honore deja ctx.Done() (provider.go:232).
- Pourquoi : filet qui borne la latence percue meme si un swap deborde ponctuellement. Complementaire au 1.
- Fichiers : provider.go:16 (defaultReadyTimeout), :174-238 (Get), shared_query_helpers.go:26, handlers/home.go:140-146.
- A REUTILISER : budget segmente deja present pour la progression post-sync (progressionSharedReadBudget=45s /
  Attempt=10s) — modele exact a repliquer cote lecture. Servir HomeMatchesCache TTL (home_service.go:214) en degradation.
- Effort : moyen. Risque : faible si 503 Retry-After (retente idempotente front).

### ROBUSTESSE SECONDAIRE (durcit l'auth sous charge ; PAS la cause du switch casse)

**Chantier 4 — Sante pool proactive + backoff par-token + fin du scorched-earth (Axe 1).**
- Quoi : (a) sur un 429 API Halo, marquer malsain UNIQUEMENT le slot du lease, pas les 7 ; cooldown global
  seulement si N tokens prennent 429 simultanement (vrai signal quota IP/app). (b) corriger le reset : meme avec
  Retry-After present, garder/incrementer consecutive429 et prendre max(retryAfter, globalCooldown<<shift) +
  jitter + plancher (5s). (c) DIRECTIVE UTILISATEUR : transformer le refresherLoop reactif en sonde de sante
  PROACTIVE — un probe leger en background qui maintient l'ensemble "sain" a jour hors du chemin de requete, de
  sorte qu'une vraie requete ne tire que dans du connu-bon et ne DECOUVRE jamais un token malsain par un 429.
- Fichiers : pool.go:457-461 (scorched-earth), :408-411 (reset consecutive429), :477-548 (refresherLoop reactif
  a faire evoluer en probe), :233 (spin acquire).
- A REUTILISER : le modele PAR-SLOT existe deja pour 401/403 (MarkUnhealthy sur lease.Gamertag, RC-1) — l'etendre
  au 429. Le backoff exponentiel existe deja (juste neutralise).
- Effort : rapide (a,b) / moyen (c probe proactive). Risque : faible.

**Chantier 5 — Borner la concurrence sync (invariant B, tokens).**
- Quoi : eg.SetLimit(syncFetchParallelism) sur la Phase 2 (au lieu de "Pas de SetLimit", engine.go:422) avec
  min(poolSize-reserve, config) ; idealement semaphore de fond distinct du canal round-robin + reserve de K
  slots pour l'interactif.
- Note : LARGEMENT ABSORBE si V2 est active (Chantier 1) — V2 a deja FetchSharedParallelism=8/FetchPlayerParallelism=4.
- Fichiers : engine.go:422, pool.go:160/197, main.go:1751. Effort : rapide (SetLimit). Risque : faible.

**Chantier 6 — Serialiser les exchanges XBL (Axe 1, 2e source de 429).**
- Quoi : limiteur de concurrence GLOBAL (semaphore ~2 ou rate.Limiter) partage par TOUS les appelants de
  ExchangeAccessTokenWithDescriptor : le refresherLoop (7 go Refresh en rafale, pool.go:514) ET le chemin
  user-facing refreshTokensFromDB (registry_auth.go:169). Refresh par vagues de 1-2. Retry respectant Retry-After
  sur le 429 XBL (halo_exchange.go:56, client nu aujourd'hui).
- Pourquoi : elimine le 429 user.auth.xboxlive.com. Distinct du 429 API Halo (deux chemins, meme endpoint XBL,
  aucune coordination).
- Fichiers : pool.go:507-544, halo_exchange.go:56, resolver.go:154, registry_auth.go:169. Effort : moyen. Risque : faible.

### DURCISSEMENT LONG TERME (optionnel)

**Chantier 7 — Snapshot immuable pour les lectures user-facing (invariant C, cible finale).**
Servir les lectures shared depuis le Parquet versionne immuable (internal/ops/snapshot*.go +
SnapshotPreferredSharedReader, deja livres en SCOPED MatchView) -> zero verrou partage avec le writer par
construction. Cutover global reverte car world_csr_leaderboard (non-match-immutable) casse sans fallback. Fix =
reconstruire tout le schema shared en :memory du snapshot + fallback live garanti table-par-table. Effort lourd,
risque frontiere-de-fraicheur. A n'engager qu'apres lecture des compteurs Phase 0 en prod.

## 5. Ce qu'on NE fait PAS

- Allonger les timeouts (readyTimeout, WriteTimeout, bootstrap 5s). Masque le stall, aggrave la latence. Le budget
  doit RETRECIR (fail-fast), pas grandir.
- Ajouter un simple catch front sur le switch. Le rollback silencieux existe deja ; le fix est backend (le
  /bootstrap ne doit jamais throw).
- Rustines sur le cooldown pool (baisser Retry-After, desactiver le cooldown global). Le fix est structurel (Ch. 4).
- Sortir le sync hors-process maintenant. Isolation ideale mais chantier a part ; 1+2+3 suffisent au symptome.
- Laisser un flag LEVELUP_SYNC_PIPELINE=v2 permanent si V2 devient le chemin durable — retirer le gate.
- Toucher au SQL d'ecriture / reintroduire des UPSERT concurrents (bug ART). INSERT-only via BatchBuilder/Persister.

## 6. Sequencement livrable

- Livraison 1 (le user constate "le switch marche toujours") : **Chantier 2** (bootstrap title-aware + non
  bloquant). Rapide, risque faible, fix le plus direct du symptome exact. Fait deja reussir le switch meme sync
  Infinite en cours.
- Livraison 2 ("c'est fluide pendant un sync") : **Chantier 1** (resserrer la fenetre RW via activation V2 apres
  soak) + **Chantier 3** (budget court user-facing) en filet parallele.
- Livraison 3 (durcissement auth) : **Chantiers 4, 5, 6** (sante proactive + backoff par-token, borne concurrence
  — absorbee par V2 —, serialisation XBL).
- Livraison 4 (optionnel) : Chantier 7.

## 7. Verification empirique

- Reproduire d'abord : lancer un sync (Infinite, delta gros) et pendant le sync faire un switch H5 + charger
  /pages/home. Attendu AVANT : GetMatchCount context deadline exceeded, SharedReader unavailable, rollback silencieux.
- Compteurs Phase 0 (deja livres, /debug/vars + carte admin "Contention DB") : mesurer AVANT/APRES
  shared_provider_rw_window_ms (avg/max), _blocked_window_ms, _reader_stall_ns_total, _readers_delayed_total,
  _swap_failures_total. Critere Chantier 1 : rw_window_max_ms de ~dizaines de s a < 1500ms. Ajouter une garde de test.
- Test de charge local : boucler des GET /players/{slug}/pages/home (~20 req/s) PENDANT un cycle auto-sync.
  Attendu APRES : p99 home < ~2-3s, zero 500 home_page_error, au pire des 503 Retry-After brefs.
- Switch de titre en boucle pendant un sync Infinite continu : POST session + GET /bootstrap reussissent 100%
  (plus de rollback). Verifier isTitleSwitching se resout sans passer par le catch (appShellStore.ts:191).
- Auth sous burst : logs/*.log — plus de flood "Acquire failed" / "aucun slot sain", plus de thrash 1s, plus de
  429 XBL currentRequests. Un 429 isole ne marque plus que 1 slot (Chantier 4).
- Regression ART : no_art_patterns_test.go reste vert ; aucun nouveau ON CONFLICT DO UPDATE.

## 8. Honnetete sur les incertitudes

- Axe 4 (bootstrap) est en partie DEJA mitige (timeout 5s + catch -> profile_ready_no_sync ; routing /setup gate
  par players==0 pas par setup_state). La degradation est "largement benigne" sur instance configuree. MAIS le
  couplage bootRepo->mauvais titre est un vrai bug (cause directe du rollback du switch), pas un faux positif.
- Axe 5 (timeouts HTTP) n'est PAS une cible autonome : le budget 31s vient entierement du readyTimeout=30s du
  provider. Son fix "reordonner acquireSharedWriter" = le meme que Chantier 1. Seul ajout propre = budget court
  user-facing (Chantier 3).
- Activation V2 (Chantier 1) : gate historiquement OFF. Le soak multi-joueurs reel est le vrai risque — parity
  V1/V2 a re-valider sur dataset heterogene PVP+PVE avant de retirer le fallback. Si ecart de parity, retomber sur
  la variante V1 manuelle (deplacer acquireSharedWriter dans la phase insert), plus risquee mais localisee.
- Multiplier les mini-swaps par batch (Chantier 1) augmente le NOMBRE de swaps ; valider que le cout fixe
  drain+close+reopen (mesure sub-second) reste < au gain. C'est ce que les compteurs Phase 0 permettent de trancher.

## Note de perimetre

Ce plan est DISTINCT de PLAN_TITLE_SWITCH_FILTER_LEAK.md (fuite de filtre localStorage au switch + comptes
auth_only synchronises a tort). Les deux se completent :
- PLAN_TITLE_SWITCH_FILTER_LEAK = quand le switch REUSSIT mais laisse un etat incoherent (filtre Infinite sur H5).
- CE PLAN = quand le switch ECHOUE silencieusement (bootstrap timeout) + lenteur generale.
Ordre suggere : Chantier 2 d'ici (rend le switch fiable) est le plus prioritaire, puis le filter-leak, puis le
fond (Chantier 1). A arbitrer avec l'utilisateur.
