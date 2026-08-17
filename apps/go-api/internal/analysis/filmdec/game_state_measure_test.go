package filmdec

// game_state_measure_test.go — INSTRUMENT DE MESURE des deux entites non corporelles :
// le MOTEUR DE PARTIE (ti=0) et le JOUEUR (ti=5). Phase 0 des lots B et P.
//
// LES QUESTIONS POSEES, ET LE SEUIL DE CHACUNE, ECRITS AVANT LA MESURE (D13) :
//
//	B.0.1  L'ancrage tient-il sur des bandes de 1 et 8 slots ?  -> debit reel / debit des deux
//	       bandes de controle, et purete de la bande contre la grammaire de l'archetype. Le
//	       temoin d'etalonnage est ti=4 (lot C : 98,7-99,8 %).
//	B.0.2  Le round-timer est-il l'horloge officielle ?  -> pente contre l'horloge du film
//	       (|pente + 1| <= 2 %), valeur initiale contre `regulation.toml` (+/- 1 s), instant du
//	       premier decompte contre `originMs` (MESURE PUBLIEE, jamais une correction : D4).
//	B.0.3  La mort subite (i6) est-elle la source exacte de la prolongation ?  -> i6 non nul en
//	       fin de match sur les matchs flagues, i6 toujours nul sur les temoins non flagues.
//	B.0.5  Les etats (i2) et les manches (i4) sont-ils datables ?  -> transitions, et bornes de
//	       manche contre le moment ou le score de manche se fige (lot A).
//	P.0.1  Quels canaux de ti=5 parlent ?  -> recensement d'annonces au masque par composant.
//	P.0.2  Les 8 octets d'i11 sont-ils le chargement de depart ?  -> bijection octet -> famille
//	       stable sur >= 90 % des vies, MEME table sur 3 films.
//	P.0.5  Les etats du joueur sont-ils lisibles ?  -> i2/i3/i12/i14/i15/i18/i19/i20.
//
// LECTURE SEULE, garde par GAME_FILM, saute partout ailleurs (CI comprise). UN SEUL film par
// processus (D17 : la machine de l'utilisateur a deja plante sur un balayage de corpus).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GAME_FILM=C:/.../data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestGameEntitiesPhase0$' -timeout 30m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	gameFilmEnv = "GAME_FILM"
	// gameOutEnv designe le repertoire ou deposer les TSV. Vide = aucun fichier ecrit (le
	// journal du test suffit a relire la mesure).
	gameOutEnv = "GAME_OUT"
	// gameClockHuntEnv arme la chasse au slot de l horloge. Elle marche TOUS les slots et
	// coute environ 50 s par film : elle ne s arme qu a la demande.
	gameClockHuntEnv = "CLOCK_HUNT"
)

func TestGameEntitiesPhase0(t *testing.T) {
	dir := os.Getenv(gameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", gameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	sc, err := ScanFilmGameEntities(dir)
	if err != nil {
		t.Fatalf("balayage des entites de partie impossible : %v", err)
	}
	short := gameShort8(dir)
	gameLogBands(t, sc)
	gameLogClass(t, sc, GameEngineTypeIndex, "ti=0 moteur de partie")
	gameLogClass(t, sc, PlayerEngineTypeIndex, "ti=5 joueur")
	gameLogClass(t, sc, ProbeWitnessTypeIndex, "ti=4 temoin d'ancrage")
	gameLogControls(t, sc)
	gameLogEngineFields(t, sc)
	gameLogPlayerFields(t, sc)
	gamePlayerSlotProfile(t, sc)
	gamePlayerStateValues(t, sc)
	gameChainPass(t, dir, short)
	gameHuntEngineSlot(t, dir, sc)
	gameDumpTSV(t, short, sc)
}

// gameChainPass est la SECONDE LECTURE du meme film, par la chaine sequentielle. Elle rend des
// records certains (aucun ancrage probabiliste) au prix d'une couverture d'un tiers, et c'est
// elle qui tranche les seuils : la voie par bande ne fait que la corroborer.
func gameChainPass(t *testing.T, dir, short string) {
	t.Helper()
	recs, held, st, err := ScanFilmGameEntitiesChain(dir)
	if err != nil {
		t.Logf("CHAINE SEQUENTIELLE impossible : %v", err)
		return
	}
	t.Logf("CHAINE SEQUENTIELLE · paquets %d dont PROPRES %d (%.1f %%) · records %d dont "+
		"ABOUTIS %d (%.1f %%) · ti=0 %d · ti=5 %d", st.Packets, st.PacketsClean,
		gamePct(st.PacketsClean, st.Packets), st.Records, st.RecordsClean,
		gamePct(st.RecordsClean, st.Records), st.EngineRecords, st.PlayerRecords)
	t.Logf("CHAINE · ARME EN MAIN : records de bipede confirmes %d dont porteurs d une identite "+
		"d arme %d (%.1f %%) · echantillons %d", st.BipedRecords, st.HeldWeaponReads,
		gamePct(st.HeldWeaponReads, st.BipedRecords), len(held))
	gameLogChainTI(t, st)
	if len(recs) == 0 {
		t.Log("CHAINE SEQUENTIELLE : aucun record de ti=0 ni de ti=5 n'a abouti")
		return
	}
	gameChainSlots(t, recs)
	gameChainFields(t, recs)
	gameChainDump(t, short, recs)
	gameSummary(t, short, st, recs)
}

// gameSummary ajoute UNE ligne par film a la synthese du lot. C'est elle qui porte les
// denominateurs du CR : sans une table a une ligne par film, comparer treize journaux revient
// a recopier des chiffres a la main, et c'est ainsi qu'un denominateur se perd.
func gameSummary(t *testing.T, short string, st GameChainStats, recs []GameEntityRecord) {
	t.Helper()
	out := os.Getenv(gameOutEnv)
	if out == "" {
		return
	}
	var engine, player, respawn, active int
	perSlot := map[uint32]int{}
	fields := make([]int, PlayerStateFieldCount)
	for _, r := range recs {
		switch r.TI {
		case GameEngineTypeIndex:
			engine++
		case PlayerEngineTypeIndex:
			player++
			perSlot[r.Slot]++
			if r.HasRespawn {
				respawn++
				if r.Respawn.Active {
					active++
				}
			}
			for f := 0; f < PlayerStateFieldCount; f++ {
				if r.PlayerSeen[PlayerStateField(f)] {
					fields[f]++
				}
			}
		}
	}
	counts := make([]int, 0, len(perSlot))
	for _, n := range perSlot {
		counts = append(counts, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(counts)))
	top8 := 0
	for i, n := range counts {
		if i >= 8 {
			break
		}
		top8 += n
	}
	cells := []string{
		short, fmt.Sprint(st.Chunks), fmt.Sprint(st.Packets), fmt.Sprint(st.PacketsClean),
		fmt.Sprint(st.RecordsClean), fmt.Sprint(engine), fmt.Sprint(player),
		fmt.Sprint(len(perSlot)), fmt.Sprint(top8), fmt.Sprint(respawn), fmt.Sprint(active),
	}
	for f := 0; f < PlayerStateFieldCount; f++ {
		cells = append(cells, fmt.Sprint(fields[f]))
	}
	gameAppendLine(t, filepath.Join(out, "synthese_films.tsv"),
		"short8\tchunks\tpaquets\tpaquets_propres\trecords_aboutis\tti0\tti5\tti5_slots\t"+
			"ti5_top8\trespawn\trespawn_actif\ti2_softkill\ti3_tracking\ti6_desired\ti11_loadout\t"+
			"i12_resploc\ti14_lives\ti15_betrayer\ti17_aiming\ti18_active\ti19_joining\ti20_malleable",
		strings.Join(cells, "\t"))
}

// gameAppendLine ajoute une ligne au fichier, en ecrivant l'en-tete a la creation.
func gameAppendLine(t *testing.T, path, header, line string) {
	t.Helper()
	_, err := os.Stat(path)
	fresh := os.IsNotExist(err)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("synthese %s : %v", path, err)
	}
	defer f.Close()
	if fresh {
		if _, err := fmt.Fprintln(f, header); err != nil {
			t.Fatalf("synthese %s : %v", path, err)
		}
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		t.Fatalf("synthese %s : %v", path, err)
	}
}

func gamePct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// gameLogChainTI journalise le recensement des archetypes ABOUTIS. C'est l'equivalent hors
// ligne de la ventilation par archetype que le lot C avait obtenue par capture Cheat Engine :
// deux methodes independantes, comparables record a record.
func gameLogChainTI(t *testing.T, st GameChainStats) {
	t.Helper()
	type kv struct {
		ti uint32
		n  int
	}
	rows := make([]kv, 0, len(st.ByTI))
	for ti, n := range st.ByTI {
		rows = append(rows, kv{ti, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	var parts []string
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("ti=%d:%d", r.ti, r.n))
	}
	t.Logf("CHAINE · VENTILATION DES RECORDS ABOUTIS PAR ARCHETYPE : %s", strings.Join(parts, " "))
	var conf []string
	for _, r := range rows {
		if n := st.ByTIConfirmed[r.ti]; n > 0 {
			conf = append(conf, fmt.Sprintf("ti=%d:%d", r.ti, n))
		}
	}
	t.Logf("CHAINE · LES MEMES, LIAISON CONFIRMEE PAR UNE IMAGE-CLE : %s", strings.Join(conf, " "))
}

// gameChainSlots rend la ventilation par slot des records certains de ti=5.
func gameChainSlots(t *testing.T, recs []GameEntityRecord) {
	t.Helper()
	per := map[uint32]int{}
	for _, r := range recs {
		if r.TI == PlayerEngineTypeIndex {
			per[r.Slot]++
		}
	}
	slots := make([]uint32, 0, len(per))
	for s := range per {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return per[slots[i]] > per[slots[j]] })
	var parts []string
	for _, s := range slots {
		parts = append(parts, fmt.Sprintf("%d:%d", s, per[s]))
	}
	t.Logf("CHAINE · ti=5 PAR SLOT (%d slots) : %s", len(per), strings.Join(parts, " "))
}

// gameChainFields rend, pour chaque champ de ti=5 et de ti=0, la distribution des valeurs des
// records CERTAINS. C'est le tableau sur lequel P.0.5 et B.0.5 se jugent.
func gameChainFields(t *testing.T, recs []GameEntityRecord) {
	t.Helper()
	t.Logf("CHAINE · ti=0 CHAMPS")
	for f := 0; f < GameEngineFieldCount; f++ {
		fl := GameEngineField(f)
		hist, n, gated := map[string]int{}, 0, 0
		for _, r := range recs {
			if r.TI != GameEngineTypeIndex || !r.EngineSeen[fl] {
				continue
			}
			n++
			if !r.EnginePresent[fl] {
				gated++
				continue
			}
			hist[gameJoin(r.EngineVal[fl])]++
		}
		t.Logf("    %-48s lectures %5d · porte fermee %5d · distinctes %4d · %s",
			fl, n, gated, len(hist), gameTopValues(hist, 6))
	}
	nrt := 0
	for _, r := range recs {
		if r.TI == GameEngineTypeIndex && r.HasRoundTimer {
			nrt++
		}
	}
	t.Logf("    %-48s CAPTURE : %d lectures certaines", compGameEngineRoundTimer, nrt)
	t.Logf("CHAINE · ti=5 CHAMPS")
	for f := 0; f < PlayerStateFieldCount; f++ {
		fl := PlayerStateField(f)
		hist, n, gated := map[string]int{}, 0, 0
		for _, r := range recs {
			if r.TI != PlayerEngineTypeIndex || !r.PlayerSeen[fl] {
				continue
			}
			n++
			if !r.PlayerPresent[fl] {
				gated++
				continue
			}
			hist[gameJoin(r.PlayerVal[fl])]++
		}
		t.Logf("    %-48s lectures %5d · porte fermee %5d · distinctes %4d · %s",
			fl, n, gated, len(hist), gameTopValues(hist, 6))
	}
	gameChainRespawn(t, recs)
}

// gameChainRespawn rend la distribution du compte a rebours de reapparition (ti=5 i1), le seul
// canal de ti=5 qui porte une duree — c'est lui que B.0.4 mesure.
func gameChainRespawn(t *testing.T, recs []GameEntityRecord) {
	t.Helper()
	n, active := 0, 0
	hist := map[string]int{}
	for _, r := range recs {
		if r.TI != PlayerEngineTypeIndex || !r.HasRespawn {
			continue
		}
		n++
		if r.Respawn.Active {
			active++
		}
		hist[fmt.Sprintf("%v/%d/%d", r.Respawn.Active, r.Respawn.T0, r.Respawn.T1)]++
	}
	t.Logf("    %-48s CAPTURE : %d lectures certaines dont ACTIF %d · distinctes %d · %s",
		compPlayerRespawnTimer, n, active, len(hist), gameTopValues(hist, 6))
}

func gameJoin(v []uint64) string {
	parts := make([]string, 0, len(v))
	for _, x := range v {
		parts = append(parts, fmt.Sprint(x))
	}
	return strings.Join(parts, ",")
}

// gameChainDump depose les records certains, la matiere premiere des items P.0.2 / P.0.5.
func gameChainDump(t *testing.T, short string, recs []GameEntityRecord) {
	t.Helper()
	out := os.Getenv(gameOutEnv)
	if out == "" {
		return
	}
	rows := make([]string, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, strings.Join([]string{
			fmt.Sprint(r.TI), fmt.Sprint(r.Chunk), fmt.Sprint(r.PacketIndex),
			fmt.Sprint(r.TimestampUS), fmt.Sprint(r.Slot), fmt.Sprint(r.Gen),
			gameIdxString(r.Idx), gameRespawnCell(r),
			gamePlayerVals(r, PlayerSoftKill), gamePlayerVals(r, PlayerTargetTracking),
			gamePlayerVals(r, PlayerDesiredRespawnPlayer), gamePlayerVals(r, PlayerLoadout),
			gamePlayerVals(r, PlayerDesiredRespawnLocation), gamePlayerVals(r, PlayerLives),
			gamePlayerVals(r, PlayerLastBetrayer), gamePlayerVals(r, PlayerControlAiming),
			gamePlayerVals(r, PlayerActiveInGame), gamePlayerVals(r, PlayerPendingJoinInProgress),
			gamePlayerVals(r, PlayerMalleableProperties),
		}, "\t"))
	}
	gameWriteTSV(t, filepath.Join(out, short+"_chaine.tsv"),
		"ti\tchunk\tpacket\tt_us\tslot\tgen\tmask\trespawn\tsoftkill\ttracking\tdesired_player\t"+
			"loadout\tresp_loc\tlives\tbetrayer\taiming\tactive\tjoining\tmalleable", rows)
}

func gameRespawnCell(r GameEntityRecord) string {
	if !r.HasRespawn {
		return ""
	}
	return fmt.Sprintf("%v/%d/%d", r.Respawn.Active, r.Respawn.T0, r.Respawn.T1)
}

// gamePlayerSlotProfile rend la repartition des lectures ti=5 PAR SLOT.
//
// POURQUOI CETTE VENTILATION DECIDE DU DENOMINATEUR. La bande observee de ti=5 compte 32
// slots, alors qu'une partie d'arene n'a que huit joueurs : le moteur declare des entites
// joueur pour la capacite maximale du serveur, et vingt-quatre d'entre elles ne parleront
// jamais. Comparer le debit de la bande entiere au temoin fantome divise donc le signal par
// quatre AVANT toute mesure. La ventilation par slot dit combien de slots portent
// reellement du trafic, et c'est ce nombre-la qui est le denominateur honnete.
func gamePlayerSlotProfile(t *testing.T, sc GameEntityScan) {
	t.Helper()
	if len(sc.Player) == 0 {
		return
	}
	per := map[uint32]int{}
	for _, r := range sc.Player {
		per[r.Slot]++
	}
	slots := make([]uint32, 0, len(per))
	for s := range per {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return per[slots[i]] > per[slots[j]] })
	var parts []string
	for i, s := range slots {
		if i >= 16 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", s, per[s]))
	}
	t.Logf("ti=5 VENTILATION PAR SLOT (%d slots ont rendu une lecture) : %s",
		len(per), strings.Join(parts, " "))
	nb, vd := sc.Stats[GameEntityClassNeighbour], sc.Stats[GameEntityClassVoid]
	top := 0
	for i, s := range slots {
		if i >= 8 {
			break
		}
		top += per[s]
	}
	n := len(slots)
	if n > 8 {
		n = 8
	}
	if n > 0 {
		t.Logf("ti=5 DEBIT DES 8 SLOTS LES PLUS BAVARDS : %.1f lectures/slot contre %.1f "+
			"(voisinage) et %.1f (vide) -> x%.2f et x%.2f", float64(top)/float64(n),
			gameWantedPerSlot(nb), gameWantedPerSlot(vd),
			gameRatio(float64(top)/float64(n), gameWantedPerSlot(nb)),
			gameRatio(float64(top)/float64(n), gameWantedPerSlot(vd)))
	}
}

// gamePlayerStateValues rend, pour chaque champ de ti=5, la distribution de ses valeurs.
// C'est la matiere de P.0.5 : un champ dont toutes les lectures rendent la meme valeur ne
// porte aucun etat, et un champ dont les valeurs sont uniformement etalees sur son domaine
// est du bruit — les deux se voient ici et nulle part ailleurs.
func gamePlayerStateValues(t *testing.T, sc GameEntityScan) {
	t.Helper()
	if len(sc.Player) == 0 {
		return
	}
	t.Logf("ti=5 DISTRIBUTION DES VALEURS (P.0.5)")
	for f := 0; f < PlayerStateFieldCount; f++ {
		fl := PlayerStateField(f)
		hist, n, gated := map[string]int{}, 0, 0
		for _, r := range sc.Player {
			if !r.PlayerSeen[fl] {
				continue
			}
			n++
			if !r.PlayerPresent[fl] {
				gated++
				continue
			}
			hist[gamePlayerVals(r, fl)]++
		}
		if n == 0 {
			t.Logf("    %-48s AUCUNE lecture", fl)
			continue
		}
		t.Logf("    %-48s lectures %5d · porte fermee %5d · valeurs distinctes %4d · %s",
			fl, n, gated, len(hist), gameTopValues(hist, 6))
	}
}

// gameTopValues rend les k valeurs les plus frequentes d'un histogramme.
func gameTopValues(hist map[string]int, k int) string {
	type kv struct {
		v string
		n int
	}
	rows := make([]kv, 0, len(hist))
	for v, n := range hist {
		rows = append(rows, kv{v, n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].v < rows[j].v
	})
	var parts []string
	for i, r := range rows {
		if i >= k {
			break
		}
		v := r.v
		if len(v) > 40 {
			v = v[:40] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s(%d)", v, r.n))
	}
	return strings.Join(parts, " ")
}

// gameHuntEngineSlot cherche le slot du moteur de partie quand les images-cles ne le portent
// pas. LE CRITERE EST LA GRAMMAIRE, PAS LES VALEURS : on compte par slot les en-tetes dont le
// masque tient dans les 27 composants de ti=0 et annonce le round-timer. Chercher un slot par
// ce que ses valeurs devraient valoir construirait le resultat.
func gameHuntEngineSlot(t *testing.T, dir string, sc GameEntityScan) {
	t.Helper()
	if len(sc.Bands[GameEngineTypeIndex]) > 0 {
		return
	}
	t.Logf("CHASSE ti=0 : la bande est VIDE (aucun slot d'archetype 0 dans les images-cles) "+
		"— recherche du slot par la grammaire de l'archetype et l'annonce de %s",
		compGameEngineRoundTimer)
	rows, err := HuntArchetypeSlots(dir, GameEngineTypeIndex, compGameEngineRoundTimer)
	if err != nil {
		t.Logf("CHASSE ti=0 impossible : %v", err)
		return
	}
	tot, mid := 0, 0.0
	vals := make([]int, 0, len(rows))
	for _, r := range rows {
		tot += r.WithMust
		vals = append(vals, r.WithMust)
	}
	sort.Ints(vals)
	if len(vals) > 0 {
		mid = float64(vals[len(vals)/2])
	}
	t.Logf("CHASSE ti=0 : %d slots ont porte au moins un en-tete · total annonces i5 %d "+
		"· mediane par slot %.1f", len(rows), tot, mid)
	for i, r := range rows {
		if i >= 10 {
			break
		}
		t.Logf("    slot %5d · en-tetes %6d · dans la grammaire %6d · annoncent i5 %6d",
			r.Slot, r.Records, r.InGrammar, r.WithMust)
	}
	if os.Getenv(gameClockHuntEnv) == "" {
		t.Logf("CHASSE HORLOGE non demandee (%s absent) : la passe qui marche tous les slots "+
			"coute ~50 s par film et n'a rien rendu sur les films deja mesures", gameClockHuntEnv)
		return
	}
	gameHuntClock(t, dir)
}

// gameHuntClock cherche le slot de l'horloge par sa SIGNATURE : une valeur qui decroit de
// facon monotone sur toute la partie. Le critere d'identification ne fixe ni la pente, ni la
// valeur de depart, ni l'instant de depart — les trois grandeurs que B.0.2 mesure ensuite
// contre des oracles exterieurs.
func gameHuntClock(t *testing.T, dir string) {
	t.Helper()
	cands, err := HuntGameEngineClock(dir)
	if err != nil {
		t.Logf("CHASSE HORLOGE impossible : %v", err)
		return
	}
	t.Logf("CHASSE HORLOGE : %d slots ont rendu au moins une lecture d'horloge", len(cands))
	shown := 0
	for _, c := range cands {
		if shown >= 15 {
			break
		}
		shown++
		mono := 0.0
		if c.Down+c.Up > 0 {
			mono = float64(c.Down) / float64(c.Down+c.Up)
		}
		t.Logf("    slot %5d · lectures %6d · decroit %5d / croit %5d (monotonie %.2f) "+
			"· A [%.1f , %.1f] s · pente %.3f s/s · fenetre %.1f s",
			c.Slot, c.Samples, c.Down, c.Up, mono, c.MinA, c.MaxA, c.Slope,
			float64(c.LastUS-c.FirstUS)/1e6)
	}
}

// gameShort8 rend le nom court du film (le repertoire).
func gameShort8(dir string) string {
	return filepath.Base(strings.TrimRight(filepath.Clean(dir), string(filepath.Separator)))
}

func gameLogBands(t *testing.T, sc GameEntityScan) {
	t.Helper()
	t.Logf("FILM %s · chunks %d · paquets delta %d · horloge film %d us · delta [%d , %d] us "+
		"(%.1f s)", gameShort8(os.Getenv(gameFilmEnv)), sc.Chunks, sc.Packets, sc.FilmClockUS,
		sc.FirstPacketUS, sc.LastPacketUS, float64(sc.LastPacketUS-sc.FirstPacketUS)/1e6)
	t.Logf("BANDES · ti=0 %s · ti=5 %s · ti=4 %s · controle voisin %d · controle vide %d "+
		"· slots ambigus ecartes %d · comblement NON applique (aurait ajoute %d slots)",
		gameSlotList(sc.Bands[GameEngineTypeIndex]), gameSlotList(sc.Bands[PlayerEngineTypeIndex]),
		gameSlotList(sc.Bands[ProbeWitnessTypeIndex]), len(sc.Bands[GameEntityClassNeighbour]),
		len(sc.Bands[GameEntityClassVoid]), sc.Ambiguous, sc.Filled)
	t.Logf("SONDE ti=4 high-frequency : %d annonces recues par le hook pendant ce balayage",
		sc.ProbeWitness)
	gameLogKeyframeCensus(t, sc)
}

// gameLogKeyframeCensus journalise QUELS archetypes les images-cles portent, et avec combien
// de slots. Sans ce recensement, une bande vide ne se distingue pas d'un archetype absent.
func gameLogKeyframeCensus(t *testing.T, sc GameEntityScan) {
	t.Helper()
	tis := make([]int, 0, len(sc.KeyframeTICensus))
	for ti := range sc.KeyframeTICensus {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	var parts []string
	for _, ti := range tis {
		parts = append(parts, fmt.Sprintf("ti=%d:%d", ti, sc.KeyframeTICensus[ti]))
	}
	t.Logf("RECENSEMENT DES IMAGES-CLES (slots distincts par archetype) : %s",
		strings.Join(parts, " "))
	slots := make([]uint32, 0, len(sc.KeyframeSlotTI))
	for s := range sc.KeyframeSlotTI {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var low []string
	for i, s := range slots {
		if i >= 24 {
			break
		}
		low = append(low, fmt.Sprintf("%d->%v", s, sc.KeyframeSlotTI[s]))
	}
	t.Logf("PREMIERS SLOTS DES IMAGES-CLES : %s", strings.Join(low, " "))
}

// gameSlotList rend la liste triee des slots d'une bande (bornee, pour rester lisible).
func gameSlotList(band map[uint32]bool) string {
	s := make([]uint32, 0, len(band))
	for k := range band {
		s = append(s, k)
	}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	if len(s) == 0 {
		return "(vide)"
	}
	if len(s) > 12 {
		return fmt.Sprintf("%d slots [%d .. %d]", len(s), s[0], s[len(s)-1])
	}
	return fmt.Sprintf("%d slots %v", len(s), s)
}

func gameLogClass(t *testing.T, sc GameEntityScan, class int, label string) {
	t.Helper()
	st := sc.Stats[class]
	if st == nil {
		t.Logf("%s : AUCUNE statistique (archetype absent du registre)", label)
		return
	}
	pur, ok := st.Purity()
	pure := "n/a"
	if ok {
		pure = fmt.Sprintf("%.2f %%", 100*pur)
	}
	t.Logf("%s · bande %d slots · peuples %d · records %d (%.0f/slot) · DANS LA GRAMMAIRE %d "+
		"· avec un composant vise %d · marche ABOUTIE %d · CASSEE %d · purete %s "+
		"(hors grammaire %d / %d composants) · plancher de bruit %.1f",
		label, st.BandSize, st.Slots, st.Records, st.RecordsPerSlot(), st.InGrammar,
		st.WithWanted, st.Walked, st.Broken, pure, st.OutOfGrammar, st.GrammarLen,
		st.NoiseFloor())
	gameLogCensus(t, sc, class, st)
}

// gameLogCensus journalise le recensement d'annonces au masque, avec le facteur au-dessus du
// plancher de bruit de la bande. Un composant qui ne se detache pas de ce plancher n'est pas
// mesure : il est indistinguable du bruit, et c'est ce qu'il faut ecrire.
func gameLogCensus(t *testing.T, sc GameEntityScan, class int, st *GameEntityStats) {
	t.Helper()
	floor := st.NoiseFloor()
	arch := gameArchOf(sc, class)
	type row struct {
		i, n int
	}
	var rows []row
	for i := 0; i < worldObjectMaxComponent; i++ {
		if st.MaskCensus[i] > 0 {
			rows = append(rows, row{i, st.MaskCensus[i]})
		}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].n > rows[b].n })
	if len(rows) > 20 {
		rows = rows[:20]
	}
	for _, r := range rows {
		x := 0.0
		if floor > 0 {
			x = float64(r.n) / floor
		}
		t.Logf("    i%-2d %-52s annonces %7d  x%.1f du plancher", r.i, arch[r.i], r.n, x)
	}
}

// gameArchOf rend les noms de composants de l'archetype d'une classe (vide pour un controle).
func gameArchOf(sc GameEntityScan, class int) map[int]string {
	out := map[int]string{}
	if class < 0 {
		return out
	}
	dir := os.Getenv(gameFilmEnv)
	reg, err := gameEntityRegistry(dir)
	if err != nil {
		return out
	}
	arch, ok := reg.Archetype(class)
	if !ok {
		return out
	}
	for i, n := range arch.Components {
		out[i] = n
	}
	return out
}

func gameLogControls(t *testing.T, sc GameEntityScan) {
	t.Helper()
	for _, c := range []struct {
		class int
		label string
	}{
		{GameEntityClassNeighbour, "CONTROLE voisinage"},
		{GameEntityClassVoid, "CONTROLE vide (haut de l'espace)"},
	} {
		st := sc.Stats[c.class]
		if st == nil {
			continue
		}
		t.Logf("%s · bande %d slots · peuples %d · records %d (%.0f/slot) · DANS LA GRAMMAIRE "+
			"ti=5 %d (%.0f/slot) · avec un composant vise %d (%.0f/slot) · plancher %.1f",
			c.label, st.BandSize, st.Slots, st.Records, st.RecordsPerSlot(), st.InGrammar,
			gamePerSlot(st.InGrammar, st.Slots), st.WithWanted,
			gamePerSlot(st.WithWanted, st.Slots), st.NoiseFloor())
	}
	gameLogRatios(t, sc)
}

// gameLogRatios rend le RAPPORT REEL / FANTOME, la grandeur qui dit si une bande a mesure
// quelque chose. Un rapport de 1 signifie que l'archetype n'est pas distinguable du bruit.
func gameLogRatios(t *testing.T, sc GameEntityScan) {
	t.Helper()
	nb, vd := sc.Stats[GameEntityClassNeighbour], sc.Stats[GameEntityClassVoid]
	for _, ti := range []int{GameEngineTypeIndex, PlayerEngineTypeIndex, ProbeWitnessTypeIndex} {
		st := sc.Stats[ti]
		if st == nil || st.RecordsPerSlot() == 0 {
			continue
		}
		t.Logf("RAPPORT REEL/FANTOME ti=%d · EN-TETES BRUTS %.0f/slot contre %.0f (voisinage) "+
			"et %.0f (vide) -> x%.2f et x%.2f", ti, st.RecordsPerSlot(), gameRPS(nb),
			gameRPS(vd), gameRatio(st.RecordsPerSlot(), gameRPS(nb)),
			gameRatio(st.RecordsPerSlot(), gameRPS(vd)))
		t.Logf("RAPPORT REEL/FANTOME ti=%d · COMPOSANT VISE %.1f/slot contre %.1f (voisinage) "+
			"et %.1f (vide) -> x%.2f et x%.2f", ti, gameWantedPerSlot(st),
			gameWantedPerSlot(nb), gameWantedPerSlot(vd),
			gameRatio(gameWantedPerSlot(st), gameWantedPerSlot(nb)),
			gameRatio(gameWantedPerSlot(st), gameWantedPerSlot(vd)))
	}
}

func gameRPS(st *GameEntityStats) float64 {
	if st == nil {
		return 0
	}
	return st.RecordsPerSlot()
}

// gamePerSlot rend un compte ramene au slot peuple.
func gamePerSlot(n, slots int) float64 {
	if slots == 0 {
		return 0
	}
	return float64(n) / float64(slots)
}

// gameWantedPerSlot rend le debit de records PORTANT UN COMPOSANT VISE — la seule grandeur
// dont le rapport reel/fantome dit quelque chose sur la mesure reellement faite.
func gameWantedPerSlot(st *GameEntityStats) float64 {
	if st == nil {
		return 0
	}
	return gamePerSlot(st.WithWanted, st.Slots)
}

func gameRatio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b
}

func gameLogEngineFields(t *testing.T, sc GameEntityScan) {
	t.Helper()
	st := sc.Stats[GameEngineTypeIndex]
	if st == nil {
		return
	}
	t.Logf("ti=0 CHAMPS (records dont la marche a abouti : %d)", st.Walked)
	for f := 0; f < GameEngineFieldCount; f++ {
		t.Logf("    %-48s masque %6d · LU %6d · porte fermee %6d",
			GameEngineField(f), st.WithField[f], st.Read[f], st.Gated[f])
	}
	n := 0
	for _, r := range sc.Engine {
		if r.HasRoundTimer {
			n++
		}
	}
	t.Logf("    %-48s CAPTURE : %d records portent une horloge de manche",
		compGameEngineRoundTimer, n)
}

func gameLogPlayerFields(t *testing.T, sc GameEntityScan) {
	t.Helper()
	st := sc.Stats[PlayerEngineTypeIndex]
	if st == nil {
		return
	}
	t.Logf("ti=5 CHAMPS (records dont la marche a abouti : %d)", st.Walked)
	for f := 0; f < PlayerStateFieldCount; f++ {
		t.Logf("    %-48s masque %6d · LU %6d · porte fermee %6d",
			PlayerStateField(f), st.WithField[f], st.Read[f], st.Gated[f])
	}
	n := 0
	for _, r := range sc.Player {
		if r.HasRespawn {
			n++
		}
	}
	t.Logf("    %-48s CAPTURE : %d records portent un compte a rebours",
		compPlayerRespawnTimer, n)
}

// gameDumpTSV depose les records lus sous GAME_OUT, un fichier par archetype. Sans GAME_OUT,
// rien n'est ecrit : l'instrument reste lisible dans son seul journal.
func gameDumpTSV(t *testing.T, short string, sc GameEntityScan) {
	t.Helper()
	out := os.Getenv(gameOutEnv)
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("repertoire de sortie %s : %v", out, err)
	}
	gameWriteTSV(t, filepath.Join(out, short+"_ti0.tsv"),
		"chunk\tpacket\tt_us\tslot\tmask\tstate\tround\trt_a\trt_b\trt_qa\trt_qb\trt_tail\tsd_a\tsd_b\tsd_c\tgrace_a\tgrace_b\tgrace_c\tcond",
		gameEngineRows(sc))
	gameWriteTSV(t, filepath.Join(out, short+"_ti5.tsv"),
		"chunk\tpacket\tt_us\tslot\tmask\trespawn_active\trespawn_t0\trespawn_t1\tsoftkill\ttracking\tdesired_player\tloadout\tresp_loc\tlives\tbetrayer\taiming\tactive\tjoining\tmalleable",
		gamePlayerRows(sc))
	t.Logf("TSV deposes dans %s", out)
}

func gameWriteTSV(t *testing.T, path, header string, rows []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("ecriture %s : %v", path, err)
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, header); err != nil {
		t.Fatalf("ecriture %s : %v", path, err)
	}
	for _, r := range rows {
		if _, err := fmt.Fprintln(f, r); err != nil {
			t.Fatalf("ecriture %s : %v", path, err)
		}
	}
}

func gameEngineRows(sc GameEntityScan) []string {
	out := make([]string, 0, len(sc.Engine))
	for _, r := range sc.Engine {
		rt := []string{"", "", "", "", ""}
		if r.HasRoundTimer {
			rt = []string{
				fmt.Sprintf("%.3f", r.RoundTimer.A), fmt.Sprintf("%.3f", r.RoundTimer.B),
				fmt.Sprintf("%d", r.RoundTimer.QA), fmt.Sprintf("%d", r.RoundTimer.QB),
				fmt.Sprintf("%d", r.RoundTimer.Tail),
			}
		}
		sd := gameVals(r, GameEngineSuddenDeath, 3)
		gr := gameVals(r, GameEngineGracePeriod, 3)
		out = append(out, strings.Join([]string{
			fmt.Sprint(r.Chunk), fmt.Sprint(r.PacketIndex), fmt.Sprint(r.TimestampUS),
			fmt.Sprint(r.Slot), gameIdxString(r.Idx),
			gameVal1(r, GameEngineState), gameVal1(r, GameEngineRound),
			rt[0], rt[1], rt[2], rt[3], rt[4],
			sd[0], sd[1], sd[2], gr[0], gr[1], gr[2],
			gameVal1(r, GameEngineRoundConditions),
		}, "\t"))
	}
	return out
}

func gamePlayerRows(sc GameEntityScan) []string {
	out := make([]string, 0, len(sc.Player))
	for _, r := range sc.Player {
		ra, r0, r1 := "", "", ""
		if r.HasRespawn {
			ra, r0, r1 = fmt.Sprint(r.Respawn.Active), fmt.Sprint(r.Respawn.T0), fmt.Sprint(r.Respawn.T1)
		}
		out = append(out, strings.Join([]string{
			fmt.Sprint(r.Chunk), fmt.Sprint(r.PacketIndex), fmt.Sprint(r.TimestampUS),
			fmt.Sprint(r.Slot), gameIdxString(r.Idx), ra, r0, r1,
			gamePlayerVals(r, PlayerSoftKill), gamePlayerVals(r, PlayerTargetTracking),
			gamePlayerVals(r, PlayerDesiredRespawnPlayer), gamePlayerVals(r, PlayerLoadout),
			gamePlayerVals(r, PlayerDesiredRespawnLocation), gamePlayerVals(r, PlayerLives),
			gamePlayerVals(r, PlayerLastBetrayer), gamePlayerVals(r, PlayerControlAiming),
			gamePlayerVals(r, PlayerActiveInGame), gamePlayerVals(r, PlayerPendingJoinInProgress),
			gamePlayerVals(r, PlayerMalleableProperties),
		}, "\t"))
	}
	return out
}

// gameIdxString rend le masque sous forme lisible (i2|i5|i6).
func gameIdxString(idx []int) string {
	parts := make([]string, 0, len(idx))
	for _, i := range idx {
		parts = append(parts, fmt.Sprintf("i%d", i))
	}
	return strings.Join(parts, "|")
}

// gameVal1 rend la premiere valeur d'un champ de ti=0, ou "" (non vu / porte fermee).
func gameVal1(r GameEntityRecord, f GameEngineField) string {
	if !r.EngineSeen[f] || !r.EnginePresent[f] || len(r.EngineVal[f]) == 0 {
		return ""
	}
	return fmt.Sprint(r.EngineVal[f][0])
}

// gameVals rend les n premieres valeurs d'un champ de ti=0.
func gameVals(r GameEntityRecord, f GameEngineField, n int) []string {
	out := make([]string, n)
	if !r.EngineSeen[f] || !r.EnginePresent[f] {
		return out
	}
	for i := 0; i < n && i < len(r.EngineVal[f]); i++ {
		out[i] = fmt.Sprint(r.EngineVal[f][i])
	}
	return out
}

// gamePlayerVals rend les valeurs d'un champ de ti=5, separees par des virgules. Une porte
// fermee rend "x" et non "0" : confondre les deux fabriquerait des transitions inexistantes.
func gamePlayerVals(r GameEntityRecord, f PlayerStateField) string {
	if !r.PlayerSeen[f] {
		return ""
	}
	if !r.PlayerPresent[f] && len(r.PlayerVal[f]) == 0 {
		return "x"
	}
	parts := make([]string, 0, len(r.PlayerVal[f]))
	for _, v := range r.PlayerVal[f] {
		parts = append(parts, fmt.Sprint(v))
	}
	if !r.PlayerPresent[f] {
		return "x:" + strings.Join(parts, ",")
	}
	return strings.Join(parts, ",")
}
