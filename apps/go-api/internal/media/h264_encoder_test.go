package media

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// encodersLGPL : extrait de `ffmpeg -hide_banner -encoders` du build win64-lgpl-shared livré
// (pas de libx264).
const encodersLGPL = `Encoders:
 V..... = Video
 A..... = Audio
 ------
 V....D libopenh264          OpenH264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)
 V....D h264_amf             AMD AMF H.264 Encoder (codec h264)
 V....D h264_mf              H264 via MediaFoundation (codec h264)
 V....D h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V....D libwebp              libwebp WebP image (codec webp)
 A....D aac                  AAC (Advanced Audio Coding)
`

// encodersGPL : un build complet (libx264 présent, h264_mf aussi sous Windows).
const encodersGPL = `Encoders:
 ------
 V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)
 V....D h264_mf              H264 via MediaFoundation (codec h264)
 A....D aac                  AAC (Advanced Audio Coding)
`

func TestParseFFmpegComponents_LGPLListing(t *testing.T) {
	set := ParseFFmpegComponents(encodersLGPL)
	for _, want := range []string{"libopenh264", "h264_mf", "libwebp", "aac"} {
		if !set[want] {
			t.Errorf("encodeur %q attendu dans la liste parsée", want)
		}
	}
	for _, unwanted := range []string{"libx264", "=", "Encoders:"} {
		if set[unwanted] {
			t.Errorf("%q ne doit pas figurer dans la liste parsée", unwanted)
		}
	}
	if len(ParseFFmpegComponents("")) != 0 {
		t.Error("une sortie vide doit produire un ensemble vide")
	}
}

func TestH264Candidates_OrderFollowsPreference(t *testing.T) {
	cases := []struct {
		name    string
		listing string
		want    []H264Encoder
	}{
		{"build LGPL", encodersLGPL, []H264Encoder{EncoderMediaFoundation, EncoderOpenH264}},
		{"build GPL", encodersGPL, []H264Encoder{EncoderLibx264, EncoderMediaFoundation}},
		{"aucun encodeur H.264", "Encoders:\n ------\n A....D aac AAC\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := H264Candidates(ParseFFmpegComponents(tc.listing))
			if len(got) != len(tc.want) {
				t.Fatalf("candidats = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("candidats = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestH264VideoArgs_TableOfEncoders(t *testing.T) {
	cases := []struct {
		enc        H264Encoder
		wantPairs  [][2]string
		wantAbsent []string
	}{
		{EncoderLibx264, [][2]string{{"-c:v", "libx264"}, {"-preset", "veryfast"}, {"-crf", "20"}, {"-pix_fmt", "yuv420p"}},
			[]string{"-b:v", "-g", "-force_key_frames", "-rc_mode"}},
		{EncoderMediaFoundation, [][2]string{{"-c:v", "h264_mf"}, {"-b:v", "12M"}, {"-g", "240"}, {"-force_key_frames", "expr:gte(t,n_forced*4)"}, {"-pix_fmt", "yuv420p"}},
			[]string{"-preset", "-crf", "-rc_mode"}},
		{EncoderOpenH264, [][2]string{{"-c:v", "libopenh264"}, {"-rc_mode", "bitrate"}, {"-b:v", "12M"}, {"-pix_fmt", "yuv420p"}},
			[]string{"-preset", "-crf"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.enc), func(t *testing.T) {
			args := h264VideoArgs(tc.enc, 4)
			for _, p := range tc.wantPairs {
				if !argsContainPair(args, p[0], p[1]) {
					t.Errorf("%v : paire %v absente", args, p)
				}
			}
			for _, a := range tc.wantAbsent {
				for _, got := range args {
					if got == a {
						t.Errorf("%v : l'option %q n'existe pas pour %s", args, a, tc.enc)
					}
				}
			}
		})
	}
}

func TestBuildHLSArgs_UsesTheRetainedEncoderOnlyWhenReencoding(t *testing.T) {
	audios := []audioRendition{{Slug: "a0", MapSpec: "0:a:0", Action: actionCopy, Default: true}}
	re := buildHLSArgs(hlsPlan{VideoAction: actionReencode, Audios: audios}, "src.webm", "out", 4, EncoderOpenH264)
	if !argsContainPair(re, "-c:v", "libopenh264") {
		t.Errorf("réencodage : encodeur retenu absent\n%v", re)
	}
	cp := buildHLSArgs(hlsPlan{VideoAction: actionCopy, VideoSrcCodec: "h264", Audios: audios}, "src.mkv", "out", 4, "")
	if !argsContainPair(cp, "-c:v", "copy") {
		t.Errorf("copie : -c:v copy attendu\n%v", cp)
	}
}

func TestChooseH264_FallsBackAndLogsRefusals(t *testing.T) {
	probed := []H264Encoder{}
	probe := func(_ context.Context, enc H264Encoder) error {
		probed = append(probed, enc)
		if enc == EncoderMediaFoundation {
			return errors.New("Media Foundation indisponible")
		}
		return nil
	}
	got, err := chooseH264(context.Background(), []H264Encoder{EncoderMediaFoundation, EncoderOpenH264}, probe)
	if err != nil || got != EncoderOpenH264 {
		t.Fatalf("chooseH264 = %q, %v ; attendu libopenh264", got, err)
	}
	if len(probed) != 2 {
		t.Errorf("sondes = %v, attendu les deux candidats dans l'ordre", probed)
	}
}

func TestChooseH264_NoneUsableIsATypedError(t *testing.T) {
	refuse := func(context.Context, H264Encoder) error { return errors.New("refus") }
	if _, err := chooseH264(context.Background(), []H264Encoder{EncoderLibx264}, refuse); !errors.Is(err, ErrNoH264Encoder) {
		t.Errorf("tous refusés : err = %v, attendu ErrNoH264Encoder", err)
	}
	if _, err := chooseH264(context.Background(), nil, refuse); !errors.Is(err, ErrNoH264Encoder) {
		t.Errorf("aucun candidat : err = %v, attendu ErrNoH264Encoder", err)
	}
}

// stubEncodersListing remplace le lanceur par le processus d'aide : avec `-encoders` il écrit
// listing ; sans, c'est un essai d'encodage, accepté (sortie 0) ou refusé selon probeFails. Le
// résolveur est dirigé vers un faux binaire pour que le chemin résolu existe. Rend les compteurs
// de listes et d'essais lancés.
func stubEncodersListing(t *testing.T, listing string, probeFails bool) (listings, probes *int) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "ffmpeg"+binarySuffix())
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvFFmpegDir, filepath.Dir(bin))
	resetCapabilitiesCache()
	t.Cleanup(resetCapabilitiesCache)
	listings, probes = new(int), new(int)
	prev := ffmpegCommand
	t.Cleanup(func() { ffmpegCommand = prev })
	ffmpegCommand = func(ctx context.Context, args []string) *exec.Cmd {
		mode := "out"
		text := ""
		if hasArg(args, "-encoders") {
			*listings++
			text = listing
		} else {
			*probes++
			if probeFails {
				mode, text = "err", "encodeur indisponible"
			}
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestFFmpegHelperProcess", "--", mode, text)
		cmd.Env = append(os.Environ(), "FFMPEG_HELPER_PROCESS=1")
		return cmd
	}
	return listings, probes
}

func hasArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestSelectedH264Encoder_ListsOnceAndCaches(t *testing.T) {
	listings, probes := stubEncodersListing(t, encodersLGPL, false)
	for i := 0; i < 3; i++ {
		enc, err := SelectedH264Encoder(context.Background())
		if err != nil || enc != EncoderMediaFoundation {
			t.Fatalf("appel %d : %q, %v ; attendu h264_mf", i, enc, err)
		}
	}
	if *listings != 1 || *probes != 1 {
		t.Errorf("liste lancée %d fois, sonde %d fois ; attendu une fois chacune", *listings, *probes)
	}
}

func TestSelectedH264Encoder_NoEncoderAtAll(t *testing.T) {
	stubEncodersListing(t, "Encoders:\n ------\n A....D aac AAC\n", false)
	if _, err := SelectedH264Encoder(context.Background()); !errors.Is(err, ErrNoH264Encoder) {
		t.Errorf("err = %v, attendu ErrNoH264Encoder", err)
	}
}

func TestSelectedH264Encoder_ListedButUnusableIsRefusedNotCached(t *testing.T) {
	_, probes := stubEncodersListing(t, encodersLGPL, true)
	for i := 0; i < 2; i++ {
		if _, err := SelectedH264Encoder(context.Background()); !errors.Is(err, ErrNoH264Encoder) {
			t.Fatalf("appel %d : err = %v, attendu ErrNoH264Encoder", i, err)
		}
	}
	if *probes != 4 { // deux candidats par appel, un échec n'est pas gardé
		t.Errorf("sondes = %d, attendu 4 (un échec n'est pas gardé)", *probes)
	}
}
