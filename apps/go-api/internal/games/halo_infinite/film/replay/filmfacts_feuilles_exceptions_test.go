package replay

// filmfacts_feuilles_exceptions_test.go — LES FEUILLES QUE LE FICHIER DE FAITS NE PORTE PAS, UNE
// PAR UNE, AVEC LEUR PREUVE (jalon J11.0-bis, 2026-09-28).
//
// # CE QU UNE EXCEPTION AFFIRME, ET COMMENT C EST PROUVE
//
// Qu aucun octet du document publie n en depend : le rejeu depuis les faits rend le meme artefact
// que le decodage du film sans elle. La preuve tient en deux pieces, et une troisieme la mesure :
//
//  1. LE RELEVE AU VERIFICATEUR DE TYPES (2026-09-28, `golang.org/x/tools/go/packages` sur tout le
//     module hors tests, usages resolus par le compilateur — promotions par embarquement comprises
//     — du champ exact) : chaque champ ci-dessous n est ECRIT ou LU que par le BALAYAGE
//     (`internal/grammar`, et `decodeFilm*Scan` de ce paquet), jamais par l assemblage
//     (`BuildFromPositions` et ses calques, `replaybuild`). Les sites sont cites.
//  2. [TestFeuillesNonTransporteesHorsDuDocument] : aucun type proprietaire d une exception n est
//     atteignable depuis [ReplayDocument]. Un champ qui voyagerait par COPIE DE STRUCTURE (une
//     serialisation JSON, par exemple) echapperait au releve 1 — ce test le dirait.
//  3. LA MESURE S8 (`cmd/replay-equiv -deux-passes`) : sur les vingt films de reference, l artefact
//     rejoue depuis les faits est identique A L OCTET a l artefact decode du film.
//
// # CE QUE LA TABLE N EST PAS
//
// Un lieu ou ranger un champ qui derange. Une entree neuve est une DECISION ecrite : le releve 1
// refait pour ce champ, et le S8 rejoue. Une entree dont le champ se met a faire l aller-retour est
// MORTE, et `TestCodecCouvreFilmInputs` refuse de la garder.

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// tracabilite : la coordonnee d un record DANS LE FILM (chunk, paquet, bit). Le rejeu depuis les
// faits ne relit pas le film : elle n a rien a y designer.
const tracabilite = "tracabilite de balayage (coordonnee dans le film) : ecrite par le balayeur, lue " +
	"par lui seul (tri stable ou appariement a la lecture) ; "

// feuillesNonTransportees : champ declare -> preuve. Les sites sont ceux du releve du 2026-09-28.
var feuillesNonTransportees = map[string]string{
	// Positions de bipede et de vehicule (meme type, meme codec : `encodePositionSection`).
	"grammar.BipedPosition.Chunk":       tracabilite + "grammar/offline_biped_band.go:scanBipedChunks",
	"grammar.BipedPosition.PacketIndex": tracabilite + "grammar/offline_biped_band.go:scanBipedChunks",
	"grammar.componentDirs.MaskBits": "masque de composants du record, tracabilite : seul " +
		"grammar/offline_biped.go:ScanBipedRecords l ecrit (cf. filmfacts_directions.go, 7,4 Mio evites)",
	"grammar.BodyVitality.F5c": "bit de vitalite brut : seul grammar/vitality.go:decodeObjectBodyVitality l ecrit ; " +
		"l assemblage lit Body.Health (HealthAt), transporte",
	"grammar.BodyVitality.F5d": "idem F5c (decodeObjectBodyVitality)",
	"grammar.BodyVitality.F5e": "idem F5c (decodeObjectBodyVitality)",
	"grammar.BodyVitality.Q": "quantum de sante brut : seul decodeObjectBodyVitality l ecrit ; l assemblage " +
		"lit Body.Health, transporte",
	"grammar.ShieldVitality.Block64": "champ brut du bouclier : seul grammar/vitality.go:decodeObjectShieldVitality " +
		"l ecrit ; l assemblage lit Shield.Shield (ShieldAt) et Shield.Q (buildOvershieldEpisodes), transportes",
	"grammar.ShieldVitality.F66":          "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.F67":          "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.F68":          "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.F69":          "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.RegenPresent": "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.HasRegen0":    "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.Regen0":       "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.HasRegen1":    "idem Block64 (decodeObjectShieldVitality)",
	"grammar.ShieldVitality.Regen1":       "idem Block64 (decodeObjectShieldVitality)",
	// Records de creation de bipede.
	"grammar.BipedCreation.Chunk": tracabilite + "grammar/biped_creation.go:scanPayload, " +
		"grammar/birth_loadouts.go:ScanBirthLoadouts/lire",
	"grammar.BipedCreation.PacketIndex": tracabilite + "idem Chunk",
	"grammar.BipedCreation.BitPos":      tracabilite + "grammar/biped_creation.go:scanPayload, grammar/birth_loadouts.go:lire",
	"grammar.BipedCreation.Version": "garde de lecture du record : seul grammar/biped_creation.go:readCreation " +
		"l ecrit et la teste",
	"grammar.BipedCreation.Representation": "garde de lecture du record : seul readCreation l ecrit et la " +
		"compare a BipedRepresentationName",
	// Tirs.
	"grammar.FireEvent.Chunk":       tracabilite + "grammar/fire_events.go:ScanFireEvents",
	"grammar.FireEvent.PacketIndex": tracabilite + "grammar/fire_events.go:ScanFireEvents",
	"grammar.FireEvent.Short": "forme du record lue : seul grammar/fire_events.go:decodeFireEvent l ecrit " +
		"(reperee au sondage du 2026-09-28)",
	"grammar.FireEvent.Bloc": "presence du bloc 0x68 du record : seul decodeFireEvent l ecrit (reperee au " +
		"sondage du 2026-09-28) ; tir_continu.go lit le Bloc d un AUTRE type (la vue de controle)",
	"grammar.UnitRef.Probe": "temoin de sonde de la reference d unite : seul decodeFireEvent l ecrit ; " +
		"l assemblage lit Unit.Present/Slot/Gen, transportes",
	"grammar.GrenadeThrow.Chunk":       tracabilite + "grammar/grenade_events.go:ScanGrenadeThrows",
	"grammar.GrenadeThrow.PacketIndex": tracabilite + "grammar/grenade_events.go:ScanGrenadeThrows",
	"grammar.GrenadeThrow.BitPos":      tracabilite + "grammar/grenade_events.go:scanGrenadeThrows",
	// Armes, inventaires, capacites.
	"types.KeyframeLoadout.Chunk":         tracabilite + "grammar/keyframe_etats_fenetre.go:emettre",
	"types.KeyframeLoadout.PacketIndex":   tracabilite + "grammar/keyframe_etats_fenetre.go:emettre",
	"types.KeyframeInventory.Chunk":       tracabilite + "grammar/keyframe_etats_fenetre.go:emettre",
	"types.KeyframeInventory.PacketIndex": tracabilite + "grammar/keyframe_etats_fenetre.go:emettre",
	"types.KeyframeInventory.GrenadesByPosition": "mode de lecture des grenades par la fenetre : ecrit par " +
		"grammar/inventory_decode.go:keyframeInventoriesDe, compte par grammar/keyframe_etats_fenetre.go:rendreParLaFenetre ; RETIRE du " +
		"codec en v18 faute de lecteur (cf. filmfacts.go)",
	"types.InventoryDelta.Chunk":       tracabilite + "grammar/inventory_delta.go:readRecord",
	"types.InventoryDelta.PacketIndex": tracabilite + "grammar/inventory_delta.go:readRecord",
	"types.InventoryDelta.Ammo": "munitions lues dans le delta : ecrites par grammar/inventory_delta_ammo.go:" +
		"collectAmmo, lues par refuseAmmoIfContaminated (la PORTE du canal, au balayage) ; le VERDICT, " +
		"InventoryDeltaAmmoRefused, est transporte ; RETIRE du codec en v18 faute de lecteur",
	"types.HeldWeaponChange.Chunk": tracabilite + "grammar/held_weapon_changes.go:ScanHeldWeaponChanges",
	"types.HeldWeaponChange.Low": "mot bas brut de la reference d arme : seul ScanHeldWeaponChanges l ecrit " +
		"(repere au sondage du 2026-09-28)",
	"types.BipedPickup.Chunk":        tracabilite + "grammar/biped_pickups.go:ScanBipedPickups",
	"types.BirthLoadout.Chunk":       tracabilite + "grammar/birth_loadouts.go:lire",
	"types.BirthLoadout.PacketIndex": tracabilite + "grammar/birth_loadouts.go:lire",
	"types.AbilityRank.Chunk":        tracabilite + "grammar/ability_rank.go:ScanAbilityRanks",
	"types.AbilityRank.PacketIndex":  tracabilite + "grammar/ability_rank.go:ScanAbilityRanks",
	"types.AbilityRank.Counter": "compteur de rotation : ecrit par grammar/ability_rank.go:ScanAbilityRanks ; " +
		"il borne une lecture, il ne se publie pas (cf. filmfacts.go, liste des champs par type). Le " +
		"compteur publie est EquipmentChange.Counter, transporte",
	"types.EquipmentChange.Chunk": tracabilite + "grammar/equipment_changes.go:assembleEquipmentChanges, " +
		"cmpChangementDuFilm (tri)",
	"types.EquipmentChange.PacketIndex": tracabilite + "idem Chunk",
	"types.AbilityImpulse.Chunk":        tracabilite + "grammar/ability_impulses.go:publish",
	"types.AbilityImpulse.PacketIndex":  tracabilite + "grammar/ability_impulses.go:publish",
	"types.AbilityCharge.Chunk":         tracabilite + "grammar/ability_charges.go:publish",
	"types.AbilityCharge.PacketIndex":   tracabilite + "grammar/ability_charges.go:publish",
	"types.CamoRead.Chunk":              tracabilite + "grammar/camo_state.go:ScanCamoStates",
	"types.CamoRead.PacketIndex":        tracabilite + "grammar/camo_state.go:ScanCamoStates",
	"types.GrappleRead.Chunk":           tracabilite + "grammar/grapple_state.go:account",
	"types.GrappleRead.PacketIndex":     tracabilite + "grammar/grapple_state.go:account",
	// Objets du monde et vehicules.
	"grammar.WorldObjectKeyframes.Band": "bande de slots du recensement : lue par le BALAYAGE qui suit " +
		"(replay/build_ground_weapons.go:decodeFilmPadScan, replay/build_vehicles.go:decodeFilmVehicleScan, " +
		"grammar/navpoint_radial_scan.go) ; l assemblage lit SeenUS et les instants, transportes",
	"grammar.BipedAim.Chunk":         tracabilite + "grammar/offline_aim_only.go:ScanBipedAimOnly",
	"grammar.BipedAim.PacketIndex":   tracabilite + "grammar/offline_aim_only.go:ScanBipedAimOnly",
	"types.VehicleEvent.Chunk":       tracabilite + "grammar/event_list.go:ScanVehicleEvents",
	"types.VehicleEvent.PacketIndex": tracabilite + "grammar/event_list.go:ScanVehicleEvents",
	"grammar.FrameConfig.Obs": "l OBSERVATEUR du balayage (des fonctions) : pas une donnee, lu par la seule " +
		"marche (grammar/frame_*.go) ; les six champs de donnees du cadre sont transportes " +
		"(cf. filmfacts_mortsdobjet.go:cadreSerialisable)",
}

// TestFeuillesNonTransporteesHorsDuDocument : la seconde piece de la preuve. Aucun type
// proprietaire d une exception n est atteignable depuis le document publie : un champ perdu ne
// peut donc pas y arriver par copie de structure.
func TestFeuillesNonTransporteesHorsDuDocument(t *testing.T) {
	proprietaires := map[string]bool{}
	for c := range feuillesNonTransportees {
		proprietaires[c[:strings.LastIndex(c, ".")]] = true
	}
	vus := map[reflect.Type]bool{}
	var fautifs []string
	var visiter func(reflect.Type, string)
	visiter = func(ty reflect.Type, chemin string) {
		if vus[ty] {
			return
		}
		vus[ty] = true
		if ty.Name() != "" && proprietaires[nomDeTypeQualifie(ty)] {
			fautifs = append(fautifs, nomDeTypeQualifie(ty)+" sous "+chemin)
		}
		switch ty.Kind() {
		case reflect.Struct:
			for field := range ty.Fields() {
				visiter(field.Type, chemin+"."+field.Name)
			}
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visiter(ty.Elem(), chemin)
		case reflect.Map:
			visiter(ty.Key(), chemin)
			visiter(ty.Elem(), chemin)
		}
	}
	visiter(reflect.TypeFor[ReplayDocument](), "ReplayDocument")
	if len(vus) < 50 {
		t.Fatalf("%d type(s) visites depuis ReplayDocument : la mesure ne mesure plus rien", len(vus))
	}
	sort.Strings(fautifs)
	for _, f := range fautifs {
		t.Errorf("le type %s est PUBLIE : une feuille qu il porte et que le fichier de faits perd "+
			"changerait l artefact rejoue — retirer l exception et transporter le champ", f)
	}
}
