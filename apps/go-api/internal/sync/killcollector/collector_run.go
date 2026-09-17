package killcollector

// collector_run.go — L EXECUTION D UNE PASSE SUR UN MATCH : le point d entree `CollectMatch`,
// l enchainement `collect`, et les quatre ecritures qu'il declenche.
//
// Sorti de `collector.go` par deplacement pur au lot 2.7 (2026-09-16, scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change, a une extraction pres, signalee en
// tete de `decodeFilmForMatch`. `collector.go` garde le TYPE, son injection de dependances et
// ses etats ; ce fichier porte ce qu'il FAIT.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/haloclient"
)

// CollectMatch : LE point d entree, un match a la fois.
//
// Il journalise a l ENTREE et a la SORTIE avec `match_id` et la duree — c est ce qui rendra le
// backfill lisible pendant qu il tourne, et c est aussi la seule facon de voir venir l anomalie
// de cout du BTB.
//
// Contrat d erreur : une erreur rendue est une VRAIE panne (reseau, base, decodage casse). Un
// film absent, un film sans kill-feed, un depassement de delai et une capability absente sont
// des OUTCOMES, pas des erreurs — la passe appelante continue.
// Le nombre de morts ecrites est RENDU, pas seulement journalise : la synthese d une passe de
// masse doit pouvoir le totaliser. Une premiere version le laissait dans le log — le compteur de
// `KillSourceSummary` restait donc a zero pendant que chaque match en annoncait des dizaines,
// et c est le genre de zero qu on croit longtemps.
func (c *KillSourceCollector) CollectMatch(ctx context.Context, matchID string) (KillSourceOutcome, int, error) {
	if !c.caps.Has(games.CapFilmKillSource) {
		// Degradation gracieuse : ni panic, ni erreur remontee — le cycle continue.
		slog.DebugContext(ctx, "killsource: capability absente, passe ignoree",
			"match_id", matchID, "capability", string(games.CapFilmKillSource),
			"err", games.ErrCapabilityNotSupported)
		return OutcomeNotSupported, 0, nil
	}

	start := time.Now()
	slog.InfoContext(ctx, "killsource: decodage du film — debut", "match_id", matchID)

	// LA LIMITE DE TEMPS PAR MATCH. Elle couvre telechargement ET decodage : le cout observe
	// est domine par le decodage, mais un CDN qui ne repond pas bloquerait tout autant.
	matchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	outcome, deaths, err := c.collect(matchCtx, matchID)
	dur := time.Since(start)

	// Le depassement de delai se reconnait au ctx du MATCH, pas a celui de l appelant : un
	// arret demande par l appelant (Ctrl-C, shutdown) n est pas un abandon du collecteur.
	if err != nil && errors.Is(matchCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		observability.AddInt(metricTimeout, 1)
		slog.WarnContext(ctx, "killsource: abandon sur limite de temps",
			"match_id", matchID, "duration", dur, "limite", c.timeout)
		return OutcomeTimeout, 0, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "killsource: decodage du film — echec",
			"match_id", matchID, "duration", dur, "err", err)
		return outcome, 0, err
	}

	slog.InfoContext(ctx, "killsource: decodage du film — fin",
		"match_id", matchID, "duration", dur, "resultat", string(outcome), "morts", deaths)
	return outcome, deaths, nil
}

// decodeFilmForMatch telecharge les chunks du match, en reconstitue le film et le decode.
// Il rend un outcome AUTRE que [OutcomeWritten] quand la passe n a RIEN a publier — film
// absent, film expire, film sans kill-feed — et l appelante s arrete alors la.
//
// L OUTCOME EST LE SIGNAL, PAS `res == nil` : tester le resultat ferait avaler en silence un
// decodeur qui rendrait `(nil, nil)`, la ou le code d origine poursuivait (et se serait
// arrete bruyamment). Le contrat reste donc mot pour mot celui d avant l extraction.
//
// EXTRAIT DE `collect` AU LOT 2.7 (2026-09-16), et c est la SEULE reecriture du lot hors
// deplacement pur : le bloc est recopie ligne pour ligne, seules ses sorties changent de
// forme (`OutcomeX, 0, err` devient `nil, nil, nil, OutcomeX, err`). Il ramene `collect`
// de 91 a 53 lignes et lui rend la lecture que l en-tete du paquet annonce — telecharger,
// decoder, ecrire, chacun a sa place.
func (c *KillSourceCollector) decodeFilmForMatch(ctx context.Context, matchID string) (
	[]haloclient.FilmChunk, *decfilm.Film, *decfilm.Result, KillSourceOutcome, error,
) {
	chunks, found, err := FilmChunksForMatch(ctx, c.client, matchID)
	if err != nil {
		// EXPIRATION PARTIELLE = DEFINITIVE (bilan fork ChaseWoodhams 2026-09-11, point 4b) :
		// le manifeste repond encore mais un chunk (blob CDN pre-signe) rend 404/410. Sans
		// cette classification, l erreur remontait telle quelle et CollectMatches la comptait
		// en Errors — jamais en OutcomeNoFilm — donc marquerFilmParOutcome ne posait JAMAIS
		// MBitFilmAbsent : le match restait candidat A VIE aux passes `--online` suivantes
		// (le filtre `MBitFilmAbsent` ne l excluait jamais). haloclient.IsFilmGoneErr
		// distingue ce lien DEFINITIVEMENT mort (404/410, manifeste ou blob) d une panne
		// transitoire (503, rate-limit, reseau) qui doit rester une erreur retentee.
		if haloclient.IsFilmGoneErr(err) {
			observability.AddInt(metricNoFilm, 1)
			slog.WarnContext(ctx, "killsource: film expire definitivement (404/410 sur le manifeste ou un blob, manifeste possiblement relu du cache local) — classe absent",
				"match_id", matchID, "err", err)
			return nil, nil, nil, OutcomeNoFilm, nil
		}
		observability.AddInt(metricDecodeError, 1)
		return nil, nil, nil, OutcomeNoFilm, err
	}
	if !found {
		observability.AddInt(metricNoFilm, 1)
		return nil, nil, nil, OutcomeNoFilm, nil
	}

	// LE FILM EST DECOMPRESSE ET DECOUPE UNE SEULE FOIS POUR TOUTE LA PASSE (lot 1 de
	// PLAN_CUISSON_PERF, items 1.4 et 1.6) : les morts le lisent ci-dessous, les positions plus
	// bas — la ou chacune des cinq lectures payait avant sa propre decompression du film entier.
	film, err := FilmOf(chunks)
	if err != nil {
		observability.AddInt(metricDecodeError, 1)
		return nil, nil, nil, OutcomeNoFilm, fmt.Errorf("chargement du film %s: %w", matchID, err)
	}

	// LA CONFIGURATION GELEE, celle qui a produit les chiffres publies, PLUS LA CARTE DU MATCH.
	// Ne jamais passer d autre Options ici sans une raison ecrite : ce sont elles qui
	// definissent le decodage. `Carte` n en est pas une : c est une DONNEE d entree, la meme
	// entree de catalogue que la passe des positions resout deja (`resolveMapBounds`), et sans
	// elle la marche des morts lit ses positions aux largeurs d UNE AUTRE CARTE (lot 3.4.1).
	opts := decfilm.DefaultOptions()
	opts.Carte = c.carteDuMatch(ctx, matchID)
	res, err := decfilm.Decode(ctx, matchID, film, &opts)
	if err != nil {
		// Un film sans kill-feed n est pas une panne : c est un film dont on ne peut rien
		// publier. Le distinguer evite qu un backfill s arrete sur un vieux match.
		if errors.Is(err, decfilm.ErrNoKillFeed) {
			observability.AddInt(metricNoKillFeed, 1)
			slog.InfoContext(ctx, "killsource: film sans kill-feed, rien a publier",
				"match_id", matchID)
			return nil, nil, nil, OutcomeNoKillFeed, nil
		}
		observability.AddInt(metricDecodeError, 1)
		return nil, nil, nil, OutcomeNoFilm, fmt.Errorf("decodage %s: %w", matchID, err)
	}
	return chunks, film, res, OutcomeWritten, nil
}

// collect : le corps, sans la journalisation ni la mesure de duree.
//
// UN FILM TELECHARGE UNE FOIS, DEUX TABLES ECRITES. Les morts (`match_kill_events`) et les tirs
// (`match_weapon_shots`) sortent de la MEME passe : re-decoder un film pour la seconde table
// couterait une seconde fois la passe chere du chantier, et donnerait deux occasions de sortir
// deux etats differents de la meme base.
func (c *KillSourceCollector) collect(ctx context.Context, matchID string) (KillSourceOutcome, int, error) {
	chunks, film, res, outcome, err := c.decodeFilmForMatch(ctx, matchID)
	if outcome != OutcomeWritten {
		return outcome, 0, err
	}

	ids, err := c.roster.IdentitiesForMatch(ctx, matchID)
	if err != nil {
		return OutcomeNoFilm, 0, fmt.Errorf("roster %s: %w", matchID, err)
	}

	batch := BuildKillSourceBatch(matchID, res, ids)
	fusionnees, err := c.write(ctx, batch)
	if err != nil {
		observability.AddInt(metricWriteError, 1)
		return OutcomeWritten, 0, err
	}
	publiees := len(fusionnees)
	publishKillSourceMetrics(res, batch)

	// LA VENTILATION DES TIRS, SUR LES MEMES CHUNKS. Son echec ne remet PAS en cause les morts :
	// elles sont deja ecrites, elles sont justes, et les deux tables repondent a deux questions
	// differentes. Il se journalise et se compte — jamais il n avale la passe.
	c.collectShots(ctx, matchID, chunks, ids)

	// LES POSITIONS, SUR LE MEME FILM (G.2bis) — memes garanties que les tirs : best-effort,
	// jamais de remise en cause des morts deja ecrites.
	//
	// DEUX LISTES, ET ELLES NE SONT PAS INTERCHANGEABLES (correction de la revue de 7C) :
	//
	//	batch.Deaths   la liste PRE-FUSION. Seule population qui peut structurellement avoir
	//	               une POSITION — un kill recupere par le producteur credit-seul n a aucune
	//	               position a offrir (cf. l en-tete de positions.go).
	//	fusionnees     ce que le JOURNAL a reellement ecrit (MergeCreditAndFilm : credit + film,
	//	               orphelins compris). C est la liste que les FAITS D ISOLEMENT doivent
	//	               suivre : une mort credit-seule absente du fil n aurait sinon aucun
	//	               contexte, ET sa victime ne serait jamais « en attente » aux morts
	//	               suivantes — elle passerait pour vivante et hors de vue.
	c.collectPositions(ctx, matchID, film, res, ids, batch.Deaths, fusionnees)

	// LE NUMERATEUR DE PRECISION PAR ARME (weapon_accuracy + distance), Infinite depuis le film.
	// Best-effort au meme titre que les tirs : son echec ou son absence (film non sur disque,
	// capability absente) ne remet pas en cause les morts. Meme raison de sante que collectShots.
	c.collectHits(ctx, matchID, chunks, ids)

	return OutcomeWritten, publiees, nil
}

// collectShots : la seconde ecriture de la passe — `shared.match_weapon_shots`.
//
// BEST-EFFORT ASSUME, ET C EST UN CHOIX : les morts sont la donnee qui motive le chantier (46,5 %
// de doublons dans `killer_victim_pairs`), les tirs sont un enrichissement. Remonter une erreur
// de ventilation ferait recompter le match comme un echec alors que ses morts sont en base — et
// un backfill le rejouerait, redecodant un film pour rien.
func (c *KillSourceCollector) collectShots(
	ctx context.Context, matchID string, chunks []haloclient.FilmChunk, parts MatchIdentities,
) {
	// DEUX FAMILLES DE DONNÉES, DEUX CAPABILITIES (§4.3 du plan de branchement). Le film
	// est déjà téléchargé et décodé pour les morts ; un titre peut malgré tout exposer le
	// kill enrichi SANS exposer la ventilation des tirs — elle a ses propres réserves
	// (Fiesta et BTB non livrables). Dégradation gracieuse : on n'écrit rien, on ne
	// remonte pas d'erreur, et la passe des morts reste acquise.
	if !c.caps.Has(games.CapFilmWeaponShots) {
		slog.DebugContext(ctx, "killsource: tirs par arme — capability absente, ventilation ignorée",
			"match_id", matchID, "capability", string(games.CapFilmWeaponShots),
			"err", games.ErrCapabilityNotSupported)
		return
	}
	batch := BuildWeaponShotsBatch(matchID, ReplicationChunks(chunks), parts.XUIDs, parts.ShotsFired)
	if err := c.writeShots(ctx, batch); err != nil {
		observability.AddInt(metricShotsWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: ecriture de la ventilation des tirs echouee",
			"match_id", matchID, "err", err)
		return
	}
	publishShotsPass(ctx, batch, len(parts.XUIDs))
}

// write : la FUSION puis l ecriture, sous le lease RW de shared (ADR 0013 — un seul writer par DB).
//
// ─── LE DECODAGE NE PUBLIE PLUS SEUL ───────────────────────────────────────────────────────
//
// Depuis l inversion de preseance (2026-08-03), une passe de film ne remplace plus rien : elle se
// POSE sur la base credit du match. La base est relue ici, dans le meme handle que l ecriture
// (`persist.CreditBaseForMatch` — les couples de `killer_victim_pairs`, dedupliques sur
// l identite, exactement la definition que la migration de reprise emploie), puis
// `persist.MergeCreditAndFilm` apparie sur `(match_id, time_ms)` a tolerance ZERO.
//
// UN MATCH SANS COUPLE REND UNE BASE VIDE, et ce n est pas une erreur : toutes les lignes du film
// deviennent alors des orphelines et la passe vaut ce qu elle valait avant l inversion. Le cas se
// produit sur un match dont la completion combat n a pas encore tourne — le producteur live
// recomposera la passe quand elle arrivera.
//
// Le chemin est le persister DEDIE et pas `BatchBuilder` : une passe de decodage arrive sur un
// match DEJA insere, donc sans `Shared.Match`, et `SharedPersister` y serait un no-op. Le chemin
// builder existe (`SetKillSource`) et reste le bon quand un film serait pret des le sync
// primaire — ce qui n arrive pas aujourd hui.
// ELLE REND LA LISTE FUSIONNEE, pas seulement son compte : c est celle que le journal porte, et
// les faits d isolement doivent la suivre (cf. l appel a collectPositions).
func (c *KillSourceCollector) write(ctx context.Context, film persist.KillSourceBatch) ([]persist.KillEventInsert, error) {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return nil, fmt.Errorf("lease shared %s: %w", film.MatchID, err)
	}
	defer release()

	base, err := persist.CreditBaseForMatch(ctx, db, film.MatchID)
	if err != nil {
		return nil, err
	}
	batch, st, err := persist.MergeCreditAndFilm(base, film)
	if err != nil {
		return nil, err
	}
	persist.PublishMergeStats(ctx, film.MatchID, st)
	slog.InfoContext(ctx, "killsource: passe fusionnee sur la base credit",
		"match_id", film.MatchID, "base_credit", len(base.Deaths), "lignes_film", len(film.Deaths),
		"publiees", len(batch.Deaths), "enrichies", st.Enriched, "orphelins", st.Orphans)
	if err := persist.NewKillSourcePersister(db).PersistPass(ctx, batch); err != nil {
		return nil, err
	}
	return batch.Deaths, nil
}

// writeShots : l ecriture des tirs, sous son PROPRE lease.
//
// DEUX LEASES COURTS PLUTOT QU UN LONG. Le lease RW de shared est la ressource la plus disputee
// du process (un seul writer, ADR 0013) : le tenir pendant la ventilation — resolution d indices
// au bit pres et scan de tous les chunks — le bloquerait pour rien. Les deux tables sont
// append-only et independantes : rien n exige qu elles s ecrivent dans la meme transaction.
func (c *KillSourceCollector) writeShots(ctx context.Context, batch persist.WeaponShotsBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", batch.MatchID, err)
	}
	defer release()
	return persist.NewWeaponShotsPersister(db).PersistPass(ctx, batch)
}
