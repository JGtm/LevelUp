// Package ops — media_capture_source.go : D'OÙ VIENT L'HEURE D'UNE CAPTURE, ET CE QU'ELLE DIT.
//
// Une capture n'a qu'UNE heure lisible à l'indexation, notée H : celle de son nom de fichier
// quand le logiciel de capture l'y a écrite, sinon la date de modification du fichier. Selon la
// source, H est le DÉBUT ou la FIN de la capture :
//
//   - un clip de RELECTURE (« Replay » d'OBS, « record that » de la Game Bar ou de la console
//     Xbox, Instant Replay de ShadowPlay) est écrit au moment où l'on demande de garder les N
//     dernières secondes : son nom porte l'instant de la demande, qui est la FIN du clip ;
//   - un ENREGISTREMENT lancé à la main (OBS, ShadowPlay) est nommé à son démarrage : DÉBUT ;
//   - la date de modification d'un fichier est sa dernière écriture : FIN.
//
// La fenêtre stockée dans media_files (capture_start_utc, capture_end_utc) se déduit de H, de
// l'ancre de la source et de la durée (captureWindow). La clé de la source est écrite dans
// media_files.capture_time_source : c'est elle qui rend le recalcul des lignes antérieures
// idempotent (media_capture_recalc.go).
//
// PRÉCISION. Les noms OBS, Game Bar et ShadowPlay portent la seconde (tronquée : la vraie heure
// est dans [H, H+1 s[) ; les clips de console Xbox ne portent que la minute ([H, H+60 s[).
//
// captureTimeSources est la SEULE table des sources : aucun motif de nom ni aucune ancre ne
// s'écrit ailleurs (TestCaptureSources_KeysAreUnique tient l'unicité des clés).
package ops

import (
	"os"
	"regexp"
	"time"
)

// captureAnchor : ce que l'heure lue d'une source dit de la capture.
type captureAnchor string

const (
	captureAnchorStart captureAnchor = "start" // H = début de la capture
	captureAnchorEnd   captureAnchor = "end"   // H = fin de la capture
)

// Clés des sources, telles qu'écrites dans media_files.capture_time_source.
const (
	captureSourceOBSReplay           = "obs_replay"
	captureSourceOBSRecording        = "obs_recording"
	captureSourceGameBar             = "game_bar"
	captureSourceShadowPlayReplay    = "shadowplay_replay"
	captureSourceShadowPlayRecording = "shadowplay_recording"
	captureSourceXboxConsole         = "xbox_console"
	captureSourceFileDate            = "file_date"
)

// Horodatages des noms de fichier. Groupes : 1 année, 2 mois, 3 jour, 4 heure, 5 minute,
// 6 seconde (absente du motif de console Xbox).
const (
	dashedStamp = `(\d{4})-(\d{2})-(\d{2}) (\d{2})-(\d{2})-(\d{2})`       // 2026-04-19 17-10-54
	dottedStamp = `(\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})` // 2024.11.15 - 21.30.45
	xboxStamp   = `(\d{4})-(\d{2})-(\d{2})_(\d{2})h(\d{2})`               // 2019-12-12_22h49
)

// captureTimeSource : une source d'heure de capture. pattern nil = date du fichier.
type captureTimeSource struct {
	Key     string
	Anchor  captureAnchor
	pattern *regexp.Regexp
}

// fromName indique si l'heure de la source se lit dans le nom (donc reste la même quand le
// fichier est converti ou recopié, contrairement à sa date de modification).
func (s captureTimeSource) fromName() bool { return s.pattern != nil }

// captureTimeSources : les sources lisibles dans un nom, dans l'ORDRE D'ESSAI (la première qui
// reconnaît le nom l'emporte), puis la date du fichier, toujours en dernier.
var captureTimeSources = []captureTimeSource{
	// OBS, tampon de relecture : préfixe « Replay » par défaut, nommé à la sauvegarde. Mesuré
	// sur le parc local (27 clips) : date de modification = H + 0 à 1,5 s, et la dernière image
	// du clip est l'instant de la sauvegarde.
	{captureSourceOBSReplay, captureAnchorEnd, regexp.MustCompile(`(?i)^replay[ _-]+` + dashedStamp)},
	// OBS, enregistrement : nom par défaut « %CCYY-%MM-%DD %hh-%mm-%ss », nommé au démarrage.
	{captureSourceOBSRecording, captureAnchorStart, regexp.MustCompile(`^` + dashedStamp)},
	// Game Bar de Windows : « <Jeu> AAAA-MM-JJ HH-MM-SS » — clips « Record that ».
	{captureSourceGameBar, captureAnchorEnd, regexp.MustCompile(dashedStamp)},
	// ShadowPlay, Instant Replay : suffixe « .DVR », nommé à la sauvegarde.
	{captureSourceShadowPlayReplay, captureAnchorEnd, regexp.MustCompile(`(?i)` + dottedStamp + `\.\d{2}\.DVR`)},
	// ShadowPlay, enregistrement manuel : nommé au démarrage.
	{captureSourceShadowPlayRecording, captureAnchorStart, regexp.MustCompile(dottedStamp)},
	// Console Xbox, « Xbox, record that » (clips Halo 5, téléchargés par l'application Xbox).
	{captureSourceXboxConsole, captureAnchorEnd, regexp.MustCompile(xboxStamp)},
	// Date de modification du fichier (file.lastModified du navigateur à l'envoi, sinon mtime
	// du disque) : la dernière écriture, donc la fin.
	{captureSourceFileDate, captureAnchorEnd, nil},
}

// captureSourceByKey rend la source de clé key (table captureTimeSources) ; la date du fichier
// pour une clé inconnue.
func captureSourceByKey(key string) captureTimeSource {
	for _, s := range captureTimeSources {
		if s.Key == key {
			return s
		}
	}
	return captureTimeSources[len(captureTimeSources)-1]
}

// captureSourceOfName rend la source dont le motif reconnaît le nom, false si aucun.
func captureSourceOfName(name string) (captureTimeSource, bool) {
	for _, s := range captureTimeSources {
		if s.pattern != nil && s.pattern.MatchString(name) {
			return s, true
		}
	}
	return captureTimeSource{}, false
}

// captureInstant : l'heure H lue pour une capture, et la source qui l'a donnée.
type captureInstant struct {
	At     time.Time // UTC
	Source captureTimeSource
}

// captureInstantFromName lit H dans le nom de fichier, interprété dans le fuseau loc (les
// logiciels de capture écrivent l'heure locale). nil si aucun motif ne reconnaît le nom ou si
// loc est nil.
func captureInstantFromName(name string, loc *time.Location) *captureInstant {
	if loc == nil {
		return nil
	}
	src, ok := captureSourceOfName(name)
	if !ok {
		return nil
	}
	m := src.pattern.FindStringSubmatch(name)
	year := mustAtoi(m[1])
	if year == 0 {
		return nil
	}
	sec := 0
	if len(m) > 6 {
		sec = mustAtoi(m[6])
	}
	at := time.Date(year, time.Month(mustAtoi(m[2])), mustAtoi(m[3]),
		mustAtoi(m[4]), mustAtoi(m[5]), sec, 0, loc).UTC()
	return &captureInstant{At: at, Source: src}
}

// resolveCaptureInstant rend H pour le fichier path, par ordre de priorité : le nom (heure du
// logiciel de capture), la date de modification envoyée par le navigateur (clientUnix, unix
// secondes), la date de modification sur le disque. nil si rien n'est lisible.
func resolveCaptureInstant(path, baseName string, clientUnix *int64, loc *time.Location) *captureInstant {
	if inst := captureInstantFromName(baseName, loc); inst != nil {
		return inst
	}
	fileDate := captureSourceByKey(captureSourceFileDate)
	if clientUnix != nil && *clientUnix > 0 {
		return &captureInstant{At: time.Unix(*clientUnix, 0).UTC(), Source: fileDate}
	}
	if info, err := os.Stat(path); err == nil {
		return &captureInstant{At: info.ModTime().UTC(), Source: fileDate}
	}
	return nil
}

// captureTimes : la fenêtre d'une capture telle qu'elle s'écrit dans media_files. Un champ nil
// s'écrit NULL.
type captureTimes struct {
	Start       *time.Time
	End         *time.Time
	DurationSec *float64
}

// captureWindow déduit la fenêtre d'une capture de son heure H (inst), de son ancre et de sa
// durée, sans IO :
//   - image : un instant, début = fin = H, durée 0 ;
//   - vidéo de durée D connue : [H, H+D] si H est un début, [H-D, H] si H est une fin ;
//   - vidéo de durée inconnue (ffprobe absent ou en échec) : H est la seule heure connue. Ancrée
//     au début, la fin reste NULL ; ancrée à la fin, H sert de début ET de fin (la frise du rejeu
//     pose alors le clip ponctuellement, l'association au match lit le début) ;
//   - format inconnu : H en début, rien d'autre.
//
// inst nil : ni début ni fin, la durée reste celle qu'on sait.
func captureWindow(kind string, inst *captureInstant, duration float64, durationKnown bool) captureTimes {
	var out captureTimes
	switch kind {
	case mediaKindImage:
		zero := 0.0
		out.DurationSec = &zero
		if inst != nil {
			out.Start, out.End = timePtr(inst.At), timePtr(inst.At)
		}
	case mediaKindVideo:
		if durationKnown {
			d := duration
			out.DurationSec = &d
		}
		if inst == nil {
			return out
		}
		out.Start, out.End = videoBounds(inst, duration, durationKnown)
	default:
		if inst != nil {
			out.Start = timePtr(inst.At)
		}
	}
	return out
}

// videoBounds : début et fin d'une vidéo d'heure inst (cf. captureWindow).
func videoBounds(inst *captureInstant, duration float64, durationKnown bool) (start, end *time.Time) {
	span := time.Duration(duration * float64(time.Second))
	switch {
	case inst.Source.Anchor == captureAnchorEnd && durationKnown:
		return timePtr(inst.At.Add(-span)), timePtr(inst.At)
	case inst.Source.Anchor == captureAnchorEnd:
		return timePtr(inst.At), timePtr(inst.At)
	case durationKnown:
		return timePtr(inst.At), timePtr(inst.At.Add(span))
	default:
		return timePtr(inst.At), nil
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// conversionTimes : ce qu'une conversion de format réécrit de la fenêtre d'une ligne déjà
// indexée. Une heure lue dans le NOM ne change pas avec le format : la fenêtre recalculée est
// juste et s'écrit avec sa source. Rien n'est réécrit (nil, que le COALESCE de l'UPDATE garde)
// quand l'heure vient de la date du fichier (celle de la conversion, pas de la capture), ni
// quand la durée de la vidéo convertie est inconnue (ffprobe en échec) : la fenêtre recalculée
// réduirait alors à H une fenêtre [H-D, H] déjà juste.
func conversionTimes(inst *captureInstant, kind string, win captureTimes) (start, end *time.Time, source *string) {
	if inst == nil || !inst.Source.fromName() || (kind == mediaKindVideo && win.DurationSec == nil) {
		return nil, nil, nil
	}
	return win.Start, win.End, &inst.Source.Key
}
