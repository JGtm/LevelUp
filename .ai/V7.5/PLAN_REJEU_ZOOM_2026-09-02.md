# PLAN — Le zoom du rejeu 2D

> Branche `wt/rejeu-zoom` depuis `feat/v75`. Demande utilisateur du 2026-09-02, apres la
> hauteur elastique : « pour le zoom tu peux peut-etre soit laisser le deplacement a la souris
> soit mettre un genre de croix directionnelle ? », et « le zoom peut etre un element en
> surimpression dans un angle au niveau de la map ».

## L'idee qui rend le lot faisable : LE ZOOM EST UN CHANGEMENT DE BORNES

La tentation est d'ajouter `{echelle, panX, panY}` a `canvasView` et de les appliquer partout.
Ce serait toucher `worldToCanvas`, `canvasScale`, le survol, les quatre calques, les infobulles
— et introduire deux facons de dire ou tombe un point.

Or `canvasView` porte deja des BORNES, et la projection est entierement definie par elles :
`worldToCanvas` mappe `bounds` vers la toile. **Zoomer, c'est retrecir les bornes ; se deplacer,
c'est les translater.** Rien d'autre ne change :

- `worldToCanvas` — inchange ;
- `canvasScale` — inchange ;
- le SURVOL — inchange, il lit la meme projection (l'invariant du depot tient tout seul) ;
- les infobulles, le fond de carte (`backgroundRect` prend `bounds`) — inchanges ;
- les CALQUES STATIQUES — ils cuisent depuis `view`, donc ils cuisent la fenetre visible a la
  resolution de l'ecran : **la memoire ne bouge pas avec le zoom**. C'est exactement ce que la
  crainte de l'utilisateur visait ; la formulation par les bornes l'evite par construction.

## Decisions

- **D1 — Le zoom retrecit les bornes, il ne multiplie pas une echelle.** Cf. ci-dessus.
- **D2 — Le rapport d'aspect de la scene est PRESERVE** : les deux dimensions sont divisees par
  le meme facteur. Sans cela, `usefulHeight` (plafond de hauteur par carte) et le cadrage
  divergeraient a chaque cran.
- **D3 — La fenetre visible reste DANS la scene.** On ne peut pas se deplacer hors carte : le
  centre est borne pour que la fenetre ne sorte jamais. A zoom 1 elle vaut la scene entiere,
  donc le centre n'a qu'une position possible — la croix se desactive d'elle-meme.
- **D4 — Niveaux DISCRETS (1x, 1,5x, 2x, 3x), pas un zoom continu.** Un zoom continu recuirait
  les calques a chaque cran de molette ; des paliers rendent chaque changement rare et
  previsible. C'est aussi ce qui permet de nommer l'etat a l'ecran.
- **D5 — LA CROIX DIRECTIONNELLE, PAS LE GLISSER.** La demande disait « soit le deplacement a la
  souris SOIT une croix directionnelle » : c'est un choix, et je prends la croix. Deux raisons.
  D'abord l'horloge tourne : glisser pendant que l'action bouge, c'est poursuivre le jeu a la
  souris. Ensuite et surtout, un glisser change le cadrage a chaque mouvement de pointeur, donc
  RECUIT les quatre calques a chaque image — il faudrait un blit decale et une cuisson a marge
  pour l'eviter. La croix va par pas DISCRETS : une cuisson par clic, comme un redimensionnement
  de fenetre. Le glisser reste ouvert si l'usage le reclame, avec son cout instruit.
- **D6 — RIEN A FAIRE SUR LES CALQUES, et c'est la consequence de D1.** Ils cuisent depuis le
  cadrage : avec des bornes retrecies, ils cuisent la FENETRE a la resolution de l'ecran. Leur
  surface ne depend donc PAS du niveau de zoom — la crainte d'une memoire qui enfle avec le
  grossissement est evitee par construction. Et D5 rend les recuissons rares : une par clic.
- **D7 — REVISEE : L'EXPORT SUIT LA VUE.** J'avais ecrit l'inverse (« l'export force le zoom a
  1 »), par analogie avec le facteur de surechantillonnage. L'analogie ne tient pas. Ce facteur
  varie SANS QUE PERSONNE LE VOIE — d'ou la necessite de le neutraliser. Le zoom, lui, est un
  geste deliberé et affiche : « ce que je vois est ce que j'exporte » est la regle la plus
  previsible, et c'est celle de tous les outils de capture. Forcer le retour a 1x aurait de plus
  demande de cabler une action React dans la boucle d'export, pour contredire une intention.

## Etape 1 — La geometrie, pure et testee

- [x] 1.1 `replayLogic.visibleBounds(scene, zoom, cx, cy)` — retrecit et translate, en bornant.
- [x] 1.2 Niveaux de zoom et leur ordre, en constante partagee.
- [x] 1.3 Tests : zoom 1 rend la scene ; l'aspect est preserve ; la fenetre ne sort jamais ;
      le centre se reborne quand le zoom baisse ; aller-retour sans derive.

## Etape 2 — L'etat

- [x] 2.1 `useReplayZoom` : niveau, centre, `zoomIn/zoomOut/reset`, `panStep`.
- [x] 2.2 Le deplacement se compte en UNITES MONDE, pas en pixels : le meme geste doit couvrir
      la meme distance de carte quel que soit le zoom.
- [x] 2.3 Tests.

## Etape 3 — Le cadrage

- [x] 3.1 `useReplayView` prend `zoom` et `center`, expose `viewBounds` (la fenetre) EN PLUS de
      `bounds` (la scene) — les deux sont necessaires et ne disent pas la meme chose.
- [x] 3.2 Verifier que le survol suit sans une ligne de plus (c'est le test de D1).

## Etape 4 — Les calques

- [~] 4.1 RIEN A FAIRE — consequence de D1 (les calques cuisent la fenetre, pas la scene).
- [x] 4.2 Test de la memoire : la surface cuite ne depend PAS du niveau de zoom.

## Etape 5 — La surimpression d'angle

- [x] 5.1 `ReplayZoomControl` : croix directionnelle, niveau courant, retour a 1x.
- [x] 5.2 i18n FR + EN.

## Etape 6 — Cablage et export

- [x] 6.1 `ReplayCanvas` — ATTENTION, il est a 652 lignes pour un plafond de 665. Le glisser
      passe par `hoverHandlers` (deja en place) plutot que par de nouveaux attributs.
- [~] 6.2 L'export suit la vue — D7 REVISEE, rien a cabler.

## Hors perimetre

- Le SUIVI D'UN JOUEUR (camera qui recentre sur quelqu'un). C'est le vrai usage d'un rejeu, et
  c'est un lot a lui seul : il demande un sujet, une transition, et une recuisson par
  hysteresis a chaque image. Le zoom manuel est son socle.
- Le zoom a la molette : il est continu par nature, donc contraire a D4.

## Journal

### 2026-09-02 — Lot CLOS

Gate : `tsc -b` EXIT=0 ; `vitest run src/features/match-replay src/features/match-view` ->
**168 fichiers, 2353 tests, 0 echec** (18 neufs) ; `npm run lint` EXIT=0, **0 erreur**, 23
avertissements prexistants. `ReplayCanvas.tsx` : 661 lignes pour un plafond de 665.

**Ce que le lot a coute, et pourquoi si peu.** La decision D1 — le zoom retrecit les bornes — a
fait fondre le perimetre. `worldToCanvas`, `canvasScale`, le survol, les trois familles
d'infobulles, la carte de chaleur et les quatre calques statiques n'ont pas ete touches d'une
ligne : ils lisent tous la projection, et la projection est definie par ses bornes. Le SEUL
endroit ou le zoom apparait dans le dessin est le fond de carte, qui recevait `bounds` (la
scene) et recoit desormais `canvasView.bounds` (la fenetre) — sans quoi l'image serait restee
cadree large pendant que les joueurs zoomaient.

**Ce qui a ete verifie plutot que suppose** : la surface a cuire est INVARIANTE au zoom, epinglee
par deux tests qui passent par sa consequence (`fitWidth` et `usefulHeight` rendent la meme
valeur a tous les paliers) plutot que par la formule. C'est ce qui repond, par construction, a
la crainte d'une memoire qui enfle avec le grossissement.

**Deux decisions revisees en cours de route, et il faut le dire** :

- **D5** — la demande disait « soit le deplacement a la souris SOIT une croix directionnelle ».
  J'ai lu « les deux » au premier tour et ecrit le plan ainsi. C'est un choix, et la croix
  gagne : l'horloge tourne, et surtout un glisser recuirait les calques a chaque mouvement de
  pointeur (il faudrait un blit decale et une cuisson a marge). La croix va par pas discrets.
- **D7** — j'avais ecrit « l'export force le zoom a 1 », par analogie avec le facteur de
  surechantillonnage. L'analogie ne tient pas : ce facteur varie SANS QUE PERSONNE LE VOIE, d'ou
  la necessite de le neutraliser ; le zoom est un geste delibere et affiche. « Ce que je vois est
  ce que j'exporte » est plus previsible, et c'est la regle de tous les outils de capture.

**Decouverte, non traitee** : `bounds` etait une dependance devenue inutile de `draw` (son seul
usage etait le fond de carte, passe a `canvasView.bounds`). Retiree — c'etait un vrai residu,
pas un avertissement a taire.

## Ce qui reste ouvert

- **Gate visuel** — jamais regarde a l'ecran. Les points a juger : la lisibilite a 3x, le pas de
  la croix (un quart de fenetre : trop grand ? trop petit ?), et la place de la surimpression au
  coin bas-droit face au fil.
- **Le SUIVI D'UN JOUEUR** reste le vrai usage d'un rejeu, et le zoom manuel en est le socle.
- ~~Le glisser~~ — FAIT le 2026-09-02 (etape 8), et sans le cout memoire redoute.

## Etape 7 — Molette et clavier (demande du 2026-09-02, apres le lot)

- [x] 7.1 `canvasToWorld` — l'inverse EXACT de `worldToCanvas`, teste par aller-retour. Il
      existe pour une seule raison : savoir quel point du monde est sous le pointeur.
- [x] 7.2 `zoomTowards` — le centre qui laisse un point IMMOBILE quand le zoom change :
      `c' = p + (c - p) x (zoomAvant / zoomApres)`.
- [x] 7.3 `useReplayZoom.zoomAt(dir, towards)` — MEME chemin que les boutons (memes paliers,
      meme rebornage), avec le point a garder fixe en plus.
- [x] 7.4 `useReplayWheelZoom` — ecouteur NON PASSIF pose a la main (React attache `onWheel` en
      passif, `preventDefault` y est ignore et la page defilerait sous la carte), accumulateur
      de delta (un pave tactile emet des dizaines d'evenements de quelques pixels par geste).
- [x] 7.5 Clavier dans `useReplayShortcuts` — `+`/`=` et `-`/`_`, Maj+fleches pour la croix.
      UN SEUL ecouteur clavier, pas deux : les fleches nues valent le saut temporel, et deux
      ecouteurs concurrents sur les memes touches finissent par se marcher dessus.
- [x] 7.6 Tests : aller-retour de projection, point vise immobile, molette = memes paliers.

Gate : `tsc -b` EXIT=0 ; **168 fichiers, 2360 tests, 0 echec** ; lint 0 erreur, 23
avertissements prexistants. `ReplayCanvas.tsx` : 663/665.

DEUX AVERTISSEMENTS NEUFS TRAITES A LEUR CAUSE, pas tus : une reference ecrite pendant le
RENDU (React la reserve au calcul — une valeur ecrite la peut etre perdue si le rendu est
abandonne) est passee dans un effet, dans les deux hooks concernes.

## Etape 8 — Le glisser (2026-09-02)

- [x] 8.1 `layerOffset` — ou poser un calque deja cuit quand le cadrage a bouge depuis sa
      cuisson. Le pixel (0,0) du calque designe un point du monde ; on demande ou ce point
      tombe dans la projection COURANTE. Exact, parce qu'un deplacement est une translation
      pure et que les deux projections partagent leur echelle.
- [x] 8.2 `useReplayZoom.panBy` en unites MONDE — le primitif ; la croix et le glisser s'y
      ramenent tous les deux. Le hook ne connait toujours pas les pixels.
- [x] 8.3 `useReplayDrag` — conversion pixels -> monde par `canvasScale`, capture du pointeur
      (sans elle, un glisser qui deborde du terrain se terminerait sans `pointerup` et la carte
      resterait collee au curseur), et le drapeau `dragging`.
- [x] 8.4 `useReplayStaticLayers.frozen` — les quatre cuissons s'arretent pendant le geste et
      reprennent au relachement. JAMAIS pendant un zoom : l'echelle change alors, et un
      decalage ne replacerait pas une image cuite pour une autre echelle.
- [x] 8.5 `hoverHandlers(layers, pan?)` — le glisser passe par la, pas par un second
      `{...spread}` sur la meme balise : deux spreads ne se composent pas, le second ecrase le
      `onPointerMove` du premier et le survol mourrait en silence.
- [x] 8.6 Tests : 4 sur `layerOffset` (dont l'invariant « replace un point exactement ou un
      recuit l'aurait mis »), 6 sur `useReplayDrag` (sens, conversion, gel).

**LE COUT MEMOIRE REDOUTE N'A PAS ETE PAYE.** L'estimation annoncee (x2,25, ~92 Mo) supposait
une cuisson a MARGE. Le blit decale la rend inutile : on recopie l'image existante ailleurs,
sans rien cuire de plus. **Surcout memoire : ZERO.**

Le seul artefact est une bande non peinte au bord d'attaque pendant le geste, sur les calques
cuits — pas sur le fond de carte, dessine directement, qui suit donc parfaitement. Elle
disparait au relachement. C'est le prix assume de ne rien couter : la solution sans bande
demanderait de la memoire en permanence pour un geste occasionnel.

Gate : `tsc -b` EXIT=0 ; **169 fichiers, 2370 tests, 0 echec** ; lint 0 erreur, 23
avertissements prexistants. `ReplayCanvas.tsx` : 664/665.
