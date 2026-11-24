package replayartifacts

// cuisson_bilan.go — LE BILAN D'UN CYCLE DE CUISSON : ce que le cycle a fait et pourquoi il n'a rien fait.
// Extrait de cuisson.go pour tenir le seuil de taille de fichier ; mêmes déclarations, même paquet.

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// bilanCuisson : ce que le cycle a fait, et POURQUOI il n'a rien fait quand c'est le cas.
//
// Un simple couple (construits, films) ne distingue pas « tout était déjà à jour » de « tous
// les films sont expirés » ni de « la cuisson a échoué cinq fois » — trois situations qui
// appellent trois actions différentes et qui s'écrivaient toutes « 0 ».
type bilanCuisson struct {
	construits  int
	filmsSauves int
	dejaAJour   int
	sansFilm    int
	// reportes : films que le serveur n'a pas encore FINALISÉS (lot L3, cf. cuisson_non_finalise.go)
	// — ni archivés ni cuits, repris au cycle suivant. SÉPARÉS DE `sansFilm`, qui dit « absent ou
	// expiré » : un film reporté existe, et il sera complet dans une minute.
	reportes int
	// reportesModeJeu : matchs dont la cuisson n'a pas démarré parce que le jeu tournait (mode jeu,
	// [gameproc.Mode.HeavyWorkAllowed]). Leur film est persisté ; l'artefact se construit au rattrapage
	// qui suit la partie. Comptés avec les reportés dans `traites`.
	reportesModeJeu int
	// reportesVerrou : matchs dont la cuisson n'a pas demarre parce qu'un autre decodage de film tenait le verrou solo de la
	// machine (file de regeneration des rejeux, action admin). Leur film est persiste ; l'artefact se construit au cycle
	// suivant. Ni un echec ni un avertissement par film.
	reportesVerrou int
	echecs         int
	// ecartes : films MIS DE COTE, jamais comptes en echec : cle ecrite absente de la table de
	// profil (lot 3.1.1, `replaybuild.ErrUnknownFilmKey`), ou film au cache que la cuisson refuse
	// comme NON FINALISE (lot L3, `finalise.ErrFilmNonFinalise`). SEPARE DES ECHECS, et ce n est
	// pas cosmetique : un echec appelle un diagnostic, un ecarte appelle une LIGNE DE TABLE
	// (`docs/RUNBOOK_FILM_PROFILES.md`) ou un film a completer — les melanger noierait l evenement
	// « le jeu a change » dans le bruit des cuissons qui plantent. Le journal PAR MATCH dit lequel.
	ecartes      int
	budgetEpuise bool
	// horsCatalogue : films refusés pour carte HORS CATALOGUE de bornes, par carte (ses noms
	// candidats, tels que la cuisson les a reçus). Comptés aussi dans `echecs` ; signalés une fois
	// par carte et par cycle ([signalerCartesHorsCatalogue]).
	horsCatalogue map[string]int
	// ranges : les artefacts RANGÉS par ce cycle, à dériver une fois TOUTE cuisson terminée
	// (cf. derivations.go). Ils voyagent dans le bilan plutôt que d'être dérivés ici : un
	// burst writer au milieu d'une boucle de décodage est exactement ce que le découpage du
	// paquet interdit.
	//
	// UNE SEULE LISTE POUR LES QUATRE DÉRIVATIONS (2026-09-06) : elles lisent le MÊME
	// document rangé. Deux listes séparées auraient divergé au premier crochet ajouté — c'est
	// déjà ce que disait l'ancien champ `usage`, qui servait aussi les statistiques d'Assaut.
	ranges []ArtefactRange
}

// noterHorsCatalogue compte un film refusé pour sa carte hors catalogue.
func (b *bilanCuisson) noterHorsCatalogue(mapNames []string) {
	if b.horsCatalogue == nil {
		b.horsCatalogue = make(map[string]int, 1)
	}
	b.horsCatalogue[fmt.Sprint(mapNames)]++
}

// signalerCartesHorsCatalogue dit en WARN, UNE FOIS PAR CARTE ET PAR CYCLE, chaque carte hors
// catalogue de bornes et le nombre de ses films que le cycle n'a pas pu cuire : une carte absente
// du catalogue bloque TOUS ses films, et cela doit se voir sans le niveau DEBUG. Ordre des cartes
// stable (tri) : deux cycles identiques écrivent le même journal.
func signalerCartesHorsCatalogue(ctx context.Context, d Deps, parCarte map[string]int) {
	cartes := make([]string, 0, len(parCarte))
	for carte := range parCarte {
		cartes = append(cartes, carte)
	}
	sort.Strings(cartes)
	for _, carte := range cartes {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — carte hors catalogue de bornes, films non cuits",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "carte", carte, "films", parCarte[carte])
	}
}

// traites : les matchs que la boucle a examinés et décidés (construits, à jour, sans film,
// reportés, en échec) — la ligne « traites » des journaux de budget.
func (b bilanCuisson) traites() int {
	return b.construits + b.dejaAJour + b.sansFilm + b.reportes + b.reportesModeJeu + b.reportesVerrou + b.echecs
}

// aReprendre : les matchs d'un lot de `selectionnes` que la passe laisse à reprendre au cycle suivant,
// et que le cycle suivant PEUT traiter — ceux qu'elle n'a pas atteints (budget de temps, annulation)
// et ceux qu'elle a reportés (film pas encore finalisé, partie en cours, autre décodage en cours).
//
// Un match essayé et non résolu n'y figure pas : échec de cuisson, film mis de côté (profil
// inconnu), carte hors catalogue. La cadence courte qui le reprendrait sans fin ne le ferait pas
// avancer ; il se reprend à la cadence réglée.
func (b bilanCuisson) aReprendre(selectionnes int) int {
	pasAtteints := selectionnes - b.traites() - b.ecartes
	return max(0, pasAtteints) + b.reportes + b.reportesModeJeu + b.reportesVerrou
}
