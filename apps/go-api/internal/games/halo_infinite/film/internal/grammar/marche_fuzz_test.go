package grammar

// marche_fuzz_test.go — T5 DE LA SPECIFICATION DE LA REPRESENTATION INTERMEDIAIRE (ADR 0037) : la
// marche d un payload QUELCONQUE, dans les deux phases, ne panique pas, et ce qu elle range est
// borne par les bits du payload. Le harnais est [FuzzFilmRecordReaders] : ses graines (payloads
// reels et troncatures) rejouent ce contrat a chaque `go test`.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// largeurMinimaleDUneAncre : deux ancres d image-cle sont a 64 bits au moins l une de l autre
// (l en-tete `[id:32][field:26][ti:6]`, `keyframe_world.go`).
const largeurMinimaleDUneAncre = 64

// marcherUnPayloadQuelconque marche `pay` comme une trame delta depuis la tete du paquet, sur un
// monde ou deux slots sont vivants sous un archetype a composants (dont un sans lecteur), puis
// comme un paquet d image-cle, et verifie les bornes de ce qui est range : un record par bit au
// plus (chaque record lit au moins son prefixe de type), 64 composants par record au plus (le
// masque), un tour de vue C par bit au plus ; une ancre d image-cle par 64 bits au plus.
func marcherUnPayloadQuelconque(t *testing.T, pay []byte) {
	t.Helper()
	bits := len(pay) * 8
	w := mondeDeCarte("object-scale-component", composantSansLecteurJ40)
	cfg := cadreDeCarte()
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, pay, w, cfg, departDeTrame{bit: DefaultPacketPreambleBits}, &l)
	p := &lecture.Paquet{Payload: pay, Debut: lecture.DebutEnTete}
	rangerLaTrame(p, &l, true, nil)
	if len(p.Records) > bits || len(p.Comps) > 64*len(p.Records) || len(p.VueC.Entrees) > bits+1 {
		t.Fatalf("trame de %d bits : %d record(s), %d composant(s), %d tour(s) de vue C — hors des bornes",
			bits, len(p.Records), len(p.Comps), len(p.VueC.Entrees))
	}
	m := marcheDesImagesCles{reg: w.Reg, ctx: ContexteParDefaut()}
	m.marcherLePaquet(0, FilmPacket{Type: PacketTypeKeyframe, Size: len(pay)}, pay)
	if n := len(m.paquet.Records); n > bits/largeurMinimaleDUneAncre+1 || len(m.paquet.Comps) > 64*n {
		t.Fatalf("image-cle de %d bits : %d record(s), %d composant(s) — hors des bornes", bits, n, len(m.paquet.Comps))
	}
}

// TestLaMarcheDUnPayloadQuelconqueResteBornee : les cas limites du harnais, hors fuzz — payload
// vide, un octet, un octet plein, et un paquet synthetique dont la vue C est lue.
func TestLaMarcheDUnPayloadQuelconqueResteBornee(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123, 0)
	bw.bits(0x2a5, 10)
	for _, pay := range [][]byte{nil, {0x00}, {0xff}, {0xff, 0xff, 0xff}, bw.buf} {
		marcherUnPayloadQuelconque(t, pay)
	}
}
