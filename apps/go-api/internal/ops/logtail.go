// Package ops — logtail.go : lecture efficace de la fin des fichiers de logs
// JSON slog par module (logs/{module}.log) pour le viewer du dashboard admin.
//
// Contraintes : fichiers de 30-40 Mo → JAMAIS de ReadFile complet. Lecture
// par la fin en chunks (ReadAt en remontant), budget de scan borné, parse
// best-effort ligne par ligne (une ligne non-JSON — panique brute, rotation
// lumberjack — devient une entrée Raw plutôt qu'une erreur).
package ops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	logTailDefaultN    = 200
	logTailMaxN        = 1000
	logTailChunkSize   = 256 * 1024      // 256 KiB par ReadAt
	logTailMaxScan     = 8 * 1024 * 1024 // budget 8 MiB par requête
	logTailMaxLineSize = 64 * 1024       // garde-fou ligne pathologique
)

// logModuleRe : validation stricte du nom de module (anti path-traversal).
var logModuleRe = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

// LogModule décrit un fichier logs/{module}.log.
type LogModule struct {
	Module     string
	SizeBytes  int64
	ModifiedAt time.Time
}

// ListLogModules liste les modules disponibles (basename sans .log), triés
// par taille décroissante (les plus actifs d'abord).
func ListLogModules(logsDir string) ([]LogModule, error) {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("logtail: lecture du dossier logs: %w", err)
	}
	var out []LogModule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		module := strings.TrimSuffix(e.Name(), ".log")
		if !logModuleRe.MatchString(module) {
			continue // fichiers de rotation lumberjack (timestampés) exclus
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, LogModule{Module: module, SizeBytes: info.Size(), ModifiedAt: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SizeBytes > out[j].SizeBytes })
	return out, nil
}

// LogTailOptions paramètre TailModuleLog.
type LogTailOptions struct {
	// N : nombre max d'entrées APRÈS filtres (défaut 200, plafond 1000).
	N int
	// Level : seuil minimal (debug|info|warn|error). Vide = tout.
	Level string
	// Contains : sous-chaîne insensible à la casse (msg, err, raw, request_id, event_id).
	Contains string
	// Since : ne pas remonter avant cet instant (zero = pas de borne).
	// Early-exit : la lecture s'arrête dès qu'une ligne plus ancienne est vue.
	Since time.Time
	// BeforeOffset : curseur ARRIÈRE de « charger plus » (offset octet absolu,
	// exclusif). La lecture démarre juste avant cet offset — la tranche déjà
	// affichée (à partir de BeforeOffset) n'est pas re-scannée. 0 = depuis la
	// fin (première page). Permet de paginer AU-DELÀ du budget 8 MiB (chaque
	// page rebudgette depuis son curseur), impossible en repartant de la fin.
	BeforeOffset int64
}

// LogEntry est une ligne de log parsée (best-effort).
type LogEntry struct {
	Time      time.Time
	Level     string // debug|info|warn|error|unknown
	Msg       string
	Module    string
	RequestID string
	EventID   string
	Err       string
	Source    string         // "file.go:42 (func)" compact, vide si absent
	Fields    map[string]any // attrs restants (hors clés extraites)
	Raw       string         // ligne brute si non-JSON
}

// LogTailResult est le résultat d'un tail.
type LogTailResult struct {
	// Entries : du plus récent au plus ancien.
	Entries      []LogEntry
	ScannedBytes int64
	// Truncated : budget de scan épuisé avant d'atteindre N entrées — il
	// existe peut-être des lignes plus anciennes correspondant aux filtres.
	Truncated bool
	// NextOffset : offset octet absolu du DÉBUT de la ligne la plus ancienne
	// renvoyée — curseur à repasser en BeforeOffset pour « charger plus »
	// (tranche strictement plus ancienne, sans recouvrement). 0 = début de
	// fichier atteint.
	NextOffset int64
	// HasMore : des lignes plus anciennes restent au-delà de NextOffset (N
	// atteint sans toucher le début, ou budget épuisé). false = début de
	// fichier atteint, plus rien à charger.
	HasMore bool
}

// TailModuleLog lit les dernières lignes de logs/{module}.log qui passent les
// filtres. Module invalide ou hors du dossier → erreur (anti-traversal).
func TailModuleLog(logsDir, module string, opts LogTailOptions) (LogTailResult, error) {
	var res LogTailResult
	if !logModuleRe.MatchString(module) {
		return res, fmt.Errorf("logtail: module invalide %q", module)
	}
	path := filepath.Join(logsDir, module+".log")
	if cleaned := filepath.Clean(path); !strings.HasPrefix(cleaned, filepath.Clean(logsDir)+string(filepath.Separator)) {
		return res, fmt.Errorf("logtail: chemin hors du dossier logs")
	}

	f, err := os.Open(path)
	if err != nil {
		return res, fmt.Errorf("logtail: ouverture %s: %w", module, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return res, err
	}

	n := opts.N
	if n <= 0 {
		n = logTailDefaultN
	}
	if n > logTailMaxN {
		n = logTailMaxN
	}

	// Curseur « charger plus » : démarrer juste avant BeforeOffset (exclusif)
	// plutôt qu'à la fin du fichier — la tranche déjà affichée n'est pas relue.
	startPos := info.Size()
	if opts.BeforeOffset > 0 && opts.BeforeOffset < startPos {
		startPos = opts.BeforeOffset
	}

	collectTail(f, startPos, n, opts, &res)
	return res, nil
}

// collectTail remonte le fichier par chunks depuis la fin et collecte les
// entrées filtrées (plus récentes d'abord).
//
// le découper disperserait l'invariant carry/pos.
//
//nolint:gocyclo // boucle de lecture inversée : chunking + carry + filtres,
func collectTail(f *os.File, startPos int64, n int, opts LogTailOptions, res *LogTailResult) {
	contains := strings.ToLower(strings.TrimSpace(opts.Contains))
	minRank, hasLevel := levelRank(opts.Level)
	pos := startPos
	var carry []byte

	for pos > 0 && len(res.Entries) < n {
		if res.ScannedBytes >= logTailMaxScan {
			res.Truncated, res.HasMore = true, true
			return
		}
		readSize := int64(logTailChunkSize)
		if readSize > pos {
			readSize = pos
		}
		chunkStart := pos - readSize
		buf := make([]byte, readSize)
		if _, err := f.ReadAt(buf, chunkStart); err != nil {
			res.Truncated, res.HasMore = true, true
			return
		}
		pos = chunkStart
		res.ScannedBytes += readSize

		data := buf
		if len(carry) > 0 {
			data = append(data, carry...) //nolint:makezero // concat volontaire (suite de la dernière ligne du chunk)
		}
		// data couvre [chunkStart, chunkStart+len(data)) : offset absolu de
		// chaque ligne = chunkStart + son décalage dans data.
		lines, offsets := splitLinesWithOffsets(data, chunkStart)
		if chunkStart > 0 {
			// La première « ligne » du chunk est incomplète (sa tête est dans
			// le chunk précédent) → reportée au prochain tour.
			carry = append([]byte(nil), lines[0]...)
			if len(carry) > logTailMaxLineSize {
				carry = carry[:logTailMaxLineSize]
			}
			lines, offsets = lines[1:], offsets[1:]
		} else {
			carry = nil
		}

		for i := len(lines) - 1; i >= 0; i-- {
			line := bytes.TrimSpace(lines[i])
			if len(line) == 0 {
				continue
			}
			entry := parseLogLine(line)
			if !opts.Since.IsZero() && !entry.Time.IsZero() && entry.Time.Before(opts.Since) {
				// Les lignes au-dessus sont encore plus anciennes : stop.
				return
			}
			if hasLevel {
				rank, known := levelRank(entry.Level)
				if !known || rank < minRank {
					continue
				}
			}
			if contains != "" && !entryContains(entry, contains) {
				continue
			}
			res.Entries = append(res.Entries, entry)
			res.NextOffset = offsets[i] // début de la ligne la plus ancienne renvoyée
			if len(res.Entries) >= n {
				res.HasMore = offsets[i] > 0 // rien de plus ancien si on est au début du fichier
				return
			}
		}
	}
}

// splitLinesWithOffsets découpe data en lignes (sur '\n') et retourne l'offset
// absolu du DÉBUT de chaque ligne (base = offset de data[0] dans le fichier).
func splitLinesWithOffsets(data []byte, base int64) ([][]byte, []int64) {
	lines := bytes.Split(data, []byte{'\n'})
	offsets := make([]int64, len(lines))
	off := base
	for i, l := range lines {
		offsets[i] = off
		off += int64(len(l)) + 1 // +1 : le séparateur '\n'
	}
	return lines, offsets
}

// levelRank retourne le rang d'un niveau (debug<info<warn<error) et si le
// niveau est connu.
func levelRank(level string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return 0, true
	case "info":
		return 1, true
	case "warn", "warning":
		return 2, true
	case "error":
		return 3, true
	default:
		return 0, false
	}
}

// entryContains : match insensible à la casse sur les champs textuels.
func entryContains(e LogEntry, needle string) bool {
	for _, s := range []string{e.Msg, e.Err, e.Raw, e.RequestID, e.EventID} {
		if s != "" && strings.Contains(strings.ToLower(s), needle) {
			return true
		}
	}
	return false
}

// extractedLogKeys : clés promues en champs typés (exclues de Fields).
var extractedLogKeys = map[string]struct{}{
	"time": {}, "level": {}, "msg": {}, "module": {}, "request_id": {},
	"event_id": {}, "err": {}, "source": {},
}

// parseLogLine parse une ligne JSON slog ; ligne non-JSON → entrée Raw.
func parseLogLine(line []byte) LogEntry {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return LogEntry{Level: "unknown", Raw: string(line)}
	}
	e := LogEntry{
		Level:     strings.ToLower(asLogString(m["level"])),
		Msg:       asLogString(m["msg"]),
		Module:    asLogString(m["module"]),
		RequestID: asLogString(m["request_id"]),
		EventID:   asLogString(m["event_id"]),
		Err:       asLogString(m["err"]),
	}
	if ts := asLogString(m["time"]); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			e.Time = t
		}
	}
	if src, ok := m["source"].(map[string]any); ok {
		file := asLogString(src["file"])
		if idx := strings.LastIndexAny(file, `/\`); idx >= 0 {
			file = file[idx+1:]
		}
		if file != "" {
			e.Source = fmt.Sprintf("%s:%v", file, src["line"])
		}
	}
	for k, v := range m {
		if _, skip := extractedLogKeys[k]; skip {
			continue
		}
		if e.Fields == nil {
			e.Fields = map[string]any{}
		}
		e.Fields[k] = v
	}
	return e
}

func asLogString(v any) string {
	s, _ := v.(string)
	return s
}
