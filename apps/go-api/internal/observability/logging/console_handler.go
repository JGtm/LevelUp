package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

// ConsoleHandler est un slog.Handler qui produit un format compact lisible
// pour la console, optimisé pour le scanning humain :
//
//	14:30:08 [INFO] sync.postSync: pipeline démarré matches_inserted=3
//	14:30:09 [WARN] halo_api: GET HTTP error status=429 url=…/spnkr
//
// Conventions :
//   - Time : HH:MM:SS (pas la date complète, déjà dans les fichiers JSON).
//   - Level : `[INFO]` / `[WARN]` / `[ERROR]` / `[DEBUG]`, padding à 5 chars
//     pour alignement vertical des messages.
//   - Message : tel quel, sans préfixe ni quotes.
//   - Attrs : `key=value` espace-séparés, après le message. Les valeurs
//     contenant un espace sont quotées (`"...."`).
//   - Tronquage : si la ligne dépasse MaxWidth, suffixe `…` (Unicode horizontal
//     ellipsis, 1 rune = 3 bytes). 0 = pas de tronquage.
//   - Skip attrs : `event_id`, `request_id`, `source.*` masqués par défaut sur
//     console (préservés dans logs/{module}.log JSON).
//   - Couleurs ANSI : optionnel pour TTY interactif (LEVELUP_LOG_COLOR=on).
//
// Toute fonctionnalité multi-module (dispatch fichier) reste assurée par
// MultiModuleHandler qui peut wrapper ce ConsoleHandler comme sa sortie console.
type ConsoleHandler struct {
	out       io.Writer
	level     slog.Leveler
	maxWidth  int
	color     bool
	skipAttrs map[string]bool // clés d'attrs à NE PAS afficher sur console

	// État interne pour WithAttrs/WithGroup (accumulés entre clones).
	attrs  []slog.Attr
	groups []string

	mu sync.Mutex // protège les écritures concurrentes
}

// ConsoleHandlerOptions configure le ConsoleHandler.
type ConsoleHandlerOptions struct {
	// Level : niveau minimal des records traités. nil → slog.LevelInfo.
	Level slog.Leveler
	// MaxWidth : tronquage des lignes (caractères). 0 = pas de tronquage.
	// Défaut 200 si nil (configuré via LEVELUP_LOG_MAX_LINE).
	MaxWidth int
	// Color : activer les couleurs ANSI pour les niveaux. False par défaut
	// (auto-détection TTY non implémentée pour rester portable Windows).
	Color bool
	// SkipAttrs : clés d'attributs à ne pas afficher sur console (préservés
	// dans les fichiers JSON). Si nil, défaut = {event_id, request_id,
	// source.function, source.file, source.line, source}.
	SkipAttrs []string
}

// NewConsoleHandler crée un ConsoleHandler avec les options données.
func NewConsoleHandler(out io.Writer, opts ConsoleHandlerOptions) *ConsoleHandler {
	level := slog.Leveler(slog.LevelInfo)
	if opts.Level != nil {
		level = opts.Level
	}
	maxWidth := opts.MaxWidth
	if maxWidth < 0 {
		maxWidth = 0
	}
	skip := defaultSkipAttrs()
	if opts.SkipAttrs != nil {
		skip = make(map[string]bool, len(opts.SkipAttrs))
		for _, k := range opts.SkipAttrs {
			skip[k] = true
		}
	}
	return &ConsoleHandler{
		out:       out,
		level:     level,
		maxWidth:  maxWidth,
		color:     opts.Color,
		skipAttrs: skip,
	}
}

// defaultSkipAttrs retourne les clés masquées par défaut sur console.
// event_id/request_id : utiles dans les fichiers (grep), bruit en console.
// source.* : noms de fonction longs, encombrent l'œil.
func defaultSkipAttrs() map[string]bool {
	return map[string]bool{
		"event_id":        true,
		"request_id":      true,
		"source.function": true,
		"source.file":     true,
		"source.line":     true,
		"source":          true,
	}
}

// Enabled implémente slog.Handler.
func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implémente slog.Handler — format compact + truncation + skip attrs.
func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	// Time HH:MM:SS (date complète préservée en JSON file).
	buf.WriteString(r.Time.Format("15:04:05"))
	buf.WriteByte(' ')

	// Level [XXXX] padded
	buf.WriteString(formatLevel(r.Level, h.color))
	buf.WriteByte(' ')

	// Message
	buf.WriteString(r.Message)

	// Attrs accumulés via WithAttrs (préservés pendant tout le chain)
	for _, a := range h.attrs {
		if h.skipAttrs[a.Key] {
			continue
		}
		buf.WriteByte(' ')
		writeAttr(&buf, a)
	}

	// Attrs du record courant
	r.Attrs(func(a slog.Attr) bool {
		if h.skipAttrs[a.Key] {
			return true // skip, continuer iteration
		}
		buf.WriteByte(' ')
		writeAttr(&buf, a)
		return true
	})

	// Tronquage si dépassement.
	line := buf.String()
	if h.maxWidth > 0 && lineWidth(line) > h.maxWidth {
		line = truncateLine(line, h.maxWidth)
	}
	line += "\n"

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write([]byte(line))
	return err
}

// WithAttrs implémente slog.Handler — accumule les attrs pour les clones.
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := h.clone()
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

// WithGroup implémente slog.Handler — accumule les groupes pour les clones.
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	clone := h.clone()
	clone.groups = append(clone.groups, name)
	return clone
}

// clone copie superficiellement le handler (out + level + mu partagés).
func (h *ConsoleHandler) clone() *ConsoleHandler {
	return &ConsoleHandler{
		out:       h.out,
		level:     h.level,
		maxWidth:  h.maxWidth,
		color:     h.color,
		skipAttrs: h.skipAttrs,
		attrs:     append([]slog.Attr(nil), h.attrs...),
		groups:    append([]string(nil), h.groups...),
	}
}

// formatLevel rend [INFO] / [WARN] / [ERROR] / [DEBUG] padded à 7 chars
// (inclut les crochets) pour alignement vertical des messages.
func formatLevel(l slog.Level, color bool) string {
	var label string
	switch {
	case l >= slog.LevelError:
		label = "[ERROR]"
	case l >= slog.LevelWarn:
		label = "[WARN] "
	case l >= slog.LevelInfo:
		label = "[INFO] "
	case l >= slog.LevelDebug:
		label = "[DEBUG]"
	default:
		// Custom level — affiche le niveau brut.
		label = fmt.Sprintf("[%-5d]", l)
	}
	if !color {
		return label
	}
	// ANSI couleurs (TTY uniquement, off par défaut sous Windows cmd).
	switch {
	case l >= slog.LevelError:
		return "\x1b[31m" + label + "\x1b[0m" // rouge
	case l >= slog.LevelWarn:
		return "\x1b[33m" + label + "\x1b[0m" // jaune
	case l >= slog.LevelInfo:
		return "\x1b[34m" + label + "\x1b[0m" // bleu
	default:
		return "\x1b[90m" + label + "\x1b[0m" // gris
	}
}

// writeAttr écrit "key=value" dans buf. Quote les valeurs avec espaces.
// Pour les types non scalaires (group, slice), fallback sur fmt.Sprintf.
func writeAttr(buf *bytes.Buffer, a slog.Attr) {
	buf.WriteString(a.Key)
	buf.WriteByte('=')
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		writeMaybeQuoted(buf, s)
	case slog.KindInt64:
		buf.WriteString(strconv.FormatInt(v.Int64(), 10))
	case slog.KindUint64:
		buf.WriteString(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindFloat64:
		buf.WriteString(strconv.FormatFloat(v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.WriteString(strconv.FormatBool(v.Bool()))
	case slog.KindDuration:
		buf.WriteString(v.Duration().String())
	case slog.KindTime:
		buf.WriteString(v.Time().Format("15:04:05.000"))
	default:
		writeMaybeQuoted(buf, fmt.Sprint(v.Any()))
	}
}

// writeMaybeQuoted écrit s, en ajoutant des guillemets si s contient des
// espaces, des `=`, ou si vide.
func writeMaybeQuoted(buf *bytes.Buffer, s string) {
	if s == "" || strings.ContainsAny(s, " \t\"=") {
		buf.WriteByte('"')
		// Échapper les guillemets internes
		for _, r := range s {
			if r == '"' {
				buf.WriteByte('\\')
			}
			buf.WriteRune(r)
		}
		buf.WriteByte('"')
		return
	}
	buf.WriteString(s)
}

// lineWidth retourne le nombre de runes UTF-8 (pas d'octets). Comptage approximatif
// pour le truncation — pas de gestion des width des emojis ou CJK.
func lineWidth(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// truncateLine tronque s à maxWidth-1 runes et ajoute `…` (= 1 rune).
func truncateLine(s string, maxWidth int) string {
	if maxWidth <= 1 {
		return "…"
	}
	keep := maxWidth - 1
	count := 0
	for i := range s {
		if count == keep {
			return s[:i] + "…"
		}
		count++
	}
	return s
}
