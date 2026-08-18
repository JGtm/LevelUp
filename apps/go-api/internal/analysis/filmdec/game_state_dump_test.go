package filmdec

// game_state_dump_test.go — LES SORTIES SUR DISQUE de l instrument des entites non
// corporelles (`game_state_measure_test.go` porte l en-tete, le contrat et les seuils) : les
// TSV par archetype, la synthese a une ligne par film, et les cellules qui les formatent.
// Scinde du premier pour tenir le seuil de 500 lignes par fichier.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
