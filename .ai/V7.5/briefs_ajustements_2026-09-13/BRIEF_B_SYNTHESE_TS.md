# LOT B — Synthèse + Timeseries

Worktree : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-ajust-synthese-ts` · branche `feat/ajust-synthese-ts` · port Vite **5182**.
Lis d'abord `BRIEF_COMMUN.md` (même dossier). Ce lot touche probablement le Go (contrat de la page Timeseries) : skills `arch-rules` + `go-features`, GOCACHE privé.

## Repérage déjà fait (vérifie sur pièces)

- Synthèse : query `features/synthesis/queries.ts:10-35` (`POST /pages/synthesis`). Page `features/synthesis/SynthesisPage.tsx`.
- « Meilleures stats » = `SynthesisPage.tsx:200-280` (8 AccentCard en dur). Les six entrées visées sont dans le sous-bloc « Objectifs » `SynthesisPage.tsx:443-477` (grille `grid-cols-3`) : clés `synthesis.kpi.flag_returns`, `flag_steals`, `flag_carrier_time`, `zone_secures`, `zone_time`, `skull_carrier_time` (`lib/i18n/manifests/synthesis.toml:396-426`).
- « Portée des engagements » : `features/synthesis/SynthesisWeaponRangeSection.tsx` (section `:369-430`), montée `SynthesisPage.tsx:516` sous `useCapability('weapon_range')` (`:178`), donnée `SynthesisPageResponse.weapon_range`. Textes : `synthesis.weapon_range.heading` (toml `:453-455`), `lede` (`:457-459`, rendu `:395-397`), `card_title` (`:461-463`, rendu `:402-403`), `card_count` (`:465-467`, rendu `:404-414`), `range_subtitle`/`_detail` (`:469-475`, rendu `:329-332` via `SubtitleRow` + `InfoTooltip`), `elevation_subtitle`/`_detail` (`:477-483`, rendu `:346-349`). Tuiles de tête `RangeTiles :92-152`.
- « Usages d'équipement » : `features/_shared/usage/EquipmentUsageSection.tsx` (export `:199-214`, `EquipmentCard :134-175`, `PadControlCard :177-197`), monté `SynthesisPage.tsx:522` avec `mode="solo"`, donnée `SynthesisPageResponse.equipment_usage`. Libellés `features/_shared/usage/usageI18n.ts` : `blockEquipment :183`, `viewEquipmentPartsSolo :270`, `blockPadControl :184`, `viewWeaponPartsSolo :272`, **`segEnemy :214` (« Eux (anonyme) ») / `:315` EN**. Ce composant est AUSSI monté sur Escouade (lot voisin D1 fait le même découpage en deux blocs côté Escouade) : fais le découpage DANS le composant partagé, en gardant `mode`, pour que les deux pages en profitent — le superviseur fusionnera.
- « Activité par jour et heure » : `features/synthesis/SynthesisHeatmapChart.tsx`, monté `SynthesisPage.tsx:825`. Dernier changement de format : commit `7568a7bd2` (« la heatmap de Synthese passe par le wrapper canonique ») — AVANT : heatmap ECharts écrite dans le fichier (`buildHeatmapOption(cells, locale)`), `yAxis.inverse:true` (Lundi en haut), titres d'axes, `visualMap` vertical à droite (`orient:'vertical', right:30, itemWidth:12, itemHeight:140`, formatteur %). APRÈS : `Heatmap2DChart` mode divergent, légende horizontale en pied, plus de titres d'axes. Le fichier a été retiré de l'allowlist de `heatmapSingleImpl.guard.test.ts`.
- Timeseries : `features/timeseries/TimeseriesPage.tsx` (onglets `summary | distributions | progression`, `:39-45`, corps `:167-199`). Progression : `TimeseriesPage.progression.tsx` — « Premier frag / première mort » `:107-123` (grille `lg:grid-cols-2`, gauche `FirstBloodLanes` `components/charts/FirstBloodLanes.tsx:129-175`, droite `TimeseriesPerMinuteTrend`) ; « Engagement » + « Écart d'engagement cumulé » `:218-236`, pleine largeur, enfants directs du `space-y-8` dans un `FeatureGate capability="engagement"` ; composants `features/engagement/EngagementTimeseriesSection.tsx` et `features/timeseries/TimeseriesEngagementGapTrend.tsx`, même query `useEngagementTimeseries`.
- Summary : `TimeseriesPage.summary.tsx` importe déjà `SynthesisWeaponAccuracyChart` depuis `features/synthesis` (`:24`, import cross-feature toléré par `tools/lint-cross-feature-imports.mjs`) ; sunburst / outils de destruction `:220-238`, précision `:245-255`.
- Contrat Go de la page Timeseries : cherche le handler `pages/timeseries` (`internal/api/handlers/`), le service et le DTO `TimeseriesPageResponse` ; `weapon_range` et `equipment_usage` sont aujourd'hui calculés pour la page Synthèse (trouve le service qui les produit et RÉUTILISE-le : même scope de filtres).

## Items

### B.1 — Retirer six cartes d'objectif
Supprime les six cartes (Retours de drapeau, Vols de drapeau, Temps porteur (drapeau), Zones sécurisées, Temps en zone, Temps porteur (crâne)) de `SynthesisPage.tsx:443-477`, leurs clés TOML (FR+EN) et tout test qui les asservit. Si le sous-bloc « Objectifs » devient vide, supprime-le entièrement (titre, gate `hasObjectiveStats`, champ front devenu inutile — si le champ Go n'est plus lu par PERSONNE, retire-le du DTO aussi ; vérifie par grep avant). Ne touche pas aux mêmes libellés sur Timeseries/Escouade/Match view.

### B.2 — Migration vers Timeseries
- « Portée des engagements » → onglet **Synthèse** de Timeseries (pendant de la précision par arme), inséré en pleine largeur entre le bloc sunburst (`:220-238`) et le bloc précision (`:245`).
- « Usages d'équipement » → onglet **Progression**, après le `FeatureGate engagement` (`:236`) et avant `TimeseriesIntensityProfile` (`:240`), `mode="solo"`.
- Retire les deux montages de `SynthesisPage.tsx` (`:516`, `:522`) et tout ce qui ne servait qu'à eux sur Synthèse (imports, gates, champs de réponse si plus lus).
- Données : expose `weapon_range` et `equipment_usage` sur la réponse de la page Timeseries (Go : même producteur que Synthèse, même filtres/scope, capability `weapon_range` respectée → champ omis si non supporté ; régénère `openapi.yaml` + `make generate-types` ; tests Go du handler/service). Pas de query front séparée si le DTO de page suffit.
- Capability : conserve le gate `useCapability('weapon_range')` côté Timeseries.

### B.3 — Retouches de « Portée des engagements » (après migration)
- Supprime la phrase `lede` (« À quelle distance vous fraguez… ») : rendu + clés FR/EN.
- `card_title` devient « Portée par arme » / « Range by weapon ».
- Supprime `card_count` (« 32 armes · 6 146 frags… ») : rendu + clés.
- Supprime le sous-titre « Portée » et son (i) (`range_subtitle`, `range_subtitle_detail`) ; le (i) (même texte de détail) passe sur le TITRE du bloc « Portée par arme » (regarde comment `SectionCard` porte un `InfoTooltip` dans le titre ailleurs, ex. `cardTitleAdornment` de `EquipmentUsageSection.tsx:62-71`).
- « Dénivelé » devient SON PROPRE bloc (`SectionCard` distinct, titre « Dénivelé » avec son (i) `elevation_subtitle_detail`), sur la même rangée que « Portée par arme » si la largeur le permet (grille `lg:grid-cols-2`), sinon en dessous — regarde le rendu et choisis ce qui reste lisible (le graphe de portée est large : si les étiquettes d'armes deviennent illisibles en demi-largeur, empile). Dis ce que tu as choisi et pourquoi.
- Les tuiles de tête (`RangeTiles`) restent au-dessus.

### B.4 — Usages d'équipement en blocs distincts (composant partagé)
- « Usages d'équipement » (barres) et « Ma part de l'équipement du lobby » (donut) : deux `SectionCard` distincts, sur la même rangée (`grid lg:grid-cols-2`). Idem « Contrôle des armes spéciales » / « Ma part des armes spéciales du lobby ». Les sous-titres `ViewTitle` disparaissent (ils deviennent les titres des cartes). Conserve `mode` solo/squad (« Notre part … » en squad).
- `segEnemy` : « Eux (anonyme) » → « Équipe adverse » / « Opposing team ». Vérifie que le donut le rend et que la doctrine (adversaire agrégé, jamais nommé) tient.
- Le garde-fou `noLocalUsageCopies.guard.test.ts` doit rester vert.

### B.5 — « Activité par jour et heure » : restaurer le format précédent
Objectif : le rendu d'AVANT `7568a7bd2` (Lundi en haut, titres d'axes, `visualMap` vertical à droite avec %, tooltip identique). Deux voies, choisis la plus propre et dis-le : (a) étendre `Heatmap2DChart` avec les options manquantes (`yAxisInverse`, `axisNames`, `visualMapOrient:'vertical'`) — préférable si ≤ 40 lignes et sans casser les autres consommateurs (tests du wrapper) ; (b) revenir à un builder local (alors ré-ajouter le fichier à l'allowlist de `heatmapSingleImpl.guard.test.ts` avec justification datée). Compare ta capture APRÈS avec le rendu d'avant (tu peux `git show 7568a7bd2^:apps/web/src/features/synthesis/SynthesisHeatmapChart.tsx` pour lire l'ancien builder).

### B.6 — « Premier frag / première mort » centré en hauteur
Dans la rangée `lg:grid-cols-2` (`progression.tsx:107`), la carte de gauche est collée en haut. Fais en sorte que son contenu soit centré verticalement dans la hauteur de la rangée (par ex. `items-center` sur la grille, ou `h-full` + `flex flex-col justify-center` sur la carte). Vérifie sur capture aux deux largeurs (1440 et 1024).

### B.7 — « Écart d'engagement cumulé » à droite de « Engagement »
Les deux dans une grille `grid grid-cols-1 gap-4 lg:grid-cols-2` à l'intérieur du `FeatureGate`, Engagement à gauche, Écart cumulé à droite. Vérifie que les deux graphes restent lisibles en demi-largeur (capture).

Captures APRÈS : Synthèse (page entière), Timeseries onglets Synthèse et Progression.
