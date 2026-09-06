// Package teammates — teammates_squad_echange.go : L'ECHANGE de la page Escouade.
//
// « Qui echange pour qui », la distribution du delai, le taux d'echange du camp et son
// ecart a l'habituel. Une section de plus du `pageData`, servie par le meme appel que les
// autres — la page ne gagne aucune cle de cache.
//
// ─── CE FICHIER NE CALCULE AUCUN TAUX ──────────────────────────────────────────────────
//
// Tout ce qui est mesure vient de `analysis/coordination` : `Echanges` (qui a venge qui,
// dans la fenetre), `Ripostes` (les memes, sans borne, pour les deux barres hors fenetre)
// et `Mesurer` (la seule forme sous laquelle un taux quitte le domaine). Ce fichier
// PROJETTE : il decoupe des perimetres, nomme des joueurs, et range des delais dans des
// intervalles. Un quotient ecrit ici serait un second taux d'echange sous le meme nom.
//
// ─── UNE SEULE LECTURE DE BASE, DEUX PERIMETRES ────────────────────────────────────────
//
// Le journal des morts est lu UNE FOIS, sur tout l'historique du joueur (aucune carte,
// aucun filtre : `TacticalQuery{PlayerXUID}`), puis resserre EN GO sur deux ensembles de
// matchs :
//
//	le PERIMETRE FILTRE   les matchs de la composition retenus par les filtres de la page ;
//	l'HABITUEL            tout l'historique de la composition — la reference.
//
// C'est la mecanique de baseline du briefing de l'Explorateur (buildBriefingBaseline :
// le scope compare a l'historique complet, dont il est un sous-ensemble). La lire en deux
// requetes filtrees differemment aurait donne deux univers a reconcilier ; et l'habituel
// exige de toute facon l'historique entier.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// bornesDelaiMs decoupe la distribution du delai : chaque valeur ouvre un intervalle,
// le dernier n'a pas de borne haute. Les cinq premieres couvrent la fenetre d'echange,
// les deux dernieres sont HORS fenetre.
//
// C'EST LA SEULE DEFINITION DES INTERVALLES. bucketDelai la PARCOURT au lieu de la
// recopier en arithmetique : la version precedente divisait par 1 000 et bornait a 4,
// et changer une borne ici l'aurait laissee ranger les delais selon l'ancienne
// (correction G1, revue du 2026-09-06).
var bornesDelaiMs = []int64{0, 1000, 2000, 3000, 4000, 5000, 7000}

// bucketDelai rend l'indice d'intervalle d'un delai de riposte.
//
// LA BORNE DE LA FENETRE EST INCLUSE, comme dans coordination.chercheVengeur
// (`delai > fenetre` sort) : une riposte a 5 000 ms exactement est un echange, a
// 5 001 ms n'en est pas un. C'est la SEULE asymetrie de ce decoupage, et elle porte sur
// la borne qui decide du taux — d'ou son traitement explicite plutot qu'un `<=` glisse
// dans la boucle, qui rendrait tous les intervalles fermes des deux cotes.
func bucketDelai(delaiMs int64) int {
	dernier := len(bornesDelaiMs) - 1
	for i := 0; i < dernier; i++ {
		haut := bornesDelaiMs[i+1]
		if delaiMs < haut {
			return i
		}
		// Le delai EGAL a la fenetre appartient au dernier intervalle qui la termine,
		// pas au premier qui la depasse.
		if delaiMs == haut && haut == coordination.FenetreEchangeMs {
			return i
		}
	}
	return dernier
}

// buildSquadEchange assemble la section « echange ».
//
// Retourne nil dans les cas ou il n'y a rien a dire, et c'est une OMISSION assumee (jamais
// des zeros, qui se liraient comme une contre-performance) :
//
//	lecteur non cable            aucune lecture possible ;
//	porte de capability fermee   ce titre ne nomme pas le tueur de chaque mort ;
//	perimetre vide               aucun match retenu, ou aucun joueur nommable ;
//	journal en echec             journalise puis degrade — les autres blocs restent servis ;
//	aucune mort mesuree          AUCUN match du perimetre ne porte de journal (c'est aussi
//	                             l'etat d'un titre dont les films ont tous expire).
func (s *TeammatesService) buildSquadEchange(
	ctx context.Context,
	scopeRows, habituelRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadEchange {
	if s.tacticalRepo == nil || !games.JournalDesMortsFiable(s.caps) {
		return nil
	}
	// Perimetre : memes matchs et memes joueurs que les autres blocs de la page.
	scopeIDs, xuidsOrdered, gtByXUID := firstBloodScope(scopeRows, mainGamertag, mainXUID, teammates)
	if len(scopeIDs) == 0 || len(xuidsOrdered) == 0 || mainXUID == "" {
		return nil
	}
	habituelIDs, _, _ := firstBloodScope(habituelRows, mainGamertag, mainXUID, teammates)

	lecture, err := s.tacticalRepo.KillEvents(ctx, domain.TacticalQuery{PlayerXUID: mainXUID})
	if err != nil {
		slog.WarnContext(ctx, "teammates_echange_journal_en_echec",
			"player", mainGamertag, "matchs", len(scopeIDs), "err", err)
		return nil
	}

	scope := restreindreAuxMatchs(lecture, scopeIDs)
	mesures := matchsMesures(scope)
	if mesures == 0 {
		slog.InfoContext(ctx, "teammates_echange_section_retiree_sans_mesure",
			"player", mainGamertag, "matchs", len(scopeIDs),
			"cause", "aucun match du perimetre ne porte de journal des morts publiable")
		return nil
	}

	out := &domain.SquadEchange{
		Joueurs:       joueursDuRoster(xuidsOrdered, gtByXUID),
		FenetreMs:     coordination.FenetreEchangeMs,
		MatchsMesures: mesures,
		MatchsTotal:   len(scopeIDs),
	}
	// LE DENOMINATEUR « PAR MATCH », C'EST LES MATCHS MESURES (correction G2,
	// 2026-09-06). Le numerateur ne peut venir que des matchs dont le journal des
	// morts est lisible ; diviser par TOUS les matchs du filtre ferait varier la
	// grandeur avec la couverture de film au lieu du jeu — 20 matchs sur 20 decodes
	// et 2 sur 20 rendraient 0,20 et 0,02 pour exactement le meme jeu. Le compte du
	// filtre reste publie a part (MatchsTotal), et le bandeau dit « N mesures sur M ».
	bilan := coordination.Echanges(scope.Events, scope.Univers.Equipes)
	campScope := campDuJoueur(scope.Univers.Equipes, mainXUID)
	out.Couverture = couvertureDuCamp(bilan.Morts, campScope, mesures)
	out.Cellules = cellulesDuRoster(bilan.Paires, gtByXUID, mesures)
	out.Delais = distributionDesDelais(scope, campScope)

	habituel := restreindreAuxMatchs(lecture, habituelIDs)
	bilanHabituel := coordination.Echanges(habituel.Events, habituel.Univers.Equipes)
	out.Habituel = couvertureDuCamp(
		bilanHabituel.Morts, campDuJoueur(habituel.Univers.Equipes, mainXUID),
		matchsMesures(habituel))
	out.MatchsHabituel = len(habituelIDs)

	slog.InfoContext(ctx, "teammates_echange",
		"player", mainGamertag, "matchs", out.MatchsTotal, "matchs_mesures", out.MatchsMesures,
		"morts_vengeables", out.Couverture.N, "morts_vengees", out.Couverture.Brut,
		"echantillon_faible", out.Couverture.EchantillonFaible,
		"matchs_habituel", out.MatchsHabituel, "paires", len(out.Cellules))
	return out
}

// restreindreAuxMatchs decoupe la lecture sur un ensemble de matchs. L'UNIVERS est decoupe
// AVEC les evenements : un match retenu qui ne porte aucune mort compte au denominateur
// « par match », et le deduire des evenements l'effacerait (defaut mesure en phase 1 du
// plan tactique).
func restreindreAuxMatchs(lecture domain.TacticalKillEvents, matchIDs []string) domain.TacticalKillEvents {
	garde := make(map[string]struct{}, len(matchIDs))
	for _, id := range matchIDs {
		garde[id] = struct{}{}
	}
	out := domain.TacticalKillEvents{
		Univers: domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}},
		Events:  make([]domain.KillEvent, 0, len(lecture.Events)),
	}
	for _, m := range lecture.Univers.Matchs {
		if _, ok := garde[m.MatchID]; ok {
			out.Univers.Matchs = append(out.Univers.Matchs, m)
		}
	}
	for matchID, equipes := range lecture.Univers.Equipes {
		if _, ok := garde[matchID]; ok {
			out.Univers.Equipes[matchID] = equipes
		}
	}
	for _, e := range lecture.Events {
		if _, ok := garde[e.MatchID]; ok {
			out.Events = append(out.Events, e)
		}
	}
	return out
}

// matchsMesures compte les matchs du perimetre qui portent AU MOINS une mort au journal.
// Un match sans aucune ligne n'a pas ete decode (ou son film a expire) : il n'est pas
// « un match sans mort », il est un match NON MESURE, et le bandeau de couverture le dit.
func matchsMesures(lecture domain.TacticalKillEvents) int {
	vus := make(map[string]struct{}, len(lecture.Univers.Matchs))
	for _, e := range lecture.Events {
		vus[e.MatchID] = struct{}{}
	}
	return len(vus)
}

// campDuJoueur rend le predicat « ce joueur est de MON CAMP dans ce match » : moi ET mes
// coequipiers DU MATCH (decision produit de l'utilisateur, 2026-09-06).
//
// Une identite vide (bot, environnement) ou absente de la composition n'a pas d'equipe :
// elle n'est d'aucun camp, et lui en deviner un serait une invention.
func campDuJoueur(equipes domain.EquipesParMatch, moi string) func(matchID, xuid string) bool {
	return func(matchID, xuid string) bool {
		if xuid == "" {
			return false
		}
		if xuid == moi {
			return true
		}
		duMatch := equipes[matchID]
		monEquipe, jeSuisLa := duMatch[moi]
		son, ilEstLa := duMatch[xuid]
		return jeSuisLa && ilEstLa && son == monEquipe
	}
}

// couvertureDuCamp mesure le taux d'echange sur les morts de mon camp.
//
// Le denominateur du TAUX exclut les morts NON VENGEABLES (tueur inconnu, equipes
// inconnues) : compter comme un echec une mort que personne ne pouvait venger
// fausserait la mesure. `matchs` est le denominateur de la quantite PAR MATCH, et
// c'est le nombre de matchs MESURES — cf. G2.
func couvertureDuCamp(morts []domain.MortSuivie, camp func(string, string) bool, matchs int) domain.Couverture {
	vengeables, vengees := 0, 0
	for _, m := range morts {
		if !m.Vengeable || !camp(m.MatchID, m.VictimeXUID) {
			continue
		}
		vengeables++
		if m.Vengee {
			vengees++
		}
	}
	return coordination.Mesurer(vengees, vengeables, matchs)
}

// joueursDuRoster fige l'ordre des axes de la matrice : joueur principal d'abord, puis les
// coequipiers selectionnes — le meme ordre que les autres blocs de la page.
func joueursDuRoster(xuidsOrdered []string, gtByXUID map[string]string) []domain.SquadEchangeJoueur {
	out := make([]domain.SquadEchangeJoueur, 0, len(xuidsOrdered))
	for _, x := range xuidsOrdered {
		out = append(out, domain.SquadEchangeJoueur{XUID: x, Gamertag: gtByXUID[x]})
	}
	return out
}

// cellulesDuRoster garde les couples dont LES DEUX cotes sont au roster. `matchs` est
// le nombre de matchs MESURES : c'est le denominateur de la quantite par match (G2).
//
// Un vengeur de passage (allie non selectionne) compte bien au KPI — il est de mon camp —
// mais n'a aucune ligne dans la matrice : la page ne sait pas le nommer, et afficher un
// xuid nu serait pire que l'ecarter (doctrine SquadAssistPairsTable). Les axes de la
// matrice sont le roster, et rien d'autre.
func cellulesDuRoster(paires []domain.PaireEchange, gtByXUID map[string]string, matchs int) []domain.SquadEchangeCell {
	out := make([]domain.SquadEchangeCell, 0, len(paires))
	for _, p := range paires {
		vengeurGT, okV := gtByXUID[p.VengeurXUID]
		vengeGT, okC := gtByXUID[p.VengeXUID]
		if !okV || !okC {
			continue
		}
		cell := domain.SquadEchangeCell{
			VengeurXUID: p.VengeurXUID, VengeurGamertag: vengeurGT,
			VengeXUID: p.VengeXUID, VengeGamertag: vengeGT,
			Nombre: p.Nombre,
		}
		if matchs > 0 {
			cell.ParMatch = float64(p.Nombre) / float64(matchs)
		}
		out = append(out, cell)
	}
	return out
}

// distributionDesDelais range les ripostes subies par mon camp dans les sept intervalles.
//
// La lecture SANS BORNE (coordination.Ripostes) est la seule qui puisse alimenter les deux
// intervalles hors fenetre. Sous la fenetre elle coincide exactement avec Echanges (meme
// noyau, meme « premier vengeur valide »), ce qui autorise l'histogramme entier a se
// construire sur elle seule.
func distributionDesDelais(lecture domain.TacticalKillEvents, camp func(string, string) bool) []domain.SquadEchangeBucket {
	comptes := make([]int, len(bornesDelaiMs))
	for _, m := range coordination.Ripostes(lecture.Events, lecture.Univers.Equipes) {
		if !m.Vengee || !camp(m.MatchID, m.VictimeXUID) {
			continue
		}
		comptes[bucketDelai(m.DelaiMs)]++
	}
	out := make([]domain.SquadEchangeBucket, 0, len(bornesDelaiMs))
	for i, debut := range bornesDelaiMs {
		b := domain.SquadEchangeBucket{
			DebutMs:     debut,
			Ouvert:      i == len(bornesDelaiMs)-1,
			HorsFenetre: debut >= coordination.FenetreEchangeMs,
			Nombre:      comptes[i],
		}
		if !b.Ouvert {
			b.FinMs = bornesDelaiMs[i+1]
		}
		out = append(out, b)
	}
	return out
}
