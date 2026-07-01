# Plan — Fuite de filtre au switch de titre + comptes auth_only synchronisés à tort

Statut : PLANIFIE (pas encore implémenté)
Date : 2026-07-01
Branche cible : `fix/title-switch-filter-leak-and-authonly-sync` (depuis `fix/h5-ui-adjustments-batch`)

## Contexte

L'utilisateur signale que le switch de titre (Halo Infinite <-> Halo 5) "bug toujours",
et remarque que l'URL Infinite porte un paramètre `?f=` (filtre encodé en base64) absent
côté H5. Demande aussi d'examiner `logs/` à la racine (terminal incomplet).

Investigation menée : 3 agents Explore (front / backend / logs) + 1 agent Plan de
validation + lecture directe du code.

### Constat 1 — Fuite de filtre au switch de titre (CAUSE DU BUG RESSENTI)

`useSoloFilterStore` / `useSquadFilterStore` (Zustand + `persist` localStorage) sont
GLOBAUX, non scopés par titre. `appShellStore.switchTitle()` purge le cache TanStack
Query mais ne touche JAMAIS ces deux stores.

Résultat : `picked_sessions` / `cascade.{modes,maps,playlists}` sélectionnés sur Halo
Infinite survivent en localStorage et sont ré-appliqués tels quels dès qu'une page
consommant ces stores se monte sur Halo 5 — alors que ces labels/IDs n'ont aucun sens
dans le catalogue H5.

Le `?f=` visible dans l'URL Infinite est la CONSEQUENCE visible (chaque mutation du store
réécrit l'URL via `encodeToUrl`), pas la cause : la vraie fuite est le localStorage
partagé entre titres.

Fichiers clés :
- `apps/web/src/stores/createFilterStore.ts` — factory ; `encodeToUrl` L152-162,
  `decodeFromUrl` L164-176 (ne valide que `filter_mode`, pas le contenu),
  `resetFilters` L239-248, `onRehydrateStorage` L367-377.
- `apps/web/src/stores/soloFilterStore.ts` — instance globale `urlEnabled:true urlParam:'f'`.
- `apps/web/src/stores/squadFilterStore.ts` — instance globale `urlEnabled:false`.
- `apps/web/src/stores/appShellStore.ts` L155-198 — `switchTitle()` : purge queryClient
  (L178-179) mais pas les filtres.

### Constat 2 — Comptes auth_only traités comme cibles de sync (BRUIT LOGS/METRIQUES)

5 comptes Halo Infinite (`DankerGlue`, `QuiteSiren`, `Trimbutton`, `GeleJugefi`,
`UppedJoker`) ont `db_path:""` + `auth_only:true` dans `db_profiles.json`. Confirmé par
l'utilisateur : ils n'auront JAMAIS de DB, ils existent uniquement pour fournir des
refresh tokens au pool de sync partagé.

Le filtre canonique `domain.SyncablePlayers()`
(`apps/go-api/internal/domain/bootstrap.go:63-71`) ne filtre que sur `SyncEnabled`, PAS
sur `AuthOnly`. Ces 5 comptes fantômes passent donc dans la liste transmise à
`runOnceV2()` à chaque cycle -> warnings en boucle observés dans les logs :
`sync.v2: joueur en échec ... open player DB DankerGlue`, `snapshot: player DB indisponible`.

IMPORTANT (validé par lecture de `discovery.go` / `known_loader.go`) : pour ces comptes
l'échec se produit AVANT tout appel réseau (l'ouverture DuckDB locale échoue
immédiatement ; `ListUnknownMatches` / le pool ne sont jamais atteints). Ce fix est donc
principalement un fix d'HYGIENE de logs et de métriques dashboard (`failed_count` de
`auto_sync` plus pollué par 5 fantômes) — PAS une preuve de réduction de la pression
pool/429.

### Constat 3 — Backend routing du switch de titre : SAIN

Middleware `TitleExtractor`, endpoint `POST /session/context`, `NewSyncEngineForTitle`
(dette `auto_sync.go:838` déjà résolue) fonctionnent correctement. Aucun bug de
routage/session à corriger.

## Fix 1 — Reset des filtres contextuels au switch de titre

Fichier : `apps/web/src/stores/appShellStore.ts`

Dans `switchTitle()`, insérer les deux `resetFilters()` JUSTE APRES l'étape 2
(`set({ currentTitleSlug: titleSlug })`), AVANT l'étape 3 (`queryClient.cancelQueries()`) :

```ts
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'

// ... dans switchTitle(), après :
setApiTitleSlug(titleSlug)
set({ currentTitleSlug: titleSlug })

// 2bis. Réinitialiser les filtres contextuels (solo/squad) : leur state
// (picked_sessions, cascade modes/maps/playlists) peut référencer des
// labels/IDs spécifiques à l'ANCIEN titre — même catégorie de state que le
// cache TanStack purgé juste après.
useSoloFilterStore.getState().resetFilters()
useSquadFilterStore.getState().resetFilters()

// 3. Annuler les requêtes EN VOL ...
await queryClient.cancelQueries()
```

Pourquoi cet ordre : `resetFilters()` (déjà existant) remet `DEFAULT_FILTER_CONTEXT` et
réécrit immédiatement l'URL/localStorage. Le faire tôt et synchrone évite qu'un F5
pendant la fenêtre [titre changé, filtre pas encore reset] ne relise un `?f=` obsolète.
Pas de risque d'import circulaire (`appShellStore -> soloFilterStore/squadFilterStore ->
createFilterStore`, aucun retour).

Pas besoin de toucher `TitleSwitcher.tsx` (`navigate()` sans `search` ne préserve déjà
pas les params).

Trade-off accepté : si le switch échoue et rollback, les filtres restent reset — dommage
collatéral mineur, à mentionner dans le commit.

Autres stores vérifiés — AUCUN autre n'a ce problème : `useSpartanAppearanceStore` et
`useSettingsDraftStore.lastPlayerSlugByTitle` utilisent déjà un pattern
`byTitle: Record<string, ...>` correct ; `useRelationsPrefsStore`, `useAssetDrawerStore`,
`useFeedbackDrawerStore` sont globaux mais sans donnée métier liée à un titre ;
`useSetupFlowStore` / `useSessionContextStore` ne persistent pas.

Tests :
- `apps/web/src/stores/appShellStore.switchTitle.test.ts` (existant "correction #10") :
  adapter pour vérifier que les deux `resetFilters()` sont appelés, positionnés après
  `setApiTitleSlug` et avant `cancelQueries` (tracking `calls: string[]` déjà en place).
- Nouveau test : polluer les stores (`setCascade`/`setSessions`) avant switch, vérifier
  retour à `DEFAULT_FILTER_CONTEXT` après.
- Nouveau test : no-op si titre déjà courant -> filtres pollués intacts.

## Fix 2 — Exclure les comptes auth_only du filtre canonique de sync

Fichier : `apps/go-api/internal/domain/bootstrap.go` (`SyncablePlayers`, ~L63-71)

```go
func SyncablePlayers(players []PlayerSummary) []PlayerSummary {
	out := make([]PlayerSummary, 0, len(players))
	for _, p := range players {
		if p.SyncEnabled && !p.AuthOnly {
			out = append(out, p)
		}
	}
	return out
}
```

Mettre à jour le docstring (L54-62) : mentionner l'exclusion `AuthOnly` (profils sans DB,
jamais des cibles de sync valides) + lister les 4 chemins bénéficiaires.

Callers bénéficiaires (tous vérifiés sans risque de régression) :
- `internal/scheduler/auto_sync.go:609` — cause directe des logs observés.
- `cmd/server/main.go:1913` — setup watcher daemon.
- `internal/service/friends_orchestrator_service.go:93` — recompute is_with_friends
  (skip `IsDemo` déjà présent L96-98, ce fix étend la même logique).
- `internal/service/fanout_service.go:54` — cibles fan-out (skip `IsDemo` déjà L69).

NE PAS toucher (ces call-sites ont légitimement besoin des comptes auth_only) :
- `internal/platform/auth/pool/discovery.go:89` — scan du pool de credentials (raison
  d'être de ces comptes).
- `internal/worldenrich/wiring.go:196,253,272` — round-robin multi-comptes enrichissement.
- `internal/service/bootstrap_service.go` (`excludeAuthOnly()` L120/L332/L346-357) —
  filtre SEPARE et déjà correct pour les listes front-facing (sélecteur joueur, favoris).
  Deux filtres à rôles distincts, ne pas fusionner.

Tests : `apps/go-api/internal/domain/syncable_players_test.go`
- `TestSyncablePlayers_ExcludesAuthOnly` (RealPlayer gardé, DankerGlue exclu).
- `TestSyncablePlayers_ExcludesAuthOnly_EvenIfSyncEnabled` (auth_only + pause -> exclu).

## Hors scope (chantier séparé)

Le problème de perf backend plus large révélé par les logs (`pool: cooldown global
déclenché`, `PolicyAnyPublic` à sec, `duckdb.OpenReadOnly` "chemin introuvable" en pleine
phase `post_sync`, timeout 31s sur `GetHomePage` de JGtm par contention du shared reader)
est INDEPENDANT du switch de titre et plus risqué (dimensionnement pool 7 tokens, logique
cooldown global, fenêtre RW du SharedDBProvider pendant sync.v2). Fix 2 réduit un peu le
travail inutile mais ne le résout pas. A proposer comme chantier séparé si le
ralentissement persiste.

## Branche + commits

Nouvelle branche `fix/title-switch-filter-leak-and-authonly-sync` depuis
`fix/h5-ui-adjustments-batch`. 2 commits :
1. `fix(sync): exclure les profils auth_only du filtre SyncablePlayers`
2. `fix(shell): reset des filtres solo/squad au switch de titre`

## Vérification

Backend :
```
cd apps/go-api
go test ./internal/domain/... ./internal/scheduler/... ./internal/service/... -v
go build ./...
```
Puis en dev local, déclencher un cycle de sync et vérifier dans `logs/general.log` /
`logs/scheduler.log` que les 5 gamertags auth_only n'apparaissent plus dans
`sync.v2: joueur en échec` / `snapshot: player DB indisponible`.

Frontend :
```
cd apps/web
npm run typecheck
npm run test -- appShellStore
```
Puis test manuel : sur Halo Infinite appliquer un filtre de session ; switcher vers
Halo 5 ; naviguer vers la page équivalente H5 et confirmer que le filtre est revenu à
l'état par défaut ; vérifier `localStorage['levelup-solo-filter-v1']` dans les devtools.

Obligatoire avant de rendre la main : entrée `.ai/thought_log.md` (date, statut, décision,
résultats, prochaine étape), en notant que Fix 2 est un fix d'hygiène/observabilité et non
une preuve de résolution du ralentissement pool/429.
