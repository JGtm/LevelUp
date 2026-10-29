package replay

// porteurs_au_sync.go — LES INTERVALLES DE PORT D'OBJECTIF, LUS AU SYNC PAR L'ASSEMBLEUR DE LA CUISSON.
//
// # POURQUOI (plan `.ai/archive/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, décision V6, voie (b) retenue le 2026-09-28)
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
//	toutes   la table des sièges du film et les occupants par entité (le registre de la cuisson) ;
//	drapeau  statborg + bursts de capture, pont par manche, objets du monde
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

	"github.com/JGtm/LevelUp/libs/film/internal/constat"
	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/facts/objectives"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar/signaux"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
	"github.com/JGtm/LevelUp/libs/film/types"
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
	// ViesSansIdentite : les vies du document interne qu'aucune voie n'a nommées
	// (`coverage.bridge.unnamedLives`). Un défaut de nommage, jamais une donnée : non nul, il dit
	// qu'un portage de ces vies n'a pas de porteur.
	ViesSansIdentite int
	// Replis : le compteur des replis que la lecture et l'assemblage ont déclenchés (pose des
	// largeurs, calques, consultations des séries nommées et du pont par manche), ou nil quand
	// rien n'a été lu. Ce chemin n'écrit aucun document : sans ce champ, ces comptes mouraient ici
	// (revue finale P1-b). L'appelant les VERSE une fois à son propre compteur de passe — le
	// collecteur, `portagesDuMatch` — qui les publie avec les autres replis du film. Pointeur, et
	// pas rapport, pour que le bilan reste comparable.
	Replis *fallback.Compteur
}

// LecturesDesPorteurs dit quelles lectures du film ont été faites en plus de la passe de positions.
// `Identite` (la table des sièges et les occupants par entité, [lireSiegesEtOccupants]) se lit sous
// toute garde : c'est ce qui donne au document interne le registre d'identité de la cuisson.
type LecturesDesPorteurs struct {
	Identite, Statborg, ObjetsDuMonde, ArmesTenues bool
}

// PortagesAuSync rend les intervalles de port d'objectif par xuid, sur l'horloge du MATCH.
func PortagesAuSync(ctx context.Context, e EntreePorteursAuSync) (map[uint64][]IntervalleDePort, BilanPortages) {
	doc, b := e.DocumentInterne(ctx)
	if b.Gardes.Aucune() {
		return nil, b
	}
	portages := portagesDuDocument(doc, premierPaquetUS(e.Identite.Positions), &b)
	return portages, b
}

// DocumentInterne assemble le document que [PortagesAuSync] relit : celui de la cuisson, par
// [BuildFromPositions], sur les lectures du collecteur, la table des sièges du film et les occupants
// par entité ([lireSiegesEtOccupants], la lecture de la cuisson), puis les calques de la garde. Il
// n'est pas publié. Hors des quatre familles à porteur, rien n'est lu ni assemblé : document vide.
//
// Le bilan compte ses vies restées sans identité (`ViesSansIdentite`) : la même règle qu'à la
// cuisson, une vie est un humain ou un bot.
func (e EntreePorteursAuSync) DocumentInterne(ctx context.Context) (ReplayDocument, BilanPortages) {
	b := BilanPortages{Gardes: GardesDeLaVariante(e.Variante)}
	if b.Gardes.Aucune() {
		return ReplayDocument{}, b
	}
	fb := fallback.NouveauCompteur()
	b.Replis = fb
	poserProfilPuisCarte(ctx, e.Contexte, e.MatchID, Options{ProfilDeBalayage: e.ProfilDeBalayage, Fallbacks: fb})
	opt := e.optionsDuRegistre(fb, lireSiegesEtOccupants(ctx, e.Film, e.Contexte, e.MatchID))
	b.Lectures = e.lireLesPorteurs(ctx, b.Gardes, &opt)
	b.Lectures.Identite = true
	// Le document n'est pas publié : le titre n'y sert à rien, il reste vide, et le journal des places,
	// qui dit un défaut du roster publié, se tait (cf. Options.documentInterne).
	opt.documentInterne = true
	doc := BuildFromPositions(ctx, e.MatchID, "", e.Identite.Positions, nil, opt)
	// CE QUE LES LECTURES DU CONTEXTE ONT CONSTATE (grammar, lot J12.3 — ADR 0034 D-4) se journalise
	// ICI, sous le contexte de l appelant : le collecteur a deja releve celles du pont.
	JournaliserDiagnostics(ctx, e.Contexte.Diagnostics().Relever())
	if doc.Coverage != nil {
		b.ViesSansIdentite = doc.Coverage.Bridge.UnnamedLives
	}
	return doc, b
}

// optionsDuRegistre pose les entrées qui refont le registre de la cuisson : les lectures du
// collecteur, puis la table des sièges et les occupants que [lireSiegesEtOccupants] a lus.
func (e EntreePorteursAuSync) optionsDuRegistre(fb *fallback.Compteur, so siegesEtOccupants) Options {
	carte := e.Carte
	return Options{
		MapQuant: &carte, Labels: e.Libelles, Fallbacks: fb,
		RosterXUIDs: e.Identite.RosterXUIDs, Participants: e.Identite.Participants,
		Bots: e.Identite.Bots, Deaths: e.Identite.Deaths, PlayerIndices: e.Identite.PlayerIndices,
		BipedCreations: e.Identite.BipedCreations, ScoreboardTeams: e.Equipes,
		FilmTable: so.table, PlayerTeams: so.equipes, TeamScan: so.rapport, PlayerEntities: so.entites,
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
		monde := e.Carte.Range()
		_, poses := decodeFilmPlacements(ctx, e.Contexte, e.MatchID, &monde)
		opt.Pads = decodeFilmPadScans(ctx, e.Contexte, e.MatchID, &monde, poses.Calibration.Widths)
		lu.ObjetsDuMonde = true
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
//
// LES IMAGES-CLÉS SONT LUES POUR LEURS SEULES ARMES ([grammar.ScanArmesDesImagesCles]) : les mêmes
// armes que la cuisson, record pour record, sans l'inventaire ni la marque de portage que le sync ne
// lit pas, et sans les fenêtres qui les rendraient derrière la lecture.
func (e EntreePorteursAuSync) lireLesArmesTenues(ctx context.Context) []types.HeldWeaponChange {
	loadouts, err := grammar.ScanArmesDesImagesCles(e.Contexte, loadoutFamilies())
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
