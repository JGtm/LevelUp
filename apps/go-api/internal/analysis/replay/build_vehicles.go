package replay

// build_vehicles.go — LE CABLAGE du calque des VEHICULES : ce que BuildFromFilm DECODE du film
// (`ti=40`), et ce que BuildFromPositions en ASSEMBLE.
//
// MEME FORME QUE `build_ground_weapons.go`, et pour la meme raison : `build.go` est deja
// au-dessus du seuil de 500 lignes gele par la baseline. Le decodage, le journal et les refus du
// calque vivent donc ici, ensemble ; `build.go` ne porte qu un appel par cote.
//
// TROIS LECTURES DU FILM, ET PAS UNE DE PLUS :
//
//	1. le RECENSEMENT des images-cles (`ScanFilmWorldObjectKeyframes(dir, 40)`) — il rend d un
//	   seul parcours la BANDE de slots `ti=40` et les VIES `(slot, gen)` avec leurs instants de
//	   recensement. C est lui, et lui seul, qui BORNE la fin de vie (cf. build_vehicles_end.go) ;
//	2. les CREATIONS (`ScanFilmVehicleCreationsForBand`) — la position de NAISSANCE et le mot
//	   d identite du chassis (`MPPWord32`) ;
//	3. les TRAJECTOIRES (`ScanFilmBipedPositionsForBand` sur la bande `ti=40`) — le nuage des
//	   positions du vehicule, avec la VELOCITE `i1` d ou sort le cap.
//
// Une QUATRIEME lecture, additive et non fatale : les EVENEMENTS d embarquement / sortie
// (`ScanFilmVehicleEvents`), qui datent a la milliseconde les episodes d occupation.
//
// LES LARGEURS DU BLOC MPP SONT CELLES DE CE FILM, exactement comme pour les socles : le mot
// d identite de 32 bits se lit derriere deux champs de largeur VARIABLE, mesures par la
// calibration des poses `ti=37` sur le MEME film. Sans les reinstaller, le balayage lirait
// l identite aux largeurs PAR DEFAUT et AUCUNE famille de chassis ne se resoudrait — en silence.
//
// CE QUE CE CALQUE NE PUBLIE PAS, ET C EST UNE REFUTATION MESUREE : la DESTRUCTION. Le rapport
// `.ai/V7.5/film_re/V3_DESTRUCTION_DATEE_2026-09-02.md` a mesure 460 vies de vehicule sur
// 12 films : ZERO occupant encore a bord a la fin serree du flux, mort a bord ANTI-correlee
// (3,8 % contre 21,3 % au temoin), et un vehicule qui replique encore 13 a 36 s (mediane par lot)
// APRES avoir ete quitte. La fin de vie publiee ici est donc une BORNE de recensement, et sa
// cause vaut `unknown` — jamais `destruction`.
//
// HORS LIGNE : `decodeFilmVehicleScan` fait de l I/O disque sur tout le film et n est appelee que
// par `BuildFromFilm`, sous `LockProcessDecode`. `attachVehicles` est PUR.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// VehicleScan porte ce qu une lecture du film rend sur les VEHICULES (`ti=40`).
//
// UNE ENTREE DE DONNEES, comme `PadScans` : `Scanned` faux veut dire « pas lu », ce qui n est pas
// « aucun vehicule ». La couverture publie la distinction (cf. VehicleCoverage.Scanned).
type VehicleScan struct {
	// Scanned dit que le film a ete BALAYE jusqu au bout. Faux : archetype absent des
	// images-cles, creations illisibles, ou pas de film du tout (assemblage sur positions figees).
	Scanned bool
	// Keyframes porte la BANDE de slots `ti=40` et le RECENSEMENT qui borne les fins de vie.
	Keyframes filmdec.WorldObjectKeyframes
	// Creations sont les records de creation acceptes : position de naissance + bloc MPP.
	Creations []filmdec.EquipmentCreation
	Stats     filmdec.EquipmentCreationStats
	// Positions est le nuage NON decime des positions de vehicule, lu a la grammaire bipede
	// (porte 5 bits) sur la bande `ti=40` — la seule qui rende 99,4 a 100 % de pas sous 35 m/s
	// (cadrage vehicules du 2026-08-31).
	Positions []filmdec.BipedPosition
	// Events sont les embarquements et les sorties de la liste d evenements des paquets delta.
	// Absents = episodes d occupation bornes par le seul trou de position (repli mesure).
	Events []filmdec.VehicleEvent
}

// decodeFilmVehicleScan decode les QUATRE lectures du calque des vehicules sur le meme film et
// aux memes largeurs de bloc MPP que la chaine des socles.
//
// TROIS PANNES, TROIS PHRASES, et la distinction est le point — meme doctrine que
// `decodeFilmPadScan`. Un film sans `ti=40` aux images-cles (aucune bande) n est pas un film dont
// les creations sont illisibles, et ni l un ni l autre n est un film dont le nuage de positions
// manque. Le calque se tait entierement plutot que de publier des vehicules sans trajectoire.
//
// HORS LIGNE — appelee par BuildFromFilm, sous LockProcessDecode.
func decodeFilmVehicleScan(
	filmDir string, wr *filmdec.Vec3Range, mpp filmdec.MPPWidths,
) VehicleScan {
	defer gwInstallMPPWidths(mpp)()
	kf := filmdec.ScanFilmWorldObjectKeyframes(filmDir, filmdec.VehicleTypeIndex)
	if len(kf.Band) == 0 {
		slog.Info("vehicules : aucun slot ti=40 aux images-cles — rejeu sans ce calque",
			"filmDir", filmDir, "imagesCles", len(kf.TimesUS))
		return VehicleScan{}
	}
	cre, st, err := filmdec.ScanFilmVehicleCreationsForBand(filmDir, wr, kf.Band)
	if err != nil {
		slog.Warn("vehicules : records de creation illisibles — rejeu sans ce calque",
			"err", err, "filmDir", filmDir)
		return VehicleScan{}
	}
	pos, err := filmdec.ScanFilmBipedPositionsForBand(filmDir, kf.Band, vehicleScanOptions(wr))
	if err != nil {
		slog.Warn("vehicules : nuage de positions illisible — AUCUN vehicule publie (sans lui, une"+
			" vie recensee n aurait ni trajectoire ni cap)", "err", err, "filmDir", filmDir)
		return VehicleScan{}
	}
	out := VehicleScan{Scanned: true, Keyframes: kf, Creations: cre, Stats: st, Positions: pos}
	out.Events = decodeFilmVehicleEvents(filmDir)
	slog.Info("vehicules : balayage ti=40",
		"slots", st.Slots, "ancres", st.Anchors, "creationsAcceptees", st.Accepted,
		"imagesCles", len(kf.TimesUS), "viesRecensees", len(kf.SeenUS),
		"echantillons", len(pos), "evenements", len(out.Events))
	return out
}

// decodeFilmVehicleEvents lit les embarquements et les sorties. ADDITIF ET NON FATAL : leur
// absence rend les episodes d occupation au seul trou de position, qui est la primitive de repli
// MESUREE (86,3 % des trous portent leur sortie, et 100 % de ces sorties ferment le trou a
// +/-2 s — rapport V3_DESTRUCTION_DATEE_2026-09-02, gate 6).
func decodeFilmVehicleEvents(filmDir string) []filmdec.VehicleEvent {
	ev, err := filmdec.ScanFilmVehicleEvents(filmDir)
	if err != nil {
		slog.Warn("vehicules : liste d evenements illisible — episodes d occupation bornes par le"+
			" seul trou de position", "err", err, "filmDir", filmDir)
		return nil
	}
	return ev
}

// vehicleScanOptions rend les reglages du nuage `ti=40`. TROIS ecarts au bipede, tous documentes
// par `ScanFilmBipedPositionsForBand` et tous necessaires :
//
//   - `RequireTag1` DESARME : le tag de 2 bits est la generation du handle, et les objets du
//     monde en emploient les quatre valeurs. Arme, la bande ne rendrait qu un quart du nuage ;
//   - `CaptureDirs` ARME : c est lui qui livre la VELOCITE `i1`, seule orientation validee d un
//     vehicule en mouvement (V1a.3 : ecart median 1,7 a 2,1 deg au deplacement sur 4 films,
//     temoin par melange 51 a 88 deg) ;
//   - les filtres de post-traitement (`MaxSpeedMPS`, `IsolationGapMS`) gardent leur valeur de
//     production : ce sont ceux sous lesquels le nuage vehicule a ete mesure (V1a).
func vehicleScanOptions(wr *filmdec.Vec3Range) filmdec.ScanFilmOptions {
	opt := filmdec.DefaultScanFilmOptions()
	opt.RequireTag1 = false
	opt.CaptureDirs = true
	opt.WorldRange = wr
	return opt
}

// attachVehicles pose le calque des VEHICULES sur le document : une vie par vehicule recense, sa
// trajectoire sur l axe de frames, ses episodes d occupation, sa couverture, et le journal qui
// porte les memes denominateurs que l artefact.
//
// `bipeds` est le nuage NON decime des BIPEDES (celui des joueurs), trie par instant : c est lui
// qui porte les TROUS de position d ou sortent les episodes d occupation. `own` donne le pont
// slot -> xuid, sans lequel un episode reste anonyme (il est publie quand meme : le vehicule est
// occupe, seul son occupant est inconnu).
func attachVehicles(
	doc *ReplayDocument, scan VehicleScan, bipeds []filmdec.BipedPosition,
	own OwnerReport, clock replayClock,
) {
	tracks, cov := buildVehicleTracks(scan, bipeds, own, clock)
	doc.Vehicles = tracks
	doc.Coverage.Vehicles = &cov
	logVehicleCoverage(&cov)
}
