package service

// replay_map_background_relief.go — L'IMAGE ET LES DONNÉES DU RELIEF D'UN FOND DE CARTE.
//
// Le relief d'une carte tient en deux fichiers du dossier des fonds, nommés par le sidecar
// (`reliefImage`, `reliefData`) : l'image des hauteurs (`<cle>.relief.webp`), dans le cadre et au
// calage de l'image du fond, et ses données (`<cle>.relief.json.gz`, JSON compressé) : bornes de hauteur,
// étages, traits. Ils se servent comme l'image plan : sous la même clé, par la même cascade de résolution et
// avec les mêmes gardes sur le nom de fichier. Une carte sans relief n'a pas de `reliefImage` :
// même absence que sans fond.
//
// LE RELIEF NE SURVIT PAS À SON FOND. Les données recopient le calage auquel le relief a été projeté.
// Si ce calage n'est plus celui du sidecar, le fond a changé depuis : le relief est périmé, il est
// journalisé et la route répond comme pour un relief absent. Le web retombe alors sur le Plan au lieu
// de dessiner un relief décalé.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/mapfondsidecar"
	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// versionSchemaRelief est la seule version du schéma des données du relief que le service sert : une
// autre version est un relief que ce serveur ne sait pas lire, tenu pour absent.
const versionSchemaRelief = 1

// tailleMaxDonneesRelief borne la taille DÉCOMPRESSÉE des données du relief (le plus gros mesuré :
// moins de 2 Mo). Le fichier est un contenu de disque : la borne protège la mémoire d'un fichier piégé.
const tailleMaxDonneesRelief = 32 << 20

// calageDuRelief est le calage que les données du relief recopient : celui du fond publié auquel le
// relief a été projeté.
type calageDuRelief struct {
	MetersPerPixel float64 `json:"metersPerPixel"`
	OriginX        float64 `json:"originX"`
	OriginY        float64 `json:"originY"`
	WidthPx        int     `json:"widthPx"`
	HeightPx       int     `json:"heightPx"`
}

// estCelui dit si ce calage est celui du sidecar : même origine, même échelle, mêmes dimensions.
// L'égalité est exacte : le relief recopie les nombres du sidecar, et un écart d'un pixel le décale.
func (c calageDuRelief) estCelui(fond replay.MapBackgroundCalibration) bool {
	return c.MetersPerPixel == fond.MetersPerPixel && c.OriginX == fond.OriginX &&
		c.OriginY == fond.OriginY && c.WidthPx == fond.WidthPx && c.HeightPx == fond.HeightPx
}

// MapBackgroundReliefImage retourne les octets de l'image du relief de la carte du match, et son type
// MIME. Retourne ErrMapBackgroundNotAvailable quand la carte n'a pas de fond, ou pas de relief.
func (s *replayService) MapBackgroundReliefImage(ctx context.Context, matchID string) ([]byte, string, error) {
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, "", err
	}
	return s.readReliefImage(ctx, key)
}

// MapBackgroundReliefImageForMap est la jumelle de MapBackgroundReliefImage keyée par CARTE (map_id).
func (s *replayService) MapBackgroundReliefImageForMap(ctx context.Context, mapID string) ([]byte, string, error) {
	key, err := s.resolveBackgroundKeyForMap(ctx, mapID)
	if err != nil {
		return nil, "", err
	}
	return s.readReliefImage(ctx, key)
}

// MapBackgroundReliefData retourne les octets JSON des données du relief de la carte du match, tels
// que la cuisson les a écrits. Retourne ErrMapBackgroundNotAvailable quand la carte n'a pas de fond,
// pas de relief, ou un relief périmé (cf. l'en-tête du fichier).
func (s *replayService) MapBackgroundReliefData(ctx context.Context, matchID string) ([]byte, error) {
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return s.readReliefData(ctx, key)
}

// MapBackgroundReliefDataForMap est la jumelle de MapBackgroundReliefData keyée par CARTE (map_id).
func (s *replayService) MapBackgroundReliefDataForMap(ctx context.Context, mapID string) ([]byte, error) {
	key, err := s.resolveBackgroundKeyForMap(ctx, mapID)
	if err != nil {
		return nil, err
	}
	return s.readReliefData(ctx, key)
}

// readReliefImage lit l'image du relief d'une clé de fond. Le sidecar doit être lisible : une image
// sans calage ne se superpose à rien.
func (s *replayService) readReliefImage(ctx context.Context, key string) ([]byte, string, error) {
	full, err := s.loadFullSidecar(ctx, key)
	if err != nil {
		return nil, "", err
	}
	if full.ReliefImage == "" {
		return nil, "", port.ErrMapBackgroundNotAvailable
	}
	chemin, mime, err := s.imageFileNamed(ctx, key, full.ReliefImage)
	if err != nil {
		return nil, "", err
	}
	return lisFichierImage(chemin, mime, key)
}

// readReliefData lit les données du relief d'une clé de fond : le JSON compressé du disque, décompressé,
// puis contrôlé (version de schéma, calage du sidecar). Rendu en clair, tel que la cuisson l'a écrit :
// seuls la version et le calage en sont lus. Tout manquement du contenu est un relief absent, journalisé.
func (s *replayService) readReliefData(ctx context.Context, key string) ([]byte, error) {
	full, err := s.loadFullSidecar(ctx, key)
	if err != nil {
		return nil, err
	}
	if full.ReliefData == "" {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	chemin, err := s.reliefDataFile(ctx, key, full.ReliefData)
	if err != nil {
		return nil, err
	}
	compresse, err := os.ReadFile(chemin) //nolint:gosec // asset versionné, nom validé par reliefDataFile
	if errors.Is(err, os.ErrNotExist) {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	if err != nil {
		return nil, err
	}
	blob, err := decompresseDonneesRelief(compresse)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : données du relief illisibles, relief tenu pour absent",
			"err", err, "cle", key, "fichier", full.ReliefData, "titleSlug", s.titleSlug)
		return nil, port.ErrMapBackgroundNotAvailable
	}
	if !s.donneesReliefAuCalage(ctx, key, full, blob) {
		return nil, port.ErrMapBackgroundNotAvailable
	}
	return blob, nil
}

// donneesReliefAuCalage dit si les données du relief sont servables : d'une version de schéma connue
// et projetées dans le calage du sidecar. Un manquement est journalisé.
func (s *replayService) donneesReliefAuCalage(ctx context.Context, key string, full *mapfondsidecar.MapBackground, blob []byte) bool {
	var donnees struct {
		SchemaVersion int            `json:"schemaVersion"`
		Calibration   calageDuRelief `json:"calibration"`
	}
	if err := json.Unmarshal(blob, &donnees); err != nil {
		slog.WarnContext(ctx, "rejeu 2D : données du relief illisibles, relief tenu pour absent",
			"err", err, "cle", key, "fichier", full.ReliefData, "titleSlug", s.titleSlug)
		return false
	}
	if donnees.SchemaVersion != versionSchemaRelief {
		slog.WarnContext(ctx, "rejeu 2D : version de schéma du relief inconnue, relief tenu pour absent",
			"version", donnees.SchemaVersion, "attendue", versionSchemaRelief,
			"cle", key, "fichier", full.ReliefData, "titleSlug", s.titleSlug)
		return false
	}
	if !donnees.Calibration.estCelui(full.Calibration) {
		slog.WarnContext(ctx, "rejeu 2D : relief périmé, calage différent de celui du fond : relief tenu pour absent",
			"cle", key, "fichier", full.ReliefData, "calageRelief", donnees.Calibration,
			"calageFond", full.Calibration, "titleSlug", s.titleSlug)
		return false
	}
	return true
}

// decompresseDonneesRelief décompresse le gzip des données du relief. La lecture est bornée à
// tailleMaxDonneesRelief : un fichier qui la dépasse est refusé, jamais lu en entier.
func decompresseDonneesRelief(compresse []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(compresse))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	blob, err := io.ReadAll(io.LimitReader(zr, tailleMaxDonneesRelief+1))
	if err != nil {
		return nil, err
	}
	if len(blob) > tailleMaxDonneesRelief {
		return nil, fmt.Errorf("données du relief au-delà de %d octets décompressés", tailleMaxDonneesRelief)
	}
	return blob, nil
}

// loadFullSidecar lit le sidecar complet d'une clé de fond ; une absence ou une lecture en échec rend
// l'absence du fond, journalisée.
func (s *replayService) loadFullSidecar(ctx context.Context, key string) (*mapfondsidecar.MapBackground, error) {
	path := title.NewPathResolver(s.repoRoot).MapBackgroundMetaPath(s.titleSlug, key)
	full, err := mapfondsidecar.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.DebugContext(ctx, "rejeu 2D : pas de fond figé pour cette clé", "cle", key, "titleSlug", s.titleSlug)
		} else {
			slog.WarnContext(ctx, "rejeu 2D : fond de carte illisible, fichier annexe non servi",
				"err", err, "cle", key, "path", path, "titleSlug", s.titleSlug)
		}
		return nil, port.ErrMapBackgroundNotAvailable
	}
	return full, nil
}

// reliefDataFile valide le nom du fichier des données du relief lu dans un sidecar et rend son
// chemin : un nom simple, de la forme `<cle>.relief.json.gz` (mapfondsidecar.EstDonneesRelief).
func (s *replayService) reliefDataFile(ctx context.Context, key, nomFichier string) (string, error) {
	nom := filepath.Base(nomFichier)
	if nom != nomFichier || strings.ContainsAny(nom, `/\`) || !mapfondsidecar.EstDonneesRelief(nom) {
		slog.ErrorContext(ctx, "fond de carte : nom de fichier des données du relief refusé",
			"fichier", nomFichier, "cle", key, "titleSlug", s.titleSlug)
		return "", port.ErrMapBackgroundNotAvailable
	}
	return title.NewPathResolver(s.repoRoot).MapBackgroundImageFilePath(s.titleSlug, nom), nil
}
