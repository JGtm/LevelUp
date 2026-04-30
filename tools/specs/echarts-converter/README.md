# `@levelup/echarts-converter`

Convertit les fichiers YAML de spec graphique (`.ai/charts_specs/<page>/*.yaml`) en
options ECharts (`option` à passer à `echarts.init().setOption()`).

Sert de **preuve de reproductibilité** : si le converter produit une option ECharts conforme
à la version Plotly d'origine, c'est que le format YAML est suffisant pour la migration.

## Pourquoi cet outil

La migration LevelUp Python+Streamlit → Go+React+ECharts nécessite que les graphes restent
visuellement et fonctionnellement identiques. Le YAML décrit chaque chart de façon
indépendante du framework de rendu ; le converter prouve que le YAML couvre tous les
détails dont a besoin ECharts.

## Usage

```bash
cd tools/specs/echarts-converter
npm install

# Convertir un YAML en ECharts option (stdout)
npm run convert -- ../../../.ai/charts_specs/synthesis/05_solo_squad_duel.yaml

# Convertir tous les YAML d'une page
npm run convert -- --all ../../../.ai/charts_specs/synthesis/

# Avec un dataset mock
npm run convert -- ../../../.ai/charts_specs/synthesis/05_solo_squad_duel.yaml \
                   --data fixtures/synthesis_05_mock.json
```

## Sortie

Pour `XX_chart.yaml`, génère `_generated/XX_chart.option.json` (option ECharts complète,
prête à `setOption()`).

## Quand le converter émet un warning

- Champ YAML reconnu mais non encore implémenté → warning + `option` partielle.
- Champ inconnu → warning + ignoré.
- Manque de données mock pour les valeurs runtime → warning + valeurs synthétiques.

Tout warning indique soit un manque dans le converter, soit un manque dans le schéma YAML
à enrichir. Liste des warnings = liste des points à arbitrer.

## Limites connues

- **Pas de phase runtime Python** : les `expression` du YAML (ex. `height: max(320, 70*len)`)
  sont évaluées avec des hypothèses (ex. `len(metrics) = 6`) plutôt qu'avec les vraies
  valeurs d'exécution. Quand un env Python sera dispo, on pourra ajouter une phase qui
  capture les vraies valeurs via `fig.to_dict()` et les injecte ici.
- **Pas de rendu visuel** : on émet `option` JSON, pas de screenshot. La validation visuelle
  (comparaison Plotly vs ECharts) se fait à part, soit dans le navigateur via `apps/web/`,
  soit avec un Puppeteer headless en complément.
