package replay

// filmfacts_canaux.go — LES CANAUX DE FIN DE FLUX DU CODEC DES FAITS DE FILM.
//
// # CE QU ILS FERMENT
//
// Jusqu au 2026-09-14, le chemin du fixture RECOPIAIT la sequence de balayages de `BuildFromFilm`
// et cinq canaux entiers y manquaient — donc manquaient AUSSI au codec, donc aux huit goldens
// d assemblage, qui affirmaient « prises et lachers d arme decodes=0 » et « vehicules
// balaye=false » la ou la production publie ces calques (decouverte D7 du lot 0.D). Le codec
// appelle desormais l etage de PRODUCTION (`scanFilmInputs`), qui remplit ces canaux : sans leur
// codec, `TestGoldenInputsFidelite` rougirait — l assemblage sur entrees FRAICHES ne serait plus
// celui des entrees RELUES.
//
// Les six : `BipedCreations`, `WeaponChanges`, `Pickups` (+ stats), `EquipmentChanges`
// (+ stats), `ZoomEvents` et `Vehicles`. Le sixieme n est pas dans la liste du plan : c est
// l inventaire champ par champ de [FilmInputs] contre le codec qui l a fait apparaitre — la
// LUNETTE n existait pas non plus dans la copie, et `Options.Scoped` en sort.
//
// # CHAQUE ENCODEUR EST COLLE A SON DECODEUR
//
// Les deux moities d un codec se desynchronisent quand elles vivent dans deux fichiers : c est ce
// qui a coute la magie v10 (une section inseree d un cote seulement). Ici, elles se lisent
// ensemble.
//
// L INVENTAIRE DU TYPE, LUI, EST UN TEST : [TestCodecCouvreFilmInputs]
// (`golden_inputs_canaux_test.go`) exige que tout champ de [FilmInputs] soit soit serialise ici,
// soit nomme comme deliberement absent.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ---------------------------------------------------------------------------
// Creations de bipede — le lien DIRECT corps -> joueur (ti=35)
// ---------------------------------------------------------------------------

// encodeBipedCreations / decodeBipedCreations : ce que le registre d identite LIT d un record de
// creation, et rien de plus — la cle de vie (slot, generation), l index de participant avec SA
// PORTE, et l instant. `Version`, `Representation`, `BitPos`, `Chunk` et `PacketIndex` sont de la
// tracabilite de balayage : aucun assemblage ne les lit.
//
// `HasIndex` VOYAGE AVEC L INDEX, et il le faut : une porte fermee n est PAS un index nul
// (cf. filmdec/biped_creation.go). Relu a zero sans son temoin, le corps serait attribue au
// participant 0.
func encodeBipedCreations(w *gwriter, creations []grammar.BipedCreation) {
	w.u(uint64(len(creations)))
	var lastTS uint64
	for _, c := range creations {
		w.u(c.TimestampUS - lastTS) // les records sortent du balayage dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Generation))
		w.bool8(c.HasIndex)
		w.u(uint64(c.ParticipantIndex))
	}
}

func decodeBipedCreations(r *greader) []grammar.BipedCreation {
	n := int(r.u())
	out := make([]grammar.BipedCreation, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := grammar.BipedCreation{TimestampUS: lastTS, Slot: uint32(r.u()), Generation: uint32(r.u())}
		c.HasIndex = r.bool8()
		c.ParticipantIndex = uint32(r.u())
		out = append(out, c)
	}
	return out
}

// ---------------------------------------------------------------------------
// Prises et lachers d arme — le flux delta (i43..i46)
// ---------------------------------------------------------------------------

// encodeWeaponChanges / decodeWeaponChanges : l emplacement, l identite d arme et la QUALIFICATION
// du changement. `Low` (la variante cosmetique) et `Chunk` sont du diagnostic de balayage.
//
// `Kind` EST UNE CHAINE, et elle voyage telle quelle : c est elle que le calque publie
// (« prise », « lacher »), et la re-deriver a la relecture serait un second decideur.
func encodeWeaponChanges(w *gwriter, changes []types.HeldWeaponChange) {
	w.u(uint64(len(changes)))
	var lastTS uint64
	for _, c := range changes {
		w.u(c.TimestampUS - lastTS) // le balayage rend les emissions dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.i(int64(c.SlotIndex))
		w.u(uint64(c.Family))
		w.u(uint64(c.Previous))
		w.str(string(c.Kind))
	}
}

func decodeWeaponChanges(r *greader) []types.HeldWeaponChange {
	n := int(r.u())
	out := make([]types.HeldWeaponChange, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := types.HeldWeaponChange{TimestampUS: lastTS, Slot: uint32(r.u())}
		c.SlotIndex = int(r.i())
		c.Family = uint32(r.u())
		c.Previous = uint32(r.u())
		c.Kind = types.HeldWeaponChangeKind(r.str())
		out = append(out, c)
	}
	return out
}

// ---------------------------------------------------------------------------
// Ramassages natifs — l evenement `biped_pickup`
// ---------------------------------------------------------------------------

// encodePickups / decodePickups : l evenement et SES STATISTIQUES.
//
// LES STATS VOYAGENT AVEC LA LISTE, et il le faut : elles portent `MultiEvent`, c est-a-dire la
// mesure de ce que le canal ne peut PAS voir (un ramassage en 2e position d une liste lui
// echappe). Une liste vide sans elles serait indistinguable d un film sans ramassage.
func encodePickups(w *gwriter, pickups []types.BipedPickup, st types.BipedPickupStats) {
	w.u(uint64(len(pickups)))
	var lastTS uint64
	for _, p := range pickups {
		w.u(p.TimestampUS - lastTS)
		lastTS = p.TimestampUS
		w.u(uint64(p.Slot))
		w.u(uint64(p.CatalogID))
		w.byte8(p.Class)
	}
	w.u(uint64(st.Packets))
	w.u(uint64(st.Type9))
	w.u(uint64(st.Type8))
	w.u(uint64(st.OtherType))
	w.u(uint64(st.Published))
	w.u(uint64(st.MultiEvent))
	w.u(uint64(st.RefusedNoRef))
	w.u(uint64(st.RefusedNoCatalog))
	w.u(uint64(st.RefusedOffBand))
	w.u(uint64(st.UnexpectedWideRef))
}

func decodePickups(r *greader) ([]types.BipedPickup, types.BipedPickupStats) {
	n := int(r.u())
	out := make([]types.BipedPickup, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		p := types.BipedPickup{TimestampUS: lastTS, Slot: uint32(r.u()), CatalogID: uint32(r.u())}
		p.Class = r.byte8()
		out = append(out, p)
	}
	st := types.BipedPickupStats{
		Packets: int(r.u()), Type9: int(r.u()), Type8: int(r.u()), OtherType: int(r.u()),
		Published: int(r.u()), MultiEvent: int(r.u()), RefusedNoRef: int(r.u()),
		RefusedNoCatalog: int(r.u()), RefusedOffBand: int(r.u()), UnexpectedWideRef: int(r.u()),
	}
	return out, st
}

// ---------------------------------------------------------------------------
// Ramassages et consommations d equipement — le flux delta (i48)
// ---------------------------------------------------------------------------

// encodeEquipmentChanges / decodeEquipmentChanges : le changement et LE TEMOIN DE COMPLETUDE.
//
// `Counter`, `Recovered` et `Gap` sont dans le blob parce que la couverture les publie : ce canal
// est le seul du rejeu qui sache s auto-mesurer, et un `Gap` relu a zero affirmerait une chaine
// saine la ou des emissions manquent.
func encodeEquipmentChanges(w *gwriter, changes []types.EquipmentChange, st types.EquipmentChangeStats) {
	w.u(uint64(len(changes)))
	var lastTS uint64
	for _, c := range changes {
		w.u(c.TimestampUS - lastTS)
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Counter))
		w.i(int64(c.Rank))
		w.i(int64(c.Previous))
		w.str(string(c.Kind))
		w.bool8(c.Recovered)
		w.i(int64(c.Gap))
	}
	w.u(uint64(st.Walk.Records))
	w.u(uint64(st.Walk.WithI48))
	w.u(uint64(st.Walk.Read))
	w.u(uint64(st.Walk.Unread))
	w.u(uint64(st.Walk.Gated))
	w.u(uint64(st.Lives))
	w.u(uint64(st.Repeats))
	w.u(uint64(st.CounterJumps))
	w.u(uint64(st.MissedEstimate))
	w.u(uint64(st.LivesFirstOffSpec))
	w.u(uint64(st.Spawned))
	w.u(uint64(st.Taken))
	w.u(uint64(st.Spent))
	w.u(uint64(st.Recovered))
}

func decodeEquipmentChanges(r *greader) ([]types.EquipmentChange, types.EquipmentChangeStats) {
	n := int(r.u())
	out := make([]types.EquipmentChange, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := types.EquipmentChange{TimestampUS: lastTS, Slot: uint32(r.u()), Counter: uint32(r.u())}
		c.Rank = int(r.i())
		c.Previous = int(r.i())
		c.Kind = types.EquipmentChangeKind(r.str())
		c.Recovered = r.bool8()
		c.Gap = int(r.i())
		out = append(out, c)
	}
	var st types.EquipmentChangeStats
	st.Walk = types.AbilityRankStats{
		Records: int(r.u()), WithI48: int(r.u()), Read: int(r.u()),
		Unread: int(r.u()), Gated: int(r.u()),
	}
	st.Lives = int(r.u())
	st.Repeats = int(r.u())
	st.CounterJumps = int(r.u())
	st.MissedEstimate = int(r.u())
	st.LivesFirstOffSpec = int(r.u())
	st.Spawned = int(r.u())
	st.Taken = int(r.u())
	st.Spent = int(r.u())
	st.Recovered = int(r.u())
	return out, st
}

// ---------------------------------------------------------------------------
// Lunette — les bascules de la liste d evenements
// ---------------------------------------------------------------------------

// encodeZoomEvents / decodeZoomEvents : les bascules BRUTES.
//
// C EST LA LISTE QUI VOYAGE, PAS LA FERMETURE. L assemblage consomme `Options.Scoped`, une
// fonction (slot, instant) -> palier, que `FilmInputs.applyTo` reconstruit par
// `buildScopedLookup` a partir de ces evenements et des vies lues dans les positions. Une
// fermeture ne se serialise pas ; ses ENTREES, si.
func encodeZoomEvents(w *gwriter, events []grammar.ZoomEvent) {
	w.u(uint64(len(events)))
	var lastTS uint64
	for _, z := range events {
		w.u(z.TimestampUS - lastTS)
		lastTS = z.TimestampUS
		w.u(uint64(z.Slot))
		w.i(int64(z.Level))
	}
}

func decodeZoomEvents(r *greader) []grammar.ZoomEvent {
	n := int(r.u())
	out := make([]grammar.ZoomEvent, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		out = append(out, grammar.ZoomEvent{
			TimestampUS: lastTS, Slot: uint32(r.u()), Level: int(r.i()),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Vehicules (ti=40) — cinq lectures d un meme film
// ---------------------------------------------------------------------------

// encodeVehicleScan / decodeVehicleScan : les cinq lectures que l assemblage consomme.
//
// `Scanned` OUVRE LA SECTION, et c est le temoin qui distingue « aucun vehicule sur cette carte »
// de « le calque n a pas ete lu » : `buildVehicleTracks` se tait entierement quand il est faux.
//
// `Stats` (`EquipmentCreationStats`) N EST PAS TRANSPORTE : aucun assemblage ne le lit — la
// couverture du calque se calcule sur les vies, les creations et les episodes. Le fixture porte
// ce que l assemblage consomme, ni plus ni moins ; [TestCodecCouvreFilmInputs] tient la liste des
// champs deliberement absents.
func encodeVehicleScan(w *gwriter, s VehicleScan) {
	w.bool8(s.Scanned)
	encodeKeyframes(w, s.Keyframes)
	encodeCreations(w, s.Creations)
	encodePositionSection(w, s.Positions)
	encodeVehicleEvents(w, s.Events)
	encodeVehicleAims(w, s.Aims)
}

func decodeVehicleScan(r *greader, lay profile.I0Layout, world profile.Vec3Range) VehicleScan {
	s := VehicleScan{Scanned: r.bool8()}
	s.Keyframes = decodeKeyframes(r)
	s.Creations = decodeCreations(r)
	s.Positions = decodePositionSection(r, lay, world)
	s.Events = decodeVehicleEvents(r)
	s.Aims = decodeVehicleAims(r)
	return s
}

// encodeVehicleEvents / decodeVehicleEvents : les embarquements et les sorties.
//
// LES TEMOINS DE VALIDITE VOYAGENT AVEC LEURS REFERENCES (`OccupantPresent`, `VehicleSlotValid`,
// `SeatValid`) : un slot relu a zero sans son temoin nommerait l entite 0 — un vrai slot — la ou
// l evenement ne nommait rien. `OccupantSonde`, `OccupantInBand` et `VehicleGen` sont les
// controles independants que le calque consulte pour trancher entre deux vies de meme slot.
func encodeVehicleEvents(w *gwriter, events []types.VehicleEvent) {
	w.u(uint64(len(events)))
	var lastTS uint64
	for _, e := range events {
		w.u(e.TimestampUS - lastTS)
		lastTS = e.TimestampUS
		w.i(int64(e.Kind))
		w.bool8(e.OccupantPresent)
		w.i(int64(e.OccupantSonde))
		w.u(uint64(e.OccupantSlot))
		w.bool8(e.OccupantInBand)
		w.u(uint64(e.VehicleSlot))
		w.bool8(e.VehicleSlotValid)
		w.u(uint64(e.VehicleGen))
		w.u(uint64(e.Seat))
		w.bool8(e.SeatValid)
	}
}

func decodeVehicleEvents(r *greader) []types.VehicleEvent {
	n := int(r.u())
	out := make([]types.VehicleEvent, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		e := types.VehicleEvent{TimestampUS: lastTS, Kind: int(r.i())}
		e.OccupantPresent = r.bool8()
		e.OccupantSonde = int(r.i())
		e.OccupantSlot = uint32(r.u())
		e.OccupantInBand = r.bool8()
		e.VehicleSlot = uint32(r.u())
		e.VehicleSlotValid = r.bool8()
		e.VehicleGen = uint32(r.u())
		e.Seat = uint32(r.u())
		e.SeatValid = r.bool8()
		out = append(out, e)
	}
	return out
}

// encodeVehicleAims / decodeVehicleAims : les visees des bipedes qui ne repliquent PLUS leur
// position — celles des occupants. Bande `ti=35`, pas `ti=40` : la visee publiee sur un episode
// est celle de l HOMME a bord, jamais du chassis.
func encodeVehicleAims(w *gwriter, aims []grammar.BipedAim) {
	w.u(uint64(len(aims)))
	var lastTS uint64
	for _, a := range aims {
		w.u(a.TimestampUS - lastTS)
		lastTS = a.TimestampUS
		w.u(uint64(a.Slot))
		w.u(uint64(a.YawRaw))
		w.u(uint64(a.PitchRaw))
	}
}

func decodeVehicleAims(r *greader) []grammar.BipedAim {
	n := int(r.u())
	out := make([]grammar.BipedAim, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		out = append(out, grammar.BipedAim{
			TimestampUS: lastTS, Slot: uint32(r.u()),
			YawRaw: uint32(r.u()), PitchRaw: uint32(r.u()),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// LES QUATRE CANAUX GARDES PAR L APPELANT (lot 4.1.1-b, 2026-09-17)
// ---------------------------------------------------------------------------

// encodeGardesDeMode / decodeGardesDeMode serialisent les QUATRE canaux que le codec ne portait
// pas : le marqueur de portage du drapeau, l etat des zones avec son temoin, et l anneau
// d armement de la bombe.
//
// # POURQUOI ILS N Y ETAIENT PAS, ET POURQUOI C ETAIT UN TROU
//
// La table `champsNonTransportes` les declarait absents « parce que le fixture ne fournit AUCUNE
// garde » : les trois calques ne se balaient que si `Options.Flag` / `.Zone` / `.Bomb` portent la
// garde de mode, et le fixture d entrees n en fournit pas. C etait vrai POUR UN FIXTURE et FAUX
// EN PRODUCTION — tout CTF remplit `FlagMarks`, tout KOTH/Strongholds `ZoneReads`, tout Assaut
// armable `BombReads`. Le codec passant en production au lot 4.1.1-a, l absence de ces canaux
// aurait rendu un artefact rejoue SANS calque de drapeau, SANS zones et SANS armement sur les
// films qui en portent : la table est donc VIDEE et son mecanisme SUPPRIME avec sa derniere
// entree (meme doctrine que les trois allowlists de `film_layers_deps_test.go`).
//
// `ZoneScanned` VOYAGE AVEC `ZoneReads`, et il le faut : une liste vide et un balayage QUI N A
// PAS EU LIEU ne disent pas la meme chose, et la couverture publie la difference (meme lecon que
// le temoin `Scanned` de la v13).
func encodeGardesDeMode(w *gwriter, in FilmInputs) {
	encodeCarrierMarkScan(w, in.FlagMarks)
	w.u(uint64(len(in.ZoneReads)))
	for _, z := range in.ZoneReads {
		w.u(uint64(z.Slot))
		w.u(z.TimestampUS)
		w.i(int64(z.Field))
		w.i(int64(z.FilmIndex))
		w.i(int64(z.Tag))
		w.u(z.Value)
		w.bool8(z.HasValue)
		w.bool8(z.Chained)
	}
	w.bool8(in.ZoneScanned)
	w.u(uint64(len(in.BombReads)))
	for _, b := range in.BombReads {
		w.u(uint64(b.Slot))
		w.i(int64(b.TMS))
		w.byte8(b.Q)
		w.bool8(b.Chained)
	}
}

func decodeGardesDeMode(r *greader, in *FilmInputs) {
	in.FlagMarks = decodeCarrierMarkScan(r)
	n := int(r.u())
	in.ZoneReads = make([]grammar.ManagedPropertyRead, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		in.ZoneReads = append(in.ZoneReads, grammar.ManagedPropertyRead{
			Slot:        uint32(r.u()),
			TimestampUS: r.u(),
			Field:       grammar.ManagedPropertyField(r.i()),
			FilmIndex:   int(r.i()),
			Tag:         int(r.i()),
			Value:       r.u(),
			HasValue:    r.bool8(),
			Chained:     r.bool8(),
		})
	}
	if len(in.ZoneReads) == 0 {
		in.ZoneReads = nil
	}
	in.ZoneScanned = r.bool8()
	n = int(r.u())
	in.BombReads = make([]types.NavpointRadialRead, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		in.BombReads = append(in.BombReads, types.NavpointRadialRead{
			Slot: uint32(r.u()), TMS: int32(r.i()), Q: r.byte8(), Chained: r.bool8(),
		})
	}
	if len(in.BombReads) == 0 {
		in.BombReads = nil
	}
}

// encodeCarrierMarkScan / decodeCarrierMarkScan : les marques de portage du drapeau ET leur
// DENOMINATEUR (les instants de chaque image-cle balayee, marque ou non). Sans le denominateur,
// « 3 portages confirmes » ne dit pas si les autres ont ete observes — c est ecrit en tete de
// `grammar.CarrierMarkScan`, et c est pourquoi les quatre champs voyagent ensemble.
func encodeCarrierMarkScan(w *gwriter, s grammar.CarrierMarkScan) {
	w.u(uint64(len(s.Marks)))
	var last uint64
	for _, m := range s.Marks {
		w.u(m.TimestampUS - last) // instants non decroissants dans l ordre du film
		last = m.TimestampUS
		w.u(uint64(m.Slot))
	}
	w.u(uint64(len(s.KeyframeUS)))
	last = 0
	for _, ts := range s.KeyframeUS {
		w.u(ts - last)
		last = ts
	}
	w.u(uint64(s.Records))
	w.u(uint64(s.BipedRecords))
}

func decodeCarrierMarkScan(r *greader) grammar.CarrierMarkScan {
	var s grammar.CarrierMarkScan
	n := int(r.u())
	s.Marks = make([]grammar.CarrierMark, 0, n)
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		s.Marks = append(s.Marks, grammar.CarrierMark{TimestampUS: last, Slot: uint32(r.u())})
	}
	if len(s.Marks) == 0 {
		s.Marks = nil
	}
	n = int(r.u())
	s.KeyframeUS = make([]uint64, 0, n)
	last = 0
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		s.KeyframeUS = append(s.KeyframeUS, last)
	}
	if len(s.KeyframeUS) == 0 {
		s.KeyframeUS = nil
	}
	s.Records, s.BipedRecords = int(r.u()), int(r.u())
	return s
}
