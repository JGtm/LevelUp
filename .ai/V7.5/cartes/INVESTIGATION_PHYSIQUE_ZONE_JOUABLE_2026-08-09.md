# INVESTIGATION — la zone jouable se lit-elle dans la COLLISION ? (2026-08-09)

**VERDICT : NO-GO comme delimiteur de zone jouable — porte fermee par la mesure.**

> CONTRE-VERIFICATION DU SUPERVISEUR (2026-08-09, meme jour). Rejoue depuis le worktree :
> stride 192 confirme par l'arithmetique sur les DEUX cartes (754 176/3 928 et 1 421 184/7 402,
> reste 0) ; la table de l'oracle reproduit A LA DECIMALE (95,21 / 93,82 / 95,21 / 94,67 /
> 92,95 %). Verdict NO-GO confirme — et il tient meme sans l'oracle : sur 10 357 instances de
> rendu, 9 454 portent de la physique (91 %), donc un tri sur « a de la collision » ne peut
> arithmetiquement pas resorber 149 % d'exces.
>
> DEUX RESERVES A GARDER :
>
> 1. **Le temoin d'offset NE SEPARE PAS.** Le scan trouve un « triple unitaire » a 100 % sur
>    CINQ offsets (0x48, 0x78, 0x7c, 0x80, 0x84), pas au seul nominal — l'affirmation
>    « 0,0 % a +4 octets » est contredite par 0x7c. Le decodage est peut-etre juste, il n'est
>    PAS etabli par ce temoin. Ne pas traiter ces offsets comme acquis sans un temoin qui
>    departage — c'est exactement le piege de T1 (handoff §8), paye trois fois.
> 2. **Le document de rejeu `data/cache/replays/halo_infinite/000d5950.json` n'existe pas dans
>    le worktree** : `data/` est ignore par git et un worktree ne partage pas les fichiers non
>    suivis. `TestSondePhysiqueRendu` s'y declare SKIP EN SILENCE — un oracle absent qui passe
>    au vert. Le copier depuis le depot principal avant toute mesure d'oracle ici.
Le bloc `instanced physics instances` se decode integralement (acquis reutilisable), mais
le decor porte lui aussi de la collision joueur : les parois et rochers arretent balles et
Spartans exactement comme le sol de l'arene. « A une collision Play » ne separe pas
« praticable » de « mur ».

Sondes : `apps/go-api/internal/himap/physique_sonde_gamefiles_test.go` (non commite,
3 tests : `TestSondePhysique`, `TestSondePhysiqueRendu`, `TestSondePhysiqueResolution`).

## 1. Le bloc se decode — structure verifiee

Stride REEL : **192 o (0xC0)**, la somme des champs du plugin donne 176 — les 16 octets
excedentaires sont du padding `0xBC` en fin d'enregistrement, apres `m_scene`. Preuve du
stride : tailles de bloc 754 176 (cliffhanger) et 1 421 184 (catalyst) divisees par les
comptes du root block (3 928 et 7 402) donnent toutes deux exactement 192.

Offsets confirmes (appariement par rang du plugin, `blockByPluginName`) :

| champ | offset | verification |
|---|---|---|
| `m_collisionTagReference` | 0x00 | GlobalID a +8, 100 % resolvable (voir §2) |
| `Scale` | 0x3C | valeurs metriques plausibles (0,5 a 9 m) |
| `Forward/Left/Up` | 0x48/0x54/0x60 | base orthonormee sur 100,0 % des 3 928 + 7 402 instances ; **mutation jouee** : +4 octets -> 0,0 % |
| `Position` | 0x6C | 3 902/3 928 et 7 402/7 402 dans la boite du bsp (marge 10 %) |
| `m_typeMask` | 0x78 | 3 valeurs distinctes, 0 hors des 4 drapeaux declares (Bullet/Play/InvisibleWall/Render) |
| `m_guid` | 0x7C | s'apparie a l'`external_guid` (@0x12C) des instances de rendu |
| havok body IDs | 0x80 | tous 0xFFFFFFFF sur disque (rempli au runtime) |

Piege d'instrument paye : le scan d'orthonormalite comptait les NaN (0xFFFFFFFF du tableau
havok) comme unitaires — `math.Abs(NaN-1) > seuil` est FAUX. Faux offsets candidats 0x78-0x84
expliques, seul 0x48 est reel.

## 2. Vers quoi il pointe

Groupe **`scgt`** (static collision geometry). **552/552** references distinctes de
cliffhanger se resolvent sur les 132 modules de l'installation (l'index habituel
carte+globals n'en voit que 195 — les scgt vivent ailleurs). Le FORMAT scgt reste non
decode : la lecture float32 brute est le faux positif deja documente (95-97 % des points
sur une croix a l'origine) et Reclaimer ne le decode pas non plus (ses sources ignorent
totalement ce bloc — verifie dans `ScenarioStructureBspTag.cs`).

## 3. L'emprise 2D — la voie qui marche sans scgt

Pas besoin du maillage de collision : `m_guid` -> `external_guid` apparie chaque instance
physique a son instance de RENDU, qu'on sait deja rasteriser. Couverture :

    carte        rendu     avec physique   avec physique Play
    cliffhanger  10 357    9 454 (91 %)    7 881 (76 %)
    catalyst     11 468    10 482 (91 %)   10 112 (88 %)

typeMask : cliffhanger 3 357 x Play+Bullet+InvWall+Render, 564 sans Play, 7 sans Bullet ;
catalyst 7 244 / 157 / 1. Tout le monde porte InvisibleWall — ce drapeau ne separe rien.

## 4. Verdict de l'oracle (29 221 positions, reference validee, cadre du gate)

Sur les 5 961 instances dessinables de cliffhanger (regles validees : tous modules,
tranche [-12;+28], echelle, ombres ecartees) :

    filtre          gardees   manquants   exces     ACCORD   oracle strict   a 1 px
    aucun            5 961      4,0 %     149,3 %   38,5 %     95,21 %       96,78 %
    grain 0,005      5 102     10,8 %      33,8 %   66,7 %     93,82 %       96,13 %
    physique-any     5 498      4,0 %     129,3 %   41,9 %     95,21 %       96,78 %
    physique-play    4 690      4,7 %     129,0 %   41,6 %     94,67 %       96,32 %
    play-et-grain    3 961     12,2 %      30,7 %   67,2 %     92,95 %       95,43 %

- `physique-any` ne perd RIEN (positions et manquants identiques au sans-tri) mais ne
  retire que 20 points d'exces sur 149 — tri « gratuit » mais tres faible.
- `physique-play` perd 158 positions jouees : perdant par le critere annonce.
- `play-et-grain` gagne 0,5 point d'accord sur le grain seul mais perd 254 positions
  jouees : perdant aussi.
- Catalyst : Play garde 88 % des instances — incapable de resorber l'exces architectural
  qui y a refute le grain.

## 5. NO-GO — ce qui bloque precisement

Le masque de collision dit « le joueur ENTRE EN COLLISION avec ceci », pas « le joueur
PEUT MARCHER ici ». Falaises, parois et decor proche portent Play au meme titre que le
sol : sur 5 961 instances dessinables, seules 463 n'ont aucune physique et 1 271 n'ont
pas Play — l'exces de 149 % vit presque entier dans de la geometrie a collision.
C'est une propriete du jeu, pas un defaut de decodage : aucun reglage du filtre ne peut
extraire du bloc une information qu'il ne porte pas.

## 6. Portes fermees (ne pas rejouer)

1. **Presence de physique = zone jouable** : exces 149,3 -> 129,3 %, accord 38,5 -> 41,9 %
   (le grain fait 66,7 %).
2. **Drapeau Play = zone jouable** : exces 129,0 %, ET perd 158 positions jouees.
3. **Play en complement du grain** : +0,5 point d'accord contre 254 positions perdues.
4. **InvisibleWall comme frontiere** : 100 % des instances physiques le portent.
5. **Havok body IDs** : 0xFFFFFFFF sur disque, aucune information.
6. **Resoudre scgt via carte+globals** : 195/552 seulement — il faut l'installation
   entiere (132 modules), et meme resolu le format reste a decoder (faux positif float32
   documente). Sans promesse pour la ZONE : la collision inclut les murs par nature.

Reste ouverte (handoff §10.10) : l'ACCESSIBILITE — composante connexe de sol praticable
atteignable depuis les ancres. Le present negatif la renforce : c'est la connexite au
point de depart, pas la matiere, qui distingue l'arene du decor.
