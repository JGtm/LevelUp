# Maquette des réglages du rejeu : notes

Fichier à ouvrir : `index.html` (autonome, 433 Ko, aucun réseau). État mémorisé dans `localStorage`, clés `maquette-reglages-*`. Le bandeau violet en haut n'appartient pas à la page simulée : variante A ou B du panneau, thème, mode de jeu, « Comparer avec l'existant », « Réinitialiser la maquette », et le compte de commandes.

Contenu du match simulé : un extrait de 60 secondes (boucle), 8 joueurs en deux équipes (« Joueur A1 » à « Joueur B4 »), carte « Solution » dont les trois fonds (Plan, Rendu, Relief), la carte des hauteurs, les 10 emplacements d'arme et les deux bases de drapeau viennent des fichiers versionnés de `data/titles/halo_infinite/reference/`. Les 7 zones nommées, les armes, les équipements posés, les répliques et les trajets sont inventés pour la démonstration (cette carte n'a pas de zones nommées dans les données).

## Compte des commandes visibles à l'ouverture du panneau

Règle de comptage : une ligne de réglage visible, un interrupteur maître visible, la rangée de préréglages (1), le champ de recherche (variante B). Match de Capture du drapeau, cadre Standard, fond Plan, Vies replié dans l'existant.

| | Commandes | Détail |
|---|---|---|
| Existant | 29 sur 43 | 10 sections ; le groupe Vies replié cache 8 réglages ; 34 si on le déplie |
| Variante A | 14 | préréglages (1), maître de la rubrique (1), 12 lignes de la rubrique Joueurs ; 14 à 15 selon la rubrique et le cadre |
| Variante B | 19 | préréglages (1), 5 maîtres, 12 lignes de Joueurs ouverte, recherche (1) |

La proposition porte 51 réglages dans ses cinq rubriques (43 aujourd'hui) : 11 viennent des répliques, trois remontent de la barre ou de la colonne (Son, Volume, Filtre du fil).

## Correspondance R1 à R66

| Réglage de l'inventaire | Dans la proposition |
|---|---|
| R1 Préréglages | En tête du panneau, en puces ; couvrent 37 réglages (tous les interrupteurs des cinq rubriques + le filtre du fil), pas 16. Souples : « Combat · modifié », « Revenir à Combat », « Enregistrer comme ma vue » |
| R2 Visée, R3 Traînée | Rubrique Joueurs ; épinglées par défaut |
| R4 Zones | Carte, renommé « Zones nommées » (lève le triple sens du mot « zones ») |
| R5 Fond | Carte, segmenté |
| R6 Traits du relief | Carte, sous Fond. FUSIONNÉ : une seule valeur pour tous les fonds (au lieu d'une clé par fond), allumée par défaut |
| R7 Plan des murs | Carte, sous Fond. Plus grisé : non proposé sur le fond Plan (sans objet) |
| R8 Étage | Carte, segmenté Tout / Bas / Étage 1 / Étage 2 (plus de pastilles : une seule forme de choix court) + curseur « Hauteur » quand une coupe est active |
| R9 Suivre le joueur ouvert | Carte, renommé « Étage de la fiche ouverte » (« suivre » ne désigne plus que la caméra) |
| R10 Emplacements d'arme, R11 Armes au sol | Objets |
| R12 Armes spéciales seulement, R13 armes de départ | Objets, sous Armes au sol ; R13 devient « Y compris les armes de départ » |
| R14 Véhicules, R15 Équipements posés | Objets |
| R16 Objets lâchés au sol, R17 Objets non identifiés | Objets, sous Équipements posés ; R16 devient « Objets lâchés » (lève le double « au sol ») |
| R18 Objectifs du mode, R19 Drapeaux | Objets, Drapeaux sous Objectifs du mode |
| R20 VIP, R21 Porteur du crâne, R22 Bombe | Prévus comme sous-calques d'Objectifs du mode ; NON maquettés (aucun des deux modes simulés n'en a) |
| R23 Vies (groupe repliable) | SUPPRIMÉ comme groupe replié : sous-titre « Vies » dans Joueurs, lisible sans clic. Le compte « N sur M » passe à la rubrique |
| R24 Apparitions, R25 Morts, R26 Tueurs | Joueurs, sous Vies |
| R27 Probabilité de réapparition | Joueurs, sous Vies |
| R28 Dessin | Joueurs, sous la probabilité ; options renommées « Teintes et points / Teintes / Points » |
| R29 Qui, R30 Quand | Joueurs, sous Vies, affichés si une trace est allumée ; « Joueur suivi » devient « Point de vue » |
| R31 Orientation | Carte, segmenté |
| R32 Fil des éliminations | Fil, « Fil posé sur le plan » (épinglable). Au Standard : phrase en pointillé au lieu de la ligne (réglage sans objet) |
| R33 Barre de lecture fine | Pied du panneau, ligne « Lecture », au Plein cadre seulement |
| R34 Effets de tirs, R35 Effets de mort | Joueurs |
| R36 Carte de chaleur, R37 Mesure, R38 Durée | Carte (titre et libellé ne se répètent plus : un interrupteur, ses deux options dessous) |
| R39 Couleur des points | Joueurs, segmenté |
| R40 à R43 Sons par catégorie | Son, sous « Catégories » |
| R44 Son, R45 Volume | Barre (interrupteur et volume, inchangés) ET rubrique Son : même état, un seul endroit de décision |
| R46 Vitesse, R47 Lecture automatique | Barre, inchangés (lecture automatique allumée par défaut dans la maquette seulement, pour qu'elle vive à l'ouverture) |
| R48 Cadre, R49 Plein écran | Barre, inchangés ; T, F, Échap comme dans l'application |
| R50 Zoom, R51 Cadrage | Croix du Standard, boutons de la barre en Cinéma et Plein cadre, inchangés |
| R52 Plan tourné | Barre d'une ligne, inchangé (n'apparaît que si l'Auto a tourné le plan ; ne se produit pas avec ce plan) |
| R53 Caméra : suivre | Renommé « Caméra : suivre » (menu de la barre, icône cible des tuiles et des fiches) |
| R54 Point de vue | Liste de la frise au Standard, 2e liste du menu caméra ailleurs ; libellé « Point de vue » partout |
| R55 Pistes de la frise | Chevron de la frise, inchangé |
| R56 2D / 3D | Non maquetté (inerte aujourd'hui) |
| R57 Filtre du fil | Fil, liste « Filtre du fil » : réglable aussi au Cinéma et au Plein cadre, où la colonne est cachée (constat 19). Reste dans le titre du fil au Standard |
| R58 Suivre, R59 Masquer, R60 Tout afficher, R61 Seul, R62 Fiche | Tuiles, rangée de fiches, colonne et fiche, inchangés |
| R63 à R66 Export | Dialogue inchangé dans son contenu (non fonctionnel) ; proposition : « Inclure le son » suit le son de la page |
| Nouveaux (répliques) | Joueurs : Répliques en bulles, Icône de voix. Fil : Répliques dans le fil, cinq familles, Fréquence. Son : Voix des répliques, Langue des répliques |

Supprimés ou fusionnés : R6 (trois clés en une), R7 (plus de réglage grisé), R23 (groupe replié), R8 (pastilles remplacées par le segmenté), et la fusion de l'emplacement du son (barre + Son = un état).

## Ce qui a été arbitré

1. **Joueurs reste la plus grosse rubrique (12 lignes).** Les Vies y sont un sous-titre, pas un groupe replié : la probabilité de réapparition, allumée par défaut, n'est plus cachée.
2. **Un préréglage = les interrupteurs des cinq rubriques + le filtre du fil.** Hors préréglages : fond, orientation, étage, couleur des points, Qui / Quand, Dessin, Mesure / Durée, Fréquence, son, volume, langue. « Modifié » = au moins un réglage couvert diffère de la vue de départ.
3. **Mes vues** : 3 au plus, renommées en place, étoile = vue d'ouverture, liste « À l'ouverture d'un rejeu » (« Dernière vue utilisée » par défaut, donc comportement actuel conservé sans choix), rattachement facultatif à un mode (il passe avant la vue d'ouverture, pour ce mode seulement). Ajout non demandé : « Mettre à jour cette vue » quand on modifie l'une de ses vues.
4. **Réinitialiser** rétablit les réglages d'usine ; il garde Mes vues et les épingles.
5. **Accès rapide** : interrupteurs seulement, 5 au plus, rangé à gauche du bouton Vue. Un réglage épinglé sans objet (le fil posé au Standard) reste affiché, grisé, avec la raison en infobulle. Masqué dans la vue « existant ».
6. **Répliques** : « Répliques dans le fil » est un interrupteur ET une valeur du filtre du fil (affichée seulement quand l'interrupteur est allumé). Familles et fréquence filtrent ensemble bulles, icône, fil et voix. Fréquence : « plus fréquentes » = phrases dites 4 fois ou plus dans l'extrait ; « rares » = 2 fois ou moins. Les cris sans paroles n'existent pas dans la maquette.
7. **Pied de panneau** plutôt qu'une rubrique « Lecture » : une seule ligne (barre fine) et un seul cadre concerné.
8. **Variantes** : A (onglets verticaux, 104 px + 278 px de contenu, une rubrique à la fois) et B (accordéon + recherche, Joueurs ouvert). Même état, mêmes préréglages, mêmes Mes vues au-dessus du corps.
9. **« Comparer avec l'existant »** montre aussi les réglages que l'application masque selon le match (VIP, crâne, bombe), marqués « absent de ce match » ; les 4 préréglages y portent leurs 16 calques d'origine et la mention « Personnalisé ».
10. Un seul bouton « Vue » dans le panneau actuel affiche le nom du préréglage ou « Personnalisé » ; dans la proposition, « Combat · modifié ».

## Ce qui n'est pas fonctionnel ou pas fidèle

- Hors périmètre : 2D / 3D, retour, lien « Fiche du match », capture d'image et export (le dialogue s'ouvre, rien n'est écrit), plein écran du système (bouton du navigateur seulement).
- Son : pas de vrai son d'arme ; un petit bip par catégorie (volume respecté) et un témoin visuel au-dessus de la carte. Pas de voix enregistrées : la réplique est affichée et un bip joue.
- Orientation et fond retenus globalement, pas par carte ; les réglages ne sont pas partagés avec la Tactique ni « Occupation du terrain ».
- Les lignes du fil, les noms d'armes courts, les médailles, les équipements posés, les armes au sol et l'étage (3 rangs tirés des hauteurs) sont des approximations de l'application, pas ses calculs.
- L'en-tête, la rangée de fiches et la colonne sont recomposés d'après les composants réels (cotes du fichier d'inventaire), mais pas rendus avec leur code. La barre d'application en haut est un décor.
- Extrait de 60 secondes : l'horloge va de 0:00 à 1:00.
- Traits du relief et plan des murs : dessinés par le navigateur à partir de la carte des hauteurs, pas du maillage de navigation.

## Vérification faite (statique)

- Les trois scripts extraits passent `node --check` ; ESLint `no-undef` : 0 erreur (aucun identifiant indéfini).
- Page chargée dans jsdom (DOM simulé, canevas neutralisé) : démarrage, ouverture du panneau, onglets, interrupteurs, maîtres, préréglages (modifié / revenir), Mes vues (enregistrer, rattacher, défaut), épingles (limite de 5), variantes A et B, recherche, comparaison, trois cadres, menus de barre, fiche, Seul / Masquer, mode Assassin, thème, réinitialisation : aucune erreur levée.
- Dessin de la carte rejoué dans node-canvas (png, couleurs converties en hex pour l'essai) : fonds Plan, Rendu, Relief, coupe par étage, chaleur, vies, probabilité, bulles, suivi avec zoom, orientation : relus sur images.
- NON vérifié : le rendu CSS de la page entière dans un vrai navigateur (aucun navigateur lancé, comme demandé), le thème clair, les largeurs 1280 et 1920, les bips.
