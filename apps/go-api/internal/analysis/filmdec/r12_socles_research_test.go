package filmdec

// r12_socles_research_test.go — LE SOCLE : ses REAPPARITIONS, datees par les creations ti=37.
//
// L'INDICE. Le releve Theater note que le socle du repulseur porte une JAUGE DE CHARGEMENT
// VISIBLE, et il date trois reapparitions : 4:49, 5:56, 9:14. Si l'objet reapparait, le film
// doit porter un RECORD DE CREATION de l'entite d'equipement a cet instant.
//
// CE QUE CET INSTRUMENT MESURE, ET CE QU'IL NE MESURE PAS. Il date les CREATIONS d'entites
// d'archetype ti=37 dont le bloc `object-multiplayer-properties` porte le GlobalID de l'`eqip`
// du repulseur (`0x7ca85adc`, `replay_labels.toml` `[[equipment_objects]]`). Il ne dit RIEN de
// la jauge elle-meme : une reapparition datee n'est pas un etat de rechargement.
//
// LES BORNES METRIQUES SONT PASSEES ARBITRAIRES, ET C'EST LEGITIME ICI.
// `ScanFilmEquipmentCreations` exige un `*Vec3Range` — mais il ne s'en sert QUE pour convertir
// les quanta en metres. Les LARGEURS de bits, elles, viennent de `WorldObjectPrecision`, pose
// depuis le layout du film par `r12Prepare`. Cet instrument ne publie AUCUNE position : il ne
// publie que des INSTANTS et des identifiants. Les X/Y/Z rendus sont donc faux et ne sont
// jamais lus — c'est ecrit ici pour qu'aucun lecteur ne les reprenne.
//
// GARDES : `R12_FILMS`, `R12_IDS`. Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE :
//
//	CGO_ENABLED=0 R12_FILMS=<repo>/data/cache/film_chunks R12_IDS=215e7022 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR12Socles$' -count=1 -timeout 60m -v

import (
	"sort"
	"testing"
)

// r12EqipRepulseur : le GlobalID du tag `eqip` du repulseur. Source :
// `config/titles/halo_infinite/mappings/replay_labels.toml`, `[[equipment_objects]]`
// `id = "0x7ca85adc"`, `family = "repulsor"`, `kind = "carried"`. C'est l'OBJET ; le
// `jpt! 0x07104b31` du kill est son EFFET DE DEGAT, un tag different (cf. par. 1.5 du rapport).
const r12EqipRepulseur = uint32(0x7ca85adc)

// r12AncresSocle : les reapparitions au socle relevees au Theater, en temps de FILM.
var r12AncresSocle = []r12Ancre{
	{"S1", 4*60000 + 49000},
	{"S2", 5*60000 + 56000},
	{"S3", 9*60000 + 14000},
}

func TestR12Socles(t *testing.T) {
	for _, dir := range r12FilmDirs(t) {
		r12SoclesOneFilm(t, dir)
	}
}

func r12SoclesOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	// Bornes ARBITRAIRES : voir l'en-tete. Aucune position n'est publiee par cet instrument.
	bidon := Vec3Range{{-1000, 1000}, {-1000, 1000}, {-1000, 1000}}
	cre, st, err := ScanFilmEquipmentCreations(dir, &bidon)
	if err != nil {
		t.Fatalf("%s : creations ti=37 illisibles : %v", s.id, err)
	}
	t.Logf("=== FILM %s — CREATIONS ti=37 ===", s.id)
	t.Logf("  %d creations lues, %d slots dans la bande (stats : %+v)", len(cre), st.Slots, st)

	parID := map[uint32][]int64{}
	sansID := 0
	for _, c := range cre {
		if !c.MPPPresent[MPPWord32] {
			sansID++
			continue
		}
		id := uint32(c.MPPVal[MPPWord32])
		parID[id] = append(parID[id], s.ms(c.TimestampUS))
	}
	t.Logf("  %d creations sans identifiant MPP", sansID)
	ids := make([]uint32, 0, len(parID))
	for id := range parID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return len(parID[ids[a]]) > len(parID[ids[b]]) })
	for _, id := range ids {
		v := parID[id]
		sort.Slice(v, func(a, b int) bool { return v[a] < v[b] })
		mark := ""
		if id == r12EqipRepulseur {
			mark = "   <<<< REPULSEUR (eqip)"
		}
		var txt string
		for i, ms := range v {
			if i >= 30 {
				break
			}
			txt += r12MMSS(ms) + " "
		}
		t.Logf("    id=%08x  n=%-4d %s%s", id, len(v), txt, mark)
	}

	rep := parID[r12EqipRepulseur]
	if len(rep) == 0 {
		t.Logf("  MESURE B3 : aucune creation portant %08x — le canal ne date aucune "+
			"reapparition sur ce film", r12EqipRepulseur)
		return
	}
	sort.Slice(rep, func(a, b int) bool { return rep[a] < rep[b] })
	t.Logf("  MESURE B3 — appariement des reapparitions au socle (tolerance +/- 5 s) :")
	hit := r12Pair(t, r12AncresSocle, rep, 5000, 0)
	dec := r12Pair(t, r12AncresSocle, rep, 5000, 30000)
	t.Logf("  MESURE B3 — VERDICT : %d/%d apparies, temoin decale +30 s : %d/%d",
		hit, len(r12AncresSocle), dec, len(r12AncresSocle))
	t.Logf("  (rappel : les ramassages du releve sont P1 3:48, P2 4:55, P3 8:14, P4 9:50)")
	_ = r12Pair(t, r12AncresPrise, rep, 5000, 0)
}
