package filmdec

// r9_ti37_identite_research_test.go — PORTE (b) du lot R9 : l'entite ti=37 PORTEE, dont
// l'identite vient des HANDLES d'i26 et non des records de creation.
//
// LE VERROU QUE CE LOT FAIT SAUTER. R8 a instruit les composants d'etat de ti=37
// (`equipment-activated`, `equipment-charges-remaining`) et les a trouves bavards — 856
// lectures de charges et 45 baisses sur `00ba2e1c` — mais son verdict est reste FAIBLE parce
// que la jointure d'IDENTITE passait par `EquipmentPlacement.Life` -> `GlobalID`, qui ne
// nomme que 217 des 1 641 vies porteuses d'un record d'etat (13 %).
//
// LA JOINTURE NEUVE. i26 `unit-equipment-component` du BIPEDE publie jusqu'a 7 handles
// `porte(1) + valeur(13) + queue(2)` — les largeurs exactes d'un slot d'entite et d'une
// generation. Un handle relie donc UNE entite ti=37 a UN bipede. Et le bipede, lui, est deja
// nomme par le canal i48 (`ScanFilmAbilityRanks`), le juge d'identite qui a tranche le
// propulseur au lot R8. **L'identite de l'objet ne vient plus de sa creation : elle vient du
// RANG DE CAPACITE de celui qui le porte.**
//
// SEUILS ECRITS AVANT LA MESURE (pre-inscription, cf. RAPPORT_R9 par. 1.2) :
//
//	appartenance  les valeurs des handles presents tombent dans la BANDE DE SLOTS ti=37 a
//	              un taux qui ecrase celui de la bande temoin. Sinon la jointure est fausse
//	              et rien de ce qui suit ne vaut.
//	couverture    > 40 % des entites ti=37 vues en etat recoivent une identite (13 % avant)
//	charges       >= 0,5 BAISSE de `charges-remaining` par entite d'identite repulseur
//	activation    ou, a defaut, une transition d'`activated` concentree sur ces entites
//	oracle        a l'instant d'une baisse, le VOISIN du porteur montre un pic au-dessus du
//	              P90 du temoin aleatoire
//
// TEMOINS POSITIFS OBLIGATOIRES : le GRAPPIN (instants certains de `ScanFilmGrappleReads`) et
// le PROPULSEUR (instants certains de l'impulsion i57 tag==1, etablie par R8) doivent, EUX,
// montrer leur baisse de charge ou leur transition d'activation. Une methode qui ne les
// retrouve pas ne prouve rien sur le repulseur — c'est ce controle qui a fait tomber la piste
// des poses `deployed` au lot R8.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9Ti37Identite$' -count=1 -timeout 180m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// r9Ent identifie UNE VIE d'entite ti=37 : la paire (slot, generation), jamais le seul slot
// (le pool reboucle et la generation ne fait que 2 bits). C'est `EquipmentLifeKey`, repris
// tel quel pour que la jointure avec les poses se fasse sans conversion.
type r9Ent = EquipmentLifeKey

// r9Owner est UNE observation de portage : ce bipede tenait cette entite a cet instant.
type r9Owner struct {
	Biped uint32
	TSUS  uint64
}

// r9IdentStats porte les denominateurs de la jointure — sans eux, un taux ne se juge pas.
type r9IdentStats struct {
	emissions, entries, present int
	dansBande, horsBande        int
	entsI26, entsEtat           int
	entsJointes, entsNommees    int
}

// r9CollectOwners lit i26 et rend, par entite, la liste des portages observes. Il compte au
// passage l'APPARTENANCE des valeurs a la bande de slots ti=37 : c'est le controle qui dit si
// le handle designe bien une entite d'equipement.
func r9CollectOwners(
	t *testing.T, dir string, band map[uint32]bool, st *r9IdentStats,
) map[r9Ent][]r9Owner {
	t.Helper()
	em, err := ScanFilmUnitEquipment(dir)
	if err != nil {
		t.Fatalf("balayage i26 : %v", err)
	}
	out := map[r9Ent][]r9Owner{}
	for _, e := range em {
		st.emissions++
		for _, en := range e.Read.Entries {
			st.entries++
			if !en.Present {
				continue
			}
			st.present++
			if band[en.Val] {
				st.dansBande++
			} else {
				st.horsBande++
			}
			k := r9Ent{Slot: en.Val, Gen: en.Tail}
			out[k] = append(out[k], r9Owner{Biped: e.Slot, TSUS: e.TimestampUS})
		}
	}
	st.entsI26 = len(out)
	return out
}

// r9Identify attribue a chaque entite le RANG DE CAPACITE majoritaire de ses porteurs, lu
// dans la vie du porteur et anterieurement a l'observation (meme regle qu'au lot R8).
func r9Identify(
	owners map[r9Ent][]r9Owner, ranks []AbilityRank, lives map[uint32][]r8LifeSpan,
) map[r9Ent]int {
	out := map[r9Ent]int{}
	for ent, obs := range owners {
		votes := map[int]int{}
		for _, o := range obs {
			if r := r8RankInLife(ranks, lives, o.Biped, o.TSUS); r >= 0 {
				votes[r]++
			}
		}
		best, bestN := -1, 0
		for r, n := range votes {
			if n > bestN {
				best, bestN = r, n
			}
		}
		out[ent] = best
	}
	return out
}

// r9EntSeries regroupe les records d'etat d'une entite, tries par instant.
func r9EntSeries(states []EquipmentStateSample) map[r9Ent][]EquipmentStateSample {
	out := map[r9Ent][]EquipmentStateSample{}
	for _, s := range states {
		k := r9Ent{Slot: s.Slot, Gen: s.Gen}
		out[k] = append(out[k], s)
	}
	for _, v := range out {
		sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
	}
	return out
}

// r9Drop est UNE baisse de charges (ou une transition d'activation) datee et attribuee.
type r9Drop struct {
	Ent      r9Ent
	TSUS     uint64
	De, Vers uint64
	Fam      string // identite par la POSE (canal de R8)
	Rang     int    // identite par le HANDLE i26 -> rang i48 du porteur (canal de ce lot)
	Createur uint64 // valeur du champ `equipment-creator-component` (R(5)), -1 si absente
	HasCrea  bool
	Champ    EquipmentField
}

// r9FindDrops parcourt la serie d'une entite et rend ses baisses de charge et ses
// transitions d'activation. Une porte fermee (`Present` faux) n'est PAS une valeur nulle :
// elle est sautee, sans quoi on fabriquerait des transitions qui n'existent pas.
func r9FindDrops(
	ent r9Ent, serie []EquipmentStateSample, fam string, rang int,
) (drops, acts []r9Drop) {
	prevC, hasC := uint64(0), false
	prevA, hasA := uint64(0), false
	crea, hasCrea := uint64(0), false
	mk := func(ts, de, vers uint64, f EquipmentField) r9Drop {
		return r9Drop{Ent: ent, TSUS: ts, De: de, Vers: vers, Fam: fam, Rang: rang,
			Createur: crea, HasCrea: hasCrea, Champ: f}
	}
	for _, s := range serie {
		if s.Seen[EquipCreator] && s.Present[EquipCreator] {
			crea, hasCrea = s.Val[EquipCreator], true
		}
		if s.Seen[EquipCharges] && s.Present[EquipCharges] {
			v := s.Val[EquipCharges]
			if hasC && v < prevC {
				drops = append(drops, mk(s.TimestampUS, prevC, v, EquipCharges))
			}
			prevC, hasC = v, true
		}
		if s.Seen[EquipActivated] && s.Present[EquipActivated] {
			v := s.Val[EquipActivated]
			if hasA && v != prevA {
				acts = append(acts, mk(s.TimestampUS, prevA, v, EquipActivated))
			}
			prevA, hasA = v, true
		}
	}
	return drops, acts
}

// r9RangAgg agrege, par rang de capacite, ce que les entites de ce rang ont montre.
type r9RangAgg struct {
	ents, records, avecCharge, baisses, transitions int
}

func TestR9Ti37Identite(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9Ti37OneFilm(t, dir)
	}
}

func r9Ti37OneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	speeds := r8BuildSpeeds(pos)
	lives := r8Lives(speeds)
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles : %v", err)
	}
	var st r9IdentStats
	band := worldObjectSlotBand(dir, CountFilmChunks(dir), EquipmentTypeIndex)
	owners := r9CollectOwners(t, dir, band, &st)
	ident := r9Identify(owners, ranks, lives)
	pl, _, err := ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("poses ti=37 illisibles : %v", err)
	}
	fam := map[r9Ent]string{}
	for _, p := range pl {
		fam[p.Life] = r8FamilyOf(p.GlobalID)
	}
	states, sstat, err := ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("etat ti=37 illisible : %v", err)
	}
	series := r9EntSeries(states)
	st.entsEtat = len(series)
	r9LogJointure(t, dir, st, sstat, band, series, ident)
	named := r9CountNamed(series, fam)
	t.Logf("  identite par POSE (canal de R8) : %d vies nommees sur %d", named, len(series))
	ctx := r9Ctx{series: series, ident: ident, fam: fam, owners: owners,
		speeds: speeds, pos: r8PosBySlot(pos)}
	ctx.r9LogParRang(t)
}

// r9CountNamed compte les vies d'etat que le canal des POSES sait nommer.
func r9CountNamed(series map[r9Ent][]EquipmentStateSample, fam map[r9Ent]string) int {
	n := 0
	for ent := range series {
		if f, ok := fam[ent]; ok && f != "" && f != "(sans identite)" {
			n++
		}
	}
	return n
}

// r9LogJointure publie LES DENOMINATEURS de la jointure : appartenance des handles, taux de
// recouvrement entre les entites vues en i26 et celles vues en etat, couverture d'identite.
func r9LogJointure(t *testing.T, dir string, st r9IdentStats, sstat EquipmentStateStats,
	band map[uint32]bool, series map[r9Ent][]EquipmentStateSample, ident map[r9Ent]int) {
	t.Helper()
	for ent := range series {
		if r, ok := ident[ent]; ok {
			st.entsJointes++
			if r >= 0 {
				st.entsNommees++
			}
		}
	}
	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}
	t.Logf("%s : i26 emissions=%d entrees=%d presentes=%d | bande ti=37 : %d slots",
		filepath.Base(dir), st.emissions, st.entries, st.present, len(band))
	t.Logf("  APPARTENANCE des handles : dansBande=%d (%.1f %%) horsBande=%d",
		st.dansBande, pct(st.dansBande, st.present), st.horsBande)
	t.Logf("  etat ti=37 : records=%d marches=%d cassees=%d | lectures charges=%d activated=%d",
		sstat.Records, sstat.Walked, sstat.Broken, sstat.Read[EquipCharges], sstat.Read[EquipActivated])
	t.Logf("  entites : i26=%d etat=%d | jointes=%d (%.1f %% de l'etat) NOMMEES=%d (%.1f %%)",
		st.entsI26, st.entsEtat, st.entsJointes, pct(st.entsJointes, st.entsEtat),
		st.entsNommees, pct(st.entsNommees, st.entsEtat))
}

// r9Ctx porte le contexte d'un film (regle des <= 5 parametres : la table croisee a besoin de
// six sources, elles voyagent ensemble).
type r9Ctx struct {
	series map[r9Ent][]EquipmentStateSample
	ident  map[r9Ent]int    // identite par HANDLE i26 -> rang i48 du porteur
	fam    map[r9Ent]string // identite par POSE (canal de R8)
	owners map[r9Ent][]r9Owner
	speeds r8SpeedIndex
	pos    map[uint32][]BipedPosition
}

// r9Key nomme une entite : la famille de la POSE si elle est connue, sinon le rang i48 du
// porteur, sinon « ? ». Les deux canaux sont publies COTE A COTE et jamais fondus : ils ne
// couvrent pas les memes vies, et melanger deux couvertures fabriquerait un taux faux.
func (c r9Ctx) r9Key(ent r9Ent) string {
	if f, ok := c.fam[ent]; ok && f != "" && f != "(sans identite)" {
		return f
	}
	if r, ok := c.ident[ent]; ok && r >= 0 {
		return fmt.Sprintf("(rang i48 %d)", r)
	}
	return "?"
}

// r9LogParRang publie LE TABLEAU DECISIF : par identite, combien d'entites, combien de
// records d'etat, combien de BAISSES de charge et de transitions d'activation. Le GRAPPIN et
// le PROPULSEUR y sont les TEMOINS POSITIFS : s'ils ne montrent rien, la methode est muette et
// le zero du repulseur ne dit rien.
func (c r9Ctx) r9LogParRang(t *testing.T) {
	t.Helper()
	agg := map[string]*r9RangAgg{}
	var allDrops []r9Drop
	for ent, serie := range c.series {
		k := c.r9Key(ent)
		a := agg[k]
		if a == nil {
			a = &r9RangAgg{}
			agg[k] = a
		}
		a.ents++
		a.records += len(serie)
		for _, s := range serie {
			if s.Seen[EquipCharges] && s.Present[EquipCharges] {
				a.avecCharge++
			}
		}
		rang := -2
		if r, ok := c.ident[ent]; ok {
			rang = r
		}
		d, ac := r9FindDrops(ent, serie, k, rang)
		a.baisses += len(d)
		a.transitions += len(ac)
		allDrops = append(allDrops, d...)
	}
	keys := make([]string, 0, len(agg))
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("  par IDENTITE d'objet (pose ; a defaut rang i48 du porteur)")
	t.Logf("    %-22s %8s %9s %11s %9s %12s", "identite", "entites", "records", "avecCharge",
		"BAISSES", "transitActiv")
	for _, k := range keys {
		a := agg[k]
		t.Logf("    %-22s %8d %9d %11d %9d %12d", k, a.ents, a.records, a.avecCharge,
			a.baisses, a.transitions)
	}
	c.r9LogDrops(t, allDrops)
}

// r9LogDrops detaille les baisses de charge : quelle identite, quel createur, quel porteur, et
// — pour la question propre au repulseur — ce que montrent le PORTEUR et son VOISIN.
func (c r9Ctx) r9LogDrops(t *testing.T, drops []r9Drop) {
	t.Helper()
	sort.SliceStable(drops, func(i, j int) bool { return drops[i].TSUS < drops[j].TSUS })
	parFam := map[string][]float64{}
	voisParFam := map[string][]float64{}
	for i, d := range drops {
		p := r9OwnerAt(c.owners[d.Ent], d.TSUS)
		pk := 0.0
		if p != 0 {
			pk, _ = c.speeds.peak(p, d.TSUS, r8PeakWindowUS)
			parFam[d.Fam] = append(parFam[d.Fam], pk)
			if vp, ok := r9NeighbourPeak(c.pos, c.speeds, p, d.TSUS); ok {
				voisParFam[d.Fam] = append(voisParFam[d.Fam], vp)
			}
		}
		if i < 50 {
			t.Logf("    baisse ent=(%d,%d) t=%8.2fs %d->%d id=%-20s createur=%d/%v"+
				" porteurI26=%-5d pic=%.2f",
				d.Ent.Slot, d.Ent.Gen, float64(d.TSUS)/1e6, d.De, d.Vers, d.Fam,
				d.Createur, d.HasCrea, p, pk)
		}
	}
	keys := make([]string, 0, len(parFam))
	for k := range parFam {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("    ORACLE aux instants de baisse (pic m/s, porteur connu par i26 seulement)")
	for _, k := range keys {
		t.Logf("      %-22s n=%-4d porteur med=%.2f p90=%.2f | voisin n=%-4d med=%.2f p90=%.2f",
			k, len(parFam[k]), r8Quantile(parFam[k], 0.5), r8Quantile(parFam[k], 0.9),
			len(voisParFam[k]), r8Quantile(voisParFam[k], 0.5), r8Quantile(voisParFam[k], 0.9))
	}
}

// r9OwnerAt rend le bipede qui portait l'entite au plus pres de `at` (0 si aucun).
func r9OwnerAt(obs []r9Owner, at uint64) uint32 {
	best, bestD, ok := uint32(0), uint64(0), false
	for _, o := range obs {
		d := r8AbsDiff(o.TSUS, at)
		if !ok || d < bestD {
			best, bestD, ok = o.Biped, d, true
		}
	}
	if !ok {
		return 0
	}
	return best
}

// r9NeighbourPeak rend le pic de vitesse du bipede le plus proche du porteur (hors lui-meme)
// dans le rayon de la question du repulseur.
func r9NeighbourPeak(
	pos map[uint32][]BipedPosition, speeds r8SpeedIndex, porteur uint32, at uint64,
) (float64, bool) {
	me, ok := r8NearestAt(pos[porteur], at)
	if !ok {
		return 0, false
	}
	best, found := 0.0, false
	for slot, list := range pos {
		if slot == porteur {
			continue
		}
		o, ok := r8NearestAt(list, at)
		if !ok || r8Dist2(float64(me.X), float64(me.Y), float64(o.X), float64(o.Y)) >
			r8NeighbourRadiusFilmM {
			continue
		}
		if p, n := speeds.peak(slot, at, r8PeakWindowUS); n > 0 && (!found || p > best) {
			best, found = p, true
		}
	}
	return best, found
}
