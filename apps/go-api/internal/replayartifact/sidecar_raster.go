package replayartifact

// sidecar_raster.go — la lecture en place du fichier tactique d'un match (le sidecar d'occupation, `rasters/{short8}.json.gz`) :
// son verdict de fraîcheur, plein ou par son seul en-tête. L'écriture est `Write`, comme pour un artefact.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain"
)

// LireSidecarRasterCourant lit le fichier tactique en place et dit s'il est exploitable EN L'ÉTAT : format, grille et
// unité de temps courants ([domain.SidecarRasterCourant], le prédicat unique, que le service partage).
//
// Absent : silence et (_, faux), c'est le cas nominal d'un rejeu jamais projeté. Illisible ou d'un autre format :
// (_, faux) ET une ligne de journal — un fichier corrompu n'est pas une absence, et le taire ferait disparaître un
// incident derrière une réécriture muette. La comparaison à l'artefact (`artifact_schema_version`) n'est pas ici :
// elle suppose de tenir l'artefact en main.
func LireSidecarRasterCourant(ctx context.Context, path string) (domain.TacticalRasterSidecar, bool) {
	raw, err := ReadBytes(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "rasters tactiques : sidecar illisible, il sera reecrit",
				"path", path, "raison", "lecture", "err", err)
		}
		return domain.TacticalRasterSidecar{}, false
	}
	var s domain.TacticalRasterSidecar
	if err := json.Unmarshal(raw, &s); err != nil {
		slog.WarnContext(ctx, "rasters tactiques : sidecar illisible, il sera reecrit",
			"path", path, "raison", "parse", "err", err)
		return domain.TacticalRasterSidecar{}, false
	}
	if !domain.SidecarRasterCourant(&s) {
		slog.DebugContext(ctx, "rasters tactiques : sidecar d'un autre temps, il sera reecrit",
			"path", path, "schema_version", s.SchemaVersion, "pas_m", s.PasM,
			"pas_echantillon_ms", s.PasEchantillonMs)
		return domain.TacticalRasterSidecar{}, false
	}
	return s, true
}

// enteteSidecarOctets : combien d'octets du début du fichier tactique suffisent à lire son en-tête (format, grille,
// unité de temps), qui précède les joueurs dans le document.
const enteteSidecarOctets = 2048

// SidecarRasterCourantRapide dit si le fichier tactique est courant (même prédicat que [LireSidecarRasterCourant],
// [domain.SidecarRasterCourant]) sans le décoder en entier : seul son en-tête est lu, au début du document (le fichier
// pèse près d'un mégaoctet, l'en-tête quelques dizaines d'octets). Un en-tête illisible, ou un fichier absent, retombe sur
// la lecture complète, qui journalise ce qu'il faut.
func SidecarRasterCourantRapide(ctx context.Context, path string) bool {
	if entete, ok := lireEnteteSidecar(path); ok {
		return domain.SidecarRasterCourant(&entete)
	}
	_, courant := LireSidecarRasterCourant(ctx, path)
	return courant
}

// lireEnteteSidecar lit le format, la grille et l'unité de temps au début du fichier tactique. ok faux : le début du
// document ne les porte pas tous (fichier absent ou illisible, autre disposition) ; l'appelant lit alors le fichier en
// entier. Seules les valeurs scalaires qui précèdent les joueurs sont parcourues.
func lireEnteteSidecar(path string) (domain.TacticalRasterSidecar, bool) {
	head, err := ReadHead(path, enteteSidecarOctets)
	if err != nil {
		return domain.TacticalRasterSidecar{}, false
	}
	dec := json.NewDecoder(bytes.NewReader(head))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return domain.TacticalRasterSidecar{}, false
	}
	var s domain.TacticalRasterSidecar
	trouves := 0
	for trouves < 3 && dec.More() {
		cle, err := dec.Token()
		if err != nil {
			return domain.TacticalRasterSidecar{}, false
		}
		valeur, err := dec.Token()
		if err != nil {
			return domain.TacticalRasterSidecar{}, false
		}
		if _, structure := valeur.(json.Delim); structure {
			return domain.TacticalRasterSidecar{}, false // les joueurs commencent avant la fin de l'en-tête
		}
		nombre, _ := valeur.(float64)
		switch cle {
		case "schema_version":
			s.SchemaVersion, trouves = int(nombre), trouves+1
		case "pas_m":
			s.PasM, trouves = nombre, trouves+1
		case "pas_echantillon_ms":
			s.PasEchantillonMs, trouves = int(nombre), trouves+1
		}
	}
	return s, trouves == 3
}
