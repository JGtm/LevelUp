# Handoff — chantier des fonds de carte, etat au 2026-08-27 19h30

> Ecrit pendant que le chantier Isolation tourne. Tout ce qui n'est PAS Isolation est ici, avec
> son etat exact, ce qui bloque, et la premiere action a faire. Le registre
> `REGISTRE_CARTES.md` reste la source des verdicts carte par carte ; ce fichier est la carte
> des chantiers ouverts.

## 0. Ou en est la production

| | nombre |
|---|---|
| fonds publies | **84** |
| cartes closes avec verdict utilisateur | 11 natives + le lot des 18 Forge |
| cartes du jeu recensees (enquete du 27/08) | ~164 |
| variantes `.mvar` en depot local | **257** (199 avant le 27/08) |

Branche de travail : `wt/cartes-revue-par-carte`, poussee. Dernier commit de reference :
`1abbdf772`.

## 1. Les 29 variantes telechargees le 27/08 — A INGERER PUIS CUIRE

Telechargees par la voie ANONYME (aucun jeton) : page publique de l'asset ->
bloc `__NEXT_DATA__` -> `Files.Prefix` -> `prefixe + map.mvar`. Le script est
`scratchpad/telecharge.sh` ; il garde AUSSI le fichier-lien `<canevas>.mvar`, qui nomme le
canevas dont la declaration a besoin.

**PIEGE PAYE** : un asset Forge porte DEUX `.mvar`. `map.mvar` est la carte (300 Ko a 1,6 Mo) ;
`<canevas>.mvar` est un lien de ~17 Ko qui ne fait que nommer le canevas. Le premier passage a
rapporte 32 fichiers de 17 Ko — les liens. Ne jamais prendre le premier `.mvar` de la liste.

Cartes recuperees, avec leur canevas :

| carte | canevas | | carte | canevas |
|---|---|---|---|---|
| Interference | fo13_frost | | Cliffside (2e asset) | fo05_desert |
| Apostle | fo11_blank | | Critical Dewpoint (2e) | fo11_blank |
| Aqueduct | fo03_space | | Domicile (2e) | fo05_desert |
| Arrival | fo03_space | | Dynasty (2e) | fo08_wetland |
| Daimyo | fo11_blank | | Empyrean - Ranked | fo11_blank |
| Dead Water | fo08_wetland | | Goliath (2e) | fo11_blank |
| Diminished | fo11_blank | | High Ground (2e) | **fo09_academy** |
| Exiled | fo05_desert | | Isolation (2e) | fo08_wetland |
| Kusini Bay | fo11_blank | | Perilous (2e) | fo08_wetland |
| Last Broadcast | fo08_wetland | | Rat's Nest (2e) | fo08_wetland |
| Vallaheim | fo05_desert | | Shiro (2e) | fo05_desert |
| Waterworks | fo08_wetland | | The Pit (2e et 3e) | fo09_academy |
| Suban | fo03_space | | Lattice | fo13_frost |
| Breaker / Cliffhanger / Scarr Firefight | natives | | | |

**Attention sur High Ground** : le 2e asset est sur `fo09_academy` quand le fond publie est sur
`fo08_wetland`. Canevas different = geometrie differente : il faut le CUIRE, pas l'aliaser.

**Marche a suivre, par carte** :

```
cmd/mapobj-build --from-file .ai/re_dump/mapvar/<slug>_map.mvar --map-id <uuid>   # hors ligne
# puis declarer dans internal/himap/cartes_forge.go (MapID, Nom, FichierMvar, ModuleCanevas)
# puis cuire : mapfond-build --natives=false --forge --maps <uuid>
```

Le garde-rail `TestFondForgeJamaisSousCleModule` exige qu'une carte declaree ait son fond
publie : declarer et cuire vont dans le meme commit.

## 2. Argyle et Detachment — DEBLOQUEES, rien a telecharger

Le registre les classait « canevas inconnu ». **C'est faux** : leur `.mvar` est en depot depuis
le 31/07. 47 matchs a debloquer sans reseau.

- Argyle : `dd600260-d91c-4d77-9990-3f35873c90a1`, canevas `fo11_blank`, 22 matchs.
- Detachment : `d39600e2-3c35-4a3a-bdf5-7b3cbdde98e1`, canevas `fo09_academy`, 25 matchs.

Meme marche a suivre qu'au §1.

## 3. Live Fire — la carte la plus jouee sans fond (71 matchs)

Deux verrous, aucun n'est une donnee manquante :

1. **Le choix du bsp.** Sa geometrie n'est pas dans son module (`sgh_interlock` : 6 fichiers, un
   `levl` de 2,3 Mo, aucun sbsp) mais dans `pc/globals/common-rtx-new.module`, qui porte QUATRE
   sbsp. Le premier — 12 556 instances, X [-16,7 ; +46,5], Y [-10,1 ; +53,7] — contient ses
   24 ancres. Le reglage `moduleGeometrie` existe et fonctionne, mais `ChoisitBSP` retient le
   bsp qui porte le PLUS d'ancres et deux des quatre les portent toutes : le mauvais est choisi
   (le rendu montre une antenne satellite et un escalier). **Critere a trouver** : a nombre
   d'ancres egal, prendre la plus petite emprise, ou l'altitude la plus proche du niveau de jeu.
   Verrouiller par un test qui verifie que Forbidden, Chasm et Catalyst choisissent le meme bsp
   qu'avant.
2. **La resolution par nom.** `live fire` est ABSENT de `map_quant_bounds.json` : meme cuite, la
   carte ne serait pas resolue. Publier sous la cle module `sgh_interlock` couvre ses 3 assets
   d'un coup.

Preuve rejouee : `TestGeometrieLiveFireDansCommon` (internal/himap).

## 4. La derive d'identifiant d'asset — 16 map_id joues sans fond, 131 matchs

Une carte peut etre jouee sous un GUID different de celui sous lequel son fond est publie. Par
nom elle semble presente ; au rejeu elle est sans fond. Cas mesures : Salvation, Dynasty,
Shogun, Houseki, Cole Protocol, Starboard, Shiro.

Les seconds assets sont maintenant TELECHARGES (§1) : soit on les cuit, soit on pose un index
`map_id -> cle de fond`. **La voie durable est l'index**, et elle doit s'accompagner du
garde-rail qui suit.

**Le garde-rail qui remplace la decouverte a l'oeil** : un test qui joint
`match_registry.map_id` sur `map_backgrounds/` et echoue sur tout map_id joue sans fond
resolvable, avec une allowlist explicite et datee pour les blocages connus.

## 5. Vacancy et ~65 autres cartes — identifiant inconnu

Vacancy, Serenity, Showdown Arena, Narrows, Guardian, Construct, Canopy, Ascension... sont
connues par leur NOM seulement. La route de telechargement part du `map_id` : sans lui, rien.

`/hi/search?assetKind=MapModePair` (authentifie) a rendu 104 cartes de la rotation, **Vacancy
n'y etait pas**.

**Action proposee, non validee** : ajouter a `mapobj-build` une RECHERCHE PAR NOM sur
`/hi/search`, avec les jetons du store (ADR 0023, jamais de re-capture). Cela resoudrait
Vacancy et la soixantaine d'autres, et deviendrait l'outil de veille.

Note : `944396dd-5661-4a16-b1d8-a6053f762c55` s'appelle **Narrows** (mesure du 27/08). Sa ligne
au registre et sa declaration doivent etre renommees.

## 6. House of Reckoning — native jamais cuite

Module `pve_house` installe, aucun blocage technique. Son asset UGC ne porte pas de `map.mvar`
(c'est une native). Il lui faut une entree dans `map_quant_bounds.json` puis une cuisson native
sous la cle `pve_house`.

## 7. Ce que Reclaimer a repondu, et qu'il ne faut pas re-instruire

- **Il ne lit AUCUNE geometrie de collision**, pour aucun titre Halo. `CollisionModel` n'est
  qu'une propriete declaree dans 6 fichiers, jamais suivie d'un decodage.
- Il ecarte, au niveau SECTION de maillage, les LOD et les shadow proxies
  (`HaloInfiniteCommon.cs:182`) ; nous ne le faisons qu'au niveau INSTANCE. **Non mesure chez
  nous** — c'est la seule piste Reclaimer qui reste ouverte.
- Le jeu porte un drapeau **`exclude from intel map`** (bit 12 du champ `Flags` a l'offset 0x78
  des instances, `sbsp.xml:720`). **Deja decode chez nous, jamais lu.** Il ne vaut que pour le
  chemin NATIF — les objets Forge ont leur propre champ de drapeaux, celui du §8.
- Le canevas Forge porte son terrain dans des `tccg` (1 024 cellules sur `fo08_wetland`), hors
  du chemin de rendu : `dessineCanevas` ne pouvait donc rien donner, ce que la mesure a
  confirme (PNG identique a l'octet).

## 8. Les leviers livres le 26 et 27/08, tous en donnee

Tous se declarent dans `data/titles/halo_infinite/reference/map_fond_reglages.json`, avec
`raison` (>= 80 caracteres) et `gateLe`. `TestReglagesFondJustifies` refuse une entree sans les
deux ou sans effet.

| levier | ce qu'il fait |
|---|---|
| `style`, `echelle` | habillage ; l'echelle est AUTOMATIQUE depuis le 26/08 (3 000 px de grille) |
| `ecreteToits` + `plafondArene` | vide les pixels dont aucune surface n'est a hauteur de jeu |
| `rogneAuxZones` + `margeZones` | masque des callouts (natives seulement) |
| `combleTrous` | aplat de sol suppose dans les trous FERMES |
| `boiteUtile` | rectangle monde, levier manuel |
| `plancherTranche` / `plafondTranche` | coupe basse / haute de la tranche de rendu |
| `substitutionSansPortee` | etend la substitution au-dela de la portee des ancres |
| `rogneAuxVolumesDeMort` | equivalent Forge du masque de callouts |
| `moduleGeometrie` | va chercher la geometrie dans un AUTRE module (Live Fire) |
| `typesExclus` | ecarte des types d'objet Forge |
| `drapeauxExclus` | ecarte des objets par leur champ de drapeaux (§9) |
| `plafondObjets` | ecarte un objet par l'altitude ou il est POSE |
| `solVuDuDessous` | retient la surface la plus BASSE au-dessus du sol joue |
| `minceurMin` | ecarte les modeles filaires (critere refute, garde par prudence) |
| `seuilArete` | denivele a partir duquel on souligne un bord |
| `dessineCanevas` | pose la geometrie du canevas sous la carte |

**Regle des deux regimes, a ne pas oublier** : le taux de couverture decide de l'ecretage. Sous
1/3 la voie de reference native ne se declenche jamais et l'ecretage est le seul chemin ;
au-dessus elle fait deja le travail et l'ecretage ne peut que retirer du sol.

## 9. Les deux instruments de diagnostic, a utiliser AVANT de regler

1. **« Qui peint l'image »** — le rendu retient le type d'objet qui gagne chaque pixel, et la
   cuisson Forge journalise le classement. C'est ce qui a designe le dome d'Isolation en une
   ligne apres deux jours de tatonnements. **A lire avant tout reglage sur une carte Forge.**
2. **Le champ de drapeaux des objets Forge** (champ 7 du `.mvar`, jamais lu avant le 27/08) :
   sur Isolation il vaut 21 pour 4 384 des 5 042 objets, et **1 pour les 32 pieces du dome**.
   C'est le seul critere qui ait separe le dome du reste — ni la categorie, ni l'emprise, ni
   l'aire de maillage, ni la couverture au sol n'y parviennent.

## 10. Ce qui a ete REFUTE et qu'il ne faut pas re-essayer

- Toutes les coupes en ALTITUDE contre la bouillie : la paroi d'un dome descend jusqu'au sol.
- Les criteres de forme portes par le MODELE (emprise, aire, couverture au sol) : le type
  coupable d'Isolation sort au rang 222 sur 271 et 145 sur 270 selon le critere.
- Les statistiques globales d'image (bruit de luminance, alignement des contours, couverture,
  densite d'objets) : aucune ne separe les cartes propres des cartes en bouillie.
- Le pelage par type au-dela du premier retrait : il coute une ancre d'objectif, donc du sol.
- `dessineCanevas` sur une carte batie AU-DESSUS de son terrain (Isolation) : PNG identique.

## 11. Incidents d'outillage a ne pas repayer

- **Arreter une tache de fond tue le processus enveloppe, pas la boucle** : trois boucles de
  cuisson ont tourne en parallele sur la meme file, la machine paginait. Le script porte
  desormais un verrou `mkdir`, un refus de demarrer si un `mapfond-build` tourne, et un `trap`.
- **La cuisson pagine au-dela de trois cartes par process** : cuire par LOTS DE 3, un process
  par lot. En une seule passe, le temps par carte passe de 137 s a 236 s.
- **Un filet anti-boucle qui regarde le code de sortie ne se declenche jamais** sur une carte
  incuisable, parce que le binaire sort en erreur : compter les TENTATIVES.
- **Un identifiant NEGATIF passe a `node -e` est pris pour une option de node.**
- **`grep -c` sur un mot accentue peut mentir** : 8 comptees pour 22 fonds reellement ecrits.
  Le critere de reprise est le SIDECAR sur disque, jamais un comptage de lignes de journal.
