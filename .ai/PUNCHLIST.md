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
6. **Skills agent** : [.claude/skills/arch-rules](../.claude/skills/arch-rules/SKILL.md) · [delivery-checklist](../.claude/skills/delivery-checklist/SKILL.md) · [plan-review](../.claude/skills/plan-review/SKILL.md). À invoquer avant tout commit.

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
| `[R]` | OP | **P4.A — Synthesis Plotly → ECharts** : 2 charts (`bipolaireChart` + `heatmapChart`) dans `SynthesisPage.tsx` (lignes 365 et 414). Le premier ressemble à un BarStacked, le second à un Heatmap2D — wrappers déjà dispo. | typecheck OK + vitest OK + thought_log + commit `feat(synthesis): ...` | `feat/op-synthesis-echarts` | [.ai/PLAN_SYNTHESIS_GO_PORTAGE.md](PLAN_SYNTHESIS_GO_PORTAGE.md) + chunks P3.B/D pour le pattern (cf. [thought_log](thought_log.md)) |
| `[R]` | OP | **P4.B — Squad Contributions Plotly → ECharts** : 2 charts dans `SquadContributionsPage.tsx`. À auditer en début de chunk pour identifier le bon wrapper (probablement Bar ou TimeseriesLine). | idem + commit `feat(squad-contributions): ...` | `feat/op-squad-contributions-echarts` | [.ai/PLAN_SQUAD_GO_PORTAGE.md](PLAN_SQUAD_GO_PORTAGE.md) |
| `[R]` | OP | **P4.C — Squad Synergies Plotly → ECharts** : 5 charts dans `SquadSynergiesPage.tsx` + helpers `features/squad/charts/{heatmapChart,hsPkChart,timelineChart}.ts`. Ces helpers **construisent du Plotly server-side** — il faut migrer aussi côté Go. Le plus gros chunk de la liste. | idem + commit `feat(squad-synergies): ...` | `feat/op-squad-synergies-echarts` | [.ai/PLAN_SQUAD_GO_PORTAGE.md](PLAN_SQUAD_GO_PORTAGE.md) + [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) (non-committé, lecture utile) |
| `[R]` | OP | **P4.D — Final cleanup Plotly** : retirer `apps/web/src/components/ui/plotly-chart.tsx`, retirer `react-plotly.js` de `apps/web/package.json`, vérifier qu'il ne reste rien (`grep -r react-plotly apps/web/src`). | tests OK + commit `cleanup(plotly): ...` | même branche que P4.C | bouchonner après P4.C |

> **Suggestion d'ordre** : P4.A en premier (le plus simple, valide ton workflow). Puis P4.B. P4.C est lourd (5 charts + portage Go) — si tu te retrouves en milieu de C avec peu de temps, **commit WIP** et laisse-moi le finir. P4.D = trivial après C.

## Backlog (si P4.A→D liquidé, attaque dans l'ordre)

| St | Owner | Chunk | Effort | Ref |
|----|-------|-------|--------|-----|
| `[R]` | OP | **Squad/Sessions overhaul** §1-3 livrés (Go multi-sessions + SessionMultiSelect + SquadLayout + 19 tests). Plan §3-6 (Q29 friends, recompute, notifs) = backlog séparé. | lourd | [.ai/PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md](PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md) — branche `feat/op-squad-sessions-multiselect` |
| `[R]` | OP | **Form score intra-match** — réflexion en cours | moyen | [.ai/REFLEXION_FORM_SCORE_INTRA_MATCH.md](REFLEXION_FORM_SCORE_INTRA_MATCH.md) — exploratoire |
| `[ ]` | OP | **Finition multi-titres** — labels/capabilities restants | moyen | [.ai/PLAN_FINITION_MULTI_TITLE.md](PLAN_FINITION_MULTI_TITLE.md) |

## Blocked / décisions à trancher (review queue — **pas en autonomie**)

- **Garder ou pas le composant `OutcomeSequenceTape` actuel ?** — design valide en interne mais pas confirmé en prod. À discuter à mon retour.
- **Squad Synergies (P4.C) implique de la migration Go** : si OP arrive là, mieux vaut **commit WIP** et me laisser finir — c'est cross-stack et nécessite un audit du contrat API.
- **Date de retour de GS à confirmer** quand on aura un planning précis.

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
