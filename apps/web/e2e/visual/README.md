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

## Exécution — mode démo (recommandé, DÉTERMINISTE)

Une seule commande : elle génère la fixture démo synthétique dans une racine de
données ISOLÉE (`tests/fixtures/demo-root`, jamais le `data/` du poste), démarre
l'API en mode démo dessus + Vite, joue le harnais sur `demo-player`, puis arrête
les serveurs. Ne pas lancer `make dev` en parallèle (mêmes ports).

```bash
make demo-visual                                  # comparer aux baselines
make demo-visual ARGS="--update-snapshots"        # (re)générer les baselines
make demo-visual ARGS="--verify-determinism"      # preuve : 2 générations, 0 diff
bash scripts/demo-visual-harness.sh --skip-seed   # réutiliser la fixture existante
```

Le générateur (`levelup seed-demo --synthetic`, `internal/ops/seed_demo_synthetic.go`)
est à graine et date d'ancrage FIXES : 60 matchs, 5 sessions sur ~3 semaines,
4 playlists, 3 joueurs (`DemoPlayer` + `DemoPlayer2`/`DemoPlayer3`). Deux
générations produisent les mêmes lignes — d'où zéro diff pixel entre deux passes.
Les fichiers `.duckdb` diffèrent au bit près (agencement de blocs, `written_at`
`DEFAULT now()`) : c'est normal, aucune donnée lue par l'app n'en dépend.

La fixture (~59 Mo) est GÉNÉRÉE, jamais versionnée. Compter ~20 s de génération
et ~1,5 min de captures.

**Le mode démo est hermétique au réseau.** L'API démo n'émet aucun appel tiers
(Halo, Xbox, gamecms, OAuth Microsoft) : `internal/platform/netguard` coupe les
sorties dès `LEVELUP_DEMO_MODE=true`, et chaque saut est tracé
(`demo mode: external fetch skipped surface=…`). Sans cela, une démo lancée sur
un poste porteur de vrais tokens s'authentifie et interroge l'API Halo pour les
xuid FACTICES de la fixture — 400/404 avec 4 tentatives, ~12 s par appel — ce qui
affame le chargement des pages : les captures partaient alors en timeout à un run
et en « données absentes » au suivant. Vérification rapide après un run :

```bash
grep -c halowaypoint tests/demo-harness-logs/demo-api.log   # doit afficher 0
```

## Exécution — sur les données locales du poste

Prérequis : `make dev` (API :8000 + Vite :5173).

```bash
cd apps/web
npx playwright test --project=visual                      # comparer aux baselines
npx playwright test --project=visual --update-snapshots   # (re)générer
E2E_VISUAL_PLAYER=JGtm npx playwright test --project=visual   # viser un joueur peuplé
```

À réserver aux cas où l'on veut voir un rendu sur données réelles : ces captures
dérivent d'une passe à l'autre dès qu'une synchro tourne (c'est ce qui a motivé le
mode démo). Une page sans graphe (données absentes) SKIPPE au lieu d'échouer ;
`/lab/charts` fonctionne toujours.

### Pages Escouade : la sélection est un état UI, pas une donnée

`squad/synergies` et `squad/dynamique` n'affichent aucun graphe tant qu'aucun
coéquipier n'est confirmé — la sélection vit dans
`localStorage['squad-teammates-<playerSlug>']`. `prepareVisualPage`
(`_helpers/visual.ts`) l'amorce avant navigation ; le défaut ne vaut que pour
`demo-player`. Pour un joueur réel, nommer ses coéquipiers :

```bash
E2E_VISUAL_PLAYER=JGtm E2E_VISUAL_SQUAD_TEAMMATES="Gt1,Gt2" \
  npx playwright test --project=visual
```

Sans cette variable, les deux pages Escouade skippent sur un joueur réel.

## Points d'attention

- **`--update-snapshots` n'écrit QUE les baselines de pages applicatives.** Le
  projet `visual` embarque aussi `lab-charts`, dont les baselines sont
  VERSIONNÉES : `scripts/demo-visual-harness.sh` restreint donc toute régénération
  à `app-pages.visual.spec.ts`. Régénérer la vitrine se fait explicitement :
  `npx playwright test --project=visual lab-charts --update-snapshots`.
- **Drift connu de la baseline `lab` (constaté 2026-08-04).** Sur le poste de
  développement Windows courant, `lab-charts` échoue de façon REPRODUCTIBLE à
  6018 pixels (0,01 %) contre la baseline committée : celle-ci a été générée dans
  un état de rendu antérieur (police / version d'ECharts / OS). L'écart est stable
  au pixel près d'un run à l'autre — ce n'est pas de l'instabilité, c'est une
  baseline à regénérer volontairement, dans un lot dédié qui statue sur la diff.
  En attendant, `make demo-visual` sort en 1 pour cette seule raison alors que les
  7 pages applicatives sont vertes.
- **Projet séparé, hors CI.** Le projet `visual` est exclu du projet `chromium` que
  lance la CI : les baselines PNG sont liées à la plateforme de génération (le
  suffixe `-win32` / `-linux` du nom de fichier). Étendre le gate à la CI suppose de
  générer les baselines sur le runner ubuntu.
- **Animation ECharts.** `animations: 'disabled'` de Playwright ne couvre que
  CSS/Web Animations. L'attente de fin de rendu canvas est faite par
  `waitForChartsSettled` (`_helpers/visual.ts`).
- **Seuils serrés** (`threshold` 0.15, `maxDiffPixelRatio` 0.002) : une diff doit
  être analysée, pas absorbée en élargissant la tolérance.
- **Bandeau d'accueil masqué.** `HomeHeroBanner` tire son visuel AU HASARD parmi 7
  images statiques et en change toutes les 45 s. Le `Math.random` seedé ne le fige
  pas (le tirage dépend du nombre d'appels consommés avant son montage, qui varie
  avec l'ordre d'arrivée des réponses réseau) : il valait 8 % de pixels différents
  entre deux passes sur données identiques. Il est donc `mask`é dans la capture de
  page — élément purement décoratif, aucune perte de couverture.
- **Un chart n'égale pas un canvas.** Depuis `echarts` 6.1.0, `Heatmap2DChart` est
  rendu sur 3 couches zrender (3 `<canvas>` pour une seule instance) contre 1 en
  5.6.0. Les captures indexées des charts qui SUIVENT une heatmap dans le DOM sont
  donc décalées de +2 : avant de conclure à une régression sur `...-canvas-07`,
  vérifier qu'on compare bien le même graphe.
