package filmdec

// profile_globales_test.go — LA DOUBLE ECRITURE, PROUVEE (item 2.1.3 du PLAN_DECODEUR_FILM).
//
// Le lot 2.1 resout le profil mais ne le fait lire par AUCUN lecteur de bits : les globales de
// paquet decident encore. Ce qui rend la bascule du lot 2.2 possible sans risque, c est la
// PREUVE que les deux disent la meme chose, bobine par bobine. Ces tests sont cette preuve.
//
// LE JOUR OU L UN D EUX ROUGIT, la question n est pas « quel test reparer » : c est que le
// profil et le decodeur ont diverge, et l un des deux a tort.
//
// LE VOLET WORLD-OBJECT est dans `replay` (`TestProfilEgaleGlobalesWorldObject`), parce que
// `installWorldObjectPrecision` y vit : c est le meme test, de l autre cote de la frontiere.

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmsource"
)

// TestProfilEgaleGlobales : sur chaque bobine par build, le profil resolu et les globales que la
// production installe rendent les MEMES valeurs.
func TestProfilEgaleGlobales(t *testing.T) {
	verifierMouvementEgaleGlobales(t)
	verifierCadreEgaleConstantes(t)
	for _, b := range bobinesIdentite() {
		film := bobineFilm(t, b.film)
		p := ResolveProfile(film, nil)
		verifierMPPEgaleGlobales(t, b, film, p)
		verifierSlotsEgaleRapport(t, b, p)
		if maj, _ := FilmMajorVersion(film); maj != p.Highlight().MajorVersion {
			t.Errorf("%s : version majeure du profil %d, lecture de production %d", b.film,
				p.Highlight().MajorVersion, maj)
		}
	}
}

// verifierMPPEgaleGlobales confronte `Profile.MPP` a ce que [InstallFilmFormatMPP] pose.
//
// DEUX CAS, ET ILS NE SE CONFONDENT PAS : une largeur POSEE doit se retrouver a l identique dans
// les globales ; une largeur INDETERMINEE (formats 20, 21, 24, 25) doit laisser les globales
// EXACTEMENT ou elles etaient — c est ce que la calibration attend pour decider a sa place.
func verifierMPPEgaleGlobales(t *testing.T, b bobineIdentite, film *filmsource.Film, p Profile) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	avant := CurrentMPPWidths()
	restore, err := InstallFilmFormatMPP(film)
	if err != nil {
		t.Fatalf("%s (%s) : format refuse par la production alors que le profil le connait : %v",
			b.film, b.build, err)
	}
	apres := CurrentMPPWidths()
	restore()
	switch {
	case p.MPP().Valid() && apres != p.MPP():
		t.Errorf("%s : MPP du profil %v, MPP installee %v", b.film, p.MPP(), apres)
	case !p.MPP().Valid() && apres != avant:
		t.Errorf("%s : largeur MPP indeterminee au profil, mais la production a installe %v "+
			"(avant %v) — la calibration ne deciderait plus", b.film, apres, avant)
	}
}

// verifierSlotsEgaleRapport confronte `Profile.Slots` au rapport de [ReadPlayerTable], qui porte
// aujourd hui la seule lecture de production de la largeur de personnalisation.
func verifierSlotsEgaleRapport(t *testing.T, b bobineIdentite, p Profile) {
	t.Helper()
	chunk0 := bobineChunk00(t, b.film)
	id, err := ReadFilmIdentity(chunk0)
	if err != nil {
		t.Fatalf("%s : identite illisible : %v", b.film, err)
	}
	_, rep, err := ReadPlayerTable(chunk0, id)
	if err != nil {
		t.Fatalf("%s : table des joueurs illisible : %v", b.film, err)
	}
	if rep.PersoBytes != p.Slots().PersoBytes || rep.ProfileDeltaBits != p.Slots().DeltaBits {
		t.Errorf("%s : slots du profil {%d o, %+d bits}, rapport de production {%d o, %+d bits}",
			b.film, p.Slots().PersoBytes, p.Slots().DeltaBits, rep.PersoBytes, rep.ProfileDeltaBits)
	}
}

// verifierCadreEgaleConstantes confronte `Profile.Keyframe` aux constantes que
// `walkKeyframeFullState` lit. La regle est `172 + etat(ti)`, jamais un nombre : les 172 sont
// verifies ici, et `EtatParDefautPorte` est confronte a la table des deserialiseurs.
func verifierCadreEgaleConstantes(t *testing.T) {
	t.Helper()
	k := ResolveProfile(nil, nil).Keyframe()
	if k.EnTeteBits != keyframeFullStateHeaderBits || k.MotDeTailleBits != keyframeFullStateSizeBits {
		t.Errorf("cadre du profil {%d, %d}, constantes de la marche {%d, %d}", k.EnTeteBits,
			k.MotDeTailleBits, keyframeFullStateHeaderBits, keyframeFullStateSizeBits)
	}
	if got := k.CadreBits(); got != 172 {
		t.Errorf("cadre de %d bits, attendu 172 (108 + 2 x 32)", got)
	}
	for ti := uint32(0); ti < objectArchetypeCount; ti++ {
		_, table := defaultStateDeserByTI[ti]
		attendu := table || ti == BipedTypeIndex
		if k.EtatParDefautPorte(ti) != attendu {
			t.Errorf("ti=%d : EtatParDefautPorte=%v, table des deserialiseurs %v", ti,
				k.EtatParDefautPorte(ti), attendu)
		}
	}
}

// verifierMouvementEgaleGlobales confronte `Profile.Movement` aux variables de paquet que les
// lecteurs de position lisent aujourd hui.
//
// IL LIT LES GLOBALES TELLES QU ELLES SONT, et c est voulu : un test du paquet qui les laisserait
// sales est lui-meme un defaut, et ce test est l endroit ou il se voit.
//
// IL A MAIGRI DE CINQ LIGNES AU LOT 2.2.a : le descripteur de traversee, la largeur d axe
// absolue et les trois drapeaux de position ne sont plus des variables de paquet — les lecteurs
// les prennent au profil que porte le lecteur de bits, ce que prouve
// [TestLecteurPorteLeProfilDeMouvement]. Ne restent ici que les valeurs des familles 2.2.b et
// 2.2.e, encore en globales.
func verifierMouvementEgaleGlobales(t *testing.T) {
	t.Helper()
	m := ResolveProfile(nil, nil).Movement()
	ecarts := []struct {
		nom             string
		profil, globale any
	}{
		{"DeltaQuantum", m.DeltaQuantum, DeltaQuantum},
		{"DeltaAxisWidth", m.DeltaAxisWidth, DeltaAxisWidth},
		{"Range", m.Range, WorldPositionRange},
		{"MobilityActionExtraBits", m.MobilityActionExtraBits, MobilityActionExtraBits},
	}
	for _, e := range ecarts {
		if e.profil != e.globale {
			t.Errorf("Movement.%s : profil %v, globale de paquet %v — le lot 2.2 basculerait "+
				"les lecteurs sur une valeur differente de celle qu ils lisent", e.nom,
				e.profil, e.globale)
		}
	}
}

// TestLecteurPorteLeProfilDeMouvement — LA BASCULE DU LOT 2.2.a, PROUVEE.
//
// Trois affirmations, et chacune ferme un chemin par lequel un lecteur pourrait se retrouver
// avec une autre valeur que celle du profil :
//
//	AU REPOS      un lecteur neuf porte EXACTEMENT [mouvementDuProfil] — donc l heritage de
//	              processus (`mouvement_herite.go`), au repos, vaut l invariant du profil.
//	PAR LE CADRE  `DefaultFrameConfig().Mouvement` dit la meme chose, pour les portes de
//	              balayage qui reconstruisent un cadre.
//	EN TETE       une valeur posee en tete de balayage arrive jusqu aux accesseurs que les
//	              deserialiseurs appellent — et n en modifie aucun autre lecteur.
func TestLecteurPorteLeProfilDeMouvement(t *testing.T) {
	release := LockProcessDecode()
	defer release()
	invariant := ResolveProfile(nil, nil).Movement()
	if got := NewBitReader(nil).mv; got != invariant {
		t.Errorf("lecteur neuf : mouvement %+v, profil %+v", got, invariant)
	}
	if got := DefaultFrameConfig().Mouvement; got != invariant {
		t.Errorf("cadre par defaut : mouvement %+v, profil %+v", got, invariant)
	}
	br, temoin := NewBitReader(nil), NewBitReader(nil)
	pose := invariant
	pose.Traversal = PrecisionDescriptor{IndexW: 3, AxisW: [3]uint{11, 12, 13}}
	pose.AbsoluteAxisW, pose.FullPrecision = 19, true
	pose.DeltaHasHandleTail, pose.CalibratedSkip = true, true
	br.poserMouvement(pose)
	switch {
	case br.traversal() != pose.Traversal:
		t.Errorf("traversal() rend %+v, pose %+v", br.traversal(), pose.Traversal)
	case br.absoluteAxisW() != pose.AbsoluteAxisW:
		t.Errorf("absoluteAxisW() rend %d, pose %d", br.absoluteAxisW(), pose.AbsoluteAxisW)
	case !br.fullPrecision() || !br.deltaHasHandleTail() || !br.calibratedSkip():
		t.Errorf("les trois drapeaux poses ne sont pas rendus : %v/%v/%v", br.fullPrecision(),
			br.deltaHasHandleTail(), br.calibratedSkip())
	case temoin.mv != invariant:
		t.Errorf("poser le profil sur un lecteur a change un AUTRE lecteur : %+v", temoin.mv)
	}
}

// TestProfilHighlightEgaleLeParseur : l implantation que le profil NOMME est celle que
// `analysis.ParseHighlightEvents` APPLIQUE.
//
// LA BRANCHE VIT DANS `analysis`, LE NOM DANS LE PROFIL : deux endroits, donc une divergence
// possible. Ce test la ferme sans dedoubler la grammaire — il plante un gamertag a l offset que
// le profil annonce, et verifie que le parseur le retrouve.
func TestProfilHighlightEgaleLeParseur(t *testing.T) {
	for _, majeure := range []int{0, 37, 38, 39, 40, 41, 42} {
		h := highlightDuProfil(majeure, majeure != 0)
		const tag = "TemoinDuProfil"
		chunk := chunkTempsFortTemoin(tag, h.GamertagOffsetBytes)
		evs, err := analysis.ParseHighlightEvents(chunk, majeure)
		if err != nil {
			t.Fatalf("majeure %d : %v", majeure, err)
		}
		if len(evs) != 1 {
			t.Fatalf("majeure %d : %d evenements lus, attendu 1 — le temoin synthetique ne "+
				"porte qu un bloc", majeure, len(evs))
		}
		if evs[0].Gamertag != tag {
			t.Errorf("majeure %d : le profil annonce l implantation %q a l octet %d, le parseur "+
				"rend %q au lieu de %q", majeure, h.Implantation, h.GamertagOffsetBytes,
				evs[0].Gamertag, tag)
		}
	}
}

// chunkTempsFortTemoin fabrique UN bloc d evenement de temps fort, avec le gamertag plante a
// `offset` octets du debut du bloc de 60.
//
// La forme est celle que `analysis.scanEvents` cherche : du remplissage, le XUID en u64
// little-endian, l octet `0x2d`, l octet `0xc0`, les 60 octets d evenement, puis le marqueur de
// fin `00 00 2e e0`. Le decoupage exact est celui de l en-tete de `highlight_event_parser.go`.
func chunkTempsFortTemoin(tag string, offset int) []byte {
	const (
		octetsEvenement = 60
		typeHintMort    = 20 // ni medaille ni mode : le parseur rend un "death"
	)
	ev := make([]byte, octetsEvenement)
	for i, u := range utf16.Encode([]rune(tag)) {
		binary.LittleEndian.PutUint16(ev[offset+2*i:], u)
	}
	ev[47] = typeHintMort
	binary.BigEndian.PutUint32(ev[48:52], 1234)
	out := make([]byte, 0, 2+8+2+octetsEvenement+4)
	out = append(out, 0x11, 0x11) // remplissage : le scanner exige 8 bits avant le XUID
	out = binary.LittleEndian.AppendUint64(out, 2_500_000_000_000_000)
	out = append(out, 0x2d, 0xc0)
	out = append(out, ev...)
	return append(out, 0x00, 0x00, 0x2e, 0xe0)
}
