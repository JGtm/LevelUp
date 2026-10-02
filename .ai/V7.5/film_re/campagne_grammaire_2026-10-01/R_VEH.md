# R-VEH — recherches R-L4 (véhicules `ti=40`) et R-L6 (ordre des plages) (2026-10-02)

> Campagne de grammaire, phase 2, recherches préalables (PLAN §6.1, tableau des recherches ; §6.2
> L4 et L6a). Worktree temporaire `LevelUp-wt-cg-veh`, HEAD détachée sur `fe18bf67c`. Rien n'est
> commité. Aucun fichier suivi modifié ; `grammar.Rev` reste `grammar-2026-09-27.3`. Aucune base,
> aucune cuisson, aucun backfill. Films lus en place, un à la fois, sentinelle `filmproc` 4 Gio
> (pics relevés : 52 à 254 Mio). Ghidra (HI_1_13_0) et modules installés en lecture seule.
>
> Convention (celle de la campagne) : **mesuré** = compté par une sonde sur les films ;
> **établi** = mesuré ET lu dans l'exécutable (ou dans les tags installés) ; **estimé** = dérivé
> d'une mesure par une hypothèse écrite ; **supposé** = non mesuré. « Fermé » (paquet) = reste nul
> ET aucun invariant de l'écrivain contredit (D2, juge `cmJuge`) ; le chiffre brut est publié à côté.
> Pour les records d'image-clé, il n'existe pas de juge d'invariants : « fermé » = la marche
> atterrit au bit près sur le premier bit du record suivant ; les témoins décalés et l'oracle `n2`
> en tiennent lieu (§1.4).

---

## Corrections du 2026-10-02 (ajoutées après coup ; le texte d'origine ci-dessous n'est pas réécrit)

Sources : verdicts adverses du chantier veh (`VERIFICATIONS_ADVERSES_R.md`, « Chantier veh »),
`CRITIQUE_COMPLETUDE_R.md` (points 4, 13, 14, 22, 24), `SURCOUCHE_UNIQUE.md` §4.4, §5 et `R_COMB_2.md`.

1. **§0 point 6, « lectures fausses … mesuré pour les trois » — NON CONFIRMÉ.** Mécanisme MPP 8/3
   mesuré (bits + fermeture) ; lecture fausse ÉTABLIE pour `77ef810a` seulement, SUPPOSÉE pour
   `4118381d` et `d0b40d0a` (`d0b40d0a` se relit à l'identique sous 8/3 à une ancre décalée, sur un
   film des formats 20-21 ; 878 châssis restent inconnus à 8/3 ; réutilisation de slot non exclue).
   La proposition « ne plus les identifier » reste défendable.
2. **§4, « critère de bascule de `PorteeBaseline` rempli » (R-VEH-1) — NON CONFIRMÉ.** Le critère
   écrit porte sur les 591 records `ti=35` BORNÉS du corpus R7 (`TestKF35CBaselineScope`), et celui de
   `GrammaireEcrivainI0` exige une non-régression delta ; ni l'un ni l'autre n'est rejoué ; la seule
   variante delta qui pose `i0` (avec la portée) perd 9 573 paquets ; sans 8/3, `ti=35` baisse sur
   `111fa685` (15 → 10) et `11de8353` (11 → 6). Formulation juste : « sur 2 008 voisins `ti=35` du
   format 27, la lecture portée + `i0` écrivain ferme 2 006 records ; le critère écrit n'a pas été
   rejoué ».
3. **`n2` modal de version-31** (§1.4 l. 172, §1.6 l. 194) : **`0x890`** (2192), et non `0x8a0`.
4. La largeur MPP 8 bits des formats 24-25 est MESURÉE, pas établie (aucun exécutable de ces builds).
5. §1.2 : la classe « non-VTOL, types 1, 3, 6, 7 » inclut le type 6, qui est VTOL. R-VEH-5 omet un
   sixième écrivain de la portée, `FUN_142e31bf8`.
6. §3.2 : la perte de 9 573 paquets (« portée sur les NEW ») est celle de la COMBINAISON portée + `i0`
   écrivain (`rvehPorteeNeuf` pose les deux) ; « en delta, les NEW restent quantifiés » n'est établi
   que pour cette combinaison.
7. **L6a hors Live Fire (§0 point 11, §5.4, §6 ; critique point 13)** : **13** films à deux plages, et
   non 12 (le tableau §5.4 a 13 lignes). 9 films sur 13 ont des pertes BRUTES ; en net de paquets sains
   (critère du gate 2, colonne « Sains nets » du §5.4), **7** films baissent (`084a804d`, `111fa685`,
   `11de8353`, `4f77afc1`, `51ebbc0f`, `c75f33b8`, `fb1a1a72`). Le relevé de type BIS_2 reproduit vaut
   102 / 139, pas 101 / 136.
8. **R-VEH-2, explication de l'ancienne contradiction MPP** — démentie par les TSV de la note : 8/3
   SEUL fait déjà monter les images-clés des formats 24-25 (52 391 → 53 291 fermés), et fait BAISSER
   les formats 20-21 qui partagent la case (17 398 → 17 038). Cause probable de l'ancienne baisse
   246 → 182 : ce regroupement (supposé).
9. **Brut et sains (critique point 4)** : « +26 843 records utiles » (L4 delta, §0 point 7, §3.1, §6)
   et « 12-26 % à 72-84 % du dénominateur fixe » (formats 24-25 sous LM) sont BRUTS. En sains
   (R-COMB-2) : L4a seul +1 403 paquets / +27 364 utiles sains, marginal dans C11 +4 440 / +115 847
   (gate 2 en défaut en marginal : `d9781168` −11 utiles) ; sous LM seul, sur le fixe de R-COMB-2, de
   49,6 % (`1c4c63c2`) à 79,9 % (`11de8353`) d'utiles sains par film (calcul awk du PLAN §6.5.3).
10. **Pertes de LM** (« pertes non jugées », §3.3, §6) : jugées par R-COMB-2 (§5.4) : 750 sains perdus
    en brut (443 devenus contredits, 307 non fermés), 79 195 gains sains, aucun film en baisse nette.
    LM × L6a mesuré en contexte de production sur `60ae07c4` : +22 194 sains (critique point 22).
11. **Rejoué après J12** : `r_veh_delta.tsv` et `r_veh_ti40_variantes.tsv` identiques à l'octet sous la
    surcouche unique ; `TestRVehIndex` et `TestRVehImageCle` non rejoués (supposés identiques).
    `r_veh_overlay/` (versions d'avant J12 de cinq fichiers) est remplacée par
    `surcouche_unique_postj12/`.
12. **§8, édition d'une sonde par `python3` (critique point 24)** : vérifiée juste depuis
    (`SURCOUCHE_UNIQUE.md` §5 : relecture de `rvVariante`, `gofmt`, `vet`, identité avec la copie de
    `LevelUp-wt-cg-veh`, mesure à l'octet) ; diff de l'édition impossible. L'en-tête de
    `r_veh_ti40_variantes.tsv` a 13 colonnes pour 15 (défaut d'en-tête, pas de `rvVariante`).
13. Gate de la note : `archlint` rouge à `fe18bf67c` (`TestNoExpiredTODO`, hors chantier), soldé depuis.

## 0. Résultat en onze points

1. **R-L4 (a) — ce n'est pas UNE largeur `ti=40` qui est fausse, ce sont trois LECTURES**, toutes
   trois communes à d'autres archétypes (établi) :
   - **la portée `DAT_144e61ea0`** : le lecteur d'état complet du jeu la pose à 1 autour de l'état
     par défaut (`FUN_142e2bfd0` @142e2c46f / @142e2c530) ET pour toute la boucle de composants
     (`FUN_142e2c690` @142e2c6b8 / @142e2c76a) ; sous elle, `FUN_14076f91c` rend vrai et chaque
     position `FUN_14076e494` se lit en `R(96)` brut. Le Go lit quantifié (bascule
     `PorteeBaseline`, défaut faux) ;
   - **le chemin absolu d'`i0` de l'écrivain** (`GrammaireEcrivainI0`, défaut faux) ;
   - **le découpage MPP 8/3** sur les formats de film 24 et 25 (le Go y lit 9/5, la case du profil
     est vide : `profile.MPPPourFormat`, « les deux oracles se contredisent »).
2. **Preuve** : sur les 11 films des formats 24 à 27, **3 058 / 3 058** records `ti=40` d'image-clé
   à voisin consécutif ferment (production : 14 ; BIS_2 « composants + porte par châssis » : 137).
   Témoins (boucle décalée d un bit) : +1 bit 102 / 1 325 et 84 / 1 733 ; −1 bit 63 / 1 325 (§1.4).
   Oracle `n2` (taille de l état véhicule, `0x8d8` sur HI_1_13_0) : 4 166 des 4 189 records de ces
   11 films (les autres : sans état par défaut, `n1 = 0`, ou ancres fortuites).
3. **Les pièces montées** (326/326 « au-delà de leur frontière » sur `4f77afc1`, D-52) : toutes
   portent la porte `bVar14 = 1` de l'état par défaut ; leur feuille 4 se lit `R(96)` et non
   quantifiée (+25 bits, 470/470 records : `n2` passe de `0x01507c00` à `0x8d8`).
4. **La porte `+0x818` par châssis est confirmée** par la fermeture : porte levée partout 2 952,
   posée partout 120, par châssis 3 058 (les 106 records d'écart sont les Falcon et les Wasp).
5. **Reste** : la famille de formats 20-21 (HI_1_4_1, version-31, version-33) : 0 / 2 036. L'état
   par défaut y est juste sous 8/3 (n2 constant sur 100 % des records), une largeur de la boucle de
   composants dépendante du build reste fausse ; le balayage par composant est diffus (§1.6).
6. **R-L4 (b) — les châssis `77ef810a`, `4118381d`, `d0b40d0a` sont des LECTURES FAUSSES** (établi
   pour le mécanisme, mesuré pour les trois) : sur les formats <= 25, lu à 9 bits, le premier champ
   MPP décale le mot `MPPWord32` d'un bit. Relu à 8/3, **0 → 84 %** des records `ti=40` d'image-clé
   lisent un châssis connu des tags installés (4 526 / 5 417) ; les trois slots des trois châssis
   sont des **Wasp** (`b65b3b4a`, VTOL) en image-clé, ce qui concorde avec leurs annonces `i34`.
7. **Effet sur la carte v2 (paquets delta)** : les 16 composants `ti=40` + porte posée en delta :
   **+1 436 / −0 paquets**, 35 gains contredits, +26 843 records utiles (21 films). Le découpage MPP
   8/3 sur les formats 24-25 : **+80 979 / −2 105 paquets** (2 646 gains contredits), records utiles
   fermés de **12-26 % à 72-84 %** du dénominateur fixe selon le film (§3).
8. **Écarté par la mesure** : la portée sur les records NEW du flux delta (−9 573 paquets sur le
   format 27) et la feuille 4 brute dans les NEW (−3 à −7) : en delta, les NEW se lisent quantifiés.
9. **R-L6 — règle établie** : les plages d'une carte sont les entrées (220 octets, `0xdc`) du bloc
   structure-BSP de son `levl`, dans cet ordre (`FUN_140be9a14`) ; `DAT_144632be0` vaut 1 bit pour
   une seule plage, `ceil(log2(n))` sinon ; l'écrivain n'écrit un index que si sa plage est valide
   ET contient le point (`FUN_140770640`, `FUN_14077084c`), sinon la porte posée. Les huit modules à
   deux sbsp du corpus : plage 0 = arène, plage 1 = décor lointain (l'hypothèse de BIS_2 est
   établie). Illusion et Fragmentation : **une seule plage** (aucun sbsp externe) — un index 1 y est
   impossible chez l'écrivain.
10. **Correction de D-54 et de BIS_2 §3.3 (mesuré)** : relus dans la lecture FINALE des records
   retenus, les paquets fermés qui portent un index impossible sont **0** (Illusion 0, Fragmentation
   0 / 0 / 0, au lieu de 101 et 136 / 25 / 2) ; sur Illusion, 0 des 172 992 lectures finales d'index
   vaut 1. Le relevé de BIS_2 comptait aussi les lectures d'ESSAI de l'inférence. Il n'y a donc ni
   fermeture factice à retirer, ni plage cachée. Même correction pour les cartes à deux plages :
   `4f77afc1` lit l'index 1 **une** fois sur 498 751 lectures finales (le relevé disait 7 966).
11. **L6a hors Live Fire : aucun gain, gate par film non tenu** (contexte de production, bornes de la
   plage 1 lues dans les modules) : 12 films à deux plages, +15 / −18 paquets, dont 12 des 15 gains
   contredits par le juge ; 9 films sur 12 perdent 1 à 3 paquets. Le lot L6a reste borné à Live
   Fire (seule carte du corpus dont les lectures finales désignent une autre plage que la jouée).

---

## 1. R-L4 (a) — la lecture des records `ti=40` d'image-clé

### 1.1 Instruments (tag `research && campagne_overlay`, aucun fichier de production)

| Fichier (sous `film/internal/grammar/`) | Rôle |
|---|---|
| `r_veh_ti40_research_test.go` | marche des records `ti=40` bornés, porte par châssis (`CAMPAGNE_PHYSIQUE`), relevé de l'état par défaut feuille par feuille (`rvPrefixe`), recherche de la position vraie de `n2` (`rvOuEstN2`), balayage « frontière de composant décalée de d bits » (`TestRVehTi40Balayage`), diagnostic par record |
| `r_veh_ti40_variantes_research_test.go` | A/B des lectures (`TestRVehTi40Variantes`), oracle `n2` sans grammaire (`TestRVehTi40ChercheN2`), bits de l'état par défaut (`TestRVehTi40BitsEtat`) |
| `r_veh_ti40_bits_research_test.go` | bits des records à porte `bVar14` |
| `r_veh_imagecle_research_test.go` | la même lecture sur TOUS les archétypes |
| `r_veh_delta_research_test.go` | effet sur la carte v2 (paquets) |
| `r_veh_chassis_research_test.go` (tag `research`) | R-L4 (b) |
| `r_veh_index_research_test.go` | R-L6 : index de plage des lectures FINALES (§5.3) |
| `internal/himap/r_veh_regions_research_test.go` (tag `research`) | R-L6 : plages des cartes lues dans les modules installés (§5.2) |

Surcouche : `r_veh_overlay/` = les trois copies de BIS_2 (inchangées) + `traverse.go` (bascule
`rvehPorteeNeuf` : portée sur tout record NEW) + `default_state_ti40.go` (bascule `rvehPorteeEtat` :
feuille 4 en `R(96)`). `overlay.json` pointe vers CE worktree. Outillage de mesure seulement (D10).

### 1.2 Le premier indice : l'oracle `n2` désigne l'état par défaut des pièces montées

`n2` est la taille du tampon que `vtable[0x88]` remplit ; pour `ti=40`, `FUN_14058c2ec` fait
`memset(param_5, 0, 0x8d8)` (T6 §2.2) : `n2` doit valoir `0x8d8` = 2 264 sur HI_1_13_0.

Mesuré sur `4f77afc1` (`r_veh_ti40_records_4f77afc1_production.tsv` ; sous la lecture retenue : `r_veh_ti40_records_4f77afc1_portee_i0.tsv`) :

| Classe (type de physique) | Records | `n2` lu (production) | position vraie de `0x8d8` |
|---|---|---|---|
| non-VTOL, porte `bVar14 = 0` (types 1, 3, 6, 7) | 661 | 2 264 (661/661) | d = 0 |
| pièces montées (type 5), porte `bVar14 = 1` | 470 | `0x01507c00` (388), `0x01507cb6` (82) | **d = +25** (388), **+57** (82) |

Les 470 records à `bVar14 = 1` sont exactement les pièces montées (aucun autre châssis ne la pose
sur ce film). Bits relevés (`r_veh_ti40_bits.tsv`) depuis `bVar14` jusqu'au `n2` vrai :

```
1 | 96 x 0 | 1 | 10000000 | 0001010100000111110 | 0 | 0 | n2 = 00000000000000000000100011011000
b14  R(96)   porte   R(8)        R(19)            c3  opt32
     FUN_1411b259c  FUN_140c1e79c                FUN_14076dc04
```

Le Go lit, au même endroit, `lireE494` quantifié (porte 0, index 1 bit, axes 15/15/17) puis
`FUN_140c1e79c` : 77 bits au lieu de 105, d'où le décalage de 25 bits (et 57 quand l'`opt32` de la
liste est présent).

### 1.3 Ce que dit l'exécutable (Ghidra, HI_1_13_0, lecture seule ; extraits dans `r_veh_ghidra/`)

- `FUN_14076f91c` : `return DAT_144e61ea0 != 0 || DAT_145121140 == 1` (`FUN_14076f91c.c`).
- `FUN_14076e494` : garde `f91c` vraie → `FUN_1411b259c` = `FUN_1406d676c(..., 0x60)` = `R(96)`
  (`DIS_14076e494.txt`, `FUN_1411b259c.c`).
- Feuille 4 de l'état par défaut `ti=40` : bloc froid `@1424a3a02` = `FUN_14076e494(lecteur,
  +0x64, 0x10, 0, param_5, 0)` puis `FUN_140c1e79c` (`cold_1424a3a02.txt`, octets lus en mémoire).
- **Les écritures de `DAT_144e61ea0`** (`xrefs_144e61ea0.json`, `portee_DAT_144e61ea0.txt`,
  `portee_autres_lecteurs.txt`) :
  - `FUN_142e2bfd0` (image-clé, par entité) : `MOV [DAT_144e61ea0], 1` @142e2c46f, juste avant
    `CALL [RAX+0x60]` @142e2c47b (état par défaut), remise à 0 @142e2c530 ;
  - `FUN_142e2c690` (boucle des 64 composants de l'image-clé) : `MOV ESI, 1 ; MOV [DAT_144e61ea0],
    SIL` @142e2c6b0-142e2c6b8 à l'entrée, remise à 0 @142e2c76a ;
  - `FUN_142e309b4`, `FUN_142e31a0c`, `FUN_142e30b9c`, `FUN_142e2d08c`, `FUN_142e2d6d4` posent
    aussi 1 à l'entrée (lecteurs NEW et voisins). **Mais la mesure §3.2 montre que les NEW du flux
    delta du film ne sont pas lus sous la portée** : ces fonctions ne sont pas (ou pas seules) le
    chemin des NEW du film. Non remonté.

Conclusion établie : dans une image-clé, l'état par défaut ET les composants se lisent sous la
portée. Cela corrige l'ancienne lecture du dépôt (`profil_balayage.go` : « la lèvent juste AVANT
l'appel vtable[0x60] et la rabaissent juste après ») sur un point : la boucle de composants est
aussi sous la portée. La mesure R7-c (`keyframe_baseline_scope_test.go`), qui avait rejeté la
portée, l'avait jouée **sans** le chemin absolu d'`i0` de l'écrivain ; jouées ensemble, les deux
bascules ferment (§1.5, et §4 pour `ti=35`).

### 1.4 A/B des lectures (`r_veh_ti40_variantes.tsv`, 21 films, 7 059 records bornés)

Records à voisin consécutif (`slot suivant = slot + 1` : la seule frontière qui ne peut pas
sauter un record) fermés, porte `+0x818` par châssis sauf mention :

| Variante | Format 27 (5 films, 1 325) | Formats 24-25 (6 films, 1 733) | Formats 20-21 (3 films, 2 036) |
|---|---|---|---|
| production | 5 | 9 | 0 |
| BIS_2 : 16 composants + porte par châssis | 95 | 42 | 0 |
| + portée sur l'état par défaut seule | 147 | 43 | 0 |
| + portée sur tout le record seule | 104 | 53 | 0 |
| + chemin absolu d'`i0` de l'écrivain seul | 95 | 28 | 0 |
| **+ portée sur tout le record + `i0` écrivain** | **1 325** | 41 | 0 |
| **idem + MPP 8/3** | 157 (témoin négatif) | **1 733** | 0 |
| idem, porte posée partout | 55 | 65 | — |
| idem, porte levée partout | 1 275 | 1 677 | — |
| témoin : boucle décalée de +1 bit | 102 (7,7 %) | 84 (4,8 %) | 0 |
| témoin : boucle décalée de −1 bit | 63 (4,8 %) | non joué | — |

Par film (voisins fermés / voisins), lecture retenue : `4f77afc1` 928/928, `bfecd02b` 230/230,
`396cfc92` 150/150, `81c02726` 14/14, `0797ce72` 3/3 (format 27, 9/5) ; `084a804d` 677/677,
`1c4c63c2` 253/253, `e5adf7b2` 349/349, `11de8353` 319/319, `111fa685` 113/113, `60ae07c4` 22/22
(formats 24-25, 8/3). Records sans MPP (`n1 = 0`) : comptés, et ferment comme les autres.

Témoins : le décalage d un bit ferme 0 à 8 % des records sur neuf films ; deux films y sont peu
sensibles (`396cfc92` +1 : 50/150 ; `60ae07c4` +1 : 21/22 — longues plages de bits nuls en tête de
boucle). Ils restent loin des 100 % de la lecture retenue, et la lecture retenue est indépendamment
confirmée par `n2` (valeur modale sur 98 à 100 % des records de chaque film : `0x8d8` sur les formats 24 à 27,
`0x8a0` version-31, `0x89c` version-33 et HI_1_4_1).

Records non voisins (`slot suivant ≠ slot + 1`) : aucun ne ferme, tous « sous » la frontière ; sur
`4f77afc1` les écarts sont des multiples de 108 bits (−108 ×32, −324 ×24, −432 ×12, −216 ×8) ;
aucun ne tombe sur un en-tête lisible, génération 0 comprise (`sous_sur_ancre` = 0). **Supposé** :
des records d'en-tête seul (108 bits) que la marche d'ancres ne voit pas (lien L9 / D-51).

### 1.5 Le balayage par composant (`TestRVehTi40Balayage`) : comment la cause a été localisée

Pour chaque record non fermé et chaque composant lu, la boucle reprend à la fin de ce composant
décalée de d bits (|d| <= 32) ; une largeur fixe fausse donne un d modal. Sur `4f77afc1`,
production (`r_veh_ti40_balayage_4f77afc1_production.tsv`) : `type5`, « fin de l état par défaut, d = −24 » ferme 269 des 432 pièces montées non
fermées (le reste se répartit sur des d isolés, < 140). C'est ce signal qui a désigné l'état par
défaut ; l'oracle `n2` l'a ensuite chiffré (+25, §1.2) — le −24 du balayage vient de ce que la
reprise relit `n2` et `i0` à une position où leur lecture compense un bit. Sous la portée sur
l'état par défaut seule, le balayage devient diffus (meilleur : `i00` d = +29, 131 / 380) : il n'y
a plus de largeur fixe fausse, mais une lecture (portée + `i0`) qui change plusieurs largeurs à la
fois.

### 1.6 Le reste : formats 20-21 (HI_1_4_1, version-31, version-33)

- L'état par défaut est juste sous MPP 8/3 + portée : `n2` modal sur 100 % des records (`0x89c`,
  `0x8a0`) — mesuré.
- 0 / 2 036 voisins ferment ; les arrêts sont surtout « sur » (1 572 / 1 776 sur `a349fea8`).
- Balayage par composant sous la lecture retenue (`a349fea8`, `a521164d`, `50247b26`) : diffus
  (meilleurs : `type5 i08 object-constraint` d = −23 et −25, 94 / 665 ; `type1 i07` d = +4 et
  `i08` d = +12, 48 / 462). Plusieurs largeurs de composants y diffèrent (cohérent avec D-26 : la
  famille HI_1_4_1 = v31 = v33 a ses propres registres). **Non identifié** ; à instruire avec R-L3
  (même famille de builds).

### 1.7 Ce qui change pour le lot L4

- La « largeur fausse » cherchée n'est pas une largeur `ti=40` : L4 doit embarquer la **lecture de
  l'image-clé sous la portée + `i0` écrivain** (un lot transverse, §4) et, pour les formats 24-25,
  **le découpage MPP 8/3** (lot transverse, §3.3). Sans eux, porter les 16 composants ne ferme que
  95 + 42 records d image-clé (formats 24 à 27).
- La porte `+0x818` par châssis (table §4.1 de BIS_2) est validée par la fermeture sur les deux
  familles de formats (VTOL : Falcon `0000254b`, Wasp `b65b3b4a`).
- La mention « `i41`/`i42` NE PAS PORTER » est démentie en image-clé (lus partout, 16 bits chacun,
  et la fermeture à 100 % les contient).

---

## 2. R-L4 (b) — les châssis `77ef810a`, `4118381d`, `d0b40d0a`

### 2.1 La cause commune : le premier champ MPP fait 8 bits sur les formats <= 25

Bits du bloc MPP du même châssis (tourelle `bcfb852f`) sur deux builds (`r_veh_ti40_bits_etat.tsv`) :

```
HI_1_13_0 (4f77afc1) : 000000000 10111100111110111000010100101111 ...   lu à 9 bits -> bcfb852f
HI_1_10_0 (084a804d) : 00000000 10111100111110111000010100101111 ...    lu à 9 bits -> 79f70a5f
                                                                        lu à 8 bits -> bcfb852f
```

Lu à 9 bits, le châssis vaut `(vrai << 1) | bit suivant`. `FUN_141fd72c0` lit 9 bits sur HI_1_13_0
(`ADD [RCX+0x2c], 9`, `FUN_141fd72c0.c`) : la largeur 8 des builds anciens est **mesurée** (un seul
exécutable, D6).

| Mesure (`r_veh_ti40_variantes.tsv`, `r_veh_chassis_neufs.tsv`) | 9/5 | 8/3 |
|---|---|---|
| records `ti=40` d'image-clé des formats <= 25 dont le châssis est connu des tags installés | 0 / 5 417 | **4 526 / 5 417 (83,6 %)** |
| idem, format 27 | 1 457 / 1 642 | 0 / 1 642 |
| NEW `ti=40` de la marche des trames, formats <= 25, châssis connu | 0 / 140 | 72 / 140 |
| idem, égal au châssis d'image-clé du même slot | — | 93 / 140 |
| créations du balayeur de production (`ScanVehicleCreations`), formats <= 25, châssis connu | 0 / 2 077 | 822 / 2 077 |

### 2.2 Les trois châssis

| Châssis (9/5) | Source | Relu à 8/3 | Châssis d'image-clé du slot (8/3) | Verdict |
|---|---|---|---|---|
| `77ef810a` | NEW de la marche, `084a804d` (1 lecture, paquet fermé, NEW lu au bout) | `3bf7c085` (inconnu) | **`b65b3b4a` Wasp** | lecture fausse du NEW (même à 8/3, le mot ne vaut pas le châssis du slot) |
| `4118381d` | balayeur de créations, `084a804d` (1) | absent du balayeur à 8/3 | **`b65b3b4a` Wasp** | fausse création du balayeur |
| `d0b40d0a` | balayeur de créations, `50247b26` (1 à 9/5, 1 autre position à 8/3) | — | **`b65b3b4a` Wasp** | fausse création du balayeur |

Les trois slots sont des Wasp (type 6, VTOL) : leurs DELTA qui annoncent `i34` (457, 373, 2 523,
MESURES_CIBLEES T6-C1) sont des annonces de VTOL, conformes à la loi de l'écrivain
(`FUN_142f09c74`). La proposition du PLAN §6.2 L4 (« ne plus les identifier ») tient ; sa
justification change : ce ne sont pas des châssis inconnus, ce sont des lectures fausses.
Un **châssis inconnu en image-clé** reste possible à 8/3 (878 records sur les formats <= 25, par
exemple `c52938d3`, 4 NEW concordants avec l'image-clé) : tags d'anciens builds absents de
l'installation.

---

## 3. Effet sur la carte de fermeture v2 (paquets delta, 21 films)

Instrument `TestRVehDeltaTi40`, contexte des instruments, juge des invariants de BIS_1 sur chaque
marche. Contrôle : la variante `reference` rend la carte v2 à l'identique (649 075 paquets,
dont 629 142 sur les 20 films ; `fb1a1a72` 22 318 fermés, 161 678 / 179 254 utiles, comme
`mb_variantes.tsv`). TSV : `r_veh_delta.tsv`, synthèse par film `r_veh_delta_synthese.tsv`.

### 3.1 Les 16 composants `ti=40`, porte posée en delta (loi de l'écrivain)

| Format | Gagnés | Perdus | Gagnés contredits | Utiles gagnés |
|---|---|---|---|---|
| 27 | 1 329 | 0 | 11 | 24 813 |
| 24-25 | 105 | 0 | 22 | 2 028 |
| 20-21 | 2 | 0 | 2 | 2 |
| **total** | **1 436** | **0** | **35 (2,4 %)** | **26 843** |

Par film (gagnés) : `bfecd02b` 729, `4f77afc1` 596, `084a804d` 31, `1c4c63c2` 26, `e5adf7b2` 24,
`111fa685` 15, `11de8353` 9, `81c02726` 3, `a349fea8` 2, `d9781168` 1 ; aucun film ne perd.
« Hors cadre » monte sur 7 films (`084a804d` +150, `a349fea8` +127, …) : des paquets jusque-là
arrêtés sur une désynchronisation NOMMÉE `ti=40` vont plus loin et s'arrêtent en vue C — le
déplacement de cause que D-64 décrit pour `ti=43`. Aucun paquet n'est perdu. L'estimation du plan
(« ≤ 1 666 paquets ») est remplacée par cette mesure.

### 3.2 Écartés par la mesure

| Variante | Format 27 | 24-25 | 20-21 |
|---|---|---|---|
| portée sur tout record NEW (tous archétypes) | +50 / **−9 573** | +115 / −126 | +1 / −3 |
| 16 composants + feuille 4 brute dans les NEW | +1 326 / −3 | +106 / −7 | +2 / −1 |

En delta, les records NEW du film se lisent **quantifiés** (la mesure R7-c tient pour eux) ; la
portée vaut pour l'image-clé seule.

### 3.3 Le découpage MPP 8/3 sur la carte v2 (découverte, hors du périmètre strict de R-L4)

| Format | Gagnés | Perdus | Gagnés contredits | Utiles gagnés |
|---|---|---|---|---|
| 24-25 (6 films) | **80 979** | 2 105 | 2 646 (3,3 %) | 1 729 890 |
| 24-25, + 16 composants `ti=40` | **83 050** | 2 092 | 2 724 | 1 785 802 |
| 20-21 (témoin) | 32 | 34 | 25 | 67 |
| 27 (témoin négatif) | 632 | **118 413** | 511 | 860 |

Records utiles fermés (dénominateur FIXE = maximum des utiles lus sur les marches de la campagne
— `mb_variantes`, `mb3_ti3`, `mb2_ti43` — et celles de cette note ; variable entre parenthèses ;
« 8/3 + 16 composants » ; brut, paquets contredits compris) :

| Film | Build | Fixe max | Référence | 8/3 + composants | Paquets fermés réf. → var. | Gagnés / perdus (contredits) |
|---|---|---|---|---|---|---|
| `084a804d` | HI_1_10_0 | 731 523 | 12,2 % (16,6 %) | **81,2 %** (81,2 %) | 5 235 → 22 716 | +17 697 / −216 (380) |
| `111fa685` | HI_1_10_0 | 316 811 | 13,5 % (17,4 %) | **77,1 %** (77,1 %) | 4 152 → 12 071 | +7 974 / −55 (127) |
| `1c4c63c2` | HI_1_10_0 | 968 454 | 17,6 % (21,9 %) | **72,0 %** (72,0 %) | 19 354 → 40 136 | +22 388 / −1 606 (1 970) |
| `e5adf7b2` | HI_1_11_0 | 359 795 | 22,2 % (27,7 %) | **83,6 %** (83,6 %) | 4 285 → 12 318 | +8 100 / −67 (104) |
| `60ae07c4` | HI_1_8_0 | 316 124 | 26,3 % (31,0 %) | **74,9 %** (74,9 %) | 13 969 → 32 635 | +18 756 / −90 (69) |
| `11de8353` | HI_1_9_0 | 316 305 | 20,7 % (26,4 %) | **84,1 %** (84,1 %) | 5 739 → 13 816 | +8 135 / −58 (74) |

Le dénominateur fixe monte avec la mesure (D-42) : sur ces six films, le maximum est désormais
celui de la variante 8/3. Les pertes (2 092 paquets, dont 1 606 sur `1c4c63c2`) ne sont pas
jugées une à une ici (le juge compte les gains contredits, pas les pertes saines) : à instruire
avant tout lot. **Cette mesure contredit la décision écrite dans `profile/build_profile.go`**
(« poser 8/3 ferait descendre le ratchet de couverture de 246 à 182 records fermés », mesure des
images-clés de 2026-09-15, faite sans la portée ni le chemin `i0` de l'écrivain) : sous la lecture
de §1, les images-clés montent aussi (§4).

---

## 4. Élargissement : la même lecture sur tous les archétypes d'image-clé

`TestRVehImagesClesTousArchetypes` (`r_veh_imagecle_archetypes.tsv`), records à voisin consécutif
fermés, archétypes qui bougent :

| Format | ti | Voisins | Production | Portée + `i0` écrivain | 8/3 + portée + `i0` | 8/3 seul |
|---|---|---|---|---|---|---|
| 27 | **35 (bipède)** | 2 008 | 142 | **2 006** | 94 | 86 |
| 27 | 40 | 1 325 | 5 | **1 325** | 157 | 5 |
| 27 | 38 / 42 / 37 | 56 668 / 3 701 / 2 331 | 3 217 / 419 / 36 | inchangé | 1 994 / 224 / 33 | idem |
| 24-25 | **35** | 2 532 | 94 | 98 | **2 297** | 186 |
| 24-25 | 40 | 1 733 | 9 | 41 | **1 733** | 9 |
| 24-25 | 38 / 42 | 29 029 / 2 635 | 1 447 / 117 | inchangé | **1 979 / 379** | 1 979 / 379 |
| 20-21 | 38 / 42 | 14 551 / 3 352 | 973 / 179 | inchangé | 625 / 152 (baisse) | idem |

Le bipède `ti=35` passe de 7 % à 99,9 % de records fermés sur le format 27 : c'est le critère de
bascule écrit pour `PorteeBaseline` (« l'atterrissage bit-exact des 591 records `ti=35` bornés
au-dessus de 50 % », `profil_balayage.go`) — rempli, à condition de poser aussi
`GrammaireEcrivainI0`. Aucun archétype ne baisse sous « portée + `i0` » sur le format 27. Sur les
formats 20-21, 8/3 fait baisser `ti=38` et `ti=42` : la famille 20-21 n'est pas la famille 24-25.

---

## 5. R-L6 — l'ordre des plages et l'index 1 des cartes à une plage

### 5.1 La règle (établie)

- `FUN_140be9a14` (chargement de carte, `FUN_140be9a14.c`) parcourt les entrées du bloc
  structure-BSP du scénario (pas `0xdc` = 220 octets, compte en `scénario+0x7bc`), pose pour chaque
  entrée valide le bit de `DAT_1445ccb60` (plage valide) et ses bornes `DAT_14462cbe0 + r*0x18`,
  puis `DAT_144632be0 = 1` si le compte vaut 1, `FUN_1406d310c()` (le `ceil(log2)` du dépôt) sinon.
- `FUN_140770640` (écrivain) garde l'index proposé seulement si sa plage est valide ET contient le
  point ; sinon `FUN_14077084c` cherche une plage valide qui le contient ; à défaut l'index vaut −1
  et `FUN_1407eb6a8` écrit la porte posée (`W(1) = index == -1`).
- Donc : **les plages d'une carte sont les entrées de ce bloc, dans son ordre ; un index lu hors de
  `[0, n)` est impossible chez l'écrivain.**

### 5.2 Les cartes du corpus, lues dans les modules installés (`r_veh_regions_levl.tsv`)

Instrument `internal/himap/r_veh_regions_research_test.go` : tags sbsp locaux, tags sbsp des 10
modules `ds/globals` que le `levl` référence, ordre (`ordreRegionsBSP`, la règle de
`BSPQuantification`) et chaque bloc du `levl` qui les cite (offsets : entrées de 220 octets).

| Module (cartes du corpus) | Plages (ordre du `levl`) | Largeurs niveau 0x10 |
|---|---|---|
| `fo05_desert` (Banished Narrows, Flood Gulch, Fortitude), `fo06_deepsea` (Dredge), `fo08_wetland` (Perilous, Thunderhead), `fo09_academy` (Command), `fo11_blank` (Curfew, Solitude), `fo13_frost` (Snowbound) | 0 : arène ; 1 : décor lointain | 0 : [15 15 17] ; 1 : [18 18 18] |
| `ridgeline` (Cliffhanger) | 0 : `5f388668` ; 1 : `86a62795` | [13 13 14] ; [19 20 16] |
| `btb_exiled` (Oasis) | 0 : `1a48c67a` ; 1 : `b5a2d747` | [15 15 14] ; [18 18 16] |
| `btb_fragmentation` (Fragmentation, Fragmentation Heavies) | **0 : `69d4b9ed` seule** (bloc de 220 octets) | [17 17 15] |
| `ctf_illusion` (Illusion) | **0 : `ab9f24c0` seule** (bloc de 220 octets) | [18 18 17] |
| `sgh_interlock` (Live Fire, contrôle) | 0 `7047b96f`, 1 `d88e1d88`, 2 `a59f5052`, 3 `91c336c1` (tous dans `common`) | = BIS_2 |

- Sur les cartes à deux sbsp, l'hypothèse de BIS_2 (« plage 1 = le second sbsp ») est **établie** :
  plage 0 = arène dans les huit modules, plage 1 = décor lointain.
- Illusion et Fragmentation n'ont **aucun** sbsp externe (les 4 tags sbsp des 10 modules globaux
  `ds/globals` sont ceux de Live Fire) : une seule plage, index sur 1 bit, index 1 impossible chez
  l'écrivain. **Supposé**, non vérifié : les modules `pc/` et `any/` (non lus) ne portent pas de sbsp
  de plus.

### 5.3 Les index impossibles : la lecture finale, pas le relevé (mesuré)

Instrument `r_veh_index_research_test.go` (`TestRVehIndexImpossibles`, contexte de production,
juge de BIS_1) : pour chaque paquet, chaque composant porté de chaque record RETENU par la marche est
relu seul à son `StartBit`, et les index de plage lus sont relevés ; un index `>=` nombre de plages
de la carte est impossible. Contrôle : Live Fire `0797ce72` relit l'index 3 4 022 fois (relevé de
BIS_2 : 4 203, essais compris). TSV : `r_veh_index_impossibles_final.tsv`.

| Film | Carte (plages) | Fermés | Fermés avec index impossible (final) | Non fermés idem | Lectures finales par index |
|---|---|---|---|---|---|
| `396cfc92` | Illusion (1) | 22 828 | **0** (BIS_2 : 101) | 0 | 0 : 172 992 |
| `e5adf7b2` | Fragmentation (1) | 4 285 | **0** (BIS_2 : 136) | 2 | −1 : 5 ; 0 : 242 507 ; 1 : 3 |
| `a349fea8` | Fragmentation Heavies (1) | 463 | **0** (BIS_2 : 25) | 1 | −1 : 4 ; 0 : 351 351 ; 1 : 1 |
| `a521164d` | Fragmentation Heavies (1) | 701 | **0** (BIS_2 : 2) | 3 | −1 : 5 ; 0 : 137 764 ; 1 : 3 |
| `4f77afc1` | Flood Gulch (2) | 22 910 | 0 | 0 | −1 : 6 ; 0 : 498 744 ; **1 : 1** |
| `0797ce72` | Live Fire (4) | 19 124 | 0 | 0 | −1 : 23 ; 0 : 15 ; 1 : 145 308 ; 2 : 7 ; 3 : 4 022 |

Les « 101 et 136 paquets fermés qui lisent un index 1 » (D-54) étaient des lectures d'ESSAI
(inférence de largeur, chaînes de tête) faites dans des paquets qui ferment par ailleurs. La
question « la carte déclare-t-elle une plage de plus ? » est close (non, §5.2), et la question
« fermetures factices ? » aussi (non). Le relevé par composant de BIS_2 (`mb2_index_de_plage.tsv`)
et le mien (`r_veh_l6a_index_par_film.tsv`, agrégé par film et par index ; la colonne
`paquets_fermes` y est une somme sur les composants) mesurent les essais autant que les lectures
retenues : ils ne disent pas ce que le film porte.

### 5.4 L6a hors Live Fire, contexte de production (gate par film)

Instrument : `TestCampagneBis2PositionsProduction` de BIS_2, rejoué sous ma surcouche, bornes de
la plage 1 des huit modules à deux sbsp (§5.2) et des plages 0, 2, 3 de Live Fire
(`r_veh_l6a_positions_production.tsv`, `r_veh_l6a_index_impossibles.tsv`). Variante
`reference+lecture-par-index` contre `reference` :

| Film | Gagnés | Perdus | Gagnés contredits | Sains nets |
|---|---|---|---|---|
| `084a804d` | 4 | 3 | 4 | −3 |
| `111fa685` | 0 | 2 | 0 | −2 |
| `11de8353` | 3 | 1 | 3 | −1 |
| `4f77afc1` | 1 | 3 | 0 | −2 |
| `51ebbc0f` | 0 | 3 | 0 | −3 |
| `bfecd02b` | 2 | 0 | 2 | 0 |
| `c75f33b8` | 1 | 1 | 1 | −1 |
| `d9781168` | 3 | 1 | 2 | 0 |
| `f75e7053` | 1 | 1 | 0 | 0 |
| `fb1a1a72` | 0 | 3 | 0 | −3 |
| `bcb6d393`, `bf15f7ab`, `50247b26` | 0 | 0 | 0 | 0 |
| **12 films à deux plages** | **15** | **18** | **12** | **−15** |
| contrôle `0797ce72` (Live Fire) | 3 269 | 3 | 3 | +3 263 |
| contrôle `60ae07c4` (Live Fire) | 1 806 | 49 | 4 | +1 753 |

Les contrôles Live Fire reproduisent BIS_2 §3.3 au paquet près. Verdict (mesuré) : hors Live Fire,
la lecture par index ne change que des lectures d'essai ; elle ne gagne rien et fait perdre 1 à 3
paquets sur 9 films sur 12. Le gate « aucune baisse par film » n'est pas tenu : L6a reste Live Fire
seul. La donnée de carte (bornes de toutes les plages) peut être écrite pour toutes les cartes — la
règle §5.1 la définit —, sans effet de fermeture mesurable sur le corpus.

---

## 6. Gains et dénominateurs : récapitulatif pour le plan

| Mesure | Brut | Sains (juge) | Records utiles |
|---|---|---|---|
| L4 composants + porte posée (delta, 21 films) | +1 436 / −0 paquets | +1 401 | +26 843 ; fixe : `4f77afc1` 72,3 → 74,9 %, `bfecd02b` 85,6 → 88,0 % |
| MPP 8/3 + L4 composants (formats 24-25, 6 films) | +83 050 / −2 092 | gains sains +80 326 ; pertes non jugées | fixe : 12-26 % → 72-84 % selon le film (§3.3) |
| Image-clé `ti=40`, lecture complète (11 films) | 14 → 3 058 / 3 058 voisins | pas de juge d'image-clé ; témoins ±1 bit ≤ 7,7 % | dénominateur fixe = les 5 094 voisins bornés des 14 films à `ti=40` |
| Image-clé `ti=35`, portée + `i0` (format 27) | 142 → 2 006 / 2 008 voisins | idem | — |
| L6a hors Live Fire (12 films) | +15 / −18 | −15 | inchangé |

Dénominateur FIXE des records utiles = par film, le maximum des records utiles lus sur {BIS_1
`mb_variantes.tsv`, BIS_3 `mb3_ti3.tsv`, BIS_2 `mb2_ti43.tsv`, marches de cette note}
(`r_veh_delta_synthese.tsv`, colonne `fixe_max`) ; il monte sur les six films des formats 24-25 (le
maximum devient la variante 8/3) et ne bouge pas ailleurs. Variable = utiles lus par la marche.

---

## 7. Découvertes (consignées, non traitées)

- **R-VEH-1 (majeure) — portée `DAT_144e61ea0` et chemin `i0` de l'écrivain en image-clé, tous
  archétypes** : la lecture établie ici sur `ti=40` ferme aussi le bipède `ti=35` (2 006 / 2 008
  voisins sur le format 27, contre 142) sans faire baisser un autre archétype (§4). Le critère de
  bascule de `PorteeBaseline` (`profil_balayage.go`) est rempli, à condition de poser
  `GrammaireEcrivainI0` avec elle ; la mesure R7-c qui l'avait rejetée jouait la portée seule.
  Les deux bascules ne valent QUE pour l'image-clé : sur les NEW du flux delta, la portée fait
  perdre 9 573 paquets (§3.2). À instruire comme un lot transverse (lien L9 : les records
  d'image-clé fermés deviennent un témoin de marche).
- **R-VEH-2 (majeure) — découpage MPP 8/3 sur les formats 24-25** : +80 979 / −2 105 paquets sur
  six films (§3.3), et les châssis lus deviennent réels (§2). Contredit la décision écrite dans
  `profile/build_profile.go` (case laissée vide, « les deux oracles se contredisent » ; la mesure de
  fermeture citée y était faite sans la portée ni le chemin `i0` de l'écrivain). Les pertes (1 606
  sur `1c4c63c2`) sont à juger avant tout lot. Formats 20-21 : 8/3 ne gagne rien (+32 / −34) et fait
  baisser `ti=38` / `ti=42` d'image-clé ; leur découpage reste à trouver.
- **R-VEH-3** — la famille de formats 20-21 (HI_1_4_1, v31, v33) a au moins une largeur de
  composant `ti=40` différente (0 / 2 036 voisins, état par défaut juste) ; à instruire avec R-L3.
- **R-VEH-4** — les records d'image-clé à voisin non consécutif s'arrêtent 108·k bits avant la
  frontière (`4f77afc1`) sans tomber sur un en-tête lisible, génération 0 comprise : des records
  d'en-tête seul que la marche d'ancres ne voit pas (supposé ; lien L9 / D-51).
- **R-VEH-5** — `FUN_142e309b4`, `FUN_142e31a0c`, `FUN_142e30b9c`, `FUN_142e2d08c`, `FUN_142e2d6d4`
  posent aussi la portée, mais les NEW du film ne sont pas lus sous elle (mesuré) : le chemin des NEW
  du film n'est pas remonté. La note de `profil_balayage.go` (« la lèvent juste avant vtable[0x60]
  et la rabaissent juste après ») est incomplète pour `FUN_142e2c690`.
- **R-VEH-6** — `ScanVehicleCreations` (balayeur de production) rend des créations à châssis
  inexistant (`4118381d`, `d0b40d0a`) : faux positifs du balayeur, visibles dès qu'on relit le slot
  en image-clé.
- **R-VEH-7** — le relevé des index de plage de la surcouche (`bis2NoterIndex`) compte les lectures
  d'essai : tout chiffre « lectures d'index » de BIS_2 §3.3 est à relire (§5.3). Vrai de tout relevé
  posé dans un lecteur que l'inférence appelle.
- **R-VEH-8** — témoins décalés peu discriminants sur deux films (`396cfc92`, `60ae07c4`) : un
  décalage d'un bit en tête de boucle y retombe souvent sur la même structure (longues plages de bits
  nuls) ; un témoin de hasard à décalages de 2 à 8 bits serait plus robuste.

---

## 8. Limites et écarts aux règles

- Un seul exécutable (HI_1_13_0) : la largeur 8 du premier champ MPP des formats <= 25 est
  **mesurée** (fermeture, `n2`, châssis réels), pas lue dans un exécutable de ces builds (D6).
- Les pertes des variantes 8/3 ne sont pas passées au juge (le juge de BIS_1 compte les gains
  contredits) ; seuls les gains contredits sont publiés.
- Les mesures `ti=40` et la carte delta sont jouées sous le contexte des instruments (région 0,
  découpage d'`i0` auto-détecté) ; le contexte de production n'est joué que pour R-L6.
- Écart à la règle « pas de Python » : une édition de sonde (`r_veh_ti40_research_test.go`, ajout de
  la structure de variante) a été faite par un appel `python3` en ligne de commande (remplacement de
  texte) ; aucun fichier Python n'a été créé ni laissé. Les éditions suivantes sont en `perl` et par
  l'outil d'édition.

---

## 9. Gate (depuis `apps/go-api` de ce worktree)

Relevé le 2026-10-02 après la dernière édition des sondes :

| Commande | Résultat |
|---|---|
| `gofmt -l` sur les 8 sondes `r_veh_*` et les 5 copies de `r_veh_overlay/` | vide |
| `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/ ./internal/himap/` | vert (rc 0) |
| `go vet -tags=research,campagne_overlay -overlay=.ai/.../r_veh_overlay/overlay.json ./internal/games/halo_infinite/film/internal/grammar/` | vert (rc 0) |
| `go test -count=1 ./internal/archlint/` | **160 PASS, 1 FAIL** : `TestNoExpiredTODO`, un `TODO(expiry:2026-10-01)` échu dans `internal/api/handlers/json_huma_coverage_test.go:34` (fichier inchangé depuis juillet ; l'échéance est passée le 2026-10-01). Sans lien avec les fichiers de cette note ; non corrigé (aucun fichier suivi ne se modifie ici). Les ratchets de taille (`film_file_size_test.go`, sondes ≤ 462 lignes), de tag `research` et `gamefiles` passent. |

`git status` : 12 entrées non suivies, toutes neuves (cette note, `r_veh_tsv/`, `r_veh_ghidra/`,
`r_veh_overlay/`, 8 sondes) ; aucun fichier suivi modifié ; `grammar.Rev` inchangé.
