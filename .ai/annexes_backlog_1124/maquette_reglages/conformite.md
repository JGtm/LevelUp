# Conformité du code à la maquette des réglages (variante A) : item M.4e

Date : 2026-11-24. Branche `bl24/m`. Rien n'a été vu à l'écran par l'agent qui a écrit ce document : chaque ligne est relue
dans le code et dans les tests (statut « conforme » = le code et au moins un test disent la même chose que la maquette ;
« à voir à l'écran » = le rendu CSS n'a pas été regardé). La maquette (`index.html`, `notes.md`) fait foi ; les chemins sont
sous `apps/web/src/features/match-replay/` sauf mention.

Comment lire : `settings/` = le dossier des réglages ; la table des réglages est `settings/settingsTable.ts` ; ses libellés
sont rangés dans `settings/settingsText.ts` (textes d'origine) et `i18n/settings.i18n.ts` (le panneau).

## 1. Écarts avec la maquette (un par ligne, avec la raison)

| # | Écart | Raison |
|---|---|---|
| E1 | Aucun réglage de répliques : bulles, icône de voix, répliques dans le fil (et la valeur « Répliques » du filtre du fil), cinq familles, fréquence, voix des répliques | Hors périmètre de M.4 (piste R). La table s'étend par une ligne (voir §4). |
| E2 | « Langue des répliques » : jamais codé | Décision D-15 (langue de l'interface, par `config/locales.toml`). |
| E3 | Rubriques sans les phrases des répliques (Joueurs : « regard, trajet, effets, vies » ; Fil : « Le fil des événements du match » ; Son : « Sons, volume et catégories ») | Conséquence de E1. |
| E4 | Préréglages : 31 réglages couverts (28 de la maquette hors répliques + VIP, porteur du crâne et bombe), pas 37 | Les 9 réglages de répliques manquent (E1) ; VIP, crâne et bombe, « prévus comme sous-calques » mais non maquettés, restent couverts comme dans les préréglages d'avant (ils suivent « Drapeaux » dans les quatre tables). Texte du préréglage Objectif sans « et leurs répliques ». |
| E5 | VIP, Porteur du crâne, Bombe : codés, sous « Objectifs du mode », non épinglables | La maquette ne les montre pas (notes : « NON maquettés ») ; le code existant les porte. |
| E6 | « Drapeaux » (et VIP, crâne, bombe) restent affichés quand « Objectifs du mode » est éteint ; la maquette les masque | Leur couche reste dessinée « objectifs éteints » : masquer leur commande la rendrait inatteignable. Le masquage à mère éteinte est une propriété de la table (`hideWhenParentOff`), posée pour les filtres seulement. |
| E7 | Appliquer un préréglage ou une vue n'écrit que les réglages que le match propose ; « modifié » ne compte que ceux-là | Demande du propriétaire (un réglage sans objet ne rend pas la vue modifiée) et règle d'avant (un préréglage ne touche jamais un calque absent du match). La maquette écrit et compare tous les réglages couverts. |
| E8 | Mode de jeu d'une vue : six « enjeux » lus dans le film (à drapeau, à crâne, à VIP, à bombe, à zones, sans objectif), pas les deux modes fixes de la maquette (Capture du drapeau, Assassin) | Le film ne porte aucun identifiant de mode stable ; `mode_category` de l'en-tête est une catégorie de liste de parties (Assassin, Fiesta, BTB, Ranked…), pas un mode. L'enjeu lu du film (`matchModeKey`) ne dépend ni du titre ni de la langue. **Choix de produit à confirmer.** |
| E9 | Pas de toasts : le résultat d'un geste se dit dans une ligne d'état sous le panneau (région `role="status"`) | La maquette n'a pas d'équivalent dans l'application ; même texte. Un nom de vue vide laisse le champ ouvert sans message. |
| E10 | Accès rapide nommé « Accès rapide aux réglages » (maquette : « aux calques ») | Vocabulaire : le groupe porte des réglages, pas seulement des calques. |
| E11 | Infobulles des lignes : celles d'avant (longues, avec la réserve de mesure), pas les phrases courtes de la maquette ; options de choix idem | « Chaque réglage garde son effet exact » : on ne perd pas l'explication. Libellés renommés comme la maquette (Zones nommées, Objets lâchés, Y compris les armes de départ, Étage de la fiche ouverte, Point de vue, Teintes et points, Fil posé sur le plan). |
| E12 | Étage : le segmenté suit les étages détectés (3 dans la maquette), la valeur du curseur dit le rang (« Étage 1 »), jamais un nombre | Règle d'avant (D-14 : jamais de mètres) ; le rang est celui des pastilles d'avant. |
| E13 | Une rubrique sans aucune ligne pour ce match n'a pas d'onglet (Objets sur un Assassin, Son sans piste sonore) | Règle d'avant : pas de commande qui ne commande rien. La maquette simule deux matchs qui ont toutes les rubriques. |
| E14 | « Traits du relief » : pas de fusion en une valeur ; les trois clés `replay-relief-lines.<fond>` restent (allumés d'office sur le Relief, éteints sur le Plan et le Rendu). Le panneau porte UN interrupteur qui lit et écrit la clé du fond affiché. La maquette et ses notes (R6) fusionnaient en une valeur allumée d'office | Le rendu par défaut du Plan a été validé tel quel par le propriétaire (rejeu, Tactique, « Occupation du terrain »), et une valeur allumée d'office sur le Plan imposerait le calcul de l'image des hauteurs à l'ouverture ; le réglage est partagé avec les deux autres pages. |
| E15 | (voir la correction suivante : réglages partagés hors des préréglages) | — |
| E16 | Aucun des outils de la maquette : variante B, recherche, « Comparer avec l'existant », thème, sélecteur de mode, « Rouvrir le rejeu », « Réinitialiser la maquette », compte de commandes | Outillage de démonstration. |
| E17 | Pas de bandeau d'état « Point de vue : … · Caméra : … · Étage : … » sur la carte, ni de témoin visuel du son | Décor de la maquette, sans équivalent dans l'application. |
| E18 | Fiche d'un joueur : pas de bouton « Caméra : suivre » ni « Prendre le point de vue », pas de sous-titre « point de vue du rejeu » | Absents de l'inventaire (R58 à R62 « inchangés ») et de la fiche d'application, où Suivre et Masquer vivent sur les tuiles ; non codés. |
| E19 | Le commutateur 2D / 3D se nomme « Dimension du rejeu » (avant : « Vue du rejeu ») | Règle « Vue ne désigne que le panneau » ; la maquette ne le montre pas. |
| E20 | Les couches du panneau portent `role="dialog"` ; l'accès rapide, `role="toolbar"` sans navigation aux flèches | Comme la maquette ; aucune navigation à flèches dans l'accès rapide (boutons tabulables). |

## 2. Grille de conformité : chaque élément de la maquette

Légende : **C** = conforme, **E#** = écart du tableau 1, **VE** = à voir à l'écran (le rendu n'a pas été regardé).

### Panneau (variante A)

| Élément | État | Où |
|---|---|---|
| Bouton « Vue » : icône, « Vue », nom en gras, « · modifié » | C (VE) | `settings/ReplayViewMenu.tsx` |
| Panneau 416 px, portail, ancré sous le bouton, défile en interne, ni voile ni piège de focus | C (VE) | `settings/ReplayViewPanel.tsx` |
| En-tête « Vue » + croix « Fermer la vue » | C | idem |
| Échap, croix et clic dehors ferment ; focus entre au panneau puis revient au bouton | C | `useDismissable` ; tests `ReplayViewPanel.test.tsx`, `ReplayViewMenu.test.tsx` |
| Échap ferme la couche la plus intérieure d'abord (champ de nom, confirmation de remise à zéro) | C | `ReplayViewChips.tsx`, `ReplayViewPanel.tsx` (`PanelFooter`) |
| Section « Préréglages » : quatre puces Complet, Combat, Déplacements, Objectif | C | `ReplayViewChips.tsx`, `viewPresets.ts` |
| Puce active pressée ; « · modifié » dans la puce ; pointillé quand modifiée | C (VE) | `ReplayViewChips.tsx` |
| Ligne d'état : « Combat · modifié », n réglages changés, « Revenir à Combat », « Mettre à jour cette vue » (vue enregistrée), « Enregistrer comme ma vue » | C | idem |
| Ligne d'état au repos : phrase du préréglage + « Un préréglage est un point de départ… » | C | idem |
| Formulaire de nom (24 caractères), Enregistrer, Annuler ; limite de trois vues dite | C | idem |
| Puces de Mes vues séparées des préréglages, étoile sur la vue d'ouverture | C | idem |
| « Mes vues (n sur 3) » repliable ; vide : phrase d'aide | C | `ReplayViewChips.tsx`, `ReplayMyViews.tsx` |
| Vue : renommer en place, étoile (vue d'ouverture), supprimer, rattachement à un mode « facultatif » | C (mode : E8) | `ReplayMyViews.tsx` |
| Liste « À l'ouverture d'un rejeu » : Dernière vue utilisée, préréglages, vues ; phrase sur le mode et le défaut | C | idem |
| Vue rattachée à un mode : passe avant la vue d'ouverture, pour ce mode | C (mode : E8) | `savedViews.ts` (`openingViewId`), `useViewsController.ts` |
| Cinq onglets verticaux (104 px) avec compte « N sur M » ; une rubrique visible | C (VE) | `ReplayRubricTabs.tsx` |
| En-tête de rubrique : titre, compte, interrupteur maître à trois états, phrase | C (E3) | idem |
| Sous-titres « Vies » (Joueurs) et « Catégories » (Son) en majuscules espacées | C (VE) | idem, `settingsTable.ts` (`sub`) |
| Filles en retrait, plus discrètes ; filtres masqués quand leur mère est éteinte | C (E6) | `ReplaySettingRows.tsx`, `settingsAvailability.ts` |
| Punaise au survol, au focus, et quand épinglé ; cinq au plus, message à la sixième | C | `ReplaySettingRows.tsx`, `quickAccess.ts` |
| Segmenté en colonne : libellé au-dessus, options sur toute la largeur | C (VE) | `ReplaySettingRows.tsx` (`ChoiceRow`) |
| Pied : « Lecture » + Barre de lecture fine, Plein cadre seulement | C | `ReplayViewPanel.tsx` |
| Pied : « Réinitialiser » + confirmation ; garde Mes vues, la vue d'ouverture et les épingles | C | `ReplayViewPanel.tsx`, `useViewsController.ts` |
| Ligne d'état du résultat d'un geste | E9 | `ReplayViewPanel.tsx` |

### Rubrique Joueurs

| Ligne (libellé) | État | Réglage et clé |
|---|---|---|
| Visée, Traînée (épinglées d'usine) | C | `replay-show-aim`, `replay-show-trail` |
| Couleur des points : Par équipe / Par joueur | C | `replay-marker-colors` |
| Effets de tirs, Effets de mort | C | `replay-show-shot-fx`, `replay-show-kill-fx` |
| Vies : Apparitions, Morts, Tueurs | C | `replay-show-spawns` / `deaths` / `killers` |
| Qui : Tous / Mon équipe / Point de vue ; Quand : Toute la partie / Jusqu'ici ; affichés si une trace est allumée | C | `replay-lives-who`, `replay-lives-when` |
| Probabilité de réapparition ; Dessin : Teintes et points / Teintes / Points (sous la probabilité) | C | `replay-show-respawn-forecast`, `replay-respawn-forecast-draw` |
| Refus de la probabilité : bloc pointillé avec le motif | C | `ReplayRubricTabs.tsx` (`RubricNotes`) |
| Répliques en bulles, Icône de voix | E1 | — |

### Rubrique Carte

| Ligne | État | Réglage et clé |
|---|---|---|
| Fond : Plan / Rendu / Relief | C | `replay-background-style` (partagée) |
| Traits du relief (sous Fond ; celui du fond affiché ; allumés d'office sur le Relief seulement) | C (E14) | `replay-relief-lines.<fond>` (`lib/replay/useReliefLines.ts`, inchangé) |
| Plan des murs (sous Fond ; non proposé sur le fond Plan) | C | `replay-show-wall-plan` (partagée) |
| Étage : Tout / rangs ; curseur « Hauteur » quand une coupe est active | C (E12) | `settings/ReplayFloorRow.tsx` ; non mémorisé |
| Étage de la fiche ouverte (sous Étage ; grisé et expliqué sans fiche) | C | `relief.floor.follow` |
| Orientation : Auto / 0° / 90° / 180° / 270° | C | `replay-orientation.<carte>` (partagée) |
| Zones nommées | C | `replay-show-zones` |
| Carte de chaleur ; Mesure, Durée dessous | C | `replay-show-heatmap`, `replay-heatmap-mode`, `replay-heatmap-span` |

### Rubrique Objets

| Ligne | État | Réglage et clé |
|---|---|---|
| Emplacements d'arme ; Armes au sol ; Armes spéciales seulement et Y compris les armes de départ (dessous) | C | `replay-show-weapon-pads`, `-ground-weapons`, `-ground-weapons-special-only`, `-ground-weapons-starting` |
| Véhicules | C | `replay-show-vehicles` |
| Équipements posés ; Objets lâchés et Objets non identifiés (dessous) | C | `replay-show-placements`, `-dropped-placements`, `-unnamed-placements` |
| Objectifs du mode ; Drapeaux dessous | C (E6) | `replay-show-mode-objectives`, `replay-show-flag-carries` |
| VIP, Porteur du crâne, Bombe | E5 | `replay-show-vip-crown`, `-skull-carrier`, `-bomb-carrier` |

### Rubrique Fil

| Ligne | État | Où |
|---|---|---|
| Filtre du fil : liste avec le nombre de lignes de chaque filtre ; réglable aussi au Cinéma et au Plein cadre | C | `replay-feed-filter` (`hooks/useFeedFilterChoice.ts`) ; la colonne du Standard garde son menu |
| Fil posé sur le plan (épinglé d'usine) ; phrase en pointillé au Standard | C | `replay-show-plan-feed` |
| Répliques dans le fil, familles, Fréquence | E1 | — |

### Rubrique Son

| Ligne | État | Où |
|---|---|---|
| Son (interrupteur), Volume (curseur, grisé et à zéro sans son) : même état que la barre | C | `useReplaySound` ; la barre reste inchangée |
| Catégories : Armes, Grenades, Mêlée, Équipements | C | `replay-sound-categories-off` |
| Voix des répliques, Langue des répliques | E1, E2 | — |

### Accès rapide

| Élément | État | Où |
|---|---|---|
| Barre à gauche du bouton Vue, dans les trois cadres, cinq boutons au plus | C (VE) | `settings/ReplayQuickAccess.tsx` |
| Défaut : Visée, Traînée, Carte de chaleur, Fil posé sur le plan (+ bulles : E1) | C | `settings/quickAccess.ts` |
| Bouton = interrupteur, aria-pressed, infobulle « : activé / désactivé » | C | idem |
| Réglage épinglé sans objet : affiché, grisé, raison en infobulle | C | idem |

### Hors panneau

| Élément | État | Où |
|---|---|---|
| Menu caméra : « Caméra : suivre » puis « Point de vue » | C | `ui/ReplayCameraMenu.tsx`, `i18n/playerPanel.i18n.ts` |
| Tuile : « Caméra : suivre » / « Ne plus suivre », noms accessibles « Suivre X (caméra) » | C | `ui/ReplayTileActionButtons.tsx` |
| Liste de point de vue de la frise nommée « Point de vue » | C | `i18n` (`viewpointLabel`) |
| Mention « point de vue » à côté du joueur regardé | C | `ui/ReplayRosterRow.tsx` |
| Export : « Inclure le son » suit le son de la page | C | `export/ReplayExportDialog.tsx` |
| Barres, vitesse, lecture automatique, cadres, zoom, Plan tourné, raccourcis | C (inchangés) | — |
| Fiche : Seul / Masquer / Tout afficher | C (inchangés ; E18) | — |

## 3. Où est chaque réglage de l'inventaire (R1 à R66) dans le code livré

| Réglage | Dans le code livré |
|---|---|
| R1 Préréglages | `settings/viewPresets.ts` (tables), `settings/ReplayViewChips.tsx`, `settings/useViewsController.ts` ; clé `replay-view-preset` (vue active) |
| R2 Visée, R3 Traînée | table `aim`, `trail` : rubrique Joueurs ; état `useReplaySettings.ts` |
| R4 Zones | `zones` (« Zones nommées »), Carte |
| R5 Fond | `bg`, Carte ; `useBackgroundStyle.ts` |
| R6 Traits du relief | `reliefLines`, Carte ; `lib/replay/useReliefLines.ts` (inchangé : une clé par fond) |
| R7 Plan des murs | `wallPlan`, Carte (non proposé sur le fond Plan) |
| R8 Étage | `floor`, Carte ; `settings/ReplayFloorRow.tsx` |
| R9 Suivre le joueur ouvert | `floorFollow` (« Étage de la fiche ouverte »), sous Étage |
| R10 Emplacements d'arme, R11 Armes au sol | `weaponPads`, `groundWeapons`, Objets |
| R12, R13 | `groundSpecial`, `groundStarting`, sous Armes au sol |
| R14 Véhicules, R15 Équipements posés | `vehicles`, `placements`, Objets |
| R16, R17 | `placeDropped` (« Objets lâchés »), `placeUnnamed`, sous Équipements posés |
| R18 Objectifs du mode, R19 Drapeaux | `objectives`, `flags`, Objets |
| R20 VIP, R21 Crâne, R22 Bombe | `vip`, `skull`, `bomb`, sous Objectifs du mode (E5) |
| R23 Vies (groupe replié) | supprimé comme groupe : sous-titre « Vies » ; clé `replay-lives-group-open` effacée par la reprise |
| R24 à R26 | `spawns`, `deaths`, `killers`, Joueurs |
| R27, R28 | `forecast`, `forecastDraw`, Joueurs |
| R29, R30 | `livesWho` (« Point de vue » au lieu de « Joueur suivi »), `livesWhen`, Joueurs |
| R31 Orientation | `orientation`, Carte |
| R32 Fil des éliminations | `planFeed` (« Fil posé sur le plan »), Fil |
| R33 Barre de lecture fine | `thinBar`, pied du panneau (Plein cadre) |
| R34, R35 Effets | `shotFx`, `killFx`, Joueurs |
| R36 à R38 Chaleur | `heatmap`, `heatMode`, `heatSpan`, Carte |
| R39 Couleur des points | `colors`, Joueurs |
| R40 à R43 Sons par catégorie | `catWeapon`, `catGrenade`, `catMelee`, `catEquip`, Son |
| R44 Son, R45 Volume | `sound`, `volume`, Son, ET barre (inchangée) : même état (`useReplaySound`) |
| R46 Vitesse, R47 Lecture automatique | barres : `ui/ReplaySpeedMenu.tsx`, `ui/ReplayAutoPlayToggle.tsx` (inchangés) |
| R48 Cadre, R49 Plein écran | `ui/ReplayFrameButtons.tsx` (inchangés) |
| R50 Zoom, R51 Cadrage | `ui/ReplayZoomControl.tsx`, barre d'une ligne (inchangés) |
| R52 Plan tourné | `ui/ReplayRotatedButton.tsx` (inchangé) |
| R53 Caméra : suivre | menu caméra `ui/ReplayCameraMenu.tsx` + icône cible des tuiles (renommés) |
| R54 Point de vue | liste de la frise `ui/ReplayViewpointSelect.tsx` et 2e liste du menu caméra (nom « Point de vue ») |
| R55 Pistes de la frise | `ui/ReplayTimelineTracks.tsx` (inchangé) |
| R56 2D / 3D | `ui/ReplayDimensionSwitch.tsx` (nom du groupe : E19) |
| R57 Filtre du fil | `feedFilter`, rubrique Fil ; menu du titre du fil inchangé |
| R58 à R62 | tuiles, rangée, colonne, fiche (inchangés ; noms accessibles « (caméra) ») |
| R63 à R66 Export | `export/ReplayExportDialog.tsx` ; R65 s'ouvre sur le son de la page |

## 4. Ajouter un réglage (les répliques, par exemple)

Une ligne dans `settings/settingsTable.ts` (rubrique, forme, sous-titre, mère, défaut, couvert par les préréglages ou non,
épinglable), son libellé dans `settings/settingsText.ts`, ses valeurs dans les tables de `settings/viewPresets.ts` (un
préréglage qui ne dit rien du réglage ne le touche pas), sa disponibilité dans `isProposed` (`settingsAvailability.ts`),
et son lien à l'état dans `settingsBinding.ts`. Le panneau, l'accès rapide, le « N sur M », l'interrupteur maître, le
compte des modifications et Mes vues n'ont rien à changer. L'icône d'accès rapide : `settings/SettingIcons.tsx`.

## 5. Reprise des préférences (ancien état -> nouvel état)

| Ancien état | Nouvel état |
|---|---|
| Les ~49 clés `replay-*` des réglages | relues telles quelles (aucune ne change de nom) |
| `replay-relief-lines.<fond>` (trois clés) | inchangées : aucune reprise |
| Préréglage Combat (ou Déplacements) retenu, treize calques encore à son état, traces des vies sans clé | `replay-show-deaths` et `-killers` (ou `-spawns`) écrits à `true` une fois, pour que la vue reste la sienne |
| `replay-lives-group-open` | effacée (le groupe ne se replie plus) |
| `replay-view-preset` (dernier préréglage) | la vue active ; un préréglage s'y lit comme avant ; absente, invalide ou « personnalise » : Complet |
| Rien (premier lancement) | marque `replay-prefs-version` = 2 ; les épingles d'usine, aucune vue, pas de vue d'ouverture |
| Clés neuves | `replay-views` (3 vues au plus, validées), `replay-view-default`, `replay-pins`, `replay-view-tab` |

Idempotente : la marque de version empêche un second passage ; aucune valeur n'est jamais réécrite par-dessus une clé existante. Test : `settings/settingsMigration.test.ts`.

## 6. Points à contrôler à l'écran (rien n'a été vu par l'agent)

1. Panneau Variante A au Standard, Cinéma, Plein cadre : onglets, maître à trois états, filles en retrait, pointillé de la puce modifiée, défilement du panneau sur un écran bas.
2. Accès rapide : cinq boutons, grisé du fil au Standard, infobulles ; même état que la ligne du panneau.
3. Traits du relief : l'interrupteur du panneau suit le fond affiché (allumé sur le Relief, éteint sur le Plan et le Rendu) ; la Tactique et « Occupation du terrain » montrent la même chose qu'avant.
4. Préréglage Combat puis un réglage à la main : « Combat · modifié », Revenir, Enregistrer comme ma vue, trois vues, vue d'ouverture, rattachement à un mode (E8).
5. Menu caméra « Caméra : suivre » / « Point de vue », mention « point de vue » sur la ligne, liste de la frise.
6. Export : la case « Inclure le son » suit le son de la page.
