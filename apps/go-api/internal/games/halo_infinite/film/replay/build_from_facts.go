package replay

// build_from_facts.go — REJOUER UN DOCUMENT DEPUIS LES FAITS PERSISTES, SANS TOUCHER AU FILM
// (lot 4.1.2 du PLAN_DECODEUR_FILM, 2026-09-17).
//
// # LES DEUX PORTES, ET CE QUI LES SEPARE
//
//	[BuildFromFilmAvecFaits]  BALAIE le film, assemble, ET rend les faits a persister
//	[BuildFromFacts]          assemble DEPUIS les faits — aucun octet de film, aucun chunk
//	                          decompresse, aucune grammaire exercee
//
// `BuildFromPositions` est PUR depuis toujours : il ne lui manquait que ses entrees. C est tout ce
// que ce fichier fait — reposer les entrees et l appeler. Le pendant public existait deja en trois
// lignes, en test (`FilmFacts.options()`).
//
// # CE QUI SE RESTITUE, ET POURQUOI CHACUN
//
//	les ENTREES      `FilmInputs.applyTo` — la SEULE ecriture des entrees sur les options, la
//	                 meme que le chemin du film emploie.
//	l IDENTITE       sans elle `coverage.decoder.build` sort vide et le bloc `registry` est
//	                 absent : l ambiguite que D-7 interdit. Elle voyage en POINTEUR, parce que
//	                 son absence a un sens (film sans section d identification).
//	les REPLIS       par [fallback.Compteur.Cumuler], et le rapport persiste ne porte QUE le
//	                 balayage. MESURE DU 2026-09-17 : 2 des 18 sites de `Declenche` sont du
//	                 balayage, 16 de l assemblage. Les seize se re-declenchent TOUT SEULS ici,
//	                 puisque l assemblage rejoue : cumuler un rapport d apres-assemblage les
//	                 compterait DEUX FOIS, et `coverage.fallbacks` ne serait plus le meme.
//
// # CE QUI NE SE RESTITUE PAS, ET N A PAS A L ETRE
//
// Les entrees de l APPELANT (catalogues de zones, socles, roster, faits de la base, table de
// reglement) arrivent par `opt`, exactement comme sur le chemin du film : elles ne viennent pas du
// film, donc elles ne sont pas dans les faits. C est la frontiere ecrite en tete d `options.go`.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// BuildFromFacts assemble le document de rejeu DEPUIS des faits persistes.
//
// AUCUN OCTET DE FILM. L appelant a la charge d avoir verifie la FRAICHEUR des faits
// ([FilmFactsEntete.Utilisable]) : cette fonction ne re-juge pas, elle assemble.
//
// HORS LIGNE comme son jumeau — ne jamais appeler depuis un chemin de requete ; l API sert
// l artefact pre-construit.
func BuildFromFacts(matchID, titleSlug string, f *FilmFactsFile, opt Options) ReplayDocument {
	if opt.Fallbacks == nil {
		opt.Fallbacks = fallback.NouveauCompteur()
	}
	// LES REPLIS DU BALAYAGE, CUMULES AVANT L ASSEMBLAGE : ceux de l assemblage tomberont
	// ensuite, dans le meme compteur, exactement comme sur le chemin du film.
	opt.Fallbacks.Cumuler(f.Fallbacks)
	opt.FilmIdentity = f.Identity
	in := f.Facts.FilmInputs
	in.applyTo(&opt)
	return BuildFromPositions(matchID, titleSlug, in.Positions, in.Fire, opt)
}

// faitsDuBalayage capture les faits d un film JUSTE APRES son balayage — appelee par
// [BuildFromFilmAvecFaits], entre le balayage et l assemblage.
//
// LE MOMENT EST LA REGLE. Le compteur de replis nait AVANT le premier balayage
// (`build_from_film.go`) et vit jusqu a la fin de l assemblage ; capturer son rapport ICI, et pas
// a la sortie de `BuildFromPositions`, est ce qui fait que le rapport persiste ne porte QUE le
// balayage. Cf. l en-tete de ce fichier pour la mesure qui l exige.
//
// LA CLE DE CUISSON VIENT DES ACCESSEURS DE LA PRODUCTION, jamais d une copie de la regle :
// [grammar.FilmContext.I0Layout] rend le decoupage EMPLOYE (impose par le catalogue, ou
// auto-detecte en repli sur une entree de carte invalide) et [grammar.FilmContext.ImposedLayout]
// dit LEQUEL DES DEUX. Sur Live Fire les deux divergent d un bit sur X, donc d un facteur deux sur
// le pas de quantification : c est exactement ce que la cle de cuisson protege.
//
// DECOUPAGE ILLISIBLE = AUCUN FAIT (nil), JAMAIS DES FAITS SANS CLE. Sans le decoupage employe,
// les quanta ne se redequantifient pas ; ecrire les faits quand meme produirait un fichier dont la
// relecture rendrait des coordonnees fausses. L appelant journalise la degradation.
func faitsDuBalayage(matchID string, fc *grammar.FilmContext, opt Options,
	in FilmInputs,
) *FilmFactsFile {
	lay, err := fc.I0Layout()
	if err != nil {
		return nil
	}
	var module string
	if opt.MapQuant != nil {
		module = opt.MapQuant.Module
	}
	return &FilmFactsFile{
		Coverage: *couvertureDuDecodeur(opt.FilmIdentity),
		Facts: FilmFacts{
			Film:           matchID,
			MapModule:      module,
			AxisW:          lay.AxisW,
			LayoutDetected: fc.ImposedLayout() == nil,
			FilmInputs:     in,
		},
		Identity:  identiteDeFaits(opt.FilmIdentity),
		Fallbacks: opt.Fallbacks.Rapport(),
	}
}

// identiteDeFaits recopie l identite du film pour le fichier de faits.
//
// PAR VALEUR DERRIERE UN POINTEUR NEUF : les faits ne doivent pas partager d objet avec les
// options d une cuisson en cours, sans quoi une ecriture ulterieure sur l une changerait l autre.
// nil reste nil — son sens est « le film ne porte AUCUNE section d identification ».
func identiteDeFaits(id *profile.FilmIdentity) *profile.FilmIdentity {
	if id == nil {
		return nil
	}
	copie := *id
	return &copie
}
