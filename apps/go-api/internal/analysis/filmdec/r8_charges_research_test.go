package filmdec

// r8_charges_research_test.go — PISTE B du lot R8 : le canal des COMPOSANTS de l'entite
// EQUIPEMENT (ti=37), joint a l'IDENTITE de l'objet.
//
// CE QUI EST TESTE. `equipment_state.go` decode six composants nommes de ti=37 — dont
// `equipment-charges-remaining` (i27), `equipment-activated` (i21), `equipment-energy`
// (i24) et `equipment-creator` (i23). Son en-tete affirme depuis 2026-08-17 qu'« une
// charge qui decroit DATE un usage », et personne n'y est retourne : le canal n'a jamais
// ete JOINT a l'identite de l'objet (`EquipmentPlacement.Life` -> `GlobalID`), donc jamais
// confronte au repulseur ni au propulseur.
//
// LE TEMOIN POSITIF, ECRIT AVANT LA MESURE, et il est eliminatoire. Les instants d'usage
// du GRAPPIN sont connus et CERTAINS : `grappleLines[]` de l'artefact, decode d'un canal
// independant. Si les vies ti=37 d'identite `grapple` ne montrent AUCUNE transition a ces
// instants, le canal est disqualifie et le negatif du lot ne dependra pas des cibles. S'il
// les montre, la meme lecture sur `repulsor` / `thruster` rend leurs usages.
//
// GARDES : `R8_FILMS` (dossier des `film_chunks`), `R8_ARTIFACTS` (artefacts, pour le
// temoin), `R8_IDS` (identifiants a balayer — OBLIGATOIRE ici, un film coute cher).
// `LockProcessDecode` tenu pendant tout le decodage d'un film : les hooks sont des globaux
// de paquet.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_ARTIFACTS=<repo>/data/cache/replays/halo_infinite R8_IDS=00ba2e1c \
//	  go test ./internal/analysis/filmdec/ -run '^TestR8ChargesIdentite$' -timeout 60m -v

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r8FilmsEnv  = "R8_FILMS"
	r8BoundsEnv = "R8_BOUNDS"
)

// r8MapRange rend les BORNES DE DEQUANTIFICATION du film, identifiees SANS base de donnees :
// `DetectI0Layout` lit dans le film les largeurs d'axe et l'index de region, et le catalogue
// versionne (`map_quant_bounds.json`) dit quelles cartes portent ces largeurs. Le catalogue
// documente lui-meme cette egalite comme un CONTROLE (`MapQuantEntry.AxisWidths`) : on s'en
// sert ici a l'envers, comme d'une cle — et l'ambiguite est benigne, plusieurs cartes
// partageant les memes largeurs partagent aussi la meme AABB (canevas de Forge).
//
// SANS BORNES JUSTES, RIEN NE MARCHE : la largeur de quantification par axe en depend, donc
// la position decodee, donc l'oracle qui CONFIRME les records de creation. Mesure a l'appui
// — avec un cube unite, `00ba2e1c` rend 13 poses la ou la production en rend 537.
func r8MapEntry(t *testing.T, dir string) MapQuantEntry {
	t.Helper()
	path := os.Getenv(r8BoundsEnv)
	if path == "" {
		t.Skipf("%s absent : sans bornes de carte le decodage ne rend que des quanta", r8BoundsEnv)
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible dans %s : %v", dir, err)
	}
	var got []MapQuantEntry
	var names []string
	for name, e := range cat.Maps {
		if e.AxisWidths != lay.AxisW || e.Region != lay.Region {
			continue
		}
		if !r8HasRange(got, e.Range()) {
			got = append(got, e)
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(got) == 0 {
		t.Fatalf("aucune carte du catalogue ne porte les largeurs %v (region %d) du film %s",
			lay.AxisW, lay.Region, filepath.Base(dir))
	}
	t.Logf("%s : largeurs i0 %v region %d -> %d AABB distincte(s), cartes %v",
		filepath.Base(dir), lay.AxisW, lay.Region, len(got), names)
	return got[0]
}

func r8HasRange(all []MapQuantEntry, r Vec3Range) bool {
	for _, x := range all {
		if x.Range() == r {
			return true
		}
	}
	return false
}

// r8Families nomme les GlobalID `eqip` du corpus. SOURCE UNIQUE de ces valeurs :
// `config/titles/halo_infinite/mappings/replay_labels.toml`, section `[[equipment_objects]]`.
// Recopiees ici parce qu'un instrument de `filmdec` ne doit pas importer la couche des
// manifestes ; toute divergence se lit en comparant les deux tables.
var r8Families = map[uint32]string{
	0x7ca85adc: "repulsor",
	0x430dda48: "thruster",
	0xeef5d48d: "thruster",
	0x273fe0eb: "grapple",
	0x8c77ffe7: "grapple",
	0x2974c233: "wall",
	0x8e2dc574: "wall",
	0x528fce46: "wall_panel",
	0x686b40c9: "wall_panel",
	0x72b63d69: "sensor",
	0x72199cba: "sensor",
	0x4744d742: "threat_seeker",
	0x32d97758: "repair_field",
	0x4396db42: "shroud_screen",
	0x730dc70f: "translocator_beacon",
	0xb781197a: "powerup_overshield",
	0xe7be9f5c: "powerup_camo",
	0x0f5716ff: "grenade_spike",
	0xaada07f3: "grenade_dynamo",
	0xbcabbe43: "grenade_frag",
	0xcaaadcb0: "grenade_plasma",
}

// r8FieldNames nomme les six champs pour la sortie.
var r8FieldNames = [EquipmentFieldCount]string{
	"deployed", "activated", "creator", "energy", "energyDelay", "charges",
}

// r8LifeStat resume ce que le canal dit d'UNE vie d'objet.
type r8LifeStat struct {
	family                       string
	samples                      int
	seen, present                [EquipmentFieldCount]int
	drops                        []uint64 // instants (us) ou les charges DECROISSENT
	acts                         []uint64 // instants ou `activated` CHANGE de valeur
	firstCharge, lastCharge      uint64
	haveCharge                   bool
	creator                      map[uint64]bool
	minCharge, maxCharge, deltas uint64
}

// r8ScanFilm decode UN film et rend, par vie ti=37, ce que le canal des composants dit.
// Detient `LockProcessDecode` pendant tout le decodage.
func r8ScanFilm(t *testing.T, dir string) map[EquipmentLifeKey]*r8LifeStat {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	// LES LARGEURS D'AXE SONT UN GLOBAL DE PAQUET, et sans elles le lecteur world-object
	// dequantifie tout aux largeurs de Cliffhanger : mesure a l'appui, `00ba2e1c` rend
	// 13 poses au defaut contre 537 avec les siennes. Restauration a la sortie.
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()
	pl, pst, err := ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("poses ti=37 illisibles dans %s : %v", dir, err)
	}
	ident := map[EquipmentLifeKey]uint32{}
	for _, p := range pl {
		ident[p.Life] = p.GlobalID
	}
	samples, sst, err := ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("etat ti=37 illisible dans %s : %v", dir, err)
	}
	t.Logf("%s : poses=%d (decoupage %s, ancres=%d confirmes=%d vies=%d) identites=%d"+
		" | records=%d walked=%d broken=%d slots=%d",
		filepath.Base(dir), len(pl), pst.Calibration.Widths.String(), pst.Anchors,
		pst.Confirmed, pst.Lives, len(ident),
		sst.Records, sst.Walked, sst.Broken, sst.Slots)
	for f := 0; f < EquipmentFieldCount; f++ {
		t.Logf("  champ %-12s masque=%6d lu=%6d porteFermee=%6d",
			r8FieldNames[f], sst.WithField[f], sst.Read[f], sst.Gated[f])
	}
	return r8Aggregate(samples, ident)
}

// r8Aggregate replie les echantillons par vie et y cherche les TRANSITIONS.
func r8Aggregate(
	samples []EquipmentStateSample, ident map[EquipmentLifeKey]uint32,
) map[EquipmentLifeKey]*r8LifeStat {
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].TimestampUS < samples[j].TimestampUS
	})
	out := map[EquipmentLifeKey]*r8LifeStat{}
	prevCharge := map[EquipmentLifeKey]uint64{}
	prevAct := map[EquipmentLifeKey]uint64{}
	for _, s := range samples {
		k := EquipmentLifeKey{Slot: s.Slot, Gen: s.Gen}
		st := out[k]
		if st == nil {
			st = &r8LifeStat{family: r8FamilyOf(ident[k]), creator: map[uint64]bool{}}
			out[k] = st
		}
		st.samples++
		for f := 0; f < EquipmentFieldCount; f++ {
			if s.Seen[f] {
				st.seen[f]++
			}
			if s.Present[f] {
				st.present[f]++
			}
		}
		r8Transitions(st, s, k, prevCharge, prevAct)
	}
	return out
}

// r8Transitions met a jour les transitions d'une vie a partir d'un echantillon.
func r8Transitions(
	st *r8LifeStat, s EquipmentStateSample, k EquipmentLifeKey,
	prevCharge, prevAct map[EquipmentLifeKey]uint64,
) {
	if s.Present[EquipCreator] {
		st.creator[s.Val[EquipCreator]] = true
	}
	if s.Present[EquipCharges] {
		v := s.Val[EquipCharges]
		if !st.haveCharge {
			st.haveCharge, st.firstCharge, st.minCharge, st.maxCharge = true, v, v, v
		}
		st.lastCharge = v
		if v < st.minCharge {
			st.minCharge = v
		}
		if v > st.maxCharge {
			st.maxCharge = v
		}
		if p, ok := prevCharge[k]; ok && v < p {
			st.drops = append(st.drops, s.TimestampUS)
			st.deltas += p - v
		}
		prevCharge[k] = v
	}
	if s.Present[EquipActivated] {
		v := s.Val[EquipActivated]
		if p, ok := prevAct[k]; ok && v != p {
			st.acts = append(st.acts, s.TimestampUS)
		}
		prevAct[k] = v
	}
}

func r8FamilyOf(id uint32) string {
	if id == 0 {
		return "(sans identite)"
	}
	if f, ok := r8Families[id]; ok {
		return f
	}
	return "(inconnu)"
}

// r8FamilyTally agrege les vies par famille.
type r8FamilyTally struct {
	lives, samples, withCharge, drops, acts, dropSum int
}

func TestR8ChargesIdentite(t *testing.T) {
	dirs := r8FilmDirs(t)
	for _, dir := range dirs {
		stats := r8ScanFilm(t, dir)
		byFam := map[string]*r8FamilyTally{}
		for _, st := range stats {
			ft := byFam[st.family]
			if ft == nil {
				ft = &r8FamilyTally{}
				byFam[st.family] = ft
			}
			ft.lives++
			ft.samples += st.samples
			if st.haveCharge {
				ft.withCharge++
			}
			ft.drops += len(st.drops)
			ft.acts += len(st.acts)
			ft.dropSum += int(st.deltas)
		}
		r8LogFamilies(t, filepath.Base(dir), byFam)
	}
}

func r8LogFamilies(t *testing.T, id string, byFam map[string]*r8FamilyTally) {
	t.Helper()
	keys := make([]string, 0, len(byFam))
	for k := range byFam {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("%s — vies ti=37 par famille", id)
	t.Logf("  %-22s %6s %8s %8s %8s %8s %8s",
		"famille", "vies", "records", "avecChg", "baisses", "sommeChg", "chgActif")
	for _, k := range keys {
		f := byFam[k]
		t.Logf("  %-22s %6d %8d %8d %8d %8d %8d",
			k, f.lives, f.samples, f.withCharge, f.drops, f.dropSum, f.acts)
	}
}

// r8FilmDirs rend les dossiers de film a balayer. `R8_IDS` est OBLIGATOIRE : un film coute
// plusieurs minutes de decodage, un balayage non borne serait un piege.
func r8FilmDirs(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(r8FilmsEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", r8FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(os.Getenv(r8IDsEnv), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, filepath.Join(root, s))
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s obligatoire avec %s (un film coute cher a decoder)", r8IDsEnv, r8FilmsEnv)
	}
	return out
}
