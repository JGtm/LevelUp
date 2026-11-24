# Audit de conception 0.3 — refonte de la page du rejeu (2026-11-17)

Lecture seule, sur `release/v8.3.0` (`3ae75c78f5`). Plan audité : `.ai/PLAN_REFONTE_UI_REJEU_2026-11-15.md`
(amendé le 2026-11-17). Maquettes lues : A, J, K, L, D, H. Chemins relatifs à
`apps/web/src/features/match-replay/` sauf mention. « Lignes comptées » = ce que compte la règle ESLint
`max-lines` (500, hors commentaires et lignes vides, `apps/web/eslint.config.js:90`), mesuré par ESLint.
Le worktree de l'extraction du moteur 2D n'a pas été lu : les lignes de `ui/ReplayCanvas.tsx` sont celles
d'avant l'extraction.

## 1. Plan contre code

### 1.1 Section 2 du plan (« Ce qui existe »)

| Affirmation | Verdict | Ce que le code dit |
|---|---|---|
| Page : grille `xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)]`, score puis toile à gauche, fiches puis fil ou panneau à droite | exact | `ReplayPage.tsx:281`, `313-389`, `416-461` |
| `ReplayPage.tsx` « frôle le seuil » | à nuancer | 469 lignes brutes, 295 comptées : la règle de lint laisse 205 lignes ; le seuil de la règle 5 se lit sur les lignes brutes |
| Plein écran : deux étages en un geste, couches d'Échap, coupé pendant un export, touche F | exact | `hooks/useReplayFullscreen.ts:66-67`, `100-168` ; `hooks/useReplayShortcuts.ts:131` |
| Toile « au-delà du seuil, gelée par la baseline » | faux | 810 lignes brutes, **498 comptées** : sous le plafond de 2 lignes, aucune exemption (`eslint-disable max-lines` absent du fichier) ; il n'y a pas de baseline pour ce fichier |
| Toile « centrée, à la largeur de la carte » | périmé | la toile prend la largeur du BLOC depuis le 2026-09-03 (`hooks/useReplayView.ts:195`) ; seule sa hauteur suit la carte (`181-187`) ; `mapOverlays` se pose donc sur toute la largeur du bloc (`ui/ReplayCanvas.tsx:754-780`) |
| Légende de chaleur en bas à gauche du bloc, zoom en bas à droite, barre bord à bord | exact | `ui/ReplayCanvas.tsx:791-792`, `798-805` ; `ui/ReplayHeatmapLegend.tsx:40` ; `ui/ReplayZoomControl.tsx:39` |
| Barre : frise, lecture, ±10 s, vitesse, son, capture, export, plein écran | incomplet | il y a aussi l'interrupteur de lecture automatique (`ui/ReplayTransport.tsx:128-142`) et « Recommencer » (`148-150`) |
| Taille du terrain : pas de 8 px, délai de 120 ms, plafond 720 px | exact | `hooks/useReplayViewport.ts:62-64`, `80-83` ; `hooks/useReplayView.ts:62` |
| Orientation retenue par carte, aucune règle automatique | exact | `settings/useReplayOrientation.ts:27-45` ; `settings/replayPreferences.ts:168-187` ; aucun lecteur hors du rejeu |
| Fiches : `GABARIT_NORMAL` (235 px), `GABARIT_COMPACT` (115 × 62) | à corriger | aucun des deux gabarits ne porte de largeur (`model/cardGabarit.ts:31-85`) ; 235 n'existe qu'en commentaire, 115 est un littéral de classe (`ui/ReplayTeams.tsx:97`) |
| Jauges : bouclier 5 px sur santé 3 px | exact | `model/cardGabarit.ts:99-100` (4 et 2 px en compact, `128-129`) |
| Score : lecture pure, rien en chacun pour soi | exact | `ui/ReplayScoreBanner.tsx:81-96` |
| Fil, filtre, lieu du tueur et jumeau Go | exact | `ui/ReplayKillFeed.tsx:135-144` ; `model/killerZone.ts:183`, `223` |
| Fiche détaillée, panneau ancré, i18n | exact | `ui/ReplayPlayerPanel.tsx` ; `settings/useAnchoredPanel.ts` ; `i18n/i18n.ts:18-21` |
| « Sous la zone de lecture : carte de chaleur, usage de l'équipement » (D-11, S.4) | **faux** | la page s'arrête à la grille (`ReplayPage.tsx:464-468`). La carte de chaleur est une section du panneau « Vue » (`settings/ReplayViewPanel.tsx:36`) ; l'usage de l'équipement est monté par la page du match (`apps/web/src/features/match-view/MatchViewTabArsenal.tsx:166`) |

### 1.2 Items des lots

| Item | Verdict | Constat |
|---|---|---|
| A.1 | faisable | Le module pur ne doit contenir aucun test de la touche Échap : le garde-rail n'admet qu'un fichier qui la lit sans être une couche (`hooks/escapeLayers.guard.test.ts:33-38`). À écrire dans l'item : Échap en Plein cadre rend le cadre d'où l'on vient ; en Cinéma, Échap ne fait rien (T en sort) |
| A.2 | plus gros | La touche F arrive aux raccourcis par trois relais : `ui/ReplayCanvas.tsx:711`, `hooks/useReplayTimeline.ts:95`, `134`, `224`, `hooks/useReplayShortcuts.ts:46`, `165`. Ajouter T touche donc aussi `useReplayTimeline.ts` et une ligne de `ReplayCanvas.tsx` (fichier d'E.1) ; `ui/ReplayTransport.tsx:261-280` lit `fullscreen.active`. L'annonce de redimensionnement (`hooks/useReplayFullscreen.ts:157-162`) doit valoir pour toute bascule de cadre, Cinéma compris. L'écoute d'Échap reste dans `useReplayFullscreen.ts` |
| A.3 | **à revoir** | Deux composants de cadre qui montent chacun la toile la DÉMONTENT à chaque bascule : position de lecture, son, zoom, export en cours et cuissons vivent dans `ReplayCanvas` (`ui/ReplayCanvas.tsx:263-268`, `282`, `700-735`). Aujourd'hui la bascule ne change qu'une classe sur le même nœud (`ReplayPage.tsx:280-284`). Il faut UN gabarit à zones, la toile toujours au même endroit de l'arbre (section 2). Autres points : l'en-tête actuel est un fil d'Ariane d'une ligne partagé avec la page du match (`apps/web/src/features/match-view/MatchHeader.tsx:86-130`), non réutilisable pour A, K, D ; le titre « Rejeu » et le badge d'administration (`ReplayPage.tsx:201-219`), le bandeau de régénération (`240`), l'avis « instant non calé » (`295-301`) et les états de chargement (`241-258`) n'ont pas de place dans les maquettes ; le garde `ui/rosterHeight.guard.test.ts:26-66` lit le texte de `ReplayPage.tsx` et tombe au déplacement ; la disposition sous 1280 px (`ReplayPage.tsx:274-275`) n'est pas dite |
| S.1 | faisable | Le bandeau porte une marge basse (`ui/ReplayScoreBanner.tsx:98`), des barres de 28 px (`244`, 24 px dans A), les pastilles de manche au-dessus (`158`) et la garde de colline dessous (`272`) : il lui faut des cotes nommées par emplacement, sans toucher `readScoreBanner` |
| S.2 | plus gros | Un gabarit « ligne » est recommandé (section 2). Les effets à garder (D-17) sont des fonctions privées de `ui/ReplayPlayerCard.tsx:310-341` et `544-606` ; la scène d'effets est bâtie dans `ui/ReplayTeams.tsx:141-176` : les deux s'extraient AVANT S.2. Lectures déjà là : `model/playerCardReadings.ts:90`, `model/inventoryReading.ts:90`, `model/equippedLogic.ts:66`. La notion de joueur « suivi » n'existe pas hors fiche ouverte (voir E.4) |
| S.3 | **écart** | La ligne de frag actuelle (`ui/ReplayFeedLines.tsx:292-312`) : horloge (gouttière de 42 px, `model/feedRow.ts:25`), lieu du TUEUR juste après (5,5 rem, `ui/ReplayKillLineParts.tsx:34`), tueur, arme, victime, badge « porteur », médailles, assistance. La maquette (A-14) : instant, tueur, arme, victime, lieu, à colonnes fixes, sans médaille ni assistance. « Seul le lieu change » ne peut pas tenir (ordre, largeurs, trois éléments hors maquette). À garder : la ligne cliquable (`321-343`) et la ligne choisie (`323-329`) |
| S.4 | **écart** | Rien sous le lecteur (1.1). « La fixation suit le composant qui reste » est infaisable : `replayTeams.4v4.html` fixe le DOM de `ReplayTeams` au gabarit normal (`ui/ReplayTeams.dom4v4.test.tsx`), que ni la colonne ni la rangée ne rendent. Après S.4 et B.2 perdent leur appelant : `ui/ReplayTeams.tsx`, `ui/ReplayPlayerCard.tsx`, `model/cardDensity.ts`, les deux gabarits actuels, `EliminatedRow` (`ui/ReplayVitality.tsx:154`), les sept `ui/ReplayTeams.*.test.tsx`, `ui/ReplayPlayerCard.compact.test.tsx`, les deux fixations, `test/replayTeamsFromBoard.tsx`. Cette suppression (règle 7) est un item à part entière, avec le report des cas de test sur les lignes et les tuiles |
| B.1 | infaisable tel qu'écrit | Aucune largeur à lire dans le gabarit (1.1) : ajouter des cotes nommées (largeur minimale de tuile, écart, largeur du score, marges). Avec 115 px : un 4v4 demande 8 × 115 + 270 + 8 × 6 + 2 × 24 = 1286 px (il ne tient donc pas à 1280), un 5v5 1528 px (il ne tient pas à 1440). Le plan ne donne pas l'attendu de ces deux cas |
| B.2 | **écart** | Voir 4.1. La grille dépend de l'effectif (3v3, 4v4, 5v5) : colonnes par `style`, comme `ui/ReplayTeams.tsx:220` (une valeur arbitraire Tailwind interpolée ne produit aucune règle) |
| C.1 | exact | `nameZone` (`model/killerZone.ts:183`) sert tel quel. `killerZones` (`223`) et `hooks/useKillerZones.ts` perdent leur appelant. Recommandation : `victimZones` dans le même fichier, renommé `model/feedZone.ts` ; le libellé `killFeedKillerZone` (`i18n/fr.ts:90`) devient celui du lieu de la victime |
| C.2 | plus gros | (1) Légende et zoom sont rendus par la toile (`ui/ReplayCanvas.tsx:791-792`) : C.2 dépend d'E.1. (2) Le filtre vaut « Tout » par défaut (`model/feedFilterLogic.ts:39`) et le Cinéma n'a pas de menu : prises d'armes et objectifs chasseraient les éliminations des 5 lignes (question 5). (3) La liste filtrée se calcule dans le composant (`ui/ReplayKillFeed.tsx:135-144`) : à sortir dans un hook, trois lecteurs l'attendent (colonne, 2 lignes de J, plan). (4) Le fil posé sur le plan ne doit pas capter le pointeur (survol, glisser, molette de la carte) |
| C.3 | exact | `settings/ReplaySettingsToggle.tsx:75` ; clé dans `settings/replayPreferences.ts` |
| D.1 | **écart** | Le panneau n'a PAS de bloc « À l'instant » : tête, trois actions, listes de totaux (`ui/ReplayPlayerPanel.tsx:117-159` ; `model/playerPanelLogic.ts:17-22`). L'instant lu est écrit dans la tête (`107`, `126-128`). Extraire « sans changer le rendu » donne un corps sans « À l'instant » : le bloc dessiné dans J et L est à construire (item manquant, sans attendre 0.4 puisqu'il est dessiné) |
| D.2 | plus gros | `ReplayFeedOrPanel` ne remplace que la cellule du fil, les équipes restent au-dessus (`ui/ReplayFeedOrPanel.tsx:27-36` ; `ReplayPage.tsx:412-461`) ; J remplace TOUTE la colonne. Le fichier `ui/ReplayPlayerPanel.tsx` doit rester celui qui écoute Échap (liste du garde, `hooks/escapeLayers.guard.test.ts:74-84`). Le focus est rendu à la fiche par `data-replay-player-card` (`ui/ReplayPlayerPanel.tsx:72`, `ui/ReplayPlayerCard.tsx:167`) : lignes et tuiles doivent porter l'attribut |
| D.3 | plus gros | `useAnchoredPanel` aligne le bord DROIT du panneau sur son déclencheur et borne la hauteur à la fenêtre (`settings/useAnchoredPanel.ts:65-76`). L aligne le bord GAUCHE sur la tuile, part du haut du plan et s'arrête au bas du plan (`CinemaRangeeFiche.dc.html:91`) : le hook reçoit deux options (côté d'alignement, élément borne). Le pion cerclé est un nouveau peintre de la scène, hors du JSX (fichier du moteur extrait) ; la sélection actuelle est un anneau `ring-2 ring-ring` (`ui/ReplayPlayerCard.tsx:184`) |
| E.1 | **écart** | `transport` ne peut pas être un élément fourni par le cadre : lecture, frise, horloge, son, capture, zoom, orientation et vitesse naissent dans la toile (`ui/ReplayCanvas.tsx:265`, `273`, `282`, `314`, `700-735`). Il faut une fonction de rendu qui reçoit un objet de commandes. K-8 (le plan remplit la hauteur, barre en bas, hors du bloc bordé) touche aussi `hooks/useReplayView.ts:181-187` (hauteur utile) et la mesure par soustraction de `hooks/useReplayViewport.ts:154-155` : ces deux hooks entrent dans la table des fichiers partagés (E.1, puis F.2, puis G.1) |
| E.2 | exact | Sortir aussi les icônes (`ui/ReplayTransport.tsx:283-469`), partagées avec la barre compacte |
| E.3 | écart de critère | « Mêmes commandes que le Standard » est faux pour K : ni lecture automatique, ni « Recommencer », ni repères de médias, ni piste de score, ni repli des pistes (question 7). Réutilisables tels quels : l'horloge écrit déjà « 3:51 / 7:45 » (`hooks/useReplayClock.ts:84`) ; le curseur doit rester le même champ avec `data-replay-timeline` (`ui/ReplayTimelineTracks.tsx:328`, `hooks/useReplayShortcuts.ts:36`, `141`). K dessine les frags de TOUS les joueurs à la couleur du camp : cette liste existe (`hooks/useReplayTimeline.ts:161`, `347`) mais n'est pas dans l'objet `timeline` (`230-257`) : à y ajouter |
| E.4 | plus gros | Suivre n'existe que fiche ouverte : une bascule est ignorée panneau fermé (`model/playerPanelLogic.ts:70`), ouvrir une autre fiche ou fermer remet tout à zéro (`62-66`). Suivre depuis le menu caméra demande un joueur suivi indépendant de la fiche : changement du modèle pur, à faire avant S.2, B.2 et E.4. La liste « Point de vue » existe (`hooks/useReplayTimeline.ts:196-199`, `ui/ReplayViewpointSelect.tsx:53`) |
| F.1 | exact | `lib/replay/orientation.ts` (types en `30-33`) |
| F.2 | plus gros | `readOrientation` rend 0 quand rien n'est retenu (`settings/replayPreferences.ts:179-181`) : il faut un lecteur qui distingue « absent ». L'orientation naît dans la toile (`ui/ReplayCanvas.tsx:273`), la boîte en `268`, les bornes de la carte dans `hooks/useReplayView.ts:169` : l'Auto se résout dans `useReplayView`. Deux gardes lisent le texte `turns: orientation.turns` (`test/orientationWiring.guard.test.ts:20-27`, `test/playerFocusWiring.guard.test.ts:42`) : à mettre à jour dans le même commit. Un quart de tour remet le zoom à zéro (`hooks/useReplayView.ts:228`) : l'hystérésis de D-14 en devient obligatoire |
| F.3 | exact | À gauche du bouton « Vue » (`settings/ReplayViewMenu.tsx:30`) |
| G.1 | exact | `fitCanvasHeight` lit la constante importée (`hooks/useReplayViewport.ts:53-57`, `80-83`) : elle devient un paramètre |

## 2. Découpage des composants

### 2.1 Principe : un gabarit à zones, la toile montée une seule fois

`ui/frames/ReplayFrameLayout.tsx` porte la structure, la même dans les trois cadres : zones `tete`, `rangee`,
`scene`, `colonne`. Le cadre ne change que des classes (table fermée par cadre, classe du Plein cadre comprise)
et le contenu des zones. `scene` reçoit toujours le même élément, au même endroit de l'arbre : React garde
l'instance de la toile. Test obligatoire en A.3 : « basculer de cadre ne remonte pas la toile ».

### 2.2 Fichiers à créer

| Fichier | Responsabilité | Item | Taille visée |
|---|---|---|---|
| `model/replayFrame.ts` | type du cadre, transitions pures | A.1 | 60 |
| `ui/frames/ReplayFrameLayout.tsx` | les zones et leurs classes par cadre | A.3 | 120 |
| `ui/frames/ReplayStage.tsx` | l'unique montage de la toile, déplacé de `ReplayPage.tsx:321-389`, avec les surcouches de carte ; reçoit les fonctions de rendu de la barre et des superpositions | A.3, E.1 | 160 |
| `ui/frames/ReplayFrameHeader.tsx` | les trois en-têtes (A, K, D) ; porte le titre de page et le badge d'administration | A.3, S.1 | 150 |
| `ui/frames/ReplayStandardFrame.tsx` | contenu des zones du Standard (colonne, zone de lecture actuelle) | A.3, S.4, D.2 | 150 |
| `ui/frames/ReplayCinemaFrame.tsx` | contenu des zones du Cinéma et du Plein cadre (rangée ou colonne de repli, barre compacte, fil sur le plan) | A.3, B.2, C.2, D.3, E.3 | 200 |
| `hooks/useCardScene.ts` | âges de lecture, scène d'effets, calque de score : sorti de `ui/ReplayTeams.tsx:141-176` | avant S.2 | 70 |
| `ui/ReplayCardFx.tsx` | couches d'effets sous et sur une fiche, sorties de `ui/ReplayPlayerCard.tsx:310-341` et `490-606`, rendu inchangé | avant S.2 | 170 |
| `ui/ReplayRosterColumn.tsx` | camps, en-têtes, défilement, places | S.2 | 150 |
| `ui/ReplayRosterRow.tsx` | une ligne joueur, ligne d'attente comprise | S.2 | 200 |
| `ui/ReplayRosterRowDetail.tsx` | bloc déplié du joueur suivi (dessin de H) | S.2 | 90 |
| `model/rowLayout.ts` | règle D-12 sur les cotes nommées | B.1 | 50 |
| `ui/ReplayTeamsRow.tsx` | la rangée : tuiles, score, ligne d'objectif | B.2 | 120 |
| `ui/ReplayPlayerTile.tsx` | la tuile de K, tuile d'attente comprise | B.2 | 200 |
| `hooks/useFeedView.ts` | fil complet, filtre, lignes visibles à l'instant : sorti de `ui/ReplayKillFeed.tsx:135-144` | S.3 | 70 |
| `hooks/useFeedZones.ts`, `model/feedZone.ts` | lieu de la victime (remplacent `useKillerZones.ts`, `killerZone.ts`) | C.1 | 50, 140 |
| `ui/ReplayFeedOverlay.tsx` | 5 lignes sur le plan | C.2 | 90 |
| `ui/ReplayFeedExtract.tsx` | 2 lignes au pied de la fiche J | D.2 | 60 |
| `ui/ReplayPlayerPanelBody.tsx` | assemble les sections de la fiche | D.1, T.5 | 200 |
| `ui/ReplayPlayerNow.tsx` | bloc « À l'instant » dessiné dans J et L | item à ajouter | 150 |
| `ui/ReplayPlayerInstantSections.tsx` | sections du lot T | T.5 | 220 |
| `ui/ReplayPlayerPicker.tsx` | sélecteur des joueurs de J | D.2 | 70 |
| `ui/ReplayPlayerPopover.tsx` | fenêtre L | D.3 | 120 |
| `ui/ReplayTransportIcons.tsx` | icônes de la barre | E.2 | 190 |
| `ui/ReplayCaptureButtons.tsx`, `ui/ReplayFrameButtons.tsx` | capture et export ; boutons de cadre | E.2 | 110, 90 |
| `ui/ReplayTransportCompact.tsx`, `ui/ReplayTimelineCompact.tsx` | barre d'une ligne ; frise compacte | E.3 | 200, 120 |
| `ui/ReplayCameraMenu.tsx`, `ui/ReplayRotatedButton.tsx` | menu caméra ; « Plan tourné » | E.4, F.2 | 130, 40 |
| `ui/ReplayDimensionSwitch.tsx` | emplacement 2D / 3D | F.3 | 40 |
| `model/playerInstantStats.ts`, `killBreakdown.ts`, `playerMovementStats.ts`, `playerInstantEstimates.ts`, `playerThrowsPickups.ts` | lectures pures du lot T | T.1 à T.4, T.7 | 80 à 200 |

Fichiers existants : `ReplayPage.tsx` descend vers 180 lignes comptées (chargement, modèle, choix du cadre).
`ui/ReplayCanvas.tsx` ne reçoit que les deux fonctions de rendu et perd le montage de la barre, du zoom et de
la légende. `ui/ReplayTransport.tsx` passe de 340 à environ 200 lignes comptées. `model/cardGabarit.ts`
(63 comptées) reçoit deux gabarits, « ligne » (A-9) et « tuile » (K-3, K-4), et les cotes de rangée ; l'union
`bodyPx` s'étend, ce qui oblige à compléter chaque table fermée à la compilation. Le garde
`ui/cardGabarit.guard.test.ts` ajoute les nouveaux composants à sa liste. `model/playerPanelLogic.ts` reçoit
le joueur suivi indépendant de la fiche.

### 2.3 Dictionnaires

`i18n/fr.ts` : 490 lignes comptées ; `i18n/en.ts` : 487 ; `i18n/i18nContract.ts` : 339. Il reste 10 lignes.

1. **Avant tout item qui ajoute une chaîne** (Haiku, mécanique) : sortir `equipmentUsage` (`i18n/fr.ts:432`,
   `i18n/en.ts:427`, contrat `i18n/i18nContract.ts:44-80`) vers `i18n/equipmentUsage.i18n.ts`, sur le patron de
   `i18n/playerPanel.i18n.ts`. Environ 60 lignes rendues par langue, aucun appelant ne change.
2. **Dans le même commit, par le pilote** : créer les sous-dictionnaires des lots, contrat et deux langues,
   et les monter (une ligne dans `fr.ts`, `en.ts` et le contrat pour chacun) : `i18n/frames.i18n.ts` (lots A,
   E, F, C.3), `i18n/roster.i18n.ts` (S.2, B.2, sélecteur de J), `i18n/playerInstant.i18n.ts` (bloc
   « À l'instant », lot T). Le fil écrit dans `i18n/feedLines.i18n.ts` (44 lignes), la fiche dans
   `i18n/playerPanel.i18n.ts` (143 lignes).
3. Ensuite aucun lot ne touche plus `fr.ts`, `en.ts` ni le contrat : la règle « un écrivain à la fois » devient
   une règle par sous-dictionnaire.

## 3. Risques

| Risque | Ce qui le couvre, ou ce qu'il faut ajouter |
|---|---|
| Toile démontée à la bascule de cadre | Rien aujourd'hui. Test à écrire en A.3 (2.1) |
| Couches d'Échap | `hooks/escapeLayers.guard.test.ts` : attribut écrit en clair dans tout fichier qui écoute Échap ; un seul fichier exempté. Couches neuves à signaler : fenêtre L, menu caméra. Test à ajouter : Échap ferme la fenêtre, puis la fiche, puis le Plein cadre, une frappe par couche. Le dialogue d'export n'est pas une couche (aucune écoute dans `export/ReplayExportDialog.tsx`) : état d'aujourd'hui, noté en 7 |
| Export vidéo | Le clip se dessine à son propre format (`hooks/useReplayView.ts:200-202`, `232-240`, `255`) et ne repeint que l'écran de fin et la fin de manche (`export/exportOverlayPanels.ts:1-27`) : fiches, fil et barre n'y entrent pas, avant comme après. À tenir : aucune bascule pendant un export (`hooks/useReplayFullscreen.ts:101`, `117`), touche T et nouveaux boutons compris ; test à ajouter en A.2. Les commandes de lecture neutralisées (`ui/ReplayTransport.tsx:87`, `105`) le sont aussi dans la barre compacte (test prévu en E.3) |
| Couches z | Aujourd'hui : surcouches de carte 10 sans pointeur (`ui/ReplayVictoryOverlay.tsx:173`), bouton « Vue » et menus 20, média agrandi 30, cadre 40 (`hooks/useReplayFullscreen.ts:66-67`), panneau « Vue » en portail 50 (`settings/ReplayViewPanel.tsx:334`). À poser : fil sur le plan SOUS les surcouches (l'écran de fin le couvre) et sans pointeur ; fenêtre L en portail à 50 |
| Portails | `settings/useAnchoredPanel.ts` se recale sur `resize` et `scroll` : toute bascule de cadre doit annoncer un redimensionnement (A.2). Options neuves pour L (D.3) couvertes par `settings/useAnchoredPanel.test.ts` |
| Plein écran natif | `fullscreenchange` reste la vérité (`hooks/useReplayFullscreen.ts:127-140`). Tests de `hooks/useReplayFullscreen.test.tsx` à étendre : natif seul depuis chaque cadre, chute du natif sans quitter le Plein cadre, double appui |
| Redimensionnement et cuissons | Une bascule est un changement de largeur ET de hauteur : une recuisson des quatre calques après 120 ms. La mesure par soustraction suppose un conteneur à la taille de son contenu (`hooks/useReplayViewport.ts:154-158`) : un conteneur étiré par la mise en page fige la hauteur à sa valeur de départ. Tests de `hooks/useReplayViewport.test.ts` à étendre ; M-1 pour la mémoire |
| Raccourcis | Un seul écouteur (`hooks/useReplayShortcuts.ts:54-56`) : T s'y ajoute, jamais ailleurs. Une liste déroulante du menu caméra coupe les raccourcis tant qu'elle a le focus : le rendre après le choix, comme `ui/ReplayViewpointSelect.tsx:20-27`. Lignes, tuiles et boutons neufs portent `PRESS_GUARDS` (`ui/replayPressGuards.guard.test.ts`). Test à ajouter : le curseur compact porte `data-replay-timeline` |
| Gardes qui lisent du texte | `ui/rosterHeight.guard.test.ts` (texte de `ReplayPage.tsx`), `test/orientationWiring.guard.test.ts`, `test/playerFocusWiring.guard.test.ts`, `layers/sceneBinding.guard.test.ts`, `ui/vehicleWiring.guard.test.ts` (texte de `ReplayCanvas.tsx`) : chaque item qui déplace le code visé déplace le garde dans le même commit |
| Carte plus petite en Standard sur grand écran | Conséquence de A : à 1920 px de large la carte passe d'environ 1380 px à 840 px. Le Cinéma est la réponse ; à dire au user avant R.4 |
| Rendu toutes les 150 ms | La page se rend à chaque publication d'image : lignes, tuiles et fil sur le plan suivent `ui/ReplayTeams.perf.test.tsx`, à reporter sur les nouveaux composants |

## 4. Grille de conformité contre le code

### 4.1 K-3 et K-4 : fiche compacte actuelle contre tuile de K

| Point | Fiche compacte actuelle | Tuile de K |
|---|---|---|
| Hauteur, lignes | 62 px, trois lignes (`ui/ReplayPlayerCard.tsx:115-125`) | 58 px, deux lignes |
| Largeur | fluide, 115 px au minimum, par la grille (`ui/ReplayTeams.tsx:97`) | fluide, parts égales : déjà compatible |
| Pastille du camp | absente | carré de 8 px devant le nom |
| Nom | capitales, gras, 11,5 px, seul sur sa ligne (`ui/ReplayPlayerCard.tsx:117-118`, `139`) | casse du nom, 12 px, graisse moyenne |
| F / M / A | ligne 2, à droite des jauges (`120`, `402-405`) | ligne 1, à droite du nom |
| Jauges | deux barres, 4 et 2 px | une barre de 3 px, donc deux barres (V4) |
| Arme | cellule de 48 px en ligne 3 (`model/cardGabarit.ts:118-119`) | aucune |
| Grenades | type sélectionné, sans nombre (`model/cardGabarit.ts:122`) | icône et nombre |
| Équipement | vignette de 16 px, charges en infobulle | icône, éteinte sans équipement |
| Mort | bloc hachuré « ÉLIMINÉ » et compte à rebours (`ui/ReplayVitality.tsx:198`) | « Réapparition 5,8 s » à la place des jauges, reste à 40 % |
| Joueur suivi | aucun état | bordure à la couleur de suivi, nom en gras |
| Joueur ouvert | anneau (`ui/ReplayPlayerCard.tsx:184`) | fond plus clair, bordure claire |

Pour avoir la rangée de K : un gabarit « tuile » à deux lignes rendu par `ui/ReplayPlayerTile.tsx`, sur les
mêmes lectures et les mêmes couches d'effets que la fiche (2.2). Effets gardés sans perte : éclats de mort et de
réapparition, équipement qui s'allume, nuage et éclairs, contour du capteur, croix de réparation, fourreau de
translocation, filigrane de porteur. Deux effets de D-17 n'ont plus de support dans la tuile de K : le
glissement au changement d'arme (plus de cellule d'arme, `ui/ReplayWeaponsRow.tsx:132-164`) et le mot
« ÉLIMINÉ ». Proposition : la bande hachurée reste sur la ligne 2 et porte « Réapparition 5,8 s » ; le
glissement d'arme reste dans la colonne du Standard et dans la fiche. C'est la question 1.

### 4.2 K-7, A-2, L-4

- **K-7.** Le document porte la cible : `scoreTimeline.targetScore` (`apps/go-api/internal/domain/replaydoc/objectives.go:20`,
  `apps/web/src/lib/api/generated.ts:12113`), lue par `model/scoreBannerLogic.ts:186` et `216`. Trois limites :
  absente quand la variante n'est pas dans la table mesurée ; sur un mode à manches c'est le plafond d'UNE
  manche ; aucune unité (« éliminations ») ni dans le document ni dans l'en-tête (`score_points_label` est le
  score « X - Y », `apps/go-api/internal/domain/match_view.go:69-73`). Écrire l'unité demande un libellé par mode
  dans les fichiers de configuration du titre et un champ d'en-tête, donc du Go hors T.6 : question 6.
- **A-2.** Les cinq données sont affichées aujourd'hui, sur UNE ligne : joueur, mode et carte
  (`apps/web/src/features/match-view/MatchHeader.tsx:110-118`), date et file (`ui/ReplayMatchRecall.tsx:43-55`). La forme sur deux
  lignes n'existe pas. Le joueur est écrit par son identifiant d'adresse, pas par son nom affiché.
- **L-4.** L'instant lu est écrit, mais dans la tête de la fiche, après les compteurs (`ui/ReplayPlayerPanel.tsx:107`,
  `126-128`). Il n'y a ni titre « À l'instant », ni bouclier et santé chiffrés, ni arme nommée : tout le bloc est
  à construire (D.1). J n'écrit pas le nom de l'arme (icône et munitions, `FicheJoueur.dc.html:170`), L l'écrit
  (`CinemaRangeeFiche.dc.html:215`).
- **L-5** n'est pas « inchangé » : aujourd'hui cinq listes titrées (`model/playerPanelLogic.ts:310-334`), dans L une
  seule section et des barres par arme. Le contenu reste (D-23), la forme change : à dessiner dans 0.4.
- **H-3 et « Quitter »** sont dessinés (`RoninCinema.dc.html:207`, `PleinCadre.dc.html:119`) : la liste « sans dessin » les
  comptait à tort.

### 4.3 Les éléments sans dessin : une forme pour chacun

| N° | Élément | Forme proposée | Choix à montrer |
|---|---|---|---|
| 1 | Icônes de cadre | trois boutons ronds fantômes au format du bouton plein écran actuel (`ui/ReplayTransport.tsx:271-275`), 30 px dans la barre compacte ; cadre actif en `bg-accent`, `aria-pressed` ; infobulles « Cinéma (T) », « Plein cadre (F) », « Plein écran du système » | non |
| 2 | « Plan tourné » | dessiné dans H : capsule de 30 px, icône de rotation, libellé, bordure à la couleur de suivi, infobulle qui dit le gain | non |
| 3 | « Quitter » | dessiné dans D : `Button` bordé, croix, « Quitter », capsule « Échap » | non |
| 4 | Menu caméra | petit panneau au-dessus de l'icône, sur le patron de `ui/ReplaySpeedMenu.tsx:55-56` ; deux listes déroulantes titrées, « Suivre » (avec « Aucun ») et « Point de vue » (`ui/ReplayViewpointSelect.tsx`) | oui (question 9) |
| 5 | Option « Fil des éliminations », position « Auto » | `SettingsToggle` dans le panneau « Vue » ; « Auto » en tête du segmenté d'orientation (`settings/ReplayOrientationSection.tsx:24`, `36-41`) | non |
| 6 | Chacun pour soi | colonne sans en-tête d'équipe ni pastille de camp, lignes dans l'ordre des places ; cellule de score vide | non |
| 7 | Grande équipe | lignes de 34 px dans une zone qui défile, en-têtes d'équipe collés en haut, fil gardé à 5 lignes : environ 11 joueurs visibles sur 24 | oui (question 8) |
| 8 | Places libres, pas encore apparus | habit tireté des tuiles d'attente (`model/playerCardFx.ts:219`), « Place libre » ou « Pas encore apparu » (`i18n/fr.ts:327-329`) à la place des jauges | non |
| 9 | Lignes hors élimination | en colonne : forme actuelle sur une ligne, horloge sur la colonne de 30 px, pictogramme sur la colonne de l'arme ; sur le plan : question 5 | oui |
| 10 | Contenu du lot T | maquette 0.4 | oui (prévu) |

Ajouts de l'audit, portés dans la grille : place du titre de page et du badge d'administration ; bandeau de
régénération, avis et états de chargement dans les trois cadres ; pastilles de manche et garde de colline dans un
en-tête de 56 px et une rangée de 58 px ; médailles, badge « porteur » et assistance d'une ligne de frag ; ligne
de frag choisie ; fenêtre de moins de 1280 px.

## 5. Lot T

**Formules à réutiliser**
- FDA du jeu : `matchFda` (`apps/web/src/lib/fda.ts:58`), déjà lue par la fiche (`ui/ReplayCountersBadge.tsx:141`).
  Compteurs à l'instant : `playerCountersAt` (`apps/web/src/lib/replay/scoreTimeline.ts`, appelée en `ui/ReplayPlayerPanel.tsx:103`).
- « Arme lourde » : le bloc `weaponTiers` du document ne porte qu'un drapeau (`apps/go-api/internal/domain/replaydoc/map.go:197-200`).
  La catégorie est `weaponLabels[...].role` (`apps/go-api/internal/domain/replaydoc/document.go:272-275`), valeurs lues aujourd'hui
  `sniper`, `power`, `special` (`layers/useReplayWeaponPads.ts:193`, `214`). Un frag du fil porte la clé d'arme du
  titre et le tir à la tête (`apps/web/src/features/match-view/_momentum.ts:67-84`) ; la jointure se fait par `weaponLabels[...].key`.
  Mêlée et grenade : famille de `doc.killEffects[clé]` (`model/killFx.ts:134`). T.2 fixe la table des valeurs sur pièces.
  L'interface dit aujourd'hui « armes de puissance » (`i18n/playerPanel.i18n.ts:72`), D-20 dit « arme lourde » : à
  trancher dans 0.4.

**Médailles : la source d'aujourd'hui**
- Requête : contexte du match du rejeu, `useMatchReplayContext` (`apps/web/src/lib/replay/queries.ts:112-118`), clé
  `matchReplayContext(playerSlug, titleSlug, matchId)` (`apps/web/src/lib/query/keys.ts:174-175`). **La langue n'est pas dans la
  clé**, alors que libellé et description y sont résolus par le serveur.
- Lecture : `collectMedalEvents(context.highlight_events)` (`model/replayModel.ts:167`, `model/killFeedLogic.ts:105-123`),
  rattachement au frag du même joueur à 500 ms près (`model/killFeedLogic.ts:526-537`). La fiche n'affiche aucune
  médaille aujourd'hui.
- Ce que T.6 substitue : `medals[]` du document (`apps/go-api/internal/domain/replaydoc/medals.go:8-14`), instant, joueur, indice, nom
  anglais. Le type `MedalEvent` (`model/killFeedLogic.ts:87-98`) garde sa forme, seul `collectMedalEvents` change de
  source ; les frags, eux, restent lus dans le contexte.
- **Contradiction dans le texte de T.6** : « résolu dans la langue de la requête » et « sur le patron de
  `service/replay_vehicle_labels.go` » ne disent pas la même chose. Ce patron pose dans le document une table de
  libellés par langue (`apps/web/src/lib/api/generated.ts:15289-15295`), lue par `pickText`, et la clé du document n'a alors pas
  besoin de la langue (`apps/web/src/lib/query/keys.ts:170-171`). Recommandation : suivre le patron, table `medalLabels` par nom de
  médaille ; un document de plusieurs mégaoctets n'est pas rechargé au changement de langue.

**Clés du document par item** (contrat web `apps/web/src/lib/api/generated.ts:11808-11902`, normalisé par
`apps/web/src/lib/replay/replayNormalize.ts` ; contrat Go `apps/go-api/internal/domain/replaydoc/`)

| Item | Clés |
|---|---|
| T.1 | `scoreTimeline.players` (frags, morts, assistances), `tracks` et leurs points (vies, bouclier), `medals` (parfaits, médailles), frags du contexte pour le tir à la tête |
| T.2 | frags du contexte, `weaponLabels`, `killEffects` |
| T.3 | `scoreActions` (`apps/go-api/internal/domain/replaydoc/score_actions.go:8-14`) et `coverage.scoreActions.complete` (`:19`) ; `shots`, `hits`, `damage` datés, gardés par `shotStats[].sufficient` (`apps/go-api/internal/domain/replaydoc/hits.go:26-36`). `shotStats` ne porte que des totaux de match : l'« à l'instant » se cumule sur les événements datés. `shots` désigne un emplacement de vie, pas un joueur |
| T.4 | `grenades` (rang du type, `apps/web/src/lib/api/generated.ts:8081-8097`), `pickups` (`:11076-11088`), `grenadeLabels` |
| T.7 | `stances` (`apps/go-api/internal/domain/replaydoc/inventory.go:102`), genres `sprint` et `jumpDerived` (`apps/web/src/lib/replay/replayReadyTypes.ts:330-337`), `coverage.stances`. Aucun lecteur web aujourd'hui : T.7 est le premier |

## 6. Questions au user

1. **Tuile de la rangée.** La tuile de la maquette tient sur deux lignes, sans arme : le glissement au changement
   d'arme et le mot « ÉLIMINÉ » n'y ont plus de place. Recommandation : suivre la maquette, garder la bande
   hachurée avec « Réapparition 5,8 s », et garder le glissement d'arme dans la colonne du Standard et la fiche.
2. **Joueur « suivi ».** Sur les maquettes un joueur est « suivi » dès l'ouverture (ligne dépliée, bordure, gras
   dans le fil), alors qu'aujourd'hui la caméra ne suit personne tant qu'on ne le demande pas. Recommandation :
   « suivi » désigne le joueur regardé, celui de la page au départ, et « Suivre » reste l'action de caméra, à part.
3. **Sous le lecteur du Standard.** La maquette annonce « carte de chaleur, usage de l'équipement », mais la page
   n'a rien sous le lecteur : la chaleur est un réglage de « Vue », l'usage de l'équipement est sur la page du
   match. Recommandation : ne rien ajouter dans cette version.
4. **Ligne du fil.** La ligne de la maquette n'a ni médailles, ni assistance, ni badge « porteur », que la ligne
   d'aujourd'hui affiche. Recommandation : garder les médailles dans la colonne, entre la victime et le lieu, et
   passer l'assistance et le badge en infobulle.
5. **Fil posé sur le plan.** Le plan lui applique le filtre du Standard, qui vaut « Tout » : prises d'armes et
   objectifs chasseraient les éliminations des cinq lignes, sans menu pour le changer. Recommandation :
   éliminations seules sur le plan, comme sur la maquette.
6. **Objectif sous le score.** Le rejeu connaît le nombre à atteindre (pas pour tous les modes) mais pas l'unité.
   Recommandation : écrire « Premier à 50 » quand le nombre est connu, rien sinon, et mettre l'unité au backlog.
7. **Barre d'une ligne.** La maquette n'y met ni la lecture automatique, ni « Recommencer », ni les repères de
   médias, alors que le plan promet les mêmes commandes qu'en Standard. Recommandation : suivre la maquette, la
   touche R reste active.
8. **Grande équipe.** Dans la colonne de 360 px, 24 joueurs en lignes défilent : environ 11 visibles à la fois,
   contre 24 aujourd'hui. Recommandation : lignes de la maquette qui défilent, en-têtes d'équipe collés en haut ;
   une forme resserrée peut être montrée à côté.
9. **Menu de la caméra.** Aucune maquette ne le montre. Recommandation : un petit panneau avec deux listes
   déroulantes, « Suivre » et « Point de vue ».

## 7. Découvertes hors périmètre (notées, non traitées)

- La clé de requête du contexte du rejeu ne porte pas la langue alors que la réponse est traduite
  (`apps/web/src/lib/query/keys.ts:174-175`).
- Le commentaire Go de `Stance` dit le sprint réfuté et le saut non prouvé, alors que le document les publie
  (`apps/go-api/internal/domain/replaydoc/inventory.go:96-107`).
- Le dialogue d'export n'est pas une couche d'Échap : Échap, dialogue ouvert, quitte le plein écran.
- La section 2 du plan cite `ReplayTransport.tsx` à 469 lignes et `ReplayPlayerCard.tsx` à 606 : lignes brutes ;
  comptées, 340 et 382.
