package replay

// vehicules_v1a_test.go — INSTRUMENTS DE MESURE du lot V1a (vehicules, 2026-08-31).
//
// TROIS MESURES, TROIS GATES, ET LEURS SEUILS SONT ECRITS ICI AVANT TOUTE EXECUTION.
//
//	V1.2 (fonction)  la NOUVELLE entree `ScanFilmBipedPositionsForBand` rend-elle, sur la bande
//	                 `ti=40`, la meme continuite de trajectoire que la boucle jetable du
//	                 cadrage ? Seuil : >= 99 % de pas sous 35 m/s.
//	V1.3             le cap `i2` du vehicule est-il le cap qu'il SUIT ? Seuil : moyenne
//	                 circulaire de l'ecart sous 15 deg, mediane des ecarts absolus sous 30 deg,
//	                 temoin par permutation a ~90 deg.
//	V1a.4            l'oracle geometrique du 18/08 (1,5 m pendant 3 s), REJOUE sur des positions
//	                 de vehicule justes. Aucun seuil de reussite : c'est une MESURE, et son
//	                 verdict s'ecrit avec ses chiffres.
//
// CE QUI EST REUTILISE, ET POURQUOI. Le corpus (`v0Corpus`), les bornes de carte (`v0Bornes`),
// le critere de continuite (`v0ContinuitePositions`, 35 m/s / pas <= 2 s) et l'oracle
// geometrique lui-meme (`attPeriodesABord`, `attBandeFantome`) viennent des instruments
// existants. Les reecrire aurait rendu les chiffres INCOMPARABLES a ceux du cadrage et du lot
// du 18/08, ce qui est precisement ce qu'on veut confronter. Ce fichier depend donc de
// `vehicules_v0_cadrage_test.go` et `attachement_phase0_*_test.go` : ils vivent ensemble.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  go test ./internal/analysis/replay/ -run TestV1a -v -timeout 120m

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v1aSeuilContinuite est le gate de fonction de V1.2, ecrit avant mesure : le cadrage a releve
// 99,4 / 100,0 / 99,8 / 99,9 % sur les quatre films de reference par la boucle jetable.
const v1aSeuilContinuite = 0.99

// v1aVitesseMinMPS borne par le BAS les echantillons entrant dans l'oracle du cap : sous cette
// vitesse le cap de deplacement n'est plus defini (un vehicule a l'arret pointe ou il veut).
// Le seuil est celui du cadrage § 5.1, et il n'a pas bouge : 5 m/s.
const v1aVitesseMinMPS = 5.0

// v1aCapMoyenneMaxDeg / v1aCapMedianeMaxDeg sont les DEUX seuils de l'oracle V1.3.
const (
	v1aCapMoyenneMaxDeg = 15.0
	v1aCapMedianeMaxDeg = 30.0
)

// v1aOptions rend les reglages de balayage d'une bande d'objets du monde par la grammaire
// bipede : bornes de la carte, `RequireTag1` desarme (le tag est la generation du handle) et,
// au choix de l'appelant, les deux filtres de post-traitement.
//
// LE FLUX BRUT EST LE DEFAUT DES MESURES DE CE LOT. Avec `MaxSpeedMPS` et `IsolationGapMS` a
// zero, aucun post-filtre ne peut etre soupconne d'avoir SELECTIONNE les echantillons qui
// donnent raison a l'oracle — un filtre en m/s, en particulier, ecarterait par construction une
// partie des pas que la mesure de continuite compte.
func v1aOptions(wr *filmdec.Vec3Range, filtres bool) filmdec.ScanFilmOptions {
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange, opt.RequireTag1 = wr, false
	if !filtres {
		opt.MaxSpeedMPS, opt.IsolationGapMS = 0, 0
	}
	return opt
}

// v1aBandeVehicule releve la bande de slots de l'archetype vehicule aux images-cles.
func v1aBandeVehicule(dir string) map[uint32]bool {
	return filmdec.ScanFilmWorldObjectKeyframes(dir, int(attVehiculeTI)).Band
}

// ---------------------------------------------------------------------------------------
// V1.2 — GATE DE FONCTION : la nouvelle entree, sur la bande vehicule.
// ---------------------------------------------------------------------------------------

// TestV1aContinuiteNouvelleEntree confronte la NOUVELLE entree au critere du cadrage.
//
// DEUX VARIANTES parce que le cadrage mesurait un flux BRUT (sa boucle jetable ne passait par
// aucun post-traitement) tandis que la nouvelle entree, elle, applique `DropIsolated` puis
// `DropTeleports` quand ils sont armes. Publier les deux est la seule facon de dire si un ecart
// avec le cadrage vient de la grammaire ou des filtres.
func TestV1aContinuiteNouvelleEntree(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v1aContinuiteUnFilm(t, root, f)
	}
}

// v1aContinuiteUnFilm mesure UN film.
func v1aContinuiteUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V1.2 %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	for _, v := range []struct {
		nom     string
		filtres bool
	}{{"flux brut", false}, {"filtres par defaut", true}} {
		pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, bande, v1aOptions(&wr, v.filtres))
		if err != nil {
			t.Logf("V1.2 %s [%s] : %v", f.ID, v.nom, err)
			continue
		}
		pas, part := v0ContinuitePositions(pos)
		verdict := "SOUS LE SEUIL"
		if part >= v1aSeuilContinuite {
			verdict = "PASSE"
		}
		t.Logf("V1.2 %s (%s) [%-18s] bande %3d slots · %6d echantillons · %6d pas mesures · "+
			"%.2f %% sous %.0f m/s (seuil %.0f %%) %s",
			f.ID, f.Carte, v.nom, len(bande), len(pos), pas, 100*part, v0PlausibleMPS,
			100*v1aSeuilContinuite, verdict)
	}
}

// ---------------------------------------------------------------------------------------
// V1.3 — L'ORACLE DU CAP i2.
// ---------------------------------------------------------------------------------------

// v1aEcarts porte les ecarts angulaires d'une confrontation, en degres signes.
type v1aEcarts []float64

// v1aStats rend la moyenne circulaire (deg), la mediane des ecarts absolus (deg) et la
// concentration R de la distribution. R vaut 1 pour des ecarts tous egaux, 0 pour du bruit
// uniforme : c'est lui qui distingue « bien centre » de « centre par compensation ».
func (e v1aEcarts) v1aStats() (moyenne, mediane, r float64) {
	if len(e) == 0 {
		return 0, 0, 0
	}
	var sc, ss float64
	abs := make([]float64, 0, len(e))
	for _, d := range e {
		rad := d * math.Pi / 180
		sc, ss = sc+math.Cos(rad), ss+math.Sin(rad)
		abs = append(abs, math.Abs(d))
	}
	sc, ss = sc/float64(len(e)), ss/float64(len(e))
	sort.Float64s(abs)
	return math.Atan2(ss, sc) * 180 / math.Pi, abs[len(abs)/2], math.Hypot(sc, ss)
}

// v1aWrap180 ramene un ecart d'angle dans ]-180, 180].
func v1aWrap180(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d <= -180 {
		d += 360
	}
	return d
}

// v1aCapDeg rend le cap en degres d'un vecteur du plan XY.
func v1aCapDeg(x, y float32) float64 { return math.Atan2(float64(y), float64(x)) * 180 / math.Pi }

// v1aVitesseMPS rend la norme de la velocite i1 d'un echantillon, en m/s. La norme passe par
// `dist3` — l'unique ecriture de la formule euclidienne du paquet (garde-rail
// `TestUneSeuleFormuleDeDistance3D`) : une norme est la distance a l'origine.
func v1aVitesseMPS(p filmdec.BipedPosition) (float64, bool) {
	v, ok := p.VelocityVector()
	if !ok {
		return 0, false
	}
	return dist3(v, [3]float32{}), true
}

// TestV1aOracleCapI2 — LE CAP DU VEHICULE EST-IL LE CAP QU'IL SUIT ?
//
// CE QUE LA MESURE EPROUVE, ET CE QU'ELLE N'EPROUVE PAS. Le cadrage a montre que le CURSEUR
// atteint i2 dans 92 a 95 % des records `ti=40` ; il n'a pas montre que la VALEUR lue y est un
// cap. `consumeByName` route l'orthographe `object-forward-and-up-dynamic-precision` du
// vehicule vers le deserialiseur du bipede — une reutilisation heritee de `ti=38`, jamais
// mesuree sur `ti=40`. Ce test la mesure.
//
// TROIS CONFRONTATIONS, DONT UN TEMOIN OBLIGATOIRE :
//
//	i2 contre DEPLACEMENT   le cap lu contre celui des positions consecutives — l'oracle du plan.
//	i2 contre VELOCITE i1   les deux directions du MEME record, sans passer par les positions.
//	                        Elle situe une eventuelle faute : dans i2, ou dans la position.
//	TEMOIN (permutation)    les memes caps, melanges : si l'accord survit au melange, il ne
//	                        venait pas des donnees mais de la forme des deux distributions.
func TestV1aOracleCapI2(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v1aCapUnFilm(t, root, f)
	}
}

// v1aCapUnFilm mesure l'oracle du cap sur UN film.
func v1aCapUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V1.3 %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	opt := v1aOptions(&wr, false)
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, bande, opt)
	if err != nil {
		t.Logf("V1.3 %s : %v", f.ID, err)
		return
	}
	caps, depl, vel, vit := v1aConfronteCaps(pos)
	v1aPublieCap(t, f, "i2 contre DEPLACEMENT", len(pos), caps, depl)
	v1aPublieCap(t, f, "i2 contre VELOCITE i1", len(pos), caps, vel)
	// LA CONFRONTATION QUI LOCALISE LA FAUTE. i1 et i2 sont lus dans le MEME record, i1 en
	// premier : si la direction d'i1 s'accorde au deplacement et pas i2, la faute est dans i2
	// seul ; si i1 echoue aussi, c'est le curseur qui derive apres i0 et aucun des deux n'est
	// lisible. Sans cette ligne, un echec de l'oracle ne dirait pas OU chercher.
	v1aPublieCap(t, f, "i1 VELOCITE contre DEPLACEMENT", len(pos), vel, depl)
	if len(vit) > 0 {
		sort.Float64s(vit)
		t.Logf("V1.3 %s (%s) — vitesses i1 des paires retenues : mediane %.1f m/s, "+
			"min %.1f, max %.1f (seuil d'entree %.0f m/s)",
			f.ID, f.Carte, vit[len(vit)/2], vit[0], vit[len(vit)-1], v1aVitesseMinMPS)
	}
}

// v1aConfronteCaps rend, pour les echantillons retenus, le cap i2 et les deux references.
//
// REGLE DE SELECTION, ECRITE AVANT LA MESURE : l'echantillon porte i2 ET i1, sa velocite i1
// depasse `v1aVitesseMinMPS`, et il existe un echantillon SUIVANT du meme slot a moins de
// `v0PasMaxUS` dont il est separe par un deplacement horizontal non nul.
func v1aConfronteCaps(pos []filmdec.BipedPosition) (caps, depl, vel v1aEcarts, vit []float64) {
	parSlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			parSlot[p.Slot] = append(parSlot[p.Slot], p)
		}
	}
	slots := make([]uint32, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		ech := parSlot[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 0; i+1 < len(ech); i++ {
			m, ok := v1aMesureCap(ech[i], ech[i+1])
			if !ok {
				continue
			}
			caps, depl, vel = append(caps, m.Cap), append(depl, m.Deplacement), append(vel, m.Velocite)
			vit = append(vit, m.VitesseMPS)
		}
	}
	return caps, depl, vel, vit
}

// v1aMesureCap porte les trois caps d'une paire d'echantillons et la vitesse i1 du premier.
type v1aMesureCapPaire struct {
	Cap, Deplacement, Velocite, VitesseMPS float64
}

// v1aMesureCap rend les trois caps d'une paire d'echantillons consecutifs, et dit si la paire
// est retenue par la regle de selection.
func v1aMesureCap(a, b filmdec.BipedPosition) (v1aMesureCapPaire, bool) {
	var m v1aMesureCapPaire
	if !a.HasAim || !v0PasCompte(a.TimestampUS, b.TimestampUS) {
		return m, false
	}
	vitesse, okv := v1aVitesseMPS(a)
	if !okv || vitesse <= v1aVitesseMinMPS {
		return m, false
	}
	dx, dy := b.X-a.X, b.Y-a.Y
	if math.Hypot(float64(dx), float64(dy)) == 0 {
		return m, false
	}
	av, oka := a.AimVector()
	if !oka {
		return m, false
	}
	vv, _ := a.VelocityVector()
	m = v1aMesureCapPaire{
		Cap:         v1aCapDeg(av[0], av[1]),
		Deplacement: v1aCapDeg(dx, dy),
		Velocite:    v1aCapDeg(vv[0], vv[1]),
		VitesseMPS:  vitesse,
	}
	return m, true
}

// v1aPermutation rend un melange DETERMINISTE des indices 0..n-1.
//
// UN DECALAGE D'UN CRAN NE FAIT PAS UN TEMOIN, et la mesure du 2026-08-31 l'a montre : deux
// echantillons consecutifs du MEME slot sont separes d'une demi-seconde et portent presque le
// meme cap, de sorte que le decalage PRESERVE l'association qu'il est cense detruire — le
// temoin decale rendait 1,7 deg de mediane, exactement comme le vrai appariement. Un melange
// complet croise les slots et les instants : lui detruit bien l'association, et rien d'autre
// (la population de caps et celle de references sont inchangees).
func v1aPermutation(n int) []int {
	ordre := make([]int, n)
	for i := range ordre {
		ordre[i] = i
	}
	r := rand.New(rand.NewPCG(1, 2)) // graine figee : la mesure doit se rejouer a l'identique
	r.Shuffle(n, func(i, j int) { ordre[i], ordre[j] = ordre[j], ordre[i] })
	return ordre
}

// v1aPublieCap confronte une serie de caps a une serie de references, avec son temoin.
func v1aPublieCap(t *testing.T, f v0Film, quoi string, echantillons int, caps, ref v1aEcarts) {
	t.Helper()
	if len(caps) == 0 {
		t.Logf("V1.3 %s (%s) [%s] — aucune paire retenue (velocite i1 > %.0f m/s ET pas <= %d us)",
			f.ID, f.Carte, quoi, v1aVitesseMinMPS, v0PasMaxUS)
		return
	}
	ecarts := make(v1aEcarts, 0, len(caps))
	temoin := make(v1aEcarts, 0, len(caps))
	melange := v1aPermutation(len(caps))
	for i := range caps {
		ecarts = append(ecarts, v1aWrap180(caps[i]-ref[i]))
		temoin = append(temoin, v1aWrap180(caps[melange[i]]-ref[i]))
	}
	moy, med, r := ecarts.v1aStats()
	tmoy, tmed, tr := temoin.v1aStats()
	verdict := "ECHOUE"
	if math.Abs(moy) < v1aCapMoyenneMaxDeg && med < v1aCapMedianeMaxDeg {
		verdict = "PASSE"
	}
	t.Logf("V1.3 %s (%s) [%s] — %d echantillons, %d paires retenues · moyenne circulaire "+
		"%+.1f deg (seuil %.0f) · mediane |ecart| %.1f deg (seuil %.0f) · R %.3f · %s\n"+
		"    TEMOIN (appariement detruit par melange) : moyenne %+.1f deg · mediane |ecart| "+
		"%.1f deg · R %.3f",
		f.ID, f.Carte, quoi, echantillons, len(ecarts), moy, v1aCapMoyenneMaxDeg, med,
		v1aCapMedianeMaxDeg, r, verdict, tmoy, tmed, tr)
}

// ---------------------------------------------------------------------------------------
// V1a.4 — LE REJEU DE L'ORACLE GEOMETRIQUE DU 18/08, SUR DES POSITIONS JUSTES.
// ---------------------------------------------------------------------------------------

// TestV1aOracleGeometrique rejoue la coincidence prolongee (1,5 m pendant 3 s).
//
// POURQUOI LE REJEU. L'oracle du 18/08 tournait sur des positions de vehicule decodees par la
// grammaire des objets du monde, dont le cadrage a mesure qu'elle est FAUSSE pour `ti=40`. Son
// resultat n'etait donc pas « negatif », il n'etait pas mesure. Ici les deux nuages sortent du
// MEME decodeur, avec les MEMES bornes.
//
// DEUX CONTROLES SANS LESQUELS LE CHIFFRE NE VAUT RIEN :
//
//	INTERSECTION DES BANDES  si un slot appartient a la fois a la bande bipede et a la bande
//	                         vehicule, le meme record est lu des deux cotes et produit une
//	                         coincidence a distance nulle. C'est la premiere chose publiee.
//	TEMOIN FANTOME           les memes bipedes contre une bande de MEME cardinalite faite de
//	                         slots jamais vus porter le moindre archetype.
func TestV1aOracleGeometrique(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v1aGeometriqueUnFilm(t, root, f)
	}
}

// v1aGeometriqueUnFilm rejoue l'oracle sur UN film.
func v1aGeometriqueUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V1a.4 %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	// LE NUAGE BIPEDE GARDE LES REGLAGES DE PRODUCTION, `RequireTag1` COMPRIS : c'est ceux
	// qu'employait l'oracle du 18/08 (`attNuages`), et le rejeu ne doit changer QUE la grammaire
	// des positions de vehicule. Desarmer le tag ici ajouterait des faux positifs de bipede et
	// gonflerait mecaniquement les coincidences.
	optBip := filmdec.DefaultScanFilmOptions()
	optBip.WorldRange = &wr
	bip, err := filmdec.ScanFilmBipedPositions(dir, optBip)
	if err != nil {
		t.Logf("V1a.4 %s : balayage des bipedes : %v", f.ID, err)
		return
	}
	v1aIntersectionBandes(t, f, dir, bande, bip)
	veh, err := filmdec.ScanFilmBipedPositionsForBand(dir, bande, v1aOptions(&wr, true))
	if err != nil {
		t.Logf("V1a.4 %s : balayage des vehicules : %v", f.ID, err)
		return
	}
	pistes := v1aPistes(veh)
	v1aPublieOracle(t, f, "VEHICULES", pistes, bip)
	// LE SECOND VOLET DE L'ORACLE DU 18/08, sans lequel le rejeu serait partiel. Le modele du
	// plan dit qu'un enfant attache NE REPLIQUE PLUS sa position : si c'est vrai, un embarquement
	// se voit comme un TROU du flux de position dont le dernier point est pres d'un vehicule, et
	// le petit nombre de periodes s'EXPLIQUE au lieu d'infirmer. Les deux lectures ne valent que
	// publiees ensemble.
	attLogTrous(t, f.ID, pistes, bip)
	pres, tot := v1aPresenceDeFond(pistes, bip)
	t.Logf("V1a.4 %s (%s) — PRESENCE DE FOND : %d des %d echantillons de bipede sont a moins de "+
		"%.1f m d'un vehicule (%.1f %%). C'est le denominateur des trous : « X %% des trous "+
		"s'ouvrent pres d'un vehicule » ne dit rien tant qu'on ignore cette part.",
		f.ID, f.Carte, pres, tot, attBordRayonM, 100*attPart(pres, tot))
	vus, autres := attBandesKeyframe(dir)
	fantome := attBandeFantome(vus, autres)
	if len(fantome) == 0 {
		t.Logf("V1a.4 %s : aucun slot libre pour une bande fantome", f.ID)
		return
	}
	fveh, err := filmdec.ScanFilmBipedPositionsForBand(dir, fantome, v1aOptions(&wr, true))
	if err != nil {
		t.Logf("V1a.4 %s : bande fantome : %v", f.ID, err)
		return
	}
	v1aPublieOracle(t, f, "TEMOIN FANTOME", v1aPistes(fveh), bip)
}

// v1aIntersectionBandes publie le recouvrement des deux bandes — le premier controle.
func v1aIntersectionBandes(t *testing.T, f v0Film, dir string, bande map[uint32]bool,
	bip []filmdec.BipedPosition) {
	t.Helper()
	slotsBip := attSlotsBipede(bip)
	commun := 0
	for s := range slotsBip {
		if bande[s] {
			commun++
		}
	}
	t.Logf("V1a.4 %s (%s) — CONTROLE des bandes : %d slots de bipede EMIS, %d slots dans la "+
		"bande ti=%d, %d en COMMUN (un slot commun fabrique une coincidence a distance nulle) · "+
		"%d chunks", f.ID, f.Carte, len(slotsBip), len(bande), attVehiculeTI, commun,
		filmdec.CountFilmChunks(dir))
}

// v1aPresenceDeFond rend le nombre d'echantillons de bipede qui sont, A LEUR INSTANT, a moins
// de attBordRayonM d'un vehicule, et le total.
//
// C'EST LE DENOMINATEUR QUI MANQUAIT A L'ORACLE DU 18/08. « 46 % des trous du flux s'ouvrent
// pres d'un vehicule » ne dit rien tant qu'on ignore quelle part du TEMPS un bipede passe pres
// d'un vehicule : si c'est 45 %, le chiffre est du hasard ; si c'est 3 %, il est massif.
func v1aPresenceDeFond(veh []filmdec.ProjectileTrack, bip []filmdec.BipedPosition) (int, int) {
	pres, total := 0, 0
	for _, b := range bip {
		if !b.HasWorld {
			continue
		}
		total++
		if _, ok := attVehiculeLePlusProche(b, veh); ok {
			pres++
		}
	}
	return pres, total
}

// v1aPistes regroupe des positions par slot en pistes triees — la forme qu'attend l'oracle.
func v1aPistes(pos []filmdec.BipedPosition) []filmdec.ProjectileTrack {
	parSlot := map[uint32][]filmdec.ProjectileSample{}
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		parSlot[p.Slot] = append(parSlot[p.Slot], filmdec.ProjectileSample{
			TimestampUS: p.TimestampUS, Chunk: p.Chunk, X: p.X, Y: p.Y, Z: p.Z,
		})
	}
	slots := make([]uint32, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	out := make([]filmdec.ProjectileTrack, 0, len(slots))
	for _, s := range slots {
		pts := parSlot[s]
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimestampUS < pts[j].TimestampUS })
		out = append(out, filmdec.ProjectileTrack{Slot: s, Pts: pts})
	}
	return out
}

// v1aPublieOracle applique l'oracle et publie ce qu'il rend.
func v1aPublieOracle(t *testing.T, f v0Film, quoi string, veh []filmdec.ProjectileTrack,
	bip []filmdec.BipedPosition) {
	t.Helper()
	periodes := attPeriodesABord(veh, bip)
	med, tot := v1aDurees(periodes)
	cand, ambig := v1aCandidats(periodes)
	t.Logf("V1a.4 %s (%s) [%s] — %d pistes · %d periodes « a bord » (< %.1f m pendant >= %d ms) "+
		"· duree mediane %.1f s, cumul %.0f s · %d vehicules porteurs d'au moins une periode · "+
		"%d conducteurs candidats au maximum sur un meme vehicule · %d periodes CHEVAUCHEES par "+
		"une autre sur le meme vehicule",
		f.ID, f.Carte, quoi, len(veh), len(periodes), attBordRayonM, attBordDureeMS, med, tot,
		len(cand), v1aMaxCandidats(cand), ambig)
}

// v1aDurees rend la duree mediane et le cumul des periodes, en secondes.
func v1aDurees(periodes []attPeriode) (mediane, cumul float64) {
	if len(periodes) == 0 {
		return 0, 0
	}
	d := make([]float64, 0, len(periodes))
	for _, p := range periodes {
		s := float64(p.T1-p.T0) / 1e6
		d = append(d, s)
		cumul += s
	}
	sort.Float64s(d)
	return d[len(d)/2], cumul
}

// v1aCandidats rend, par vehicule, l'ensemble des bipedes candidats, et le nombre de periodes
// qu'une AUTRE periode du meme vehicule chevauche dans le temps — c'est cette derniere qui dit
// si l'attribution d'un conducteur est possible : deux candidats simultanes ne se departagent
// pas par la geometrie.
func v1aCandidats(periodes []attPeriode) (map[uint32]map[uint32]bool, int) {
	parVehicule := map[uint32][]attPeriode{}
	cand := map[uint32]map[uint32]bool{}
	for _, p := range periodes {
		parVehicule[p.VehiculeSlot] = append(parVehicule[p.VehiculeSlot], p)
		if cand[p.VehiculeSlot] == nil {
			cand[p.VehiculeSlot] = map[uint32]bool{}
		}
		cand[p.VehiculeSlot][p.BipedeSlot] = true
	}
	ambig := 0
	for _, ps := range parVehicule {
		for i, a := range ps {
			for j, b := range ps {
				if i == j || a.BipedeSlot == b.BipedeSlot {
					continue
				}
				if a.T0 <= b.T1 && b.T0 <= a.T1 {
					ambig++
					break
				}
			}
		}
	}
	return cand, ambig
}

// v1aMaxCandidats rend le plus grand nombre de bipedes candidats sur un meme vehicule.
func v1aMaxCandidats(cand map[uint32]map[uint32]bool) int {
	maxi := 0
	for _, s := range cand {
		if len(s) > maxi {
			maxi = len(s)
		}
	}
	return maxi
}
