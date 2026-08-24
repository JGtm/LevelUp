# PLAN — Temps mort par joueur (rejeu 2D, calcul côté web)

Date : 2026-08-24. Origine : REGISTRE_REPORTS ligne « lots B et P clos » (le temps mort se
mesure SANS ti=5, par les trajectoires du document : médiane 8-10 s par mort, 865-1 136 s
cumulés par match) + backlog Notion « Rejeu 2D — bilan du 18/08/2026 ». Go utilisateur :
2026-08-24 (« pas d'application immédiate, mais autant l'avoir en cas de besoin »).

Branche : `wt/temps-mort-web` (worktree dédié, base feat/v75). Exécution sous le contrat du
skill `plan-execution`.

## Objectif et critère de succès

Calculer le temps mort cumulé PAR JOUEUR côté web, depuis les vies du document de rejeu déjà
servi (AUCUN bump de schéma, AUCUNE re-cuisson — la re-cuisson de masse est de toute façon
bloquée par la bombe RAM consignée au registre), et l'afficher dans la fiche joueur du rejeu.

Succès = la fiche joueur affiche « Temps mort mm:ss », valeurs plausibles sur les artefacts
témoins (médiane par mort de l'ordre de 8-10 s), tsc/vitest/lint verts, parité i18n FR/EN.

## Décisions TRANCHÉES (ne pas rouvrir en cours de lot)

1. Définition : temps mort d'un joueur = somme des intervalles entre la FIN d'une vie
   (mort) et le DÉBUT de la vie suivante du même joueur. PAS d'intervalle de tête (avant la
   première vie) ni de queue (après la dernière vie : un joueur vivant à la fin ou parti du
   match n'accumule rien). Les vies viennent de la normalisation existante du document
   (tracks/vies par joueur) — vérifier l'existant AVANT d'écrire : la structure des vies
   par joueur existe déjà côté web (elle sert la vitalité, les croix de mort, fireMark).
2. Surface : la fiche joueur du rejeu (là où vivent score/frags/morts/assistances), variante
   pleine OBLIGATOIRE. Variante compacte : seulement si la rangée existante l'accepte sans
   casser la parité de rangées (piège consigné au registre : fiche morte en compact,
   rangée fusionnée min-h-[18px]) ; sinon pleine seulement, et le consigner au CR.
3. Format : mm:ss (pas de pourcentage, pas de graphique). Libellés : FR « Temps mort »,
   EN « Time dead » — dans les DEUX tables i18n du feature ET le contrat de clés
   (`i18nContract.ts`), même geste que les retraits du commit 08f6980b0 mais en ajout.
4. Agrégat d'équipe : HORS PÉRIMÈTRE (suite possible, à consigner en §Découvertes si une
   surface naturelle saute aux yeux — ne pas l'implémenter).
5. Pas d'interrupteur, pas d'option : la ligne s'affiche toujours (un match sans mort
   affiche 00:00). Aucun flag.

## Hors périmètre (fermé)

- Toute modification Go, schéma d'artefact, contrat OpenAPI, re-cuisson.
- `ReplayCanvas.tsx` : cliquet à 797 lignes — ne RIEN y ajouter ; la logique vit dans un
  module dédié, le rendu dans le composant de fiche existant.
- Agrégat d'équipe, page match hors rejeu, exports.

## Phase 0 — Logique pure + tests

- [x] 0.1 Vérifier l'existant : où les vies par joueur sont-elles déjà dérivées côté web
      (replayNormalize / rosterLogic / useReplayStaticLayers) ? RÉUTILISER cette dérivation,
      ne pas re-parser les tracks.
      VÉRIFIÉ SUR PIÈCES : `rosterLogic.buildPlayers(doc, scoreboard)` groupe les traces par
      xuid en `ReplayPlayer.lives` DÉJÀ TRIÉES par `trackWindow(l).start`, et
      `replayLogic.trackWindow` borne chaque vie (startFrame ?? 0, endFrame ?? t du dernier
      point). C'est la dérivation qui sert la vitalité, les croix de mort et fireMark : le
      module la consomme, il ne re-parse aucune track. `rosterLogic.ts` est à 499 L (seuil
      500) — d'où le module dédié, conformément au plan.
- [x] 0.2 Module `deadTimeLogic.ts` (features/match-replay) : `deadTimeByPlayer(...)` ->
      millisecondes cumulées par joueur, borné à la fenêtre du match.
      `deadTimeByPlayer(players, doc): Map<xuid, ms>` (conversion par `frameToMs`, donc par
      `frameIntervalMs` de l'artefact) + `formatDeadTime(ms): 'mm:ss'`. Tri défensif et
      COUVERTURE COURANTE (`covered`) plutôt que comparaison à la seule vie précédente : deux
      vies qui se chevauchent ne fabriquent pas de faux trou. Bornage `[0, frameCount-1]`.
- [x] 0.3 Tests unitaires (vitest) : joueur sans mort = 0 ; mort sans respawn (fin de match,
      abandon) = rien d'accumulé pour cet intervalle ; deux vies contiguës sans trou = 0 ;
      cas nominal multi-vies ; vies désordonnées en entrée (tri défensif).
      14 tests (`deadTimeLogic.test.ts`), dont en plus : pas d'intervalle de tête, vies
      chevauchantes, bornage `frameCount`, vie anonyme (sans xuid) qui n'entre chez personne,
      artefact sans `frameIntervalMs` (cadence de repli), et le format.

Gate 0 : `npx vitest run` sur les nouveaux tests, verts. Clore avant phase 1.
PASSÉ le 2026-08-24 : `npx vitest run src/features/match-replay/deadTimeLogic.test.ts` ->
1 fichier, 14 tests, 0 échec. `npx tsc -b` -> 0 erreur.

## Phase 1 — Affichage fiche joueur + i18n

- [ ] 1.1 Ligne « Temps mort » dans la fiche joueur pleine (composant de fiche existant —
      PAS ReplayCanvas), format mm:ss, libellé via les tables i18n du feature.
- [ ] 1.2 Clés ajoutées aux DEUX tables (FR et EN) ET au contrat `i18nContract.ts` (la
      parité est typée `Record<Locale, T>` : tsc est le gate).
- [ ] 1.3 Variante compacte : appliquer la décision n°2.
- [ ] 1.4 Aucune couleur en dur (tokens sémantiques uniquement) ; pas d'emoji.

Gate 1 (commandes exactes, depuis `apps/web/` du worktree) :
- `npx tsc -b` -> 0 erreur ;
- `npx vitest run src/features/match-replay` -> 0 échec ;
- `npx eslint src/features/match-replay --max-warnings=-1` -> 0 erreur nouvelle ;
- plausibilité : sur 2 artefacts témoins du cache du principal (ex. `000d5950`,
  `7344d24f`), imprimer (test ou script jetable de worktree) le temps mort par joueur et
  vérifier l'ordre de grandeur vs le registre (médiane 8-10 s par mort) — chiffres au CR.

## Garde-rails d'exécution

- `npm ci` dans `apps/web` du worktree (autorisé, précédent worktrees frères) ; vitest peut
  devoir tourner hors sandbox.
- Les artefacts témoins vivent dans le dépôt principal
  (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/`), lecture seule.
- Fichiers <= 500 L, fonctions <= 80 L ; aucune logique métier dans le composant (module
  `*_logic.ts` / `deadTimeLogic.ts`).
- Aucun fichier `.ai/` du principal modifié (journal/registre = superviseur).

## Découvertes

(consigner ici — rien corriger)

## CR attendu

Statut de chaque item, sorties des gates 0 et 1 (copiées), chiffres de plausibilité,
captures de décision (compacte oui/non et pourquoi). Commits atomiques
`temps-mort(pN): ...`, JAMAIS `git add -A`.
