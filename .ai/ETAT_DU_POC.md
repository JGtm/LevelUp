# ÉTAT DU POC DE REJEU 2D — ce qu'il contient VRAIMENT

> Mis à jour le 2026-07-27. **À relire avant d'annoncer qu'une chose « est dans le POC ».**
>
> Ce document existe à cause d'un quiproquo réel : j'ai annoncé « structure livrée » alors que
> le POC n'affichait que des BOÎTES ENGLOBANTES, et l'utilisateur cherchait la carte qu'il avait
> validée. Puis j'ai conclu à tort que la carte concernait un AUTRE terrain. Les deux erreurs
> viennent de la même cause : aucun document ne disait, calque par calque, ce qui est affiché et
> d'où ça vient.

## CE QUE LE BANDEAU DIT MAINTENANT — ajouté le 2026-07-28

Le bandeau « Données » ne publie plus un numérateur seul. Trois tuiles ont changé :

| tuile | valeur | ce qu'elle empêche |
|---|---|---|
| tirs placés / lisibles | **475 / 519** | croire à l'exhaustivité d'un calque partiel |
| lancers placés / lisibles | **63 / 70** | idem |
| verdict de publication | **nominal** | publier un calque dont on ne sait pas ce qu'il a perdu |

L'infobulle du verdict ventile les rejets : slot introuvable 44, slot ambigu 0, hors fenêtre 0,
sans trajectoire publiée 0. **La somme fait exactement 519** — l'invariant est testé côté Go.

**Le dénominateur est le nombre de records que LE FILM PORTE, pas le nombre de tirs du match.**
Le film ne sérialise pas les tirs manqués : 519 records pour 2 228 tirs effectués et 595 touchés.
Confondre les deux ferait passer un plafond de format pour un échec de décodage.

## LE PIÈGE DE VOCABULAIRE À NE PLUS REFAIRE

| Le mot | Ce qu'il désigne dans le POC | Ce qu'il NE désigne PAS |
|---|---|---|
| « structure » | 9 630 **boîtes englobantes** (AABB) d'instances de géométrie | la carte reconstruite depuis les triangles |
| « la map » | ambigu — TOUJOURS préciser : boîtes, sol reconstruit, ou rendu PNG | — |
| « grenades » | 27 **lancers** localisés dans le temps et l'espace | l'inventaire porté (types équipés, comptes) |
| « loadouts » | 150 relevés d'**armes** issus des keyframes | les grenades ou capacités équipées |

## LE PIÈGE DE NOM DE CARTE — il m'a fait conclure faux

**`Cliffhanger` (nom affiché) et `ridgeline` (nom du module) sont LA MÊME CARTE.**

Le document de rejeu porte les deux : `"map":"Cliffhanger"` et `"mapFolder":"ridgeline"`. Tous
les artefacts de géométrie sont nommés `*_ridgeline.*`. J'en ai déduit qu'ils concernaient un
autre terrain et je l'ai annoncé à l'utilisateur — **c'était faux**. Le contrôle qui tranche en
deux secondes : les callouts du POC contiennent « Fer à cheval » et « Pont », qui sont
précisément les zones de la carte étudiée.

## CALQUES PRÉSENTS — provenance et limites de chacun

| Calque | Contenu | D'où viennent les données | Défaut par défaut |
|---|---|---|---|
| Carte validée (fond) | l’IMAGE validée le 26/07, extraite de l’artefact, recadrée | `.ai/re_dump/carte_validee_v1.png` — calage ajusté sur les trajectoires : échelle 0,0920 m/px, X0 -43,5, Y1 61,0 ; **95,6 % des 29 217 positions de joueur tombent sur le sol** | **actif** |
| Boîtes englobantes | 9 630 AABB, ou 6 182 emprises orientées | `cmd/mapstruct-build` — AABB monde @0x7C, **aucune forme** | masqué quand le sol est actif |
| Trajectoires | 98 pistes de joueur | `i0` du film, décodé | actif |
| Tirs | **475** événements localisés (sur **519** que le film porte) | `FireEvent` + pont **fil des morts** (`replay/lives.go`) ; le vote visée→slot est SUPPRIMÉ | actif |
| Effets d'arme (bouton dédié) | l'éclat d'un tir et l'effet d'une mort prennent la forme de la FAMILLE de l'arme : balistique, plasma, dur-lumière, choc, explosif, mêlée, aiguilles, plus un repli `sobre`. Sur une MORT, l'effet est orienté tueur→victime | mesuré : le libellé de l'arme (`shots[6]`, 147/147 ; `feed[].w.n`, 87/93, les 6 autres par la nature `w.cl`). Parti pris NON mesuré : le regroupement en familles et l'aspect. Direction : positions des deux joueurs relues dans les trajectoires, **89 des 93 morts** ont le couple complet — les 4 autres n'ont aucun axe. Sous `prefers-reduced-motion`, marqueur statique | actif |
| Lancers de grenade | **63** sur 70 décodés | marqueur `0x4C0C00` ; le plafond du pont slot→joueur est levé | actif |
| Projectiles | 439 trajectoires | archétype `ti=41` | actif |
| Zones de callout | 16 libellés FR, 25 px blanc cerné de noir | `.ai/re_dump/callout_*` | actif |
| Objectifs | drapeau, crâne, zones | terrain, **pas** ce match (Slayer) | masqué |
| Bouclier | barre au-dessus du marqueur | `i5` | actif |

## CE QUI N'EST PAS DANS LE POC, ET POURQUOI

| Absent | Raison exacte |
|---|---|
| **Inventaire équipé** (grenades portées, capacité) | `i47` et `i48` sont décodés, mais **seulement via la capture Cheat Engine** : la lecture hors ligne dépend d'un scan dont la précision (5,5 %) ou le rappel (24 %) est insuffisant selon le gabarit employé. Rien de publiable. |
| **Arme en main** | `i43` a une forme longue de ~200 bits non décodée. `i42` (7 valeurs distinctes) est décodable mais non câblé. |
| **État actif des capacités** | `i57` est mesuré (bit 0 = interrupteur, 990 bascules) mais non câblé dans `consumeByName`. |
| **Nom des capacités** | impossible hors ligne par les tags (`fileNameSize = 0` en build release). Seule voie : banques Wwise, non tentée. |
| **43 lancers de grenade sur 70** | plafonnés par le pont slot→joueur (50 slots liés sur 99). |
| **Mêlée** | recette documentée fiable depuis juin, jamais implémentée. |

## RÈGLES POUR LA SUITE

1. **Ne jamais écrire « X est dans le POC » sans avoir ouvert le fichier et vérifié le calque.**
   Le contrôle coûte trente secondes : chercher la fonction de dessin, pas la clé de données.
   Une donnée embarquée mais jamais tracée est le cas exact qui s'est produit avec `spoly`.
2. **Toujours nommer la carte par ses DEUX noms** quand un artefact est en jeu.
3. **Distinguer « décodé » de « publiable ».** Un composant lu grâce à la capture Cheat Engine
   n'est pas exploitable en production : les 948 autres films n'ont pas de capture.
4. **Mettre ce fichier à jour dans le même commit** que tout ajout ou retrait de calque.

## OÙ SONT LES CHOSES

    <scratchpad de la session>/replay_demo.html   le POC (à ÉDITER, jamais à régénérer)

**LE CHEMIN DU POC CHANGE À CHAQUE SESSION, et c'est un vrai problème.** Le fichier vit dans le
scratchpad de la session qui l'a produit ; une nouvelle session ne le retrouve qu'en cherchant.
Au 2026-07-28 il est dans
`…/Temp/claude/c--…-filmdec-continuation/ca29e2c3-0a78-4cab-bb19-841af338b5fc/scratchpad/`.
C'est exactement ce que l'ÉTAPE 6 du plan doit supprimer, en portant les calques dans
`features/match-replay`. En attendant, **le retrouver avant d'annoncer quoi que ce soit** :

    find /tmp/claude -name replay_demo.html

**Mise à jour des données** : ne pas rééditer le bloc à la main (2,9 Mo sur une ligne). Deux
commandes, depuis `apps/go-api` :

    CGO_ENABLED=0 LEVELUP_REPO_ROOT=<worktree> go run ./cmd/replay-build --map Cliffhanger 000d5950 <filmDir>
    CGO_ENABLED=0 go run ./cmd/tmp_pocupdate -poc <html> -artifact <worktree>/data/cache/replays/halo_infinite/000d5950.json

    scratchpad/sc/floor_poc.png          le sol reconstruit, recadré et géoréférencé
    scratchpad/sc/layers_marchable.npy   la source : cellules praticables par tranche d'altitude
    scratchpad/sc/layers_meta.npy        CELL=0,10 · origine (-45, -60) · grille 1200x1200
    scratchpad/structure_callouts.png    le rendu validé en V1 le 26/07
    .ai/HANDOFF_GEOMETRIE_TRIANGLES.md   la recette complète, à porter en Go
    .ai/ADDENDUM_ETAT_DE_L_ART_*.md      tout ce que la capture du dispatch a tranché
