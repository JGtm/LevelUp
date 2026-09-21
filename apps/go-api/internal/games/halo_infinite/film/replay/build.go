package replay

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// DefaultFrameIntervalMS est le pas de la grille de rééchantillonnage, en millisecondes.
// Les records du film arrivent à ~60 Hz par entité : à 100 ms (10 Hz) le rendu reste
// fluide (le client interpole) tout en divisant le volume de points par ~4.
const DefaultFrameIntervalMS = 100

// coordScale arrondit les coordonnées au centimètre : le quantum du décodeur est de
// ~1,4 cm, deux décimales ne perdent donc rien et allègent nettement le JSON.
const coordScale = 100

// BuildFromPositions assemble le document à partir de positions déjà décodées. PUR
// (aucune I/O) : c'est le cœur testable de l'assemblage.
//
// TIMELINE : les positions portent l'horodatage du paquet en MICROSECONDES ; elles sont
// rééchantillonnées sur une grille uniforme de FrameIntervalMS relative au premier paquet.
// Point.T est donc l'index de frame, et FrameIntervalMS donne son échelle réelle.
//
// L'ASSEMBLAGE EST UNE SUITE DE PASSES, ET L'ORDRE EN EST LE CONTRAT (lot 2.7 volet
// publication, 2026-09-16). Cette fonction faisait 474 lignes d'affilée ; elle en appelle
// maintenant quatorze, dans le MEME ordre, sur le MEME état porté par [assemblage]. Aucune
// passe ne s'est deplacee, aucun appel n'a change d'argument : la preuve en est l'equivalence
// octet pour octet du corpus fige (D4 du PLAN_DECODEUR_FILM). Ce que chaque passe a besoin de
// la precedente est ecrit en tete de chacune ; ce que l'ordre protege est ecrit ici :
//
//	les pistes AVANT tout calque      un calque ne publie que ce qui a une trajectoire publiee ;
//	le registre AVANT les equipes     l'equipe se pose sur des vies deja nommees ;
//	la couverture APRES les mesures   `doc.Coverage` n'existe qu'a `composerLaCouverture`, et
//	                                  les calques qui mesurent avant elle posent leur compte
//	                                  ensuite (projectiles, equipes, sieges) ;
//	les tirs embarques APRES les vehicules, la palette AVANT les impulsions, les replis EN
//	DERNIER (`clore`).
func BuildFromPositions(matchID, titleSlug string, pos []grammar.BipedPosition,
	fire []grammar.FireEvent, opt Options) ReplayDocument {
	a := &assemblage{matchID: matchID, opt: opt, pos: pos, fire: fire}
	if !a.ouvrir(titleSlug) {
		a.poserLesCalquesProduits()
		return a.doc
	}
	a.poserLesPistes()
	a.poserLesEquipesEtLeRoster()
	a.poserTirsProjectilesEtGrenades()
	a.poserScoreEtObjectifs()
	a.poserEpisodesDEquipement()
	a.composerLaCouverture()
	a.poserGrappinEtPoses()
	a.poserPrisesEtSocles()
	a.poserArmesAuSolEtVehicules()
	a.poserObjectifsVivants()
	a.poserLibellesEtInventaire()
	a.poserCapacitesEtTranslocations()
	a.poserImpulsionsEtCharges()
	a.clore()
	// LES CALQUES PRODUITS EN DERNIER, APRES `clore` : la table se lit sur le document FINI, et
	// la seconde porte des tirs comme les replis ecrivent encore pendant les passes precedentes
	// (cf. layers.go). Le document sans aucune piste la pose aussi, dans son retour anticipe :
	// sans quoi `layers` absent porterait DEUX sens, celui que le schema 62 ferme.
	a.poserLesCalquesProduits()
	return a.doc
}

// assemblage porte l'état que les passes de [BuildFromPositions] se transmettent.
//
// POURQUOI UNE STRUCTURE ET PAS DES PARAMETRES : vingt-neuf valeurs voyagent d'une passe a
// l'autre — le depot borne une fonction a cinq parametres, et les enchainer en retours ferait
// des signatures de dix valeurs dont l'ordre serait le seul garde-fou. Elle n'est PAS un objet
// de metier : elle ne porte aucune regle, seulement ce que la passe precedente a produit.
//
// ELLE NE SORT JAMAIS DE CE FICHIER ET DE SES PASSES : `BuildFromPositions` rend `a.doc` par
// VALEUR, exactement comme avant.
type assemblage struct {
	// Les entrées de l'appelant, jamais modifiées après `ouvrir` (sauf `opt.Fallbacks`, que
	// `ouvrir` garantit non nil — cf. Options.Fallbacks, D14).
	matchID string
	opt     Options
	pos     []grammar.BipedPosition
	fire    []grammar.FireEvent

	// Le document en construction, et l'horloge sur laquelle toutes les passes posent.
	doc      ReplayDocument
	interval int
	sorted   []grammar.BipedPosition
	origin   uint64
	step     uint64

	// Le registre d'identité et ce que la pose des pistes en a tiré.
	reg            IdentityRegistry
	trackCov       TrackCoverage
	unnamed        unnamedLivesReport
	equipes        teamPublication
	viesTotal      int
	viesNommees    int
	viesSlotAmbigu int
	siegeCov       SeatCoverage

	// Les mesures faites AVANT `composerLaCouverture` et posées par elle.
	shotCov     LayerCoverage
	grenCov     LayerCoverage
	shotOrphans []orphanShot
	projCov     *ProjectileCoverage
	teamCov     TeamCoverage
	// stanceCov est la couverture des ETATS DE MOUVEMENT (schema 65) : mesuree au pliage des
	// pistes, posee dans `doc.Coverage` par `composerLaCouverture`.
	stanceCov StanceCoverage
	objCov    LayerCoverage
	scoreCov  *ScoreCoverage

	// Ce que les passes suivantes se repassent.
	clock            scoreClock
	clotureesParMort map[int]bool
	killsRead        bool
	gwObjs           []gwPickupObject
	palette          *AbilityPalette
}

// ouvrir pose le document vide, trie les positions et établit l'horloge. Rend `false` quand le
// film ne porte aucune position : l'artefact est alors publié tel quel, sans aucun calque —
// c'est le tout premier `return doc` de l'ancienne fonction, mot pour mot.
func (a *assemblage) ouvrir(titleSlug string) bool {
	a.interval = a.opt.frameIntervalMS()
	a.opt.Fallbacks = a.opt.compteurDeReplis() // cf. Options.Fallbacks (D14) : jamais nil a partir d'ici
	a.doc = ReplayDocument{
		SchemaVersion:   SchemaVersion,
		MatchID:         a.matchID,
		TitleSlug:       titleSlug,
		FrameIntervalMS: a.interval,
		Geometry:        a.opt.Geometry,
		GeometryBounds:  geometryBounds(a.opt.Geometry),
		Structure:       a.opt.Structure,
		StructureBounds: surfaceBounds(a.opt.Structure),
	}
	if len(a.pos) == 0 {
		return false
	}
	a.sorted = append([]grammar.BipedPosition(nil), a.pos...)
	sort.SliceStable(a.sorted, func(i, j int) bool { return a.sorted[i].TimestampUS < a.sorted[j].TimestampUS })

	a.origin = a.sorted[0].TimestampUS
	a.step = uint64(a.interval) * 1000
	a.doc.FrameCount = frameSpan(a.sorted, a.origin, a.step)
	a.doc.DurationMS = a.doc.FrameCount * a.interval
	return true
}

// fireRefs réduit les événements de tir à ce que les fermetures ont le droit de connaître : QUI
// et QUAND. L'arme et la visée sont volontairement laissées dehors — elles n'ont aucun pouvoir
// de désignation, et les rendre visibles à la fermeture rouvrirait la porte au vote supprimé le
// 2026-07-28.
func fireRefs(fire []grammar.FireEvent) []FireEventRef {
	out := make([]FireEventRef, len(fire))
	for i, e := range fire {
		out[i] = FireEventRef{FilmIndex: e.FilmIndex, TimestampUS: e.TimestampUS}
	}
	return out
}

// keepShotsOfPublishedTracks écarte les tirs dont le slot n'a pas de trajectoire publiée
// (track trop courte) : le client n'aurait rien à quoi les rattacher.
func keepShotsOfPublishedTracks(shots []Shot, tracks []Track) []Shot {
	return keepOfPublishedTracks(shots, tracks,
		func(s Shot, published map[uint32]bool) bool { return published[s.Slot] })
}

// frameSpan renvoie le nombre de frames couvrant tout le film (dernier index + 1).
func frameSpan(sorted []grammar.BipedPosition, origin, step uint64) int {
	last := sorted[len(sorted)-1].TimestampUS
	return int((last-origin)/step) + 1
}
