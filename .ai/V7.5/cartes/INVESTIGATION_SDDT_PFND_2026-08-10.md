# INVESTIGATION — la zone jouable est-elle dans `sddt` ou `pfnd` ? (2026-08-10)

**VERDICT : NO-GO pour les deux tags — portes fermees par la mesure.**

`sddt` se decode INTEGRALEMENT (acquis reutilisable) : il porte la frontiere de MORT
(une coquille convexe de 6 a 8 plans) et les surfaces d'eau (la riviere de Cliffhanger,
233 prismes minces). La coquille contient 100,00 % des positions jouees ET l'essentiel
des 149 % d'exces : elle dit « ou tu meurs », pas « ou tu peux te tenir ».
`pfnd` ne porte PAS de navmesh : 55 liens de saut/escalade (40 sur Catalyst), aucune
donnee surfacique — la definition positive de la zone jouable n'existe dans aucun des
deux tags.

Sondes (non commitees) : `apps/go-api/internal/himap/sonde_sddt_pfnd_gamefiles_test.go`,
`sonde_sddt_banc_gamefiles_test.go`, `sonde_sddt_xy_gamefiles_test.go`.
Tags : variante `any/` UNIQUEMENT (verifie : aucun `sddt`/`pfnd` dans `pc/` ni `ds/`).
ridgeline `sddt#1303` (136 884 o) · `pfnd#1361` (8 576 o) · catalyst `sddt#1652`
(3 240 o) · `pfnd#1726` (5 872 o). Reclaimer n'implemente NI l'un NI l'autre (grep
sur les 7 refs `LevelUp-re/scratchpad_recherche/refs/*.cs`) — tout est lu par la
struct-table generique (`tagCandidates` + `blockAbs` + liens type 1), sans plugin.

## 1. `sddt` se decode — structure etablie, temoins joues

Arbre (ridgeline ; catalyst identique mais sans le bloc des volumes) :

    root (544 o)
      @0x44 -> 1 x 112 o, portant a +0x50 une chaine de 101 o (noms, compresses)
      @0x58 -> 1 x 28 o,  portant a +0x08 les TRIANGLES-FRONTIERES (12 x 68 o | 20 x 68 o)
      @0x6c -> 1 x 112 o, chaine de 2 629 o
      @0x80 -> 1 x 4 o
      @0x94 -> VOLUMES 233 x 100 o (ridgeline seulement ; catalyst : 0)
      puis ~20 champs tag-block VIDES

    volume (100 o) : u16 type @0x00 (233 x valeur 0) · f32 0,1 @0x04
      @0x24 tag-block -> 5 plans de 16 o (a, b, c, d)
      @0x38 tag-block -> 8 triangles de 36 o (3 sommets vec3)
      @0x4c AABB propre (6 f32 : min/max entrelaces)

    triangle-frontiere (68 o) : normale vec3 + d + centroide vec3 + rayon + 3 sommets vec3

Temoins JOUES (pas supposes) :

- **plans** : normales unitaires sur 1 165/1 165 plans a l'offset 0 (100,0 %),
  **0,0 % a +4 octets** — la mutation separe, contrairement au temoin de la sonde
  physique (reserve n°1 de l'investigation collision : soldee ici).
- **coherence plans <-> triangles** : les 5 592 sommets des 1 864 triangles tombent
  TOUS sur les plans de leur volume (residu max 0,0000) et TOUS dans la boite monde
  du bsp (0 hors, marge 10 m).
- **triangle-frontiere** : centroide lu == moyenne des 3 sommets a 0,0000 pres sur
  12 + 20 records ; rayon == distance max centroide-sommet.
- **orientation des prismes** : le centroide des sommets de chaque volume est dans son
  prisme (233/233) au sens `n.p <= d` ; le sens inverse echoue 233/233.
- **orientation de la coquille** : `n.p >= d` contient 29 221/29 221 positions jouees.

## 2. Les volumes sont en coordonnees monde — dessines

Coquille ridgeline : boite x ±75, y ±60, plancher z = -80,59, plafond z = +20 —
les 12 triangles sont les 6 faces. Catalyst : 8 plans dont un toit incline
(normales (0, ±0,90, -0,43)), emprise x [-33,1 ; +31,6], y ±58, z [-46,1 ; +42,4].
Les 233 prismes ridgeline vivent dans x [3,5 ; 29,8], y [13,5 ; 21,9], z [-5,2 ; -2,8]
(dimension mediane 0,98 x 0,71 x 1,08 m). DESSINE (`SONDE_PNG_DIR`,
`sonde_sddt_*.png`) : la bande rouge des prismes SERPENTE — c'est LA RIVIERE de
Cliffhanger, le motif « water groups » du structure design classique. La frontiere
verte de la coquille n'effleure que l'angle haut-droit du cadre de la reference.

## 3. Sont-ce des kill volumes ? Ce que l'oracle repond

- La coquille est une frontiere de RETENUE : 100,00 % des positions jouees dedans.
  C'est la donnee que decrit l'utilisateur (mort/compte a rebours en sortant) — elle
  existe, elle se lit, mais elle est GROSSIERE : tout le decor de l'arene est dedans.
- Les 233 prismes contiennent 128/29 221 positions jouees (0,438 %) : pas des kill
  volumes stricts (des traversees de la riviere y laissent des echantillons), des
  surfaces d'eau.

## 4. `pfnd` se decode — et ne porte pas de navmesh

    root (96 o) : @0x10 -> 2 x 24 o (config) · @0x24 -> 8 x 36 o (ridgeline seulement)
                  @0x4c -> 1 x 60 o -> @0x00 -> 55 x 96 o (catalyst : 40 x 96 o)

    record 96 o : header (flags 0x30/0x38, hash) + QUATRE points vec3 = deux paires
    a deux altitudes (~1,4 m d'ecart) = un LIEN DE FRANCHISSEMENT (saut/escalade),
    le « off-mesh link » classique. Champ tag-block a +0x4c : VIDE sur les 95 records.

55 + 40 liens, ~220 points : AUCUNE donnee surfacique. Le navmesh n'est ni ici ni
ailleurs dans les modules de carte (aucun tag `pfnd` en `pc/` ni `ds/`) — il est
genere au runtime ou hors modules. L'emprise des liens ne COUVRE rien : la
« definition positive » de la zone jouable n'existe pas sur disque dans ce tag.

## 5. Chiffres du banc (Cliffhanger, cadre du gate, reglages valides)

Reproduction des references a la decimale (l'oracle a REELLEMENT tourne) :

    config              manquants   exces     ACCORD   positions gardees
    aucun tri               4,0 %   149,3 %    38,5 %        95,21 %
    grain 0,005            10,8 %    33,8 %    66,7 %        93,82 %
    coquille sddt          16,7 %   115,8 %    38,6 %        84,91 %
    coquille + grain       18,1 %    31,3 %    62,4 %        84,09 %
    coquille laterale       4,0 %   148,7 %    38,6 %        95,21 %

- La coquille COMPLETE perd 10,3 points de positions : son plafond z = +20 applique
  au TOIT de matiere du champ d'altitude efface des colonnes ou le joueur marche a
  un etage inferieur (meme limitation structurelle que l'accessibilite).
- La coquille LATERALE (4 murs) ne retire que 0,6 point d'exces : les murs sont hors
  du cadre utile. **L'exces de 149 % vit DANS la coquille.**
- Catalyst (oracle faible) : la coquille retire 46,4 % des pixels, 19/19 ancres
  gardent du sol, 19/19 ancres dans la coquille. Elle mord — mais elle garde le
  decor INTERIEUR, et le critere de succes est defini sur Cliffhanger.

## 6. NO-GO — ce qui bloque precisement

Critere annonce : garder >= 95 % des positions jouees ET accord > 66,7 %. Meilleure
configuration sddt : 38,6 % d'accord (coquille laterale, positions sauves) ou
84,91 % de positions (coquille complete). Aucune ne s'approche du critere. La raison
est STRUCTURELLE, pas un defaut de decodage : la coquille delimite le volume de survie,
qui contient l'arene ET son decor visible ; l'exces est du decor DANS le volume de
survie. `pfnd` n'a simplement pas la donnee. C'est le meme echec de principe que la
collision (« a une collision » ne separe pas « praticable » de « mur ») : « on y
survit » ne separe pas « on y marche » de « on le regarde ».

## 7. Portes fermees (ne pas rejouer)

1. **Coquille sddt comme zone jouable** : accord 38,6 % (vs 66,7 % du grain), et la
   version complete perd 10,3 points de positions jouees.
2. **Coquille en complement du grain** : accord 62,4 % — PIRE que le grain seul.
3. **Les 233 volumes comme kill volumes de lisiere** : c'est la riviere (26 x 8 m,
   au nord de l'arene), 0,438 % des positions jouees dedans.
4. **pfnd comme navmesh** : 55/40 liens de franchissement, zero surface, et aucun
   tag pfnd dans pc/ ni ds/.
5. **Reclaimer comme source pour ces tags** : n'implemente ni sddt ni pfnd.

Reste non decode (sans enjeu pour la zone) : les deux blocs de 112 o et leurs chaines
compressees (noms), le header 28 o des triangles-frontieres (type soft ceiling vs
kill), les 2 x 24 o + 1 x 60 o de config pfnd.

## Acquis reutilisables hors zone jouable

- La coquille sddt = CADRE NATUREL d'une carte sans reference : sur Catalyst elle
  encadre l'arene bien mieux que les ancres (x [-33 ; 32] contre un cadre d'ancres de
  129 x 143 m) et retire 46 % de pixels de decor lointain sans percer une ancre.
  Candidate a rejouer comme CADRAGE (pas comme zone) sur les 36 cartes.
- La riviere (233 prismes) est une donnee de gameplay lisible (surfaces d'eau).
- La lecture generique des tags sans plugin (liens type 1 de la struct-table) est
  desormais un outil paye : elle a decode sddt et pfnd en une seance.
