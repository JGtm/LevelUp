package replay

// build_ground_weapons.go — LE CABLAGE du calque des SOCLES : ce que BuildFromFilm DECODE du
// film, et ce que BuildFromPositions en ASSEMBLE.
//
// Extrait de `build.go` et de `ground_weapon_objects.go` au correctif de revue du 2026-08-17 :
// le lot des socles avait pousse `build.go` de 621 a 640 lignes, au-dessus d un seuil deja gele
// par la baseline. Le decodage, le journal et les refus du calque vivent donc ici, ensemble ;
// `build.go` ne porte plus qu un appel par cote.
//
// TROIS LECTURES DU FILM PAR ARCHETYPE, ET PAS UNE DE PLUS : une marche des images-cles (bande
// de slots + recensement), une marche des paquets delta pour les records de CREATION, une pour
// les pistes de position. La marche des images-cles en rend deux pour le prix d une.
//
// DEUX ARCHETYPES, UN SEUL DECODEUR (2026-08-19). Les ARMES AU SOL (`ti=42`) et les objets
// d EQUIPEMENT du monde (`ti=37`, d ou sortent les socles de POWER-UP) se lisent par la MEME
// sequence : seuls changent le typeIndex et le deserialiseur du default-state, que
// `padArchetype` porte. Deux copies de cette sequence auraient diverge au premier correctif —
// et celui du 2026-08-17 (largeurs MPP a reinstaller) montre que ce correctif arrive.
//
// HORS LIGNE : `decodeFilmPadScans` fait de l I/O disque sur tout le film et n est appelee que
// par `BuildFromFilm`. `attachWeaponPads` est PUR.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// padArchetype dit CE QUI CHANGE d'un archétype d'objet du monde à l'autre : son typeIndex, le
// mot qui le nomme au journal, et le balayage de créations qui sait dérouler SON default-state.
//
// LE NOM EST AU JOURNAL, ET IL LE FAUT : « ti=42 » n'est pas un message d'exploitation. Un
// build qui se tait sur les socles doit dire LEQUEL des deux calques s'est tu.
type padArchetype struct {
	ti    int
	label string
	scan  func(fc *grammar.FilmContext, wr *profile.Vec3Range, band map[uint32]bool) (
		[]types.EquipmentCreation, types.EquipmentCreationStats, error)
}

// groundWeaponArchetype / worldEquipmentArchetype : les deux voies de la chaîne des socles.
func groundWeaponArchetype() padArchetype {
	return padArchetype{
		ti: grammar.GroundWeaponTypeIndex, label: "armes au sol (ti=42)",
		scan: grammar.ScanGroundWeaponCreationsForBand,
	}
}

func worldEquipmentArchetype() padArchetype {
	return padArchetype{
		ti: grammar.EquipmentTypeIndex, label: "power-ups de socle (ti=37)",
		scan: grammar.ScanEquipmentCreationsForBand,
	}
}

// decodeFilmPadScans décode LES DEUX voies de la chaîne des socles sur le même film et aux mêmes
// largeurs de bloc MPP.
//
// LES DEUX SORTENT ENSEMBLE parce qu'elles entrent ensemble dans l'assemblage : les socles des
// deux natures se publient dans le MÊME `weaponPads`, et une voie décodée sans l'autre
// laisserait l'artefact affirmer « aucun socle de power-up » là où il faudrait dire « pas lu ».
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPadScans(
	fc *grammar.FilmContext, matchID string, wr *profile.Vec3Range, mpp profile.MPPWidths,
) PadScans {
	return PadScans{
		Weapons:  decodeFilmPadScan(fc, matchID, wr, mpp, groundWeaponArchetype()),
		Powerups: decodeFilmPadScan(fc, matchID, wr, mpp, worldEquipmentArchetype()),
	}
}

// decodeFilmPadScan décode UN archétype d'objet du monde et JOURNALISE ce qu'il en est.
//
// TROIS PANNES, TROIS PHRASES, et la distinction est le point. Un film sans l'archétype aux
// images-clés (aucune bande) n'est pas un film dont les créations sont illisibles, et ni l'un ni
// l'autre n'est un film dont les pistes delta manquent — ce dernier cas est le plus TRAÎTRE,
// parce qu'une piste absente rend `HasDelta` faux et ferait passer TOUTE apparition `spawned`
// pour un objet apparu au repos. Le calque se tait donc entièrement plutôt que de publier des
// socles fabriqués par une lecture manquante.
//
// LES LARGEURS DU BLOC MPP SONT CELLES DE CE FILM, ET C'EST UN CORRECTIF DE REVUE (2026-08-17).
// Le mot d'identité de 32 bits se lit derrière deux champs de largeur VARIABLE, mesurés par la
// calibration des poses `ti=37` sur le MÊME film (9/5 en Quick Play, 8/3 sur les films BTB
// mesurés) — et `ScanFilmEquipmentPlacements` les RESTAURE en sortant. Sans les réinstaller ici,
// le balayage lisait l'identité aux largeurs PAR DÉFAUT : sur un film calibré autrement, aucune
// création n'aurait résolu de famille, le calque aurait publié zéro socle, et rien ne l'aurait
// dit (découverte 8 du plan des armes au sol). Largeurs non mesurées (calibration refusée) : on
// garde le défaut, et le compteur `kept` de la couverture reste le témoin.
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPadScan(
	fc *grammar.FilmContext, matchID string, wr *profile.Vec3Range, mpp profile.MPPWidths,
	arch padArchetype,
) WorldObjectScan {
	defer gwInstallMPPWidths(fc, gwWidthsForFilm(fc, mpp))()
	kf := grammar.ScanWorldObjectKeyframes(fc.Film(), arch.ti)
	if len(kf.Band) == 0 {
		slog.Warn("socles : aucun slot de l archetype aux images-cles — rejeu sans ce calque",
			"archetype", arch.label, "match_id", matchID, "imagesCles", len(kf.TimesUS))
		return WorldObjectScan{}
	}
	cre, st, err := arch.scan(fc, wr, kf.Band)
	if err != nil {
		slog.Warn("socles : records de creation illisibles — rejeu sans ce calque",
			"archetype", arch.label, "err", err, "match_id", matchID)
		return WorldObjectScan{}
	}
	tracks, err := grammar.ScanWorldObjectsForBand(fc, wr, kf.Band)
	if err != nil {
		slog.Warn("socles : pistes delta illisibles — AUCUN socle publie (sans elles, toute"+
			" apparition passerait pour un objet apparu au repos)",
			"archetype", arch.label, "err", err, "match_id", matchID)
		return WorldObjectScan{}
	}
	slog.Info("socles : balayage d archetype",
		"archetype", arch.label, "slots", st.Slots, "ancres", st.Anchors, "acceptees", st.Accepted,
		"imagesCles", len(kf.TimesUS), "viesRecensees", len(kf.SeenUS), "pistesDelta", len(tracks))
	return WorldObjectScan{Scanned: true, Creations: cre, Stats: st, Keyframes: kf, Tracks: tracks}
}

// gwInstallMPPWidths installe les largeurs du bloc MPP MESURÉES sur ce film SUR LE CONTEXTE, et
// rend leur restauration. Largeurs non renseignées (calibration refusée) : rien n'est installé —
// l'invariant du profil vaut mieux qu'un découpage nul, qui ne lirait aucune identité du tout.
// gwWidthsForFilm rend les largeurs MPP a INSTALLER pour ce film : celles que porte sa VERSION
// DE FORMAT quand la grammaire les a relues chez l ecrivain (format 27), sinon les largeurs
// CALIBREES sur le film.
//
// C EST L ARBITRAGE DU 2026-09-15 (lot 1.9.1 bis, pas 3), et il a un nom des deux cotes : quand
// la lecture decide, la calibration n est plus qu un CONTROLE (le rapport publie ce qu elle
// mesure) ; quand il n y a pas de largeur relue, la calibration decide encore, et c est le repli
// `repli_largeurs_mpp_calibrees_sur_le_film` du registre (condition `format_sans_profil_relu`,
// ordre `apres_lecture` — le format se lit avant).
//
// LA CLE EST LA VERSION DE FORMAT, ET PLUS LE BUILD (revue de jalon M1, 2026-09-15, constat 2).
// Ce site resolvait `BuildProfileFromFilm`, donc la table des SEPT builds en dur : un film au
// format 27 dont le build en est absent — un patch du jeu qui ne change pas le format — tombait
// sur les largeurs calibrees alors que la grammaire etait disponible, et `formatSansProfil`
// rendait faux, donc ni compteur ni avertissement. Le repli tournait DEVANT une lecture (D14 b),
// et dans la meme cuisson les socles se decoupaient aux largeurs calibrees pendant que les poses
// d equipement se decoupaient aux largeurs relues. Les deux sites partagent desormais
// [grammar.MPPWidthsForFilm].
//
// MESURE AVANT LA BASCULE (cache, 657 films au 2026-09-15, `TestMPPResolutionCorpus`) : 6 films
// portent un build hors table (5 sans section d identification au format 20, 1 `HI_1_5_1` au
// format 23) et AUCUN d eux n est a un format dont la largeur est relue — zero octet cuit ne
// change sur ce cache. Le gain est de tenir le parc NEUF : au prochain build hors table au
// format 27, la lecture decide au lieu de la calibration.
func gwWidthsForFilm(fc *grammar.FilmContext, calibrees profile.MPPWidths) profile.MPPWidths {
	res := grammar.MPPWidthsForFilm(fc.Film())
	if res.Relue() {
		return res.Widths
	}
	// SITE 2 DU REPLI `repli_largeurs_mpp_calibrees_sur_le_film`, et il sert DEUX chemins de
	// cuisson (armes au sol et vehicules). Le compteur seul : l avertissement est emis une fois
	// par film par `avertirFormatSansProfil`.
	if res.FormatInconnu {
		publierFormatSansProfil(res.FormatVersion)
	}
	return calibrees
}

func gwInstallMPPWidths(fc *grammar.FilmContext, w profile.MPPWidths) func() {
	if !w.Valid() {
		return func() {}
	}
	prev := fc.PoserMPP(w)
	return func() { fc.PoserMPP(prev) }
}

// attachWeaponPads pose le calque des SOCLES sur le document : les socles des DEUX natures,
// leurs occupations achevees, leur couverture, et le journal qui porte les memes denominateurs
// que l artefact.
//
// LE NUAGE EST CELUI QUI N EST PAS DECIME, et il le faut : la datation d une disparition se joue
// a la frame, la decimation perdrait le passage de joueur qui la borne. Le calque publie les
// socles du MATCH — aucun catalogue de carte, aucun ramasseur.
//
// `positions` doit etre TRIE par instant (c est le cas de `sorted` dans BuildFromPositions).
// LES DEUX TABLES D IDENTITE viennent du manifeste du titre : les objets d objectif (que la voie
// des ARMES ecarte) et les familles d equipement (dont la voie des POWER-UPS tire sa
// selectivite). Tables vides = comportement d avant le 2026-08-18 / le 2026-08-19.
// Le retour est la liste des OBJETS INDIVIDUELS de la voie des armes : le calque des armes au
// sol (schéma 27) les consomme après coup — même chaîne, deux publications.
func attachWeaponPads(
	doc *ReplayDocument, scans PadScans, positions []grammar.BipedPosition, clock replayClock,
	cat LabelCatalog,
) []gwPickupObject {
	// L ADAPTATION DU CATALOGUE SE FAIT ICI, PAS DANS L ASSEMBLAGE : `buildWeaponPads` est PUR
	// et ne prend que les deux tables qu il consomme, jamais le catalogue entier du titre.
	var objs []gwPickupObject
	doc.WeaponPads, doc.PadPickups, doc.Coverage.GroundWeapons, objs = buildWeaponPads(
		scans, positions, clock,
		padCatalogs{ObjectiveObjects: cat.ObjectiveObjects, EquipmentFamilies: cat.EquipmentFamilies})
	logGroundWeaponCoverage(doc.Coverage.GroundWeapons)
	return objs
}
