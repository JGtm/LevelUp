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
6. AMENDEMENT du 2026-08-24 (revue adversariale, ronde 1 — constats 1 et 2, arbitrés par le
   coordinateur) : la mesure est REFUSÉE (`null`, la fiche écrit « — » avec son infobulle)
   dès qu'une trace SANS xuid chevauche (intersection strictement positive) au moins un trou
   du joueur, et pour tout joueur du roster SANS AUCUNE vie. Un trou entre deux vies nommées
   n'est pas une mort : le pont slot -> xuid est incomplet sur une partie du corpus. Pas
   d'attribution devinée, pas de soustraction — on refuse le chiffre, on ne le rafistole pas.
   La décision n°5 tient toujours : la ligne est TOUJOURS présente, seule sa valeur change.

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

- [x] 1.1 Ligne « Temps mort » dans la fiche joueur pleine (composant de fiche existant —
      PAS ReplayCanvas), format mm:ss, libellé via les tables i18n du feature.
      `DeadTimeRow` dans `ReplayTeams.tsx` (composant local, en PIED de fiche : c'est la
      seule ligne qui ne change pas avec la lecture, l'intercaler couperait le bloc de
      l'état courant). Le cumul est calculé UNE FOIS pour toute la colonne (`useMemo` sur
      `[groups, doc]`) et descendu en prop : aucune fiche ne le recalcule, aucune image ne
      le refait. `ReplayCanvas.tsx` NON TOUCHÉ (797 L, cliquet intact).
- [x] 1.2 Clés ajoutées aux DEUX tables (FR et EN) ET au contrat `i18nContract.ts` (la
      parité est typée `Record<Locale, T>` : tsc est le gate).
      Une seule clé, `deadTimeLabel` : FR « Temps mort », EN « Time dead ».
- [x] 1.3 Variante compacte : appliquer la décision n°2.
      NON — pleine seulement. La compacte n'a AUCUNE rangée libre (nom+compteurs, vitalité,
      armes+inventaire fondus) et son objet est d'être plus courte : lui ajouter une rangée
      annulerait ce qu'elle gagne (leçon C1 du 18/08, « la compacte serait plus HAUTE que la
      validée »). La seule rangée partagée est celle du NOM, où la valeur aurait serré le
      gamertag tronqué contre les compteurs dans la mise en page la plus étroite du rejeu.
      La parité de rangées morte/vivante reste tenue dans les DEUX variantes (la ligne ne
      dépend d'aucun état vital) ; le test de la compacte passe de `avant - 2` à `avant - 3`.
- [x] 1.4 Aucune couleur en dur (tokens sémantiques uniquement) ; pas d'emoji.
      `text-muted-foreground` seul, aucun hex ni classe de couleur Tailwind ; diff vérifié
      sans emoji.

Gate 1 (commandes exactes, depuis `apps/web/` du worktree) :
- `npx tsc -b` -> 0 erreur ;
- `npx vitest run src/features/match-replay` -> 0 échec ;
- `npx eslint src/features/match-replay --max-warnings=-1` -> 0 erreur nouvelle ;
- plausibilité : sur 2 artefacts témoins du cache du principal (ex. `000d5950`,
  `7344d24f`), imprimer (test ou script jetable de worktree) le temps mort par joueur et
  vérifier l'ordre de grandeur vs le registre (médiane 8-10 s par mort) — chiffres au CR.

Gate 1 PASSÉ le 2026-08-24 (depuis `apps/web/` du worktree) :
- `npx tsc -b` -> 0 erreur ;
- `npx vitest run src/features/match-replay` -> 69 fichiers, 1027 tests, 0 échec ;
- `npx eslint src/features/match-replay --max-warnings=-1` -> 0 erreur, 1 avertissement
  PRÉEXISTANT et hors périmètre (`ReplayFeedName.tsx:50`, react-refresh) ;
- plausibilité (sonde jetable `src/lib/replay/_tmp_plausibilite.test.ts`, supprimée après
  mesure — les tests commités ne dépendent pas du cache local) :
  - `000d5950` (schéma 18, 4 985 images, 100 ms/image, 499 s, 8 joueurs) : temps mort par
    joueur 73,8 s à 135,5 s (1:13 à 2:15), soit 15 % à 27 % de la partie ; 85 trous ;
    MÉDIANE PAR MORT 8,1 s (min 2,7 s, max 54,2 s) ; cumul de match 769 s ;
  - `7344d24f` (schéma 18, 5 689 images, 100 ms/image, 569 s, 8 joueurs) : 120,8 s à
    167,3 s (2:00 à 2:47), soit 21 % à 29 % ; 109 trous ; MÉDIANE PAR MORT 10,1 s
    (min 1,8 s, max 18,3 s) ; cumul de match 1 097 s.
  Verdict : la médiane par mort tombe exactement dans la fourchette 8-10 s du registre. Le
  cumul par match encadre la fourchette 865-1 136 s sans y tomber des deux côtés (769 s et
  1 097 s) — cf. §Découvertes, non corrigé.
  CES CHIFFRES SONT CEUX D'AVANT LA RONDE 1 : ils comptaient comme temps mort des trous
  occupés par des vies non rattachées. Mesures faisant foi : §Ronde 1 ci-dessous.

## Ronde 1 — corrections de revue adversariale (2026-08-24)

Constats arbitrés par le coordinateur ; ronde 2 relira. Statut :

- `[x]` **Constat 1 (P1)** — trou occupé par une vie SANS xuid : `deadTimeByPlayer` rend
  `number | null`, refus dès une intersection strictement positive avec un trou. Le contact
  de bornes (intersection nulle) ne refuse PAS — sans cette précision la ligne serait muette
  partout. `DeadTimeRow` écrit « — » et porte l'infobulle explicative.
- `[x]` **Constat 2** — joueur du roster sans aucune vie : même refus. Le commentaire de
  `DeadTimeRow` qui affirmait « le film date toutes les vies » (faux sur 5 artefacts du
  cache) est réécrit : il décrit maintenant les deux issues, mesure ou refus.
- `[x]` **Constat 3 (P2)** — tri défensif éprouvé pour de bon : nouveau test construisant un
  `ReplayPlayer` À LA MAIN, vies en désordre, sans passer par `buildPlayers`. VÉRIFIÉ PAR
  MUTATION : `.sort` retiré -> ce test seul échoue (`expected +0 to be 140000`), 22 autres
  passent ; `.sort` remis -> 23/23.
- `[x]` **Constat 4 (P2)** — le cas nuisible est couvert : vie anonyme ENTRE deux vies
  nommées -> `null`. Les cas inoffensifs (anonyme avant la première vie, après la dernière,
  contact de bornes) restent testés à part pour interdire le sur-refus.
- `[~]` **Constat 5 (P2)** — non corrigé sur arbitrage : cf. Découverte 7.

Gates de la ronde 1 (depuis `apps/web/`, `node_modules/.tmp` purgé avant `tsc`) :
- `npx tsc -b` -> 0 erreur ;
- `npx vitest run src/features/match-replay` -> 69 fichiers, 1 040 tests, 0 échec ;
- `npx eslint src/features/match-replay --max-warnings=-1` -> 0 erreur, 1 avertissement
  préexistant hors périmètre.

Plausibilité APRÈS refus (sonde jetable, supprimée) — mesurés / refusés, et médiane sur les
seuls joueurs mesurés :
- `000d5950` (499 s, 8 joueurs, 6 vies non rattachées) : **1 mesuré / 7 en « — »** ; le seul
  mesuré (whiteknight2519) 73,8 s (01:13), médiane par mort 8,1 s, plus long trou 8,8 s ;
- `7344d24f` (569 s, 8 joueurs, 5 vies non rattachées) : **3 mesurés / 5 en « — »** ;
  120,9 s / 130,7 s / 159,2 s (02:00 à 02:39), médiane par mort 10,1 s, plus long trou
  18,3 s ; cumul des mesurés 411 s sur 40 trous ;
- `64e8adfa` (834 s, 8 joueurs, 11 vies non rattachées) : **1 mesuré / 7 en « — »** ;
  Bel homme 21 à 92,4 s (01:32), médiane par mort 10,2 s. flamesamurai, le cas du
  relecteur, est bien REFUSÉ (il aurait affiché 311,1 s, soit 05:11).

Les médianes des joueurs mesurés restent dans la fourchette 8-10 s du registre, et le cas
prouvé faux ne s'affiche plus. Le TAUX DE REFUS, lui, est très élevé : cf. Découverte 6, qui
est la question à trancher en ronde 2. RÉSOLU EN RONDE 1b — voir ci-dessous.

## Ronde 1b — affinement de la règle et gel de dette (2026-08-24)

Deux points arbitrés par le coordinateur après vérification de la ronde 1.

- `[x]` **Taux de refus (Découverte 6) — règle AFFINÉE, pas assouplie.** Une trace anonyme
  n'invalide le trou `G` d'un joueur que si elle y est CONTENUE (`start >= G.start` ET
  `end <= G.end`, bornes en images après clamp). Justification, écrite en tête du module :
  UN JOUEUR N'A QU'UN BIPÈDE À LA FOIS — une trace qui déborde sur une vie NOMMÉE du joueur
  ne peut pas être une vie de lui, il était ailleurs, incarné ; elle ne prouve donc rien
  contre son trou. Seule une trace qui POURRAIT être la vie manquante force le refus. C'est
  une preuve d'EXCLUSION, pas une attribution devinée : on ne rattache toujours rien à
  personne. La clause « durée non nulle » de la ronde 1 est CONSERVÉE (une trace ponctuelle
  ne montre personne en train de jouer) — les deux conditions se cumulent.
- `[x]` **Gel de dette : `ReplayTeams.test.tsx` ramené à sa taille d'avant-lot.** Tous les
  tests de temps mort (phase 1 ET ronde 1) sont partis dans `ReplayTeamsDeadTime.test.tsx`
  (nouveau fichier, 165 L). `ReplayTeams.test.tsx` fait de nouveau 881 L — EXACTEMENT sa
  taille d'origine — et son diff vs `b16ba17e5` tient en 2 lignes (le compteur de rangées de
  la fiche compacte, `avant - 2` -> `avant - 3`, et son commentaire). La Découverte 8 est
  donc close. `TRACK` est recopié localement dans le nouveau fichier (2e et dernière copie,
  règle « <= 2 ») : exporter une constante d'un fichier de test pour l'importer dans un autre
  créerait une dépendance entre suites.

Tests ajoutés (dans `deadTimeLogic.test.ts`, nouveau bloc « seule une vie anonyme CONTENUE
dans le trou refuse ») : les DEUX cas réels du relecteur en fixtures aux coordonnées exactes
(`64e8adfa` slot 607 [5541..7496] dans [5441..7519] ; `000d5950` slot 588 [3853..4170] dans
[3714..4256]) -> refus ; débordement sur la vie suivante, sur la précédente, enjambement
complet, caméra de fin de match courant jusqu'à `frameCount` -> MESURÉ ; deux traces dont une
seule contenue -> refus ; trace ponctuelle dans le trou -> mesuré. Inchangés et re-vérifiés :
`lives` vide -> null, anonyme avant la première vie / après la dernière -> inoffensif.

Gates ronde 1b (`node_modules/.tmp` purgé avant `tsc`) :
- `npx tsc -b` -> 0 erreur ;
- `npx vitest run src/features/match-replay` -> 70 fichiers, 1 047 tests, 0 échec ;
- `npx eslint src/features/match-replay --max-warnings=-1` -> 0 erreur, 1 avertissement
  préexistant hors périmètre.

Plausibilité — AVANT (règle large, ronde 1) puis APRÈS (règle contenue, ronde 1b) :

| artefact | ronde 1 | ronde 1b | médiane/mort des mesurés | plus long trou |
|---|---|---|---|---|
| `000d5950` | 1 mesuré / 7 « — » | **2 / 6** | 8,1 s | 8,8 s |
| `7344d24f` | 3 mesurés / 5 « — » | **8 / 0** | 10,1 s | 18,3 s |
| `64e8adfa` | 1 mesuré / 7 « — » | **3 / 5** | 10,1 s | 10,7 s |
| TOTAL | 5 / 19 | **13 mesurés / 11 refus** | | |

VÉRIFIÉ NOMMÉMENT (assertions de la sonde, pas une lecture à l'oeil) : `JGtm` sur `000d5950`
et `flamesamurai` sur `64e8adfa` restent REFUSÉS — les deux cas prouvés par la revue.
`7344d24f` passe à zéro refus : ses 5 traces anonymes courent toutes jusqu'à la dernière
image (5 647 à 5 688 sur 5 689), donc aucune n'est contenue dans un trou. C'est exactement le
motif « caméra de fin de match » que l'affinement devait cesser de punir. Les 11 refus qui
restent sont des traces courtes et tardives, réellement contenues : le doute y est réel.

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

1. **Le délai de réapparition n'est PAS une constante entre matchs.** Les deux témoins
   donnent 8,1 s et 10,1 s de médiane par mort, chacun très serré autour de sa valeur
   (les huit joueurs d'un même match affichent la MÊME médiane à 0,1 s près). Le palier
   « 8,0 s » consigné jusqu'ici vient du film de référence, c'est-à-dire d'UN match : il
   dépend visiblement du mode ou de la playlist. Rien à corriger — c'est au contraire ce
   qui valide le refus d'une constante dans le calcul — mais tout commentaire du dépôt qui
   présente 8,0 s comme un palier du JEU est trop fort d'un cran.
2. **Cumul par match sous la fourchette du registre sur `000d5950`** : 769 s mesurés contre
   la bande 865-1 136 s consignée (`7344d24f` tombe dedans, 1 097 s). Écart non
   investigué : la bande du registre peut avoir été mesurée sur d'autres matchs, ou sur une
   définition incluant les bornes de tête/queue que ce lot exclut délibérément. Aucun
   changement de définition n'a été fait pour rentrer dans la bande.
3. ~~**Trous longs réels** : maximum 54,2 s sur `000d5950`.~~ **AMENDÉE (ronde 1) : cette
   découverte était FAUSSE.** Le trou de 54,2 s de JGtm n'est pas une longue mort mais une
   VIE NON PONTÉE (trace anonyme du slot 588, 317 points, preuve du relecteur). Après le
   refus, le plus long trou d'un joueur mesuré tombe à 8,8 s sur `000d5950`, 18,3 s sur
   `7344d24f` et 10,6 s sur `64e8adfa`. Ce qui reste vrai : le cumul se fait sur des
   intervalles LUS, jamais sur « nombre de morts x délai médian ».
4. **Surface naturelle pour l'agrégat d'équipe** (HORS PÉRIMÈTRE, non implémenté) :
   `ReplayTeamHeader.tsx` porte déjà le score de la colonne à l'instant lu et reçoit
   `players` — il pourrait sommer les cumuls de ses joueurs sans nouvelle dérivation. À
   traiter comme un lot à part si le besoin se confirme.
5. **La sonde jetable a dû sortir du dossier `match-replay`** : le garde-rail
   `testDoc.guard.test.ts` interdit à tout `*.test.ts` du dossier d'appeler
   `normalizeReplayDocument` hors de la fixture. Sonde donc placée sous `src/lib/replay/`
   puis supprimée. Garde-rail correct, simple contrainte à connaître pour les mesures
   ponctuelles sur artefacts réels.
6. **TRANCHÉE EN RONDE 1b (règle affinée : contenue dans le trou). Taux de refus ramené de
   19/24 à 11/24.** Ce qui suit est le constat d'origine, gardé pour la trace. La règle
   arbitrée (toute trace sans xuid qui chevauche un trou invalide le joueur) refuse
   **19 joueurs sur 24** sur les trois témoins (1/8, 3/8, 1/8 mesurés). La ligne affiche donc
   « — » quatre fois sur cinq. La règle est appliquée telle qu'arbitrée ; ce qui suit est la
   MESURE qui permettra de la réviser ou de la confirmer, pas une proposition de correctif.
   Diagnostic des traces sans xuid (sonde jetable, supprimée) :
   - elles sont TOUTES groupées en FIN de match : `000d5950` -> les 6 vivent entre les images
     3 795 et 4 945 sur 4 985 (dernier quart) ; `7344d24f` -> les 5 entre 5 117 et 5 688 sur
     5 689 (dernier dixième) ; `64e8adfa` -> les 11 entre 3 700 et 8 336 sur 8 337, dont 9
     après l'image 5 541 ;
   - leurs durées vont de 2 s (4 points) à 196 s (1 866 points — c'est le slot 607 de
     `64e8adfa`, celui du relecteur) ;
   - `team` vaut -1 sur toutes.
   Autrement dit une poignée de traces tardives suffit à invalider presque tout le monde,
   parce que presque tout le monde meurt au moins une fois en fin de partie. Deux lectures
   possibles, à départager avec une donnée que ce lot n'a pas cherchée (le commentaire de
   `rosterLogic.buildPlayers` dit que les traces sans xuid sont « les caméras et les
   spectateurs de fin de partie », ce qui n'est pas compatible avec 1 866 points de
   déplacement) : soit ces traces sont de vraies vies mal pontées et le refus est juste, soit
   une partie sont des caméras et le refus est trop large. Trancher demande de savoir
   distinguer une caméra d'un bipède — hors périmètre de cette ronde.
7. **Constat 5 de la revue, consigné et NON corrigé (arbitrage coordinateur)** : l'invariant
   « le temps mort n'est pas recalculé par image » (le `useMemo` sur `[groups, doc]` dans
   `ReplayTeams`) n'est protégé par aucun test. Accepté : éprouver une mémoïsation demande
   d'espionner le module ou de compter des rendus, deux tests réputés fragiles. Le risque
   résiduel est une régression de performance silencieuse sur les gros BTB, pas un faux
   affichage.
8. **CLOSE EN RONDE 1b.** ~~`ReplayTeams.test.tsx` dépasse le seuil de 500 L : 881 L avant ce
   lot, 981 L après.~~ Les tests de temps mort sont partis dans `ReplayTeamsDeadTime.test.tsx`
   et le fichier est revenu à 881 L, sa taille exacte d'avant-lot : la dette gelée n'est pas
   accrue. Elle reste une dette (881 L pour un seuil à 500), simplement pas de notre fait.

## CR attendu

Statut de chaque item, sorties des gates 0 et 1 (copiées), chiffres de plausibilité,
captures de décision (compacte oui/non et pourquoi). Commits atomiques
`temps-mort(pN): ...`, JAMAIS `git add -A`.
