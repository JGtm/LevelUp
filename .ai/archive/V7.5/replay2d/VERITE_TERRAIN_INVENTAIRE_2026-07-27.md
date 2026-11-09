# VÉRITÉ TERRAIN — inventaire au spawn, relevée en Theater le 2026-07-27

> Film `9e8fb31b-ea96-4848-a3b0-03117171d01e` — Cliffhanger, Slayer:Arena Super Fiesta,
> 24/07/2026 19h21.
>
> **RELEVÉ PAR L'UTILISATEUR, PAS PAR LE DÉCODEUR.** C'est la seule source non circulaire dont
> on dispose pour nommer les types de grenade et les capacités : aucun de ces libellés n'existe
> dans les fichiers du jeu en build de release.

## L'INSTANT DU RELEVÉ — 00:25, soit l'image 250

J'avais demandé une lecture à **00:03**, en croyant que c'était le spawn. L'utilisateur a lu à
**00:25**, le début RÉEL du match, en précisant : « ton timing a foiré ».

**L'instant du relevé est donc 00:25 = image 250** (cadence 10 images/s). Toute confrontation se
fait à cette image, pas à l'image 34 où j'avais pris mes états — celle-ci est antérieure au début
du match et ne reflète pas les loadouts de spawn.

L'horloge du décodeur, elle, est **bonne** : le fil des morts produit sa première entrée à
**31,6 s**, en plein dans la fenêtre 30-40 s observée par l'utilisateur, et ce calque est décodé
par un chemin indépendant. Il n'y a donc aucun décalage à corriger avant d'exploiter ce tableau.

## LE RELEVÉ

| joueur | grenades | capacité | arme en main | chargeur | réserve |
|---|---|---|---|---|---|
| aldusbroncus | **Dynamo × 2** | propulseur (dash), 5 utilisations | Cremator | 6 | 6 |
| JGtm | **Frag × 2** | propulseur (dash), 5 utilisations | lance-roquettes | 2 | 2 |
| LORD PEINX13 | **Spike × 2** | capteur de menace, 4 utilisations | déchiqueteur | 8 | 16 |
| IKE ILYA | **Plasma × 2** | grappin, 5 utilisations | MA40 | 25 | 75 |
| whiteknight2519 | **Dynamo × 2** | grappin, 5 utilisations | Hydra | 6 | 6 |
| VitaminA1688 | **Frag × 2** | propulseur (dash), 5 utilisations | Bulldog | 12 | 12 |
| Akatsuki fire17 | **Dynamo × 2** | mur portatif, 1 utilisation | déchiqueteur | 8 | 16 |
| JAVIERLOLITO54 | **Plasma × 2** | grappin, 5 utilisations | marteau | charge 100 % | — |

Observation complémentaire : à 00:45 `aldusbroncus` est mort, et **son compteur de dash est passé
de 5 à 4**.

## CE QUE CE RELEVÉ PERMET, ET CE QU'IL NE PERMET PAS

**PERMET — les munitions (item 6.2 de la liste Notion).** Huit couples chargeur/réserve, avec
l'arme correspondante. C'est de quoi calibrer `i30`/`i33` (munitions) et `i31`/`i34` (chargeur),
dont seule la largeur est mesurée à ce jour. Les valeurs couvrent une plage utile : 2/2, 6/6,
8/16, 12/12, 25/75.

**PERMET — le contrôle interne des capacités.** Trois joueurs portent le **propulseur** et trois
le **grappin** : deux groupes de trois, plus un capteur de menace et un mur portatif isolés. Tout
index de capacité que le décodeur attribue doit respecter ce partitionnement.

**PERMET — la distribution des grenades.** Trois Dynamo, deux Frag, deux Plasma, un Spike sur
huit joueurs. Les quatre types du jeu sont représentés.

**PERMET — l'appariement compteur → type**, à condition de lire le décodeur à l'image 250 et non
à l'image 34. La confrontation que j'ai tentée en fin de session comparait deux instants
différents ; elle ne mesurait rien et ses conclusions sont à jeter. Refaite au bon instant, elle
tranche : quatre types de grenade répartis sur huit joueurs, avec trois Dynamo, deux Frag, deux
Plasma et un Spike, cela contraint fortement l'affectation des quatre compteurs d'`i22`.

## POURQUOI CE DOCUMENT EXISTE

Trois raisons, dans l'ordre d'importance :

1. **Les noms de capacités ne sont récupérables d'aucune autre façon.** Mesuré : sur 11 tags
   `eqip`, un seul porte une chaîne lisible, et aucun libellé n'apparaît dans les 238 tables
   `uslg` — cohérent avec `fileNameSize = 0` en build de release. Ce relevé est la source.
2. **Les munitions n'avaient aucune vérité terrain.** Les largeurs de `i30`/`i33` sont mesurées
   mais leur grammaire ne l'est pas ; sans valeurs attendues, on ne peut pas la valider.
3. **Une observation humaine ne se refait pas à volonté.** Elle coûte du temps à l'utilisateur.
   Elle se consigne donc intégralement, y compris les détails que je n'ai pas demandés — le
   marteau à 100 % de charge, le compteur de dash passé à 4 — parce qu'on ne sait pas encore
   lesquels serviront.

## REMARQUE DE L'UTILISATEUR À TRAITER

> « pour les armes il y avait les noms donc je suppose que pour tout le reste aussi »

C'est une objection légitime et il faut y répondre précisément : les **armes** sont nommables
parce que leur identifiant est un tag `proj`/`jpt!` dont le nom survit dans les archives. Les
**capacités** ne le sont pas parce que `i48` ne porte pas un identifiant de tag mais un **index
dans la palette d'équipement du match** — un numéro d'ordre, pas une référence. Ce point est
mesuré, pas supposé, mais il mérite d'être revérifié : si `i48` portait en réalité un tag `eqip`
décalé (comme les grenades portent un tag `proj` décalé d'un bit), le nommage deviendrait
possible par catalogue. **À tester.**
