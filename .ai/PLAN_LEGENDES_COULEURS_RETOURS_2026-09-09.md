# PLAN — Légendes de graphes : couleurs, nomenclature, textes (retours utilisateur 2026-09-09)

> Statut : CLOS (E1 → E7) · Branche : `feat/v75` · Périmètre : `apps/web` uniquement.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'action
> exécutable, chaque item statué).

## Cause racine (vérifiée sur pièces)

Rendu ECharts 6.1 hors navigateur (`echarts.init(null, null, {ssr:true, renderer:'svg'})`,
sonde jetable) :

- Série `bar` dont la couleur ne vit QUE sur les points (`data: [{value, itemStyle:{color}}]`)
  et pas au niveau série → la pastille de légende est peinte avec la **palette PAR DÉFAUT
  d'ECharts** (`#5070dd`, `#b6d634`, `#505372`, `#ff994d`, `#0ca8df`, `#ffd10a`, `#fb628b`,
  `#785db0`, `#3fbe95`), aucune de ces couleurs n'étant dans le graphe.
- Une entrée `legend.data` sous forme d'OBJET portant `itemStyle.color` / `lineStyle.color`
  reprend la main : la couleur par défaut disparaît du rendu.
- Une série `line` avec `lineStyle.color` explicite est déjà correcte (le trait de l'icône
  suit) — d'où les graphes où l'utilisateur dit « les couleurs sont bonnes ».

Corollaire : le correctif générique est de faire porter la couleur à l'ENTRÉE DE LÉGENDE,
pas seulement à la série.

## Décisions utilisateur (2026-09-09, fermes)

- D1 — « Portée par arme » : l'axe des armes perd le suffixe `×n/n` (les effectifs restent
  dans l'infobulle et le tableau). Les graduations de distance RESTENT.
- D2 — « Dénivelé » / « Portée » : le chrome de la carte INTERNE du graphe est supprimé
  (cadre, fond, arrondi) ; le graphe se pose directement dans la carte de section.
- D3 — Solo « FDA » : deux entrées de légende `FDA ≥ 1` (vert) et `FDA < 1` (rouge), plus
  la tendance. Chacune masquable.

---

## E1 — Socle : couleur explicite d'entrée de légende

- [x] `components/charts/_utils.ts` : `legendEntries(...)` — fabrique les entrées
      `legend.data` avec `itemStyle`/`lineStyle` explicites (+ `dashed`).
- [x] `LEGEND_ITEM_WIDTH_LINE` : largeur de pastille lisible pour un trait pointillé.
- [x] Test `_utils.test.ts` : couleur portée, pointillé, ordre, liste vide. (29 tests verts)

Le garde-rail de rendu vit en E6 : il ne peut passer qu'une fois les builders migrés.

Gate : `npm run test:run -- src/components/charts/_utils.test.ts`

## E2 — Page Solo (`features/timeseries`)

- [x] « FDA » — deux séries seuil (vert ≥ 1 / rouge < 1) + tendance, légende à couleurs
      explicites, infobulle qui n'affiche que la barre présente.
- [x] « Écart cumulé au FDA attendu » — courbe pointillée `width: 1 → 2`, pastille de
      légende élargie.
- [x] « Durée de vie moyenne » — hauteur alignée sur sa voisine « Assistances » (plus de
      graphe ancré en haut d'une carte trop grande).
- [x] « Performance solo par mois » — pastille élargie + couleurs explicites (le pointillé
      « MMR équipe » devient identifiable).
- [x] « Taux de victoire — Session vs Historique » — couleurs explicites + `tokenCssVar`
      remplacé par `resolveToken` (une CSS var n'est pas peignable dans un canvas).
- [x] « Performance par carte — Session vs Historique » — couleurs explicites.

Gate : `npm run test:run -- src/features/timeseries src/features/squad/charts/winRateVsHistoryBulletChart.test.ts src/features/squad/charts/mapPerfVsHistoryChart.test.ts`

## E3 — Sessions (`features/session-detail`)

- [x] « Écart cumulé au FDA attendu » — même retouche d'épaisseur qu'en E2.
- [x] « Rendement · Résistance » — couleurs explicites + socle de légende commun.

Gate : `npm run test:run -- src/features/session-detail`

## E4 — Synthèse (`features/synthesis`)

- [x] Encre des morts : `chart-series-3` (couleur des assistances) → `outcome-loss`
      (convention frags/morts du reste de l'app). Le dénivelé garde ses encres propres.
- [x] Légende de portée centrée sous le graphe, sans les mentions de position.
- [x] Légende de dénivelé centrée sous le graphe.
- [x] Détails de lecture (`— bâton du 10e au 90e centile…`, `— d'où vous avez fragué…`)
      déplacés dans une infobulle ⓘ à côté du titre de bloc.
- [x] D1 — suffixe `×n/n` retiré des libellés d'axe.
- [x] Pseudo-arme « Chute et environnement » retirée de la section (la distance n'a pas de
      sens sur cette donnée).
- [x] Barres de dénivelé épaissies.
- [x] Pied de carte retiré (seuil + note de couverture).
- [x] D2 — chrome de la carte interne supprimé.

Gate : `npm run test:run -- src/features/synthesis`

## E5 — Escouade (`features/squad`)

- [x] Sous-charts performance (Assistances, Précision, FDA, Durée de vie, Performance,
      Folie meurtrière) — couleurs explicites par joueur.
- [x] « Rang & MMR équipe » — couleurs explicites (CSR/LUSR par joueur + MMR pointillé).
- [x] « Stats par minute » — légende rétablie.
- [x] « Rendement » / « Résistance » — légende standard en pied de graphe à la place des
      étiquettes de fin de courbe.
- [x] « Isolement et couverture » — légende ramenée au socle commun ; texte de plancher et
      de définition déplacé dans une infobulle ⓘ.
- [x] « Frags / Morts » — dernière phrase (« clique Bonus… ») retirée de l'aide.
- [x] « Délai d'échange » — phrase narrative et note de couverture retirées ; définition et
      fenêtre déplacées dans une infobulle ⓘ.
- [x] Taux de victoire / Performance par carte : couverts par E2 (mêmes builders).

Gate : `npm run test:run -- src/features/squad`

## E6 — Garde-rail de rendu

- [x] `legendPalette.guard.test.ts` : rendu SSR (`echarts.init(null, null, {ssr:true})`) des
      builders migrés ; aucune couleur de `tokens.color.theme` (palette par défaut ECharts,
      lue depuis le paquet — pas recopiée) ne doit apparaître dans le SVG produit.

Gate : `npm run test:run -- src/components/charts/legendPalette.guard.test.ts`

## E7 — Rendement / Résistance : reformulation des aides

Choix utilisateur (2026-09-09) : variante 3 pour les deux, MOINS sa dernière phrase
(« L'échelle ne bouge pas d'une session à l'autre ») et sans le verbe « concluent »,
remplacé par « portent » — le mot déjà employé par la variante 1.

- [x] `features/squad/i18n.ts` : `help` unique découpé en `rendementHelp` / `resistanceHelp`
      (FR + EN), l'ancien texte de 120 mots retiré.
- [x] Escouade : chaque carte porte SON aide ⓘ.
- [x] Match view : `MatchVsStatCard` sait porter une aide (prop `help`, absente = rendu
      inchangé) ; les deux tuiles la reçoivent. Texte identique MOINS la mention des zones de
      fond — il n'y a pas de zones sur une tuile KPI.
- [x] Accueil : aide COMBINÉE sur la tuile « Rendement / Résist. », qui porte les deux
      indicateurs sous un libellé abrégé.
- [x] Tests : une aide par carte (Escouade + Match view, l'échange des deux textes échoue),
      aide combinée sur l'accueil, et aucune icône ajoutée aux tuiles qui n'en veulent pas.

Gate : `npm run test:run -- src/features/squad src/features/match-view src/features/home`

## Gate global

`npm run typecheck` · `npm run lint` · `node ../../tools/lint-no-hardcoded-colors.mjs` ·
`npm run test:run`

---

## Journal

### 2026-09-09 — E1 à E4

Cause racine établie par rendu SSR AVANT toute modification (cf. section « Cause racine ») :
la palette par défaut d'ECharts apparaît dans le rendu dès qu'une série ne porte pas de
couleur au niveau série. Le correctif est donc porté par l'ENTRÉE de légende, pas par la
série — un seul helper, `legendEntries`.

E2/E3 — deux retouches n'étaient PAS des couleurs et ont été traitées comme telles :
l'épaisseur du pointillé « FDA attendu » (1 → 2) et la hauteur de « Durée de vie moyenne »
(240 → 320, alignée sur sa voisine de rangée). Sur les deux graphes d'écart cumulé, les
couleurs de légende n'ont VOLONTAIREMENT pas été reprises en main : l'utilisateur les dit
bonnes, et les deux séries portent déjà leur `lineStyle.color` (dégradé divergent / encre de
série) — seule la largeur de pastille change, pour que le tireté se lise.

E4 — le côté « morts » de la portée par arme empruntait `chart-series-3`, qui est l'encre des
ASSISTANCES ailleurs dans l'app ; il passe à `outcome-loss`. La rampe du DÉNIVELÉ garde ses
encres propres (`chart-series-1/3` + gris d'axe) : elle dit d'où part le tir, pas qui tue qui,
et un rouge « d'en haut » se lirait comme un jugement. Suppressions : le suffixe `×n/n` des
étiquettes d'axe (D1), la pseudo-arme « Chute et environnement », la ligne « sous le seuil »
et la note de couverture — avec leurs clés i18n et `belowThresholdNames`, devenus morts.

### 2026-09-09 — E5, E6

E5 — au-delà des couleurs, deux graphes changeaient de langage : « Rendement » / « Résistance »
portaient l'identité des joueurs par une ÉTIQUETTE DE FIN de courbe (canal qu'aucun autre
graphe n'emploie, et qui ne se clique pas), « Isolement et couverture » posait sa propre mise
en forme de légende. Les deux passent au socle commun. Trois textes descendent en infobulle ⓘ
(planchers de l'isolement, définition + fenêtre du délai d'échange) et la phrase narrative du
délai disparaît — avec sa clé i18n `squad.echange.delay_narrative` et son accesseur, sans quoi
le garde-rail `squadEchange.i18n.test.ts` (« aucun accesseur que rien n'affiche ») mordait.

E6 — le garde-rail rend chaque graphe migré hors navigateur et refuse toute couleur de
`tokens.color.theme` (lue dans le paquet ECharts, jamais recopiée). Vérifié qu'il MORD :
en remettant `legend.data` en chaînes nues sur « Stats par minute », les deux tests de ce
graphe échouent ; le retour à `legendEntries` les repasse au vert. Un trait d'épaisseur nulle
est ignoré — zrender écrit un `stroke` sur toute forme, y compris invisible (zones de fond
`markArea`), et le compter accusait « Performance » pour une couleur que personne ne voit.

### Vérification

Rien n'a été vérifié À L'ÉCRAN : le serveur de dev local est derrière l'authentification Xbox,
et je ne saisis pas d'identifiants. Les preuves sont le rendu SSR (E6) et la suite complète.

Gates passés dans la session : `npm run test:run` (659 fichiers, 7041 tests verts),
`npm run typecheck`, `npx eslint src` (0 erreur, 29 avertissements PRÉEXISTANTS dont 2
directives inutiles dans `features/admin/*`, hors périmètre), `lint-no-hardcoded-colors`
(0 violation), `lint-no-hardcoded-fields` (0 violation).

### Découvertes (non traitées — règle 7)

- `winRateVsHistoryBulletChart` peignait ses barres « historique » avec
  `tokenCssVar('chart-series-1')`, c.-à-d. la chaîne `var(--ac-chart-series-1)` : un canvas
  ne résout pas les variables CSS. Corrigé ici parce que le graphe est au périmètre.
- Plusieurs autres graphes hors périmètre présentent la même cause racine que E1
  (`TimeseriesPerformanceTrend`, `TimeseriesSpreeHeadshots`, `SessionFdaBars`…). Non
  traités : hors de la liste de l'utilisateur.
