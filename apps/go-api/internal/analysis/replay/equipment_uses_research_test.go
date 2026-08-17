package replay

// equipment_uses_research_test.go — DATER L'USAGE D'UN EQUIPEMENT PAR SES CHARGES (lot D,
// phase 0 : MESURER, aucune publication).
//
// LA THESE, ET CE QUI PEUT LA TUER. `equipment-charges-remaining` (ti=37 i27, R(8)) est le
// canal le plus bavard de l'archetype (16 125 annonces sur 12 films, 3,1x l'energie) et
// `equipment-energy-delay-ticks-left` (i26, R(10)) le second (10 608). Un compteur de charges
// qui DECROIT date un usage par construction — a condition que la decroissance coincide avec
// un geste de joueur mesure AILLEURS. Sans cette coincidence, un compteur qui bouge n'est
// qu'un compteur qui bouge, et les trois voies deja refutees le rappellent : `activated` (i21,
// 81 transitions sur 12 matchs), le R(24) d'i57 (valeurs quasi toutes uniques, pas un handle),
// les naissances ti=37 (densite au niveau des temoins decales).
//
// L'ORACLE EST LE GRAPPIN, et il est INDEPENDANT du canal mesure. Le corps tag==3 d'i59
// (`grapple_state.go`) date les usages de grappin par PAIRES a 0,150 s (le tir, puis
// l'accroche), chacune attribuee a un slot de bipede. Cet oracle ne lit pas un seul bit de
// ti=37 : si les decrements de charge d'un objet grappin tombent sur ces instants, la
// coincidence n'est pas une tautologie.
//
// LES SEUILS SONT ECRITS AVANT LA MESURE (D13), et ils viennent du plan, pas de la sortie :
// appariement >= 80 %, temoin decale de +7 s <= 10 %, coherence du pont objet -> joueur
// >= 90 %, controle croise pose <-> decrement >= 80 %.
//
// TROIS TEMOINS, PAS UN. (a) les MEMES evenements decales de +7 s — il teste le hasard
// temporel ; (b) les memes evenements contre les decrements des AUTRES familles — il teste la
// specificite de l'identite (si un evenement de grappin s'apparie aussi bien a un decrement de
// mur, l'identite ne sert a rien) ; (c) le plancher de bruit de l'ancrage lui-meme, publie par
// le balayage des poses (ancres / acceptes / confirmes par l'oracle de position).
//
// L'IDENTITE DE L'OBJET N'EST PAS DEVINEE : elle vient du GlobalID du tag `eqip` du record de
// CREATION (`ScanFilmEquipmentPlacements`, calibration MPP par oracle de position), puis du
// manifeste du titre — LA MEME table que la production. Le grappin y porte deux identifiants
// (0x273fe0eb `ability_grapple_hook`, 0x8c77ffe7 meme modele `hlmt`).
//
// LECTURE SEULE : aucune base, aucun artefact ecrit, aucun champ de document. UN SEUL decodage
// filmdec par process (LockProcessDecode), UN SEUL film par process (D17).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQ_FILM=<principal>/data/cache/film_chunks/000d5950 \
//	  EQ_BOUNDS=<principal>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  EQ_OUT=<repo>/.ai/V7.5/replay2d/registre_film/lotD \
//	  go test ./internal/analysis/replay/ -run '^TestEquipmentUsesPhase0$' -timeout 60m -v
//
// `EQ_BOUNDS` est FACULTATIF : sans lui les positions restent en coordonnees normalisees
// [0,1] de l'AABB (ce qui suffit a tout ce qui est mesure ici, l'appariement etant temporel),
// et l'attribution du poseur — qui exige des METRES — est declaree non mesurable au lieu
// d'etre calculee dans une unite muette.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/mappings"
)

const (
	eqUsesFilmEnv   = "EQ_FILM"
	eqUsesBoundsEnv = "EQ_BOUNDS"
	eqUsesMapEnv    = "EQ_MAP"
	eqUsesOutEnv    = "EQ_OUT"
)

// Seuils et fenetres, ECRITS AVANT LA MESURE (plan lot D, gate 0).
const (
	// eqUsesWindowUS est la demi-fenetre d'appariement evenement <-> decrement (± 0,5 s).
	eqUsesWindowUS = 500_000
	// eqUsesWitnessUS est le decalage du temoin (a). +7 s : au-dela de toute latence de
	// replication, en deca de l'intervalle entre deux usages successifs d'une meme vie.
	eqUsesWitnessUS = 7_000_000
	// eqUsesPairMin / eqUsesWitnessMax : le gate 0 du lot.
	eqUsesPairMin    = 0.80
	eqUsesWitnessMax = 0.10
	// eqUsesCoherenceMin est le seuil du pont objet -> joueur (D.0.3).
	eqUsesCoherenceMin = 0.90
	// eqUsesCrossMin est le seuil du controle croise pose <-> decrement (D.0.4).
	eqUsesCrossMin = 0.80
	// eqUsesCreationSlackUS tolere l'ecart entre le record de CREATION d'une vie d'objet et
	// son premier paquet delta : la creation precede, de quelques dizaines de ms.
	eqUsesCreationSlackUS = 500_000
	// eqUsesGrappleFamily est la famille du manifeste qui sert d'oracle.
	eqUsesGrappleFamily = "grapple"
)

func TestEquipmentUsesPhase0(t *testing.T) {
	dir := os.Getenv(eqUsesFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", eqUsesFilmEnv)
	}
	short := filepath.Base(dir)
	release := filmdec.LockProcessDecode()
	defer release()

	entry, metres := eqUsesEntry(t, dir)
	defer installWorldObjectPrecision(entry, dir)()
	wr := entry.Range()
	t.Logf("FILM %s · largeurs d'axe %v · bornes %s", short, entry.AxisWidths,
		map[bool]string{true: "MONDE (metres)", false: "NORMALISEES [0,1]"}[metres])

	samples, st, err := filmdec.ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("etat d'equipement illisible : %v", err)
	}
	eqUsesLogVolumes(t, st)
	eqUsesRawTransitions(t, samples)

	placements, pst, err := filmdec.ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("poses d'equipement illisibles : %v", err)
	}
	families := eqUsesFamilies(t)
	eqUsesLogAnchorNoise(t, pst)

	lives, attached := eqUsesBuildLives(placements, families, samples)
	eqUsesLogValues(t, samples, attached)
	eqUsesLogLives(t, lives)
	if !pst.Calibration.Widths.Valid() {
		t.Logf("VERDICT D.0.2/D.0.3/D.0.4 NON CALCULABLES sur ce film : le decoupage MPP n'a"+
			" pas ete tranche (%s) — aucune identite d'objet, donc aucun oracle", pst.Calibration)
		eqUsesWriteTSV(t, short, lives, nil)
		return
	}

	drops := eqUsesChargeDrops(lives)
	rises := eqUsesDelayRises(lives)
	eqUsesOracles(t, dir, lives, [2][]eqSignal{drops, rises})
	eqUsesGeneralise(t, lives, drops, placements, families)
	eqUsesEnergyDelay(t, lives, drops, rises)
	eqUsesOwners(t, eqUsesPositions(t, dir, wr, metres), placements, families)
	eqUsesWriteTSV(t, short, lives, drops)
}

// eqUsesOracles joue D.0.2 et D.0.3 : les canaux candidats contre l'oracle du grappin, chacun
// avec ses deux temoins, puis le pont objet -> joueur du canal qui a apparie quelque chose.
//
// TROIS CANAUX, dans l'ordre du plan : (1) la decroissance de charge (i27, prescrit), (2) la
// hausse du delai de recharge (i26, le repli prescrit si l'equipement porte n'est pas un objet
// decrementable), (3) la NAISSANCE de l'objet grappin — hors plan, mais c'est le controle qui
// EXPLIQUE un negatif au lieu de le constater.
func eqUsesOracles(t *testing.T, dir string, lives []eqLife, sig [2][]eqSignal) {
	t.Helper()
	reads, gst, err := filmdec.ScanFilmGrappleReads(dir)
	if err != nil {
		t.Fatalf("evenements de grappin illisibles : %v", err)
	}
	ids, n := eqUsesGrappleIDs(lives)
	t.Logf("== D.0.2 ORACLE GRAPPIN == i59 : %d records biped · masque∋i59 %d · lus %d"+
		" · tag==3 %d · corps casses %d · EVENEMENTS %d · USAGES (paires tir/accroche) %d",
		gst.Records, gst.WithI59, gst.Read, gst.Tag3, gst.BodyBroken, len(reads),
		eqUsesGrappleUses(reads))
	if n == 0 {
		t.Log("  IDENTITE : AUCUN objet ti=37 de famille `grapple` sur ce film — l'equipement" +
			" porte n'a pas d'entite du monde ici. D.0.2 non calculable sur i27 ; seul le repli" +
			" i26 reste jouable, et il l'est sur les vies des autres familles seulement.")
		return
	}
	t.Logf("  IDENTITE : %d objet(s) ti=37 de famille `grapple`, par GlobalID de tag `eqip` du"+
		" record de CREATION (manifeste du titre, meme table que la production) : %s", n, ids)
	births := eqUsesBirths(lives)
	canaux := []struct {
		nom string
		sig []eqSignal
	}{
		{"i27 decroissance (prescrit)", sig[0]},
		{"i26 hausse (repli prescrit)", sig[1]},
		{"naissance de l'objet (controle)", births},
	}
	for _, c := range canaux {
		in := eqUsesFamilySig(c.sig, lives, eqUsesGrappleFamily, true)
		out := eqUsesFamilySig(c.sig, lives, eqUsesGrappleFamily, false)
		pairs := eqUsesOracle(t, c.nom, reads, in, out)
		eqUsesBridge(t, c.nom, pairs, reads)
	}
}

// eqUsesPositions rend le nuage des bipedes TRIE, ou nil quand les bornes ne sont pas en
// metres — la distance du poseur n'a alors aucun sens (cf. eqUsesOwners).
func eqUsesPositions(
	t *testing.T, dir string, wr filmdec.Vec3Range, metres bool,
) []filmdec.BipedPosition {
	t.Helper()
	if !metres {
		return nil
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &wr
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("nuage des bipedes indisponible : %v", err)
		return nil
	}
	sort.Slice(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	return pos
}

// eqUsesRawTransitions publie le comptage BRUT des transitions, groupe par la seule paire
// (slot, generation) — c'est le chiffre du lot 0 (i26 126/172, i27 194/263 sur `000d5950`),
// et il n'est PAS comparable au comptage par vie identifiee. Publier les deux cote a cote est
// ce qui permet de voir combien de ces « transitions » sont en realite des sauts d'un objet a
// un autre, la generation ne faisant que 2 bits.
func eqUsesRawTransitions(t *testing.T, samples []filmdec.EquipmentStateSample) {
	t.Helper()
	type key struct{ slot, gen uint32 }
	series := map[key][]filmdec.EquipmentStateSample{}
	for _, s := range samples {
		k := key{s.Slot, s.Gen}
		series[k] = append(series[k], s)
	}
	var trans, pairs, down [filmdec.EquipmentFieldCount]int
	for _, ss := range series {
		sort.SliceStable(ss, func(a, b int) bool { return ss[a].TimestampUS < ss[b].TimestampUS })
		for i := 1; i < len(ss); i++ {
			for f := 0; f < filmdec.EquipmentFieldCount; f++ {
				if !ss[i-1].Present[f] || !ss[i].Present[f] {
					continue
				}
				pairs[f]++
				if ss[i].Val[f] != ss[i-1].Val[f] {
					trans[f]++
				}
				if ss[i].Val[f] < ss[i-1].Val[f] {
					down[f]++
				}
			}
		}
	}
	t.Logf("== D.0.1 TRANSITIONS BRUTES (groupees par (slot, generation) SEULE — le comptage"+
		" du lot 0, sur %d cles) ==", len(series))
	for _, f := range []filmdec.EquipmentField{filmdec.EquipEnergyDelay, filmdec.EquipCharges} {
		t.Logf("  %-46s %d transitions sur %d paires consecutives · dont %d en BAISSE",
			f, trans[f], pairs[f], down[f])
	}
}

// eqUsesEntry rend l'entree de quantification : le catalogue quand il est fourni (la carte est
// AUTO-DETECTEE par la signature des largeurs lues dans le film, meme methode qu'i59_anchor),
// sinon les largeurs du film avec des bornes normalisees. Le second rend `metres` faux, et
// tout ce qui exige des metres est alors declare non mesurable.
func eqUsesEntry(t *testing.T, dir string) (filmdec.MapQuantEntry, bool) {
	t.Helper()
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible dans %s : %v", dir, err)
	}
	fallback := filmdec.MapQuantEntry{
		Module: "(bornes normalisees)", AxisWidths: lay.AxisW,
		Min: [3]float32{0, 0, 0}, Max: [3]float32{1, 1, 1},
	}
	path := os.Getenv(eqUsesBoundsEnv)
	if path == "" {
		return fallback, false
	}
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible (%s) : %v", path, err)
	}
	if name := os.Getenv(eqUsesMapEnv); name != "" {
		e, err := cat.Lookup(name)
		if err != nil {
			t.Fatalf("carte %q absente du catalogue : %v", name, err)
		}
		return e, true
	}
	var hits []string
	var found filmdec.MapQuantEntry
	for name, e := range cat.Maps {
		if e.AxisWidths == lay.AxisW {
			hits, found = append(hits, name), e
		}
	}
	sort.Strings(hits)
	if len(hits) != 1 {
		t.Logf("signature %v : %d cartes candidates %v — bornes normalisees, poseur non"+
			" mesurable (renseigner %s pour trancher)", lay.AxisW, len(hits), hits, eqUsesMapEnv)
		return fallback, false
	}
	t.Logf("carte auto-detectee par signature %v : %s", lay.AxisW, hits[0])
	return found, true
}

// eqUsesFamilies lit le manifeste du titre — LA MEME table que la production.
func eqUsesFamilies(t *testing.T) map[uint32]string {
	t.Helper()
	path := filepath.Join(repoRootForTest(t), "config", "titles", "halo_infinite",
		"mappings", "replay_labels.toml")
	labels, err := mappings.LoadReplayLabelsFromFile(path)
	if err != nil {
		t.Fatalf("manifeste des libelles illisible (%s) : %v", path, err)
	}
	return labels.EquipmentObjects()
}

// eqUsesLogVolumes publie D.0.1 : ce que le masque annonce et ce que la marche a lu, champ par
// champ. Sans ces denominateurs, une distribution de valeurs ne se juge pas.
func eqUsesLogVolumes(t *testing.T, st filmdec.EquipmentStateStats) {
	t.Helper()
	t.Logf("== D.0.1 VOLUMES == records delta ti=37 %d · slots %d · masque∋(un des 6) %d"+
		" · marche ABOUTIE %d · CASSEE %d", st.Records, st.Slots, st.WithAny, st.Walked, st.Broken)
	for f := 0; f < filmdec.EquipmentFieldCount; f++ {
		t.Logf("  %-46s masque %6d · LU %6d · porte fermee %5d",
			filmdec.EquipmentField(f), st.WithField[f], st.Read[f], st.Gated[f])
	}
	for _, i := range []int{26, 27} {
		if i < len(st.MaskCensus) {
			t.Logf("  recensement brut au masque i%d : %d", i, st.MaskCensus[i])
		}
	}
	if i := 19; i < len(st.MaskCensus) {
		t.Logf("  D.0.3 (alternative) i%d `item-ignore-player` annonce %d fois — sa grammaire"+
			" est R(1)[+R(5)] (components_world.go:124), soit 5 bits : la MEME forme que i23"+
			" `equipment-creator`, deja REFUTEE comme slot de bipede (0 valeur sur 1 328 dans"+
			" la bande) ET comme index de joueur (28 valeurs distinctes sur un film a"+
			" 8 joueurs). Sa VALEUR n'est pas lisible en phase 0 : le deser la jette, et poser"+
			" un hook est de la plomberie (interdite ici, lecture seule)", i, st.MaskCensus[i])
	}
}

// eqUsesLogValues publie la distribution des valeurs d'i27 et d'i26 : « charges » exige des
// entiers petits, « compte a rebours » une plage large.
func eqUsesLogValues(t *testing.T, samples []filmdec.EquipmentStateSample, attached []bool) {
	t.Helper()
	t.Log("== D.0.1 DISTRIBUTION DES VALEURS TRANSMISES — REEL (lecture rattachee a une vie" +
		" CONFIRMEE par l'oracle de position) contre FANTOME (aucune vie confirmee) ==")
	for _, f := range []filmdec.EquipmentField{filmdec.EquipCharges, filmdec.EquipEnergyDelay} {
		for _, reel := range []bool{true, false} {
			hist := map[uint64]int{}
			n := 0
			for i, s := range samples {
				if !s.Present[f] || attached[i] != reel {
					continue
				}
				hist[s.Val[f]]++
				n++
			}
			nom := map[bool]string{true: "REEL   ", false: "FANTOME"}[reel]
			if n == 0 {
				t.Logf("  %s %-38s aucune valeur", nom, f)
				continue
			}
			t.Logf("  %s %-38s %5d valeurs · %3d distinctes · max %4d · %s",
				nom, f, n, len(hist), eqUsesMaxKey(hist), eqUsesHist(hist, 16))
		}
	}
}

// eqUsesLogAnchorNoise publie le TEMOIN (c) : le plancher de bruit de l'ancrage par bande.
// L'en-tete NEW de ti=37 n'est pas selectif (un quart des positions de bit tirees au hasard le
// passent sur un film BTB) ; c'est l'oracle de position qui fait la mesure, et l'ecart entre
// les trois compteurs EST le rapport reel/fantome.
func eqUsesLogAnchorNoise(t *testing.T, st filmdec.EquipmentPlacementStats) {
	t.Helper()
	pur := 0.0
	if st.Accepted > 0 {
		pur = 100 * float64(st.Confirmed) / float64(st.Accepted)
	}
	t.Logf("== TEMOIN (c) PLANCHER DE BRUIT DE L'ANCRAGE == calibration %s · vies %d"+
		" · ancres %d · acceptes %d · CONFIRMES par l'oracle de position %d (%.2f %% des"+
		" acceptes) · poses %d", st.Calibration, st.Lives, st.Anchors, st.Accepted,
		st.Confirmed, pur, st.Placements)
	t.Log("  purete etablie de l'ancrage ti=37 en DELTA (hors de ce lot, ecrite au registre" +
		" ECS) : 97,2 % de 628 368 echantillons i0 dans l'emprise des bipedes du meme film," +
		" 12 films sur 12 ; temoin d'ancrage a 1 slot (ti=4) : 98,7-99,8 % (lot C phase 1a)." +
		" `ScanFilmEquipmentState` ne prend pas de bande en parametre : un tirage FANTOME du" +
		" balayage d'etat exigerait de modifier du code de production, hors phase 0.")
}

// eqUsesMaxKey rend la plus grande cle d'un histogramme.
func eqUsesMaxKey(h map[uint64]int) uint64 {
	var max uint64
	for k := range h {
		if k > max {
			max = k
		}
	}
	return max
}

// eqUsesHist rend un histogramme trie, borne a `cap` classes.
func eqUsesHist(h map[uint64]int, limit int) string {
	keys := make([]uint64, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	out := ""
	for i, k := range keys {
		if i == limit {
			out += fmt.Sprintf(" … %d classes au total", len(keys))
			break
		}
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("%d:%d", k, h[k])
	}
	return out
}

// eqUsesPct rend un pourcentage lisible avec son denominateur.
func eqUsesPct(n, d int) string {
	if d == 0 {
		return fmt.Sprintf("%d/0 (non calculable)", n)
	}
	return fmt.Sprintf("%d/%d (%.1f %%)", n, d, 100*float64(n)/float64(d))
}

// eqUsesVerdict rend le verdict d'un seuil, jamais une appreciation.
func eqUsesVerdict(ok bool) string {
	if ok {
		return "SEUIL TENU"
	}
	return "SEUIL NON TENU"
}
