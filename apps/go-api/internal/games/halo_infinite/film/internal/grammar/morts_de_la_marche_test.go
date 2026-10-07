package grammar

// morts_de_la_marche_test.go — LES MORTS QUE LA MARCHE REND A KILLSOURCE, ET LE CRITERE DE SA
// CALIBRATION, SUR UNE BOBINE.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// handleAbsent est la valeur d un handle du dead-state que le flux ne porte pas.
const handleAbsent = 0xFFFFFFFF

// TestLesMortsDeLaMarcheSeRelisentALeurPosition : sur la bobine de la marche ([bobineMarcheDir]),
// chaque dead-state `Mort` que [LireLesMortsDeLaMarche] rend se relit a sa position : sa trame est une
// trame delta du film (position du chunk, rang, horodatage), et quand son composant est dans la
// trace, le bit `Mort` y vaut 1 et le handle de tete, s il est lu, suit sa garde. Les chunks de la
// bobine sont contigus : la position et le numero d un chunk y coincident. MUTATION — le bit du
// composant decale d un bit : ROUGE.
func TestLesMortsDeLaMarcheSeRelisentALeurPosition(t *testing.T) {
	film := bobineDeLaMarche(t)
	lus, err := LireLesMortsDeLaMarche(NewFilmContext(film))
	if err != nil {
		t.Fatal(err)
	}
	trames := map[[2]int]types.Packet{}
	for _, p := range film.AllPackets() {
		trames[[2]int{p.Chunk, p.Index}] = p
	}
	avecBit, avecHandle := 0, 0
	for _, m := range lus.Lus {
		p, ok := trames[[2]int{m.PositionDuChunk, m.Index}]
		if !ok || p.Type != int(PacketTypeDelta) || p.TS != m.TS {
			t.Fatalf("mort %+v : trame absente ou differente", m)
		}
		if m.Bit < 0 {
			continue
		}
		avecBit++
		if source.BitAt(p.Payload, m.Bit) != 1 {
			t.Errorf("mort %+v : le bit Mort vaut 0 a %d", m, m.Bit)
		}
		if m.Dead.SrcTag0 == handleAbsent {
			continue
		}
		avecHandle++
		if source.BitAt(p.Payload, m.Bit+1) != 1 ||
			uint32(source.BitsAt(p.Payload, m.Bit+2, 32)) != m.Dead.SrcTag0 { //nolint:gosec // R(32)
			t.Errorf("mort %+v : handle de tete %08x absent de sa position", m, m.Dead.SrcTag0)
		}
	}
	if avecBit == 0 || avecHandle == 0 || lus.Stats.Packets == 0 {
		t.Fatalf("%d mort(s), %d a leur position, %d avec un handle, %d trame(s) : la bobine ne prouve rien",
			len(lus.Lus), avecBit, avecHandle, lus.Stats.Packets)
	}
}

// TestLeCritereDeCalibrationLitSousLEnTeteDeLaMarche : les scores de la calibration ne dependent
// pas de l identifiant bas des cadres candidats — c est celui de l en-tete de la marche — et un
// second calcul rend les memes scores (le monde est restaure apres chaque essai). MUTATION —
// l identifiant bas du cadre garde : ROUGE.
func TestLeCritereDeCalibrationLitSousLEnTeteDeLaMarche(t *testing.T) {
	film := bobineDeLaMarche(t)
	ech := EchantillonDeCalibration{Taille: 60, OctetsMin: 100}
	cadre := NewFilmContext(film).CadreDeBalayage()
	faux := cadre
	faux.IDLowBits = cadre.IDLowBits - 4
	scores, n, err := NewFilmContext(film).ScoresDeCalibration([]FrameConfig{cadre, faux}, ech, 8)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 || scores[0] == 0 {
		t.Fatalf("echantillon %d, score %d : la bobine ne prouve rien", n, scores[0])
	}
	if scores[1] != scores[0] {
		t.Errorf("identifiant bas du cadre a %d : score %d, contre %d sous l en-tete", faux.IDLowBits,
			scores[1], scores[0])
	}
	encore, _, err := NewFilmContext(film).ScoresDeCalibration([]FrameConfig{cadre, faux}, ech, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(encore, scores) {
		t.Errorf("second calcul %v, premier %v", encore, scores)
	}
}

// bobineDeLaMarche charge la bobine de la marche ([bobineMarcheDir]) : son registre et ses trames delta.
func bobineDeLaMarche(t *testing.T) *source.Film {
	t.Helper()
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine de la marche : %v", err)
	}
	return film
}
