# Plan — Refonte page Communauté > Relations (Hub segmentable)

> Statut : PLAN (validé sur mocks `.ai/mocks/relations/relations-mockups.html`, proposition 1 « Hub segmentable »).
> Mémoire liée : `project_relations_page_redesign`, `feedback_fr_ui_no_anglicisms_solid_pills`.

## Objectif & critère de succès

Faire de **Relations** le hub d'exploration du réseau social du joueur, distinct de :
- **Carrière** (aperçu tous-temps : top-10 + butterfly némésis/proie) — **inchangé**.
- **Explorer** (détail d'UN joueur : recherche, briefing, heatmap horaire, face-à-face) — **inchangé**.

**Succès Phase 1** : la page affiche, à partir d'un **vrai endpoint backend**, l'ensemble des joueurs récurrents avec données réelles (taux de victoire allié/ennemi, frags échangés, badges, dernière rencontre), au style des conventions de l'app, libellés 100 % FR. Plus aucune donnée fabriquée côté front.

## Constat (état actuel)

- Front : `useRelationsPage` → `GET /pages/career/encounters` (Q10 léger) → `mapCareerEncountersToRelations` **fabrique** `RelationsPageResponse` (wins=0, win_rate=null, pas de frags échangés, last_seen=null). Page dégradée.
- Backend : **aucune** struct Go `RelationsPageResponse` (grep vide). L'endpoint `/pages/palmares/relations` est **mocké en test** (`apps/web/src/test/handlers.ts:530`) mais **non implémenté**.
- La donnée riche existe déjà : `Q26CareerTopEncountersTpl` (`internal/platform/duckdb/queries_career_encounters.go`) calcule wins_as_ally/vs_enemy, kills_dealt/deaths_suffered, last_seen — mais capée à 10 et servie à Carrière.
- Badges existants : `internal/analysis/narrative/encounter.go` (`ComputeEncounterBadges` : ally_plus, tough_enemy, coriace, ordinal).
- Lecture seule, agrégation → **aucun risque ART / append-only** (pas d'écriture).

## Architecture cible (couches Go — arch-rules)

| Couche | Fichier | Rôle |
|---|---|---|
| domain | `internal/domain/relations.go` (nouveau) | `RelationsPage`, `RelationInsight` (champs riches + badges + catégorie) |
| analysis | `internal/analysis/narrative/encounter.go` (étendu) | nouveaux badges purs (duo_gagnant, cameleon, de_longue_date, recrue, proie_favorite) + seuils |
| analysis | `internal/analysis/relations/categorize.go` (nouveau) | catégorisation pure (alliés/rivaux/noyau dur/synergies) + overview |
| port | `internal/port/services.go` (étendu) | `RelationsService` interface |
| service | `internal/service/relations_service.go` (nouveau) | orchestration repo + analysis → `domain.RelationsPage` |
| platform | `internal/platform/duckdb/queries_career_encounters.go` + `relations_repo.go` | Q26 paramétré (limit, first_seen, cascade) + méthode `GetRelations` |
| handler | `internal/api/handlers/` (career.go ou community.go) | `GET /players/{slug}/pages/palmares/relations` (decode→service→encode) |

Pattern repo existant (`career_service_encounters.go` → `career_repo_encounters.go`) réutilisé. Logging `slog.*Context` à chaque opération significative + erreurs.

## Badges — définitions, seuils (ajustables), source

| Badge | Statut | Règle (seuils proposés) | Source donnée | Token couleur |
|---|---|---|---|---|
| Allié+ | existant | (inchangé) | Q26 | narrative-encounter-ally-plus (vert) |
| Coriace | existant | (inchangé) faible taux de victoire vs lui | Q26 | tough-enemy (rouge) |
| Dur à cuire | existant | (inchangé) ratio frags/morts défavorable | Q26 | tough-enemy (rouge) |
| Total N× | existant | (inchangé) rencontres croisées | Q26 | ordinal (bleu) |
| **Duo gagnant** | nouveau | taux victoire allié ≥ 60 % ET ≥ 10 matchs alliés | Q26 (wins_as_ally) | nouveau (vert) |
| **Caméléon** | nouveau | min(allié,ennemi)/total ≥ 0.4 ET total ≥ 10 | Q26 (ally/enemy_count) | nouveau (bleu) |
| **De longue date** | nouveau | first_seen > 6 mois OU total ≥ 80 | Q26 + MIN(start_time) | nouveau (bleu) |
| **Recrue** | nouveau | first_seen < 30 j ET total ≥ 4 | Q26 + MIN(start_time) | nouveau (bleu) |
| **Proie favorite** | nouveau | ratio duel > 1.5 ET ≥ 6 matchs ennemis | Q26 (kills/deaths) | nouveau (vert) |
| **Aussi sur Halo 5** | nouveau (Phase 3) | co-rencontré sur un autre titre ≥ 3 matchs | cross-title (xuid global) | nouveau (bleu/violet) |

« Bête noire » = **titre du KPI hero** (rival n°1, superlatif unique), pas un badge de ligne.
Rendu : badges existants en style `NarrativeBadge` actuel (teinté) ; nouveaux en plein + texte blanc (contraste AA). Cf. `feedback_fr_ui_no_anglicisms_solid_pills`.

## Phases

### Phase 1 — Vrai endpoint + page hub (cœur, livrable seul) — effort moyen

Backend :
1. `domain/relations.go` : `RelationInsight` (xuid, gamertag, total, ally/enemy counts+wins, winrates, kills_dealt, deaths_suffered, ratio, first_seen, last_seen, badges []EncounterBadge, category) + `RelationsPage` (overview + listes).
2. `platform/duckdb` : généraliser Q26 (paramètre `limit` ; ajouter `MIN(start_time) AS first_seen`) ; méthode repo `GetRelations(ctx, xuid, opts)`.
3. `analysis/narrative/encounter.go` : ajouter les 5 nouveaux badges (fonctions pures + seuils constants nommés).
4. `analysis/relations/categorize.go` : overview + tri/catégorisation (alliés fréquents, synergies, némésis, proies, noyau dur).
5. `port` + `service/relations_service.go` : assemblage. Dégradation si pas de données → page vide gracieuse.
6. `handler` : `GET /players/{slug}/pages/palmares/relations` + enregistrement route + registry.

Frontend :
7. `lib/api/types.ts` : enrichir `RelationInsight` (badges, kills_dealt, deaths_suffered, first_seen_at, category) + garder `RelationsPageResponse`.
8. `features/palmares/queries.ts` : `useRelationsPage` → vrai endpoint ; **supprimer** `mapCareerEncountersToRelations` + `CareerEncounters*` fakes.
9. `features/palmares/PalmaresRelationsPage.tsx` : réécriture « hub » — hero 3 KPI (KpiCard, accent token), tableau unifié (langage `MatchEncountersTable` : SplitBar, NarrativeBadge), section « Noyau dur » (cards flottantes `bg-card`), chips de catégorie (Tous/Noyau dur/Alliés/Rivaux/Récents) en filtre client.
10. i18n : `manifests/palmares.toml` complété FR+EN (corriger « Win rate »→« Taux de victoire », libellés colonnes, KPI hero, sections) ; `manifests/squad.toml` (narrative.encounter.*) pour les nouveaux badges ; regen `build_i18n_manifests.mjs`.
11. Tokens : `lib/accessibility/semantic-tokens.ts` + `palettes/*` + `globals.css` (ac-*) — nouveaux tokens badges. Aucun hex/Tailwind couleur dans le composant.

Tests Phase 1 :
- analysis : badges (table-driven, seuils limites) + categorize.
- service : mock `port` repo.
- handler : `httptest` (200 nominal + page vide).
- duckdb : `:memory:` sur la requête étendue (CGO requis, cf. `reference_duckdb_query_go`).
- front : `PalmaresRelationsPage.test.tsx` mis à jour (endpoint réel mocké) + typecheck + vitest hors sandbox.

Done Phase 1 : page rendue depuis l'endpoint réel, badges existants + nouveaux, FR complet, `go test ./...` + `npm run typecheck/lint` + vitest verts, entrée `thought_log.md`.

### Phase 2 — Segmentation active (le différenciateur) — effort moyen

- Barre filtres calquée Synthèse : `ExperienceDropdown` + `SaisonPill` + `PeriodePill` + `MultiSelectFilter` (playlist, mode) + **Vue Solo/Escouade** (nouveau). Counts cascade via `/filters/resolve` (`useFiltersPreview`), pattern pending→committed + bouton « Analyser » (cf. `SynthesisPage.tsx`).
- Backend : l'endpoint accepte `FilterContextInput`/cascade (experience_types, playlists, modes, période/saison) → clauses WHERE sur Q26. Dimension **solo/escouade** = `player_match_enrichment.is_with_friends` (jointure player DB) → nouvelle clause.
- Tests : service avec cascade ; front : interaction filtres.
- Done : un segment (ex. « Classé ») change réellement le tableau et les KPI.

### Phase 3 — Cross-jeu + Moments & Rivalités (avancé / v2) — effort lourd

- **Badge cross-jeu** « Aussi sur Halo 5 » : vérif co-rencontre inter-titres. xuid global ; pour chaque autre titre du `TitleRegistry`, lecture `shared.match_participants` via `PathResolver.SharedDBPath(otherSlug)` (jamais de `filepath.Join` direct), co-occurrence ≥ N(=3). Cache léger. Dégradation si titre indisponible.
- **Heatmap agrégé** (joueurs × tranches horaires) : endpoint + composant aligné sur `ExplorerActivityHeatmapChart` (rampe heatmap-cold→hot, libellés). À valider en vrai (lisibilité — réserve user).
- **Rivalités** : taux de victoire glissant vs rival + frise de duels (outcome tape) + écart de frags comblé. Série temporelle par rival (fenêtre glissante) → coût backend ; chart léger (SVG ou wrapper ECharts).
- Done : badge cross-jeu visible ; onglet/section Moments rendu ; décision go/no-go heatmap agrégé selon lisibilité réelle.

## Multi-titres / capabilities (plan-review §3-4)

- Page transverse **non gatée** (dérive des matchs) — cohérent avec `navL1Sections.tsx`.
- Titre courant via `ctxkeys.TitleSlug(ctx)` ; chemins via `PathResolver` ; cross-titre (Phase 3) via `TitleRegistry` + `PathResolver`, branché sur présence de données, jamais sur le slug.
- Pas de nouveau FieldKey nécessaire (réutilise les colonnes encounter existantes).

## Branche & livraison

- Branche : **`feat/relations-hub`** (depuis `main` à jour ; ne jamais travailler sur `main`). 1 branche, N commits (un par phase/sous-étape).
- Commit uniquement sur autorisation explicite (cf. `feedback_ask_before_commit`).
- `thought_log.md` à chaque phase. Merge → `main` = déploiement prod auto (prévenir).

## Risques / points ouverts

- Saison dispo pour `SaisonPill` sur la page (réutiliser `useActiveSeason`) — à confirmer Phase 2.
- Coût cross-titre Phase 3 (N titres × requête co-occurrence) — borné + caché.
- Seuils badges à valider sur données réelles (les valeurs ci-dessus sont des points de départ).
