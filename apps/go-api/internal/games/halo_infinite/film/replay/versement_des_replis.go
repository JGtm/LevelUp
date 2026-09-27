package replay

// versement_des_replis.go — LA SEULE PORTE ENTRE LES COMPTES DE REPLIS QUE LES COUCHES BASSES RENDENT
// EN DONNEES ET LE COMPTEUR DE LA CUISSON (lot J8.7 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25,
// decision 1 du superviseur, 2026-09-27).
//
// # POURQUOI LES COMPTES VOYAGENT EN DONNEES
//
// `grammar` et `profile` sont SOUS le registre des replis (`facts/fallback`) : ils ne peuvent pas
// l importer. `killsource` et `objectives` le PEUVENT mais ne le DOIVENT PAS (ADR 0034 amende, DU-2
// (c)) : l empreinte de leur couche porterait alors chaque modification du registre, et toute entree
// neuve rouvrirait le backlog de recuisson. Leurs replis se comptent donc en entiers NOMMES dans ce
// qu ils rendent deja — le rapport du contexte de film, le resultat du kill-feed, le resultat des
// objectifs —, et c est ICI, dans la couche de publication, que chaque compte prend son nom de
// registre.
//
// # UNE TABLE, ET PAS DES APPELS
//
// [versementsDesReplis] associe chaque entree du registre au CHAMP SOURCE qui la compte. La table
// est ce qui se TESTE : `TestChaqueChampDuRapportDeGrammaireEstVerse` pose un a un chaque champ des
// rapports et exige qu il arrive au compteur sous le nom de sa ligne — un champ sans ligne, ou une
// ligne qui lit un autre champ, rougit. Le garde-rail `archlint/no_unregistered_fallback_sites_test.go`
// lit les noms de la table comme des declenchements : chaque entree doit citer ce fichier en site.
//
// # QUAND LA TABLE EST JOUEE
//
// DEUX FOIS PAR CUISSON, SUR DEUX SOURCES DISJOINTES, et c est ce qui interdit le double compte :
//
//	a la fin du BALAYAGE    le rapport du contexte de film ([versementDuBalayage]). Il tombe AVANT
//	                        la capture du rapport de replis que les faits persistes portent : une
//	                        republication depuis les faits le reprend par `Cumuler`, sans balayer.
//	a l ASSEMBLAGE          ce que les options apportent (ajoute au sous-lot suivant du lot J8.7).
//
// Une source absente vaut zero : `DeclencheN` ignore un compte nul, le rapport ne porte que ce qui
// s est declenche.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// sourcesDeReplis porte les comptes qu une passe de la table lit. Un champ nul ne verse rien.
type sourcesDeReplis struct {
	// grammaire : le rapport du contexte de film ([grammar.FilmContext.ComptesDesReplis]).
	grammaire grammar.ComptesDesReplis
}

// ligneDeVersement : une ligne de la table — l entree du registre, et le champ qui la compte.
type ligneDeVersement struct {
	nom  fallback.Nom
	lire func(sourcesDeReplis) int
}

// versementsDesReplis : LA TABLE. Une ligne par (entree, champ source) ; une entree dont le fait se
// decide dans deux couches a deux lignes, une par couche.
var versementsDesReplis = []ligneDeVersement{
	{fallback.NomChunkDeReplicationSaute, func(s sourcesDeReplis) int { return s.grammaire.ChunksDeReplicationSautes }},
	{fallback.NomTempsFortsDernierNumero, func(s sourcesDeReplis) int { return s.grammaire.TempsFortsAuDernierNumero }},
	{fallback.NomLargeursMppCalibreesSurLeFilm, func(s sourcesDeReplis) int { return s.grammaire.LargeursMPPCalibrees }},
	{fallback.NomI0PorteEtRegionParDefaut, func(s sourcesDeReplis) int { return s.grammaire.I0PorteEtRegionParDefaut }},
	{fallback.NomBandeBipedeComblee, func(s sourcesDeReplis) int { return s.grammaire.SlotsBipedesComblees }},
	{fallback.NomLargeursMondeParDefautConservees, func(s sourcesDeReplis) int { return s.grammaire.LargeursMondeParDefaut }},
	{fallback.NomIndexDeRegionLargeurUn, func(s sourcesDeReplis) int { return s.grammaire.IndexDeRegionLargeurUn }},
	{fallback.NomLargeursMppParDefaut, func(s sourcesDeReplis) int { return s.grammaire.LargeursMPPParDefaut }},
	{fallback.NomChunksApresTrouAbandonnes, func(s sourcesDeReplis) int { return s.grammaire.ChunksApresTrouAbandonnes }},
	{fallback.NomAncreSansVieDeltaEcartee, func(s sourcesDeReplis) int { return s.grammaire.AncresSansVieDelta }},
	{fallback.NomRegistreInconnuSansLecteurDeTroncature, func(s sourcesDeReplis) int { return s.grammaire.RegistreInconnu }},
	{fallback.NomAmorceGrenadeProfilDeReference, func(s sourcesDeReplis) int { return s.grammaire.AmorceGrenadeDeReference }},
	{fallback.NomControleCorruptionSectionAbsente, func(s sourcesDeReplis) int { return s.grammaire.ControleDeCorruptionNonDeclare }},
	{fallback.NomLocalisationLargeurLibre, func(s sourcesDeReplis) int { return s.grammaire.LocalisationsALargeurLibre }},
}

// verserLesReplis joue la table sur `s` : chaque ligne verse son champ au compteur sous son nom.
func verserLesReplis(fb *fallback.Compteur, s sourcesDeReplis) {
	for _, v := range versementsDesReplis {
		fb.DeclencheN(v.nom, v.lire(s))
	}
}

// versementDuBalayage verse au compteur de la cuisson le rapport des replis de `grammar` et
// `profile` que le contexte du film a accumule pendant le balayage. Appelee UNE fois, entre le
// balayage et la capture des faits ([BuildFromFilmAvecFaits]).
func versementDuBalayage(fb *fallback.Compteur, fc *grammar.FilmContext) {
	verserLesReplis(fb, sourcesDeReplis{grammaire: fc.ComptesDesReplis()})
}
