package haloclient

// halo_client_mvar.go — LA VARIANTE DE CARTE (.mvar), capacite OPTIONNELLE du client.
//
// POURQUOI SUR LE CLIENT ET PAS AILLEURS : il porte deja les jetons de la session de sync
// (spartan + clearance), ceux-la memes que l'UGC exige. Aller chercher des jetons ailleurs pour
// un appel marginal violerait la source unique de l'ADR 0023 — et AUCUNE re-capture n'a lieu
// ici : on se sert de la chaine deja rafraichie par l'appelant.
//
// Le rattrapage du catalogue de cartes (internal/sync/replayartifacts) asserte cette capacite
// comme il asserte `ChunksFetcher` : un client qui ne la porte pas desarme le rattrapage, il
// ne casse rien.

import (
	"context"
	"fmt"
	"path/filepath"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/mapcatalog"
)

// FetchMvarForMap rend le contenu du `.mvar` d'une carte et le nom de fichier retenu.
//
// `mvarFile` est le fichier que le catalogue d'objectifs DECLARE pour cette carte. Il est
// prefere quand l'asset le porte : sur une carte Forge, deux `.mvar` cohabitent (canevas et
// rack) et un seul est celui du catalogue. A defaut, le premier de la liste.
func (c *HaloAPIClient) FetchMvarForMap(ctx context.Context, mapID, mvarFile string,
) ([]byte, string, error) {
	ugc := mapcatalog.NewClient(&domain.HaloTokens{
		SpartanToken: c.spartanToken, ClearanceToken: c.clearanceToken,
	})
	asset, err := ugc.FetchAsset(ctx, mapID, "")
	if err != nil {
		return nil, "", err
	}
	chemins := asset.MvarPaths()
	if len(chemins) == 0 {
		return nil, "", fmt.Errorf("aucun .mvar dans l asset %s", mapID)
	}
	choisi := chemins[0]
	for _, p := range chemins {
		if filepath.Base(p) == mvarFile {
			choisi = p
			break
		}
	}
	blob, err := ugc.FetchMvar(ctx, asset, choisi)
	if err != nil {
		return nil, "", err
	}
	return blob, filepath.Base(choisi), nil
}
