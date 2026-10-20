// Package himap — cuisson_navmesh.go : LE FOND D'UNE CARTE FORGE PAR SON MAILLAGE DE NAVIGATION.
//
// CE QUE CA RESOUT, ET POURQUOI RIEN D'AUTRE N'Y ARRIVAIT. Une carte Forge organique — Isolation
// en tete — est un EMPILEMENT DE COQUES au-dessus de son arene : un type d'objet a 32
// exemplaires, pose entre Z 136 et 160 quand le sol joue est a Z 117, peint 82,7 % de l'image ;
// le retirer decouvre une deuxieme couche, puis une troisieme. Deux jours de mesures ont
// elimine toutes les soustractions : ecretage a 4/2/1 m, tranche plafonnee a +3/+6/+12,
// bornage aux volumes de mort, substitution de reference, exclusion par emprise, par aire de
// maillage, par couverture au sol, par drapeau, par altitude de pose. Le pelage lui-meme
// s'arrete : des le premier retrait il coute une ancre d'objectif, c'est-a-dire du SOL.
//
// LA SORTIE N'ETAIT PAS DE MIEUX SOUSTRAIRE MAIS DE CHANGER DE SOURCE. Chaque carte Forge
// publie un `navmesh.blob` a cote de sa variante : le maillage ou l'on MARCHE. Sur Isolation il
// vit entre Z 112,54 et 124,08 — ONZE METRES SOUS la premiere couche de coques. Il ne les
// contient pas : il n'y a rien a peler.
//
// L'ORACLE EST PASSE AVANT D'ECRIRE CE FICHIER (internal/hinavmesh, TestOracleAncresDansLeMaillage) :
// 24 des 25 ancres d'objectif d'Isolation tombent DANS un polygone du maillage, ecart d'altitude
// median 7,4 cm ; 13 sur 13 sur Kiken'na, dans un repere tout autre. Une ancre d'objectif etant
// du terrain joue par definition, le maillage decode EST le sol.
package himap

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/hinavmesh"
)

// OptionsCuissonNavmesh decrit un fond a rendre depuis un maillage de navigation.
type OptionsCuissonNavmesh struct {
	// Blob est le contenu brut du `navmesh.blob` publie avec la carte.
	Blob []byte
	// Ancres sont les positions monde des objectifs — elles cadrent l'image, comme partout
	// ailleurs dans la chaine.
	Ancres [][3]float64
	// Cle est le nom sous lequel l'asset sera publie (le map_id de la carte).
	Cle string
	// Echelle est le cote d'un pixel en metres. Zero = echelle automatique.
	Echelle float64
	// CibleCadrePx : voir OptionsCuisson.CibleCadrePx.
	CibleCadrePx int
}

// CuitCarteNavmesh rend le fond d'une carte a partir de son maillage de navigation.
//
// La chaine est la MEME que les autres a partir du rendu : meme cadre sur les ancres, meme
// tranche de jeu, meme habillage, meme calage publie. Seule la source des triangles change —
// et c'est tout l'interet : un fond issu du navmesh se superpose exactement aux positions de
// joueurs, puisque les deux vivent dans le repere monde.
func CuitCarteNavmesh(ctx context.Context, opts OptionsCuissonNavmesh) (*Rendu, BilanCuisson, error) {
	b := BilanCuisson{Module: opts.Cle, Ancres: len(opts.Ancres)}
	if len(opts.Ancres) == 0 {
		return nil, b, ErrSansAncre
	}
	m, err := hinavmesh.Decode(opts.Blob)
	if err != nil {
		return nil, b, fmt.Errorf("maillage de navigation : %w", err)
	}
	tris := m.Triangles()
	if len(tris) == 0 {
		return nil, b, fmt.Errorf("maillage de navigation vide")
	}

	r := CadreSurAncresEchelle(opts.Ancres, EchellePourCadre(opts.Ancres, opts.Echelle, opts.CibleCadrePx))
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	r.Tranche(TrancheDeJeu(zJeu))
	r.NiveauDeJeu(zJeu)

	for _, t := range tris {
		r.triangle(
			[3]float64{t[0].X, t[0].Y, t[0].Z},
			[3]float64{t[1].X, t[1].Y, t[1].Z},
			[3]float64{t[2].X, t[2].Y, t[2].Z})
	}
	b.Dessinees = len(tris)
	JugeParLesAncres(r, &b, opts.Ancres)
	slog.InfoContext(ctx, "carte cuite depuis le maillage de navigation", "carte", opts.Cle,
		"faces", len(m.Faces), "sommets", len(m.Sommets), "triangles", len(tris),
		"aireAuSol", fmt.Sprintf("%.0f m2", m.AireAuSol()),
		"ancres", fmt.Sprintf("%d/%d", b.AncresAvecSol, b.AncresDansLeCadre))
	return r, b, nil
}
