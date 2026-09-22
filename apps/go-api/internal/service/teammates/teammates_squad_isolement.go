// Package teammates — teammates_squad_isolement.go : LE NUAGE « POURQUOI LA VENGEANCE NE
// VIENT PAS » de la section « echange » de la page Escouade (plan tactique, phase 7,
// item 7.7 ; contrat refondu le 2026-09-19, PLAN_AJUSTEMENTS_PRE_V75 decision 4).
//
// ─── DEUX LECTURES DEJA CALCULEES, JAMAIS UN TROISIEME ALGO ────────────────────────────
//
// L'ISOLEMENT vient de `match_death_context` (lot 7C, AU SYNC) : la MEME lecture, le MEME
// rayon PAR MATCH que la lecture « isole » de l'onglet Tactique. LA RIPOSTE vient de
// `analysis/coordination.Ripostes` — la meme mecanique que « qui echange pour qui », mais
// SANS BORNE DE FENETRE.
//
// ─── LA LECTURE SANS BORNE EST UNE MATIERE DE DESSIN, JAMAIS UNE RIPOSTE ───────────────
//
// LA REGLE DES 5 s VAUT PARTOUT (decision utilisateur du 2026-09-22). Le nuage CONTINUE de
// montrer une mort dont le tueur est tombe a 30 s — sinon le lecteur ne saurait pas si les
// ripostes manquees arrivent a 5,2 s ou a 40 s —, mais il ne la DIT plus « ripostee » :
// `pointsDuJoueur` compare le delai a `coordination.FenetreEchangeMs` (borne comprise, la
// MEME regle que `bucketDelai` dans teammates_squad_echange.go) et tranche entre trois
// etats exclusifs, `Vengee` / `HorsFenetre` / ni l'un ni l'autre. Le taux de la carte
// « Riposte », le delai median du bloc et `MedianeDelaiMs` du repere ne comptent que les
// morts vengees DANS la fenetre : une seule definition de la riposte dans toute la page.
//
// ET LA MATIERE DE DESSIN A UN PLAFOND (meme decision du 2026-09-22) :
// `coordination.PlafondRiposteTardiveMs` (60 s, borne comprise). Au-dela, le tueur est mort
// de sa propre vie — reapparition 5 a 10 s, vie moyenne 20 a 40 s en arene — et son delai
// ne dit plus rien de la mort initiale : le point redevient MUET (ni etat, ni delai) et se
// pose dans la bande « jamais ripostee ». Le bloc publie `plafond_ms` a cote de
// `fenetre_ms` : le client ne code jamais 60 000 en dur.
//
// Ce fichier ne fait que JOINDRE ces deux lectures sur la cle exacte
// (match_id, victim_xuid, time_ms) et projeter. Il ne calcule aucun taux : `PartIsolee` et
// `Couverture` du repere sortent de `coordination.Isolement` et `coordination.Mesurer`.
//
// ─── POURQUOI UN POINT PAR MORT ET PLUS PAR SESSION ────────────────────────────────────
//
// L'ancien contrat posait un point par (joueur, session). Sur l'usage NOMINAL de la page —
// une soiree filtree — cela ne rendait qu'un point par joueur : trois points, pas un nuage.
// Un point par mort avec des axes CONTINUS (distance rapportee au radar x delai avant
// riposte) rend la dispersion reelle, et le gros point par joueur garde la lecture
// resumee.
package teammates

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// buildSquadIsolementNuage assemble le nuage. `scope` est la lecture d'echange DEJA
// restreinte au meme perimetre filtre que le reste de la section (`restreindreAuxMatchs`
// dans buildSquadEchange) : memes matchs, memes joueurs.
//
// Absent (nil) : table de rayon non cablee, aucun match du perimetre a rayon connu,
// journal d'isolement en echec, ou aucune mort du roster localisee — une OMISSION, jamais
// un nuage vide qui se lirait comme une mesure a zero.
func (s *TeammatesService) buildSquadIsolementNuage(
	ctx context.Context,
	scope domain.TacticalKillEvents,
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

	// Les morts du roster, par joueur, sur les seuls matchs du perimetre A RAYON CONNU :
	// un match sans rayon ne peut porter aucune abscisse, et une mort sans abscisse ni
	// bande « hors de vue » n'a nulle part ou se poser.
	mortsParJoueur := make(map[string][]domain.MortContexte, len(xuidsOrdered))
	for _, m := range ctxLecture.Morts {
		if _, ok := rayon[m.MatchID]; !ok {
			continue
		}
		if _, connu := gtByXUID[m.VictimXUID]; !connu {
			continue
		}
		mortsParJoueur[m.VictimXUID] = append(mortsParJoueur[m.VictimXUID], m)
	}

	ripostes := indexerRipostes(coordination.Ripostes(scope.Events, scope.Univers.Equipes))
	bilanEch := coordination.Echanges(scope.Events, scope.Univers.Equipes)
	mesures := matchsMesures(scope)

	out := &domain.SquadNuageIsolement{
		Morts:                     []domain.SquadIsolementMort{},
		Reperes:                   []domain.SquadIsolementRepere{},
		PlancherEchantillonFaible: coordination.SeuilEchantillonFaible,
		FenetreMs:                 coordination.FenetreEchangeMs,
		PlafondMs:                 coordination.PlafondRiposteTardiveMs,
	}
	for _, xuid := range xuidsOrdered {
		morts := mortsParJoueur[xuid]
		if len(morts) == 0 {
			continue
		}
		points := pointsDuJoueur(morts, xuid, gtByXUID[xuid], rayon, ripostes)
		out.Morts = append(out.Morts, points...)

		bilanIso := coordination.Isolement(
			mortsAExaminer(morts), rayon, len(rayon))
		out.Reperes = append(out.Reperes, domain.SquadIsolementRepere{
			XUID: xuid, Gamertag: gtByXUID[xuid],
			NbMorts:              len(points),
			MedianeDistanceRatio: medianeRatio(points),
			MedianeDelaiMs:       medianeDelai(points),
			PartIsolee:           bilanIso.Couverture,
			Couverture:           couvertureDuJoueur(bilanEch.Morts, xuid, mesures),
		})
	}
	if len(out.Morts) == 0 {
		return nil
	}

	slog.InfoContext(ctx, "teammates_isolement_nuage",
		"player", gtByXUID[mainXUID], "morts", len(out.Morts), "reperes", len(out.Reperes),
		"matchs_sans_rayon", sansRayon)
	return out
}

// cleRiposte est la cle EXACTE de jointure entre le contexte de mort et sa riposte : le
// match, la victime, et l'instant sur l'horloge du match. Aucune tolerance : les deux
// lectures sortent du MEME artefact de rejeu, sur le meme axe de temps — une jointure
// approchee apparierait deux morts voisines du meme joueur.
type cleRiposte struct {
	matchID string
	xuid    string
	timeMs  int64
}

// indexerRipostes range les morts suivies par leur cle de jointure. Une collision (deux
// morts du meme joueur au meme instant du meme match) garde la PREMIERE : le journal ne
// devrait pas en produire, et ecraser inventerait un choix.
func indexerRipostes(morts []domain.MortSuivie) map[cleRiposte]domain.MortSuivie {
	out := make(map[cleRiposte]domain.MortSuivie, len(morts))
	for _, m := range morts {
		k := cleRiposte{matchID: m.MatchID, xuid: m.VictimeXUID, timeMs: m.TimeMs}
		if _, deja := out[k]; deja {
			continue
		}
		out[k] = m
	}
	return out
}

// pointsDuJoueur projette les morts d'UN joueur en petits points du nuage, et decide de
// l'ETAT de chacun — c'est LE point de decision des trois etats du contrat.
//
//	delai <= FenetreEchangeMs        Vengee : une riposte, la meme que la carte
//	                                 « Riposte » (borne COMPRISE, cf. bucketDelai) ;
//	<= PlafondRiposteTardiveMs       HorsFenetre : le delai est publie pour que le point
//	                                 reste VISIBLE, mais ce n'est pas une riposte ;
//	au-dela du plafond, ou aucune    ni l'un ni l'autre, aucun delai.
//	riposte connue
//
// Une mort SANS riposte connue (absente du journal des kills : mort non revendiquee, ou
// match dont le journal ne porte pas cet instant) n'a pas de delai — elle va dans la bande
// haute, jamais a un delai invente. Une mort dont le tueur n'est tombe qu'APRES le plafond
// y va aussi, et pour la meme raison : personne n'a riposte.
func pointsDuJoueur(
	morts []domain.MortContexte, xuid, gamertag string,
	rayon map[string]float64, ripostes map[cleRiposte]domain.MortSuivie,
) []domain.SquadIsolementMort {
	out := make([]domain.SquadIsolementMort, 0, len(morts))
	for _, m := range morts {
		r := rayon[m.MatchID]
		p := domain.SquadIsolementMort{
			XUID: xuid, Gamertag: gamertag, MatchID: m.MatchID, TimeMs: m.TimeMs,
			HorsDeVue: m.PlusProcheM == nil,
		}
		if m.PlusProcheM != nil && r > 0 {
			ratio := *m.PlusProcheM / r
			p.DistanceRatio = &ratio
		}
		if suivie, ok := ripostes[cleRiposte{matchID: m.MatchID, xuid: xuid, timeMs: m.TimeMs}]; ok && suivie.Vengee {
			delai := suivie.DelaiMs
			// `<=` DEUX FOIS : les deux bornes sont INCLUSES, comme dans
			// coordination.chercheVengeur (`delai > fenetre` sort) et comme dans
			// bucketDelai — une riposte a 5 000 ms exactement EST une riposte, et une
			// chute a 60 000 ms exactement est encore « hors fenetre ».
			switch {
			case delai <= coordination.FenetreEchangeMs:
				p.DelaiMs = &delai
				p.Vengee = true
			case delai <= coordination.PlafondRiposteTardiveMs:
				p.DelaiMs = &delai
				p.HorsFenetre = true
			default:
				// AU-DELA DU PLAFOND, LE POINT REDEVIENT MUET (decision utilisateur du
				// 2026-09-22) : ni etat, ni delai. Le tueur est mort de sa propre vie, et
				// publier son delai laisserait le nuage suggerer un lien de cause a effet
				// que le jeu ne porte plus. La mort reste dessinee — dans la bande
				// « jamais ripostee », a sa place.
			}
		}
		out = append(out, p)
	}
	return out
}

// mortsAExaminer convertit les morts localisees en la forme que `coordination.Isolement`
// consomme — le taux d'isolement du repere reste calcule par le domaine, jamais ici.
func mortsAExaminer(morts []domain.MortContexte) []domain.MortAExaminer {
	out := make([]domain.MortAExaminer, 0, len(morts))
	for _, m := range morts {
		out = append(out, domain.MortAExaminer{
			MatchID: m.MatchID, X: m.X, Y: m.Y,
			PlusProcheM: m.PlusProcheM, Visibles: m.Visibles, HorsDeVue: m.HorsDeVue,
		})
	}
	return out
}

// medianeRatio rend la mediane des abscisses PRESENTES (les morts hors de vue n'en ont
// pas). nil quand aucune mort du joueur n'avait de coequipier visible.
func medianeRatio(points []domain.SquadIsolementMort) *float64 {
	v := make([]float64, 0, len(points))
	for _, p := range points {
		if p.DistanceRatio != nil {
			v = append(v, *p.DistanceRatio)
		}
	}
	if len(v) == 0 {
		return nil
	}
	sort.Float64s(v)
	m := v[len(v)/2]
	if len(v)%2 == 0 {
		m = (v[len(v)/2-1] + v[len(v)/2]) / 2
	}
	return &m
}

// medianeDelai rend la mediane du delai des morts VENGEES — donc des seules ripostes DANS
// LA FENETRE (`p.Vengee` ne vaut plus vrai hors fenetre, cf. pointsDuJoueur) : la meme
// population que `delaiMedianDesEchanges` du bloc Riposte. Une mort `HorsFenetre` porte un
// delai mais n'entre pas ici ; l'inclure tirerait le repere vers une population que le taux
// ne compte pas. nil quand aucune mort n'est vengee : une mediane a zero placerait le
// repere sur l'axe, ce qui se lirait « riposte immediate ».
func medianeDelai(points []domain.SquadIsolementMort) *int64 {
	v := make([]int64, 0, len(points))
	for _, p := range points {
		if p.Vengee && p.DelaiMs != nil {
			v = append(v, *p.DelaiMs)
		}
	}
	if len(v) == 0 {
		return nil
	}
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	m := v[len(v)/2]
	if len(v)%2 == 0 {
		m = (v[len(v)/2-1] + v[len(v)/2]) / 2
	}
	return &m
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
