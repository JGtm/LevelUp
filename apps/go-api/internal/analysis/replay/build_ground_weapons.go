package replay

// build_ground_weapons.go — LE CABLAGE du calque des ARMES AU SOL : ce que BuildFromFilm
// DECODE du film, et ce que BuildFromPositions en ASSEMBLE.
//
// Extrait de `build.go` et de `ground_weapon_objects.go` au correctif de revue du 2026-08-17 :
// le lot des socles avait pousse `build.go` de 621 a 640 lignes, au-dessus d un seuil deja gele
// par la baseline. Le decodage, le journal et les refus du calque vivent donc ici, ensemble ;
// `build.go` ne porte plus que deux appels.
//
// TROIS LECTURES DU FILM, ET PAS UNE DE PLUS : une marche des images-cles (bande de slots
// `ti=42` + recensement), une marche des paquets delta pour les records de CREATION, une pour
// les pistes de position. La marche des images-cles en rend deux pour le prix d une.
//
// HORS LIGNE : `decodeFilmGroundWeapons` fait de l I/O disque sur tout le film et n est appelee
// que par `BuildFromFilm`, sous `LockProcessDecode`. `attachWeaponPads` est PUR.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// decodeFilmGroundWeapons décode les armes au sol du film et JOURNALISE ce qu'il en est.
//
// TROIS PANNES, TROIS PHRASES, et la distinction est le point. Un film sans archétype `ti=42`
// aux images-clés (aucune bande) n'est pas un film dont les créations sont illisibles, et ni
// l'un ni l'autre n'est un film dont les pistes delta manquent — ce dernier cas est le plus
// TRAÎTRE, parce qu'une piste absente rend `HasDelta` faux et ferait passer TOUTE apparition
// `spawned` pour un objet apparu au repos. Le calque se tait donc entièrement plutôt que de
// publier des socles fabriqués par une lecture manquante.
//
// LES LARGEURS DU BLOC MPP SONT CELLES DE CE FILM, ET C'EST UN CORRECTIF DE REVUE (2026-08-17).
// Le mot d'identité de 32 bits se lit derrière deux champs de largeur VARIABLE, mesurés par la
// calibration des poses `ti=37` sur le MÊME film (9/5 en Quick Play, 8/3 sur les films BTB
// mesurés) — et `ScanFilmEquipmentPlacements` les RESTAURE en sortant. Sans les réinstaller ici,
// le balayage `ti=42` lisait l'identité aux largeurs PAR DÉFAUT : sur un film calibré autrement,
// aucune création n'aurait résolu d'arme, le calque aurait publié zéro socle, et rien ne l'aurait
// dit (découverte 8 du plan). Largeurs non mesurées (calibration refusée) : on garde le défaut,
// et le compteur `kept` de la couverture reste le témoin.
//
// HORS LIGNE — appelée par BuildFromFilm, sous LockProcessDecode.
func decodeFilmGroundWeapons(
	filmDir string, wr *filmdec.Vec3Range, mpp filmdec.MPPWidths,
) GroundWeaponScan {
	defer gwInstallMPPWidths(mpp)()
	kf := filmdec.ScanFilmGroundWeaponKeyframes(filmDir)
	if len(kf.Band) == 0 {
		slog.Warn("armes au sol : aucun slot d'archetype ti=42 aux images-cles — rejeu sans socles",
			"filmDir", filmDir, "imagesCles", len(kf.TimesUS))
		return GroundWeaponScan{}
	}
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(filmDir, wr, kf.Band)
	if err != nil {
		slog.Warn("armes au sol : records de creation ti=42 illisibles — rejeu sans socles",
			"err", err, "filmDir", filmDir)
		return GroundWeaponScan{}
	}
	tracks, err := filmdec.ScanFilmWorldObjectsForBand(filmDir, wr, kf.Band)
	if err != nil {
		slog.Warn("armes au sol : pistes delta ti=42 illisibles — AUCUN socle publie (sans elles,"+
			" toute apparition passerait pour un objet apparu au repos)",
			"err", err, "filmDir", filmDir)
		return GroundWeaponScan{}
	}
	slog.Info("armes au sol : balayage ti=42",
		"slots", st.Slots, "ancres", st.Anchors, "acceptees", st.Accepted,
		"imagesCles", len(kf.TimesUS), "viesRecensees", len(kf.SeenUS), "pistesDelta", len(tracks))
	return GroundWeaponScan{Scanned: true, Creations: cre, Stats: st, Keyframes: kf, Tracks: tracks}
}

// gwInstallMPPWidths installe les largeurs du bloc MPP MESURÉES sur ce film et rend leur
// restauration. Largeurs non renseignées (calibration refusée) : rien n'est installé — le défaut
// de paquet vaut mieux qu'un découpage nul, qui ne lirait aucune identité du tout.
//
// L'APPELANT DOIT DÉTENIR LockProcessDecode : ce sont des globaux de paquet (même contrat que
// `installWorldObjectPrecision`).
func gwInstallMPPWidths(w filmdec.MPPWidths) func() {
	if !w.Valid() {
		return func() {}
	}
	prev := filmdec.SetMPPWidths(w)
	return func() { filmdec.SetMPPWidths(prev) }
}

// attachWeaponPads pose le calque des SOCLES sur le document : les socles, leurs occupations
// achevees, leur couverture, et le journal qui porte les memes denominateurs que l artefact.
//
// LE NUAGE EST CELUI QUI N EST PAS DECIME, et il le faut : la datation d une disparition se joue
// a la frame, la decimation perdrait le passage de joueur qui la borne. Le calque publie les
// socles du MATCH — aucun catalogue de carte, aucun ramasseur.
//
// `positions` doit etre TRIE par instant (c est le cas de `sorted` dans BuildFromPositions).
func attachWeaponPads(
	doc *ReplayDocument, scan GroundWeaponScan, positions []filmdec.BipedPosition, clock replayClock,
) {
	doc.WeaponPads, doc.PadPickups, doc.Coverage.GroundWeapons = buildWeaponPads(scan, positions, clock)
	logGroundWeaponCoverage(doc.Coverage.GroundWeapons)
}
