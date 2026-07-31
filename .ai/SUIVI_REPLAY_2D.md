# SUIVI — rejeu 2D

> Fichier de suivi tenu à jour au fil des sessions. Dernière mise à jour : **2026-07-28** (soir).
>
> Il remplace la question « où en est-on ? ». Trois documents lui répondent ensemble et ne se
> recouvrent pas :
> - **celui-ci** : ce qui est fait, ce qui reste, dans quel ordre, avec les blocages ;
> - `RECETTE_LOADOUT_2026-07-27.md` : ce qui est décodé, avec les mesures ;
> - `CAHIER_DES_CHARGES_POC.md` : ce que l'écran doit montrer, et pourquoi.
>
> Voisins utiles : `ETAT_DE_L_ART_CHANTIER_VOISIN.md` (index du worktree `filmdec-killweapon`),
> `METHODE_RETRO_INGENIERIE_FILM.md` (comment chercher, et comment se tromper).

---

## LE POC N'ÉTAIT PAS VIABLE — trois défauts sur quatre sont RÉPARÉS (2026-07-28)

Constat posé avec l'utilisateur le 2026-07-28, puis traité par
`PLAN_REJEU_2D_FIABILISATION.md` (étapes 1 à 5 closes ; étape 6 ouverte).

| ce qui n'était pas viable | avant | après | où |
|---|---|---|---|
| le rattachement événement → joueur est un **vote** | 26 slots couverts sur 99 | **90 vies nommées sur 105** par le fil des morts. **Le vote est SUPPRIMÉ**, pas mis en secours | étape 2 |
| les événements non rattachés sont **jetés sans trace** | 53 records perdus, trouvés en regardant l'écran | **475 + 44 = 519, exactement** ; chaque rejet a une cause nommée | étape 3 |
| le décodeur **ne publie pas sa couverture** | 147 publiés (chiffre lui-même faux, cf. plus bas), écart écrit nulle part | le bandeau porte **475 / 519** et un **verdict de publication** | étape 4 |
| tout vit en **injection manuelle** dans un fichier HTML | — | **NON TRAITÉ** — c'est l'étape 6, la seule qui touche la production | étape 6 |

**Ce qui a réellement débloqué la situation n'est pas ce que le plan prévoyait.** La cible était
`player-representation-component` (archétype 5, rang `i21`), qui sérialise le lien joueur → entité.
Le composant existe bel et bien, mais il est **inatteignable en l'état** : le parcours séquentiel du
flux delta ne tient pas la distance sur ce film (125 records ti=5 rencontrés, 47 % de désync, des
slots impossibles). L'échec est mesuré et consigné au plan. C'est le **repli** — nommer chaque vie
par la mort qui la termine, lue dans le chunk highlight — qui porte le résultat.

**Résultat mesuré, avec ses témoins** :

| mesure | valeur | témoin |
|---|---|---|
| vies nommées par la mort qui les termine | 90 / 105 | 10 (morts replacées au hasard) |
| écart d'appariement | médiane 34 ms, maximum 36 ms | — |
| slots changeant de porteur | **0 / 90** | — |
| tirs publiés | **475 / 519 = 91,5 %** | 398 par le vote seul |
| arme du tir dans le loadout du slot désigné | **405 / 418 = 96,9 %** | **3,7 %** (autre slot vivant) |

**Le critère de succès du plan (> 85 %) est atteint : 91,5 %.**

**LE VOTE A ÉTÉ SUPPRIMÉ, PAS MIS EN SECOURS** — décision de l'utilisateur le 2026-07-28 : « je
préfère rien afficher que quelque chose de complètement faux ». Coût mesuré avant de supprimer :
496 → 475 tirs, 68 → 63 lancers. Gain : les désaccords entre méthodes passent de 4 à **0**, et
100 % de ce qui est affiché vient d'une lecture. Les 24 tirs qui changeaient de propriétaire selon
la méthode employée ne sont plus publiés du tout.

**UN CHIFFRE DU DIAGNOSTIC ÉTAIT PÉRIMÉ, et il faut le dire** : « 147 tirs publiés, soit 28 % »
datait d'avant l'ajout de la source « lancers de grenade » au vote. Le point de départ réel était
**398 tirs, soit 77 %**. Le gain est donc 398 → 496, pas 147 → 496. La direction du diagnostic
reste juste — le vote était bien le goulot — mais son ampleur était surestimée.

---

## PRIORITÉS ARRÊTÉES AVEC L'UTILISATEUR — 2026-07-28

| rang | quoi | statut |
|---|---|---|
| 1 | Médailles en images · compteur de réapparition · reprise des visuels d'armes | à faire |
| 2 | Hors Notion : trou de tirs, réfutation structure, productionisation | **important, à ma charge** |
| 3 | États actifs des capacités (21.1) · état vivant des objectifs (15.2) · dispositifs de carte (12) | à faire |
| 4 | Découpage des zones sur le décor (9, 9.2, 16) | **priorité moyenne** — exige une règle valable pour **toutes** les cartes, pas seulement Cliffhanger |
| — | Véhicules | **ÉCARTÉ** par l'utilisateur |

---

## FAIT

### Décodage

| quoi | preuve |
|---|---|
| Grenades — rang → type | **trois chaînes indépendantes** : 35 lancers sur 35 appariés aux décréments, table `grenade_types` du binaire, adressage `tagref` du chantier voisin. Question close |
| Munitions et jauge d'énergie | union à deux branches lue dans le binaire ; table de 16 armes, réserve multiple entier du chargeur 16/16 ; jauge = fraction [0,1] sur 12 bits |
| Sélecteur d'emplacement `i42` | sens établi par oracle interne (l'emplacement dont les munitions bougent), 94,7 % et 95,4 % |
| Carte mémoire du loadout | base `0x7F0`, pas `0x90`, 4 entrées ; fermeture arithmétique indépendante `0x7F0 + 4×0x90 = 0xA30` |
| Armes de kill | 93 morts sur 93, 87 nommées, tag brut conservé à côté du libellé |
| Pont slot → joueur | 8/8, croisé avec le relevé Theater |
| Palette de capacités `sofd` | 27 entrées nommées — **mais le contrôle échoue 2/4, voir blocages** |

### Affichage

Fil des éliminations en colonne dédiée, avec l'arme de kill à la place de la croix, l'horodatage,
les assistances surlignées en bleu et leurs trois états distincts. Fiche joueur avec icônes du jeu,
rangée unifiée sans libellé, compteur K/D/A coloré, vie en vert et bouclier en bleu pleins, mort en
rouge tenu avec clignotement et éclat vert au retour. Libellés de carte contourés. Effets de tir
par famille d'arme sur les morts.

### Corrections de fond

- **Le POC mélangeait deux films** — cause racine de deux jours de contradictions. Réparé ; chaque
  bloc porte désormais l'identifiant de son film.
- **Quatre défauts graves de mise en page**, trouvés par la première vérification visuelle mesurée :
  carte 428 → 1054 px, fil 10 → 660 px, chevauchement horloge/score supprimé, 324 troncatures → 0.

---

## BLOCAGES ET RÉSERVES — à lire avant de reprendre

### La mesure décisive des capacités n'a JAMAIS tourné

La palette `sofd` donne mur = rang 2 et répulseur = rang 6 ; ma table annonçait mur = 3 et
capteur = 6. **Contrôle : 2 confirmés sur 4.** Le motif est net — les deux confirmés sont adossés à
un **triplet** de joueurs, les deux contredits à une **observation unique**.

**Ce qu'il faut faire, et c'est court** : le champ est lu sur **3 bits** dans le record de biped ;
le binaire décrit **6 bits précédés d'une porte**. Relire les mêmes records sur 6 bits et vérifier
si le mur ressort au rang 2 et le capteur au rang 1. Cette mesure est désignée comme prioritaire
depuis le 2026-07-27 et n'a toujours pas été faite.

### La palette n'est pas globale

Sur 46 équipements présents dans plusieurs `sofd`, **20 changent de rang**. Le `sofd` employé est
choisi à l'exécution. **Tant qu'on ne sait pas lequel s'applique à un film donné, la table n'est pas
industrialisable.** Les trois `sofd` de la famille A partagent un préfixe identique sur les rangs
0 à 9, ce qui sauve l'exploitation sur cette famille.

### Objets au sol — bloqué, et proprement

La position des entités `ti=42` et `ti=37` **n'est pas décodable aujourd'hui**. Trois routes
essayées, trois réfutées sur pièce : sur la voie delta, 5 échantillons contre **1 006 sur un jeu de
slots fantôme** de même cardinalité — le signal est sous le bruit. Le discriminant de récurrence
spatiale n'a donc pas pu être évalué : **son entrée n'existe pas**.

### ~~Défaut d'affichage ouvert — l'âge de la lecture n'est pas montré~~ — RÉGLÉ le 2026-07-28

Observé par l'utilisateur à 1:06 sur LORD PEINX13, reproduit à l'identique (slot 520, image-clé
563, image courante 660, **âge 9,7 s**) puis corrigé : une lecture récente est franche, une lecture
ancienne s'estompe, l'infobulle donne l'âge exact et le numéro de l'image-clé.

**La mesure préalable contredisait l'hypothèse de départ**, et c'est elle qui justifie la
correction. Sur les 4 985 images et les 8 joueurs, 21 899 fiches affichent un état d'inventaire :
âge **médian 8,4 s** (quartiles 3,9 et 13,7 s, maximum 39,9 s), et **7,1 % seulement** ont moins
d'une seconde. L'estompage sert donc en permanence, il n'est pas un ornement.

**Reste ouvert, découvert en chemin** : 9 080 fiches sur 39 880 (22,8 %) affichent des armes sans
aucune ligne d'inventaire, et rien ne signale cette lacune. Le cas « on ne sait pas » y est muet.

### ~~Rappel des tirs — 34 secondes manquantes~~ — DIAGNOSTIQUÉ le 2026-07-28, réparation à appliquer

**Le signal était là depuis le début.** Énumération sans ancre sur les 27 morceaux : le record de
dégât (type 105) est présent 832 fois, dont 519 en variante longue, et **31 records longs tombent
avant 66,0 s, le premier à 30,2 s** — 1,4 s avant la première mort. Cause réelle : un **rejet en
aval**, la porte `uniqueSlotFor` ne couvrant que 26 slots sur 99 et aucune vie d'avant 66 s.

**Réparation mesurée** : nommer chaque vie par la mort qui la termine, au lieu de la faire voter.
147 → 443 tirs rattachés, 0 → 31 avant 66 s, non-régression 125/125.

#### CE QUE LE FILM PORTE VRAIMENT — à ne jamais surinterpréter

Le film ne porte pas les tirs : il porte les **records de dégât**. Mesuré sur `000d5950` :

| | valeur |
|---|---|
| tirs partis (API) | **2 228** |
| tirs qui touchent (API) | **595** |
| records longs dans le film | **519** |
| rapport record / touche | **0,87** |

**Le plafond n'est donc pas « 100 % des tirs effectués » mais « environ 87 % des tirs qui ont
touché ».** Les 1 633 tirs manqués ne sont pas perdus par notre décodeur : ils ne sont pas dans le
film. Avec un pont parfait on atteindrait 479 des 519 records, soit 92 % de ce qui existe.

Correction au passage : le « ratio 0,24 » laissé non résolu au journal était calculé contre
`shots_fired`, mauvais dénominateur.

### ~~La permutation index → joueur~~ — REQUALIFIÉE le 2026-07-28 (soir) : c'est NOTRE tri, pas le format

**Correction d'une annonce trop large.** J'ai publié la permutation comme une découverte sur le
film. Vérification faite à la demande de l'utilisateur — qui trouvait étrange qu'un index de joueur
soit instable — c'est presque certainement **notre propre artefact**.

Le `pi` du roster est un **tri alphabétique ASCII**, majuscules avant minuscules :

    0 Akatsuki fire17 · 1 IKE ILYA · 2 JAVIERLOLITO540 · 3 JGtm
    4 LORD PEINX13 · 5 VitaminA1688 · 6 aldusbroncus · 7 whiteknight2519

Ce n'est pas un index du jeu, c'est un ordre que **nous** avons imposé. Le film porte l'index
interne du jeu. Les deux n'ont aucune raison de coïncider, et leur écart n'apprend rien sur le
format — il dit seulement qu'on a comparé notre tri à leur numérotation.

L'intuition de l'utilisateur était juste sur les deux points : l'index joueur **est** stable, et il
ne change ni en cours de partie ni selon le composant.

**Ce qui reste vrai et utile** : la bijection mesurée entre l'index du film et les huit joueurs.
Elle sert de table de correspondance, pas de découverte.

### ~~LA VRAIE CAUSE DES VIES NON NOMMÉES — un vote, pas une lecture~~ — RÉPARÉ le 2026-07-28

**Le vote a été remplacé par une lecture** (étape 2 du plan). Ce qui suit reste le diagnostic
d'origine, conservé parce qu'il explique la structure du code actuel : `owners.go` porte encore
les deux sources votées, mais **en repli explicite et compté** (`OwnerReport.FromFallback` = 6
slots sur 96). Le repli n'est pas du code « au cas où » : il couvre un cas identifié — un film
sans fil des morts, ou un mode sans mort — et son déclenchement est publié.

L'utilisateur a posé la bonne question : « tu lis le fichier comme une phrase, ou tu dis
"le bouclier c'est l'emplacement X, je prends la valeur, je ferme" ? En séquentiel ça peut
introduire des erreurs, alors qu'on veut lire ce qui pointe de manière indépendante. »

Réponse, lue dans `replay/owners.go` : le rattachement d'un événement à un joueur se fait par
**vote heuristique**.

    votes[slot][g.PlayerIndex]++      // les lancers de grenade votent
    out[slot] = best                  // le plus voté gagne

Puis `uniqueSlotFor` exige qu'**exactement un** slot corresponde à l'instant, sinon l'événement est
jeté **en silence**. Avec 70 lancers pour 99 vies, la carte ne couvre que **26 slots sur 99**.

**C'est la cause commune** des vies non nommées, du trou de tirs, et du plafond de rattachement.
La réparation identifiée — nommer chaque vie par la mort qui la termine — remplace le vote par une
lecture d'un fait daté. C'est le même geste que celui qui a débloqué le chantier voisin : atteindre
directement ce qu'on cherche au lieu de tout parcourir.

### La permutation, pour mémoire

L'index d'attaquant du record **n'est pas** l'index de joueur du roster. Deux numérotations
différentes, et la bijection a été trouvée par deux chaînes sans pièce commune qui donnent le même
résultat chiffre pour chiffre :

    0 whiteknight2519 · 1 JAVIERLOLITO540 · 2 JGtm · 3 LORD PEINX13
    4 IKE ILYA · 5 Akatsuki fire17 · 6 aldusbroncus · 7 VitaminA1688

Elle n'affecte pas le pipeline des tirs (le pont y est film-seul) mais **elle détruit toute
confrontation film contre base**, et elle explique très probablement le témoin resté inexpliqué.

### ~~« Inventaire non lu »~~ — C'ÉTAIT UN BUG, corrigé le 2026-07-28 (soir)

**L'utilisateur l'a trouvé par le raisonnement, pas par l'observation** : « c'est une structure
universelle et prédictible, on a forcément une valeur ou 0, il n'y a pas d'entre-deux ».

Il a raison, et la cause est une règle du format qui n'avait pas été appliquée ici : **le flux est
différentiel, donc un composant absent d'un record veut dire INCHANGÉ, pas INCONNU.** Le rejeu
fait déjà le bon geste pour le bouclier — il remonte en arrière jusqu'à la dernière mesure — mais
`invAt` rendait le dernier état et rien d'autre. Un joueur dont les grenades avaient été lues à
l'image 563 et dont le record suivant ne portait que la capacité les voyait donc disparaître.

**Correction** : chaque champ est désormais reporté séparément, comme pour le bouclier.
**Le report est sûr parce qu'un slot EST une vie** — il change à chaque réapparition, donc il ne
peut jamais franchir une mort, et une dotation ne survit pas à son porteur.

**Mesure avant / après**, sur un balayage d'une image sur cinquante :

| | avant | après |
|---|---|---|
| fiches vivantes sans ligne d'inventaire | **22,8 %** | **5,7 %** |

Ce qui reste est une vraie lacune : un champ **jamais lu depuis le début de la vie**. « Pas encore
transmis » et « transmis puis inchangé » restent deux états distincts, et seul le premier
s'affiche comme une lacune.

### Ce que la mesure d'origine disait, et sa réserve

Question posée par l'utilisateur : est-ce parfois un compteur simplement **vide** ?

**Non, mesuré.** Sur les 184 états : 120 portent grenades et capacité, 52 ne portent ni l'un ni
l'autre, 12 portent la capacité seule — et **zéro état a des grenades présentes toutes à zéro**.
Le cas « compteur à zéro affiché comme non lu » ne se produit pas.

Deux choses trouvées en mesurant :

- **Le compteur d'utilisations est absent sur 132 lectures de capacité sur 132.** Ce n'est pas
  « vide », c'est « jamais localisé ». Le « ? » est universel, pas occasionnel.
- **RÉSERVE NON TRANCHÉE** : cette mesure porte sur ce qui **sort** de l'extraction, pas sur ce que
  le record contient. La grammaire exige un en-tête à 4 et des valeurs sous 3 ; un état hors bornes
  est jeté en silence et apparaîtrait ici comme « absent ». C'est la distinction « le format ne
  porte pas X » contre « notre lecteur ne trouve pas X », et elle reste ouverte sur ce point.

---

## À FAIRE

### Rang 1 — sans recherche préalable

- [ ] **Médailles en images** (Notion 19). 44 événements déjà datés en base, calés à 22 ms sur les
      tués. Il manque les visuels.
- [x] **Compteur de réapparition** (Notion 6) — **fait le 2026-07-28**. Le nombre est **lu** dans
      le film (image de départ de la vie suivante), pas déduit d'une constante. Distribution
      mesurée : 90 épisodes de mort, **82** avec retour lisible, **médiane 8,0 s**, 66 sur 82
      (80,5 %) à 7,9-8,0 s. Deux réserves portées à l'écran : c'est une **borne haute** (film en
      delta, un joueur immobile n'émet rien — les 4 épisodes au-delà de 9 s sont ce cas), et les
      **8 épisodes sans retour** affichent « retour ? ». Anticipation FFA/BTB : **non traitée**,
      un seul mode disponible.
- [ ] **Reprise des visuels d'armes** (Notion 20.3). **Dépriorisé le 2026-07-28** par
      l'utilisateur : « c'est du détail de mapping, c'est rien ». Les 28 fichiers sont nommés en français
      partiel et parfois faux : `Cremator.png` est la Cindershot, `Carabine.png` la Pulse Carbine.
      **À reprendre entièrement, pas au cas par cas** — sinon un vrai Cremator héritera du visuel
      de la Cindershot.

### Rang 2 — hors Notion, à ma charge

**Préalable transverse — `PROPOSITION_FIABILITE_RATTACHEMENT.md`.** Les trois défauts relevés par
l'utilisateur (vote au lieu de lecture, tri alphabétique pris pour une identité, rejet silencieux)
n'en font qu'un : **on a remplacé des lectures par des inférences, puis comparé nos inférences
entre elles**. Quatre corrections proposées, la première débloquant à elle seule le trou de tirs.

- [x] **Combler le trou de 34 secondes des tirs**, avec le fil des morts comme oracle.
      **Fait le 2026-07-28** — et par une voie plus large que prévu : le fil des morts ne sert pas
      d'oracle de contrôle, il est devenu la SOURCE UNIQUE du rattachement. 475 tirs sur 519.
- [~] **Réfutation de la structure — CLOSE le 2026-07-28 par l'utilisateur.** « J'avais validé le
      visuel rendu par Reclaimer, je vois pas ce que tu veux d'autre. » La carte reconstruite a
      donc été confrontée à un outil EXTERNE et indépendant, ce qui vaut mieux qu'une réfutation
      interne : une validation par une source qui ne partage aucune pièce avec notre décodeur est
      exactement le contrôle qu'on cherchait. Ne pas rouvrir sans motif neuf.
- [x] **Productionisation — FAITE le 2026-07-28** (étape 6 du plan). `features/match-replay`
      porte désormais la structure de carte, les tirs, les lancers et la **couverture**
      (« 475 / 519 » avec la cause de chaque rejet, et un verdict de publication). Servi
      **en local uniquement** : garde daté, avec sa cible de retrait et son critère mesurable.

### Rang 3 — demande du décodage

- [ ] **États actifs des capacités** (Notion 21.1) : surbouclier en encadré doré, camouflage en
      effet de verre, translocateur en bordure animée du bleu électrique au jaune orangé.
- [ ] **État vivant des objectifs** (Notion 15.2) : qui porte le drapeau, où en est une capture.
      28 composants `managed-objective-*` sur l'archétype `ti=11`. `object-reference` est en `i3`,
      donc **avant le mur** ; `interaction-filter` en `i4` est polymorphe à 6 sous-types et bloque
      tout ce qui suit. **Livrable en deux temps** : `i0..i3` maintenant, le reste après reverse.
- [ ] **Dispositifs de carte** (Notion 12). Le canon humain est trouvé — 41 propulsions verticales
      en 2 grappes contre 75 chutes éparpillées sur 11 endroits. Reste à industrialiser.

### Rang 4 — priorité moyenne, exige une règle générale

- [ ] **Découper les zones de callout sur le décor réel** (Notion 9, 9.2, 16), via le bloc
      `instanced physics instances` du `sbsp` (@0x1E4). **Contrainte posée par l'utilisateur : la
      solution doit valoir pour les 30 cartes, pas seulement pour Cliffhanger.** Deux précautions
      déjà identifiées : découper **par étage** (on a `top`/`bottom` par prisme), et **conserver le
      polygone brut à côté du découpé** — le brut est la donnée du jeu, le découpé notre
      interprétation.

### Non planifié

- [ ] Carte de chaleur (Notion 13) · sons d'arme au kill (Notion 14) · POC 2 sur Catalyst en CTF
      (Notion 18, film `64e8adfa`, pré-requis : confirmer la présence d'entités `ti=11`).

### Écarté

- **Véhicules** — écarté par l'utilisateur le 2026-07-28. Pour mémoire, si la question revient :
  l'archétype 40 partage la charpente du biped, donc position, vitesse, vie et bouclier sont déjà
  décodables ; et le nommage de leur armement est résolu dans le worktree voisin (règle
  R-VÉHICULE, la Gungoose est leur cas d'ancrage).

---

## LE POC ENTRE DANS L'APP — 2026-07-28 (soir)

Plan : `PLAN_POC_DANS_L_APP.md`. L'étape 6 du plan de fiabilisation avait sorti le rejeu du POC ;
elle n'avait pas porté **l'écran**. Le POC est un poste de travail à quatre colonnes, la
production n'en avait qu'une.

| ce qui manquait | état |
|---|---|
| fond de carte tramé (le vrai sol) | **fait** — remplace l'empilement de 10 223 rectangles |
| cône de visée, bouclier, anneaux d'étage, apparition, mort, projectiles | **fait** — les données étaient déjà dans l'artefact, seul le rendu manquait |
| identité des joueurs | **fait** — 90 traces nommées sur 99, 8 joueurs |
| colonnes d'équipe et fiches joueur | **fait** — K/D/A, bouclier, armes portées, réapparition lue |
| fil des éliminations | **à faire** — l'appariement kill ↔ death n'est pas mesuré |
| inventaire (grenades, capacité, munitions) | **à faire** — vit encore dans `cmd/tmp_kfinv` |
| effets de tir par famille, médailles | **à faire** |

**LE FILM PORTE LES GAMERTAGS LUI-MÊME** — trouvé en cherchant d'où venaient les noms : 32
octets UTF-16LE dans le chunk highlight, à côté du xuid, déjà décodés par
`analysis.ParseHighlightEvents`. Conséquence : l'artefact nomme les joueurs **sans base de
données**, et la propriété « tout le rejeu tient hors ligne » est préservée. Le roster ainsi lu
**reproduit exactement** la bijection index → joueur du chantier — une **troisième** chaîne
indépendante vers le même résultat.

**Ce que le film ne porte pas** : l'ÉQUIPE et les compteurs du match. Ils vivent dans la base et
se joignent côté client **par xuid** — jamais par un index, qui est un ordre et non une identité.

**Deux tests étaient rouges depuis le commit précédent** : le garde « servi en local » de
l'étape 6 a été posé sans adapter les tests du handler. Réparés, et le branchement du garde
gagne le test qui lui manquait.

**Revue visuelle : différée, à la charge de l'utilisateur.** Aucun gate visuel n'est déclaré
passé.

---

## JOURNAL DES MISES À JOUR DE CE FICHIER

- **2026-07-28** — création, après arbitrage des priorités avec l'utilisateur.
- **2026-07-28** — compteur de réapparition livré ; défaut de l'âge de lecture réglé et sorti des
  blocages. Une lacune neuve consignée à sa place (22,8 % de fiches sans ligne d'inventaire, sans
  marqueur).
- **2026-07-28 (soir)** — le POC entre dans l'app : sol tramé, marqueurs joueur complets,
  identité (xuid + gamertag lus DANS le film), colonnes d'équipe et fiches joueur. Le fil des
  éliminations, l'inventaire et les effets restent à faire. Revue visuelle différée.
- **2026-07-28 (nuit)** — étapes 1 à 5 du plan de fiabilisation closes. Le rattachement passe du
  vote à la lecture du fil des morts (475 tirs sur 519, contre 398) — **le vote est supprimé, pas
  mis en secours** ; plus aucun rejet silencieux
  (invariant testé) ; la couverture et un verdict de publication sont publiés dans l'artefact ET
  au bandeau du POC ; `PlayerIndex` renommé `FilmIndex` avec garde-rail. **Étape 1 échouée, son
  échec est mesuré** : `i21` est inatteignable tant que le parcours delta dérive.
