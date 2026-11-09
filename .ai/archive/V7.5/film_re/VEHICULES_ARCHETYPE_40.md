# LES VÉHICULES DANS LE FORMAT DE FILM — archétype 40

> Établi le 2026-07-27, en réponse à une question de l'utilisateur : un kill à la Gungoose sur
> Launch Site, un Ghost sur High Ground — le `chunk_00` ne devrait-il pas les mentionner ?
>
> Reconnaissance seulement : ce document dit où sont les véhicules et ce qui reste à faire.
> Rien n'est encore décodé.

## LA RÉPONSE EN DEUX TEMPS

**Le registre du `chunk_00` ne dira jamais « ce match avait une Gungoose ».** C'est un **schéma
statique**, bit-à-bit identique d'un film à l'autre — l'empreinte FNV de ses 1067 entrées vaut
`a413610cd08e4355` sur Cliffhanger comme sur Catalyst. Il décrit la forme des données, pas leur
contenu.

**Mais il décrit l'archétype véhicule au complet**, et c'est là que tout se joue : savoir quels
véhicules étaient dans un match donné se lit dans le **flux**, en comptant les entités d'archétype
40 instanciées et en lisant leur composant d'identité.

## L'ARCHÉTYPE 40 — le véhicule

48 composants. Il partage la charpente du biped (archétype 35) et ajoute ce qui lui est propre.

**Partie commune avec le joueur** — donc décodable avec les recettes déjà écrites :

    i0  object-position-dynamic-precision       i4  object-body-vitality
    i1  object-translational-velocity           i5  object-shield-vitality
    i2  object-forward-and-up                   i11 object-dead-state
    i3  object-angular-velocity                 i18 unit-control
    i21 unit-desired-aiming-vector              i22 unit-grenade-counts
    i26 unit-equipment                          i28 unit-active-camo-state

Conséquence immédiate : **la position, la vitesse, l'orientation, la vie et le bouclier d'un
véhicule se lisent déjà**, avec le code existant. Un véhicule peut être dessiné sur la carte du
rejeu sans aucun décodage supplémentaire.

**Partie propre au véhicule** :

    i30 vehicle-auto-turret-triggers            i34 vehicle-type-physics
    i31 vehicle-auto-turret-aiming-vector       i35 vehicle-auto-turret-target
    i32 vehicle-transformed-or-desired-open     i36 vehicle-sentry-state
    i33 vehicle-type-state                      i37 vehicle-emp-timer

## CE QUI EST FERMÉ DE MON CÔTÉ — le chemin des armes de joueur ne sert pas ici

Mesuré : `weapon-state-type-info` n'apparaît **que dans l'archétype 35**, en quatre exemplaires
(`i43` à `i46`). L'archétype 40 n'en porte aucun. L'identifiant d'arme d'une Gungoose n'est donc
pas lisible par la recette craquée pour les armes de joueur.

## MAIS LA QUESTION EST DÉJÀ RÉSOLUE DANS LE WORKTREE VOISIN

**Correction apportée après lecture de `filmdec-killweapon`.** Ce document a d'abord été écrit
comme si le nommage des armes de véhicule était ouvert. Il ne l'est pas : le chantier voisin l'a
résolu, **et la Gungoose est son cas d'ancrage nommé**.

Référence : `.ai/ETAT_DE_L_ART_KILLWEAPON.md` §2.5, sections de journal 7ter.45 à 7ter.48.

**La règle R-VÉHICULE** : un tag `weap` est un armement de véhicule si un `vehi` le référence
**ou** s'il pend à la chaîne `vcdd -> sofd -> sofa -> uwfa -> weap`. Cela isole **62 `weap` sur
194**, dont 46 par un `vehi` direct et 16 par la chaîne `vcdd`.

**L'ancre Gungoose passe EXCLUSIVEMENT par la chaîne `vcdd`** — c'est mesuré et vérifié, pas
supposé.

Force de la règle : disjonction **totale** d'avec le catalogue des armes de joueur — **0 des 62**
armements de véhicule porte un nom du catalogue, là où 10,2 étaient attendus par hasard,
`p = 1,1e-06`. Et `uwfa` compte 16 entrées dans tout le jeu, 16 sur 16 référencées par un `sofa`,
0 atteignable depuis un `wcfg`.

**La distinction tourelle / fixe est établie** par deux signaux sans rapport entre eux : la classe
ASCII du `weap` et la nomenclature de la banque du châssis (`_tur_` contre `_veh_`), à 22/23 et
17/22. Confirmée en Theater : le **Ghost est « fixe »**, une tourelle UNSC arrachée et portée à la
main est « tourelle ». Les deux exemples que l'utilisateur cite — Gungoose et Ghost — sont donc
tous deux déjà couverts.

## LA CONNEXION QUE PERSONNE N'AVAIT FAITE — `sofd`

La chaîne du voisin passe par le groupe de tags **`sofd`**. C'est **exactement le même groupe** que
celui où se résout l'identité de capacité spartan trouvée le même jour de mon côté
(`../replay2d/RECETTE_LOADOUT_2026-07-27.md` §9 : `FUN_1407E7648` parcourt le bloc `sofd`, compare
`entrée+0x18` au handle de définition et rend le rang).

Deux chantiers séparés ont buté sur la même structure sans le savoir. **Si `sofd` est la table
d'équipement/armement d'un match, alors le rang de capacité et l'armement de véhicule se lisent au
même endroit** — et le nommage des capacités, que j'ai déclaré fermé côté exécutable, pourrait
s'ouvrir par le chemin de tags qu'ils ont déjà parcouru.

**C'est la piste la plus prometteuse pour les capacités, et elle vient de leur travail.**
À vérifier : leur outillage sait-il énumérer un bloc `sofd` ? Si oui, la palette des capacités est
à portée.

## CE QUI RESTE VRAIMENT OUVERT DE MON CÔTÉ

Le nommage de l'armement est résolu chez eux. Ce qui reste ouvert ici, c'est **le véhicule
lui-même dans le rejeu** :

**1. Compter les entités d'archétype 40 dans un film, et lire leur `vehicle-type-state`.**
C'est ce qui donne la liste des véhicules présents et leur identité. Le traverseur sait déjà
parcourir les records par archétype.

**2. Les dessiner.** Position, vitesse, orientation, vie et bouclier sont déjà décodables.
C'est le gain le plus immédiat et il ne dépend d'aucune recherche supplémentaire.

## UNE ANOMALIE À TRANCHER

`.ai/V7.5/film_re/RECETTE_DECODAGE_FILM_CHUNKS.md` §47 annonce **174 archétypes valides**. L'outil
`cmd/tmp_archlist`, qui lit le même registre, en compte **118**. Le commentaire d'en-tête du même
outil dit 118 dans son titre et 118 à l'exécution.

L'un des deux chiffres est faux. L'écart n'est pas anodin : si le compte réel est 174, il existe
56 archétypes que personne n'a jamais listés, et l'un d'eux peut porter ce qu'on cherche.
**À trancher par une lecture directe du registre**, pas par arbitrage entre deux documents.

## POURQUOI CE CHANTIER EST BIEN PLACÉ

Le terrain est balisé : l'archétype est identifié, la moitié de ses composants est déjà décodable
avec l'existant, la méthode est écrite (`METHODE_RETRO_INGENIERIE_FILM.md`), et la vérité terrain
est fournie gratuitement par deux observations de l'utilisateur.

Ce qui manque n'est pas de la compréhension, c'est du temps de machine.
