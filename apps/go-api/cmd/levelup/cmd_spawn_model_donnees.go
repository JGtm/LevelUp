package main

// cmd_spawn_model_donnees.go — LA LECTURE DES REAPPARITIONS DU PARC, un artefact a la fois.
//
// Chaque artefact est lu, projete vers ses evenements de reapparition (compacts : quelques
// positions par evenement), puis relache : rien ne survit d'un match a l'autre que les
// evenements. Le balayage du parc entier ne charge jamais les
// artefacts ensemble.
//
// Chaque reapparition porte DEUX instantanes des autres joueurs : a la reapparition, et a la
// mort qui la precede (ce que le rejeu pourrait afficher pendant le compte a rebours).

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/sync/replayartifacts"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// evenementDouble : une reapparition vue a deux instants.
type evenementDouble struct {
	famille    string
	reap, mort tactical.EvenementReapparition
}

// bilanLecture compte ce que la lecture a vu et ecarte, pour qu'aucun ecart ne soit muet.
type bilanLecture struct {
	artefacts, sansRegistre, horsFamille, lus, echecs int
	// reapparitions : celles que la lecture du film decide ; sansLectureFilm : documents qui n'en portent
	// aucune lecture, donc sans reapparition.
	reapparitions, sansLectureFilm int
	sansEquipe, nonFinis           int
	parFamille                     map[string]int
	// comptesSansJoueur : comptes a rebours qu'aucun xuid ne rattache (bots, index inconnu).
	comptesSansJoueur int
}

// jeuReapparitions : tout ce que les mesures lisent.
type jeuReapparitions struct {
	evts []evenementDouble
	// noms : identifiant de carte -> nom, tel que le registre le donne.
	noms  map[string]string
	bilan bilanLecture
}

// optionsLecture : ce que la lecture doit garder.
type optionsLecture struct {
	titleSlug  string
	familles   map[string]bool
	limite     int
	registre   map[string]infoMatch
	artefactsS []string
}

// lireLeParc parcourt les artefacts demandes et rend les evenements. Un artefact en echec est
// journalise et compte, jamais fatal.
func lireLeParc(ctx context.Context, cfg *config.AppConfig, o optionsLecture) *jeuReapparitions {
	jeu := &jeuReapparitions{noms: map[string]string{}}
	jeu.bilan.parFamille = map[string]int{}
	jeu.bilan.artefacts = len(o.artefactsS)
	pr := cfg.PathResolver()
	debut := time.Now()
	for i, short := range o.artefactsS {
		info, ok := o.registre[short]
		switch {
		case !ok:
			jeu.bilan.sansRegistre++
			continue
		case !o.familles[info.famille]:
			jeu.bilan.horsFamille++
			continue
		}
		if o.limite > 0 && jeu.bilan.lus >= o.limite {
			break
		}
		doc, err := replayartifacts.LireArtefactRange(pr.ReplayArtifactPath(o.titleSlug, short))
		if err != nil {
			slog.ErrorContext(ctx, "spawn-model : artefact illisible", "short_id", short, "err", err)
			jeu.bilan.echecs++
			continue
		}
		jeu.bilan.lus++
		jeu.bilan.parFamille[info.famille]++
		jeu.absorber(doc, info)
		if jeu.bilan.lus%200 == 0 {
			slog.InfoContext(ctx, "spawn-model : lecture du parc",
				"lus", jeu.bilan.lus, "sur", len(o.artefactsS), "rang", i, "evenements", len(jeu.evts),
				"duration", time.Since(debut).Round(time.Second))
		}
	}
	slog.InfoContext(ctx, "spawn-model : lecture terminee", "lus", jeu.bilan.lus, "evenements", len(jeu.evts),
		"duration", time.Since(debut).Round(time.Second))
	return jeu
}

// absorber projette un document vers ses evenements.
func (j *jeuReapparitions) absorber(doc *replay.ReplayDocument, info infoMatch) {
	if j.noms[info.carteID] == "" {
		j.noms[info.carteID] = info.carteNom
	}
	entree := replayartifacts.EntreeOccupationDeLArtefact(doc)
	j.bilan.comptesSansJoueur += entree.ComptesSansJoueur
	if !tactical.LectureDuFilm(entree) {
		j.bilan.sansLectureFilm++
	}
	j.evenementsDuMatch(entree, info)
}

// evenementsDuMatch ajoute les reapparitions d'un match.
func (j *jeuReapparitions) evenementsDuMatch(entree tactical.EntreeOccupation, info infoMatch) {
	ix := tactical.NouvelIndexVies(entree)
	spawns := ix.Spawns()
	for i, p := range entree.Pistes {
		s, ok := spawns[i]
		if !ok {
			continue
		}
		if !s.Reapparition || s.Mort == nil {
			continue
		}
		j.bilan.reapparitions++
		equipe := p.Equipe
		if equipe < 0 {
			equipe = ix.EquipeDuJoueur(p.XUID)
		}
		reel := tactical.Point3{X: s.X, Y: s.Y, Z: s.Z}
		mort := tactical.Point3{X: s.Mort.X, Y: s.Mort.Y, Z: s.Mort.Z}
		if equipe < 0 {
			j.bilan.sansEquipe++
			continue
		}
		if !reel.Fini() || !mort.Fini() {
			j.bilan.nonFinis++
			continue
		}
		j.evts = append(j.evts, evenementDouble{
			famille: info.famille,
			reap:    info.evenement(tactical.EvenementAuInstant(s.Instantane, equipe, mort), reel),
			mort:    info.evenement(tactical.EvenementAuInstant(ix.Vivants(p.XUID, s.Mort.Frame), equipe, mort), reel),
		})
	}
}

// evenement rattache a un instantane le match, la carte et le point reel de la reapparition.
func (info infoMatch) evenement(e tactical.EvenementReapparition, reel tactical.Point3) tactical.EvenementReapparition {
	e.MatchID, e.CarteID, e.Reel = info.matchID, info.carteID, reel
	return e
}

// evenementsDeFamilles rend, pour les familles demandees, les evenements a l'instant de la
// reapparition.
func (j *jeuReapparitions) evenementsDeFamilles(familles map[string]bool) []tactical.EvenementReapparition {
	var out []tactical.EvenementReapparition
	for _, e := range j.evts {
		if familles[e.famille] {
			out = append(out, e.reap)
		}
	}
	return out
}

// evenementsAvecLesDeux rend les evenements dont LES DEUX instantanes ont au moins un ennemi :
// la population commune des deux mesures, pour qu'elles se comparent.
func (j *jeuReapparitions) evenementsAvecLesDeux(familles map[string]bool) (reap, mort []tactical.EvenementReapparition) {
	for _, e := range j.evts {
		if familles[e.famille] && len(e.reap.Ennemis) > 0 && len(e.mort.Ennemis) > 0 {
			reap, mort = append(reap, e.reap), append(mort, e.mort)
		}
	}
	return reap, mort
}
