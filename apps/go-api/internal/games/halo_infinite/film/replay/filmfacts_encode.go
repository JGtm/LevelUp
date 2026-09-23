package replay

// filmfacts_encode.go — L ENCODEUR DES FAITS DE FILM.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7 : un fichier de
// 1 560 lignes, deux fonctions de ~290), passe en PRODUCTION le 2026-09-17 (lot 4.1.1-a).
// DEPLACEMENTS PURS : aucune ligne de logique changee, le decoupage suit les SECTIONS du blob,
// et chaque section a sa fonction.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

const (
	gpHasWorld  byte = 1 << 0
	gpHasYaw    byte = 1 << 1
	gpHasBody   byte = 1 << 2
	gpHasShield byte = 1 << 3
)

// EncodeFilmFacts serialise les faits d un film. Le format est decrit en tete de filmfacts.go ;
// la SUITE DES SECTIONS est celle que `DecodeFilmFacts` relit, dans le meme ordre, et toute
// insertion au milieu monte `filmFactsMagic` DANS LE MEME COMMIT.
// EncodeFilmFactsAvecErreur est [EncodeFilmFacts] qui REND SON ECHEC.
//
// Le codec est sans erreur sur tout ce qu il ecrit a la main ; une seule charge peut echouer (les
// morts d objet, en JSON), et un echec avale produirait un fichier qui se relit comme un fait
// FAUX. Les appelants qui ECRIVENT sur le disque passent par ici ; [EncodeFilmFacts] reste la
// forme sans erreur pour les tests et les aller-retours en memoire.
func EncodeFilmFactsAvecErreur(g *FilmFacts) ([]byte, error) {
	w := encodeurDeFaits(g)
	return w.b, w.echec
}

func EncodeFilmFacts(g *FilmFacts) []byte {
	return encodeurDeFaits(g).b
}

func encodeurDeFaits(g *FilmFacts) *gwriter {
	w := &gwriter{b: []byte(filmFactsMagic)}
	encodeEntete(w, g)
	encodePositionSection(w, g.Positions)
	encodeBipedCreations(w, g.BipedCreations)
	encodeEvenements(w, g)
	encodeMarcheImageCle(w, g.KeyframeWalk)
	encodeNaissances(w, g.BirthLoadouts, g.BirthLoadoutStats)
	encodeWeaponChanges(w, g.WeaponChanges)
	encodePickups(w, g.Pickups, g.PickupStats)
	encodeInventaire(w, g)
	encodeCanauxDelta(w, g)
	encodeEquipmentChanges(w, g.EquipmentChanges, g.EquipmentChangeStats)
	encodeCapacites(w, g)
	encodeEtatsDeMouvement(w, g)
	encodeZoomEvents(w, g.ZoomEvents)
	encodeMonde(w, g)
	encodeVehicleScan(w, g.Vehicles)
	encodeQueue(w, g)
	return w
}

// encodeEntete ecrit l en-tete du blob.
func encodeEntete(w *gwriter, g *FilmFacts) {
	w.str(g.Film)
	// LE MODULE DE LA CARTE OUVRE LE BLOB (lot 0.D.3 bis). Les positions y sont des QUANTA :
	// sans l entree de catalogue qui les a produites, elles ne se dequantifient pas — et avec
	// la MAUVAISE, elles se dequantifient en coordonnees FAUSSES, pas approximatives
	// (cf. DequantBipedAxis). Le module est donc ecrit ici et VERIFIE a la relecture.
	w.str(g.MapModule)
	for a := 0; a < 3; a++ {
		w.u(uint64(g.AxisW[a]))
	}
	w.bool8(g.LayoutDetected)
	// LES STATS DE BALAYAGE QUE L ASSEMBLAGE CONSOMME (revue R1, constat R1-1). Elles ne sont
	// pas des donnees mais des VERDICTS du decodeur, et le document les publie : sans elles le
	// golden figeait une constante (« canal munitions refuse : false », version de film absente)
	// et la fidelite etait aveugle DES DEUX COTES.
	w.bool8(g.InventoryDeltaAmmoRefused)
	w.bool8(g.FilmMajorVersion != nil)
	if g.FilmMajorVersion != nil {
		w.i(int64(*g.FilmMajorVersion))
	}
	w.u(g.FilmClockOriginUS)
}

// encodeEvenements ecrit tirs, equipements de depart, lancers et projectiles.
func encodeEvenements(w *gwriter, g *FilmFacts) {
	var lastTS uint64
	w.u(uint64(len(g.Fire)))
	lastTS = 0
	for _, e := range g.Fire {
		w.u(e.TimestampUS - lastTS)
		lastTS = e.TimestampUS
		w.i(int64(e.FilmIndex))
		w.u(e.WeaponID)
		w.bool8(e.HasAim)
		if e.HasAim {
			for a := 0; a < 3; a++ {
				w.f32(e.Aim[a])
			}
		}
	}

	w.u(uint64(len(g.Loadouts)))
	for _, l := range g.Loadouts {
		w.u(l.TimestampUS)
		w.u(uint64(l.Slot))
		w.u(uint64(len(l.Families)))
		for _, f := range l.Families {
			w.u(uint64(f))
		}
	}

	w.u(uint64(len(g.Grenades)))
	for _, t := range g.Grenades {
		w.u(t.TimestampUS)
		w.i(int64(t.FilmIndex))
		w.u(uint64(t.TypeID))
	}

	encodeTracks(w, g.Projectiles)
}

// encodeInventaire ecrit les inventaires d image-cle et leurs deltas.
func encodeInventaire(w *gwriter, g *FilmFacts) {
	var lastTS uint64
	w.u(uint64(len(g.Inventory)))
	for _, inv := range g.Inventory {
		w.u(inv.TimestampUS)
		w.u(uint64(inv.Slot))
		w.bool8(inv.GrenadesRead)
		for _, c := range inv.Grenades {
			w.u(uint64(c))
		}
		w.i(int64(inv.SelectedGrenadeRank))
		w.i(int64(inv.AbilityRank))
		w.i(int64(inv.DrawnSlot))
		w.u(uint64(inv.AmmoCandidates))
		w.bool8(inv.AmmoRead)
		for _, a := range inv.Ammo {
			encodeAmmo(w, a)
		}
	}

	w.u(uint64(len(g.InventoryDeltas)))
	lastTS = 0
	for _, d := range g.InventoryDeltas {
		w.u(d.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = d.TimestampUS
		w.u(uint64(d.Slot))
		w.u(uint64(len(d.Grenades)))
		for _, c := range d.Grenades {
			w.u(uint64(c))
		}
		w.bool8(d.SelRead)
		w.i(int64(d.Sel))
		w.u(uint64(d.Mask))
	}

}

// encodeCanauxDelta ecrit rangs de capacite, camouflage, grappin et translocations.
func encodeCanauxDelta(w *gwriter, g *FilmFacts) {
	var lastTS uint64
	w.u(uint64(len(g.AbilityRanks)))
	lastTS = 0
	for _, a := range g.AbilityRanks {
		w.u(a.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = a.TimestampUS
		w.u(uint64(a.Slot))
		w.i(int64(a.Rank))
	}

	w.u(uint64(len(g.CamoStates)))
	lastTS = 0
	for _, cr := range g.CamoStates {
		w.u(cr.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = cr.TimestampUS
		w.u(uint64(cr.Slot))
		w.u(uint64(cr.Q))
	}

	w.u(uint64(len(g.GrappleReads)))
	lastTS = 0
	for _, gr := range g.GrappleReads {
		w.u(gr.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = gr.TimestampUS
		w.u(uint64(gr.Slot))
		w.bool8(gr.Heavy)
		for a := 0; a < 3; a++ {
			w.u(uint64(gr.PosQ[a]))
		}
	}

	w.u(uint64(len(g.Translocations)))
	lastTS = 0
	for _, tr := range g.Translocations {
		w.u(tr.TimestampUS - lastTS) // le scan rend les evenements tries par instant
		lastTS = tr.TimestampUS
		w.u(uint64(tr.Slot))
		// LE VA-ET-VIENT VOYAGE AVEC SON TEMOIN (v12) : sans lui, un saut sans position
		// serait indistinguable d un saut vers l origine du monde.
		w.bool8(tr.HasPositions)
		for a := 0; a < 3; a++ {
			w.f32(tr.From[a])
		}
		for a := 0; a < 3; a++ {
			w.f32(tr.To[a])
		}
	}

	// LES IMPULSIONS DE CAPACITE (v13) : le scan les rend TRIEES par instant, d ou le delta.
	// Les STATS suivent la liste — c est le temoin `Absent` qui distingue « ce film ne
	// transmet pas le composant » de « personne ne s en est servi ».
}

// encodeCapacites ecrit les impulsions et les charges de capacite, stats comprises.
func encodeCapacites(w *gwriter, g *FilmFacts) {
	var lastTS uint64
	w.u(uint64(len(g.AbilityImpulses)))
	lastTS = 0
	for _, im := range g.AbilityImpulses {
		w.u(im.TimestampUS - lastTS)
		lastTS = im.TimestampUS
		w.u(uint64(im.Slot))
		w.bool8(im.Predicted)
	}
	w.u(uint64(g.AbilityImpulseStats.Records))
	w.u(uint64(g.AbilityImpulseStats.WithI57))
	w.u(uint64(g.AbilityImpulseStats.WithI59))
	w.u(uint64(g.AbilityImpulseStats.Read))
	w.u(uint64(g.AbilityImpulseStats.Unread))
	w.u(uint64(g.AbilityImpulseStats.Tag1))
	w.bool8(g.AbilityImpulseStats.Absent)
	// `Scanned` VOYAGE AVEC LES AUTRES : sans lui, un fixture rendrait une couverture de zeros
	// indistinguable d un balayage qui n a jamais tourne (constat H1 de la revue de ronde 1).
	w.bool8(g.AbilityImpulseStats.Scanned)

	// LES CHARGES RESTANTES (v14) : le scan les rend TRIEES par instant, d ou le delta. Les
	// STATS suivent la liste, `Absent` et `Scanned` compris — memes temoins, memes raisons
	// que les impulsions ci-dessus.
	w.u(uint64(len(g.AbilityCharges)))
	lastTS = 0
	for _, ac := range g.AbilityCharges {
		w.u(ac.TimestampUS - lastTS)
		lastTS = ac.TimestampUS
		w.u(uint64(ac.Slot))
		w.u(uint64(ac.Emplacement))
		w.u(uint64(ac.Charges))
		w.u(uint64(ac.Low))
	}
	w.u(uint64(g.AbilityChargeStats.Records))
	w.u(uint64(g.AbilityChargeStats.WithI56))
	w.u(uint64(g.AbilityChargeStats.Read))
	w.u(uint64(g.AbilityChargeStats.Unread))
	w.u(uint64(g.AbilityChargeStats.Armed))
	w.bool8(g.AbilityChargeStats.Absent)
	w.bool8(g.AbilityChargeStats.Scanned)

	// Les POSES, puis la CALIBRATION qui les rend lisibles. Les deux vont ensemble : une
	// liste vide ne dit pas la meme chose selon que le film a tranche sa largeur ou non.
}

// encodeEtatsDeMouvement ecrit les ETATS DE MOUVEMENT (v24) : la liste est rendue TRIEE par
// instant, d ou le delta d horodatage. Les STATS suivent — c est `Absent` qui distingue « ce film
// ne transmet pas les composants » de « personne ne s est accroupi », et `MapWidths` qui dit sous
// quelles largeurs la marche a lu.
func encodeEtatsDeMouvement(w *gwriter, g *FilmFacts) {
	w.u(uint64(len(g.MovementStates)))
	var lastTS uint64
	for _, r := range g.MovementStates {
		w.u(r.TimestampUS - lastTS)
		lastTS = r.TimestampUS
		w.u(uint64(r.Slot))
		w.str(r.Kind)
		w.bool8(r.On)
		w.u(uint64(r.Progress))
		w.u(uint64(r.Chunk))
		w.u(uint64(r.PacketIndex))
	}
	st := g.MovementStateStats
	w.u(uint64(st.Records))
	w.u(uint64(st.Read))
	w.bool8(st.Absent)
	w.bool8(st.Scanned)
	w.u(uint64(st.Packets))
	w.u(uint64(st.EventPackets))
	w.u(uint64(st.EventPacketsLocated))
	w.u(uint64(st.EventPacketsUnlocated))
	w.u(uint64(st.Desyncs))
	w.u(uint64(st.SlotUnbound))
	w.u(uint64(st.Duplicates))
	for _, x := range st.MapWidths {
		w.u(uint64(x))
	}
}

// encodeMonde ecrit les poses d equipement et les deux voies de socles.
func encodeMonde(w *gwriter, g *FilmFacts) {
	var lastTS uint64
	w.u(uint64(len(g.Placements)))
	lastTS = 0
	for _, p := range g.Placements {
		w.u(p.T0US - lastTS) // les poses sont triees par instant de creation
		lastTS = p.T0US
		w.u(p.T1US)
		w.u(uint64(p.Life.Slot))
		w.u(uint64(p.Life.Gen))
		w.f32(p.X)
		w.f32(p.Y)
		w.f32(p.Z)
		w.u(uint64(p.GlobalID))
		w.u(uint64(p.Points))
	}
	encodeStatsDePose(w, g.PlacementStats)

	encodeSpawnEvents(w, g)

	encodeWorldObjectScan(w, g.Pads.Weapons)
	encodeWorldObjectScan(w, g.Pads.Powerups)
}

// encodeSpawnEvents ecrit les evenements 103 « une PIECE a ete engendree » et les
// denominateurs de leur balayage (lot 1.9.1, magie REPLAYINPUTS22).
//
// LES DENOMINATEURS SONT ECRITS AUTANT QUE LES EVENEMENTS, et ce n est pas du remplissage : un
// fixture a ZERO evenement doit pouvoir dire s il vient d un film muet ou d un film que le
// lecteur ne sait pas lire — c est exactement l ecart entre `a521164d` (0 sur 4 956 listes) et
// un film sans mur.
func encodeSpawnEvents(w *gwriter, g *FilmFacts) {
	w.u(uint64(len(g.SpawnEvents)))
	var lastTS uint64
	for _, e := range g.SpawnEvents {
		w.u(e.TimestampUS - lastTS) // le balayage rend les evenements tries par instant
		lastTS = e.TimestampUS
		w.i(int64(e.Chunk))
		w.i(int64(e.PacketIndex))
		w.u(uint64(e.Spawned.Slot))
		w.u(uint64(e.Spawned.Gen))
		w.bool8(e.SpawnedValid)
		w.u(uint64(e.Source.Slot))
		w.u(uint64(e.Source.Gen))
		w.bool8(e.SourceValid)
		w.bool8(e.Ref2Present)
	}
	for _, v := range []int{
		g.SpawnStats.Chunks, g.SpawnStats.Packets, g.SpawnStats.Lists, g.SpawnStats.Events,
		g.SpawnStats.WithSpawned, g.SpawnStats.WithSource, g.SpawnStats.Ref2,
	} {
		w.u(uint64(v))
	}
}

// encodeQueue ecrit les morts et la table des index de joueur.
func encodeQueue(w *gwriter, g *FilmFacts) {

	w.u(uint64(len(g.Deaths)))
	for _, d := range g.Deaths {
		w.u(d.XUID)
		w.str(d.Gamertag)
		w.i(d.TimeMS)
	}

	w.u(uint64(g.PlayerIndices.Readings))
	w.u(uint64(g.PlayerIndices.Disagreements))
	xuids := make([]uint64, 0, len(g.PlayerIndices.ByXUID))
	for x := range g.PlayerIndices.ByXUID {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	w.u(uint64(len(xuids)))
	for _, x := range xuids {
		w.u(x)
		w.i(int64(g.PlayerIndices.ByXUID[x]))
	}

	encodeFilmTable(w, g.FilmTable)
	encodePlayerTeams(w, g.PlayerTeams, g.TeamScan)
}

// encodePlayerTeams ecrit l EQUIPE DE CHAQUE JOUEUR (v21, lot 1.7) : la table `index de joueur
// -> designateur` et le RAPPORT de sa lecture. Les deux, parce qu une table vide et une lecture
// refusee ne disent pas la meme chose, et que `coverage.teams` publie la difference.
func encodePlayerTeams(w *gwriter, teams map[int]int, rep grammar.TeamScanReport) {
	idx := make([]int, 0, len(teams))
	for i := range teams {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	w.u(uint64(len(idx)))
	for _, i := range idx {
		w.i(int64(i))
		w.i(int64(teams[i]))
	}
	w.bool8(rep.ArchetypeAbsent)
	w.bool8(rep.ComponentMismatch)
	w.str(rep.Component)
	for _, n := range []int{rep.Packets, rep.Records, rep.Read, rep.Unreached,
		rep.OutOfDomainIndex, rep.OutOfDomainValue, rep.Entities, rep.EntityDivergences,
		rep.IndexDivergences, rep.Indices, rep.NoTeam} {
		w.u(uint64(n))
	}
}

// encodeFilmTable ecrit la TABLE DES JOUEURS DU FILM (v20, lot 1.6). Elle porte son REFUS comme
// elle porte ses sieges : une table non lue n'est pas une table vide, et le document publie la
// difference (`coverage.identity.filmTable.refus`).
func encodeFilmTable(w *gwriter, t FilmPlayerTable) {
	w.str(t.Build)
	w.str(string(t.Refusal))
	w.u(uint64(t.Occupied))
	w.u(uint64(t.Vacant))
	w.bool8(t.InterleavedVacant)
	w.u(uint64(len(t.Seats)))
	for _, s := range t.Seats {
		w.i(int64(s.FilmIndex))
		w.u(s.XUID)
		w.str(s.Gamertag)
	}
}
