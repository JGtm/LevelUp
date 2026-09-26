package replay

// gardes_de_cuisson.go — LES GARDES DE L APPELANT SOUS LESQUELLES DES FAITS ONT ETE CUITS (lot J3.4
// du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (d), constat RA1-1).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Trois canaux du balayage ne se lisent QUE si l appelant le commande — le drapeau (CTF), l etat des
// zones (catalogue de zones du mode), l anneau d armement (famille bomb) — et la table d index de
// joueur se lit sur le ROSTER de l appelant. Les faits persistes figeaient donc des lectures
// COMMANDEES, mais la fraicheur ne comparait que le codec, les revisions et la cle de cuisson. Une
// cuisson sans faits de base (ouvrier, `replay-build` sans `--facts`, base indisponible) ecrivait
// des faits sans zones, sans armement et sur un roster vide ; la reparation suivante les
// rejouait : KOTH sans `zoneStates`, `coverage.bombArmings` « balaye, zero lecture » — faux, et
// fige jusqu a la prochaine montee de revision.
//
// # LA REGLE
//
//	derivation  [GardesDe] tire les gardes d `Options` EN UN POINT ; le balayage emploie les MEMES
//	            predicats pour decider ce qu il lit ([drapeauBalayable], [zonesBalayables],
//	            [bombeBalayable]) — les faits disent donc exactement ce qui a ete lu.
//	inscription les gardes voyagent dans l EN-TETE des faits (codec 2), a cote des revisions.
//	comparaison [FilmFactsEntete.Utilisable] refuse des faits cuits SANS une garde que la cuisson
//	            demande, ou sous un autre roster ; un SUR-ENSEMBLE sert, et le rejeu retire ce que
//	            la cuisson ne demande pas ([FilmInputs.restreindreAuxGardes]) — il rend alors ce
//	            qu un decodage sous les gardes demandees rendrait.

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// ErrFilmFactsGardes : les faits ont ete cuits sous des gardes de l appelant qui ne couvrent pas
// celles de la cuisson courante. PERIMES pour CETTE cuisson : la reponse est de redecoder.
var ErrFilmFactsGardes = errors.New("faits de film : cuits sous d autres gardes de l appelant")

// GardesDeCuisson : ce que l appelant a commande au balayage, et que les faits ont donc lu.
type GardesDeCuisson struct {
	// Drapeau : le marqueur de portage et la jauge de retour du drapeau ont ete balayes.
	Drapeau bool
	// Zones : l etat des zones (`ti=13`) a ete balaye pour le catalogue de zones de l appelant.
	Zones bool
	// Bombe : l anneau d armement (`ti=12`) a ete balaye.
	Bombe bool
	// Roster : l empreinte des xuids que l appelant a fournis ([empreinteDuRoster]) — la table
	// d index de joueur se lit sur eux.
	Roster [sha256.Size]byte
}

// GardesDe rend les gardes qu une cuisson sous `opt` commande — LE point de derivation.
func GardesDe(opt Options) GardesDeCuisson {
	return GardesDeCuisson{
		Drapeau: drapeauBalayable(opt.Flag),
		Zones:   zonesBalayables(opt.Zone),
		Bombe:   bombeBalayable(opt.Bomb),
		Roster:  empreinteDuRoster(opt.RosterXUIDs),
	}
}

// drapeauBalayable : le calque du drapeau est demande ET les trois signaux du film disent CTF.
func drapeauBalayable(in FlagInput) bool {
	return in.Scanned && flagFilmSignalsOf(in).IsFlagFilm()
}

// zonesBalayables : l appelant a fourni un catalogue de zones (Bastion, colline de KOTH).
func zonesBalayables(in ZoneInput) bool { return len(in.Zones) > 0 }

// bombeBalayable : l appelant reconnait un match de la famille bomb ET fournit l horloge du
// manifeste sur laquelle l anneau se date.
func bombeBalayable(in BombInput) bool { return in.Scanned && len(in.ChunkStartMS) > 0 }

// empreinteDuRoster rend l empreinte de l ENSEMBLE des xuids : tries, sans doublon ni zero — les
// normalisations de `rosterOf`. Un roster vide a une empreinte, fixe.
func empreinteDuRoster(xuids []uint64) [sha256.Size]byte {
	ensemble := make([]uint64, 0, len(xuids))
	for _, x := range xuids {
		if x != 0 {
			ensemble = append(ensemble, x)
		}
	}
	slices.Sort(ensemble)
	ensemble = slices.Compact(ensemble)
	h := sha256.New()
	for _, x := range ensemble {
		_, _ = fmt.Fprintf(h, "%d\n", x)
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

// couvre rend nil si des faits cuits sous `g` servent une cuisson qui demande `d`.
func (g GardesDeCuisson) couvre(d GardesDeCuisson) error {
	for _, c := range []struct {
		nom           string
		demande, cuit bool
	}{{"drapeau", d.Drapeau, g.Drapeau}, {"zones", d.Zones, g.Zones}, {"bombe", d.Bombe, g.Bombe}} {
		if c.demande && !c.cuit {
			return fmt.Errorf("%w : %s demande, les faits ne l ont pas balaye", ErrFilmFactsGardes, c.nom)
		}
	}
	if d.Roster != g.Roster {
		return fmt.Errorf("%w : roster de l appelant different de celui de la cuisson des faits",
			ErrFilmFactsGardes)
	}
	return nil
}

// restreindreAuxGardes retire des entrees les canaux gardes que `d` ne demande pas, en leur
// donnant la valeur qu un balayage sous `d` leur aurait donnee.
func (in *FilmInputs) restreindreAuxGardes(d GardesDeCuisson) {
	if !d.Drapeau {
		in.FlagMarks = grammar.CarrierMarkScan{}
		in.FlagGauge, in.FlagGaugeScanned = nil, false
	}
	if !d.Zones {
		in.ZoneReads, in.ZoneScanned = nil, false
	}
	if !d.Bombe {
		in.BombReads = nil
	}
}

// encodeGardesDeCuisson / decodeGardesDeCuisson : les gardes dans l en-tete des faits.
func encodeGardesDeCuisson(w *gwriter, g GardesDeCuisson) {
	w.bool8(g.Drapeau)
	w.bool8(g.Zones)
	w.bool8(g.Bombe)
	w.b = append(w.b, g.Roster[:]...)
}

func decodeGardesDeCuisson(r *greader) GardesDeCuisson {
	g := GardesDeCuisson{Drapeau: r.bool8(), Zones: r.bool8(), Bombe: r.bool8()}
	copy(g.Roster[:], r.tranche(sha256.Size))
	return g
}
