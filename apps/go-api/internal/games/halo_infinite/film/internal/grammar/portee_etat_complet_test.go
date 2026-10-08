package grammar

// portee_etat_complet_test.go — LA PORTEE `DAT_144e61ea0` DE LA LECTURE D ETAT COMPLET
// ([Lecteur.portee]) : ou la marche la pose, ce qu elle change, et ce qui ne la pose jamais.
//
// La branche absolue d i0 sous la portee lit la forme de `FUN_1406cfe44` (h, R(96), queue, R(2) si
// les trois flottants sont finis) ; un flottant non fini et un handle dont la largeur n est pas
// etablie arretent la lecture ([ArretDuLecteur]). La marche d etat complet pose la portee autour
// de l etat par defaut (`n1 > 0`) et de la boucle de composants (`n2 > 0`), et la retire a chaque
// sortie ; le record NEW et le DELTA ne la posent pas.
//
// Mutations jouees le 2026-10-08, chacune rouge puis retiree : la portee posee dans
// `TraverseEntity` ; la remise a faux oubliee (etat par defaut, boucle) ; le R(2) lu avant la
// queue ; la portee posee sur la boucle sans l etre sur l etat par defaut.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// motFlottantFini : 32 bits du flux dont l image memoire (octets inverses) a un exposant non plein.
const motFlottantFini = 0x55555555

// motInfini : 32 bits du flux dont l image memoire est 0x7f800000 (+Inf). Lu sans inverser ses
// octets, il serait fini (exposant nul) : le test distingue les deux lectures.
const motInfini = 0x0000807f

// ecrireI0AbsoluSousLaPortee ecrit i0 tel que l ecrivain d etat complet le pose sur la branche
// absolue : bUsePred = 0, bDelta = 0, h, trois mots bruts, puis (h = 1) une queue sans handle
// resolu et a region etendue (sel = 0, region = 1, ext = 1, R(11)), puis le R(2). Rend le nombre
// de bits ecrits.
func ecrireI0AbsoluSousLaPortee(w *bitWriter, h bool, mots [3]uint64, avecR2 bool) int {
	debut := w.n
	w.bits(0, 2) // bUsePred, bDelta
	if h {
		w.bit(1)
	} else {
		w.bit(0)
	}
	for _, m := range mots {
		w.bits(m, 32)
	}
	if h {
		w.bit(0)              // sel : pas de handle resolu
		w.bit(1)              // region presente
		w.bit(1)              // region etendue
		w.bits(motif(11), 11) // mot de region
	}
	if avecR2 {
		w.bits(0b10, 2)
	}
	return w.n - debut
}

// troisMots rend trois mots identiques.
func troisMots(m uint64) [3]uint64 { return [3]uint64{m, m, m} }

// TestLaBrancheAbsolueDI0SousLaPorteeLitLaFormeDuJeu : h = 0, 1 + 1 + 1 + 96 + 2 = 101 bits ;
// h = 1, la queue entre les 96 bits et le R(2). Lire le R(2) avant la queue lirait 108 bits au
// lieu de 115.
func TestLaBrancheAbsolueDI0SousLaPorteeLitLaFormeDuJeu(t *testing.T) {
	for _, h := range []bool{false, true} {
		var w bitWriter
		n := ecrireI0AbsoluSousLaPortee(&w, h, troisMots(motFlottantFini), true)
		w.bits(^uint64(0), 64) // un lecteur qui deborde lit des bits poses
		br := sousLaPortee(LecteurSur(w.buf))
		consumeObjectPositionDynamicPrecisionD(br, br.traversal())
		attendu := 101
		if h {
			attendu = 115
		}
		if n != attendu || br.BitPos() != attendu || br.arret != ArretAucun {
			t.Errorf("h=%v : %d bits lus (ecrits %d), arret %v ; attendu %d, aucun arret", h,
				br.BitPos(), n, br.arret, attendu)
		}
	}
}

// TestUnFlottantNonFiniArreteLaLecture : `FUN_140492128` juge l image memoire du flottant ; un
// exposant plein fait echouer le lecteur, sans lire le R(2).
func TestUnFlottantNonFiniArreteLaLecture(t *testing.T) {
	var w bitWriter
	ecrireI0AbsoluSousLaPortee(&w, false, [3]uint64{motFlottantFini, motInfini, motFlottantFini}, true)
	br := sousLaPortee(LecteurSur(w.buf))
	consumeObjectPositionDynamicPrecisionD(br, br.traversal())
	if br.arret != ArretPositionNonFinie || br.BitPos() != 99 {
		t.Fatalf("flottant infini : arret %v apres %d bits, attendu %v apres 99 (sans le R(2))",
			br.arret, br.BitPos(), ArretPositionNonFinie)
	}
}

// TestLaGardeDuHandleArreteSousLeMoteurUn : sous la portee, un handle present (h = 1) dans un film
// qui n exclut pas le type de moteur 1 arrete la lecture avant la queue ; h = 0 se lit.
func TestLaGardeDuHandleArreteSousLeMoteurUn(t *testing.T) {
	for _, h := range []bool{false, true} {
		var w bitWriter
		ecrireI0AbsoluSousLaPortee(&w, h, troisMots(motFlottantFini), true)
		br := sousLaPortee(LecteurSur(w.buf))
		p := br.Profil()
		p.Grammaire.MoteurUnPossible = true
		br.PoserProfil(p)
		consumeObjectPositionDynamicPrecisionD(br, br.traversal())
		if h && (br.arret != ArretLargeurHandleMoteurUn || br.BitPos() != 99) {
			t.Errorf("h=1 sous le moteur 1 possible : arret %v apres %d bits, attendu %v apres 99",
				br.arret, br.BitPos(), ArretLargeurHandleMoteurUn)
		}
		if !h && (br.arret != ArretAucun || br.BitPos() != 101) {
			t.Errorf("h=0 sous le moteur 1 possible : arret %v apres %d bits, attendu aucun apres 101",
				br.arret, br.BitPos())
		}
	}
}

// enTeteDImageCle ecrit l en-tete de 108 bits d un record d etat complet d archetype `ti` ; les mots
// de taille et ce qui suit sont a l appelant.
func enTeteDImageCle(w *bitWriter, ti uint64) {
	w.bits(0, keyframeRecordTIBit)
	w.bits(ti, 6)
	w.bits(0, 108-keyframeRecordTIBit-6)
}

// registreI0 rend un registre dont l archetype 35 ne porte que i0.
func registreI0() *Registry {
	reg := &Registry{Archetypes: make([]Archetype, 41)}
	reg.Archetypes[35] = Archetype{Index: 35, Components: []string{kf7dI0}}
	return reg
}

// TestLaMarcheDEtatCompletPoseLaPortee : un etat complet `n1 = 0`, `n2 = 1` dont i0 est absolu se
// lit sous la portee et ferme ; un flottant non fini l arrete au debut de i0, cause nommee ; un
// handle sous le moteur 1 possible aussi.
func TestLaMarcheDEtatCompletPoseLaPortee(t *testing.T) {
	cas := []struct {
		nom      string
		h        bool
		mots     [3]uint64
		moteurUn bool
		arret    ArretDuLecteur
	}{
		{"fini, h = 0", false, troisMots(motFlottantFini), false, ArretAucun},
		{"fini, h = 1", true, troisMots(motFlottantFini), false, ArretAucun},
		{"infini", false, [3]uint64{motInfini, motFlottantFini, motFlottantFini}, false, ArretPositionNonFinie},
		{"h = 1, moteur 1 possible", true, troisMots(motFlottantFini), true, ArretLargeurHandleMoteurUn},
	}
	for _, c := range cas {
		var w bitWriter
		enTeteDImageCle(&w, 35)
		w.bits(0, 32) // n1 = 0
		w.bits(1, 32) // n2 = 1
		debutI0 := w.n
		n := ecrireI0AbsoluSousLaPortee(&w, c.h, c.mots, true)
		w.bits(^uint64(0), 64)
		ctx := ContexteParDefaut()
		ctx.Profil.Grammaire.MoteurUnPossible = c.moteurUn
		tr := WalkKeyframeFullState(w.buf, 0, registreI0(), ctx)
		switch {
		case c.arret == ArretAucun && (tr.DesyncAt != -1 || tr.EndBit != debutI0+n):
			t.Errorf("%s : arret a %d (%v), fin %d ; attendu aucun arret, fin %d", c.nom, tr.DesyncAt,
				tr.Arret, tr.EndBit, debutI0+n)
		case c.arret != ArretAucun && (tr.Arret != c.arret || tr.DesyncAt != 0 || tr.EndBit != debutI0):
			t.Errorf("%s : arret %v a %d, fin %d ; attendu %v a 0, fin %d (debut d i0)", c.nom, tr.Arret,
				tr.DesyncAt, tr.EndBit, c.arret, debutI0)
		}
	}
}

// largeurVersionEtMPPNuls rend la largeur que lisent, sur un flux nul, les deux premieres feuilles
// de l etat par defaut `ti=40` (version, bloc MPP).
func largeurVersionEtMPPNuls() int {
	br := LecteurSur(make([]byte, 64))
	consumeVersionPrefix(br)
	lireLeBlocMPPDeLEtat(br)
	return br.BitPos()
}

// TestLaPorteeCouvreLEtatParDefaut : sous la portee de `FUN_142e2bfd0` (`n1 > 0`), la feuille
// quaternion de l etat par defaut `ti=40` (`FUN_14076e494`) lit R(96) ; `n2 = 0` arrete le record
// sur son second mot de taille.
func TestLaPorteeCouvreLEtatParDefaut(t *testing.T) {
	var w bitWriter
	enTeteDImageCle(&w, 40)
	w.bits(1, 32) // n1 = 1
	w.bits(0, largeurVersionEtMPPNuls())
	w.bit(1) // porte bVar14 : la feuille quaternion suit
	for range 3 {
		w.bits(motFlottantFini, 32) // FUN_14076e494 sous la portee : R(96)
	}
	w.bit(1)        // FUN_140c1e79c : porte a 1, pas de R(19)
	w.bits(0xa5, 8) // FUN_140c1e79c : magnitude R(8)
	w.bits(0, 19)   // FUN_14076dc04
	w.bits(0, 2)    // porte cVar3 = 0, opt32 ferme
	w.bits(0, 32)   // n2 = 0
	fin := w.n
	w.bits(^uint64(0), 64)
	reg := &Registry{Archetypes: make([]Archetype, 41)}
	reg.Archetypes[40] = Archetype{Index: 40, Components: []string{compVehicleEmpTimer}}
	tr := WalkKeyframeFullState(w.buf, 0, reg, ContexteParDefaut())
	if tr.TypeIndex != 40 || tr.DesyncAt != -1 || tr.EndBit != fin {
		t.Fatalf("etat par defaut ti=40 sous la portee : ti=%d arret %d fin %d, attendu ti=40, aucun "+
			"arret, fin %d", tr.TypeIndex, tr.DesyncAt, tr.EndBit, fin)
	}
}

// TestLaTrameMediaDeLEtatParDefautBipedeSousLaPortee : la trame media de l etat par defaut du
// bipede (`FUN_140f44c38`, CALL 142451b5d -> `FUN_14076e494`) lit R(96) sous la portee, puis
// `FUN_1407f2058` ; hors portee, la meme feuille lit la position quantifiee (porte, index, axes).
func TestLaTrameMediaDeLEtatParDefautBipedeSousLaPortee(t *testing.T) {
	var w bitWriter
	for range 3 {
		w.bits(motFlottantFini, 32)
	}
	w.bit(1) // FUN_1407f2058 : porte a 1, pas de R(5)
	w.bits(^uint64(0), 64)
	br := sousLaPortee(LecteurSur(w.buf))
	consumeBipedDefaultStateMediaFrame(br)
	if br.BitPos() != 97 {
		t.Fatalf("trame media sous la portee : %d bits lus, attendu 96 + 1", br.BitPos())
	}
	hors := LecteurSur(w.buf)
	consumeBipedDefaultStateMediaFrame(hors)
	if hors.BitPos() == 97 {
		t.Fatal("trame media hors portee : 97 bits lus, la lecture quantifiee ne se distingue pas")
	}
}

// TestLaPorteeRetombeASaSortie : le bloc de l etat par defaut et la boucle de composants rendent
// un lecteur hors portee, la boucle aussi quand un lecteur de composant echoue.
func TestLaPorteeRetombeASaSortie(t *testing.T) {
	var w bitWriter
	w.bits(1, 32) // n1 = 1 ; l etat par defaut de ti=11 : version R(1) = 0
	w.bit(0)
	w.bits(1, 32) // n2 = 1
	br := LecteurSur(w.buf)
	if !consumeFullStateDefaultBlock(br, 11, false) || br.portee {
		t.Fatalf("bloc de l etat par defaut : portee %v en sortie, attendu faux", br.portee)
	}
	for _, mots := range [][3]uint64{troisMots(motFlottantFini), {motInfini, motInfini, motInfini}} {
		var wi bitWriter
		ecrireI0AbsoluSousLaPortee(&wi, false, mots, true)
		bi := LecteurSur(wi.buf)
		tr := EntityTrace{DesyncAt: -1, Mask: ^uint64(0)}
		traverserSousLaPortee(bi, registreI0().Archetypes[35], &tr)
		if bi.portee || bi.arret != ArretAucun {
			t.Errorf("boucle (arret %v) : portee %v, arret du lecteur %v en sortie ; attendu faux, aucun",
				tr.Arret, bi.portee, bi.arret)
		}
	}
}

// TestLeRecordNeufEtLeDeltaNePosentPasLaPortee : un record NEW ([TraverseEntity]) et un DELTA
// ([decodeDelta]) lisent la branche absolue d i0 hors portee — 47 bits (`FUN_14076e524` a la largeur
// de la carte, puis le R(2)), et non 101.
func TestLeRecordNeufEtLeDeltaNePosentPasLaPortee(t *testing.T) {
	reg := &Registry{Archetypes: make([]Archetype, 12)}
	reg.Archetypes[11] = Archetype{Index: 11, Components: []string{kf7dI0}}
	ecrireI0Quantifie := func(w *bitWriter) {
		w.bits(0, 2)          // bUsePred, bDelta
		w.bit(0)              // precHigh
		w.bit(0)              // porte : l index suit
		w.bit(0)              // index de region (1 bit)
		w.bits(motif(13), 13) // axes de la carte (13/13/14)
		w.bits(motif(13), 13) //
		w.bits(motif(14), 14) //
		w.bits(0b01, 2)       // R(2)
	}
	masqueI0 := func(w *bitWriter) { // masque epars {0}
		w.bit(0)
		w.bits(1, 3)
		w.bits(0, 6)
	}

	var wn bitWriter
	wn.bits(11, 6) // typeIndex
	wn.bit(0)      // etat par defaut de ti=11 : version R(1) = 0
	wn.bit(0)      // porte du record NEW
	masqueI0(&wn)
	ecrireI0Quantifie(&wn)
	finNeuf := wn.n
	wn.bits(^uint64(0), 64)
	if tr := TraverseEntity(LecteurSur(wn.buf), reg, 0); tr.DesyncAt != -1 || tr.EndBit != finNeuf {
		t.Errorf("record NEW : arret %d fin %d, attendu aucun arret, fin %d", tr.DesyncAt, tr.EndBit, finNeuf)
	}

	var wd bitWriter
	wd.bit(0) // selecteur de base
	masqueI0(&wd)
	ecrireI0Quantifie(&wd)
	finDelta := wd.n
	wd.bits(^uint64(0), 64)
	monde := NewWorld(reg)
	monde.BindWildcard(7, 11)
	if tr := decodeDelta(LecteurSur(wd.buf), monde, 7); tr.DesyncAt != -1 || tr.EndBit != finDelta {
		t.Errorf("DELTA : arret %d fin %d, attendu aucun arret, fin %d", tr.DesyncAt, tr.EndBit, finDelta)
	}
}

// TestLeTypeDeMoteurVientDuFilm : [moteurUnPossible] suit ce que le film declare, et un contexte de
// film le derive a chaque rendu, par-dessus un profil pose.
func TestLeTypeDeMoteurVientDuFilm(t *testing.T) {
	avec := func(v profile.VarianteDePartie) profile.Profile {
		id := profile.FilmIdentity{Build: "HI_1_13_0", Variante: v}
		return profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Identite: id, IdentiteLue: true}, nil)
	}
	cas := []struct {
		nom     string
		p       profile.Profile
		attendu bool
	}{
		{"identite non lue", profile.Resoudre(profile.ClesDuFilm{}, nil), true},
		{"variante non lue", avec(profile.VarianteDePartie{}), true},
		{"variante absente", avec(profile.VarianteDePartie{Lue: true}), false},
		{"type 2", avec(profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: 2}), false},
		{"type 1", avec(profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: 1}), true},
	}
	for _, c := range cas {
		if g, _ := grammaireSousFilm(grammaireDuProfil(), c.p); g.MoteurUnPossible != c.attendu {
			t.Errorf("%s : MoteurUnPossible %v, attendu %v", c.nom, g.MoteurUnPossible, c.attendu)
		}
	}
	fc := NewFilmContextForMap(bobineDeContexte(t, "fb1a1a72"), nil, nil)
	etranger := ProfilDeBalayageParDefaut()
	etranger.Grammaire.MoteurUnPossible = true
	fc.PoserProfilDeBalayage(etranger)
	if fc.ProfilDeBalayage().Grammaire.MoteurUnPossible {
		t.Error("contexte de film (type 2) : un profil pose lui rend le type de moteur 1 possible")
	}
	if p := fc.PreuveDImageCle(); p == nil || p.ctx.Profil.Grammaire.MoteurUnPossible {
		t.Error("preuve d image-cle : le type de moteur du film n y est pas derive")
	}
}
