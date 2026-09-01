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

// nomDeLaVariante est le nom que porte, dans un asset UGC, le fichier de la VARIANTE JOUEE.
//
// CE LITTERAL EST UNE MESURE, PAS UNE CONVENTION DEVINEE — et il a failli coûter cher. Un asset
// de carte sert souvent DEUX `.mvar` : la carte de BASE, nommee d'apres le niveau
// (`btb_highpower.mvar`, `ctf_aquarius.mvar`), et la VARIANTE jouee, nommee `map.mvar`. Les
// deux ont des socles DIFFERENTS.
//
// PREUVE (2026-09-01), par les comptes d'objets que le catalogue enregistre lui-meme :
//
//	Highpower Sentry Defense  catalogue 421 objets · map.mvar 421 · btb_highpower.mvar 524
//	Aquarius - Ranked         catalogue 236 objets · map.mvar 236 · ctf_aquarius.mvar 349
//
// Une passe de re-validation qui avait pris « le plus gros fichier » a produit des socles
// deplaces de 22 a 80 METRES sur neuf cartes — des chiffres qui ne decrivaient aucune mise a
// jour du jeu, mais la carte de BASE plaquee sur la variante. Avec `map.mvar`, sept de ces neuf
// ecarts disparaissent et six cartes retombent socle pour socle.
const nomDeLaVariante = "map.mvar"

// FetchMvarForMap rend le contenu du `.mvar` d'une carte et le nom de fichier retenu.
//
// ORDRE DE PREFERENCE, et il n'est pas negociable : `map.mvar` d'abord (la VARIANTE, cf.
// ci-dessus), puis le fichier que le catalogue d'objectifs declare, puis le premier de la
// liste. Le fichier declare vient APRES parce qu'il peut nommer la carte de base : le
// catalogue d'objectifs enregistre, pour plusieurs cartes, le nom du niveau et non celui de
// la variante.
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
		}
	}
	for _, p := range chemins {
		if filepath.Base(p) == nomDeLaVariante {
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
