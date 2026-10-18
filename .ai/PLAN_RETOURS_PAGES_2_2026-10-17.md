# Plan — états vides « aucun match concerné » + libellés de la Synthèse (2026-10-17)

Contrat : skill `plan-execution` (fait foi). Branche `feat/retours-pages-2`, worktree
`../LevelUp-wt-retours-escouade`, base `c7decf35b` (feat/v75). Plans clos de référence :
`.ai/PLAN_RETOURS_ESCOUADE_2026-10-17.md`, `.ai/PLAN_RETOURS_PAGES_2026-10-17.md`.

## 0. Décisions de l'utilisateur (2026-10-17)

- Blocs aujourd'hui retirés quand aucun match du périmètre n'est concerné (Objectif de Sessions et
  des Séries temporelles, bandeau d'objectifs, anneau des frags sans frag) : « En un sens oui, sur
  la page Escouade normalement on a décidé d'afficher un placeholder mais avec un message qui dit
  que la session n'a pas de match pour ce bloc ou un truc comme ça, vérifie. »
- Noms de séries en dur et non traduits de la Synthèse (« Top 1 », « Autres », « Taux Top (%) »,
  « Win % ») : « oui corrige aussi ».
- Règles déjà posées le même jour : manque de données = bloc affiché + état vide conforme (bloc
  parent + `EmptyStateNotice` dedans) ; textes factuels FR/EN, sans personne ni impératif, sans
  action impossible ni mot interne ; en FR pas d'anglicisme (« Taux de victoire », jamais « Win % »).

## 1. Constats à vérifier sur pièces avant de coder

- Précédent « placeholder avec message » trouvé dans l'Escouade : `features/squad/emprise/HabitCard.tsx`
  / `habit.logic.ts:37` (`kind: 'empty'`) avec `empriseStrings.ts:282`
  (« Aucune prise » / « Aucune prise de ressource sur les matchs filmés des soirées de la
  composition. ») ; manifests `first_blood.toml:12` (« Aucun premier frag ni première mort sur ce
  périmètre »), `frags.toml:35` (« Aucun frag sur ce périmètre »).
- Contre-exemple dans l'Escouade même : `features/squad/objectif/SquadObjectiveSection.tsx:14` et
  `SquadPreviousSessionsSection.tsx:10` RETIRENT les sections d'objectif sans match à objectif.
- E1 : chercher dans `.ai/thought_log.md`, `.ai/V7.5/` et les plans Escouade / Emprise / Objectif
  (grep « objectif », « placeholder », « bloc placeholder », « aucun match ») la décision exacte
  rappelée par l'utilisateur, et la citer (date, fichier, ligne). Si introuvable, le dire ; la
  règle s'applique quand même (décision du jour).

## 2. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non traité (justification).

### Q1 — Règle et forme
- [x] Q1.1 (décision retrouvée : mémoire utilisateur `feedback_etats_vides_fonction_vs_donnees.md`, retours du 2026-10-17 : « il faut des placeholders, sinon on pense à un bug » ; capability absente = bloc absent, données absentes = bloc AFFICHÉ avec titre et état vide ; + `.ai/PLAN_RETOURS_ESCOUADE_2026-10-17.md` D4/E5, qui l'a appliquée à l'Emprise ; aucun texte de l'utilisateur ne vise l'Objectif nommément avant celui du jour. Forme retenue : intertitre gardé + bloc parent `Card` + `EmptyStateNotice` dedans, constat « Aucun <objet> sur ce périmètre ») Rapporter la décision trouvée (E1) et la forme de référence retenue (titre gardé, bloc
      parent, `EmptyStateNotice` « Aucun <objet> sur ce périmètre » ou équivalent par page).
- [x] Q1.2 (`components/ui/scope-empty.tsx` : `ScopeEmptyState` / `ScopeEmptyCard` / `ScopeEmptySection` ; textes `components/ui/scope-empty-text.ts` ; garde `scopeEmpty.guard.test.ts` ; textes ajoutés à `textesSansPersonne.test.ts`) Si la même forme sert à 3+ endroits : composant ou helper partagé + garde-rail (règle
      n°6) — réutiliser `features/_shared/filmGap/` ou son voisin si pertinent.

### Q2 — Blocs « aucun match concerné »
- [x] Q2.1 (`objectif.logic.ts objectiveExpected` ; `SquadObjectiveSection.tsx` : deux sections gardées avec `ScopeEmptyCard` ; `SquadPreviousSessionsSection.tsx` : `ObjectifFrame` + `ScopeEmptyState` ; prédicat d'onglet `empriseTabHasContent` inchangé : l'onglet est visible dès que le titre lit le film, et le constat ne s'affiche que sur une page qui a du contenu ; capability absente ou lecture en échec : bloc retiré) Escouade : Objectif et Répartition de l'objectif (onglet Emprise), « Rapport de force,
      soirée après soirée » (Soirées précédentes) : bloc affiché avec « Aucun match à objectif sur
      ce périmètre » (FR) / EN équivalent, au lieu d'être retiré. Le prédicat d'onglet
      (`empriseTabHasContent`) reste cohérent.
- [x] Q2.2 (Sessions : `sessionEmprise.logic.ts` `objectiveEmpty` + `sessionScopeEmpties`, carte d'ancrage `objective_balance` dans `useSessionEmpriseCards.tsx` ; Séries temporelles : `usages.logic.ts objectiveEmpty` + `ScopeEmptySection` dans `TimeseriesPage.usages.tsx`, affiché seulement si l'onglet a autre chose à montrer ; bandeau : `TimeseriesObjectiveCard.tsx` constat `objectiveActions`) Sessions et Séries temporelles : Objectif, bandeau d'objectifs.
- [x] Q2.3 (`FragSunburst` : sans frag, carte (ou bloc nu) avec `EmptyStateNotice`, texte `scopeEmptyText('frags')` = clé `frags.empty.no_data` ; Sessions `sessionScopeEmpties` ; Synthèse : retrait du `fragDistribution &&` ; Séries temporelles : le résumé montait déjà l'anneau sans condition ; Vue match : `MatchFragCard` garde la carte A et `MatchViewTabArsenal` garde « Frags et armes » dès que l'onglet montre du terrain ; Escouade : `SquadFragBreakdownCard` avait déjà son état vide dans la carte) Anneau des frags (`FragSunburst` et cartes qui le portent) sans frag : bloc affiché,
      « Aucun frag sur ce périmètre » (réutiliser la clé `frags.toml` si elle convient).
- [x] Q2.4 (tableau au §4 ; corrigés : Escouade Mécaniques de frag / Points d'impact / Appui et portée, Sessions XP et outils, Séries temporelles XP, Synthèse Détails des frags et Objectifs, Vue match Répartition des frags) Inventaire des autres blocs retirés pour « aucun match concerné » sur Escouade, Sessions,
      Séries temporelles, Synthèse, Vue match : tableau au rapport ; corriger ceux de ces pages
      selon la même règle. Ne pas toucher aux blocs absents par capability du titre (fonction
      absente).

### Q3 — Libellés de la Synthèse
- [x] Q3.1 (manifeste `synthesis.charts.top_weeks_*` + `topWeeksNames` ; `SynthesisBipolaireChart` : `synthesis.scope.solo_label/squad_label` ; `SynthesisOutcomesByGroupChart` : repli de l'axe `common.charts.axis_matches` ; garde `seriesNames.guard.test.ts`) `SynthesisTopWeeksChart` et autres noms de séries / légendes en dur de
      `features/synthesis` : clés i18n FR/EN, FR sans anglicisme (« Taux de victoire »).

### Q4 — Clôture
- [x] Q4.1 (tsc -b --force, vitest 969 fichiers / 10058 tests, eslint 0 erreur, lint:colors, lint:fields, knip 0/0/0, imports croisés 7<=7, parité Go de la clé de capability ; aucun Go modifié)  `rm -rf node_modules/.tmp && npx tsc -b --force`, vitest complet, `npm run lint`,
      `lint:colors`, `lint:fields`, knip, lint des imports croisés ; Go si touché (paquets + archlint).
- [x] Q4.2 (entrée du 2026-10-17 en tête de thought_log.md ; §3 rempli) Entrée `.ai/thought_log.md`, découvertes au §3.
- [x] Q4.3 (cinq commits Q1, Q2.1, Q2.2, Q2.3, Q2.4, Q3 + clôture ; aucun push ; disk-hygiene -Lot retours-pages-2 -Apply) Un commit par étape sur `feat/retours-pages-2` ; pas de push ; disk-hygiene du lot.

## 3. Découvertes (non traitées, hors périmètre)

- Décision d'origine : aucune phrase de l'utilisateur ne vise « l'Objectif » nommément avant celle du 2026-10-17 ; la règle retrouvée est la mémoire `feedback_etats_vides_fonction_vs_donnees.md` (retours du 2026-10-17) et `.ai/PLAN_RETOURS_ESCOUADE_2026-10-17.md` D4/E5 (Emprise). Les sections d'objectif de l'Escouade (`SquadObjectiveSection`, `SquadPreviousSessionsSection`) étaient donc le contre-exemple écrit avant la règle.
- Lecture en échec (`formes_retenues.unavailable_reason = load_failed`) : le bloc d'objectif se retire sans rien dire à l'écran (une erreur de lecture n'est pas « aucun match à objectif »). Un état « lecture en échec » commun à l'Emprise, aux Séries temporelles et aux Sessions reste à décider. [Traité le 2026-10-18 : `.ai/PLAN_RETOURS_FIN_2026-10-18.md` R1.]
- Escouade › Emprise › Rapport de force par soirée : les notes « sous le minimum » / « première soirée » (`objectifStrings.ts`) disent encore « la carte se masque » alors que la carte reste affichée (texte antérieur au lot du 17/10).
- `squad_impact_history` absent de la réponse (titre sans événements horodatés, équipe alliée non lue, lecture en échec) : la carte « Points d'impact par soirée et par rôle » reste absente faute de signal distinguant capability et donnée (le Go renvoie `nil` pour les deux). [Traité le 2026-10-18 : `.ai/PLAN_RETOURS_FIN_2026-10-18.md` R2, capability `match.events.timeline`.]
- Synthèse : tuiles de KPI individuelles à zéro (incidents, séries de victoires ou de défaites, véhicules, détournements, tirs) restent absentes (une tuile, pas un bloc titré) ; `SynthesisPage.test.tsx` garde 14 tests ignorés sur l'ancienne interface.
- Vue match : « Objectifs » (propre au mode du match : un match d'un mode sans objectif n'a pas ce bloc), courbes et événements de score (mode non pris en charge), `MatchAssistChart` et positions (film non lu, signalé par `analysis_pending_parts`) : non modifiés ; à confirmer avec l'utilisateur si un mode sans objectif doit lui aussi dire « aucun match à objectif ».
- Titre du graphe des meilleures semaines : « Matchs Top vs Total par semaine » devient « Victoires et autres matchs, par semaine » (les séries ne s'appellent plus « Top 1 » / « Autres »).
- Nouvelle clé de capability de données consommée par le front : `analytics.career_xp_estimate` (`DATA_CAPABILITIES`, parité Go vérifiée).

## 4. Inventaire Q2.4 (blocs retirés pour « aucun match concerné »)

Capability absente (fonction absente du titre) : le bloc reste absent.

| Page | Bloc | Avant | Après (FR) |
|---|---|---|---|
| Escouade › Emprise | Objectif ; Répartition de l'objectif dans l'escouade | retirés (intertitres compris) | intertitres gardés + « Aucun match à objectif sur ce périmètre » (Q2.1) |
| Escouade › Emprise | Rapport de force, soirée après soirée | carte retirée | carte titrée + « Aucun match à objectif sur ce périmètre » (Q2.1) |
| Escouade › Contributions | Mécaniques de frag natives (capability présente) | carte retirée sans donnée | carte titrée + « Aucune donnée / Aucune donnée pour cette sélection » |
| Escouade › Contributions | Points d'impact par soirée et par rôle | carte retirée sans soirée | carte titrée + état vide du graphe |
| Escouade › Synergies | Appui et portée | section retirée sans appui ni portée | titre qui lit le film : intertitre gardé + « Analyse du film en cours » / « Film non analysé » / « Aucune assistance ni portée de frag sur ce périmètre » ; titre sans film : absente |
| Sessions | Objectif (rapport de force, fiche) | retiré | intertitre « Objectif » + « Aucun match à objectif sur ce périmètre » (Q2.2) |
| Sessions | Répartition des frags ; Outils de destruction | cartes retirées sans frag | cartes titrées + « Aucun frag sur ce périmètre » / état vide du graphe (Q2.3) |
| Sessions | XP de carrière estimée | carte retirée sans estimation | titre avec la capability `analytics.career_xp_estimate` : carte titrée + état vide ; sans : absente |
| Séries temporelles › Usages | Objectif | retiré | intertitre + « Aucun match à objectif sur ce périmètre » (Q2.2), seulement si l'onglet montre autre chose |
| Séries temporelles | Bandeau Objectifs | retiré | titre gardé + « Aucune action d'objectif sur ce périmètre » (Q2.2) ; capability `objective_stats` absente : absent |
| Séries temporelles › Résumé | Répartition des frags (anneau) | rien (`null`) | carte titrée + « Aucun frag sur ce périmètre » (Q2.3) |
| Séries temporelles › Progression | XP de carrière estimée | carte retirée sans match estimé | capability : carte titrée + état vide ; sans : absente |
| Synthèse | Répartition des frags (anneau) | retiré sans distribution | carte titrée + « Aucun frag sur ce périmètre » (Q2.3) |
| Synthèse | Détails des frags | retiré sans distribution | carte titrée + état vide du graphe |
| Synthèse | Objectifs (KPI) | retiré sans stats | titre gardé + « Aucun match à objectif sur ce périmètre » (pas de stats) ou « Aucune action d'objectif sur ce périmètre » (stats à zéro) ; capability absente : absent |
| Vue match › Armes et terrain | Répartition des frags | carte retirée sans frag | carte titrée + « Aucun frag sur ce périmètre » ; « Frags et armes » reste dès que l'onglet montre du terrain (Q2.3) |

Examinés, non modifiés (justification) :
- Capability du titre absente : FDA attendu, dégâts subis, MMR d'équipe, précision par arme native, engagement, bloc de coordination, portée par arme, mécaniques de frag natives (parent `FeatureGate`).
- Graphes qui retournent `null` dans leur constructeur d'option (Séries temporelles : `Timeseries*Trend`, `*Density`, `TimeseriesFormCharts`, `TimeseriesSquadAdapted`, `Timeseries ScatterWithTrend`) : leur `ChartCard` affiche déjà titre + état vide.
- Escouade › Dynamique et Contributions (hors cartes ci-dessus), Sessions (graphes de `_chartSections`), Synthèse (heatmap, semaines, résultats par carte) : cartes toujours montées avec état vide (plans clos du jour).
- Synthèse : tuiles de KPI individuelles à zéro (incidents, séries, véhicules, détournements) : une tuile, pas un bloc titré (voir §3).
- Vue match : « Objectifs » (propre au mode du match), courbes et événements de score (mode non pris en charge), `MatchAssistChart` et positions (film, signal `analysis_pending_parts`) : voir §3.
- Escouade › `SquadObjectivesPanel` (objectifs d'escouade à atteindre) : sans lien avec les matchs du périmètre.
