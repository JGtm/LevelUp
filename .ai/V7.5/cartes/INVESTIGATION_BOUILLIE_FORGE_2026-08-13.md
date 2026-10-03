# Investigation BOUILLIE Forge — cause chiffree et prototype de regle (2026-08-13)

> Le gate visuel utilisateur a REFUSE EN BLOC les 33 fonds Forge de masse du lot map_id
> (commit `79cf8e803`) : « aucune carte exploitable, bouillie au milieu, forme patatoide
> gribouillee » ; Goliath tolerable, Domicile « trop de toits » ; pilotes Starboard/Dredge
> « plutot ok » mais « on voit encore trop les plafonds et toits ». Vagabond, Corpo et les
> natives sont validees. L'oracle des ancres (438/441) n'avait rien vu — il ne juge pas
> l'aspect. Ce document etablit la cause PAR LA MESURE, propose une regle universelle,
> prouve la non-regression au bit des fonds valides, et livre les temoins du gate.

## 1. L'instrument

`TestSondeBouillie` (`sonde_bouillie_gamefiles_test.go`, BROUILLON NON VERSIONNE dans
`internal/himap/`) cuit 12 cartes Forge — 5 validees/tolerees, 7 refusees — avec la voie de
reference ARMEE mais NON appliquee, et mesure par pixel : profil vertical de la surface
haute contre la reference interpolee des ancres, taux de couverture (formule de production)
sur le cadre entier / dans la portee des ancres (25 m) / au-dela, residu apres substitution
simulee, picks sous le sol, rugosite du champ substitue, qualite de la reference
(leave-one-out), surplomb reel au droit des ancres. 12/12 PASS en 33 min :

	go test ./internal/himap/ -run TestSondeBouillie -timeout 45m -v

## 2. Le tableau qui separe (sonde, 2026-08-13)

`horsP` = part de la matiere au-dela de PorteeAncre (25 m) de toute ancre. `taux` = part de
matiere qui cache un sol praticable, cadre ENTIER / dans la PORTEE (seuil de production
SeuilCarteCouverte = 33,3 % applique au cadre entier). `couv>=6` = part de matiere dans la
portee dont la surface haute est >= 6 m au-dessus de la reference. `surplombAncres` =
mediane, au droit des ancres, de la surface haute au-dessus de leur sol. `residu>=4` = part
de matiere restant >= 4 m au-dessus de la reference APRES la substitution de production
simulee (dedans / dehors). `sous-sol` = picks de substitution <= -4 m sous la reference.

| carte | verdict user | horsP | taux entier | taux portee | couv>=6 | surplombAncres | residu>=4 dedans | dehors | sous-sol |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Vagabond | validee (« a revoir ») | 40,0 % | 78,8 % | 77,6 % | 86,1 % | 19,1 m | 20,6 % | 99,9 % | 0,0 % |
| Corpo | validee | 14,7 % | 45,8 % | 52,9 % | 31,8 % | 4,6 m | 12,1 % | 64,6 % | 0,3 % |
| Starboard | pilote plutot ok | 6,6 % | 36,8 % | 38,8 % | 84,2 % | 6,2 m | 54,5 % | 72,0 % | 0,3 % |
| Dredge | pilote plutot ok | 18,3 % | 40,3 % | 45,4 % | 98,4 % | 18,0 m | 12,0 % | 92,0 % | 30,5 % |
| Goliath | tolerable | 2,4 % | 53,7 % | 55,0 % | 62,1 % | 9,9 m | 25,7 % | 90,0 % | 0,5 % |
| Domicile | REFUSEE (toits) | 50,9 % | **16,2 %** | 28,6 % | 94,2 % | 8,6 m | 63,6 % | 93,7 % | 3,3 % |
| The Pit | REFUSEE | 8,1 % | **24,8 %** | 26,9 % | 56,1 % | 2,7 m | 44,9 % | 20,9 % | 0,4 % |
| Fortress | REFUSEE | 33,9 % | **27,4 %** | 41,3 % | 34,0 % | 10,2 m | 1,0 % | 1,6 % | 18,6 % |
| Empyrean | REFUSEE | 35,8 % | **26,9 %** | 41,9 % | 89,4 % | 14,4 m | 42,4 % | 84,7 % | 13,4 % |
| Dynasty | REFUSEE | 66,0 % | 73,3 % | 95,8 % | 83,4 % | 7,2 m | 0,0 % | 58,9 % | 0,0 % |
| Absolution | REFUSEE | 0,7 % | 80,5 % | 81,0 % | 55,9 % | 9,3 m | 3,7 % | 1,3 % | 1,4 % |
| Smallhalla | REFUSEE | 69,3 % | 73,1 % | 95,1 % | 98,6 % | 8,6 m | 2,9 % | 93,5 % | 1,4 % |

Reference d'ancres : erreur leave-one-out MEDIANE 0,0 a 1,9 m sur les 12 cartes (max 8,2 m
en marge de Domicile). **La reference n'est pas la cause** — hypothese « ancres peu
nombreuses/coplanaires » ecartee comme cause primaire (Corpo, 4 ancres strictement
colineaires 0x27 m, est VALIDEE).

## 3. La cause — trois mecanismes, tous chiffres

La lecture visuelle qui les relie (temoins du gate) : la « forme patatoide » EST la
frontiere de l'union des disques de 25 m autour des ancres (couture circulaire de la
substitution, litterale sur Dynasty et Smallhalla) ; la « bouillie » est le champ substitue
a l'interieur ; les « toits » sont tout ce que la substitution n'a pas touche.

1. **Le seuil 1/3 est mesure sur le CADRE ENTIER et le cadre Forge n'est pas celui d'une
   native.** Sur une native, le cadre est plein de terrain a hauteur de jeu ; sur une carte
   Forge, il est plein de vide, de coquille et de decor flottant SANS sol pres de la
   reference — qui gonflent le denominateur sans jamais compter comme « couvert ». Quatre
   refusees sont restees `covered=false` A TORT : Domicile 16,2 % (28,6 % dans la portee),
   Fortress 27,4 % (41,3 %), Empyrean 26,9 % (41,9 %) — deux auraient depasse le seuil
   mesure dans la portee. Resultat : AUCUNE substitution, plein cadre de toits (surplomb
   aux ancres 8,6 a 14,4 m). The Pit reste sous le seuil meme dans la portee (26,9 %) :
   ses etages reels vivent a plus de TolSolReference (3 m) de la reference (7/14 ancres
   couvertes seulement) — le critere « sol praticable » est trop strict pour elle.
2. **La substitution s'arrete a PorteeAncre (25 m) quand le cadre va a 50 m (MargeCadre)
   et que l'arene reelle deborde.** Matiere hors portee des refusees couvertes : Dynasty
   66,0 %, Smallhalla 69,3 %, Domicile 50,9 % — avec un couvercle hors portee de 41 a 93 %.
   L'image publiee est donc : un disque substitue borde d'une couture circulaire, entoure
   de coquille rendue telle quelle.
3. **« La plus proche de la reference » n'enleve pas le couvercle, elle choisit un pixel.**
   Elle prend aussi les SOUS-SOLS (Dredge : 30,5 % des pixels en portee montrent une
   surface <= -4 m sous la reference — structure sous la plateforme du canevas deepsea),
   elle remplace des etages joues par le sol, et la ou aucun sol n'existe sous le toit elle
   garde une surface haute : residu >= 4 m au-dessus de la reference APRES substitution
   jusqu'a 63,6 % dans la portee (Domicile) et 20,6 a 99,9 % hors portee sur TOUTES les
   cartes a canopee.

Hypotheses ecartees par la mesure : reference degeneree (LOO §2), mauvaise variante `.mvar`
(jointure `objects_n` catalogue<->mvar verte sur les 12), tranche mal translatee (profils
verticaux coherents avec `zJeu` ; les couvercles sont DANS la tranche [-12;+28], c'est
precisement pourquoi ils se dessinent).

## 4. La regle prototype — L'ARENE PAR LA REFERENCE (ecretage per-pixel)

**Enonce.** Sur une carte FORGE non gelee : chaque pixel montre LA PLUS HAUTE surface de la
bande `[ref - PlancherArene ; ref + PlafondArene]` (ref = surface de reference interpolee
des ancres, evaluee PAR PIXEL, sur TOUT le cadre) ; un pixel sans surface dans la bande
devient VIDE. Deux constantes universelles, AUCUN reglage par carte :

- `PlafondArene = 6 m` = EcartPlafondMin (2 m, un Spartan tient debout) + TolSolReference
  (3 m, sols en pente) + 1 m de marge. Un etage JOUE vit a moins de 6 m de sa reference
  locale ; les couvercles mesures vivent au-dela (surplomb median aux ancres des refusees :
  7,2 a 18,0 m — tableau §2).
- `PlancherArene = 12 m` = -TrancheDeJeuMin, le plancher de tranche existant exprime contre
  la reference locale.

**Pourquoi elle repond aux trois mecanismes.** Plus de seuil de declenchement (mecanisme 1
elimine — la regle est toujours active sur Forge) ; plus de frontiere a 25 m (mecanisme 2 —
la bande s'applique au cadre entier, la couture disparait) ; « la plus haute DANS la
bande » au lieu de « la plus proche » (mecanisme 3 — les sous-sols sont invisibles par
construction, les etages joues sous 6 m restent rendus, et un pixel qui n'a QUE du
couvercle devient vide : la silhouette devient le plan de l'arene, ce qui est l'objet meme
de la demande).

**Pourquoi FORGE seulement.** Une carte Forge est un rack d'objets dans le vide : hors de
la bande il n'y a que la coquille, jamais une vallee a montrer. Les natives ont un terrain,
une frontiere `sddt` et un decor valides par l'utilisateur — leur chaine ne bouge pas d'un
bit (verifie §6). C'est une distinction de CHAINE (deja le cas pour frontiere et eau), pas
un reglage par carte.

**Le gel en DONNEE (pas une branche par carte).** `CarteForge.FondFige` (raison ecrite)
rejoue la regle historique pour les fonds Forge deja VALIDES par l'utilisateur : Vagabond
(« a revoir », registre 2026-08-13 : ni l'ameliorer ni le degrader avant son propre gate)
et Corpo. Pattern sanctionne au registre (exception declaree en donnee + raison + oracle).
Mesure qui l'impose : Vagabond est indiscernable des refusees sur TOUS les axes du §2
(canopee 86 % en portee, surplomb 19,1 m, 40 % hors portee) — aucune regle universelle ne
peut a la fois corriger la masse et laisser Vagabond identique au bit. L'impasse est
documentee ici, le gel en donnee est la sortie.

## 5. Ce que le prototype rend (cuissons brouillon, mesures)

Cuisson `--out-dir` scratch (jamais les fonds publies), 7 temoins :

| carte | substituees | videes (matiere supprimee) | ancres avec sol | surplomb ancres apres |
|---|---:|---:|---|---:|
| Domicile | 185 057 | 454 215 (58,9 %) | 8/8 (= publie) | -5,5 m |
| The Pit | 75 870 | 106 275 (31,0 %) | 14/15 (= publie) | -2,7 m |
| Goliath | 124 526 | 16 061 (6,9 %) | 7/8 (= publie) | -3,7 m |
| Starboard | 120 914 | 148 175 (44,5 %) | 8/8 (= publie) | -6,0 m |
| Dredge | 436 915 | 48 102 (9,6 %) | 8/8 (= publie) | -5,2 m |
| Dynasty | 525 541 | 152 895 (12,5 %) | 8/8 (= publie) | -4,4 m |
| Smallhalla | 1 172 710 | 229 032 (15,1 %) | 27/27 (= publie) | -5,5 m |

L'oracle des ancres est INCHANGE partout. Le couvercle disparait (les grandes plaques de
toit de Domicile, le champ de conteneurs de The Pit, la canopee de Dredge) et l'interieur
de l'arene devient visible ; la couture circulaire n'existe plus.

**Compromis et defauts residuels connus, a juger au gate — je ne juge pas l'aspect :**

1. Les plafonds et etages a MOINS de 6 m au-dessus de la reference restent rendus (surplomb
   residuel aux ancres 2,7 a 6,0 m sur les 7 temoins). Si le gate dit « encore trop de
   toits », le cran mesurable est `PlafondArene` 6 -> 4 ; s'il dit « trop vide », 6 -> 8.
2. Smallhalla : le sculpt organique (canyon, canopee basse) traverse la bande partout — le
   gribouillis central subsiste. Cause DISTINCTE des trois mecanismes (la geometrie jouee
   elle-meme est organique) ; peut rester non satisfaisante.
3. Les piliers decoratifs (hexagones de Domicile) dont seule la tranche coupe la bande se
   rendent en contour.

## 6. Non-regression PROUVEE (SHA256, re-cuisson complete en brouillon)

Re-cuisson des **56 fonds** (19 natives + 37 Forge) avec le prototype, `--out-dir` scratch,
puis comparaison SHA256 aux fonds publies :

- **21/21 fonds VALIDES IDENTIQUES AU BIT** : les 19 natives (btb_drydock, btb_engine,
  btb_exiled, btb_fragmentation, btb_highpower, catalyst, chasm, ctf_aquarius, ctf_bazaar,
  ctf_breaker, ctf_forbidden, ctf_illusion, forest, ridgeline, sgh_blueprint,
  sgh_crystalcaves, sgh_streets, va_behemoth, va_launchsite) + Vagabond (105f5d84) +
  Corpo (8be179f7).
- **35/35 fonds modifies = exactement le perimetre vise** : les 33 de masse + les 2 pilotes
  (qui devaient s'ameliorer sur les toits).
- **0 echec de cuisson** sur les 56 ; determinisme verifie : les 7 temoins cuits deux fois
  (deux invocations independantes) rendent des PNG identiques au bit.

## 7. Pistes refutees ici et portes deja fermees (ne pas rejouer)

- Reference d'ancres degeneree comme cause du refus : REFUTEE (LOO §2 ; Corpo colineaire
  validee). Elle reste un facteur de PRECISION en marge de cadre (Domicile max 8,2 m).
- Rugosite du champ substitue comme discriminant du verdict : REFUTEE — Dynasty (0,03
  m/paire) et Smallhalla (0,05) sont plus LISSES que Corpo validee (0,12) ; la « bouillie »
  percue est la couture + la coquille + le sculpt, pas le bruit du champ.
- Part d'ancres couvertes, seuil par pixel sans reference : deja refutees au lot toits
  (INVESTIGATION_TOITS_2026-08-13.md §2) — non rejouees.
- Accessibilite pietonne et collision : portes FERMEES (registre, handoff §3) — non
  rejouees.

## 8. Reste a faire (conditions de reprise)

1. **GATE VISUEL utilisateur** : `Desktop/gate_cartes_v75/bouillie_proto/` — 7 paires
   `{carte}_actuel.png` / `{carte}_proto.png` (domicile, goliath, dynasty, smallhalla,
   the_pit, starboard, dredge).
2. Si gate OK : lot de PRODUCTION = appliquer le diff (annexe A, non committe — les
   fichiers modifies sont revenus a l'etat du depot), re-cuire les 35 fonds non geles vers
   `map_backgrounds/`, temoins unitaires de la voie d'arene (casser/voir rouge/revert),
   statuer la semantique sidecar (`covered`/`cellsSubstituted` + publier `videes`), MAJ
   HANDOFF_CARTES + registre.
3. Si « encore trop de toits » : `PlafondArene` 6 -> 4 (mesure d'appui : surplomb residuel
   aux ancres 2,7-6,0 m) ; si « trop vide » : 6 -> 8. Un seul cran, re-gate.
4. Smallhalla-like (sculpt organique) : cause distincte, a instruire separement si le gate
   la maintient refusee.
5. Vagabond et Corpo : GELS en donnee — a re-juger a leur propre gate (deja au registre).
6. La sonde `sonde_bouillie_gamefiles_test.go` reste en brouillon non versionne dans
   `internal/himap/` (elle ne depend pas du prototype) ; la promouvoir ou la supprimer au
   lot de production.

## Annexe A — diff du prototype (5 fichiers, +114/-4, NON COMMITTE)

Copie de travail : `scratchpad/proto_bouillie.patch` de la session. Reproduction integrale :

```diff
diff --git a/apps/go-api/cmd/mapfond-build/cuisson.go b/apps/go-api/cmd/mapfond-build/cuisson.go
--- a/apps/go-api/cmd/mapfond-build/cuisson.go
+++ b/apps/go-api/cmd/mapfond-build/cuisson.go
@@ -181,6 +181,7 @@ func (e *environnement) cuitForge(ctx context.Context) []bilanAsset {
 			Ancres:              c.ancres,
 			CheminModuleCanevas: himap.CheminCanevasForge(carte),
 			Cle:                 carte.MapID,
+			FondFige:            carte.FondFige != "",
 		})
 		if err != nil {
 			slog.ErrorContext(ctx, "cuisson Forge", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
diff --git a/apps/go-api/internal/himap/cartes_forge.go b/apps/go-api/internal/himap/cartes_forge.go
--- a/apps/go-api/internal/himap/cartes_forge.go
+++ b/apps/go-api/internal/himap/cartes_forge.go
@@ -24,6 +24,12 @@ type CarteForge struct {
 	FichierMvar string
 	// ModuleCanevas est le dossier du module sur lequel la carte est batie.
 	ModuleCanevas string
+	// FondFige : raison ecrite du GEL du fond publie (vide = pas de gel). Un fond gele a ete
+	// VALIDE par l'utilisateur : la cuisson rejoue la regle qui l'a produit (voie de
+	// reference historique) au lieu de la voie d'arene, et l'asset reste identique au bit.
+	// Exception declaree en DONNEE, jamais une branche par carte dans le code (pattern
+	// sanctionne au registre, gate toits 2026-08-13).
+	FondFige string
 }
 
 // Les canevas Forge installes : les 8 dossiers de module sur lesquels les cartes
@@ -52,12 +58,16 @@ var CartesForge = []CarteForge{
 		Nom:           "Vagabond",
 		FichierMvar:   "vagabond_map.mvar",
 		ModuleCanevas: CanevasWetland,
+		FondFige: "fond valide au gate utilisateur du 2026-08-10, juge « a revoir » (registre " +
+			"2026-08-13) : ni l'ameliorer ni le degrader avant son propre gate",
 	},
 	{
 		MapID:         "8be179f7-8940-4868-b881-44cad1ca8711",
 		Nom:           "Corpo",
 		FichierMvar:   "corpo_map.mvar",
 		ModuleCanevas: CanevasBlank,
+		FondFige: "fond valide au gate utilisateur (migration map_id 2026-08-13) : " +
+			"intouchable au bit tant qu'un nouveau gate ne le rejuge pas",
 	},
 	// Pilotes du lot fonds par map_id (2026-08-13) : seules cartes jouees SEULES sur leur
 	// canevas. Preuve level_id : Starboard -747133697 (0xD377A4FF) -> fo03_space, Dredge
diff --git a/apps/go-api/internal/himap/cuisson_forge.go b/apps/go-api/internal/himap/cuisson_forge.go
--- a/apps/go-api/internal/himap/cuisson_forge.go
+++ b/apps/go-api/internal/himap/cuisson_forge.go
@@ -70,6 +70,10 @@ type OptionsCuissonForge struct {
 	// Cle est le nom sous lequel l'asset sera publie (cf. BilanCuisson.Module) : le map_id
 	// de la carte (cartes_forge.go).
 	Cle string
+	// FondFige : le fond publie de cette carte est VALIDE par l'utilisateur et gele en
+	// DONNEE (CarteForge.FondFige) — la cuisson rejoue la regle qui l'a produit (voie de
+	// reference historique), jamais la voie d'arene. PROTOTYPE bouillie 2026-08-13.
+	FondFige bool
 }
 
 // CuitCarteForge rend le fond de carte d'une carte Forge en posant les modeles de ses objets.
@@ -94,17 +98,32 @@ func CuitCarteForge(ctx context.Context, opts OptionsCuissonForge) (*Rendu, Bila
 	b.NiveauDeJeu = zJeu
 	r.Tranche(TrancheDeJeu(zJeu))
 	r.NiveauDeJeu(zJeu)
-	// MEME regle que la chaine native : la voie de reference contre les toits
-	// (rendu_reference.go). Une carte Forge a ciel ouvert reste sous le seuil et n'est pas
-	// touchee ; la regle est universelle, pas une affaire de chaine.
+	// La voie de reference est armee dans tous les cas (elle porte la grille `ref`).
+	// PROTOTYPE bouillie 2026-08-13 : une carte Forge NON gelee rend l'ARENE par la bande
+	// [ref-PlancherArene ; ref+PlafondArene] (ArmeBandeArene / AppliqueBandeArene) — la
+	// substitution historique (seuil 1/3 mesure sur le cadre entier, portee 25 m) rendait la
+	// coquille : dilution du taux par la matiere hors portee, couture circulaire a 25 m,
+	// picks sous-sol (INVESTIGATION_BOUILLIE_FORGE_2026-08-13.md). Une carte GELEE
+	// (FondFige) rejoue la regle historique : son fond publie est valide par l'utilisateur.
 	s := NewSurfaceReference(opts.Ancres)
 	r.ArmeReference(s)
+	if !opts.FondFige {
+		r.ArmeBandeArene()
+	}
 
 	poseObjetsForge(ctx, r, &b, opts.Objets, idx, forge)
 	if b.ObjetsDessines == 0 {
 		return nil, b, fmt.Errorf("aucun des %d objets Forge n'a de modele rtgo", len(opts.Objets))
 	}
-	b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s)
+	if opts.FondFige {
+		b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s)
+	} else {
+		b.TauxCouverture = r.TauxCouvertureMesure()
+		subst, videes := r.AppliqueBandeArene()
+		b.CellulesSubstituees, b.CarteCouverte = subst, true
+		slog.InfoContext(ctx, "voie d'arene appliquee (prototype bouillie)", "cle", b.Module,
+			"substituees", subst, "videes", videes)
+	}
 	if b.VolumesDeMort == 0 {
 		b.degrade(ctx, "aucun volume de mort reconnu — l'empreinte des types a peut-etre bouge")
 	}
diff --git a/apps/go-api/internal/himap/rendu.go b/apps/go-api/internal/himap/rendu.go
--- a/apps/go-api/internal/himap/rendu.go
+++ b/apps/go-api/internal/himap/rendu.go
@@ -92,6 +92,11 @@ type Rendu struct {
 	dRef []float64
 	zRef []float64
 	nRef [][3]float64
+	// zBande / nBande : la voie d'ARENE de la chaine Forge — la plus haute surface dans la
+	// bande [ref-PlancherArene ; ref+PlafondArene] (PROTOTYPE bouillie 2026-08-13,
+	// rendu_reference.go). Nil tant que `ArmeBandeArene` n'a pas ete appelee.
+	zBande []float64
+	nBande [][3]float64
 	// eau : cellules couvertes par un volume d'eau (PoseEau, cf. sddt.go). Un habillage —
 	// jamais consulte par le z-buffer ni par les metriques du banc.
 	eau []bool
@@ -192,6 +197,13 @@ func (r *Rendu) triangleBorne(a, b, c [3]float64, lo, hi [3]float64) {
 				if d := math.Abs(z - r.ref[k]); d < r.dRef[k] {
 					r.dRef[k], r.zRef[k], r.nRef[k] = d, z, nrm
 				}
+				// Voie d'ARENE (prototype bouillie) : la plus haute surface dans la
+				// bande autour de la reference. Strictement `>`, comme la voie haute.
+				if r.zBande != nil {
+					if d := z - r.ref[k]; d <= PlafondArene && d >= -PlancherArene && z > r.zBande[k] {
+						r.zBande[k], r.nBande[k] = z, nrm
+					}
+				}
 			}
 		}
 	}
diff --git a/apps/go-api/internal/himap/rendu_reference.go b/apps/go-api/internal/himap/rendu_reference.go
--- a/apps/go-api/internal/himap/rendu_reference.go
+++ b/apps/go-api/internal/himap/rendu_reference.go
@@ -51,6 +51,74 @@ const TolSolReference = 3.0
 // n'est pas touchee du tout — son PNG reste identique au bit.
 const SeuilCarteCouverte = 1.0 / 3
 
+// PlafondArene / PlancherArene : la bande d'altitude, RELATIVE A LA REFERENCE PER-PIXEL, dans
+// laquelle vit l'arene d'une carte FORGE (PROTOTYPE bouillie, 2026-08-13 —
+// INVESTIGATION_BOUILLIE_FORGE_2026-08-13.md).
+//
+// PlafondArene = EcartPlafondMin (2 m, un Spartan tient debout) + TolSolReference (3 m,
+// dispersion des sols en pente) + 1 m de marge : un etage JOUE vit a moins de 6 m de sa
+// reference locale. Les COUVERCLES mesures par la sonde bouillie vivent au-dela : surplomb
+// median aux ancres 7,2 a 18,0 m sur les cartes refusees (Dynasty 7,2 · Domicile 8,6 ·
+// Smallhalla 8,6 · Goliath 9,9 · Fortress 10,2 · Empyrean 14,4 · Dredge 18,0).
+//
+// PlancherArene = -TrancheDeJeuMin : le plancher de tranche existant, exprime contre la
+// reference locale au lieu du seul niveau median.
+const (
+	PlafondArene  = 6.0
+	PlancherArene = 12.0
+)
+
+// ArmeBandeArene alloue la voie d'ARENE : la plus haute surface dans la bande
+// [ref-PlancherArene ; ref+PlafondArene]. Ne fait rien si la reference n'est pas armee.
+// Chaine FORGE uniquement (une carte Forge est un rack d'objets dans le vide : hors bande il
+// n'y a que la coquille, jamais une vallee a montrer).
+func (r *Rendu) ArmeBandeArene() {
+	if r.zRef == nil {
+		return
+	}
+	r.zBande = make([]float64, r.NX*r.NY)
+	r.nBande = make([][3]float64, r.NX*r.NY)
+	for k := range r.zBande {
+		r.zBande[k] = math.Inf(-1)
+	}
+}
+
+// AppliqueBandeArene rend l'ARENE au lieu de la coquille : chaque pixel montre la plus haute
+// surface de la bande autour de la reference — sur TOUT le cadre, pas seulement dans la
+// portee des ancres — et un pixel sans surface dans la bande devient VIDE (c'etait un
+// couvercle, un surplomb ou du decor flottant : il n'appartient pas a l'arene).
+//
+// C'est la SEULE passe du chantier cartes qui supprime de la matiere : la silhouette devient
+// le plan de l'arene, c'est son objet meme (mission bouillie). Rend le nombre de pixels
+// substitues et le nombre de pixels vides.
+func (r *Rendu) AppliqueBandeArene() (substituees, videes int) {
+	if r.zBande == nil {
+		return 0, 0
+	}
+	defer func() {
+		r.ref, r.dRef, r.zRef, r.nRef, r.zBande, r.nBande = nil, nil, nil, nil, nil, nil
+	}()
+	for k := range r.z {
+		if math.IsInf(r.z[k], -1) {
+			continue
+		}
+		if math.IsInf(r.zBande[k], -1) {
+			r.z[k] = math.Inf(-1)
+			videes++
+			continue
+		}
+		if r.z[k] != r.zBande[k] {
+			r.z[k], r.n[k] = r.zBande[k], r.nBande[k]
+			substituees++
+		}
+	}
+	return substituees, videes
+}
+
+// TauxCouvertureMesure expose la mesure de couverture AVANT application d'une passe — la
+// sonde et le bilan la publient meme quand la voie d'arene remplace la substitution.
+func (r *Rendu) TauxCouvertureMesure() float64 { return r.tauxCouverture() }
+
 // ArmeReference precalcule la grille d'altitude de reference du rendu et alloue la voie de
 // reference du z-buffer. A appeler AVANT de projeter les maillages : les triangles deja poses
 // ne sont pas rejoues.
```

## Annexe B — protocole de re-verification

1. Sonde : `go test ./internal/himap/ -run TestSondeBouillie -timeout 45m -v` (12 cartes,
   ~33 min ; log complet de la session au scratchpad `sonde_bouillie.log`).
2. Prototype : appliquer l'annexe A, `go vet ./internal/himap/ ./cmd/mapfond-build/`, tests
   `TestReference*` + `TestCartesForgeDeclarations` + `TestFondForgeJamaisSousCleModule`.
3. Brouillon : `go run ./cmd/mapfond-build --out-dir <scratch>` (JAMAIS les fonds publies),
   natives puis Forge ; SHA256 contre `data/titles/halo_infinite/reference/map_backgrounds`.
   Attendu : 21 identiques (19 natives + Vagabond + Corpo), 35 differents, 0 echec.
