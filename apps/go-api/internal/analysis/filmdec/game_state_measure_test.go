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

// gameShort8 rend le nom court du film (le repertoire).
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
