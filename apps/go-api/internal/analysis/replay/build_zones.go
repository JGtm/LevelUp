package replay

// build_zones.go — LE CABLAGE du calque de L'ETAT DES ZONES : ce que l'appelant fournit, ce que
// `BuildFromFilm` decode, et ce que `BuildFromPositions` en assemble.
//
// Il vit a part de `build.go` pour la meme raison que `build_score.go` et
// `build_objectives_live.go` : l'assemblage garde UNE ligne par calque, le detail vit a cote de
// la donnee qu'il produit. `build.go` depasse deja 500 lignes, et la regle du depot est de ne pas
// accroitre cette dette.
//
// # DEUX LECTURES, ET UNE SEULE EST FAITE ICI
//
//	les PROPRIETES `ti=13`   ICI, dans `BuildFromFilm`, comme les autres balayages filmdec
//	le CATALOGUE de zones    l'appelant (catalogue de carte du titre, joint par map_id)
//	le ROSTER                l'appelant (les camps viennent de la base, jamais du film)
//	les CAPTURES nommees     deja assemblees dans `doc.Objectives` — un seul decodage du statborg
//
// # LE BALAYAGE NE SE PAIE QUE LA OU IL SERT
//
// `ti=13` n'est balaye que si l'appelant a fourni un catalogue de zones. Sans lui, aucun
// intervalle ne serait publiable (la carte slot -> zone n'aurait pas de cible), et le balayage
// est une marche BIT A BIT de tous les paquets delta du film — le meme ordre de grandeur que le
// balayage des positions. Le payer sur un Slayer serait payer pour rien : c'est la regle deja
// tenue par le marqueur de portage du drapeau, qui ne balaye que les films de CTF.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// decodeFilmZoneReads balaye les proprietes reseau de `ti=13` et JOURNALISE ce qu'il en est.
//
// TOUT ECHEC EST NON FATAL : un film dont l'archetype n'est pas au registre, ou dont aucun slot
// n'apparait aux images-cles, reste un rejeu parfaitement valide — simplement sans etat de zone.
// Le refus est journalise, jamais avale.
func decodeFilmZoneReads(filmDir, matchID string, zones int) []filmdec.ManagedPropertyRead {
	if zones == 0 {
		slog.Debug("rejeu : aucune zone au catalogue — proprietes ti=13 non balayees",
			"match_id", matchID, "filmDir", filmDir)
		return nil
	}
	sc, err := filmdec.ScanFilmManagedProperties(filmDir)
	if err != nil {
		slog.Info("rejeu : proprietes ti=13 illisibles — rejeu sans etat de zone",
			"err", err, "match_id", matchID, "filmDir", filmDir)
		return nil
	}
	slog.Info("rejeu : proprietes ti=13 balayees",
		"match_id", matchID, "slots", sc.Slots, "records", sc.Records, "marches", sc.Walked,
		"cassees", sc.Broken, "chainees", sc.Chained, "lectures", len(sc.Reads))
	return sc.Reads
}

// attachZoneStates pose l'etat des zones sur le document, avec sa couverture et son journal.
//
// LES CAPTURES VIENNENT DE `doc.Objectives`, PAS D'UN SECOND DECODAGE : elles y sont deja posees
// sur la grille de frames (origine du film retranchee) et filtrees aux joueurs dont une piste est
// publiee. Les re-decoder ici en ferait un second lecteur du meme fait.
func attachZoneStates(doc *ReplayDocument, opt Options, c replayClock) {
	states, cov := buildZoneStates(opt.Zone, zoneCtx{
		origin: c.origin, step: c.step, frames: doc.FrameCount,
		intervalMS: doc.FrameIntervalMS, tracks: doc.Tracks,
		actions: doc.Objectives, matchID: doc.MatchID,
	})
	doc.ZoneStates = states
	if doc.Coverage != nil {
		doc.Coverage.Zones = cov
	}
	logZoneStatesCoverage(doc.MatchID, cov)
}

// logZoneStatesCoverage journalise ce que le calque publie — et ce qu'il a ecarte.
//
// DEUX PHRASES, PARCE QUE DEUX SITUATIONS APPELLENT DEUX REPONSES : des slots de jauge qu'aucune
// capture ne rattache sont un trou d'appariement (le calque est partiel) ; une valeur de canal
// qui n'est pas un camp connu est une SURPRISE — le corpus n'en connait que trois — et elle doit
// se voir arriver.
func logZoneStatesCoverage(matchID string, cov *ZonesCoverage) {
	if cov == nil {
		return
	}
	if cov.Unpaired > 0 {
		slog.Warn("rejeu : slots de jauge NON apparies — zones absentes de l'etat publie",
			"match_id", matchID, "nonApparies", cov.Unpaired, "apparies", cov.Paired,
			"captures", cov.Captures, "attribuees", cov.Attributed)
	}
	if cov.UnknownOwner > 0 {
		slog.Warn("rejeu : valeurs de proprietaire INCONNUES — intervalles non ouverts",
			"match_id", matchID, "valeurs", cov.UnknownOwner)
	}
	slog.Info("rejeu : etat des zones",
		"match_id", matchID, "methode", cov.Method, "catalogue", cov.Catalog,
		"slots", cov.Slots, "apparies", cov.Paired, "intervalles", cov.Spans,
		"periodesColline", cov.HillPeriods, "proprietaireVerifie", cov.OwnerChecked,
		"proprietaireConcordant", cov.OwnerAgreed)
}
