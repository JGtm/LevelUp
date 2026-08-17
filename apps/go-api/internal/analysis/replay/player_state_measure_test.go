package replay

// player_state_measure_test.go — INSTRUMENT DE MESURE des items qui ont besoin du PONT entre
// le film et les joueurs : B.0.4 (le temps mort), P.0.3 (l'arme en main image par image) et
// P.0.4 (la seconde source de visee). Phase 0 des lots B et P.
//
// POURQUOI CET INSTRUMENT VIT DANS `replay` ET NON DANS `filmdec`. Les trois items comparent
// une donnee de `filmdec` a une donnee qui n'existe qu'ici : les VIES de bipede decoupees dans
// les trajectoires, le fil des morts qui les NOMME (`lives.go`), et le flux des tirs rattache a
// son tireur. Les porter dans `filmdec` obligerait a y recopier tout ce pont ; les porter ici
// ne coute qu'un import.
//
// LES SEUILS, ECRITS AVANT LA MESURE (D13) :
//
//	B.0.4  fenetre `Active` du compte a rebours de reapparition (ti=5 i1) contre le delai
//	       mort -> reapparition suivante, a +/- 1 s, sur >= 90 % des morts. Plus le temps mort
//	       cumule par equipe et par match.
//	P.0.3  au frame de chaque TIR, l'arme en main du tireur (i43-i46, `World.HeldWeapon`) est
//	       de la meme FAMILLE que l'arme du tir, sur >= 90 % des tirs. Plus la latence mediane
//	       d'un changement d'arme et la couverture obtenue.
//	P.0.4  quand la visee du CORPS (i21, `BipedPosition.YawRaw`) et celle du JOUEUR (ti=5 i17)
//	       tombent a <= 100 ms l'une de l'autre sur le meme joueur, |delta cap| <= 5° sur
//	       >= 90 % des paires. Plus la couverture AJOUTEE sur les points sans cap.
//
// LE PONT ti=5 -> BIPEDE N'EST PAS SUPPOSE, IL EST MESURE. Rien ne dit a priori que le slot 52
// de l'entite joueur porte le meme joueur que le slot 513 du bipede. La seule jointure
// disponible sans base est TEMPORELLE : le compte a rebours de reapparition d'une entite joueur
// s'active quand SON joueur meurt, et les vies de bipede se terminent par une mort datee. On
// apparie donc chaque slot ti=5 au slot de bipede dont les fins de vie coincident le mieux avec
// ses fenetres actives, et on PUBLIE la qualite de cet appariement — s'il est faible, les items
// qui en dependent sont negatifs, et c'est le resultat.
//
// LECTURE SEULE, garde par PLAYER_FILM, saute partout ailleurs (CI comprise). UN SEUL film par
// processus (D17).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 PLAYER_FILM=C:/.../data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run '^TestPlayerBridgePhase0$' -timeout 30m -v

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	playerFilmEnv = "PLAYER_FILM"
	playerOutEnv  = "PLAYER_OUT"
	// psPairWindowUS : ecart maximal entre une lecture de visee du corps et une lecture de
	// visee du joueur pour que les deux forment une paire (100 ms, seuil du plan P.0.4).
	psPairWindowUS = 100_000
	// psShotWindowUS : anteriorite maximale d'une lecture d'arme en main pour valoir « arme du
	// tireur au moment du tir ». Deux secondes : le rattachement des tirs du rejeu se joue a
	// ~120 ms, mais le canal i43-i46 n'est transmis qu'au CHANGEMENT d'arme — exiger une
	// lecture dans la meme frame mesurerait la frequence de transmission, pas l'accord.
	psShotWindowUS = 2_000_000
)

func TestPlayerBridgePhase0(t *testing.T) {
	dir := os.Getenv(playerFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", playerFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	in := psLoad(t, dir)
	psLogInputs(t, in)
	psHeldWeapon(t, in)
	psAimPairs(t, in)
	psRespawn(t, in)
	psDump(t, in)
}

// psInputs porte tout ce que le film a rendu, une seule lecture par flux.
type psInputs struct {
	dir, short string
	pos        []filmdec.BipedPosition
	shots      []filmdec.FireEvent
	held       []filmdec.HeldWeaponSample
	player     []filmdec.GameEntityRecord
	chain      filmdec.GameChainStats
	deaths     []Death
	lives      []lifeSpan
	// own porte le pont slot de bipede -> joueur, construit par le chemin de PRODUCTION.
	own OwnerReport
}

// psLoad lit le film une fois pour toutes. Chaque flux manquant est DIT, jamais remplace.
func psLoad(t *testing.T, dir string) psInputs {
	t.Helper()
	in := psInputs{dir: dir, short: filepath.Base(strings.TrimRight(filepath.Clean(dir), string(filepath.Separator)))}
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	// AUCUNE BORNE DE CARTE N EST DEMANDEE, et c est un choix : les trois items mesures ici ne
	// comparent que des INSTANTS, des SLOTS et des CAPS. Exiger les bornes du BSP obligerait a
	// resoudre le catalogue de la carte pour chaque film du corpus, sans qu une seule
	// coordonnee monde n entre dans un seul des trois calculs. Contrepartie ASSUMEE et dite :
	// sans bornes, le filtre de vitesse est inoperant, donc une position aberrante n est plus
	// ecartee — elle peut allonger une vie, jamais en fabriquer une.
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions de bipede illisibles : %v", err)
	}
	in.pos = pos
	if in.shots, err = filmdec.ScanFilmFireEvents(dir); err != nil {
		t.Logf("tirs illisibles : %v", err)
	}
	if in.deaths, err = ScanFilmDeaths(dir); err != nil {
		t.Logf("fil des morts illisible : %v", err)
	}
	recs, held, st, err := filmdec.ScanFilmGameEntitiesChain(dir)
	if err != nil {
		t.Fatalf("chaine sequentielle impossible : %v", err)
	}
	in.player, in.held, in.chain = recs, held, st
	sorted := append([]filmdec.BipedPosition(nil), pos...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	in.pos = sorted
	tracks := indexBySlot(sorted)
	in.lives = buildLifeSpans(tracks)
	var idx PlayerIndexTable
	if len(in.deaths) > 0 {
		if raw, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(in.deaths)); err == nil {
			idx, _ = injectiveOrEmpty(raw)
		}
	}
	in.own = buildOwners(tracks, in.deaths, idx, fireRefs(in.shots))
	return in
}

func psLogInputs(t *testing.T, in psInputs) {
	t.Helper()
	t.Logf("FILM %s · positions %d · tirs %d · morts %d · vies %d · slots nommes %d "+
		"(vies nommees %d / %d)", in.short, len(in.pos), len(in.shots), len(in.deaths),
		len(in.lives), len(in.own.Owner), in.own.DeathsNamed, in.own.LivesTotal)
	t.Logf("CHAINE · paquets %d dont propres %d · records de bipede confirmes %d dont porteurs "+
		"d une identite d arme %d · echantillons d arme en main %d · records ti=5 %d",
		in.chain.Packets, in.chain.PacketsClean, in.chain.BipedRecords, in.chain.HeldWeaponReads,
		len(in.held), in.chain.PlayerRecords)
}

// psHeldWeapon mesure P.0.3 : au frame de chaque tir, l'arme en main du tireur est-elle de la
// famille du tir ?
//
// LE TIREUR EST DESIGNE PAR L'EVENEMENT, PAS DEDUIT. `FireEvent.FilmIndex` porte l'index de
// joueur ECRIT PAR LE FILM ; le pont `own.Owner` donne le meme index pour un slot de bipede.
// La comparaison ne repose donc sur aucune heuristique de proximite.
func psHeldWeapon(t *testing.T, in psInputs) {
	t.Helper()
	bySlot := map[uint32][]filmdec.HeldWeaponSample{}
	for _, h := range in.held {
		bySlot[h.Slot] = append(bySlot[h.Slot], h)
	}
	for s := range bySlot {
		sort.Slice(bySlot[s], func(i, j int) bool {
			return bySlot[s][i].TimestampUS < bySlot[s][j].TimestampUS
		})
	}
	slotsOfIndex := map[int][]uint32{}
	for slot, idx := range in.own.Owner {
		slotsOfIndex[idx] = append(slotsOfIndex[idx], slot)
	}
	var total, withHeld, agree int
	changes := psWeaponChanges(bySlot)
	for _, sh := range in.shots {
		if sh.WeaponID == 0 {
			continue
		}
		total++
		fam, ok := psHeldAt(bySlot, slotsOfIndex[sh.FilmIndex], sh.TimestampUS)
		if !ok {
			continue
		}
		withHeld++
		if fam == uint32(sh.WeaponID>>32) {
			agree++
		}
	}
	t.Logf("P.0.3 ARME EN MAIN · tirs %d · tirs avec une arme en main connue %d (%.1f %%) "+
		"· ACCORD famille %d (%.1f %% des tirs couverts, %.1f %% de tous les tirs) "+
		"· seuil 90 %% · %s", total, withHeld, psPct(withHeld, total), agree,
		psPct(agree, withHeld), psPct(agree, total), psVerdict(agree, withHeld, 90))
	t.Logf("P.0.3 CHANGEMENTS D ARME · %d transitions relevees sur %d slots · latence mediane "+
		"entre deux lectures du meme slot %.2f s", changes.n, len(bySlot), changes.medianS)
	psHeldWeaponCensus(t, in)
}

// psHeldWeaponCensus dit POURQUOI le canal rend ce qu il rend : sur les records de bipede
// certains, les quatre composants d identite d arme (i43-i46) sont-ils seulement ANNONCES ?
// Un canal jamais annonce et un canal annonce mais illisible appellent des conclusions
// opposees, et seul ce recensement les separe.
func psHeldWeaponCensus(t *testing.T, in psInputs) {
	t.Helper()
	var parts []string
	for i := 40; i <= 47; i++ {
		parts = append(parts, fmt.Sprintf("i%d:%d", i, in.chain.BipedMask[i]))
	}
	top := 0
	for i := 0; i < len(in.chain.BipedMask); i++ {
		if in.chain.BipedMask[i] > top {
			top = in.chain.BipedMask[i]
		}
	}
	t.Logf("P.0.3 ANNONCES AU MASQUE (records de bipede certains %d, composant le plus annonce "+
		"%d fois) · %s", in.chain.BipedRecords, top, strings.Join(parts, " "))
}

// psWeaponChangeStats resume les transitions d'arme en main.
type psWeaponChangeStats struct {
	n       int
	medianS float64
}

// psWeaponChanges compte les transitions et l'ecart median entre deux lectures d'un slot —
// c'est la CADENCE du canal, la grandeur qui borne toute latence mesurable.
func psWeaponChanges(bySlot map[uint32][]filmdec.HeldWeaponSample) psWeaponChangeStats {
	var gaps []float64
	n := 0
	for _, hs := range bySlot {
		for i := 1; i < len(hs); i++ {
			gaps = append(gaps, float64(hs[i].TimestampUS-hs[i-1].TimestampUS)/1e6)
			if hs[i].Family != hs[i-1].Family {
				n++
			}
		}
	}
	sort.Float64s(gaps)
	med := 0.0
	if len(gaps) > 0 {
		med = gaps[len(gaps)/2]
	}
	return psWeaponChangeStats{n: n, medianS: med}
}

// psHeldAt rend la derniere famille d'arme lue sur l'un des slots du joueur avant `at`.
func psHeldAt(bySlot map[uint32][]filmdec.HeldWeaponSample, slots []uint32, at uint64) (uint32, bool) {
	var best uint64
	var fam uint32
	found := false
	for _, s := range slots {
		hs := bySlot[s]
		i := sort.Search(len(hs), func(k int) bool { return hs[k].TimestampUS > at })
		if i == 0 {
			continue
		}
		h := hs[i-1]
		if at-h.TimestampUS > psShotWindowUS {
			continue
		}
		if !found || h.TimestampUS > best {
			best, fam, found = h.TimestampUS, h.Family, true
		}
	}
	return fam, found
}

// psAimPairs mesure P.0.4 : la visee du JOUEUR (ti=5 i17, cubemap 19 bits) s'accorde-t-elle
// avec celle du CORPS (i21, cap quantifie sur 12 bits) quand les deux tombent a <= 100 ms ?
//
// LE PONT SLOT ti=5 -> SLOT DE BIPEDE EST CELUI QUE `psBridge` MESURE. Sans lui, comparer deux
// caps reviendrait a comparer deux joueurs au hasard — et le resultat serait un accord de 5 %,
// c'est-a-dire le hasard, ce qui ne dirait rien du canal.
func psAimPairs(t *testing.T, in psInputs) {
	t.Helper()
	bridge, quality := psBridge(t, in)
	if len(bridge) == 0 {
		t.Logf("P.0.4 VISEE · aucun pont entite joueur -> bipede : item NON MESURABLE sur ce film")
		return
	}
	byBiped := map[uint32][]filmdec.BipedPosition{}
	for _, p := range in.pos {
		if p.HasYaw {
			byBiped[p.Slot] = append(byBiped[p.Slot], p)
		}
	}
	var pairs, within int
	var deltas []float64
	for _, r := range in.player {
		if r.TI != filmdec.PlayerEngineTypeIndex || !r.PlayerSeen[filmdec.PlayerControlAiming] {
			continue
		}
		if !r.PlayerPresent[filmdec.PlayerControlAiming] || len(r.PlayerVal[filmdec.PlayerControlAiming]) == 0 {
			continue
		}
		bslot, ok := bridge[r.Slot]
		if !ok {
			continue
		}
		p, ok := psNearestYaw(byBiped[bslot], r.TimestampUS)
		if !ok {
			continue
		}
		v, ok := filmdec.DecodeAimVectorChecked(uint32(r.PlayerVal[filmdec.PlayerControlAiming][0]), 19)
		if !ok {
			continue
		}
		body, ok := p.AimHeadingDeg()
		if !ok {
			continue
		}
		pairs++
		d := psAngleDelta(psHeadingDeg(v), float64(body))
		deltas = append(deltas, d)
		if d <= 5 {
			within++
		}
	}
	sort.Float64s(deltas)
	med := 0.0
	if len(deltas) > 0 {
		med = deltas[len(deltas)/2]
	}
	t.Logf("P.0.4 VISEE · pont %d entites joueur (qualite %s) · paires a <= 100 ms %d "+
		"· |delta cap| <= 5° %d (%.1f %%) · ecart median %.1f° · seuil 90 %% · %s",
		len(bridge), quality, pairs, within, psPct(within, pairs), med,
		psVerdict(within, pairs, 90))
	noYaw := 0
	for _, p := range in.pos {
		if !p.HasYaw {
			noYaw++
		}
	}
	t.Logf("P.0.4 COUVERTURE · points sans cap de corps %d / %d (%.1f %%) · lectures de visee "+
		"joueur disponibles %d — couverture AJOUTEE au mieux %.2f %%", noYaw, len(in.pos),
		psPct(noYaw, len(in.pos)), pairs, psPct(pairs, noYaw))
}

// psHeadingDeg projette un vecteur unitaire monde sur le cap [0,360[, meme convention que
// `BipedPosition.AimHeadingDeg` (atan2(Y, X) des positions dequantifiees).
func psHeadingDeg(v [3]float32) float64 {
	d := math.Atan2(float64(v[1]), float64(v[0])) * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

// psNearestYaw rend la position de bipede la plus proche dans le temps, si elle tombe dans la
// fenetre d'appariement.
func psNearestYaw(ps []filmdec.BipedPosition, at uint64) (filmdec.BipedPosition, bool) {
	best, found := filmdec.BipedPosition{}, false
	var bd uint64 = psPairWindowUS + 1
	for _, p := range ps {
		d := at - p.TimestampUS
		if p.TimestampUS > at {
			d = p.TimestampUS - at
		}
		if d < bd {
			bd, best, found = d, p, true
		}
	}
	if bd > psPairWindowUS {
		return filmdec.BipedPosition{}, false
	}
	return best, found
}

// psBridge apparie chaque slot d'entite joueur au slot de bipede dont les fins de vie
// coincident le mieux avec ses fenetres de reapparition ACTIVES.
//
// LE RESULTAT EST RENDU AVEC SA QUALITE, jamais seul : un appariement construit sur trois
// coincidences n'est pas un pont, et le presenter comme tel ferait passer du hasard pour une
// mesure.
func psBridge(t *testing.T, in psInputs) (map[uint32]uint32, string) {
	t.Helper()
	active := map[uint32][]uint64{}
	for _, r := range in.player {
		if r.TI == filmdec.PlayerEngineTypeIndex && r.HasRespawn && r.Respawn.Active {
			active[r.Slot] = append(active[r.Slot], r.TimestampUS)
		}
	}
	ends := map[uint32][]uint64{}
	for _, l := range in.lives {
		ends[l.slot] = append(ends[l.slot], uint64(l.to))
	}
	out := map[uint32]uint32{}
	hits, tries := 0, 0
	for ps, times := range active {
		var bestSlot uint32
		bestN := 0
		for bs, es := range ends {
			n := 0
			for _, at := range times {
				for _, e := range es {
					if psAbsU(at, e) <= 5_000_000 {
						n++
						break
					}
				}
			}
			if n > bestN {
				bestN, bestSlot = n, bs
			}
		}
		tries += len(times)
		if bestN == 0 {
			continue
		}
		hits += bestN
		out[ps] = bestSlot
	}
	q := fmt.Sprintf("%d fenetres actives appariees sur %d", hits, tries)
	t.Logf("PONT ti=5 -> BIPEDE · entites joueur avec au moins une fenetre active %d "+
		"· appariees %d · %s", len(active), len(out), q)
	return out, q
}

// psRespawn mesure B.0.4 : la fenetre `Active` du compte a rebours contre le delai reel entre
// la mort et la reapparition suivante du meme joueur.
func psRespawn(t *testing.T, in psInputs) {
	t.Helper()
	var active, reads int
	for _, r := range in.player {
		if r.TI != filmdec.PlayerEngineTypeIndex || !r.HasRespawn {
			continue
		}
		reads++
		if r.Respawn.Active {
			active++
		}
	}
	gaps := psDeathToSpawnGaps(in)
	sort.Float64s(gaps)
	med := 0.0
	if len(gaps) > 0 {
		med = gaps[len(gaps)/2]
	}
	t.Logf("B.0.4 TEMPS MORT · lectures du compte a rebours %d dont ACTIVES %d (%.2f %%) "+
		"· morts du fil %d · intervalles mort -> reapparition mesurables %d · mediane %.2f s",
		reads, active, psPct(active, reads), len(in.deaths), len(gaps), med)
	if active == 0 || len(gaps) == 0 {
		t.Logf("B.0.4 VERDICT · NEGATIF : denominateur nul d'un cote au moins — l'appariement " +
			"fenetre active / delai de reapparition n'est pas mesurable sur ce film")
		return
	}
	t.Logf("B.0.4 COUVERTURE · %d fenetres actives pour %d morts (%.2f %%) · seuil 90 %% · %s",
		active, len(in.deaths), psPct(active, len(in.deaths)),
		psVerdict(active, len(in.deaths), 90))
	psTeamDowntime(t, in, gaps)
}

// psDeathToSpawnGaps rend, en secondes, l'intervalle entre la fin d'une vie et le debut de la
// vie suivante DU MEME JOUEUR (jamais du meme slot : le slot migre a la reapparition).
func psDeathToSpawnGaps(in psInputs) []float64 {
	byXUID := map[uint64][]lifeSpan{}
	for _, l := range in.lives {
		if x, ok := in.own.SlotXUID[l.slot]; ok {
			byXUID[x] = append(byXUID[x], l)
		}
	}
	var out []float64
	for _, ls := range byXUID {
		sort.Slice(ls, func(i, j int) bool { return ls[i].from < ls[j].from })
		for i := 1; i < len(ls); i++ {
			if d := ls[i].from - ls[i-1].to; d > 0 {
				out = append(out, float64(d)/1e6)
			}
		}
	}
	return out
}

// psTeamDowntime rend le temps mort cumule, par joueur puis somme — le chiffre que la phase 2
// du lot B voulait afficher en en-tete. L'EQUIPE n'est pas resolue ici : le film ne la donne
// pas de facon fiable (decision de 2026-06 sur les evenements), et l'inventer serait pire que
// de rendre le total par joueur.
func psTeamDowntime(t *testing.T, in psInputs, gaps []float64) {
	t.Helper()
	total := 0.0
	for _, g := range gaps {
		total += g
	}
	t.Logf("B.0.4 TEMPS MORT CUMULE · %d intervalles · total %.1f s · moyenne %.2f s "+
		"(equipe NON resolue : le film ne la porte pas de facon fiable)",
		len(gaps), total, total/float64(maxInt(len(gaps), 1)))
}

// psDump depose les echantillons bruts sous PLAYER_OUT.
func psDump(t *testing.T, in psInputs) {
	t.Helper()
	out := os.Getenv(playerOutEnv)
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("repertoire de sortie %s : %v", out, err)
	}
	path := filepath.Join(out, in.short+"_arme_en_main.tsv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("ecriture %s : %v", path, err)
	}
	defer f.Close()
	fmt.Fprintln(f, "slot\tt_us\tfamille")
	for _, h := range in.held {
		fmt.Fprintf(f, "%d\t%d\t%d\n", h.Slot, h.TimestampUS, h.Family)
	}
	t.Logf("TSV arme en main depose dans %s (%d lignes)", path, len(in.held))
}

func psPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// psVerdict rend TENU / NON TENU contre un seuil en pourcentage, ou « denominateur nul ».
func psVerdict(num, den, seuil int) string {
	if den <= 0 {
		return "NON MESURABLE (denominateur nul)"
	}
	if psPct(num, den) >= float64(seuil) {
		return "TENU"
	}
	return "NON TENU"
}

func psAbsU(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// psAngleDelta rend l'ecart angulaire minimal entre deux caps en degres.
func psAngleDelta(a, b float64) float64 {
	d := a - b
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	if d < 0 {
		return -d
	}
	return d
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
