package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalFSStore implémente BinaryStore sur le filesystem local.
// Les binaires sont stockés sous {RootDir}/{kind}/{titleID}/{id}[.{variant}].{ext}
// L'écriture est atomique via tmp+rename.
type LocalFSStore struct {
	// RootDir est le répertoire racine du cache (ex: "data/cache").
	RootDir string
	// RootOverrides permet de pointer certains kinds vers des répertoires spécifiques.
	// Ex: KindMapImage → "static/maps/" pour les maps statiques existantes.
	RootOverrides map[Kind]string
}

// NewLocalFSStore crée un LocalFSStore avec le répertoire racine donné.
func NewLocalFSStore(rootDir string) *LocalFSStore {
	return &LocalFSStore{
		RootDir:       rootDir,
		RootOverrides: make(map[Kind]string),
	}
}

// WithRootOverride configure un répertoire alternatif pour un Kind donné.
func (s *LocalFSStore) WithRootOverride(kind Kind, dir string) *LocalFSStore {
	s.RootOverrides[kind] = dir
	return s
}

// Path retourne le chemin FS attendu pour une Ref.
func (s *LocalFSStore) Path(ref Ref) string {
	root := s.RootDir
	if override, ok := s.RootOverrides[ref.Kind]; ok {
		root = override
	}
	base := ref.ID
	if ref.Variant != "" {
		base = ref.ID + "." + ref.Variant
	}
	ext := extensionForKind(ref.Kind)
	if !strings.HasSuffix(base, ext) {
		base += ext
	}
	return filepath.Join(root, string(ref.Kind), ref.TitleID, base)
}

// LookupBinary retourne le payload binaire pour la ref donnée.
// Retourne (nil, nil) si le fichier n'existe pas.
func (s *LocalFSStore) LookupBinary(_ context.Context, ref Ref) (*BinaryPayload, error) {
	path := s.Path(ref)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("localfs: read %s: %w", path, err)
	}
	return &BinaryPayload{
		ContentType: detectContentType(data),
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

// PersistBinary écrit le payload de façon atomique (tmp + rename).
func (s *LocalFSStore) PersistBinary(_ context.Context, ref Ref, p BinaryPayload) error {
	path := s.Path(ref)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%w: mkdir %s: %v", ErrPersistFailed, filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, p.Bytes, 0o644); err != nil {
		return fmt.Errorf("%w: write tmp %s: %v", ErrPersistFailed, tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("%w: rename %s → %s: %v", ErrPersistFailed, tmp, path, err)
	}
	return nil
}

// extensionForKind retourne l'extension de fichier par défaut pour un Kind.
func extensionForKind(k Kind) string {
	switch k {
	case KindMedalImage,
		KindMapImage,
		KindChallengeBadge,
		KindBPTrackImage,
		KindBPBackground,
		KindSpartanEmblem,
		KindSpartanBanner,
		KindSpartanBackdrop,
		KindCareerRankImage,
		KindAchievementImage:
		return ".png"
	default:
		return ".json"
	}
}

// detectContentType détecte le MIME type depuis les magic bytes.
func detectContentType(data []byte) string {
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		return "image/png"
	}
	if len(data) >= 3 && bytes.Equal(data[:3], []byte{0xff, 0xd8, 0xff}) {
		return "image/jpeg"
	}
	return "application/octet-stream"
}

// contentHash retourne les 8 premiers octets du SHA-256 en hex.
func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}
