package replay

// document_stances.go — LES ETATS DE MOUVEMENT, EN INTERVALLES PAR VIE (schema 65, lot 5.3.6).
//
// # CE QUE CE CALQUE PUBLIE
//
// `grammar.ScanMovementStates` rend des LECTURES — une suite de TRANSITIONS, parce qu un delta
// ne porte ces composants que quand l etat CHANGE. Ce fichier les replie en INTERVALLES bornes
// aux vies publiees : `{slot, kind, t0, t1}` sur le meme axe de temps que `Point.T`.
//
// QUATRE GENRES LUS — `crouch` (`i29`), `slide` (`i62`), `clamber` (`i54`), `sprint` (`i57`) —
// ET UN DERIVE, `jumpDerived` (schema 66, lots 5.9.4 et 5.9.5). Le derive n est pas lu dans un
// composant : c est l integrale de la vitesse verticale d `i1`, reconnue a sa HAUTEUR (0,85 m
// +/- 10 %, mesuree sur deux films au lot 5.7.5). Son nom le dit, et
// `coverage.stances.jumpsDerived` le compte a part.
//
// LE SPRINT EST LU, ET SA FENTE EST NOMMEE PAR L IMAGE (lot 5.9.5) : `i57` porte l INDEX DE LA
// FENTE DE CAPACITE ACTIVE, et `FUN_1407e9ce4` aiguille sur le groupe de tag de la definition
// pour appeler, par fente, un desenregistreur qui teste l index actif contre SA fente —
// `'saev'` esquive 0, `'sasp'` SPRINT 1, `'sagh'` grappin 2. Le controle qui valide la lecture
// de l index est celui du grappin : sur `4f77afc1` la vitesse au sol pendant les intervalles de
// la fente 2 atteint 5,84 m/s au p90 contre 2,88 hors — la traction.
//
// # LE PLIEUR N EST PAS RECOPIE, ET C EST LA REGLE 6
//
// La machine qui ouvre sur une lecture POSEE, ferme sur une lecture LEVEE, ferme a la fin de la
// vie et borne l intervalle a la fenetre publiee est `episodeAccum` (`equipment_episodes.go`).
// Elle est employee TELLE QUELLE ici, et sa sortie est convertie : une seconde copie de cette
// regle finirait par diverger de la premiere. `episodeAccum` emet un `EquipmentEpisode` — c est
// le prix de la reutilisation, et il se paie en une boucle de conversion plutot qu en une
// deuxieme machine a etats.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Stance est UN intervalle d etat de mouvement, porte par une vie.
type Stance struct {
	// Slot designe la Track concernee — donc une VIE, pas un joueur (meme regle que les autres
	// calques : le slot migre aux reapparitions).
	Slot uint32 `json:"slot"`
	// Kind est le genre d etat : `crouch`, `slide`, `clamber`, `sprint` (LUS) ou `jumpDerived`
	// (DERIVE — cf. `types.MovementJumpDerived`). Un genre inconnu du client ne doit recevoir
	// AUCUN libelle — jamais celui d un voisin.
	Kind string `json:"kind"`
	// T0 / T1 bornent l intervalle sur le meme axe que `Point.T`. T1 est soit la transition
	// MESUREE de fin, soit la fin de la vie.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
}

// StanceCoverage dit ce que la marche a lu et ce qu elle a jete — les denominateurs sans
// lesquels « N intervalles » ne se juge pas.
//
// UNE COUVERTURE PARTIELLE EST UN RESULTAT, PAS UN ECHEC : la plupart des vies ne s accroupissent
// ni ne glissent, et zero intervalle sur un film sans transition est la valeur juste.
type StanceCoverage struct {
	// Scanned dit que le balayage a eu lieu. Faux = il a refuse, et tout le reste est sans
	// valeur.
	Scanned bool `json:"scanned"`
	// Absent dit que l archetype bipede du film ne declare AUCUN des trois composants — un fait
	// mesure, distinct d un film ou personne ne bouge.
	Absent bool `json:"absent,omitempty"`
	// Records est le nombre de records `ti=35` que la marche a traverses, Desyncs ceux qu elle
	// n a pas pu finir. Mesure de reference : 3 desyncs sur 97 447 records (`bfecd02b`).
	Records int `json:"records"`
	Desyncs int `json:"desyncs"`
	// Reads est le nombre de LECTURES retenues, Intervals le nombre d intervalles publies. Le
	// second est toujours plus petit : une lecture qui ne change pas l etat ne produit rien.
	Reads     int `json:"reads"`
	Intervals int `json:"intervals"`
	// JumpEpisodes est le nombre de montees FERMEES examinees par la derivation du saut,
	// JumpsDerived celles dont la hauteur integree tombe dans la fenetre de
	// `types.SpartanJumpHeightM`. Le rapport des deux est la SELECTIVITE de la derivation :
	// sans lui, « N sauts » ne se juge pas. ZERO sur un artefact dont le film ne replique pas
	// la vitesse — c est `Scanned` qui dit si la marche a tourne.
	JumpEpisodes int `json:"jumpEpisodes,omitempty"`
	JumpsDerived int `json:"jumpsDerived,omitempty"`
	// ByKind compte les intervalles par genre. Une cle ABSENTE veut dire « aucun intervalle de
	// ce genre », jamais « genre non mesure » — c est `Scanned` qui le dit.
	ByKind map[string]int `json:"byKind,omitempty"`
	// Lives est le nombre de vies publiees portant au moins un intervalle, TracksTotal le
	// nombre de vies publiees du document (le denominateur).
	Lives       int `json:"lives"`
	TracksTotal int `json:"tracksTotal"`
	// Dropped compte les lectures ECARTEES faute de vie publiee ou faute de slot lie au bipede
	// (l attribution de slot du chemin d inference est partielle pour un record NEW — D13 de la
	// note 5.3). Les attribuer a tort serait pire que de les jeter, et ce compteur dit le prix.
	Dropped int `json:"dropped,omitempty"`
	// EventPacketsUnlocated compte les paquets a liste d evenements dont la signature n a pas
	// localise la trame : leurs records ne sont PAS lus. Sans ce compteur, « N intervalles » ne
	// dit pas sur quelle part du film ils portent.
	EventPacketsUnlocated int `json:"eventPacketsUnlocated,omitempty"`
	// MapWidths est le triplet de largeurs d axe employe par le chemin absolu d `i0`. PUBLIE
	// parce que c est le pre-requis le plus facile a oublier : un triplet qui n est pas celui de
	// la carte du match rend la marche muette (cf. `grammar/movement_states.go`).
	MapWidths [3]uint `json:"mapWidths,omitempty"`
}

// stanceInputs porte ce dont l assemblage a besoin. Une structure plutot que sept parametres :
// le seuil de CLAUDE.md est de cinq, et la liste grandirait au premier genre de plus.
type stanceInputs struct {
	reads         []types.MovementStateRead
	stats         types.MovementStateStats
	origin, step  uint64
	tracks        []Track
	closedByDeath map[int]bool
}

// buildStances replie les lectures en intervalles et rend la couverture.
func buildStances(in stanceInputs) ([]Stance, StanceCoverage) {
	cov := StanceCoverage{Scanned: in.stats.Scanned, Absent: in.stats.Absent,
		Records: in.stats.Records, Desyncs: in.stats.Desyncs, Reads: len(in.reads),
		TracksTotal: len(in.tracks), Dropped: in.stats.SlotUnbound,
		EventPacketsUnlocated: in.stats.EventPacketsUnlocated, MapWidths: in.stats.MapWidths,
		JumpEpisodes: in.stats.JumpEpisodes, JumpsDerived: in.stats.JumpsDerived}
	if len(in.tracks) == 0 || in.step == 0 || len(in.reads) == 0 {
		return nil, cov
	}
	windows := trackFrameWindows(in.tracks, in.closedByDeath)
	parCle := map[stanceKey][]types.MovementStateRead{}
	for _, r := range in.reads {
		if _, ok := windows[r.Slot]; !ok {
			cov.Dropped++ // vie non publiee : aucune fiche ou poser l intervalle
			continue
		}
		k := stanceKey{slot: r.Slot, kind: r.Kind}
		parCle[k] = append(parCle[k], r)
	}
	out := stancesDesCles(parCle, windows, in.origin, in.step)
	sortStances(out)
	cov.Intervals = len(out)
	cov.ByKind = map[string]int{}
	// CE N EST PAS L ENSEMBLE DES SLOTS PUBLIES, et la nuance compte : c est l ensemble des vies
	// qui PORTENT au moins un intervalle, derive des intervalles qu on vient de produire — pas
	// une seconde derivation de `doc.Tracks` (celle-la n a qu un proprietaire,
	// `published_tracks.go`). D ou un ensemble d appartenance, et non une table de booleens qui
	// ressemblerait a l autre.
	lives := map[uint32]struct{}{}
	for _, s := range out {
		cov.ByKind[s.Kind]++
		lives[s.Slot] = struct{}{}
	}
	cov.Lives = len(lives)
	if len(cov.ByKind) == 0 {
		cov.ByKind = nil
	}
	return out, cov
}

// stanceKey est la cle de pliage : UN genre, UNE vie. Deux genres du meme slot sont deux
// machines a etats independantes — s accroupir et glisser ne s excluent pas dans le flux.
type stanceKey struct {
	slot uint32
	kind string
}

// stancesDesCles deroule le plieur sur chaque (vie, genre), dans un ordre DETERMINISTE.
func stancesDesCles(parCle map[stanceKey][]types.MovementStateRead,
	windows map[uint32][]lifeWindow, origin, step uint64) []Stance {
	cles := make([]stanceKey, 0, len(parCle))
	for k := range parCle {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].slot != cles[j].slot {
			return cles[i].slot < cles[j].slot
		}
		return cles[i].kind < cles[j].kind
	})
	// DIMENSIONNEE SUR LES CLES : un (vie, genre) rend au moins un intervalle dans le cas
	// courant, et la borne evite les reallocations sur les films a 200 vies.
	out := make([]Stance, 0, len(cles))
	for _, k := range cles {
		list := parCle[k]
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].TimestampUS < list[j].TimestampUS
		})
		// LE PLIEUR DE `equipment_episodes.go`, EMPLOYE TEL QUEL (cf. l en-tete) : `fam` porte
		// le genre le temps du pliage, et la conversion le remet dans `Kind`.
		var episodes []EquipmentEpisode
		acc := episodeAccum{slot: k.slot, fam: k.kind, windows: windows[k.slot], out: &episodes}
		for _, r := range list {
			acc.sample(frameOf(r.TimestampUS, origin, step), r.On)
		}
		acc.finish()
		for _, e := range episodes {
			out = append(out, Stance{Slot: e.Slot, Kind: e.Fam, T0: e.T0, T1: e.T1})
		}
	}
	return out
}

// sortStances ordonne les intervalles sur (t0, slot, genre) — un ordre TOTAL, comme tous les
// calques du document. Un tri partiel ferait dependre l artefact du parcours d une map.
func sortStances(out []Stance) {
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.T0 != b.T0 {
			return a.T0 < b.T0
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Kind < b.Kind
	})
}
