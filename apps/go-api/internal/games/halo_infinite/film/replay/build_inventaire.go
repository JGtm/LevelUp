package replay

// build_inventaire.go — LES QUATRE DERNIERES PASSES DE L ASSEMBLAGE : les libelles et
// l inventaire, les capacites et les translocations, les impulsions et les charges, puis la
// cloture (le journal de couverture et les replis).
//
// DEPLACEMENT PUR depuis le corps de `BuildFromPositions` (lot 2.7 volet publication,
// 2026-09-16) : chaque passe est le bloc d instructions qu elle etait, dans le meme ordre, avec
// ses commentaires de mesure ; seule la designation des variables a change (`doc` -> `a.doc`).
// Voir `build.go` pour l ordre des passes et ce qu il protege.

import "log/slog"

// poserLibellesEtInventaire publie les tables de libelles, les morts sans revendication et le
// calque d inventaire avec sa couverture gardee.
func (a *assemblage) poserLibellesEtInventaire() {
	a.doc.WeaponLabels = buildWeaponLabels(a.doc.Loadouts, a.doc.Shots, a.doc.WeaponPads, a.opt.Labels)
	// La table weapon_key -> famille d'effet voyage telle quelle : les kills du feed sont
	// keyés par weapon_key (résolution base), pas par identifiant d'arme film — sans elle,
	// aucun effet de mort n'est joignable côté client.
	if len(a.opt.Labels.Effects) > 0 {
		a.doc.KillEffects = a.opt.Labels.Effects
	}
	// Les morts sans revendication ne sont publiées que pour les joueurs dont une trajectoire
	// l'est : le client déduit ces lignes DE SES PISTES, une entrée sans piste ne rencontrerait
	// jamais de ligne à décorer (même règle que les tirs, lancers et actions d'objectif).
	a.doc.NeutralDeaths = keepNeutralDeathsOfPublishedTracks(a.opt.NeutralDeaths, a.doc.Tracks, a.reg.PontEpure())
	builtInv, invDroppedOrigin := buildInventory(a.opt.Inventory, a.origin, a.step)
	a.doc.Inventory = keepInventoryOfPublishedTracks(builtInv, a.doc.Tracks)
	// COUVERTURE DU CALQUE INVENTAIRE (audit AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 5),
	// journalisée comme les autres calques — construction dans inventory.go, avec le type
	// qu'elle publie.
	//
	// L'AFFECTATION EST GARDÉE, comme celles des calques frères (Grapple, FlagCarries, Zones) :
	// quand l'appelant n'a RIEN fourni à lire — inventaire illisible, cf. le repli `inventory = nil`
	// de BuildFromFilm —, la couverture reste ABSENTE. Publier {0,0,0,0} affirmerait « lecture
	// faite, zéro trouvé », qui est le contraire de ce qui s'est passé ; l'ABSENCE dit encore autre
	// chose, et la doctrine de coverage.go repose sur cette distinction.
	attachInventoryCoverage(&a.doc, a.opt.Inventory, builtInv, invDroppedOrigin)
	// POURQUOI UNE LECTURE D'INVENTAIRE EST VIDE : le croisement avec le fil des morts se fait
	// ICI, où les morts et leur decalage d'horloge existent — pas dans le projecteur
	// (cf. inventory_dead_readings.go).
	logInventoryEmptyCoverage(a.doc.Inventory, markInventoryDeadReadings(a.doc.Inventory, a.opt.Deaths, a.reg,
		replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount, fb: a.opt.Fallbacks}))
	// LES GRENADES ONT LEUR PROPRE AXE, alimente par les deux canaux (cf. grenade_reads.go) :
	// ils n'ont pas la meme cadence, et les verser dans `Inventory` ferait masquer une lecture
	// pleine par une lecture partielle — la cellule de munitions se viderait.
	builtGren := buildGrenadeReads(a.opt.Inventory, a.opt.InventoryDeltas, a.origin, a.step)
	a.doc.GrenadeReads = keepGrenadeReadsOfPublishedTracks(builtGren, a.doc.Tracks)
	attachGrenadeReadCoverage(&a.doc, builtGren, a.opt.InventoryDeltaAmmoRefused)
}

// poserCapacitesEtTranslocations publie les rangs de grenade, les lectures de capacite, les
// changements d equipement, les teleportations, puis CLASSE la palette du match — l identite
// que la passe suivante emploie.
func (a *assemblage) poserCapacitesEtTranslocations() {
	// Les rangs de grenade sont publiés dès qu'un calque les référence : l'inventaire
	// (compteurs portés) OU les lancers (Grenade.Rank). Les conditionner au seul
	// inventaire laissait des lancers pointer une table absente.
	if len(a.doc.Inventory) > 0 || len(a.doc.Grenades) > 0 || len(a.doc.GrenadeReads) > 0 {
		a.doc.GrenadeLabels = a.opt.Labels.Grenades
	}
	// La capacite portee a son PROPRE calque : ses deux canaux ne vivent pas sur la meme
	// horloge (i48 dans les deltas, l'ancre dans les images-cles), et ils publient la MEME
	// grandeur — le rang de palette.
	//
	// LE BRUIT DE BALAYAGE EST ECARTE AVANT TOUT AUTRE FILTRE (RAPPORT_E0_2026-09-10 §3) : un
	// rang hors du domaine plausible d'une palette (abilityRankDomainMax) n'a pas sa place
	// dans le classement ni le nommage, et le rejet est COMPTE (AbilityCoverage.ScanNoise),
	// jamais muet.
	rawAbilities := buildAbilityReads(a.opt.AbilityRanks, a.opt.Inventory, a.origin, a.step)
	cleanAbilities, abilityNoise := rejectAbilityScanNoise(rawAbilities)
	a.doc.Abilities = keepAbilitiesOfPublishedTracks(cleanAbilities, a.doc.Tracks)
	abilityCov := buildAbilityCoverage(rawAbilities, cleanAbilities, a.doc.Abilities, abilityNoise)
	a.doc.Coverage.Abilities = &abilityCov
	logAbilityCoverage(abilityCov)
	// LA PALETTE SE CLASSE AVANT DE NOMMER, et un film ambigu ne recoit AUCUN nom : le
	// meme rang designe des capacites differentes d'une palette a l'autre.
	// LES RAMASSAGES ET LES CONSOMMATIONS d'equipement, sur le meme axe et avec les MEMES
	// rangs que doc.Abilities : c'est AbilityLabels qui les nomme, ou pas. Les annonces de
	// reapparition sont ECARTEES ici — ce ne sont pas des ramassages.
	ecChanges, ecCov := buildEquipmentChanges(
		a.opt.EquipmentChanges, a.opt.EquipmentChangeStats, a.origin, a.step)
	a.doc.EquipmentChanges = keepEquipmentChangesOfPublishedTracks(ecChanges, a.doc.Tracks)
	a.doc.Coverage.EquipmentChanges = &ecCov
	logEquipmentChangeCoverage(ecCov)
	// LES TÉLÉPORTATIONS du translocateur, datées par l'événement 117 — même axe, même
	// règle de publication que les autres calques (rien avant l'origine, rien sans piste).
	var trCov TranslocationCoverage
	a.doc.Translocations, trCov = buildTranslocations(a.opt.Translocations, a.doc.Tracks, a.origin, a.step)
	a.doc.Coverage.Translocations = &trCov
	logTranslocationCoverage(trCov)
	a.palette = classifyAbilityPalette(a.doc.Abilities, a.opt.Labels.Abilities)
	a.doc.AbilityLabels = abilityLabelsUsed(a.doc.Abilities, a.palette)
	slog.Info("rejeu : a.palette de capacites",
		"a.palette", paletteIDOrNone(a.palette), "lectures", len(a.doc.Abilities),
		"rangsNommes", len(a.doc.AbilityLabels))
}

// poserImpulsionsEtCharges publie les impulsions de capacite et les charges restantes, nommees
// par la palette classee a la passe precedente. Leurs couvertures ne se posent que si le
// balayage a tourne.
func (a *assemblage) poserImpulsionsEtCharges() {
	// LES IMPULSIONS DE CAPACITE, APRES la palette et non avant : leur identite est le RANG
	// i48 de la vie, et c'est la palette du match qui le nomme (le propulseur vaut 5 en
	// famille A et 21 en famille B). Un film non classe ne rend donc aucune impulsion —
	// mieux vaut muet que faux, la meme regle que `abilityLabelsUsed`.
	var aiCov AbilityImpulseCoverage
	a.doc.AbilityImpulses, aiCov = buildAbilityImpulses(abilityImpulseInputs{
		reads: a.opt.AbilityImpulses, stats: a.opt.AbilityImpulseStats, ranks: a.opt.AbilityRanks,
		lives: a.reg.Vies(), palette: a.palette, measured: a.opt.Labels.AbilityImpulseFamilies,
	}, a.doc.Tracks, a.origin, a.step)
	// LA COUVERTURE NE SE PUBLIE QUE SI LE BALAYAGE A TOURNE — patron `attachInventoryCoverage`
	// (inventory.go), et pour la raison qu'il documente : publier {0,0,0,...} affirmerait
	// « lecture faite, zero trouve », qui est le contraire de ce qui s'est passe quand le
	// balayage n'a jamais commence (BuildFromFilm degrade alors en `nil, AbilityImpulseStats{}`,
	// cf. son warn). L'ABSENCE de bloc dit encore autre chose, et c'est la distinction sur
	// laquelle repose toute la doctrine de coverage.go. Un balayage qui aboutit le pose, meme
	// vide et meme `componentAbsent`.
	if a.opt.AbilityImpulseStats.Scanned {
		a.doc.Coverage.AbilityImpulses = &aiCov
		logAbilityImpulseCoverage(aiCov)
	} else {
		slog.Warn("rejeu : impulsions de capacite NON BALAYEES — aucune couverture publiee",
			"lectures", len(a.opt.AbilityImpulses))
	}
	// LES CHARGES RESTANTES, par la MEME palette et la MEME jointure d'identite que les
	// impulsions (le rang i48 de la vie, nomme par la palette du match) — un film non classe
	// ne rend donc aucune lecture : mieux vaut muet que faux.
	var acCov AbilityChargeCoverage
	a.doc.AbilityCharges, acCov = buildAbilityCharges(abilityChargeInputs{
		reads: a.opt.AbilityCharges, stats: a.opt.AbilityChargeStats, ranks: a.opt.AbilityRanks,
		lives: a.reg.Vies(), palette: a.palette, measured: a.opt.Labels.AbilityChargeFamilies,
	}, a.doc.Tracks, a.origin, a.step)
	// LA COUVERTURE NE SE PUBLIE QUE SI LE BALAYAGE A TOURNE — le patron exact du bloc
	// ci-dessus (`attachInventoryCoverage`, et la lecon H1 de la seconde passe de revue P3) :
	// publier {0,0,...} quand le balayage n'a jamais commence affirmerait « lecture faite,
	// zero trouve », le contraire de ce qui s'est passe. Un balayage qui aboutit la pose,
	// meme vide et meme `componentAbsent`.
	if a.opt.AbilityChargeStats.Scanned {
		a.doc.Coverage.AbilityCharges = &acCov
		logAbilityChargeCoverage(acCov)
	} else {
		slog.Warn("rejeu : charges d equipement NON BALAYEES — aucune couverture publiee",
			"lectures", len(a.opt.AbilityCharges))
	}
}

// clore journalise la couverture par calque depuis l ARTEFACT, puis publie les replis. EN
// DERNIER : cf. fallbacks_publication.go.
func (a *assemblage) clore() {
	// LES CHIFFRES DU JOURNAL SONT CEUX DE L'ARTEFACT, et c'est `doc.Coverage.Shots` qu'il faut
	// lire, plus la copie locale : la SECONDE PORTE des tirs (`attachVehicleShots`) déplace des
	// événements de « sans slot » vers « rattachés » APRÈS que `buildCoverage` a figé la
	// couverture. Journaliser `shotCov` publierait un compte périmé à côté d'un artefact à jour.
	shotsPub := a.doc.Coverage.Shots
	slog.Info("rejeu : couverture par calque",
		"tirsRattaches", shotsPub.Attached, "tirsDisponibles", shotsPub.Available,
		"tirsSansSlot", shotsPub.NoSlot, "tirsAmbigus", shotsPub.Ambiguous,
		"tirsHorsFenetre", shotsPub.OutOfWindow, "tirsNonPublies", shotsPub.Unpublished,
		"grenadesRattachees", a.grenCov.Attached, "grenadesDisponibles", a.grenCov.Available,
		"verdictTirs", a.doc.Coverage.Verdict["shots"],
		"verdictGrenades", a.doc.Coverage.Verdict["grenades"],
		"verdictPont", a.doc.Coverage.Verdict["bridge"])
	attachFallbackCoverage(&a.doc, a.opt.Fallbacks) // EN DERNIER : cf. fallbacks_publication.go
}
