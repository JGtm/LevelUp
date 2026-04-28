# PUNCHLIST — Handover GS → OP

> **Source de vérité courte** : ce qui reste, qui fait quoi, comment reprendre.
> Détail technique → liens vers `.ai/PLAN_*.md`. Pas de prose ici.
> **Update obligatoire** : statut mis à jour dans le **même commit** qui touche le chunk.

## Contexte

- **GS** off à partir du **2026-04-28** — **date de retour TBD**.
- **OP** prend la relève. Review au retour de GS.
- Pas de merge sur `main` pendant l'absence — tout en branches `feat/op-*`.
- Si tu finis tout le sprint actif, attaque le backlog. Sinon laisse propre, on revoit ensemble.

## Légende statuts

`[ ]` backlog · `[~]` WIP · `[?]` blocked · `[R]` await GS review · `[x]` done

## Démarrer en 5 minutes

1. **Lire le dernier thought_log** (5 dernières entrées) : [.ai/thought_log.md](thought_log.md)
2. **Branche active** : `feat/foundations-axes-1-3-4` — tout est ici.
3. **Smoke check** : tout doit être vert au départ.
   ```bash
   cd apps/go-api && go test ./...
   cd apps/web && npm run typecheck && npx vitest run
   ```
   Référence : `Tests 616 passed (78 files)` côté front au 2026-04-28.
4. **Visualiser la stack ECharts** : démarre le front, va sur `/lab/charts` — tu vois les 11 wrappers en sandbox.
5. **Conventions projet** : [CLAUDE.md](../CLAUDE.md) (couleurs tokens, i18n manifest, taille fonctions max 80L).
6. **Skills agent** : [.claude/skills/arch-rules](../.claude/skills/arch-rules/SKILL.md) · [delivery-checklist](../.claude/skills/delivery-checklist/SKILL.md) · [plan-review](../.claude/skills/plan-review/SKILL.md) · [foundations-usage](../.claude/skills/foundations-usage/SKILL.md). À invoquer avant tout commit.

## Phase 4 méta — État (livré 2026-04-28)

- `[x]` **P4M.A** ADRs — `docs/adr/000{1,2,3,4}.md` (ECharts, canonical, i18n manifest, narrative). Commit `21929869`.
- `[x]` **P4M.B** FOUNDATIONS_GUIDE — EN + FR (`docs/FOUNDATIONS_GUIDE.md` + `docs/FR/`). Commit `30bb01b0`.
- `[x]` **P4M.C** READMEs — `analysis/{temporal,breakdown,narrative}/README.md` + `components/charts/README.md`. Commit `9acfe0ea`.
- `[x]` **P4M.D** Skill `foundations-usage` + CLAUDE.md + project_map.md. Ce commit.

**Phase 4 méta complète.** Branche `feat/foundations-docs-skills` prête à merger après revue.

## Just shipped (8 derniers chunks Phase 3)

| Hash | Chunk | Synthèse |
|------|-------|----------|
| `345f6b32` | P3 Option C | ChartsShowcasePage `/lab/charts` (galerie 11 wrappers) |
| `4b79a35d` | P3 Option B | Retrait 14 champs DTO Plotly orphelins (timeseries + citations) |
| `143ca345` | P3.F+G | TimeseriesCombatYield + TimeseriesKdaBars Plotly → ECharts |
| `718d9379` | P3.E | Cleanup Plotly fields dead SessionCompare |
| `9363f2b2` | P3.D | DonutChart wrapper + outcomes SessionCompare |
| `6cf940aa` | P3.C | Home + Palmares legacy `i18n.ts` → manifests TOML |
| `b655d0f2` | P3.B | HistogramChart + ScatterChart wrappers + cleanup 4 Plotly |
| `038bd92a` | P3.A | Media `i18n.ts` → manifest TOML |

**TimeseriesPage et SessionComparePage = 100% ECharts.** Détails dans `.ai/thought_log.md`.

## Sprint actif (priorité top → bottom)

| St | Owner | Chunk | Done = | Branche | Ref |
|----|-------|-------|--------|---------|-----|
| `[x]` | GS | **P4.A — Synthesis Plotly → ECharts** ✅ Review GS 2026-04-28 : ChartCard + Heatmap2DChart en place, builder pur `buildBipolaireOption`, tests adaptés via `vi.mock`, aucun blocker. Commit `20563701`. | livré | `feat/op-synthesis-echarts` | [.ai/PLAN_SYNTHESIS_GO_PORTAGE.md](PLAN_SYNTHESIS_GO_PORTAGE.md) |
| `[x]` | GS | **P4.B — Squad Contributions Plotly → ECharts** ✅ Review GS 2026-04-28 : RadarChart wrapper, helpers purs `buildRadarSeries` + `availableMetrics` (filtre multi-titres correct), 3 empty states conservés. Commit `edfd1342`. | livré | `feat/op-squad-contributions-echarts` | [.ai/PLAN_SQUAD_GO_PORTAGE.md](PLAN_SQUAD_GO_PORTAGE.md) |
| `[x]` | GS | **P4.C — Squad Synergies Plotly → ECharts** ✅ Review GS 2026-04-28 : 19 tests vitest verts, 3 helpers réécrits en pure TS, Plotly retiré. Note : les helpers étaient frontend-only, pas server-side comme craint. **Nit** : `timelineChart.test.ts:9` fixture incomplète (TS strict). Commit `6b62ffab`. | livré | `feat/op-squad-synergies-echarts` | [.ai/PLAN_SQUAD_GO_PORTAGE.md](PLAN_SQUAD_GO_PORTAGE.md) |
| `[x]` | GS | **P4.D — Final cleanup Plotly** ✅ Review GS 2026-04-28 : 0 import actif PlotlyChart, package.json clean, 2 commentaires historiques OK. Cleanup complémentaire `483c593e` retire `CareerPageCharts` orphelin Go + openapi.yaml. Commit `16fb335e`. | livré | même branche que P4.C | — |

> **Suggestion d'ordre** : P4.A en premier (le plus simple, valide ton workflow). Puis P4.B. P4.C est lourd (5 charts + portage Go) — si tu te retrouves en milieu de C avec peu de temps, **commit WIP** et laisse-moi le finir. P4.D = trivial après C.

## Backlog (si P4.A→D liquidé, attaque dans l'ordre)

| St | Owner | Chunk | Effort | Ref |
|----|-------|-------|--------|-----|
| `[x]` | GS | **Squad/Sessions §1-2** ✅ Review GS 2026-04-28 : Go `PickedSquadSessions []string` + tri `StartedAt`, `SessionMultiSelect` 232 LoC + 19 tests pass, localStorage cohérent, queryKeys 5e arg propre. Bonus fix storage mock Node v25 (478/478). Commit `d2e2e42b`. | lourd | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) — branche `feat/op-squad-sessions-multiselect` |
| `[x]` | GS | **Squad/Sessions §3** ✅ Review GS 2026-04-28 : `filterTopRowsToFriends` Go (5 tests purs), `AddFriendFlow` partagé (7 tests vitest), CTA `onAddAsFriend`. Filtre branché sur la donnée pas sur le slug → multi-titres compatible. Commits `575da592` + `30d86be9` + `e9a39112`. | lourd | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md §3+§3bis](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) — branche `feat/op-squad-friends-flow` |
| `[x]` | GS | **Squad/Sessions §4** ✅ Review GS 2026-04-28 : `friends_recompute.go` (233 LoC) + orchestrator (171 LoC) sous seuils, sémantique additive idempotente garde FALSE, multi-titres via `LoadPlayers()`, suite sync + service PASS. **Nit** : pas de test direct sur wrapper `RecomputeIsWithFriends` (core testé via §7). Commit `4f68d923`. | très lourd | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md §4](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) — branche `feat/op-squad-friends-flow` |
| `[x]` | GS | **Squad/Sessions §5** ✅ Review GS 2026-04-28 : `BuildSessionLabelsList` extrait dans `service/session_labels.go` (factorisation), filter solo après `FilterContext` (ordre correct), localStorage pattern aligné. **Nit** : pas de test dédié au helper et au filter (logique simple, couverture indirecte via service tests). Commit `3f122f1e`. | moyen | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md §5](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) — branche `feat/op-squad-friends-flow` |
| `[x]` | GS | **Squad/Sessions §6** ✅ Review GS strict-mode 2026-04-29 (commits `1da52fb8` + `41788324`) : architecture (couches Go respectées, failsafe panic recover cohérent), multi-titres (slug paramétré, pas de hardcoding), tests stricts (11 sub-tests `TestNewFriendsAdded` + 7 tests notify : set diff case-insensitive, trim, plural FR, no-op edge cases), i18n (6 clés FR/EN Discord, NotificationCategory 12→14, ALL_CATEGORIES synchronized). **Dette pré-existante** détectée : `log.Printf` au lieu de `slog` dans tout `notify/notifiers.go` (pas spécifique §6). | moyen | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md §6](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) |
| `[x]` | GS | **Squad/Sessions §7** — Hook auto-recompute `is_with_friends` post-sync delta (engine `WithFriendsLoader` + `RecomputeIsWithFriendsCore` core helper sans leases + câblage SyncHandler + scheduler auto_sync + 3 tests purs). Comble le gap : avant ce hook, tout match nouvellement sync restait `is_with_friends=FALSE` jusqu'au PATCH /settings ou CLI manuel. | rapide | branche `feat/op-squad-friends-flow` (commit GS) |
| `[R]` | OP | **Form score intra-match** — réflexion en cours | moyen | [.ai/REFLEXION_FORM_SCORE_INTRA_MATCH.md](REFLEXION_FORM_SCORE_INTRA_MATCH.md) — exploratoire |
| `[x]` | GS | **Finition multi-titres** — Phases 1–5 livrées (audit terrain 2026-04-28) + Phase 6 static FS title-scoping bouclée bout en bout (branche `feat/multi-title-static-fs-rescope`, 6 commits f4a69679 → 758870a5). Migration FS de 328 fichiers + 180 rows DB UPDATE + 3e adapter `TitleAssetURLAdapter` + cleanup flag transitionnel. | moyen | [.ai/PLAN_FINITION_MULTI_TITLE.md](PLAN_FINITION_MULTI_TITLE.md) |

## Blocked / décisions à trancher (review queue — **pas en autonomie**)

- ~~**`OutcomeSequenceTape` à garder ?**~~ ✅ **Validé en prod** (2026-04-28, GS retour) — composant gardé tel quel, mention prévue dans `components/charts/README.md` (chunk Phase 4 méta P4M.C).
- ~~**Squad Synergies (P4.C) cross-stack**~~ ✅ **Livré par OP** dans la même branche.

## Conventions OP (lis ça une fois)

- **Branches** : `feat/op-<sujet-court>`. Une branche par chunk = facile à reviewer à mon retour.
- **Commits** : reprends le format des 8 derniers commits — `feat(scope): description courte - chunk PX.Y` + bullets dans le body. Voir `git log --oneline -15`.
- **thought_log** : entry obligatoire avant commit. Format : `## [YYYY-MM-DD] feat(...): ...` + Statut · Décision · Résultats · Prochaine étape. Signe avec ton initiale OP en fin d'entrée.
- **PUNCHLIST** : ce fichier — update statut **dans le même commit** que le chunk. Quand tu commences : `[ ]` → `[~]`. Quand tu finis : `[~]` → `[R]` (await GS review).
- **Pas de merge sur `main`** : push tes branches, c'est tout.
- **Tests** : ne désactive pas un test qui passe. Si un test casse à cause de tes changements, tu **fixes** ; tu ne `.skip()` pas.
- **i18n** : aucune string FR/EN hardcodée dans `features/` ou `components/`. Utilise `formatMessage(xxxManifest, key, locale)`. Voir manifests dans `apps/web/src/lib/i18n/manifests/`.
- **Couleurs** : aucun hex / aucune classe Tailwind couleur (`text-red-*`, `bg-green-*`). Utilise `tokenCssVar('outcome-win')` ou `resolveToken('outcome-win')`. Voir `lib/accessibility/`.

## Si tu es bloqué (decision tree)

1. Relire la **ref du chunk** dans `.ai/PLAN_*.md` (colonne Ref du sprint actif).
2. Skim le **skill** correspondant (`.claude/skills/{plan-review,delivery-checklist,arch-rules}`).
3. `grep -r "<motif>" apps/web/src` ou `apps/go-api/internal` pour trouver un précédent.
4. **Toujours bloqué** : `git commit -am "WIP(P4.X): description du blocage"`, statut `[?]`, passe au chunk suivant. Ping GS en MP si critique.

## Smoke check de fin de session (à faire avant de te déconnecter)

```bash
# Go
cd apps/go-api && go test ./... && go vet ./...
# Front
cd apps/web && npm run typecheck && npx vitest run
# Hardcoded fields lint
node tools/lint-no-hardcoded-fields.mjs
# Cherche les fmt.Println oubliés
grep -rn "fmt\.Println\|log\.Printf" apps/go-api/internal/
```

Tout doit être vert. Si rouge → fix avant de pusher OU commit WIP avec `[?]` dans la PUNCHLIST.

## Références

**Plans détaillés** :
- [.ai/PLAN_META_FOUNDATIONS_GO.md](PLAN_META_FOUNDATIONS_GO.md) — vue d'ensemble Phase 1+2+3
- [.ai/PLAN_SYNTHESIS_GO_PORTAGE.md](PLAN_SYNTHESIS_GO_PORTAGE.md) — pour P4.A
- [.ai/PLAN_SQUAD_GO_PORTAGE.md](PLAN_SQUAD_GO_PORTAGE.md) — pour P4.B + P4.C
- [.ai/PLAN_TIMESERIES_GO_PORTAGE.md](PLAN_TIMESERIES_GO_PORTAGE.md) — historique TimeseriesPage (déjà livrée)
- [.ai/PLAN_FINITION_MULTI_TITLE.md](PLAN_FINITION_MULTI_TITLE.md) — backlog multi-titres

**Architecture & conventions** :
- [CLAUDE.md](../CLAUDE.md) — règles projet (taille fonctions, couleurs, i18n)
- [.claude/skills/arch-rules](../.claude/skills/arch-rules/) — couches Go + multi-titres
- [.claude/skills/delivery-checklist](../.claude/skills/delivery-checklist/) — go/no-go avant commit
- [.claude/skills/plan-review](../.claude/skills/plan-review/) — grille de revue de plan

**Sandbox visuelle** :
- Route `/lab/charts` (en local) — galerie des 11 wrappers ECharts dispo

**Historique des décisions** :
- [.ai/thought_log.md](thought_log.md) — chronologique, 5 dernières entrées suffisent pour le contexte récent
