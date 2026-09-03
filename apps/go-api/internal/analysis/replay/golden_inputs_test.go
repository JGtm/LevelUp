package replay

// golden_inputs_test.go — LES ENTREES DECODEES DU FILM DE REFERENCE, FIGEES.
//
// POURQUOI CE FICHIER EXISTE. `BuildFromPositions` est deja PUR — il ne lui manquait que ses
// entrees. Tant qu elles ne vivaient que dans un film de 20,2 Mo hors depot, les chiffres du
// chantier (475/519 tirs, 90/105 vies nommees, 70 lancers, 439 projectiles, 184 etats
// d inventaire) n etaient verrouilles par AUCUN test : ils vivaient dans des Markdown et dans
// un artefact genere. Un refactor de l assemblage pouvait les deplacer en silence.
//
// CE QUE CE FIXTURE EST, ET CE QU IL N EST PAS. C est un ETAGE 1 : les entrees DEJA DECODEES,
// serialisees, rejouees dans l assemblage pur. Il verrouille l ASSEMBLAGE et rien d autre — un
// changement du DECODAGE ne le fait pas bouger, c est le travail de l etage 2 (la mini-bobine,
// minifilm_test.go). Les deux etages sont complementaires et ne se remplacent pas.
//
// ZERO OCTET DE FILM. Les tests de golden_assembly_test.go ne lisent que ce fichier fige. Le
// film n intervient qu a la REGENERATION, ci-dessous, qui est la seule porte d ecriture.
//
// REGENERATION (la seule ; un golden ne s edite JAMAIS a la main) :
//
//	REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run GoldenInputs -update
//
// puis, la sortie figee elle-meme :
//
//	go test ./internal/analysis/replay/ -run GoldenAssembly -update
//
// CE QUE LE FIXTURE PORTE, ET POURQUOI PAS PLUS. Il porte exactement les champs que
// l assemblage CONSOMME (cf. la liste par type ci-dessous). Il ne porte NI la geometrie NI la
// structure de carte : celles-ci ne viennent pas du film mais de catalogues versionnes a part,
// et les inclure ferait grossir le fixture d un ordre de grandeur pour verrouiller un chargement
// de fichier, pas un decodage.
//
// LE FORMAT EST DELTA-CODE, ET L ORDRE D ORIGINE EST PRESERVE. `BuildFromPositions` trie ses
// positions par `sort.SliceStable` et numerote les traces dans l ORDRE DE PREMIERE APPARITION
// des slots : reordonner les positions changerait la sortie. Le codec conserve donc la suite
// telle que le decodeur l a produite, et ne delta-code que les VALEURS (horodatage global,
// coordonnees par slot).

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

var updateGolden = flag.Bool("update", false, "reecrire les fichiers figes de testdata/")

// goldenFilm est le film de reference du chantier : Cliffhanger, Fiesta, 8 joueurs.
const goldenFilm = "000d5950"

// goldenDir est le repertoire des sorties figees.
const goldenDir = "testdata"

// goldenInputsPath est le fixture d entrees, versionne.
func goldenInputsPath() string {
	return filepath.Join(goldenDir, "inputs_"+goldenFilm+".bin.gz")
}

// goldenInputsMagic identifie le format et sa version. Un fixture d une autre version est une
// ERREUR, jamais une lecture « au mieux » : un decodage decale rendrait des chiffres plausibles.
//
// LA MAGIE S INCREMENTE DANS LE MEME COMMIT QUE LA SUITE DES SECTIONS — c est la seule chose
// qui rend la garde utile. Laisser la magie en arriere (constat de revue du 2026-08-25 : le
// commentaire annoncait v10, la constante disait encore v9) fait ACCEPTER un fixture d une
// autre version par la garde, et le decodage decale meurt plus loin sur un « uvarint illisible »
// a un offset arbitraire : bruyant par chance, pas par construction, et le message ne dit pas
// quoi faire. [TestGoldenInputsVersionGuard] verrouille le refus explicite.
//
// v11 (2026-09-03, lot P1 lecture fiable de l equipement, schema 38) : le fixture porte les
// TELEPORTATIONS du translocateur (evenements type 117) que BuildFromFilm decode desormais —
// et que le decodage des positions CONSOMME AUSSI (exemption du filtre de vitesse a ±200 ms,
// decision D2) : sans elles le fixture ne porterait ni le calque `translocations` ni les
// positions telles que la production les decode. Le film de reference (Fiesta Cliffhanger)
// peut n en porter aucune : la liste vide se serialise, et le zero se fige avec le reste.
//
// v10 (2026-08-25, lot 4.4 du suivi delta de l inventaire) : le fixture porte les lectures
// d INVENTAIRE DELTA (compteurs de grenades i22 et jeu selectionne i47) que BuildFromFilm
// decode desormais. Sans elles le golden d assemblage n exercerait JAMAIS le second canal de
// l axe `grenadeReads` — il figerait un document que la production ne produit plus.
//
// v4 (2026-08-16, PLAN_EQUIPEMENT_TI37 phase 1) : la position serialise AUSSI le QUANTUM
// brut du bouclier (Shield.Q — la regle du surbouclier est `q > 64`, et le clamp de
// ShieldFraction efface l information), et le fixture porte les lectures CamoStates (i28
// queue[1]) que BuildFromFilm decode desormais.
//
// v5 (2026-08-16, PLAN_GRAPPIN_LIGNE phase 1) : le fixture porte les lectures GrappleReads
// (corps tag==3 d i59 — tir et accroche de grappin, quanta de position aux largeurs de la
// carte) que BuildFromFilm decode desormais.
//
// v6 (2026-08-18, PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) : le fixture porte les POSES
// d equipement (records de creation ti=37 confirmes par l oracle de position) ET la
// CALIBRATION du bloc de replication mesuree sur ce film. La calibration en fait partie parce
// que l assemblage la PUBLIE : sans elle, une liste vide de poses serait indistinguable d un
// film sans equipement, alors que ce peut etre un film dont la largeur n a pas ete tranchee.
//
// v7 (2026-08-17, PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : le fixture porte les ARMES AU SOL —
// records de creation ti=42 (position i0 et identite MPP), RECENSEMENT des images-cles qui
// borne les disparitions, et pistes de position qui disent si l objet a bouge. LES TROIS
// ENSEMBLE, parce qu il en manque une et le calque ment : sans les pistes, toute apparition
// passerait pour un objet apparu au repos, donc pour un socle.
//
// v8 (2026-08-18, PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1) : la position serialise AUSSI
// le SECOND scalaire d i21 (`PitchRaw`, l elevation de visee), que l assemblage publie
// desormais dans `Point.p`. Sans lui le fixture ne porterait qu un des deux angles du meme
// composant, et le golden verrouillerait un document dont toutes les visees sont a plat —
// c est-a-dire pas celui que la production sert.
//
// v9 (2026-08-19, PLAN_POWERUP_SOCLE_CATALYST phase 8) : le fixture porte la SECONDE voie de la
// chaine des socles — les creations `ti=37` avec leur recensement d images-cles et leurs pistes
// delta, d ou sortent les socles de POWER-UP. Elle est serialisee par le MEME codec que la voie
// des armes (une seule forme, `WorldObjectScan`), a la suite, et non a sa place : les deux
// entrent ensemble dans l assemblage.
const goldenInputsMagic = "REPLAYINPUTS11\n"

// goldenInputs porte les entrees de BuildFromPositions decodees du film de reference.
//
// LES CHAMPS SERIALISES, PAR TYPE — ce sont ceux que l assemblage consomme :
//
//	BipedPosition     Slot · TimestampUS · X/Y/Z · HasWorld · HasYaw+YawRaw+PitchRaw ·
//	                  HasBody+Body.Health · HasShield+Shield.Shield+Shield.Q
//	FireEvent         TimestampUS · FilmIndex · WeaponID · HasAim+Aim
//	KeyframeLoadout   TimestampUS · Slot · Families
//	GrenadeThrow      TimestampUS · FilmIndex · TypeID
//	ProjectileTrack   Slot · Gen · Pts(TimestampUS · X/Y/Z · AtRest)
//	KeyframeInventory tout, sauf Chunk/PacketIndex (traçabilite dans le film, pas une entree)
//	AbilityRank       Slot · TimestampUS · Rank (le compteur de rotation n entre pas dans
//	                  l assemblage : il borne une lecture, il ne se publie pas)
//	CamoRead          Slot · TimestampUS · Q (la voie queue[1] d i28 — l interrupteur)
//	GrappleRead       Slot · TimestampUS · Heavy · PosQ (les quanta de l ancre, aux
//	                  largeurs d axe de la carte)
//	Death             XUID · Gamertag · TimeMS
//	PlayerIndexTable  entier
//	ClockOriginUS     l horodatage du premier paquet du film (l origine publiee en depend)
type goldenInputs struct {
	Film        string
	Positions   []filmdec.BipedPosition
	Fire        []filmdec.FireEvent
	Loadouts    []filmdec.KeyframeLoadout
	Grenades    []filmdec.GrenadeThrow
	Projectiles []filmdec.ProjectileTrack
	Inventory   []KeyframeInventory
	// AbilityRanks : les identites de capacite lues dans les paquets delta (i48). Elles sont
	// DANS le fixture parce que l assemblage les consomme — sans elles, le golden verrouillerait
	// un document dont les capacites se limitent a la fenetre 16..23 des images-cles.
	AbilityRanks []filmdec.AbilityRank
	// CamoStates : les transmissions de la voie d etat du camouflage (i28 queue[1]). MEME
	// raison : l assemblage en fait les episodes d equipement — sans elles le golden
	// verrouillerait un document sans camo, donc pas celui que la production sert. Le film
	// de reference (Fiesta) en porte 698, strictement binaires (0:617 · 4095:81) : le
	// DASH du mode Fiesta allume le canal, PAS un power-up ramasse — ce mode ne pose
	// aucun equipement au sol (enseignement utilisateur du 2026-08-16, cf. la
	// distribution des durees verrouillee par camo_duration_distribution_test.go). i28
	// est l etat de l unite, pas celui du seul equipement rang 8 (controle du
	// 2026-08-16, cf. renderEquipment).
	CamoStates []filmdec.CamoRead
	// InventoryDeltas : les lectures d inventaire des paquets DELTA (grenades). Second canal de
	// l axe `grenadeReads` — cf. grenade_reads.go.
	InventoryDeltas []filmdec.InventoryDelta
	// GrappleReads : les evenements de grappin (corps tag==3 d i59, tir et accroche avec
	// leurs quanta d ancre). MEME raison : l assemblage en fait les tractions du schema 8 —
	// sans elles le golden verrouillerait un document sans grappin, donc pas celui que la
	// production sert.
	GrappleReads []filmdec.GrappleRead
	// Translocations : les teleportations du translocateur (evenements type 117). MEME
	// raison : l assemblage en fait le calque du schema 38 — et la production les passe
	// AUSSI au filtre de vitesse (exemption D2), ce que decodeFilmInputs rejoue.
	Translocations []filmdec.TranslocatorTeleport
	// Placements / PlacementStats : les POSES d equipement et la CALIBRATION du bloc de
	// replication. MEME raison que les deux precedents : l assemblage en fait le calque du
	// schema 9. La calibration voyage avec la liste parce que la couverture la publie.
	Placements     []filmdec.EquipmentPlacement
	PlacementStats filmdec.EquipmentPlacementStats
	// Pads : ce que le film rend sur les SOCLES — la voie des ARMES AU SOL (creations ti=42) et
	// celle des POWER-UPS (creations ti=37), chacune avec le recensement des images-cles qui
	// borne les presences et les pistes de position qui disent si l objet a bouge. MEME raison
	// que les precedents : l assemblage en fait le calque des socles (schemas 11 puis 17).
	//
	// LES DEUX VOIES SONT DANS LE FIXTURE, et il le faut : c est la SECONDE qui decide si un
	// power-up de socle est publie. Sans elle, le golden verrouillerait un document que la
	// production ne sert plus.
	Pads    PadScans
	Deaths  []Death
	Indices PlayerIndexTable
	// ClockOriginUS est l horodatage moteur du premier paquet du film, c est-a-dire le zero de
	// l horloge des highlight events (cf. origin.go). Il est DANS le fixture parce que
	// l origine publiee est une entree de l assemblage comme une autre — sans lui, le golden
	// verrouillerait un document sans origine, donc pas celui que la production sert.
	ClockOriginUS uint64
}

// options rend les Options d assemblage portees par le fixture. La geometrie et la structure
// sont volontairement absentes (cf. l en-tete).
func (g *goldenInputs) options() Options {
	return Options{
		Loadouts:          g.Loadouts,
		Grenades:          g.Grenades,
		Projectiles:       g.Projectiles,
		Inventory:         g.Inventory,
		AbilityRanks:      g.AbilityRanks,
		CamoStates:        g.CamoStates,
		InventoryDeltas:   g.InventoryDeltas,
		GrappleReads:      g.GrappleReads,
		Translocations:    g.Translocations,
		Placements:        g.Placements,
		PlacementStats:    g.PlacementStats,
		Pads:              g.Pads,
		Deaths:            g.Deaths,
		PlayerIndices:     g.Indices,
		FilmClockOriginUS: g.ClockOriginUS,
	}
}

// ---------------------------------------------------------------------------
// Encodeur / decodeur
// ---------------------------------------------------------------------------

// cmScale convertit une coordonnee en CENTIMETRES ENTIERS.
//
// CE N EST PAS UNE PERTE, ET C EST MESURABLE : toute coordonnee publiee par l assemblage passe
// par `round2` (arrondi au centieme), sans exception — traces, tirs, lancers, projectiles. Le
// centimetre entier est donc exactement la precision que la sortie porte. Coder un float32 brut
// couterait 12 octets par position pour une decimale que personne ne lit.
const cmScale = 100

// gwriter accumule un flux binaire. Les entiers sont en varint : les deltas d horodatage et de
// position tiennent sur un a deux octets, ce qui fait tout le poids du fixture.
type gwriter struct{ b []byte }

func (w *gwriter) u(v uint64)   { w.b = binary.AppendUvarint(w.b, v) }
func (w *gwriter) i(v int64)    { w.b = binary.AppendVarint(w.b, v) }
func (w *gwriter) byte8(v byte) { w.b = append(w.b, v) }
func (w *gwriter) f32(v float32) {
	w.b = binary.LittleEndian.AppendUint32(w.b, math.Float32bits(v))
}
func (w *gwriter) str(s string) {
	w.u(uint64(len(s)))
	w.b = append(w.b, s...)
}
func (w *gwriter) bool8(v bool) {
	if v {
		w.byte8(1)
		return
	}
	w.byte8(0)
}

// greader relit le flux. Toute incoherence est une ERREUR remontee, jamais une valeur nulle
// servie en silence.
type greader struct {
	b   []byte
	off int
	err error
}

func (r *greader) u() uint64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Uvarint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("uvarint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) i() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("varint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) byte8() byte {
	if r.err != nil {
		return 0
	}
	if r.off >= len(r.b) {
		r.err = fmt.Errorf("fin de flux prematuree a l offset %d", r.off)
		return 0
	}
	v := r.b[r.off]
	r.off++
	return v
}

func (r *greader) f32() float32 {
	if r.err != nil {
		return 0
	}
	if r.off+4 > len(r.b) {
		r.err = fmt.Errorf("float32 tronque a l offset %d", r.off)
		return 0
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.b[r.off:]))
	r.off += 4
	return v
}

func (r *greader) str() string {
	n := int(r.u())
	if r.err != nil {
		return ""
	}
	if r.off+n > len(r.b) {
		r.err = fmt.Errorf("chaine tronquee a l offset %d", r.off)
		return ""
	}
	s := string(r.b[r.off : r.off+n])
	r.off += n
	return s
}

func (r *greader) bool8() bool { return r.byte8() == 1 }

// Drapeaux de presence d une position, sur un octet.
const (
	gpHasWorld  byte = 1 << 0
	gpHasYaw    byte = 1 << 1
	gpHasBody   byte = 1 << 2
	gpHasShield byte = 1 << 3
)

// encodeGoldenInputs serialise les entrees. Format decrit en tete de fichier.
func encodeGoldenInputs(g *goldenInputs) []byte {
	w := &gwriter{b: []byte(goldenInputsMagic)}
	w.str(g.Film)
	w.u(g.ClockOriginUS)

	// Table des slots : un slot tient sur 13 bits, mais un film n en emploie qu une centaine.
	// L indirection ramene 2 octets a 1 sur chaque position.
	slotIdx := map[uint32]int{}
	var slots []uint32
	for _, p := range g.Positions {
		if _, ok := slotIdx[p.Slot]; !ok {
			slotIdx[p.Slot] = len(slots)
			slots = append(slots, p.Slot)
		}
	}
	w.u(uint64(len(slots)))
	for _, s := range slots {
		w.u(uint64(s))
	}

	w.u(uint64(len(g.Positions)))
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for _, p := range g.Positions {
		w.u(p.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = p.TimestampUS
		w.u(uint64(slotIdx[p.Slot]))
		var fl byte
		if p.HasWorld {
			fl |= gpHasWorld
		}
		if p.HasYaw {
			fl |= gpHasYaw
		}
		if p.HasBody {
			fl |= gpHasBody
		}
		if p.HasShield {
			fl |= gpHasShield
		}
		w.byte8(fl)
		if p.HasWorld {
			cur := [3]int64{cmOf(p.X), cmOf(p.Y), cmOf(p.Z)}
			prev := lastXYZ[p.Slot]
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			lastXYZ[p.Slot] = cur
		}
		if p.HasYaw {
			// LES DEUX ANGLES D I21, ensemble : le cap et l elevation viennent du MEME
			// composant et partagent leur validite. En serialiser un seul rendrait un
			// fixture ou toutes les visees sont a plat.
			w.u(uint64(p.YawRaw))
			w.u(uint64(p.PitchRaw))
		}
		if p.HasBody {
			w.f32(p.Body.Health)
		}
		if p.HasShield {
			w.f32(p.Shield.Shield)
			w.byte8(p.Shield.Q) // le QUANTUM : la regle du surbouclier (q > 64) le lit, pas la valeur clampee
		}
	}

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

	w.u(uint64(len(g.Inventory)))
	for _, inv := range g.Inventory {
		w.u(inv.TimestampUS)
		w.u(uint64(inv.Slot))
		w.bool8(inv.GrenadesRead)
		for _, c := range inv.Grenades {
			w.u(uint64(c))
		}
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
	}

	// Les POSES, puis la CALIBRATION qui les rend lisibles. Les deux vont ensemble : une
	// liste vide ne dit pas la meme chose selon que le film a tranche sa largeur ou non.
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
	w.i(int64(g.PlacementStats.Calibration.Widths.Lead))
	w.i(int64(g.PlacementStats.Calibration.Widths.Index))
	w.i(int64(g.PlacementStats.Calibration.Agree))
	w.u(uint64(g.PlacementStats.Lives))
	w.u(uint64(g.PlacementStats.Anchors))
	w.u(uint64(g.PlacementStats.Accepted))
	w.u(uint64(g.PlacementStats.Confirmed))

	encodeWorldObjectScan(w, g.Pads.Weapons)
	encodeWorldObjectScan(w, g.Pads.Powerups)

	w.u(uint64(len(g.Deaths)))
	for _, d := range g.Deaths {
		w.u(d.XUID)
		w.str(d.Gamertag)
		w.i(d.TimeMS)
	}

	w.u(uint64(g.Indices.Readings))
	w.u(uint64(g.Indices.Disagreements))
	xuids := make([]uint64, 0, len(g.Indices.ByXUID))
	for x := range g.Indices.ByXUID {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	w.u(uint64(len(xuids)))
	for _, x := range xuids {
		w.u(x)
		w.i(int64(g.Indices.ByXUID[x]))
	}
	return w.b
}

// encodeTracks / decodeTracks serialisent une liste de pistes d objet du monde (positions
// delta-codees au centimetre, comme tout le reste du fixture).
//
// ELLES SONT EXTRAITES PARCE QUE DEUX LISTES LES EMPRUNTENT : les trajectoires de projectile
// (schema 3) et les pistes d armes au sol (schema 11, qui disent si un objet a BOUGE). Une
// seconde copie du codec aurait diverge au premier champ ajoute, et un fixture qui se relit de
// travers rend des chiffres plausibles — c est le pire des defauts pour un golden.
func encodeTracks(w *gwriter, tracks []filmdec.ProjectileTrack) {
	w.u(uint64(len(tracks)))
	for _, tr := range tracks {
		w.u(uint64(tr.Slot))
		w.u(uint64(tr.Gen))
		w.u(uint64(len(tr.Pts)))
		var pts uint64
		var prev [3]int64
		for _, s := range tr.Pts {
			w.u(s.TimestampUS - pts)
			pts = s.TimestampUS
			cur := [3]int64{cmOf(s.X), cmOf(s.Y), cmOf(s.Z)}
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			prev = cur
			w.bool8(s.AtRest)
		}
	}
}

func decodeTracks(r *greader) []filmdec.ProjectileTrack {
	n := int(r.u())
	out := make([]filmdec.ProjectileTrack, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		tr := filmdec.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		var ts uint64
		var prev [3]int64
		for j := 0; j < np && r.err == nil; j++ {
			ts += r.u()
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			prev = cur
			tr.Pts = append(tr.Pts, filmdec.ProjectileSample{
				TimestampUS: ts, X: fromCM(cur[0]), Y: fromCM(cur[1]), Z: fromCM(cur[2]),
				AtRest: r.bool8(),
			})
		}
		out = append(out, tr)
	}
	return out
}

// encodeWorldObjectScan / decodeWorldObjectScan serialisent ce que le film rend sur UN archetype
// d objet du monde : les records de CREATION (position i0, instant, identite MPP), le
// RECENSEMENT des images-cles qui borne les disparitions, et les pistes de position qui disent
// si l objet a bouge.
//
// UN SEUL CODEC POUR LES DEUX VOIES (armes `ti=42`, power-ups `ti=37`) : elles ont la meme
// forme, et un second codec aurait diverge du premier au premier champ ajoute.
//
// LA BANDE DE SLOTS N EST PAS SERIALISEE, et c est deliberé : l assemblage ne la lit pas (elle
// sert au seul balayage, qui a deja eu lieu). Le fixture porte ce que l assemblage CONSOMME,
// pas ce que le decodage a traverse.
func encodeWorldObjectScan(w *gwriter, s WorldObjectScan) {
	w.bool8(s.Scanned)
	w.u(uint64(len(s.Creations)))
	var lastTS uint64
	for _, c := range s.Creations {
		w.u(c.TimestampUS - lastTS) // les creations sortent du balayage dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Gen))
		w.f32(c.X)
		w.f32(c.Y)
		w.f32(c.Z)
		w.bool8(c.MPPPresent[filmdec.MPPWord32])
		w.u(c.MPPVal[filmdec.MPPWord32])
	}
	w.u(uint64(s.Stats.Slots))
	w.u(uint64(s.Stats.Anchors))
	w.u(uint64(s.Stats.Accepted))
	w.u(uint64(len(s.Keyframes.TimesUS)))
	lastTS = 0
	for _, t := range s.Keyframes.TimesUS {
		w.u(t - lastTS)
		lastTS = t
	}
	// L ORDRE DES CLES EST RENDU TOTAL : une map Go s itere au hasard, et un fixture dont les
	// octets changent a chaque regeneration n est plus un fixture.
	keys := make([]filmdec.EquipmentLifeKey, 0, len(s.Keyframes.SeenUS))
	for k := range s.Keyframes.SeenUS {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Slot != keys[j].Slot {
			return keys[i].Slot < keys[j].Slot
		}
		return keys[i].Gen < keys[j].Gen
	})
	w.u(uint64(len(keys)))
	for _, k := range keys {
		w.u(uint64(k.Slot))
		w.u(uint64(k.Gen))
		seen := s.Keyframes.SeenUS[k]
		w.u(uint64(len(seen)))
		lastTS = 0
		for _, t := range seen {
			w.u(t - lastTS)
			lastTS = t
		}
	}
	encodeTracks(w, s.Tracks)
}

func decodeWorldObjectScan(r *greader) WorldObjectScan {
	s := WorldObjectScan{Scanned: r.bool8()}
	n := int(r.u())
	s.Creations = make([]filmdec.EquipmentCreation, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := filmdec.EquipmentCreation{TimestampUS: lastTS, Slot: uint32(r.u()), Gen: uint32(r.u())}
		c.X, c.Y, c.Z = r.f32(), r.f32(), r.f32()
		c.MPPPresent[filmdec.MPPWord32] = r.bool8()
		c.MPPVal[filmdec.MPPWord32] = r.u()
		s.Creations = append(s.Creations, c)
	}
	s.Stats.Slots, s.Stats.Anchors, s.Stats.Accepted = int(r.u()), int(r.u()), int(r.u())
	n = int(r.u())
	s.Keyframes.TimesUS = make([]uint64, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		s.Keyframes.TimesUS = append(s.Keyframes.TimesUS, lastTS)
	}
	n = int(r.u())
	s.Keyframes.SeenUS = make(map[filmdec.EquipmentLifeKey][]uint64, n)
	for k := 0; k < n && r.err == nil; k++ {
		key := filmdec.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		seen := make([]uint64, 0, np)
		lastTS = 0
		for j := 0; j < np && r.err == nil; j++ {
			lastTS += r.u()
			seen = append(seen, lastTS)
		}
		s.Keyframes.SeenUS[key] = seen
	}
	s.Tracks = decodeTracks(r)
	return s
}

// encodeAmmo serialise un emplacement de munitions. LES TROIS CAS SONT DISTINCTS (chargeur,
// jauge, rien) : un drapeau par pointeur, jamais un zero qui vaudrait absence.
func encodeAmmo(w *gwriter, a SlotAmmo) {
	w.bool8(a.Mag != nil)
	if a.Mag != nil {
		w.u(uint64(*a.Mag))
	}
	w.bool8(a.Res != nil)
	if a.Res != nil {
		w.u(uint64(*a.Res))
	}
	w.bool8(a.Gauge != nil)
	if a.Gauge != nil {
		w.b = binary.LittleEndian.AppendUint64(w.b, math.Float64bits(*a.Gauge))
	}
	w.u(uint64(a.Overheat))
	w.u(uint64(a.Flags))
}

func decodeAmmo(r *greader) SlotAmmo {
	var a SlotAmmo
	if r.bool8() {
		v := uint32(r.u())
		a.Mag = &v
	}
	if r.bool8() {
		v := uint32(r.u())
		a.Res = &v
	}
	if r.bool8() {
		if r.off+8 > len(r.b) {
			r.err = fmt.Errorf("jauge tronquee a l offset %d", r.off)
			return a
		}
		v := math.Float64frombits(binary.LittleEndian.Uint64(r.b[r.off:]))
		r.off += 8
		a.Gauge = &v
	}
	a.Overheat = uint32(r.u())
	a.Flags = uint32(r.u())
	return a
}

// decodeGoldenInputs relit le fixture.
func decodeGoldenInputs(blob []byte) (*goldenInputs, error) {
	if len(blob) < len(goldenInputsMagic) || string(blob[:len(goldenInputsMagic)]) != goldenInputsMagic {
		return nil, fmt.Errorf("fixture d entrees : magie absente ou version inconnue — regenerer")
	}
	r := &greader{b: blob, off: len(goldenInputsMagic)}
	g := &goldenInputs{Film: r.str()}
	g.ClockOriginUS = r.u()

	nSlots := int(r.u())
	slots := make([]uint32, 0, nSlots)
	for k := 0; k < nSlots && r.err == nil; k++ {
		slots = append(slots, uint32(r.u()))
	}

	n := int(r.u())
	g.Positions = make([]filmdec.BipedPosition, 0, n)
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for k := 0; k < n && r.err == nil; k++ {
		var p filmdec.BipedPosition
		lastTS += r.u()
		p.TimestampUS = lastTS
		si := int(r.u())
		if si >= len(slots) {
			return nil, fmt.Errorf("index de slot %d hors table (%d)", si, len(slots))
		}
		p.Slot = slots[si]
		fl := r.byte8()
		if fl&gpHasWorld != 0 {
			p.HasWorld = true
			prev := lastXYZ[p.Slot]
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			lastXYZ[p.Slot] = cur
			p.X, p.Y, p.Z = fromCM(cur[0]), fromCM(cur[1]), fromCM(cur[2])
		}
		if fl&gpHasYaw != 0 {
			p.HasYaw = true
			p.YawRaw = uint32(r.u())
			p.PitchRaw = uint32(r.u())
		}
		if fl&gpHasBody != 0 {
			p.HasBody = true
			p.Body.Health = r.f32()
		}
		if fl&gpHasShield != 0 {
			p.HasShield = true
			p.Shield.Shield = r.f32()
			p.Shield.Q = r.byte8()
		}
		g.Positions = append(g.Positions, p)
	}

	n = int(r.u())
	g.Fire = make([]filmdec.FireEvent, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		var e filmdec.FireEvent
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
	g.Loadouts = make([]filmdec.KeyframeLoadout, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		l := filmdec.KeyframeLoadout{TimestampUS: r.u(), Slot: uint32(r.u())}
		nf := int(r.u())
		for j := 0; j < nf && r.err == nil; j++ {
			l.Families = append(l.Families, uint32(r.u()))
		}
		g.Loadouts = append(g.Loadouts, l)
	}

	n = int(r.u())
	g.Grenades = make([]filmdec.GrenadeThrow, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Grenades = append(g.Grenades, filmdec.GrenadeThrow{
			TimestampUS: r.u(), FilmIndex: int(r.i()), TypeID: uint32(r.u()),
		})
	}

	g.Projectiles = decodeTracks(r)

	n = int(r.u())
	g.Inventory = make([]KeyframeInventory, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		inv := KeyframeInventory{TimestampUS: r.u(), Slot: uint32(r.u())}
		inv.GrenadesRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Grenades[j] = uint32(r.u())
		}
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
	g.InventoryDeltas = make([]filmdec.InventoryDelta, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		d := filmdec.InventoryDelta{TimestampUS: lastTS, Slot: uint32(r.u())}
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

	n = int(r.u())
	g.AbilityRanks = make([]filmdec.AbilityRank, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityRanks = append(g.AbilityRanks,
			filmdec.AbilityRank{TimestampUS: lastTS, Slot: uint32(r.u()), Rank: int(r.i())})
	}

	n = int(r.u())
	g.CamoStates = make([]filmdec.CamoRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.CamoStates = append(g.CamoStates,
			filmdec.CamoRead{TimestampUS: lastTS, Slot: uint32(r.u()), Q: uint16(r.u())})
	}

	n = int(r.u())
	g.GrappleReads = make([]filmdec.GrappleRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		gr := filmdec.GrappleRead{TimestampUS: lastTS, Slot: uint32(r.u()), Heavy: r.bool8()}
		for a := 0; a < 3; a++ {
			gr.PosQ[a] = uint32(r.u())
		}
		g.GrappleReads = append(g.GrappleReads, gr)
	}

	n = int(r.u())
	g.Translocations = make([]filmdec.TranslocatorTeleport, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.Translocations = append(g.Translocations,
			filmdec.TranslocatorTeleport{TimestampUS: lastTS, Slot: uint32(r.u())})
	}

	n = int(r.u())
	g.Placements = make([]filmdec.EquipmentPlacement, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		p := filmdec.EquipmentPlacement{T0US: lastTS, T1US: r.u()}
		p.Life = filmdec.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		p.X, p.Y, p.Z = r.f32(), r.f32(), r.f32()
		p.GlobalID, p.Points = uint32(r.u()), int(r.u())
		g.Placements = append(g.Placements, p)
	}
	g.PlacementStats = filmdec.EquipmentPlacementStats{ByID: map[uint32]int{}}
	g.PlacementStats.Calibration.Widths = filmdec.MPPWidths{Lead: int(r.i()), Index: int(r.i())}
	g.PlacementStats.Calibration.Agree = int(r.i())
	g.PlacementStats.Lives = int(r.u())
	g.PlacementStats.Anchors = int(r.u())
	g.PlacementStats.Accepted = int(r.u())
	g.PlacementStats.Confirmed = int(r.u())
	g.PlacementStats.Placements = len(g.Placements)

	g.Pads.Weapons = decodeWorldObjectScan(r)
	g.Pads.Powerups = decodeWorldObjectScan(r)

	n = int(r.u())
	g.Deaths = make([]Death, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Deaths = append(g.Deaths, Death{XUID: r.u(), Gamertag: r.str(), TimeMS: r.i()})
	}

	g.Indices = PlayerIndexTable{ByXUID: map[uint64]int{}}
	g.Indices.Readings = int(r.u())
	g.Indices.Disagreements = int(r.u())
	n = int(r.u())
	for k := 0; k < n && r.err == nil; k++ {
		x := r.u()
		g.Indices.ByXUID[x] = int(r.i())
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(r.b) {
		return nil, fmt.Errorf("fixture d entrees : %d octet(s) non consomme(s) — format desynchronise",
			len(r.b)-r.off)
	}
	return g, nil
}

func cmOf(v float32) int64 { return int64(math.Round(float64(v) * cmScale)) }

func fromCM(v int64) float32 { return float32(float64(v) / cmScale) }

// loadGoldenInputs relit le fixture versionne. AUCUN OCTET DE FILM.
func loadGoldenInputs(t *testing.T) *goldenInputs {
	t.Helper()
	raw, err := os.ReadFile(goldenInputsPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("fixture d entrees illisible : %v — regenerer avec "+
			"REPLAY_FILM_DIR=<film> go test -run GoldenInputs -update", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture d entrees : gzip illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture d entrees : decompression : %v", err)
	}
	g, err := decodeGoldenInputs(buf.Bytes())
	if err != nil {
		t.Fatalf("fixture d entrees : %v", err)
	}
	return g
}

// TestGoldenInputsRoundTrip : le codec ne perd rien de ce que l assemblage consomme.
//
// C EST LE FILET DU FIXTURE LUI-MEME. Un codec qui perdrait un champ produirait un golden
// parfaitement stable et parfaitement faux. On le verifie par la seule mesure qui compte :
// l assemblage rejoue sur les entrees RELUES doit rendre le MEME document que sur les entrees
// d origine — ici, les entrees d origine etant deja le fixture, on controle qu un second
// aller-retour est un point fixe, octet pour octet.
func TestGoldenInputsRoundTrip(t *testing.T) {
	g := loadGoldenInputs(t)
	blob := encodeGoldenInputs(g)
	again, err := decodeGoldenInputs(blob)
	if err != nil {
		t.Fatalf("second decodage : %v", err)
	}
	if got := encodeGoldenInputs(again); !bytes.Equal(blob, got) {
		t.Fatalf("le codec n est pas un point fixe : %d octets contre %d — un champ se perd "+
			"ou se reconstruit differemment a chaque tour", len(got), len(blob))
	}
	docA := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, g.options())
	docB := BuildFromPositions(goldenFilm, "halo_infinite", again.Positions, again.Fire, again.options())
	if renderAssembly(docA) != renderAssembly(docB) {
		t.Error("l assemblage differe entre les entrees relues et leur re-serialisation : " +
			"le codec perd un champ que BuildFromPositions consomme")
	}
}

// TestGoldenInputsVersionGuard : un fixture d une AUTRE version est refuse par la MAGIE, et le
// message dit quoi faire.
//
// C EST LE VERROU DU CONSTAT DE REVUE DU 2026-08-25. La section `InventoryDeltas` a ete inseree
// au milieu du flux sans que la magie bouge : un fixture de la version precedente passait la
// garde, puis mourait plus loin sur « uvarint illisible a l offset N » — un message qui parle
// d octets alors que le probleme est une version. Le test relit le corps COURANT precede de la
// magie PRECEDENTE : la seule reponse acceptable est le refus de version.
func TestGoldenInputsVersionGuard(t *testing.T) {
	const previousMagic = "REPLAYINPUTS10\n"
	if previousMagic == goldenInputsMagic {
		t.Fatal("la magie precedente et la courante sont identiques : le test ne prouve plus rien")
	}
	body := encodeGoldenInputs(loadGoldenInputs(t))[len(goldenInputsMagic):]
	stale := append([]byte(previousMagic), body...)
	_, err := decodeGoldenInputs(stale)
	if err == nil {
		t.Fatal("un fixture d une autre version a ete accepte : la garde de version ne sert a rien")
	}
	if !strings.Contains(err.Error(), "version inconnue") {
		t.Fatalf("la version doit etre refusee POUR CE QU ELLE EST ; message obtenu : %v", err)
	}
}

// TestGoldenInputsRegenerate : LA SEULE PORTE D ECRITURE DU FIXTURE.
//
// Elle exige DEUX conditions explicites — `-update` et `REPLAY_FILM_DIR` — parce qu une
// regeneration accidentelle transformerait n importe quelle regression en « nouvelle
// reference ». C est la meme discipline que le paquet killsource.
func TestGoldenInputsRegenerate(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	switch {
	case !*updateGolden:
		t.Skip("regeneration du fixture : passer -update (et REPLAY_FILM_DIR)")
	case dir == "":
		t.Skip("regeneration du fixture : REPLAY_FILM_DIR non defini")
	}
	g, err := decodeFilmInputs(goldenFilm, dir)
	if err != nil {
		t.Fatalf("decodage du film %s : %v", dir, err)
	}
	blob := encodeGoldenInputs(g)
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(blob); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", goldenDir, err)
	}
	if err := os.WriteFile(goldenInputsPath(), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture du fixture : %v", err)
	}
	t.Logf("fixture reecrit : %s (%d octets brut, %d compresse) — %d positions, %d tirs, "+
		"%d loadouts, %d lancers, %d projectiles, %d inventaires, %d lectures grappin, "+
		"%d morts, %d index",
		goldenInputsPath(), len(blob), buf.Len(), len(g.Positions), len(g.Fire), len(g.Loadouts),
		len(g.Grenades), len(g.Projectiles), len(g.Inventory), len(g.GrappleReads),
		len(g.Deaths), len(g.Indices.ByXUID))
}

// decodeFilmInputs rejoue EXACTEMENT la sequence de decodage de BuildFromFilm — c est ce qui
// garantit que le fixture porte les memes entrees que la production. Les bornes de carte sont
// celles de Cliffhanger, lues dans le catalogue versionne du titre.
func decodeFilmInputs(film, dir string) (*goldenInputs, error) {
	entry, err := goldenMapQuant()
	if err != nil {
		return nil, err
	}
	// MEME GESTE QUE LA PRODUCTION (cf. installWorldObjectPrecision) : les largeurs d'axe du
	// chemin world-object viennent de l'entree de catalogue, pas du defaut de paquet. Sur
	// Cliffhanger les deux coincident — c'est precisement pourquoi l'oubli avait survecu des
	// mois : le film de reference est le SEUL sur lequel il ne se voit pas.
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: entry.AxisWidths})
	wr := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &wr
	scan.CaptureDirs = true
	// MEME GESTE QUE LA PRODUCTION (BuildFromFilm) : les teleportations se lisent AVANT les
	// positions, parce qu elles exemptent le filtre de vitesse (decision D2). Sans ce geste,
	// le fixture porterait des positions que la production ne decode plus.
	translocs := filmdec.ScanFilmTranslocatorTeleports(dir)
	scan.TeleportExemptions = filmdec.TeleportExemptionsOf(translocs)
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, err
	}
	g := &goldenInputs{Film: film, Positions: pos, Translocations: translocs}
	if g.Fire, err = filmdec.ScanFilmFireEvents(dir); err != nil {
		return nil, err
	}
	if g.Loadouts, err = filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies()); err != nil {
		return nil, err
	}
	if g.Inventory, _, err = ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0); err != nil {
		return nil, err
	}
	if g.AbilityRanks, _, err = filmdec.ScanFilmAbilityRanks(dir); err != nil {
		return nil, err
	}
	if g.InventoryDeltas, _, err = filmdec.ScanFilmInventoryDeltas(dir); err != nil {
		return nil, err
	}
	if g.CamoStates, _, err = filmdec.ScanFilmCamoStates(dir); err != nil {
		return nil, err
	}
	if g.GrappleReads, _, err = filmdec.ScanFilmGrappleReads(dir); err != nil {
		return nil, err
	}
	if g.Placements, g.PlacementStats, err = filmdec.ScanFilmEquipmentPlacements(dir, &wr); err != nil {
		return nil, err
	}
	// Les armes au sol passent par LA MEME fonction que BuildFromFilm : le fixture porte ce que
	// la production decode, pas une variante de lecture — largeurs MPP calibrees comprises.
	g.Pads = decodeFilmPadScans(dir, &wr, g.PlacementStats.Calibration.Widths)
	if g.Grenades, err = filmdec.ScanFilmGrenadeThrows(dir); err != nil {
		return nil, err
	}
	if g.Projectiles, err = filmdec.ScanFilmProjectiles(dir, &wr); err != nil {
		return nil, err
	}
	if g.Deaths, err = ScanFilmDeaths(dir); err != nil {
		return nil, err
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(g.Deaths))
	if err != nil {
		return nil, err
	}
	table, collisions := injectiveOrEmpty(idx)
	if collisions > 0 {
		return nil, fmt.Errorf("index de joueur non injectif (%d collisions) — fixture refuse", collisions)
	}
	g.Indices = table
	// L origine d horloge est lue par la MEME fonction que BuildFromFilm : le fixture porte
	// l entree, pas une valeur recopiee a la main.
	if g.ClockOriginUS, err = ScanFilmClockOrigin(dir); err != nil {
		return nil, err
	}
	return g, nil
}

// goldenMapQuant rend l'ENTREE DE CATALOGUE de Cliffhanger : bornes ET largeurs d'axe, comme
// `replay.Options.MapQuant` les recoit en production. Les dissocier laisserait la regeneration
// armer les bornes en oubliant les largeurs — l'erreur meme que le lot du 2026-08-15 corrige.
//
// Si le catalogue change, [TestGoldenAssembly] tombe et le diff dit exactement ce qui a bouge.
func goldenMapQuant() (filmdec.MapQuantEntry, error) {
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		return filmdec.MapQuantEntry{}, fmt.Errorf("catalogue de bornes %s : %w", path, err)
	}
	entry, err := cat.Lookup("Cliffhanger")
	if err != nil {
		return filmdec.MapQuantEntry{}, err
	}
	return entry, nil
}
