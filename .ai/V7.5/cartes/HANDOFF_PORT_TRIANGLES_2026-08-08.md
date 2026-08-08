# Handoff — portage de la chaine des TRIANGLES en Go (2026-08-08)

> Ecrit en fin de session, sur avancee critique. Branche `feat/v75`, worktree
> `LevelUp-wt-replay2d`. Plan d'execution : `PLAN_PORT_TRIANGLES_GO.md` (memes dossiers).
> Plan de reference pour le VISUEL : `../../PLAN_BELLE_CARTE_TRIANGLES.md` — c'est LUI qui
> fait foi sur le rendu attendu et sur le gate humain.
>
> **Etat en une ligne** : la chaine decode et rend une carte reconnaissable, la primitive de
> rendu est trouvee, trois observations utilisateur restent ouvertes.

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

## 2. T4 — la dequantification n'est PAS tranchee

Trois metriques essayees sur les MEMES octets pour departager `u16` brut et `i16 + 32768` :

	ecart aux bornes declarees   : 16,9 mm contre 2,1 mm  -> donne raison a la FAUSSE
	longueur mediane d'arete     : 0,0189 contre 0,0196   -> ne separe pas
	part d'aretes > 1/4 de boite : 0,0821 contre 0,0897   -> ne separe pas

La premiere est BIAISEE (une rotation du quantum disperse les valeurs vers les extremes,
donc epouse mieux les bornes par construction) — **ne pas la reintroduire**. L'echec des
deux autres est instructif : la rotation ne DECHIRE pas les maillages, elle les DECALE
chacun d'une demi-boite ; leur forme reste intacte, seul leur registre mutuel casse.

**L'ORACLE EXISTE, il n'a pas encore ete joue** : `ETAT_DU_POC.md` documente que la carte
validee porte **95,6 % des 29 217 positions de joueur tombant sur le sol** (calage
0,0920 m/px, X0 -43,5, Y1 61,0). Rejouer ces positions contre le rendu produit tranche T4 ET
juge le rendu. **C'est le premier geste a faire a la reprise.**

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

## 4. LES TROIS OBSERVATIONS UTILISATEUR — ouvertes, ce sont les prochains temoins

1. **Un pont manque en bas, a gauche de celui qui est visible.** Le bornage [-12 ; +28]
   venait du prototype et **n'est pas universel** — verifie : elargi a [-45 ; +45], la
   couverture passe de 17,5 % a 19,2 % et du contenu de basse altitude apparait en haut.
   Avertissement pose dans `volume.go`. **Le pont manquant n'a pas encore ete retrouve.**
2. **Un ilot en haut a gauche, impraticable.** Candidat : les elements bleu fonce apparus
   avec la tranche elargie. Non confirme.
3. **La partie GAUCHE devrait porter des structures praticables** — elle reste blanche, et
   l'elargissement en Z ne la ramene PAS. **Hypothese a tester en premier** : ces structures
   viennent des MODULES GLOBAUX, ecartes par le filtre « module de la carte seul ».

## 5. LE FILTRE DE MODULE — la tension a resoudre

Decouverte D1 : 74 % des instances resolvent dans les modules globaux (`common`,
`multiplayer`, `multiplayer_r3`). C'est **exact**, et c'est ce qui enterrait la carte : un
seul tag de `common` pese 976 instances et 4,3 M de sommets dans la zone centrale (rochers,
debris). Le prototype ne les voyait pas — il ne lisait qu'un module.

Filtrer sur le module de la carte rend la carte lisible (9 832 -> 2 730 instances, 61 % ->
13 % de pixels). **Mais c'est probablement trop brutal** : l'observation 3 suggere que les
globaux portent AUSSI de vraies structures. Piste : ne pas trier par module mais par
PRATICABILITE — garder du global ce qui forme un sol degage, ecarter le reste. Le volume
sait deja le faire.

## 6. Ou sont les choses

	internal/himodule/resources.go        blob de ressources (manquait entierement)
	internal/himap/moduleindex.go         index GlobalID -> module (carte + globaux)
	internal/himap/rtgo.go                tag rtgo, PerMeshData
	internal/himap/geometry.go            descripteurs, LOD, dequantification, triangles
	internal/himap/volume.go              volume, praticabilite, coupe   <- LA PRIMITIVE
	internal/himap/heightfield.go         champ d'altitude — APPROCHE ABANDONNEE, cf. §3
	internal/himap/carte_gamefiles_test.go  produit l'artefact de revue (ecrit un PNG)
	internal/himap/deploy_root.go         ou est installe le jeu (LEVELUP_HALO_DEPLOY)

Commande : `PROBE_MULTI=1 PROBE_PNG=<sortie.png> go test ./internal/himap/ -run TestCarteCliffhanger -count=1 -v`

Images de la session (hors depot, Bureau) : `cliffhanger_CARTE_multiniveaux.png`,
`cliffhanger_CARTE_z45.png`, `cliffhanger_POINTS_carte_seule.png`, et la reference
`.ai/V7.5/dumps/carte_validee_v1.png`.

## 7. Ordre de reprise

1. **Jouer l'oracle** des 29 217 positions contre le rendu — tranche T4 et juge le rendu.
2. **Tester l'hypothese du §4.3** : reintegrer les globaux en filtrant par praticabilite,
   voir si la partie gauche se remplit.
3. Retrouver le pont manquant (§4.1) une fois 1 et 2 faits.
4. **Dette** : `volume.go` et `heightfield.go` n'ont pas de temoin sur donnees reelles ;
   `carte_gamefiles_test.go` n'affirme rien. Le gate visuel de `PLAN_BELLE_CARTE_TRIANGLES`
   exige un artefact cote a cote avec la reference — pas encore produit.

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
