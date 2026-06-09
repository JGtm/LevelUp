# Plan — Auto-drain du catalogue d'assets

> Statut : proposé (2026-06-09). Non implémenté. Fait suite au fix livré
> `fix(sync): construit pair_name "{mode} on {map}" quand le catalogue manque`
> (commit 5a30af116) qui couvre l'AFFICHAGE ; ce plan apporte les noms
> CANONIQUES + la bonne catégorie de filtre.

## Objectif & critère de succès

Brancher `CatalogFetcherService.Drain()` automatiquement pour que les noms
canoniques des nouveaux assets (pair / map / variant / playlist) soient
récupérés sans CLI manuel.

Succès : un nouveau contenu Halo synchronisé voit son `pair_name` résolu en nom
canonique (avec préfixe `Arena:` / `Community:` -> bonne catégorie de filtre)
sous N heures, serveur allumé, sans intervention.

## Contexte

- La sync enfile déjà les assets inconnus dans `catalog_fetch_queue`
  (`internal/sync/catalog_enqueue.go`). OK.
- `CatalogFetcherService.Drain()` (`internal/service/catalog_fetcher_service.go`)
  n'est appelé que par les CLI `populate-*`. Aucun scheduler / boot / post-sync.
  La file n'est donc jamais vidée automatiquement -> noms en GUID jusqu'au drain
  manuel ("drain mensuel via CLI", design intentionnel d'origine).
- Le fallback de construction (déjà livré) couvre l'affichage. L'auto-drain est
  un enrichissement de qualité (nom canonique + catégorie), PAS un correctif
  bloquant -> différable sereinement.

## Approche recommandée : cron périodique in-process (option B)

Précédent quasi identique : `scheduler/world_leaderboard_cron.go` (réseau +
écriture DB, serveur allumé, garde de fraîcheur anti hot-reload Air). On le clone.

| Option | Verdict |
|---|---|
| A. Hook post-sync | Rejeté : latence HTTP + écriture metadata sur le hot-path sync |
| B. Cron périodique | RECOMMANDÉ : découplé, borné, clone d'un pattern éprouvé en prod |
| C. Boot-only | Rejeté : rate le contenu entre deux redémarrages |

### Design `CatalogDrainCron` (miroir de `WorldLeaderboardCron`)

- `NewCatalogDrainCron(metaDB *sql.DB, fetcher *service.CatalogFetcherService, interval time.Duration)`.
- `Run(ctx)` : un `RunOnce` immédiat + boucle `ticker` (intervalle configurable).
- `RunOnce(ctx)` :
  - garde "queue vide -> skip" (COUNT sur `catalog_fetch_queue`) ;
  - garde de fraîcheur (dernier drain récent -> skip, anti hot-reload Air) ;
  - sinon `fetcher.Drain(ctx, titleSlug)` ; log `DrainResult`.
- `RunOnce` exporté -> endpoint admin optionnel `_admin/catalog/drain` (forcer).
- Démarrage dans `cmd/server/main.go` à côté de `worldLbCron` (~L968), en
  goroutine, arrêt propre via `context` (cf. note shutdown ~L819).
- `metaDB` (RW, ~L487) et un Discovery/Resolver (~L749-752) sont déjà construits
  dans le main -> wiring in-process trivial, zéro lock cross-process
  (contrairement au CLI qui exige le serveur stoppé).

## Bénéfices

1. Catégorie de filtre correcte : le nom canonique porte le préfixe (`Arena:`)
   -> `InferModeCategoryFromPairName` classe bien (la construction donne "Other").
2. Toutes les surfaces bénéficient du vrai nom localisé (asset_translations à
   jour), pas seulement celles qui normalisent.
3. Zéro CLI manuel -> système auto-réparant (directive sync convergent autonome).
4. Idempotent, best-effort, sur un pattern déjà validé.

## Inconvénients / risques

1. Tokens (principal) : `Drain` fetch via l'API Discovery -> 403 si tokens
   périmés. Conséquence : drain échoue, noms non récupérés -- mais le fallback de
   construction couvre l'affichage entre-temps. Mitigations : backoff +
   `attempts` / `markError` (déjà dans `CatalogFetcherService`), log d'alerte sur
   taux d'échec.
2. Charge API : un gros backlog = beaucoup d'appels. Mitigation : borne
   d'entrées par cycle + intervalle large.
3. Écriture metadata périodique : in-process via le handle RW du serveur (pas de
   lock externe), mais concurrente aux lectures handlers -> fenêtre courte
   (upsert par asset, rapide).
4. Complexité : +1 cron, +tests, +ordre de shutdown.

## Impacts

- Perf : négligeable (basse fréquence, hors hot-path).
- DB : écritures metadata (`map_mode_pair_definitions`, `asset_translations`,
  `pair_mode_label_translations`, `playlists_catalog`, etc.).
- Multi-titre : itérer les titres via `HasCapability`, jamais
  `slug == "halo_infinite"`. Un seul titre aujourd'hui.
- Observabilité : log `DrainResult` (pairs / maps / variants / playlists +
  erreurs). Un taux d'erreur élevé = signal tokens périmés (lien ADR 0023).
- Architecture : `scheduler/` = orchestration, réutilise `CatalogFetcherService`
  (service/). Pas de nouveau `analysis` / `domain` / repo.

## Phases (livrables indépendants)

1. `CatalogDrainCron` + tests (mock fetcher : queue vide / non-vide, garde
   fraîcheur, erreur best-effort non-fatale).
2. Wiring `cmd/server/main.go` (goroutine `Run` + shutdown propre).
3. (option) endpoint admin `_admin/catalog/drain` loopback (forcer un cycle).
4. thought_log (+ ADR si on fige l'intervalle).

## Done definition

Nouveau contenu synchronisé -> après <= 1 cycle, `pair_name` canonique en base,
catégorie correcte, sans CLI ; drain best-effort (jamais de 500 / panic) ; logs
exploitables.

## Décisions ouvertes

1. Intervalle : 6 h (réactif) ou 24 h (poli, comme le world leaderboard) ?
2. Endpoint admin de drain à la demande : oui / non ?
3. Borne par cycle (ex. max 200 assets / drain) pour lisser la charge API ?
4. Tokens périmés : juste logguer, ou émettre une notif `data_health_warning`
   sur 403 répétés ?
