package filmdec

// residus_pied_research_test.go — PHASE 5b, RESIDU 6 : CE QUE VAUT L'OCTET 55 DU PIED.
//
// LA CONTRADICTION, DATEE ET JAMAIS TRANCHEE. Trois affirmations du depot se contredisent sur
// l'equipe d'un evenement d'objectif lu dans le pied de film (chunk de type 3, blocs de
// 60 octets) :
//
//	`objectiveevents/film.go` lit l'octet 55 sous le nom `teamRaw` et le commente
//	  « NON fiable sur certains matchs -> a confirmer via roster » ;
//	`objectiveevents/extract.go` tranche : « le champ team du film etant non fiable, l'equipe
//	  d'un event vient TOUJOURS du roster via le xuid de l'acteur » ;
//	`.ai/archive/V7/RESEARCH_THEATER_RE.md:534` identifie l'equipe a `b37`/`b38`
//	  (« CONFIRME : le split 4/4 des events colle exactement au roster DB »), sa ligne 624 la
//	  dit fiable en `b55`, et sa ligne 645 la dit FAUSSE.
//
// Les phases 3 et 4 ont laisse la question ouverte faute d'equipe de reference interne. Elle ne
// l'est plus : la phase 3 a etabli l'equipe par joueur dans la TRAME (designateur de 4 bits a
// 186 bits du record ti=9) et la phase 5b a rendu le lecteur de la table des slots exact sur les
// sept builds. On dispose donc d'une equipe PROUVEE par XUID, interne au film, et l'octet 55 se
// mesure contre elle SANS passer par la base.
//
// LE BALAYAGE EST AVEUGLE, ET C'EST LE POINT. On ne teste pas `b55` : on teste LES SOIXANTE
// octets du bloc, l'un apres l'autre, contre l'equipe prouvee de l'acteur. Trois lectures sont
// essayees par octet (la valeur brute, la valeur moins un, le bit de poids faible), parce que
// les trois conventions existent deja dans le format. Si un octet porte l'equipe, il sortira a
// 100 % et les 59 autres donneront le plancher de bruit MESURE (methode, regle 4). Si aucun ne
// sort, le negatif est publie avec ce plancher.
//
// LE BLOC EST CELUI DE LA PRODUCTION, transcrit sans changement depuis
// `objectiveevents/film.go` (`scanTh10Events` / `decodeTh10Block`, 2026-09-13) : on localise un
// XUID (prefixe 0x2d ou 0x25, suffixe 0xc0), on cherche le marqueur de fin `00 00 2e e0` de son
// bloc, on recule de 60 octets, on exige `type_hint == 10` a l'octet 47. Aucun code de
// production n'est appele ni modifie : le paquet `objectiveevents` n'exporte pas ces fonctions.
//
// Garde CHUNK00_FILMS. Lecture seule.

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

const (
	rpBlocOctets = 60 // la taille du bloc d'evenement du pied
	rpOctetType  = 47 // type_hint
	rpOctetSlot  = 36
	rpOctetTeam  = 55 // ce que la production lit sous le nom `teamRaw`
	rpTypeHint10 = 10
	// Bornes de plausibilite d'un XUID Xbox, identiques a celles du balayage de chunk_00.
	rpXuidLo = uint64(0x0009000000000000)
	rpXuidHi = uint64(0x000A000000000000)
)

// rpOctetA lit un octet a un offset BIT arbitraire (transcription de `readByteAtBit`).
func rpOctetA(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	i, off := bit/8, uint(bit%8)
	if off == 0 {
		return d[i]
	}
	return d[i]<<off | d[i+1]>>(8-off)
}

// rpU64LE lit un uint64 petit-boutiste a un offset bit arbitraire.
func rpU64LE(d []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(rpOctetA(d, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

// rpBloc : un evenement du pied, avec ses 60 octets bruts et le XUID de son acteur.
type rpBloc struct {
	xuid   uint64
	t      int
	octets [rpBlocOctets]byte
}

// rpScan rejoue `scanTh10Events` sur un chunk decompresse et rend les blocs complets.
func rpScan(d []byte) []rpBloc {
	total := len(d) * 8
	var out []rpBloc
	vus := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if rpOctetA(d, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := rpOctetA(d, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xs := xe - 64
		if vus[xs] {
			continue
		}
		x := rpU64LE(d, xs)
		if x <= rpXuidLo || x >= rpXuidHi {
			continue
		}
		vus[xs] = true
		if b, ok := rpDecode(d, xs, total); ok {
			b.xuid = x
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// rpDecode cherche le marqueur de fin du bloc et rend ses 60 octets (transcription de
// `decodeTh10Block`).
func rpDecode(d []byte, xs, total int) (rpBloc, bool) {
	win := xs + 20000
	if win > total {
		win = total
	}
	for b := xs; b <= win-32; b++ {
		if rpOctetA(d, b) != 0 || rpOctetA(d, b+8) != 0 ||
			rpOctetA(d, b+16) != 0x2e || rpOctetA(d, b+24) != 0xe0 {
			continue
		}
		ebs := b - rpBlocOctets*8
		if ebs < xs || int(rpOctetA(d, ebs+rpOctetType*8)) != rpTypeHint10 {
			return rpBloc{}, false
		}
		var bl rpBloc
		for k := 0; k < rpBlocOctets; k++ {
			bl.octets[k] = rpOctetA(d, ebs+k*8)
		}
		bl.t = int(bl.octets[48])<<24 | int(bl.octets[49])<<16 |
			int(bl.octets[50])<<8 | int(bl.octets[51])
		return bl, true
	}
	return rpBloc{}, false
}

// rpEquipesProuvees rend la table `xuid -> equipe` etablie DANS LE FILM : le rang de la table
// des slots de `chunk_00` (lecteur corrige de R2, qui enjambe les slots vacants) donne le XUID,
// la i-eme entite ti=9 de la premiere image-cle retenue donne le designateur, et l'equipe vaut
// `brut - 1`. La table n'est rendue que si les deux cardinaux coincident : sinon l'appariement
// par rang n'a pas de sens, et le film est publie comme ecarte.
func rpEquipesProuvees(dir string, d []byte) (map[uint64]int, int, int) {
	v, err := equipePrepare(dir)
	if err != nil || !v.Longueur {
		return nil, 0, 0
	}
	delta, _, _ := rsDelta(d)
	enrs, _ := rsChaine(d, delta)
	lu := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
	if len(enrs) != len(lu) || len(enrs) == 0 {
		return nil, len(enrs), len(lu)
	}
	out := map[uint64]int{}
	for i, e := range enrs {
		out[e.xuid] = lu[i] - 1
	}
	return out, len(enrs), len(lu)
}

// rpAccords : pour chaque octet du bloc et chacune des trois lectures, le nombre d'evenements
// dont la valeur vaut l'equipe prouvee de l'acteur.
type rpAccords struct {
	brut, moinsUn, bitBas [rpBlocOctets]int
	compares              int
	valeurs               [rpBlocOctets]map[int]int
}

// rpAjoute confronte un bloc a l'equipe prouvee de son acteur.
func (a *rpAccords) rpAjoute(b rpBloc, eq int) {
	a.compares++
	for k := 0; k < rpBlocOctets; k++ {
		v := int(b.octets[k])
		if a.valeurs[k] == nil {
			a.valeurs[k] = map[int]int{}
		}
		a.valeurs[k][v]++
		if v == eq {
			a.brut[k]++
		}
		if v-1 == eq {
			a.moinsUn[k]++
		}
		if v&1 == eq {
			a.bitBas[k]++
		}
	}
}

// TestResidusPiedOctetEquipe execute P-BAL : le balayage aveugle des 60 octets du bloc contre
// l'equipe PROUVEE de l'acteur, avec son plancher de bruit mesure.
func TestResidusPiedOctetEquipe(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	acc := &rpAccords{}
	films, ecartes, evts := 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		eqs, nEnr, nTrame := rpEquipesProuvees(dir, d)
		if eqs == nil {
			ecartes++
			t.Logf("%-10s : ECARTE — %d rang(s) de la table contre %d entite(s) ti=%d",
				filepath.Base(dir), nEnr, nTrame, equipeTI)
			continue
		}
		blocs, connus := rpFilm(dir, eqs, acc)
		films++
		evts += blocs
		t.Logf("%-10s : %d evenement(s) th=10 dans le pied, %d a acteur d'equipe prouvee "+
			"(%d joueurs)", filepath.Base(dir), blocs, connus, len(eqs))
	}
	if acc.compares == 0 {
		t.Skip("aucun evenement exploitable")
	}
	t.Logf("=== P-BAL === %d film(s) retenus, %d ecarte(s), %d evenement(s) th=10 vus, "+
		"%d confrontes a l'equipe prouvee", films, ecartes, evts, acc.compares)
	rpRapport(t, acc)
}

// rpFilm lit le pied d'un film et confronte ses blocs. Rend (blocs vus, blocs confrontes).
func rpFilm(dir string, eqs map[uint64]int, acc *rpAccords) (vus, connus int) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return 0, 0
	}
	pied, err := ReadFilmChunk(dir, n)
	if err != nil {
		return 0, 0
	}
	for _, b := range rpScan(pied) {
		vus++
		eq, ok := eqs[b.xuid]
		if !ok || eq < 0 {
			continue
		}
		connus++
		acc.rpAjoute(b, eq)
	}
	return vus, connus
}

// rpRapport publie le classement des octets et le detail des trois octets en litige.
func rpRapport(t *testing.T, a *rpAccords) {
	t.Helper()
	type ligne struct {
		k, n int
		quoi string
	}
	var ls []ligne
	for k := 0; k < rpBlocOctets; k++ {
		ls = append(ls, ligne{k, a.brut[k], "brut"}, ligne{k, a.moinsUn[k], "valeur-1"},
			ligne{k, a.bitBas[k], "bit bas"})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
	t.Logf("  les dix meilleures lectures sur %d evenements :", a.compares)
	for i := 0; i < 10 && i < len(ls); i++ {
		t.Logf("    b%-2d %-8s : %d/%d (%.1f %%)", ls[i].k, ls[i].quoi, ls[i].n, a.compares,
			100*float64(ls[i].n)/float64(a.compares))
	}
	for _, k := range []int{37, 38, rpOctetTeam, rpOctetSlot} {
		t.Logf("  b%-2d : brut %d/%d, valeur-1 %d/%d, bit bas %d/%d ; valeurs observees %s",
			k, a.brut[k], a.compares, a.moinsUn[k], a.compares, a.bitBas[k], a.compares,
			rpHisto(a.valeurs[k]))
	}
	parfaits := 0
	for _, l := range ls {
		if l.n == a.compares {
			parfaits++
		}
	}
	t.Logf("  lectures en accord PARFAIT : %d sur %d essayees (60 octets x 3 lectures) — "+
		"c'est le plancher de bruit MESURE", parfaits, len(ls))
}

// rpHisto rend une distribution triee, tronquee aux six valeurs les plus frequentes.
func rpHisto(m map[int]int) string {
	type kv struct{ k, n int }
	var xs []kv
	for k, n := range m {
		xs = append(xs, kv{k, n})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	s := ""
	for i, x := range xs {
		if i == 6 {
			s += fmt.Sprintf("... (%d valeurs)", len(xs))
			break
		}
		s += fmt.Sprintf("%d:x%d ", x.k, x.n)
	}
	return s
}

// TestResidusPiedDrapeaux publie le contenu brut des octets 37 a 43 et de l'octet 55, evenement
// par evenement, sur les premiers evenements de chaque film. C'est le releve qui permet de dire
// ce que l'octet 55 EST, une fois etabli qu'il n'est pas l'equipe.
func TestResidusPiedDrapeaux(t *testing.T) {
	rel := LockProcessDecode()
	defer rel()
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		eqs, _, _ := rpEquipesProuvees(dir, d)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		pied, err := ReadFilmChunk(dir, n)
		if err != nil {
			continue
		}
		blocs := rpScan(pied)
		t.Logf("=== %s === %d evenement(s) th=10", filepath.Base(dir), len(blocs))
		for i, b := range blocs {
			if i >= 8 {
				break
			}
			eq := -2
			if v, ok := eqs[b.xuid]; ok {
				eq = v
			}
			t.Logf("  t=%7d xuid %19d equipe prouvee %2d | b36 %3d | b37..b43 % x | b55 %3d",
				b.t, b.xuid, eq, b.octets[rpOctetSlot], b.octets[37:44], b.octets[rpOctetTeam])
		}
	}
}
