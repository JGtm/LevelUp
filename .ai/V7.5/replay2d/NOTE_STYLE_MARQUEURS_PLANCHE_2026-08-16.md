# Note — le style de marqueurs VALIDE sur la planche du 2026-08-16 (item A1)

> Verdict utilisateur : « Parfait. Je veux exactement ce style pour les points, la croix de
> mort et la trainee. L'icone de visee je veux celui-la mais un peu plus prononce, juste un
> peu. Je prefere ce rendu a l'actuel. » Cette note donne les PARAMETRES EXACTS du schema de
> la planche (canvas, pixels d'ECRAN — multiplier par le DPR comme le fait `replayMarkers.ts`).
> Elle est destinee au plan d'habillage (`PLAN_HABILLAGE_REJEU_2D.md`, autre session), qui
> traite deja « diminuer la taille des points, supprimer le baton, pseudo sous les points ».
> Aucun hex ci-dessous n'est a recopier dans `features/` : ce sont des tokens a resoudre.

## Le point (marqueur de vie)

- disque plein, rayon **3,4 px** ; couleur = couleur de trace du joueur (token de palette).
- **marqueur d'etage** : anneau de rayon **6,5 px**, trait **1 px**, encre de lisibilite du
  theme (`--foreground` en sombre), alpha 0,9 — un seul anneau sur la planche ; garder la
  regle actuelle du nombre d'anneaux par etage si elle existe.
- pas de « baton » (le trait qui sortait du point est supprime — demande d'habillage).

## La trainee (7 s)

- polyligne des positions, trait **1,6 px**, `lineCap round`, couleur de trace ;
- alpha qui **monte vers le present** : de 0,08 (queue) a ~0,63 (tete), lineaire sur la
  fenetre de 7 s (sur la planche : alpha = 0,08 + 0,55 x (i / n)).

## Le cone de visee (maintenu 5 s) — « un peu plus prononce » que la planche

- secteur circulaire de rayon **46 px** (planche) -> **~52 px** ; demi-ouverture **0,38 rad**
  (planche) -> **~0,42 rad** ; degrade RADIAL de la couleur de trace au centre vers
  transparent au bord ; alpha global **0,45** (planche) -> **~0,55**. C'est ce « un peu plus »
  qui est demande : rayon +13 %, ouverture +10 %, opacite +0,10 — a valider a l'oeil, pas au
  chiffre.
- fraicheur : le cone se degrade quand la lecture de regard vieillit (regle actuelle
  conservee).

## La croix de mort (1,5 s)

- deux diagonales, demi-taille **5 px**, trait **1,6 px**, `lineCap round`, couleur de trace
  du mort (ou encre destructive selon la regle actuelle — la planche utilisait le rouge
  d'equipe adverse), alpha 0,9.

## L'anneau d'apparition (0,8 s)

- cercle qui s'ouvre de 2 -> 14 px, trait 1,2 px, alpha 0,8 -> 0, couleur de trace.

## Ce que la planche NE montrait pas et qui reste a la regle actuelle

Le nombre d'anneaux par etage, la couleur exacte des traces (tokens de palette), le cerne des
libelles. La planche est un SCHEMA valide comme style, pas un rendu de production : le plan
d'habillage transpose ces valeurs dans `replayMarkers.ts` et les soumet au gate visuel.
