package replay

// build_from_film.go — LE DECODAGE D'UN FILM, ET RIEN QUE LUI.
//
// DEPLACEMENT PUR depuis `build.go` (lot 1 de PLAN_CUISSON_PERF, 2026-09-02), et c'est la
// resorption que le plan avait prevue (note N-G du §8). `build.go` melangeait deux travaux de
// nature differente : DECODER un film (~290 lignes de balayages sequentiels, un par calque) et
// ASSEMBLER un document a partir de positions deja decodees (`BuildFromPositions`, PUR et
// testable sans fichier). Le fichier pesait 928 lignes ; la separation rend chacun lisible seul.
// Le contenu est repris TEL QUEL, commentaires compris — aucune ligne de logique n'a change.
//
// C'EST ICI QUE LE FILM EST CONSOMME, ET NULLE PART AILLEURS DANS `replay` : les balayages
// recoivent un `*filmsource.Film` deja charge (une seule decompression par cuisson) et les
// enveloppes `dir` sont interdites en production (garde-rail
// `internal/archlint/no_film_reread_test.go`).
//
// GARDE-RAILS QUI LISENT CE FICHIER : `observe_test.go` (la liste fermee des etapes observees,
// dans l'ordre du source) et `world_object_precision_guard_test.go` (l'installation des largeurs
// d'axe sous le verrou). Les deux le parsent PAR SON NOM — s'il demenage, ils le suivent.

import (
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// BuildFromFilm décode les positions bipeds des SEULS chunks du film DEJA CHARGE et en
// assemble le document de rejeu 2D. Aucune entrée Cheat Engine.
//
// LE FILM EST UN PARAMETRE DEPUIS LE LOT 1 (2026-09-02, PLAN_CUISSON_PERF item 1.2), et c'est
// tout le gain : les ~20 balayages ci-dessous relisaient et redecompressaient le film ENTIER
// chacun leur tour. Ils consomment desormais le meme `*filmsource.Film`, charge UNE fois par
// l'appelant (`replaybuild.BuildBytes`). Aucun d'eux ne touche plus le disque.
//
// HORS LIGNE par construction — ne jamais appeler depuis un chemin de requête ; l'API sert
// l'artefact pré-construit.
func BuildFromFilm(matchID, titleSlug string, film *filmsource.Film, opt Options) (ReplayDocument, error) {
	if opt.MapQuant == nil {
		return ReplayDocument{}, fmt.Errorf("%w (match %s) : le document de rejeu exige l'entrée de catalogue de la carte",
			filmdec.ErrUnknownMapBounds, matchID)
	}
	// UN SEUL decodage filmdec a la fois par process (verrou de paquet partage avec
	// killsource.Decode) : les balayages ci-dessous lisent et ecrivent les globaux de
	// filmdec (dont compWidthObs, sans verrou propre). Tenu jusqu'au retour : l'assemblage
	// pur qui suit est negligeable devant le decodage, et relacher plus tot inviterait un
	// entrelacement entre deux sous-balayages du MEME film.
	release := filmdec.LockProcessDecode()
	defer release()
	// Les largeurs d'axe du chemin WORLD-OBJECT sont un global de paquet : installées ici,
	// sous le verrou, pour TOUT le decodage du film, et restaurees au retour.
	defer installWorldObjectPrecision(*opt.MapQuant, matchID)()
	worldRange := opt.MapQuant.Range()
	scan := filmdec.DefaultScanFilmOptions()
	if opt.Scan != nil {
		scan = *opt.Scan
	}
	scan.WorldRange = &worldRange
	// Le DÉCOUPAGE d'i0 vient du CATALOGUE, comme les bornes dont il est déduit — le
	// découpage lu dans le film (DetectI0Layout) est le contrôle, jamais l'entrée (doctrine
	// écrite sur WorldObjectPrecision, appliquée au bipède depuis le lot C catalogues
	// 2026-08-27 : sur une carte à plus de 2 régions, l'auto-détection lit l'index de
	// région comme un bit d'axe et le décodeur rejetterait tous les records — Live Fire).
	// Un opt.Scan qui force déjà son Layout (instruments) reste maître ; une entrée sans
	// largeurs (catalogue antérieur au champ) laisse l'auto-détection, comme le chemin
	// world-object laisse son défaut — jamais des largeurs nulles.
	if lay := opt.MapQuant.Layout(); scan.Layout == nil && lay.Valid() {
		scan.Layout = &lay
	}
	// Le cap de visée (Point.H) se lit dans le MÊME record que la position : la capture des
	// directions est donc toujours active pour l'artefact. Elle n'altère aucune position
	// (lecture seule après le vec3 d'i0).
	scan.CaptureDirs = true
	// LE CONTEXTE DU FILM EST OUVERT UNE FOIS, ICI, ET TOUS LES BALAYAGES LE PARTAGENT (lot 2
	// de PLAN_CUISSON_PERF, 2026-09-03). Il porte les trois derivations qui ne dependent que du
	// film — bande de slots bipede, decoupage d'i0 AUTO-DETECTE, registre chunk_00 — que huit,
	// six et douze balayages recalculaient chacun pour leur compte sur le film pourtant deja
	// charge. Il ne LIT rien a la construction : chaque derivation est calculee au premier
	// balayage qui la demande, donc a l'endroit exact ou elle l'etait avant (cf.
	// filmdec/film_context.go), et les suivants la lisent.
	//
	// LE DECOUPAGE DU CONTEXTE EST L'AUTO-DETECTE, PAS CELUI DU CATALOGUE ci-dessus : le lot 2
	// ne change aucune sortie. Faire lire `opt.MapQuant.Layout()` aux six balayages delta —
	// ce qui corrige Live Fire — est le lot 3, et lui seul (D3bis).
	fc := filmdec.NewFilmContext(film)
	// L'HORLOGE DES BALAYAGES PART ICI, et pas a l'entree de la fonction : ce qui precede est
	// l'attente du verrou process et la lecture du catalogue, qui ne sont le temps d'aucun
	// balayage. A partir d'ici, chaque `opt.observe` ferme le balayage qu'il annonce
	// (cf. observe.go).
	opt.clock = &stepClock{last: time.Now()}
	positions, err := filmdec.ScanBipedPositions(film, scan)
	if err != nil {
		return ReplayDocument{}, err
	}
	opt.observe("positions", positions)
	// Les tirs sont décodés du MÊME film et sur la MÊME horloge que les positions ; leur
	// absence n'est pas fatale (un film sans event de tir reste un rejeu valide).
	shots, err := filmdec.ScanFireEvents(film)
	if err != nil {
		slog.Warn("events de tir illisibles — rejeu sans tirs", "err", err, "match_id", matchID)
		shots = nil
	}
	opt.observe("fire", shots)
	// Armes portées : lues dans les keyframes du MÊME film, sur la MÊME horloge. Leur
	// absence n'est pas fatale (un rejeu sans armes reste un rejeu valide).
	loadouts, err := filmdec.ScanKeyframeLoadouts(film, loadoutFamilies())
	if err != nil {
		slog.Warn("keyframes illisibles — rejeu sans armes portées", "err", err, "match_id", matchID)
		loadouts = nil
	}
	opt.Loadouts = loadouts
	opt.observe("loadouts", loadouts)
	// PRISES ET LACHERS D'ARME : le composant d'identite d'arme n'entre au masque du flux
	// delta que lorsqu'un emplacement CHANGE (cf. filmdec/held_weapon_changes.go). Le
	// predicat de spawn vient des loadouts qu'on vient de lire : sans lui, la PREMIERE
	// emission d'un emplacement serait comptee comme une prise alors qu'elle peut n'etre que
	// la re-annonce d'une arme deja portee. Absence non fatale — le rejeu sort sans
	// ramassages, jamais avec des ramassages devines.
	weaponChanges, wStats, err := filmdec.ScanHeldWeaponChanges(fc, spawnSetFrom(loadouts))
	if err != nil {
		slog.Warn("changements d arme illisibles — rejeu sans ramassages", "err", err, "match_id", matchID)
		weaponChanges = nil
	} else {
		slog.Info("ramassage : changements d arme lus",
			"recordsDelta", wStats.Records, "masquePorteur", wStats.WithComponent,
			"emissions", wStats.Emissions, "repetitions", wStats.Repeats)
	}
	opt.WeaponChanges = weaponChanges
	opt.observe("heldWeaponChanges", weaponChanges)
	opt.observe("heldWeaponChanges.stats", wStats)
	// RAMASSAGES NATIFS : l'evenement `biped_pickup` de la liste d'evenements, en tete des
	// paquets delta. AUTRE SOURCE que le canal ci-dessus (qui lit un composant du bipede
	// PENDANT la traversee d'un record) : celui-ci lit des bits que personne d'autre ne lit,
	// avant la trame. Il date a la milliseconde ET nomme le ramasseur. Absence non fatale.
	pickups, pStats, err := filmdec.ScanBipedPickups(fc)
	if err != nil {
		slog.Warn("ramassages natifs illisibles — rejeu sans ramassages natifs", "err", err, "match_id", matchID)
		pickups, pStats = nil, filmdec.BipedPickupStats{}
	} else {
		slog.Info("ramassage natif : evenements lus",
			"paquets", pStats.Packets, "type9", pStats.Type9, "type8", pStats.Type8,
			"publies", pStats.Published, "listesMultiples", pStats.MultiEvent,
			"refusesSansRef", pStats.RefusedNoRef, "refusesSansIdentifiant", pStats.RefusedNoCatalog,
			"refusesHorsBande", pStats.RefusedOffBand, "refLargeInattendue", pStats.UnexpectedWideRef)
	}
	opt.Pickups, opt.PickupStats = pickups, pStats
	opt.observe("pickups", pickups)
	opt.observe("pickups.stats", pStats)
	// Inventaire complet : MÊMES images-clés, MÊME horloge, même record de biped que les armes
	// portées. Absence non fatale — un rejeu sans grenades reste un rejeu valide.
	inventory, invStats, err := ScanKeyframeInventory(film, loadoutFamilies(), 0)
	if err != nil {
		slog.Warn("inventaire illisible — rejeu sans grenades ni munitions", "err", err, "match_id", matchID)
		inventory = nil
	} else {
		slog.Info("inventaire : lectures de keyframe",
			"chunks", invStats.Chunks, "chunksIllisibles", invStats.ChunksUnread,
			"imagesCles", invStats.Keyframes, "records", invStats.Records,
			"grenadesParAncre", invStats.GrenadesByAnchor, "grenadesParPosition", invStats.GrenadesByPosition)
	}
	opt.Inventory = inventory
	opt.observe("inventory", inventory)
	opt.observe("inventory.stats", invStats)
	// Inventaire suivi dans les paquets DELTA : les compteurs de grenades (i22) et le jeu
	// selectionne (i47), transmis AU CHANGEMENT donc places la ou l'etat bouge. Absence non
	// fatale — l'axe des grenades retombe sur les seules images-cles.
	invDeltas, dStats, err := filmdec.ScanInventoryDeltas(fc)
	if err != nil {
		slog.Warn("inventaire delta illisible — grenades sans rafraichissement entre images-cles",
			"err", err, "match_id", matchID)
		invDeltas = nil
	} else {
		slog.Info("inventaire delta : lectures",
			"recordsDelta", dStats.Records, "masqueAvecI22", dStats.WithI22,
			"i22Lues", dStats.I22Read, "i22Implausibles", dStats.Implausible,
			"masqueAvecI47", dStats.WithI47, "i47Lues", dStats.I47Read,
			"accord", dStats.Accord, "accordVerifies", dStats.AccordChecked,
			"canalMunitionsRefuse", dStats.AmmoRefused)
	}
	opt.InventoryDeltas = invDeltas
	opt.InventoryDeltaAmmoRefused = dStats.AmmoRefused
	opt.observe("inventoryDeltas", invDeltas)
	opt.observe("inventoryDeltas.stats", dStats)
	// Identite de la capacite portee : lue dans les paquets DELTA, sur la MEME horloge. Rare
	// (une transmission par vie environ) mais elle porte le rang COMPLET, la ou les images-cles
	// ne voient que 16..23. Absence non fatale — le rejeu retombe sur cette seule fenetre.
	abilityRanks, aStats, err := filmdec.ScanAbilityRanks(fc)
	if err != nil {
		slog.Warn("identites de capacite illisibles — rejeu sans rang complet", "err", err, "match_id", matchID)
		abilityRanks = nil
	} else {
		slog.Info("capacites : lectures d i48",
			"recordsDelta", aStats.Records, "masqueAvecI48", aStats.WithI48,
			"lues", aStats.Read, "illisibles", aStats.Unread, "sansIdentite", aStats.Gated)
	}
	opt.AbilityRanks = abilityRanks
	opt.observe("abilityRanks", abilityRanks)
	opt.observe("abilityRanks.stats", aStats)
	// RAMASSAGES ET CONSOMMATIONS D'EQUIPEMENT : meme composant qu'au-dessus (i48), autre
	// question — non plus « que porte ce joueur » mais « que vient-il de ramasser ou d'user ».
	// Le temoin de NAISSANCE vient des positions BRUTES lues plus haut : sans lui, une
	// reapparition equipee serait comptee comme un ramassage, ce qui double le decompte sur
	// les modes ou les joueurs renaissent equipes. Absence non fatale.
	equipChanges, eStats, err := filmdec.ScanEquipmentChanges(fc, birthOfLives(positions))
	if err != nil {
		slog.Warn("changements d equipement illisibles — rejeu sans ramassages d equipement",
			"err", err, "match_id", matchID)
		equipChanges, eStats = nil, filmdec.EquipmentChangeStats{}
	} else {
		slog.Info("equipement : changements lus",
			"emissions", eStats.Walk.Read, "vies", eStats.Lives,
			"ramassages", eStats.Taken, "consommations", eStats.Spent,
			"reapparitions", eStats.Spawned, "manqueesEstimees", eStats.MissedEstimate)
	}
	opt.EquipmentChanges, opt.EquipmentChangeStats = equipChanges, eStats
	opt.observe("equipmentChanges", equipChanges)
	opt.observe("equipmentChanges.stats", eStats)
	// Etat du camouflage : la voie i28 queue[1], lue dans les paquets DELTA, sur la MEME
	// horloge (cf. filmdec/camo_state.go). Absence non fatale — le rejeu sort sans episodes
	// de camouflage, jamais avec des episodes devines.
	camoStates, cStats, err := filmdec.ScanCamoStates(fc)
	if err != nil {
		slog.Warn("etat de camouflage illisible — rejeu sans episodes de camo", "err", err, "match_id", matchID)
		camoStates = nil
	} else {
		slog.Info("camouflage : lectures d i28 queue[1]",
			"recordsDelta", cStats.Records, "masqueAvecI28", cStats.WithI28,
			"lues", cStats.Read, "illisibles", cStats.Unread, "sansVoie", cStats.NoChannel)
	}
	opt.CamoStates = camoStates
	opt.observe("camoStates", camoStates)
	opt.observe("camoStates.stats", cStats)
	// Evenements de grappin : le corps tag==3 d'i59, lu dans les paquets DELTA, sur la
	// MEME horloge (cf. filmdec/grapple_state.go). Absence non fatale — le rejeu sort sans
	// tractions de grappin, jamais avec des tractions devinees.
	grappleReads, gStats, err := filmdec.ScanGrappleReads(fc)
	if err != nil {
		slog.Warn("evenements de grappin illisibles — rejeu sans tractions", "err", err, "match_id", matchID)
		grappleReads = nil
	} else {
		slog.Info("grappin : lectures d i59 tag==3",
			"recordsDelta", gStats.Records, "masqueAvecI59", gStats.WithI59,
			"lues", gStats.Read, "illisibles", gStats.Unread,
			"tag3", gStats.Tag3, "corpsCasses", gStats.BodyBroken)
	}
	opt.GrappleReads = grappleReads
	opt.observe("grappleReads", grappleReads)
	opt.observe("grappleReads.stats", gStats)
	// POSES d'equipement : records de CREATION de l'archetype 37, sur la MEME horloge
	// (cf. equipment_placements.go — decodage, journal et refus y vivent ensemble).
	// LA LUNETTE (schema 24) : les bascules vivent dans la liste d'evenements en tete de
	// paquet, pas dans les records — un balayage separe, sans verrou (il ne touche aucun etat
	// global de decodage). Le maintien est borne : au-dela, on cesse d'affirmer plutot que de
	// prolonger une entree dont la sortie n'a pas ete lue (cf. filmdec.ZoomStateAt).
	// Reconstruction a plusieurs causes de fermeture (cf. zoom_state.go) ; les vies viennent
	// des positions deja balayees, aucune lecture supplementaire.
	zoomEvents := filmdec.ScanZoomEvents(film)
	// L'OBSERVATEUR PASSE APRES LA RECONSTRUCTION, ET C'EST VOULU : `buildScopedLookup` est un
	// O(n) sur les evenements qu'on vient de balayer, il appartient a l'etape `zoomEvents`.
	// Observe avant lui, l'horloge de observe() aurait impute son cout a l'etape SUIVANTE
	// (`placements`), qui ne fait pourtant rien de la lunette.
	opt.Scoped = buildScopedLookup(zoomEvents, buildLifeSpans(indexBySlot(positions)), zoomHoldUS)
	opt.observe("zoomEvents", zoomEvents)
	opt.Placements, opt.PlacementStats = decodeFilmPlacements(fc, matchID, &worldRange)
	opt.observe("placements", opt.Placements)
	opt.observe("placements.stats", opt.PlacementStats)
	// SOCLES : archetypes 42 (armes) et 37 (power-ups), sur la MEME horloge, AUX LARGEURS MPP que
	// la calibration des POSES vient de mesurer sur ce film (cf. build_ground_weapons.go).
	opt.Pads = decodeFilmPadScans(fc, matchID, &worldRange, opt.PlacementStats.Calibration.Widths)
	opt.observe("pads", opt.Pads)
	// MARQUEUR DE PORTAGE : le controle independant du calque du drapeau, lu aux images-cles du
	// MEME film — sur les seuls films de CTF (cf. build_objectives_live.go).
	opt.Flag.Marks = decodeFilmCarrierMarks(film, matchID, opt.Flag)
	opt.observe("carrierMarks", opt.Flag.Marks)
	// PROPRIETES RESEAU ti=13 : l'etat des zones (jauge de capture, proprietaire), lu dans les
	// paquets delta du MEME film — sur les seuls matchs dont l'appelant a fourni le catalogue de
	// zones (cf. build_zones.go).
	opt.Zone.Reads = decodeFilmZoneReads(fc, matchID, len(opt.Zone.Zones))
	opt.Zone.Scanned = len(opt.Zone.Zones) > 0
	opt.observe("zoneReads", opt.Zone.Reads)
	// ANNEAU D'ARMEMENT ti=12 : la jauge d'armement de la bombe, lue dans les paquets delta du
	// MEME film — sur les seuls matchs que l'appelant reconnait Assaut armable (cf.
	// bomb_armings.go ; jamais One Bomb, ou le canal ne tient pas).
	opt.Bomb.Reads = decodeFilmBombReads(fc, matchID, opt.Bomb)
	opt.observe("bombReads", opt.Bomb.Reads)
	// Lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME horloge.
	// Absence non fatale, comme les tirs et les armes portées.
	grenades, err := filmdec.ScanGrenadeThrows(film)
	if err != nil {
		slog.Warn("paquets delta illisibles — rejeu sans lancers de grenade", "err", err, "match_id", matchID)
		grenades = nil
	}
	opt.Grenades = grenades
	opt.observe("grenades", grenades)
	// Trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
	proj, err := filmdec.ScanProjectiles(film, &worldRange)
	if err != nil {
		slog.Warn("projectiles illisibles — rejeu sans trajectoires", "err", err, "match_id", matchID)
		proj = nil
	}
	opt.Projectiles = proj
	opt.observe("projectiles", proj)
	// Le fil des morts NOMME les vies. Sans lui, le pont est vide et NI les tirs NI les lancers
	// ne sont publiés : ce n'est pas une dégradation cosmétique, d'où un warn explicite.
	deaths, err := ScanDeaths(film)
	if err != nil {
		slog.Warn("fil des morts illisible — aucun tir ni lancer ne sera publie",
			"err", err, "match_id", matchID)
		deaths = nil
	}
	opt.Deaths = deaths
	opt.observe("deaths", deaths)
	// L'index de joueur SE LIT dans le film (cf. player_index.go) : le roster vient du fil des
	// morts, et les 5 bits qui précèdent chaque xuid donnent son index. Sans cette table, aucun
	// tir ni lancer n'est publié — comme sans le fil des morts.
	if len(deaths) > 0 {
		idx, err := ScanPlayerIndices(film, rosterFromDeaths(deaths))
		if err != nil {
			slog.Warn("index de joueur illisible — aucun tir ni lancer ne sera publie",
				"err", err, "match_id", matchID)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			slog.Warn("index de joueur NON INJECTIF — table ecartee",
				"collisions", collisions, "match_id", matchID)
		}
		opt.PlayerIndices = table
	}
	opt.observe("playerIndices", opt.PlayerIndices)
	// L'origine d'horloge du film : deux en-têtes de paquet, aucune estimation (cf.
	// origin.go). Son absence n'est pas fatale — le document sort sans origine, et le
	// client retombe sur l'appariement.
	clockUS, err := ScanClockOrigin(film)
	if err != nil {
		slog.Warn("origine d'horloge illisible — rejeu sans origine publiee", "err", err, "match_id", matchID)
		clockUS = 0
	}
	opt.FilmClockOriginUS = clockUS
	opt.observe("clockOrigin", clockUS)
	return BuildFromPositions(matchID, titleSlug, positions, shots, opt), nil
}
