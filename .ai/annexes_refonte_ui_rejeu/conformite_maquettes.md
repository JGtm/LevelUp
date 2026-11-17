# Grille de conformité aux maquettes — refonte de la page du rejeu (2026-11-17)

Relevé fait par le pilote le 2026-11-17 sur les copies de `.ai/annexes_refonte_ui_rejeu/maquettes/`
(canevas https://claude.ai/artifact/GkqDthVG6jGxx3A5WLfXrr, version 26). Sert de référence aux briefs des
agents, à la relecture sur pièces de chaque item d'interface, à la revue finale R.1 et à la validation R.4 du
plan `.ai/PLAN_REFONTE_UI_REJEU_2026-11-15.md`. L'audit 0.3 la complète et la corrige contre le code ; toute
ligne ajoutée ou modifiée porte sa date.

Passe de l'audit 0.3 faite le 2026-11-17 contre le code de `release/v8.3.0` (`3ae75c78f5`) : les ajouts sont
marqués « (audit 0.3, 2026-11-17) ». Les constats, avec fichier et ligne, et les questions au user sont dans
`.ai/annexes_refonte_ui_rejeu/audit_conception.md`. Aucune ligne retirée, aucun écart voulu ajouté.
(audit 0.3, 2026-11-17)

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
| A-1 | Page | contenu borné à 1200 px, centré | S.1 | plan ; conséquence relevée : sur un écran de 1920 px la carte du Standard passe d'environ 1380 px à 840 px de large (audit 0.3, 2026-11-17) |
| A-2 | En-tête | 64 px ; trois cellules : retour (bouton 32 px) et titre sur deux lignes (petite ligne « joueur · file · date », puis « mode — carte ») ; score ; droite | S.1 | plan ; la petite ligne est à vérifier contre l'en-tête actuel (0.3) ; vérifié : les cinq données sont affichées aujourd'hui, mais sur UNE ligne de fil d'Ariane partagée avec la page du match (joueur écrit par son identifiant d'adresse, mode et carte, puis date et file en badge) ; la forme sur deux lignes est à construire dans un en-tête propre au rejeu ; A dessine l'en-tête à 64 px, J et H à 56 px : une seule cote à retenir (audit 0.3, 2026-11-17) |
| A-3 | Score de l'en-tête | deux barres de 24 px, le nombre dans la barre, horloge au milieu ; cellule de 400 px dans A, 440 px dans J | S.1 | plan : 440 px retenus (J est postérieure), manches et prise de zone comprises ; le bandeau actuel a des barres de 28 px, une marge basse, les pastilles de manche au-dessus et la garde de colline dessous : il lui faut des cotes nommées par emplacement (audit 0.3, 2026-11-17) |
| A-4 | Droite de l'en-tête | cadres, 2D / 3D, bouton « Fiche » | S.1 | V1, V2 ; « Fiche » = lien vers la fiche du match |
| A-5 | Corps | bloc arrondi (rayon 12 px, bordure) ; grille carte puis colonne de 360 px ; 600 px de haut à 1440 px ; carte centrée, 12 px de marge ; rien par-dessus hors « Vue », 2D / 3D et zoom | S.4 | plan ; aujourd'hui le bloc bordé entoure la toile ET la barre de lecture ; en Standard la zone de lecture reste celle d'aujourd'hui (V5), la barre hors du bloc ne concerne donc que le Cinéma et le Plein cadre (K-8) (audit 0.3, 2026-11-17) |
| A-6 | Colonne | marge interne 12 px par 14 px, 14 px entre les blocs, filet à gauche | S.2, S.4 | plan |
| A-7 | En-tête d'équipe | pastille carrée de 8 px à la couleur du camp, nom du camp, 20 px de haut | S.2 | plan |
| A-8 | En-tête de colonnes | une seule fois, sous la première équipe : Arme, Gr., Éq., F / M / A ; 10,5 px, aligné à droite | S.2 | plan ; hauteur relevée : 16 px (audit 0.3, 2026-11-17) |
| A-9 | Ligne joueur | 34 px ; grille : nom et jauge, arme (22 px), grenades (24 px), équipement (14 px), F / M / A (54 px) ; nom 12,5 px ; jauge de 130 px sous le nom | S.2 | plan, V4 ; H dessine la même jauge à 110 px ; écart de colonnes : 6 px (audit 0.3, 2026-11-17) |
| A-10 | Joueur suivi | nom en gras, mention « suivi » à la couleur de suivi ; ligne dépliée NON dessinée dans A : dessin = bloc `.det` de H (deux colonnes : munitions de l'arme en main, arme en réserve grisée, « Fragmentation ×2 », charge d'équipement avec sa barre) | S.2 | plan, référence H ; aucun état « suivi » n'existe aujourd'hui hors fiche ouverte, et la caméra ne suit personne à l'ouverture : sens de « suivi » à trancher par le user (question 2 de l'audit) (audit 0.3, 2026-11-17) |
| A-11 | Joueur mort | « Réapparition dans 5,8 s » à la place des jauges ; nom, icônes et F / M / A à 40 % | S.2 | plan, V9 ; aujourd'hui : bloc hachuré « ÉLIMINÉ » et compte à rebours en secondes (audit 0.3, 2026-11-17) |
| A-12 | Grenades, équipement | icône et nombre ; éteints quand il n'y en a pas | S.2 | plan |
| A-13 | Titre du fil | filet au-dessus ; « Éliminations », compte et « Fil complet » | S.3 | V7 |
| A-14 | Ligne du fil | 28 px ; grille 30, 86, 20, 86, reste : instant (gris, 11,5 px), tueur (couleur de son camp), icône d'arme, victime (couleur de son camp), lieu (gris, 11,5 px) ; ellipse au débordement ; plus récente en tête | S.3, C.1 | plan, V8 ; la ligne actuelle : horloge (42 px), lieu du TUEUR juste après (5,5 rem), tueur, arme, victime, badge « porteur », médailles, assistance ; l'ordre et les largeurs changent donc, pas seulement le lieu ; médailles, badge et assistance ne sont pas dans la maquette (question 4 de l'audit) ; la ligne reste cliquable et la ligne choisie garde sa marque (audit 0.3, 2026-11-17) |
| A-15 | Zone de lecture | barre d'une ligne et pistes | A.3, E.2 | V5 |
| A-16 | Sous le lecteur | carte de chaleur, usage de l'équipement | S.4 | plan (inchangé) ; **faux contre le code** : la page n'a rien sous le lecteur aujourd'hui (la carte de chaleur est un réglage du panneau « Vue », l'usage de l'équipement est sur la page du match) ; question 3 de l'audit (audit 0.3, 2026-11-17) |

## Tableau K — Cinéma

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| K-1 | En-tête | 52 px : retour, titre sur deux lignes, à droite cadres et « Fiche » | A.3 | plan, V1, V2 ; même constat que A-2 : en-tête à construire, l'actuel est un fil d'Ariane d'une ligne (audit 0.3, 2026-11-17) |
| K-2 | Corps | lecteur sur la largeur du corps, 24 px de marge latérale, 8 px en bas | A.3 | plan |
| K-3 | Rangée | 58 px de haut ; 4 tuiles, score (270 px), 4 tuiles ; tuiles FLUIDES qui se partagent la largeur ; 6 px d'écart ; 8 px sous la rangée | B.1, B.2 | **ajout** : la rangée remplit la largeur ; la cote de 115 px du gabarit compact sert de minimum à la règle D-12, pas de largeur fixe ; vérifié : la fiche compacte est déjà fluide (115 px au minimum) mais cette cote n'est PAS dans le gabarit, c'est un littéral de classe : B.1 l'y ajoute, avec l'écart et la largeur du score ; à 115 px un 4v4 demande 1286 px de corps, un 5v5 1528 px (audit 0.3, 2026-11-17) |
| K-4 | Tuile | ligne 1 : pastille du camp, nom (ellipse), F / M / A à droite ; ligne 2 : jauges, grenades (icône et nombre), équipement (icône) ; pas d'arme | B.2 | **ajout** : contenu et ordre de la maquette ; écart avec la fiche compacte actuelle relevé par 0.3 et tranché à Gate 0 ; V4, V9 ; relevé : la fiche compacte fait 62 px et trois lignes (nom seul en capitales grasses ; jauges puis F / M / A ; arme en main, grenade sélectionnée sans nombre, équipement), sans pastille de camp ; la tuile de K fait 58 px et deux lignes ; cotes de K : marge interne 7 px par 9 px, rayon 8 px, ligne 2 de 13 px ; deux effets de V9 n'ont plus de support dans la tuile de K (glissement au changement d'arme, mot « ÉLIMINÉ ») : question 1 de l'audit, détail en 4.1 (audit 0.3, 2026-11-17) |
| K-5 | Tuile du joueur suivi | bordure à la couleur de suivi, nom en gras | B.2 | plan ; aucun état « suivi » aujourd'hui : voir A-10 (audit 0.3, 2026-11-17) |
| K-6 | Tuile d'un mort | « Réapparition 5,8 s » à la place des jauges ; reste à 40 % | B.2 | plan, V9 ; proposition : la bande hachurée actuelle reste sur la ligne 2 et porte « Réapparition 5,8 s » (audit 0.3, 2026-11-17) |
| K-7 | Score au centre | barres et horloge, et dessous l'objectif du match (« Premier à 50 éliminations », 11 px, gris) | B.2 | **ajout** : ligne d'objectif ; 0.3 dit si le document de rejeu la porte ; sinon retour au user ; réponse : le document porte le NOMBRE (`scoreTimeline.targetScore`), absent quand la variante n'est pas mesurée, et qui vaut le plafond d'une manche sur un mode à manches ; il ne porte PAS l'unité (« éliminations »), l'en-tête du match non plus ; question 6 de l'audit ; avec les pastilles de manche et la garde de colline du bandeau actuel, score et objectif ne tiennent pas dans 58 px : proposition, l'objectif cède quand le mode a des manches (audit 0.3, 2026-11-17) |
| K-8 | Plan | bloc arrondi qui prend la hauteur restante, carte centrée | A.3, G.1 | plan ; aujourd'hui la hauteur de la toile s'arrête à la hauteur utile de la carte et la barre est dans le bloc bordé : remplir la hauteur et sortir la barre du bloc touche le cadrage et la mesure de hauteur, à porter dans E.1 (audit 0.3, 2026-11-17) |
| K-9 | Fil sur le plan | bas gauche du bloc (16 px, 12 px), 350 px de large, 5 lignes, plus récente en bas, ni en-tête ni fond, ombre portée du texte et des icônes ; colonnes de A-14 ; les deux lignes les plus anciennes estompées (70 %, puis 85 %) | C.2 | plan ; **ajout** : estompage des deux plus anciennes ; K ne montre que des éliminations, alors que le plan y applique le filtre du Standard (« Tout » par défaut, sans menu en Cinéma) : question 5 de l'audit ; le fil ne capte pas le pointeur et passe sous l'écran de fin (audit 0.3, 2026-11-17) |
| K-10 | Barre d'une ligne | 56 px : caméra, son ; lecture (rond plein de 36 px), −10, +10, temps « 3:51 / 7:45 », frise (repères au-dessus du rail, rail, bande de dominance dessous, curseur), vitesse ; zoom − et + ; capture, export, plein écran système ; séparateurs verticaux entre groupes | E.3 | plan, V1 (icônes de cadre à droite), V3 ; K n'a ni lecture automatique, ni « Recommencer », ni repères de médias, ni piste de score, que la barre du Standard porte : question 7 de l'audit ; les repères de K sont les frags de TOUS les joueurs à la couleur du camp ; le temps « 3:51 / 7:45 » est déjà écrit sous cette forme par l'horloge actuelle (audit 0.3, 2026-11-17) |
| K-11 | Note « Clic sur une fiche… » | annotation de maquette | aucun | hors interface |

## Tableau D — Plein cadre

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| D-1 | En-tête | aucune navigation de l'app ; 52 px : titre sur deux lignes ; cadres avec les touches T et F en clair | A.3 | plan (« Quitter / Échap, titre ») ; **ajout** : touches T et F rappelées dans les infobulles des icônes de cadre ; « Quitter » EST dessiné dans D : bouton bordé de 32 px, croix, « Quitter », capsule « Échap », à gauche, à la place du retour ; D n'a pas de bouton « Fiche » (audit 0.3, 2026-11-17) |
| D-2 | Reste | rangée, plan, fil et barre de K ; marges de 16 px | B.2, C.2, E.3 | plan ; marge basse relevée : 6 px (audit 0.3, 2026-11-17) |

## Tableau L — fiche en fenêtre (Cinéma, Plein cadre)

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| L-1 | Fenêtre | 330 px de large, accrochée sous la tuile cliquée, hauteur bornée par le plan, défilement interne, fond surélevé, bordure, rayon 12 px, ombre ; non bloquante : plan et fil restent visibles | D.3 | plan ; précision : bord GAUCHE aligné sur celui de la tuile, 8 px sous le haut du plan, hauteur au plus celle du plan moins 16 px ; le panneau ancré actuel aligne le bord droit et borne à la fenêtre : deux options à lui ajouter (audit 0.3, 2026-11-17) |
| L-2 | Identité | pastille, nom (16 px, gras), camp, F / M / A, capsule « Échap », croix | D.3 | plan |
| L-3 | Actions | Suivre, Seul, Masquer en segment de trois colonnes égales | D.1 | plan ; aujourd'hui trois boutons séparés à largeur égale (audit 0.3, 2026-11-17) |
| L-4 | À l'instant | titre suivi de l'instant lu (« 3:51 ») ; Bouclier et Santé (barre de 5 px et valeur) ; arme en main NOMMÉE avec ses munitions ; arme en réserve grisée ; « Fragmentation ×2 » ; « Équipement · 1 charge » et sa barre | D.1, T.5 | plan ; l'instant à côté du titre est à vérifier contre le panneau actuel (0.3) ; vérifié : l'instant lu est écrit aujourd'hui, mais dans la tête de la fiche après les compteurs ; le bloc « À l'instant » lui-même N'EXISTE PAS (ni titre, ni bouclier et santé chiffrés, ni arme nommée, ni grenades, ni équipement) : il est à construire, ce n'est pas une extraction (audit 0.3, 2026-11-17) |
| L-5 | Totaux du match | Précision « 41 % · 118 / 288 », dégâts infligés et subis, score ; barres par arme (nom, barre de 90 px, nombre) | D.1 | plan (inchangé) ; **pas inchangé** : aujourd'hui cinq listes titrées (précision, dégâts, détail du score, objectif, éliminations par arme avec icône et « ×N ») ; L montre une seule section et des barres par arme ; le contenu reste (D-23), la forme est à dessiner dans la maquette 0.4 (audit 0.3, 2026-11-17) |
| L-6 | Joueur ouvert | tuile au fond plus clair et bordure claire ; pion cerclé clair | D.3 | plan ; aujourd'hui la fiche ouverte porte un anneau ; le pion cerclé est un dessin nouveau dans la scène de la carte (audit 0.3, 2026-11-17) |
| L-7 | Sélecteur | aucun : la rangée en tient lieu | D.3 | plan |

## Tableau J — fiche du Standard

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| J-1 | Colonne | la colonne de 360 px devient la fiche ; contenu défilant ; « Éliminations » fixé en bas, hors défilement | D.2 | plan ; retour, sélecteur, identité et actions sont DANS la zone qui défile ; aujourd'hui la fiche ne remplace que la cellule du fil, les équipes restent au-dessus (audit 0.3, 2026-11-17) |
| J-2 | Retour | « Équipes » précédé d'un chevron, capsule « Échap » à droite | D.2 | plan |
| J-3 | Sélecteur de joueurs | 8 rectangles arrondis de 34 px par 22 px, deux initiales, couleur du camp, espace entre les camps, mort estompé, sélection par double anneau | D.2 | plan |
| J-4 | Identité | pastille, nom (17 px, gras), camp, F / M / A (15 px) | D.2 | plan |
| J-5 | Actions | Suivre, Seul, Masquer | D.1 | plan ; « Suivre » actif se lit par le segment plein (audit 0.3, 2026-11-17) |
| J-6 | Sections | « À l'instant » et « Totaux du match » comme L-4 et L-5, avec « Éliminations à la tête » | D.1, T.5 | plan, V10 ; différence avec L-4 : J n'écrit pas le nom de l'arme (icône et munitions seulement), L l'écrit ; retenir L (postérieure) (audit 0.3, 2026-11-17) |
| J-7 | Éliminations | titre et lien « Fil complet » à droite ; 2 lignes au format A-14 ; le joueur ouvert en gras | D.2 | plan ; plus récente en tête (audit 0.3, 2026-11-17) |
| J-8 | Frise | repères des autres estompés | aucun | V6 |

## Tableau H — rotation

| Réf | Élément | Relevé | Item | Statut |
|---|---|---|---|---|
| H-1 | Rotation | seul le plan tourne d'un quart de tour ; la mise en page ne change pas | F.1, F.2 | plan |
| H-2 | Colonne en Cinéma | carte puis colonne de 360 px (équipes en lignes, fil), score dans l'en-tête | B.2, S.2 | plan : dessin du repli de D-12 ; en-tête de 56 px, cellule de score de 440 px (audit 0.3, 2026-11-17) |
| H-3 | Bouton « Plan tourné » | non dessiné | F.2 | sans dessin ; **corrigé : il EST dessiné dans H**, dans la barre, entre la vitesse et le zoom : capsule de 30 px (marge 10 px, rayon 8 px), icône de rotation et libellé « Plan tourné », bordure et texte à la couleur de suivi, infobulle « Plan tourné automatiquement : +42 % de taille. Cliquer pour revenir à l'orientation d'origine. » (audit 0.3, 2026-11-17) |

## Sans dessin (liste fermée)

Aucune maquette ne montre ces éléments. Leur forme reprend le vocabulaire des tableaux qui font foi (hauteurs
de 28, 30 et 34 px, capsules, segments, boutons d'icône de la barre) et les composants de l'app. L'audit 0.3
propose une forme pour chacun ; quand il y a un vrai choix de forme, il est montré au user à Gate 0.

1. Icônes de cadre (Cinéma, Plein cadre, plein écran système) dans la barre de lecture.
   Forme proposée : trois boutons ronds fantômes au format du bouton plein écran actuel de la barre, 30 px
   dans la barre compacte ; cadre actif sur fond d'accent, état porté par `aria-pressed` ; infobulles
   « Cinéma (T) », « Plein cadre (F) », « Plein écran du système ». Pas de choix à montrer.
   (audit 0.3, 2026-11-17)
2. Bouton « Plan tourné » de la barre compacte.
   Dessiné dans H (voir H-3) : la ligne reste pour mémoire, la forme est celle de la maquette.
   (audit 0.3, 2026-11-17)
3. « Quitter » du Plein cadre.
   Dessiné dans D (voir D-1) : la ligne reste pour mémoire, la forme est celle de la maquette (bouton bordé
   de l'app, capsule « Échap »). (audit 0.3, 2026-11-17)
4. Menu caméra à deux sections (« Suivre », « Point de vue »).
   Forme proposée : petit panneau au-dessus de l'icône, sur le patron du menu de vitesse, avec deux listes
   déroulantes titrées : « Suivre » (dont « Aucun ») et « Point de vue » (la liste actuelle de la frise).
   **Choix à montrer au user** (question 9 de l'audit). (audit 0.3, 2026-11-17)
5. Option « Fil des éliminations » et position « Auto » de l'orientation dans le panneau « Vue ».
   Forme proposée : l'interrupteur des réglages du panneau (`SettingsToggle`) ; « Auto » en première position
   du segmenté d'orientation existant. Pas de choix à montrer. (audit 0.3, 2026-11-17)
6. Match sans deux camps (chacun pour soi) : colonne sans score.
   Forme proposée : lignes dans l'ordre des places, sans en-tête d'équipe ni pastille de camp ; cellule de
   score de l'en-tête vide. Pas de choix à montrer. (audit 0.3, 2026-11-17)
7. Grande équipe : part des équipes qui défile, fil gardé à 5 lignes au moins.
   Forme proposée : lignes de 34 px dans une seule zone qui défile, en-têtes d'équipe collés en haut ;
   environ 11 joueurs visibles sur 24, contre 24 aujourd'hui. **Choix à montrer au user** (question 8 de
   l'audit). (audit 0.3, 2026-11-17)
8. Places libres et joueurs pas encore apparus, en ligne et en tuile.
   Forme proposée : l'habit tireté des tuiles d'attente actuelles, le nom estompé, « Place libre » ou « Pas
   encore apparu » à la place des jauges. Pas de choix à montrer. (audit 0.3, 2026-11-17)
9. Lignes du fil qui ne sont pas des éliminations (objectifs, entrées et sorties, prises), en colonne et sur le plan.
   Forme proposée en colonne : la forme actuelle sur une ligne, l'horloge sur la colonne de 30 px, le
   pictogramme sur la colonne de l'arme, le texte sur le reste. Sur le plan : **choix à montrer au user**
   (question 5 de l'audit : éliminations seules, ou fil filtré). (audit 0.3, 2026-11-17)
10. Contenu du lot T dans la fiche (répartition des éliminations, séries, estimations, « En passant ») :
    maquette à valider avant T.5.
    À dessiner aussi dans cette maquette : la forme des « Totaux du match » (L-5) et le mot retenu entre
    « arme lourde » (D-20) et « armes de puissance » (libellé actuel). (audit 0.3, 2026-11-17)
11. Titre de page « Rejeu » et badge d'administration de la version du rejeu, aujourd'hui dans le fil
    d'Ariane. Forme proposée : « mode — carte » devient le titre de page ; le badge se pose à gauche du
    bouton « Fiche » (du titre en Plein cadre), pour l'administrateur seulement. Pas de choix à montrer.
    (audit 0.3, 2026-11-17)
12. Bandeau de régénération des rejeux, avis « instant non calé », états de chargement, vide et erreur.
    Forme proposée : les composants actuels, sous l'en-tête et au-dessus du corps, dans les trois cadres.
    Pas de choix à montrer. (audit 0.3, 2026-11-17)
13. Pastilles de manche et garde de colline du score, dans un en-tête de 56 à 64 px et une rangée de 58 px.
    Forme proposée : pastilles au-dessus des barres, garde dessous, barres de 24 px ; dans la rangée, la
    ligne d'objectif cède la place aux pastilles. Pas de choix à montrer. (audit 0.3, 2026-11-17)
14. Médailles, badge « porteur » et assistance d'une ligne de frag, présents aujourd'hui, absents de A-14.
    **Choix à montrer au user** (question 4 de l'audit). (audit 0.3, 2026-11-17)
15. Ligne de frag choisie (clic sur une ligne : saut à l'instant et mise en valeur sur la carte).
    Forme proposée : la marque actuelle (fond de jeton et deux filets aux couleurs des deux camps), sur la
    ligne à colonnes fixes ; aucune interaction sur le fil posé sur le plan. Pas de choix à montrer.
    (audit 0.3, 2026-11-17)
16. Fenêtre de moins de 1280 px de large. Forme proposée : comme aujourd'hui, la colonne passe sous la
    carte ; en Cinéma, la règle D-12 donne déjà la colonne. Pas de choix à montrer.
    (audit 0.3, 2026-11-17)

## Usage

- Brief d'un agent d'interface : le ou les fichiers de maquette de son tableau, les lignes de cette grille
  qu'il réalise, les écarts voulus qui le concernent. Son compte rendu statue chaque ligne avec sa preuve (nom
  du test, fixation DOM).
- Clôture d'un item : le pilote relit la fixation DOM contre les lignes de la grille avant de cocher.
- R.1 : la conformité à cette grille est un axe de la revue finale.
- R.4 : le user valide à l'écran avec le canevas à côté ; chaque témoin renvoie aux lignes de la grille.

## Décisions du user à la Gate 0 (2026-11-17)

Réponses à des questions fermées, après l'audit 0.3. Elles prévalent sur les lignes ci-dessus.

| Réf | La maquette montre | Ce qui se fait | Décision |
|---|---|---|---|
| V12 | K-3, K-4 : tuile à deux lignes, sans arme | Fiche compacte actuelle à trois lignes, arme comprise, tous les effets gardés ; de K restent la rangée qui remplit la largeur, la bordure de suivi et l'état « ouverte » | « Garder l'arme dans la fiche » |
| V13 | K-10 : barre d'une ligne sans lecture automatique, sans « Recommencer », sans repères de médias | Les trois entrent dans la barre d'une ligne | « Tout reprendre » |
| V14 | K-7 : objectif du match sous le score | Rien sous le score | « Rien sous le score » |
| V15 | A-14 : ligne du fil à cinq colonnes | Colonne du Standard : médailles entre la victime et le lieu, assistance et badge « porteur » en infobulle ; sur le plan : ligne nue | « Médailles gardées » |
| V16 | Lignes de 34 px pour tous les effectifs | Grande équipe : lignes resserrées (nom, jauges, F / M / A), sur maquette à valider | « Lignes resserrées » |

- « Suivi » (A-10, K-5) désigne le joueur regardé, celui de la page au départ ; « Suivre » est l'action de caméra, à part.
- Fil posé sur le plan (K-9) : éliminations seules.
- A-16 : rien sous le lecteur (la page n'a aucune section à cet endroit).

| Réf | La maquette montre | Ce qui se fait | Décision |
|---|---|---|---|
| V17 | A-14, K-9, J-7 : instant, tueur, arme, victime, lieu | Instant, lieu, tueur, arme, victime, partout (colonne, plan, pied de fiche) ; le lieu reste celui de la victime | user, 2026-11-17 : « chrono - zone de callout - tueur - arme - victime » |

## Tableau M — fiche enrichie (validé par le user le 2026-11-17)

Fichier `FicheInstant.dc.html`. Fait foi pour le contenu de la fiche, en colonne (J) et en fenêtre (L).

| Réf | Élément | Relevé | Item |
|---|---|---|---|
| M-1 | État à l'instant | titre et instant lu ; bouclier et santé (barre et valeur) ; arme en main nommée avec munitions ; réserve grisée ; grenades ; équipement porté avec sa charge | D.0 |
| M-2 | Depuis le début du match | lignes libellé à gauche, valeur à droite : FDA, série en cours, meilleure série, tirs à la tête, éliminations parfaites, réapparitions, vie en cours, durée moyenne d'une vie, boucliers perdus, score et rang (« 800 · 2e sur 8 ») | T.1, T.3 |
| M-3 | Éliminations par arme | une barre empilée par classe d'arme, aux couleurs des classes de l'application (jetons `frag-*`), puis un tableau, pastille de la classe devant son nom : catégorie en gras puis ses armes en retrait ; colonnes « À l'instant », « Part », « Match » ; catégorie vide absente ; catégories : autres armes, armes lourdes, grenade, mêlée | T.2 |
| M-4 | Tirs et dégâts | capsule « estimation » à côté du titre ; précision avec tirs touchés sur tirés ; dégâts infligés et subis ; par arme : tirs et dégâts ; section absente pour un joueur à lecture partielle | T.3 |
| M-5 | Grenades lancées | une ligne par type | T.4 |
| M-6 | Ramassages | armes, grenades | T.4 |
| M-7 | Équipement | tableau par famille : pris, utilisé, perdu | T.8 |
| M-8 | Médailles | comme sur les tuiles de match : icône de 32 px, nom dessous sur une ligne tronquée, « ×n » ; rangée centrée | T.6 |
| M-9 | Totaux du match | séparé par un filet ; précision, dégâts, score, éliminations à la tête ; pas d'armes | D.1 |
| M-10 | Gestes | séparé par un filet ; titre et instant lu ; sprint (temps cumulé) ; sauts avec la capsule « estimation » | T.7 |
| M-11 | Lecture partielle | ni score, ni tirs et dégâts ; aucun texte de remplacement | T.3 |
| M-12 | Pied de la colonne | « Éliminations » et « Fil complet » ; 2 lignes dans l'ordre instant, lieu, tueur, arme, victime | D.2 |
