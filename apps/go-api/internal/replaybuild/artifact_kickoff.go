package replaybuild

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replayartifact"
)

// CompteurKickoffLecturesCompletes : artefacts lus en entier par [ArtifactKickoff] faute de fichier d'en-tete qui porte
// le coup d'envoi. Sur un parc dont les en-tetes sont complets, il reste a zero.
const CompteurKickoffLecturesCompletes = "replay_artifact_kickoff_full_reads_total"

// ArtifactKickoff lit, pour l'artefact de rejeu rangé à path, le coup d'envoi que le document publie
// (`t0FilmMs`, en ms depuis le début canonique du match), TEL QUEL : aucun détecteur n'est rejoué.
//
// OÙ IL SE LIT. `t0FilmMs` suit les pistes dans le document sérialisé : le début décompressé du fichier
// ([replayartifact.ReadHead]) ne l'atteint pas. Il se lit donc dans le fichier d'en-tête voisin (artifact_meta.go), que
// l'écrivain d'artefact tient à jour et qui fait foi quand il est présent, pas plus ancien que l'artefact et de sa
// taille. Un en-tête absent, périmé ou d'avant ce champ fait lire l'artefact en entier, une fois ; avec majEntete,
// l'en-tête est alors écrit ou complété pour que le passage suivant ne relise rien. La valeur rendue est la même dans les
// deux cas.
//
// Rend (nil, false, nil) quand aucun artefact n'est rangé, (nil, true, nil) quand le document ne publie pas de coup
// d'envoi (schéma antérieur au champ, ou refus du décodeur), et une erreur quand l'artefact existe mais ne se lit pas.
func ArtifactKickoff(ctx context.Context, path string, majEntete bool) (*int64, bool, error) {
	f, err := replayartifact.Locate(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("lecture artefact: %w", err)
	}
	m, metaFaitFoi := lireMeta(f.Path, f.Info)
	if metaFaitFoi && m.Kickoff != nil {
		return m.Kickoff.T0FilmMs, true, nil
	}
	observability.IncCounter(CompteurKickoffLecturesCompletes)
	c, err := replayartifact.Read(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("lecture artefact: %w", err)
	}
	var head struct {
		T0FilmMs *int64 `json:"t0FilmMs"`
	}
	if err := json.Unmarshal(c.JSON, &head); err != nil {
		return nil, true, fmt.Errorf("parse artefact: %w", err)
	}
	if majEntete {
		completerEntete(ctx, c, m, metaFaitFoi, head.T0FilmMs)
	}
	return head.T0FilmMs, true, nil
}

// completerEntete range le coup d'envoi lu dans le fichier d'en-tete de l'artefact c : complete l'en-tete qui fait foi
// (ses autres champs sont gardes), sinon l'ecrit en entier comme [ArtifactHeader].
func completerEntete(ctx context.Context, c replayartifact.Content, m artifactMeta, metaFaitFoi bool, t0 *int64) {
	if metaFaitFoi {
		m.Kickoff = &metaKickoff{T0FilmMs: t0}
		ecrireMeta(ctx, c.Path, m)
		return
	}
	d, ok := digestFromBytes(c.JSON)
	if !ok {
		return
	}
	d.Bytes = c.StoredBytes
	ecrireEntete(ctx, c.Path, d, "")
}
