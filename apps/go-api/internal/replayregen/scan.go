// Package replayregen — la file de régénération des rejeux après une mise à jour de l'application de bureau
// (ADR 0045).
//
// Un rejeu rangé (data/cache/replays/{titre}/{short8}.json) porte la version de schéma et les révisions de
// couche qui l'ont produit. Une mise à jour qui les fait monter laisse les rejeux déjà rangés « en retard » :
// ils restent lisibles, mais ne portent pas ce que la nouvelle version sait y mettre. Ce paquet les compte et,
// sur demande EXPLICITE de l'utilisateur, les régénère un par un.
//
// Il ne connaît aucune recette de cuisson : la reconstruction d'un film est une fonction injectée (`Deps.Rebuild`),
// celle de l'action admin « construire le rejeu » (enfant borné, verrou solo, StoreArtifact). Il ne lance jamais
// rien de lui-même : seul un démarrage demandé, ou la reprise d'une file que l'utilisateur avait lancée
// (`replay_regen_pending`), met la file en marche.
package replayregen

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// Candidate : un rejeu rangé à régénérer.
type Candidate struct {
	TitleSlug string
	// MatchID : la forme courte, celle du nom du fichier ; l'action de reconstruction l'accepte.
	MatchID string
	modTime time.Time
}

// isStale dit si un artefact lu est en retard sur le binaire : pas à jour (schéma ou révisions de couche) ET pas
// plus récent que lui. Un artefact d'un schéma plus récent (retour à une version antérieure de l'application) n'est
// pas à refaire : le garde anti-régression du rangement refuserait le résultat.
func isStale(d replaybuild.Digest) bool {
	return !d.UpToDate() && d.SchemaVersion <= replay.SchemaVersion
}

// artifactStale lit UN artefact et dit s'il est en retard ; faux quand il a disparu ou n'est pas lisible (rien à
// régénérer d'un fichier absent ; un fichier illisible se réécrit à l'ouverture, comportement inchangé).
func artifactStale(path string) bool {
	d, ok := replaybuild.ArtifactDigest(path)
	return ok && isStale(d)
}

// scanTitle liste les rejeux en retard d'un titre dont le film est encore dans l'archive, du plus récemment
// rangé au plus ancien. blocked écarte ceux qu'une régénération a déjà refusés dans ce processus.
func scanTitle(ctx context.Context, dir, slug string, filmPresent func(slug, matchID string) bool,
	blocked func(slug, matchID string) bool) ([]Candidate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Candidate
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || replaybuild.EstMarqueDerivations(name) {
			continue
		}
		short := strings.TrimSuffix(name, ".json")
		if blocked(slug, short) || !artifactStale(filepath.Join(dir, name)) || !filmPresent(slug, short) {
			continue
		}
		c := Candidate{TitleSlug: slug, MatchID: short}
		if info, err := e.Info(); err == nil {
			c.modTime = info.ModTime()
		}
		out = append(out, c)
	}
	return out, nil
}

// sortRecentFirst : les rejeux rangés le plus récemment d'abord (ce sont ceux que le joueur ouvre).
func sortRecentFirst(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].modTime.After(c[j].modTime) })
}
