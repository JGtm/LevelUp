package service

// replay_vehicle_weapons.go — LE REGISTRE DES ARMES DE VEHICULE, pose A LA REQUETE (schema 69,
// retours du rejeu du 2026-09-23, lot M4a).
//
// CE QUE CA OUVRE. Un tir d arme de vehicule publie `Shot.Weapon = 0x<tag>00000000` ; aucun
// registre d armes de JOUEUR ne le nomme. Le client dessinait et faisait sonner ces tirs par trois
// tables a lui, clees par des tags jamais confrontes a un film. Le TITRE declare desormais ces
// armes (`config/titles/{slug}/mappings/vehicle_weapons.toml` : forme, teinte, son, montage, avec
// leur preuve) et ce fichier en pose, pour les seules armes que les tirs du document emploient,
// la table `vehicleWeapons` keyee par la cle meme du tir.
//
// A LA REQUETE ET NON AU BUILD : meme patron et meme raison que `resolveVehicleLabels` — une
// resolution du titre qui s ameliore (une teinte tranchee, un son reconstruit) ne se fige pas dans
// un artefact deja cuit.
//
// TITLE-AGNOSTIC : le fichier est cherche dans les mappings du TITRE du service
// (`PathResolver.TitleMappingsDir`). Un titre qui n en declare pas n a pas d arme de vehicule
// nommee : le document sort sans la table, et le client garde son rendu neutre — jamais une
// erreur, jamais le registre d un autre titre.

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strconv"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/mappings"
)

// vehicleWeaponsFile : le nom du registre dans les mappings d un titre.
const vehicleWeaponsFile = "vehicle_weapons.toml"

// resolveVehicleWeapons pose `doc.VehicleWeapons` : une entree par arme DU REGISTRE qu au moins un
// tir du document emploie. Absente (pas vide) quand aucune ne l est.
func (s *replayService) resolveVehicleWeapons(ctx context.Context, doc *replay.ReplayDocument) {
	if doc == nil || len(doc.Shots)+len(doc.Bursts) == 0 {
		return
	}
	path := filepath.Join(title.NewPathResolver(s.repoRoot).TitleMappingsDir(s.titleSlug), vehicleWeaponsFile)
	set, err := mappings.LoadVehicleWeaponsFromFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		slog.DebugContext(ctx, "rejeu 2D : aucun registre d armes de vehicule pour ce titre",
			"titleSlug", s.titleSlug)
		return
	}
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : registre des armes de vehicule illisible — tirs de"+
			" vehicule sans style ni son", "err", err, "titleSlug", s.titleSlug)
		return
	}
	out := vehicleWeaponsUsed(doc.Shots, doc.Bursts, set)
	if len(out) > 0 {
		doc.VehicleWeapons = out
	}
}

// vehicleWeaponsUsed rend l entree du registre de chaque arme employee par les tirs et par les
// rafales de tir continu (schema 71).
func vehicleWeaponsUsed(shots []replay.Shot, bursts []replay.FireBurst,
	set *mappings.VehicleWeaponSet) map[string]replay.VehicleWeapon {
	cles := make([]string, 0, len(shots)+len(bursts))
	for _, sh := range shots {
		cles = append(cles, sh.Weapon)
	}
	for _, b := range bursts {
		cles = append(cles, b.Weapon)
	}
	out := map[string]replay.VehicleWeapon{}
	for _, cle := range cles {
		if _, vu := out[cle]; vu || cle == "" {
			continue
		}
		tag, ok := vehicleWeaponTagOf(cle)
		if !ok {
			continue
		}
		w, ok := set.Weapon(tag)
		if !ok {
			continue
		}
		out[cle] = vehicleWeaponOf(w)
	}
	return out
}

// vehicleWeaponTagOf rend le tag du registre (8 chiffres hex majuscules) d une cle de tir, ou faux
// si la cle n est pas celle d une arme de vehicule. La cle est RECOMPOSEE par
// `replay.VehicleWeaponKey`, le seul ecrivain du gabarit : aucune copie du format ici.
func vehicleWeaponTagOf(key string) (string, bool) {
	if len(key) != len("0x")+16 {
		return "", false
	}
	v, err := strconv.ParseUint(key[2:], 16, 64)
	if err != nil {
		return "", false
	}
	tag := uint32(v >> 32)
	if replay.VehicleWeaponKey(tag) != key {
		return "", false
	}
	return key[2:10], true
}

// vehicleWeaponOf projette une entree du registre vers la forme du document.
func vehicleWeaponOf(w mappings.VehicleWeapon) replay.VehicleWeapon {
	out := replay.VehicleWeapon{
		Vehicle: w.Vehicle, En: w.En, Fr: w.Fr, Fire: w.Fire, Fx: w.Fx, Tint: w.Tint, Sound: w.Sound,
		Loop: w.Loop,
	}
	if w.Mount != nil {
		out.Mount = &replay.VehicleWeaponMount{Aim: w.Mount.Aim, AX: w.Mount.AX, AY: w.Mount.AY}
	}
	return out
}
