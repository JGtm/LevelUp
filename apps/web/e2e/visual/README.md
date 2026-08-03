# Régression visuelle des graphes (Playwright `toHaveScreenshot`)

Tous les graphes passent par ECharts et `vitest` mocke `echarts` partout : sans ce
harnais, aucun test n'exerce le rendu réel. Ce trou de couverture est ce qui a fait
fermer sans merger la PR Dependabot du bump `echarts` 5 -> 6 (correctif XSS
CVE-2026-45249) : rien ne permettait de mesurer l'effet d'un changement du moteur
de rendu de toute l'application.

## Deux périmètres, deux politiques de baseline

| Spec | Surface | Baselines |
|---|---|---|
| `lab-charts.visual.spec.ts` | `/lab/charts` — 11 wrappers, données statiques, sans backend | **versionnées** (`__screenshots__/lab/`) |
| `app-pages.visual.spec.ts` | 7 pages applicatives denses (home, timeseries, synthèse, squad x2, session, relations) | **non versionnées** (`__screenshots__/app/`, gitignoré) |

Les captures de pages applicatives dépendent des données locales et peuvent contenir
des gamertags réels : elles servent de référence AVANT/APRÈS pour une opération
donnée, pas de garde-rail partagé.

## Exécution

Prérequis : `make dev` (API :8000 + Vite :5173).

```bash
cd apps/web
npx playwright test --project=visual                      # comparer aux baselines
npx playwright test --project=visual --update-snapshots   # (re)générer
E2E_VISUAL_PLAYER=JGtm npx playwright test --project=visual   # viser un joueur peuplé
```

Une page sans graphe (données absentes) SKIPPE au lieu d'échouer. Le défaut
`demo-player` ne rend les pages applicatives que si la fixture démo est peuplée ;
`/lab/charts` fonctionne toujours.

## Points d'attention

- **Projet séparé, hors CI.** Le projet `visual` est exclu du projet `chromium` que
  lance la CI : les baselines PNG sont liées à la plateforme de génération (le
  suffixe `-win32` / `-linux` du nom de fichier). Étendre le gate à la CI suppose de
  générer les baselines sur le runner ubuntu.
- **Animation ECharts.** `animations: 'disabled'` de Playwright ne couvre que
  CSS/Web Animations. L'attente de fin de rendu canvas est faite par
  `waitForChartsSettled` (`_helpers/visual.ts`).
- **Seuils serrés** (`threshold` 0.15, `maxDiffPixelRatio` 0.002) : une diff doit
  être analysée, pas absorbée en élargissant la tolérance.
- **Un chart n'égale pas un canvas.** Depuis `echarts` 6.1.0, `Heatmap2DChart` est
  rendu sur 3 couches zrender (3 `<canvas>` pour une seule instance) contre 1 en
  5.6.0. Les captures indexées des charts qui SUIVENT une heatmap dans le DOM sont
  donc décalées de +2 : avant de conclure à une régression sur `...-canvas-07`,
  vérifier qu'on compare bien le même graphe.
