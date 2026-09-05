// measure.go — la mesure d'UN match, et le rapport de l'ensemble.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
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
	// corrected / correctedNull : les MEMES evenements reposes sur l'axe du rejeu apres
	// retrait de l'origine publiee par l'artefact (cf. correctedActions). Vides quand le
	// film ne publie pas d'origine — on ne devine jamais une correction.
	corrected     []replay.ObjectiveAction
	correctedNull []replay.ObjectiveAction
	originMS      int64
	hasOrigin     bool
	tracks        []replay.Track
	zones         []replay.Zone
	roster        []replay.RosterEntry
	frameCount    int
	err           error
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
	lines, err := loadPlayerLines(ctx, r.db, m.full)
	if err != nil {
		res.err = err
		return res
	}
	// LE FILM EST CHARGE UNE FOIS pour les trois consommateurs de cette mesure (les
	// enregistrements d'entite, le pont d'identite par instants de mort, et le decodage
	// complet) : jamais un `*Film` d'un cote et une enveloppe `dir` de l'autre — ce serait
	// deux decompressions du meme film.
	film, ok, err := filmcache.LoadFilm(r.cacheDir, m.short)
	if err != nil {
		res.err = err
		return res
	}
	if !ok {
		res.err = fmt.Errorf("manifeste de film absent")
		return res
	}
	identified := identifyZoneActions(lines, film)
	res.identified = len(identified)

	doc, err := replay.BuildFromFilm(m.short, r.slug, film,
		replay.Options{MapQuant: m.quant, Objectives: identified})
	if err != nil {
		res.err = err
		return res
	}
	res.actions, res.tracks, res.zones = doc.Objectives, doc.Tracks, m.zones.Zones
	res.roster = doc.Roster
	res.frameCount = doc.FrameCount
	res.nullAction = shiftBy(doc.Objectives, doc.FrameCount, nullShiftFrames)
	if doc.OriginMs != nil {
		res.originMS, res.hasOrigin = *doc.OriginMs, true
		res.corrected = correctedActions(identified, *doc.OriginMs,
			doc.FrameIntervalMS, doc.FrameCount, doc.Tracks)
		res.correctedNull = shiftBy(res.corrected, doc.FrameCount, nullShiftFrames)
	}
	return res
}

// correctedActions repose les evenements identifies sur l'axe de temps du rejeu en
// retranchant l'ORIGINE PUBLIEE PAR L'ARTEFACT.
//
// # Pourquoi cette fonction existe
//
// Les deux entrees du croisement ne vivent pas sur la meme horloge, et le decalage est
// LU, pas estime :
//
//	les actions   TimeMS = ms depuis le PREMIER PAQUET DU FILM (horloge du manifeste,
//	              `objectiveevents.StatRecords` : `meta.StartMS + (f.us - base)/1000`) ;
//	les positions Point.T = frame depuis le PREMIER PAQUET DE POSITION
//	              (`build.go` : `origin = sorted[0].TimestampUS`).
//
// L'ecart entre ces deux zeros est exactement ce que l'artefact publie sous `originMs`
// (origin.go), mesure de 3,6 s a 50,8 s selon le match. `buildObjectiveActions` divise
// TimeMS par le pas de grille SANS le retrancher : les actions sont donc posees
// `originMs` TROP TARD sur l'axe du rejeu. C'est ce que le balayage d'horloge du lot 4
// voyait comme un « retard » de signe negatif, et pourquoi trois films sur huit piquaient
// sur la borne de -10 s : leur origine la depassait.
//
// La reconstruction repart des evenements IDENTIFIES plutot que des actions deja posees :
// celles-ci ont perdu, dans `buildObjectiveActions`, tout ce qui tombait au-dela de la
// derniere frame — c'est-a-dire precisement les actions de fin de match que la correction
// ramene dans la fenetre. Post-decaler la sortie les laisserait dehors.
//
// Le filtre par track publiee reproduit `dropUnpublishedActions` : sans lui, la comparaison
// AVANT/APRES ne porterait pas sur le meme denominateur.
func correctedActions(evs []objectiveevents.IdentifiedEvent, originMS int64,
	intervalMS, frameCount int, tracks []replay.Track) []replay.ObjectiveAction {
	if intervalMS <= 0 || frameCount <= 0 {
		return nil
	}
	published := map[string]bool{}
	for _, t := range tracks {
		if t.XUID != "" {
			published[t.XUID] = true
		}
	}
	out := make([]replay.ObjectiveAction, 0, len(evs))
	for _, e := range evs {
		if e.XUID == "" || !published[e.XUID] {
			continue
		}
		// Une action ANTERIEURE au premier paquet de position n'a pas de frame : la division
		// entiere de Go tronquant vers zero, la tester avant division evite de la poser sur
		// la frame 0 comme si elle y avait eu lieu.
		rel := int64(e.TimeMS) - originMS
		if rel < 0 {
			continue
		}
		t := int(rel / int64(intervalMS))
		if t >= frameCount {
			continue
		}
		out = append(out, replay.ObjectiveAction{T: t, XUID: e.XUID, Stat: e.Stat, TimeMS: e.TimeMS})
	}
	return out
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
//
// LE PONT EST LE PONT RESOLU DEPUIS LE 2026-08-18, ET C'EST UN CORRECTIF MESURE. L'appariement
// par TOTAUX compare les compteurs du film aux lignes de match ; un film que le Theater rend
// TRONQUE ne les atteint jamais et l'appariement rend alors ZERO slot sur huit (mesure :
// `64e8adfa` et `24dbb67d`, plan objectifs vivants phase 0). Le repli par INSTANTS DE MORT
// n'emprunte rien a la base, tient sur un film tronque, et ne se declenche que s'il nomme
// STRICTEMENT plus de slots — un film complet rend donc exactement ce qu'il rendait avant.
func identifyZoneActions(lines []objectiveevents.PlayerLine,
	film *filmsource.Film) []objectiveevents.IdentifiedEvent {
	named := objectiveevents.NamedEvents(film, objectiveevents.ObjectiveTypeZone)
	identity, st := objectiveevents.SlotIdentityResolved(film, lines, deathInstantsOf(film))
	if st.Source != objectiveevents.IdentitySourceTotals || st.Conflicts > 0 {
		fmt.Printf("    pont d'identite : voie %q (%d par totaux, %d par instants de mort, "+
			"%d desaccords ecartes)\n", st.Source, st.ByTotals, st.ByDeaths, st.Conflicts)
	}
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

// deathInstantsOf lit le fil des morts du film et le met dans la forme qu'attend le pont
// d'identite. Un fil illisible rend une liste vide : le pont retombe alors sur les seuls
// totaux, exactement comme avant ce correctif — une degradation, jamais une erreur.
func deathInstantsOf(film *filmsource.Film) []objectiveevents.DeathInstant {
	deaths, err := replay.ScanDeaths(film)
	if err != nil {
		fmt.Printf("    fil des morts illisible (%v) — pont d'identite par totaux seuls\n", err)
		return nil
	}
	out := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS),
		})
	}
	return out
}
