package replay

// filmfacts_decode.go — LE DECODEUR DES FAITS DE FILM.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7), passe en PRODUCTION
// le 2026-09-17 (lot 4.1.1-a) sous son nom de production. DEPLACEMENTS PURS : aucune ligne de
// logique changee, seuls les noms d API et les messages ont suivi (cf. filmfacts.go).

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// DecodeFilmFacts relit les faits d un film. `entry` est l entree de catalogue de SA carte : les
// positions sont des quanta, et les relire avec une autre entree rendrait des coordonnees
// FAUSSES — d ou les deux erreurs typees [ErrFilmFactsCarte] et [ErrFilmFactsDecoupage].
func DecodeFilmFacts(blob []byte, entry profile.MapQuantEntry) (*FilmFacts, error) {
	g, r, lay, world, err := decodeEntete(blob, entry)
	if err != nil {
		return nil, err
	}
	g.Positions = decodePositionSection(r, lay, world)
	g.BipedCreations = decodeBipedCreations(r)
	decodeEvenements(r, g)
	g.KeyframeWalk = decodeMarcheImageCle(r)
	g.BirthLoadouts, g.BirthLoadoutStats = decodeNaissances(r)
	g.WeaponChanges = decodeWeaponChanges(r)
	g.Pickups, g.PickupStats = decodePickups(r)
	decodeInventaire(r, g)
	decodeCanauxDelta(r, g)
	g.EquipmentChanges, g.EquipmentChangeStats = decodeEquipmentChanges(r)
	decodeCapacites(r, g)
	decodeEtatsDeMouvement(r, g)
	g.ZoomEvents = decodeZoomEvents(r)
	decodeMonde(r, g)
	g.Vehicles = decodeVehicleScan(r, lay, world)
	decodeQueue(r, g)
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(r.b) {
		return nil, fmt.Errorf("faits de film : %d octet(s) non consomme(s) — format desynchronise",
			len(r.b)-r.off)
	}
	return g, nil
}

// decodeEntete relit l en-tete, verifie carte et decoupage, et rend le lecteur arme.
func decodeEntete(blob []byte, entry profile.MapQuantEntry) (
	*FilmFacts, *greader, profile.I0Layout, profile.Vec3Range, error,
) {
	if len(blob) < len(filmFactsMagic) || string(blob[:len(filmFactsMagic)]) != filmFactsMagic {
		return nil, nil, profile.I0Layout{}, profile.Vec3Range{}, fmt.Errorf("faits de film : magie absente ou version inconnue — redecoder le film")
	}
	r := &greader{b: blob, off: len(filmFactsMagic)}
	g := &FilmFacts{Film: r.str()}
	g.MapModule = r.str()
	for a := 0; a < 3; a++ {
		g.AxisW[a] = uint(r.u())
	}
	g.LayoutDetected = r.bool8()
	g.InventoryDeltaAmmoRefused = r.bool8()
	if r.bool8() {
		v := int(r.i())
		g.FilmMajorVersion = &v
	}
	if err := verifierCleDeCuisson(g.MapModule, g.AxisW, g.LayoutDetected, entry); err != nil {
		return nil, nil, profile.I0Layout{}, profile.Vec3Range{}, err
	}
	// LE DECOUPAGE VIENT DU BLOB, LES BORNES DU CATALOGUE : le premier dit comment le film a
	// quantifie, le second ou la carte commence et finit. Melanger les deux sources est ce qui
	// rendait des coordonnees fausses sur Live Fire.
	lay, world := profile.I0Layout{AxisW: g.AxisW}, entry.Range()
	g.FilmClockOriginUS = r.u()
	return g, r, lay, world, nil
}

// verifierCleDeCuisson confronte la CLE DE CUISSON relue a l entree de catalogue fournie.
//
// # POURQUOI DES ERREURS TYPEES, ET DANS LES DEUX SENS
//
// Les positions sont des QUANTA : `DequantBipedAxis` les rendrait avec les bornes de la mauvaise
// carte ou le pas du mauvais decoupage sans rien signaler — des coordonnees FAUSSES, pas
// approximatives.
//
// LE BLOB ET LE CATALOGUE DOIVENT DIRE LA MEME CHOSE, DANS LES DEUX SENS (revue R1, constat
// R1-2) : un blob qui se dit « du CATALOGUE » doit porter le decoupage que la regle tranche
// aujourd hui — sinon le catalogue a bouge sous lui ; un blob qui se dit « AUTO-DETECTE » doit
// venir d une carte dont l entree est INVALIDE — sinon il a ete cuit hors de la regle. Sur
// `60ae07c4` la detection rendait `13/12/11` la ou le catalogue rend `12/12/11`.
//
// UNE SEULE COPIE (CLAUDE.md regle 6) : l en-tete du blob des entrees ET l en-tete du FICHIER de
// faits ([FilmFactsEntete.Utilisable]) appellent cette fonction. Deux copies de cette regle
// auraient divergé au premier ajustement de catalogue.
func verifierCleDeCuisson(mapModule string, axisW [3]uint, layoutDetected bool,
	entry profile.MapQuantEntry,
) error {
	if mapModule != entry.Module {
		return fmt.Errorf("%w : faits cuits pour %q, entree de catalogue fournie %q",
			ErrFilmFactsCarte, mapModule, entry.Module)
	}
	impose := grammar.NewFilmContextForMap(nil, &entry, nil).ImposedLayout()
	switch {
	case !layoutDetected && (impose == nil || impose.AxisW != axisW):
		return fmt.Errorf("%w : faits au decoupage %v (dit du CATALOGUE), catalogue %v",
			ErrFilmFactsDecoupage, axisW, imposeAxisW(impose))
	case layoutDetected && impose != nil:
		return fmt.Errorf(
			"%w : faits disant leur decoupage %v AUTO-DETECTE, or le catalogue en impose un (%v)",
			ErrFilmFactsDecoupage, axisW, impose.AxisW)
	}
	return nil
}

// decodeEvenements relit tirs, equipements de depart, lancers et projectiles.
func decodeEvenements(r *greader, g *FilmFacts) {
	var lastTS uint64
	n := int(r.u())
	g.Fire = make([]grammar.FireEvent, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		var e grammar.FireEvent
		lastTS += r.u()
		e.TimestampUS = lastTS
		e.FilmIndex = int(r.i())
		e.WeaponID = r.u()
		if e.HasAim = r.bool8(); e.HasAim {
			for a := 0; a < 3; a++ {
				e.Aim[a] = r.f32()
			}
		}
		g.Fire = append(g.Fire, e)
	}

	n = int(r.u())
	g.Loadouts = make([]types.KeyframeLoadout, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		l := types.KeyframeLoadout{TimestampUS: r.u(), Slot: uint32(r.u())}
		nf := int(r.u())
		for j := 0; j < nf && r.err == nil; j++ {
			l.Families = append(l.Families, uint32(r.u()))
		}
		g.Loadouts = append(g.Loadouts, l)
	}

	n = int(r.u())
	g.Grenades = make([]grammar.GrenadeThrow, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Grenades = append(g.Grenades, grammar.GrenadeThrow{
			TimestampUS: r.u(), FilmIndex: int(r.i()), TypeID: uint32(r.u()),
		})
	}

	g.Projectiles = decodeTracks(r)

}

// decodeInventaire relit les inventaires d image-cle et leurs deltas.
func decodeInventaire(r *greader, g *FilmFacts) {
	var lastTS uint64
	n := int(r.u())
	g.Inventory = make([]KeyframeInventory, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		inv := KeyframeInventory{TimestampUS: r.u(), Slot: uint32(r.u())}
		inv.GrenadesRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Grenades[j] = uint32(r.u())
		}
		inv.SelectedGrenadeRank = int(r.i())
		inv.AbilityRank = int(r.i())
		inv.DrawnSlot = int(r.i())
		inv.AmmoCandidates = int(r.u())
		inv.AmmoRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Ammo[j] = decodeAmmo(r)
		}
		g.Inventory = append(g.Inventory, inv)
	}

	n = int(r.u())
	g.InventoryDeltas = make([]types.InventoryDelta, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		d := types.InventoryDelta{TimestampUS: lastTS, Slot: uint32(r.u())}
		if gn := int(r.u()); gn > 0 {
			d.Grenades = make([]uint32, 0, gn)
			for j := 0; j < gn && r.err == nil; j++ {
				d.Grenades = append(d.Grenades, uint32(r.u()))
			}
		}
		d.SelRead = r.bool8()
		d.Sel = int(r.i())
		d.Mask = uint32(r.u())
		g.InventoryDeltas = append(g.InventoryDeltas, d)
	}

}

// decodeCanauxDelta relit rangs de capacite, camouflage, grappin et translocations.
func decodeCanauxDelta(r *greader, g *FilmFacts) {
	var lastTS uint64
	n := int(r.u())
	g.AbilityRanks = make([]types.AbilityRank, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityRanks = append(g.AbilityRanks,
			types.AbilityRank{TimestampUS: lastTS, Slot: uint32(r.u()), Rank: int(r.i())})
	}

	n = int(r.u())
	g.CamoStates = make([]types.CamoRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.CamoStates = append(g.CamoStates,
			types.CamoRead{TimestampUS: lastTS, Slot: uint32(r.u()), Q: uint16(r.u())})
	}

	n = int(r.u())
	g.GrappleReads = make([]types.GrappleRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		gr := types.GrappleRead{TimestampUS: lastTS, Slot: uint32(r.u()), Heavy: r.bool8()}
		for a := 0; a < 3; a++ {
			gr.PosQ[a] = uint32(r.u())
		}
		g.GrappleReads = append(g.GrappleReads, gr)
	}

	n = int(r.u())
	g.Translocations = make([]types.TranslocatorTeleport, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		tr := types.TranslocatorTeleport{TimestampUS: lastTS, Slot: uint32(r.u())}
		tr.HasPositions = r.bool8()
		for a := 0; a < 3; a++ {
			tr.From[a] = r.f32()
		}
		for a := 0; a < 3; a++ {
			tr.To[a] = r.f32()
		}
		g.Translocations = append(g.Translocations, tr)
	}

}

// decodeCapacites relit les impulsions et les charges de capacite, stats comprises.
func decodeCapacites(r *greader, g *FilmFacts) {
	var lastTS uint64
	n := int(r.u())
	g.AbilityImpulses = make([]types.AbilityImpulse, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityImpulses = append(g.AbilityImpulses, types.AbilityImpulse{
			TimestampUS: lastTS, Slot: uint32(r.u()), Predicted: r.bool8()})
	}
	g.AbilityImpulseStats = types.AbilityImpulseStats{
		Records: int(r.u()), WithI57: int(r.u()), WithI59: int(r.u()),
		Read: int(r.u()), Unread: int(r.u()), Tag1: int(r.u()), Absent: r.bool8(),
		Scanned: r.bool8(),
	}

	n = int(r.u())
	g.AbilityCharges = make([]types.AbilityCharge, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityCharges = append(g.AbilityCharges, types.AbilityCharge{
			TimestampUS: lastTS, Slot: uint32(r.u()),
			Emplacement: int(r.u()), Charges: int(r.u()), Low: int(r.u())})
	}
	g.AbilityChargeStats = types.AbilityChargeStats{
		Records: int(r.u()), WithI56: int(r.u()),
		Read: int(r.u()), Unread: int(r.u()), Armed: int(r.u()),
		Absent: r.bool8(), Scanned: r.bool8(),
	}

}

// decodeEtatsDeMouvement relit les ETATS DE MOUVEMENT (v24), stats comprises.
func decodeEtatsDeMouvement(r *greader, g *FilmFacts) {
	n := int(r.u())
	g.MovementStates = make([]types.MovementStateRead, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.MovementStates = append(g.MovementStates, types.MovementStateRead{
			TimestampUS: lastTS,
			Slot:        uint32(r.u()),
			Kind:        r.str(),
			On:          r.bool8(),
			Progress:    uint32(r.u()),
			Chunk:       int(r.u()),
			PacketIndex: int(r.u()),
		})
	}
	st := types.MovementStateStats{
		Records: int(r.u()), Read: int(r.u()), Absent: r.bool8(), Scanned: r.bool8(),
		Packets: int(r.u()), EventPackets: int(r.u()), EventPacketsLocated: int(r.u()),
		EventPacketsUnlocated: int(r.u()), Desyncs: int(r.u()), SlotUnbound: int(r.u()),
		Duplicates: int(r.u()),
	}
	for i := range st.MapWidths {
		st.MapWidths[i] = uint(r.u())
	}
	g.MovementStateStats = st
}

// decodeMonde relit les poses d equipement et les deux voies de socles.
func decodeMonde(r *greader, g *FilmFacts) {
	var lastTS uint64
	n := int(r.u())
	g.Placements = make([]types.EquipmentPlacement, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		p := types.EquipmentPlacement{T0US: lastTS, T1US: r.u()}
		p.Life = types.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		p.X, p.Y, p.Z = r.f32(), r.f32(), r.f32()
		p.GlobalID, p.Points = uint32(r.u()), int(r.u())
		g.Placements = append(g.Placements, p)
	}
	g.PlacementStats = decodeStatsDePose(r)

	decodeSpawnEvents(r, g)

	g.Pads.Weapons = decodeWorldObjectScan(r)
	g.Pads.Powerups = decodeWorldObjectScan(r)

}

// decodeSpawnEvents relit les evenements 103 et leurs denominateurs.
func decodeSpawnEvents(r *greader, g *FilmFacts) {
	n := int(r.u())
	g.SpawnEvents = make([]types.EquipmentSpawnEvent, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		e := types.EquipmentSpawnEvent{TimestampUS: lastTS}
		e.Chunk, e.PacketIndex = int(r.i()), int(r.i())
		e.Spawned = types.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		e.SpawnedValid = r.bool8()
		e.Source = types.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		e.SourceValid = r.bool8()
		e.Ref2Present = r.bool8()
		g.SpawnEvents = append(g.SpawnEvents, e)
	}
	g.SpawnStats = types.EquipmentSpawnStats{
		Chunks: int(r.u()), Packets: int(r.u()), Lists: int(r.u()), Events: int(r.u()),
		WithSpawned: int(r.u()), WithSource: int(r.u()), Ref2: int(r.u()),
	}
}

// decodeQueue relit les morts et la table des index de joueur.
func decodeQueue(r *greader, g *FilmFacts) {
	n := int(r.u())
	g.Deaths = make([]Death, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Deaths = append(g.Deaths, Death{XUID: r.u(), Gamertag: r.str(), TimeMS: r.i()})
	}

	g.PlayerIndices = PlayerIndexTable{ByXUID: map[uint64]int{}}
	g.PlayerIndices.Readings = int(r.u())
	g.PlayerIndices.Disagreements = int(r.u())
	n = int(r.u())
	for k := 0; k < n && r.err == nil; k++ {
		x := r.u()
		g.PlayerIndices.ByXUID[x] = int(r.i())
	}

	g.FilmTable = decodeFilmTable(r)
	g.PlayerTeams, g.TeamScan = decodePlayerTeams(r)
}

// decodePlayerTeams relit l EQUIPE DE CHAQUE JOUEUR et le rapport de sa lecture (v21, lot 1.7).
func decodePlayerTeams(r *greader) (map[int]int, grammar.TeamScanReport) {
	var teams map[int]int
	if n := int(r.u()); n > 0 {
		teams = make(map[int]int, n)
		for k := 0; k < n && r.err == nil; k++ {
			i := int(r.i())
			teams[i] = int(r.i())
		}
	}
	rep := grammar.TeamScanReport{
		ArchetypeAbsent: r.bool8(), ComponentMismatch: r.bool8(), Component: r.str(),
	}
	for _, p := range []*int{&rep.Packets, &rep.Records, &rep.Read, &rep.Unreached,
		&rep.OutOfDomainIndex, &rep.OutOfDomainValue, &rep.Entities, &rep.EntityDivergences,
		&rep.IndexDivergences, &rep.Indices, &rep.NoTeam} {
		*p = int(r.u())
	}
	return teams, rep
}

// decodeEntitesDesJoueurs relit LES OCCUPANTS DU MATCH (SchemaDesFaits 4, lot M2.2), dans l ordre
// ou [encodeEntitesDesJoueurs] les ecrit. Les tranches vides relisent NIL, comme a la production.
func decodeEntitesDesJoueurs(r *greader) grammar.PlayerEntityScan {
	s := grammar.PlayerEntityScan{Scanned: r.bool8()}
	n := int(r.u())
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		s.KeyframesUS = append(s.KeyframesUS, last)
	}
	n = int(r.u())
	for k := 0; k < n && r.err == nil; k++ {
		s.Entities = append(s.Entities, grammar.PlayerEntity{
			Slot: int(r.u()), Index: int(r.i()), Team: int(r.i()), FirstKF: int(r.u()),
			LastKF: int(r.u()), Seen: int(r.u()), Unstable: r.bool8(),
		})
	}
	return s
}

// decodeFilmTable relit la TABLE DES JOUEURS DU FILM (v20, lot 1.6).
func decodeFilmTable(r *greader) FilmPlayerTable {
	t := FilmPlayerTable{Build: r.str(), Refusal: FilmTableRefusal(r.str())}
	t.Occupied, t.Vacant = int(r.u()), int(r.u())
	t.InterleavedVacant = r.bool8()
	n := int(r.u())
	if n == 0 {
		return t
	}
	t.Seats = make([]FilmPlayerSeat, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		t.Seats = append(t.Seats, FilmPlayerSeat{
			FilmIndex: int(r.i()), XUID: r.u(), Gamertag: r.str()})
	}
	return t
}
