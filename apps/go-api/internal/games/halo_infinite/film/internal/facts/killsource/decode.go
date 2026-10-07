package killsource

// decode.go — L API PUBLIQUE : UNE FONCTION.
//
// Tout le reste du paquet est prive. C est deliberé : le branchement doit tenir en quelques
// lignes, et un appelant ne doit avoir aucun moyen de composer les etapes dans un autre ordre —
// l ORDRE EST LE RESULTAT (la marche decide, le scan rattrape, l auto-infligee ne comble que les
// trous, la mort de bot est une population a part).
//
// LE PROFIL DE BALAYAGE EST RENDU, PLUS LAISSE DANS LE PROCESSUS (lot 2.3). La calibration
// map-dependante retient des largeurs — descripteur de traversee, largeur d axe absolue,
// `param_4` — dont la cuisson du rejeu heritait par l etat du processus (decouverte D1 du lot
// 2.2.a). [Result.ProfilCalibre] les PORTE : `replaybuild` le passe a `replay.BuildFromFilm`,
// qui le pose sur le contexte du film.

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Erreurs rendues par [Decode]. Elles se testent avec `errors.Is`.
var (
	// ErrNoChunk : le film ne porte aucun chunk (film nil compris).
	ErrNoChunk = errors.New("killsource: aucun chunk")
	// ErrNoPacket : aucun paquet de replication : le film n est pas exploitable.
	ErrNoPacket = errors.New("killsource: aucun paquet de replication (type 0)")
	// ErrNoKillFeed : aucun chunk HIGHLIGHT porteur d evenements `kill`. Sans kill-feed il n y a
	// pas de verite de credit, donc pas de bijection indice -> joueur, donc rien de publiable.
	ErrNoKillFeed = errors.New("killsource: kill-feed introuvable (aucun chunk HIGHLIGHT)")
	// ErrRegistry : le chunk 0 ne porte pas un registre ECS lisible.
	ErrRegistry = errors.New("killsource: registre ECS illisible")
	// ErrCarteAbsente : aucune entree de catalogue PORTANT DES LARGEURS n a ete fournie
	// (`Options.Carte` nil, ou entree sans largeurs d axe). Le film est MIS DE COTE, jamais decode
	// aux largeurs d une autre carte (regle utilisateur du 2026-09-27 : « le flux du film est la
	// seule source fiable. Pas de repli. »). Meme famille que la cle de film inconnue (D-4
	// d ADR 0034) : le film est la, ce qui manque est une donnee cote depot.
	ErrCarteAbsente = errors.New("killsource: carte du match absente — film mis de cote, " +
		"jamais decode aux largeurs d une autre carte")
)

func errRegistry(err error) error { return fmt.Errorf("%w: %w", ErrRegistry, err) }

// LA SERIALISATION DE PAQUET A DISPARU AU LOT 2.3 : il n y a plus d etat partage a proteger.
// Ce decodage et la cuisson du rejeu qui le suit dans le meme processus portent chacun leur
// profil de balayage et leur observation ; ce que l'un lit, l'autre ne peut plus l'ecraser.

// decodeCtx : l etat d une passe. Il n est jamais rendu a l appelant.
type decodeCtx struct {
	name      string
	opts      Options
	film      *film
	feed      *killFeed
	roster    *roster
	walkRes   *walkResult
	scanCands []candidate
	// killEvents : les kill-events (code 85) du film, localises UNE fois. Ils servent DEUX
	// lectures qui ne se ressemblent pas : le COUPLE (tueur, victime) d un kill sans mort en
	// face (lot 1.9.3, `feed_couples.go`) et l ASSISTANT (`assist.go`).
	killEvents *assistScan
	// couples : d ou vient le couple de chaque instant du kill-feed.
	couples  types.CoupleStats
	mult     map[multKey]int
	calib    calibration
	bijScore int
	// diag : les DIAGNOSTICS du decodage (lot J12.3, ADR 0034 D-4) — ce que ce paquet journalisait,
	// rendu a l orchestrateur par [Result.Diagnostics].
	diag constat.Diagnostics
}

// Decode : LA fonction publique. Elle lit un film DEJA CHARGE et rend, pour chaque mort, LES
// DEUX VERITES — le credit que le jeu affiche, et la source du degat fatal — plus le drapeau de
// leur divergence, la provenance technique de chaque ligne, les denominateurs nommes et la
// metrique de sante.
//
// `film` est charge par l appelant (`source.LoadDir` depuis le cache disque,
// `source.Load(source.MemoryChunks(...), meta)` depuis des chunks telecharges) : depuis le
// lot 1 de PLAN_CUISSON_PERF (item 1.4), ce paquet ne lit plus le disque et ne decompresse plus
// rien lui-meme, et une cuisson qui decode aussi le rejeu 2D partage LE MEME film. `film` nil ou
// vide rend [ErrNoChunk].
//
// `opts` porte la configuration GELEE ([DefaultOptions]) PLUS LA CARTE DU MATCH. LA CARTE EST
// OBLIGATOIRE depuis le 2026-09-27 (regle utilisateur : « le flux du film est la seule source
// fiable. Pas de repli. ») : sans entree de catalogue portant des largeurs — `opts` nil compris —
// [Decode] rend [ErrCarteAbsente] AVANT de lire quoi que ce soit, et l appelant met le film de
// cote. Seul un instrument de recherche decode sans carte, et il le NOMME
// ([Options.RechercheSansCarte]). `name` sert uniquement a etiqueter la mesure de sante
// (identifiant de film ou de match, au choix de l appelant).
func Decode(ctx context.Context, name string, film *source.Film, opts *Options) (*Result, error) {
	o := DefaultOptions()
	if opts != nil {
		o = *opts
	}
	o.normalize()
	if err := exigerLaCarte(o); err != nil {
		return nil, err
	}

	c := &decodeCtx{name: name, opts: o}
	if err := c.prepare(ctx, film); err != nil {
		return nil, err
	}
	return c.finish(), nil
}

// ProfilDeDepart rend le PROFIL DE BALAYAGE dont ce paquet part, avant toute calibration :
// l invariant du profil, plus la GENERATION STRICTE.
//
// LE `param_4` N EN FAIT PLUS PARTIE (lot 5.1.7, 2026-09-18). Ce profil forcait le `param_4` du
// moteur a ZERO, et la calibration le remplacait ensuite par la valeur d un balayage de 0 a 5 —
// le repli `repli_parametre_etat_record_infere`, dont la cible de retrait etait « lot qui
// trouvera la source LUE de `param_4` (registre ECS par composant, ou table du build) ». Elle
// est trouvee : `param_4` EST le niveau que le registre du film porte par composant
// (`grammar.Archetype.Level`, cf. `grammar/component_param4.go`). Il ne se force plus, il se lit,
// et le balayage a disparu avec le champ qu il ecrivait.
// LA GENERATION STRICTE EN FAIT PARTIE, et c est le second fait de production que ce lot rend
// explicite : `resetGlobals` levait `SetStrictGeneration(true)` pour tout le PROCESSUS et ne le
// rabaissait jamais — la cuisson du rejeu qui suivait decodait donc, elle aussi, en generation
// stricte. Le profil le porte desormais, et `replaybuild` le passe comme le reste.
//
// `SetMobilityActionBodyPorted(true)` a disparu sans rien changer : c etait deja le defaut, et
// le seul ecrivain contraire est un instrument de mesure qui pose desormais SON profil.
func ProfilDeDepart() grammar.ProfilDeBalayage {
	p := grammar.ProfilDeBalayageParDefaut()
	p.Grammaire.GenerationStricte = true
	return p
}

// ProfilDeDepartPourCarte rend le PROFIL DE DEPART sous l entree de catalogue de la CARTE du
// match : l invariant de [ProfilDeDepart], plus les largeurs d axe et la largeur d index de
// plage que la carte impose au chemin absolu de position ([profile.MapQuantEntry.PrecisionAbsolue]).
//
// `carte` nil, ou une entree sans largeurs (catalogue anterieur au champ, entree fabriquee a la
// main), LAISSE l invariant et le second rendu dit `false`. En production ce cas n atteint plus
// la calibration : [Decode] le refuse ([ErrCarteAbsente], 2026-09-27). Il ne reste ouvert qu aux
// instruments de recherche ([Options.RechercheSansCarte]) — l invariant est l entree
// `cliffhanger` du catalogue, c est-a-dire UNE carte appliquee a toutes.
//
// C EST LE MEME GESTE QUE `replay.installWorldObjectPrecision`, PAR LE MEME APPEL
// (`PoserLargeursObjetDuMondeDepuisDecoupage`) : les deux chemins de decodage du depot posent
// desormais la carte de la meme facon, et il n y a pas deux regles a maintenir.
//
// LA PRESENCE DE LA CARTE SE LIT SUR L ENTREE ([carteApplicable]), JAMAIS SUR UNE DIFFERENCE DE
// LARGEURS (lot J7.7, constat FK-7, puis 2026-09-27) : sur Cliffhanger, dont l entree EST
// l invariant, rien ne change — et la carte passait pour absente.
func ProfilDeDepartPourCarte(carte *profile.MapQuantEntry) (grammar.ProfilDeBalayage, bool) {
	p := ProfilDeDepart()
	if !carteApplicable(carte) {
		return p, false
	}
	// Les replis de la pose ne se comptent pas ICI : [carteApplicable] vient d exiger des largeurs
	// d axe non nulles, et la cuisson pose la MEME carte sur son contexte, qui compte l index de
	// region (lot J8.7 ; `repli_carte_absente_largeurs_par_defaut` est retire depuis le 2026-09-27,
	// carte obligatoire).
	_ = p.PoserLargeursObjetDuMondeDepuisDecoupage(carte.Layout())
	return p, true
}

// avertirReplisDeCalibration DIT ce que la calibration n a pas lu — jamais de degradation
// silencieuse (CLAUDE.md regle 3).
//
//	carte absente (RECHERCHE SEULEMENT)            les largeurs conservees sont celles d UNE
//	                                               carte (l entree `cliffhanger` du catalogue).
//	                                               En production [Decode] refuse ce cas avant la
//	                                               calibration ([ErrCarteAbsente]).
//	repli_controle_corruption_section_absente      le film ne porte pas de section
//	                                               d identification, donc pas le bit de
//	                                               `chunk_00 + 0x0CB45C` qui decide du `R(1)`
//	                                               de garde par composant : l invariant tient.
//
// EXTRAITE DE `prepare` AU LOT 5.18.2 : le second avertissement y portait la fonction a 84
// lignes, au-dela du seuil de 80 du depot (ratchet `archlint/film_function_length_test.go`).
func (c *decodeCtx) avertirReplisDeCalibration() {
	if !c.calib.CarteLue {
		c.signaler(DiagRechercheSansCarte, constat.NiveauWarn,
			"killsource: RECHERCHE SANS CARTE — la marche des morts lit ses "+
				"positions aux largeurs d axe PAR DEFAUT, celles d une autre carte",
			"film", c.name, "largeurs", c.calib.LueAxisW, "indexW", c.calib.LueIndexW)
	}
	if !c.calib.ControleDeCorruptionLu {
		c.signaler(DiagControleCorruptionAbsent, constat.NiveauWarn,
			"killsource: le film ne declare pas son controle de corruption par "+
				"composant (pas de section d identification) — la grammaire garde son invariant",
			"film", c.name)
	}
}

// prepare : les cinq etapes qui precedent la publication. Aucune ne consulte l arme.
func (c *decodeCtx) prepare(ctx context.Context, src *source.Film) error {
	var err error
	if c.film, err = loadFilm(src); err != nil {
		return err
	}
	if !c.film.versionLue {
		// Le film ne porte pas son registre (`chunk_00`) : la version reste inconnue et le
		// kill-feed se lit avec le decoupage historique « gamertag en tete ». Sur un film de
		// version 39-40 cela rend un roster effondre — silence interdit, cf. CLAUDE.md n 3.
		c.signaler(DiagVersionIllisible, constat.NiveauWarn, "killsource: version de film illisible, decoupage historique",
			"film", c.name, "film_major_version", c.film.majorVersion)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if c.feed, err = loadKillFeed(c.film); err != nil {
		return err
	}
	// LA TABLE DU FILM AVANT LE ROSTER : c est elle qui porte le lien direct
	// `index de joueur <-> gamertag` (lot 1.8, film_table.go). Un refus est NOMME, journalise
	// ici, et le decodeur retombe alors sur l inference entiere.
	table := readFilmTable(c.film)
	if table.Refusal != FilmTableRead {
		c.signaler(DiagTableNonLue, constat.NiveauWarn, "killsource: table des joueurs du film NON LUE — la bijection "+
			"retombe entierement sur l inference par les votes du kill-feed",
			"film", c.name, "build", table.Build, "cause", string(table.Refusal))
	}
	// PUIS LE MOTIF DU XUID (lot 5.2b.1, index_motif.go) : la table de `chunk_00` est ecrite a
	// L OUVERTURE du film, donc elle ignore les REMPLACANTS. Les cinq bits qui precedent le motif
	// du xuid dans les chunks de replication, eux, les portent — c est la lecture que le rejeu
	// publie sous le nom `PlayerIndexTable`, et c est la MEME table d identite : une seule, pas
	// une deuxieme liste.
	motif := lireIndexParMotif(c.film, table.slots, c.feed)
	if motif.desaccords > 0 || motif.absents > 0 || motif.tueursEcartes > 0 {
		c.signaler(DiagLienParMotif, constat.NiveauDebug, "killsource: lien par motif de xuid",
			"film", c.name, "lectures", motif.lectures, "epingles", len(motif.nomParIndex),
			"desaccords", motif.desaccords, "absents", motif.absents, "tueurs_ecartes", motif.tueursEcartes)
	}
	// LES BOTS ET LEUR EQUIPE, LUE DANS LEUR ENTREE BOT_METADATA au build que la table a lu.
	c.roster = buildRoster(c.feed, c.botsDuFilm(table.Build), c.opts.Bots, table, motif)
	if n := len(c.roster.unpinned); n > 0 {
		// FK-1 (lot J7.2) : une perte PUBLIEE (`Coverage.BotsNonEpingles`) et DITE, jamais muette.
		c.signaler(DiagBotsNonEpingles, constat.NiveauWarn, "killsource: bot(s) NON EPINGLE(S) — leur slot tombe sur un siege que "+
			"la table du film NOMME (un humain le tient), ou hors des 32 indices ; leurs morts et "+
			"celles qu ils infligent ne se publient pas",
			"film", c.name, "bots", n, "borne_humains", c.roster.borneHumains)
	}
	// LE COUPLE (TUEUR, VICTIME) SE LIT AU KILL-EVENT 85 (lot 1.9.3), et il se lit ICI : la
	// decomposition du kill-feed exige les kill-events et le roster EPINGLE, et la bijection
	// exige la decomposition. L ordre est donc force, et il est le resultat — resoudre les
	// couples APRES la bijection les ferait dependre de l inference qu ils alimentent.
	c.killEvents = scanKillEvents(c.film)
	c.couples = c.feed.resoudreCouples(c.killEvents.recs, c.roster)

	if err = c.marcher(ctx, src); err != nil {
		return err
	}
	c.scanCands = scanFilm(c.film, c.roster.nPlay)
	if err = ctx.Err(); err != nil {
		return err
	}
	// LA BIJECTION EST AJUSTEE SUR LES CANDIDATS DU SCAN, pas sur les lignes publiees : elle
	// doit exister AVANT toute publication, sinon elle serait ajustee sur ses propres reponses.
	var score int
	c.roster.perm, score = solveBijection(c.roster, c.feed.pairs, c.scanCands, c.opts.BijectionRestarts)
	c.bijScore = score
	return nil
}

// finish : la passe hybride, les denominateurs, la sante.
func (c *decodeCtx) finish() *Result {
	p := c.run()
	kills := p.kills()
	cov := c.coverage(kills, p)
	stats := p.stats(c.walkRes)
	// L ASSISTANT SE LIT AILLEURS QUE LA SOURCE, et le decodage reste separe jusqu au bout : la
	// liste d evenements pour l assistant, la boucle de records pour la source. Un echec de l un
	// ne doit rien retirer a l autre.
	stats.Assist = c.attachAssists(kills, c.killEvents)
	stats.Couples = c.couples
	res := &Result{
		Kills:           kills,
		UnclaimedDeaths: p.unclaimed,
		Coverage:        cov,
		Stats:           stats,
		Roster:          c.roster.public(),
		Calibration:     c.calib.String(),
		ProfilCalibre:   c.calib.Profil,
		BijectionMargin: bijectionMargin(c.roster, c.feed.pairs, c.scanCands, c.bijScore),
		// DETERMINEE quand l inference n avait qu UNE SEULE affectation a rendre — ce que
		// `Inferred <= 1` ne suffisait pas a dire des lors qu il peut rester plus de noms libres
		// que d indices libres (cf. [FilmTablePinning.AffectationUnique]).
		BijectionDetermined: c.roster.table.AffectationUnique(),
	}
	res.Health = c.health(p, cov)
	// La sonde a porte relachee ne tourne QUE si la couverture est incomplete : c est le seul
	// regime ou elle porte de l information, et elle coute cher sur un gros film.
	if cov.Covered < cov.RealPairs {
		probe := c.relaxedProbe(coveredInstants(kills))
		res.Probe = &probe
		res.Health.TagOutOfCatalogueScan = probe.Uncovered
	}
	res.Stats.Replis = c.replisDuResultat(kills, p.unclaimed, res.Probe != nil)
	res.Diagnostics = c.diag.Relever()
	return res
}

// coverage : LES DENOMINATEURS, chacun nomme. LES DEUX POPULATIONS DE BOT SONT COMPTEES A PART :
// ni l une ni l autre n a JAMAIS pu entrer dans les couples reconstruits (le kill-feed est
// humain-seul), donc les additionner a `Covered` fabriquerait un taux qui n existe nulle part —
// et un numerateur qui deborderait son denominateur des le premier film a bot.
//
// LES COUPLES FANTOMES — des couples RECOLLES dont le kill est en realite une mort de BOT — SORTENT
// DU DENOMINATEUR : ce ne sont pas des morts manquees, ce sont des morts qui N EXISTENT PAS (la
// reconstruction a pris la victime du voisin, verifie en Theater). DEPUIS LE LOT J7.4 (FK-4), un
// couple n est fantome que si le temps 4 a PUBLIE sa mort de bot ([pass.fantomes]) : c est ce qui
// tient `Covered <= RealPairs` — une ligne humaine publiee a cet instant resterait sinon au
// numerateur apres le retrait de son couple, et masquerait une mort manquee ailleurs.
func (c *decodeCtx) coverage(kills []Kill, p *pass) Coverage {
	cov := Coverage{
		ReconstructedPairs: len(c.feed.pairs),
		GhostPairs:         len(p.fantomes),
		SameInstantPairs:   len(c.feed.real),
		FeedKills:          c.feed.nKills,
		FeedDeaths:         c.feed.nDeaths,
		BotDeaths:          p.botStats.Published,
		BotKillerDeaths:    p.botKillerStats.Published,
		BotsNonEpingles:    len(c.roster.unpinned),
	}
	cov.RealPairs = cov.ReconstructedPairs - cov.GhostPairs
	for _, k := range kills {
		if k.Read.Origin != OriginBot && k.Read.Origin != OriginBotKiller {
			cov.Covered++
		}
	}
	return cov
}

// coveredInstants : les instants deja publies, pour la sonde.
func coveredInstants(kills []Kill) map[int]bool {
	m := make(map[int]bool, len(kills))
	for _, k := range kills {
		m[k.TimeMS] = true
	}
	return m
}

// marcher : la calibration, puis la marche des morts, sur le contexte du film ouvert sous la carte du
// match — la marche des trames de la grammaire, sous le profil calibre et le decoupage MPP que la
// grammaire resout pour le film. Ce que le registre, la phase des images-cles et la marche
// constatent tombe dans les diagnostics.
func (c *decodeCtx) marcher(ctx context.Context, src *source.Film) error {
	fc := grammar.NewFilmContextForMap(src, c.opts.Carte, nil)
	reg, err := fc.Registry()
	if err != nil {
		return errRegistry(err)
	}
	if d, ok := grammar.DiagnosticRegistreInconnu(reg); ok {
		c.diag.Signaler(d)
	}
	if c.calib, err = calibrate(fc, c.opts.Views, c.opts.Carte); err != nil {
		return err
	}
	fc.Diagnostics().Verser(&c.diag)
	c.avertirReplisDeCalibration()
	if err = ctx.Err(); err != nil {
		return err
	}
	c.signalerLeDecoupageMPP(poserLeProfil(fc, c.calib.Profil))
	if c.walkRes, err = marcherLesMorts(fc, c.film, c.roster); err != nil {
		return err
	}
	fc.Diagnostics().Verser(&c.diag)
	return nil
}

// poserLeProfil pose le profil `p` sur le contexte de la marche, puis le decoupage du bloc MPP que la
// grammaire resout pour le film ([grammar.FilmContext.ResolutionMPP] : celui de sa version de format,
// ou celui qu il declare), comme la cuisson le pose apres son profil
// (`replay.poserLeDecoupageMPPDuFilm`) ; elle rend la resolution. Rien de resolu : le contexte garde
// le decoupage du profil. Le profil calibre ne porte pas ce decoupage : la cuisson le pose elle-meme.
func poserLeProfil(fc *grammar.FilmContext, p grammar.ProfilDeBalayage) grammar.ResolutionMPP {
	fc.PoserProfilDeBalayage(p)
	res := fc.ResolutionMPP()
	if res.Decide() {
		fc.PoserMPP(res.Widths)
	}
	return res
}
