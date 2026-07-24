package media

import (
	"context"
	"strings"
	"testing"
)

// vid + n pistes audio AAC (copy direct — le codec n'affecte pas le FilterComplex).
func manualStreams(n int) []AVStreamDetail {
	s := []AVStreamDetail{{CodecType: "video", CodecName: "h264"}}
	for i := 0; i < n; i++ {
		s = append(s, AVStreamDetail{CodecType: "audio", CodecName: "aac"})
	}
	return s
}

func TestPlanAudioRenditionsManual_GameTrack0(t *testing.T) {
	// jeu = piste 0, voix = pistes 1 & 2 → voices = amix(1,2), full = amix(0,1,2).
	src := manualStreams(3)[1:]
	audios, fc := planAudioRenditionsManual(src, []string{"game", "voice", "other"})
	if len(audios) != 3 {
		t.Fatalf("len(audios) = %d, want 3", len(audios))
	}
	if audios[0].Slug != "game" || audios[1].Slug != "voices" || audios[2].Slug != "full" {
		t.Fatalf("slugs = [%q,%q,%q], want [game,voices,full]", audios[0].Slug, audios[1].Slug, audios[2].Slug)
	}
	if audios[0].MapSpec != "0:a:0" { // jeu = piste directe 0
		t.Errorf("game MapSpec = %q, want 0:a:0", audios[0].MapSpec)
	}
	if audios[1].MapSpec != "[voices]" || audios[2].MapSpec != "[full]" {
		t.Errorf("voices/full MapSpec = %q/%q, want [voices]/[full]", audios[1].MapSpec, audios[2].MapSpec)
	}
	if !audios[2].Default || audios[0].Default || audios[1].Default {
		t.Errorf("Default = [%v,%v,%v], want [false,false,true]", audios[0].Default, audios[1].Default, audios[2].Default)
	}
	want := strings.Join([]string{amixFilter([]int{1, 2}, "voices"), amixFilter([]int{0, 1, 2}, "full")}, ";")
	if fc != want {
		t.Errorf("FilterComplex =\n  %q\nwant\n  %q", fc, want)
	}
}

func TestPlanAudioRenditionsManual_GameNotTrack0(t *testing.T) {
	// Cas IMPOSSIBLE à exprimer via audioLayout : jeu = piste 2, voix = pistes 0 & 1.
	src := manualStreams(3)[1:]
	audios, fc := planAudioRenditionsManual(src, []string{"voice", "other", "game"})
	if audios[0].Slug != "game" || audios[0].MapSpec != "0:a:2" {
		t.Errorf("game = (%q,%q), want (game,0:a:2)", audios[0].Slug, audios[0].MapSpec)
	}
	want := strings.Join([]string{amixFilter([]int{0, 1}, "voices"), amixFilter([]int{0, 1, 2}, "full")}, ";")
	if fc != want {
		t.Errorf("FilterComplex =\n  %q\nwant\n  %q", fc, want)
	}
}

func TestPlanAudioRenditionsManual_MultipleGameTracks(t *testing.T) {
	// jeu = pistes 0 & 2 → game = amix(0,2) ; voix = piste 1 (directe).
	src := manualStreams(3)[1:]
	audios, fc := planAudioRenditionsManual(src, []string{"game", "voice", "game"})
	if audios[0].MapSpec != "[game]" || audios[1].MapSpec != "0:a:1" {
		t.Errorf("game/voices MapSpec = %q/%q, want [game]/0:a:1", audios[0].MapSpec, audios[1].MapSpec)
	}
	want := strings.Join([]string{
		amixFilter([]int{0, 2}, "game"),
		amixFilter([]int{0, 1, 2}, "full"),
	}, ";")
	if fc != want {
		t.Errorf("FilterComplex =\n  %q\nwant\n  %q", fc, want)
	}
}

func TestPlanAudioRenditionsManual_SingleCategoryDegradesToFullMix(t *testing.T) {
	// Toutes les pistes en "game" → aucune séparation possible → mix complet unique.
	src := manualStreams(2)[1:]
	audios, fc := planAudioRenditionsManual(src, []string{"game", "game"})
	if len(audios) != 1 {
		t.Fatalf("len(audios) = %d, want 1 (mix complet unique)", len(audios))
	}
	if audios[0].Slug != "a0" || audios[0].MapSpec != "[full]" || !audios[0].Default {
		t.Errorf("rendition = (%q,%q,default=%v), want (a0,[full],true)", audios[0].Slug, audios[0].MapSpec, audios[0].Default)
	}
	if fc != amixFilter([]int{0, 1}, "full") {
		t.Errorf("FilterComplex = %q", fc)
	}
}

func TestPlanAudioRenditionsManual_MonoTrack(t *testing.T) {
	src := manualStreams(1)[1:]
	audios, fc := planAudioRenditionsManual(src, []string{"game"})
	if len(audios) != 1 || audios[0].Slug != "a0" || audios[0].MapSpec != "0:a:0" || fc != "" {
		t.Errorf("mono = (%d,%q,%q,fc=%q), want (1,a0,0:a:0,'')", len(audios), audios[0].Slug, audios[0].MapSpec, fc)
	}
}

func TestPlanHLSManual_EndToEnd(t *testing.T) {
	// jeu = piste 1, voix = pistes 0 & 2 → VarStreamMap ordonné game/voices/full.
	plan, err := planHLSManual(manualStreams(3), []string{"voice", "game", "voice"})
	if err != nil {
		t.Fatalf("planHLSManual: %v", err)
	}
	if len(plan.Audios) != 3 || plan.Audios[0].MapSpec != "0:a:1" {
		t.Fatalf("game map = %q (want 0:a:1), n=%d", plan.Audios[0].MapSpec, len(plan.Audios))
	}
	if !strings.Contains(plan.VarStreamMap, "name:game") || !strings.Contains(plan.VarStreamMap, "name:full,default:yes") {
		t.Errorf("VarStreamMap = %q", plan.VarStreamMap)
	}
}

func TestBuildHLSPlan_ManualBypassNoAnalyze(t *testing.T) {
	// Rôles cohérents (3 == 3 pistes) → mapping manuel (aucun ffmpeg / analyse).
	streams := manualStreams(3)
	audios := audioStreamsOnly(streams)
	plan, err := buildHLSPlan(context.Background(), "unused.mkv", streams, audios,
		HLSOptions{ManualAudioRoles: []string{"voice", "game", "voice"}})
	if err != nil {
		t.Fatalf("buildHLSPlan manuel: %v", err)
	}
	// Preuve du bypass : le jeu est la piste 1 (l'auto/audioLayout mettrait game=0:a:0).
	if plan.Audios[0].Slug != "game" || plan.Audios[0].MapSpec != "0:a:1" {
		t.Errorf("game = (%q,%q), want (game,0:a:1)", plan.Audios[0].Slug, plan.Audios[0].MapSpec)
	}
}

func TestBuildHLSPlan_MismatchFallsBackToAuto(t *testing.T) {
	// Rôles incohérents (2 rôles pour 1 piste audio) → fallback auto. 1 piste audio
	// (< 2) court-circuite l'analyse NNLS → planHLS pur, aucun ffmpeg requis.
	streams := manualStreams(1)
	audios := audioStreamsOnly(streams)
	plan, err := buildHLSPlan(context.Background(), "unused.mkv", streams, audios,
		HLSOptions{ManualAudioRoles: []string{"game", "voice"}})
	if err != nil {
		t.Fatalf("buildHLSPlan fallback: %v", err)
	}
	if len(plan.Audios) != 1 || plan.Audios[0].Slug != "a0" {
		t.Errorf("fallback auto = %d renditions (slug %q), want 1 (a0)", len(plan.Audios), plan.Audios[0].Slug)
	}
}
