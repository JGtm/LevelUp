# Plan — deux placeholders restants : lecture de l'objectif en échec, points d'impact absents (2026-10-18)

Contrat : skill `plan-execution` (fait foi). Branche `feat/retours-fin`, worktree
`../LevelUp-wt-retours-fin`, base `a63814647` (feat/v75). Plans de référence :
`.ai/PLAN_RETOURS_PAGES_2_2026-10-17.md` (§3 découvertes : ces deux points), règle « fonction absente =
bloc absent, donnée absente = bloc affiché avec titre et constat ».

## 0. Décision de l'utilisateur (2026-10-18)

« Mets-moi des placeholders pour ces deux points ouverts » :

1. Lecture de l'objectif en échec (`formes_retenues.unavailable_reason = load_failed`) : le bloc se retire
   aujourd'hui sans rien dire. Bloc AFFICHÉ (titre + bloc parent + `EmptyStateNotice`) avec un constat
   factuel (« Lecture de l'objectif indisponible »). Forme commune à 3+ endroits : helper partagé + garde-rail.
2. « Points d'impact par soirée et par rôle » (Escouade › Contributions) absent quand la réponse n'a pas
   `squad_impact_history` : distinguer titre sans la fonction (carte absente) et titre qui l'a mais historique
   absent ou en échec (carte AFFICHÉE avec un constat). Journaliser côté Go toute dégradation.

Textes FR et EN factuels : ni « vous/tu/mon », ni impératif, ni action impossible, ni mot interne, ni anglicisme.

## 1. Constats vérifiés sur pièces avant de coder

- `load_failed` est posé par `squadagg/squad_formes.go:78` (lecture du résumé d'usage en échec, journalisé en
  Error) ; `unsupported` par `:62` (dépôt absent). Consommateurs de `objectiveExpected` : Escouade
  (`SquadObjectiveSection`, `SquadPreviousSessionsSection`), Sessions (`sessionEmprise.logic`), Séries
  temporelles (`usages.logic`) : quatre sites, deux types de constat. Synthèse et bandeau d'objectifs des
  Séries temporelles lisent les agrégats d'objectif, pas `formes_retenues` : non concernés.
- `squad_impact_history` nil : `buildSquadImpact` (`teammates_squad_impact.go`) mêle sans sélection, sans
  soirée, aucun rôle attribué (événements absents ou lecture Q32 en échec, déjà journalisée en Warn) ; le cas
  « aucun rôle » n'est pas journalisé. Les rôles du barème (`squadimpact/roles.go`) sont presque tous lus de
  la timeline horodatée : la fonction est portée par la capability de données `match.events.timeline`
  (déclarée par les deux titres, clé déjà dans le vocabulaire Go).

## 2. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non traité (justification).

### R1 — Lecture de l'objectif en échec
- [x] R1.1 (kind renommé `objectiveUnavailable` : la garde `noUnknownMentions` interdit « lecture indisponible » / « reading unavailable » ; FR « Objectif indisponible pour ce périmètre », EN « Objective unavailable for this scope ») Kind `objectiveUnreadable` dans `components/ui/scope-empty-text.ts` (FR/EN), ajouté à
      `allScopeEmptyTexts` (garde « sans personne ») et aux phrases de `scopeEmpty.guard.test.ts`.
- [x] R1.2 (`objectif.logic.ts`) `objectiveExpected` remplacé par `objectiveGap(block)` (`objectif.logic.ts`) : le kind à afficher
      (`objective` / `objectiveUnreadable`) ou null (bloc absent : capability absente ou périmètre mesuré).
- [x] R1.3 (garde dans `scopeEmpty.guard.test.ts`) Quatre consommateurs migrés : `SquadObjectiveSection`, `SquadPreviousSessionsSection`,
      `sessionEmprise.logic` + `useSessionEmpriseCards`, `usages.logic` + `TimeseriesPage.usages`. Tests
      mis à jour (les cas `load_failed` affichent le constat) ; garde-rail : plus d'`objectiveExpected`.
- [x] R1.4 Gate : `tsc -b --force`, vitest complet, eslint, lint:colors, lint:fields, knip, imports croisés.
      Commit R1.

### R2 — Points d'impact par soirée et par rôle
- [x] R2.1 (clé `match.events.timeline` ajoutée à `DATA_CAPABILITIES`) Web : `SquadImpactHistoryCard` accepte un historique absent (titre gardé + `ChartCard` en état vide
      avec constat) ; `SquadContributionsPage` la monte si l'historique existe OU si le titre déclare
      `match.events.timeline` (`useDataCapability`) ; titre sans la capability : carte absente.
- [x] R2.2 (`teammates_squad_impact.go`, test `TestBuildSquadImpact_HistoriqueAbsentJournalise`) Go : journaliser (slog) le cas « historique absent » de `buildSquadImpact` avec sa raison (aucune
      soirée / aucun rôle attribué), test de la raison ; commentaire du contrat (`domain/teammates.go`) à jour.
- [x] R2.3 Gate web (idem R1.4) + Go : `go test` des paquets touchés + `./internal/archlint/...`, golangci-lint
      `--new-from-merge-base=origin/main`. Commit R2.

### R3 — Clôture
- [x] R3.1 Plans `PLAN_RETOURS_PAGES_2` §3 : les deux découvertes annotées (traitées par ce lot).
- [x] R3.2 Entrée `.ai/thought_log.md` du 2026-10-18 ; commit de clôture ; `disk-hygiene -Lot retours-fin -Apply`.

## 3. Découvertes (non traitées, hors périmètre)

- La garde `noUnknownMentions.guard.test.ts` (2026-10-09) interdit « lecture indisponible » / « reading unavailable » : l'exemple de l'utilisateur n'est pas reprenable tel quel ; formulation voisine retenue (R1.1).
- Historique d'impact : « aucun rôle attribué » regroupe le titre sans événement exploitable ET la lecture des événements en échec (journalisée en Warn, `teammates_impact_events_load_failed`) ; l'écran dit un seul constat. Distinguer les deux demanderait un champ de contrat Go (non fait : aucun besoin exprimé).
- `SquadEmprisePage.tsx` : sans section de film, sans écart de film et sans match à objectif, la page dit « film non décodé » avant d'atteindre les sections d'objectif, même si la lecture de l'objectif a échoué ; en pratique la même panne donne aussi `film_load_failed` (écart non nul), non vérifié sur un cas réel.
- Les notes « sous le minimum » / « première soirée » de `objectifStrings.ts` disent encore « la carte se masque » (découverte du plan pages-2, toujours ouverte).
- `unavailable_reason: 'unsupported'` est aussi posé quand le dépôt ou le xuid manque (`squadagg/squad_formes.go:62`), pas seulement pour une capability absente : le bloc se retire dans les deux cas (comportement inchangé).
