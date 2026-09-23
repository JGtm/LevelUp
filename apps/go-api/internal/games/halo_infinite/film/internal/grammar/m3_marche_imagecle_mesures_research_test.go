//go:build research

package grammar

// m3_marche_imagecle_mesures_research_test.go — LOT M3.1, LES MESURES QUI ONT DECIDE LA REGLE DU
// BALAYEUR (suite de `m3_marche_imagecle_research_test.go`, qui porte les variantes) : les ancres
// ajoutees et leur confirmation par une autre image-cle, les ancres perdues, le recensement des
// en-tetes exacts de bipede, le cout de chaque lecture, la preuve « rouge avant », et le cas
// ti=9 de l image-cle d avant-match de la mini-bobine bcb6d393 (le repli d election qui reste).
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_MARCHE=1 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestM3(Marche|Recal)' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"testing"
	"time"
)

// m3N1 est la table apprise sur le payload EN COURS de marche : archetype -> n1 vu (et combien
// de fois). Elle est remplie par [m3WalkRG] a chaque ancre retenue.
type m3N1 map[int]map[uint64]int

// m3ScanRG est RECAL generalise : un candidat de generation 1 est SIGNE si c est un bipede
// exact (mot 35) OU si son mot de taille n1 (a +108) egale un n1 deja vu au moins une fois pour
// son archetype sur les ancres retenues de CE payload. Le plus proche candidat signe gagne s il
// precede l ancre elue.
func m3ScanRG(buf []byte, from, prevSlot, total, maxWin int, appris m3N1) (at int, fin bool) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	signe := -1
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q < end && q+64 <= total; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				fin = true
				break
			}
			continue
		}
		sentStreak = 0
		s, ti, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q, false
		}
		if signe < 0 && g == 1 {
			if ti == BipedTypeIndex || appris[ti][kfReadBits(buf, q+108, 32)] > 0 {
				signe = q
			}
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	if signe >= 0 && (at < 0 || signe < at) {
		return signe, fin
	}
	return at, fin
}

// m3WalkRG est la marche sous RG, fenetre glissante.
func m3WalkRG(buf []byte, maxWin int) []KeyframeRec {
	total := len(buf) * 8
	appris := m3N1{}
	scan := func(from, prev int) int {
		for f := from; f+64 <= total; f += maxWin {
			at, fin := m3ScanRG(buf, f, prev, total, maxWin, appris)
			if at >= 0 || fin {
				return at
			}
		}
		return -1
	}
	width, seen := map[int]int{}, map[int]int{}
	var out []KeyframeRec
	pos, prev := 1, -1
	if _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		pos = scan(pos, prev)
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		if appris[ti] == nil {
			appris[ti] = map[uint64]int{}
		}
		appris[ti][kfReadBits(buf, pos+108, 32)]++
		st := pos + 64
		nat := -1
		if w, has := width[ti]; has {
			if _, _, jg, vok := kfValidAnchor(buf, st+w, slot, total); vok && jg == 1 {
				nat = st + w
			}
		}
		if nat < 0 {
			nat = scan(st, slot)
		}
		out = append(out, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		prev = slot
		if nat < 0 {
			break
		}
		kfApprendreLargeur(width, seen, ti, nat-st)
		pos = nat
	}
	return out
}

// TestM3MarcheExtras : les ancres que RECAL-sans (sans fenetre) ou RECAL ajoute a la production
// sont-elles CONFIRMEES par une autre image-cle (meme slot, meme archetype, lues par la meme
// variante dans un AUTRE paquet d image-cle) ? Une ancre prise a une position fausse ne se repete
// presque jamais.
func TestM3MarcheExtras(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	type ancre struct{ slot, ti int }
	vars := []m3VarMarche{
		{"PROD", kfScanFenetreBits, m3ScanProdAncien},
		{"RECAL", kfScanFenetreBits, m3ScanRecal},
		{"RECAL-sans", 0, m3ScanRecal},
		{"RECAL-ext", kfScanFenetreBits, m3ScanRecalExt},
	}
	par := map[string][][]KeyframeRec{}
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, v := range vars {
				par[v.nom] = append(par[v.nom], m3Walk(pay, v.maxWin, v.scan))
			}
		}
	}
	for _, paire := range [][2]string{{"PROD", "RECAL"}, {"RECAL", "RECAL-sans"}, {"RECAL", "RECAL-ext"},
		{"RECAL-ext", "RECAL-sans"}} {
		ref, var2 := par[paire[0]], par[paire[1]]
		// confirmation : combien d images-cles de var2 portent (slot, ti)
		vus := map[ancre]int{}
		for _, recs := range var2 {
			for _, r := range recs {
				vus[ancre{r.Slot, r.TI}]++
			}
		}
		var extras, confirmes, bip, perdues int
		for k := range var2 {
			apres := map[int]bool{}
			for _, r := range var2[k] {
				apres[r.Bit] = true
			}
			for _, r := range ref[k] {
				if !apres[r.Bit] {
					perdues++
				}
			}
			avant := map[int]bool{}
			for _, r := range ref[k] {
				avant[r.Bit] = true
			}
			for _, r := range var2[k] {
				if avant[r.Bit] {
					continue
				}
				extras++
				if r.TI == BipedTypeIndex {
					bip++
				}
				if vus[ancre{r.Slot, r.TI}] >= 2 {
					confirmes++
				}
			}
		}
		t.Logf("== %s -> %s : %d ancres ajoutees (dont bipedes %d), CONFIRMEES par une autre image-cle %d ; "+
			"ancres PERDUES %d", paire[0], paire[1], extras, bip, confirmes, perdues)
	}
}

// TestM3RecalN1 : les en-tetes EXACTS de bipede (mot d archetype 35) — ceux que le recalage
// peut prendre — portent-ils tous le mot de taille n1 des bipedes ? Recensement exhaustif des
// positions du payload.
func TestM3RecalN1(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	n1s := map[uint64]int{}
	var exacts, gens int
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for q := 0; q+140 <= total; q++ {
				if kfReadBits(pay, q+32, 32) != BipedTypeIndex {
					continue
				}
				id := kfReadBits(pay, q, 32)
				if id == kfSent || id>>30 == 0 || int(id&0x3FFFFFFF) >= kfTableCap {
					continue
				}
				exacts++
				if id>>30 != 1 {
					gens++
				}
				n1s[kfReadBits(pay, q+108, 32)]++
			}
		}
	}
	t.Logf("== en-tetes EXACTS de bipede : %d (generation != 1 : %d) ; n1 %v", exacts, gens, n1s)
}

// TestM3MarcheDuree chronometre les deux lectures retenues sur tout le film.
func TestM3MarcheDuree(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	var pays [][]byte
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type == PacketTypeKeyframe {
				pays = append(pays, pk.Payload(data))
			}
		}
	}
	for _, v := range []m3VarMarche{{"PROD", kfScanFenetreBits, m3ScanProdAncien},
		{"RECAL", kfScanFenetreBits, m3ScanRecal}, {"RECAL-sans", 0, m3ScanRecal},
		{"RECAL-ext", kfScanFenetreBits, m3ScanRecalExt}} {
		debut := time.Now()
		n := 0
		for _, p := range pays {
			n += len(m3Walk(p, v.maxWin, v.scan))
		}
		t.Logf("== duree %-10s : %v pour %d images-cles (%d ancres)", v.nom, time.Since(debut), len(pays), n)
	}
}

// TestM3MarchePerdues : les ancres que la production porte et que RECAL-ext ne porte plus
// sont-elles confirmees par d AUTRES images-cles de la production (meme slot, meme archetype) ?
func TestM3MarchePerdues(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	type ancre struct{ slot, ti int }
	var prod, ext [][]KeyframeRec
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			prod = append(prod, m3Walk(pay, kfScanFenetreBits, m3ScanProdAncien))
			ext = append(ext, m3Walk(pay, kfScanFenetreBits, m3ScanRecalExt))
		}
	}
	vus := map[ancre]int{}
	for _, recs := range prod {
		for _, r := range recs {
			vus[ancre{r.Slot, r.TI}]++
		}
	}
	parTI := map[int]int{}
	var perdues, confirmees int
	for k := range prod {
		garde := map[int]bool{}
		for _, r := range ext[k] {
			garde[r.Bit] = true
		}
		for _, r := range prod[k] {
			if garde[r.Bit] {
				continue
			}
			perdues++
			parTI[r.TI]++
			if vus[ancre{r.Slot, r.TI}] >= 2 {
				confirmees++
				t.Logf("   perdue CONFIRMEE : kf %d slot %d ti %d gen %d bit %d (vue %d fois)", k, r.Slot, r.TI,
					r.Gen, r.Bit, vus[ancre{r.Slot, r.TI}])
			}
		}
	}
	t.Logf("== PROD -> RECAL-ext : %d ancres perdues, dont confirmees ailleurs par la production %d ; par ti %v",
		perdues, confirmees, parTI)
}

// TestM3MarcheExtrasDetail publie les ancres ajoutees par RECAL-ext et NON confirmees.
func TestM3MarcheExtrasDetail(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	type ancre struct{ slot, ti int }
	var rec, ext [][]KeyframeRec
	var tailles []int
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			rec = append(rec, m3Walk(pay, kfScanFenetreBits, m3ScanRecal))
			ext = append(ext, m3Walk(pay, kfScanFenetreBits, m3ScanRecalExt))
			tailles = append(tailles, len(pay)*8)
		}
	}
	vus := map[ancre]int{}
	for _, recs := range ext {
		for _, r := range recs {
			vus[ancre{r.Slot, r.TI}]++
		}
	}
	for k := range ext {
		avant := map[int]bool{}
		for _, r := range rec[k] {
			avant[r.Bit] = true
		}
		for i, r := range ext[k] {
			if avant[r.Bit] || vus[ancre{r.Slot, r.TI}] >= 2 {
				continue
			}
			prec := ext[k][max(i-1, 0)]
			t.Logf("   kf %d : slot %d ti %d gen %d bit %d / %d (%.1f%%) ; precedent slot %d ti %d bit %d", k, r.Slot,
				r.TI, r.Gen, r.Bit, tailles[k], 100*float64(r.Bit)/float64(tailles[k]), prec.Slot, prec.TI, prec.Bit)
		}
	}
}

// TestM3PreuveRougeAvant rejoue les payloads synthetiques du lot M3.1 (keyframe_world_test.go)
// sous la REGLE DE LA BASE fe7079f41 (`m3ScanProdAncien`) : elle doit y echouer.
func TestM3PreuveRougeAvant(t *testing.T) {
	recs := m3Walk(kfPayloadPrefixeBipedes(), kfScanFenetreBits, m3ScanProdAncien)
	bip := 0
	for _, r := range recs {
		if r.TI == BipedTypeIndex {
			bip++
		}
	}
	t.Logf("BASE, cas P2 : %d records, %d bipedes (nouveau : 3 bipedes) — %+v", len(recs), bip, recs)
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, 5, 300)
	w.bits(0, kfScanFenetreBits+5000)
	kfEcrireRecord(w, 1, 11, 5, 300)
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	recs = m3Walk(w.buf, kfScanFenetreBits, m3ScanProdAncien)
	t.Logf("BASE, fenetre vide : %d records (nouveau : 2)", len(recs))
}

// TestM3DiagEquipesPremierPaquet : quels paquets d image-cle de bcb6d393 portent des ti=9, et
// lesquels, sous la marche courante ET sous l ancienne regle.
func TestM3DiagEquipesPremierPaquet(t *testing.T) {
	film := bobineFilm(t, "bcb6d393")
	fc := NewFilmContext(film)
	reg, _ := fc.Registry()
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(raw)
			for _, nom := range []string{"NOUVEAU", "ANCIEN", "RG"} {
				var recs []KeyframeRec
				switch nom {
				case "NOUVEAU":
					recs = WalkKeyframeWorld(pay)
				case "ANCIEN":
					recs = m3Walk(pay, kfScanFenetreBits, m3ScanProdAncien)
				default:
					recs = m3WalkRG(pay, kfScanFenetreBits)
				}
				var idx []int
				for _, r := range recs {
					if r.TI != managedPlayerTypeIndex {
						continue
					}
					if i, _, ok := lireEquipeDuRecord(pay, r.Bit, reg, ContexteParDefaut()); ok {
						idx = append(idx, i)
					}
				}
				t.Logf("chunk %d paquet %d t=%d %s : %d records, ti9 index %v", c, pk.Index, pk.TimestampUS/100000,
					nom, len(recs), idx)
			}
		}
	}
}

// TestM3DiagTi9Exacts : dans le premier paquet d image-cle de bcb6d393, les en-tetes EXACTS
// (gen 1, mot d archetype 9) contre les ancres ti=9 de la marche.
func TestM3DiagTi9Exacts(t *testing.T) {
	film := bobineFilm(t, "bcb6d393")
	fc := NewFilmContext(film)
	raw, paquets, _ := fc.ChunkAt(1)
	pay := paquets[0].Payload(raw)
	total := len(pay) * 8
	for q := 0; q+140 <= total; q++ {
		if kfReadBits(pay, q+32, 32) != managedPlayerTypeIndex {
			continue
		}
		id := kfReadBits(pay, q, 32)
		if id>>30 != 1 || int(id&0x3FFFFFFF) >= kfTableCap {
			continue
		}
		t.Logf("exact ti9 : bit %d slot %d n1 %d", q, id&0x3FFFFFFF, kfReadBits(pay, q+108, 32))
	}
	recs, st := WalkKeyframeWorldStats(pay)
	for i, r := range recs {
		if r.TI == managedPlayerTypeIndex || (i > 0 && recs[i-1].Slot+1 != r.Slot) {
			t.Logf("marche : bit %d slot %d ti %d", r.Bit, r.Slot, r.TI)
		}
	}
	t.Logf("stats %+v", st)
}

// TestM3TemoinImageCle1494 : sur 81c02726, les bipedes lus a CHAQUE image-cle par la marche de
// production, contre la regle de la base — le temoin du plan est l image-cle du morceau 9
// (t 1494), ou la base ne lisait qu UN bipede sur huit.
func TestM3TemoinImageCle1494(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			var avant, apres []int
			for _, r := range m3Walk(pay, kfScanFenetreBits, m3ScanProdAncien) {
				if r.TI == BipedTypeIndex {
					avant = append(avant, r.Slot)
				}
			}
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI == BipedTypeIndex {
					apres = append(apres, r.Slot)
				}
			}
			t.Logf("chunk %2d t=%d : base %d bipedes %v ; production %d bipedes %v", ch,
				pk.TimestampUS/100000, len(avant), avant, len(apres), apres)
		}
	}
}
