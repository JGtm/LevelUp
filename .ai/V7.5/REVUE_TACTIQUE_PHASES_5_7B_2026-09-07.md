# Revue adversariale — Chantier Tactique, Phase 5 (vue d'analyse) et Phase 7B (nuage isolement x couverture)

Contexte frais, lecture seule, feat/v75. Vérification systématique sur pièces au HEAD
actuel (les commits cités datent du 2026-09-07, HEAD n'a pas bougé depuis sur ces
fichiers sauf mention contraire).

Périmètre vérifié :
- Phase 5 : `git diff 0df2aa226 bf5d465bd -- apps/web` (17 fichiers, feature `tactical/`)
- Phase 7B : `git diff bf5d465bd c868e2b1d` (22 fichiers, Go `service/teammates` +
  `domain/squad_isolement.go` + web `features/squad/`)

Tests ciblés exécutés (lecture seule, pas de `go test ./...` ni vitest complet) :
- `npx vitest run src/features/tactical/TacticalAnalysisView.test.tsx src/features/tactical/tacticalView.logic.test.ts` → 27/27 verts
- `npx vitest run src/features/squad/SquadIsolementNuageCard.test.tsx src/features/squad/squadIsolement.logic.test.ts` → 18/18 verts
- `go test ./internal/service/teammates/... -run Isolement -v` → 2/2 verts

Points explicitement cités dans le contrat de revue et VÉRIFIÉS SANS DÉFAUT (pour
mémoire, aucun constat à leur sujet) : plancher `TACTICAL_CELL_FLOOR=3` (client) contre
`tactical.PlancherMatchsParCellule=3` (serveur, `apps/go-api/internal/analysis/tactical/merge.go:16`)
— alignés ; ordre des bandeaux « en attente » / « non disponible » — non inversé
(`TacticalAnalysisView.tsx:146-147`, `tacticalView.logic.ts:60-61`) ; taille de bulle du
nuage — plafonnée à 30 px et planchée à 6 px (`squadIsolement.logic.ts:93-102`) ; session
sous 5 morts examinées — bien exclue côté Go (`teammates_squad_isolement.go:89-91`,
couvert par `TestBuildSquadIsolementNuage`, cas S2/S3/S4) ; alignement clic/cellule peinte
— la conversion pixel→monde (`cellFromClick`, `tacticalView.logic.ts:103-119`) et la
peinture (`drawTacticalHeatmap`, `heatPaint.ts:210-214`) partagent la même convention
canvas = exactement `[min_x,max_x]×[min_y,max_y]`, donc mathématiquement cohérentes même
après un redimensionnement fenêtre (le conteneur garde son `aspectRatio` CSS, la mise à
l'échelle du bitmap `canvas` reste uniforme) ; absence de dpr (`k:1`, `TacticalPlanCard.tsx:92`)
dégrade la netteté, pas l'alignement — déjà consigné en P2 assumé dans
`.ai/DECOUVERTES_TACTIQUE_2026-09-07.md` ligne 72-75, non re-signalé ici.

---

## P1

### P1-1 — KPI « Échange » et « Isolement » de l'onglet Tactique : aucune réserve d'échantillon affichée

- **Fichier** : `apps/web/src/features/tactical/TacticalAnalysisView.tsx:184-199` (fonction `buildKpiCards`)
- **Fait** : la tuile KPI « Échange après ma mort » et la tuile « Morts en isolement » sont
  construites uniquement à partir de `data.echange.taux` / `data.isolement.taux` +
  `kpiSecondary(brut, n)`. Le champ `echantillon_faible` que le contrat publie sur ces deux
  objets (`Couverture.echantillon_faible`, généré dans `apps/web/src/lib/api/generated.ts:5637`,
  exposé par le type `TacticalCouverture = components['schemas']['Couverture']`,
  `apps/web/src/lib/api/types.ts:3123`) n'est lu nulle part dans ce fichier ni dans
  `tacticalView.logic.ts`, ni dans `i18n.ts` (aucune clé `tactical.kpi.*low_sample*`
  n'existe dans `apps/web/src/lib/i18n/manifests/tactical.toml`).
- **Pourquoi c'est risqué** : le plan (`.ai/PLAN_TACTIQUE_2026-09-06.md` ligne 92,
  « Plancher d'échantillon | 30 morts... en dessous « échantillon faible » ») et la
  doctrine du paquet Go (`analysis/coordination/measure.go:8-11`, « le drapeau ne cache pas
  la valeur, il interdit de la comparer ») exigent que tout taux sous 30 événements soit
  marqué. Un pourcentage d'échange/isolement affiché sans réserve alors que la carte est
  juste au-dessus de son plancher d'entrée (10 matchs, souvent < 30 morts vengeables/
  examinées cumulées) se lit comme une mesure fiable alors qu'elle ne l'est pas. Le même
  diff (phase 7B, `SquadIsolementNuageCard`) applique correctement cette réserve pour la
  MÊME mesure d'échange — la divergence est interne au chantier, pas une contrainte
  externe manquante.
- **Correction minimale** : dans `buildKpiCards`, ajouter un indicateur (texte ou style)
  quand `data.echange?.echantillon_faible` / `data.isolement?.echantillon_faible` est vrai,
  sur le modèle de `SquadEchangeKpi.tsx:52-53`.
- **Comment le prouver** : test qui construit un `TacticalRaster` avec
  `echange: { taux: 1, brut: 3, n: 3, echantillon_faible: true, ... }` et vérifie qu'un
  texte/état « échantillon faible » apparaît dans la carte KPI — ce test échoue
  aujourd'hui (aucune assertion de ce type dans `TacticalAnalysisView.test.tsx`, la
  fixture `RASTER_NOMINAL` fixe `echantillon_faible: false` sur les deux objets, ligne
  76-77, donc le chemin vrai n'est jamais exercé).

### P1-2 — Nuage « isolement x couverture » (Escouade) : l'échantillon faible n'est signalé qu'en opacité, jamais en texte

- **Fichier** : `apps/web/src/features/squad/SquadIsolementNuageCard.tsx:210-223` (formatter
  de tooltip) et `apps/web/src/features/squad/squadIsolementStrings.ts:34`
- **Fait** : `pointAttenue`/`opaciteDuPoint` (`squadIsolement.logic.ts:84-90`) réduisent
  l'opacité d'un point ECharts (`0.35` vs `0.8`) quand `part_isolee.echantillon_faible`
  est vrai — seul encodage visuel. La chaîne `t.lowSample` (résolue depuis
  `squad.isolement.low_sample`, ajoutée dans `apps/web/src/lib/i18n/manifests/squad.toml`
  ligne ~825) est définie dans `squadIsolementStrings.ts:34` mais n'est référencée dans
  AUCUN composant : `grep "t\.lowSample\b" apps/web/src/features/squad/*.tsx` ne remonte
  que `SquadEchangeKpi.tsx:53` et `SquadEchangeMatrixCard.tsx:90`, jamais
  `SquadIsolementNuageCard.tsx`. Le tooltip (lignes 210-223) sert `isoRate`, `isoBrut`,
  `isoN`, `covRate`, `covBrut`, `covN` mais jamais un signal « échantillon faible ».
- **Pourquoi c'est risqué** : une opacité à 0.35 sur un fond de graphe n'est pas un
  signal fiable (contraste, daltonisme, petit point noyé dans le nuage) ; la doctrine du
  dépôt pour cette exacte mesure (même diff, cartes voisines `SquadEchangeKpi` /
  `SquadEchangeMatrixCard`) est un texte explicite « échantillon faible ». Un point à
  5-10 morts examinées (juste au-dessus du plancher de publication) peut se peindre en
  plein quadrant « loin et sans secours » (zone d'alerte visuelle, `warningColor`) sans
  qu'aucun texte n'indique que ce point n'a pas la robustesse statistique que sa position
  dans un quadrant nommé suggère.
- **Correction minimale** : ajouter la mention `t.lowSample` dans la chaîne
  `t.tooltip(...)` (ou dans un badge dédié) quand `p.part_isolee.echantillon_faible` (ou
  `p.couverture.echantillon_faible`) est vrai.
- **Comment le prouver** : test qui construit un point avec `echantillon_faible: true` et
  vérifie qu'un texte contenant « échantillon faible » apparaît (dans le DOM ou dans
  l'option ECharts construite) — `SquadIsolementNuageCard.test.tsx` ne teste aujourd'hui
  ni le tooltip ni l'opacité, seulement l'état vide, le rendu nominal, le pied de carte et
  la parité FR/EN (vérifié en relisant le fichier en entier, 96 lignes).

---

## P2

### P2-1 — Quatre clés i18n Phase 5 ajoutées puis jamais branchées (code mort)

- **Fichier** : `apps/web/src/features/tactical/i18n.ts:73-76`
- **Fait** : les getters `planOf` (`tactical.plan.title_of`), `footerHeatmap`
  (`tactical.plan.footer_heatmap`), `footerRoutes` (`tactical.plan.footer_routes`) et
  `footerEmpty` (`tactical.plan.footer_empty`) sont exportés dans l'objet `TacticalText`
  mais ne sont appelés par aucun composant (`grep -rn "\.planOf\b\|\.footerHeatmap\|\.footerRoutes\|\.footerEmpty" apps/web/src --include=*.tsx --include=*.ts`
  ne remonte que leur définition). Le titre H2 réel utilise `analysisPageTitle`
  (`i18n.ts:90-91`, `tactical.analysis.page_title`) et le pied du plan utilise seulement
  `unitForQuestion` + `footerFloor` + `sourceForQuestion` (`TacticalPlanCard.tsx:122-127`),
  jamais les quatre clés ci-dessus.
- **Pourquoi c'est un défaut** : CLAUDE.md règle « 0 code mort » — « ce qu'on débranche du
  routing/des callers, on le supprime avec ses tests et imports ». Ici il ne s'agit pas
  d'un débranchement mais d'un jamais-branché : quatre paires FR/EN (huit chaînes) plus
  quatre entrées dans `apps/web/src/lib/i18n/generated/tactical.ts` existent sans aucun
  consommateur, ce qui gonflera la surface de maintenance i18n (traduction, revue) sans
  valeur.
- **Correction minimale** : supprimer les quatre getters, leurs clés dans
  `tactical.toml`, et regénérer `lib/i18n/generated/tactical.ts` — ou les brancher si une
  légende de rampe détaillée / un texte de routes était réellement prévu pour item 5.4
  (le plan §Phase 5 ligne 862-865 ne mentionne que unité + plancher + source au pied du
  plan, pas de quatrième texte).
- **Comment le prouver** : après suppression, `npm run lint` (garde i18n éventuel) et
  `tsc` doivent rester verts ; un test qui importe `getTacticalText('fr')` et vérifie
  l'absence des clés confirmerait la suppression, mais la preuve la plus directe est le
  grep ci-dessus, déjà à zéro résultat côté consommateurs.

---

## Décompte

- P0 : 0
- P1 : 2
- P2 : 1

## Verdict (3 lignes)

Aucun défaut P0 trouvé sur ces deux phases : les invariants explicitement ciblés par le
contrat de revue (plancher cellule, ordre des bandeaux, taille de bulle, plancher de
session, alignement clic/peinture) tiennent tous à la vérification sur pièces. Les deux
P1 relèvent du même angle mort — une réserve d'échantillon (« échantillon faible »)
documentée et déjà implémentée ailleurs dans le même chantier, mais oubliée dans les
tuiles KPI de la vue d'analyse Tactique et dans le nuage Escouade — donc livrables après
un correctif ciblé et de faible risque (pas de changement de contrat, pur affichage) ;
en l'état, les deux phases sont fonctionnellement correctes mais présentent par endroits
des taux sans la mise en garde que la doctrine du projet exige, et devraient être
corrigées avant merge plutôt que consignées en dette.
