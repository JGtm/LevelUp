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
//	a l ASSEMBLAGE          ce que les options apportent ([Options.ReplisHorsBalayage]) : le
//	                        resultat du kill-feed, les comptes des objectifs, le rapport de la
//	                        construction. L appelant les recalcule — ou les relit dans les faits —
//	                        a CHAQUE cuisson, et ils ne sont jamais dans le rapport persiste du
//	                        balayage : versees ici, ils comptent une fois sur les deux chemins.
//
//	LE REPLI A LA CONSULTATION ([ReplisHorsBalayage.Consultations], lot J8.7-bis) suit le
//	second chemin : l enregistreur du document se remplit PENDANT l assemblage (et la construction
//	de la cuisson), et il est lu ICI, une fois, a la cloture.
//
// Une source absente vaut zero : `DeclencheN` ignore un compte nul, le rapport ne porte que ce qui
// s est declenche.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// ReplisHorsBalayage : ce que l appelant de la cuisson apporte en comptes de replis (cf.
// [Options.ReplisHorsBalayage]).
type ReplisHorsBalayage struct {
	// KillSource : le resultat du kill-feed de la cuisson (decode sur le chemin du film, RELU dans
	// les faits sur celui des faits), ou nil. Ses statistiques portent les comptes de ses replis.
	KillSource *killsource.Result
	// Objectifs : les comptes des replis du lecteur d objectifs — le balayage du statborg (porte par
	// la section statborg des faits) et la construction du pont d identite par manche.
	Objectifs objectives.ComptesDesReplis
	// Construction : le RAPPORT des replis que la construction de la cuisson (`replaybuild`) a comptes
	// elle-meme, sous leurs noms de registre (morts neutres, relais de bot, zones,
	// resolution des frags). Recalcule a chaque cuisson, sur les deux chemins.
	Construction []fallback.Declenchement
	// Consultations : l enregistreur du repli qui se declenche a la LECTURE des series nommees (lot
	// J8.7-bis, 2026-09-28), partage par tout le document —
	// la construction de la cuisson et chaque calque y notent les evenements DISTINCTS qu ils
	// consultent. Nil : l assemblage en ouvre un ([assemblage.ouvrir]).
	Consultations *objectives.ReplisALaConsultation
}

// sourcesDeReplis porte les comptes qu une passe de la table lit. Un champ nul ne verse rien.
type sourcesDeReplis struct {
	// grammaire : le rapport du contexte de film ([grammar.FilmContext.ComptesDesReplis]).
	grammaire grammar.ComptesDesReplis
	// killsource : les statistiques du kill-feed (zero sans resultat) et ses deux comptes lus sur
	// le resultat (indices inferes de la bijection).
	killsource killsource.Stats
	// bijectionInferee : `Result.Roster.FilmTable.Inferred`.
	bijectionInferee int
	// objectifs : les comptes du lecteur d objectifs ([ReplisHorsBalayage.Objectifs]).
	objectifs objectives.ComptesDesReplis
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
	{fallback.NomAncrageBipedeApresLaMarche, func(s sourcesDeReplis) int { return s.grammaire.AncragesBipedesApresLaMarche }},
	{fallback.NomPistesDuMondeApresLaMarche, func(s sourcesDeReplis) int { return s.grammaire.PistesDuMondeApresLaMarche }},
	{fallback.NomCreationsDuMondeApresLaMarche, func(s sourcesDeReplis) int { return s.grammaire.CreationsDuMondeApresLaMarche }},

	// `killsource` : ses statistiques (lot J8.7, sous-lot killsource). Une entree que le decodeur
	// comptait DEJA sous un autre nom est lue a sa source d origine, sans recopie.
	{fallback.NomRecordDesynchroniseJete, func(s sourcesDeReplis) int { return s.killsource.Replis.RecordsDesynchronises }},
	{fallback.NomDeadstateIndiceHorsRoster, func(s sourcesDeReplis) int { return s.killsource.Replis.IndicesHorsRoster }},
	{fallback.NomDeadstateCategorieHorsEnum, func(s sourcesDeReplis) int { return s.killsource.Replis.CategoriesHorsEnum }},
	{fallback.NomLocalisationLargeurLibre, func(s sourcesDeReplis) int { return s.killsource.Replis.LocalisationsALargeurLibre }},
	{fallback.NomRosterNomInvente, func(s sourcesDeReplis) int { return s.killsource.Replis.NomsInventes }},
	{fallback.NomBijectionHongroiseDuFeed, func(s sourcesDeReplis) int { return s.bijectionInferee }},
	{fallback.NomCoupleRecolleSurLeVoisin, func(s sourcesDeReplis) int { return s.killsource.Couples.Recolles }},
	{fallback.NomChunkDuPiedParArgmax, func(s sourcesDeReplis) int { return s.killsource.Replis.PiedParArgmax }},
	{fallback.NomChaineEvenementCodeNonModelise, func(s sourcesDeReplis) int { return s.killsource.Replis.ChainesArretees }},
	{fallback.NomKillRattrapeHorsVueA, func(s sourcesDeReplis) int { return s.killsource.Replis.KillsRattrapes }},
	{fallback.NomTypeDeChunkPerduDuManifeste, func(s sourcesDeReplis) int { return s.killsource.Replis.TypeDeChunkPerdu }},
	{fallback.NomMortNonRevendiqueeLaPlusProche, func(s sourcesDeReplis) int { return s.killsource.Appariement.NonRevendiqueeFenetre }},
	{fallback.NomMortDeBotPremierCandidat, func(s sourcesDeReplis) int { return s.killsource.Appariement.BotFenetre }},
	{fallback.NomAppariementParFenetreTemporelle, func(s sourcesDeReplis) int {
		return s.killsource.Appariement.Fenetre + s.killsource.Assist.ParLaFenetre
	}},
	{fallback.NomSondeNonLanceePorteRelachee, func(s sourcesDeReplis) int { return s.killsource.Replis.SondeNonLancee }},
	{fallback.NomLibelleDeSourceAutres, func(s sourcesDeReplis) int { return s.killsource.Replis.LibellesAutres }},
	{fallback.NomControleCorruptionSectionAbsente, func(s sourcesDeReplis) int { return s.killsource.Replis.ControleDeCorruptionNonDeclare }},
	{fallback.NomLargeurMotDePoigneeInferee, func(s sourcesDeReplis) int { return s.killsource.Replis.MotDePoigneeInfere }},
	{fallback.NomTypeDAssistanceApprisParFilm, func(s sourcesDeReplis) int { return s.killsource.Replis.AssistancesAuTypeAppris }},

	// `objectives` : le balayage du statborg et la construction du pont d identite par manche (lot
	// J8.7, sous-lot objectives).
	{fallback.NomEnregistrementStatborgAbandonne, func(s sourcesDeReplis) int { return s.objectifs.EnregistrementsAbandonnes }},
	{fallback.NomComposantsStatborgArretes, func(s sourcesDeReplis) int { return s.objectifs.ComposantsArretes }},
	{fallback.NomTableIdentiteVide, func(s sourcesDeReplis) int { return s.objectifs.TablesIdentiteVides }},
	{fallback.NomDebutDeMancheAuMinimum, func(s sourcesDeReplis) int { return s.objectifs.DebutsDeMancheAuMinimum }},
	{fallback.NomSlotAbandonneAuPremierArrive, func(s sourcesDeReplis) int { return s.objectifs.SlotsAbandonnes }},
	{fallback.NomEmissionHorsDomaineJetee, func(s sourcesDeReplis) int { return s.objectifs.EmissionsHorsDomaineJetees }},
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

// versementDeLAssemblage verse au compteur de la cuisson ce que l appelant apporte
// ([Options.ReplisHorsBalayage]). Appelee UNE fois, en fin d assemblage, juste avant la publication
// de `coverage.fallbacks` ([assemblage.clore]) — sur les DEUX chemins, film et faits.
func versementDeLAssemblage(fb *fallback.Compteur, r ReplisHorsBalayage) {
	s := sourcesDeReplis{objectifs: r.Objectifs.Plus(r.Consultations.ComptesDesReplis())}
	if r.KillSource != nil {
		s.killsource = r.KillSource.Stats
		s.bijectionInferee = r.KillSource.Roster.FilmTable.Inferred
	}
	verserLesReplis(fb, s)
	fb.Cumuler(r.Construction)
}

// consultations rend l enregistreur des replis a la consultation du document
// ([ReplisHorsBalayage.Consultations]) : c est lui que chaque lecture de series nommees ou
// d identite par manche du rejeu recoit.
func (o Options) consultations() *objectives.ReplisALaConsultation {
	return o.ReplisHorsBalayage.Consultations
}

// enregistreurDesConsultations rend l enregistreur que l appelant a fourni, ou en ouvre un : sans lui,
// un document construit hors de la cuisson (outil, test) consulterait ses series sans rien compter.
// Appelee UNE fois, a l ouverture de l assemblage — les calques recoivent ensuite la meme valeur.
func (o Options) enregistreurDesConsultations() *objectives.ReplisALaConsultation {
	if o.ReplisHorsBalayage.Consultations != nil {
		return o.ReplisHorsBalayage.Consultations
	}
	return &objectives.ReplisALaConsultation{}
}

// VerserLesReplisDuContexte verse au compteur `fb` le rapport des replis de `grammar` et `profile`
// accumule par le contexte de film `fc` — la meme table que [versementDuBalayage], pour un appelant
// HORS de la cuisson : le collecteur de sync, qui ouvre son propre contexte (pont d identite, pose des
// largeurs, lectures des porteurs) et le verse UNE fois, a la sortie de sa passe (revue finale,
// 2026-10-02). Un contexte nil ne verse rien.
func VerserLesReplisDuContexte(fb *fallback.Compteur, fc *grammar.FilmContext) {
	if fc == nil {
		return
	}
	versementDuBalayage(fb, fc)
}
