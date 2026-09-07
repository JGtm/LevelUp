// Package teammates — teammates_squad_isolement.go : LE NUAGE « ISOLEMENT X COUVERTURE »
// de la section « echange » de la page Escouade (plan tactique, phase 7, item 7.7).
//
// ─── DEUX MESURES DEJA CALCULEES, JAMAIS UN TROISIEME ALGO ─────────────────────────────
//
// L'ISOLEMENT vient de `match_death_context` (lot 7C, AU SYNC) : la MEME lecture, le MEME
// rayon PAR MATCH et le MEME `analysis/coordination.Isolement` que la lecture « isole » de
// l'onglet Tactique — seule change la population comparee (un joueur x une session, plutot
// qu'un axe « qui » x une carte). LA COUVERTURE vient de `analysis/coordination.Echanges`,
// la MEME mesure que la matrice ci-dessus, restreinte au SEUL joueur du point plutot qu'au
// camp. Ce fichier ne fait QUE decouper ces deux mesures par session et les assembler — la
// meme discipline que teammates_squad_echange.go ("ce fichier ne calcule aucun taux").
//
// ─── POURQUOI LA SESSION ET PAS LE MATCH ────────────────────────────────────────────────
//
// Un point par mort donnerait un axe binaire (vengee ou non), pas un nuage. Un point par
// match donnerait des taux calcules sur trois morts, qui ne valent que 0, 33, 50 ou 100 % —
// un damier, pas une dispersion. La session est la plus petite maille ou un taux veut dire
// quelque chose (maquette echange-escouade.html, carte « Pourquoi la vengeance ne vient
// pas »).
package teammates

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// buildSquadIsolementNuage assemble le nuage. `scope` est la lecture d'echange DEJA
// restreinte au meme perimetre filtre que le reste de la section (`restreindreAuxMatchs`
// dans buildSquadEchange) : memes matchs, memes joueurs.
//
// Absent (nil) : table de rayon non cablee, journal d'isolement en echec, ou aucune session
// ne franchit `domain.PlancherMortsSessionIsolement` — une OMISSION, jamais un nuage vide.
func (s *TeammatesService) buildSquadIsolementNuage(
	ctx context.Context,
	scope domain.TacticalKillEvents,
	scopeRows []domain.SquadMatchRow,
	xuidsOrdered []string,
	gtByXUID map[string]string,
	mainXUID string,
) *domain.SquadNuageIsolement {
	if len(s.radarRange) == 0 {
		return nil
	}
	rayon, sansRayon := rayonParMatchDuScope(scope.Univers.Matchs, s.radarRange)
	if len(rayon) == 0 {
		return nil
	}

	// LECTURE SEPAREE, MEME PORT : le contexte de mort (voisinage au sync) ne voyage pas
	// dans `TacticalKillEvents` — c'est une table differente, jointe par la meme cle
	// (match_id, victim_xuid, time_ms) que le journal des kills.
	ctxLecture, err := s.tacticalRepo.MortsAvecContexte(ctx, domain.TacticalQuery{PlayerXUID: mainXUID})
	if err != nil {
		slog.WarnContext(ctx, "teammates_isolement_journal_en_echec",
			"player", gtByXUID[mainXUID], "err", err)
		return nil
	}

	matchesDuScope := make(map[string]struct{}, len(scope.Univers.Matchs))
	for _, m := range scope.Univers.Matchs {
		matchesDuScope[m.MatchID] = struct{}{}
	}
	mortsParMatch := make(map[string][]domain.MortContexte, len(ctxLecture.Morts))
	for _, m := range ctxLecture.Morts {
		if _, in := matchesDuScope[m.MatchID]; !in {
			continue
		}
		mortsParMatch[m.MatchID] = append(mortsParMatch[m.MatchID], m)
	}

	sessions, labels := sessionsDuScope(scopeRows, matchesDuScope)
	if len(sessions) == 0 {
		return nil
	}

	points := make([]domain.SquadIsolementPoint, 0, len(xuidsOrdered)*len(labels))
	for _, xuid := range xuidsOrdered {
		for _, label := range labels {
			matchIDs := sessions[label]
			morts := mortsDuJoueurSurSession(mortsParMatch, xuid, matchIDs)
			bilanIso := coordination.Isolement(morts, rayon, matchsAvecRayon(matchIDs, rayon))
			if bilanIso.Couverture.N < domain.PlancherMortsSessionIsolement {
				continue
			}

			sessionScope := restreindreAuxMatchs(scope, matchIDs)
			bilanEch := coordination.Echanges(sessionScope.Events, sessionScope.Univers.Equipes)
			couv := couvertureDuJoueur(bilanEch.Morts, xuid, matchsMesures(sessionScope))

			points = append(points, domain.SquadIsolementPoint{
				XUID: xuid, Gamertag: gtByXUID[xuid], SessionLabel: label,
				MortsExaminees: bilanIso.Examinees, MortsIsolees: len(bilanIso.Isolees),
				PartIsolee: bilanIso.Couverture, Couverture: couv,
			})
		}
	}
	if len(points) == 0 {
		return nil
	}

	slog.InfoContext(ctx, "teammates_isolement_nuage",
		"player", gtByXUID[mainXUID], "points", len(points), "sessions", len(labels),
		"matchs_sans_rayon", sansRayon)
	return &domain.SquadNuageIsolement{
		Points:                    points,
		PlancherMortsSession:      domain.PlancherMortsSessionIsolement,
		PlancherEchantillonFaible: coordination.SeuilEchantillonFaible,
	}
}

// mortsDuJoueurSurSession filtre les morts LOCALISEES du joueur, sur les seuls matchs de la
// session, et les convertit en `domain.MortAExaminer` — la forme que `coordination.Isolement`
// consomme.
func mortsDuJoueurSurSession(
	mortsParMatch map[string][]domain.MortContexte, xuid string, matchIDs []string,
) []domain.MortAExaminer {
	out := make([]domain.MortAExaminer, 0, len(matchIDs))
	for _, matchID := range matchIDs {
		for _, m := range mortsParMatch[matchID] {
			if m.VictimXUID != xuid {
				continue
			}
			out = append(out, domain.MortAExaminer{
				MatchID: m.MatchID, X: m.X, Y: m.Y,
				PlusProcheM: m.PlusProcheM, Visibles: m.Visibles, HorsDeVue: m.HorsDeVue,
			})
		}
	}
	return out
}

// rayonParMatchDuScope resout la portee du radar de chaque match MESURE du perimetre, par sa
// variante — MEME logique que `TacticalService.rayonsParMatch` (service Tactique), reprise
// ici parce que la source (`s.radarRange map[string]int`) vit sur un service DIFFERENT, avec
// sa propre injection (cf. WithRadarRange). Un match dont la variante n'a pas de rayon SORT
// de l'univers de la lecture, pas seulement de son numerateur (correction G2, doctrine
// reprise telle quelle).
func rayonParMatchDuScope(matchs []domain.TacticalMatch, radar map[string]int) (map[string]float64, int) {
	out := make(map[string]float64, len(matchs))
	sans := 0
	for _, m := range matchs {
		if !m.Mesure {
			continue
		}
		metres, ok := radar[strings.TrimSpace(m.GameVariantName)]
		if !ok || metres <= 0 {
			sans++
			continue
		}
		out[m.MatchID] = float64(metres)
	}
	return out, sans
}

// matchsAvecRayon compte, parmi les matchs d'UNE session, ceux qui ont un rayon connu — le
// denominateur `ParMatch` de `coordination.Isolement` pour ce point.
func matchsAvecRayon(matchIDs []string, rayon map[string]float64) int {
	n := 0
	for _, id := range matchIDs {
		if _, ok := rayon[id]; ok {
			n++
		}
	}
	return n
}

// couvertureDuJoueur mesure le taux d'echange des morts d'UN SEUL joueur — la meme mesure
// que `couvertureDuCamp`, restreinte a une victime plutot qu'a un camp entier.
func couvertureDuJoueur(morts []domain.MortSuivie, xuid string, matchs int) domain.Couverture {
	vengeables, vengees := 0, 0
	for _, m := range morts {
		if !m.Vengeable || m.VictimeXUID != xuid {
			continue
		}
		vengeables++
		if m.Vengee {
			vengees++
		}
	}
	return coordination.Mesurer(vengees, vengeables, matchs)
}

// sessionsDuScope groupe les matchs DU PERIMETRE par session, dedupliques par match_id (une
// ligne de `scopeRows` peut apparaitre plusieurs fois, une par coequipier). Un match sans
// session (label nil ou vide) n'entre dans AUCUNE session : le nuage se lit par soiree, pas
// par match isole. Les labels sont rendus TRIES par l'instant du plus ancien match de la
// session — ordre stable et independant de l'iteration d'une map.
func sessionsDuScope(
	scopeRows []domain.SquadMatchRow, matchesDuScope map[string]struct{},
) (map[string][]string, []string) {
	sessions := make(map[string][]string)
	plusAncien := make(map[string]time.Time)
	vus := make(map[string]struct{}, len(scopeRows))
	for _, r := range scopeRows {
		if _, doublon := vus[r.MatchID]; doublon {
			continue
		}
		vus[r.MatchID] = struct{}{}
		if _, in := matchesDuScope[r.MatchID]; !in {
			continue
		}
		if r.SessionLabel == nil || *r.SessionLabel == "" {
			continue
		}
		label := *r.SessionLabel
		sessions[label] = append(sessions[label], r.MatchID)
		if t, ok := plusAncien[label]; !ok || r.StartTime.Before(t) {
			plusAncien[label] = r.StartTime
		}
	}
	labels := make([]string, 0, len(sessions))
	for l := range sessions {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool { return plusAncien[labels[i]].Before(plusAncien[labels[j]]) })
	return sessions, labels
}
