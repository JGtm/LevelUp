# Grille de conformité aux maquettes — refonte de la page du rejeu (2026-11-17)

Relevé fait par le pilote le 2026-11-17 sur les copies de `.ai/annexes_refonte_ui_rejeu/maquettes/`
(canevas https://claude.ai/artifact/GkqDthVG6jGxx3A5WLfXrr, version 26). Sert de référence aux briefs des
agents, à la relecture sur pièces de chaque item d'interface, à la revue finale R.1 et à la validation R.4 du
plan `.ai/PLAN_REFONTE_UI_REJEU_2026-11-15.md`. L'audit 0.3 la complète et la corrige contre le code ; toute
ligne ajoutée ou modifiée porte sa date.

## Règle

1. Les tableaux qui font foi : A, J, K, L, D, H. Les tableaux B, C, E, F, G, I ne sont jamais donnés à un
   agent, sauf le bloc cité en A-10.
2. **La maquette fait foi sur tout ce qu'aucune décision datée du user ne contredit.** Les écarts voulus sont
   la liste fermée ci-dessous. Tout autre écart est un défaut à corriger, ou une question posée au user AVANT
   le code.
3. Les cotes relevées sont celles d'un écran de 1440 px. Elles entrent dans le code par les cotes nommées
   existantes (`model/cardGabarit.ts`) ou par une constante nommée, jamais par un littéral répété.
4. Couleurs et police des maquettes (valeurs hexadécimales, Roboto) ne se recopient pas : tokens de l'app
   (`team-ally`, `team-enemy`, tokens de surface, de texte et de focus).

| Tableau | Fichier | Fait foi pour |
|---|---|---|
| A | `Main.dc.html` | Standard : page, en-tête, colonne (équipes en lignes, fil) |
| J | `FicheJoueur.dc.html` | fiche détaillée du Standard (la colonne devient la fiche) |
| K | `CinemaRangee.dc.html` | Cinéma : rangée de fiches autour du score, fil sur le plan, barre d'une ligne |
| L | `CinemaRangeeFiche.dc.html` | fiche détaillée en fenêtre (Cinéma, Plein cadre) |
| D | `PleinCadre.dc.html` | Plein cadre : K sans l'app autour |
| H | `RoninCinema.dc.html` | rotation du plan ; colonne quand la rangée ne tient pas ; ligne dépliée du joueur suivi |

## Écarts voulus (liste fermée, décision citée)

| Réf | La maquette montre | Ce qui se fait | Décision |
|---|---|---|---|
| V1 | Cadres en segment de texte dans l'en-tête | Icônes à droite de la barre de lecture, dans les trois cadres | D-9, D-10 |
| V2 | Segment 2D / 3D dans l'en-tête, 3D grisée « Bientôt » | Superposition à gauche de « Vue », invisible tant que la 3D n'existe pas | D-9, F.3 |
| V3 | Bouton « Vue » en bout de barre de lecture | « Vue » reste en superposition en haut à droite de la toile | D-9 |
| V4 | Une seule jauge par joueur | Bouclier et santé en deux barres distinctes | D-3 |
| V5 | A : barre d'une ligne et pistes Lecture, joueur, Coéquipiers, Dominance, Médias | Standard : zone de lecture et zoom d'aujourd'hui, plus les icônes de cadre | D-8, D-18 |
| V6 | J et L : repères des autres joueurs estompés dans la frise | Frise inchangée | D-18 |
| V7 | A : titre du fil « Éliminations · 29 · Fil complet » | Titre = menu de filtre actuel | D-11 |
| V8 | Joueur suivi signalé par le seul gras dans le fil | Ligne qui concerne le joueur suivi : fond légèrement teinté (token), aucun liseré | D-5 |
| V9 | Fiches statiques | Tous les effets des fiches gardés (mort, réapparition, changement d'arme, équipement, « ÉLIMINÉ ») | D-17 |
| V10 | J et L : « À l'instant » réduit à l'état du joueur | Contenu enrichi du lot T, sur maquette validée avant T.5 | D-19 à D-24, plan maître DM-3 |
| V11 | J : en-tête avec cadre « Cinéma » coché, barre d'une ligne | En Standard : en-tête de A (A-2) et zone de lecture actuelle | D-6, D-8 |

## Tableau A — Standard

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| A-1 | Page | contenu borné à 1200 px, centré | S.1 | plan |
| A-2 | En-tête | 64 px ; trois cellules : retour (bouton 32 px) et titre sur deux lignes (petite ligne « joueur · file · date », puis « mode — carte ») ; score ; droite | S.1 | plan ; la petite ligne est à vérifier contre l'en-tête actuel (0.3) |
| A-3 | Score de l'en-tête | deux barres de 24 px, le nombre dans la barre, horloge au milieu ; cellule de 400 px dans A, 440 px dans J | S.1 | plan : 440 px retenus (J est postérieure), manches et prise de zone comprises |
| A-4 | Droite de l'en-tête | cadres, 2D / 3D, bouton « Fiche » | S.1 | V1, V2 ; « Fiche » = lien vers la fiche du match |
| A-5 | Corps | bloc arrondi (rayon 12 px, bordure) ; grille carte puis colonne de 360 px ; 600 px de haut à 1440 px ; carte centrée, 12 px de marge ; rien par-dessus hors « Vue », 2D / 3D et zoom | S.4 | plan |
| A-6 | Colonne | marge interne 12 px par 14 px, 14 px entre les blocs, filet à gauche | S.2, S.4 | plan |
| A-7 | En-tête d'équipe | pastille carrée de 8 px à la couleur du camp, nom du camp, 20 px de haut | S.2 | plan |
| A-8 | En-tête de colonnes | une seule fois, sous la première équipe : Arme, Gr., Éq., F / M / A ; 10,5 px, aligné à droite | S.2 | plan |
| A-9 | Ligne joueur | 34 px ; grille : nom et jauge, arme (22 px), grenades (24 px), équipement (14 px), F / M / A (54 px) ; nom 12,5 px ; jauge de 130 px sous le nom | S.2 | plan, V4 |
| A-10 | Joueur suivi | nom en gras, mention « suivi » à la couleur de suivi ; ligne dépliée NON dessinée dans A : dessin = bloc `.det` de H (deux colonnes : munitions de l'arme en main, arme en réserve grisée, « Fragmentation ×2 », charge d'équipement avec sa barre) | S.2 | plan, référence H |
| A-11 | Joueur mort | « Réapparition dans 5,8 s » à la place des jauges ; nom, icônes et F / M / A à 40 % | S.2 | plan, V9 |
| A-12 | Grenades, équipement | icône et nombre ; éteints quand il n'y en a pas | S.2 | plan |
| A-13 | Titre du fil | filet au-dessus ; « Éliminations », compte et « Fil complet » | S.3 | V7 |
| A-14 | Ligne du fil | 28 px ; grille 30, 86, 20, 86, reste : instant (gris, 11,5 px), tueur (couleur de son camp), icône d'arme, victime (couleur de son camp), lieu (gris, 11,5 px) ; ellipse au débordement ; plus récente en tête | S.3, C.1 | plan, V8 |
| A-15 | Zone de lecture | barre d'une ligne et pistes | A.3, E.2 | V5 |
| A-16 | Sous le lecteur | carte de chaleur, usage de l'équipement | S.4 | plan (inchangé) |

## Tableau K — Cinéma

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| K-1 | En-tête | 52 px : retour, titre sur deux lignes, à droite cadres et « Fiche » | A.3 | plan, V1, V2 |
| K-2 | Corps | lecteur sur la largeur du corps, 24 px de marge latérale, 8 px en bas | A.3 | plan |
| K-3 | Rangée | 58 px de haut ; 4 tuiles, score (270 px), 4 tuiles ; tuiles FLUIDES qui se partagent la largeur ; 6 px d'écart ; 8 px sous la rangée | B.1, B.2 | **ajout** : la rangée remplit la largeur ; la cote de 115 px du gabarit compact sert de minimum à la règle D-12, pas de largeur fixe |
| K-4 | Tuile | ligne 1 : pastille du camp, nom (ellipse), F / M / A à droite ; ligne 2 : jauges, grenades (icône et nombre), équipement (icône) ; pas d'arme | B.2 | **ajout** : contenu et ordre de la maquette ; écart avec la fiche compacte actuelle relevé par 0.3 et tranché à Gate 0 ; V4, V9 |
| K-5 | Tuile du joueur suivi | bordure à la couleur de suivi, nom en gras | B.2 | plan |
| K-6 | Tuile d'un mort | « Réapparition 5,8 s » à la place des jauges ; reste à 40 % | B.2 | plan, V9 |
| K-7 | Score au centre | barres et horloge, et dessous l'objectif du match (« Premier à 50 éliminations », 11 px, gris) | B.2 | **ajout** : ligne d'objectif ; 0.3 dit si le document de rejeu la porte ; sinon retour au user |
| K-8 | Plan | bloc arrondi qui prend la hauteur restante, carte centrée | A.3, G.1 | plan |
| K-9 | Fil sur le plan | bas gauche du bloc (16 px, 12 px), 350 px de large, 5 lignes, plus récente en bas, ni en-tête ni fond, ombre portée du texte et des icônes ; colonnes de A-14 ; les deux lignes les plus anciennes estompées (70 %, puis 85 %) | C.2 | plan ; **ajout** : estompage des deux plus anciennes |
| K-10 | Barre d'une ligne | 56 px : caméra, son ; lecture (rond plein de 36 px), −10, +10, temps « 3:51 / 7:45 », frise (repères au-dessus du rail, rail, bande de dominance dessous, curseur), vitesse ; zoom − et + ; capture, export, plein écran système ; séparateurs verticaux entre groupes | E.3 | plan, V1 (icônes de cadre à droite), V3 |
| K-11 | Note « Clic sur une fiche… » | annotation de maquette | aucun | hors interface |

## Tableau D — Plein cadre

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| D-1 | En-tête | aucune navigation de l'app ; 52 px : titre sur deux lignes ; cadres avec les touches T et F en clair | A.3 | plan (« Quitter / Échap, titre ») ; **ajout** : touches T et F rappelées dans les infobulles des icônes de cadre |
| D-2 | Reste | rangée, plan, fil et barre de K ; marges de 16 px | B.2, C.2, E.3 | plan |

## Tableau L — fiche en fenêtre (Cinéma, Plein cadre)

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| L-1 | Fenêtre | 330 px de large, accrochée sous la tuile cliquée, hauteur bornée par le plan, défilement interne, fond surélevé, bordure, rayon 12 px, ombre ; non bloquante : plan et fil restent visibles | D.3 | plan |
| L-2 | Identité | pastille, nom (16 px, gras), camp, F / M / A, capsule « Échap », croix | D.3 | plan |
| L-3 | Actions | Suivre, Seul, Masquer en segment de trois colonnes égales | D.1 | plan |
| L-4 | À l'instant | titre suivi de l'instant lu (« 3:51 ») ; Bouclier et Santé (barre de 5 px et valeur) ; arme en main NOMMÉE avec ses munitions ; arme en réserve grisée ; « Fragmentation ×2 » ; « Équipement · 1 charge » et sa barre | D.1, T.5 | plan ; l'instant à côté du titre est à vérifier contre le panneau actuel (0.3) |
| L-5 | Totaux du match | Précision « 41 % · 118 / 288 », dégâts infligés et subis, score ; barres par arme (nom, barre de 90 px, nombre) | D.1 | plan (inchangé) |
| L-6 | Joueur ouvert | tuile au fond plus clair et bordure claire ; pion cerclé clair | D.3 | plan |
| L-7 | Sélecteur | aucun : la rangée en tient lieu | D.3 | plan |

## Tableau J — fiche du Standard

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| J-1 | Colonne | la colonne de 360 px devient la fiche ; contenu défilant ; « Éliminations » fixé en bas, hors défilement | D.2 | plan |
| J-2 | Retour | « Équipes » précédé d'un chevron, capsule « Échap » à droite | D.2 | plan |
| J-3 | Sélecteur de joueurs | 8 rectangles arrondis de 34 px par 22 px, deux initiales, couleur du camp, espace entre les camps, mort estompé, sélection par double anneau | D.2 | plan |
| J-4 | Identité | pastille, nom (17 px, gras), camp, F / M / A (15 px) | D.2 | plan |
| J-5 | Actions | Suivre, Seul, Masquer | D.1 | plan |
| J-6 | Sections | « À l'instant » et « Totaux du match » comme L-4 et L-5, avec « Éliminations à la tête » | D.1, T.5 | plan, V10 |
| J-7 | Éliminations | titre et lien « Fil complet » à droite ; 2 lignes au format A-14 ; le joueur ouvert en gras | D.2 | plan |
| J-8 | Frise | repères des autres estompés | aucun | V6 |

## Tableau H — rotation

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| H-1 | Rotation | seul le plan tourne d'un quart de tour ; la mise en page ne change pas | F.1, F.2 | plan |
| H-2 | Colonne en Cinéma | carte puis colonne de 360 px (équipes en lignes, fil), score dans l'en-tête | B.2, S.2 | plan : dessin du repli de D-12 |
| H-3 | Bouton « Plan tourné » | non dessiné | F.2 | sans dessin |

## Sans dessin (liste fermée)

Aucune maquette ne montre ces éléments. Leur forme reprend le vocabulaire des tableaux qui font foi (hauteurs
de 28, 30 et 34 px, capsules, segments, boutons d'icône de la barre) et les composants de l'app. L'audit 0.3
propose une forme pour chacun ; quand il y a un vrai choix de forme, il est montré au user à Gate 0.

1. Icônes de cadre (Cinéma, Plein cadre, plein écran système) dans la barre de lecture.
2. Bouton « Plan tourné » de la barre compacte.
3. « Quitter » du Plein cadre.
4. Menu caméra à deux sections (« Suivre », « Point de vue »).
5. Option « Fil des éliminations » et position « Auto » de l'orientation dans le panneau « Vue ».
6. Match sans deux camps (chacun pour soi) : colonne sans score.
7. Grande équipe : part des équipes qui défile, fil gardé à 5 lignes au moins.
8. Places libres et joueurs pas encore apparus, en ligne et en tuile.
9. Lignes du fil qui ne sont pas des éliminations (objectifs, entrées et sorties, prises), en colonne et sur le plan.
10. Contenu du lot T dans la fiche (répartition des éliminations, séries, estimations, « En passant ») :
    maquette à valider avant T.5.

## Usage

- Brief d'un agent d'interface : le ou les fichiers de maquette de son tableau, les lignes de cette grille
  qu'il réalise, les écarts voulus qui le concernent. Son compte rendu statue chaque ligne avec sa preuve (nom
  du test, fixation DOM).
- Clôture d'un item : le pilote relit la fixation DOM contre les lignes de la grille avant de cocher.
- R.1 : la conformité à cette grille est un axe de la revue finale.
- R.4 : le user valide à l'écran avec le canevas à côté ; chaque témoin renvoie aux lignes de la grille.
