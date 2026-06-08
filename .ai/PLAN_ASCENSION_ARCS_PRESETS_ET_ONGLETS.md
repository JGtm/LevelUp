# Plan — Gestion des arcs Prestige : presets + suppression + split en 2 onglets

> Statut : PLAN VALIDÉ (décisions tranchées 2026-06-08, voir § Décisions tranchées). Auteur : agent IA. Date : 2026-06-07.
> Branche : **`fix/enrichment-convergence` (branche courante)** — décision utilisateur de rester sur la branche en cours, en commits séquentiels (≠ suggestion initiale `feat/prestige-arc-management`).

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

## Faits vérifiés (2026-06-08) — corrections au plan initial

Exploration de validation effectuée avant implémentation. Le plan est globalement exact ; corrections :

1. **Impl DuckDB localisée** : `PrestigeArcRepo` (table `arc`) + `PrestigeChallengeRepo` (table `challenge`) sont dans [prestige_player_repo.go](apps/go-api/internal/platform/duckdb/prestige_player_repo.go) (`PrestigeArcRepo` ~L166, `PrestigeChallengeRepo` ~L21). Pattern d'accès : `ctx, cancel := context.WithTimeout(ctx, 5*time.Second)` puis `r.db.Exec/QueryRow`. C'est le pattern à suivre pour `Delete`/`DetachFromArc`. Presets : `PrestigePresetArcRepo` dans [prestige_metadata_repo.go](apps/go-api/internal/platform/duckdb/prestige_metadata_repo.go) (tables `preset_arc` + `preset_arc_step`).
2. **`MyArcsSection` n'est PAS un fichier séparé** : il est défini *inline* dans [AscensionProfileTab.tsx](apps/web/src/features/ascension/AscensionProfileTab.tsx) (~L293-362), tout comme `MyObjectivesSection` (~L127-192). `CreateArcForm`/`ArcSummary`/`CreateChallengeForm` vivent dans `features/prestige/components/`. → Lot A.4 (bouton supprimer) édite l'inline ; Lot C (split) extrait ces blocs inline vers le nouveau composant.
3. **`ChallengeFilter.ArcID`** porte le commentaire `"" interdit (utilise NoArc à la place)` → une sémantique « sans arc » existe déjà. À regarder avant d'écrire `DetachFromArc` (mettre `arc_id = NULL` et vérifier la cohérence lecture avec le concept `NoArc`).
4. **Lot B.1 imprécis** : [arc_composer.go](apps/go-api/internal/progression/coach_advisor/arc_composer.go) compose des arcs depuis des **signaux** (dynamique, `TryCompose`/`ArcSpec`), PAS depuis `PresetArc`/`PresetArcStep`. La logique d'adoption *preset* (create arc + N challenges depuis `PresetArcStep`) est à relocaliser (réf. `service_generate.go:411`) — le helper partagé `composeArcFromPreset` reste à extraire/écrire au Lot B, mais pas depuis `arc_composer.go`.
5. **`ChallengeFilter` n'a pas de champ `Metric`** ([repository.go:31](apps/go-api/internal/prestige/repository.go#L31)) : pour le cooldown (Lot D) il faut soit ajouter un champ `Metric *string`, soit filtrer en mémoire les terminaux sur la même métrique.
6. **`api.delete<T>()` existe déjà** côté client ([client.ts:157](apps/web/src/lib/api/client.ts#L157)) — pas besoin de l'ajouter (contrairement à la note du Lot A.4).

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

## Lot D — Activer le cooldown anti-farming + retour UI

**Priorité : haute (bug : le cooldown est défini mais jamais appliqué). Effort : moyen.**

### Constat
Le cooldown post-terminaison (abandon/expiration/complétion) est entièrement défini mais **jamais branché** :
- `IsCooldownActive(tuning, previousChallenges, now)` ([lifecycle.go:143](apps/go-api/internal/prestige/lifecycle.go#L143)) et `CooldownEndsAt` ([lifecycle.go:106](apps/go-api/internal/prestige/lifecycle.go#L106)) existent mais ne sont **appelés nulle part** en prod.
- `ErrCooldownActive` ([lifecycle.go:24](apps/go-api/internal/prestige/lifecycle.go#L24)) + mapping HTTP **429 `cooldown_active`** ([handlers/prestige.go:488](apps/go-api/internal/api/handlers/prestige.go#L488)) sont prêts mais jamais déclenchés.
- Durées dans `tuning.go` : `AbandonedHours: 48`, `ExpiredHours: 12`, `CompletedHours: 0` ([tuning.go:189](apps/go-api/internal/prestige/tuning.go#L189)). Cooldown **mode pilote uniquement** (`CooldownEndsAt` retourne zéro si `Mode != ModePilote`, [lifecycle.go:110](apps/go-api/internal/prestige/lifecycle.go#L110)) — cohérent avec les quotas.

### D.1 — Backend : brancher le check + abaisser la durée
- **Abaisser `AbandonedHours: 48 → 24`** dans `DefaultTuning()` (et la `tuning.toml` si surchargée). (Décision tranchée.)
- `CreateChallenge` ([service.go:203](apps/go-api/internal/prestige/service.go#L203)) : après `validateCreateRequest` et avant/autour de `checkQuotas` (~L209, pilote-only), insérer :
  1. Lister les défis **terminaux** du joueur **sur la même métrique** : `Challenges.List(ctx, ChallengeFilter{UserID, TitleSlug, ...})`. **Pré-requis** : ajouter `Metric *string` à `ChallengeFilter` ([repository.go:31](apps/go-api/internal/prestige/repository.go#L31)) + l'honorer dans `PrestigeChallengeRepo.List` (SQL `AND metric = ?`), OU filtrer en mémoire. Préférer le champ `Metric` (réutilisable par l'enrichissement D.2).
  2. `if IsCooldownActive(tuning, prev, now) { return ErrCooldownActive }`.
- Garde-rail : ne s'applique qu'au mode pilote (le mode libre n'a pas de cooldown — ne pas bloquer la création libre).
- Tests : `service_full_test.go` — `TestCreateChallenge_CooldownBlocks` (pilote, métrique en cooldown → `ErrCooldownActive`), `TestCreateChallenge_CooldownExpired_OK`, `TestCreateChallenge_FreeMode_NoCooldown`.

### D.2 — Backend : enrichir SuggestTemplates avec l'état cooldown (UI proactive)
- Ajouter à `Template` ([types.go:141](apps/go-api/internal/prestige/types.go#L141)) un champ **non persisté** `CooldownEndsAt *time.Time` (enrichi à la demande, comme `Challenge.CurrentValue`).
- Dans `SuggestTemplates` ([service.go](apps/go-api/internal/prestige/service.go)) : pour chaque template suggéré, calculer `CooldownEndsAt` via les défis terminaux du joueur sur la métrique du template (réutilise la requête D.1 + `CooldownEndsAt`). Renseigné uniquement en mode pilote.
- Tests : `TestSuggestTemplates_AnnotatesCooldown`.

### D.3 — Frontend : retour UI (picker proactif + form réactif)
- **Proactif (le « bon endroit »)** : dans le sélecteur de template de [CreateChallengeForm.tsx](apps/web/src/features/prestige/components/CreateChallengeForm.tsx) (modes hybride/automatique), si `template.cooldownEndsAt` est dans le futur → badge « Disponible dans Xh » + option **désactivée** (non sélectionnable). Helper de formatage du delta (réutiliser un util de durée existant, FR/EN).
- **Réactif** : sur erreur `cooldown_active` (429) du `useCreateChallenge`, afficher un message clair dans le `SubmitRow` (« Métrique en cooldown, disponible dans Xh ») plutôt que le message brut serveur. La gouttière d'erreur inline existe déjà ([CreateChallengeForm.tsx:403](apps/web/src/features/prestige/components/CreateChallengeForm.tsx#L403)).
- Côté type front : ajouter `cooldownEndsAt?: string` au type `Template` (lib/prestige.ts) + mapping.
- i18n (`prestige/i18n.ts`, FR sans franglais + EN) : `cooldownBadge` (interpolation Xh/Xj), `cooldownErrorTitle`, `cooldownAvailableIn`.
- Tests : test du badge (template en cooldown → désactivé + label) ; test du chemin 429 (message friendly).

### D.4 — Note de cohérence avec Lot A
La règle d'exemption « arc récent < 1h » (cf. Décisions tranchées) s'appuie sur ce cooldown désormais vivant : à la suppression d'un arc créé il y a < 1h, les objectifs sont **hard-deletés** (donc aucun enregistrement terminal → aucun cooldown) ; ≥ 1h → abandon normal → cooldown 24h s'applique. → Lot A dépend de Lot D (ordre D avant A).

---

## Décisions tranchées (2026-06-08)

> Validées avec l'utilisateur en questions interactives. Remplacent les « décisions ouvertes » du plan initial.

1. **Périmètre & branche** : les **3 lots + Lot D**, tous sur la **branche courante `fix/enrichment-convergence`** en commits séquentiels (pas de nouvelle branche). Ordre : **C → D → A → B**.
2. **« Supprimer les objectifs » (cascade Lot A)** = **abandon (soft)** par défaut (réutilise le lifecycle, garde la télémétrie), SAUF cas d'exemption ci-dessous.
3. **Cooldown** : **le brancher** (Lot D — c'est un bug qu'il soit dormant) **ET abaisser 48h → 24h**.
4. **Exemption à la suppression d'arc** : **arc créé il y a < 1h → exempt** (objectifs hard-deletés, zéro cooldown) ; **≥ 1h → abandon + cooldown 24h**. Signal : `arc.CreatedAt` (pas de calcul de progression nécessaire).
5. **Retour UI cooldown** : **proactif** (badge « dispo dans Xh » + option désactivée dans le sélecteur de template, via `Template.cooldownEndsAt`) **+ réactif** (message clair sur le 429).
6. **Label du nouvel onglet (Lot C)** : **« Entraînement »** (FR). Alternative gardée en réserve : « Académie » (résonance Halo *Academy*) si préférence in-universe.

### Décisions restées hors-scope (dette, non traitée ici)
- **i18n NavL1** : labels L2 FR-only hardcodés — on garde FR pour l'instant (i18n-isation de la nav = hors-scope).
- **Multi-titres** : `TITLE_SLUG='halo_infinite'` hardcodé ([AscensionProfileTab.tsx:42](apps/web/src/features/ascension/AscensionProfileTab.tsx#L42)) — dette existante ; **ne pas la propager** dans les nouveaux composants (passer le slug en prop).

## Ordre recommandé

`Lot C (split, léger, débloque la place)` → `Lot D (brancher cooldown + UI)` → `Lot A (suppression d'arc, s'appuie sur D pour l'exemption)` → `Lot B (presets, le plus lourd)`.
Chaque lot = un ou plusieurs commits sur `fix/enrichment-convergence` ; **thought_log à chaque lot**.

## Checklist livraison (rappel delivery-checklist)
- [ ] `go test ./... && go vet ./...` verts (nouveaux tests repo/service/handler)
- [ ] `tsc` + `eslint` + `vitest` verts ; query keys dans `lib/query/keys.ts` ou `arcKeys`/`presetKeys`
- [ ] Strings FR (sans franglais) **et** EN dans `prestige/i18n.ts`
- [ ] Pas de couleur hex/Tailwind en dur ; ownership vérifié sur Delete/Adopt
- [ ] `routeTree.gen.ts` régénéré (non édité main) ; thought_log par lot
