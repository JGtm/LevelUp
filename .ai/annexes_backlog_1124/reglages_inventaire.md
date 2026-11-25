# Réglages du rejeu 2D : inventaire de l'existant (étape M.1)

Date : 2026-11-24. Lecture seule du code, rien n'a été lancé (ni serveur, ni navigateur, ni test).
Base lue : branche `release/v8.3.0`, arbre propre.

## Conventions

- `mr/` = `apps/web/src/features/match-replay/` ; `lib/` = `apps/web/src/lib/`. Les numéros de ligne sont ceux des fichiers à la date ci-dessus.
- **Vérifié** = relu dans le fichier cité. **Déduit** = raisonné sans preuve ; la mention dit ce qui le vérifierait. Sans mention, une ligne est vérifiée.
- « Réglage » = une commande dont l'état change ce que la page montre, joue ou mémorise. Les actions sans état (capturer une image, recommencer, reculer de 10 s) sont rangées à part (section 1.8).
- Persistance : tout est dans `localStorage` du navigateur (via `lib/replay/preferenceStore.ts`). Vérifié : aucune préférence du rejeu n'est écrite côté serveur, ni dans l'URL (seul `?t=&clock=` ouvre le rejeu à un instant, `mr/ReplayPage.tsx:105`). Donc aucune portée « compte » : un autre navigateur ou l'application de bureau repart des défauts. Les clés commencent par `replay-`.

## Ce qui est supposé ou n'a pas pu être établi

- Supposé : « utilisés réellement » (demandé par le backlog) ne peut pas se lire dans le code. Aucune mesure d'usage n'existe. À établir : demander à l'utilisateur ses réglages habituels, ou instrumenter.
- Supposé : l'aspect visuel des cellules (ex. un segmenté à trois options dans une demi-colonne de 190 px) n'a pas été vu ; il est déduit des classes CSS.
- Non établi : le sens exact de la valeur de capacité `can_view_replay_3d` (le code dit lui-même que le nom est supposé, `mr/settings/useReplayDimension.ts:9`). Le commutateur 2D/3D est donc inerte tant que le serveur ne la publie pas.
- Non trouvé sur le poste : l'artefact de la maquette des répliques n'a pas été ouvert (page privée) ; ses sources locales l'ont été (section 4).

---

## 1. Tous les réglages

Chiffres : **66 réglages** au total : **43 dans le panneau « Vue »** (R1 à R43), **19 dans les barres, menus, colonnes et fiches** (R44 à R62) et **4 dans le dialogue d'export** (R63 à R66). **49 sont mémorisés** dans le navigateur, **17 ne le sont pas**. Détail des comptes en 1.9.

Colonnes : Où = surface et composant (`fichier:ligne`) ; Défaut = valeur à la première ouverture, puis clé `localStorage` et portée.

### 1.1 Préréglage de vue (panneau Vue, section « Préréglages »)

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R1 | Préréglages : Complet, Combat, Déplacements, Objectif (+ « Personnalisé », jamais choisi, déduit) | Panneau Vue, 1re section ; `mr/settings/ReplayViewPresets.tsx:27` ; le bouton « Vue » porte son nom (`mr/settings/ReplayViewMenu.tsx:45`) | 4 boutons à choix exclusif sur 2 colonnes, avec une 2e ligne de description ; « Personnalisé » = aucun pressé + phrase | Complet = état d'ouverture. Clé `replay-view-preset` (dernier préréglage choisi, sert seulement à départager deux préréglages égaux). Navigateur, tous les rejeux. La vue active se DÉDUIT de l'état des 16 calques, rien n'est écrit pour elle | Allume/éteint d'un coup les 16 calques (R2-R4, R10, R11, R14 à R16, R18 à R22, R24 à R26), seulement ceux que le match porte. Ne touche JAMAIS : fond, traits du relief, plan des murs, étage, filles « armes spéciales / de départ / non identifiés », probabilité de réapparition, Qui/Quand, orientation, fil du plan, barre fine, effets, chaleur, couleur des points, son. Tables : Complet = tout sauf Vies ; Combat = visée, traînée, véhicules, 5 objectifs, morts, tueurs ; Déplacements = traînée, zones, apparitions ; Objectif = traînée, zones, 5 objectifs (`mr/settings/viewPresetLogic.ts:91-96`). Complet et Combat se confondent sur un match sans zone ni véhicule ni objectif (`viewPresetLogic.ts:180-184`) | `mr/settings/useReplayViewMenu.ts:293-303`, `mr/settings/viewPresetLogic.ts`, `mr/settings/replayPreferences.ts:94` |

### 1.2 Calques du panneau Vue (section « Calques »)

Quatre blocs : « Joueurs », « Terrain », « Équipements posés » (sans titre), « Objectifs ». Les groupes titrés sont en grille à 2 colonnes (`mr/settings/ReplaySettingsLayers.tsx:432-440`). Un calque absent du film n'est pas proposé (règle « pas de commande qui ne commande rien »).

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R2 | Visée | Calques > Joueurs ; `ReplaySettingsLayers.tsx:226` | Interrupteur | Activé. `replay-show-aim`. Tous les rejeux | Cône du regard sur chaque joueur ; aussi le cône du conducteur d'un véhicule (`mr/ui/ReplayCanvas.tsx:427`). Toujours proposé. Compte dans les préréglages | `mr/settings/useReplaySettings.ts:183` |
| R3 | Traînée | idem ; `:227` | Interrupteur | Activé. `replay-show-trail`. Tous les rejeux | Les 7 dernières secondes derrière chaque point. Toujours proposé. Compte dans les préréglages | `useReplaySettings.ts:185` |
| R4 | Zones | Calques > Terrain ; `:240` | Interrupteur | Activé. `replay-show-zones`. Tous les rejeux | Zones nommées officielles de la carte. Proposé seulement si la carte a des zones nommées | `useReplaySettings.ts:184` |
| R5 | Fond : Plan / Rendu / Relief | Calques > Terrain ; `:251` | Segmenté 3 positions (ordre Plan, Rendu, Relief) ; n'existe que s'il y a au moins 2 fonds disponibles (`lib/replay/reliefViewLogic.ts:33-39`) | Plan. `replay-background-style`. GLOBAL : tous les rejeux, toutes les cartes, ET la Tactique et « Occupation du terrain » qui lisent et écrivent la même clé (`mr/settings/useMapViewPreferences.ts`) | Choisit l'image de fond. Repli : un Relief non lisible se lit Plan, un Plan absent se lit Rendu (`lib/replay/mapBackgroundVariant.ts` `resolveBackgroundVariant`). Le fond retenu après repli, pas le fond voulu, est affiché dans le segmenté | `mr/settings/useBackgroundStyle.ts`, `mr/settings/useReplayTerrainFlags.ts:269` |
| R6 | Traits du relief | Calques > Terrain ; `:260` | Interrupteur | Dépend du fond : activé sur Relief, éteint sur Plan et Rendu (`lib/replay/useReliefLines.ts:26-27`). Clé PAR FOND `replay-relief-lines.<fond>`. Partagé avec les autres vues 2D | Ruptures de hauteur, courbes de niveau, liseré de la zone jouable. Proposé seulement si le relief est lisible | `lib/replay/useReliefLines.ts`, `mr/hooks/useReplayRelief.ts:102,133` |
| R7 | Plan des murs | Calques > Terrain ; `:269` | Interrupteur, désactivé et affiché éteint quand le fond posé est « Plan » | Activé. `replay-show-wall-plan`. Partagé avec les autres vues 2D | Dessin vectoriel des murs par-dessus l'image. Sur Plan, les murs sont déjà dans l'image : l'interrupteur est grisé avec l'infobulle « Déjà dans le plan » (`ReplaySettingsLayers.tsx:272-275`). Proposé seulement si le fond porte un plan (Live Fire et deux cartes Forge : non) | `useReplayTerrainFlags.ts:268`, `lib/replay/mapBackgroundVariant.ts` `isWallPlanDrawn` |
| R8 | Étage : Tout / une pastille par étage (Bas, Étage 1…) / curseur « Hauteur » | Calques > Terrain, pleine largeur ; `mr/settings/ReplayFloorControls.tsx:91-125` | Pastilles (aria-pressed) + curseur de plage, pas 0,25 | « Tout » (aucune coupe). NON mémorisé : état local réinitialisé quand la carte change (`mr/hooks/useReplayFloorChoice.ts:35-46`, clé de portée `mapId ?? matchId`) | Le fond, ses traits et le relief suivent la coupe ; AUCUN joueur n'est masqué. Absent si aucun relief ; remplacé par une phrase en pointillé si la carte n'a pas de maillage de navigation. Au-delà de 6 pastilles la rangée défile | `useReplayFloorChoice.ts`, `mr/hooks/useReplayRelief.ts:103,134` |
| R9 | Suivre le joueur ouvert | Calques > Terrain, sous le curseur d'étage ; `ReplayFloorControls.tsx:126` | Interrupteur | Éteint. NON mémorisé. Un choix manuel d'étage l'éteint | La coupe prend l'étage du joueur dont la fiche est ouverte, à chaque instant. Désactivé tant qu'aucune fiche de joueur n'est ouverte, avec une phrase explicative | `useReplayFloorChoice.ts`, `mr/hooks/useReplayReliefFollow.ts` |
| R10 | Emplacements d'arme | Calques > Terrain ; `ReplaySettingsLayers.tsx:281` | Interrupteur | Activé. `replay-show-weapon-pads` | Socles et râteliers, avec état plein/vide et compte à rebours. Proposé si le film en porte (absent en Super Fiesta Forge). Compte dans les préréglages | `useReplayTerrainFlags.ts:231` |
| R11 | Armes au sol | idem ; `:289` | Interrupteur | Activé. `replay-show-ground-weapons` | Armes abandonnées, dessinées là où elles sont tombées. Proposé si le film en porte | `useReplayTerrainFlags.ts:235` |
| R12 | Armes spéciales seulement | idem, « fille » de R11 ; `:299` | Interrupteur | Éteint. `replay-show-ground-weapons-special-only` | Ne garde que les armes de puissance. Affiché seulement avec R11 allumé | `useReplayTerrainFlags.ts:239` |
| R13 | Afficher aussi les armes de départ au sol | idem, « fille » de R11 ; `:307` | Interrupteur | Éteint. `replay-show-ground-weapons-starting` | Réaffiche les armes de départ lâchées (masquées par défaut : tout le monde les porte). Affiché seulement avec R11 allumé | `useReplayTerrainFlags.ts:243` |
| R14 | Véhicules | idem ; `:318` | Interrupteur | Activé. `replay-show-vehicles` | Véhicules, trajectoire, occupant, explosion de destruction. Proposé si le film en porte | `useReplayTerrainFlags.ts:267` |
| R15 | Équipements posés | Calques, bloc sans titre (une seule colonne) ; `:334` | Interrupteur | Activé. `replay-show-placements` | Mur, capteur, traqueur, écran occultant, faille, champ de réparation. Proposé si des poses sont dessinables | `useReplayTerrainFlags.ts:219` |
| R16 | Objets lâchés au sol | « fille » de R15 ; `:341` | Interrupteur | Activé. `replay-show-dropped-placements` | Power-ups et équipements lâchés à la mort (anneau pointillé). Affiché seulement avec R15 allumé ET au moins un objet lâché. Compte dans les préréglages | `useReplayTerrainFlags.ts:227` |
| R17 | Objets non identifiés | « fille » de R15 ; `:349` | Interrupteur | Éteint. `replay-show-unnamed-placements` | Points neutres pour les poses sans nom. Affiché seulement avec R15 allumé ET s'il y en a | `useReplayTerrainFlags.ts:223` |
| R18 | Objectifs du mode | Calques > Objectifs ; `:372` | Interrupteur | Activé. `replay-show-mode-objectives` | Collines, bases, zones de capture, socles, points de livraison : commande 3 calques d'un coup. Groupe entier absent hors des modes à objectif (Assassin) | `useReplayTerrainFlags.ts:247` |
| R19 | Drapeaux | idem ; `:380` | Interrupteur | Activé. `replay-show-flag-carries` | Vie des drapeaux de capture. Proposé seulement si le film en porte | `useReplayTerrainFlags.ts:251` |
| R20 | VIP | idem ; `:388` | Interrupteur | Activé. `replay-show-vip-crown` | Couronne du VIP courant | `useReplayTerrainFlags.ts:255` |
| R21 | Porteur du crâne | idem ; `:396` | Interrupteur | Activé. `replay-show-skull-carrier` | Crâne d'Oddball sur son porteur | `useReplayTerrainFlags.ts:259` |
| R22 | Bombe | idem ; `:404` | Interrupteur | Activé. `replay-show-bomb-carrier` | Bombe d'Assaut portée ou posée | `useReplayTerrainFlags.ts:263` |

### 1.3 Groupe « Vies » (panneau Vue, `mr/settings/ReplayLivesSection.tsx`)

Groupe qui SE REPLIE. Replié, il n'écrit que « N sur M » (`:91`). Il disparaît entièrement si aucun de ses éléments n'est proposé.

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R23 | Vies (en-tête repliable) | `:130` | Bouton chevron (aria-expanded) | REPLIÉ. `replay-lives-group-open` | Montre ou cache R24 à R30. Le résumé « N sur M » compte les 3 traces ET la probabilité de réapparition | `ReplayLivesSection.tsx:116` |
| R24 | Apparitions | `:137` | Interrupteur | Éteint, SAUF si le préréglage retenu l'allume et que les 13 autres calques sont encore à son état (`livesDefault`). `replay-show-spawns` | Goutte creuse orientée à chaque apparition. Proposé si au moins une marque à poser | `mr/settings/useReplayLivesFlags.ts:77-83,117` |
| R25 | Morts | `:137` | Interrupteur | Idem (`replay-show-deaths`) | Croix à chaque mort | `useReplayLivesFlags.ts:118` |
| R26 | Tueurs | `:137` | Interrupteur | Idem (`replay-show-killers`) | Triangle creux au tueur et trait fin vers la victime | `useReplayLivesFlags.ts:119` |
| R27 | Probabilité de réapparition | `:143` | Interrupteur | ALLUMÉ. `replay-show-respawn-forecast`. Hors préréglages | Pendant le compte à rebours d'un joueur mort : zones teintées et point réel. Proposé quand la réponse du serveur est prête ; sinon un bloc pointillé donne le motif du refus (`:131`). Suit Seul et Masquer | `useReplayLivesFlags.ts:120` |
| R28 | Dessin : Zones et points / Zones / Points | `:150` | Segmenté 3 positions | « Zones et points ». `replay-respawn-forecast-draw`. | Affiché seulement avec R27 allumé | `useReplayLivesFlags.ts:124-130` |
| R29 | Qui : Tous / Mon équipe / Joueur suivi | `:107` | Segmenté 2 ou 3 positions | Tous. `replay-lives-who`. « Joueur suivi » est retiré s'il n'y a pas de joueur suivi dans le match ; un choix mémorisé « suivi » retombe sur « Tous » pour ce match sans effacer la préférence | Restreint les traces R24 à R26 (pas R27). Affiché seulement si au moins une trace est allumée (`:159`) | `useReplayLivesFlags.ts:131`, `mr/model/livesMarks.ts:330` |
| R30 | Quand : Toute la partie / Jusqu'ici | `:108` | Segmenté 2 positions | Toute la partie. `replay-lives-when` | Idem R29 : filtre temporel des traces | `useReplayLivesFlags.ts:132` |

### 1.4 Autres sections du panneau Vue

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R31 | Orientation : Auto, 0°, 90°, 180°, 270° | Panneau Vue ; `mr/settings/ReplayOrientationSection.tsx:44` | Segmenté 5 positions | Auto = aucun choix retenu. `replay-orientation.<idCarte>` : PAR CARTE, partagé avec la Tactique et « Occupation du terrain ». Choisir Auto efface la clé | Tourne le plan par quarts de tour horaires. Auto : le plan se tourne seul quand il y gagne de la place (règle `useAutoTurned`, dépend de la zone d'affichage donc du cadre). Section absente sans identifiant de carte | `mr/settings/useReplayOrientation.ts`, `replayPreferences.ts:116-139` |
| R32 | Fil des éliminations | Panneau Vue ; `mr/settings/ReplayPlanFeedSection.tsx:24` | Interrupteur | ALLUMÉ. `replay-show-plan-feed`. Une seule valeur pour tous les cadres | Pose les 5 dernières éliminations en bas à gauche du plan. Section absente hors Cinéma et Plein cadre, ET absente quand les équipes sont en colonne (fenêtre étroite, chacun pour soi) : `planFeedPosed` (`mr/model/rowLayout.ts:173`) | `useReplaySettings.ts:189`, `mr/ui/frames/replayFrameSlots.tsx:92-107` |
| R33 | Barre de lecture fine | Panneau Vue ; `mr/settings/ReplayThinBarSection.tsx:24` | Interrupteur | ALLUMÉ. `replay-thin-bar-full-frame` | Au Plein cadre : barre posée sur le bas du plan, fine et translucide pendant la lecture, pleine à la pause, au survol, au clavier. Éteint : barre pleine sous le plan. Section absente hors Plein cadre | `useReplaySettings.ts:190`, `mr/model/transportBarLogic.ts:64,79` |
| R34 | Effets de tirs | Panneau Vue > Effets ; `ReplayViewPanel.tsx:191` | Interrupteur (2 colonnes, avec Effets de mort) ; le (i) du titre dit « Un tir s'affiche lorsqu'il applique un dégât » | Activé. `replay-show-shot-fx` | Éclair de bouche sur chaque tir décodé | `useReplaySettings.ts:187` |
| R35 | Effets de mort | idem ; `:197` | Interrupteur | Activé. `replay-show-kill-fx` | Trait tueur vers victime à l'instant de l'élimination | `useReplaySettings.ts:188` |
| R36 | Carte de chaleur | Panneau Vue (titre et interrupteur portent le même libellé) ; `mr/ui/ReplayHeatmapSection.tsx:46,49` | Interrupteur | ÉTEINT. `replay-show-heatmap` | Calque de synthèse du match. Une légende « rare / fréquent » se pose dans un coin du bloc quand il est allumé (en haut si le fil du plan ou la barre posée occupent le bas, `ReplayCanvas.tsx:646`) | `useReplaySettings.ts:186` |
| R37 | Mesure : Présence / Éliminations | idem ; `:65` | Segmenté | Présence. `replay-heatmap-mode` | Affiché seulement avec R36 allumé ET s'il y a au moins 2 lectures ; « Éliminations » retiré si aucune mort n'est localisable. Le panneau montre la lecture SERVIE, pas la demandée (`ReplayCanvas.tsx:587`) | `useReplaySettings.ts:200` |
| R38 | Durée : Partie / Progressif | idem ; `:80` | Segmenté | Partie. `replay-heatmap-span` | Match entier ou seulement jusqu'à l'image lue. Affiché seulement avec R36 allumé | `useReplaySettings.ts:203` |
| R39 | Couleur des points : Par équipe / Par joueur | Panneau Vue ; `ReplayViewPanel.tsx:228` | 2 boutons à choix exclusif (`SettingsChoice`) | Par équipe. `replay-marker-colors` | Couleur du camp, ou couleur stable par joueur | `useReplaySettings.ts:206` |
| R40 | Sons par catégorie : Armes | Panneau Vue, dernière section ; `ReplayViewPanel.tsx:259` | Interrupteur (grille 2 colonnes) | Activé. `replay-sound-categories-off` (liste JSON des catégories COUPÉES, vide par défaut) | Coupe une famille de sons. Section absente si la piste du match n'a aucun son. Aucun effet audible tant que le son (R44) est coupé ou que la vitesse dépasse 2× | `mr/sound/useReplaySound.ts:82,187,234` |
| R41 | … Grenades | idem | idem | idem (même clé) | idem | idem |
| R42 | … Mêlée | idem | idem | idem | idem | idem |
| R43 | … Équipements | idem | idem | idem | idem. Le 5e type interne « Objectifs » ne se règle plus (retiré le 2026-09-02, `mr/sound/replaySound.ts:441-465`) : ses sons jouent toujours | idem |

### 1.5 Barre de lecture, menus, commandes de la carte

Deux barres : la barre COMPLÈTE du Standard (`mr/ui/ReplayTransport.tsx`) et la barre d'UNE LIGNE du Cinéma et du Plein cadre (`mr/ui/ReplayTransportCompact.tsx` + `ReplayCompactBarControls.tsx`).

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R44 | Son (icône haut-parleur) | Barre complète `ReplayTransport.tsx:64` ; barre d'une ligne `ReplayCompactBarControls.tsx:57` ; raccourci M ; `mr/sound/ReplaySoundControls.tsx:35` | Bouton à état (aria-pressed) | COUPÉ. `replay-sound-on`. Tous les rejeux. Au rechargement la préférence est « activé » mais le lecteur audio ne naît qu'au premier geste (`useReplaySound.ts` en-tête) | Sons d'armes, grenades, équipements. Absent si la piste n'a aucun son. S'estompe quand la vitesse dépasse 2× (infobulle « à cette vitesse les sons se chevaucheraient ») | `useReplaySound.ts:339` |
| R45 | Volume des sons | idem ; `ReplaySoundControls.tsx:50` | Curseur 0 à 100, pas 5, 58 px | 70 %. `replay-sound-volume`. Tombe à zéro et se désactive quand le son est coupé, le niveau réglé revient avec le son | Niveau global | `useReplaySound.ts:85,340` |
| R46 | Vitesse | Barre complète `:65` ; barre d'une ligne `:95` ; `mr/ui/ReplaySpeedMenu.tsx:42` | Bouton + liste : 0,5× 1× 1,5× 2× 4× | 1×. `replay-speed`. Tous les rejeux | Cadence de lecture. 4× est annoté « son coupé » dans la liste (`SOUND_MAX_SPEED = 2`, `mr/sound/replaySoundCursor.ts:35`) | `useReplaySettings.ts:210`, `SPEED_MULTIPLIERS` `:61` |
| R47 | Lecture automatique | Barre complète `:63` ; barre d'une ligne `:83` ; `mr/ui/ReplayAutoPlayToggle.tsx:26` | Bouton `role="switch"`, icône seule (mini-interrupteur 28×16) | ÉTEINT : le rejeu s'ouvre en pause au coup d'envoi. `replay-autoplay` | Ne décide que de l'état de départ, lu une fois au montage (`mr/hooks/useReplayPlayback.ts:332`). Basculer ne met pas en lecture le rejeu ouvert | `useReplaySettings.ts:209` |
| R48 | Cadre : Standard / Cinéma / Plein cadre | Barre complète `:84` ; barre d'une ligne `:108` ; `mr/ui/ReplayFrameButtons.tsx:60-81` ; raccourcis T (Cinéma) et F (Plein cadre) ; bouton « Quitter (Échap) » de l'en-tête au Plein cadre (`mr/ui/frames/ReplayFrameHeader.tsx:203`) | 3 boutons icône à état | Standard à chaque ouverture. NON mémorisé, ni URL, ni stockage (`mr/hooks/useReplayFrame.ts` en-tête). Inerte pendant un export | Change la mise en page : en-tête, rangée de fiches ou colonne, barre, taille du plan. Change AUSSI le contenu du panneau Vue (R32, R33), la forme du zoom (R50) et la présence du menu caméra (R53, R54) | `mr/hooks/useReplayFrame.ts`, `mr/model/replayFrame.ts` |
| R49 | Plein écran | idem, séparé des cadres par un filet ; `ReplayFrameButtons.tsx:84-94` | Bouton icône | Éteint. NON mémorisé | Plein écran du système, s'ajoute à n'importe quel cadre ; depuis le Standard il passe d'abord par le Plein cadre. Inerte pendant un export | `mr/hooks/useReplayFullscreen.ts` |
| R50 | Zoom : paliers 1, 1,5, 2, 3 | Standard : croix flottante coin bas droit du bloc carte, avec niveau « 2x » (`mr/ui/ReplayZoomControl.tsx:44-58`) ; Cinéma et Plein cadre : seulement « − » et « + » dans la barre (`ReplayCompactBarControls.tsx:98-105`) ; raccourcis + = − _ ; molette | Boutons ; molette ; clavier | 1×. NON mémorisé | Grossit la carte, mêmes paliers pour les trois gestes (`lib/replay/replayLogic.ts:199`) | `mr/hooks/useReplayZoom.ts` |
| R51 | Cadrage (déplacement) | Standard : 4 flèches ▲◀▶▼ et ⌂ de la croix (`ReplayZoomControl.tsx:67-81`) ; Cinéma et Plein cadre : AUCUN bouton ; raccourcis Maj + flèche, 0 (tout voir) ; glisser sur la carte | Boutons ; clavier ; glisser | Centré. NON mémorisé | Déplace la fenêtre quand le zoom dépasse 1×. Sans effet à 1× | `useReplayZoom.ts`, `mr/hooks/useReplayDrag.ts` |
| R52 | Plan tourné | Barre d'une ligne seulement, entre la vitesse et le zoom ; `mr/ui/ReplayRotatedButton.tsx:43` | Bouton texte, visible seulement quand l'Auto a tourné le plan | n.a. | Clic = retient 0° pour la carte (écrit la clé d'orientation de R31) et redresse le plan. Absent de la barre complète ; au Standard le retour passe par R31 (`ReplayTransport.tsx:40`) | `mr/ui/replayTransportCommands.ts:123` |
| R53 | Caméra : Suivre (Aucun / un joueur par camp) | Barre d'une ligne seulement, bouton icône à gauche du son ; `mr/ui/ReplayCameraMenu.tsx:131-141,190` ; en Standard : icône cible des tuiles (R58) | Menu à 2 listes, coche à droite | Aucun. NON mémorisé | Cadre la carte sur un joueur ; un geste manuel (zoom, glisser) éteint le suivi. Identique au bouton « Suivre » des tuiles. Le bouton prend le fond d'accent tant qu'on suit quelqu'un | `mr/hooks/usePlayerPanel.ts`, `mr/model/cameraMenu.ts` |
| R54 | Point de vue (par les yeux de qui) | Barre d'une ligne : 2e liste du menu caméra, titre « Point de vue » (`ReplayCameraMenu.tsx:142-151`) ; barre complète : liste déroulante native en tête des pistes de la frise, libellé accessible « Joueur suivi » (`mr/ui/ReplayTimelineTracks.tsx:212`, visible seulement pistes dépliées) | Menu ; `<select>` natif | Le joueur du match. NON mémorisé (rangé avec la clé du match, `mr/hooks/useReplayViewpoint.ts`) | Change le camp de référence : couleurs allié/adverse, pistes de la frise, calques d'objectif, son. Ne touche ni le curseur ni la lecture. Un joueur sans ligne de tableau de score est inerte | `useReplayViewpoint.ts` |
| R55 | Pistes de la frise (replier / déplier) | Barre complète seulement ; chevron sous les pistes ; `ReplayTimelineTracks.tsx:308` | Bouton chevron (aria-expanded) | DÉPLIÉES. `replay-timeline-expanded` | Replié : il ne reste que le curseur et les horloges. Cache aussi le sélecteur R54 de la barre complète | `mr/hooks/useReplayTimeline.ts:152` |
| R56 | 2D / 3D | À gauche du bouton « Vue », coin haut droit de la carte ; `mr/ui/ReplayDimensionSwitch.tsx:27` | Segmenté 2 positions | 2D. NON mémorisé | Rendu seulement si la capacité `can_view_replay_3d` du serveur est vraie (aujourd'hui inerte, voir « non établi ») | `mr/settings/useReplayDimension.ts:40-44` |

### 1.6 Colonne, fiches, joueurs

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R57 | Filtre du fil : Tout, Éliminations, Objectifs, puis « Proposées en plus » : Armes et bonus, Véhicules, Médailles, Assistances | Titre du fil de la colonne, qui EST le bouton du menu ; `mr/ui/ReplayFeedFilterMenu.tsx:38-103` (monté par `mr/ui/ReplayKillFeed.tsx:131`) | Menu de 7 entrées, chacune avec son nombre de lignes | Tout. `replay-feed-filter`. Valable pour tous les matchs ; « Objectifs » retiré des matchs sans ligne d'objectif, le filtre mémorisé inexistant retombe sur Tout sans réécriture (`mr/model/feedFilterLogic.ts:88-93`) | Filtre la liste de la colonne seulement (la frise et les effets lisent le fil entier). N'agit PAS sur le fil posé sur le plan (R32), qui montre toujours les 5 dernières éliminations (`mr/ui/ReplayFeedOverlay.tsx:6-9`). Inaccessible quand la colonne est cachée (voir constats) | `mr/settings/useFeedFilter.ts`, `mr/hooks/useFeedView.ts` |
| R58 | Suivre / Ne plus suivre (par joueur) | Icône cible de chaque tuile ou ligne, visible au survol ou au focus, toujours visible si actif (`mr/ui/ReplayTileActionButtons.tsx:100`) ; tuiles de la colonne, de la rangée et fiches compactes | Bouton icône à état | Personne. NON mémorisé, remis à zéro à chaque match | Caméra : voir R53 (même état). Suivre un joueur masqué lève son masquage | `mr/hooks/usePlayerPanel.ts`, `mr/model/playerPanelLogic.ts` |
| R59 | Masquer / Afficher (par joueur) | Icône œil de chaque tuile ou ligne ; `ReplayTileActionButtons.tsx:108` | Bouton icône à état | Personne masqué. NON mémorisé | Retire le joueur de la carte (pion, traînée, nom). La tuile écrit « masqué » ou barre l'œil | idem |
| R60 | Tout afficher | En tête de la colonne des équipes et de la rangée de fiches, seulement si au moins un joueur est masqué ; `mr/ui/ReplayShowAllButton.tsx:16`, `ReplayRosterColumn.tsx:92`, `ReplayTeamsRow.tsx:199` | Bouton | n.a. | Réaffiche tous les joueurs masqués | idem |
| R61 | Seul / « Seul · tout réafficher » | Dans la fiche d'un joueur ; `mr/ui/fiche/FicheActions.tsx:28` | Bouton à état | Inactif. NON mémorisé | Masque tous les autres joueurs ; l'œil d'une tuile en réaffiche un. Suit aussi la probabilité de réapparition (R27) | `usePlayerPanel.ts` |
| R62 | Fiche ouverte (clic sur une tuile ou une ligne) | Colonne, rangée ; la fiche REMPLACE la colonne au Standard (`mr/ui/frames/ReplayStandardFrame.tsx:63-119`) ou s'ouvre en fenêtre ancrée au Cinéma et au Plein cadre (`mr/ui/ReplayPlayerPopover.tsx`) | Clic ; fermeture par la croix, Échap, « Équipes » | Aucune. NON mémorisé | Sert de joueur « ouvert » à R9 et à l'anneau du pion | `usePlayerPanel.ts` |

### 1.7 Dialogue d'export (bouton export de la barre, `mr/export/ReplayExportDialog.tsx`)

| # | Libellé | Où | Contrôle · valeurs | Défaut · mémorisation · portée | Effet · dépendances | Porté par |
|---|---|---|---|---|---|---|
| R63 | Format | Dialogue d'export ; `ReplayExportDialog.tsx:75` | Choix : 1080p (1920×1080), 720p (1280×720) | 1080p. `replay-export-format` | Taille du clip | `mr/settings/replayPreferences.ts:35-45` |
| R64 | Début / Fin | idem ; `:73` | Deux bornes de la plage | Tout le rejeu. NON mémorisé | Plage exportée | état local |
| R65 | Inclure le son | idem ; `:74` | Case | COCHÉE (alors que le son de la page est coupé par défaut). NON mémorisé | Piste audio du clip | état local |
| R66 | Cadrage : Carte entière / Cadrage actuel | idem ; `:76` | Choix exclusif | Carte entière. NON mémorisé (contextuel, décision D6 dans `mr/export/exportFormats.ts`) | Proposé seulement si le zoom est au-dessus de 1× | état local |

### 1.8 Commandes sans état (hors décompte)

Lecture/Pause (Espace, K), Recommencer (R), −10 s et +10 s (←/→, J/L, `SKIP_SECONDS = 10`, `mr/layers/replayCanvasConfig.ts:65`), image par image (`,` et `.`), curseur de la frise (glisser, clic), Capturer l'image, Enregistrer la vidéo (seulement si l'export n'est pas disponible), Exporter, Fermer la fiche, « Équipes » (retour de la fiche à la colonne), lien « Fiche » vers la fiche du match, retour (flèche), badge de schéma (administrateur).

### 1.9 Comptes

| Famille | Nombre | Détail |
|---|---|---|
| Panneau « Vue » | 43 | R1 à R43. Seuls R1, R2, R3, R34, R35, R36 et R39 sont proposés sans condition (R5 demande au moins 2 fonds) ; tous les autres dépendent de ce que le film, la carte, le cadre ou l'état d'un autre réglage portent |
| Barres, menus, colonnes, carte | 19 | R44 à R62 |
| Dialogue d'export | 4 | R63 à R66 |
| Total | 66 | |
| Mémorisés (`localStorage`) | 49 | tous sauf R8, R9, R48 à R51, R53, R54, R56, R58 à R62, R64 à R66 (R52 écrit la clé de R31) |
| Non mémorisés | 17 | R8, R9, R48, R49, R50, R51, R53, R54, R56, R58, R59, R60, R61, R62, R64, R65, R66 |
| Clés `replay-*` distinctes lues dans le code | 45 | dont 25 clés `replay-show-*`, et 2 clés par motif (`replay-orientation.<carte>`, `replay-relief-lines.<fond>`) |

(Les chiffres de la section 1.9 sont recomptés sur les tableaux ci-dessus ; les 45 clés viennent d'un balayage `grep` de `'replay-…'` dans `apps/web/src` hors tests, dont quelques noms qui ne sont pas des réglages : classes CSS `replay-flash-*`, `replay-wswap-l`, clé de requête `replay-regen`.)

---

## 2. Carte des emplacements

Légende des cadres : S = Standard, C = Cinéma, P = Plein cadre.

| # | Surface | Composant | Cadres | Réglages, dans l'ordre affiché |
|---|---|---|---|---|
| E1 | Bouton « Vue » + nom du préréglage actif, coin haut droit de la carte | `mr/settings/ReplayViewMenu.tsx:26-47` | S, C, P | Ouvre E2. Affiche le préréglage ou « Personnalisé » (aria-label combiné) |
| E2 | Panneau « Vue » (portail sur `body`, 416 px, défile en interne, se ferme par Échap, croix ou clic dehors) | `mr/settings/ReplayViewPanel.tsx:341-406` | S, C, P (contenu variable) | Préréglages (R1) > Calques : Joueurs (R2, R3) ; Terrain (R4, R5, R6, R7, R8, R9, R10, R11, R12, R13, R14) ; Équipements posés (R15, R16, R17) ; Objectifs (R18 à R22) > Vies (R23 à R30) > Orientation (R31) > Fil des éliminations (R32, C et P seulement) > Barre fine (R33, P seulement) > Effets (R34, R35) > Carte de chaleur (R36 à R38) > Couleur des points (R39) > Sons par catégorie (R40 à R43) |
| E3 | Commutateur 2D/3D, à gauche du bouton Vue | `mr/ui/ReplayDimensionSwitch.tsx` | S, C, P, mais rendu seulement avec la capacité serveur | R56 |
| E4 | Barre de lecture complète, sous le plan, dans le bloc (fond `bg-background/40`) | `mr/ui/ReplayTransport.tsx` | S | Frise à pistes (R54 en tête des pistes, R55 chevron) ; rangée de gauche : R47, R44 et R45, R46 ; centre : recommencer, −10, lecture, +10 ; droite : capture, export, R48, R49 |
| E5 | Barre de lecture d'une ligne (56 px ; 2 lignes quand la frise passe sous 240 px) | `mr/ui/ReplayTransportCompact.tsx`, `ReplayCompactBarControls.tsx:56-108` | C, P | Dans l'ordre, groupes séparés par un filet de 1×20 px : menu caméra (R53, R54), son et volume (R44, R45) ; lecture, recommencer, −10, +10, horloge, lecture automatique (R47) ; frise sans pistes ni R55 ; vitesse (R46) ; Plan tourné (R52, si l'Auto a tourné le plan), zoom − et + (R50) ; capture, export, cadres (R48), plein écran (R49) |
| E6 | Commande de zoom flottante (croix), coin bas droit du bloc carte | `mr/ui/ReplayZoomControl.tsx` | S | R50 (− niveau +), R51 (4 flèches, ⌂) |
| E7 | Menu de vitesse (liste qui s'ouvre vers le haut) | `mr/ui/ReplaySpeedMenu.tsx:58-87` | S, C, P | R46 |
| E8 | Menu caméra (2 listes côte à côte, 176 px chacune, ouvert vers le haut) | `mr/ui/ReplayCameraMenu.tsx` | C, P | R53, R54 |
| E9 | Légende de la carte de chaleur, coin du bloc | `mr/ui/ReplayHeatmapLegend.tsx` | S, C, P, si R36 allumé | aucun réglage |
| E10 | Fil posé sur le plan (5 lignes, bas gauche, 350 px de large, ne capte pas le pointeur) | `mr/ui/ReplayFeedOverlay.tsx` | C, P, rangée posée et R32 allumé | aucun réglage : sans menu ni filtre |
| E11 | Colonne des équipes et du fil (360 px au Standard ; 30 rem max au Cinéma et au Plein cadre) | `mr/ui/frames/ReplayStandardFrame.tsx`, `ReplayCinemaFrame.tsx` | S toujours ; C et P seulement quand les équipes sont en colonne (sinon cachée, mais montée) | R60 ; tuiles ou lignes avec R58 et R59 ; menu de filtre du fil R57 (titre du fil) |
| E12 | Rangée de fiches (deux camps autour du score) | `mr/ui/ReplayTeamsRow.tsx` | C, P quand elle tient (2 camps, largeur suffisante) | R60 ; tuiles avec R58 et R59 |
| E13 | Fiche d'un joueur : colonne entière au Standard, fenêtre ancrée (330 px) au Cinéma et au Plein cadre | `mr/ui/ReplayPlayerPanel.tsx`, `ReplayPlayerPopover.tsx` | S, C, P | R61 (Seul) en tête ; R62 se ferme ici. Aucun réglage d'affichage propre |
| E14 | En-tête de cadre | `mr/ui/frames/ReplayFrameHeader.tsx` | S, C, P | Retour et lien « Fiche » (S, C) ; « Quitter (Échap) » (P) ; pas de sélecteur de cadre (il est dans la barre) |
| E15 | Dialogue d'export | `mr/export/ReplayExportDialog.tsx` | S, C, P | R63, R64, R65, R66 |
| E16 | Raccourcis clavier (hors champ de saisie, sans Ctrl, Cmd ni Alt) | `mr/hooks/useReplayShortcuts.ts:127-136` | S, C, P | Espace ou K : lecture ; ← → ou J L : ±10 s ; , et . : image par image ; M : R44 ; R : recommencer ; F : R48 (Plein cadre) ; T : R48 (Cinéma) ; + = − _ : R50 ; 0 : retour à toute la carte ; Maj + flèche : R51 ; Échap : ferme la couche la plus intérieure (panneau, menu, fiche) puis le Plein cadre |
| E17 | Gestes sur la carte | `mr/hooks/useReplayWheelZoom.ts`, `useReplayDrag.ts` | S, C, P | Molette : R50 ; glisser : R51 (éteint Suivre, R53) |

Notes de lecture :
- Aucun raccourci clavier ne bascule un calque, un effet, un préréglage ou la carte de chaleur. Seuls M (son), F et T (cadre), + − 0 et Maj + flèche (zoom, cadrage) touchent un réglage.

---

## 3. Constats (factuels, sans design)

### 3.1 Concentration et hétérogénéité

1. **Un panneau de 43 réglages**, 416 px de large, qui défile en interne, avec 10 sections de grammaires différentes (R1 à R43). Vérifié : `ReplayViewPanel.tsx:364-404`. Les calques et le terrain seuls font 21 réglages (R2 à R22), dont 4 filles (R12, R13, R16, R17), répartis dans 4 blocs de tailles inégales.
2. **Quatre formes de contrôle pour un choix exclusif**, sans règle visible : boutons à choix exclusif avec description (préréglages, `SettingsChoice`) ; boutons à choix exclusif sans description (couleur des points, même composant) ; segmenté (fond, orientation, qui/quand, dessin, mesure, durée) ; pastilles (étage). Hors panneau : liste déroulante (vitesse, filtre du fil, caméra), `<select>` natif (point de vue de la frise), boutons d'icône (cadre).
3. **Le bloc « Équipements posés » n'a pas de titre** (`ReplaySettingsLayers.tsx:332`) alors que ses voisins en ont ; il reste en une colonne alors que le reste est en grille de 2 colonnes (raison documentée : ses filles se lisent dessous).
4. **« Carte de chaleur » est à la fois le titre de section et le libellé de son seul interrupteur** (`ReplayHeatmapSection.tsx:46,49`).
5. **Le groupe « Joueurs » ne contient que Visée et Traînée**, alors que d'autres réglages qui parlent des joueurs sont ailleurs : Couleur des points (R39), Effets de tirs et de mort (R34, R35), les traces Vies (R24 à R26), Seul et Masquer (R59, R61). Déduit par la lecture des libellés ; aucune règle écrite ne dit pourquoi.

### 3.2 Doubles et proches

6. **Quatre réglages dessinent une mort ou un tueur, dans trois sections** : Effets de mort (trait tueur vers victime, R35), Morts (croix, R25), Tueurs (triangle creux et trait fin vers la victime, R26), carte de chaleur en mode Éliminations (R37). Les infobulles de R35 et R26 décrivent chacune un trait du tueur vers la victime (`i18n/fr.ts:213`, `i18n/lives.i18n.ts` clé `layerKillersHint`).
7. **Le son est réglé à deux endroits** : interrupteur et volume dans la barre (R44, R45), catégories dans le panneau Vue (R40 à R43), avec une explication de l'écart dans le code (`ReplayViewPanel.tsx:146-147`). Plus la case « Inclure le son » de l'export (R65), cochée par défaut alors que le son de la page est coupé par défaut.
8. **Le mot « suivre / suivi » désigne trois choses différentes** : la caméra (« Suivre », R53 et R58) ; le point de vue (libellé accessible « Joueur suivi » de la frise, `i18n/fr.ts` clé `viewpointLabel`, et l'option « Joueur suivi » de Qui, R29, dont l'infobulle dit « le joueur dont on regarde le rejeu ») ; la coupe d'étage (« Suivre le joueur ouvert », R9). Le menu caméra, lui, appelle le point de vue « Point de vue » (`roster.cameraViewpointHeading`).
9. **Le mot « Vue » désigne trois choses** : le bouton et le panneau d'affichage, le groupe 2D/3D (« Vue du rejeu »), et « Point de vue ».
10. **Le mot « zones » désigne trois choses** : le calque Zones (zones nommées de la carte, R4), les zones des Objectifs du mode (R18), et les « Zones » du dessin de la probabilité de réapparition (R28).
11. **« Au sol » apparaît deux fois** : Armes au sol (R11) et Objets lâchés au sol (R16), deux calques distincts dans deux blocs.
12. **Orientation en deux endroits** : section du panneau (R31) et bouton « Plan tourné » de la barre d'une ligne (R52), qui écrit la même clé. Le bouton n'existe pas au Standard.
13. **Zoom sous deux formes selon le cadre** : croix avec niveau, flèches et ⌂ au Standard (R50, R51) ; seulement − et + au Cinéma et au Plein cadre, sans niveau ni déplacement ni ⌂. Le déplacement y passe par le glisser ou Maj + flèche.
14. **Point de vue et suivi en deux endroits selon le cadre** : menu caméra (C, P) ou frise et tuiles (S).
15. **Fil des éliminations en deux notions** : R32 (poser 5 éliminations sur le plan) et R57 (filtrer le fil de la colonne). Leur état est indépendant et ils ne se parlent pas.

### 3.3 Contradictions apparentes et effets absents

16. **Le préréglage « Complet » dit « Tous les calques, sauf le groupe Vies »** (`i18n/fr.ts:159`) alors que le groupe Vies contient aussi la probabilité de réapparition (R27), ALLUMÉE par défaut et jamais touchée par un préréglage : le texte est vrai pour les 3 traces, faux pour la 4e commande du groupe. Le résumé replié « N sur M » la compte.
17. **R7 « Plan des murs » est affiché éteint et désactivé sur le fond Plan** alors que sa valeur stockée est « allumé » (`ReplaySettingsLayers.tsx:272`) : l'interrupteur dit l'inverse de la clé.
18. **R6 « Traits du relief » change de défaut selon le fond** (allumé sur Relief, éteint ailleurs) : un seul interrupteur à l'écran, trois clés derrière.
19. **R57 est inaccessible dans la disposition « rangée posée »** (C et P) : le menu du filtre vit dans le titre du fil de la colonne, et la colonne est cachée (`colonneCachee`, `ReplayCinemaFrame.tsx:170`) ; le fil du plan n'a ni menu ni filtre. Le filtre mémorisé s'applique alors à une colonne qu'on ne voit pas. Vérifié dans les composants ; l'effet à l'écran est déduit.
20. **R9 et l'étage (R8)** : la coupe ne masque aucun joueur ; la phrase le dit dans le (i). Mais R59 et R61 masquent des joueurs. Deux sens de « masquer » : un calque ou un joueur.
21. **Sans effet selon le contexte** (le panneau les retire ou les désactive, donc pas de piège visible) : R32 (hors C et P, ou équipes en colonne), R33 (hors P), R12 et R13 (calque mère éteint), R16 et R17 (idem), R28 (R27 éteint), R29 et R30 (aucune trace allumée), R37 et R38 (R36 éteint), R40 à R43 (pas de son). **Mais deux restent actifs sans effet visible** : R40 à R43 tant que R44 est coupé (le son est coupé par défaut) ; R47 « Lecture automatique » ne fait rien sur le rejeu ouvert (le dit son infobulle).
22. **Défaut allumé caché** : R27 (probabilité) est allumée par défaut mais vit dans le groupe Vies REPLIÉ par défaut (R23) ; seul « 1 sur 4 » en dit l'existence.
23. **Les préréglages ne couvrent que 16 des 43 réglages du panneau** (les 16 calques de `LAYER_KEYS`). Ils laissent aux réglages manuels : fond, traits, plan des murs, étage, 3 filles, forecast, Qui/Quand, orientation, fil, barre fine, effets, chaleur (3), couleurs, sons (4). Un seul calque basculé à la main fait afficher « Personnalisé » sur le bouton Vue (déduit : état exact à vérifier à l'écran).
24. **Deux préréglages peuvent être identiques** sur un match sans zone, véhicule ni objectif (Complet et Combat, `viewPresetLogic.ts:180-184`) : le dernier choisi gagne, sinon le premier de la liste.

### 3.4 Dépendances par cadre, carte, mode

25. **Le contenu du panneau change avec le cadre** : R32 absent au Standard, R33 absent hors Plein cadre ; le même panneau a donc 41, 42 ou 43 réglages selon le cadre (hors disponibilités du film).
26. **Le contenu change avec la carte et le mode** : fond (R5, absent sans 2 fonds), R6 (relief lisible), R7 (plan), R8 et R9 (maillage de navigation), R4 (zones), R10 à R14, R15 à R17, groupe Objectifs entier, R24 à R27 (marques disponibles et réponse du serveur), R31 (carte connue), R37 (morts localisables), R29 (option « Joueur suivi »).
27. **Le préréglage influence le défaut d'un autre réglage** : R24 à R26 prennent leur défaut du préréglage retenu quand ils n'ont jamais été écrits (`useReplayLivesFlags.ts:77-83`), règle de compatibilité née de l'ajout du groupe Vies.

### 3.5 Mémorisation

28. **Défauts mémorisés par réglage, jamais par ensemble** : 49 clés indépendantes, aucun « profil » ni export/import de réglages, aucune remise à zéro globale (déduit : aucune commande de ce genre dans le panneau, relue). La maquette des répliques propose elle un « Réinitialiser les réglages ».
29. **Mémorisés alors que leurs voisins ne le sont pas** : R48 (cadre) n'est pas mémorisé, mais R32 et R33, qui n'ont de sens que dans certains cadres, le sont ; R8 (étage) n'est pas mémorisé alors que R5 et R6 (fond, traits) le sont ; R63 (format d'export) l'est, R64 à R66 non ; R55 (pistes dépliées) l'est, R54 (point de vue) non ; R31 est retenu PAR CARTE, R5 pour toutes les cartes.
30. **Portée large** : R5, R7, R31 sont lus par la Tactique et « Occupation du terrain » : un changement dans le rejeu s'y voit (vérifié, `useMapViewPreferences.ts`). Aucun autre réglage n'est partagé.
31. **Aucune portée « compte »** : les réglages ne suivent pas l'utilisateur d'un navigateur ou d'un poste à l'autre, ni vers l'application de bureau.

### 3.6 Nombre de clics (depuis un rejeu ouvert, panneau fermé ; déduit du code, à confirmer à l'écran)

| Action | Clics | Remarque |
|---|---|---|
| Lecture automatique, son (activer), cadre, plein écran | 1 | boutons de barre ; M, T, F au clavier |
| Vitesse | 2 | ouvrir le menu, choisir |
| Un calque (R2 à R22), un effet, la couleur, un préréglage, le fond, les traits, l'orientation, une catégorie de son | 2 | ouvrir Vue, agir ; le panneau reste ouvert pour enchaîner |
| Étage (pastille ou curseur) | 2 | + 1 pour ouvrir une fiche si on veut « Suivre le joueur ouvert » |
| Une trace de vie, la probabilité, son dessin | 3 | ouvrir Vue, DÉPLIER le groupe, agir ; le dessin (R28) demande en plus R27 allumé |
| Qui / Quand des traces | 3 à 4 | si aucune trace n'est allumée, d'abord en allumer une |
| Carte de chaleur : lecture ou durée | 3 | ouvrir Vue, allumer, choisir |
| Fil sur le plan (R32) | 2 | seulement C ou P avec rangée posée ; depuis le Standard : +1 pour changer de cadre |
| Barre fine (R33) | 3 depuis le Standard | cadre Plein cadre (1) + Vue + interrupteur |
| Filtre du fil | 2 | ouvrir le titre, choisir ; hors rangée posée seulement |
| Suivre un joueur | 1 (icône de tuile, au survol) ou 2 (menu caméra en C et P) | |
| Point de vue | 2 | menu caméra (C, P) ou liste de la frise (S, pistes dépliées) |
| Seul | 2 | ouvrir la fiche, Seul |
| Plan tourné | 1 | bouton de barre (C, P) quand l'Auto a tourné le plan |
| Format d'export | 2 | ouvrir le dialogue, choisir |

---

## 4. Ce que les répliques ajouteraient

Source : entrée « Répliques des Spartans : affichage » du backlog (`.ai/BACKLOG.md:269-281`), plan du 2026-11-24 (`.ai/PLAN_BACKLOG_2026-11-24.md`, D-6 à D-8, piste M et piste R) et sources locales de la maquette (version 2) : page du corps, script et rapport de mesure, lus sur le poste (hors dépôt, dossier de sorties du chantier du décodeur, sous-dossier des répliques). Rien n'est inventé : ce qui suit est ce qui y est écrit.

### 4.1 Décisions déjà prises (backlog, 2026-11-24)

- Les trois affichages plaisent et doivent être **combinables** (bulles sur les joueurs, voix, répliques dans le fil), avec des **filtres**.
- On n'affiche que les **répliques parlées** : les cris sans paroles (36 % des répliques jouées) ne s'affichent jamais.
- Les réglages du rejeu sont à rationaliser d'abord (entrée « Rationaliser les réglages d'affichage », priorité à traiter avant l'affichage des répliques) ; les réglages des répliques entrent dans cette organisation, pas dans un panneau de plus (plan D-7, D-8).

### 4.2 Les réglages que la maquette (v2) porte

| Réglage envisagé | Contrôle dans la maquette | Valeur par défaut de la maquette | Source |
|---|---|---|---|
| Bulles sur les joueurs : icône de son sur le joueur + bulle avec le texte, qui s'efface en 2 à 3 secondes (durée de vie de 2,6 s dans le script, entrée en fondu 0,15 s, sortie en fondu 0,5 s) | Case | allumée | corps de la maquette ; script (`LIFE = 2.6`) |
| Voix : icône sur le joueur ; la voix est jouée si le son est activé, sinon seule l'icône | Case | allumée | corps de la maquette |
| Répliques dans le fil : les répliques rejoignent le fil du match ; éteint, le fil ne montre que les éliminations | Case + deux puces dans le fil (« Éliminations », « Répliques ») | allumée | corps de la maquette |
| Quelles répliques : cinq familles (Ennemi repéré, Boucliers, Équipier tombé, Félicitations, Autres) avec leur compte, boutons « Tout » et « Aucun », compteur « N sur M répliques parlées affichées » | Cinq cases | toutes cochées | script (`FAMS`) |
| Fréquence : Toutes / Sans les plus fréquentes / Rares seulement (≤ 2 occurrences) | Segmenté 3 positions | Toutes | script (`freqPass`) |
| Langue des répliques : Français / English | Segmenté 2 positions | Français | corps de la maquette |
| Son : « Activer le son » + volume (désactivé tant que le son est coupé) | Bouton à état + curseur 0-100 | son coupé ; volume 70 | corps ; script (`vol: 0.7`) |
| Réinitialiser les réglages | Bouton | n.a. | corps |
| Filtre par équipe ou par joueur | **Non présent dans la maquette** et non écrit au backlog. Le backlog parle seulement de « leurs propres interrupteurs et filtres » (`BACKLOG.md:265`) | n.a. | non établi |
| Durée d'affichage réglable | **Non présent** : la durée est une constante du script | n.a. | non établi |

### 4.3 Ce que la maquette dit rester à trancher (quatre décisions)

(a) Réglages par défaut à l'ouverture (proposition de la maquette : tout allumé, toutes familles, toutes fréquences). (b) Emplacement : un panneau « Répliques » dans le menu d'affichage, ou des puces dans le fil, ou les deux. (c) Son activé par défaut ou non. (d) Voix versées : une variante par son (7 Mo en français) ou toutes (19 Mo par langue).

### 4.4 Ce que cela représente par rapport à l'existant

Déduit, en additionnant : 3 affichages + 5 familles + 1 fréquence + 1 langue = jusqu'à 10 commandes nouvelles (+ volume et son déjà présents en R44 et R45, + puces de fil). À vérifier : le son des répliques réutiliserait-il R44 et R45 (le backlog ne le dit pas) et la langue des répliques suivrait-elle la langue de l'interface (non écrit).

---

## 5. Pour la maquette réaliste

### 5.1 Structure des trois cadres

Même arbre dans les trois cadres : en-tête, zone d'avis, rangée de fiches (C et P seulement), grille « plan + colonne » (`mr/ui/frames/ReplayFrameLayout.tsx:103-130`). La scène (canvas + barre + panneau) est montée une fois ; basculer de cadre ne change que les classes, l'en-tête, la rangée, la colonne.

| Élément | Standard | Cinéma | Plein cadre |
|---|---|---|---|
| Racine | `flex flex-col`, dans la page de l'application | `flex flex-col`, dans la page | `fixed inset-0 z-40 overflow-y-auto bg-background p-3`, fond de la fenêtre, sans navigation de l'application |
| Largeur du corps | page bornée centrée : contenu max 1200 px + marges latérales de 24 px de chaque côté (boîte 1248 px, `mr/model/cotesStandard.ts`) | pleine largeur du corps de l'application, marge latérale 24 px, marge basse 8 px (`mr/model/cotesCorps.ts`) | marge latérale 4 px + 12 px de cadre, marge basse 6 px |
| En-tête | 64 px de haut, 3 cellules : gauche (flèche retour 32 px + titre sur 2 lignes : ligne 1 `text-xs text-muted-foreground`, ligne 2 `text-sm font-semibold`), centre = bandeau de score de 440 px, droite (lien « Fiche » `h-7 text-xs`, badge de schéma admin) ; écart de cellules 16 px | 52 px (1 ligne) ou 56 px (3 cellules avec score) | 52 px ou 56 px ; bouton « Quitter » `h-8 rounded-md border bg-card px-3 text-sm` + capsule `Échap` ; pas de lien Fiche |
| Corps | `space-y-4 pb-3` | `space-y-2` | `my-auto shrink-0 space-y-2` |
| Bloc plan + colonne | UN seul bloc `overflow-hidden rounded-xl border bg-card`, grille `minmax(0,1fr) 360px` à partir de `xl` (1280 px) ; sous `xl` la colonne passe sous la carte (max 80 vh) | grille `minmax(0,1fr) minmax(0,30rem)`, écart `gap-3`, chaque côté avec son propre cadre ; colonne cachée quand la rangée est posée | idem Cinéma |
| Colonne | 360 px, marges 12 px haut-bas et 14 px latéral, écart entre blocs 14 px, filet gauche `border-l`, équipes au-dessus (elles cèdent), fil dessous (5 lignes mini) ; la fiche d'un joueur REMPLACE toute la colonne | équipes (max 30 rem de haut) puis fil ; la fiche est une fenêtre ancrée | idem Cinéma, descend jusqu'au bas du cadre |
| Rangée de fiches | aucune | 2 camps de fiches autour du score de 270 px ; fiche min 115 px, écart 6 px ; en grande équipe 6 emplacements de 2 fiches empilées par camp (min 80 px, hauteur 74 px) | idem |
| Bloc de la toile | `relative p-3 pb-0` dans le bloc ; la barre est dans le bloc, bord à bord sous le plan | bloc `overflow-hidden rounded-lg border bg-card`, `pb-3` ; la barre est SOUS le bloc, à 6 px, sur le fond de page | barre fine posée sur le bas du plan dans le bloc (`absolute inset-x-px bottom-px z-20 rounded-b-[7px]`) ; ou sous le bloc si R33 éteint |
| Hauteur du plan | `CANVAS_HEIGHT_MIN` 360 px à `CANVAS_HEIGHT_CEILING` 720 px, marge basse 16 px, pas de 8 px (`mr/hooks/useReplayView.ts:62`, `useReplayViewport.ts:60-62`) | idem | idem, la fenêtre entière est offerte |
| Barre | complète : socle `mt-3 bg-background/40 px-3 pb-2.5 pt-3`, frise à pistes au-dessus, puis 3 zones (gauche : lecture automatique, son, vitesse ; centre : recommencer, −10, lecture 40×40, +10 ; droite : capture, export, 3 cadres, plein écran) | une ligne de 56 px : boutons carrés de 30 px, rayon 8 px, lecture 36 px, séparateurs 1×20 px, frise de 30 px min 240 px ; passe sur 2 lignes si la frise manque de place | idem ; forme fine = échelle 0,8, hauteur 32 px, `bg-background/45 backdrop-blur-[6px]`, pleine à la pause, au survol, au clavier |
| Zoom | croix flottante `absolute bottom-2 right-3`, `rounded-md border bg-card/85 p-1`, touches 20×20 px | − et + dans la barre | idem |
| Fil posé sur le plan | non | bas gauche, 5 lignes de 28 px, 350 px de large, marge gauche 16 px, bas 12 px (monte de 56 px quand la barre est posée sur le plan) ; texte à ombre portée `--scrim`, 2 lignes les plus anciennes estompées (0,7 et 0,85) | idem |
| Bouton « Vue » | `absolute right-2 top-2 z-20` DANS la toile, `h-8 rounded-md border border-border px-2.5 text-xs`, `bg-card/85 backdrop-blur-sm`, icône + « Vue » + nom du préréglage en gras (`font-semibold text-foreground`) ; ouvert : `bg-accent text-accent-foreground` | idem | idem |

Colonne des équipes : ligne de joueur 34 px (`GABARIT_LIGNE.rowPx`), nom 12,5 px ; fiche compacte 34 px (empilée) ou 24 px (ligne), nom 11,5 px, pastille de camp 6 px (`mr/model/cardGabarit.ts`). Ligne du fil : 28 px, colonnes fixes instant 30 / lieu 70 / tueur 70 / arme 18 / victime 70 px, texte secondaire 11,5 px (`mr/model/cotesFeedLine.ts`). Titre du fil : bandeau HUD `font-mono text-[10px] font-bold uppercase tracking-[.16em]` avec filet gauche 2 px neutre (`mr/model/hudBand.ts`).

### 5.2 Apparence des menus et panneaux existants

| Surface | Classes et dimensions (vérifiées) |
|---|---|
| Panneau « Vue » | portail, `fixed z-50 flex flex-col gap-3 overflow-y-auto rounded-lg border border-border bg-card px-3 py-3 text-sm shadow-xl`, largeur 416 px (max `100vw - 1rem`), ancré sous le bouton, bord droit aligné, écart 8 px, marge d'écran 8 px, hauteur mini 160 px (`mr/settings/useAnchoredPanel.ts`). En-tête : `h2 text-sm font-semibold` + croix 24×24. Titre de section : `h3 text-xs font-medium text-muted-foreground`. Titre de groupe de calques : `text-[11px] font-medium uppercase tracking-[0.06em] text-muted-foreground/70`. Grille de calques : `grid grid-cols-2 gap-x-3 gap-y-0.5` |
| Interrupteur de réglage | bouton `role="switch"` `h-7 w-full px-2 text-xs rounded`, libellé à gauche (tronqué), rail 28×16 px à droite : allumé `border-primary bg-primary` avec pastille `bg-primary-foreground` décalée de 10 px ; éteint `border-border bg-muted` avec pastille `bg-muted-foreground` ; survol `bg-muted` ; désactivé `opacity-50` (`mr/settings/ReplaySettingsToggle.tsx:75-127`) |
| Choix exclusif à boutons | `Button` variante `default` (pressé) ou `ghost`, taille `sm`, `h-7 px-2 text-xs` ; avec description : `h-auto flex-col items-start py-1.5`, description `text-[11px] opacity-80` |
| Segmenté | libellé `text-xs text-muted-foreground` à gauche, rail `flex rounded-md border border-border p-px`, options `rounded-[5px] px-2 py-1 text-[11px] font-medium` ; active `bg-primary text-primary-foreground` ; inactive `text-muted-foreground hover:bg-accent` |
| Pastilles d'étage | `h-[26px] rounded-full border px-[11px] text-[11.5px] font-medium tabular-nums` ; pressée `border-primary bg-primary text-primary-foreground` ; curseur `accent-primary` ; valeur à droite `min-w-[6ch] tabular-nums` |
| Menus de barre (vitesse, caméra) | `absolute bottom-[calc(100%+6px)]` (s'ouvrent vers le haut), `rounded-lg border border-border bg-popover p-1 shadow-xl`, entrées `h-7` (caméra : 28 px), `text-[12.5px]` (caméra `text-xs`), actif `bg-accent text-accent-foreground` + coche à droite pour le menu caméra ; caméra : 2 listes de 176 px, écart 8 px, hauteur max 320 px |
| Menu du filtre du fil | `absolute left-1.5 top-7 z-20 w-60 rounded-lg border bg-popover p-1 shadow-xl`, entrées `h-7 text-xs` avec glyphe, libellé, compte à droite (`tabular-nums text-muted-foreground`) ; séparateur « Proposées en plus » `text-3xs uppercase tracking-wider` |
| Fiche en fenêtre | `fixed z-50 border border-border bg-popover text-popover-foreground shadow-xl`, 330 px de large, rayon 12 px, marges 12/14 px, capsule `Échap` + croix 26×26 (rayon 6) |
| Bloc d'état vide | composant `EmptyStateNotice` (`components/ui/empty-state`), taille `compact` dans le groupe Vies ; bord pointillé `--placeholder-border` |
| Info (i) | SVG 12 px, cercle + point, `text-muted-foreground`, explication en infobulle `title` |
| Infobulles | attribut `title` natif partout (aucun composant d'infobulle) |
| Boutons de barre d'une ligne | carrés 30×30 px, rayon 8 px, transparents, survol `bg-accent text-accent-foreground`, anneau de focus `ring-2 ring-ring` ; état pressé `bg-accent text-accent-foreground` |
| Lecture | rond plein 36 px (barre d'une ligne) ou 40 px (barre complète) |
| Vitesse (barre complète) | `h-8 rounded-full border border-input px-3 text-[12.5px] font-medium tabular-nums` + chevron |
| Boutons capture/export (barre complète) | conteneur `rounded-full border bg-muted/40 p-1`, capture `bg-secondary`, export `bg-primary text-primary-foreground`, enregistrement `bg-destructive` |

### 5.3 Couleurs réellement employées

La page n'emploie presque pas de `--ac-*` pour son habillage : les menus, panneaux, boutons et fonds utilisent les jetons sémantiques de l'interface (`--background`, `--card`, etc.), définis dans `apps/web/src/styles/globals.css` (`:root` et `[data-theme='dark']` = thème sombre, qui est le thème par défaut ; `[data-theme='light']` = thème clair). Les jetons `--ac-*` ne servent que dans la scène (équipes, zones, fx) et quelques accents de fiche. Règle projet : aucun hex ni classe de couleur Tailwind dans `features/` (skill `color-tokens`).

**Jetons d'interface employés par la page** (vérifié dans `globals.css:20-160`) :

| Jeton | Thème sombre | Thème clair |
|---|---|---|
| `--background` | oklch(0.145 0 0) | oklch(0.985 0.004 255) |
| `--foreground` | oklch(0.985 0 0) | oklch(0.255 0.02 255) |
| `--card` (fond du bloc, du panneau Vue) | oklch(0.205 0 0) | oklch(0.995 0.002 255) |
| `--popover` (menus de barre, filtre, fiche en fenêtre) | oklch(0.269 0 0) | oklch(0.995 0.002 255) |
| `--primary` (interrupteur allumé, segment actif, pastille d'étage, export) | oklch(0.922 0 0) (gris très clair) | oklch(0.56 0.22 262) (bleu) |
| `--primary-foreground` | oklch(0.205 0 0) | oklch(0.985 0 0) |
| `--secondary` (bouton capture, lecture automatique allumée) | oklch(0.269 0 0) | oklch(0.95 0.01 255) |
| `--accent` (survol, état pressé) | oklch(0.371 0 0) | oklch(0.92 0.02 255) |
| `--accent-foreground` | oklch(0.985 0 0) | oklch(0.24 0.02 255) |
| `--muted` (rail d'interrupteur éteint, survol de ligne) | oklch(0.269 0 0) | oklch(0.96 0.008 255) |
| `--muted-foreground` (libellés secondaires, icônes de barre) | oklch(0.708 0 0) | oklch(0.52 0.015 255) |
| `--border` | oklch(0.275 0 0) | oklch(0.9 0.01 255) |
| `--input` (bordure de la vitesse, fond de piste de volume) | oklch(0.325 0 0) | oklch(0.93 0.01 255) |
| `--ring` (anneau de focus, bouton « Plan tourné ») | oklch(0.556 0 0) | oklch(0.63 0.12 262) |
| `--destructive` (enregistrement) | oklch(0.704 0.191 22.216) | oklch(0.637 0.208 25.331) |
| `--scrim` / `--on-scrim` (ombre du fil sur le plan, voile) | oklch(0 0 0) / oklch(1 0 0) | identiques |
| `--placeholder-border` (bloc d'état vide pointillé) | oklch(0.55 0 0) | = `--border` |
| `--radius` | 0.5rem (donc `rounded-lg` 8 px, `rounded-md` 6 px, `rounded-xl` 12 px, `rounded-sm` 4 px) | idem |

**Jetons `--ac-*` employés par la page** (noms relevés par balayage des appels `tokenCssVar` et `resolveToken` dans `mr/` et `lib/replay/`, valeurs de la palette « default » dans `lib/accessibility/palettes/default.ts` ; les `--ac-*` ont la MÊME valeur dans les deux thèmes et changent avec la palette choisie par l'utilisateur : « default », Okabe-Ito, Cividis, Tol Bright) :

| Jeton | Rôle dans le rejeu | Valeur (palette default, identique sombre et clair) |
|---|---|---|
| `--ac-team-ally` | camp allié (pions, pastilles, fil, barres de score) | #3B82F6 (Okabe-Ito : #0072B2) |
| `--ac-team-enemy` | camp adverse | #EF4444 (Okabe-Ito : #D55E00) |
| `--ac-zone-neutral` | objectif sans camp | #9CA3AF |
| `--ac-divergent-neutral` | « aucun camp » des marques, socles, véhicules | #60A5FA |
| `--ac-success`, `--ac-destructive`, `--ac-info` | états de la santé, du bouclier, jauges | #10B981, #EF4444, #3B82F6 |
| `--ac-legendary` | encadré du surbouclier | #FBBF24 |
| `--ac-stat-kills`, `--ac-stat-deaths`, `--ac-stat-assists` | frags, morts, assistances | #059669, #F43F5E, #0284C7 |
| `--ac-frag-heavy`, `--ac-frag-shoulder`, `--ac-frag-unattributed` | classes de frags du fil | #7C3AED, #06B6D4, #60A5FA |
| `--ac-marker-outline-band` / `--ac-marker-outline-rim` | cerne deux tons des pions | #080C12 / #FFFFFF |
| `--ac-map-relief-base` | teinte de la rampe du fond Relief | #3F7D8F, Pétrole, depuis le 2026-11-25 (Okabe-Ito et Tol Bright : #7559B5 ; Cividis : #3F8C4F) |
| `--ac-outcome-draw`, `--ac-bonus`, `--ac-extreme` | accents de l'écran de fin et de la fiche | #3B82F6, #A855F7, #C026D3 |
| `--ac-chart-series-1` à `8` | couleur « par joueur » (R39) et zones nommées | #93C5FD, #6366F1, #4F46E5, #4338CA, #3730A3, #10B981, #F59E0B, #EC4899 |

Les valeurs des palettes Cividis et Tol Bright pour les équipes sont des constantes (`SAFE_BLUE`, `SAFE_VERMILLION`, `TOL_BLUE`, `TOL_RED`) que je n'ai pas résolues : non relevées. Un prototype fidèle prend les jetons d'interface ci-dessus pour tout l'habillage et la palette « default » pour les équipes.

### 5.4 Polices et tailles

- Police : **Roboto** auto-hébergée, police variable graisses 100 à 900, repli `system-ui, sans-serif` (`globals.css:6-9,391`). Valeur numérique : `tabular-nums` partout où une valeur change (horloge, vitesse, comptes).
- Échelle employée : `text-sm` 14 px (titre du panneau, titre du match, bouton Quitter), `text-xs` 12 px (réglages, boutons), 12,5 px (barres, vitesse, horloge, noms de ligne), 11,5 px (pastilles d'étage, texte secondaire), 11 px (`text-3xs`, descriptions de préréglage, titres de groupe, segmentés), 10,5 px (touches de zoom, décompte), 10 px (légende de chaleur, bandeau du fil, mono), 9,5 px (libellés de pistes de la frise, majuscules espacées).
- Le bandeau du fil et ses chiffres : `font-mono font-bold uppercase tracking-[.16em]` à 10 px.
- Majuscules à espacement 0,06 em pour les titres de groupe ; 0,14 em pour les libellés de pistes.

### 5.5 Icônes

Aucune bibliothèque d'icônes dans la feature (vérifié : pas de `lucide-react` ni équivalent dans `package.json`, pas d'import de bibliothèque dans `mr/`). Toutes les icônes sont des SVG en ligne dessinés à la main, `viewBox="0 0 16 16"` en général, trait `currentColor` de 1,5, extrémités arrondies :

- `SlidersIcon` (bouton Vue) : 3 traits horizontaux (y = 3,5 ; 8 ; 12,5) + 3 disques de rayon 2 (`mr/settings/SlidersIcon.tsx`).
- Barre : `PlayIcon`, `PauseIcon`, `RestartIcon`, `CameraIcon` (capture), `ExportIcon`, `ExportSpinner`, `AutoPlayIcon` (mini-interrupteur 28×16, pastille 3,2 de rayon), `FullscreenIcon` (4 coins vers l'extérieur ou l'intérieur), `StandardFrameIcon` (carré 11×11 rx 1,5 + trait horizontal), `CinemaFrameIcon` (rectangle 13×8), `FullFrameIcon` (rectangle 13×12 + rectangle intérieur plein à 35 %), `VideoCameraIcon` (menu caméra, viewBox 24) (`mr/ui/ReplayTransportIcons.tsx`).
- Haut-parleur : local à `ReplaySoundControls.tsx:78-100` (corps plein, 2 ondes, ou croix quand coupé).
- Tuiles : cible (`TargetIcon`) et œil (`EyeIcon`, barré quand masqué), 16 px (`ReplayTileActionButtons.tsx`).
- Chevron de groupe 10 px (Vies), chevron de vitesse 10 px, chevron du filtre 9×6, coche 12 px (menu caméra), croix × (fermeture), flèche de retour (en-tête), rotation (« Plan tourné », 14 px), (i) 12 px.
- Zoom : glyphes texte `−`, `+`, `▲`, `◀`, `▶`, `▼`, `⌂` (pas de SVG).
- Glyphes d'événements du fil et des filtres : `EventGlyph` (`mr/ui/ReplayEventGlyph.tsx`, table `mr/model/feedGlyphs.ts`).

### 5.6 Détails de comportement à reproduire

- Le panneau Vue ne masque pas la carte : pas de fond assombri, pas de piège de focus, la lecture continue derrière ; fermeture par Échap, croix ou clic dehors ; le focus entre dans le panneau à l'ouverture et revient au bouton à la fermeture.
- Échap ferme la couche la plus intérieure (panneau, menu, fiche) avant de quitter le Plein cadre (`data-replay-escape-layer`).
- Les interrupteurs se mettent à jour partout en même temps (registre d'abonnés par clé).
- Les réglages indisponibles ne sont pas rendus (règle « pas de commande qui ne commande rien »), sauf R7 (grisé), R9 (grisé avec explication), R28 à R30 (retirés).
- Le bouton Vue porte le nom du préréglage actif, ou « Personnalisé ».
- Les calques et le panneau sont conçus pour le français ; chaque libellé existe en français et en anglais (`mr/i18n/fr.ts`, `en.ts`, `frames.i18n.ts`, `lives.i18n.ts`, `relief.i18n.ts`, `respawnForecast.i18n.ts`, `playerPanel.i18n.ts`, `roster.i18n.ts`).

---

## 6. Dérives de documentation relevées en passant (non traitées)

- `ReplayViewPanel.tsx:246` parle de « CINQ OUI/NON indépendants » pour les sons : il y en a quatre (`SOUND_CATEGORIES`, `mr/sound/replaySound.ts:459-464`).
- Plusieurs en-têtes parlent encore du « tiroir de réglages » (`ReplaySettingsToggle.tsx`, `useReplaySettings.ts`, `ReplayHeatmapSection.tsx`) alors que le composant est le panneau « Vue ».
- `mr/settings/useReplayDimension.ts:9` : le nom de la capacité 3D est une hypothèse écrite dans le code.

Zéro fix opportuniste : rien n'a été corrigé.
