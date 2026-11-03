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
		// LE MOINS CHER D'ABORD : le registre des refus, puis le film (un test d'existence), puis l'en-tête de l'artefact.
		if blocked(slug, short) || !filmPresent(slug, short) || !artifactStale(ctx, filepath.Join(dir, name)) {
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
