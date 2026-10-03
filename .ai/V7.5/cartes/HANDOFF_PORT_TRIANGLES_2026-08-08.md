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

---

## 9. ETAT AU 2026-08-09 SOIR — LE RENDU EST TROUVE, DEUX MORCEAUX MANQUENT

### 9.1 Ce qui est acquis et commite

**La carte est un RENDU DE MAILLAGE vu du dessus, pas une carte de praticabilite.**
`internal/himap/rendu.go` (commit `4cd36e5b9`) : z-buffer par pixel, normale de la face
retenue, eclairage de Lambert oblique. Trois lignes, aucun reglage par carte. Au calage de la
carte validee (0,0920 m/px, X0 -43,5, Y1 61,0), l'arene est SUPERPOSABLE : plateforme
circulaire, batiments, structure diagonale, annexe sud aux memes formes et memes positions.

Le meilleur rendu est `rendu_v2.png` : module de la carte SEUL, 2 730 instances, 17,8 % de
pixels. Le comparatif projette la reconstruction DANS la grille de pixels de la reference —
deux images decalees de deux metres se comparent mal, l'oeil comble les ecarts.

**Deux erreurs de methode corrigees ce jour, a ne pas refaire :**
- « l'arene est illisible sous les rochers » a ete lu comme un probleme de SELECTION alors que
  c'etait un probleme d'ECLAIRAGE. Deux jours perdus a construire volume / degagement / ancres.
  Le calcul juste existait des le premier jour.
- « le LOD est innocent » (refutation du 08/08) etait FAUX. Le defaut n'etait pas le PLAFOND de
  triangles mais l'ORDRE : on retenait le PREMIER LOD sous le plafond au lieu du plus FIN. Le
  test mesurait le bon parametre au mauvais endroit et passait vert pour de mauvaises raisons.

### 9.2 CE QUI MANQUE, ET LES CINQ PORTES FERMEES

L'utilisateur signale deux manques sur `rendu_v2.png` : **le second pont en bas a gauche** et
**une zone en haut a gauche non rendue**. Ce sont les grandes DALLES PLATES de la reference.

| hypothese | mesure | verdict |
|---|---|---|
| modules globaux | 86,7 % de couverture, arene noyee | non |
| maillages aberrants (bornage a la boite) | 86,5 % | sans effet |
| decor en surplomb (plafond deduit des ancres, +6,95 m) | 85,3 % | sans effet |
| instances non resolues | 525, petits accessoires disperses (verifie a l'image) | non |
| second bsp du module | 100 %, dalle uniforme de fond de scene | non |

Diagnostic complementaire : **0 maillage nil, 0 instance `QuickDeleted`**. Rien n'est ecarte
par erreur dans le chemin actuel.

### 9.3 LA DETTE DE METHODE — a solder AVANT toute nouvelle hypothese

**Le code de Reclaimer n'a jamais ete lu dans ce chantier.** Quand l'utilisateur a demande de
repartir de Reclaimer, la recette du z-buffer a ete INFEREE de l'image validee, pas lue dans la
source. Les cinq hypotheses ci-dessus sont des suppositions — mesurees et honnetement
refutees, mais des suppositions. Y compris la sixieme, non testee : « les dalles viendraient de
la geometrie NON INSTANCIEE du sbsp ». Elle est plausible et n'est adossee a aucune source.

Nuance a garder sur l'altitude : seul le PLAFOND a ete teste. Ni plancher, ni tranche fermee.
L'altitude n'est donc pas refutee, seulement une de ses formes.

### 9.4 ORDRE DE REPRISE — commencer par LIRE, pas par supposer

1. **Ouvrir les sources Reclaimer** (`scratchpad/refs/*.cs`, cf. §6 du handoff du 26/07 ;
   sinon les retelecharger depuis `Gravemind2401/Reclaimer`) :
   `Reclaimer.Blam/Blam/HaloInfinite/ScenarioStructureBspTag.cs` et `RuntimeGeoTag.cs`.
   Etablir la LISTE de ce que Reclaimer lit dans un sbsp et que notre chaine ne lit pas — nous
   n'avons porte que le bloc `instanced geometry instances` a l'offset 420.
2. Seulement ensuite, decider quoi rendre. Ne pas rouvrir les cinq portes du §9.2.
3. Le gate reste le comparatif au pixel pres avec `carte_validee_v1.png`
   (`rendu_gamefiles_test.go`).

### 9.5 Etat du depot

Commite et pousse : `rendu.go`, le comparatif, l'oracle, le balayage des 27 cartes, les
diagnostics. NON commite a l'arret : les sondes du §9.2 (bornage et plafond dans le rendu,
compteurs de causes, empreinte des instances non resolues). Elles ne servent plus le rendu mais
portent les mesures qui ferment les portes — les conserver comme instruments, ou les retirer en
gardant les chiffres ici.

---

## 10. ETAT AU 2026-08-09 (seconde session) — LA DETTE DE LECTURE EST SOLDEE

Le §9.4 ordonnait de LIRE Reclaimer avant de supposer quoi que ce soit. Fait :
`Reclaimer.Blam/Blam/HaloInfinite/ScenarioStructureBspTag.cs` et `RuntimeGeoTag.cs`
(`LevelUp-re/scratchpad_recherche/refs/`), croises avec le plugin `sbsp.xml`. **Les deux
concordent a l'octet pres sur les 320 d'une instance**, et deux offsets independants le
confirment sur nos tags : `clusters` @300 = 0x12c et `instanced geometry instances`
@420 = 0x1a4, tous deux retrouves par l'appariement par rang du plugin. La chaine de lecture
n'etait pas fausse — elle etait INCOMPLETE.

### 10.1 LA LISTE demandee au §9.4 point 1

`ScenarioStructureBspTag` :

| champ | offset | notre chaine |
|---|---|---|
| `Clusters` | 300 | **non lu** — voir §10.2, la porte se ferme par la mesure |
| `GeometryInstances` | 420 | lu |

`BspGeometryInstanceBlock` (320 o) :

| champ | offset | notre chaine |
|---|---|---|
| `TransformScale` (Vector3) | 0x00 | lu, **JAMAIS APPLIQUE** -> corrige, §10.3 |
| `Transform` (3x4) | 0x0C | lu (forward / left / up / position) |
| `RuntimeGeoMeshReference` | 0x3C | lu (GlobalID a +8) |
| `MeshIndex` | 0x74 | lu |
| `BoundsIndex` | 0x76 | lu sous le nom du plugin, `unique io index` — Reclaimer ne s'en sert pas non plus |
| `Material` | 0xF0 | non lu — materiaux, hors geometrie |
| `FlagsOverride` | 0x110 | **non lu** -> porte, §10.4 |
| `Guid` | 0x12C | non lu — identite d'edition |

`RuntimeGeoTag` — seul champ encore non porte : **`MeshResourcePackingPolicy` @186**
(`SingleResource` vs `ResourcePerPermutation`). Nous concatenons les ressources en un blob
unique ; si un tag declarait une ressource par permutation, les offsets de descripteurs
seraient relatifs a chaque ressource. Aucun symptome mesure a ce jour (0 maillage nil,
0 descripteur hors bornes) — inscrit au registre des reports.

**Et les regles de rendu de `GetContent()`**, qui valaient autant que les offsets :
sauter l'instance sans tag, sauter le `MeshIsCustomShadowCaster`, appliquer
`SetTransform(TransformScale, translation, rotation)`. Les deux dernieres nous manquaient.

### 10.2 SIXIEME PORTE FERMEE — la geometrie non instanciee n'existe pas ici

L'hypothese non testee du §9.3 (« les dalles viendraient de la geometrie NON INSTANCIEE du
sbsp ») est **REFUTEE par la mesure**, pas par un raisonnement. Inventaire des tag-blocks
racine du sbsp de Cliffhanger (`TestReclaimerBlocsRacine`) :

    clusters                  count=1        <- une seule entree
    meshes                    count=0        <- AUCUN maillage non instancie
    compression info          count=0
    mesh resource groups      count=0
    raw_resources             count=0
    instanced geometry instances  count=10357

Toute la geometrie de rendu passe par les instances. Reclaimer dit la meme chose de son
cote : il DECLARE `Clusters` et ne le rend jamais. La porte est fermee.

### 10.3 CE QUI ETAIT FAUX DEPUIS LE DEBUT — l'echelle d'instance

`instances.go` portait « Scale : champ @0x00. Repute vestigial ». **Il ne l'est pas.**
Reclaimer l'applique separement de la rotation, et deux faits independants l'exigeaient :
la base est ORTHONORMEE (temoin `TestRidgelineInstancesGeometry`, |v|=1 a 1e-3 sur les
10 357 instances), donc l'echelle ne peut pas y etre bakee.

Mesure (`TestReclaimerEchelleInstance`) : **12 009 des 14 328 instances portent une echelle
differente de 1**, de -38,86 a +116,33 (les valeurs negatives repondent au drapeau
`negative scale` de `flags2`).

**LE TEMOIN QUI SEPARE** — ecart du maillage transforme a la boite monde DECLAREE de
l'instance (@0x7C), source independante du maillage, en fraction de sa diagonale :

    5 817 instances            median   p90
    sans echelle               0,2238   0,3785
    avec echelle               0,0665   0,1550     <- 3,4x mieux

    restreint aux 4 641 instances a scale != 1
    sans echelle               0,2622   0,4408
    avec echelle               0,0816   0,1529

`LocalToWorld` applique desormais l'echelle ; `LocalToWorldSansEchelle` ne survit que comme
lecture concurrente pour ce temoin (meme role que `DequantSigne`).

**Piege pose par ce changement** : une `Instance{}` litterale a maintenant `Scale = (0,0,0)`
et ecrase tout le maillage sur sa position. `instanceIdentite()` des tests unitaires porte
desormais un scale unitaire explicite.

### 10.4 30 % DES INSTANCES SONT DES PROJECTEURS D'OMBRE

`mesh flags override` @0x110, bit 3 `mesh is custom shadow caster` : **4 343 des 14 328
instances**. Reclaimer les saute — leur maillage est un volume de projection d'ombre, pas de
la geometrie visible.

Que l'offset soit le bon se VERIFIE : le champ prend **22 valeurs distinctes et aucune ne
sort des 11 drapeaux declares** par le plugin. Un offset faux etalerait les valeurs sur
65 536. Et la confirmation vient du rendu : ecarter ces instances retire **47 % des
instances rendues** (2 730 -> 1 433) pour **0,1 point de couverture** (14,2 % -> 14,1 %).
Elles etaient donc redondantes avec de la vraie geometrie — c'est exactement ce qu'est un
proxy d'ombre.

### 10.5 Ce qui a change dans le rendu, et ce qui reste au juge

`rendu_gamefiles_test.go` prend desormais le bsp DESIGNE PAR LES ANCRES (`choisitBSP`, deja
la regle partout ailleurs) au lieu de « tous les bsp » — prendre tous les bsp ramenait
l'horizon de 6 619 x 10 471 m, dont la dalle de fond de scene couvre 100 % du cadre, porte
deja fermee au §9.2.

Trois artefacts produits, chaque correction isolable (variables `RENDU_SANS_ECHELLE`,
`RENDU_GARDE_OMBRES`) :

    A  lecture d'avant           2 730 instances rendues   15,7 % de pixels
    B  + echelle                 2 730                     14,2 %
    C  + ombres ecartees         1 433                     14,1 %

A reproduit exactement l'etat de `rendu_v2.png` (2 730 instances) : la comparaison est
attribuable.

**LE GATE VISUEL RESTE DU. C'est le juge, et il n'est pas dans cette session** — les deux
manques signales par l'utilisateur (second pont en bas a gauche, zone en haut a gauche) se
constatent a l'oeil sur le comparatif, pas sur un pourcentage de couverture. La couverture
n'est pas le critere : elle baisse de A a C alors que la geometrie est plus juste.

### 10.6 Ce qui n'a PAS ete touche, et pourquoi

Le plafond deduit des ancres (+6,95 m) reste actif : c'etait la configuration commitee, et
le modifier aurait rendu le A/B inattribuable. Son verdict du §9.2 (« sans effet ») a ete
rendu sur une geometrie fausse — il pourra etre rejoue, mais APRES le gate visuel, pas avant.

Aucune des cinq portes du §9.2 n'a ete rouverte.

### 10.7 LE PLANCHER MANQUAIT — et c'est lui qui vidait la carte

Gate visuel REFUSE par l'utilisateur : ni le second pont ni la zone haut-gauche. Sa question
— « t'as bien regarde l'altitude, si tu zappais pas des elements que tu supposes trop bas ? »
— etait la bonne, et elle a demande UN INSTRUMENT pour y repondre.

**L'instrument qui manquait** (`rendu_diff_gamefiles_test.go`) : comparaison PIXEL A PIXEL
avec la carte validee. Trois chiffres — matiere de la reference, MANQUANTS, EXCES — et une
troisieme vignette rouge/bleu. Jusque-la le rendu se jugeait sur une couverture globale, qui
peut monter pendant qu'un pont disparait.

Premiere mesure, et elle disqualifie tout ce qui precede : **la reference couvre 23,1 % du
cadre, nous 14,1 %, et il manque 52,1 % de sa matiere.**

**Ce que l'instrument a designe :**

| configuration | manquants | exces |
|---|---:|---:|
| module de la carte seul (l'etat commite) | **52,1 %** | 13,4 % |
| tous modules, sans tranche | 0,8 % | **328 %** |
| tous modules, tranche [-12 ; +28] | **4,0 %** | 149 % |

La geometrie du pont ET de la zone haut-gauche est donc dans les MODULES GLOBAUX. Le filtre
« module de la carte seul » retirait la moitie de la carte.

**Et l'altitude, exactement comme le disait l'utilisateur.** Repartition de ce que nous
dessinons, tous modules, sans tranche :

    sous -10 m        ~30 000 px justes   ~807 000 EN TROP   <- le decor, a 96 %
    -10 a +35 m      ~370 000 px justes   ~530 000 EN TROP
    au-dessus +35 m    ~2 500 px justes    ~56 000 EN TROP

**Le prototype le portait depuis le debut** : `s31_raster.py` borne son volume a
`ZB0, ZB1 = -12.0, 28.0`, et le handoff §3 l'ecrivait noir sur blanc — « volume BORNE a la
tranche de jeu ». La traduction en z-buffer (`rendu.go`) n'a retenu que le PLAFOND. D'ou
`Rendu.Plancher` / `Rendu.Tranche`.

Le balayage confirme le reglage du prototype : plancher -12 (le remonter a -6 fait passer les
manquants de 4,0 a 7,3 %), plafond entre +20 et +28 indifferent. **Corollaire : le plafond
« deduit des ancres » a +6,95 m etait FAUX** — 118 000 pixels justes vivent entre +10 et
+35 m, soit 31 % de la reference.

**Confirmation du §10.3 par la source qui a produit la reference** : `geo2.py`
(`instance_matrix`) porte en toutes lettres « convention vecteur-ligne, **scale ACTIF** ». Le
prototype appliquait l'echelle ; seul le portage Go l'avait perdue.

### 10.8 CE QUI RESTE OUVERT — l'exces de 149 %

Nous dessinons encore une fois et demie la matiere de la reference en trop. Ce n'est PAS une
bande d'altitude (l'exces se repartit sur toute la tranche) : c'est du decor LATERAL, le
relief autour de l'arene.

**Piste refutee sur pieces, ne pas la rejouer** : borner l'emprise au voisinage des ancres
(`PorteeAncre`, la regle de `reference.go`). Sur Cliffhanger elle rend 100 x 100 m — les
ancres sont trop dispersees pour cadrer quoi que ce soit — et n'ecarte que **92 instances**,
exces inchange (149,3 -> 149,2 %).

**Piste non testee, la plus plausible** : le prototype echantillonne des POINTS
(voir §10.10 — elle reste la meilleure candidate).

### 10.9 GATE VISUEL PASSE — et l'utilisateur a donne LE TEMOIN de ce qui reste

Validation du 2026-08-09 : « je vois le second pont et les autres zones jouables. Je valide
le rendu. » Le rendu par defaut est donc bascule sur la configuration validee — **tous les
modules, tranche `[TrancheDeJeuMin ; TrancheDeJeuMax]` = [-12 ; +28]** — et l'ancien defaut
ne survit que comme temoin (`RENDU_CARTE_SEULE`, `RENDU_TRANCHE`).

**LE TEMOIN, et il vient de l'utilisateur, pas de la session** (regle du gate visuel) :

> « la carte affiche une zone bien plus grande que la zone jouable, qui correspond a la zone
> GRISE ET ROUGE sur le rendu, la partie BLEUE n'etant pas jouable »

Autrement dit, dans la carte des ecarts : **gris + rouge = la zone jouable ; bleu = du decor**.
C'est la definition operationnelle qui manquait — la silhouette de la carte validee EST la
zone jouable, et nos 149 % d'exces sont tous hors-jeu. Un critere de zone jouable se calibre
desormais contre ce temoin, il ne s'invente plus.

Second retour utilisateur : il reste **des trous au milieu de la zone de jeu** (les 4,0 % de
manquants), qu'un FOND sous la carte masquerait — le rendu ecrit du noir la ou il n'y a pas de
matiere. A traiter comme une question de presentation, pas de geometrie.

**Piste REFUTEE ce jour — le module n'est pas le discriminant.** Contribution de chaque module,
mesuree seule, tranche posee :

    module        couverture   couvre de la reference   exces
    ridgeline        12,8 %            50,3 %            5,3 %
    common           25,7 %            60,2 %           51,2 %
    multiplayer      38,3 %            47,5 %          118,3 %
    multiplayer_r3    1,1 %             4,4 %            0,4 %

`multiplayer` est le premier pourvoyeur de bleu — mais il apporte AUSSI 47,5 % de la vraie
carte. Aucun module n'est « le decor » : ils se completent. Trier par module est impossible.

### 10.10 ORDRE DE REPRISE

1. **Le fond.** Les trous au milieu de la zone jouee se masquent par un fond, pas par de la
   geometrie. Decision de presentation.
2. **La zone jouable.** Trouver le critere qui garde le gris+rouge et jette le bleu. Ce qui
   est deja REFUTE et ne doit pas etre rejoue : le module (§10.9), l'emprise par les ancres
   (§10.8), la tranche d'altitude seule (l'exces se repartit dessus). Ce qui reste :
   - verifier d'abord l'hypothese de RASTERISATION (§10.8) — si la silhouette de la reference
     est un effet du semis de points du prototype, il n'y a aucune regle de tri a chercher ;
   - sinon, une regle d'ACCESSIBILITE : composante connexe de sol praticable atteignable
     depuis les ancres. Physique, sans reglage par carte, et calibrable sur le temoin
     ci-dessus. Attention : le filtre de PENTE seul est deja refute (§1 ter).
3. **Generaliser la tranche.** [-12 ; +28] est mesure sur UNE carte (registre des reports).

### 10.11 LA ZONE JOUABLE — LA FINESSE DU MAILLAGE SEPARE

Les deux voies demandees par l'utilisateur ont ete jouees. Un score unique les arbitre :
**l'ACCORD = intersection / union** avec la silhouette de la carte validee. Il ne se laisse pas
gagner d'un cote aux depens de l'autre — tout dessiner rend 0 manquant et un accord
catastrophique, ne rien dessiner l'inverse.

**Voie A — masquer par la silhouette de la reference.** Produite (`RENDU_MASQUE_REFERENCE`).
**A ECARTER**, et pas seulement parce qu'elle ne vaut que pour Cliffhanger : elle est
VISUELLEMENT mauvaise. Bords dechiquetes, decoupes rectangulaires, et les gros blocs de decor
qui tombent dans la silhouette sont conserves — le masque ne sait pas ce qu'il garde.

**Voie B — filtrer par la FINESSE du maillage.** Observation faite sur le rendu : le decor est
fait de facettes enormes, l'arene est finement trianglee. Mesure de l'aire projetee du triangle
MEDIAN par instance : **p50 = 0,0001 m2, p90 = 0,0161, p99 = 1,8739** — quatre ordres de
grandeur, le decor est tout entier dans la queue.

    seuil (m2)   manquants   exces    ACCORD
    aucun            4,0 %   149,3 %   38,5 %
    0,002           26,6 %    15,2 %   63,7 %
    0,003           18,3 %    21,2 %   67,4 %   <- optimum d'accord
    0,005           10,8 %    33,8 %   66,7 %   <- RETENU
    0,008            7,9 %    49,1 %   61,7 %
    0,012            7,9 %    59,9 %   57,6 %

**0,005 m2 est retenu plutot que l'optimum strict** : a 0,7 point d'accord pres, il garde
89,2 % de la zone jouable contre 81,7 %. Perdre de la structure jouee est le defaut que
l'utilisateur signale depuis le debut ; l'exces, non.

Un triangle de 50 cm2 fait ~10 cm de cote : les surfaces jouees sont modelisees a ce grain, le
decor lointain non. C'est une propriete du MAILLAGE, pas un reglage d'image — d'ou son interet
face au masque.

**CE QU'IL FAUT ENCORE PROUVER, et c'est la meme methode qu'au §1 ter** : le seuil est calibre
sur la seule carte qui possede une reference. La REGLE doit etre validee sur les autres par
l'oracle FAIBLE des ancres, et le seuil ne doit pas se retoucher carte par carte. Tant que ce
balayage n'a pas tourne, 0,005 est une valeur mesuree sur un cas, pas une constante.

**VALIDE PAR L'UTILISATEUR le 2026-08-09** (« je valide carte B finesse »). Le tri est donc
sorti du test et vit dans `zone_jouable.go` — `AireMaxTriangleJouable`, `AireMedianeProjetee`,
`EstDecorGrossier` — avec ses temoins unitaires. Il est ACTIF PAR DEFAUT ; `RENDU_AIRE_MAX=0`
le desactive pour la comparaison.

**Piege paye au passage, et c'est la meme lecon que le §8** : le premier jeu de temoins du tri
ne separait PAS. Sur des grilles uniformes, moyenne et mediane coincident — la mutation
« mediane -> moyenne » passait au VERT alors que le commentaire affirmait le contraire. Le cas
qui separe est celui que la doc decrit : un sol finement maille pose sur de grandes faces de
socle (`TestMedianeResisteAuSocle`). **Ecrire « cette mutation doit faire rougir » ne suffit
pas — il faut la jouer.**

### 10.12 LA NETTETE — trois leviers, aucun reglage par carte

« Ca manque un peu de nettete » (utilisateur, 2026-08-09), avec deux pistes nommees : une
echelle de couleur fonction de l'altitude, et un rendu plus plat. Le suspect etait le bon —
l'eclairage de `rendu.go` est un Lambert avec ambiante hemispherique `0,25 + 0,75 x d` : il ne
descend jamais sous 25 %, sature vite, donc tout se tasse vers le blanc.

`rendu_couleur.go` porte trois leviers, tous CONSTANTS, jamais regles par carte :

| levier | ce qu'il apporte | valeur |
|---|---|---|
| **paliers** | des aplats au lieu d'un degrade continu ; les faces se separent | `PaliersEclairement = 5` |
| **aretes** | une rupture d'ALTITUDE entre voisins, pas une rupture de normale | `SeuilAreteMetres = 0,5` |
| **nuancier d'altitude** | teinte par la hauteur, bornes aux centiles 2/98 | rampe sequentielle |

**Pourquoi les aretes portent sur l'altitude et non sur la normale** : deux dalles horizontales
a deux hauteurs ont exactement le meme eclairement — l'ombrage ne peut pas les separer, quel
que soit son reglage. C'est ce que le temoin `TestAreteReveleCeQueLOmbrageCache` verifie
d'abord, avant meme de tester l'arete.

Le seuil de 0,5 m est PHYSIQUE : a ~9 cm par pixel, une marche d'un demi-metre n'est pas une
pente, c'est un rebord ; en dessous, un Spartan franchit sans sauter.

**Bornes du nuancier : centiles 2/98, jamais min/max.** Lecon deja payee le 2026-08-08 — une
seule cellule a -131 m ecrasait toute la carte dans deux nuances de blanc. Sur Cliffhanger les
bornes robustes valent [-10,97 ; +25,50] m.

Quatre styles produits pour le gate : `doux` (l'existant), `plat`, `altitude`, `combine`
(variable `RENDU_STYLE`). **Le choix revient a l'utilisateur.**

**DEUX TEMOINS TAUTOLOGIQUES CORRIGES CE JOUR, meme lecon que le §8 :**
- le triangle aberrant du temoin des centiles etait pose SOUS une dalle — le z-buffer garde la
  surface la plus haute, donc la cellule aberrante n'existait jamais. Mutation min/max verte.
- une fois la cellule rendue visible, l'echantillon de 50 cellules rendait le 2e centile egal
  a la cellule aberrante elle-meme : le temoin condamnait du code juste. Il porte desormais
  20 000 cellules, l'ordre de grandeur d'une vraie carte.

C'est la quatrieme fois que ce chantier bute sur un temoin qui ne teste rien. La regle tient en
une ligne : **une mutation annoncee dans un commentaire doit etre JOUEE**.

Style `combine` VALIDE par l'utilisateur le 2026-08-09 — c'est le defaut (`StyleCarteParDefaut`).

## 11. LA DEUXIEME CARTE — CATALYST PASSE, ET REFUTE LA REGLE DE ZONE JOUABLE

`rendu_carte_gamefiles_test.go` rend une carte QUELCONQUE : cadre deduit des ancres, aucune
reference, juge = l'oracle FAIBLE des ancres (chaque ancre d'objectif a-t-elle du sol dessine
sous elle ?). C'est le renversement du §1 ter applique au rendu.

    carte                  instances   dessinees   ecartees   ancres avec sol   ecart median
    cliffhanger_ridgeline     10 357       5 102        859       14/14 (100 %)      -0,32 m
    catalyst_map              11 468       7 796        162       19/19 (100 %)      -0,29 m

**La geometrie transfere sans retouche.** Catalyst n'a jamais servi a calibrer quoi que ce
soit : la tranche de jeu, l'echelle d'instance, le tri des projecteurs d'ombre et l'ecart
median de -0,3 m (le biais de quantification connu) s'y comportent a l'identique.

**MAIS LE TRI DE LA ZONE JOUABLE NE TRANSFERE PAS**, et Catalyst le refute du premier coup :
**162 instances ecartees contre 859**. Ce n'est pas un seuil a retoucher, c'est le CRITERE qui
ne s'applique pas. Cliffhanger est une carte de ROCHE — son decor est un relief modelise
grossierement, donc le grain le designe. Catalyst est une STATION SPATIALE : ses abords sont de
l'architecture, modelisee au meme grain que l'arene. Le rendu montre l'aire de jeu correcte,
noyee dans des structures sombres qui la debordent largement.

Conclusion, et elle etait annoncee au registre : la regle du grain est valide sur les cartes
naturelles et muette sur les cartes construites. Il faut un critere qui vaille pour les deux —
piste la plus plausible, l'ACCESSIBILITE (composante connexe de sol praticable depuis les
ancres), qui ne suppose rien du materiau.

### 11.1 VAGABOND — NE PAS ESSAYER, c'est un autre chantier (mesure)

    fo08_wetland  bsp #2  : 13 281 instances sur 3 867 x 3 662 m   <- la TOILE Forge
    fo08_wetland  bsp #15 :    814 instances sur   463 x  453 m

La carte de Vagabond n'est pas dans le bsp : elle vit dans les **4 709 objets Forge du
`.mvar`** (mesure du lot 5). La chaine actuelle rendrait la toile, pas la carte. C'est l'etape 2
du plan (§4 : F1 `type_id -> tag de modele`, F2 poser les triangles du modele par la
transformation de chaque objet), pas une variante du rendu.

S'ajoute un blocage d'entree : Vagabond n'a pas de nom de module exploitable au catalogue
d'objectifs — son entree porte le module generique `map`. Sans ancres resolvables, ni cadre
ni oracle.
(`s31_raster.py`, sommets + points barycentriques a densite proportionnelle a l'aire projetee,
budget de triangles proportionnel a l'empreinte au sol, plafonne a 40 000). Un enorme maillage
de relief lointain recoit donc un budget SATURE, donc une densite de points tres faible par
metre carre, donc des cellules vides — la silhouette de la reference pourrait etre un effet de
son mode de rasterisation, pas d'un filtre de selection. A verifier avant d'inventer une regle
de tri.

## 12. LOT RIVIERE (2026-08-10) — le lecteur `sddt` est en PRODUCTION, l'eau est dessinee

### 12.1 Ce qui a change

- **`internal/himap/sddt.go`** : lecteur du tag `sddt` promu des sondes. Les DEUX natures
  sont rendues separement, et le critere est STRUCTUREL (deux champs distincts du root
  block : coquille-frontiere via `@0x58 -> @0x08`, volumes d'eau via `@0x94`) — pas un
  seuil sur le nombre de plans. La navigation generique de struct-table (meilleurTagInfo,
  liensBlocs, compteChamp) est promue avec lui ; les sondes restantes (food/scnr/stse)
  s'en servent.
- **L'eau sur le fond de carte** : `Rendu.PoseEau` marque les cellules couvertes par un
  volume d'eau SANS toucher le z-buffer (l'eau est un habillage). Regle du pont : une
  matiere au-dessus du toit du volume (+`MargeEauMetres` = 0,25 m) garde son pixel. La
  couleur vit dans `rendu_couleur.go` (`TeinteEau`, bleu desature + berge assombrie),
  appliquee par `carteSeulePNG` — y compris sur les cellules SANS matiere (l'eau n'est pas
  dans les instances de rendu, elle comble ses propres trous).
- **Sondes supprimees** (double emploi apres promotion) : `sonde_sddt_pfnd`,
  `sonde_sddt_banc`, `sonde_sddt_xy`. Leurs mesures vivent dans
  `INVESTIGATION_SDDT_PFND_2026-08-10.md` ; les temoins promus dans
  `sddt_test.go` (synthetiques, mutation d'orientation JOUEE dans le test) et
  `sddt_gamefiles_test.go` (octets reels).
- **LE BANC est promu** : `TestBancCliffhanger` (sddt_gamefiles_test.go) remplace
  `TestSondeSddtRendu`. Il ASSERT les references (accord >= 66,6 %, positions >= 93,8 %)
  et PROUVE que l'eau ne touche pas le terrain (z-buffer et normales compares a l'octet).

### 12.2 Les chiffres du banc — inchanges a la decimale

    config par defaut (grain 0,005, tranche, tous modules)
    manquants 10,8 % · exces 33,8 % · ACCORD 66,7 % · positions gardees 93,82 %
    233 volumes d'eau -> 5 467 cellules d'eau, terrain intact (compare a l'octet)

Oracle sddt rejoue en production : coquille 29 221/29 221 positions, volumes d'eau
128/29 221 (0,438 %), orientation 233/233 contre 0 en sens inverse, residu max 0,0000.

### 12.3 Generalisation — l'eau sddt est une feature GENERALE

Balayage des 31 cartes installees (`TestSddtBalayageEau`) : **10 cartes portent des volumes
d'eau** — ridgeline 233, ctf_forbidden 208, forest 51, btb_drydock 26, btb_exiled 12,
btb_engine 10, va_launchsite 6, btb_highpower 2, sgh_blueprint 2, sgh_crystalcaves 2.
29/31 portent une coquille-frontiere (exceptions : academy_tutorial et sgh_interlock, ce
dernier deja connu sans sbsp). Les canevas Forge (fo*) n'ont AUCUNE eau sddt : l'eau des
cartes Forge, si elle existe, vit ailleurs (probablement dans les objets du .mvar).

### 12.4 Note de session — ecriture concurrente dans le worktree

Pendant le lot riviere, une AUTRE session a depose `cloture_gamefiles_test.go` (hypothese
utilisateur « environnement ferme » : inondation depuis les bords) et `Volume.ZoneClose`
dans `zone_jouable.go`. Ce travail n'est PAS commite par ces lots ; ses deux appels a
l'ancienne API de sonde (`litSddt`) ont ete migres vers l'API promue (`LitSddt`,
`Coquille()`, `Contient`) pour que l'arbre compile. Son verdict reste a rendre par sa
session.

## 13. LOT PROTOTYPE FORGE (2026-08-10) — F1 FERME, VAGABOND SE REND

### 13.1 F1 — le chainon `type_id -> modele` est FERME, et il est INLINE

Trois mesures successives (sondes `sonde_forge_gamefiles_test.go`, `sonde_forge_mvar_...`) :

1. **La table de dependances des tags food est VIDE** : 457/467 type_id de Vagabond n'ont
   AUCUNE dep (les 10 restants pointent foki/fosp — kits et spawns, pas des modeles).
2. **`root+0x08` n'est PAS un lien** : il se resout 467/467... vers le tag LUI-MEME
   (0x55897BE3 = type_id 1435073507). Le « hash non craque » de l'investigation est
   l'auto-reference. Porte fermee.
3. **Le lien est INLINE dans les octets du tag** (scan de chaque u32 contre l'index complet
   de l'installation, la methode qui avait etabli type_id -> food) : les food dessinables
   portent une liste de variantes referencant `bloc` + `rtgo` + `scgt` + `rtmp` a offsets
   reguliers. **374/467 type_id portent >= 1 ref `rtgo` directe — 3 558 des 4 697 objets
   places (75,7 %)**. Les 93 type_id restants (1 139 objets) passent par `bloc` (963),
   `scen` (173), `mach` (9) — un saut supplementaire, NON traite par le prototype.

`rtgo` est le MEME groupe que la chaine sbsp : `NewRuntimeGeoAsset` les decode tel quel.

### 13.2 L'echelle — le piege verifie sur pieces, et il est INVERSE

Le champ objet [6] du .mvar n'existe pas ; le champ [9] est une struct VIDE sur
4 709/4 709 objets. **Aucune echelle dans le .mvar de Vagabond** — pose a echelle
unitaire, mesuree et non supposee. Le repere : `Left = Up x Forward` — la MEME convention
que `mapvar/containment.go`, deja validee par l'oracle de containment des zones.

### 13.3 Le prototype (`rendu_forge_gamefiles_test.go`, TestRenduForgeVagabond)

    3 558 objets dessines · 1 113 sautes (sans rtgo direct) · 38 volumes de mort EXCLUS
    ORACLE DES ANCRES : 4/4 ont du sol dessine (100 %)
    mutation JOUEE : z de pose ignore -> 0/4 ancres (rouge), revert -> 4/4 (vert)

- **Cas Vagabond traite explicitement** : module generique `map` au catalogue (partage
  avec Highpower) — selection par map_id (`105f5d84-...`), jointure VERIFIEE par le compte
  d'objets (4 709 == 4 709).
- **Tranche relative** : `TrancheDeJeuMin/Max` sont absolues (sol Cliffhanger vers z=0),
  le sol de Vagabond vit vers z=52. La meme tranche est translatee au sol deduit des
  ancres (mediane - `AncrageDecalageSol`) — une regle, pas un reglage par carte.
- **Volumes de mort** : les 4 type_id de l'empreinte fonctionnelle sont exclus ET comptes
  (38 objets sur Vagabond). Ils n'ont de toute facon aucune ref rtgo.
- Ecarts par ancre : -2,7 a -4,2 m sur trois ancres, -19,6 m sur une zone Bastion — le
  z-buffer retient la surface la plus haute (toits, passerelles), meme comportement que
  Cliffhanger. Le juge du rendu reste l'utilisateur (`vagabond_forge_proto.png`, Bureau).

### 13.4 Ce que le prototype NE fait PAS (reporte au registre)

- les 1 139 objets sans rtgo direct (saut bloc/scen/mach -> hlmt -> geometrie) ;
- le choix de VARIANTE (la premiere ref rtgo du tag est prise — les suivantes sont
  probablement les etats casses/skins) ;
- la TOILE (bsp fo08_wetland) n'est pas rendue sous les objets — a decider au gate visuel ;
- la chiralite du repere n'est verifiable qu'a l'oeil (l'oracle des ancres n'y est pas
  sensible) — gate visuel utilisateur.


## 14. LA ZONE JOUABLE EST RESOLUE — la COQUILLE, testee a l'altitude de JEU (2026-08-10)

Apres sept criteres refutes, c'est la frontiere de mort declaree par la carte qui tranche —
et le blocage n'etait pas le critere, c'etait **a quelle altitude on le testait**.

### 14.1 Le defaut de mecanisme, et sa correction

La premiere mesure de la coquille la testait au z **DESSINE** de chaque pixel. Un rocher haut
au-dessus de l'arene sort de la coquille verticalement, sa cellule est effacee, et la position
jouee EN DESSOUS perd son sol : 10 points de positions perdues, verdict NO-GO.

La bonne question n'est pas « ce pixel est-il dans la coquille » mais **« ce pixel est-il
au-dessus d'un endroit ou l'on joue »**. Testee au NIVEAU DE JEU (mediane des ancres moins
`AncrageDecalageSol`) :

    carte        effet de la coquille          cout
    Cliffhanger  garde 99 % du cadre           ZERO — accord 66,7 %, positions 93,82 %,
                 (sans effet)                  identiques au grain seul
    Catalyst     retire 47,1 % du decor        19/19 ancres gardent leur sol

**Gratuite la ou elle ne sert pas, decisive la ou le grain est muet.** C'est la premiere regle
du chantier qui ne coute rien nulle part. `Rendu.RestreintALaCoquille`, active par defaut
(`RENDU_SANS_COQUILLE` la retire).

### 14.2 La couleur : le decor RECULE au lieu d'etre supprime

`TeinteAltitude` normalisait entre les centiles 2 et 98 de TOUTE la matiere : sur une carte
encaissee, les rochers hauts prenaient le blanc, donc l'oeil, pendant que l'arene reculait dans
le sombre. **La hierarchie visuelle etait inversee.**

`TeinteNiveauDeJeu` mesure l'altitude contre le NIVEAU JOUE et non contre elle-meme
(`PorteeNiveauDeJeu` = 10 m, deux etages). Styles `jeu` et `encre`. Sur Catalyst, l'arene
ressort et les structures alentour passent au noir.

**Ce que ca change au-dela de l'esthetique** : le decor n'a plus besoin d'etre supprime pour
cesser de nuire. La zone jouable et la lisibilite sont deux problemes distincts, et le second
se resout a l'affichage.

### 14.3 CORRECTION DE DEUX VERDICTS QUE J'AVAIS ECRITS TROP FORT

1. **« Coquille comme cadre universel : NO-GO »** (§10.8 et registre) — le NO-GO portait sur la
   coquille testee a l'altitude DESSINEE. Au niveau de jeu, c'est un GO. L'entree du registre
   est corrigee.
2. **« Environnement ferme : NO-GO »** — teste sur **UNE SEULE carte**, et la moins favorable :
   Cliffhanger est une arene A CIEL OUVERT sur une falaise. Catalyst, station spatiale, n'a
   jamais ete testee. Le verdict est requalifie en « refute sur Cliffhanger, non teste
   ailleurs ». C'est la meme erreur que celle reprochee a l'investigation collision — conclure
   d'un echantillon non representatif — commise dans l'autre sens.

**Lecon, et elle vaut pour tout le chantier** : un NO-GO se mesure sur les cartes ou l'hypothese
a une CHANCE, pas seulement sur celle qu'on a sous la main.

### 14.4 LE GATE A ATTRAPE QUATRE CARTES — la coquille n'est PAS universellement sure

Balayage des 25 cartes mesurables (`TestBalayageCoquille`, 2026-08-10) :

    21 cartes    aucune ancre perdue · decor retire de 0 a 88,4 %
                 (cliffhanger 0 % · aquarius 17 % · catalyst 47 % · recharge 66 %
                  highpower 73 % · bazaar 82,7 % · streets 88,4 %)

    !! behemoth_va_behemoth        64 plans · ancres 14 ->  0 · 100,0 % efface
    !! forbidden_map               11 plans · ancres 13 ->  0 ·  92,5 %
    !! fragmentation_map           18 plans · ancres 22 -> 12 ·  78,0 %
    !! fragmentation_heavies_map   18 plans · ancres 20 ->  7 ·  76,7 %

    live_fire (2 variantes) : sous-tests en ECHEC — c'est la carte sans tag sbsp deja
    instruite au §1 ter, pas une regression de la coquille.

**CAUSE PRESUMEE, a confirmer** : `Coquille()` intersecte TOUS les triangles-frontieres du tag
comme un SEUL convexe (`CoquilleSddt` = liste de demi-espaces, `Contient` exige tous les plans).
Une carte qui declare PLUSIEURS volumes de frontiere, ou une frontiere non convexe, y perd
tout — l'intersection de volumes disjoints est vide, ce qui colle exactement a `behemoth` et
ses 64 plans pour 100 % efface. Piste du correctif : grouper les triangles par volume (comme
les volumes d'eau, deja lus en liste) et prendre l'UNION, pas l'intersection globale.

**LE GARDE, en attendant** : `CoquilleGardeLesAncres` — la coquille ne s'applique QUE si elle
contient toutes les ancres d'objectifs au niveau de jeu. Ce n'est pas une precaution
arbitraire : c'est l'oracle faible applique a la PRODUCTION et plus seulement au gate. Une
ancre est jouable par definition ; une coquille qui l'exclut est fausse pour cette carte.
Verifie : les trois coquilles fautives sont REFUSEES, 0 ancre perdue, et Catalyst garde ses
47 %.

**Ce garde protege la livraison, il ne remplace pas le correctif** — sur les quatre cartes
refusees, le decor reste entier.

### 14.5 Trois defauts de mes propres instruments, corriges

1. `peupleRendu` fait `t.Fatal` : une carte illisible tuait le balayage ENTIER. Corrige par un
   sous-test `t.Run` par carte — meme piege que celui deja documente pour
   `TestBalayageDesCartes`, reproduit sans le voir.
2. Le tableau ne s'imprimait qu'a la fin : aucun resultat partiel pendant 25 minutes de run.
3. Mon filtre de lecture des resultats ne matchait pas le format de mes propres lignes — j'ai
   d'abord conclu a un echec muet la ou le tableau existait.

### 14.6 CORRECTION DU §14.4 — la cause n'etait pas « plusieurs volumes »

Le §14.4 annonce comme cause presumee que `Coquille()` fusionnerait PLUSIEURS volumes de
frontiere. **C'est faux, et la sonde le montre** : le bloc parent (root @0x58) ne porte qu'UN
enregistrement de 28 o par carte, avec UNE liste de triangles.

    ridgeline     12 triangles ->  6 plans distincts
    catalyst      20 triangles ->  8 plans
    va_behemoth  116 triangles -> 64 plans

La vraie cause est plus simple et plus profonde : **la frontiere est un MAILLAGE FERME, pas un
convexe.** Ridgeline est une boite (6 plans), donc l'intersection des demi-espaces y tombe
juste par accident. Behemoth est concave : l'intersection de ses 64 demi-espaces est vide, d'ou
les 100 % effaces. Aucune union n'aurait corrige ca.

**Le correctif : PARITE DE RAYON** (`Sddt.ContientFrontiere`, Moller-Trumbore). Exacte sur tout
maillage ferme, convexe ou non, et sans aucun parametre. Resultat :

    carte                   tris  ancres        decor retire      avant le correctif
    behemoth                 116  14 -> 14           12,1 %       100 % efface
    forbidden                 28  13 -> 13           36,8 %        92,5 % efface
    fragmentation             60  22 -> 22           27,2 %        12 ancres perdues
    fragmentation_heavies     60  20 -> 20           22,9 %         7 ancres perdues
    catalyst                  20  19 -> 19           47,0 %       inchange
    bazaar                    12  14 -> 14           82,7 %       inchange
    cliffhanger               12  14 -> 14            0,0 %       inchange

**0 coquille refusee, 0 ancre perdue.** Le garde `FrontiereGardeLesAncres` ne se declenche plus
nulle part — il reste en place comme filet, il n'a plus rien a rattraper.

**UN PIEGE ATTRAPE PAR LE TEMOIN, avant les donnees reelles.** Le premier jet lancait le rayon
vers +X. Sur une boite axee, il vise pile la diagonale partagee par les deux triangles d'une
face : l'intersection est comptee DEUX fois, la parite devient paire, et le centre de la boite
est declare dehors. Le temoin a rougi immediatement — et il aurait rougi pareil sur les vraies
cartes, pleines de faces axees. D'ou `directionRayonParite`, volontairement de travers.

Le temoin qui SEPARE : deux boites disjointes et un point entre les deux. La parite repond
« dehors » (juste) ; l'intersection des demi-espaces rend un volume VIDE et se trompe partout —
exactement ce qui effacait behemoth.

**Lecon** : ma cause presumee etait plausible, ecrite avec assurance, et fausse. Une sonde de
trois minutes l'a defaite. Ecrire « cause presumee » ne dispense pas de la mesurer avant de
construire le correctif dessus.
