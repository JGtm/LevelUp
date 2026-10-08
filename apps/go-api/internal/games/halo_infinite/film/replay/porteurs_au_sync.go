package replay

// porteurs_au_sync.go — LES INTERVALLES DE PORT D'OBJECTIF, LUS AU SYNC PAR L'ASSEMBLEUR DE LA CUISSON.
//
// # POURQUOI (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, décision V6, voie (b) retenue le 2026-09-28)
//
// Le placement d'une vie (`placement_des_vies.go`) écarte les instants où le joueur PORTE
// l'objectif. Ce fait n'existait qu'à la cuisson du rejeu ; le plan le veut au sync, par le
// collecteur, sur le film qu'il a déjà ouvert. Le lot V0 a mesuré deux voies : relire le canal des
// armes tenues (fidélité 61,95 %, rejetée) ou appeler les LECTURES DE LA CUISSON et produire les
// calques par l'ASSEMBLEUR DE PRODUCTION (fidélité 100 %, surcoût accepté par l'utilisateur). C'est
// cette seconde voie, et rien d'autre, que ce fichier câble :
//
//	aucun calque n'est réécrit   les quatre calques sortent de [BuildFromPositions], la même
//	                             fonction que la cuisson, nourrie des positions et des lectures
//	                             du registre du COLLECTEUR ;
//	aucun pont n'est recopié     le pont par manche est [PontParManche], celui de la cuisson ;
//	aucune garde n'est recopiée  [GardesDeLaVariante] et les constructeurs d'entrée
//	                             (`porteurs_entrees.go`) sont ceux que `replaybuild` appelle.
//
// # CE QUI SE LIT, SELON LA GARDE
//
//	drapeau  statborg + bursts de capture, pont par manche, équipes du film, objets du monde
//	         (poses, puis socles à leurs largeurs calibrées : d'où les vies libres qui ferment un
//	         lâcher volontaire — sans elles la fidélité tombe à 55 %, mesure V0) ;
//	crâne    statborg, pont par manche ;
//	VIP      statborg (la couronne résout son pont elle-même) ;
//	bombe    le canal des armes tenues (images-clés d'armes, dotations de naissance, changements).
//
// Hors de ces quatre familles, RIEN n'est lu : la garde rend avant le premier octet.
//
// # CE QUI SORT
//
// Les portages par xuid, en millisecondes du MATCH (bornes incluses), l'horloge du collecteur. Les
// frames du document se convertissent par [matchClock.matchMSOfFrame], avec l'origine du document
// (premier paquet de position) et SON calage (`coverage.bridge.deathOffsetMs`). Pour le drapeau,
// seul l'état `carried` compte : `carried_open` est une BORNE HAUTE (aucun lâcher daté), pas une
// mesure — la compter retirerait de la mesure du temps où le joueur ne porte peut-être plus rien.
// Elle est comptée au bilan.

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/signaux"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// EntreePorteursAuSync porte ce que le collecteur a en main après sa passe de positions.
type EntreePorteursAuSync struct {
	MatchID string
	// Film : le film déjà chargé par le collecteur.
	Film *source.Film
	// Contexte : le contexte que la passe de positions a ouvert sur ce film. Le profil calibré y
	// est POSÉ ici, puis les largeurs de la carte — l'ordre de la cuisson (`poserProfilPuisCarte`).
	Contexte *grammar.FilmContext
	// Carte : l'entrée du catalogue de bornes de la carte du match.
	Carte profile.MapQuantEntry
	// Variante : `game_variant_name` du match — la garde de mode.
	Variante string
	// ProfilDeBalayage : le profil que `killsource` a calibré sur ce film (`ProfilCalibre`).
	ProfilDeBalayage *grammar.ProfilDeBalayage
	// Identite : les lectures qui ont construit le registre du collecteur (positions, créations,
	// fil des morts, index, roster, participants, bots). L'assembleur en refait le MÊME registre.
	Identite IdentityInput
	// Lignes : la feuille du match (xuid, frags, morts, assistances), pour le pont par manche.
	Lignes []types.PlayerLine
	// Equipes : l'equipe de chaque participant de la feuille (`match_participants.team_id`, bots
	// compris sous `bid(N.0)`), posee en `Options.ScoreboardTeams` comme a la cuisson : elle tranche
	// la contradiction d'un bot entre son entite et sa declaration (occupants_equipe_arbitree.go).
	Equipes map[string]int
	// Socles : les socles de drapeau de la carte (catalogue d'objectifs), comme à la cuisson.
	Socles []FlagSpawn
	// Libelles : le catalogue de libellés du titre (objets d'objectif du drapeau).
	Libelles LabelCatalog
	// lireStatborg : le lecteur du statborg, [objectives.StatRecordsAvecReplis] quand nil. NON
	// EXPORTE : seul le test des replis du sync y substitue des comptes, sans film.
	lireStatborg func(*source.Film, string) ([]types.StatRecord, bool, objectives.ComptesDesReplis, []constat.Diagnostic)
}

// BilanPortages dit ce que la lecture a fait, pour que l'appelant le compte et le journalise.
type BilanPortages struct {
	// Gardes : les familles lues. Toutes fausses = rien n'a été lu.
	Gardes GardesDesPorteurs
	// Lectures : ce que la garde a fait PAYER — le coût du match que l'appelant journalise.
	Lectures LecturesDesPorteurs
	// SansCalage : le document n'a pas de calage d'horloge — aucun portage ne peut être daté.
	SansCalage bool
	// Intervalles : les portages rendus.
	Intervalles int
	// XUIDIllisibles : portages dont le porteur n'est pas un xuid décimal (écartés).
	XUIDIllisibles int
	// DrapeauOuverts : portages `carried_open` (borne haute) écartés.
	DrapeauOuverts int
	// Replis : le compteur des replis que la lecture et l'assemblage ont déclenchés (pose des
	// largeurs, calques, consultations des séries nommées et du pont par manche), ou nil quand
	// rien n'a été lu. Ce chemin n'écrit aucun document : sans ce champ, ces comptes mouraient ici
	// (revue finale P1-b). L'appelant les VERSE une fois à son propre compteur de passe — le
	// collecteur, `portagesDuMatch` — qui les publie avec les autres replis du film. Pointeur, et
	// pas rapport, pour que le bilan reste comparable.
	Replis *fallback.Compteur
}

// LecturesDesPorteurs dit quelles lectures du film ont été faites en plus de la passe de positions.
type LecturesDesPorteurs struct {
	Statborg, Equipes, ObjetsDuMonde, ArmesTenues bool
}

// PortagesAuSync rend les intervalles de port d'objectif par xuid, sur l'horloge du MATCH.
func PortagesAuSync(ctx context.Context, e EntreePorteursAuSync) (map[uint64][]IntervalleDePort, BilanPortages) {
	b := BilanPortages{Gardes: GardesDeLaVariante(e.Variante)}
	if b.Gardes.Aucune() {
		return nil, b
	}
	fb := fallback.NouveauCompteur()
	b.Replis = fb
	poserProfilPuisCarte(ctx, e.Contexte, e.MatchID, Options{ProfilDeBalayage: e.ProfilDeBalayage, Fallbacks: fb})
	opt := e.optionsDuRegistre(fb)
	b.Lectures = e.lireLesPorteurs(ctx, b.Gardes, &opt)
	// Le document n'est pas publié : le titre n'y sert à rien, il reste vide, et le journal des places,
	// qui dit un défaut du roster publié, se tait (cf. Options.documentInterne).
	opt.documentInterne = true
	doc := BuildFromPositions(ctx, e.MatchID, "", e.Identite.Positions, nil, opt)
	// CE QUE LES LECTURES DU CONTEXTE ONT CONSTATE (grammar, lot J12.3 — ADR 0034 D-4) se journalise
	// ICI, sous le contexte de l appelant : le collecteur a deja releve celles du pont.
	JournaliserDiagnostics(ctx, e.Contexte.Diagnostics().Relever())
	portages := portagesDuDocument(doc, premierPaquetUS(e.Identite.Positions), &b)
	return portages, b
}

// optionsDuRegistre pose les entrées qui refont le registre du collecteur.
func (e EntreePorteursAuSync) optionsDuRegistre(fb *fallback.Compteur) Options {
	carte := e.Carte
	return Options{
		MapQuant: &carte, Labels: e.Libelles, Fallbacks: fb,
		RosterXUIDs: e.Identite.RosterXUIDs, Participants: e.Identite.Participants,
		Bots: e.Identite.Bots, Deaths: e.Identite.Deaths, PlayerIndices: e.Identite.PlayerIndices,
		BipedCreations: e.Identite.BipedCreations, ScoreboardTeams: e.Equipes,
	}
}

// lireLesPorteurs fait les lectures que chaque famille gardée exige, et pose leurs entrées.
func (e EntreePorteursAuSync) lireLesPorteurs(ctx context.Context, g GardesDesPorteurs, opt *Options) LecturesDesPorteurs {
	var lu LecturesDesPorteurs
	var recs []types.StatRecord
	var bursts []int
	var replisDuStatborg objectives.ComptesDesReplis
	if g.Drapeau || g.Crane || g.VIP {
		var diags []constat.Diagnostic
		recs, _, replisDuStatborg, diags = e.statborg()(e.Film, e.MatchID)
		JournaliserDiagnostics(ctx, diags)
		lu.Statborg = true
		bursts = signaux.CaptureBurstTimes(e.Film)
	}
	// UN SEUL ENREGISTREUR DES REPLIS A LA CONSULTATION pour le pont et les calques du document,
	// comme a la cuisson (lot J8.7-bis) : le pont le passe a sa resolution, les calques le relisent
	// dans les options.
	opt.ReplisHorsBalayage.Consultations = opt.enregistreurDesConsultations()
	pont := NouveauPontParManche(recs, deathInstantsOf(e.Identite.Deaths), e.Lignes, opt.consultations())
	if lu.Statborg {
		// LES REPLIS DU STATBORG ET DE LA CONSTRUCTION DU PONT (revue finale, 2026-10-02), comme a la
		// cuisson (`replaybuild`, `replisObjectifs`) : poses dans les options, ils sont verses UNE fois
		// au compteur, a la cloture de l assemblage (`versementDeLAssemblage`). Ils etaient jetes.
		opt.ReplisHorsBalayage.Objectifs = replisDuStatborg.Plus(pont.Identite().ComptesDesReplis())
	}
	if g.Drapeau {
		opt.Flag = EntreeDuDrapeau(recs, bursts, pont)
		opt.Flag.Spawns = e.Socles
		opt.PlayerTeams, opt.TeamScan, opt.PlayerEntities = grammar.ScanPlayerTeams(e.Contexte)
		monde := e.Carte.Range()
		_, poses := decodeFilmPlacements(ctx, e.Contexte, e.MatchID, &monde)
		opt.Pads = decodeFilmPadScans(ctx, e.Contexte, e.MatchID, &monde, poses.Calibration.Widths)
		lu.Equipes, lu.ObjetsDuMonde = true, true
	}
	opt.Skull = EntreeDuCrane(recs, g.Crane, pont)
	opt.Vip = EntreeDeLaCouronne(recs, g.VIP)
	if g.Bombe {
		// L'horloge du manifeste ne sert qu'a l'ARMEMENT, que le sync ne lit pas : sans elle, le
		// calque d'armement se tait, et le portage ne lit que la garde.
		opt.Bomb = EntreeDeLaBombe(nil, true)
		opt.WeaponChanges = e.lireLesArmesTenues(ctx)
		lu.ArmesTenues = true
	}
	return lu
}

// lireLesArmesTenues lit le canal des armes tenues dans l'ordre de la cuisson : images-clés
// d'armes, dotations de naissance, puis changements qualifiés par elles (`spawnSetFrom`).
// Absence non fatale, journalisée : la bombe sort alors sans portage.
func (e EntreePorteursAuSync) lireLesArmesTenues(ctx context.Context) []types.HeldWeaponChange {
	loadouts, _, err := grammar.ScanKeyframeLoadoutsMarche(e.Contexte, loadoutFamilies())
	if err != nil {
		slog.WarnContext(ctx, "porteurs au sync : images-cles d'armes illisibles", "match_id", e.MatchID, "err", err)
		loadouts = nil
	}
	creations := e.Identite.BipedCreations
	births, _, err := grammar.ScanBirthLoadouts(e.Contexte, creations)
	if err != nil {
		slog.WarnContext(ctx, "porteurs au sync : dotations de naissance illisibles", "match_id", e.MatchID, "err", err)
		births = nil
	}
	held, _, err := grammar.ScanHeldWeaponChanges(e.Contexte, spawnSetFrom(loadouts, births, creations))
	if err != nil {
		slog.WarnContext(ctx, "porteurs au sync : changements d'arme illisibles — bombe sans portage",
			"match_id", e.MatchID, "err", err)
		return nil
	}
	return held
}

// premierPaquetUS : l'origine de l'axe de frames d'un document construit sur ces positions (le
// plus petit horodatage, cf. `assemblage.ouvrir`).
func premierPaquetUS(pos []grammar.BipedPosition) uint64 {
	var premier uint64
	for i := range pos {
		if ts := pos[i].TimestampUS; premier == 0 || ts < premier {
			premier = ts
		}
	}
	return premier
}

// portagesDuDocument relit les quatre calques de porteur d'un document, en ms du MATCH.
func portagesDuDocument(doc ReplayDocument, origine uint64, b *BilanPortages) map[uint64][]IntervalleDePort {
	if doc.Coverage == nil || doc.Coverage.Bridge.DeathOffsetMs == nil {
		b.SansCalage = true
		return nil
	}
	h := matchClock{origin: origine, step: uint64(doc.FrameIntervalMS) * 1000,
		deathOffsetMS: *doc.Coverage.Bridge.DeathOffsetMs}
	out := map[uint64][]IntervalleDePort{}
	poser := func(xuid string, t0, t1 int) {
		x, err := strconv.ParseUint(xuid, 10, 64)
		if err != nil {
			b.XUIDIllisibles++
			return
		}
		out[x] = append(out[x], IntervalleDePort{DebutMS: h.matchMSOfFrame(t0), FinMS: h.matchMSOfFrame(t1)})
		b.Intervalles++
	}
	for _, f := range doc.FlagCarries {
		for _, s := range f.Spans {
			switch {
			case s.State == FlagStateCarried && s.XUID != nil:
				poser(*s.XUID, s.T0, s.T1)
			case s.State == FlagStateCarriedOpen:
				b.DrapeauOuverts++
			}
		}
	}
	for _, c := range doc.SkullCarries {
		poser(c.XUID, c.T0, c.T1)
	}
	for _, c := range doc.BombCarries {
		poser(c.XUID, c.T0, c.T1)
	}
	for _, c := range doc.VipCrown {
		poser(c.XUID, c.T0, c.T1)
	}
	for _, l := range out {
		// Comparateur TOTAL (DT-9, fusion J11.6) : deux portages de meme debut se departagent par
		// leur fin ; deux portages egaux sur les deux bornes sont indiscernables.
		slices.SortFunc(l, func(a, b IntervalleDePort) int {
			return cmp.Or(cmp.Compare(a.DebutMS, b.DebutMS), cmp.Compare(a.FinMS, b.FinMS))
		})
	}
	return out
}

// statborg rend le lecteur du statborg de l'entree ([objectives.StatRecordsAvecReplis] par defaut).
func (e EntreePorteursAuSync) statborg() func(*source.Film, string) (
	[]types.StatRecord, bool, objectives.ComptesDesReplis, []constat.Diagnostic,
) {
	if e.lireStatborg != nil {
		return e.lireStatborg
	}
	return objectives.StatRecordsAvecReplis
}
