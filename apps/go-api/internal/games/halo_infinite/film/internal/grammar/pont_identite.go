package grammar

// pont_identite.go — L ETAGE UNIQUE DES LECTURES DU PONT D IDENTITE (lot J4.3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-3 = S1, constat RA1-3).
//
// # CE QU IL LIT, ET DANS QUEL ORDRE
//
// Les SIX lectures du film dont le registre d identite (`replay.BuildIdentityRegistry`) est fait :
// les teleportations du translocateur, les positions bipedes AVEC les exemptions que ces
// teleportations ouvrent au filtre de vitesse (decision D2 du PLAN_LECTURE_FIABLE_EQUIPEMENT), les
// creations de bipede (le lien direct corps -> joueur), le fil des morts, la table d index de
// joueur et l origine d horloge du film. L ordre est celui des dependances : les exemptions avant
// les positions, les CREATIONS avant les positions (elles designent les generations vivantes du
// handle, lot J5.2), les morts avant l index (le roster de l index en derive).
//
// # POURQUOI UN SEUL ETAGE
//
// Deux producteurs lisent ce pont : la cuisson du rejeu (`replay`, `balayerPositions` /
// `balayerPont`) et le collecteur killsource (`sync/killcollector`, `buildPositionRows`). Jusqu a
// ce lot, le collecteur RECOPIAIT la sequence, et la recopie avait diverge : ses positions etaient
// lues sans les exemptions de translocation. Il n y a plus qu une sequence, ici ; le ratchet
// `archlint/film_pont_identite_test.go` interdit aux deux appelants d appeler l une des six
// lectures eux-memes.
//
// # CE QUI N EST PAS ICI, ET RESTE CHEZ CHAQUE APPELANT
//
// Les POLITIQUES, qui divergent et le doivent : le roster donne a la table d index (la cuisson
// part du fil des morts, le collecteur de la feuille de match — [OptionsDuPont.RosterDesMorts]),
// la capture des directions ([ScanFilmOptions.CaptureDirs], dans la base que l appelant fournit),
// l injectivite exigee de la table, et la FATALITE des erreurs : l etage ne decide d aucune, il
// rend chaque lecture avec son erreur. Il ne journalise ni ne compte rien (ADR 0034 D-4).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// OptionsDuPont : ce que l appelant fournit a l etage.
type OptionsDuPont struct {
	// Balayage est la BASE du balayage des positions (bornes du monde, decoupage d i0, capture des
	// directions). L etage y pose les exemptions de translocation ; il n y touche pas autrement.
	Balayage ScanFilmOptions
	// Carte est l entree de catalogue du match : la charge des teleportations est quantifiee a ses
	// bornes. Nil : les instants et les slots se lisent, les positions de la charge non.
	Carte *profile.MapQuantEntry
	// RosterDesMorts construit le roster de la table d index a partir du fil des morts lu. Il
	// n est appele que si le fil a rendu au moins une mort ; nil : la table n est pas lue.
	RosterDesMorts func([]types.Death) []uint64
}

// LecturesDuPont : les six lectures, chacune avec son erreur. Aucune n est fatale ICI.
type LecturesDuPont struct {
	Translocations []types.TranslocatorTeleport

	Positions    []BipedPosition
	ErrPositions error

	Creations      []BipedCreation
	StatsCreations types.BipedCreationStats
	ErrCreations   error
	// Generations : les generations vivantes du handle bipede sous lesquelles les positions ont ete
	// lues (lot J5.2). La cuisson en tire le compte du repli nomme ([GenerationsVivantes.SlotsEnRepli]).
	Generations *GenerationsVivantes

	Morts    []types.Death
	ErrMorts error

	// IndexLu dit si la table d index a ete lue : seulement quand le fil des morts a rendu au
	// moins une mort et que l appelant a fourni son roster.
	IndexLu  bool
	Index    types.PlayerIndexTable
	ErrIndex error

	OrigineUS  uint64
	ErrOrigine error
}

// ScanPontDIdentite lit le pont d identite d un film DEJA CHARGE, dans le contexte de l appelant.
func ScanPontDIdentite(fc *FilmContext, opt OptionsDuPont) LecturesDuPont {
	return etageDuPont{teleportations: ScanTranslocatorTeleports, positions: ScanBipedPositions}.lire(fc, opt)
}

// etageDuPont porte la lecture des teleportations et celle des positions comme des valeurs : c est
// la couture par laquelle un test verifie QUELLES options le balayage des positions recoit quand
// le film porte des teleportations — aucune mini-bobine du depot n en porte.
type etageDuPont struct {
	teleportations func(*source.Film, *profile.MapQuantEntry) []types.TranslocatorTeleport
	positions      func(*FilmContext, ScanFilmOptions) ([]BipedPosition, error)
}

// lire est le corps de [ScanPontDIdentite].
func (e etageDuPont) lire(fc *FilmContext, opt OptionsDuPont) LecturesDuPont {
	film := fc.Film()
	var l LecturesDuPont
	l.Translocations = e.teleportations(film, opt.Carte)
	// LES CREATIONS AVANT LES POSITIONS (lot J5.2, DT-8) : elles designent les generations VIVANTES
	// du handle, que le balayage des positions lit ensuite par le contexte (memorisees : un seul
	// balayage des creations par film).
	l.Creations, l.StatsCreations, l.ErrCreations = fc.CreationsDeBipede()
	if l.Generations = opt.Balayage.Generations; l.Generations == nil {
		l.Generations = fc.GenerationsVivantes()
	}
	l.Positions, l.ErrPositions = e.positions(fc, optionsDesPositionsDuPont(opt.Balayage, l.Translocations))
	l.Morts, l.ErrMorts = ScanDeaths(film)
	if l.ErrMorts == nil && len(l.Morts) > 0 && opt.RosterDesMorts != nil {
		l.IndexLu = true
		l.Index, l.ErrIndex = ScanPlayerIndices(film, opt.RosterDesMorts(l.Morts))
	}
	l.OrigineUS, l.ErrOrigine = ScanClockOrigin(film)
	return l
}

// optionsDesPositionsDuPont rend la base de l appelant augmentee des exemptions que les
// teleportations lues ouvrent au filtre de vitesse. Sans teleportation, les exemptions sont nil et
// le filtre est bit a bit celui d avant la decision D2.
func optionsDesPositionsDuPont(base ScanFilmOptions, tp []types.TranslocatorTeleport) ScanFilmOptions {
	base.TeleportExemptions = TeleportExemptionsOf(tp)
	return base
}
