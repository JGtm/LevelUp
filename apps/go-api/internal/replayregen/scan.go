// Package replayregen — la file de régénération des rejeux après une mise à jour de l'application de bureau
// (ADR 0045).
//
// Un rejeu rangé (data/cache/replays/{titre}/{short8}.json.gz) porte la version de schéma et les révisions de
// couche qui l'ont produit. Une mise à jour qui les fait monter laisse les rejeux déjà rangés « en retard » :
// ils restent lisibles, mais ne portent pas ce que la nouvelle version sait y mettre. Ce paquet les compte et,
// sur demande EXPLICITE de l'utilisateur, les régénère un par un.
//
// Il ne connaît aucune recette de cuisson : la reconstruction d'un film est une fonction injectée (`Deps.Rebuild`),
// celle de l'action admin « construire le rejeu » (enfant borné, verrou solo, StoreArtifact). Il ne lance jamais
// rien de lui-même : seul un démarrage demandé, la reprise d'une file que l'utilisateur avait lancée
// (`replay_regen_pending`) ou une demande explicite d'un match (seconde entrée, ci-dessous) met la file en marche.
//
// Seconde entrée : la demande EXPLICITE d'un match (`Service.Enqueue`, demand.go), pour un match qui n'a pas de rejeu.
// Elle suit le même chemin (une file FIFO persistée, un film à la fois, les portes du mode jeu, de la pause et de la
// synchronisation) ; le téléchargement du film est une fonction injectée (`Deps.FetchFilm`), comme la reconstruction.
//
// Le lot tient une seconde liste : les rejeux dont le film a quitté l'archive et dont le fichier tactique est à refaire.
// Leur fichier tactique se reprojette depuis le rejeu rangé (fonctions injectées `Deps.RasterStale` et
// `Deps.RebuildRaster`), sans film ni décodage ; le rejeu n'est pas touché.
package replayregen

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/replayartifact"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// Candidate : un rejeu rangé à régénérer.
type Candidate struct {
	TitleSlug string
	// MatchID : la forme courte, celle du nom du fichier ; l'action de reconstruction l'accepte.
	MatchID string
	modTime time.Time
	// rasterOnly : le film n'est plus à l'archive ; seul le fichier tactique se refait ([Service.processRaster]).
	rasterOnly bool
}

// isStale dit si un artefact est en retard sur le binaire : pas à jour (schéma ou révisions de couche) ET ni plus
// récent que lui par son schéma (retour à une version antérieure : le garde anti-régression du rangement refuserait le
// résultat) ni produit par une version de l'application PLUS RÉCENTE que ce binaire (même schéma, autres révisions de
// couche : l'ancien décodeur refairait un rejeu récent). Une version de producteur absente ou illisible est admissible :
// l'artefact est jugé sur ses seules marques.
func isStale(h replaybuild.Header) bool {
	return !h.UpToDate() && h.SchemaVersion <= replay.SchemaVersion &&
		!h.ProduitParUneVersionPlusRecente(replaybuild.AppVersion())
}

// artifactStale lit l'EN-TETE d'UN artefact (replaybuild.ArtifactHeader : le petit fichier voisin, l'artefact entier une
// seule fois s'il manque) et dit s'il est en retard ; faux quand il a disparu ou n'est pas lisible (rien à régénérer
// d'un fichier absent ; un fichier illisible se réécrit à l'ouverture, comportement inchangé).
func artifactStale(ctx context.Context, path string) bool {
	h, ok := replaybuild.ArtifactHeader(ctx, path)
	return ok && isStale(h)
}

// scanTests : les tests que le décompte applique à chaque rejeu rangé d'un titre.
type scanTests struct {
	// filmPresent : le film du match est dans l'archive.
	filmPresent func(slug, matchID string) bool
	// blocked écarte les matchs qu'une régénération a déjà refusés dans ce processus et ceux que la cuisson refuse par
	// règle (mode non géré : Deps.RefusedByRule).
	blocked func(slug, matchID string) bool
	// rasterStale : le fichier tactique du match est à refaire ; nil = pas de seconde liste.
	rasterStale func(slug, matchID string) bool
}

// scanTitle liste les rejeux d'un titre à remettre à jour, du plus récemment rangé au plus ancien, en deux listes :
//   - films : les rejeux en retard dont le film est encore dans l'archive (ils se refont en entier) ;
//   - rasters : les rejeux dont le film n'est plus à l'archive et dont le fichier tactique est à refaire (ils ne se
//     refont pas, leur fichier tactique se reprojette depuis le rejeu rangé).
func scanTitle(ctx context.Context, dir, slug string, t scanTests) (films, rasters []Candidate, err error) {
	entries, err := replayartifact.List(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		short := e.Short
		// LE MOINS CHER D'ABORD : le registre des refus, puis le film (un test d'existence), puis l'en-tête de
		// l'artefact (rejeu en retard) ou le petit fichier tactique (film absent).
		if t.blocked(slug, short) {
			continue
		}
		present := t.filmPresent(slug, short)
		var wanted bool
		if present {
			wanted = artifactStale(ctx, filepath.Join(dir, e.Name))
		} else {
			wanted = t.rasterStale != nil && t.rasterStale(slug, short)
		}
		if !wanted {
			continue
		}
		c := Candidate{TitleSlug: slug, MatchID: short, rasterOnly: !present}
		if info, err := e.Entry.Info(); err == nil {
			c.modTime = info.ModTime()
		}
		if present {
			films = append(films, c)
		} else {
			rasters = append(rasters, c)
		}
	}
	return films, rasters, nil
}

// sortRecentFirst : les rejeux rangés le plus récemment d'abord (ce sont ceux que le joueur ouvre).
func sortRecentFirst(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].modTime.After(c[j].modTime) })
}
