# Plan — Gestion des arcs Prestige : presets + suppression + split en 2 onglets

> Statut : PLAN (à valider avant implémentation). Auteur : agent IA. Date : 2026-06-07.
> Branche cible suggérée : `feat/prestige-arc-management` (depuis `main`).

## Contexte et état actuel

La page Ascension (route `/players/$playerSlug/ascension`, composant
[AscensionProfileTab](apps/web/src/features/ascension/AscensionProfileTab.tsx)) regroupe aujourd'hui
**deux couches** dans un seul onglet « Profil & objectifs » :

1. **Couche Prestige** : `MyObjectivesSection` (objectifs) + `MyArcsSection` (arcs).
2. **Couche Ascension/Coaching** : `CoachProposalsCard` + `CampaignTracker` + `PlayerProfileV3` + `PatternsSection` (+ `StartCampaignModal`).

État de la fonctionnalité « arcs » après le petit fix livré ce jour :

| Capacité | Backend | Front data | Front UI |
|---|---|---|---|
| Créer un arc **libre** | OK (`POST /arcs` → `service.CreateArc`) | `useCreateArc` | **LIVRÉ** (`CreateArcForm` + bouton « + Nouvel arc » dans `MyArcsSection`) |
| Lister ses arcs | OK (`GET /arcs`) | `useArcs` | OK (`ArcSummary`) |
| **Adopter un arc preset** | Partiel : `PresetArcRepo` lisible (ListByTitle/GetByID/GetSteps) **mais aucune route HTTP**, logique d'adoption seulement dans `coach_advisor/arc_composer.go` | Aucun | **ABSENT** |
| **Supprimer / annuler un arc** | **ABSENT** : `ArcRepo` n'a que Create/Get/ListByUser/MarkCompleted ; `ChallengeRepo` n'a pas de Delete ni de détachement `arc_id` | Aucun | **ABSENT** |

Faits backend de référence (vérifiés) :
- `prestige.ArcRepo` : `Create / Get / ListByUser / MarkCompleted` — pas de `Delete`. ([repository.go:43](apps/go-api/internal/prestige/repository.go#L43))
- `prestige.ChallengeRepo` : pas de `Delete` ; les défis sont **abandonnés** (status `abandoned` + cooldown 48 h) via `service.AbandonChallenge` → `MarkAbandoned`. ([service.go:432](apps/go-api/internal/prestige/service.go#L432))
- `ChallengeFilter.ArcID` permet de lister les défis d'un arc. ([repository.go:35](apps/go-api/internal/prestige/repository.go#L35))
- `prestige.Arc` n'a **pas** de champ `Status` — uniquement `CompletedAt` (lifecycle binaire). ([types.go:67](apps/go-api/internal/prestige/types.go#L67))
- Adoption d'un preset (create arc + N challenges) déjà implémentée pour le coach : [arc_composer.go](apps/go-api/internal/progression/coach_advisor/arc_composer.go) + [service_generate.go:411](apps/go-api/internal/progression/coach_advisor/service_generate.go#L411).
- Navigation L1/L2 : [NavL1.tsx](apps/web/src/components/shell/NavL1.tsx) — `L1_SECTIONS.ascension.tabs` contient `profile` (« Profil & objectifs ») et `realisations` (« Réalisations »). ([NavL1.tsx:115](apps/web/src/components/shell/NavL1.tsx#L115))

## Objectif et critères de succès

1. Un joueur peut **adopter un arc preset** depuis l'UI (parcourir le catalogue, prévisualiser les étapes, adopter → l'arc + ses objectifs sont créés).
2. Un joueur peut **supprimer / annuler un arc**, avec un choix explicite : **supprimer aussi les objectifs attachés** OU **garder les objectifs** (les détacher de l'arc → ils redeviennent des objectifs libres).
3. La page Ascension est **scindée en 2 onglets** : « Profil & objectifs » (couche Prestige seule) et un nouvel onglet « Coaching » (couche Ascension/Coaching), tous deux dans la dropdown L2 sous l'entrée L1 « Ascension ».

Critère global : `go test ./...` + `go vet ./...` verts ; `tsc` + `eslint` + `vitest` verts ; ownership respecté (un joueur ne supprime/adopte que sur ses propres données) ; FR sans franglais + EN.

---

## Lot A — Suppression / annulation d'arc (avec ou sans objectifs)

**Priorité : haute (demande explicite). Effort : moyen.**

### A.1 — Backend : couche persistance (`internal/prestige` + `internal/platform/duckdb`)
- `ArcRepo` : ajouter `Delete(ctx, id string) error` (DELETE hard sur la table `arc`, stats.duckdb par joueur — un arc est un conteneur léger sans sémantique PP/cooldown, le hard delete est justifié).
- `ChallengeRepo` : ajouter `DetachFromArc(ctx, arcID string) error` (UPDATE `challenge SET arc_id = NULL WHERE arc_id = ?`) pour l'option « sans les objectifs ».
- Implémenter les deux dans la couche DuckDB. **TODO impl** : localiser le fichier d'impl de `ArcRepo`/`ChallengeRepo` (le glob `*arc*.go` ne matche pas → impl probablement dans un `prestige_*_repo.go` regroupé) avant d'écrire le SQL. Écritures sur **player DB** : respecter le pattern d'accès DuckDB du module (pas d'`ON CONFLICT` hasardeux ; cf. [[reference_legacy_player_db_no_constraints]]).

### A.2 — Backend : service (`internal/prestige/service_arcs_squads.go`)
- Ajouter à l'interface `Service` ([service.go:50](apps/go-api/internal/prestige/service.go#L50)) : `DeleteArc(ctx, id string, opts DeleteArcOptions) error`.
- `DeleteArcOptions{ CascadeObjectives bool }` (ou enum `objectives: "delete" | "detach"`).
- Logique :
  1. `ArcRepo.Get(id)` → **vérif ownership** (`arc.UserID == callerUserID`) — sinon `ErrForbidden`. Le caller fournit le userID (depuis la session, comme les autres handlers prestige).
  2. Lister les défis de l'arc : `ChallengeRepo.List(ChallengeFilter{ArcID: &id})`.
  3. Si `CascadeObjectives` : pour chaque défi actif, `AbandonChallenge` (réutilise le lifecycle existant : status→abandoned, télémétrie, cooldown). **Décision** : « supprimer les objectifs » = **abandonner** (soft), pas hard-delete — préserve la télémétrie + le cooldown anti-farming. À confirmer (cf. Décisions ouvertes).
  4. Sinon : `ChallengeRepo.DetachFromArc(id)` (les défis perdent `arc_id`, redeviennent libres).
  5. `ArcRepo.Delete(id)`.
  6. `slog.InfoContext(ctx, "prestige: arc deleted", "arc_id", id, "user_id", userID, "cascade", opts.CascadeObjectives)`.
- Câbler dans `LazyPrestigeService` ([prestige_lazy_service.go](apps/go-api/internal/api/prestige_lazy_service.go)) (delegate `resolveByUserID` → `svc.DeleteArc`).

### A.3 — Backend : handler + route
- [handlers/prestige.go](apps/go-api/internal/api/handlers/prestige.go) : ajouter `DeleteArc(w, r)` — parse `{id}` (path) + `cascade`/`objectives` (query), appelle `svc.DeleteArc`, renvoie 204. 400 si param invalide, 403 si ownership KO, 404 si arc inconnu.
- [server.go:1010](apps/go-api/internal/api/server.go#L1010) : `r.Delete("/arcs/{id}", ph.DeleteArc)`.

### A.4 — Frontend (data + UI)
- [lib/prestige.ts](apps/web/src/lib/prestige.ts) : `deleteArc: (id, cascade) => api.delete('/arcs/${id}?objectives=' + (cascade ? 'delete' : 'detach'))` (ajouter `api.delete` si absent du client).
- `hooks/useArcs.ts` : `useDeleteArc(userId, titleSlug)` (mutation, `onSuccess` → invalider `arcKeys.list` **et** les challenges).
- UI : bouton supprimer sur l'élément d'arc (dans `MyArcsSection`, ou une action sur `ArcSummary`). Ouvre une **confirmation à 2 options** (modale ou `AskUserQuestion`-like) :
  - « Supprimer aussi les N objectifs » (cascade=true)
  - « Garder les objectifs (les détacher) » (cascade=false)
  - + Annuler.
- i18n : clés dict `prestige/i18n.ts` (FR sans franglais + EN) : `arcDeleteTitle`, `arcDeleteWithObjectives`, `arcDeleteKeepObjectives`, `arcDeleteConfirm`, interpolation du compte N.

### A.5 — Tests
- `service_full_test.go` : `TestService_DeleteArc_Cascade` (les défis passent abandoned + arc supprimé) ; `TestService_DeleteArc_Detach` (défis gardés, arc_id vidé, arc supprimé) ; `TestService_DeleteArc_Forbidden` (userID ≠ owner).
- `handlers/prestige_test.go` : httptest DELETE 204 / 403 / 404 / 400.
- `platform/duckdb` : test `:memory:` de `ArcRepo.Delete` + `ChallengeRepo.DetachFromArc`.
- Frontend : test du dialogue (les 2 chemins appellent `deleteArc` avec le bon flag).

---

## Lot B — Picker de presets d'arc (parcourir + adopter)

**Priorité : moyenne. Effort : moyen-lourd.**

### B.1 — Backend : service
- Interface `Service` : ajouter `ListArcPresets(ctx, titleSlug string) ([]PresetArc, error)` (hydrate les `Steps` via `PresetArcRepo.GetSteps`) et `AdoptPresetArc(ctx, userID, titleSlug, presetID string) (Arc, error)`.
- `AdoptPresetArc` : create arc (depuis le preset : title/description/IsPreset=true/PresetID) + N `CreateChallenge` (depuis `PresetArcStep` : TemplateID + TargetTier). **Réutiliser** la logique de [arc_composer.go](apps/go-api/internal/progression/coach_advisor/arc_composer.go) — extraire un helper partagé (`prestige.composeArcFromPreset`) plutôt que dupliquer. Vérif ownership (userID = caller).

### B.2 — Backend : handlers + routes
- `GET /arcs/presets?title_slug=...` → `ListArcPresets` (catalogue + steps pour preview).
- `POST /arcs/presets/{id}/adopt` (body `{user_id, title_slug}`) → `AdoptPresetArc` → 201 `Arc`.
- Ajouter les routes dans `server.go`.

### B.3 — Frontend
- `lib/prestige.ts` : `listArcPresets(titleSlug)` + `adoptArcPreset(id, body)`.
- `hooks/useArcs.ts` : `useArcPresets(titleSlug)` (query, cache long — catalogue versionné) + `useAdoptArcPreset(userId, titleSlug)` (mutation → invalider arcs + challenges).
- UI : composant `ArcPresetPicker` (modale ou section dépliable dans `MyArcsSection`) : liste des presets (titre, description, aperçu des étapes : N objectifs + paliers) + bouton « Adopter ». À la sélection → `adoptArcPreset` → ferme + l'arc apparaît dans la liste.
- Restaurer dans l'empty-state de `MyArcsSection` l'option preset (retirée au petit fix) une fois le picker livré : « Aucun arc en cours. Adopte un preset ou crée le tien. » + 2 boutons.
- i18n : clés dict (FR/EN).

### B.4 — Tests
- `service_full_test.go` : `TestService_AdoptPresetArc_OK` (arc + N challenges créés, ownership) ; `TestService_ListArcPresets`.
- `handlers/prestige_test.go` : httptest GET presets + POST adopt.
- Frontend : test `ArcPresetPicker` (adoption appelle le hook ; rendu de la preview des étapes).

---

## Lot C — Split en 2 onglets (« Profil & objectifs » / « Coaching ») + L1

**Priorité : moyenne (motivée par la longueur de la page après A+B). Effort : léger (refactor front + nav).**

### C.1 — Extraction du composant Coaching
- Créer `features/ascension/AscensionCoachingTab.tsx` : déplacer la `LayerSection` « Ascension — Coaching d'amélioration » (CoachProposalsCard + CampaignTracker + PlayerProfileV3 + PatternsSection + StartCampaignModal + l'état `campaignModal` + `openStartCampaign` + `useActiveCampaign` + `usePatterns`).
- `AscensionProfileTab` ne garde que la `LayerSection` Prestige (`MyObjectivesSection` + `MyArcsSection`). Retirer les imports/états devenus inutiles.

### C.2 — Route
- Nouvelle route `routes/players/$playerSlug/ascension/coaching.tsx` (`createFileRoute` → `AscensionCoachingTab`), calquée sur [ascension/index.tsx](apps/web/src/routes/players/$playerSlug/ascension/index.tsx). La `routeTree.gen.ts` se régénère (ne pas éditer à la main).

### C.3 — Navigation L1/L2
- [NavL1.tsx:115](apps/web/src/components/shell/NavL1.tsx#L115) : ajouter dans `ascension.tabs` une entrée `{ key: 'coaching', label: 'Coaching', path: '/players/$playerSlug/ascension/coaching' }` (entre `profile` et `realisations`). L'ajout dans `tabs[]` suffit à l'afficher dans la dropdown L2. `matchPathname` matche déjà `/ascension/`.
- Label : « Coaching » (l'entrée L1 est déjà « Ascension » ; le titre complet « Ascension — Coaching d'amélioration » reste le titre de section interne). À confirmer.

### C.4 — Tests
- `NavL1.test.tsx` : asserter la présence du 3e onglet.
- `AscensionProfileTab.test.tsx` : retirer les assertions qui vérifient le coaching (coach-proposals, player-profile-v3) — ne doivent plus être sur cet onglet.
- Nouveau `AscensionCoachingTab.test.tsx` : asserte coach + profil + patterns.

---

## Décisions ouvertes (à trancher avant ou pendant l'implémentation)

1. **« Supprimer les objectifs » = abandon (soft) ou hard delete ?** Recommandation : **abandon** (réutilise le lifecycle, garde télémétrie + cooldown 48 h). Alternative : ajouter `ChallengeRepo.DeleteByArc` pour un vrai delete si on veut zéro trace.
2. **Cooldown sur cascade** : abandonner N objectifs d'un coup déclenche N cooldowns 48 h sur leurs métriques. OK ? (Sinon : exempter la suppression d'arc du cooldown via un flag.)
3. **Label du nouvel onglet** : « Coaching » (court) vs « Coaching d'amélioration ».
4. **i18n NavL1** : les labels L2 actuels sont **FR-only** (hardcodés). On garde FR ou on i18n-ise la nav à cette occasion ? (hors-scope suggéré.)
5. **Multi-titres** : `TITLE_SLUG='halo_infinite'` est hardcodé côté front (`AscensionProfileTab`) — dette existante, non introduite ici ; à ne pas propager dans les nouveaux composants (passer le slug en prop).
6. **Branche** : `feat/prestige-arc-management` depuis `main` (lot conséquent, ≠ petit fix livré sur `fix/enrichment-convergence`).

## Ordre recommandé

`Lot C (split, léger, débloque la place)` → `Lot A (suppression, demande explicite)` → `Lot B (presets, le plus lourd)`.
Chaque lot est **livrable indépendamment** (1 branche, N commits ; thought_log à chaque lot).

## Checklist livraison (rappel delivery-checklist)
- [ ] `go test ./... && go vet ./...` verts (nouveaux tests repo/service/handler)
- [ ] `tsc` + `eslint` + `vitest` verts ; query keys dans `lib/query/keys.ts` ou `arcKeys`/`presetKeys`
- [ ] Strings FR (sans franglais) **et** EN dans `prestige/i18n.ts`
- [ ] Pas de couleur hex/Tailwind en dur ; ownership vérifié sur Delete/Adopt
- [ ] `routeTree.gen.ts` régénéré (non édité main) ; thought_log par lot
