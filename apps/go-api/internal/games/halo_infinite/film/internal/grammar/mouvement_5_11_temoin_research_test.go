//go:build research

package grammar

// mouvement_5_11_temoin_research_test.go — LE FILM TEMOIN CONTROLE (lot 5.11).
//
// # L ORACLE
//
// L utilisateur a fait enregistrer un film ou UN SEUL joueur est actif et n a fait QUE sauter,
// une fois, autour de la dixieme seconde. Tout champ replique qui BASCULE a cet instant, pour ce
// bipede, et NULLE PART ailleurs dans le film, est le declencheur du saut. C est le seul
// dispositif qui separe un declencheur d une correlation : sur un film de match, trente
// evenements co-occurrent avec un saut.
//
// # CE QUE CE FICHIER MESURE
//
// `TestMouvement511Registre` : le registre du film (identite, build, roster), le decoupage d `i0`
// lu dans le film, et les entrees du catalogue de bornes qui portent cette signature — la carte
// d un film se lit dans son registre, et sans ses largeurs d axe tout le reste est du bruit
// (lecon du 5.3.5).
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement511Registre$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m511Film charge le film temoin depuis la garde d environnement.
func m511Film(t *testing.T) *source.Film {
	t.Helper()
	dir := os.Getenv("MOUV511_FILM")
	if dir == "" {
		t.Skip("MOUV511_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	return film
}

func TestMouvement511Registre(t *testing.T) {
	film := m511Film(t)
	m511Identite(t, film)
	m511Carte(t, film)
	m511Chunks(t, film)
}

// m511Identite publie l identite du film et son roster.
func m511Identite(t *testing.T, film *source.Film) {
	t.Helper()
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		t.Fatalf("le film ne porte pas son chunk_00 : aucun registre")
	}
	ident, err := ReadFilmIdentity(reg)
	if err != nil {
		t.Fatalf("identite : %v", err)
	}
	t.Logf("IDENTITE : version %q · build %q · saveur %q · buildID %d · changelist %d · "+
		"format %d · blocs de registre %d",
		ident.Version, ident.Build, ident.Flavor, ident.BuildID, ident.Changelist,
		ident.FormatVersion, ident.RegistryBlocks)
	t.Logf("  DEBUT DU MATCH (horloge du film) : %d = %s", ident.MatchStartUnix,
		time.Unix(int64(ident.MatchStartUnix), 0).UTC().Format(time.RFC3339))
	slots, rep, err := ReadPlayerTable(reg, ident)
	if err != nil {
		t.Logf("ROSTER : table des joueurs illisible (%v)", err)
		return
	}
	t.Logf("ROSTER : %d enregistrements (rapport %+v)", len(slots), rep)
	for _, s := range slots {
		t.Logf("  filmIndex %2d · xuid %d · %q", s.FilmIndex, s.XUID, s.Gamertag)
	}
}

// m511Carte lit le decoupage d `i0` DANS le film et cherche les cartes du catalogue qui le
// portent. LE CATALOGUE EST INDEXE PAR NOM DE MATCH, pas par signature : quand plusieurs cartes
// partagent la signature (cartes jumelles, lot 1.9.4), l instrument les rend TOUTES et le dit.
func m511Carte(t *testing.T, film *source.Film) {
	t.Helper()
	lay, rep, err := DetectI0LayoutOf(film)
	if err != nil {
		t.Logf("DECOUPAGE i0 : non concluant (%v) — rapport %d paires", err, rep.Pairs)
		return
	}
	t.Logf("DECOUPAGE i0 LU DANS LE FILM : porte %d bits · axes %v · region %d (%d paires)",
		lay.GateBits, lay.AxisW, lay.Region, rep.Pairs)
	chemin := os.Getenv("MOUV511_BORNES")
	if chemin == "" {
		t.Logf("MOUV511_BORNES absent : pas de confrontation au catalogue")
		return
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	var hits []string
	for nom, e := range cat.Maps {
		if e.Layout().AxisW == lay.AxisW {
			hits = append(hits, fmt.Sprintf("%s (axes %v, region %d, bits d index %d)",
				nom, e.AxisWidths, e.Region, e.EffectiveRegionIndexBits()))
		}
	}
	sort.Strings(hits)
	t.Logf("CARTES DU CATALOGUE A CETTE SIGNATURE (%d sur %d) : %s",
		len(hits), len(cat.Maps), strings.Join(hits, " · "))
}

// m511Chunks publie la decoupe temporelle du film, pour ancrer la fenetre de 10 s.
func m511Chunks(t *testing.T, film *source.Film) {
	t.Helper()
	fc := NewFilmContext(film)
	var total, deltas int
	var tmin, tmax uint64
	premier := true
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		_ = data
		var d int
		for _, pk := range pks {
			total++
			if pk.Type != PacketTypeDelta {
				continue
			}
			d++
			deltas++
			if premier || pk.TimestampUS < tmin {
				tmin = pk.TimestampUS
				premier = false
			}
			if pk.TimestampUS > tmax {
				tmax = pk.TimestampUS
			}
		}
		t.Logf("  chunk %d : %d paquets, dont %d deltas", c, len(pks), d)
	}
	t.Logf("PAQUETS : %d au total, %d deltas · horodatages de %d a %d us (etendue %.3f s)",
		total, deltas, tmin, tmax, float64(tmax-tmin)/1e6)
}

// m511Marche est LA MARCHE DU JEU sur le film temoin, sous une carte donnee : trois vues,
// paquets a liste pleine localises, largeurs d axe installees. Elle rend l oracle de contenu.
type m511Marche struct {
	ti35, desync, paquets, pleins, vues int
	etalon                              map[int]int
	fautifs                             map[int]int
	parTI                               map[uint32]int
	desyncTI                            map[uint32]int
	parPaquet                           map[int]int
	kf, records                         int
}

// m511Contexte ouvre le contexte du film sous une entree de catalogue, largeurs installees.
func m511Contexte(film *source.Film, entry profile.MapQuantEntry) *FilmContext {
	fc := NewFilmContextForMap(film, &entry, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entry.Layout())
	fc.PoserProfilDeBalayage(bal)
	return fc
}

// m511Balayer marche le film (ou ses `nChunks` premiers chunks quand nChunks > 0).
func m511Balayer(fc *FilmContext, nChunks, idLow int) (*m511Marche, error) {
	reg, err := fc.Registry()
	if err != nil {
		return nil, err
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	m := &m511Marche{etalon: map[int]int{}, fautifs: map[int]int{}, parTI: map[uint32]int{},
		desyncTI: map[uint32]int{}, parPaquet: map[int]int{}}
	cfg := fc.CadreDeBalayage()
	if idLow > 0 {
		cfg.IDLowBits = idLow
	}
	w := NewWorld(reg)
	nums := fc.ChunkNumbers()
	if nChunks > 0 && len(nums) > nChunks {
		nums = nums[:nChunks]
	}
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type == PacketTypeKeyframe {
				m.kf++
			}
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
				m.pleins++
			}
			m.paquets++
			recs, vues := DecodeFrameViews(pay, w, cfg, 3, debut)
			m.vues += vues
			m.parPaquet[len(recs)]++
			for _, r := range recs {
				m.records++
				m.parTI[r.TypeIndex]++
				if r.DesyncAt >= 0 {
					m.desyncTI[r.TypeIndex]++
				}
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				m.ti35++
				if r.DesyncAt >= 0 {
					m.desync++
					m.fautifs[r.DesyncAt]++
				}
				for _, b := range []int{0, 1, 21, 25} {
					if r.Trace.Mask&(1<<uint(b)) != 0 {
						m.etalon[b]++
					}
				}
			}
		}
	}
	return m, nil
}

// TestMouvement511Carte BALAYE LE CATALOGUE : la carte d un film dont la signature d `i0` ne
// tranche pas (un seul bipede, 68 paires) se trouve par l ORACLE DE CONTENU — la bonne carte est
// celle qui rend une trame cadree, les autres decalent `i0` et tout ce qui suit est du bruit.
func TestMouvement511Carte(t *testing.T) {
	film := m511Film(t)
	chemin := os.Getenv("MOUV511_BORNES")
	if chemin == "" {
		t.Skip("MOUV511_BORNES absent")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	noms := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	type ligne struct {
		nom             string
		ti35            int
		desync          int
		i21             float64
		paquets, pleins int
	}
	var out []ligne
	for _, n := range noms {
		for low := 10; low <= 15; low++ {
			m, errB := m511Balayer(m511Contexte(film, cat.Maps[n]), 1, low)
			if errB != nil {
				t.Logf("  %s : %v", n, errB)
				continue
			}
			out = append(out, ligne{nom: fmt.Sprintf("%s / idLow %d", n, low), ti35: m.ti35,
				desync: m.desync, i21: m533bPart(m.etalon[21], m.ti35),
				paquets: m.paquets, pleins: m.pleins})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ti35 > out[j].ti35 })
	t.Logf("BALAYAGE DU CATALOGUE (chunk 1 seul, %d cartes) — les dix meilleures :", len(out))
	for i, l := range out {
		if i >= 10 {
			break
		}
		t.Logf("  %2d. %-42s ti=35 %7d · desync %5d · i21 %5.1f %% · %d paquets dont %d pleins",
			i+1, l.nom, l.ti35, l.desync, l.i21, l.paquets, l.pleins)
	}
}

// TestMouvement511Diag publie le detail d UNE marche : population de records par archetype,
// images-cles vues, paquets localises. C est le diagnostic a lire quand le compte de `ti=35`
// s effondre — il dit si la trame n est pas cadree ou si le film ne porte tout simplement pas
// les records attendus.
func TestMouvement511Diag(t *testing.T) {
	film := m511Film(t)
	chemin := os.Getenv("MOUV511_BORNES")
	nom := os.Getenv("MOUV511_CARTE")
	if chemin == "" || nom == "" {
		t.Skip("MOUV511_BORNES / MOUV511_CARTE absents")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entry, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	low := 0
	if v := os.Getenv("MOUV511_IDLOW"); v != "" {
		fmt.Sscanf(v, "%d", &low)
	}
	m, err := m511Balayer(m511Contexte(film, entry), 0, low)
	if err != nil {
		t.Fatalf("marche : %v", err)
	}
	t.Logf("MARCHE : %d paquets delta traites (dont %d a liste pleine localises) · %d vues · "+
		"%d images-cles · %d records decodes",
		m.paquets, m.pleins, m.vues, m.kf, m.records)
	tis := make([]int, 0, len(m.parTI))
	for ti := range m.parTI {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	var parts []string
	for _, ti := range tis {
		parts = append(parts, fmt.Sprintf("ti=%d : %d", ti, m.parTI[uint32(ti)])) //nolint:gosec
	}
	t.Logf("  PAR ARCHETYPE : %s", strings.Join(parts, " · "))
	var dparts []string
	for _, ti := range tis {
		dparts = append(dparts, fmt.Sprintf("ti=%d : %d", ti, m.desyncTI[uint32(ti)])) //nolint:gosec
	}
	t.Logf("  DESYNCS PAR ARCHETYPE : %s", strings.Join(dparts, " · "))
	tailles := make([]int, 0, len(m.parPaquet))
	for n := range m.parPaquet {
		tailles = append(tailles, n)
	}
	sort.Ints(tailles)
	var tparts []string
	for _, n := range tailles {
		tparts = append(tparts, fmt.Sprintf("%d rec : %d paquets", n, m.parPaquet[n]))
	}
	t.Logf("  RECORDS PAR PAQUET : %s", strings.Join(tparts, " · "))
	t.Logf("  ti=35 : %d records, %d desynchronises · etalon i0 %.1f %% · i1 %.1f %% · "+
		"i21 %.1f %% · i25 %.1f %%", m.ti35, m.desync,
		m533bPart(m.etalon[0], m.ti35), m533bPart(m.etalon[1], m.ti35),
		m533bPart(m.etalon[21], m.ti35), m533bPart(m.etalon[25], m.ti35))
}
