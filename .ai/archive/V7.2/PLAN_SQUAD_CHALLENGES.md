# Plan — Défis d'escouade (« Cap d'escouade ») : cause racine prod + boucle UI complète

> Statut : PLAN VALIDÉ (utilisateur, 2026-07-23). Auteur : agent IA. Date : 2026-07-23.
> Exécution sous contrat `plan-execution` (ordre strict, gates, statut par item).
> Branche proposée : `fix/squad-challenges-workflow` (1 branche, commits par lot).

## Contexte et diagnostic (vérifié sur pièces + test live 2026-07-23)

Symptôme rapporté : dans le bandeau « Cap d'escouade » (onglet Escouade/Synergies,
[SquadFocusStrip.tsx](apps/web/src/features/squad/SquadFocusStrip.tsx)), le bouton
« Proposer des défis » ne fait rien.

**Diagnostic** : double erreur avalée — front sans `onError` (aucun toast), back
`serviceError` qui masquait toute erreur non-sentinelle en 500 **sans log**. Corrigé le
2026-07-23 14:16 par `04eae99ba` (toast `poolError` + `slog.ErrorContext` sur la branche
default de [prestige_squads.go:488](apps/go-api/internal/api/handlers/prestige_squads.go#L488)),
poussé sur `main` → déployé prod. **La cause racine du 500 sous-jacent n'est PAS
confirmée** — le fix ne fait que la rendre visible.

**Test fonctionnel live** (API locale `:8000`, escouade de test créée puis supprimée) —
le workflow backend est fonctionnel de bout en bout :
create squad (201) → `pool/refresh` (200, 7 templates, biais coach) → create challenge
(201, auto-join créateur Heroic) → list (200) → join idempotent (204) → evaluate (200,
progression no-overlap persistée) → rename/delete squad (204).

**Éléments de cause racine** :
- [logs/prestige.log](logs/prestige.log) local (depuis le 11/07) : zéro création
  d'escouade et zéro `pool/refresh` avant le test agent → l'observation utilisateur
  venait de la **prod** (ou d'avant le 11/07).
- Le catalogue `challenge_template` est seedé par la migration `seed_prestige_catalog_v1`
  ([prestige.go](apps/go-api/internal/games/halo_infinite/migrations/prestige.go)) depuis
  `config/titles/{slug}/challenges/templates.toml`. Catalogue vide →
  `no templates available for title` = erreur non-sentinelle = **500 masqué** (avant fix).
  Candidat n°1 pour la prod. Classe d'erreur observée localement pour `halo_5`
  (aucun catalogue : `config/titles/halo_5/challenges/` n'existe pas — log 18:24
  `EnablePilotMode` halo_5 : « no templates available for cadence daily/weekly »).

**Résidu de test assumé** : ligne `squad` orpheline `sq_15c462211708c632` (create-only,
membres retirés) + défi orphelin `sc_e0997701f9a5e98a` — illustre l'item C.2 (pas de
cascade ni de suppression de défi).

## Gaps relevés (audit workflow)

| # | Problème | Où |
|---|---|---|
| A | Liste des défis actifs : `template_id` brut affiché au lieu du label FR/EN | [SquadFocusStrip.tsx:348](apps/web/src/features/squad/SquadFocusStrip.tsx#L348) |
| B | « Rejoindre » : 204 silencieux — pas de toast ni d'invalidation, état « Rejoint » (`t.joined`) jamais affiché, participants invisibles | [usePrestige.ts:26](apps/web/src/features/prestige/hooks/usePrestige.ts#L26), [SquadFocusStrip.tsx:360](apps/web/src/features/squad/SquadFocusStrip.tsx#L360) |
| C | « Réévaluer » : la progression retournée est jetée (ni valeur, ni cible, ni complétion affichées) | [SquadFocusStrip.tsx:353](apps/web/src/features/squad/SquadFocusStrip.tsx#L353) |
| D | Aucun endpoint ni UI de suppression/abandon de défi d'escouade ; `DeleteSquad` ne cascade pas → défis orphelins éternels | [prestige.go:108-120](apps/go-api/internal/api/handlers/prestige.go#L108-L120) |
| E | `expires_at` jamais renseigné (front ne l'envoie pas, service ne le calcule pas) | [service_arcs_squads.go:326](apps/go-api/internal/prestige/service_arcs_squads.go#L326) |
| F | Fenêtre d'évaluation non implémentée : `session`/`rolling_days` → fallback 50 derniers matchs escouade, sans borne `created_at` → défi complété à la création par l'historique (observé : 181 et 243 vs cible 7) | [service_squads.go:366](apps/go-api/internal/prestige/service_squads.go#L366) (`squadWindowLimit`), [squad_progress.go:125](apps/go-api/internal/prestige/squad_progress.go#L125) |
| G | `eval_type=threshold` non implémenté pour les squads (agrégation cumulative pour tout — documenté V1) alors que le pool propose des templates threshold — **fermé au Lot 6.4** : ces templates ne sont plus proposés | [squad_progress.go:123](apps/go-api/internal/prestige/squad_progress.go#L123) |
| H | Pool éphémère : état de mutation local, non persisté, non partagé entre membres (Phase 4 « simple » documentée) | [service_pilot_pool.go:174](apps/go-api/internal/prestige/service_pilot_pool.go#L174) |
| I | `requested_by` vide contourne le contrôle de membership du pool | [service_pilot_pool.go:191](apps/go-api/internal/prestige/service_pilot_pool.go#L191) |

## Objectif et critères de succès

1. Le bouton « Proposer des défis » fonctionne **en prod** (200 + pool affiché).
2. La boucle UI est complète : labels lisibles, feedback sur chaque action, progression
   visible, suppression possible.
3. La sémantique d'évaluation est honnête : un défi ne se complète pas rétroactivement.
4. Comportement propre sur un titre sans catalogue (halo_5).

Critère global : `go test ./...` + `go vet` verts ; `tsc` + `eslint` + `vitest` verts ;
tests `-tags=integration` si persist touché ; FR sans franglais + EN (`Record<Locale, T>`) ;
revue navigateur du parcours complet ; ownership/BOLA respectés (assertMemberUser).

---

## Lot 1 — Cause racine prod (P0, diagnostic sans code) — CLÔTURÉ 2026-07-23

**Constat révisé (verif sur pièces, local)** : l'hypothèse « catalogue vide → 500 » est
**RÉFUTÉE pour un déploiement normal**. Preuves :
- `config/titles/halo_infinite/challenges/templates.toml` versionné = **28 templates**,
  donc déployé sur le VPS par `git reset --hard origin/main` (`scripts/deploy.sh`).
- Archive `data/titles/halo_infinite/warehouse/metadata-prebuilt.zip` (régénérée
  2026-06-23, extraite sur clone frais par `extractPrebuiltMetadataIfAbsent`,
  [main.go:401](apps/go-api/cmd/server/main.go#L401)) = **27 `challenge_template`** +
  `seed_prestige_catalog_v1` déjà `backfill_done=TRUE`. Une prod fraîche a donc ≥27
  templates AVANT tout seed.
- Seed idempotent `ON CONFLICT DO UPDATE`, enregistré AVANT le run metadata
  ([main.go:1562](apps/go-api/cmd/server/main.go#L1562) puis :1610), `prestigeConfigDir`
  toujours non vide ([main.go:404](apps/go-api/cmd/server/main.go#L404)). Le seul cas de
  non-rejeu = `backfill_done=TRUE` (déjà seedé) ; un échec de backfill (TOML absent)
  laisse `FALSE` et **réessaie à chaque boot** ([registry.go:329](apps/go-api/internal/migration/registry.go#L329)).
- Le symptôme « ne fait rien » = double erreur avalée, **déjà corrigé** (`04eae99ba`,
  poussé prod) : la cause du 500 est désormais LOGGÉE (`unmapped service error masked as
  500`).

**Cause racine résiduelle du 500** : non confirmable sans logs/DB prod. Candidats
restants, par ordre de vraisemblance :
1. **Lock DuckDB transitoire non mappé `ErrDBLocked`** → tombe en `default` = 500 masqué
   (au lieu de 503 retryable). `ListMembers` (shared_social) et `ListByTitle` (metadata,
   via `r.db.Query` NON recovered, [prestige_metadata_repo.go:33](apps/go-api/internal/platform/duckdb/prestige/prestige_metadata_repo.go#L33)) sont les 2 lectures concernées.
   → **Découverte D1** (voir § Découvertes) : à durcir, mais hors périmètre Lot 1.
2. metadata.duckdb prod anormale (restore d'un backup antérieur au 2026-05-19, avant
   existence du TOML) — improbable.

**Statuts** :
- [x] 1.1 — Reproduit **localement** (API :8000) : le workflow complet réussit (pool 200,
      7 templates). Repro prod = `[!]` (session prod requise, non accessible depuis l'agent).
      Commande pour l'utilisateur : voir § Commandes prod ci-dessous.
- [!] 1.2 — Logs prod inaccessibles depuis l'agent (pas de SSH VPS). Commande fournie
      (§ Commandes prod). À exécuter par l'utilisateur si le 500 réapparaît.
- [x] 1.3 — Catalogue **vérifié déployé** : TOML 28 templates versionné + prebuilt 27 +
      seed idempotent. Row-count prod exact = `[!]` (DB prod inaccessible) mais emptiness
      réfutée. Local confirmé : `pool/refresh` → 7 templates.
- [~] 1.4 — Sans objet (emptiness réfutée). Si le row-count prod (1.3) s'avérait vide
      malgré tout : `DELETE FROM schema_migrations WHERE name='seed_prestige_catalog_v1'`
      puis reboot rejoue le seed. Non exécuté (pas nécessaire + pas d'accès).
- [x] 1.5 — Cause racine documentée ici + thought_log : symptôme (avalage) corrigé ;
      500 sous-jacent = lock non mappé le plus probable (Découverte D1), catalogue exclu.

### Commandes prod (pour l'utilisateur, si le 500 réapparaît)

```bash
# 1. Le clic émet maintenant un toast (poolError) — noter le code HTTP dans la console réseau.
# 2. Sur le VPS, la cause exacte du 500 est loggée :
grep "unmapped service error masked as 500" logs/prestige.log | tail -5
grep "squad pool refreshed" logs/prestige.log | tail -5   # succès = pas de 500
# 3. Vérifier le catalogue prod (metadata tenue RW par le serveur → lire une COPIE) :
cp data/titles/halo_infinite/warehouse/metadata.duckdb /tmp/meta_ro.duckdb
duckdb -readonly /tmp/meta_ro.duckdb "SELECT count(*) FROM challenge_template;"  # attendu >= 27
```

## Lot 2 — Boucle UI minimale complète (P1) — CLÔTURÉ 2026-07-23

- [x] 2.1 Backend : `ListSquadChallenges` renvoie désormais `[]SquadChallengeView`
      (nouveau type, embed `SquadChallenge` → JSON sur-ensemble) enrichi `label_fr`/
      `label_en` via `Templates.GetByID` (title-agnostic). Défi sans template → labels
      vides. Best-effort loggé (helper `enrichSquadChallenge`). Interface Service + wrapper
      Lazy + mocks mis à jour.
- [x] 2.2 Front : label localisé affiché (`c.label_fr/label_en`), fallback `t.challenge`.
      Vérifié live : `label_fr:"Briseur de couronnes"` dans la réponse API.
- [x] 2.3 Backend : `Participants` exposés dans `SquadChallengeView` (via `ListParticipants`),
      slice non-nil même vide. Vérifié live (créateur auto-joint visible).
- [x] 2.4 Front : `useJoinSquadChallenge` invalide `queryKeys.squad.challenges(squadId)`
      au succès (param `squadId` ajouté) ; toast succès/erreur ; bouton « Rejoint »
      (désactivé, variant outline) si `participants` contient le joueur — corrige B.
- [x] 2.5 Front : « Réévaluer » stocke la progression retournée (`progressByChallenge`),
      composant `SquadProgressList` (gamertag résolu via roster, valeur/cible, nb matchs,
      badge « Atteint » token `text-success`) ; toast succès/erreur — corrige C.
      Bonus : toast sur « Créer » aussi.
- [x] 2.6 Tests : Go `TestService_ListSquadChallenges_EnrichesLabelsAndParticipants` +
      `_OK` renforcé (participants non-nil) — verts. vitest `SquadFocusStrip.test.tsx`
      (label vs template_id, état Rejoint, rendu progression) — 2/2 verts. tsc + eslint
      verts. **Revue navigateur** : vérifiée par API end-to-end (réponse enrichie live) ;
      revue VISUELLE laissée à l'utilisateur (pas de MCP navigateur dans la session agent).

## Lot 3 — Cycle de vie des défis (P1/P2) — CLÔTURÉ 2026-07-23

Choix de schéma : `squad_challenge` est une table SIMPLE (PK `id`, PAS append-only) →
archivage par colonne nullable `archived_at` (UPDATE non indexé = zéro risque ART, même
garantie que `RenameSquad`). Migration additive `add_archived_at_to_squad_challenge`
(ADD COLUMN IF NOT EXISTS) + ordre canonique. Appliquée live sur halo_infinite + halo_5.

- [x] 3.1 Backend : route `DELETE /squad-challenges/{id}?requested_by=slug` →
      `AbandonSquadChallenge` (Get défi → `assertMemberUser` BOLA → `Archive`).
      Handler + actor guard + wrapper Lazy (writer shared_social). Idempotent.
- [x] 3.2 Backend : `ListBySquad` filtre `archived_at IS NULL` (défis actifs).
      Vérifié live : abandon → 204 → liste vide.
- [x] 3.3 Backend : `DeleteSquad` archive les défis actifs en cascade (best-effort loggé)
      avant de retirer les membres. Vérifié live (delete squad avec défi → 204).
- [x] 3.4 Front : bouton « Supprimer » par défi (confirm 2e clic via `confirmingDelete`,
      token `text-destructive`) + `useAbandonSquadChallenge` (invalide le cache) + toasts.
- [x] 3.5 Backend : `expires_at` calculé à la création depuis la cadence du template
      (daily +1 j / weekly +7 j / monthly +30 j, constantes nommées `squadExpiry*`) ;
      lookup best-effort. Champ `Expired` (comparé à `s.deps.Now()` UTC, jamais
      CURRENT_TIMESTAMP SQL) marque les défis dépassés → badge « Expiré » + join désactivé.
      Vérifié live : défi weekly → expires_at = created_at + 7 j exact.
- [x] 3.6 Backend : `RefreshSquadPool` exige `requested_by` non vide (400 sinon) — bypass
      du check membership fermé. Vérifié live : requested_by vide → HTTP 400.
- [x] 3.7 Tests Go : `_AbandonSquadChallenge_ArchivesWhenMember` / `_RejectsNonMember`,
      `_CreateSquadChallenge_ComputesExpiryFromCadence`, `_DeleteSquad_CascadesArchiveChallenges`,
      `_RefreshSquadPool_RequiresRequestedBy`, `_ListSquadChallenges_MarksExpired` — verts.
      Garde anti-ART (`-tags=integration`) vert, persist prestige (real DB) vert. Route
      ajoutée au test de garde acteur (2 tables). tsc + eslint + vitest + paths verts.

## Lot 4 — Sémantique d'évaluation (P2) — CLÔTURÉ 2026-07-23

- [x] 4.1 Borne basse `since` ajoutée à `SquadMatchProvider.SquadMatchMetrics` (param
      `since time.Time`) ; le provider filtre `start_time_canonical >= since` dans
      `candidateMatches`. `EvaluateSquadChallenge` calcule `since = squadEvalSince(sc, now)`
      = created_at. **Vérifié live** : défi créé + évalué immédiatement → progression 0/0
      (candidate_matches:0), plus le bug rétroactif 181/243.
- [x] 4.2 `squadEvalSince` : `rolling_days` → `max(created_at, now - N j)` ;
      `session`/`last_n_matches` bornés à created_at (le compteur `limit` gère la
      profondeur). `session` = approximation documentée (pas de découpage de session
      escouade — noté dans le commentaire, à affiner si besoin).
- [x] 4.3 Agrégation cumulative documentée dans l'UI : légende sous le pool (« Défi
      collectif : chaque membre cumule la métrique… sur les matchs postérieurs à la
      création »), FR + EN. Option « documenter » du plan retenue ; threshold squad
      complet reste au backlog (§ Hors périmètre).
- [x] 4.4 Tests Go table-driven : `TestSquadEvalSince` (5 cas : last_n/session/rolling
      récent/rolling ancien/rolling invalide), `_BoundsSinceToCreatedAt` (borne transmise
      = created_at). prestige + persist provider verts. tsc/eslint/vitest verts.

## Lot 5 — Multi-titre + finitions (P3) — CLÔTURÉ 2026-07-23

- [x] 5.1 halo_5 : dégradation gracieuse **title-agnostic** (aucun `slug ==`) — catalogue
      vide → `RefreshSquadPool` renvoie un pool VIDE (200, `[]`) au lieu d'un 500 masqué
      (loggé pour distinguer d'un bug de seed prod). Corrige aussi le risque Découverte
      D1 côté catalogue. **Vérifié live** : `title_slug=halo_5` → `{"count":0,"pool":[]}`
      HTTP 200. Test `_EmptyCatalogDegradesToEmptyPool`. Création d'un vrai catalogue h5 =
      contenu produit hors périmètre (backlog).
- [x] 5.2 Pool éphémère assumé : la légende cumulative (Lot 4.3) documente le
      comportement ; le pool reste re-générable (mutation). Persistance partagée = backlog
      (§ Hors périmètre).
- [x] 5.3 i18n manifests — **FAIT** (validé utilisateur 2026-07-23). ~40 clés migrées de
      `squadFocusStrings.ts` (objet local FR/EN) vers `lib/i18n/manifests/squad.toml`
      namespace `[squad.focus.*]` (source unique, parité vérifiée au build manifest, ADR
      0003). Interpolation ICU : `{name}`, `{n, number}`, `{axis}`, plural
      `{n, plural, one {# match} other {# matchs}}`. `squadFocusStrings.ts` devient un
      adaptateur `getSquadFocusText(locale)` qui résout via `formatMessage(squadManifest,…)`
      en gardant l'ergonomie `t.xxx` / `t.target(n)` des composants (zéro littéral en dur).
      Manifest régénéré (`build_i18n_manifests.mjs`, déterministe) + `generated/squad.ts`
      versionné. tsc --force + eslint + vitest (289 tests squad+i18n) verts.
- [x] 5.4 Renommage — **FAIT** (validé utilisateur) : « Cap d'escouade » → « Objectifs
      d'escouade » (EN « Squad focus » → « Squad objectives »), dans le manifest
      `squad.focus.title`. Fait dans le même commit que 5.3 (migration i18n) pour ne
      toucher les libellés qu'une fois.

---

## Lot 6 — Robustesse lectures metadata + honnêteté du pool (V721-08) — CLÔTURÉ 2026-07-25

Traite la Découverte D1 (cause racine résiduelle du 500 prod) et le gap G du tableau
d'audit (`eval_type=threshold` proposé au pool alors que l'évaluation est cumulative).

- [x] 6.1 **Cause du 500 prod — routage recovery des lectures metadata.** Les 6 requêtes
      de [prestige_metadata_repo.go](apps/go-api/internal/platform/duckdb/prestige/prestige_metadata_repo.go)
      passaient par `db.Query` / `db.QueryRow` PLATS (templates : `ListByTitle`:33,
      `GetByID`:52, `Suggest`:80 ; preset arcs : `ListByTitle`:224, `GetByID`:250,
      `GetSteps`:263) — routées vers `QueryRecovered` / `QueryRowRecovered`
      (Reopen + retry une fois, cf. `duckdb/db_recovery.go`). Vérifié sur pièces : le
      helper `*Recovered` ne fait PAS de traduction d'erreur de verrou (contrairement à
      l'hypothèse de reconnaissance) ; sa sémantique est la ré-ouverture sur invalidation
      — exactement la classe d'incident qui rendait `metadata.duckdb` inutilisable
      jusqu'au restart (FATAL ART / `sql: database is closed`) et donc le 500 permanent
      sur `pool/refresh`. `Replace` (les 2) passait déjà par `UpsertNoConflict`, lui-même
      sous `WithReopenOnInvalidated` — inchangé.
- [x] 6.2 **Traduction d'erreur minimale → 503.** Helper `translateLockErr(ctx, err)`
      (même fichier) : `duckdb.IsFileLockError(err)` → `dblease.ErrDBLocked` (wrap
      `errors.Join`, warn `slog` structuré). C'est la seule erreur transitoire/retryable
      de cette chaîne (un autre process tient le fichier, ou `Reopen()` ne peut pas
      rouvrir le DSN). `PrestigeHandler.serviceError`
      ([prestige_squads.go:455](apps/go-api/internal/api/handlers/prestige_squads.go#L455))
      la mappe en 503 + `Retry-After: 5`. Toute autre erreur — `sql.ErrNoRows` inclus —
      est rendue telle quelle : mapping `ErrChallengeNotFound` / `ErrArcNotFound` inchangé.
- [x] 6.3 **Tests avec mordant** — `prestige_metadata_reopen_test.go` (tag `cgo`, patron de
      `prestige_player_reopen_test.go`) : handle fermée AVANT chacun des 6 appels, chacun
      doit récupérer et rendre la ligne seedée ; 7e cas = `GetByID(absent)` doit toujours
      donner `ErrChallengeNotFound` (la recovery ne masque pas le not-found). Pré-fix,
      chaque appel remonte « sql: database is closed » → FAIL. Le maillon lock→503 n'est
      pas simulable in-process (2e process détenteur requis) : couvert par un test unitaire
      de `translateLockErr` sur les signatures DuckDB exactes + pass-through, et par
      `TestPrestigeHandler_*_DBLocked_Returns503` (handlers) pour le maillon final.
- [x] 6.4 **Pool d'escouade : plus de défi `threshold`.** `squadEligibleTemplates`
      ([service_pilot_pool.go](apps/go-api/internal/prestige/service_pilot_pool.go)) écarte
      les templates `eval_type=threshold` du pool — leur règle affichée (« atteindre X sur
      un match ») ne correspond pas à l'évaluation réelle, cumulative pour tous en V1
      ([squad_progress.go:117-124](apps/go-api/internal/prestige/squad_progress.go#L117)).
      Filtre title-agnostic (branché sur `eval_type`, aucun `slug ==`), placé AVANT le
      test de vacuité : la dégradation gracieuse existante (pool vide → 200 + `[]` + log)
      couvre désormais aussi « catalogue entièrement écarté ». Nombre d'écartés loggé en
      `slog` structuré (`discarded` / `eligible` / `catalog`). Catalogue halo_infinite :
      28 templates dont 19 threshold → 9 éligibles, pool de 6 à 9 toujours servi.
- [x] 6.5 Tests pool : `_ExcludesThresholdTemplates` (mordant : sans filtre, les threshold
      sont dans le pool), `_AllThresholdDegradesToEmptyPool` (dégradation préservée),
      `TestSquadEligibleTemplates_CountsDiscarded` (compteur du log + ordre stable).

---

## Découvertes (règle 7 — noté, pas traité hors gate)

- **D1 (Lot 1)** — Lecture DuckDB non mappée `ErrDBLocked` → 500 masqué au lieu de 503
  retryable. Concerne les lectures `RefreshSquadPool` (`ListMembers` shared_social,
  `ListByTitle` metadata via `r.db.Query`). Candidat n°1 du 500 prod résiduel.
  → **TRAITÉE au Lot 6** (2026-07-25) côté metadata : diagnostic affiné (la cause
  dominante n'est pas le verrou mais l'INVALIDATION de la handle, non récupérée faute de
  `*Recovered`) + traduction lock→`ErrDBLocked`. `ListMembers` (shared_social) utilisait
  déjà `QueryRecovered` — non concerné.
- **D2 (Lot 6, 2026-07-25)** — `prestige_social_repo.go` conserve 5 lectures mono-ligne
  PLATES `r.db.QueryRow(` (lignes 95, 110, 232, 344, 481 : `GetUserPrestige`,
  `GetUserPrestigeCrossTitle`, `PrestigeSquadRepo.Get`, `PrestigeSquadChallengeRepo.Get`,
  `CountActiveParticipants`) alors que le reste du fichier est en `QueryRecovered`. Même
  classe de trou que D1 sur shared_social. Non traité (hors périmètre V721-08, qui scope
  le fichier metadata) — à porter dans un lot « robustesse lectures shared_social ».
- **D3 (Lot 6, 2026-07-25)** — Le garde-rail `player_db_recovery_routing_test.go` ne voit
  que les formes explicites `pdb.Player.Query(` / `pdb.ReadDB().Query(` ; les repos à champ
  nu `db *duckdb.DB` (metadata, shared_social) restent invisibles au grep et ne sont pas
  fermés par le type (`PlayerReadHandle` ne couvre que les player DB, par conception).
  Un ratchet équivalent pour ces couches supposerait de traiter D2 d'abord.

## Hors périmètre (noté, non traité — règle « zéro fix opportuniste »)

- [!] **Threshold squad par match (évaluation complète)** — non traité. Justification :
  demande un 2e mode d'agrégation dans `AggregateSquadProgress` (cible atteinte sur UN
  match vs cumul) + un choix produit sur la sémantique collective (« tous les membres »
  vs « au moins un »). Le défaut *utilisateur* (règle affichée ≠ règle appliquée) est
  fermé autrement par le Lot 6.4 (ces templates ne sont plus proposés). Backlog.
- [!] **Persistance / partage du pool entre membres** — non traité. Justification : le
  pool est aujourd'hui un état de mutation local (Phase 4 « simple » assumée, gap H) ; le
  persister suppose une table `squad_pool` + une politique de péremption
  (`refresh_period_days` existe déjà en tuning mais n'est pas appliqué) — chantier de
  schéma à part entière, sans lien avec les défauts corrigés ici. Backlog.
- [!] **Catalogue de templates Halo 5** — non traité. Justification : c'est du CONTENU
  produit (rédiger `config/titles/halo_5/challenges/templates.toml` : métriques
  disponibles pour H5, paliers calibrés, libellés FR/EN), pas du code ; il exige un
  arbitrage utilisateur sur les métriques H5 pertinentes. Le comportement code sans
  catalogue est correct et testé (pool vide, 200, log — item 5.1). Backlog.
- [!] **Intégration des défis d'escouade dans Ascension / Réalisations** — non traité.
  Justification : nouvelle surface produit (agrégation des défis collectifs dans la
  progression individuelle + règles PP associées), hors du périmètre « réparer la boucle
  Escouade ». Backlog.
- [~] **Durcissement lock→503 des lectures prestige (Découverte D1)** — couvert au Lot 6
  (items 6.1/6.2) pour la voie metadata. Reste D2 (shared_social) au backlog.

## Journal d'exécution

- 2026-07-23 : plan rédigé et validé (diagnostic + audit dans le thought_log du jour).
- 2026-07-23 : **Lot 1 CLÔTURÉ**. Branche `fix/squad-challenges-workflow` créée.
  Hypothèse « catalogue vide » réfutée sur pièces (prebuilt 27 + TOML 28 + seed idempotent).
  Cause du 500 résiduel = lock non mappé le plus probable (Découverte D1), non confirmable
  sans logs prod (items 1.2/1.3-rowcount = `[!]`, commandes fournies à l'utilisateur).
  Symptôme utilisateur déjà corrigé par `04eae99ba`. → passage Lot 2.
- 2026-07-23 : **Lot 2 CLÔTURÉ**. Boucle UI complète : type `SquadChallengeView`
  (labels + participants), enrichissement service best-effort, feedback Rejoindre/Rejoint,
  affichage progression par membre. Gates verts (go build/test prestige+handlers+wire, tsc,
  eslint, vitest 2/2). Vérifié LIVE via API (réponse enrichie : label_fr + participants).
  Revue visuelle navigateur = à faire par l'utilisateur (pas de MCP navigateur agent).
  → passage Lot 3.
- 2026-07-23 : **Lot 3 CLÔTURÉ**. Cycle de vie : archivage `archived_at` (migration
  additive, colonne non indexée = zéro ART), endpoint DELETE + abandon, cascade DeleteSquad,
  `expires_at` par cadence + marquage `Expired`, garde `requested_by`. Gates verts (go build,
  tests prestige/handlers/wire/migration/persist, anti-ART integration, tsc/eslint/vitest).
  Vérifié LIVE : expires_at +7 j exact, requested_by vide → 400, abandon → 204 + liste vide,
  cascade delete → 204. → passage Lot 4.
- 2026-07-23 : **Lot 4 CLÔTURÉ**. Sémantique d'évaluation : borne basse `since` (created_at,
  resserrée rolling_days) transmise au provider qui filtre `start_time >= since`. Fin de la
  complétion rétroactive. Légende UI cumulative. Gates verts. Vérifié LIVE : éval immédiate
  → 0/0 (candidate_matches:0). → passage Lot 5.
- 2026-07-23 : **Lot 5 CLÔTURÉ — 5.1/5.2/5.3/5.4 TOUS FAITS.** Dégradation halo_5 :
  catalogue vide → pool vide 200 (title-agnostic, plus de 500). Vérifié live. 5.3
  (migration i18n de `squadFocusStrings.ts` vers le manifest `squad.toml`) et 5.4
  (« Cap d'escouade » → « Objectifs d'escouade ») ont été livrés dans la foulée, validés
  par l'utilisateur le même jour — commits `ac7ae83d7` / `93812f0ca`.
  *(Correctif de journal 2026-07-25 : cette entrée annonçait « 5.3/5.4 différés justifiés »,
  en contradiction avec les cases `[x]` du Lot 5 et avec les commits. Les cases faisaient
  foi ; le journal était périmé.)*
  **PLAN COMPLET** — les 4 gaps fonctionnels (labels, feedback, cycle de vie, éval honnête)
  sont livrés. Reste à faire par l'utilisateur : revue VISUELLE navigateur.
- 2026-07-23 : **Refactor seuil 500 L (delivery-checklist §5)**. Deux fichiers avaient
  FRANCHI 500 par le chantier : `SquadFocusStrip.tsx` (426→597) scindé en
  `squadFocusStrings.ts` (115) + `SquadObjectivesPanel.tsx` (267) + strip (240) ;
  `service_arcs_squads.go` (431→535) scindé, défis d'escouade extraits vers
  `service_squad_challenges.go` (248) + arcs (298). Tous < 500. Gates re-verts (tsc --force,
  eslint, vitest, go vet/build, intégration `-p 1` 0 FAIL). Les 3 god-files déjà >500
  (service.go, prestige_lazy_service.go, steps_shared_social.go) : ajouts minimes
  inévitables (méthode d'interface / wrapper / step de migration), dette gelée baseline.
- 2026-07-25 : **Lot 6 CLÔTURÉ (item V721-08)** — branche `feat/v7.2.1-notion-batch`.
  (a) Cause du 500 prod : les 6 lectures de `prestige_metadata_repo.go` étaient PLATES ;
  une handle metadata invalidée (FATAL ART) ou fermée (`sql: database is closed`) rendait
  le catalogue Prestige illisible jusqu'au restart → `default` du handler = 500 masqué.
  Routées vers `QueryRecovered` / `QueryRowRecovered`. (b) `translateLockErr` mappe la
  contention fichier vers `dblease.ErrDBLocked` → 503 + `Retry-After: 5`. (c) Le pool
  d'escouade n'expose plus de template `eval_type=threshold` (évaluation cumulative en V1)
  — filtre title-agnostic + log structuré des écartés, dégradation pool vide préservée.
  Tests : `prestige_metadata_reopen_test.go` (7 cas, mordant : FAIL pré-fix) +
  3 tests pool dans `service_squads_test.go`. **PLAN CLOS** : tous les items du plan sont
  statués (`[x]` / `[~]` / `[!]` justifiés). Restent au backlog les 4 items hors périmètre
  et les découvertes D2/D3. **Reste à faire par l'utilisateur : revue VISUELLE navigateur**
  du parcours Escouade (pool proposé, plus de défi « seuil », toasts, progression).
