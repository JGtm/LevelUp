# Revue adverse — feat/rejeu-refonte-ui (e87b23cc1d..a4ea770d1f, apps/ et tools/)

Une seule ronde, lecture seule. Aucun test exécuté : `apps/web/node_modules` est absent du worktree et les trois P1
sont soit de mise en page (jsdom ne les voit pas), soit démontrables par la seule lecture. `git status --short`
du worktree : vide.

## 1. Verdict

- P0 : 0. P1 : 3. P2 : 8. Soupçons : 8.
- La branche n'est pas intégrable en l'état : P1-1 (zéro inventé dans la fiche) et P1-2 (en-tête du Standard sur
  écran étroit, cadre par défaut) se corrigent en quelques lignes chacun.
- P1-3 (rangée de Grande équipe inatteignable au Cinéma) demande une décision avant correction ; les P2 se notent.

## 2. Constats

### P1-1 — « Éliminations parfaites : 0 » écrit pour un document qui ne porte pas le calque des médailles

- Où : `apps/web/src/features/match-replay/model/playerInstantStats.ts:290-292` (`medalsEstablished`), `:309`, `:351-357` ;
  `apps/web/src/lib/replay/replayNormalize.ts:234` (`medals: raw.medals ?? []`).
- Scénario : rejeu généré avant le calque des médailles (document en retard, que la page affiche avec son badge),
  joueur humain ayant réellement des médailles « Perfect ». `medalSteps` rend `[]`, `medalsEstablished` vaut vrai
  pour tout humain, `feedReadingsAt` rend `perfects: 0`, et `FicheDepuisDebut` écrit la ligne
  « Éliminations parfaites 0 » (`when(since.perfects, …)` : 0 n'est pas `null`).
- Preuve : lecture. La normalisation efface la différence entre « calque absent » et « aucune médaille » ;
  `coverage.medals` (`MedalCoverage`, `generated.ts:6590`) existe et n'a aucun lecteur web.
- Pourquoi les tests ne le voient pas : `playerInstantStats.test.ts:244` fige ce comportement
  (« un document sans médaille donne une liste vide »).
- Contrat violé : l'en-tête du module (« une valeur que le document ne permet pas d'établir est ABSENTE, jamais un zéro »).
- Correction la plus courte : n'établir les médailles que si le document déclare le calque (`doc.coverage?.medals`
  ou `doc.layers`), sinon `medals` et `perfects` à `null`.

### P1-2 — En-tête du Standard : cellule de score de 440 px fixe, aucun repli sur écran étroit

- Où : `ui/frames/ReplayFrameHeader.tsx:99` (`minmax(0,1fr) 440px minmax(0,1fr)` en style) ;
  `model/rowLayout.ts:130` (`if (frame === 'standard') return 'entete'`).
- Scénario : ouvrir un rejeu sur un téléphone de 390 px (le Standard est le cadre de chaque montage). Largeur utile
  390 − 48 = 342 px ; les pistes demandent 0 + 16 + 440 + 16 + 0 = 472 px. Les deux cellules latérales tombent à
  0 px : le titre du match disparaît, la flèche de retour (`shrink-0`) se peint sur le score, le lien
  « Fiche du match » sort par la droite, et `<main>` (`overflow-y-auto`) gagne un défilement horizontal. Sous
  820 px les cellules latérales passent déjà sous les 150 px que `cotesEnteteScore.ts` déclare nécessaires.
- Régression : avant la branche, le bandeau de score coiffait la toile à sa largeur (ancien `ReplayPage.tsx`).
- Incohérence interne : le Cinéma et le Plein cadre ont ce repli (`LARGEUR_REQUISE_ENTETE_SCORE_PX`, score sur la
  toile sous 820 px) ; le Standard en est exclu par la première ligne de `emplacementScoreDe`.
- Pourquoi les tests ne le voient pas : jsdom n'a pas de mise en page ; les tests vérifient la présence des cellules.
- Correction la plus courte : appliquer au Standard la règle de largeur d'`emplacementScoreDe`
  (`useCardRowLayout` mesure déjà le corps) et rendre l'en-tête d'une ligne quand le score va sur la toile.

### P1-3 — La rangée de Grande équipe ne peut jamais se poser au Cinéma

- Où : `apps/web/src/styles/globals.css:421-423` (`.app-shell-width { max-width: 1320px }`),
  `apps/web/src/components/shell/AppShell.tsx:111` (enveloppe de l'`Outlet`, site et bureau) ;
  `ui/frames/ReplayFrameLayout.tsx` (classes du Cinéma : aucune sortie de cette boîte) ;
  `model/cardGabarit.ts:129-134` et `model/rowLayout.ts:77` (largeur demandée : 2 × 6 × (80 + 6) + 270 + 48 = 1350 px).
- Scénario : match de Grande équipe, Cinéma, écran de 1920 px. Le corps mesuré par `useCardRowLayout` vaut au plus
  1320 px, donc `rowFitsGrandeEquipe` est faux à toute taille d'écran : toujours le repli en colonne. La rangée
  n'existe qu'au Plein cadre (fenêtre d'au moins 1350 px). Même cause : un 5 contre 5 (1528 px) ne se pose jamais au Cinéma.
- Écart non listé : G0-5 bis dit « Cinéma et Plein cadre : la rangée de K est gardée, 6 emplacements […] à 1440 px ».
  Ni le plan, ni la grille, ni l'audit ne mentionnent le plafond de 1320 px ; les cotes ont été relevées à 1440 px.
- Pourquoi les tests ne le voient pas : `rowLayout.test.ts` et `ReplayPage.row.test.tsx` donnent 1440 px au corps
  par simulacre de mesure.
- Correction la plus courte : décision du pilote. Soit le corps du Cinéma sort de `.app-shell-width`, soit la fiche
  minimale de la Grande équipe descend à 77 px (2 × 6 × 83 + 318 = 1314 px), soit l'écart est accepté et écrit.

### P2-1 — Rangée et colonne peuvent osciller au seuil (aucune hystérésis)

- Où : `hooks/useCardRowLayout.ts:37-52`, `model/rowLayout.ts` (`dispositionEquipes`, fonction pure de la largeur).
- Scénario : 4 contre 4, Cinéma ou Plein cadre ; corps entre 1286 et environ 1302 px sans barre de défilement ;
  hauteur telle que le plan est à son plancher de 360 px avec la rangée posée (environ 62 px de plus qu'en colonne).
  La rangée fait déborder la page, la barre verticale apparaît, le corps perd sa largeur, la rangée se replie, la
  barre disparaît, la rangée revient. `<main>` n'a pas de `scrollbar-gutter`.
- Limite : bande étroite ; sur le site, si le pied de page force déjà la barre, l'état est stable. Non exécuté.
- Correction : hystérésis d'une vingtaine de pixels dans `useCardRowLayout`.

### P2-2 — Tirs estimés par arme : la somme des lignes ne vaut pas le total

- Où : `model/playerInstantEstimates.ts:393` (arrondi par arme) et `:398` (arrondi du total) ; le commentaire
  `:95-96` affirme « les totaux sont les sommes de `byWeapon` ».
- Scénario : rapport du bilan 1,5 ; trois armes à 1 tir écrit. Lignes : 2 + 2 + 2 = 6 tirs ; précision écrite sur
  `round(4,5) = 5` tirs. Les deux nombres sont affichés dans la même section (`FicheTirs`).
- Correction : répartir l'arrondi (plus grand reste) ou sommer les lignes.

### P2-3 — Précision estimée au-dessus de 100 %

- Où : `model/playerInstantEstimates.ts:404` ; `ui/fiche/FicheTirs.tsx:23`, sans borne.
- Scénario : tôt dans le match, deux touches d'une arme à zone et un seul tir écrit, rapport 1 : « Précision 200 % · 2 / 1 ».
  Le type l'admet lui-même (« entre 0 et un peu plus de 1 »).
- Correction : borner l'affichage à 100 %, ou ne pas écrire la ligne quand touches > tirs.

### P2-4 — « Seul » sur un joueur pendant que la caméra en suit un autre

- Où : `model/playerPanelLogic.ts:96-104` (`toggleAction`) et `followPlayer`.
- Scénario : menu caméra, suivre X ; ouvrir la fiche de Y ; « Seul ». `followXuid` reste X : la carte ne dessine
  que Y et se cadre à chaque image sur X, qui n'est pas dessiné. Combinaison née de l'indépendance du suivi (M.1).
- Tests : chaque action est testée sur le joueur de la fiche, pas croisée avec un suivi tiers.
- Correction : « Seul » éteint le suivi d'un autre joueur (symétrique de Masquer).

### P2-5 — Plein écran du système laissé actif sur la page Standard

- Où : `hooks/useReplayFullscreen.ts:125-137`, `ui/frames/selectFrame.ts:28-30`.
- Scénario : Standard, bouton du plein écran du système (Plein cadre du geste + natif), puis touche F ou icône
  « Standard ». Le cadre se referme, le natif reste : l'état que `leaveFullFrame` déclare vouloir éviter.
- Variante : natif refusé ou API absente, puis « Quitter ». `wantedRef` reste vrai : le clic suivant sur le bouton
  ne fait rien (il ne fait que `release`).
- Correction : passer la sortie d'un Plein cadre du geste par `release`, quel que soit le chemin.

### P2-6 — Codes de décision du plan dans des commentaires de code neuf

- Où : `model/replayFrame.ts` (D-1, D-2, D-12), `model/rowLayout.ts` (D-12), `lib/replay/orientation.ts:200,205,231`
  (D-7, D-14 ; « D-7 » y désigne le seuil de l'Auto, alors que D-7 est la règle des exclusions de la fiche).
- Règle 17 de CLAUDE.md. Correction : retirer les codes.

### P2-7 — La fenêtre de fiche relit tout le lobby à chaque changement de joueur

- Où : `ui/frames/ReplayCinemaFrame.tsx:91` (`key={xuid:disposition}`), `hooks/usePlayerInstant.ts` (`docSeries`
  en `useMemo` local). La clé remonte le composant : tirs, touches et dégâts du document sont relus à chaque clic,
  ce que l'en-tête de `playerInstantSources.ts` dit éviter. Coût seulement.

### P2-8 — Clé React par nom d'arme

- Où : `ui/fiche/FicheTirs.tsx:33` (`key={w.name}`). Deux clés d'arme au même libellé donnent deux lignes de même clé.

## 3. Soupçons non démontrés

- `hooks/useResolvedTurns.ts` : `turnedBefore` survit à un changement de match quand le document est déjà en cache ;
  une carte entre 20 % et 25 % de gain s'ouvre tournée ou droite selon la précédente.
- Horloge de la barre vide après une bascule Standard / Cinéma en pause, si la toile ne change pas de taille.
- Auto pendant un export : la résolution lit la boîte de l'écran ; un redimensionnement de fenêtre pendant l'export
  peut retourner le plan.
- `equipmentGestures.ts` `gestureOwner` : lâcher daté à fin de vie + 2 alors que le slot est réattribué à fin de vie + 1.
- Deux scores « à l'instant » pour un joueur : infobulle de la tuile (calque de score, sans garde) et fiche (points par action, gardée).
- Barre d'une ligne sur écran étroit : les commandes fixes dépassent la largeur, la frise tombe à zéro.
- `replay_medal_labels.go` `medalLabelOf` : `Difficulty` prise à la première langue d'un parcours de map (ordre non fixé).
- `playerInstantStats.ts` : `respawns` compte une vie entièrement antérieure au début du match (la moyenne l'écarte).

## 4. Couverture

- Lus en entier : `playerInstantStats`, `playerInstantSources`, `killBreakdown`, `playerInstantEstimates`,
  `playerInstantView`, `playerThrowsPickups`, `playerEquipmentStats`, `equipmentGestures`, `playerMovementStats`,
  `playerInstantMedals`, `matchMedals`, `usePlayerInstant`, `playerPanelLogic`, `usePlayerPanel`,
  `useReplayPlayerFocus`, `openPlayerRing`, `useReplayOpenPlayer`, `ReplayPage`, tout `ui/frames/`, `useCardRowLayout`,
  `useReplayFrame`, `replayFrame`, `rowLayout`, `useReplayFullscreen`, `useResolvedTurns`, `useReplayOrientation`,
  `useAnchoredPanel`, `ReplayPlayerPanel`, `ReplayPlayerPanelBody`, `ReplayPlayerPopover`, `ReplayFeedOrPanel`,
  `ReplayCameraMenu`, `cameraMenu`, `ReplayTeamsRow`, `useCardScene`, `ReplayFrameButtons`, `ReplayCaptureButtons`,
  `useFeedView`, `planFeed`, `ReplayFeedOverlay`, `ReplayFeedExtract`, `FicheDepuisDebut`, `FicheEliminations`,
  `FicheTirs`, `FicheIdentite`, `FicheMedailles`, `playerInstant.i18n`, `replay_medal_labels.go`, `killsource_class.go`.
- Lus par leur diff : `ReplayCanvas`, `useReplayView`, `replayPreferences`, `orientation.ts`, `equipmentUsageLogic`,
  `equipmentKeptLogic`, `killFeedLogic`, `replayModel`, `_momentum`, `feedZone`, `useReplayShortcuts`,
  `useReplayTimeline`, les gardes modifiés, tout le Go hors tests, `globals.css`.
- En diagonale : `ReplayRosterColumn`, `ReplayTransportCompact`, `replayTransportCommands`, `playerNowReadings`, `roster.i18n`.
- Pas lus : `ReplayRosterRow`, `ReplayRosterRowDetail`, `ReplayRosterCells`, `ReplayCompactCard`, `ReplayPlayerCard`,
  `ReplayCardFx`, `ReplayPlayerNow`, `ReplayPlayerPicker`, `ReplayTimelineCompact`, `fragMarks`, `ReplayKillFeed`,
  `ReplayFeedLines`, `ReplayKillLineParts`, `ReplayScoreBanner`, `ReplayInventoryRow`, `ReplayWeaponsRow`,
  `cardGabarit` (hors cotes de rangée), les fichiers de cotes, `frames.i18n`, `ReplayViewMenu` et le panneau Vue,
  `ReplayDimensionSwitch`, `ReplayRotatedButton`, `openapi.yaml` et `generated.ts` (présence des deux champs seulement).
- Tests : aucun fichier de test neuf lu en entier (environ 12 000 lignes). Suppressions : report vérifié par sondage
  de 14 intitulés de l'ancien `ReplayTeams.test.tsx`, 12 retrouvés ; pas de contrôle cas par cas. Gardes : aucun
  relâché, cliquets resserrés. Aucun `skip`, `only`, `eslint-disable` ni `any` ajouté.
- Règles vérifiées par recherche sur les lignes ajoutées : aucune couleur en dur ni classe Tailwind de couleur,
  aucune bordure gauche colorée neuve hors celles déjà notées au plan, « KDA » en anglais seulement, aucun
  `useQuery`, `fetch` ni clé de requête en ligne, aucun fichier neuf au-delà de 500 lignes, clés de premier niveau
  des quatre sous-dictionnaires toutes lues en production.
- Conformité aux maquettes (grille V1 à V17) : non relue ligne à ligne.
- Go : aucun défaut recevable sur les couches, la journalisation, l'i18n, le contrat ni le lecteur de chaque champ servi.
