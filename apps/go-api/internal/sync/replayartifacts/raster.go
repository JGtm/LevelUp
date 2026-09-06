package replayartifacts

// raster.go — LE RASTER TACTIQUE D'UN MATCH, PROJETE AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Chaque artefact CUIT DANS CE CYCLE est projete en sidecar d'occupation
// (`data/cache/replays/{slug}/rasters/{short}.json`, cf.
// title.PathResolver.TacticalRasterPath) : par joueur nomme, le temps passe dans chaque
// cellule de 0,5 m, ses reapparitions, et sa premiere entree dans chaque cellule. C'est la
// matiere de la lecture « ou je passe mon temps » de l'onglet Tactique.
//
// # POURQUOI A LA CUISSON, ET UNE SEULE FOIS (plan tactique, decision du 2026-09-05)
//
// Le calcul part des PISTES de l'artefact : quelques centaines de milliers de points par
// match. Le refaire a chaque affichage rendrait la page inutilisable des la dizaine de
// matchs, et le mettre en cache PAR FILTRE donnerait autant d'entrees que de combinaisons
// de la barre L2 — chacune a invalider au match suivant. Un sidecar PAR MATCH est
// immuable : il ne depend d'aucun filtre, la page en somme N, et rien ne s'invalide.
//
// # LA FORME, CALQUEE SUR usage.go ET bombstats.go
//
//	SUR DISQUE, PAS LE BLOB   la projection lit l'artefact TEL QU'IL EST RANGE.
//	                          `StoreArtifact` peut REFUSER les octets candidats (garde
//	                          anti-regression) et conserver l'artefact precedent :
//	                          projeter le candidat ecrirait un raster que le disque ne
//	                          porte pas.
//	APRES TOUTE CUISSON       meme place que les trois autres projections, jamais entre
//	                          deux decodages.
//	BEST-EFFORT, JAMAIS MUET  l'echec d'un match est journalise et compte ; il n'arrete
//	                          ni le cycle ni les autres projections.
//
// # LA DIFFERENCE AVEC SES TROIS VOISINES : AUCUNE BASE
//
// Les trois autres projections ecrivent en base et prennent donc un writer partage. Ce
// sidecar-ci est un FICHIER a cote de son artefact : aucun writer, aucun lease, aucune
// regle ART. C'est aussi ce qui rend son rattrapage (`levelup tactical-rasters`) jouable
// sans toucher a la moindre base.
//
// # LE GATE : LA PORTE DE L'ETAPE ELLE-MEME
//
// `film.replay_artifact` gouverne cette projection — et elle est DEJA appliquee, en tete
// de `Run` (cf. capability.go) : sans la cle, l'etape ne cuit rien, donc `rapports` est
// vide et rien n'est projete. La relire ici ferait deux sources de verite pour une seule
// question, plus une lecture de TOML par cycle. Les gates jumeaux d'usage.go et de
// bombstats.go existent, eux, parce qu'ils portent des cles DIFFERENTES de celle de
// l'etape (`film.usage_summary`, `film.bomb_stats`).

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/atomicfile"

	"context"
)

// permSidecarRaster : les droits a la CREATION du sidecar. Meme regime que les autres
// derives de cache du depot — lisible, non executable.
const permSidecarRaster = 0o644

// ProjeterRasterTactique lit UN artefact range et en tire son sidecar d'occupation.
//
// ELLE NE LIT QUE CE QUI EXISTE DEPUIS TOUJOURS : `matchId`, `frameCount`/`tracks` et,
// dans chaque piste, le xuid, les points (x, y, t) et les bornes de vie. Aucun champ
// introduit par une montee de schema recente n'est requis — un artefact ancien (schema 20)
// se projette donc exactement comme un artefact courant, et c'est ce qui rend le
// rattrapage utile AVANT une re-cuisson du parc.
//
// `frameIntervalMs` absent (artefacts anciens) : l'echelle par defaut du decodeur
// (replay.DefaultFrameIntervalMS) est appliquee ICI, chez l'appelant du calcul pur —
// `analysis/tactical` ne devine aucune cadence.
//
// Exportee pour le rattrapage CLI (cmd/levelup/cmd_tactical_rasters.go) : le fil de l'eau
// et la passe hors ligne DOIVENT projeter a l'identique, et deux ecritures de la meme
// regle divergeraient au premier ajustement.
func ProjeterRasterTactique(path string) (domain.TacticalRasterSidecar, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return domain.TacticalRasterSidecar{}, fmt.Errorf("lecture artefact: %w", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return domain.TacticalRasterSidecar{}, fmt.Errorf("parse artefact: %w", err)
	}
	if doc.MatchID == "" {
		// Sans identifiant de match, le raster n'a pas de cle : le plancher de rarete se
		// compte en matchs DISTINCTS, et un raster anonyme ne pourrait jamais y entrer.
		return domain.TacticalRasterSidecar{}, fmt.Errorf("artefact sans matchId")
	}
	intervalle := doc.FrameIntervalMS
	if intervalle <= 0 {
		intervalle = replay.DefaultFrameIntervalMS
	}
	g := tactical.GrilleParDefaut()
	entree := tactical.EntreeOccupation{
		MatchID:           doc.MatchID,
		IntervalleFrameMs: intervalle,
		Pistes:            pistesDeLArtefact(doc.Tracks),
		Embarquements:     embarquementsDeLArtefact(doc.Vehicles),
	}
	joueurs, ignores, err := rasteriserParJoueur(g, entree)
	if err != nil {
		return domain.TacticalRasterSidecar{}, err
	}
	return domain.TacticalRasterSidecar{
		SchemaVersion:         domain.TacticalRasterSchemaVersion,
		MatchID:               doc.MatchID,
		ShortID:               titlePkg.FilmShortMatchID(doc.MatchID),
		ArtifactSchemaVersion: doc.SchemaVersion,
		PasM:                  g.PasM(),
		FrameIntervalMs:       intervalle,
		PasEchantillonMs:      tactical.PasOccupationMs,
		PointsIgnores:         ignores,
		Joueurs:               joueurs,
	}, nil
}

// pistesDeLArtefact projette les pistes du document vers les types PURS du rasterisage.
// C'est LA frontiere declaree par analysis/tactical/doc.go : ce paquet-la n'importe pas
// `analysis/replay`, c'est l'appelant qui traduit ce qu'il a.
func pistesDeLArtefact(tracks []replay.Track) []tactical.Piste {
	out := make([]tactical.Piste, 0, len(tracks))
	for _, t := range tracks {
		if t.XUID == "" {
			// Vie non nommee (le fil des morts ne l'a jamais citee, ou c'est un bot) :
			// `Occupation` l'ecarterait de toute facon, mais la porter jusque-la
			// copierait ses points pour rien.
			continue
		}
		points := make([]tactical.PointPiste, 0, len(t.Points))
		for _, p := range t.Points {
			points = append(points, tactical.PointPiste{T: p.T, X: float64(p.X), Y: float64(p.Y)})
		}
		out = append(out, tactical.Piste{
			XUID: t.XUID, Points: points, StartFrame: t.StartFrame, EndFrame: t.EndFrame,
		})
	}
	return out
}

// embarquementsDeLArtefact projette les EPISODES D'OCCUPATION de vehicule vers les types
// purs du rasterisage.
//
// # POURQUOI CE CALQUE EXISTE, ET CE QU'IL REPARE
//
// Un occupant embarque CESSE de repliquer son bipede : ce sont ses TROUS qui portent les
// episodes (`document.go`, calque `vehicles`). Or la cuisson COUPE une piste en nouvelle
// vie des qu'un trou depasse 5 s (`replay.lifeGapUS`), et ces episodes durent 13 a 36 s en
// mediane. Sans ce calque, le temps passe en vehicule ne serait mesure NULLE PART, alors
// que le match compterait comme mesure.
//
// # LE LIEN EST UNE IMBRICATION, PAS UNE CLE
//
// `VehicleRide` vit DANS `VehicleTrack.Rides`, a cote de `VehicleTrack.Samples` : la
// trajectoire a joindre est celle de la vie de vehicule qui porte l'episode. Aucun
// appariement a faire, donc aucun appariement a rater.
//
// # ON ATTRIBUE, ON N'INVENTE PAS
//
// Un episode sans occupant NOMME est ecarte (le vehicule est occupe, son occupant est
// inconnu — l'attribuer a quelqu'un serait une invention), et un vehicule sans echantillon
// n'attribue rien. La lacune residuelle est ECRITE dans domain/tactical_raster.go.
func embarquementsDeLArtefact(vehicules []replay.VehicleTrack) []tactical.Embarquement {
	out := make([]tactical.Embarquement, 0, len(vehicules))
	for _, v := range vehicules {
		if len(v.Rides) == 0 {
			continue
		}
		// La trajectoire est projetee UNE FOIS par vie de vehicule, puis partagee par ses
		// episodes : une vie de Warthog peut en porter trois (trois occupants successifs),
		// et recopier ses echantillons pour chacun triplerait la memoire pour rien. La
		// tranche est lue seule, jamais modifiee.
		points := pointsDuVehicule(v.Samples)
		for _, r := range v.Rides {
			if r.XUID == "" {
				continue
			}
			out = append(out, tactical.Embarquement{
				XUID: r.XUID, T0: r.T0, T1: r.T1, Points: points,
			})
		}
	}
	return out
}

// pointsDuVehicule projette les echantillons d'une vie de vehicule.
func pointsDuVehicule(samples []replay.VehicleSample) []tactical.PointPiste {
	out := make([]tactical.PointPiste, 0, len(samples))
	for _, s := range samples {
		out = append(out, tactical.PointPiste{T: s.T, X: float64(s.X), Y: float64(s.Y)})
	}
	return out
}

// rasteriserParJoueur reechantillonne puis compte, JOUEUR PAR JOUEUR.
//
// Le rasterisage passe par `tactical.Rasterise` sur l'univers d'UN SEUL match : la regle
// d'adressage des cellules (ancrage sur l'origine du monde) vit ainsi a un seul endroit,
// celui-la meme qui la publiera a la lecture. Les comptes sont lus par `CellulesBrutes` —
// SANS plancher de rarete : le plancher appartient a l'agregat, jamais au match (cf. sa
// doc).
func rasteriserParJoueur(g tactical.Grille, e tactical.EntreeOccupation) ([]domain.TacticalRasterJoueur, int, error) {
	occ := tactical.Occupation(g, e, tactical.PasOccupationMs)
	// VIDE MAIS PRESENT : un artefact sans piste nommee rend une liste vide, pas `null`.
	// Le fichier existe donc, et « mesure a zero » ne se confond pas avec « non mesure ».
	out := make([]domain.TacticalRasterJoueur, 0, len(occ))
	ignores := 0
	for _, j := range occ {
		raster, err := tactical.Rasterise(g, []string{e.MatchID}, j.Echantillons)
		if err != nil {
			return nil, 0, fmt.Errorf("rasterisage du joueur %s: %w", j.XUID, err)
		}
		// LES POSITIONS NON FINIES SE COMPTENT ICI, A LA CUISSON, et voyagent dans le
		// sidecar : c'est le seul endroit ou elles existent encore. Le raster d'agregat,
		// lui, part de comptes deja groupes par cellule — un point ecarte n'a jamais eu
		// de cellule, il ne peut donc pas s'y retrouver. Sans ce transport, la lecture
		// aurait publie 0 point ignore quoi qu'il arrive : un decodage qui derape se
		// serait tu.
		ignores += raster.PointsIgnores()
		out = append(out, domain.TacticalRasterJoueur{
			XUID:             j.XUID,
			Cellules:         cellulesDuRaster(raster),
			Spawns:           spawnsDeLOccupation(j.Spawns),
			PremieresEntrees: entreesDeLOccupation(j.PremieresEntrees),
		})
	}
	return out, ignores, nil
}

func cellulesDuRaster(r *tactical.Raster) []domain.TacticalRasterCellule {
	brutes := r.CellulesBrutes()
	out := make([]domain.TacticalRasterCellule, 0, len(brutes))
	for _, c := range brutes {
		out = append(out, domain.TacticalRasterCellule{Col: c.Col, Lig: c.Lig, Echantillons: c.Occurrences})
	}
	return out
}

func spawnsDeLOccupation(spawns []tactical.SpawnPiste) []domain.TacticalRasterSpawn {
	out := make([]domain.TacticalRasterSpawn, 0, len(spawns))
	for _, s := range spawns {
		out = append(out, domain.TacticalRasterSpawn{Frame: s.Frame, X: s.X, Y: s.Y})
	}
	return out
}

func entreesDeLOccupation(entrees []tactical.EntreeCellule) []domain.TacticalRasterEntree {
	out := make([]domain.TacticalRasterEntree, 0, len(entrees))
	for _, e := range entrees {
		out = append(out, domain.TacticalRasterEntree{Col: e.Col, Lig: e.Lig, Frame: e.Frame})
	}
	return out
}

// EcrireSidecarRaster serialise et depose le sidecar, ATOMIQUEMENT quand l'environnement
// le permet (platform/atomicfile). Le dossier est cree au besoin : le rattrapage tourne
// sur des postes ou aucun sidecar n'a jamais ete ecrit.
//
// Exportee pour le rattrapage CLI, meme raison que ProjeterRasterTactique : une seconde
// ecriture du meme fichier divergerait.
func EcrireSidecarRaster(path string, s domain.TacticalRasterSidecar) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("serialisation du sidecar: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creation du dossier des rasters: %w", err)
	}
	if err := atomicfile.WriteFile(path, raw, permSidecarRaster); err != nil {
		return fmt.Errorf("ecriture du sidecar: %w", err)
	}
	return nil
}

// projeterRastersTactiques projette et depose les sidecars des artefacts cuits du cycle.
//
// Best-effort de bout en bout, comme toute l'etape : aucun echec ne remonte au cycle, et
// aucun ne se tait. Un match en echec n'empeche ni les suivants ni le reste de la cuisson.
func projeterRastersTactiques(ctx context.Context, d Deps, rapports []artefactCuit) {
	if len(rapports) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	pr := titlePkg.NewPathResolver(d.RepoRoot)
	ecrits, echecs := 0, 0
	for _, r := range rapports {
		s, err := ProjeterRasterTactique(r.path)
		if err != nil {
			slog.ErrorContext(ctx, "post-sync: artefact range mais raster tactique impossible",
				"gamertag", d.Gamertag, "match_id", r.matchID, "err", err)
			echecs++
			continue
		}
		if err := EcrireSidecarRaster(pr.TacticalRasterPath(d.TitleSlug, r.matchID), s); err != nil {
			slog.ErrorContext(ctx, "post-sync: sidecar de raster tactique non ecrit",
				"gamertag", d.Gamertag, "match_id", r.matchID, "err", err)
			echecs++
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurRastersEcrits, int64(ecrits))
	observability.AddIntT(titre, CompteurRastersEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: rasters tactiques projetes",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}
