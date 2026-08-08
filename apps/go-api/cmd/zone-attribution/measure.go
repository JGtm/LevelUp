// measure.go — la mesure d'UN match, et le rapport de l'ensemble.
package main

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// runner porte le contexte commun a tous les matchs mesures.
type runner struct {
	slug     string
	cacheDir string
	db       *sql.DB
	bounds   *filmdec.MapQuantCatalog
	zones    *replay.MapObjectivesCatalog
	role     mapvar.Role
	offset   mapvar.Vec3
	maxGap   int
	rejects  rejects
	results  []result
}

// result porte de quoi rejouer le croisement a plusieurs seuils sans re-decoder le film.
type result struct {
	m eligible
	// identified = evenements d'objectif identifies par xuid, AVANT la pose sur la grille
	// de frames. Les actions posees sont celles que le rejeu a pu accrocher a une track
	// publiee ; leur ecart est un trou du rejeu, pas du croisement.
	identified int
	actions    []replay.ObjectiveAction
	nullAction []replay.ObjectiveAction
	tracks     []replay.Track
	zones      []replay.Zone
	frameCount int
	err        error
}

// zoneStats : les statistiques d'objectif du mode a zones. Les frags et assistances sont
// nommes par le meme decodeur mais ne sont PAS des actions de zone — les inclure diluerait
// le taux avec des evenements qui n'ont aucune raison de se produire dans une zone.
var zoneStats = map[string]bool{
	objectiveevents.StatZoneCaptures: true,
	objectiveevents.StatZoneSecures:  true,
}

// nullShiftFrames : decalage du TEMOIN TEMPOREL, en frames de 100 ms (30 s).
//
// Le temoin translate deplace les ZONES ; celui-ci deplace les INSTANTS. Il repond a une
// objection que le premier ne couvre pas : « les joueurs de Bastion passent leur partie
// autour des zones, donc n'importe quel instant marcherait ». Si c'etait vrai, decaler
// l'action de 30 s ne changerait pas le taux.
const nullShiftFrames = 300

// measure croise un match : film -> trajectoires + actions, puis prepare les temoins.
func (r *runner) measure(ctx context.Context, m eligible) result {
	res := result{m: m}
	src, ok, err := filmcache.Open(r.cacheDir, m.short)
	if err != nil {
		res.err = err
		return res
	}
	if !ok {
		res.err = fmt.Errorf("manifeste de film absent")
		return res
	}
	lines, err := loadPlayerLines(ctx, r.db, m.full)
	if err != nil {
		res.err = err
		return res
	}
	identified := identifyZoneActions(src, lines)
	res.identified = len(identified)

	doc, err := replay.BuildFromFilm(m.short, r.slug, filmcache.ChunkDir(r.cacheDir, m.short),
		replay.Options{WorldRange: m.world, Objectives: identified})
	if err != nil {
		res.err = err
		return res
	}
	res.actions, res.tracks, res.zones = doc.Objectives, doc.Tracks, m.zones.Zones
	res.frameCount = doc.FrameCount
	res.nullAction = shiftBy(doc.Objectives, doc.FrameCount, nullShiftFrames)
	return res
}

// shiftBy rend les MEMES actions posees `delta` frames plus loin, en enroulant sur la
// duree du match. Deterministe : aucun tirage aleatoire, la mesure se rejoue a
// l'identique.
func shiftBy(actions []replay.ObjectiveAction, frameCount, delta int) []replay.ObjectiveAction {
	if frameCount <= 0 {
		return nil
	}
	if delta == 0 {
		return actions
	}
	out := make([]replay.ObjectiveAction, len(actions))
	for i, a := range actions {
		// Le modulo de Go garde le signe du dividende : un decalage negatif donnerait un
		// index de frame negatif, donc « aucune position » pour toutes les actions du
		// debut de match — un faux effondrement du taux, lu comme une mesure.
		a.T = ((a.T+delta)%frameCount + frameCount) % frameCount
		out[i] = a
	}
	return out
}

// identifyZoneActions decode les evenements nommes du statborg, les traduit en xuid et ne
// garde que les actions de ZONE.
func identifyZoneActions(src objectiveevents.FilmSource,
	lines []objectiveevents.PlayerLine) []objectiveevents.IdentifiedEvent {
	named := objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeZone)
	identity := objectiveevents.SlotIdentity(src, lines)
	all := objectiveevents.IdentifyNamedEvents(named, identity)
	out := make([]objectiveevents.IdentifiedEvent, 0, len(all))
	for _, e := range all {
		if zoneStats[e.Stat] {
			out = append(out, e)
		}
	}
	return out
}

// printSelection dit ce qui est mesurable ET ce qui ne l'est pas, maillon par maillon.
// Un corpus annonce sans son denominateur ni ses motifs de rejet ne se lit pas.
func printSelection(all []candidate, elig []eligible, rej rejects) {
	modeOK := len(all) - rej.pasLeBonMode
	fmt.Printf("CORPUS — %d match(s) au registre, dont %d en mode a zones\n", len(all), modeOK)
	fmt.Printf("  ecartes : %d sans film en cache, %d sans bornes de carte, %d sans formes au catalogue\n",
		rej.sansFilm, rej.sansBornes, rej.sansFormes)
	fmt.Printf("  MESURABLES : %d\n", len(elig))
	for _, m := range elig {
		fmt.Printf("    %s  %-14s %d zone(s)\n", m.short, m.mapName, len(m.zones.Zones))
	}
	fmt.Println()
}
