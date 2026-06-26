package media

// hls.go — génération d'un arbre HLS-fMP4 multipiste à l'ingestion média.
//
// Symétrique de remux.go (remux WebM live), mais produit un arbre HLS statique
// (master.m3u8 + sous-playlists + segments fMP4) servi ensuite comme fichiers
// plats. Permet la sélection de piste audio (EXT-X-MEDIA TYPE=AUDIO) et le seek
// (Range sur les segments), ce que le remux WebM ne permet pas.
//
// Policy codec (cible Chrome/Firefox/Edge via hls.js) : copy par défaut. Le
// réencodage n'est déclenché que pour les codecs incompatibles avec fMP4/HLS
// (vidéo VP8/VP9 → H.264, audio exotique → AAC). L'Opus est copié tel quel.
//
// La logique de décision (planHLS, NeedsHLS, buildVarStreamMap,
// rewriteMasterAudioNames) est pure et testable sans ffmpeg ; seuls
// ProbeStreamsDetailed et BuildHLS dépendent du binaire ffmpeg/ffprobe.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// defaultSegmentDuration est la durée cible d'un segment HLS en secondes.
const defaultSegmentDuration = 4

// aacRenditionBitrate est le débit cible du réencodage AAC des renditions audio
// (réencode amix `full`/`voices`, et migration des renditions Opus legacy).
const aacRenditionBitrate = "192k"

// AVStreamDetail décrit une piste (vidéo ou audio) retournée par ffprobe,
// enrichie des tags utiles au nommage des pistes audio dans le master.
type AVStreamDetail struct {
	Index     int
	CodecType string // "video" | "audio"
	CodecName string
	Channels  int
	Language  string // tag language (ISO 639-2), "" si absent
	Title     string // tag title (ex: "Game", "Mic"), "" si absent
}

// streamAction décrit l'opération ffmpeg appliquée à un flux.
type streamAction int

const (
	actionCopy     streamAction = iota // -c copy (remux pur)
	actionReencode                     // réencodage (codec incompatible HLS)
)

// audioRendition décrit une rendition audio de SORTIE dans le plan HLS. Une
// rendition n'est plus forcément une piste source 1:1 : elle peut provenir d'un
// label de filtre (-filter_complex amix) quand on agrège plusieurs sources.
type audioRendition struct {
	Slug     string       // identifiant fichier + name: dans var_stream_map (game/voices/full ou a0)
	Display  string       // NAME écrit dans le master (post-traité) — slug machine pour le layout 2-toggles
	MapSpec  string       // argument ffmpeg -map : "0:a:0" (source directe) ou "[voices]"/"[full]" (sortie de filtre)
	Language string       // langue propagée au master si présente
	Action   streamAction // copy ou réencode AAC
	Default  bool         // DEFAULT=YES
}

// hlsPlan est le plan de transcodage dérivé des streams source — pur.
type hlsPlan struct {
	VideoAction   streamAction
	VideoCodec    string // codec cible si réencode ("h264") ; "" si copy
	Audios        []audioRendition
	FilterComplex string // -filter_complex (amix) ; "" si aucune agrégation
	VarStreamMap  string
}

// HLSOptions configure BuildHLS.
type HLSOptions struct {
	SegmentDuration int // secondes ; défaut 4 si <= 0
}

// HLSResult résume la sortie de BuildHLS.
type HLSResult struct {
	MasterPath  string   // chemin absolu du master.m3u8
	AudioTracks int      // nombre de pistes audio exposées
	Segments    int      // nombre de segments .m4s générés
	Renditions  []string // slugs des renditions audio (game/voices/full ou a0) — observabilité
}

// NeedsHLS retourne true si le fichier doit être transcodé en HLS :
//   - container non web-natif (mkv/avi) — RequiresRemux, OU
//   - plusieurs pistes audio (la sélection de piste exige HLS).
//
// Les MP4/WebM/MOV mono-piste web-natifs restent servis en direct (inchangé).
func NeedsHLS(ext string, streams []AVStreamDetail) bool {
	if RequiresRemux(ext) {
		return true
	}
	audio := 0
	for _, s := range streams {
		if s.CodecType == "audio" {
			audio++
		}
	}
	return audio > 1
}

// countAudioStreams compte les pistes audio d'une liste de streams.
func countAudioStreams(streams []AVStreamDetail) int {
	n := 0
	for _, s := range streams {
		if s.CodecType == "audio" {
			n++
		}
	}
	return n
}

// planHLS construit le plan de transcodage à partir des streams source.
// Pure : aucune IO. Erreur si pas de piste vidéo ou audio exploitable.
//
// track0IsFullMix : décision IO calculée en amont par BuildHLS (cf.
// hls_audio_analyze.go). Quand true, la piste 0 est le mix complet de sortie et
// `full` la lit directement (pas d'amix doublé / écho).
func planHLS(streams []AVStreamDetail, track0IsFullMix bool) (hlsPlan, error) {
	var plan hlsPlan
	hasVideo := false
	var srcAudios []AVStreamDetail
	for _, s := range streams {
		switch s.CodecType {
		case "video":
			if hasVideo {
				continue // une seule piste vidéo (la première)
			}
			hasVideo = true
			plan.VideoAction, plan.VideoCodec = planVideo(s.CodecName)
		case "audio":
			srcAudios = append(srcAudios, s)
		}
	}
	if !hasVideo {
		return hlsPlan{}, fmt.Errorf("planHLS: aucune piste vidéo")
	}
	if len(srcAudios) == 0 {
		return hlsPlan{}, fmt.Errorf("planHLS: aucune piste audio")
	}
	plan.Audios, plan.FilterComplex = planAudioRenditions(srcAudios, track0IsFullMix)
	plan.VarStreamMap = buildVarStreamMap(plan)
	return plan, nil
}

// planAudioRenditions dérive les renditions de sortie des pistes source.
//
//   - mono-piste → 1 rendition directe (comportement historique, pas de toggle) ;
//   - multipiste, piste 0 = mix complet (track0IsFullMix) → `full` lit la piste 0
//     directement (pas d'amix doublé), `game`/`voices` = composantes (cf.
//     fullMixRenditions) ;
//   - multipiste sinon → 3 renditions pré-mixées historiques game/voices/full.
//
// Le pré-mixage est nécessaire car les renditions HLS sont mutuellement exclusives
// à la lecture (on ne peut pas additionner deux pistes côté navigateur). Les Display
// valent les slugs machine (game/voices/full) : le lecteur les matche pour détecter
// le layout 2-toggles ; sinon il retombe sur le sélecteur par-piste.
func planAudioRenditions(src []AVStreamDetail, track0IsFullMix bool) ([]audioRendition, string) {
	if len(src) < 2 {
		return []audioRendition{singleAudioRendition(src[0])}, ""
	}
	if track0IsFullMix {
		return fullMixRenditions(src)
	}
	return componentRenditions(src)
}

// singleAudioRendition produit l'unique rendition `a0` d'une piste directe (pas de
// switch côté lecteur → copy tolérée pour les codecs fMP4-compatibles).
func singleAudioRendition(s AVStreamDetail) audioRendition {
	return audioRendition{
		Slug: "a0", Display: audioDisplay(s, 0), MapSpec: "0:a:0",
		Action: planAudio(s.CodecName), Default: true, Language: sanitizeLanguage(s.Language),
	}
}

// componentRenditions : mapping historique (piste 0 = jeu). game = piste 0, voices =
// mix des pistes 1..N, full = mix de toutes (DEFAULT). Codec unique AAC sur tout le
// groupe (évite SourceBuffer.changeType MSE, non fiable Firefox/Safari).
func componentRenditions(src []AVStreamDetail) ([]audioRendition, string) {
	game := audioRendition{
		Slug: "game", Display: "game", MapSpec: "0:a:0",
		Action: aacUniformAction(src[0].CodecName), Language: sanitizeLanguage(src[0].Language),
	}
	var fcParts []string
	voices := audioRendition{Slug: "voices", Display: "voices", Action: actionReencode}
	if len(src) == 2 {
		voices.MapSpec = "0:a:1"
		voices.Action = aacUniformAction(src[1].CodecName)
	} else {
		fcParts = append(fcParts, amixFilter(rangeIdx(1, len(src)), "voices"))
		voices.MapSpec = "[voices]"
	}
	fcParts = append(fcParts, amixFilter(rangeIdx(0, len(src)), "full"))
	full := audioRendition{
		Slug: "full", Display: "full", MapSpec: "[full]", Action: actionReencode, Default: true,
	}
	return []audioRendition{game, voices, full}, strings.Join(fcParts, ";")
}

// fullMixRenditions : piste 0 = mix complet de sortie. `full` = piste 0 directe (PAS
// d'amix → pas de doublage/écho), `game` = piste 1 (1ʳᵉ composante), `voices` = mix
// des pistes 2..N (micro + voix). Codec unique AAC. Si une seule composante (pas de
// quoi séparer jeu/voix), on sert le mix complet seul (rendition unique).
func fullMixRenditions(src []AVStreamDetail) ([]audioRendition, string) {
	if len(src)-1 < 2 {
		return []audioRendition{singleAudioRendition(src[0])}, ""
	}
	full := audioRendition{
		Slug: "full", Display: "full", MapSpec: "0:a:0",
		Action: aacUniformAction(src[0].CodecName), Default: true,
	}
	game := audioRendition{
		Slug: "game", Display: "game", MapSpec: "0:a:1",
		Action: aacUniformAction(src[1].CodecName), Language: sanitizeLanguage(src[1].Language),
	}
	voices := audioRendition{Slug: "voices", Display: "voices", Action: actionReencode}
	var fc string
	if len(src) == 3 {
		voices.MapSpec = "0:a:2"
		voices.Action = aacUniformAction(src[2].CodecName)
	} else {
		fc = amixFilter(rangeIdx(2, len(src)), "voices")
		voices.MapSpec = "[voices]"
	}
	// Ordre game/voices/full conservé (le lecteur matche par slug ; full reste DEFAULT).
	return []audioRendition{game, voices, full}, fc
}

// rangeIdx retourne [from, to) sous forme de slice d'indices audio source.
func rangeIdx(from, to int) []int {
	out := make([]int, 0, to-from)
	for i := from; i < to; i++ {
		out = append(out, i)
	}
	return out
}

// amixFilter construit un segment -filter_complex amix mixant les pistes audio
// source d'indices donnés vers le label de sortie nommé. normalize=0 conserve
// les niveaux d'origine (sinon amix divise le volume par le nombre d'entrées).
func amixFilter(srcIdx []int, label string) string {
	var ins strings.Builder
	for _, i := range srcIdx {
		fmt.Fprintf(&ins, "[0:a:%d]", i)
	}
	return fmt.Sprintf("%samix=inputs=%d:normalize=0:duration=longest[%s]", ins.String(), len(srcIdx), label)
}

// planVideo décide copy/réencode pour la vidéo. H.264/AV1/HEVC passent en fMP4
// par copy (HEVC lu seulement par certains navigateurs — toléré). VP8/VP9 ne
// sont pas supportés par HLS → réencode H.264.
func planVideo(codec string) (streamAction, string) {
	switch strings.ToLower(codec) {
	case "h264", "av1", "hevc", "h265":
		return actionCopy, ""
	default:
		return actionReencode, "h264"
	}
}

// planAudio décide copy/réencode pour une piste audio. Opus/AAC/MP3 sont
// conteneurisables en fMP4 et lus par hls.js → copy. Le reste → AAC.
func planAudio(codec string) streamAction {
	switch strings.ToLower(codec) {
	case "opus", "aac", "mp3":
		return actionCopy
	default:
		return actionReencode
	}
}

// aacUniformAction décide copy/réencode pour une rendition d'un groupe audio
// MULTIPISTE, où toutes les renditions doivent partager le même codec (AAC) :
// copy uniquement si la source est déjà AAC, sinon réencode AAC. Garantit un
// groupe audio mono-codec → la bascule de piste fonctionne sur Firefox/Safari
// (pas de SourceBuffer.changeType). À distinguer de planAudio (mono-piste, où
// la copy Opus/MP3 est sans conséquence puisqu'il n'y a pas de switch).
func aacUniformAction(codec string) streamAction {
	if strings.EqualFold(codec, "aac") {
		return actionCopy
	}
	return actionReencode
}

// audioDisplay calcule le libellé lisible d'une piste : title, sinon langue en
// majuscules, sinon "Audio N" (1-based).
func audioDisplay(s AVStreamDetail, idx int) string {
	if t := strings.TrimSpace(s.Title); t != "" {
		return t
	}
	if l := strings.TrimSpace(s.Language); l != "" {
		return strings.ToUpper(l)
	}
	return fmt.Sprintf("Audio %d", idx+1)
}

// languageRe valide un code langue (lettres uniquement) avant propagation au
// var_stream_map (évite d'injecter des caractères parasites dans les args).
var languageRe = regexp.MustCompile(`^[A-Za-z]{2,3}$`)

func sanitizeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if languageRe.MatchString(lang) {
		return strings.ToLower(lang)
	}
	return ""
}

// buildVarStreamMap construit la valeur -var_stream_map : une variante vidéo
// liée au groupe audio "aud", puis chaque rendition audio dans ce groupe.
// L'index a:N réfère l'ordinal de SORTIE (position parmi les -map audio), pas
// l'index source — une rendition peut provenir d'un filtre. Le name: pilote le
// nom de fichier (%v) ; le NAME affiché dans le master est réécrit séparément
// par rewriteMasterAudioNames.
func buildVarStreamMap(plan hlsPlan) string {
	parts := []string{"v:0,agroup:aud"}
	for i, a := range plan.Audios {
		seg := fmt.Sprintf("a:%d,agroup:aud,name:%s", i, a.Slug)
		if a.Default {
			seg += ",default:yes"
		}
		if a.Language != "" {
			seg += ",language:" + a.Language
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, " ")
}

// nameAttrRe matche l'attribut NAME="..." d'une ligne EXT-X-MEDIA.
var nameAttrRe = regexp.MustCompile(`NAME="[^"]*"`)

// rewriteMasterAudioNames réécrit l'attribut NAME des lignes
// EXT-X-MEDIA:TYPE=AUDIO dans l'ordre, avec les displays fournis. ffmpeg génère
// NAME="audio_1", "audio_2"… ; on les remplace par des libellés lisibles
// (title de piste) pour le sélecteur côté lecteur. Pure.
func rewriteMasterAudioNames(master string, displays []string) string {
	lines := strings.Split(master, "\n")
	di := 0
	for i, line := range lines {
		if !strings.HasPrefix(line, "#EXT-X-MEDIA:") || !strings.Contains(line, "TYPE=AUDIO") {
			continue
		}
		if di < len(displays) && strings.TrimSpace(displays[di]) != "" {
			repl := `NAME="` + escapeAttr(displays[di]) + `"`
			lines[i] = nameAttrRe.ReplaceAllString(line, repl)
		}
		di++
	}
	return strings.Join(lines, "\n")
}

// escapeAttr neutralise les guillemets dans un libellé inséré en attribut.
func escapeAttr(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}

// ProbeStreamsDetailed retourne les streams (vidéo + audio) d'un fichier via
// ffprobe, avec codec, canaux et tags language/title. Utilisé par le service
// pour décider NeedsHLS puis par BuildHLS pour planifier.
func ProbeStreamsDetailed(ctx context.Context, absPath string) ([]AVStreamDetail, error) {
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		absPath,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}
	var parsed struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Channels  int    `json:"channels"`
			Tags      struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse ffprobe json: %w", err)
	}
	streams := make([]AVStreamDetail, 0, len(parsed.Streams))
	for _, s := range parsed.Streams {
		streams = append(streams, AVStreamDetail{
			Index:     s.Index,
			CodecType: s.CodecType,
			CodecName: s.CodecName,
			Channels:  s.Channels,
			Language:  s.Tags.Language,
			Title:     s.Tags.Title,
		})
	}
	return streams, nil
}

// VerifyHLSPlayable confirme qu'un master.m3u8 produit est réellement
// démultiplexable par ffprobe (pas seulement présent sur disque) et expose au
// moins une piste vidéo. ffprobe suit la playlist : un manifest mal formé, des
// init/segments manquants ou des segments corrompus le font échouer.
//
// Utilisé comme garde AVANT la suppression irréversible du fichier source : tant
// que la lecture HLS n'est pas prouvée, on conserve le source (fallback remux).
func VerifyHLSPlayable(ctx context.Context, masterPath string) error {
	streams, err := ProbeStreamsDetailed(ctx, masterPath)
	if err != nil {
		return fmt.Errorf("ffprobe master HLS %q: %w", masterPath, err)
	}
	for _, s := range streams {
		if s.CodecType == "video" {
			return nil
		}
	}
	return fmt.Errorf("master HLS sans piste vidéo lisible: %s", masterPath)
}

// BuildHLS transcode srcPath en arbre HLS-fMP4 dans outDir (master.m3u8 +
// sous-playlists + segments). Copy par défaut, réencode ciblé selon planHLS.
// Réécrit les NAME du master avec les titres de piste, puis valide la sortie.
//
// ffmpeg/ffprobe doivent être dans le PATH. outDir est créé si absent.
func BuildHLS(ctx context.Context, srcPath, outDir string, opts HLSOptions) (HLSResult, error) {
	streams, err := ProbeStreamsDetailed(ctx, srcPath)
	if err != nil {
		return HLSResult{}, err
	}
	// La piste 0 est-elle le mix complet de sortie (OBS « capture de sortie ») ? Si
	// oui, `full` la lira directement au lieu d'un amix qui la doublerait (écho).
	// Décision IO (décode l'audio) faite ici pour garder planHLS pur.
	fullMix := false
	if n := countAudioStreams(streams); n >= 2 {
		ok, corr, derr := track0IsFullMix(ctx, srcPath, n)
		if derr != nil {
			slog.WarnContext(ctx, "BuildHLS: détection mix complet échouée, mapping par défaut",
				"src", srcPath, "err", derr)
		} else {
			fullMix = ok
			slog.InfoContext(ctx, "BuildHLS: analyse pistes audio",
				"src", srcPath, "audio_count", n, "track0_full_mix", ok, "envelope_corr", corr)
		}
	}
	plan, err := planHLS(streams, fullMix)
	if err != nil {
		return HLSResult{}, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return HLSResult{}, fmt.Errorf("créer outDir %q: %w", outDir, err)
	}

	segDur := opts.SegmentDuration
	if segDur <= 0 {
		segDur = defaultSegmentDuration
	}

	args := buildHLSArgs(plan, srcPath, outDir, segDur)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "BuildHLS ffmpeg failed",
			"src", srcPath, "out_dir", outDir, "stderr", stderr.String(), "err", err)
		return HLSResult{}, fmt.Errorf("ffmpeg hls: %w (stderr: %s)", err, stderr.String())
	}

	// Réécrire les NAME auto-générés du master avec les titres de piste.
	masterPath := filepath.Join(outDir, "master.m3u8")
	if err := rewriteMasterFile(masterPath, plan.Audios); err != nil {
		slog.WarnContext(ctx, "BuildHLS: réécriture NAME master échouée (non bloquant)",
			"master", masterPath, "err", err)
	}

	res, err := validateHLSOutput(outDir, len(plan.Audios))
	if err != nil {
		return HLSResult{}, err
	}
	res.Renditions = renditionSlugs(plan.Audios)
	return res, nil
}

// renditionSlugs extrait les slugs des renditions audio (pour le logging
// d'observabilité : distingue le layout pré-mixé game/voices/full du legacy a0).
func renditionSlugs(audios []audioRendition) []string {
	slugs := make([]string, len(audios))
	for i, a := range audios {
		slugs[i] = a.Slug
	}
	return slugs
}

// buildHLSArgs construit les arguments ffmpeg depuis le plan. Découpé de
// BuildHLS pour rester sous la limite de taille et faciliter les tests.
func buildHLSArgs(plan hlsPlan, src, outDir string, segDur int) []string {
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", src}
	if plan.FilterComplex != "" {
		args = append(args, "-filter_complex", plan.FilterComplex)
	}
	args = append(args, "-map", "0:v:0")
	for _, a := range plan.Audios {
		args = append(args, "-map", a.MapSpec)
	}
	if plan.VideoAction == actionCopy {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p")
	}
	for i, a := range plan.Audios {
		if a.Action == actionCopy {
			args = append(args, fmt.Sprintf("-c:a:%d", i), "copy")
		} else {
			args = append(args, fmt.Sprintf("-c:a:%d", i), "aac", fmt.Sprintf("-b:a:%d", i), aacRenditionBitrate)
		}
	}
	// ffmpeg traite les chemins de sortie HLS comme des URL (séparateur '/').
	// Sur Windows, un chemin en backslash empêche ffmpeg de résoudre le
	// répertoire de base : les init segments (nom relatif) seraient alors écrits
	// dans le CWD du process au lieu de outDir. On force donc des forward-slash.
	outSlash := filepath.ToSlash(outDir)
	args = append(args,
		"-var_stream_map", plan.VarStreamMap,
		"-master_pl_name", "master.m3u8",
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_playlist_type", "vod",
		"-hls_time", strconv.Itoa(segDur),
		"-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init_%v.mp4",
		"-hls_segment_filename", outSlash+"/seg_%v_%03d.m4s",
		outSlash+"/stream_%v.m3u8",
	)
	return args
}

// rewriteMasterFile applique rewriteMasterAudioNames au fichier master sur disque.
func rewriteMasterFile(masterPath string, audios []audioRendition) error {
	raw, err := os.ReadFile(masterPath)
	if err != nil {
		return err
	}
	displays := make([]string, len(audios))
	for i, a := range audios {
		displays[i] = a.Display
	}
	rewritten := rewriteMasterAudioNames(string(raw), displays)
	return os.WriteFile(masterPath, []byte(rewritten), 0o644)
}

// validateHLSOutput vérifie qu'un arbre HLS exploitable a été produit :
// master présent, au moins un segment, et le bon nombre de pistes audio.
func validateHLSOutput(outDir string, expectedAudios int) (HLSResult, error) {
	masterPath := filepath.Join(outDir, "master.m3u8")
	if _, err := os.Stat(masterPath); err != nil {
		return HLSResult{}, fmt.Errorf("master.m3u8 absent: %w", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return HLSResult{}, fmt.Errorf("lecture outDir: %w", err)
	}
	segments := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".m4s") {
			segments++
		}
	}
	if segments == 0 {
		return HLSResult{}, fmt.Errorf("aucun segment .m4s généré dans %q", outDir)
	}
	return HLSResult{
		MasterPath:  masterPath,
		AudioTracks: expectedAudios,
		Segments:    segments,
	}, nil
}
