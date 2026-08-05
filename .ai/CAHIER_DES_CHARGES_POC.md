# CAHIER DES CHARGES — le rejeu 2D, côté affichage

> Écrit le 2026-07-28. Rassemble les décisions d'interface données par l'utilisateur, avec leur
> raison. Elles ne se rediscutent pas : ce document sert à ne pas les redécouvrir, ni les défaire
> par inadvertance à la passe suivante.
>
> Ce qui relève du décodage est ailleurs (`V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md`). Ici, uniquement ce que
> l'écran doit montrer.

## LA RÈGLE QUI GOUVERNE TOUT LE RESTE

> **Ce qui décrit la confiance du décodeur n'a pas sa place dans l'interface.**

L'écran montre l'état du **joueur**, pas l'état de notre connaissance. Les nuances de provenance —
valeur mesurée contre valeur supposée, primaire contre secondaire, position dans l'enregistrement
— sont des informations d'atelier. Elles restent dans les données, dans les infobulles et dans les
documents ; elles ne se peignent pas.

**Une exception, et une seule** : une **lacune** se signale toujours. « On ne sait pas » et « on a
mesuré qu'il n'y a rien » sont deux états différents, et les confondre serait le seul vrai faux.
Le premier s'affiche, le second peut se taire.

## LA FICHE JOUEUR

### Vitalité

| | couleur | remarque |
|---|---|---|
| bouclier | **bleu plein** | `--vit-shield` |
| vie | **vert plein** | `--vit-health` |

Pas de distinction visuelle entre valeur mesurée et valeur supposée pleine. Le contour hachuré et
le remplissage effacé qui la portaient sont supprimés.

### Mort et réapparition — trois états successifs

1. **L'instant de la mort** : la fiche clignote en rouge, trois battements.
2. **La durée de la mort** : fond rouge tenu, liseré rouge à gauche, nom teinté. Persiste
   jusqu'à la réapparition.
3. **L'instant de la réapparition** : un éclat vert bref.

Pendant la mort, **les barres de vie et de bouclier sont masquées** : un bouclier et une vie à zéro
sont deux rectangles vides qui font croire à une mesure alors qu'il n'y a rien à mesurer.

La durée est la seule des trois qui informe en continu ; les deux autres sont des repères
d'événement, et c'est pour cela qu'ils sont courts.

### Le compteur de réapparition — décidé le 2026-07-28

Pendant la mort, la fiche dit **dans combien de temps le joueur revient**, sur la ligne libérée par
la rangée d'armes — qui est vide quand le joueur est mort et réservait pourtant sa hauteur. La
fiche n'y gagne que 2 px.

**Le nombre est LU, pas déduit d'une constante.** La réapparition se lit sur l'image de départ de
la vie suivante du même joueur : c'est déjà la donnée qui allume l'éclat vert du retour. Un compte
à rebours calé sur une médiane aurait supposé ce que le film dit.

Une barre montre l'avancement depuis la mort, qui est datée par le fil des éliminations — deux
sources distinctes, et c'est voulu.

**Ce que l'infobulle doit porter, et qui ne se peint pas** : la distribution mesurée (90 épisodes,
82 avec retour, médiane 8,0 s, 66 sur 82 à 7,9-8,0 s), le fait que ce palier vaut **pour un match**
et non pour le jeu, et surtout que le nombre est une **borne haute** — le film est codé en delta et
un joueur qui réapparaît sans bouger n'émet rien.

**La lacune se déclare** : 8 épisodes n'ont aucun retour lisible et affichent **« retour ? »**,
jamais un délai deviné. « On ne sait pas » et « il ne revient pas » sont deux états différents.

**Piège technique, déjà rencontré deux fois** : la carte d'équipe est reconstruite par `innerHTML`
à chaque image du rejeu. Une animation déclarée normalement **redémarre à chaque reconstruction**
et reste figée sur sa première image — c'est ainsi qu'une fiche est restée vert plein au lieu de
clignoter. La parade est un **délai négatif** calculé sur l'âge réel de l'événement, pour que
l'animation démarre « dans le passé » à la bonne position.

### Le compteur du joueur

Trois nombres, trois couleurs, les mêmes que partout ailleurs dans la page :

    eliminations / morts / assistances
        vert         rouge     bleu

Trois nombres collés sans distinction se lisaient comme un seul nombre à trois chiffres.

Les assistances ne comptent que les **assistants nommés** : une mort dont l'assistant est inconnu
n'est créditée à personne. Contrôle interne qui passe — le total affiché sur les huit joueurs fait
**17**, exactement le nombre d'assistants nommés mesuré par `killsource`.

### La rangée d'objets — une seule ligne

    [ armes, à gauche ]                    [ grenades et capacité, à droite ]

- **Pas de libellé.** Le nom de l'arme était écrit à droite ; il tronquait en permanence
  (324 troncatures mesurées sur 100 images) et répétait ce que la vignette montre. Il vit
  désormais dans l'infobulle, où il ne coûte rien.
- **L'objet en main est souligné en vert** — arme ou équipement, sans distinction.
- **Primaire et secondaire ne s'affichent pas.** Cette notion vient du format de fichier, pas du
  jeu : celui qui regarde un rejeu ne l'a pas. Le vert désigne donc **ce qui est tenu**, et rien
  d'autre.
- **Vignettes à fond transparent.** Une plaque noire avait été posée dessous pour porter le
  « négatif » ; elle découpait un rectangle sombre dans la carte. Le négatif se fait par filtre
  seul, et il suit le thème.
- **Grenades et capacité à la même hauteur que les armes** (17 px). Deux tailles côte à côte
  donnaient une ligne bancale où l'œil hiérarchisait ce qui ne l'est pas.
- **Quantité de grenades en « ×2 »**, pas en pastilles : à 5 px, deux points se confondaient avec
  de la ponctuation.
- **La grenade équipée** porte un encadré ambré avec fond léger. L'encadré seul ne disait rien.

### L'âge d'une lecture d'inventaire — décidé le 2026-07-28

L'inventaire ne se lit qu'aux **images-clés**, une toutes les **20,0 s** (mesuré : 24 images-clés,
écart médian 200 images, un seul écart de 400). Entre deux, l'écran montre la dernière lecture
connue. La faire passer pour l'état courant était un vrai défaut, relevé par l'utilisateur à 1:06
sur LORD PEINX13 : l'état affiché datait de **9,7 s** et rien ne le disait.

**Une lecture récente est franche, une lecture ancienne s'estompe**, et l'infobulle donne l'âge
exact avec le numéro de l'image-clé — sans quoi « c'est vieux » ne se vérifie pas. C'est la
graduation déjà employée pour le bouclier sur la carte et pour le cône de visée : on ne lui invente
pas une autre grammaire.

**Ce qui pâlit, et ce qui ne pâlit pas.** Les armes pâlissent avec les grenades, la capacité et les
munitions : quand les deux existent, la paire d'armes et l'état d'inventaire viennent de la **même
image-clé, 21 899 fois sur 21 899**. N'en estomper qu'une moitié ferait croire l'autre fraîche.
Le **lancer de grenade** ne pâlit jamais : c'est un événement daté, pas une lecture d'image-clé.

**Une mesure, pas un ornement** : l'âge médian d'une lecture affichée est de **8,4 s**, et **7,1 %
seulement** ont moins d'une seconde. Le contraire de l'hypothèse qui avait été posée.

**Le cas de la lecture à venir** : avant la première image-clé d'une vie, les armes affichées
viennent d'un relevé **postérieur** — 7 380 fiches sur 29 279 (25,2 %), jusqu'à 20,0 s en avance.
Même estompage, sur la valeur absolue de l'écart, et l'infobulle le dit en toutes lettres.

## LE FIL DES ÉLIMINATIONS

- **Colonne dédiée**, entre la carte et les équipes : carte, fil, équipe 1, équipe 2.
- **L'arme du kill remplace le symbole de croix**, en vignette.
- **Une mort assistée reçoit un fond bleuté léger.** Le bleu ne désigne aucune équipe — les deux
  ont leurs teintes sur le liseré gauche, qui reste intact. Il dit seulement « deux joueurs ont
  contribué ».
- **« Pas d'assistant » ne s'écrit pas** : 76 morts sur 93 sont dans ce cas et l'afficher noyait le
  fil. Mais **« assistant inconnu » s'affiche** — la lacune se signale, l'absence constatée non.
- **L'horodatage en mm:ss** est poussé en haut à droite de chaque ligne. Il est placé par `order`
  plutôt qu'écrit en premier : l'ordre de lecture reste « tueur, arme, victime », qui est la
  phrase, et le repère temporel se range au bout sans s'y insérer.
- Une arme non résolue **ne reçoit pas de visuel par défaut**. Un visuel arbitraire affirmerait ce
  qu'on ignore. Elle garde un marqueur explicite, et son tag brut reste consultable.

## LES LIBELLÉS SUR LA CARTE

Nom du joueur et zone où il se trouve, tous deux **contourés de noir épais**. À 11 px sans contour,
un nom posait sa couleur d'équipe sur un fond de carte qui va du blanc au noir selon l'endroit :
illisible la moitié du temps. Le contour rend le texte lisible sur n'importe quel fond **sans avoir
à choisir une couleur de compromis** qui perdrait l'appartenance d'équipe.

**Tailles retenues, en pixels d'écran** : nom du joueur **8,7 px**, zone où il se trouve
**8,5 px**. Un premier réglage à 13 et 10 px a été jugé trop lourd — les libellés se disputaient
la carte avec les noms de zone. L'épaisseur du contour suit la même réduction, faute de quoi il
mangerait la lettre au lieu de la détourer, et l'interligne aussi, faute de quoi la zone se
décollerait du nom qu'elle qualifie.

Ils restent plus petits que les libellés de zone : la hiérarchie est voulue.

**LE PIÈGE, et il explique pourquoi un premier grossissement n'avait rien changé** : la police du
canevas est en **pixels de canevas**, pas d'écran. Le canevas fait 1600 px de large et s'affiche
autour de 840 — une police déclarée à 15 px y devient **8 px réels**. Les tailles se donnent donc
en pixels d'écran et se multiplient par le facteur de réduction courant, ce qui les rend
constantes quelle que soit la fenêtre. Vaut pour le texte comme pour l'épaisseur du contour.

## LES POINTS DE JOUEUR ET LES CÔNES DE VISÉE

Même piège que les libellés, même correction : tout ce qui est **destiné à l'œil** — rayon d'un
point, longueur d'un cône, épaisseur d'un trait — se déclare en pixels d'écran et se multiplie par
le facteur de réduction. Ce qui appartient au **monde** — positions, trajectoires — n'y touche pas.

Mesuré : le noyau d'un point valait **1,4 px réel** sur un écran à 1920. Il vaut désormais 4,6 px,
et le cône passe de 13,8 à 46 px.

Deux ajouts de contraste, tous deux pour la même raison — la carte va du blanc au noir selon
l'endroit, et une couleur d'équipe s'y perd la moitié du temps :

- **liseré sombre sous le point**, posé avant le remplissage ;
- **cône dégradé** du centre vers le bord, dense à l'origine où il faut lire qui vise, transparent
  au bout où il ne faut pas masquer le décor ; et son axe doublé d'un trait sombre en dessous.

## LES EFFETS DE TIR, PAR FAMILLE D'ARME

Direction retenue. L'effet doit dire de quoi le joueur est mort, pas seulement qu'il est mort.

| famille | effet |
|---|---|
| balistique | éclair court et sec, cône étroit, blanc chaud, décroissance rapide |
| plasma | traînée bleu-violet plus lente, légère ondulation, décroissance molle |
| forerunner et sentinelle | raie continue plutôt qu'un éclair, cyan-or, épaisseur constante |
| roquette et explosif | départ épais plus halo circulaire à l'impact, orange saturé |
| **mêlée** | **pas d'éclair de bouche** — une onde courte centrée sur le tueur |
| needler | éclats roses convergents plutôt qu'un trait unique |

La mêlée mérite qu'on s'y arrête : un coup de marteau n'est pas un tir, et l'effet ne doit pas le
prétendre.

Sous `prefers-reduced-motion`, aucune animation : un marqueur statique.

## LA MISE EN PAGE

- Quatre colonnes : **carte, fil, équipe 1, équipe 2**. Sous 1400 px, le fil repasse sous les
  équipes.
- La bande latérale se dimensionne sur le **conteneur**, jamais sur la fenêtre. Une mesure en `vw`
  à l'intérieur d'un conteneur plafonné fait grossir la bande pendant que la page ne grossit plus :
  la carte rétrécissait quand l'écran s'agrandissait.
- La carte est l'objet principal : elle doit **grandir** avec la fenêtre.

### LE DÉFAUT QUI EST REVENU QUATRE FOIS — à connaître avant de toucher à la mise en page

Quatre fois, sous quatre formes, le même mécanisme :

| forme | mesure |
|---|---|
| fil écrasé par les colonnes d'équipe | 317 px de colonnes → 72 px de fil, puis 385 → 5, puis 523 → 0 |
| équipes écrasées par le fil (branche étroite) | 41 px pour les deux équipes |
| fiches joueur coupées | 175 colonnes sur 200 débordaient, le quatrième joueur invisible |
| entrées du fil compressées | rendues à 15,6 px pour 63,5 px naturels, aucun défilement |

**La cause est toujours la même** : dans un conteneur flex, les enfants ont `flex-shrink: 1` par
défaut. Quand le contenu dépasse, ils se **compressent** au lieu de déborder — et un conteneur qui
ne déborde pas ne défile jamais.

**Les deux règles qui ferment le défaut :**

1. Tout enfant d'un conteneur destiné à **défiler** porte `flex: 0 0 auto`.
2. Deux blocs voisins ne doivent **jamais** pouvoir prendre la hauteur l'un de l'autre : on répartit
   par une part fixe, jamais par « l'un grandit, l'autre se débrouille ».

Un plancher (`min-height`) ne suffit pas : il déplace le problème sans le fermer.

## CE QUI RESTE À FAIRE, CÔTÉ AFFICHAGE

- Médailles en images.
- **Signaler la lacune quand aucune ligne d'inventaire n'existe** : 9 080 fiches sur 39 880
  (22,8 %) affichent des armes sans grenades, sans capacité et sans munitions, et rien ne le dit.
  Découvert le 2026-07-28 en mesurant l'âge des lectures ; laissé ouvert plutôt que traité au vol.
- Noms de joueurs plus lisibles — plus gros que maintenant, moins que les libellés de zone,
  probablement avec un contour noir.
- États actifs des capacités : surbouclier en encadré doré, camouflage en effet de verre,
  translocateur en bordure animée du bleu électrique au jaune orangé.
- Animation d'échange quand le sélecteur bascule d'un emplacement à l'autre.
