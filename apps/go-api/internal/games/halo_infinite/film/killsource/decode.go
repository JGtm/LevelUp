package killsource

// decode.go — L API PUBLIQUE : UNE FONCTION.
//
// Tout le reste du paquet est prive. C est deliberé : le branchement doit tenir en quelques
// lignes, et un appelant ne doit avoir aucun moyen de composer les etapes dans un autre ordre —
// l ORDRE EST LE RESULTAT (la marche decide, le scan rattrape, l auto-infligee ne comble que les
// trous, la mort de bot est une population a part).
//
// SERIALISATION OBLIGATOIRE. Les parametres de replication du decodeur de bits sont des GLOBAUX
// DE PAQUET. Deux decodages simultanes dans le meme process se contamineraient, et deux
// decodages successifs aussi si les globaux n etaient pas remis a leur valeur d origine. Le
// verrou et la remise a zero sont donc a l entree, pas a la charge de l appelant.

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// Erreurs rendues par [Decode]. Elles se testent avec `errors.Is`.
var (
	// ErrNoChunk : la source ne rend aucun chunk.
	ErrNoChunk = errors.New("killsource: aucun chunk")
	// ErrNoPacket : aucun paquet de replication : le film n est pas exploitable.
	ErrNoPacket = errors.New("killsource: aucun paquet de replication (type 0)")
	// ErrNoKillFeed : aucun chunk HIGHLIGHT porteur d evenements `kill`. Sans kill-feed il n y a
	// pas de verite de credit, donc pas de bijection indice -> joueur, donc rien de publiable.
	ErrNoKillFeed = errors.New("killsource: kill-feed introuvable (aucun chunk HIGHLIGHT)")
	// ErrRegistry : le chunk 0 ne porte pas un registre ECS lisible.
	ErrRegistry = errors.New("killsource: registre ECS illisible")
)

func errRegistry(err error) error { return fmt.Errorf("%w: %w", ErrRegistry, err) }

// La serialisation vit dans `filmdec.LockProcessDecode` (verrou de PAQUET, 2026-08-13) :
// le rejeu 2D decode les memes globaux dans le meme process, et deux verrous locaux ne se
// protegent pas l'un de l'autre.

// decodeCtx : l etat d une passe. Il n est jamais rendu a l appelant.
type decodeCtx struct {
	name      string
	opts      Options
	film      *film
	feed      *killFeed
	roster    *roster
	walkRes   *walkResult
	scanCands []candidate
	mult      map[multKey]int
	calib     calibration
	bijScore  int
}

// Decode : LA fonction publique. Elle lit les chunks d un film et rend, pour chaque mort, LES
// DEUX VERITES — le credit que le jeu affiche, et la source du degat fatal — plus le drapeau de
// leur divergence, la provenance technique de chaque ligne, les denominateurs nommes et la
// metrique de sante.
//
// `opts` peut valoir nil : c est alors la configuration GELEE, celle qui a produit les chiffres
// publies. `name` sert uniquement a etiqueter la mesure de sante (identifiant de film ou de
// match, au choix de l appelant).
func Decode(ctx context.Context, name string, src ChunkSource, opts *Options) (*Result, error) {
	o := DefaultOptions()
	if opts != nil {
		o = *opts
	}
	o.normalize()

	release := filmdec.LockProcessDecode()
	defer release()
	resetGlobals()

	c := &decodeCtx{name: name, opts: o}
	if err := c.prepare(ctx, src); err != nil {
		return nil, err
	}
	return c.finish(), nil
}

// resetGlobals : remet les parametres de replication du decodeur de bits a leur valeur d origine.
// SANS CELA, enchainer deux films dans le meme process fait demarrer la calibration du second
// depuis les valeurs du premier — mesure : le score d un film passe de 1111 a 1214 selon l ordre
// d appel.
func resetGlobals() {
	filmdec.SetAbsoluteAxisW(14)
	filmdec.SetRecordStateParam(0)
	filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{6, 6, 6}}
	filmdec.SetStrictGeneration(true)
	filmdec.SetMobilityActionBodyPorted(true)
}

// prepare : les cinq etapes qui precedent la publication. Aucune ne consulte l arme.
func (c *decodeCtx) prepare(ctx context.Context, src ChunkSource) error {
	var err error
	if c.film, err = loadFilm(src); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if c.feed, err = loadKillFeed(c.film); err != nil {
		return err
	}
	c.roster = buildRoster(c.feed, loadBotMeta(c.film), c.opts.Bots)

	tl, err := newTimeline(c.film)
	if err != nil {
		return err
	}
	tl.rewind()
	c.calib = calibrate(c.film, tl, c.opts.Views)
	if err = ctx.Err(); err != nil {
		return err
	}
	c.walkRes = runWalk(c.film, tl, c.roster, c.opts.Views)
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
	stats.Assist = c.attachAssists(kills, scanKillEvents(c.film))
	res := &Result{
		Kills:           kills,
		UnclaimedDeaths: p.unclaimed,
		Coverage:        cov,
		Stats:           stats,
		Roster:          c.roster.public(),
		Calibration:     c.calib.String(),
		BijectionMargin: bijectionMargin(c.roster, c.feed.pairs, c.scanCands, c.bijScore),
	}
	res.Health = c.health(p, cov)
	// La sonde a porte relachee ne tourne QUE si la couverture est incomplete : c est le seul
	// regime ou elle porte de l information, et elle coute cher sur un gros film.
	if cov.Covered < cov.RealPairs {
		probe := c.relaxedProbe(coveredInstants(kills))
		res.Probe = &probe
		res.Health.TagOutOfCatalogueScan = probe.Uncovered
	}
	return res
}

// coverage : LES DENOMINATEURS, chacun nomme. LES DEUX POPULATIONS DE BOT SONT COMPTEES A PART :
// ni l une ni l autre n a JAMAIS pu entrer dans les couples reconstruits (le kill-feed est
// humain-seul), donc les additionner a `Covered` fabriquerait un taux qui n existe nulle part —
// et un numerateur qui deborderait son denominateur des le premier film a bot.
func (c *decodeCtx) coverage(kills []Kill, p *pass) Coverage {
	ghost := c.ghostPairs()
	cov := Coverage{
		ReconstructedPairs: len(c.feed.pairs),
		GhostPairs:         len(ghost),
		SameInstantPairs:   len(c.feed.real),
		FeedKills:          c.feed.nKills,
		FeedDeaths:         c.feed.nDeaths,
		BotDeaths:          p.botStats.Published,
		BotKillerDeaths:    p.botKillerStats.Published,
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
