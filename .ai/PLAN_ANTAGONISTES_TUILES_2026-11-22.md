# PLAN — Antagonistes à l'instant, frise éliminations / morts, actions sur les tuiles du rejeu (2026-11-22)

Statut : **écrit le 2026-11-22 à la demande du user (« variante A, fais le plan » ; livraison en 8.3.0). DU-1 à DU-4
confirmées par le user le 2026-11-22 (« ok pour les 4 »). En cours : étape 0 (go du user le 2026-11-22, « tu peux pas commencer maintenant ? »).**

Maquette de référence (validée par le user le 2026-11-22, variante A) : https://claude.ai/artifact/D24BJbsd99q5Lb8nBdYCmw.
En cas d'écart entre ce plan et la maquette, ce plan fait foi ; un écart non écrit ici est un défaut.

Contrat d'exécution : skill `plan-execution` (complément `levelup-plan`). Statuts : `[x]` fait, `[~]` couvert ailleurs
(référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture. Ordre : une étape n'est close que gate
passé, plan mis à jour et commit fait. Zéro correctif hors périmètre : toute découverte va dans « Découvertes ».

Piloté par le plan maître de la 8.3.0 (`.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`), qui fixe l'ordre entre plans.

## 0. Objectif et critère de succès

Objectif : sur la page du rejeu, montrer les antagonistes (némésis, souffre-douleur) calculés à l'instant, poser les
éliminations en haut et les morts en bas de la frise, et donner aux tuiles des joueurs deux actions directes, Suivre
et Afficher / Masquer.

Critère de succès, vérifiable à l'écran par le user sur un match réel :
1. Le canevas montre l'icône némésis et l'icône souffre-douleur devant le nom des deux antagonistes du point de vue ;
   elles changent de joueur au moment de l'élimination qui fait passer le titre, avec une pulsation.
2. À la dernière image, les deux titres du rejeu sont ceux des cartes de la page du match.
3. La frise porte une icône par changement de titre, avec l'infobulle validée ; les éliminations du point de vue
   sont en haut de sa rangée, ses morts en bas.
4. La fiche de n'importe quel joueur nomme ses antagonistes à l'instant.
5. Chaque tuile (et chaque ligne de la colonne) porte Suivre et l'œil, au survol ; actifs, ils restent visibles.
   Masquer plusieurs joueurs fonctionne ; « Tout afficher » les réaffiche ; « Seul » dans la fiche masque les autres.

## 1. État vérifié sur pièces (2026-11-22)

- Branche courante du dépôt : `release/v8.3.0`, arbre propre.
- Les éliminations du rejeu portent tueur, victime et instant : `ReplayKill` (`features/match-replay/model/killFeedLogic.ts:85`,
  hérite de `KillEvent`, `features/match-view/_momentum.ts:50`, champs `xuid`, `victimXuid`). Même source que la page
  du match : `combat_tab.highlight_events` (`features/match-view/MatchViewPage.tsx:168`).
- Camp d'un joueur sur le rejeu : le film seul (`lib/replay/filmAllegiance.ts:73`, `ofXuid`).
- Point de vue : `hooks/useReplayViewpoint.ts` ; marque `me` du canevas : `lib/replay/playerMarks.ts`.
- Cartes de la page du match : `features/match-view/MatchNemesisCards.tsx`, source `team_tab.nemesis`, départage par
  le premier de la liste (`pickMax`, ligne 121) : pas le même départage que la règle du tenant.
- Icônes : `apps/web/public/icons/nemesis-{black,white}.png`, `victim-{black,white}.png` (noir sur transparent),
  chemin par `lib/themedIcon.ts`.
- Canevas : `layers/replayMarkers.ts` **478 lignes** (seuil 500 : aucun ajout dans ce fichier), `layers/replayLabels.ts`
  72, `layers/offscreenChevron.ts` 124.
- Frise : `ui/ReplayTimelineTracks.tsx`, `ui/ReplayMarkTrack.tsx` (163 ; éliminations et morts sur la même ligne,
  distinguées par l'encre), `model/replayTimelineTracksLogic.ts` (583, au-dessus du seuil : dette gelée, ne pas
  l'accroître), garde-rail de géométrie `ui/timelineGeometry.guard.test.ts`.
- Actions du joueur : `model/playerPanelLogic.ts` (289) ; Seul et Masquer portent sur UN joueur, celui de la fiche, et
  se remettent à zéro à sa fermeture ; règle des calques `ownershipShown` (ligne 238) et `model/focusFilter.ts`.
  Boutons : `ui/fiche/FicheActions.tsx` (Suivre, Seul, Masquer). Tuile et ligne : `ui/ReplayPlayerCard.tsx` (321),
  `ui/ReplayRosterRow.tsx` (259), contrat commun `ui/replayCardPress.ts`.
- Fiche : sections sous `ui/fiche/`, composées par `ui/ReplayPlayerPanelBody.tsx`.
- Autres chantiers sur ces fichiers : le plan du backlog des 17 et 18 novembre (`chore/830-backlog-1118`, phase
  d'intégration) a modifié tuiles, fiche et barre ; la branche `feat/medailles-bouclier` modifie
  `ui/fiche/FicheDepuisDebut.tsx` et `model/playerInstantStats.ts` (session levelup-60, confirmé le 2026-11-22) ; le
  plan du décodeur (H.6, H.7) touchera le fil et la fiche à sa montée.

## 2. Décisions

### Validées par le user le 2026-11-22 (maquette et messages du jour)

- **D-1** Textes de la frise : titre « Némésis : X » ou « Souffre-douleur : X » ; phrase « Premier adversaire à
  éliminer N. », « A éliminé N k fois et passe devant Y (j). », « Premier adversaire éliminé par N. »,
  « Éliminé k fois par N, passe devant Y (j). » (N = point de vue). Pas de marque pour une égalité.
- **D-2** Fiche : section des antagonistes DU JOUEUR DE LA FICHE, quel qu'il soit : « Némésis » → « X, k éliminations
  [, à égalité avec Y] » ; « Souffre-douleur » → « X, éliminé k fois [, à égalité avec Y] ». Définition en infobulle
  de l'icône : « L'adversaire qui a éliminé N le plus de fois jusqu'à cet instant. » / « L'adversaire que N a
  éliminé le plus de fois jusqu'à cet instant. », suivie de « En cas d'égalité, le titre reste à celui qui l'avait. »
  État vide : cadre pointillé « Aucun antagoniste à cet instant. »
- **D-3** Égalité : le titre reste à celui qui le tient ; il ne change de main que sur un dépassement strict.
- **D-4** Canevas : icônes devant le nom, pour les deux antagonistes du POINT DE VUE seulement ; teinte du camp du
  joueur marqué, contour sombre des noms (`--replay-label-stroke`) ; la flèche hors cadre porte l'icône.
- **D-5** Frise : éliminations du point de vue en haut de sa rangée, morts en bas.
- **D-6** Tuiles, variante A : cible (Suivre) et œil (Afficher / Masquer) apparaissent au survol de la tuile ou à son
  focus ; une action active reste visible ; sur écran sans survol (`hover: none`), visibles à 60 %. Le clic ailleurs
  sur la tuile ouvre la fiche. Même chose sur la ligne de la colonne (`ReplayRosterRow`).
- **D-7** Masquer devient un ENSEMBLE de joueurs masqués, indépendant de la fiche, qui dure jusqu'à « Tout afficher »
  ou au changement de match. « Tout afficher » apparaît en tête de la colonne des équipes dès qu'un joueur est masqué.
- **D-8** Suivre et Masquer quittent la fiche. « Seul » y reste et masque tous les autres joueurs ; actif, il se lit
  « Seul · tout réafficher » et réaffiche tout. Après Seul, l'œil d'une tuile réaffiche un joueur.
- **D-9** Livraison dans la 8.3.0.

### Confirmées par le user le 2026-11-22 (« ok pour les 4 »)

- **DU-1** Moment du calcul : à l'instant, sans réglage. La
  maquette garde « Fin de match » pour comparer ; il n'est pas livré.
- **DU-2** Icône sur le canevas : 12 px, compte affiché à droite de l'icône.
- **DU-3** Calques qui ne sont pas des pions (effets de mort, armes au sol, poses d'équipement) avec plusieurs
  joueurs masqués. Aujourd'hui « Seul » montre un élément dont un joueur est le sien, « Masquer » cache un effet de
  mort dès que le masqué y figure. Avec un ensemble, une seule règle est possible : **un élément se dessine dès qu'un
  de ses joueurs est affiché**. Conséquence : masquer X laisse visible l'effet d'une élimination
  entre X et un joueur affiché. Le décor sans joueur reste affiché (règle du 2026-11-15 inchangée).
- **DU-4** Frise, rangée des coéquipiers : telle quelle (14 px, deux demi-rangées de 7 px ne se liraient pas).

### Tranché par le user le 2026-11-22 (né de l'étape 0 ; « ok exclus les bots, vas-y »)

- **DU-5** Place des bots. Le fil ne contient aucune élimination dont un bot est l'AUTEUR (49 manquantes sur
  l'échantillon) : un bot ne peut pas être némésis. Les cartes de la page du match excluent les bots des duels
  (`service/match_view_builders_team.go:333`, décision du 2026-09-02) ; la règle du 2026-10-09 dit que les bots
  comptent partout. Recommandation du pilote : bots exclus des antagonistes partout (rejeu et page du match), tant que
  le décodeur ne lit pas leurs éliminations. **Retenu** : un bot n'a pas d'antagoniste et n'en est jamais un (clé `bot:<nom>` ou victime sans xuid = pas un adversaire), sur le rejeu comme sur la page du match.

### Tranché par le pilote (conventions du dépôt, modifiable par le user)

- **D-10** (corrigée par l'étape 0) Adversaire = `allegiance.ofXuid(clé) === false` (`lib/replay/filmAllegiance.ts:127`) :
  autre camp selon le film, tout autre joueur en mêlée générale ; `null` = pas compté. Trahisons et morts sans tueur
  ignorées. Clé d'un joueur = sa clé du film (`ReplayPlayer.xuid`, `bot:<nom>` pour un bot) : tueur = `kill.xuid` ;
  victime = `kill.victimXuid` s'il est non vide, sinon `botKey(kill.victimGamertag)` (`lib/replay/rosterLogic.ts:62`),
  sinon l'élimination est ignorée. Le point de vue (xuid de feuille) se ramène à la clé du film par
  `trackedPlayerOf(viewpoint, players)?.xuid` (`model/playerPanelLogic.ts:165`). Jamais de comparaison à `bid(N.0)`.
  Place des bots : DU-5.
- **D-11** Une seule fonction pure, sous `apps/web/src/lib/antagonists/`, lue par le rejeu ET par la page du match
  (`MatchNemesisCards`, calculée sur `highlight_events` au dernier instant) : la dernière image du rejeu et les cartes
  ne peuvent pas diverger. `team_tab.nemesis` reste servi : `KdIndicatorCard` (`MatchStatCards.tsx`) le lit aussi.
- **D-16** (né de 0.3) La fonction écarte les doublons d'élimination (même tueur, même victime, même instant) avant
  de compter : 172 matchs sur 1 462 de la base du poste portent chaque élimination deux fois dans `highlight_events`.
  Gate F : décompte dédoublonné égal à `team_tab.nemesis` sur les 1 225 matchs de 0.3.
- **D-12** Pulsation au changement de titre : deux anneaux à la couleur du camp et l'icône qui grossit puis revient,
  1,4 s de temps réel (la durée de `EVENT_HOLD_MS`, `hooks/useReplayTiming.ts:30`) ; l'icône de l'ancien
  tenant s'efface dans le même temps. Rien de cela en mouvement réduit (`prefers-reduced-motion`).
- **D-13** Frise : les icônes de titre se posent dans une bande de 14 px au-dessus de la rangée du point de vue ; ce
  sont des boutons (infobulle au survol et au focus, clic = curseur à l'instant du changement).
- **D-14** Textes anglais : libellés existants de la page du match (`Nemesis`, `Bully target`) ; phrases rédigées
  par l'agent de B et D, relues par le pilote, montrées au user avec la validation à l'écran.
- **D-15** Suivre un joueur masqué le réaffiche ; masquer le joueur suivi arrête le suivi ; Seul sur un joueur arrête
  le suivi d'un joueur que Seul masque.

## 3. Organisation

Branche d'intégration : `feat/830-antagonistes-tuiles`, créée depuis `release/v8.3.0` **après** la fusion du plan du
backlog des 17 et 18 novembre (son R.4) : ce plan touche les tuiles et la fiche qu'il vient de modifier. Un worktree
par agent qui écrit, en frère du dépôt (`../LevelUp-wt-ant-<piste>`), jamais sous `C:/t`. Sous-branches
`ant/<piste>`, fusionnées par le pilote dans la branche d'intégration, supprimées à l'intégration.

Chemin critique : 0 → A → A.4 → (B, C.2, D) ; F après A. C.1 et E ne dépendent pas de A et partent avec lui. B, D et F attendent aussi DU-5.

| Piste | Contenu | Modèle | Dépend de | Effort |
|---|---|---|---|---|
| 0 | vérification sur pièces, témoins avant | Fable | go du user | rapide |
| A | fonction pure des antagonistes | Sonnet | 0 | rapide |
| B | canevas : icônes, pulsation, flèche | Sonnet | A | moyen |
| C | frise : C.1 haut/bas, C.2 icônes de titre | Sonnet | C.1 : 0 ; C.2 : A et C.1 | moyen |
| D | fiche : section des antagonistes | Sonnet | A ; E.4 pour `ReplayPlayerPanelBody.tsx` | rapide |
| E | tuiles, ensemble des masqués, Seul | Sonnet | 0 | moyen à lourd |
| F | page du match sur la même fonction | Sonnet | A | rapide |
| I | intégration | pilote | toutes | moyen |
| R | revue et clôture | Fable (revue) | I | moyen |

Fichiers disputés : `ui/ReplayPlayerPanelBody.tsx` (D et E) : E.4 d'abord, D.2 ensuite sur la base de E.
`ui/ReplayTimelineTracks.tsx` (C.1 et C.2) : même agent, à la suite.

Gates communs des agents (web, depuis `apps/web`) : `npx tsc -b --noEmit` (ou `make check-types` depuis la racine),
`npx vitest run <dossiers de la piste> --maxWorkers=2`, `node tools/lint-cross-feature-imports.mjs`,
`npx vitest run src/lib/i18n --maxWorkers=2` (parité des langues), `wc -l` de chaque fichier touché (≤ 500, ou pas
plus long qu'avant pour un fichier déjà au-dessus). Aucun processus vitest vivant à la fin. Aucune couleur hors
jetons (`color-tokens`), aucun `fetch` hors `lib/api`.

## Étape 0 — vérification sur pièces et témoins (Fable)

- [x] 0.1 (fait le 2026-11-22 ; section 1 exacte sauf : rejeu lu par `/replay/context`, `model/replayModel.ts:169`, même décoration serveur que `combat_tab` ; `ui/ReplayTimelineTracks.tsx` 513 lignes ; `PRESS_GUARDS` dans `ui/replayPressGuards.ts:16` ; durée de 1,4 s = `EVENT_HOLD_MS` ; icônes à alpha 1 bit, némésis 714×670, souffre-douleur 1099×1369) Relire chaque fichier cité en section 1 sur `release/v8.3.0` à jour (après la fusion du backlog
      1118) ; corriger les lignes de ce plan qui ont bougé, dans ce plan, avant tout code.
- [x] 0.2 (fait : 152 rejeux, 13 781 éliminations ; règle écrite en D-10 ; aucune élimination dont un bot est l'auteur dans le fil, d'où DU-5) Clé de joueur dans les éliminations : sur 3 rejeux du parc (un 4 contre 4, une mêlée générale, un match avec
      bots), vérifier que `xuid` et `victimXuid` d'un `ReplayKill` se rapprochent de `ofXuid` et des clés de roster
      (bots compris) ; écrire la règle de rapprochement dans ce plan.
- [x] 0.3 (fait sur une copie des bases, API du serveur de la branche : 20 matchs récents, 78 paires, 0 écart ; 1 225 matchs, 154 en écart, toutes leurs paires exactement au double, cause : éliminations en double dans `highlight_events` de 172 matchs ; d'où D-16) Comptes de la page du match : sur 20 matchs locaux, comparer `team_tab.nemesis` (killed_me, i_killed) au
      décompte de `highlight_events` par paire. Écart nul attendu ; tout écart est écrit ici et arrête F.
- [x] 0.4 (fait : 15 témoins et `empreintes.txt`, `sha256sum -c` 15 OK au contrôle du pilote ; harnais `features/match-replay/test/temoinsAntagonistes/`, 18 tests verts ; commande dans son README ; pour E, seule `focusDe` de `temoinsKit.tsx` s'adapte au nouveau type) Témoins AVANT, capturés dans le dossier `../LevelUp-temoins-ant/avant/` (frère du dépôt, hors git, gardé jusqu'à I.3) : enregistrement des appels de dessin
      (`test/recordingContext`) de 3 images sans antagoniste et 3 images avec ; rendu de la frise (`ReplayMarkTrack`)
      sur le harnais `test/timelineTracksHarness.tsx` ; sorties de `ownershipShown` et `focusFilter` sur Seul et
      Masquer d'un joueur (tests existants listés par nom).
- **Gate 0** : les quatre sorties écrites dans ce plan (règle de clé, écarts 0.3, chemin des témoins) ; aucune ligne
  de section 1 fausse.

## Piste A — fonction pure des antagonistes (Sonnet)

- [x] A.1 (fait, `5ec83bb082` : `buildAntagonists(kills, subject, isFoe)` ; dans un changement, `holder` = adversaire dont le compte bouge, `previousHolder` = tenant d'avant ; pour `egalite` le titre reste à `previousHolder` ; à instant égal, ordre des clés) `lib/antagonists/antagonists.ts` : à partir d'une liste d'éliminations triées (tueur, victime, instant), d'un
      joueur et d'un prédicat « adversaire », rend pour un instant donné les deux titres (tenant, compte, joueurs à
      égalité) et la liste des changements de titre (instant, rôle, nouveau tenant, ancien tenant et son compte,
      genre `prise` ou `égalité`). Règles D-3, D-10 et D-16 (doublons écartés).
- [x] A.2 (fait : un parcours, instantanés par élimination comptée, `at(t)` dichotomique ; test contre recalcul complet sur 400 éliminations) Lecture à l'instant en temps borné : index des changements construit une fois par (document, joueur),
      lecture par recherche dichotomique ; pas de parcours de toutes les éliminations à chaque image.
- [x] A.3 (fait : 20 tests, vus en échec puis verts, mutation `>` en `>=` détectée ; contrôle du pilote : 20 verts. « Victime bot par son nom » et « mort sans tueur » relèvent du rapprochement des clés : [~] tests de B, D et F) Tests : premier tenant ; dépassement strict ; égalité qui garde le tenant ; un joueur qui tient les deux
      titres ; trahison et mort sans tueur ignorées ; mêlée générale ; victime bot par son nom ; doublons ; instant avant toute élimination ;
      dernier instant égal au calcul sur toute la liste.
- **Gate A** : gates communs sur `src/lib/antagonists`.

## Piste A bis — entrées du rejeu (Sonnet, avant B, C.2 et D)

- [ ] A.4 `features/match-replay/model/replayAntagonists.ts` et un hook `hooks/useReplayAntagonists.ts` : depuis les éliminations du fil, l'allégeance du film et le roster, les clés rapprochées (D-10), le prédicat d'adversaire avec bots exclus (DU-5), et `buildAntagonists` gardé par (document, sujet) ; le sujet « point de vue » ramené à sa clé du film. Seule source des pistes B, C.2 et D (aucune ne refait le rapprochement). Tests : victime bot, tueur absent, point de vue bot, mêlée générale, trahison, doublons, un même index servi deux fois.
- **Gate A bis** : gates communs sur `src/features/match-replay/model` et `hooks`.

## Piste B — canevas (Sonnet)

- [ ] B.1 `layers/antagonistMarks.ts` : icônes teintées (masque alpha tiré du PNG noir, teinte passée par l'appelant,
      cache par couleur, taille et densité), dessin de l'icône avec contour, pulsation (D-12). Aucun ajout dans
      `replayMarkers.ts` au-delà du branchement (≤ 500 lignes tenues).
- [ ] B.2 `layers/replayLabels.ts` : l'étiquette accepte des icônes de titre et leur compte devant le nom (DU-2),
      centrées sous le pion, lisibles sur les deux thèmes ; même chose pour l'étiquette de la flèche hors cadre
      (`offscreenChevron.ts`).
- [ ] B.3 Branchement : hook de calque (patron `useReplay*`) qui lit la fonction de A pour le point de vue et fournit
      au calque des pions « titres de ce slot à cette image ». Un joueur masqué ne porte rien.
- [ ] B.4 Textes : nom accessible du canevas inchangé ; aucune chaîne en dur (i18n de la piste D partagé si besoin).
- [ ] B.5 Tests : enregistrement des appels de dessin (icône devant le nom, absente sans titre, pulsation puis repos,
      rien en mouvement réduit, flèche hors cadre avec icône) ; témoins 0.4 « sans antagoniste » identiques.
- **Gate B** : gates communs sur `src/features/match-replay/layers` et `src/lib/antagonists` ; témoins comparés.

## Piste C — frise (Sonnet)

- [x] C.1 (fait, `486616aa52` : barre d'élimination centrée à 6 px, de mort à 18 px, losanges suivis, badge collé en haut ; trois fixtures de la barre de lecture mises à jour ; témoins : seuls `frise.html` et `frise-piste-point-de-vue.html` changent, contrôlé par le pilote ; 117 tests ciblés verts) `ui/ReplayMarkTrack.tsx` : sur la rangée du point de vue (`tall`), éliminations dans la moitié haute, morts
      dans la moitié basse ; badges de médaille recalés sur leur marque ; rangée des coéquipiers inchangée (DU-4).
      Géométrie horizontale intouchée (`timelineGeometry.guard.test.ts` vert).
- [ ] C.2 Bande des titres (D-13) au-dessus de la rangée du point de vue, dans un composant extrait (`ui/ReplayTimelineTracks.tsx` fait 513 lignes : il ne grandit pas) : une icône par changement de titre (genre `prise` seulement), infobulle D-1, nom accessible
      « titre. phrase », clic = curseur à l'instant. Les autres pistes restent `pointer-events-none`.
- [ ] C.3 Textes D-1 en français et en anglais (D-14), dans un dictionnaire typé `Record<Locale, …>` de
      `features/match-replay/i18n/`.
- [ ] C.4 Tests : marques hautes et basses ; une icône par prise, aucune pour une égalité ; infobulle et nom
      accessible ; clic qui déplace le curseur ; témoin 0.4 de la frise comparé (seul le placement vertical change).
- **Gate C** : gates communs sur `src/features/match-replay/ui` et `model`.

## Piste D — fiche (Sonnet)

- [ ] D.1 `ui/fiche/FicheAntagonistes.tsx` : section « Antagonistes » du joueur de la fiche à l'instant (D-2), icônes
      à infobulle (même mécanisme que la frise), noms à la couleur de leur camp, état vide pointillé.
- [ ] D.2 Branchement dans `ui/ReplayPlayerPanelBody.tsx`, après l'identité, avant « Depuis le début du match »
      (fait APRÈS E.4, sur sa base).
- [ ] D.3 Textes D-2 en français et en anglais.
- [ ] D.4 Tests : titres, égalité, état vide, joueur sans adversaire, changement d'instant.
- **Gate D** : gates communs sur `src/features/match-replay/ui/fiche`.

## Piste E — tuiles et ensemble des masqués (Sonnet)

- [x] E.1 (fait, `9a208f8df9`) `model/playerPanelLogic.ts` : l'état porte un ensemble de joueurs masqués (D-7) au lieu de `solo` / `hide` ;
      événements `toggleHidden`, `solo`, `showAll` ; règles D-8 et D-15 ; l'ensemble se vide au changement de match.
- [x] E.2 (fait : la distinction « élément partagé » disparaît, une seule règle) `ownershipShown` et `model/focusFilter.ts` sur l'ensemble, règle DU-3 ; `shownAtSlot` des pions sur
      l'ensemble. Le décor sans joueur reste affiché.
- [x] E.3 (fait : `ui/ReplayTileActionButtons.tsx`, `model/playerTileActions.ts` ; infobulles par `title`) Tuile (`ui/ReplayPlayerCard.tsx`) et ligne (`ui/ReplayRosterRow.tsx`) : boutons Suivre et œil (D-6), à
      encres de jetons, `aria-pressed`, noms accessibles « Suivre X » / « Ne plus suivre X » / « Masquer X » /
      « Afficher X », infobulles ; le clic sur un bouton n'ouvre pas la fiche ; Espace et Entrée gardés comme
      aujourd'hui (`PRESS_GUARDS`, `ui/replayPressGuards.ts`). Nom pâle et mention « masqué » sur la tuile d'un joueur masqué.
- [x] E.4 (fait ; `FicheDepuisDebut.tsx` non touché) Fiche : `ui/fiche/FicheActions.tsx` ne garde que « Seul » (D-8) ; `ui/ReplayPlayerPanelBody.tsx` adapté.
- [x] E.5 (fait : `ui/ReplayShowAllButton.tsx` ; dans la rangée, en haut de la cellule du score : placement à contrôler à l'écran, I.5) « Tout afficher » en tête de la colonne des équipes (`ui/ReplayRosterColumn.tsx`, `ui/ReplayTeamsRow.tsx`
      selon le cadre), visible dès qu'un joueur est masqué.
- [x] E.6 (fait : `i18n/playerPanel.i18n.ts`) Textes en français et en anglais (Suivre, Ne plus suivre, Masquer, Afficher, Tout afficher, Seul, Seul ·
      tout réafficher, infobulles de la maquette).
- [x] E.7 (fait : vus en échec puis verts ; contrôle du pilote : 295 tests ciblés verts ; témoins : seuls `filtre-masquer.json` (76 éléments mixtes passés de retirés à dessinés, écart de DU-3) et la liste des tests changent) Tests : réducteur (masquer deux joueurs, Seul puis réafficher un joueur, suivre un masqué, masquer le suivi,
      changement de match) ; filtre des calques sur DU-3 ; tuile et ligne (boutons, clic qui n'ouvre pas la fiche,
      états actifs) ; témoins 0.4 de Seul et Masquer d'un joueur : identiques, sauf l'écart que DU-3 annonce, listé.
- [x] E.8 (fait, `c9297dc077` : mêmes composants ; boutons de 16 px comme sur la tuile, posés sur le nom ; pas de mention « masqué » faute de place, nom pâle et œil barré ; contrôle du pilote : tests de la fiche compacte verts ; taille à juger à l'écran) (né du compte rendu de E : la fiche compacte `ui/ReplayCompactCard.tsx`, employée par la colonne et la rangée au-delà de 4 contre 4, n'a ni Suivre ni œil ; sans elle, le critère 5 de la section 0 n'est pas tenu) Mêmes boutons, mêmes règles que E.3 sur la fiche compacte, avec tests.
- **Gate E** (passé : 6 064 tests du dossier verts chez l'agent, typecheck, lints, tailles ; témoins conformes) : gates communs sur `src/features/match-replay` (dossier entier : la règle des calques traverse tout).

## Piste F — page du match (Sonnet)

- [ ] F.1 `MatchNemesisCards` lit la fonction de A sur `highlight_events` au dernier instant (D-11), même filtre
      d'adversaire ; les cartes gardent leur dessin et leurs textes.
- [ ] F.2 Tests : départage par le tenant ; comptes identiques à 0.3 sur un jeu de données de test.
- **Gate F** : gates communs sur `src/features/match-view` ; 0.3 rejoué SANS serveur, en DuckDB sur une COPIE de la base partagée : décompte dédoublonné de `highlight_events` (bots exclus) égal à `killer_victim_pairs` (source de `team_tab.nemesis`) sur les 20 matchs récents et 5 matchs doublés (sinon F arrêtée, écrit ici).

## Étape I — intégration (pilote)

- [ ] I.1 Fusion des sous-branches dans `feat/830-antagonistes-tuiles` dans l'ordre A, C, E, D, B, F ; conflits
      résolus selon ce plan.
- [ ] I.2 `npx vitest run` complet une fois (`--maxWorkers=4`), `make check-types`, `make gate-push`.
- [ ] I.3 Témoins APRÈS comparés à 0.4 ; chaque écart rattaché à un item ou déclaré défaut.
- [ ] I.4 `docs/CHANGELOG.md` et notes de la 8.3.0, en français et en anglais.
- [ ] I.5 Artefact des vérifications du user (collection `verifs`) : critères 1 à 5 de la section 0 sur un match réel
      nommé par le user, avec les textes anglais de D-14.

## Étape R — revue et clôture

- [ ] R.1 Une ronde de revue adversariale du diff complet (skill `adversarial-review`, complément `levelup-revue`,
      Fable) ; P0 et P1 corrigés, P2 au backlog.
- [ ] R.2 Validation à l'écran du user (I.5) ; corrections de la liste qu'il ferme.
- [ ] R.3 Fusion dans `release/v8.3.0` (accord du user), poussée, CI lue (`gh run list --branch release/v8.3.0
      --workflow CI --limit 1`) ; sous-branches et worktrees supprimés ; `scripts/disk-hygiene.ps1 -Lot ant -Apply`.
- [ ] R.4 Entrée `.ai/thought_log.md` ; ligne du plan maître mise à jour ; tous les items statués ; plan archivé.

## Reprise de session

Lire la section « Journal d'exécution » ci-dessous, puis `git log --oneline -10` sur `feat/830-antagonistes-tuiles`
et `ListAgents` (un worktree peut être repris par une autre session). Reprendre au premier item non statué de la
piste la plus en amont.

## Journal d'exécution

- 2026-11-22 — plan écrit ; DU-1 à DU-4 confirmées par le user.
- 2026-11-22 — DU-5 tranchée par le user (bots exclus). A.4 ajoutée (entrées communes du rejeu). Lancées : A.4, F.
- 2026-11-22 — piste E close (E.8 comprise), fusionnée.
- 2026-11-22 — E.1 à E.7 faites et fusionnées (agent : 6 037 tests du dossier verts) ; E.8 ajoutée (fiche compacte sans boutons).
- 2026-11-22 — C.1 close et fusionnée ; C.2 attend DU-5 (le prédicat d'adversaire du point de vue).
- 2026-11-22 — piste A close (gate : vitest 20/20, typecheck, lint des imports, eslint), fusionnée dans la branche d'intégration.
- 2026-11-22 — étape 0 close (gate : 4 sorties écrites, section 1 corrigée, témoins contrôlés par le pilote). DU-5 posée au user. A, C.1 et E lancées.
- 2026-11-22 — dépendance levée, vérifiée : `chore/830-backlog-1118`, `bl1118/m` et `bl1118/t` ont 0 commit hors de `release/v8.3.0` ; `feat/medailles-bouclier` ne touche aucun fichier de `features/match-replay`, `features/match-view` ni `lib` en commit. Go du user. Branche `feat/830-antagonistes-tuiles` (worktree du pilote `../LevelUp-wt-ant`), étape 0 dans `../LevelUp-wt-ant-0` (`ant/0`).

## Découvertes (consignées, non traitées)

- 2026-11-22 (0.3) — 172 matchs sur 1 462 de la base du poste portent chaque élimination deux fois dans `highlight_events` (30 372 lignes sur 161 147, avril et mai 2026 surtout) ; le fil du rejeu de ces matchs les sert en double. Prod non regardée.
- 2026-11-22 (0.2) — le fil ne contient aucune élimination dont un bot est l'auteur ; le film nomme pourtant ses éliminations (`scoreActions`, `killPersonalScore`, 99 sur 99 sur un 4 contre 4), cible vide pour une victime bot.
- 2026-11-22 (0.3) — incident : le serveur lancé par l'agent de l'étape 0 sur une copie des bases a retranscodé deux captures vidéo du user (dossiers `hls` du 2026-07-19 14-51-15 et du 2026-07-29 21-21-11 réécrits, sources `.mkv` intactes). Règle ajoutée au gate F.

- 2026-11-22 — `.ai/BACKLOG.md` liste `ComputeAntagonistCounts` (`analysis/killer_victim.go:178`) sans appelant de
  production ; ce plan ne le réutilise pas (calcul web à l'instant). Son sort reste à l'entrée du backlog.
