package media

// hls.go — génération d'un arbre HLS-fMP4 multipiste à l'ingestion média.
//
// Symétrique de remux.go (remux WebM live), mais produit un arbre HLS statique
// (master.m3u8 + sous-playlists + segments fMP4) servi ensuite comme fichiers
// plats. Permet la sélection de piste audio (EXT-X-MEDIA TYPE=AUDIO) et le seek
// (Range sur les segments), ce que le remux WebM ne permet pas.
//
// Cible navigateur : Chrome/Firefox/Edge via hls.js (MSE) — c'est là que la
// sélection de piste audio (game/voices/full) et l'adaptatif sont pilotés. Safari
// et iOS lisent le HLS EN NATIF, en best-effort seulement : le lecteur natif
// n'expose PAS de sélecteur de pistes (l'utilisateur reste sur la rendition
// DEFAULT) et n'accepte pas tous les codecs (HEVC selon le matériel, Opus-in-fMP4
// non lu). D'où deux durcissements pour rester audible/lisible en natif : l'audio
// est UNIFORMISÉ en AAC (aacUniformAction), et la vidéo HEVC copiée reçoit le
// sample entry hvc1 (buildHLSArgs) que Safari exige à la place du hev1 par défaut.
//
// Policy codec : copy par défaut, réencodage ciblé pour ce qui n'est pas lisible
// partout en HLS-fMP4 — vidéo VP8/VP9 → H.264 ; audio non-AAC (Opus/MP3/…) → AAC.
// La vidéo H.264/HEVC/AV1 est copiée (HEVC avec tag hvc1).
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

// amixLimiterCeiling est le plafond linéaire (0..1) appliqué par alimiter à la
// SORTIE de chaque amix (rendition servie). amix somme les entrées SANS les diviser
// (normalize=0, choisi pour ne pas affaiblir le volume) : la somme jeu+voix peut
// dépasser le plein-échelle et écrêter. Le limiteur borne les crêtes juste sous 1.0.
// level=false → limiteur de crête pur, sans re-normalisation auto qui annulerait
// le normalize=0.
const amixLimiterCeiling = "0.98"

// codecTypeAudio est la valeur de CodecType (ffprobe) d'une piste audio.
const codecTypeAudio = "audio"

// audioRenditionGameSlug est le slug de la rendition « jeu » dans le master HLS.
const audioRenditionGameSlug = "game"

// audioRenditionFullSlug est le slug de la rendition « mix complet » dans le
// master HLS (aussi le label amix de sa sortie de filtre, cf. mapSpecFull).
const audioRenditionFullSlug = "full"

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

// mapSpecFull est le label de filtre ffmpeg de la rendition « full » (mix complet),
// partagé entre les planners auto (fullMixRenditions) et manuel (hls_audio_manual.go).
const mapSpecFull = "[full]"

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
	VideoSrcCodec string // codec vidéo SOURCE (ex. "hevc") ; pilote le tag hvc1 en copy
	Audios        []audioRendition
	FilterComplex string // -filter_complex (amix) ; "" si aucune agrégation
	VarStreamMap  string
}

// HLSOptions configure BuildHLS.
type HLSOptions struct {
	SegmentDuration int // secondes ; défaut 4 si <= 0
	// ManualAudioRoles : rôle déclaré de chaque piste audio SOURCE (ordre ffprobe :
	// ManualAudioRoles[0] = 0:a:0, ...), valeurs "game"/"voice"/"other". Non vide et
	// de longueur == nb pistes audio source ⇒ mapping MANUEL (bypass de l'analyse
	// NNLS). Vide ⇒ analyse automatique (comportement historique). Une longueur
	// incohérente est ignorée (fallback auto, WARN loggé) — jamais bloquant.
	ManualAudioRoles []string
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
// layout : décision IO calculée en amont par BuildHLS (cf. hls_audio_analyze.go).
// Quand layout.Track0FullMix, la piste 0 est le mix complet de sortie (`full` la lit
// directement, pas d'amix doublé / écho) et layout.GameComponent désigne la piste jeu.
func planHLS(streams []AVStreamDetail, layout audioLayout) (hlsPlan, error) {
	plan, srcAudios, err := collectStreamsForPlan(streams)
	if err != nil {
		return hlsPlan{}, err
	}
	plan.Audios, plan.FilterComplex = planAudioRenditions(srcAudios, layout)
	plan.VarStreamMap = buildVarStreamMap(plan)
	return plan, nil
}

// collectStreamsForPlan sépare les streams source en champ vidéo du plan et pistes
// audio ordonnées (résultat[i] = 0:a:i). Partagé par planHLS (auto) et planHLSManual
// pour ne pas dupliquer la planification vidéo. Pur. Erreur si pas de vidéo ou audio.
func collectStreamsForPlan(streams []AVStreamDetail) (hlsPlan, []AVStreamDetail, error) {
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
			plan.VideoSrcCodec = s.CodecName
			plan.VideoAction, plan.VideoCodec = planVideo(s.CodecName)
		case "audio":
			srcAudios = append(srcAudios, s)
		}
	}
	if !hasVideo {
		return hlsPlan{}, nil, fmt.Errorf("planHLS: aucune piste vidéo")
	}
	if len(srcAudios) == 0 {
		return hlsPlan{}, nil, fmt.Errorf("planHLS: aucune piste audio")
	}
	return plan, srcAudios, nil
}

// planAudioRenditions dérive les renditions de sortie des pistes source.
//
//   - mono-piste → 1 rendition directe (comportement historique, pas de toggle) ;
//   - multipiste, piste 0 = mix complet (layout.Track0FullMix) → `full` lit la piste 0
//     directement (pas d'amix doublé), `game` = composante classée jeu, `voices` = le
//     reste (cf. fullMixRenditions) ;
//   - multipiste sinon → 3 renditions pré-mixées historiques game/voices/full.
//
// Le pré-mixage est nécessaire car les renditions HLS sont mutuellement exclusives
// à la lecture (on ne peut pas additionner deux pistes côté navigateur). Les Display
// valent les slugs machine (game/voices/full) : le lecteur les matche pour détecter
// le layout 2-toggles ; sinon il retombe sur le sélecteur par-piste.
func planAudioRenditions(src []AVStreamDetail, layout audioLayout) ([]audioRendition, string) {
	if len(src) < 2 {
		return []audioRendition{singleAudioRendition(src[0])}, ""
	}
	if layout.Track0FullMix {
		return fullMixRenditions(src, layout.GameComponent)
	}
	return componentRenditions(src)
}

// singleAudioRendition produit l'unique rendition `a0` d'une piste directe. Codec
// uniformisé AAC (aacUniformAction) : même sans switch de piste, un Opus copié
// dans fMP4 est INAUDIBLE en HLS natif Safari/iOS — la copy n'est donc tolérée que
// si la source est déjà AAC, sinon réencodage AAC.
func singleAudioRendition(s AVStreamDetail) audioRendition {
	return audioRendition{
		Slug: "a0", Display: audioDisplay(s, 0), MapSpec: "0:a:0",
		Action: aacUniformAction(s.CodecName), Default: true, Language: sanitizeLanguage(s.Language),
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
	fcParts = append(fcParts, amixFilter(rangeIdx(0, len(src)), audioRenditionFullSlug))
	full := audioRendition{
		Slug: audioRenditionFullSlug, Display: audioRenditionFullSlug, MapSpec: mapSpecFull, Action: actionReencode, Default: true,
	}
	return []audioRendition{game, voices, full}, strings.Join(fcParts, ";")
}

// fullMixRenditions : piste 0 = mix complet de sortie. `full` = piste 0 directe (PAS
// d'amix → pas de doublage/écho), `game` = la composante `gameComponent` (classée
// acoustiquement, pas par position), `voices` = mix des AUTRES composantes (micro +
// Discord). Codec unique AAC. Si une seule composante (pas de quoi séparer jeu/voix),
// on sert le mix complet seul (rendition unique).
func fullMixRenditions(src []AVStreamDetail, gameComponent int) ([]audioRendition, string) {
	if len(src)-1 < 2 {
		return []audioRendition{singleAudioRendition(src[0])}, ""
	}
	if gameComponent < 1 || gameComponent >= len(src) {
		gameComponent = 1 // garde : index hors composantes → 1ère composante
	}
	full := audioRendition{
		Slug: "full", Display: "full", MapSpec: "0:a:0",
		Action: aacUniformAction(src[0].CodecName), Default: true,
	}
	game := audioRendition{
		Slug: "game", Display: "game", MapSpec: fmt.Sprintf("0:a:%d", gameComponent),
		Action:   aacUniformAction(src[gameComponent].CodecName),
		Language: sanitizeLanguage(src[gameComponent].Language),
	}
	// voices = toutes les composantes (1..N-1) SAUF la composante jeu.
	var voiceIdx []int
	for i := 1; i < len(src); i++ {
		if i != gameComponent {
			voiceIdx = append(voiceIdx, i)
		}
	}
	voices := audioRendition{Slug: "voices", Display: "voices", Action: actionReencode}
	var fc string
	if len(voiceIdx) == 1 {
		voices.MapSpec = fmt.Sprintf("0:a:%d", voiceIdx[0])
		voices.Action = aacUniformAction(src[voiceIdx[0]].CodecName)
	} else {
		fc = amixFilter(voiceIdx, "voices")
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
// les niveaux d'origine (sinon amix divise le volume par le nombre d'entrées),
// suivi d'un alimiter (amixLimiterCeiling) qui borne les crêtes de la somme pour
// éviter l'écrêtage. C'est un amix de SORTIE (rendition servie) : à distinguer des
// amix d'analyse de hls_audio_analyze.go, volontairement sans limiteur.
func amixFilter(srcIdx []int, label string) string {
	var ins strings.Builder
	for _, i := range srcIdx {
		fmt.Fprintf(&ins, "[0:a:%d]", i)
	}
	return fmt.Sprintf("%samix=inputs=%d:normalize=0:duration=longest,alimiter=limit=%s:level=false[%s]",
		ins.String(), len(srcIdx), amixLimiterCeiling, label)
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

// isHEVCCodec reconnaît un codec vidéo HEVC (H.265) — les deux noms que ffprobe
// peut retourner. Pilote l'ajout du tag hvc1 quand la vidéo est copiée.
func isHEVCCodec(codec string) bool {
	c := strings.ToLower(codec)
	return c == "hevc" || c == "h265"
}

// aacUniformAction décide copy/réencode pour une rendition audio HLS-fMP4 : copy
// uniquement si la source est déjà AAC, sinon réencode AAC. Deux raisons de forcer
// l'AAC : (1) sur un groupe MULTIPISTE, un codec unique est requis pour que la
// bascule de piste fonctionne sur Firefox/Safari (pas de SourceBuffer.changeType) ;
// (2) même en MONO-piste, l'Opus-in-fMP4 est inaudible en HLS natif Safari/iOS.
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

// VerifyHLSPlayable confirme qu'un master.m3u8 produit est exploitable AVANT la
// suppression irréversible du fichier source (tant que la lecture n'est pas
// prouvée, on conserve le source — fallback remux). Deux gardes :
//
//  1. ffprobe démultiplexe le master et y trouve une piste vidéo — un manifest mal
//     formé ou un arbre amputé de ses sous-playlists/segments ne remonte alors
//     AUCUN flux (ffprobe n'énumère que ce qu'il peut ouvrir) → pas de vidéo → rejet.
//  2. le master déclare EXACTEMENT expectedAudioTracks renditions audio. Le compte
//     vient du master lui-même (parseMasterAudioRenditions), pas de ffprobe : sur un
//     master sain ffprobe liste bien les renditions, mais son énumération dépend de
//     la version/du démuxeur HLS et reflète les sous-playlists OUVRABLES, pas celles
//     DÉCLARÉES (vérifié : master amputé → 0 flux, exit 0). Le master m3u8 est la
//     source de vérité déterministe du groupe audio → on y compte les EXT-X-MEDIA.
//     Attrape une rendition perdue au transcodage que le seul check vidéo manquait.
func VerifyHLSPlayable(ctx context.Context, masterPath string, expectedAudioTracks int) error {
	streams, err := ProbeStreamsDetailed(ctx, masterPath)
	if err != nil {
		return fmt.Errorf("ffprobe master HLS %q: %w", masterPath, err)
	}
	hasVideo := false
	for _, s := range streams {
		if s.CodecType == "video" {
			hasVideo = true
			break
		}
	}
	if !hasVideo {
		return fmt.Errorf("master HLS sans piste vidéo lisible: %s", masterPath)
	}
	raw, err := os.ReadFile(masterPath)
	if err != nil {
		return fmt.Errorf("lecture master HLS %q: %w", masterPath, err)
	}
	if got := len(parseMasterAudioRenditions(string(raw))); got != expectedAudioTracks {
		return fmt.Errorf("master HLS %q: %d renditions audio, attendu %d", masterPath, got, expectedAudioTracks)
	}
	return nil
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
	// Décide le plan audio : mapping MANUEL (rôles déclarés, bypass NNLS) quand
	// opts.ManualAudioRoles est cohérent avec la source, sinon analyse automatique
	// (OBS full-mix / classement acoustique de la composante jeu). Décision IO
	// (décode l'audio en auto) isolée dans buildHLSPlan pour garder planHLS pur.
	plan, err := buildHLSPlan(ctx, srcPath, streams, audioStreamsOnly(streams), opts)
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
	args := ffmpegQuietArgs("-y", "-i", src)
	if plan.FilterComplex != "" {
		args = append(args, "-filter_complex", plan.FilterComplex)
	}
	args = append(args, "-map", "0:v:0")
	for _, a := range plan.Audios {
		args = append(args, "-map", a.MapSpec)
	}
	if plan.VideoAction == actionCopy {
		args = append(args, "-c:v", "copy")
		// HEVC copié depuis un MKV : ffmpeg écrit par défaut le sample entry hev1,
		// que Safari refuse. hvc1 (config codec dans le sample description) est le
		// seul accepté en HLS-fMP4. Sans effet sur H.264/AV1.
		if isHEVCCodec(plan.VideoSrcCodec) {
			args = append(args, "-tag:v", "hvc1")
		}
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
