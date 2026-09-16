package grammar

// player_table_test.go — LA TABLE DES JOUEURS SE LIT SUR LES SEPT BUILDS (lot 1.5.4).
//
// CE QUE CES TESTS GARDENT, ET POURQUOI CHACUN MORD :
//
//	T-32     Sur chacune des sept bobines par build, le lecteur rend EXACTEMENT 32 slots,
//	         occupes plus vacants — la borne `0x28A00 / 0x1450` de l'ecrivain, verifiee par la
//	         lecture. Une seule largeur fausse la casse : c'est ce que T-MUT prouve.
//	T-NOM    Tout slot occupe rend un gamertag IMPRIMABLE et un XUID de la plage Xbox.
//	T-RANG   Le rang publie est l'index du slot dans la table, sans trou ni repetition.
//	T-CAL    Le calibrage LU SUR LE FILM tombe sur la valeur du PROFIL, et aucun ecart n'est en
//	         contradiction. C'est un CONTROLE : aucune lecture n'en depend (D-3, ADR 0034).
//	T-MUT    Fausser la largeur du bloc de personnalisation — de 4 octets, ou en prenant celle
//	         d'un autre build — empeche la table de fermer. Sans cela, « 32 slots » ne prouverait
//	         pas la largeur.
//	T-BUILD  Un build absent du profil rend [ErrUnknownBuild] et NOMME son compteur. Jamais une
//	         lecture au profil du build le plus proche (D-4).
//	T-COUPE  Un `chunk_00` coupe dans la table rend une erreur typee, sans panique.

import (
	"errors"
	"strings"
	"testing"
)

// bobineTable : ce que la lecture doit rendre sur chaque bobine par build.
//
// Les comptes sont ceux MESURES le 2026-09-14 par l'instrument de recherche `rsChaine`, avant
// que ce lecteur existe : c'est l'ORACLE, et le lecteur de production doit le reproduire a
// l'identique. Les largeurs viennent du profil (player_table_profile.go).
type bobineTable struct {
	film              string
	occupes, vacants  int
	persoOctets       int
	transpositionBits int
}

func bobinesTable() []bobineTable {
	return []bobineTable{
		{"a521164d", 24, 8, 2052, +1600}, // HI_1_4_1
		{"60ae07c4", 8, 24, 1312, -4320}, // HI_1_8_0
		{"11de8353", 24, 8, 1312, -4320}, // HI_1_9_0
		{"111fa685", 24, 8, 1492, -2880}, // HI_1_10_0
		{"e5adf7b2", 23, 9, 1492, -2880}, // HI_1_11_0
		{"bcb6d393", 8, 24, 1852, 0},     // HI_1_12_0
		{"fb1a1a72", 8, 24, 1852, 0},     // HI_1_13_0
	}
}

// TestReadPlayerTableSurLesBobines execute T-32, T-NOM, T-RANG et T-CAL.
func TestReadPlayerTableSurLesBobines(t *testing.T) {
	for _, b := range bobinesTable() {
		d := bobineChunk00(t, b.film)
		id, err := ReadFilmIdentity(d)
		if err != nil {
			t.Fatalf("%s : identite illisible : %v", b.film, err)
		}
		slots, rep, err := ReadPlayerTable(d, id)
		if err != nil {
			t.Fatalf("%s (%s) : %v", b.film, id.Build, err)
		}
		verifierTable(t, b, id, slots, rep)
		t.Logf("%-10s %-10q perso %4d o (%+6d bits) | %2d occupes + %2d vacants = %d | "+
			"calibrage film %+6d sur %2d ecarts, accord %v | ecarts %d/%d/%d/%d "+
			"(accord/vacant/invisible/contradiction) | candidats %d dont %d reels | "+
			"tete %d, intercale %v",
			b.film, rep.Build, rep.PersoBytes, rep.ProfileDeltaBits, rep.Occupied, rep.Vacant,
			rep.Occupied+rep.Vacant, rep.FilmDeltaBits, rep.FilmDeltaGaps,
			rep.CalibrationAgrees, rep.GapsAgree, rep.GapsVacant, rep.GapsHidden,
			rep.GapsContradict,
			rep.CandidatesScanned, rep.CandidatesReal, rep.HeadVacant, rep.InterleavedVacant)
	}
}

// verifierTable confronte une lecture a l'oracle des instruments et aux invariants.
func verifierTable(t *testing.T, b bobineTable, id FilmIdentity, slots []PlayerSlot,
	rep PlayerTableReport) {
	t.Helper()
	if rep.Occupied != b.occupes || rep.Vacant != b.vacants {
		t.Errorf("%s : %d occupes + %d vacants, oracle %d + %d",
			b.film, rep.Occupied, rep.Vacant, b.occupes, b.vacants)
	}
	if n := rep.Occupied + rep.Vacant; n != playerTableSlots { // T-32
		t.Errorf("%s : %d slots lus, la borne de l'ecrivain en impose %d", b.film, n,
			playerTableSlots)
	}
	if rep.PersoBytes != b.persoOctets || rep.ProfileDeltaBits != b.transpositionBits {
		t.Errorf("%s (%s) : profil %d o / %+d bits, attendu %d o / %+d bits", b.film, id.Build,
			rep.PersoBytes, rep.ProfileDeltaBits, b.persoOctets, b.transpositionBits)
	}
	if !rep.CalibrationAgrees || rep.FilmDeltaBits != rep.ProfileDeltaBits { // T-CAL
		t.Errorf("%s : le calibrage lu sur le film vaut %+d, le profil dit %+d (accord %v)",
			b.film, rep.FilmDeltaBits, rep.ProfileDeltaBits, rep.CalibrationAgrees)
	}
	if rep.GapsContradict != 0 {
		t.Errorf("%s : %d ecart(s) en CONTRADICTION avec la grammaire", b.film,
			rep.GapsContradict)
	}
	verifierSlots(t, b.film, slots, rep)
}

// verifierSlots execute T-NOM et T-RANG.
func verifierSlots(t *testing.T, film string, slots []PlayerSlot, rep PlayerTableReport) {
	t.Helper()
	vus := map[int]bool{}
	for i, s := range slots {
		if !gamertagImprimable(s.Gamertag) { // T-NOM
			t.Errorf("%s slot %d : gamertag %q non imprimable", film, s.FilmIndex, s.Gamertag)
		}
		if s.XUID <= slotXuidLo || s.XUID >= slotXuidHi {
			t.Errorf("%s slot %d : XUID %d hors de la plage Xbox", film, s.FilmIndex, s.XUID)
		}
		if s.SessionToken == 0 {
			t.Errorf("%s slot %d : jeton de session nul", film, s.FilmIndex)
		}
		// Le champ de 6 bits distingue un slot OCCUPE (-1) d'un slot VACANT (0) : c'est le
		// seul champ qui les separe, et c'est lui qui rend le predicat de vacance decidable.
		if s.Shorts.F6 != -1 {
			t.Errorf("%s slot %d : champ de 6 bits %d, attendu -1 sur un slot occupe",
				film, s.FilmIndex, s.Shorts.F6)
		}
		if s.FilmIndex < 0 || s.FilmIndex >= playerTableSlots || vus[s.FilmIndex] { // T-RANG
			t.Errorf("%s : rang %d invalide ou repete", film, s.FilmIndex)
		}
		vus[s.FilmIndex] = true
		if i > 0 && s.FilmIndex <= slots[i-1].FilmIndex {
			t.Errorf("%s : rang %d apres %d, l'ordre doit croitre", film, s.FilmIndex,
				slots[i-1].FilmIndex)
		}
	}
	// Sans vacant de tete ni vacant intercale, le rang EST l'index dans la tranche rendue.
	if rep.HeadVacant == 0 && !rep.InterleavedVacant {
		for i, s := range slots {
			if s.FilmIndex != i {
				t.Errorf("%s : rang %d a l'index %d alors qu'aucun vacant ne precede",
					film, s.FilmIndex, i)
			}
		}
	}
}

// TestReadPlayerTableLargeurFausseDegradeLaLecture execute T-MUT : fausser la largeur du bloc de
// personnalisation fait perdre des enregistrements, toujours.
//
// C'est ce test qui donne son sens a T-32. Un lecteur qui rendrait le meme roster quelle que soit
// la largeur ne prouverait rien du profil.
//
// CE QUE LE TEST N'AFFIRME PAS, ET POURQUOI : « avec une largeur fausse, RIEN ne ferme » serait
// faux, et le mesurer l'a montre. Une largeur trop GRANDE fait sauter la marche par-dessus le
// reste de la table et atterrir dans le bourrage de queue, ou le predicat de vacance passe
// indefiniment : « 1 occupe + 31 vacants = 32 » ferme alors, et c'est une lecture RECEVABLE au
// sens de la grammaire — simplement la plus pauvre. C'est pour cela que `chercherDepart` retient
// la plus COMPLETE, et c'est cela que ce test verifie : la bonne largeur explique STRICTEMENT
// plus d'enregistrements que n'importe quelle autre.
func TestReadPlayerTableLargeurFausseDegradeLaLecture(t *testing.T) {
	// Les quatre largeurs du profil, plus deux voisines : un decalage de 4 octets suffit-il ?
	fausses := []int{1312, 1492, 1852, 2052}
	for _, b := range bobinesTable() {
		d := bobineChunk00(t, b.film)
		id, err := ReadFilmIdentity(d)
		if err != nil {
			t.Fatalf("%s : %v", b.film, err)
		}
		candidats := balayerCandidats(d, id.BodyBit, (dernierOctetNonNul(d)+1)*8)
		finBit := len(d) * 8
		essais := append([]int(nil), fausses...)
		essais = append(essais, b.persoOctets-4, b.persoOctets+4)
		for _, faux := range essais {
			if faux == b.persoOctets {
				continue
			}
			dt, ok := chercherDepart(d, candidats, finBit, faux*8)
			if ok && len(dt.slots) >= b.occupes {
				t.Errorf("%s (%s) : largeur fausse de %d octets, %d enregistrement(s) lus — "+
					"le profil dit %d octets et en lit %d, T-MUT ne mord pas", b.film, id.Build,
					faux, len(dt.slots), b.persoOctets, b.occupes)
			}
		}
	}
}

// TestReadPlayerTableBuildInconnu execute T-BUILD : un build hors profil est refuse, jamais lu
// au profil du voisin (D-4, ADR 0034).
func TestReadPlayerTableBuildInconnu(t *testing.T) {
	d := bobineChunk00(t, "fb1a1a72")
	id, err := ReadFilmIdentity(d)
	if err != nil {
		t.Fatalf("temoin positif : %v", err)
	}
	for _, build := range []string{"", "HI_1_14_0", "HI_1_4_2", "autre chose"} {
		faux := id
		faux.Build = build
		slots, rep, err := ReadPlayerTable(d, faux)
		if !errors.Is(err, ErrUnknownBuild) {
			t.Errorf("build %q : erreur %v, attendue %v", build, err, ErrUnknownBuild)
		}
		if slots != nil || rep.Occupied != 0 {
			t.Errorf("build %q : %d slot(s) rendus malgre le refus", build, len(slots))
		}
		if !strings.Contains(err.Error(), build) && build != "" {
			t.Errorf("build %q : l'erreur ne nomme pas le build (%v)", build, err)
		}
	}
}

// TestUnknownBuildCompteur : le compteur de build inconnu porte un nom d'expvar valide (ADR
// 0009, snake_case, prefixe de categorie) quel que soit ce que le film raconte.
func TestUnknownBuildCompteur(t *testing.T) {
	cas := map[string]string{
		"HI_1_14_0":      "filmdec_unknown_build_hi_1_14_0",
		"":               "filmdec_unknown_build_sans_section",
		"a b/c":          "filmdec_unknown_build_a_b_c",
		"HI_1_13_0.BÊTA": "filmdec_unknown_build_hi_1_13_0_b_ta",
	}
	for build, veut := range cas {
		paires := UnknownBuildExpvarPairs(build)
		if len(paires) != 1 || paires[0].Value != 1 {
			t.Fatalf("build %q : %d paire(s), attendu une paire a 1", build, len(paires))
		}
		if paires[0].Name != veut {
			t.Errorf("build %q : compteur %q, attendu %q", build, paires[0].Name, veut)
		}
		for _, r := range paires[0].Name {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
				t.Errorf("build %q : le nom %q porte %q, hors du jeu snake_case d'ADR 0009",
					build, paires[0].Name, r)
			}
		}
	}
}

// TestPersonnalisationOctetsProfil : la table de profil rend les quatre largeurs mesurees, et
// leurs transpositions ferment sur les constantes que la recherche a relevees par build.
func TestPersonnalisationOctetsProfil(t *testing.T) {
	attendu := map[string][2]int{ // build -> {octets, transposition en bits}
		"HI_1_13_0": {1852, 0},
		"HI_1_12_0": {1852, 0},
		"HI_1_11_0": {1492, -2880},
		"HI_1_10_0": {1492, -2880},
		"HI_1_9_0":  {1312, -4320},
		"HI_1_8_0":  {1312, -4320},
		"HI_1_4_1":  {2052, +1600},
	}
	for build, veut := range attendu {
		octets, ok := personnalisationOctets(build)
		if !ok {
			t.Errorf("%s absent du profil", build)
			continue
		}
		if octets != veut[0] || persoDeltaBits(octets) != veut[1] {
			t.Errorf("%s : %d octets (%+d bits), attendu %d (%+d bits)", build, octets,
				persoDeltaBits(octets), veut[0], veut[1])
		}
	}
	if _, ok := personnalisationOctets("HI_1_14_0"); ok {
		t.Error("un build jamais mesure ne doit PAS avoir de largeur de profil (D-4)")
	}
}

// TestReadPlayerTableTronquee execute T-COUPE : une coupe dans la table rend une erreur typee.
func TestReadPlayerTableTronquee(t *testing.T) {
	d := bobineChunk00(t, "fb1a1a72")
	id, err := ReadFilmIdentity(d)
	if err != nil {
		t.Fatalf("temoin positif : %v", err)
	}
	slots, _, err := ReadPlayerTable(d, id)
	if err != nil || len(slots) == 0 {
		t.Fatalf("temoin positif : %d slot(s), %v", len(slots), err)
	}
	// Coupe APRES le premier enregistrement : la marche ne peut plus fermer a 32.
	coupe := append([]byte(nil), d[:(slots[0].Bit+slots[0].TotalBits)/8+1]...)
	if _, _, err := ReadPlayerTable(coupe, id); !errors.Is(err, ErrPlayerTableNotFound) &&
		!errors.Is(err, ErrChunk00Truncated) {
		t.Errorf("coupe dans la table : erreur %v, attendue %v ou %v", err,
			ErrPlayerTableNotFound, ErrChunk00Truncated)
	}
	// Corps hors du tampon : la borne est refusee avant tout balayage.
	if _, _, err := ReadPlayerTable(d[:id.BodyBit/8], id); !errors.Is(err, ErrChunk00Truncated) {
		t.Errorf("corps hors tampon : erreur %v, attendue %v", err, ErrChunk00Truncated)
	}
}
