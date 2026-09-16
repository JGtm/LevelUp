package replay

// golden_inputs_canaux_test.go — LES SIX CANAUX ENTRES AU FIXTURE AU LOT 1.0 (2026-09-14).
//
// # CE QU ILS FERMENT
//
// Jusqu au 2026-09-14, le chemin du fixture RECOPIAIT la sequence de balayages de `BuildFromFilm`
// et cinq canaux entiers y manquaient — donc manquaient AUSSI au codec, donc aux huit goldens
// d assemblage, qui affirmaient « prises et lachers d arme decodes=0 » et « vehicules
// balaye=false » la ou la production publie ces calques (decouverte D7 du lot 0.D). Le fixture
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

import (
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
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
func encodeWeaponChanges(w *gwriter, changes []grammar.HeldWeaponChange) {
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

func decodeWeaponChanges(r *greader) []grammar.HeldWeaponChange {
	n := int(r.u())
	out := make([]grammar.HeldWeaponChange, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := grammar.HeldWeaponChange{TimestampUS: lastTS, Slot: uint32(r.u())}
		c.SlotIndex = int(r.i())
		c.Family = uint32(r.u())
		c.Previous = uint32(r.u())
		c.Kind = grammar.HeldWeaponChangeKind(r.str())
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
func encodePickups(w *gwriter, pickups []grammar.BipedPickup, st grammar.BipedPickupStats) {
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

func decodePickups(r *greader) ([]grammar.BipedPickup, grammar.BipedPickupStats) {
	n := int(r.u())
	out := make([]grammar.BipedPickup, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		p := grammar.BipedPickup{TimestampUS: lastTS, Slot: uint32(r.u()), CatalogID: uint32(r.u())}
		p.Class = r.byte8()
		out = append(out, p)
	}
	st := grammar.BipedPickupStats{
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
func encodeEquipmentChanges(w *gwriter, changes []grammar.EquipmentChange, st grammar.EquipmentChangeStats) {
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

func decodeEquipmentChanges(r *greader) ([]grammar.EquipmentChange, grammar.EquipmentChangeStats) {
	n := int(r.u())
	out := make([]grammar.EquipmentChange, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := grammar.EquipmentChange{TimestampUS: lastTS, Slot: uint32(r.u()), Counter: uint32(r.u())}
		c.Rank = int(r.i())
		c.Previous = int(r.i())
		c.Kind = grammar.EquipmentChangeKind(r.str())
		c.Recovered = r.bool8()
		c.Gap = int(r.i())
		out = append(out, c)
	}
	var st grammar.EquipmentChangeStats
	st.Walk = grammar.AbilityRankStats{
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

func decodeVehicleScan(r *greader, lay grammar.I0Layout, world grammar.Vec3Range) VehicleScan {
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
func encodeVehicleEvents(w *gwriter, events []grammar.VehicleEvent) {
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

func decodeVehicleEvents(r *greader) []grammar.VehicleEvent {
	n := int(r.u())
	out := make([]grammar.VehicleEvent, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		e := grammar.VehicleEvent{TimestampUS: lastTS, Kind: int(r.i())}
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
// L INVENTAIRE, TENU PAR LE COMPILATEUR
// ---------------------------------------------------------------------------

// champsNonTransportes : les champs de [FilmInputs] que le codec ne porte PAS, avec la raison.
//
// LES TROIS SONT DES CALQUES GARDES PAR L APPELANT : le marqueur de portage du drapeau, l etat
// des zones et l anneau de la bombe ne se balaient que si `Options.Flag` / `.Zone` / `.Bomb`
// portent la garde de mode correspondante. Le fixture n en fournit AUCUNE (il ne connait ni la
// variante du match ni le catalogue de zones de la carte), donc ces champs sont vides des deux
// cotes — frais comme relus — et les serialiser figerait des zeros. Le jour ou un fixture
// porterait un catalogue de zones, ils devraient entrer au codec : c est ce que ce test force a
// decider.
var champsNonTransportes = map[string]string{
	"FlagMarks":   "calque garde par Options.Flag — vide sans garde de mode CTF",
	"ZoneReads":   "calque garde par Options.Zone.Zones — vide sans catalogue de zones",
	"ZoneScanned": "temoin du precedent",
	"BombReads":   "calque garde par Options.Bomb.Scanned — vide sans garde de mode Assaut",
}

// TestCodecCouvreFilmInputs : TOUT champ de [FilmInputs] est soit serialise, soit NOMME comme
// deliberement absent.
//
// # CE QU IL FERME, ET IL A DEJA COUTE
//
// Le fixture porte le type de la PRODUCTION depuis le lot 1.0 : un canal ajoute a l etage de
// balayage apparait donc tout seul dans `goldenInputs`, et l assemblage le lira — mais le CODEC,
// lui, ne l apprend pas. Le fixture relu rendrait alors un calque vide la ou la production en
// publie un plein, exactement le defaut que la decouverte D9 avait mesure sur sept builds.
// `TestGoldenInputsFidelite` l attrape, mais il EXIGE LE CACHE DE FILMS : il saute en CI. Ce
// test-ci, lui, ne lit aucun octet de film et tourne partout.
//
// IL SE FONDE SUR L ALLER-RETOUR, PAS SUR UNE LISTE ECRITE A LA MAIN : une valeur non nulle est
// posee dans chaque champ, le blob est encode puis relu, et un champ qui revient VIDE n a pas ete
// transporte. Une liste de noms aurait derive du codec au premier oubli.
func TestCodecCouvreFilmInputs(t *testing.T) {
	entry := goldenEntryPourTest(t)
	champs := reflect.VisibleFields(reflect.TypeOf(FilmInputs{}))
	if len(champs) < 30 {
		t.Fatalf("%d champ(s) lus sur FilmInputs : la reflexion ne mesure plus rien", len(champs))
	}
	for _, f := range champs {
		if f.Anonymous || !f.IsExported() || strings.Contains(f.Name, ".") {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			g := &goldenInputs{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths}
			v := reflect.ValueOf(&g.FilmInputs).Elem().FieldByName(f.Name)
			if !remplirChampTemoin(v) {
				t.Skipf("aucun temoin fabricable pour %s (%s)", f.Name, f.Type)
			}
			relu, err := decodeGoldenInputs(encodeGoldenInputs(g), entry)
			if err != nil {
				t.Fatalf("aller-retour sur %s : %v", f.Name, err)
			}
			transporte := !reflect.ValueOf(relu.FilmInputs).FieldByName(f.Name).IsZero()
			raison, nomme := champsNonTransportes[f.Name]
			switch {
			case transporte && nomme:
				t.Fatalf("%s est transporte par le codec ET declare absent (%q) : retirer l entree "+
					"de champsNonTransportes", f.Name, raison)
			case !transporte && !nomme:
				t.Fatalf("FilmInputs.%s N EST PAS TRANSPORTE par le codec du fixture.\n"+
					"L assemblage le consomme (cf. FilmInputs.applyTo) : relu vide, le golden "+
					"figerait un calque que la production publie plein.\n"+
					"Soit l ajouter au codec (golden_inputs_canaux_test.go) et monter "+
					"goldenInputsMagic, soit l inscrire dans champsNonTransportes avec sa raison.",
					f.Name)
			}
		})
	}
}

// remplirChampTemoin pose une valeur NON NULLE dans un champ, quelle que soit sa forme. Rend faux
// quand la forme n est pas fabricable — aucune ne l est aujourd hui, et le test le dirait.
func remplirChampTemoin(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(7)
	case reflect.Uint32, reflect.Uint64:
		v.SetUint(7)
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		if !remplirChampTemoin(p.Elem()) {
			return false
		}
		v.Set(p)
	case reflect.Slice:
		e := reflect.New(v.Type().Elem()).Elem()
		remplirStructTemoin(e)
		v.Set(reflect.Append(v, e))
	case reflect.Struct:
		return remplirStructTemoin(v)
	default:
		return false
	}
	return true
}

// remplirStructTemoin pose une valeur non nulle dans le PREMIER champ remplissable d une
// structure (recursivement). Un seul suffit : ce qu on mesure est si le champ REVIENT vide.
func remplirStructTemoin(v reflect.Value) bool {
	if v.Kind() != reflect.Struct {
		return remplirChampTemoin(v)
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		if remplirChampTemoin(f) {
			return true
		}
	}
	return false
}
