// Package service — tactical_service_grille.go : LE CHOIX DU PAS DE LA GRILLE, commun aux
// quatre substrats de l'onglet Tactique.
//
// Fichier separe de tactical_service.go (493 lignes avant ce lot, seuil du depot a 500) :
// la coupure suit la frontiere que le chantier a fait apparaitre — la-bas, QUELLE mesure on
// lit et d'ou elle vient ; ici, A QUELLE RESOLUTION on la rend.
//
// ─── LE PAS S'ADAPTE, LE PLANCHER NON (decision D6, lot 3.2 du 2026-09-09) ──────
//
// Le plancher de trois matchs distincts par cellule est ce qui rend une cellule fiable ;
// l'abaisser ne rendrait pas la carte plus vraie. C'est donc la CELLULE qui grossit —
// 0,5 -> 1 -> 2 m — jusqu'a ce que `tactical.CellulesLisiblesMin` cellules l'atteignent.
// Le detail de la regle, la suite des pas et la mesure qui fixe N vivent dans le paquet
// pur (`analysis/tactical/pas_adaptatif.go`) ; ce fichier n'est que l'orchestration.
//
// LE PAS RETENU EST PUBLIE (`TacticalRaster.PasM`) : le client en a besoin pour dessiner
// ses cellules ET pour convertir un clic en adresse de cellule. Il est aussi AFFICHE a
// l'utilisateur — un plan a 2 m est plus grossier, et cela doit se voir.
//
// LES QUATRE LECTURES PASSENT PAR ICI, et c'est voulu : un pas adaptatif applique aux
// seules positions de kill aurait laisse « par ou je sors du spawn » — la lecture la plus
// clairsemee de l'onglet, quinze secondes par vie — vide sur les memes cartes.
package service

import (
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// cellulesLisibles rend les cellules que la question fera PEINDRE, plancher applique.
//
// C'EST LA MESURE QUI DECIDE DU PAS, ET ELLE DOIT ETRE CELLE QU'ON AFFICHE : la lecture
// signee (« ou je gagne ») applique son plancher PAR COTE (trois victoires ET trois
// defaites), donc elle a moins de cellules lisibles a densite egale. Compter ici les
// cellules de la lecture NON signee aurait retenu un pas auquel le plan signe reste vide.
func cellulesLisibles(raster *tactical.Raster, question string) []domain.CelluleTactique {
	if question == domain.TacticalQuestionGagne {
		return raster.CellulesSignees()
	}
	return raster.Cellules()
}

// rasteriser choisit le pas de la grille pour une lecture qui part de POINTS (les trois
// lectures de `kill_positions`).
func rasteriser(univers domain.TacticalUnivers, question string,
	points []domain.PositionSample) (tactical.LectureAdaptative, error) {
	return tactical.ChoisirPas(tactical.PasAdaptatifsM, tactical.CellulesLisiblesMin,
		func(g tactical.Grille) (*tactical.Raster, int, error) {
			raster, err := rasteriserSurGrille(g, univers, question, points)
			if err != nil {
				return nil, 0, err
			}
			return raster, len(cellulesLisibles(raster, question)), nil
		})
}

// rasteriserSurGrille rasterise a UN pas donne, dans la forme qu'exige la question :
// SIGNEE pour « ou je gagne » (les resultats font partie de l'entree), simple sinon.
func rasteriserSurGrille(g tactical.Grille, univers domain.TacticalUnivers, question string,
	points []domain.PositionSample) (*tactical.Raster, error) {
	if question == domain.TacticalQuestionGagne {
		resultats := make(map[string]int, len(univers.Matchs))
		for _, m := range univers.Matchs {
			resultats[m.MatchID] = m.Outcome
		}
		return tactical.RasteriseAvecResultats(g, resultats, points)
	}
	ids := make([]string, 0, len(univers.Matchs))
	for _, m := range univers.Matchs {
		ids = append(ids, m.MatchID)
	}
	return tactical.Rasterise(g, ids, points)
}

// rasteriserComptes choisit le pas d'une lecture qui part de comptes DEJA agreges par
// cellule (occupation, routes, morts isolees).
//
// `comptesPour` rend les comptes ADRESSES sur la grille demandee — c'est l'appelant qui
// sait comment : les morts isolees se reprojettent depuis leurs positions, les sidecars se
// REGROUPENT (`tactical.ReadresserComptes`) parce qu'ils sont cuits a 0,5 m et qu'aucune
// recuisson n'est necessaire pour les relire plus gros.
//
// AUCUNE DE CES LECTURES N'EST SIGNEE : le comptage passe donc par `Cellules()`.
func rasteriserComptes(matchs []string,
	comptesPour func(tactical.Grille) []tactical.CompteCellule) (tactical.LectureAdaptative, error) {
	return tactical.ChoisirPas(tactical.PasAdaptatifsM, tactical.CellulesLisiblesMin,
		func(g tactical.Grille) (*tactical.Raster, int, error) {
			raster, err := tactical.RasteriseComptes(g, matchs, comptesPour(g))
			if err != nil {
				return nil, 0, err
			}
			return raster, len(raster.Cellules()), nil
		})
}

// remplirRaster habille la reponse : cellules, echelle, cadre, pas retenu.
func remplirRaster(out *domain.TacticalRaster, raster *tactical.Raster, question string) {
	out.Cellules = cellulesLisibles(raster, question)
	if question == domain.TacticalQuestionGagne {
		out.Echelle = tactical.EchelleSymetrique(out.Cellules)
		// Les DEUX denominateurs de la lecture signee, sur l'univers entier : ils ne
		// valent pas MatchsRetenus, et leur somme lui est en general inferieure (les
		// nuls et les resultats inconnus ne participent a aucun cote).
		out.MatchsVictoire = raster.NbMatchsResultat(domain.OutcomeWin)
		out.MatchsDefaite = raster.NbMatchsResultat(domain.OutcomeLoss)
	} else {
		out.Echelle = tactical.Echelle(out.Cellules)
	}
	out.PasM = raster.PasM()
	out.Bornes = raster.Bornes()
	out.PointsIgnores = raster.PointsIgnores()
}

// grilleDemandee rend la grille d'un pas RECU DU CLIENT, avec repli sur le pas par defaut.
//
// LE CLIENT RENVOIE LE PAS QU'IL A RECU (`TacticalRaster.PasM`), et c'est indispensable :
// l'adresse d'une cellule n'a de sens qu'a un pas donne — la cellule (12, 8) d'une grille
// de 2 m couvre les cellules (48..51, 32..35) d'une grille de 0,5 m. Un detail de cellule
// resolu au mauvais pas rendrait des contributions d'ailleurs, sans rien casser.
//
// UN PAS ABSENT OU INVALIDE VAUT LE PAS PAR DEFAUT, jamais une erreur : c'est ce que
// faisaient tous les appelants avant le pas adaptatif, et un client d'une version
// anterieure doit continuer a lire la meme chose.
func grilleDemandee(pasM float64) tactical.Grille {
	g, err := tactical.NouvelleGrille(pasM)
	if err != nil {
		return tactical.GrilleParDefaut()
	}
	return g
}
