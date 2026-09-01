package replay

// origine_coil_research_test.go — L'HYPOTHÈSE UTILISATEUR SUR `00007ca9` : UNE BOBINE ?
//
// L'ÉNONCÉ (utilisateur, autorité gameplay) : « peut-être une bobine / fusion coil en main, ou
// un truc du genre » — un OBJET DU MONDE PORTABLE, tenu comme une arme mais qui n'occupe JAMAIS
// un emplacement d'arme.
//
// CE QUI COLLE DÉJÀ, et ce n'est pas rien : `00007ca9` est ramassé 15 fois sur deux films,
// TOUJOURS en classe ARME, et n'apparaît dans AUCUNE émission i43..i46 (0/15) — donc jamais
// dans un emplacement d'arme. Le vocabulaire du dépôt va dans le même sens :
// `internal/games/weapons/registry.go` porte QUATRE bobines — `hinf_coil_kinetic` (« UNSC
// Fusion Coil », FR « Bobine à fusion UNSC »), `hinf_coil_plasma`, `hinf_coil_shock`,
// `hinf_coil_hardlight` — toutes classées `clsEnvironmental`, c'est-à-dire objet du décor et
// non arme portée. C'est du VOCABULAIRE qui oriente, jamais une preuve.
//
// CE QUI NE COLLE PAS ENCORE, et c'est la raison de ce fichier : sur le film 1, sept des
// occurrences tombent dans les 180 premières millisecondes du match, une par joueur — la
// signature d'une DOTATION à l'apparition, pas d'un objet ramassé en jeu. Une bobine ne se
// distribue pas au spawn.
//
// LA QUESTION EST DONC : LA POPULATION EST-ELLE MIXTE ? Deux phénomènes sous un même
// identifiant, ou un seul mal compris ?
//
// SEUILS ÉCRITS AVANT LA MESURE :
//
//	K1 — SCISSION : une occurrence est de cohorte SPAWN si elle tombe à moins d'une seconde du
//	     DÉBUT DE VIE de son ramasseur, EN MATCH sinon. Le découpage des vies est celui du
//	     paquet (`equipmentLives`, trou de plus de `lifeGapUS`), pas un seuil inventé ici.
//	K2 — si les deux cohortes sont toutes deux non vides, la population EST mixte et il faut
//	     les juger séparément. Si tout tombe dans une seule, l'identifiant désigne UN
//	     phénomène.
//	K3 — RÉCURRENCE : un point de bobine est FIXE sur la carte. Si la cohorte EN MATCH se
//	     regroupe à des positions récurrentes (>= 2 prises à moins de 2 m), c'est un objet du
//	     monde ; si elle est éparpillée, ce n'en est pas un.
//	K4 — L'ARME EN MAIN NE BOUGE PAS : déjà mesuré à 0/15, re-vérifié ici par cohorte.
//
// Gardes ORIGINE_FILM (scission, sans carte) et PICKUP_FILM + PICKUP_MAP (positions).

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// coilID est l'identifiant sous test.
const coilID uint32 = 0x00007ca9

// coilSpawnUS : au-delà d'une seconde après le début de vie, un ramassage n'est plus une
// dotation d'apparition.
const coilSpawnUS = 1_000_000

// coilVies découpe les vies par slot SUR LE SEUL AXE DU TEMPS.
//
// POURQUOI PAS `equipmentLives` : celle-ci écarte les positions sans coordonnées monde
// (`!p.HasWorld`). En lecture `QuantaOnly` — la seule possible ici, le film 2 n'ayant pas de
// carte connue — AUCUNE position n'a de monde, donc elle rend zéro vie et TOUT tombe en « sans
// vie rattachable ». C'est exactement ce que la première mesure a affiché (7/7 et 8/8), et
// c'est un défaut d'instrument, pas un résultat. Le découpage ci-dessous garde le MÊME seuil
// (`lifeGapUS`) et ne regarde que slot et instant, ce dont la scission a besoin.
func coilVies(pos []filmdec.BipedPosition) map[uint32][]equipLife {
	out := map[uint32][]equipLife{}
	tri := append([]filmdec.BipedPosition(nil), pos...)
	sort.Slice(tri, func(i, j int) bool { return tri[i].TimestampUS < tri[j].TimestampUS })
	for _, p := range tri {
		v := out[p.Slot]
		if n := len(v); n > 0 && p.TimestampUS-v[n-1].to <= lifeGapUS {
			v[n-1].to = p.TimestampUS
			out[p.Slot] = v
			continue
		}
		out[p.Slot] = append(v, equipLife{from: p.TimestampUS, to: p.TimestampUS})
	}
	return out
}

// coilVieDebut rend l'instant de début de la vie du slot qui contient `at`, et si elle existe.
func coilVieDebut(lives map[uint32][]equipLife, slot uint32, at uint64) (uint64, bool) {
	for _, v := range lives[slot] {
		if at >= v.from && at <= v.to {
			return v.from, true
		}
	}
	// Hors de toute vie répliquée : on rattache à la vie la plus proche qui précède.
	best, ok := uint64(0), false
	for _, v := range lives[slot] {
		if v.from <= at && (!ok || v.from > best) {
			best, ok = v.from, true
		}
	}
	return best, ok
}

// TestCoilCohortes — K1, K2, K4. La population est-elle mixte ?
func TestCoilCohortes(t *testing.T) {
	dir := os.Getenv("ORIGINE_FILM")
	if dir == "" {
		t.Skip("ORIGINE_FILM absent : instrument de mesure sauté")
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	// QuantaOnly : la scission n'a besoin que des SLOTS et des INSTANTS. Exiger les bornes de
	// carte ferait dépendre cette mesure d'un catalogue sans rien lui apporter — et le film 2
	// n'a pas de carte connue.
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	lives := coilVies(pos)
	kf, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("images-clés illisibles : %v", err)
	}
	chg, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, spawnSetFrom(kf))
	if err != nil {
		t.Fatalf("changements d'arme illisibles : %v", err)
	}

	var spawnN, matchN, sansVie, slotSansPosition int
	var matchDelais []uint64
	var avantPremiere []float64
	slotsSpawn := map[uint32]int{}
	armeApres := 0
	for _, p := range pickups {
		if p.CatalogID != coilID {
			continue
		}
		if len(oriTenuApres(chg, p.Slot, p.TimestampUS, 0)) > 0 {
			armeApres++
		}
		debut, ok := coilVieDebut(lives, p.Slot, p.TimestampUS)
		if !ok {
			// PAS DE VIE QUI PRÉCÈDE : soit la prise arrive AVANT la première position
			// répliquée du slot (donc au tout début de sa vie — c'est une cohorte SPAWN
			// « avant réplication »), soit le slot n'a aucune position du tout (là ce serait
			// un défaut). On les sépare au lieu de les confondre.
			sansVie++
			if v := lives[p.Slot]; len(v) > 0 {
				avance := v[0].from - p.TimestampUS
				avantPremiere = append(avantPremiere, float64(avance)/1e6)
			} else {
				slotSansPosition++
			}
			continue
		}
		d := p.TimestampUS - debut
		if p.TimestampUS < debut {
			d = 0
		}
		if d <= coilSpawnUS {
			spawnN++
			slotsSpawn[p.Slot]++
			continue
		}
		matchN++
		matchDelais = append(matchDelais, d)
	}
	tot := spawnN + matchN + sansVie
	t.Logf("== HYPOTHÈSE BOBINE — SCISSION DES OCCURRENCES DE %08x · %s ==", coilID, dir)
	t.Logf("occurrences : %d", tot)
	t.Logf("  cohorte SPAWN (<= 1 s du début de vie) : %d — sur %d slot(s) distinct(s)",
		spawnN, len(slotsSpawn))
	t.Logf("  cohorte EN MATCH                       : %d", matchN)
	t.Logf("  sans vie répliquée rattachable         : %d — dont %d AVANT la première position du slot, %d sur un slot SANS aucune position",
		sansVie, len(avantPremiere), slotSansPosition)
	if len(avantPremiere) > 0 {
		sort.Float64s(avantPremiere)
		t.Logf("     avance sur la première position répliquée, en s : %v", avantPremiere)
	}
	if len(matchDelais) > 0 {
		sort.Slice(matchDelais, func(i, j int) bool { return matchDelais[i] < matchDelais[j] })
		t.Logf("  délais depuis le début de vie (cohorte EN MATCH), en s : %v",
			coilSecondes(matchDelais))
	}
	t.Logf("K4 — arme reçue en main dans les 2 s qui suivent : %d / %d", armeApres, tot)
	t.Logf("VERDICT K2 (population MIXTE : les deux cohortes non vides) : %v", spawnN > 0 && matchN > 0)
}

// coilSecondes convertit des durées en secondes arrondies, pour un log lisible.
func coilSecondes(us []uint64) []float64 {
	out := make([]float64, 0, len(us))
	for _, v := range us {
		out = append(out, float64(v)/1e6)
	}
	return out
}

// TestCoilPositionsEnMatch — K3. La cohorte EN MATCH se regroupe-t-elle à des positions fixes ?
//
// Un point de bobine ne bouge pas : si les prises « en match » reviennent aux mêmes endroits,
// l'objet est du décor. Si elles sont éparpillées, l'hypothèse perd son meilleur appui.
func TestCoilPositionsEnMatch(t *testing.T) {
	s := glResolve(t)
	pickups, _, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	lives := coilVies(oriFlat(s.pos))
	type prise struct {
		x, y, z float32
		enMatch bool
	}
	var prises []prise
	for _, p := range pickups {
		if p.CatalogID != coilID {
			continue
		}
		pos, ok := glAt(s.pos, p.Slot, p.TimestampUS)
		if !ok {
			continue
		}
		debut, okv := coilVieDebut(lives, p.Slot, p.TimestampUS)
		enMatch := okv && p.TimestampUS > debut && p.TimestampUS-debut > coilSpawnUS
		prises = append(prises, prise{pos.X, pos.Y, pos.Z, enMatch})
	}
	t.Logf("== POSITIONS DES PRISES DE %08x · %s ==", coilID, s.dir)
	for _, e := range prises {
		lbl := "SPAWN"
		if e.enMatch {
			lbl = "EN MATCH"
		}
		t.Logf("  %-8s x=%.2f y=%.2f z=%.2f", lbl, e.x, e.y, e.z)
	}
	// Regroupement à 2 m, toutes cohortes confondues puis cohorte EN MATCH seule.
	grp := func(only bool) int {
		type c struct {
			x, y, z float32
			n       int
		}
		var cs []c
		for _, e := range prises {
			if only && !e.enMatch {
				continue
			}
			hit := -1
			for i := range cs {
				if glDist(e.x, e.y, e.z, cs[i].x, cs[i].y, cs[i].z) <= 2.0 {
					hit = i
					break
				}
			}
			if hit < 0 {
				cs = append(cs, c{e.x, e.y, e.z, 1})
				continue
			}
			cs[hit].n++
		}
		rec := 0
		for _, x := range cs {
			if x.n >= 2 {
				rec++
			}
		}
		if only {
			for _, x := range cs {
				if x.n >= 2 {
					t.Logf("  POINT RÉCURRENT (en match) x=%.2f y=%.2f z=%.2f — %d prises", x.x, x.y, x.z, x.n)
				}
			}
		}
		return rec
	}
	tous := grp(false)
	enMatch := grp(true)
	t.Logf("K3 — regroupements >= 2 prises a <= 2 m : %d (toutes cohortes) · %d (cohorte EN MATCH)",
		tous, enMatch)
}

// TestCoilCandidatsAutres — SI une bobine est ramassée dans ces films, sous quel identifiant ?
//
// Le test précédent réfute l'hypothèse pour `00007ca9` (toutes ses occurrences précèdent la
// première position de leur porteur). Mais la question de l'utilisateur reste ouverte pour les
// AUTRES identifiants : un objet du monde portable aurait exactement cette signature —
// classe ARME, jamais dans un emplacement d'arme (i43..i46), et ramassé EN COURS DE MATCH à
// des positions fixes.
//
// Ce test balaie tous les identifiants de classe ARME et publie les trois colonnes qui les
// départagent. Un candidat bobine est un identifiant : jamais vu par i43..i46 ET dont les
// prises tombent en cours de vie (pas au démarrage).
func TestCoilCandidatsAutres(t *testing.T) {
	dir := os.Getenv("ORIGINE_FILM")
	if dir == "" {
		t.Skip("ORIGINE_FILM absent : instrument de mesure sauté")
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	lives := coilVies(pos)
	kf, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("images-clés illisibles : %v", err)
	}
	chg, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, spawnSetFrom(kf))
	if err != nil {
		t.Fatalf("changements d'arme illisibles : %v", err)
	}
	vues := map[uint32]bool{}
	for _, c := range chg {
		if c.Family != filmdec.NoWeaponVariant {
			vues[c.Family] = true
		}
	}
	type stat struct{ total, enMatch, avantPos int }
	par := map[uint32]*stat{}
	for _, p := range pickups {
		if !filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		e := par[p.CatalogID]
		if e == nil {
			e = &stat{}
			par[p.CatalogID] = e
		}
		e.total++
		debut, ok := coilVieDebut(lives, p.Slot, p.TimestampUS)
		switch {
		case !ok:
			e.avantPos++
		case p.TimestampUS > debut && p.TimestampUS-debut > coilSpawnUS:
			e.enMatch++
		}
	}
	ids := make([]uint32, 0, len(par))
	for id := range par {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return par[ids[i]].total > par[ids[j]].total })
	t.Logf("== CANDIDATS « OBJET DU MONDE PORTABLE » parmi les identifiants de classe ARME · %s ==", dir)
	t.Logf("%-10s %6s %8s %10s  %s", "id", "total", "enMatch", "avantPos", "vu par i43..i46 ?")
	cand := 0
	for _, id := range ids {
		e := par[id]
		v := vues[id]
		if !v && e.enMatch > 0 {
			cand++
		}
		t.Logf("%08x   %6d %8d %10d  %v", id, e.total, e.enMatch, e.avantPos, v)
	}
	t.Logf("CANDIDATS (jamais vus par i43..i46 ET pris en cours de vie) : %d", cand)
}
