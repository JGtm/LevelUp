package replay

// filmfacts_mortsdobjet.go — LA FIN DE VIE ECRITE D UN OBJET DU MONDE, DANS LE FICHIER DE FAITS.
//
// Extrait de `filmfacts_canaux.go` le 2026-09-18 : celui-ci franchissait les 500 lignes, et la
// table de `film_file_size_test.go` est DATEE ET FERMEE au 2026-09-16 — un fichier neuf ne s y
// inscrit pas, il se coupe.
//
// CE QUE CE FICHIER PORTE : la section des morts d objet (vehicules), sa charge JSON, et la
// PROJECTION qui exclut l observateur du cadre de balayage. Cf. l en-tete de `encodeMortsDObjet`
// pour la raison, qui est une mesure du gate S8.

import (
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// encodeMortsDObjet / decodeMortsDObjet : LA FIN DE VIE ECRITE D UN VEHICULE, et les denominateurs
// de sa lecture.
//
// # CE QUE LEUR ABSENCE A COUTE, ET COMMENT ON L A SU (2026-09-18, gate S8)
//
// `encodeVehicleScan` portait SIX champs sur neuf : `Stats`, `Deaths` et `DeathStats` tombaient.
// `Deaths` est la fin de vie LUE au dead-state ecrit (lot 1.9.10) : sans elle, un artefact rejoue
// republie `VehicleTrack.End` a `unknown` pour tout le monde et perd les huit compteurs de
// `coverage.vehicles`. La mesure est nette : le S8 rendait 10/10 artefacts divergents, et apres
// correction du reste du codec les CINQ films qui perdaient encore etaient exactement les cinq
// BTB / Heavies — les seuls a porter des vehicules. `a349fea8` y perdait 223 064 octets.
//
// # EN JSON, ET C EST LA MEME DECISION QUE POUR LES SECTIONS ii-v DU FICHIER
//
// `ObjectDeathStats` porte QUATRE MAPS (`Records`, `CleanRecords`, `MaskDeclared`,
// `MaskDeclaredDesync`) et un `FrameConfig` ; `ObjectDeath` porte un `DeadState` de dix champs.
// Ecrire cela a la main, c est reconduire exactement le defaut que ce lot repare : une liste de
// champs tenue par l attention d un relecteur. `encoding/json` derive du TYPE, trie ses cles de
// map, et ne peut pas oublier un champ. Le cout est paye une fois a la cuisson.
//
// LE CADRE EST A LONGUEUR PREFIXEE, comme les sections du fichier : le blob reste relisible meme
// si cette charge grossit.
func encodeMortsDObjet(w *gwriter, morts []types.ObjectDeath, st grammar.ObjectDeathStats) {
	// L OBSERVATEUR N EST PAS UN FAIT, ET IL N EST PAS SERIALISABLE (2026-09-18).
	//
	// `ObjectDeathStats.Config` est un [grammar.FrameConfig], qui porte `Obs *Observation` — des
	// CROCHETS de fonction qu un instrument installe pour regarder passer un balayage. Ce ne sont
	// ni des donnees ni des faits du film : ils ne se persistent pas, et `encoding/json` refuse un
	// type fonction. En production `Obs` est nil et la serialisation passe ; un instrument qui en
	// poserait un ferait ECHOUER le marshal, et une charge vide serait une perte SILENCIEUSE —
	// precisement la classe de defaut que ce lot repare. On le met donc a nil A L ECRITURE, et
	// `TestLObservateurNEstPasUnFaitPersiste` epingle qu il est le SEUL membre non serialisable.
	charge, err := json.Marshal(chargeDesMortsDObjet{
		Morts: morts,
		Stats: versStatsSansCadre(st),
		Cadre: cadreSerialisable{
			HasExtraFields:      st.Config.HasExtraFields,
			IDLowBits:           st.Config.IDLowBits,
			IDBase:              st.Config.IDBase,
			NewDefaultStateBits: st.Config.NewDefaultStateBits,
			PacketPreambleBits:  st.Config.PacketPreambleBits,
			Profil:              st.Config.Profil,
		},
	})
	if err != nil {
		// JAMAIS SILENCIEUX : une charge vide se relirait comme une ABSENCE de morts, donc comme
		// un fait faux. Le flux porte l erreur, et l appelant redecodera le film.
		w.echec = fmt.Errorf("morts d objet non serialisables : %w", err)
		w.u(0)
		return
	}
	w.u(uint64(len(charge)))
	w.b = append(w.b, charge...)
}

func decodeMortsDObjet(r *greader) ([]types.ObjectDeath, grammar.ObjectDeathStats) {
	charge := r.tranche(int(r.u()))
	if r.err != nil || len(charge) == 0 {
		return nil, grammar.ObjectDeathStats{}
	}
	var out chargeDesMortsDObjet
	if err := json.Unmarshal(charge, &out); err != nil {
		r.err = fmt.Errorf("morts d objet : %w", err)
		return nil, grammar.ObjectDeathStats{}
	}
	return out.Morts, versObjectDeathStats(out.Stats, grammar.FrameConfig{
		HasExtraFields:      out.Cadre.HasExtraFields,
		IDLowBits:           out.Cadre.IDLowBits,
		IDBase:              out.Cadre.IDBase,
		NewDefaultStateBits: out.Cadre.NewDefaultStateBits,
		PacketPreambleBits:  out.Cadre.PacketPreambleBits,
		Profil:              out.Cadre.Profil,
	})
}

// chargeDesMortsDObjet est la forme SERIALISABLE de la section des morts d objet.
//
// # POURQUOI `Config` VOYAGE A PART, SOUS `Cadre`
//
// `grammar.ObjectDeathStats.Config` est un [grammar.FrameConfig], et celui-ci porte
// `Obs *Observation` — des CROCHETS DE FONCTION qu un instrument installe pour regarder passer un
// balayage. Ce ne sont ni des donnees ni des faits du film, et `encoding/json` refuse un type
// fonction : `staticcheck` (SA1026) le dit STATIQUEMENT, sur le type, et il a raison de le dire —
// en production `Obs` est nil et le marshal passerait, mais un instrument qui en poserait un
// ferait ECHOUER la serialisation, et une charge vide se relirait comme « aucune mort », un fait
// FAUX.
//
// `Cadre` porte donc les SIX champs de donnees du cadre, et l observateur ne traverse jamais le
// disque. Le `json:"-"` sur `Stats.Config` est ce qui rend l exclusion visible au compilateur
// comme au lecteur ; [TestLObservateurNEstPasUnFaitPersiste] epingle les deux moities.
type chargeDesMortsDObjet struct {
	Morts []types.ObjectDeath
	Stats statsSansCadre
	Cadre cadreSerialisable
}

// statsSansCadre : les TREIZE champs de donnees de [grammar.ObjectDeathStats]. Le quatorzieme,
// `Config`, voyage sous `Cadre` — il porte l observateur, qui n est pas un fait.
//
// UNE LISTE ECRITE A LA MAIN, ET UN RATCHET QUI LA TIENT :
// [TestStatsDeMortDObjetSontToutesPortees] compte les champs du type d origine et rougit si l un
// nait sans entrer ici. C est la contrepartie assumee de l exclusion : sans le ratchet, cette
// liste serait exactement la dette que ce lot repare.
type statsSansCadre struct {
	CadreParDefaut                                bool
	CadreLocalises, CadreDauphin, CadreEvenements int
	Keyframes, Deltas                             int
	Packets, EventPackets, LocatedPackets         int
	Records, CleanRecords                         map[uint32]int
	MaskDeclared, MaskDeclaredDesync              map[uint32]int
}

// versStatsSansCadre / versObjectDeathStats : les deux sens de la projection, cote a cote — deux
// conversions qui vivent loin l une de l autre se desynchronisent.
func versStatsSansCadre(st grammar.ObjectDeathStats) statsSansCadre {
	return statsSansCadre{
		CadreParDefaut: st.CadreParDefaut, CadreLocalises: st.CadreLocalises,
		CadreDauphin: st.CadreDauphin, CadreEvenements: st.CadreEvenements,
		Keyframes: st.Keyframes, Deltas: st.Deltas,
		Packets: st.Packets, EventPackets: st.EventPackets, LocatedPackets: st.LocatedPackets,
		Records: st.Records, CleanRecords: st.CleanRecords,
		MaskDeclared: st.MaskDeclared, MaskDeclaredDesync: st.MaskDeclaredDesync,
	}
}

func versObjectDeathStats(p statsSansCadre, cadre grammar.FrameConfig) grammar.ObjectDeathStats {
	return grammar.ObjectDeathStats{
		Config:         cadre,
		CadreParDefaut: p.CadreParDefaut, CadreLocalises: p.CadreLocalises,
		CadreDauphin: p.CadreDauphin, CadreEvenements: p.CadreEvenements,
		Keyframes: p.Keyframes, Deltas: p.Deltas,
		Packets: p.Packets, EventPackets: p.EventPackets, LocatedPackets: p.LocatedPackets,
		Records: p.Records, CleanRecords: p.CleanRecords,
		MaskDeclared: p.MaskDeclared, MaskDeclaredDesync: p.MaskDeclaredDesync,
	}
}

// cadreSerialisable : les six champs de DONNEES de [grammar.FrameConfig]. L observateur, qui est
// le septieme, n en est pas un.
type cadreSerialisable struct {
	HasExtraFields      bool
	IDLowBits           int
	IDBase              uint32
	NewDefaultStateBits int
	PacketPreambleBits  int
	Profil              grammar.ProfilDeBalayage
}
