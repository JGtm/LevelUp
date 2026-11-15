package main

// cmd_spawn_model_donnees.go — LA LECTURE DES REAPPARITIONS DU PARC, un artefact a la fois.
//
// Chaque artefact est lu, projete vers ses evenements de reapparition (compacts : quelques
// positions par evenement), puis relache : rien ne survit d'un match a l'autre que les
// evenements et la geometrie de chaque carte. Le balayage du parc entier ne charge jamais les
// artefacts ensemble.
//
// Chaque reapparition porte DEUX instantanes des autres joueurs : a la reapparition, et a la
// mort qui la precede (ce que le rejeu pourrait afficher pendant le compte a rebours).

import (
	"context"
	"log/slog"
	"math"
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
	reapFilm, reapRepli, debutsManche                 int
	sansEquipe, nonFinis                              int
	parFamille                                        map[string]int
}

// jeuReapparitions : tout ce que les mesures lisent.
type jeuReapparitions struct {
	evts      []evenementDouble
	obstacles map[string]*tactical.Obstacles
	// noms : identifiant de carte -> nom, tel que le registre le donne.
	noms  map[string]string
	bilan bilanLecture
}

// optionsLecture : ce que la lecture doit garder.
type optionsLecture struct {
	titleSlug  string
	familles   map[string]bool
	obstacles  bool
	limite     int
	registre   map[string]infoMatch
	artefactsS []string
}

// lireLeParc parcourt les artefacts demandes et rend les evenements. Un artefact en echec est
// journalise et compte, jamais fatal.
func lireLeParc(ctx context.Context, cfg *config.AppConfig, o optionsLecture) *jeuReapparitions {
	jeu := &jeuReapparitions{obstacles: map[string]*tactical.Obstacles{}, noms: map[string]string{}}
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
		jeu.absorber(doc, info, o.obstacles)
		if jeu.bilan.lus%200 == 0 {
			slog.InfoContext(ctx, "spawn-model : lecture du parc",
				"lus", jeu.bilan.lus, "sur", len(o.artefactsS), "rang", i, "evenements", len(jeu.evts),
				"duration", time.Since(debut).Round(time.Second))
		}
	}
	boites := 0
	for _, ob := range jeu.obstacles {
		boites += ob.NbBoites()
	}
	slog.InfoContext(ctx, "spawn-model : lecture terminee", "lus", jeu.bilan.lus, "evenements", len(jeu.evts),
		"cartes_avec_geometrie", len(jeu.obstacles), "boites", boites, "duration", time.Since(debut).Round(time.Second))
	return jeu
}

// absorber projette un document vers ses evenements et, si on le demande, retient la
// geometrie de sa carte la premiere fois qu'elle est vue.
func (j *jeuReapparitions) absorber(doc *replay.ReplayDocument, info infoMatch, avecObstacles bool) {
	if j.noms[info.carteID] == "" {
		j.noms[info.carteID] = info.carteNom
	}
	if avecObstacles && j.obstacles[info.carteID] == nil && len(doc.Structure) > 0 {
		j.obstacles[info.carteID] = obstaclesDuDocument(doc)
	}
	entree := replayartifacts.EntreeOccupationDeLArtefact(doc)
	j.evenementsDuMatch(entree, info)
}

// obstaclesDuDocument indexe la structure d'un document.
func obstaclesDuDocument(doc *replay.ReplayDocument) *tactical.Obstacles {
	boites := make([]tactical.Boite, 0, len(doc.Structure))
	for _, s := range doc.Structure {
		boites = append(boites, tactical.Boite{
			X0: float64(s.X0), Y0: float64(s.Y0), X1: float64(s.X1), Y1: float64(s.Y1),
			ZHaut: float64(s.Z), ZBas: float64(s.ZB),
		})
	}
	return tactical.NouveauxObstacles(boites)
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
		if s.DebutDeManche {
			j.bilan.debutsManche++
		}
		if !s.Reapparition || s.Mort == nil {
			continue
		}
		if s.Source == tactical.SourceFilm {
			j.bilan.reapFilm++
		} else {
			j.bilan.reapRepli++
		}
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
		if !fini(reel) || !fini(mort) {
			j.bilan.nonFinis++
			continue
		}
		j.evts = append(j.evts, evenementDouble{
			famille: info.famille,
			reap:    evenementDe(info, reel, mort, equipe, s.Instantane),
			mort:    evenementDe(info, reel, mort, equipe, ix.Vivants(p.XUID, s.Mort.Frame)),
		})
	}
}

func fini(p tactical.Point3) bool {
	for _, v := range []float64{p.X, p.Y, p.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// evenementDe separe un instantane en equipiers et adversaires du point de vue de `equipe` ; un
// joueur dont l'equipe est inconnue n'est compte dans aucun des deux.
func evenementDe(info infoMatch, reel, mort tactical.Point3, equipe int, vivants []tactical.Vivant) tactical.EvenementReapparition {
	e := tactical.EvenementReapparition{MatchID: info.matchID, CarteID: info.carteID, Reel: reel, Mort: mort}
	for _, v := range vivants {
		p := tactical.Point3{X: v.X, Y: v.Y, Z: v.Z}
		if !fini(p) || v.Equipe < 0 {
			// Une equipe inconnue n'est ni coequipier ni adversaire.
			continue
		}
		if v.Equipe == equipe {
			e.Equipiers = append(e.Equipiers, p)
		} else {
			e.Ennemis = append(e.Ennemis, p)
		}
	}
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
