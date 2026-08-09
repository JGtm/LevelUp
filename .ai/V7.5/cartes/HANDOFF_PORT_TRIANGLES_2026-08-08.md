# Handoff — portage de la chaine des TRIANGLES en Go (2026-08-08)

> Ecrit en fin de session, sur avancee critique. Branche `feat/v75`, worktree
> `LevelUp-wt-replay2d`. Plan d'execution : `PLAN_PORT_TRIANGLES_GO.md` (memes dossiers).
> Plan de reference pour le VISUEL : `../../PLAN_BELLE_CARTE_TRIANGLES.md` — c'est LUI qui
> fait foi sur le rendu attendu et sur le gate humain.
>
> **Etat en une ligne** : la chaine decode, l.ORACLE des 29 221 positions a tranche la
> dequantification et le filtre de module, le BORNAGE par la boite de l.instance ramene les
> trous de 11,1 % a 0,8 % ; reste la LISIBILITE du rendu.

## 1. Ce qui EST acquis, avec ses chiffres

| etape | etat | mesure |
|---|---|---|
| lecteur de module | verifie | ridgeline et sgh_streets se reproduisent a l'octet ; mutation `insOffAABB` vue rouge |
| T1 reference `rtgo` | fait | GlobalID a **+8** de `RuntimeGeoMeshReference` ; temoin = ARGMAX STRICT (l'offset 12 resout AUSSI, il ne s'en distingue que par un sur-ensemble strict de 147 instances) |
| index multi-module | fait | 11 modules, 57 251 entrees ; **9 832/10 357 instances (94,9 %)** |
| T2 maillages | fait | `Sections` @64 = tableau des maillages (60 o) ; enfants a `foff = meshIndex x 60` = LOD render data (148 o) ; **1 195 blocs, zero maillage nul** |
| T3 descripteurs | fait | tables reconnues par `size == count * stride` ; blob de ressources reconstitue (`himodule.ResourceBlob`) ; **0 descripteur hors bornes** |
| assemblage | fait | **46,6 M de triangles** en repere monde |
| rendu | **primitive trouvee** | volume + praticabilite + coupe (cf. §3) |

## 1 bis. CE QUE L'ORACLE A TRANCHE (ajout du 2026-08-08, seconde session)

L'oracle a ete joue. Il vit dans `internal/himap/carte_oracle_gamefiles_test.go` et confronte
les **29 221 positions de joueur** du document de rejeu `000d5950.json` a la carte
reconstruite : une position reellement couru avait forcement du sol sous les pieds.

| ce qui etait ouvert | verdict | la mesure qui tranche |
|---|---|---|
| **T4 dequantification** | **u16 BRUT**, definitivement | 63,6 % des positions a moins de 25 cm du sol contre 34,3 % pour `i16 + 32768`, a reglage egal |
| **z du rejeu = les pieds ?** | **oui** | ecart median SIGNE **+0,04 m** — aucun biais de repere a corriger |
| **filtre de module** | **les globaux sont INDISPENSABLES** | avec le seul module de la carte, **11,1 % des positions jouees n'ont AUCUNE matiere sous elles** ; le diagnostic remonte leur geometrie a `common` (7 036), `multiplayer` (2 568) contre `ridgeline` (564), et **0 trou orphelin** |
| **pourquoi les globaux degradaient** | **des maillages qui debordent de leur instance** | debordement median **0,231** de la diagonale de la boite declaree et **42,8** au 99e centile, contre 0,098 et 1,78 pour le module de la carte |

**La correction : le BORNAGE** (`Volume.AddMeshBorne`). La boite monde de l'instance (@0x7C du
sbsp) et le maillage (tag rtgo) viennent de deux sources independantes ; on n'ecrit que les
cellules qui tombent dans la boite. Le bornage porte sur les **cellules ecrites**, pas sur les
sommets — un triangle dont un seul sommet tombe dedans deposerait sinon de la matiere a
l'autre bout de la carte.

    reconstruction                trous   <25cm   <2m    ecart median signe
    module de la carte seul       11,3 %  63,6 %  84,0 %  +0,04 m
    tous modules, SANS bornage     0,2 %  35,9 %  67,3 %  -0,28 m
    tous modules, AVEC bornage     0,8 %  53,2 %  88,0 %  -0,04 m   <- la meilleure carte COMPLETE

**Piege de metrique a ne pas oublier** : le taux « sous 25 cm » est MONOTONE en le degagement
exige — moins on exige, plus il y a de sols, plus la mesure est flatteuse. Il ne peut donc pas
choisir le degagement. La colonne honnete est **`SANS SOL`**, qui ne bouge pas avec le
degagement (11,1 % aux trois valeurs testees) : elle mesure l'absence de matiere, pas la
generosite du critere.

**Le mecanisme, reproduit en miniature** dans `volume_test.go` : un plafond a 1 m au-dessus
d'un sol prive ce sol de son degagement, et c'est LE PLAFOND qui devient praticable. C'est
exactement ce qui se passait en grand quand on versait les modules globaux sans bornage.

### Ce qui reste ouvert cote RENDU

Le bornage regle la JUSTESSE, pas la LISIBILITE. Rendus produits (Bureau) :

    carte_tous_modules_borne.png   tous modules, bande praticable la plus haute -> ENTERREE
    carte_tranche_jeu.png          idem, tranche Z reduite a [-7 ; 11]          -> ENTERREE
    carte_coupe_tous_borne.png     idem, COUPE a z = -1,75 m                    -> structures nettes, bruit de rochers

La regle « bande praticable la plus HAUTE » ramene le decor par-dessus l'arene des que les
globaux sont la. La coupe du prototype reste la bonne primitive d'affichage. **Reste a trouver
la regle qui garde les etages joues sans ramener le decor** — piste : ne dessiner que les
bandes praticables qui portent (ou jouxtent) des positions observees, l'oracle servant cette
fois de filtre et plus seulement de juge.

## 1 ter. SUPPRIMER LA CALIBRATION (2026-08-09)

**Le probleme, pose par l'utilisateur** : regler la tranche d'altitude, l'etage a dessiner et
le cadrage carte par carte est impossible en principe. L'oracle fort n'existe que pour
Cliffhanger — une seule carte a un film decode. Les 29 autres ne pourraient jamais etre
verifiees.

**Le renversement** : on calibre la REGLE une fois sur la carte qui possede les deux oracles,
et on valide la REGLE — pas ses parametres — sur les autres, avec l'oracle FAIBLE des ancres
(`map_objectives.json`, 37 cartes, positions x/y/z dans l'espace de jeu, produites au lot 5).

**Etalonnage de l'ancre**, sans passer par la geometrie reconstruite (14 ancres de Cliffhanger
contre les positions de joueur a leur aplomb) :

    centre de l'ancre -> sol : median +0,29 m   dispersion 0,46 m   <- LE repere
    bas de zone       -> sol : median -0,49 m   dispersion 1,54 m   (down_z varie de 0,5 a 2 m)

D'ou `reference.go` : `AncrageDecalageSol = 0,29`, `PorteeAncre = 25`, `SurfaceReference`
(altitude interpolee sur les ancres) et `Volume.CarteParReference` (par cellule, le sol le plus
proche de cette surface). Le voisinage des ancres borne aussi l'emprise — **51 x 55 m** la ou le
sbsp en declare 113 x 114, et c'est ce meme bornage qui evite d'allouer 4 Go sur les grandes
cartes.

**Disparaissent** : la tranche Z reglee a la main, l'altitude de coupe choisie a l'oeil, le
recadrage. Le degagement redevient une constante physique.

### Deux pistes de rendu REFUTEES — ne pas les rejouer

| piste | verdict | mesure |
|---|---|---|
| **LOD trop grossier** | **innocent** | plafond 40 000 triangles contre illimite : chiffres identiques a l'octet sur 4 configurations. Aucun maillage de ridgeline n'atteint le plafond. |
| **filtre de PENTE** | **refute** | le prototype filtrait par l'inclinaison (`marchable_zmax`). Reintroduit, il resserre le dessin mais l'oracle le condamne : trous 0,8 % -> **4,8 %**, « sous 2 m » 88,0 % -> **82,0 %**, precision inchangee. Il supprime du vrai sol. RETIRE du code. |

### La regle TRANSFERE — balayage par les ancres

Critere : chaque ancre, ramenee au sol, doit trouver un sol a moins d'1 m.

    behemoth        14/14    illusion       14/14    forbidden     14/14
    highpower       25/28    highpower_hv   23/26    hp_sentry     17/20
    catalyst_map    18/19    chasm          14/19    recharge      13/14 (x2 variantes)
    breaker         13/20    breaker_hv     12/22    forest        11/19
    cliffhanger     10/14    catalyst       10/14    launch_site   10/15
    streets         10/14    forest_ranked   9/14    bazaar         8/14
    prism            8/14    aquarius        7/19    aquarius_rkd   6/17
    fragmentation   10/30    fragmentation_hv 7/24
    live_fire        0/14    live_fire_rkd   0/14   <- AUCUNE ancre ne trouve de sol

    BILAN : 27 cartes mesurees · 306/474 ancres (64,6 %)
    9 modules absents de l'installation (corpo, deadlock, oasis, scarr) — non mesurables ici

Catalyst, jamais utilisee pour calibrer quoi que ce soit, se comporte comme Cliffhanger qui a
servi d'etalon. Trois cartes sont a 100 %.

**Ce que le balayage DESIGNE, et c'est son interet** — il ne dilue pas l'echec dans une moyenne :

- **live_fire (sgh_interlock)** : 0 ancre sur 28 trouve un sol, sur les deux variantes.
  **INSTRUIT le 2026-08-09 — CAUSE TROUVEE, hors de notre chaine** : le fichier
  `sgh_interlock-rtx-new.module` ne contient AUCUN tag `sbsp`, et c'est le seul fichier du
  dossier. Sa geometrie n'est pas la ou celle des 26 autres se trouve. Verifie qu'elle n'est
  pas empruntee a une autre carte : son `level_id` (1253388187) ne regroupe que ses deux
  propres variantes, contrairement au partage observe sur les Heavies. **Exception reelle a
  consigner, pas un defaut a corriger** — reste a trouver ou vit sa structure.
- **fragmentation** : 10/30 avec un ecart median de **-1,54 m**, un ordre de grandeur au-dessus
  de toutes les autres — hors du regime « quantification ».
- **aquarius** : 6/17 et 7/19 avec un ecart median NORMAL (-0,15 m). Le defaut n'est donc pas
  une altitude fausse mais une dispersion — profil different des deux precedents. **Cause
  toujours inconnue.**

### Troisieme piste REFUTEE — le choix du bsp

Le diagnostic de live_fire a montre que Cliffhanger declare DEUX bsp : l'arene (113 x 114 m,
10 357 instances) et un horizon (6 619 x 10 471 m, 3 971 instances). La selection retenait
« le plus d'instances » — juste ici, mais rien ne le garantissait ailleurs, et cela ressemblait
fort a la cause du profil d'aquarius (bonne altitude, mauvais endroit).

Regle remplacee par « le bsp qui contient le plus d'ANCRES », avec repli sur le compte
d'instances. **Resultat : 306/474, identique ligne pour ligne.** Sur les 27 cartes, le bsp le
plus peuple est TOUJOURS celui des ancres. La regle est conservee parce qu'elle supprime une
dependance au hasard, mais **elle ne corrige rien** — et la cause d'aquarius reste ouverte.

**Le pont de noms** : le jeu prefixe ses dossiers par le mode d'origine (`ctf_bazaar`,
`btb_highpower`, `sgh_interlock`, `va_behemoth`) quand le catalogue dit « bazaar_map ». On
enumere donc l'installation et on apparie sur les jetons — deviner le prefixe ne marche pas.

**Biais systematique identifie** : l'ecart median tourne autour de -0,3 m, soit une demi-bande.
`SolLePlusProche` rend le CENTRE de la bande quand la surface est vers son bord bas. C'est de la
quantification, pas de la carte — a corriger dans la convention d'altitude.

## 2. T4 — TRANCHEE PAR L'ORACLE (§1 bis). Ce qui suit dit pourquoi les temoins INTERNES avaient echoue

Trois metriques essayees sur les MEMES octets pour departager `u16` brut et `i16 + 32768` :

	ecart aux bornes declarees   : 16,9 mm contre 2,1 mm  -> donne raison a la FAUSSE
	longueur mediane d'arete     : 0,0189 contre 0,0196   -> ne separe pas
	part d'aretes > 1/4 de boite : 0,0821 contre 0,0897   -> ne separe pas

La premiere est BIAISEE (une rotation du quantum disperse les valeurs vers les extremes,
donc epouse mieux les bornes par construction) — **ne pas la reintroduire**. L'echec des
deux autres est instructif : la rotation ne DECHIRE pas les maillages, elle les DECALE
chacun d'une demi-boite ; leur forme reste intacte, seul leur registre mutuel casse.

**L'ORACLE A ETE JOUE le 2026-08-08 et donne raison a `u16` brut (§1 bis).** Rappel de son calage : `ETAT_DU_POC.md` documente que la carte
validee porte **95,6 % des 29 217 positions de joueur tombant sur le sol** (calage
0,0920 m/px, X0 -43,5, Y1 61,0). Rejouer ces positions contre le rendu produit tranche T4 ET
juge le rendu.

## 3. LA PRIMITIVE DE RENDU — lue dans le prototype, pas devinee

Source : `LevelUp-re/scratchpad_recherche/py/` — `s31_raster.py`, `s37_all.py`, `s40_render.py`.
La carte validee est `structure_callouts.png` (= `artefact_map_0.png`, 618 987 octets).

	1. VOLUME : grille 0,10 m en X/Y, bandes de 0,5 m en Z, BORNEE a la tranche de jeu
	2. PRATICABLE : cellule pleine AVEC 2 m de vide au-dessus (4 bandes)
	3. COUPE horizontale a une altitude L, tolerance ±0,75 m

Porte dans `internal/himap/volume.go` (`NewVolumeZ`, `AddMesh`, `Floors`, `Slice`,
`NiveauLePlusPeuple`). Resultat sur Cliffhanger : niveau le plus peuple **z = -1,75 m**,
coupe a 5,5 % de l'emprise ; en variante multi-niveaux (bande praticable la plus haute par
cellule, teintee par l'altitude) **17,5 %** — et l'annexe sud, sa liaison et l'annexe est
apparaissent.

**Pourquoi ca marche alors que mon champ d'altitude echouait** : le volume est borne en Z et
le degagement de 2 m ecarte ce qui est ecrase sous la roche. « Le plus haut » redevient donc
un sol. Le tri se fait dans le VOLUME, pas au moment du dessin — j'ai perdu des heures a
chercher « quel etage » alors que la question ne se pose pas une fois le volume propre.

## 4. LES TROIS OBSERVATIONS UTILISATEUR — ou elles en sont

1. **Un pont manque en bas, a gauche de celui qui est visible.** **LOCALISE** : la carte des
   manques (`ORACLE_PNG`) montre deux trainees rouges verticales en bas au centre — ce sont
   les deux ponts, colonnes sans aucun sol dans le module de la carte. Le bornage + les
   globaux les ramenent (trous 11,1 % -> 0,8 %). **A confirmer a l'oeil sur un rendu lisible.**
   Piste morte a ne pas rejouer : elargir la tranche Z a [-45 ; +45] ne fait PAS reapparaitre
   le second pont — l'utilisateur l'a verifie, c'est un RETRAIT.
2. **Un ilot en haut a gauche, impraticable.** Non confirme.
3. **La partie GAUCHE devrait porter des structures praticables.** **HYPOTHESE CONFIRMEE** :
   sa geometrie est dans les modules globaux (diagnostic `TestDiagnosticTrousDeCarte`,
   0 trou orphelin). Il fallait les integrer AVEC bornage, pas les ecarter.

## 5. LE FILTRE DE MODULE — RESOLU par le bornage

Decouverte D1 : 74 % des instances resolvent dans les modules globaux (`common`,
`multiplayer`, `multiplayer_r3`). C'est **exact**, et c'est ce qui enterrait la carte : un
seul tag de `common` pese 976 instances et 4,3 M de sommets dans la zone centrale (rochers,
debris). Le prototype ne les voyait pas — il ne lisait qu'un module.

**RESOLU le 2026-08-08 (seconde session).** Filtrer sur le module de la carte rendait la carte
lisible mais AMPUTEE : 11,1 % des positions jouees s'y retrouvent sans aucun sol, dont les deux
ponts du sud. La reponse n'est ni « tout garder » ni « tout ecarter » mais **BORNER** — chaque
instance n'ecrit que dans sa propre boite monde declaree. Trous 11,1 % -> 0,8 %, et la justesse
est restauree (53,2 % sous 25 cm contre 35,9 % sans bornage). Cf. §1 bis.

## 6. Ou sont les choses

	internal/himodule/resources.go        blob de ressources (manquait entierement)
	internal/himap/moduleindex.go         index GlobalID -> module (carte + globaux)
	internal/himap/rtgo.go                tag rtgo, PerMeshData
	internal/himap/geometry.go            descripteurs, LOD, dequantification, triangles
	internal/himap/volume.go              volume, praticabilite, coupe   <- LA PRIMITIVE
	internal/himap/heightfield.go         champ d'altitude — APPROCHE ABANDONNEE, cf. §3
	internal/himap/volume_test.go         temoins unitaires : degagement, bornage, dequant
	internal/himap/carte_gamefiles_test.go  produit l'artefact de revue (ecrit un PNG)
	internal/himap/carte_oracle_gamefiles_test.go   L'ORACLE — 29 221 positions jugent la carte
	internal/himap/carte_trous_gamefiles_test.go    d'ou vient la geometrie manquante
	internal/himap/carte_globaux_gamefiles_test.go  les instances globales debordent-elles
	internal/himap/deploy_root.go         ou est installe le jeu (LEVELUP_HALO_DEPLOY)

Commande : `PROBE_MULTI=1 PROBE_PNG=<sortie.png> go test ./internal/himap/ -run TestCarteCliffhanger -count=1 -v`

Images de la session (hors depot, Bureau) : `cliffhanger_CARTE_multiniveaux.png`,
`cliffhanger_CARTE_z45.png`, `cliffhanger_POINTS_carte_seule.png`, et la reference
`.ai/V7.5/dumps/carte_validee_v1.png`.

## 7. Ordre de reprise (mis a jour le 2026-08-08, seconde session)

Les points 1 a 3 de l'ordre precedent sont FAITS (oracle joue, globaux reintegres via le
bornage, ponts localises). Ce qui reste :

1. **La regle d'affichage.** La justesse est acquise, pas la lisibilite : « bande praticable
   la plus haute » ramene le decor, la coupe a une altitude fixe perd les autres etages.
   Piste chiffree a essayer : ne retenir que les bandes praticables au voisinage des positions
   observees — l'oracle devient un FILTRE de rendu, pas seulement un juge. Verifier ensuite que
   les deux ponts et la partie gauche sont visibles a l'oeil.
2. **Le gate visuel** de `PLAN_BELLE_CARTE_TRIANGLES` : artefact cote a cote avec
   `carte_validee_v1.png`, a la meme echelle. Toujours pas produit.
3. **Le degagement.** `HeadroomBands = 4` vient du prototype ; l'oracle ne peut pas le choisir
   (metrique monotone, cf. §1 bis). Le trancher demande un critere independant — par exemple la
   hauteur de collision d'un Spartan, ou une mesure de plafond sur les positions observees.
4. **Dette restante** : `heightfield.go` est une approche abandonnee toujours au depot (son
   en-tete le dit) — la supprimer si rien ne la rappelle ; `carte_gamefiles_test.go` n'affirme
   toujours rien (c'est assume : il produit l'artefact, le juge est l'utilisateur).

## 8. Ce que cette session a appris, au-dela du code

- **Un temoin qui ne departage pas ne teste rien.** Trois fois de suite ici : le premier
  temoin de T1 passait avec le mauvais offset, l'ecart aux bornes de T4 donne raison a la
  mauvaise lecture, et deux metriques de coherence ne separent rien.
- **Avant de soupconner la donnee, verifier son AFFICHAGE.** Trois essais de plafond
  depenses sur un probleme de geometrie qui etait un probleme de dessin (echelle min/max au
  lieu de centiles, aucun ombrage).
- **Ajouter de la donnee peut degrader le resultat.** La resolution multi-module est juste
  et elle a rendu la carte illisible. « Plus complet » n'est pas « meilleur ».
- **Lire le prototype avant de re-deriver.** Une heure de lecture de `s31/s37/s40` aurait
  evite une journee de tatonnement sur le rendu.
- **Le temoin qui tranche vient de DEHORS.** Toutes les statistiques calculees sur la
  geometrie elle-meme ont echoue a departager quoi que ce soit. Les positions de joueur, qui
  ne savent rien de la quantification ni des modules, ont tranche deux questions ouvertes en
  une seule passe de 21 secondes. Elles etaient disponibles depuis le debut.
- **Se mefier d'une metrique monotone en son propre reglage.** « Sous 25 cm » s'ameliore
  mecaniquement quand on exige moins de degagement : l'optimiser aurait conduit a supprimer le
  critere. La colonne `SANS SOL`, insensible au reglage, est la seule qui mesure la carte.
- **Un test rouge peut etre une decouverte.** Le temoin unitaire du degagement a echoue parce
  qu'il reproduisait fidelement le phenomene reel — le plafond devient praticable quand il
  disqualifie le sol. C'etait le diagnostic, pas un bug de test.
