package replay

// identity_registry_health.go — LA SANTE DU PONT, TELLE QUE LA COUVERTURE LA PUBLIE.
//
// Ce fichier n'accede JAMAIS aux tables brutes : il ne passe que par les accesseurs du registre
// (garde-rail `archlint`, une seule entree d'allowlist). C'est ce qui permet de le lire sans
// avoir a verifier qu'il ne contourne pas une garde — il n'en a pas les moyens.

import (
	"context"
	"log/slog"
	"slices"
	"strconv"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/filmobs"
)

// SanteDuPont rend la sante du pont slot -> joueur : sur quoi il repose, et ce qu'il refuse.
func (r IdentityRegistry) SanteDuPont() BridgeHealth {
	return BridgeHealth{
		Slots:               len(r.IndexParSlot()),
		FromReading:         r.SlotsParLaLecture(),
		LivesNamed:          r.ViesNommeesParLaLecture(),
		LivesTotal:          r.ViesTotal(),
		IndexReadings:       r.LecturesIndex(),
		IndexDisagreements:  r.DesaccordsIndex(),
		SlotCollisions:      r.CollisionsDeSlot(),
		DeathOffsetMatched:  r.DeathOffsetMatches(),
		DeathOffsetRunnerUp: r.CalageSecond(),
		DeathOffsetMs:       r.CalageSiConnu(),
		ClosedByShot:        r.FermeturesParTir(),
		ClosedByRespawn:     r.FermeturesParReapparition(),
		ClosedContested:     r.FermeturesContestees(),
		ClosedRefused:       r.FermeturesRefusees(),

		Concordant:                 r.PontConcordant(),
		Discordant:                 r.PontDiscordant(),
		BridgeNamedLives:           r.ViesNommeesParLePont(),
		DirectByCreation:           r.ViesParCreation(),
		DirectByCreationPropagated: r.ViesParCreationPropagee(),
		BodiesWithCreation:         r.CorpsAvecCreation(),
	}
}

// buildCoverage assemble la couverture publiee et ses verdicts. `originResolved` dit si l'origine
// de la frame 0 a ete etablie — elle conditionne la justesse de l'axe de temps de TOUS les
// calques dates depuis l'horloge du film (cf. Coverage.OriginResolved).
func buildCoverage(ctx context.Context, shots, grenades, objectives LayerCoverage, reg IdentityRegistry,
	originResolved bool, score *ScoreCoverage) *Coverage {
	b := reg.SanteDuPont()
	b.warnIfCalageEtroit(ctx)
	return &Coverage{
		Shots: shots, Grenades: grenades, Objectives: objectives, Bridge: b,
		OriginResolved: originResolved, Score: score,
		Verdict: map[string]string{
			"shots":      verdictOf(shots),
			"grenades":   verdictOf(grenades),
			"objectives": verdictOf(objectives),
			"bridge":     verdictOfBridge(b),
		},
	}
}

// logRegistry ALARME sur ce que le registre n'a PAS su nommer. Un lien non resolu se publie et se
// compte (doctrine §0.2 du plan v2) ; il ne se tait jamais.
func (r IdentityRegistry) logRegistry(ctx context.Context, matchID string) {
	total := r.Section.Coverage.Total()
	slog.InfoContext(ctx, "rejeu : registre d'identite",
		"match_id", matchID,
		"slots", len(r.IndexParSlot()), "viesNommees", r.ViesNommeesParLaLecture(),
		"viesTotal", r.ViesTotal(), "lecturesIndex", r.LecturesIndex(),
		"desaccordsIndex", r.DesaccordsIndex(), "collisionsSlot", r.CollisionsDeSlot(),
		"parCreation", r.ViesParCreation(), "parCreationPropagee", r.ViesParCreationPropagee(),
		"corpsAvecCreation", r.CorpsAvecCreation(),
		"pontConcordant", r.PontConcordant(), "pontDiscordant", r.PontDiscordant(),
		"viesNommeesParLePont", r.ViesNommeesParLePont(),
		"parElimination", r.eliminated,
		"parExclusionTemporelle", r.excluded,
		"viesSansAucunCandidat", r.excludedContradictions,
		"parTableauAPI", r.ViesNommeesParLeTableau(),
		"conflitsAuTableau", r.ViesConflitAuTableau(),
		"sansCandidatAuTableau", r.ViesSansCandidatAuTableau(),
		"liensDirects", total.Direct, "liensDeduits", total.Inferred,
		"liensNonResolus", total.Unresolved)
	r.creation.alarmerSurLesRefus(ctx, matchID, r.Section.Coverage.BipedSlot.UnresolvedByCause,
		r.indexDesViesHorsTable())
	r.publierDiscordancesDuPont(ctx, matchID)
	if r.ViesNommeesParLePont() > 0 {
		slog.WarnContext(ctx, "rejeu : AUCUNE lecture directe corps -> joueur — le pont par morts a nomme en "+
			"degradation complete (cf. identity_registry_bridge.go)",
			"match_id", matchID, "vies", r.ViesNommeesParLePont())
	}
	if r.DesaccordsIndex() > 0 {
		slog.WarnContext(ctx, "rejeu : desaccord de lecture de l'index de joueur — liens directs NON publies",
			"match_id", matchID, "desaccords", r.DesaccordsIndex())
	}
	if total.Unresolved > 0 {
		slog.WarnContext(ctx, "rejeu : liens d'identite NON RESOLUS — publies et comptes, jamais inventes",
			"match_id", matchID, "liens", total.Unresolved)
	}
}

// compteurDiscordancesDuPont : expvar (ADR 0009) des vies ou le pont par morts contredit la
// lecture directe, cumule sur les films traites par le processus.
const compteurDiscordancesDuPont = "replay_pont_discordances"

// publierDiscordancesDuPont publie, UNE fois par film, les vies ou le pont par morts contredit
// la lecture directe : le compteur et une trace agregee (compte, concordances, slots). Niveau
// Info, pas Warn : le film fait foi et le nom du pont est ecarte par construction — c'est un
// fait mesure, pas une anomalie a traiter. Le detail par vie est au niveau Debug
// (verifierParLesMorts).
func (r IdentityRegistry) publierDiscordancesDuPont(ctx context.Context, matchID string) {
	n := r.PontDiscordant()
	if n == 0 {
		return
	}
	filmobs.AddInt(compteurDiscordancesDuPont, int64(n))
	slog.InfoContext(ctx, "rejeu : le pont par morts contredit le lien direct sur des vies — le film "+
		"fait foi, les noms du pont sont ecartes",
		"match_id", matchID, "discordances", n,
		"concordances", r.PontConcordant(), "slots", r.PontSlotsDiscordants())
}

// indexDesViesHorsTable rend, pour le journal, chaque index de participant LU sur une vie restee
// sans nom faute de table (`index_hors_table`), et le nombre de ces vies : « index:vies », par
// index croissant. C'est la piece que l'alarme doit porter pour qu'on ouvre l'enquete.
func (r IdentityRegistry) indexDesViesHorsTable() []string {
	par := map[uint32]int{}
	for i, l := range r.Vies() {
		if l.xuid != 0 || l.bid != "" || r.creation.causes[i] != canonical.MethodIndexOutOfTable {
			continue
		}
		if pi, lu := r.creation.indexLu[i]; lu {
			par[pi]++
		}
	}
	index := make([]uint32, 0, len(par))
	for pi := range par {
		index = append(index, pi)
	}
	slices.Sort(index)
	out := make([]string, 0, len(index))
	for _, pi := range index {
		out = append(out, strconv.FormatUint(uint64(pi), 10)+":"+strconv.Itoa(par[pi]))
	}
	return out
}
