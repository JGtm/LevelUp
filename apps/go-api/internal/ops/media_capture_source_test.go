package ops

// media_capture_source_test.go — la règle par source : une heure de capture dit le DÉBUT ou la
// FIN de la capture selon le logiciel qui l'a écrite, et la fenêtre stockée en découle.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustParis(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	return loc
}

// TestCaptureSourceOfName : chaque motif de nom est rangé dans SA source, avec son ancre.
func TestCaptureSourceOfName(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		anchor captureAnchor
	}{
		{"Replay 2026-09-01 21-46-53.mkv", captureSourceOBSReplay, captureAnchorEnd},
		{"replay 2026-09-01 21-46-53.mp4", captureSourceOBSReplay, captureAnchorEnd},
		{"2026-09-01 21-46-53.mkv", captureSourceOBSRecording, captureAnchorStart},
		{"Halo Infinite 2025-12-18 17-40-46.mp4", captureSourceGameBar, captureAnchorEnd},
		{"Recording 2026-01-15 09-00-00.mp4", captureSourceGameBar, captureAnchorEnd},
		{"Halo Infinite 2024.11.15 - 21.30.45.01.DVR.mp4", captureSourceShadowPlayReplay, captureAnchorEnd},
		{"Halo Infinite 2024.11.15 - 21.30.45.01.mp4", captureSourceShadowPlayRecording, captureAnchorStart},
		{"Halo_5_Guardians-2019-12-12_22h49.mp4", captureSourceXboxConsole, captureAnchorEnd},
	}
	for _, c := range cases {
		src, ok := captureSourceOfName(c.name)
		if !ok {
			t.Errorf("%q : aucune source reconnue, want %s", c.name, c.key)
			continue
		}
		if src.Key != c.key || src.Anchor != c.anchor {
			t.Errorf("%q : source %s/%s, want %s/%s", c.name, src.Key, src.Anchor, c.key, c.anchor)
		}
	}
	for _, name := range []string{"clip.mp4", "Halo Infinite.mp4", "2024-01-01.mp4", "OBS-Recording-2024-01-01.mp4"} {
		if src, ok := captureSourceOfName(name); ok {
			t.Errorf("%q : source %s reconnue, want aucune (date du fichier)", name, src.Key)
		}
	}
}

// TestCaptureWindow_ReplayClipEndsAtItsTime : un clip « Replay » d'OBS de durée D, nommé à
// l'heure H, couvre [H-D, H] — le nom est l'instant de la sauvegarde, la fin du clip.
func TestCaptureWindow_ReplayClipEndsAtItsTime(t *testing.T) {
	loc := mustParis(t)
	inst := captureInstantFromName("Replay 2026-09-01 21-46-53.mkv", loc)
	if inst == nil {
		t.Fatal("nom OBS Replay non reconnu")
	}
	h := time.Date(2026, 9, 1, 19, 46, 53, 0, time.UTC)
	if !inst.At.Equal(h) {
		t.Fatalf("H = %v, want %v", inst.At, h)
	}
	win := captureWindow(mediaKindVideo, inst, 28.872, true)
	wantStart := h.Add(-28872 * time.Millisecond)
	if win.Start == nil || !win.Start.Equal(wantStart) {
		t.Errorf("début = %v, want %v (H - D)", win.Start, wantStart)
	}
	if win.End == nil || !win.End.Equal(h) {
		t.Errorf("fin = %v, want %v (H)", win.End, h)
	}
	if win.DurationSec == nil || *win.DurationSec != 28.872 {
		t.Errorf("durée = %v, want 28.872", win.DurationSec)
	}
}

// TestCaptureWindow_XboxRecordThatEndsAtItsTime : un « Xbox, record that » (Halo 5, console)
// nommé à la minute H couvre [H-D, H].
func TestCaptureWindow_XboxRecordThatEndsAtItsTime(t *testing.T) {
	loc := mustParis(t)
	inst := captureInstantFromName("Halo_5_Guardians-2019-12-12_22h49.mp4", loc)
	if inst == nil {
		t.Fatal("nom de clip de console non reconnu")
	}
	h := time.Date(2019, 12, 12, 21, 49, 0, 0, time.UTC)
	win := captureWindow(mediaKindVideo, inst, 30, true)
	if win.Start == nil || !win.Start.Equal(h.Add(-30*time.Second)) || win.End == nil || !win.End.Equal(h) {
		t.Errorf("fenêtre = [%v, %v], want [%v, %v]", win.Start, win.End, h.Add(-30*time.Second), h)
	}
}

// TestCaptureWindow_RecordingStartsAtItsTime : un enregistrement OBS (nom = date seule) est
// nommé à son démarrage : il couvre [H, H+D].
func TestCaptureWindow_RecordingStartsAtItsTime(t *testing.T) {
	loc := mustParis(t)
	inst := captureInstantFromName("2026-04-19 17-10-54.mkv", loc)
	if inst == nil {
		t.Fatal("nom d'enregistrement OBS non reconnu")
	}
	h := time.Date(2026, 4, 19, 15, 10, 54, 0, time.UTC)
	win := captureWindow(mediaKindVideo, inst, 600, true)
	if win.Start == nil || !win.Start.Equal(h) || win.End == nil || !win.End.Equal(h.Add(10*time.Minute)) {
		t.Errorf("fenêtre = [%v, %v], want [%v, %v]", win.Start, win.End, h, h.Add(10*time.Minute))
	}
}

// TestCaptureWindow_FileDateEndsAtItsTime : la date de modification d'un fichier est sa
// dernière écriture, donc la fin de la capture.
func TestCaptureWindow_FileDateEndsAtItsTime(t *testing.T) {
	h := time.Date(2026, 7, 3, 19, 38, 45, 0, time.UTC)
	inst := &captureInstant{At: h, Source: captureSourceByKey(captureSourceFileDate)}
	win := captureWindow(mediaKindVideo, inst, 27, true)
	if win.Start == nil || !win.Start.Equal(h.Add(-27*time.Second)) || win.End == nil || !win.End.Equal(h) {
		t.Errorf("fenêtre = [%v, %v], want [%v, %v]", win.Start, win.End, h.Add(-27*time.Second), h)
	}
}

// TestCaptureWindow_EndAnchoredWithoutDuration : sans durée, la seule heure connue est la fin ;
// elle sert de début et de fin (la frise pose le clip ponctuellement), la durée reste NULL.
func TestCaptureWindow_EndAnchoredWithoutDuration(t *testing.T) {
	h := time.Date(2026, 7, 3, 19, 38, 45, 0, time.UTC)
	inst := &captureInstant{At: h, Source: captureSourceByKey(captureSourceOBSReplay)}
	win := captureWindow(mediaKindVideo, inst, 0, false)
	if win.Start == nil || !win.Start.Equal(h) || win.End == nil || !win.End.Equal(h) {
		t.Errorf("fenêtre = [%v, %v], want [%v, %v]", win.Start, win.End, h, h)
	}
	if win.DurationSec != nil {
		t.Errorf("durée = %v, want nil (inconnue)", *win.DurationSec)
	}
}

// TestCaptureWindow_StartAnchoredWithoutDuration : un début sans durée laisse la fin NULL.
func TestCaptureWindow_StartAnchoredWithoutDuration(t *testing.T) {
	h := time.Date(2026, 7, 3, 19, 38, 45, 0, time.UTC)
	inst := &captureInstant{At: h, Source: captureSourceByKey(captureSourceOBSRecording)}
	win := captureWindow(mediaKindVideo, inst, 0, false)
	if win.Start == nil || !win.Start.Equal(h) || win.End != nil || win.DurationSec != nil {
		t.Errorf("fenêtre = [%v, %v] durée %v, want [%v, nil] durée nil", win.Start, win.End, win.DurationSec, h)
	}
}

// TestCaptureWindow_ImageIsInstant : une image est un instant, quelle que soit la source.
func TestCaptureWindow_ImageIsInstant(t *testing.T) {
	h := time.Date(2025, 1, 10, 14, 30, 0, 0, time.UTC)
	for _, key := range []string{captureSourceOBSReplay, captureSourceOBSRecording, captureSourceFileDate} {
		inst := &captureInstant{At: h, Source: captureSourceByKey(key)}
		win := captureWindow(mediaKindImage, inst, 0, false)
		if win.Start == nil || !win.Start.Equal(h) || win.End == nil || !win.End.Equal(h) {
			t.Errorf("%s : fenêtre = [%v, %v], want [%v, %v]", key, win.Start, win.End, h, h)
		}
		if win.DurationSec == nil || *win.DurationSec != 0 {
			t.Errorf("%s : durée = %v, want 0", key, win.DurationSec)
		}
	}
}

// TestCaptureWindow_NoInstant : sans heure, ni début ni fin ; la durée reste celle qu'on sait.
func TestCaptureWindow_NoInstant(t *testing.T) {
	img := captureWindow(mediaKindImage, nil, 0, false)
	if img.Start != nil || img.End != nil || img.DurationSec == nil || *img.DurationSec != 0 {
		t.Errorf("image sans heure : [%v, %v] durée %v", img.Start, img.End, img.DurationSec)
	}
	vid := captureWindow(mediaKindVideo, nil, 10, true)
	if vid.Start != nil || vid.End != nil || vid.DurationSec == nil || *vid.DurationSec != 10 {
		t.Errorf("vidéo sans heure : [%v, %v] durée %v", vid.Start, vid.End, vid.DurationSec)
	}
}

// TestCaptureWindow_UnknownKind : un format non indexé garde son heure en début, rien d'autre.
func TestCaptureWindow_UnknownKind(t *testing.T) {
	h := time.Date(2025, 1, 10, 14, 30, 0, 0, time.UTC)
	inst := &captureInstant{At: h, Source: captureSourceByKey(captureSourceOBSReplay)}
	win := captureWindow("???", inst, 5, true)
	if win.Start == nil || !win.Start.Equal(h) || win.End != nil || win.DurationSec != nil {
		t.Errorf("kind inconnu : [%v, %v] durée %v", win.Start, win.End, win.DurationSec)
	}
}

// TestResolveCaptureInstant_Priorities : nom > date client > date disque, chacune avec sa source.
func TestResolveCaptureInstant_Priorities(t *testing.T) {
	loc := mustParis(t)
	dir := t.TempDir()
	named := filepath.Join(dir, "Replay 2026-09-01 21-46-53.mkv")
	plain := filepath.Join(dir, "clip.mp4")
	mtime := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	for _, p := range []string{named, plain} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	client := int64(1_788_000_000)

	if got := resolveCaptureInstant(named, filepath.Base(named), &client, loc); got == nil ||
		got.Source.Key != captureSourceOBSReplay {
		t.Errorf("nom reconnu : %+v, want source %s", got, captureSourceOBSReplay)
	}
	got := resolveCaptureInstant(plain, filepath.Base(plain), &client, loc)
	if got == nil || got.Source.Key != captureSourceFileDate || !got.At.Equal(time.Unix(client, 0).UTC()) {
		t.Errorf("date client : %+v, want %s à %v", got, captureSourceFileDate, time.Unix(client, 0).UTC())
	}
	got = resolveCaptureInstant(plain, filepath.Base(plain), nil, loc)
	if got == nil || got.Source.Key != captureSourceFileDate || !got.At.Equal(mtime) {
		t.Errorf("date disque : %+v, want %s à %v", got, captureSourceFileDate, mtime)
	}
	if got := resolveCaptureInstant(filepath.Join(dir, "absent.mp4"), "absent.mp4", nil, loc); got != nil {
		t.Errorf("fichier absent sans nom daté : %+v, want nil", got)
	}
}

// TestCaptureSources_KeysAreUnique : la table est la seule source des clés écrites en base.
func TestCaptureSources_KeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range captureTimeSources {
		if s.Key == "" || seen[s.Key] {
			t.Errorf("clé vide ou dupliquée : %q", s.Key)
		}
		seen[s.Key] = true
		if s.Anchor != captureAnchorStart && s.Anchor != captureAnchorEnd {
			t.Errorf("%s : ancre %q invalide", s.Key, s.Anchor)
		}
	}
}

// TestConversionTimes : une conversion de format ne réécrit la fenêtre que si l'heure vient du
// nom ET que la durée de la vidéo convertie est connue.
func TestConversionTimes(t *testing.T) {
	h := time.Date(2026, 9, 1, 19, 46, 53, 0, time.UTC)
	named := &captureInstant{At: h, Source: captureSourceByKey(captureSourceOBSReplay)}
	fileDate := &captureInstant{At: h, Source: captureSourceByKey(captureSourceFileDate)}

	known := captureWindow(mediaKindVideo, named, 28, true)
	start, end, src := conversionTimes(named, mediaKindVideo, known)
	if start == nil || !start.Equal(h.Add(-28*time.Second)) || end == nil || !end.Equal(h) ||
		src == nil || *src != captureSourceOBSReplay {
		t.Errorf("nom + durée connue : [%v, %v] %v, want [H-28 s, H] %s", start, end, src, captureSourceOBSReplay)
	}
	unknown := captureWindow(mediaKindVideo, named, 0, false)
	if start, end, src := conversionTimes(named, mediaKindVideo, unknown); start != nil || end != nil || src != nil {
		t.Errorf("ffprobe en échec : [%v, %v] %v, want rien (l'ancienne fenêtre est gardée)", start, end, src)
	}
	if start, end, src := conversionTimes(fileDate, mediaKindVideo, captureWindow(mediaKindVideo, fileDate, 28, true)); start != nil || end != nil || src != nil {
		t.Errorf("date du fichier : [%v, %v] %v, want rien (date de la conversion)", start, end, src)
	}
	img := captureWindow(mediaKindImage, named, 0, false)
	if start, _, src := conversionTimes(named, mediaKindImage, img); start == nil || src == nil {
		t.Errorf("image au nom daté : début %v source %v, want l'instant et sa source", start, src)
	}
}
