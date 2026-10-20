package replay

// positions_porte_depart.go — LA PORTE DES POSITIONS, REGLE R-B3 : aucune vie d'un corps apres le
// depart PROUVE de l'occupant qui vivait a sa creation.
//
// # CE QUE LE FILM DIT
//
// Le record de creation d'un corps porte l'index de son occupant ; l'entite `ti=9` de cet index dont
// la fenetre LARGE contient la creation est l'occupant qui y vivait (la lecture de
// identity_registry_entites.go), et l'image-cle porteuse qui prouve l'absence de cette entite apres
// sa derniere lecture borne son depart. Un corps cree avant ce depart et dont AUCUNE position ne le
// precede n'a porte aucune vie de cet occupant : ses positions posterieures ne sont pas une vie de
// joueur. Le balayage par ancrage peut en lire sur un slot que la marche des trames ne lit plus
// (registre des reports, phase D du lot des equipes du rejeu) ; la preuve d'absence des images-cles
// prime sur ces echantillons.
//
// # CE QU'ELLE NE TOUCHE PAS
//
// Un corps dont une position precede le depart : sa vie a commence avec l'occupant, elle reste
// entiere. Une entite dont l'absence n'est pas prouvee, ou deux entites de l'index qui vivent a la
// creation, ou un film sans balayage des entites : la regle se tait.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/observability"
)

// metriqueViesApresDepart : le compteur expvar des corps dont la porte ecarte les positions par R-B3.
const metriqueViesApresDepart = "rejeu_vies_apres_depart_ecartees"

// corpsDeLaPorte identifie un corps : son slot et le record de creation qui l'ouvre.
type corpsDeLaPorte struct {
	slot uint32
	tUS  int64
	gen  uint32
}

// ecarterApresLeDepart retire les positions des corps que R-B3 ecarte (cf. l'en-tete) et les compte
// dans `cov` : `ApresDepart` les positions, `CorpsApresDepart` les corps. Les positions arrivent
// triees par instant ; l'entree n'est pas modifiee.
func ecarterApresLeDepart(pos []grammar.BipedPosition, creations []grammar.BipedCreation,
	scan grammar.PlayerEntityScan, cov *couverturePorte) []grammar.BipedPosition {
	if !scan.Scanned || len(scan.Entities) == 0 || len(creations) == 0 {
		return pos
	}
	corps := corpsParSlot(creations)
	// ecartes : le verdict de R-B3 par corps, pose a sa PREMIERE position (les positions sont triees).
	ecartes := map[corpsDeLaPorte]bool{}
	out := make([]grammar.BipedPosition, 0, len(pos))
	for _, p := range pos {
		d, ok := corps[p.Slot].recordA(int64(p.TimestampUS)) //nolint:gosec // horloge du film, bien sous 2^63
		if !ok {
			out = append(out, p)
			continue
		}
		cle := corpsDeLaPorte{slot: p.Slot, tUS: d.tUS, gen: d.gen}
		ecarte, vu := ecartes[cle]
		if !vu {
			depart, prouve := departDeLOccupant(scan, d)
			ecarte = prouve && p.TimestampUS >= depart
			ecartes[cle] = ecarte
			if ecarte {
				cov.CorpsApresDepart++
			}
		}
		if ecarte {
			cov.ApresDepart++
			continue
		}
		out = append(out, p)
	}
	return out
}

// departDeLOccupant rend l'instant de l'image-cle porteuse qui prouve le depart de l'occupant vivant
// a la creation `d` : l'UNIQUE entite stable de son index dont la fenetre large contient la
// creation. Faux sans entite, avec deux entites, ou quand aucune image-cle ne prouve son absence.
func departDeLOccupant(scan grammar.PlayerEntityScan, d dateDeCreation) (uint64, bool) {
	if d.tUS < 0 {
		return 0, false
	}
	t := uint64(d.tUS)
	var elue *fenetreLarge
	for _, e := range scan.Entities {
		if e.Index != int(d.index) || e.Unstable {
			continue
		}
		f := fenetreLargeDe(scan, e)
		if !f.contient(t) {
			continue
		}
		if elue != nil {
			return 0, false
		}
		elue = &f
	}
	if elue == nil || elue.ouverteApres {
		return 0, false
	}
	return elue.a, true
}

// journaliserLesViesApresDepart dit, en AVERTISSEMENT et au compteur, les corps que R-B3 a ecartes :
// des positions que le film ne peut pas avoir ecrites pour cet occupant, jamais un silence.
func (c couverturePorte) journaliserLesViesApresDepart(ctx context.Context, matchID string) {
	if c.CorpsApresDepart == 0 {
		return
	}
	observability.AddInt(metriqueViesApresDepart, int64(c.CorpsApresDepart))
	slog.WarnContext(ctx, "rejeu : positions d'un corps posterieures au depart prouve de son occupant — "+
		"aucune vie publiee", "match_id", matchID, "corps", c.CorpsApresDepart, "positions", c.ApresDepart)
}
