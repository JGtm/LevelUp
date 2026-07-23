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
| G | `eval_type=threshold` non implémenté pour les squads (agrégation cumulative pour tout — documenté V1) alors que le pool propose des templates threshold | [squad_progress.go:123](apps/go-api/internal/prestige/squad_progress.go#L123) |
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

## Lot 4 — Sémantique d'évaluation (P2)

- [ ] 4.1 Borner les matchs comptés à `created_at` du défi (filtre dans
      `SquadMatchMetrics` ou en aval avant `AggregateSquadProgress`) — minimum vital,
      corrige le cœur de F : plus de complétion rétroactive.
- [ ] 4.2 Mapper réellement la fenêtre : `rolling_days` → borne temporelle
      `now - N jours` (max avec `created_at`) ; `session` → approximation documentée
      (borne `created_at` + gap de session existant si réutilisable —
      vérifier `internal/analysis` avant d'implémenter, skill `go-features`).
- [ ] 4.3 Court terme pour G : filtrer le pool (`RefreshSquadPool`) aux templates
      compatibles cumulatif OU documenter l'agrégation cumulative dans le libellé UI.
      L'implémentation threshold squad complète reste hors périmètre (noter au backlog).
- [ ] 4.4 Tests Go table-driven : matchs antérieurs exclus, fenêtre rolling_days,
      pool filtré.

## Lot 5 — Multi-titre + finitions (P3)

- [ ] 5.1 halo_5 : décision — créer `config/titles/halo_5/challenges/templates.toml`
      (catalogue minimal) OU dégradation propre : capability fine (`capabilities.toml`)
      → strip masqué / 503 `ErrCapabilityNotSupported` au lieu de 500. Jamais de
      `slug == "halo_5"` (ratchet).
- [ ] 5.2 Pool : assumer l'éphémère (libellé UI « Suggestions » + le pool reste
      re-générable) — la persistance partagée entre membres est notée au backlog, pas
      dans ce chantier.
- [ ] 5.3 i18n : migrer les `STRINGS` locaux de `SquadFocusStrip` vers les manifests
      squad (`squad.toml` → `i18n/generated`), parité FR/EN par typage.
- [ ] 5.4 Optionnel (goût utilisateur) : renommer « Cap d'escouade » si le terme ne
      parle pas (candidats : « Objectifs d'escouade », « Cap de l'escouade » — à
      trancher avec l'utilisateur avant 5.3 pour ne migrer les strings qu'une fois).

---

## Découvertes (règle 7 — noté, pas traité hors gate)

- **D1 (Lot 1)** — Lecture DuckDB non mappée `ErrDBLocked` → 500 masqué au lieu de 503
  retryable. Concerne les lectures `RefreshSquadPool` (`ListMembers` shared_social,
  `ListByTitle` metadata via `r.db.Query`). Candidat n°1 du 500 prod résiduel. Durcissement
  (mapper les erreurs de lock vers `ErrDBLocked`) = hors périmètre de ce chantier UX ;
  à porter dans un chantier « robustesse lectures prestige » si le log prod le confirme.

## Hors périmètre (noté, non traité — règle « zéro fix opportuniste »)

- Threshold squad complet (évaluation par match) — backlog.
- Persistance/partage du pool entre membres — backlog.
- Intégration des défis d'escouade dans la page Ascension/Réalisations — backlog.
- Durcissement lock→503 des lectures prestige (Découverte D1) — backlog robustesse.

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
