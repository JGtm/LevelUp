> ⛔ **CONSOMMÉ LE 2026-08-08 — NE PAS REPARTIR DE CE DOCUMENT.** Le lot qu'il ouvrait a été
> exécuté (timebox de 2 sessions, décision #6 du master plan) et son **verdict est NÉGATIF**.
> Plusieurs pistes qu'il nomme comme ouvertes sont désormais **fermées par mesure** : le
> compteur de tir comme source des tirs de projectile, les records de dégât comme porteurs des
> touches, et l'enregistrement créateur du slot `objet + 0x114`.
>
> **Aller à `.ai/HANDOFF_PRECISION_PROJECTILES_2026-08-08.md`** (état, ce qui reste ouvert,
> prochain geste) et `.ai/V7.5/VERDICT_PRECISION_PROJECTILES.md` §0bis (le verdict en une page).
>
> Ce qui reste juste ici : le **plafond de validation** (pas de population mono-arme pour le
> Needler — la validation passe par le contraste intra-joueur) et les **quatre interdits de
> méthode** de la section « CE QU IL NE FAUT PAS REFAIRE ». Le reste est daté.

# HANDOFF — rendre la precision universelle (les armes a projectile)

> Ecrit le 2026-07-31, en fin de contexte. **Lot CIBLE, pas un chantier** : une question, une cible,
> un verdict binaire au bout. Il ne bloque ni le branchement ni rien d autre.

## L ETAT EN TROIS PHRASES

La precision par joueur **marche** : taux de remplissage `porteurs / records`, 0,4267 contre 0,4462
a l API, r = +0,82, **sans aucune reference externe**. Par arme, elle marche sur les armes a **tir
instantane** et donne **4 armes publiables** a +-0,03 (MA40 +0,0249 · BR75 -0,0108 · Sidekick
-0,0050 · Bandit Evo +0,0132), sur les joueurs dont l arme domine >= 80 % des tirs. Elle est
**fausse d un facteur 30 a 60 sur les armes a projectile** — Needler 0,007 pour un taux reel ~0,39.

## POURQUOI ELLE ECHOUE — la cause est LUE, pas supposee

Le compteur de touches du jeu (`FUN_1408df6a4`) a **DEUX appelants** :
- `FUN_1404d8600` = application de degat (groupe `jpt!`) -> ecrit dans l enregistrement du TIR
- `FUN_142f1c44c` = **IMPACT DE PROJECTILE** (groupe `proj`) -> se declenche PLUS TARD

Un projectile part, l impact arrive apres : le porteur de l enregistrement de tir ne peut donc pas
le voir. **Ce n est pas un defaut de lecture, c est le format.**

## OU SONT CES TOUCHES — elles SONT dans le film

Evenements de **code 6** (impact sur la geometrie) et **code 7** (impact sur une ENTITE) :
80 886 et 129 390 sur 949 films, 80 % en premier evenement du paquet.
Gain LOCALISE, mesure : correlation +0,7675 aux tirs de projectile contre **-0,1929** aux tirs
hitscan ; **39 des 65 films sans tir de projectile portent EXACTEMENT ZERO code 7** ;
`(code6 + code7) / tirs de projectile = 0,9831` — un impact par tir, sans ajustement.

## LES DEUX VERROUS, ET ILS SONT BIEN MESURES

**1. L evenement ne nomme JAMAIS le tireur.** Il donne la CIBLE et l OBJET PROJECTILE.
Et il n existe **aucun evenement de creation** de projectile : la couverture des deux codes somme a
**1,0014** — une partition mesuree, donc l instrument reconnait ce qui nomme un projectile et n en
trouve pas de troisieme. Les trois voies sont fermees :
- creation : aucun evenement (couverture 1,0014)
- parente (`object-parent-state-component`, i10) : designe semantiquement la VICTIME (aiguille
  plantee, grenade collee), et l archetype projectile `ti=41` ne replique aucun proprietaire sur ses
  22 composants
- appariement temporel : **plafond 0,41** d impacts a candidat unique — et ce serait de toute facon
  le motif << same-clock >> deja formellement invalide sur ce chantier

**2. Le tag n est PAS transmis** dans le corps du code 7. Le bit qui commande sa lecture vaut 1 sur
**168 380 observations de profondeur 0, ZERO exception** (949 films). Controle positif du meme
instrument : en reecrivant le corps avec un vrai tag du catalogue, il le relit a **1,0000**. Ce n est
donc pas une cecite d instrument.

## LA SEULE PISTE QUI RESTE, ET ELLE EST NOMMEE

Le handle porte par le code 7 est un **SLOT DE REPLICATION identifie** : `objet + 0x114`, le meme
champ qui sert de porte de deduplication dans `FUN_142f1c44c`.

**La question devient donc : QUE PORTE L ENREGISTREMENT QUI CREE CE SLOT ?**

C est neuf : jusqu ici on cherchait un evenement de creation *quelque part* ; maintenant on connait
le SLOT, et on peut chercher son enregistrement d apparition. Point de depart : `FUN_141fd8460`
serialise ce champ, et la table de dispatch a **123 codes** dont on connait le decoupage.

## LE VERDICT ATTENDU, ET SON CRITERE

Si le tireur devient rattachable :
- compte de touches par joueur = hitscan (drapeau `+106`) + projectile (code 7 **DEDUPLIQUE PAR
  OBJET-SOURCE** — une roquette qui blesse trois joueurs vaut UNE touche, pas trois)
- **CRITERE** : le nombre de joueurs dont le compte EGALE `shots_hit` de l API **A L UNITE**,
  compare a ce que rend un taux constant. Une erreur moyenne ne prouve rien.
- **ET LE GAIN DOIT ETRE LOCALISE** : il doit apparaitre chez les joueurs a forte part de projectile
  et **ne rien changer** aux joueurs hitscan. Un gain uniforme est un facteur d echelle deguise.

## LE PLAFOND QU AUCUN DECODAGE NE RETIRE

Meme si le decodage reussit, **VALIDER** un taux par arme demande des joueurs mono-arme. Le Needler
n a que **2 observations** a >= 80 % de purete : personne ne joue un match entier au Needler.

**LA PARADE EXISTE ET ELLE A DEJA MARCHE** : le **contraste INTRA-JOUEUR** — deux armes, meme
joueur, meme match — qui a rendu 0/200 en permutation et MA40 contre Sidekick a z = +17. Il ne
demande aucun joueur mono-arme. **C est par la qu il faudra valider, pas par la population a arme
dominante.**

## CE QU IL NE FAUT PAS REFAIRE

- **Aucun nouveau dump** : 949 films, 2,2 M d evenements de tir, mesures faites sur 892-949 films.
  Le corpus n est pas la limite.
- **Aucun balayage bit a bit par position** : ~99,85 % de faux positifs, et il FABRIQUE des
  distributions credibles. Interdit, pas deconseille.
- **Ne pas conclure sur une mediane** : un facteur d echelle reproduit n importe quelle mediane.
- **Ne pas se fier a une reproduction hors echantillon seule** : une procedure biaisee reproduit son
  biais (mesure : 11/12 hors echantillon, puis la meme procedure sur un bloc de largeur constante
  par construction rend 15/18 avec un argmin FAUX).

## CE QUE CA VAUT SI CA MARCHE

Les armes a projectile sont une grosse part de l arsenal, et en Fiesta elles sont l essentiel de ce
qu on ramasse. La precision passerait de 4 armes a la quasi-totalite, et la Fiesta — aujourd hui
NON livrable, 6,9 % des joueurs dans la tolerance — redeviendrait exploitable.
