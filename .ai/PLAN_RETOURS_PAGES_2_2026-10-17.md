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
- [ ] Q2.2 Sessions et Séries temporelles : Objectif, bandeau d'objectifs.
- [ ] Q2.3 Anneau des frags (`FragSunburst` et cartes qui le portent) sans frag : bloc affiché,
      « Aucun frag sur ce périmètre » (réutiliser la clé `frags.toml` si elle convient).
- [ ] Q2.4 Inventaire des autres blocs retirés pour « aucun match concerné » sur Escouade, Sessions,
      Séries temporelles, Synthèse, Vue match : tableau au rapport ; corriger ceux de ces pages
      selon la même règle. Ne pas toucher aux blocs absents par capability du titre (fonction
      absente).

### Q3 — Libellés de la Synthèse
- [ ] Q3.1 `SynthesisTopWeeksChart` et autres noms de séries / légendes en dur de
      `features/synthesis` : clés i18n FR/EN, FR sans anglicisme (« Taux de victoire »).

### Q4 — Clôture
- [ ] Q4.1 `rm -rf node_modules/.tmp && npx tsc -b --force`, vitest complet, `npm run lint`,
      `lint:colors`, `lint:fields`, knip, lint des imports croisés ; Go si touché (paquets + archlint).
- [ ] Q4.2 Entrée `.ai/thought_log.md`, découvertes au §3.
- [ ] Q4.3 Un commit par étape sur `feat/retours-pages-2` ; pas de push ; disk-hygiene du lot.

## 3. Découvertes

(à remplir)
