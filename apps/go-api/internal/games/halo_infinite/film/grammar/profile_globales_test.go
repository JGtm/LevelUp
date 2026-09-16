package grammar

// profile_globales_test.go — LA DOUBLE ECRITURE, PROUVEE (item 2.1.3 du PLAN_DECODEUR_FILM).
//
// LE VOLET MOUVEMENT A DISPARU AU LOT 2.2.e, ET C EST SA REUSSITE : il confrontait le profil aux
// NEUF variables de paquet du chemin de position, une ligne chacune. Les neuf ont ete migrees
// (2.2.a en a pris cinq, 2.2.b quatre) ; il ne restait plus rien a confronter. Ce que les
// lecteurs lisent vraiment est desormais prouve par [TestLecteurPorteLeProfilDeMouvement],
// [TestProfilDePositionChangeLaConsommationDeBits] et
// [TestProfilDeQuantificationChangeLaValeurRendue] — trois tests qui mesurent la LECTURE, la ou
// celui-ci ne comparait que deux valeurs au repos.
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

	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// TestProfilEgaleGlobales : sur chaque bobine par build, le profil resolu et les globales que la
// production installe rendent les MEMES valeurs.
func TestProfilEgaleGlobales(t *testing.T) {
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

// verifierMPPEgaleGlobales confronte `profile.Profile.MPP` a ce que [InstallFilmFormatMPP] pose.
//
// DEUX CAS, ET ILS NE SE CONFONDENT PAS : une largeur POSEE doit se retrouver a l identique dans
// le profil du contexte ; une largeur INDETERMINEE (formats 20, 21, 24, 25) doit le laisser
// EXACTEMENT ou il etait — c est ce que la calibration attend pour decider a sa place.
func verifierMPPEgaleGlobales(t *testing.T, b bobineIdentite, film *source.Film, p profile.Profile) {
	t.Helper()
	fc := NewFilmContext(film)
	avant := fc.ProfilDeBalayage().MPP
	restore, err := InstallFilmFormatMPP(fc)
	if err != nil {
		t.Fatalf("%s (%s) : format refuse par la production alors que le profil le connait : %v",
			b.film, b.build, err)
	}
	apres := fc.ProfilDeBalayage().MPP
	restore()
	switch {
	case p.MPP().Valid() && apres != p.MPP():
		t.Errorf("%s : MPP du profil %v, MPP installee %v", b.film, p.MPP(), apres)
	case !p.MPP().Valid() && apres != avant:
		t.Errorf("%s : largeur MPP indeterminee au profil, mais la production a installe %v "+
			"(avant %v) — la calibration ne deciderait plus", b.film, apres, avant)
	}
}

// verifierSlotsEgaleRapport confronte `profile.Profile.Slots` au rapport de [ReadPlayerTable], qui porte
// aujourd hui la seule lecture de production de la largeur de personnalisation.
func verifierSlotsEgaleRapport(t *testing.T, b bobineIdentite, p profile.Profile) {
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

// verifierCadreEgaleConstantes confronte `profile.Profile.Keyframe` aux constantes que
// `walkKeyframeFullState` lit. La regle est `172 + etat(ti)`, jamais un nombre : les 172 sont
// verifies ici, et `EtatParDefautPorte` est confronte a la table des deserialiseurs.
func verifierCadreEgaleConstantes(t *testing.T) {
	t.Helper()
	k := ResolveProfile(nil, nil).Keyframe()
	if k.EnTeteBits != profile.KeyframeEnTeteBits || k.MotDeTailleBits != profile.KeyframeMotDeTailleBits {
		t.Errorf("cadre du profil {%d, %d}, constantes de la marche {%d, %d}", k.EnTeteBits,
			k.MotDeTailleBits, profile.KeyframeEnTeteBits, profile.KeyframeMotDeTailleBits)
	}
	if got := k.CadreBits(); got != 172 {
		t.Errorf("cadre de %d bits, attendu 172 (108 + 2 x 32)", got)
	}
	for ti := uint32(0); ti < objectArchetypeCount; ti++ {
		_, table := defaultStateDeserByTI[ti]
		attendu := table || ti == BipedTypeIndex
		if EtatParDefautPorte(ti) != attendu {
			t.Errorf("ti=%d : EtatParDefautPorte=%v, table des deserialiseurs %v", ti,
				EtatParDefautPorte(ti), attendu)
		}
	}
}

// TestProfilHighlightEgaleLeParseur : l implantation que le profil NOMME est celle que
// `grammar.ParseHighlightEvents` APPLIQUE.
//
// LA BRANCHE VIT DANS `analysis`, LE NOM DANS LE PROFIL : deux endroits, donc une divergence
// possible. Ce test la ferme sans dedoubler la grammaire — il plante un gamertag a l offset que
// le profil annonce, et verifie que le parseur le retrouve.
func TestProfilHighlightEgaleLeParseur(t *testing.T) {
	for _, majeure := range []int{0, 37, 38, 39, 40, 41, 42} {
		h := profile.HighlightDepuisMajeure(majeure, majeure != 0)
		const tag = "TemoinDuProfil"
		chunk := chunkTempsFortTemoin(tag, h.GamertagOffsetBytes)
		evs, err := ParseHighlightEvents(chunk, majeure)
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
