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
// (slog.NewJSONHandler partagé par module). I/O fichier sérialisé via mu
// par module — write small (< 4KB) atomique sur les FS modernes.
//
// Thread-safe : sharedFileState (mu + files) est partagée entre TOUS les clones
// créés via WithAttrs/WithGroup. Un seul mutex protège la map — cf. clone().
type MultiModuleHandler struct {
	console  slog.Handler
	logsDir  string
	level    slog.Leveler
	rotation RotationPolicy
	attrs    []slog.Attr
	groups   []string
	shared   *sharedFileState // pointeur partagé entre tous les clones
}

// sharedFileState regroupe le mutex et la map des file handlers.
// Partagée via pointeur entre tous les clones d'un MultiModuleHandler pour
// garantir qu'un seul mu protège la même map, quelle que soit la goroutine.
// (Correction du bug race condition : avant, chaque clone avait son propre
// sync.Mutex mais partageait la même files map → data race sous concurrence
// HTTP + goroutine de sync → certains writes fichiers silencieusement perdus.)
type sharedFileState struct {
	mu    sync.Mutex
	files map[string]*fileHandler
}

// fileHandler encapsule un slog.JSONHandler + son writer rotatif sous-jacent
// (rotation par taille, cf. rotation.go) pour fermeture propre au shutdown.
type fileHandler struct {
	handler slog.Handler
	writer  *rotatingWriter
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
// rotation : bornes de taille appliquées à CHAQUE fichier de catégorie
// (cf. rotation.go). Zéro-valeur = pas de rotation — passer
// DefaultRotationPolicy() ou Config.Rotation, jamais RotationPolicy{}.
func NewMultiModuleHandler(console slog.Handler, logsDir string, level slog.Leveler,
	rotation RotationPolicy) (*MultiModuleHandler, error) {
	if logsDir != "" {
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			return nil, fmt.Errorf("logging: mkdir logsDir %s: %w", logsDir, err)
		}
	}
	if level == nil {
		level = slog.LevelInfo
	}
	return &MultiModuleHandler{
		console:  console,
		logsDir:  logsDir,
		level:    level,
		rotation: rotation,
		shared:   &sharedFileState{files: make(map[string]*fileHandler)},
	}, nil
}

// Enabled délègue à la console (qui prend en compte LEVELUP_LOG_LEVEL).
// Le file logging ne suit PAS Enabled — il écrit dès que le level >= h.level
// pour conserver le debug en fichier même quand la console est en INFO.
func (h *MultiModuleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.console.Enabled(ctx, level) {
		return true
	}
	return h.logsDir != "" && level >= h.level.Level()
}

// Handle écrit le record vers la console (si son level la concerne) et vers le
// fichier du module (si logsDir est configuré et record.Level >= h.level).
//
// IMPORTANT : on RE-VERIFIE console.Enabled avant de déléguer. Sinon, dès que
// l'une des 2 sorties (console OU fichier) accepte le level, MultiModuleHandler.
// Enabled retourne true et slog appelle Handle — qui écrirait alors dans la
// console même si son niveau l'exclut (cas LEVELUP_LOG_LEVEL=warn +
// LEVELUP_LOGS_FILE_LEVEL=info → INFO leak dans le terminal). Les handlers slog
// standards (TextHandler/JSONHandler) ne re-filtrent pas dans Handle ; c'est
// au caller de respecter le contrat Enabled→Handle.
//
// Erreur d'écriture fichier : loguée vers stderr en best-effort, ne propage
// pas (les logs ne doivent jamais casser la requête en cours).
func (h *MultiModuleHandler) Handle(ctx context.Context, record slog.Record) error {
	var consoleErr error
	if h.console.Enabled(ctx, record.Level) {
		consoleErr = h.console.Handle(ctx, record)
	}

	if h.logsDir != "" && record.Level >= h.level.Level() {
		module := h.resolveModule(record)
		if err := h.handleFile(ctx, module, record); err != nil {
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
	for _, a := range h.attrs {
		if a.Key == moduleAttrKey && a.Value.Kind() == slog.KindString {
			return SanitizeModuleName(a.Value.String())
		}
	}
	moduleFromRecord := ""
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == moduleAttrKey && a.Value.Kind() == slog.KindString {
			moduleFromRecord = a.Value.String()
			return false
		}
		return true
	})
	if moduleFromRecord != "" {
		return SanitizeModuleName(moduleFromRecord)
	}
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
// un nouveau (crée le fichier sur disque, instancie un slog.JSONHandler).
// Utilise h.shared.mu — correctement partagé entre tous les clones.
func (h *MultiModuleHandler) fileForModule(module string) (*fileHandler, error) {
	h.shared.mu.Lock()
	defer h.shared.mu.Unlock()
	if fh, ok := h.shared.files[module]; ok {
		return fh, nil
	}

	path := filepath.Join(h.logsDir, module+".log")
	f, err := newRotatingWriter(path, h.rotation)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	jsonHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     h.level,
		AddSource: true,
	})
	var handler slog.Handler = jsonHandler
	if len(h.attrs) > 0 {
		handler = handler.WithAttrs(h.attrs)
	}
	for _, g := range h.groups {
		handler = handler.WithGroup(g)
	}

	fh := &fileHandler{handler: handler, writer: f}
	h.shared.files[module] = fh
	return fh, nil
}

// WithAttrs retourne un handler enrichi des attrs donnés (slog.With("k", "v")).
// Le clone partage le même *sharedFileState que le parent.
func (h *MultiModuleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := h.clone()
	clone.console = h.console.WithAttrs(attrs)
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

// WithGroup retourne un handler avec un préfixe de group (slog.Group).
func (h *MultiModuleHandler) WithGroup(name string) slog.Handler {
	clone := h.clone()
	clone.console = h.console.WithGroup(name)
	clone.groups = append(clone.groups, name)
	return clone
}

// clone retourne une copie du handler prête à recevoir des attrs/groups
// supplémentaires. shared est transmis par pointeur — un seul mu protège
// la même files map quelle que soit la goroutine qui utilise ce clone.
func (h *MultiModuleHandler) clone() *MultiModuleHandler {
	return &MultiModuleHandler{
		console:  h.console,
		logsDir:  h.logsDir,
		level:    h.level,
		rotation: h.rotation,
		attrs:    append([]slog.Attr(nil), h.attrs...),
		groups:   append([]string(nil), h.groups...),
		shared:   h.shared, // pointeur partagé — intentionnel
	}
}

// Close ferme proprement tous les file handles. À appeler au shutdown du serveur.
// Idempotent : sûr d'appeler plusieurs fois.
func (h *MultiModuleHandler) Close() error {
	h.shared.mu.Lock()
	defer h.shared.mu.Unlock()
	var firstErr error
	for module, fh := range h.shared.files {
		if fh.writer != nil {
			if err := fh.writer.Close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("close %s.log: %w", module, err)
			}
			fh.writer = nil
		}
		delete(h.shared.files, module)
	}
	return firstErr
}

// Discard est un io.Writer no-op, utilisable pour désactiver le file logging
// dans les tests sans modifier la structure du handler.
var Discard io.Writer = io.Discard
