package filmdec

// r7_oracle_trame_research_test.go — lot R7, TEMOIN 3 : L'ORACLE DE TRAME, le seul juge
// vraiment discriminant du cadrage d'une liste d'evenements.
//
// POURQUOI LUI ET PAS LE TAUX DE FERMETURE. Le « bit de fin de liste » est UN bit : a un
// cadrage faux, il vaut 0 une fois sur deux et la marche se declare « finie proprement »
// apres zero evenement. La mesure le confirme sur pieces (TestR7Marche) : le taux de fin
// propre est de 99,9 % au bon cadrage ET a +1 bit ET a +3 bits — INDISCRIMINANT, publie tel
// quel. C'est la lecon deja payee par le lot damage_aftermath : « le discriminant est la
// PROFONDEUR, pas le taux de fermeture ».
//
// LE JUGE. Apres le dernier evenement de la liste commence la trame de records d'entites.
// Si la marche s'arrete au bon bit, cette trame se decode et VA LOIN ; si elle s'arrete un
// bit trop tot ou trop tard, le premier record est deja du bruit. On mesure, sur les MEMES
// paquets : profondeur moyenne (records lus avant desynchronisation) au bon cadrage CONTRE
// un temoin decale de +3 bits.
//
// SEUIL ECRIT AVANT LA MESURE : facteur >= 3 sur la profondeur moyenne (le lot
// damage_aftermath avait mesure 13 sur un seul type ; 3 est une marge prudente pour une
// marche qui traverse des types varies).
//
// La largeur d'id des records (IDLowBits) est une valeur de RUNTIME qui change d'un film a
// l'autre : elle est CALIBREE par film sur les paquets a LISTE VIDE, dont le cadrage n'est
// pas en question (la trame y commence au bit 2, sans aucun evenement a traverser).
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0, balayage borne (R7_CHUNKS).
//
//	CGO_ENABLED=0 R7_ROOT=... R7_IDS=... R7_CAT=... R7_MAPS=... R7_CHUNKS=8 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7OracleTrame$' -count=1 -timeout 60m -v

import (
	"path/filepath"
	"testing"
)

// r7TrameStat agrege l'oracle sur un cadrage donne.
type r7TrameStat struct {
	paquets   int
	fermees   int
	records   int
	depassees int // trame qui deborde du payload : lecture manifestement fausse
}

func (s r7TrameStat) profondeur() float64 {
	if s.paquets == 0 {
		return 0
	}
	return float64(s.records) / float64(s.paquets)
}

func (s *r7TrameStat) cumule(o r7TrameStat) {
	s.paquets += o.paquets
	s.fermees += o.fermees
	s.records += o.records
	s.depassees += o.depassees
}

// r7CalibreIDLow choisit la largeur d'id qui maximise la profondeur de trame sur les paquets
// a LISTE VIDE (cadrage certain : la trame commence au bit 2).
func r7CalibreIDLow(reg *Registry, chunks [][]byte) (int, float64) {
	best, bestProf := 13, -1.0
	for w := 10; w <= 15; w++ {
		cfg := DefaultFrameConfig()
		cfg.IDLowBits = w
		n, recs := 0, 0
		for _, data := range chunks {
			wd := NewWorld(reg)
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 1 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 != 0 {
					continue // liste non vide : hors calibration
				}
				r, _ := DecodeFrameRecords(NewBitReader(pay), wd, cfg)
				n++
				recs += len(r)
			}
		}
		if n == 0 {
			continue
		}
		if p := float64(recs) / float64(n); p > bestProf {
			best, bestProf = w, p
		}
	}
	return best, bestProf
}

// r7Chargements lit les chunks bornes d'un film et son registre.
func r7Chargements(dir string) (*Registry, [][]byte, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, nil, err
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil, nil, err
	}
	var chunks [][]byte
	for c, n := 1, r7Chunks(dir); c <= n; c++ {
		if data, err := ReadFilmChunk(dir, c); err == nil {
			chunks = append(chunks, data)
		}
	}
	return reg, chunks, nil
}

// r7OracleFilm mesure l'oracle de trame sur un film. `garde` filtre les listes retenues
// (nil = toutes) ; `decalage` decale volontairement le depart de la trame (temoin negatif).
// Rend la mesure et le nombre de listes non marchees jusqu'au bout.
func r7OracleFilm(reg *Registry, chunks [][]byte, ctx r7Ctx, cfg FrameConfig,
	garde func([]r7Ev) bool, decalage int) (r7TrameStat, int) {
	var st r7TrameStat
	horsMarche := 0
	for _, data := range chunks {
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				_, _ = DecodeFrameRecords(NewBitReader(pay), wBase, cfg)
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 == 0 {
				continue // liste vide : le cadrage n'y est pas en question
			}
			evs, stop, _, fin := r7Marche(pay, ctx)
			if stop != r7StopFin {
				horsMarche++
				continue
			}
			if garde != nil && !garde(evs) {
				continue
			}
			r7Juge(reg, snap, pay, fin+decalage, cfg, &st)
		}
	}
	return st, horsMarche
}

// r7Juge decode la trame a partir du bit donne sur un World restaure et cumule la mesure.
func r7Juge(reg *Registry, snap WorldSnapshot, pay []byte, bit int, cfg FrameConfig, st *r7TrameStat) {
	if bit < 0 || bit+8 > len(pay)*8 {
		return
	}
	w := NewWorld(reg)
	w.Restore(snap)
	br := NewBitReader(pay)
	br.Skip(bit)
	recs, err := DecodeFrameRecords(br, w, cfg)
	st.paquets++
	st.records += len(recs)
	if err == nil {
		st.fermees++
	}
	if br.BitPos() > len(pay)*8 {
		st.depassees++
	}
}

func r7RapportTrame(t *testing.T, titre string, s r7TrameStat) {
	t.Helper()
	t.Logf("%s: %d trames · profondeur %.3f records/paquet · %d fermetures propres (%.1f %%) · %d debordements",
		titre, s.paquets, s.profondeur(), s.fermees,
		100*float64(s.fermees)/float64(max(1, s.paquets)), s.depassees)
}

// TestR7OracleTrame juge la marche par la trame qui la suit.
func TestR7OracleTrame(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	var parcJuste, parcTemoin r7TrameStat
	for _, id := range ids {
		dir := filepath.Join(root, id)
		reg, chunks, err := r7Chargements(dir)
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		cfg := DefaultFrameConfig()
		var prof float64
		cfg.IDLowBits, prof = r7CalibreIDLow(reg, chunks)
		t.Logf("== FILM %s (%d chunks) · IDLowBits calibre = %d (profondeur %.2f a liste vide) ==",
			id, len(chunks), cfg.IDLowBits, prof)
		ctx := cartes[id]
		juste, horsMarche := r7OracleFilm(reg, chunks, ctx, cfg, nil, 0)
		temoin, _ := r7OracleFilm(reg, chunks, ctx, cfg, nil, 3)
		r7RapportTrame(t, "  cadrage JUSTE ", juste)
		r7RapportTrame(t, "  temoin +3 bits", temoin)
		t.Logf("  (%d listes non marchees jusqu'au bout : type opaque)", horsMarche)
		parcJuste.cumule(juste)
		parcTemoin.cumule(temoin)
	}
	t.Logf("")
	r7RapportTrame(t, "PARC — cadrage JUSTE ", parcJuste)
	r7RapportTrame(t, "PARC — temoin +3 bits", parcTemoin)
	den := parcTemoin.profondeur()
	if den < 0.0001 {
		den = 0.0001
	}
	t.Logf("TEMOIN 3 (oracle de trame) : facteur de profondeur %.2f (seuil ecrit d'avance : 3)",
		parcJuste.profondeur()/den)
}
