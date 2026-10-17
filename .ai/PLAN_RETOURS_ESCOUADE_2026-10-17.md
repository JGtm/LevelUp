# Plan — retours du 2026-10-17 sur l'Escouade (et le bloc Médias de la Vue match)

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/retours-escouade`,
worktree `../LevelUp-wt-retours-escouade`, base `ef74ce9ac` (origin/feat/v75). Lot jumeau en
parallèle : `feat/retours-tactique` (Tactique), aucun fichier commun attendu hors
`.ai/thought_log.md`.

## 0. Décisions de l'utilisateur (messages du 2026-10-17)

- **D1 Infobulles** : « elles se superposent sur les barres ou quittent l'écran et suivent la
  souris, impossible de voir la vue d'ensemble ». Puis : « elle doit pouvoir sortir du cadre
  mais pas de la page web. Et des fois il y a trop d'infos dedans, des données mal formatées ou
  mal ordonnées. »
- **D2 « Contrôle des ressources, par soirée »** : barres « quasi transparentes, seuls les bords
  colorés, l'intérieur blanc ». « Pourquoi colorer autrement les barres des soirées
  précédentes ? Je n'ai jamais demandé ça. Et quand on les survole elles changent de couleur, je
  ne veux pas ça. »
- **D3 « Points d'impact par soirée et par rôle »** : « les couleurs piquent les yeux, ne suivent
  pas les thèmes de couleurs de la majorité de l'app, ça détonne ». Accord sur : teintes plus
  douces, alignées sur le vert / rouge du reste de l'app, en clair et en sombre.
- **D4 Emprise sans film analysé** : « il devrait y avoir au minimum des placeholders » ; « pour
  la session d'aujourd'hui je viens de voir apparaître les sections Ressources, Prises par
  joueur, Par match. Il faut des placeholders, sinon on pense à un bug ». « Respecte bien la
  convention de l'app pour ce type de placeholder. »
- **D5 Bloc Médias (Vue match)** : « on n'a pas le bloc parent, directement le bloc en
  pointillé ; ce n'était pas comme ça avant ».

## 1. Constats sur pièces (2026-10-17)

- Infobulles : réglage commun `getTooltipBase` (`apps/web/src/components/charts/_utils.ts:68`),
  sans `confine`, `position`, `appendToBody`. Tous les graphes ECharts d'Escouade passent par
  lui ; il sert aussi Vue match, Sessions, Carrière, Explorateur. Seul `confine: true` du dépôt :
  `features/squad/impact/impactHistoryChart.ts:214`. Bulles DOM : `components/ui/tooltip.tsx`.
- « Par soirée » : `features/squad/emprise/habitChart.ts:88` — soirée non comparable =
  `{ color: c.muted, borderColor: color, borderWidth: 1.5 }` ; presque toutes les soirées le sont.
- Impact : `lib/accessibility/palettes/_impactRoleColors.ts` (`IMPACT_GREEN_RED`, light + dark),
  garde `lib/accessibility/impactRoleTokens.test.ts` (contrastes décroissants, pas pâle ≥ 2:1,
  ΔE voisins ≥ 15 normal + daltonismes, teinte ≤ 12° de `divergent-pos/neg`) et
  `app/providers/theme-provider-impact.test.tsx`.
- Emprise : un bloc sans donnée se retire (`features/squad/emprise/empriseContent.ts`,
  `SquadEmprisePage.tsx`). Cause réelle du 17/10 : films des 5 matchs du jour non cuits (verrou
  de décodage tenu par une passe de parc), donc pas de résumé d'usage → blocs absents.
  Signal existant à réutiliser : `analysis_pending` / `analysis_pending_parts` (commits
  `047b97f4c`, `597d7adfc`, `ad2e1dec2`) et `components/ui/analysis-pending-slot.tsx`.
- Convention d'état vide de l'app : bloc parent (`Card` + `CardContent`, ou la carte de section
  du bloc) avec `EmptyStateNotice` (pointillé) DEDANS — ex. `features/explorer/ExplorerPage.playerMode.tsx:108`,
  `features/home/HomeBattlePassPanel.tsx:120`. Médias : `features/match-view/MatchMediaTab.tsx:52`
  rend l'`EmptyStateNotice` nu sous `DetailSection` (régression `943669b60`, 2026-10-14).

## 2. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non traité (justification).

### E1 — Infobulles : jamais hors de la page (D1)
- [x] E1.1 (fait : `components/charts/tooltipPosition.ts` + `getTooltipBase`) Dans `getTooltipBase` : la bulle peut déborder du cadre du graphe mais reste dans la
      fenêtre (ex. `appendToBody: true` + `position` fonction qui borne au viewport, bascule
      au-dessus/au-dessous et à gauche/droite du point pour ne pas masquer la marque survolée),
      `transitionDuration: 0`. Fonction de position pure, testée (bords de fenêtre).
- [x] E1.2 (fait, `confine` retire) Retirer le `confine: true` local d'`impactHistoryChart.ts` s'il contredit E1.1.
- [x] E1.3 (FragWeaponBreakdown, WeaponAccuracyChart, ActivityCalendarChart, TimeseriesKdaDensity ne servent pas l Escouade : notes en Decouvertes ; heatmap2DOption `position: top` retire car Heatmap2DChart sert l Escouade) Les tooltips recopiés à la main hors helper dans `components/charts/` (ex.
      `FragWeaponBreakdown.tsx:68`) : les rebrancher sur le helper s'ils servent l'Escouade ;
      sinon noter en découverte.
- Gate : vitest des charts + `make check-types`.

### E2 — Contenu des infobulles de l'Escouade (D1)
- [x] E2.1 (tableau au rapport et au journal) Inventaire : pour CHAQUE graphe des onglets Synergies, Contributions, Dynamique,
      Emprise (ECharts et bulles DOM), relever le contenu actuel de la bulle (lignes, ordre,
      formats). Tableau dans le rapport.
- [x] E2.2 (`components/charts/axisTooltip.ts`, builders migres) Corriger : une bulle = l'élément survolé + ses valeurs utiles, rien de redondant avec
      l'axe ; nombres au format de la locale (séparateurs, %, décimales cohérentes avec le
      graphe) ; lignes dans l'ordre visuel du graphe (empilement / légende) ; bulle des points
      d'impact raccourcie. Textes FR et EN, factuels, sans personne (garde
      `textesSansPersonne.test.ts`).
- Gate : tests des builders touchés mis à jour ; vitest squad.

### E3 — « Par soirée » : barres pleines, sans changement au survol (D2)
- [x] E3.1 `habitChart.ts` : toutes les barres pleines, couleur de leur ressource, quelle que
      soit la soirée. Plus aucun style « hors comparaison » (gris, liseré).
- [x] E3.2 (emphasis.disabled + legendHoverLink false) Le survol ne change pas la couleur des barres (emphase neutralisée sur ce graphe).
- [x] E3.3 (muted, notComparable et son entree de legende retires ; info FR/EN reecrite) La non-comparabilité reste : hors médiane, et mentionnée dans l'infobulle seulement.
      Retirer le code mort (couleur `muted` si plus lue) et mettre à jour les tests.

### E4 — Couleurs des rôles d'impact (D3)
- [ ] E4.1 Nouvelles rampes `IMPACT_GREEN_RED` (light + dark) plus douces : pas de vert
      quasi noir ni fluo ; ancrées sur `divergent-pos` / `divergent-neg` de la palette par
      défaut. Toutes les contraintes d'`impactRoleTokens.test.ts` tenues sans relâcher un seuil.
      Les autres palettes (Okabe-Ito, Cividis, Tol) : inchangées sauf si un test l'exige.
- [ ] E4.2 Rapport : valeurs avant/après, contrastes et ΔE mesurés.

### E5 — Emprise : placeholders au lieu de sections absentes (D4)
- [ ] E5.1 Règle : capability du titre absente (Halo 5 sans film) = section absente (pas la
      fonction) ; capability présente mais données du film manquantes sur le périmètre =
      section AFFICHÉE avec son intertitre et un état vide conforme à la convention (bloc
      parent + `EmptyStateNotice` dedans, ou `AnalysisPendingSlot` dans le bloc parent quand
      l'analyse est en cours).
- [ ] E5.2 Signal : le bloc `squad_emprise` doit dire, pour son périmètre, combien de matchs
      ont une analyse du film EN COURS et combien n'en ont pas. Réutiliser la règle unique
      d'`analysis_pending_parts` côté Go (pas de seconde règle) ; contrat openapi +
      `make generate-types`.
- [ ] E5.3 Libellés FR/EN : « Analyse du film en cours » / « Film analysis in progress » ;
      « Film non analysé » / « Film not analysed » (+ description factuelle courte). Sections
      concernées : toutes celles de l'onglet Emprise qui lisent le film (Bilan / Ressources,
      Prises par joueur, Par match, Prendre et s'en servir, Groupés ou isolés).
- [ ] E5.4 L'onglet Emprise reste visible dès que le titre a la capability (plus masqué faute
      de données) ; `empriseContent.ts` reste le prédicat unique page + barre d'onglets.
- Gate : vitest squad + tests Go du paquet touché (tag integration si persist/sync touché).

### E6 — Bloc Médias : bloc parent rétabli (D5)
- [ ] E6.1 `MatchMediaTab.tsx` : état vide = bloc parent (même gabarit que les autres blocs
      vides de la page) avec l'`EmptyStateNotice` dedans. Test mis à jour.

### E7 — Clôture
- [ ] E7.1 `make check-types`, `make test-web` (vitest hors sandbox), lint web, `go test` des
      paquets Go touchés, `make go-api-lint` si Go touché.
- [ ] E7.2 Entrée `.ai/thought_log.md` ; découvertes hors périmètre listées au rapport (non
      traitées).
- [ ] E7.3 Commits sur `feat/retours-escouade` uniquement, un par étape ; pas de push.

## 3. Hors périmètre

Tactique (lot jumeau) ; autres pages que l'Escouade pour le contenu des infobulles (E2) ;
autres onglets d'Escouade que l'Emprise pour les placeholders (à noter s'ils ont le même défaut).

## 4. Decouvertes (non traitees, hors perimetre)

- E1.3 : quatre infobulles ECharts recopiees hors de `getTooltipBase` ne servent pas l Escouade et gardent le placement par defaut (donc encore susceptibles de deborder de la fenetre) : `components/charts/FragWeaponBreakdown.tsx:68`, `components/charts/WeaponAccuracyChart.tsx:70` (Synthese, Series temporelles), `features/ascension/ActivityCalendarChart.tsx:115`, `features/timeseries/TimeseriesKdaDensity.tsx:192`.
- E2 : `features/squad/charts/heatmapChart.ts` (+ test) n est importe par aucun composant (code mort probable). `components/charts/OutcomeSequenceTape.tsx` : la ligne "+N" des series longues n a pas de mot. `features/squad/v2/SquadEngagementView.tsx` et `components/charts/EngagementCurve.tsx` : nom d axe Y en anglais en dur ("events / min (auto-zoom ...)"). `components/charts/heatmap2DOption.ts` : texte par defaut "Win Rate: / Matchs:" en dur (non utilise par l Escouade, qui passe formatTooltip).
