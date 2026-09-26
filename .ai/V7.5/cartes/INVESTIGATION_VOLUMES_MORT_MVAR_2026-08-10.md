# INVESTIGATION — les volumes de mort (mort instantanee / compte a rebours) (2026-08-10)

Suite de `INVESTIGATION_SDDT_PFND_2026-08-10.md` (meme jour, meme seance etendue).
Point de depart : le temoin utilisateur — sur Dynasty, une fontaine semi-enterree AU
MILIEU de la carte, atteignable, declenche une mort instantanee. Donc des volumes de
mort fins existent, et ils ne sont ni dans sddt (verdict phase 1) ni grossiers.

**VERDICT : TROUVES pour les cartes FORGE (dans le .mvar) ; NON LOCALISES pour les
cartes integrees (toutes les portes disque connues sont fermees).**

## 1. La cle : type_id du .mvar = GlobalID d'un tag `food`

Les objets du .mvar portent un `type_id` int32 jamais resolu jusqu'ici. Mesure :
**20/20 type_id testes se resolvent comme GlobalID de tags du groupe `food`**
(Forge object definition) dans `any/globals/forge/forge_objects-rtx-new.module`
(40 039 tags). C'est la jointure objet-place -> definition d'objet.

Les tags food eux-memes sont des coquilles generiques de 1 388 o, identiques a un
hash pres (+0x08 du root, non craque — ni murmur3 du nom snake_case ni GlobalID).
L'identification est donc passee par l'empreinte FONCTIONNELLE.

## 2. L'empreinte fonctionnelle — 101 cartes Forge mesurees

Sur les 101 variantes `*_map.mvar` de `.ai/re_dump/mapvar/` (dump existant du
chantier objectifs), statistiques par type d'objet A FORME (boite/cylindre) :
nombre de cartes porteuses, aire, ecart d'altitude au sol median de la carte.

| type_id | cartes | objets | aire med (m2) | dz med (m) | lecture |
|---|---|---|---|---|---|
| -588988541 | 61 | **61 (1/carte)** | 2 206 | **-5,5** | LE volume de mort principal sous la carte |
| 176825834 | 57 | 882 | 4 725 | **-5,1** | planchers de mort additionnels |
| 937132837 | 52 | 191 | 800 | **-6,7** | volumes de mort sous-sol |
| -1751270658 | 50 | 828 | 100 | **+8,0** | murs/plafonds de limite en hauteur |
| 43333489 | 61 | 582 | 136 | +0,5 | zones aux abords (85x96 m hors arene sur Dynasty) |
| -696190206 | 73 | 2 960 | 37 | 0,1 | petits volumes colles aux structures (profil spawn/influence, PAS mort) |
| 95146865 | 88 | 373 | 1 078 | +2,3 | zones de mode (labels ctf_include etc.) |

**Le temoin de la fontaine COLLE** : sur Dynasty, `-588988541` est une boite de
23 x 43 m centree en (0,5 ; 128,9 ; 72,6) — en plein centre de la carte (centre du
nuage d'objets : (3,6 ; 128,4)), ~7 m sous le sol median (79,8). Unique sur la
carte, comme sur les 60 autres.

Ces noms sont des lectures fonctionnelles, pas des noms officiels : le nom du tag
food n'est pas resolvable sur disque (hash non craque). La distinction mort
instantanee vs compte a rebours entre ces types reste a etablir (il faudrait un
rejeu sur carte Forge avec morts hors-combat horodatees).

## 3. Cartes integrees : toutes les portes disque fermees

- `.mvar` de Cliffhanger (les deux variantes) : UNIQUEMENT des zones de mode
  (ctf/strongholds/extraction/firefight/minigame), aucun volume de mort.
- `sddt` : la coquille grossiere (phase 1) et la riviere. Rien d'autre.
- `scnr` : **n'existe pas dans Halo Infinite** — zero tag scnr dans TOUTE
  l'installation (balayage complet des .module).
- `stse` : c'est l'ENVELOPPE CONVEXE de la structure (8 sommets = les coins exacts
  de la boite du bsp, blocs de 16 o = ses plans, listes 12 o = normales). Pas une
  lisiere.
- `dwsg`, `hlds`, `cage`, `sred`, `snad` : balayes (triplet-en-boite par bloc) ;
  snad porte des positions (17x24, 36x40 — probablement du son ambiant), rien au
  profil volume de mort.

Les volumes fins des cartes integrees (compte a rebours de bordure) restent non
localises : ils peuvent vivre en donnee compilee non-tag ou dans un groupe non
encore interprete. Aucun candidat nomme restant.

## 4. Consequences pour le fond de carte

- Cartes INTEGREES (Cliffhanger, Catalyst — celles rendues aujourd'hui) : rien de
  neuf pour reduire l'exces ; le verdict phase 1 tient (grain 0,005 = meilleure
  regle, coquille sddt = cadrage).
- Cartes FORGE (~100 variantes du catalogue) : le jour ou on les rend (canevas
  fo* + objets du .mvar), les volumes ci-dessus donnent le decoupage : exclure la
  matiere dans les volumes de mort, cadrer par leur enveloppe. Les donnees sont
  deja locales (`.ai/re_dump/mapvar/`) et le parseur (`mapvar.Parse` + `Shape()`)
  est en prod.

## 5. LA RIVIERE (rappel utilisateur — a integrer au rendu)

Retour utilisateur du 2026-08-10 : « pour la riviere en effet c'est bien, elle
manquait ». L'eau n'est PAS dans les instances de rendu — c'est pour ca qu'elle
manquait au fond de carte. Elle est dans `sddt` (233 prismes sur Cliffhanger,
positions et emprises exactes, lecture etablie phase 1). A dessiner comme bande
d'eau sur le fond de carte : petit travail, donnees pretes, lecteur ecrit
(`litSddt` dans les sondes). Candidat naturel pour un prochain lot cartes.

## Sondes (non commitees)

`internal/himap/sonde_food_gamefiles_test.go` (deps des tags food),
`sonde_scnr_gamefiles_test.go` (balayage groupes suspects),
`sonde_stse_gamefiles_test.go` (enveloppe convexe). L'analyse trans-cartes des
.mvar a tourne via un cmd temporaire supprime apres mesure (resultats ci-dessus) ;
elle se rejoue en quelques lignes avec `mapvar.Parse` sur `.ai/re_dump/mapvar/`.
