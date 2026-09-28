package grammar

// lecteur_position_sites_test.go — UN FLUX PAR SITE D APPEL DE `FUN_14076e524` (lot J6.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25).
//
// Chaque cas fabrique le flux de bits qu ECRIT le jeu pour UN site d appel du lecteur de position
// quantifiee — l immediat de niveau du site, la porte d index, l index sur `DAT_144632be0` bits,
// puis les trois axes aux largeurs de la ligne de CE niveau — et exige que le desserialiseur Go du
// site en consomme EXACTEMENT la longueur. Les sites, leurs appelants, leurs adresses d appel et
// leurs immediats sont ceux du releve `.ai/V7.5/film_re/RELEVES_J6_GHIDRA_2026-09-27.md` (§1, §6),
// completes le 2026-09-27 pour les trois lecteurs Go que le releve laissait « a relever »
// (descripteurs resolus en lecture seule, cf. `lecteur_position.go`).
//
// LES LARGEURS ATTENDUES SONT ECRITES EN CLAIR ICI, pas recalculees par la loi que le code emploie :
// un test qui appellerait `profile.LargeursAxeDuNiveau` pour fabriquer son attendu validerait la
// loi par elle-meme.
//
//	table DEFAUT (+/-20000), niveau 0x10       22/22/22   (releve §2.b, `DAT_1445cc9e0 + 0x10*0xc`)
//	table DEFAUT et table PAR INDEX, niveau 0x1E  26/26/26  (pas(30) < 1e-4, releve §1.b)
//	table PAR INDEX de Cliffhanger, niveau 0x10  13/13/14   (catalogue, defaut du profil)
//	bornes +/-100 (`DAT_143b8c6d0`), niveau 0x10  14/14/14  (`FUN_141f85880`, releve §2.c)

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// champDeFlux est un champ de largeur fixe, dans l ordre ou l ecrivain le pose.
type champDeFlux struct {
	v uint64
	n uint
}

// Largeurs attendues, en clair (cf. l en-tete).
var (
	axesDefautNiveau16   = [3]uint{22, 22, 22}
	axesNiveau30         = [3]uint{26, 26, 26}
	axesCarteNiveau16    = [3]uint{13, 13, 14}
	axesPrecHautNiveau16 = [3]uint{14, 14, 14}
)

// motif rend une valeur alternee de n bits — un flux fait de zeros ne distinguerait pas une porte
// lue d une porte sautee.
func motif(n uint) uint64 {
	var v uint64
	for i := uint(0); i < n; i++ {
		if i%2 == 0 {
			v |= 1 << i
		}
	}
	return v
}

// e524 rend la charge de `FUN_14076e524` telle que l ecrit `FUN_1407eb6a8` : la porte (1 = pas
// d index, table DEFAUT), l index sur `indexW` bits, puis les trois axes.
func e524(idx int, indexW uint, axes [3]uint) []champDeFlux {
	out := []champDeFlux{}
	if idx < 0 {
		out = append(out, champDeFlux{1, 1})
	} else {
		out = append(out, champDeFlux{0, 1}, champDeFlux{uint64(idx), indexW})
	}
	for _, w := range axes {
		out = append(out, champDeFlux{motif(w), w})
	}
	return out
}

// axesSeuls rend trois axes, sans porte ni index (`FUN_141f85880`).
func axesSeuls(axes [3]uint) []champDeFlux {
	return seul(fixe(axes[0]), fixe(axes[1]), fixe(axes[2]))
}

// fixe rend un champ de n bits au motif alterne.
func fixe(n uint) champDeFlux { return champDeFlux{motif(n), n} }

// bit rend un bit pose ou non.
func bit(b bool) champDeFlux {
	if b {
		return champDeFlux{1, 1}
	}
	return champDeFlux{0, 1}
}

// concat met bout a bout des morceaux de flux.
func concat(parts ...[]champDeFlux) []champDeFlux {
	var out []champDeFlux
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// seul fait d un champ un morceau.
func seul(c ...champDeFlux) []champDeFlux { return c }

// ecrireFlux rend les octets du flux et sa longueur en bits, suivis d une queue de 64 bits a 1 :
// un lecteur qui deborde lit des bits POSES, pas des zeros qui imiteraient une porte fermee.
func ecrireFlux(champs []champDeFlux) ([]byte, int) {
	w := &bitWriterMSB{}
	total := 0
	for _, c := range champs {
		w.put(c.v, c.n)
		total += int(c.n)
	}
	w.put(^uint64(0), 64)
	return w.buf, total
}

// lecteurDeSite rend un lecteur pose sur le flux, sous le profil par defaut (Cliffhanger : index
// de plage sur 1 bit, table par index 13/13/14) ou sous une carte a quatre plages (Live Fire :
// index sur 2 bits, `DAT_144632be0 = ceilLog2(4)`).
func lecteurDeSite(buf []byte, indexW uint) *Lecteur {
	br := LecteurSur(buf)
	if indexW != 1 {
		p := br.Profil()
		p.Mouvement.WorldObject = profile.PrecisionDescriptor{IndexW: indexW, AxisW: axesCarteNiveau16, Region: 1}
		br.PoserProfil(p)
	}
	return br
}

// casDeSite est UN flux d UN site d appel.
type casDeSite struct {
	nom    string
	indexW uint
	flux   []champDeFlux
	lire   func(br *Lecteur)
	// exception est la cle d une exception datee du portage (`exceptionsDuPortage`) : tant qu elle
	// tient, le site garde son ancien lecteur et le flux du jeu doit ne PAS s y fermer.
	exception string
}

// parNom rend la lecture d un composant par la chaine de dispatch de production.
func parNom(nom string, ti, niveauRegistre uint32) func(br *Lecteur) {
	return func(br *Lecteur) {
		if _, _, ok := consumeByName(br, nom, ti, niveauRegistre); !ok {
			panic(fmt.Sprintf("%s : non porte", nom))
		}
	}
}

// casDesSitesDePosition rend les flux, un ou plusieurs par site d appel.
func casDesSitesDePosition() []casDeSite {
	var cas []casDeSite
	// ti=21 i16 flock-position : FUN_140ee7270, CALL 140ee7293, niveau 0x10 (140ee7288) ; garde
	// f91c (0 bit hors portee) puis e524. GA2-2.
	for _, idx := range []int{-1, 0} {
		cas = append(cas, casDeSite{nom: fmt.Sprintf("flock-position idx=%d", idx), indexW: 1, exception: "flock-position",
			flux: e524(idx, 1, choisir(idx, axesDefautNiveau16, axesCarteNiveau16)),
			lire: parNom("flock-position-component", 21, 0)})
	}
	// GA2-3 : la largeur d index est celle de la CARTE (2 bits sur une carte a quatre plages).
	cas = append(cas, casDeSite{nom: "flock-position idx=1 carte a quatre plages", indexW: 2, exception: "flock-position",
		flux: e524(1, 2, axesCarteNiveau16), lire: parNom("flock-position-component", 21, 0)})
	// ti=44 i0 asset-transform : FUN_142ed3c64 -> 5 x FUN_142ed9530 -> FUN_14076e494(0x1E),
	// CALL 142ed9556 ; niveau 30 : 26 bits par axe dans les DEUX tables.
	for _, idx := range []int{-1, 0} {
		var f []champDeFlux
		for i := 0; i < 5; i++ {
			f = append(f, e524(idx, 1, axesNiveau30)...)
		}
		cas = append(cas, casDeSite{nom: fmt.Sprintf("asset-transform idx=%d", idx), indexW: 1,
			flux: f, lire: parNom("asset-transform-component", 44, 1)})
	}
	// ti=38 i18 generic-rigid-body-transforms : FUN_142f036f0, CALL 142f03837, niveau 0x10 ;
	// masque R(8), puis par bit FUN_140c1e79c (R(1) ; si 0 R(19) ; R(8)) et e494.
	cas = append(cas, casDeSite{nom: "generic-rigid-body-transforms", indexW: 1, exception: "ti38-i18",
		flux: concat(seul(champDeFlux{0b101, 8}),
			seul(bit(false), fixe(19), fixe(8)), e524(0, 1, axesCarteNiveau16),
			seul(bit(true), fixe(8)), e524(-1, 1, axesDefautNiveau16)),
		lire: parNom("generic-rigid-body-transforms-component", 38, 0)})
	cas = append(cas, casDesObjetsDuMonde()...)
	cas = append(cas, casDesTacmaps()...)
	cas = append(cas, casDesVecteursDeJoueur()...)
	return cas
}

// choisir rend `a` pour la table defaut (idx < 0), `b` sinon.
func choisir(idx int, a, b [3]uint) [3]uint {
	if idx < 0 {
		return a
	}
	return b
}

// casDesObjetsDuMonde : world-object i0 (FUN_14076e29c -> FUN_14076e420(0x10), CALL 14076e2c0),
// l etat d acteur d unite (FUN_14058c058, CALLs 1422cddc1 / 1422cde0e, niveau 0x10) et la trame
// media de l etat par defaut du bipede (FUN_140f44c38, CALL 142451b5d, niveau 0x10).
func casDesObjetsDuMonde() []casDeSite {
	wo := parNom(compObjectPosition, 38, 0)
	return []casDeSite{
		// precHigh = 0, porte posee : table DEFAUT, 22/22/22 — GA2-5.
		{nom: "world-object i0 precHigh=0 idx=-1", indexW: 1, exception: "world-object-i0",
			flux: concat(seul(bit(false)), e524(-1, 1, axesDefautNiveau16), seul(fixe(2))), lire: wo},
		{nom: "world-object i0 precHigh=0 idx=0", indexW: 1,
			flux: concat(seul(bit(false)), e524(0, 1, axesCarteNiveau16), seul(fixe(2))), lire: wo},
		// precHigh = 1 : FUN_141f85880 (3 x 14 sur +/-100) puis la queue mesuree de 17 bits.
		{nom: "world-object i0 precHigh=1", indexW: 1,
			flux: concat(seul(bit(true)), axesSeuls(axesPrecHautNiveau16), seul(fixe(17))), lire: wo},
		// unit-actor-state : 64 + R(8) (param 1) + R(4), puis 5 emplacements ; le premier present,
		// a = 0 et b = 0 : R(32), e494, R(8) de queue, FUN_14080d69c ferme.
		{nom: "unit-actor-state emplacement a=0 b=0", indexW: 1,
			flux: concat(seul(fixe(32), fixe(32), fixe(8), fixe(4)),
				seul(bit(true), bit(false), bit(false), fixe(32)), e524(0, 1, axesCarteNiveau16),
				seul(fixe(8), bit(false)),
				seul(bit(false), bit(false), bit(false), bit(false))),
			lire: parNom("unit-actor-state-component", 35, 1)},
		{nom: "trame media de l etat par defaut du bipede", indexW: 1,
			flux: concat(e524(-1, 1, axesDefautNiveau16), seul(bit(false), fixe(5))),
			lire: consumeBipedDefaultStateMediaFrame},
		{nom: "trame media, carte a quatre plages", indexW: 2,
			flux: concat(e524(3, 2, axesCarteNiveau16), seul(bit(true))),
			lire: consumeBipedDefaultStateMediaFrame},
	}
}

// casDesTacmaps : les tacmaps, le filtre d apparition et la zone selectionnable — tous au niveau
// 0x10, par `FUN_14076e494` ou le thunk `FUN_1424e0e38`.
func casDesTacmaps() []casDeSite {
	return []casDeSite{
		// ti=30 i0 tacmap-poiicon : FUN_142ed8418, CALL 142ed86d7 (thunk, param_6 = 0 : pas de precHigh).
		// EXCEPTION DATEE (lot J6-bis, 2026-09-28) : le site garde son ancien lecteur.
		{nom: "tacmap-poiicon", indexW: 1, exception: "tacmap-poiicon",
			flux: concat(seul(fixe(32), fixe(32), bit(true), fixe(3), fixe(32), fixe(32), fixe(9), fixe(9)),
				e524(0, 1, axesCarteNiveau16),
				seul(fixe(32), bit(false), fixe(8), fixe(8), fixe(8), fixe(8))),
			lire: parNom("tacmap-poiicon", 30, 0)},
		// ti=30 i1 tacmap-poiiconoffset : FUN_142ed485c (descripteur 143d06b00 + 0x28), thunk au 0x10.
		{nom: "tacmap-poiiconoffset", indexW: 1, flux: e524(-1, 1, axesDefautNiveau16),
			lire: parNom("tacmap-poiiconoffset", 30, 0)},
		// ti=34 i7 tacmap-waypointstate : FUN_140f04d88, CALL 140f04de0 ; R(1), R(32), garde f91c +
		// e524, puis R(1) quand le niveau du registre depasse 1.
		{nom: "tacmap-waypointstate niveau registre 2", indexW: 1,
			flux: concat(seul(bit(true), fixe(32)), e524(0, 1, axesCarteNiveau16), seul(bit(true))),
			lire: parNom("tacmap-waypointstate", 34, 2)},
		// ti=32 i0 tacmap-areaofinterest : FUN_142ed7764, CALL 142ed7853 (thunk).
		// EXCEPTION DATEE (lot R3, 2026-09-29) : le site garde son ancien lecteur.
		{nom: "tacmap-areaofinterest", indexW: 1, exception: "areaofinterest",
			flux: concat(seul(fixe(32), fixe(3)), e524(0, 1, axesCarteNiveau16), seul(fixe(12))),
			lire: parNom("tacmap-areaofinterest", 32, 0)},
		// ti=33 i0 tacmap-displayasset : FUN_142ed7d38, CALL 142ed7edf (thunk).
		// EXCEPTION DATEE (lot R3, 2026-09-29) : le site garde son ancien lecteur.
		{nom: "tacmap-displayasset", indexW: 1, exception: "displayasset",
			flux: concat(seul(fixe(32), fixe(32), fixe(2)), e524(-1, 1, axesDefautNiveau16),
				seul(fixe(64), fixe(32), fixe(64), fixe(32), bit(true))),
			lire: parNom("tacmap-displayasset", 33, 0)},
		// ti=34 i11 tacmap-cooptetherarea : FUN_142ed4198, CALL 142ed41ba (thunk).
		// EXCEPTION DATEE (lot R3, 2026-09-29) : le site garde son ancien lecteur.
		{nom: "tacmap-cooptetherarea", indexW: 1, exception: "cooptetherarea",
			flux: concat(e524(0, 1, axesCarteNiveau16), seul(fixe(12), fixe(12))),
			lire: parNom("tacmap-cooptetherarea", 34, 0)},
		// ti=20 i0 spawn-filter-type, etiquette 3 : FUN_142b6eeec, CALL 142b6ef31.
		{nom: "spawn-filter-type etiquette 3", indexW: 1,
			flux: concat(seul(champDeFlux{3, 2}, fixe(32)), e524(0, 1, axesCarteNiveau16),
				seul(fixe(3), champDeFlux{0, 4}, bit(true))),
			lire: parNom("spawn-filter-type-component", 20, 0)},
		// selectable-zone-data : FUN_141454340, CALL 14145437e (sans appelant Go aujourd hui).
		{nom: "selectable-zone-data", indexW: 1,
			flux: concat(seul(fixe(32)), e524(0, 1, axesCarteNiveau16), seul(bit(true))),
			lire: consumeSelectableZoneData},
	}
}

// casDesVecteursDeJoueur : l equipage, la destination de nuee et le lieu de reapparition desire.
func casDesVecteursDeJoueur() []casDeSite {
	return []casDeSite{
		// ti=14 i0 crew-order : FUN_142ed9120, CALL 142ed918e ; FUN_142b1cf3c, porte, e494 —
		// AUCUN bit precHigh.
		// EXCEPTION DATEE (lot R3, 2026-09-29) : le site garde son ancien lecteur.
		{nom: "crew-order", indexW: 1, exception: "crew-order",
			flux: concat(seul(fixe(3), bit(true)), e524(0, 1, axesCarteNiveau16)),
			lire: parNom("crew-order-component", 14, 0)},
		// ti=21 flock-destination : FUN_140fb8af0 (descripteur 143c96c50 + 0x28), CALL 140fb8b3e,
		// niveau 0x10 ; R(1), garde f91c + e524, R(2) quand le niveau du registre depasse 1.
		// EXCEPTION DATEE (lot J6-bis, 2026-09-28) : le site garde son ancien lecteur.
		{nom: "flock-destination niveau registre 2", indexW: 1, exception: "flock-destination",
			flux: concat(seul(bit(true)), e524(-1, 1, axesDefautNiveau16), seul(fixe(2))),
			lire: parNom("flock-destination-component", 21, 2)},
		// ti=5 i12 player-desired-respawn-location : FUN_142f03ec8 (descripteur 143d0f2f8 + 0x28),
		// CALL 142f03f0d ; porte, e494, FUN_14076dc04 = R(19).
		// EXCEPTION DATEE (lot J6-bis, 2026-09-28) : le site garde son ancien lecteur.
		{nom: "player-desired-respawn-location", indexW: 1, exception: "respawn-location",
			flux: concat(seul(bit(true)), e524(0, 1, axesCarteNiveau16), seul(fixe(19))),
			lire: parNom(compPlayerDesiredRespawnLoc, 5, 0)},
	}
}

// TestChaqueSiteDePositionLitCeQueLeJeuEcrit — J6.2 : un flux par site d appel.
func TestChaqueSiteDePositionLitCeQueLeJeuEcrit(t *testing.T) {
	for _, c := range casDesSitesDePosition() {
		t.Run(c.nom, func(t *testing.T) { verifierCasDeSite(t, c) })
	}
}

// verifierCasDeSite exige qu un site lise le flux du jeu au bit pres — sauf exception datee.
func verifierCasDeSite(t *testing.T, c casDeSite) {
	t.Helper()
	buf, total := ecrireFlux(c.flux)
	br := lecteurDeSite(buf, c.indexW)
	c.lire(br)
	got := br.BitPos()
	if c.exception == "" {
		if got != total {
			t.Fatalf("%s : %d bits lus, l ecrivain du jeu en pose %d", c.nom, got, total)
		}
		return
	}
	// ECART ATTENDU : le site est une exception datee du portage. Il ne doit PAS lire le
	// flux du jeu tant qu elle tient ; et une cle d exception retiree rend le cas strict.
	if _, tient := exceptionsDuPortage()[c.exception]; !tient {
		if got != total {
			t.Fatalf("%s : exception %q retiree mais le site ne lit pas comme le jeu (%d bits lus, %d "+
				"poses) — migrer le site vers le portage", c.nom, c.exception, got, total)
		}
		return
	}
	if got == total {
		t.Fatalf("%s : le site lit desormais comme le jeu (%d bits) alors que l exception %q tient — "+
			"retirer l exception de `exceptionsDuPortage`", c.nom, got, c.exception)
	}
}

// casDuCheminI0DuBipede : les trois lecteurs de FUN_1406cfe44 qui descendent dans e524 ou dans
// son enveloppe a precHigh. Chaque cas lit le SOUS-LECTEUR seul : le R(2) de `LAB_1406cffd7`
// appartient a `FUN_1406cfe44`, pas au site d appel.
func casDuCheminI0DuBipede() []casDeSite {
	pd := profile.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{6, 6, 6}}
	return []casDeSite{
		// Branche absolue de FUN_1406cfe44 (CALL 1406d009d, niveau 0x10 en 1406d008a) : precHigh = 1
		// -> FUN_141f85880, trois axes de 14 bits sur +/-100, puis le R(2) de LAB_1406cffd7.
		// EXCEPTION DATEE (lot R3, 2026-09-29) : cette branche garde son ancien lecteur (0 bit).
		{nom: "i0 absolu precHigh=1", indexW: 1, exception: "i0-bipede-prechigh",
			flux: concat(seul(bit(true)), axesSeuls(axesPrecHautNiveau16), seul(fixe(2))),
			lire: consumeAbsoluteWithGate},
		// FUN_140f7ea14 -> FUN_14076e4ec(0x10, cVar1 ? &DAT_143b8c6d0 : 0), CALL 140f7ea5c.
		{nom: "i0 absolu predit cVar1=1", indexW: 1,
			flux: concat(seul(bit(true)), axesSeuls(axesPrecHautNiveau16)), lire: consumePredictedAbsolute},
		{nom: "i0 absolu predit cVar1=0", indexW: 1,
			flux: concat(seul(bit(false)), e524(0, 1, axesCarteNiveau16)), lire: consumePredictedAbsolute},
		// Repli absolu de FUN_14076f3ec (CALL 14226a6c7, niveau 0x10) : bit a 1 -> e524 NU, sans
		// bit precHigh ; le R(2) de LAB_1406cffd7 le suit.
		{nom: "i0 repli absolu du delta predit", indexW: 1,
			flux: concat(seul(bit(true)), e524(0, 1, axesCarteNiveau16), seul(fixe(2))),
			lire: func(br *Lecteur) { consumePredictedDelta(br, pd) }},
	}
}

// TestLeCheminI0DuBipedeLitCeQueLeJeuEcrit — J6.2 : les sous-lecteurs d i0 qui portent e524.
func TestLeCheminI0DuBipedeLitCeQueLeJeuEcrit(t *testing.T) {
	for _, c := range casDuCheminI0DuBipede() {
		t.Run(c.nom, func(t *testing.T) { verifierCasDeSite(t, c) })
	}
}

// TestLaGrammaireDeLEcrivainI0LitLaTableAPrecHaut — la grammaire d ecrivain d i0
// (`GrammaireEcrivainI0`, FUN_14076e29c) : h = precHigh = 1 lit FUN_141f85880 (3 x 14), pas la
// porte, l index et les axes de la carte ; puis la queue de poignee (h = 1 : deux portes a 0) et
// le R(2).
func TestLaGrammaireDeLEcrivainI0LitLaTableAPrecHaut(t *testing.T) {
	flux := concat(seul(bit(false), bit(false)), seul(bit(true)), axesSeuls(axesPrecHautNiveau16),
		seul(bit(false), bit(false)), seul(fixe(2)))
	buf, total := ecrireFlux(flux)
	br := lecteurDeSite(buf, 2) // quatre plages : sous Cliffhanger, 1 + 1 + 40 = 42 = 3 x 14 masquerait l ecart
	p := br.Profil()
	p.Grammaire.GrammaireEcrivainI0 = true
	br.PoserProfil(p)
	consumeObjectPositionDynamicPrecisionD(br, br.traversal())
	if got := br.BitPos(); got != total {
		t.Fatalf("i0, grammaire de l ecrivain, h = 1 : %d bits lus, l ecrivain en pose %d", got, total)
	}
}

// TestLaTeleportationDuTranslocateurLitSousLaGarde — le second portage : `FUN_140f04fb8` (CALLs
// 140f04ff0 et 140f05023, niveau 0x10 aux deux) garde CHAQUE position par `FUN_14076f91c` ; sous la
// garde, `FUN_1411b259c` lit R(96) brut. `transloc_events.go` recopiait le lecteur sans la garde.
// Hors portee (la production), les deux lectures sont identiques : l oracle 18/18 n en depend pas.
func TestLaTeleportationDuTranslocateurLitSousLaGarde(t *testing.T) {
	entree := &profile.MapQuantEntry{Min: [3]float32{-100, -100, -10}, Max: [3]float32{100, 100, 30},
		AxisWidths: axesCarteNiveau16}
	flux := seul(fixe(96))
	buf, total := ecrireFlux(flux)
	br := LecteurSur(buf)
	p := br.Profil()
	p.Grammaire.PorteeBaseline = true
	br.PoserProfil(p)
	if _, ok := readTranslocVec(br, entree); ok {
		t.Error("sous la garde, la position est BRUTE : aucune coordonnee quantifiee a rendre")
	}
	if got := br.BitPos(); got != total {
		t.Fatalf("translocateur sous la garde, position A : %d bits lus, l ecrivain en pose %d", got, total)
	}
}

// TestLEtatParDefautTI13LitLaChargeDuVariant — GA2-4 : `FUN_140ce55e8` appelle `FUN_140ce59bc`,
// qui lit l etiquette R(4) PUIS TOUJOURS la charge de `FUN_140ce5aa4` (mode A pour g = 0, index
// -1 ; mode B pour g = 1, index 0..31).
func TestLEtatParDefautTI13LitLaChargeDuVariant(t *testing.T) {
	cas := []struct {
		nom  string
		flux []champDeFlux
	}{
		// g = 0, mode A, etiquette 3 : R(24) quantifie.
		{"g=0 mode A etiquette 3", seul(bit(false), fixe(32), bit(false), champDeFlux{3, 4}, fixe(24))},
		// g = 0, mode A, etiquette 8 : muette en mode A.
		{"g=0 mode A etiquette 8", seul(bit(false), fixe(32), bit(false), champDeFlux{8, 4})},
	}
	// g = 1, mode B : 32 variants ; le premier a l etiquette 9 (R(32)), les autres a 0 (muets).
	b := seul(bit(false), fixe(32), bit(true), champDeFlux{9, 4}, fixe(32))
	for i := 1; i < 32; i++ {
		b = append(b, champDeFlux{0, 4})
	}
	cas = append(cas, struct {
		nom  string
		flux []champDeFlux
	}{"g=1 mode B etiquette 9 puis 31 muets", b})
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := LecteurSur(buf)
			consumeDefaultStateTI13(br)
			if got := br.BitPos(); got != total {
				t.Fatalf("etat par defaut ti=13 : %d bits lus, l ecrivain en pose %d", got, total)
			}
		})
	}
}
