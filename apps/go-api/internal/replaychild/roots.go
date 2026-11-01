package replaychild

import (
	"io"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// NewRunner prepare le lanceur d'enfants d'un parent enraciné en repoRoot : l'enfant recoit les
// TROIS racines du parent (ressources, donnees, journaux) dans son environnement, et resout donc
// les memes chemins quel que soit son repertoire de travail (ADR 0008, amendement « two roots »,
// A-6). La racine de donnees n'est imposee que si le parent l'a posee explicitement : sinon
// l'enfant la deduit de la racine de ressources, comme lui.
func NewRunner(repoRoot string, out io.Writer) (*filmproc.Runner, error) {
	r := title.NewPathResolver(repoRoot).Roots()
	roots := filmproc.Roots{Resources: r.Resources, Logs: r.Logs}
	if r.DataSet {
		roots.Data = r.Data
	}
	return filmproc.NewRunnerRoots(roots, out)
}
