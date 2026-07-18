package media

// hls_audio_migrate.go — migration EN PLACE des arbres HLS multipistes legacy
// dont le groupe audio mélange des codecs (ex. `game` en Opus copié,
// `voices`/`full` en AAC). Ré-encode chaque rendition non-AAC vers AAC pour que
// le groupe devienne mono-codec : la bascule de piste redevient fiable sur
// Firefox (pas de SourceBuffer.changeType) et lisible par le HLS natif Safari.
//
// Les sources d'origine étant supprimées après le transcodage initial, on ne
// peut pas régénérer l'arbre depuis la source : on ré-encode la rendition Opus
// EXISTANTE (sous-playlist HLS) vers une nouvelle rendition AAC (temp + swap).
// La vidéo et les renditions déjà AAC ne sont pas touchées ; le master non plus
// (l'URI de la sous-playlist est inchangée). Idempotent.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// audioRenditionRef décrit une rendition audio référencée par le master.
type audioRenditionRef struct {
	Slug string // game/voices/full/a0 — dérivé du nom stream_<slug>.m3u8
	URI  string // ex "stream_game.m3u8" (relatif au dossier de l'arbre)
}

// AudioMigrationResult résume la migration d'un arbre HLS.
type AudioMigrationResult struct {
	Dir           string
	Converted     []string // slugs ré-encodés vers AAC (en dry-run : ceux qui le seraient)
	AlreadyAAC    bool     // groupe déjà mono-codec AAC → rien à faire
	NotMultiTrack bool     // < 2 renditions audio → hors périmètre (pas de switch de piste)
}

var (
	extXMediaAudioRe = regexp.MustCompile(`(?m)^#EXT-X-MEDIA:.*TYPE=AUDIO.*$`)
	uriAttrRe        = regexp.MustCompile(`URI="([^"]+)"`)
)

// parseMasterAudioRenditions extrait les renditions audio (URI + slug) d'un
// master.m3u8. Pure. Les lignes EXT-X-STREAM-INF (variante vidéo) sont ignorées
// car elles ne portent pas l'attribut URI=".
func parseMasterAudioRenditions(master string) []audioRenditionRef {
	var out []audioRenditionRef
	for _, line := range extXMediaAudioRe.FindAllString(master, -1) {
		m := uriAttrRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, audioRenditionRef{Slug: renditionSlugFromURI(m[1]), URI: m[1]})
	}
	return out
}

// renditionSlugFromURI dérive le slug du nom de fichier stream_<slug>.m3u8.
func renditionSlugFromURI(uri string) string {
	base := strings.TrimSuffix(filepath.Base(uri), ".m3u8")
	return strings.TrimPrefix(base, "stream_")
}

// renditionIsAAC ffprobe l'init segment (autonome : son moov porte la config
// codec, robuste cross-plateforme) et indique si l'audio est déjà en AAC.
func renditionIsAAC(ctx context.Context, initPath string) (bool, error) {
	streams, err := ProbeStreamsDetailed(ctx, initPath)
	if err != nil {
		return false, err
	}
	for _, s := range streams {
		if s.CodecType == "audio" {
			return strings.EqualFold(s.CodecName, "aac"), nil
		}
	}
	return false, fmt.Errorf("renditionIsAAC: aucune piste audio dans %q", initPath)
}

// MigrateHLSAudioToAAC rend mono-codec (AAC) le groupe audio d'un arbre HLS
// multipiste legacy. Idempotent : ne convertit que les renditions non-AAC. Ne
// touche ni la vidéo ni le master. dryRun=true rapporte sans rien écrire.
func MigrateHLSAudioToAAC(ctx context.Context, hlsDir string, dryRun bool) (AudioMigrationResult, error) {
	res := AudioMigrationResult{Dir: hlsDir}
	raw, err := os.ReadFile(filepath.Join(hlsDir, "master.m3u8"))
	if err != nil {
		return res, fmt.Errorf("lecture master: %w", err)
	}
	renditions := parseMasterAudioRenditions(string(raw))
	if len(renditions) < 2 {
		res.NotMultiTrack = true // mono-piste : pas de sélecteur, hors périmètre
		return res, nil
	}

	var toConvert []audioRenditionRef
	for _, ref := range renditions {
		ok, err := renditionIsAAC(ctx, filepath.Join(hlsDir, "init_"+ref.Slug+".mp4"))
		if err != nil {
			return res, fmt.Errorf("détection codec %s: %w", ref.Slug, err)
		}
		if !ok {
			toConvert = append(toConvert, ref)
		}
	}
	if len(toConvert) == 0 {
		res.AlreadyAAC = true
		return res, nil
	}
	for _, ref := range toConvert {
		res.Converted = append(res.Converted, ref.Slug)
		if dryRun {
			continue
		}
		if err := reencodeRenditionToAAC(ctx, hlsDir, ref, defaultSegmentDuration); err != nil {
			return res, err
		}
	}
	return res, nil
}

// reencodeRenditionToAAC ré-encode une rendition audio (Opus/autre) vers AAC.
// ffmpeg écrit dans un sous-dossier temporaire ; on PROUVE que la sortie est
// AAC avant tout swap destructif, puis on remplace les fichiers de la rendition.
func reencodeRenditionToAAC(ctx context.Context, dir string, ref audioRenditionRef, segDur int) error {
	tmp := filepath.Join(dir, ".aacmig-"+ref.Slug)
	if err := os.RemoveAll(tmp); err != nil {
		return fmt.Errorf("nettoyage tmp %s: %w", ref.Slug, err)
	}
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return fmt.Errorf("création tmp %s: %w", ref.Slug, err)
	}
	defer os.RemoveAll(tmp)

	// Forward-slash pour ffmpeg (sur Windows, un chemin backslash casse la
	// résolution des segments relatifs de la playlist d'entrée — cf. hls.go).
	tmpSlash := filepath.ToSlash(tmp)
	out := filepath.Base(ref.URI)
	args := ffmpegQuietArgs(
		"-y",
		"-i", filepath.ToSlash(filepath.Join(dir, ref.URI)),
		"-map", "0:a:0", "-c:a", "aac", "-b:a", aacRenditionBitrate,
		"-f", "hls", "-hls_segment_type", "fmp4", "-hls_playlist_type", "vod",
		"-hls_time", strconv.Itoa(segDur), "-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init_"+ref.Slug+".mp4",
		"-hls_segment_filename", tmpSlash+"/seg_"+ref.Slug+"_%03d.m4s",
		tmpSlash+"/"+out,
	)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg ré-encode %s: %w (stderr: %s)", ref.Slug, err, stderr.String())
	}
	if ok, err := renditionIsAAC(ctx, filepath.Join(tmp, "init_"+ref.Slug+".mp4")); err != nil || !ok {
		return fmt.Errorf("ré-encode %s: sortie non-AAC ou illisible (err=%v)", ref.Slug, err)
	}
	return swapRendition(dir, tmp, ref)
}

// swapRendition remplace les fichiers d'une rendition par ceux générés dans tmp.
// Supprime l'ancien init + les anciens segments (leur nombre peut différer après
// réencodage), déplace les nouveaux, et la sous-playlist EN DERNIER (elle ne doit
// référencer des segments présents qu'une fois ceux-ci en place). Mêmes noms de
// fichiers (init_<slug>.mp4 / seg_<slug>_*.m4s) → le master reste valide.
func swapRendition(dir, tmp string, ref audioRenditionRef) error {
	olds, _ := filepath.Glob(filepath.Join(dir, "seg_"+ref.Slug+"_*.m4s"))
	for _, f := range olds {
		_ = os.Remove(f)
	}
	_ = os.Remove(filepath.Join(dir, "init_"+ref.Slug+".mp4"))

	entries, err := os.ReadDir(tmp)
	if err != nil {
		return err
	}
	playlist := filepath.Base(ref.URI)
	for _, e := range entries {
		if e.Name() == playlist {
			continue // déplacée en dernier
		}
		if err := os.Rename(filepath.Join(tmp, e.Name()), filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("swap %s: %w", e.Name(), err)
		}
	}
	if _, err := os.Stat(filepath.Join(tmp, playlist)); err != nil {
		return fmt.Errorf("swap %s: sous-playlist absente du tmp: %w", ref.Slug, err)
	}
	return os.Rename(filepath.Join(tmp, playlist), filepath.Join(dir, playlist))
}
