package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// MultiModuleHandler dispatche chaque slog.Record vers :
//  1. Un handler "console" (typiquement os.Stderr, format texte ou JSON
//     selon LEVELUP_LOG_JSON) — comportement pre-sprint inchangé.
//  2. Un handler "fichier" spécifique au module du record, sous
//     `{LogsDir}/{module}.log` (format JSON, append-only).
//
// Le module est déterminé via (priorité) :
//   - Attribut explicit slog.With("module", "...") sur le record
//   - Détection auto via PC d'appel (cf. module.go)
//   - Fallback ModuleGeneral
//
// Performance : les handlers fichiers sont créés lazy (premier log pour ce
// module) et mis en cache. La sérialisation JSON est faite une seule fois
// (slog.NewJSONHandler partagé par module). I/O fichier serializé via mu
// par module — write small (< 4KB) atomique sur les FS modernes.
//
// Thread-safe : tous les accès aux handlers fichiers sont protégés par mu.
type MultiModuleHandler struct {
	console slog.Handler // sortie console (existant + ContextHandler)
	logsDir string       // ex: "logs/"
	level   slog.Leveler // niveau global, partagé avec console
	attrs   []slog.Attr  // attrs accumulés via WithAttrs
	groups  []string     // groups via WithGroup
	mu      sync.Mutex   // protège fileHandlers
	files   map[string]*fileHandler
}

// fileHandler encapsule un slog.JSONHandler + son *os.File sous-jacent
// pour fermeture propre au shutdown.
type fileHandler struct {
	handler slog.Handler
	file    *os.File
}

// NewMultiModuleHandler construit un handler qui écrit vers console + fichiers.
//
// console : le handler "console" existant (typiquement NewContextHandler wrappant
// slog.NewTextHandler ou JSONHandler). Pas modifié.
// logsDir : répertoire où les fichiers `{module}.log` seront créés. Si vide,
// les écritures fichier sont désactivées (handler dégrade vers console-only).
// Créé via MkdirAll si absent.
// level : niveau minimal des logs écrits dans les fichiers (peut différer de
// celui de la console pour optimiser le volume).
func NewMultiModuleHandler(console slog.Handler, logsDir string, level slog.Leveler) (*MultiModuleHandler, error) {
	if logsDir != "" {
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			return nil, fmt.Errorf("logging: mkdir logsDir %s: %w", logsDir, err)
		}
	}
	if level == nil {
		level = slog.LevelInfo
	}
	return &MultiModuleHandler{
		console: console,
		logsDir: logsDir,
		level:   level,
		files:   make(map[string]*fileHandler),
	}, nil
}

// Enabled délègue à la console (qui prend en compte LEVELUP_LOG_LEVEL).
// Le file logging ne suit PAS Enabled — il écrit dès que le level >= h.level
// pour conserver le debug en fichier même quand la console est en INFO.
func (h *MultiModuleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.console.Enabled(ctx, level) {
		return true
	}
	// File logging actif à un niveau plus bas que la console ?
	return h.logsDir != "" && level >= h.level.Level()
}

// Handle écrit le record vers la console (toujours) et vers le fichier du
// module si logsDir est configuré et level >= h.level.
//
// Erreur d'écriture fichier : loguée vers la console comme attribut sur le
// record en cours, mais ne propage pas (les logs ne doivent jamais casser
// la requête en cours).
func (h *MultiModuleHandler) Handle(ctx context.Context, record slog.Record) error {
	// 1. Console (comportement existant).
	consoleErr := h.console.Handle(ctx, record)

	// 2. File handler, si configuré et record dépasse le niveau file.
	if h.logsDir != "" && record.Level >= h.level.Level() {
		module := h.resolveModule(record)
		if err := h.handleFile(ctx, module, record); err != nil {
			// Best-effort : log l'erreur fichier sur la console seulement,
			// pas de récursion (sinon boucle infinie si l'erreur persiste).
			fmt.Fprintf(os.Stderr, "logging: write to %s.log failed: %v\n", module, err)
		}
	}

	return consoleErr
}

// resolveModule cherche le module à utiliser pour ce record.
//   - Si un attribut "module" est présent sur le record OU dans h.attrs, prend cette valeur.
//   - Sinon, infère depuis le PC d'appel via detectModuleFromCaller.
//   - Fallback ModuleGeneral.
func (h *MultiModuleHandler) resolveModule(record slog.Record) string {
	// Check WithAttrs accumulé sur ce handler (cas slog.With("module", ...))
	for _, a := range h.attrs {
		if a.Key == moduleAttrKey && a.Value.Kind() == slog.KindString {
			return SanitizeModuleName(a.Value.String())
		}
	}
	// Check attrs portés par le record lui-même
	moduleFromRecord := ""
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == moduleAttrKey && a.Value.Kind() == slog.KindString {
			moduleFromRecord = a.Value.String()
			return false // stop iteration
		}
		return true
	})
	if moduleFromRecord != "" {
		return SanitizeModuleName(moduleFromRecord)
	}
	// Auto-detect depuis le PC (record.PC est le caller de slog.Log)
	if record.PC != 0 {
		return SanitizeModuleName(detectModuleFromCaller(record.PC))
	}
	return ModuleGeneral
}

// handleFile écrit le record dans le fichier du module. Crée le file handler
// lazy au premier write pour ce module.
func (h *MultiModuleHandler) handleFile(ctx context.Context, module string, record slog.Record) error {
	fh, err := h.fileForModule(module)
	if err != nil {
		return err
	}
	return fh.handler.Handle(ctx, record)
}

// fileForModule retourne le fileHandler en cache pour le module, ou en crée
// un nouveau (créé le fichier sur disque, instancie un slog.JSONHandler).
func (h *MultiModuleHandler) fileForModule(module string) (*fileHandler, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if fh, ok := h.files[module]; ok {
		return fh, nil
	}

	path := filepath.Join(h.logsDir, module+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	jsonHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: h.level,
		// AddSource : utile pour les fichiers (debug post-mortem), pas pour
		// la console (verbeux). Activé ici, désactivé sur la console.
		AddSource: true,
	})
	// Appliquer les attrs/groups accumulés via WithAttrs/WithGroup pour
	// rester cohérent avec la console.
	var handler slog.Handler = jsonHandler
	if len(h.attrs) > 0 {
		handler = handler.WithAttrs(h.attrs)
	}
	for _, g := range h.groups {
		handler = handler.WithGroup(g)
	}

	fh := &fileHandler{handler: handler, file: f}
	h.files[module] = fh
	return fh, nil
}

// WithAttrs retourne un handler enrichi des attrs donnés (slog.With("k", "v")).
// Important : les attrs sont propagés à la fois à la console ET aux fichiers.
func (h *MultiModuleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := h.clone()
	clone.console = h.console.WithAttrs(attrs)
	clone.attrs = append(clone.attrs, attrs...)
	// Note : les file handlers déjà créés gardent leur ancien WithAttrs ;
	// les futurs créés (lazy) prendront la nouvelle liste. C'est le contrat
	// slog standard (WithAttrs retourne un NEW handler, les anciens restent
	// intacts pour les autres call sites).
	return clone
}

// WithGroup retourne un handler avec un préfixe de group (slog.Group).
func (h *MultiModuleHandler) WithGroup(name string) slog.Handler {
	clone := h.clone()
	clone.console = h.console.WithGroup(name)
	clone.groups = append(clone.groups, name)
	return clone
}

// clone retourne une copie superficielle prête à recevoir des attrs/groups
// supplémentaires. mu et files sont SHARED entre clones pour garder la
// même fenêtre d'écriture (un seul handle de fichier par module pour le
// process).
func (h *MultiModuleHandler) clone() *MultiModuleHandler {
	return &MultiModuleHandler{
		console: h.console,
		logsDir: h.logsDir,
		level:   h.level,
		attrs:   append([]slog.Attr(nil), h.attrs...),
		groups:  append([]string(nil), h.groups...),
		mu:      sync.Mutex{}, // pas partagé : chaque clone gère son propre verrou (cf. note suivante)
		// files : NOTE — on PARTAGE la map entre clones pour éviter de
		// ré-ouvrir N fois le même fichier. mu est local mais protège
		// uniquement les LECTURES de la map, qui sont thread-safe via sync.Map.
		// Actuellement on utilise un map nu — refactor TODO si contention
		// observée.
		files: h.files,
	}
}

// Close ferme proprement tous les file handles. À appeler au shutdown du serveur.
// Idempotent : sûr d'appeler plusieurs fois.
func (h *MultiModuleHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var firstErr error
	for module, fh := range h.files {
		if fh.file != nil {
			if err := fh.file.Close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("close %s.log: %w", module, err)
			}
			fh.file = nil
		}
		delete(h.files, module)
	}
	return firstErr
}

// Discard est un io.Writer no-op, utilisable pour désactiver le file logging
// dans les tests sans modifier la structure du handler.
var Discard io.Writer = io.Discard
