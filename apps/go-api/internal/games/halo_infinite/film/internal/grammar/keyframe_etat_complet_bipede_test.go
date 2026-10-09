package grammar

// keyframe_etat_complet_bipede_test.go — la lecture de l etat complet du bipede aux images-cles
// (D1.1 du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`) : le cadre de la relecture,
// la regle d admission (U-1 amendee le 2026-10-09), la publication et les sept bobines du golden en
// contexte de cuisson.
//
// Mutations jouees le 2026-10-09, chacune rouge puis retiree : portee retiree de
// [relecteurDEtatComplet] ; selection d i47 publiee en base 1 ; T1 ou T2 retire d [admettre].

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestLeRelecteurPoseLeCadreDeLaMarche : la relecture d une occurrence se fait sous le cadre de la
// boucle d etat complet (etat complet ET portee), le relecteur d etat par defaut sous aucun.
func TestLeRelecteurPoseLeCadreDeLaMarche(t *testing.T) {
	obs := &Observation{}
	br := relecteurDEtatComplet(make([]byte, 8), ContexteDeLecture{}, obs)
	if !br.etatComplet || !br.portee || br.obs != obs {
		t.Fatalf("relecteur d etat complet : etatComplet=%v portee=%v obs pose=%v, attendu vrai, vrai, vrai",
			br.etatComplet, br.portee, br.obs == obs)
	}
	nu := relecteurSur(make([]byte, 8), ContexteDeLecture{})
	if nu.etatComplet || nu.portee {
		t.Fatalf("relecteur d etat par defaut : etatComplet=%v portee=%v, attendu faux, faux", nu.etatComplet, nu.portee)
	}
}

// familleConnue rend une famille du catalogue de la grammaire.
func familleConnue(t *testing.T) uint32 {
	t.Helper()
	for f := range weaponv3.KnownWeaponHigh32Copie() {
		return f
	}
	t.Fatal("catalogue des familles vide")
	return 0
}

// lectureAdmissible rend une lecture qui passe T1 et T2 : une arme connue en emplacement 0, deux
// grenades du premier type, masque i47 = bitmap.
func lectureAdmissible(t *testing.T) lectureDEtatComplet {
	t.Helper()
	l := lectureDEtatComplet{grenadesLues: true, compte: 4, compteurs: []uint64{2, 0, 0, 0},
		jeuDeGrenadesLu: true, masque: 0b0001, selection: 1, rang: AbilitySetNoRank, vitalites: -1}
	for k := range emplacementsDArme {
		l.armeLue[k], l.armeHaute[k] = true, noVariant
	}
	l.armeHaute[0] = familleConnue(t)
	return l
}

// TestLaRegleDAdmission : (ferme OU n(i22) = 4) ET T1 ET T2, raison par raison.
func TestLaRegleDAdmission(t *testing.T) {
	const dernier = 46
	ferme := lecture.Record{Preuve: lecture.PreuveFerme, Desync: lecture.SansDesynchronisation}
	nonProuve := lecture.Record{Preuve: lecture.PreuveNonProuve, Desync: lecture.SansDesynchronisation}
	cas := []struct {
		nom     string
		rec     lecture.Record
		changer func(*lectureDEtatComplet)
		attendu raisonDeRefus
	}{
		{"ferme, T1 et T2", ferme, func(*lectureDEtatComplet) {}, refusAucun},
		{"non ferme, n(i22) = 4, T1 et T2", nonProuve, func(*lectureDEtatComplet) {}, refusAucun},
		{"non ferme, n(i22) = 3", nonProuve, func(l *lectureDEtatComplet) { l.compte = 3 }, refusNiFermeNiI22},
		{"tous les emplacements vides", ferme, func(l *lectureDEtatComplet) { l.armeHaute[0] = noVariant }, refusT1},
		{"famille hors catalogue", ferme, func(l *lectureDEtatComplet) { l.armeHaute[1] = 0x00007ca9 }, refusT1},
		{"marche arretee avant le dernier emplacement", lecture.Record{Preuve: lecture.PreuveNonProuve, Desync: 45},
			func(*lectureDEtatComplet) {}, refusT1},
		{"masque d i47 different de la bitmap", ferme, func(l *lectureDEtatComplet) { l.masque = 0b0011 }, refusT2},
		{"i47 non lu", ferme, func(l *lectureDEtatComplet) { l.jeuDeGrenadesLu = false }, refusT2},
		{"ferme, n(i22) = 3", ferme, func(l *lectureDEtatComplet) { l.compte = 3 }, refusT2},
		// Un debordement refuse le record meme ferme, T1 et T2 tenus : ses crochets ont pu ecrire des
		// valeurs lues a une autre largeur (constat 1 de la revue D1.4.6).
		{"ferme, T1 et T2, une occurrence deborde", ferme, func(l *lectureDEtatComplet) { l.debordements = 1 },
			refusDebordement},
	}
	for _, c := range cas {
		l := lectureAdmissible(t)
		c.changer(&l)
		rec := c.rec
		if got := admettre(&rec, &l, dernier); got != c.attendu {
			t.Errorf("%s : verdict %d, attendu %d", c.nom, got, c.attendu)
		}
	}
}

// TestLaPublicationDUnRecordAdmis : selection d i47 en base 1 (0 = aucune), rang de capacite borne
// au domaine 16..23, emplacement desire en main principale, munitions des quatre emplacements.
func TestLaPublicationDUnRecordAdmis(t *testing.T) {
	l := lectureAdmissible(t)
	l.jeuLu, l.jeu = true, JeuDArmes{Demande: 5, Principal: 1, Second: -1}
	l.capaciteLue, l.rang = true, 20
	for k := range emplacementsDArme {
		l.chargeurLu[k], l.reserveLue[k], l.reserve[k] = true, true, uint32(10*k) //nolint:gosec // petit
	}
	l.aChargeur[0], l.chargeur[0] = true, 31
	inv, hors := l.inventaireDe(7)
	if hors || inv.AbilityRank != 20 || inv.SelectedGrenadeRank != 0 || inv.DrawnSlot != 1 || !inv.GrenadesRead ||
		inv.Grenades[0] != 2 || !inv.AmmoRead || inv.Ammo[0].Mag == nil || *inv.Ammo[0].Mag != 31 ||
		inv.Ammo[1].Mag != nil || *inv.Ammo[3].Res != 30 {
		t.Fatalf("inventaire publie inattendu : %+v (hors domaine %v)", inv, hors)
	}
	l.selection, l.rang, l.jeu.Principal = GrenadeSetNoSelection, 3, -1
	l.reserveLue[2] = false
	inv, hors = l.inventaireDe(7)
	if !hors || inv.AbilityRank != -1 || inv.SelectedGrenadeRank != -1 || inv.DrawnSlot != -1 || inv.AmmoRead {
		t.Fatalf("selection nulle, rang 3, aucun emplacement desire, reserve manquante : %+v (hors domaine %v)", inv, hors)
	}
	if got := l.armesDe(7).Families; len(got) != 1 || got[0] != l.armeHaute[0] {
		t.Fatalf("armes publiees %08x, attendu la seule famille de l emplacement 0", got)
	}
}

// TestLaMarqueDePortageEstLaConfigurationDeLaFenetre : la marque exige les trois champs.
func TestLaMarqueDePortageEstLaConfigurationDeLaFenetre(t *testing.T) {
	l := lectureDEtatComplet{mortParDefaut: true, echelleAUn: true, vitalites: vitalitesDeLaMarque}
	if !l.porteLaMarque() {
		t.Fatal("configuration complete : marque attendue")
	}
	for _, c := range []lectureDEtatComplet{
		{mortParDefaut: false, echelleAUn: true, vitalites: vitalitesDeLaMarque},
		{mortParDefaut: true, echelleAUn: false, vitalites: vitalitesDeLaMarque},
		{mortParDefaut: true, echelleAUn: true, vitalites: 0b00011},
	} {
		if c.porteLaMarque() {
			t.Errorf("configuration incomplete %+v : marque inattendue", c)
		}
	}
}

// admissionsDesBobines : les comptes de la regle sur les sept bobines du golden en contexte de
// cuisson, mesures le 2026-10-09 (D1.1). Une baisse des admis ou un debordement rougit.
var admissionsDesBobines = map[string][2]int{ // bobine -> {bipedes, admis}
	"a521164d": {209, 10}, "60ae07c4": {236, 155}, "11de8353": {255, 205}, "111fa685": {214, 159},
	"e5adf7b2": {237, 167}, "bcb6d393": {137, 91}, "fb1a1a72": {80, 50},
}

// TestLEtatCompletDesBobines : sur chaque bobine du golden, aucune relecture ne deborde, chaque
// record bipede recoit un verdict, chaque record non admis passe aux fenetres (compte, marque
// recupere) et les familles publiees sont au catalogue.
func TestLEtatCompletDesBobines(t *testing.T) {
	for _, court := range closureMiniFilms() {
		e := etatsDeLaBobine(t, court)
		a := e.Admission
		if a.Debordements != 0 {
			t.Errorf("%s : %d relecture(s) debordent l etendue de la marche", court, a.Debordements)
		}
		if a.Admis+a.RefusNiFermeNiI22+a.RefusT1+a.RefusT2+a.RefusDebordement+a.SansCorps != a.Bipedes {
			t.Errorf("%s : verdicts %+v ne couvrent pas les %d bipedes", court, a, a.Bipedes)
		}
		verifierLesRepliesDeLaBobine(t, court, e)
		for _, l := range e.Loadouts {
			for _, f := range l.Families {
				if _, ok := weaponv3.KnownWeaponHigh32Lookup(f); !ok {
					t.Errorf("%s : famille %08x hors catalogue publiee", court, f)
				}
			}
		}
		attendu := admissionsDesBobines[court]
		if a.Bipedes != attendu[0] || a.Admis < attendu[1] {
			t.Errorf("%s : %d bipedes, %d admis ; attendu %d bipedes, au moins %d admis", court, a.Bipedes, a.Admis,
				attendu[0], attendu[1])
		}
	}
}

// contexteDeLaBobine rend le contexte d une bobine du golden, en contexte de cuisson.
func contexteDeLaBobine(t *testing.T, court string) *FilmContext {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entree, err := cat.Lookup(carteDesFaitsDEquivalence(t, court))
	if err != nil {
		t.Fatalf("carte de %s : %v", court, err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	fc.PoserLaCarteEtLeDecoupage()
	return fc
}

// etatsDeLaBobine rend la lecture des images-cles d une bobine du golden, en contexte de cuisson, et
// verifie que ses replis sont notes sur le contexte du film.
func etatsDeLaBobine(t *testing.T, court string) EtatsDesImagesCles {
	t.Helper()
	fc := contexteDeLaBobine(t, court)
	e, err := ScanEtatsDesImagesCles(fc, catalogueDesFamilles(), 0)
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	if r := fc.ComptesDesReplis(); r.FenetresArmesImageCle != e.Admission.FenetresArmes ||
		r.FenetresInventaireImageCle != e.Admission.FenetresInventaire ||
		r.FenetresMarqueDePortage != e.Admission.FenetresMarque {
		t.Errorf("%s : comptes notes sur le contexte %+v, attendu ceux de l admission %+v", court, r, e.Admission)
	}
	t.Logf("%s\t%d bipedes\t%d admis", court, e.Admission.Bipedes, e.Admission.Admis)
	return e
}

// catalogueDesFamilles rend le catalogue des familles d arme de la grammaire.
func catalogueDesFamilles() map[uint32]bool {
	known := map[uint32]bool{}
	for f := range weaponv3.KnownWeaponHigh32Copie() {
		known[f] = true
	}
	return known
}
