package killcollector

// roster.go — la resolution `gamertag -> xuid`, et la passe multi-matchs.
//
// POURQUOI CETTE RESOLUTION EXISTE : les CHUNKS DE REPLICATION ne portent aucun xuid — ils ne
// rendent que des NOMS. Le rattachement au joueur se fait donc contre le roster du match, en base.
// Son echec n est PAS une erreur : un bot n a pas de xuid, et un nom que le roster ne connait pas
// est une donnee (la ligne s ecrit avec un xuid NULL).
//
// ⚠ LA PHRASE D ORIGINE DISAIT « LE FILM NE PORTE AUCUN XUID », ET ELLE EST FAUSSE DEPUIS LE
// LOT 1.5 (2026-09-14). `chunk_00` porte trente-deux enregistrements de slot avec XUID ET
// gamertag (`grammar.ReadPlayerTable`), et le lot 1.8 les emploie pour EPINGLER le lien
// `indice -> joueur` dans le decodeur (`killsource/film_table.go`). Ce qui reste vrai, et qui
// suffit a justifier ce fichier : la table du film est celle du DEBUT du film (un joueur qui
// rejoint en cours de partie n y a pas de siege, mesure du lot 1.6), et elle ne resout pas les
// bots. La base reste donc le resolveur `gamertag -> xuid` de la PUBLICATION.

import (
	"context"
	"database/sql"
	"fmt"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
	"log/slog"
	"time"
)

// SharedRoster : la resolution nom -> xuid depuis `shared.match_participants`.
//
// La table est celle du sync primaire : elle est deja peuplee quand le collecteur passe (c est
// meme la raison pour laquelle le collecteur est une passe SEPAREE — il travaille sur des matchs
// deja au registre).
type SharedRoster struct {
	db *sql.DB
	// chargeur : l ANNUAIRE DE PASSE, optionnel (nil = la jointure par match, le defaut de tous
	// les appelants live). Cable par [SharedRoster.AvecAnnuaireDePasse], et par le seul backfill.
	chargeur *chargeurDAnnuaire
}

// NewSharedRoster construit la resolution. `db` peut etre un handle LECTURE (le collecteur
// n ecrit pas par ce chemin) — utiliser `OpenReadForQuery` si la DB est potentiellement tenue
// en RW par le process (jamais `OpenReadOnly` force : erreurs « different configuration »).
func NewSharedRoster(db *sql.DB) *SharedRoster { return &SharedRoster{db: db} }

// RosterForMatch rend `gamertag -> xuid` pour les participants du match.
//
// ⚠ La cle est le GAMERTAG parce que c est la seule quantite que le film rende. Un gamertag qui
// change entre le match et la collecte ne se rattachera pas — c est une limite connue, et elle
// se lit dans la donnee (xuid NULL), pas dans un silence.
//
// ⚠⚠ LE NOM VIENT DE `v_gamertag_lookup`, LA VUE CANONIQUE, ET C EST UNE MESURE QUI L IMPOSE.
// `match_participants.gamertag` est vide en pratique : **27 607 lignes sur 27 989 sans nom, et
// 4 xuids nommes sur 16 996**. Une premiere version de ce roster lisait cette colonne : elle
// rendait une table quasi VIDE, donc le collecteur ecrivait 16 908 morts dont **10** portaient
// un xuid de victime. Des lignes qu aucun agregat carriere ne peut joindre — c est-a-dire
// exactement ce que ce chantier existe pour produire.
//
// La vue est la SOURCE UNIQUE du resolveur `xuid -> gamertag` du depot, et sa cascade explique
// pourquoi lire une seule table ne suffit jamais : bot connu, puis `xuid_aliases`, puis
// `match_participants`, puis `killer_victim_pairs`, puis libelle masque. **`xuid_aliases` a cesse
// d etre alimente en avril 2026** ; ce sont les gamertags du KILL-FEED, portes par
// `killer_victim_pairs`, qui couvrent les adversaires croises depuis. Couverture mesuree : 18 219
// xuids, 36 masques.
//
// AMBIGUITE : si deux participants du meme match portent le meme nom, on n en garde AUCUN.
// Ecrire les morts d un joueur sous le xuid d un autre serait pire que de n en ecrire aucun.
func (r *SharedRoster) IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error) {
	parXUID, err := r.gamertagsForMatch(ctx, matchID)
	if err != nil {
		return MatchIdentities{}, err
	}
	out := MatchIdentities{
		ParXUID:    parXUID,
		ParNom:     make(map[string]string, len(parXUID)),
		ShotsFired: map[string]int{},
		Equipes:    map[string]int{},
	}
	ambigus := map[string]bool{}
	for xuid, gt := range parXUID {
		if ambigus[gt] {
			continue
		}
		if _, deja := out.ParNom[gt]; deja {
			delete(out.ParNom, gt)
			ambigus[gt] = true
			slog.WarnContext(ctx, "killsource: deux participants portent le meme nom, aucun xuid "+
				"ne sera attribue", "match_id", matchID, "gamertag", gt)
			continue
		}
		out.ParNom[gt] = xuid
	}
	if err := r.participantsForMatch(ctx, matchID, &out); err != nil {
		return MatchIdentities{}, err
	}
	return out, nil
}

// gamertagsForMatch rend `xuid -> gamertag` pour les participants du match, par la vue
// canonique.
//
// ⚠ DEPENDANCE A CONSIGNER POUR LA BASCULE (phase 2) : `v_gamertag_lookup` lit
// `killer_victim_pairs`. Supprimer cette table sans avoir d abord rebranche la vue sur
// `match_kill_events_latest` ferait retomber les adversaires sur le libelle masque — une panne
// d affichage que rien ne signalerait, sur le chemin le plus visible du produit.
func (r *SharedRoster) gamertagsForMatch(ctx context.Context, matchID string) (map[string]string, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("SharedRoster: db nil")
	}
	if r.chargeur != nil {
		return r.gamertagsParLAnnuaire(ctx, matchID)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT mp.xuid, g.gamertag
		FROM match_participants mp
		JOIN v_gamertag_lookup g ON g.xuid = mp.xuid
		WHERE mp.match_id = ? AND mp.xuid IS NOT NULL AND mp.xuid <> ''
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("SharedRoster(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string, 16)
	for rows.Next() {
		var xuid string
		var gt sql.NullString
		if err := rows.Scan(&xuid, &gt); err != nil {
			return nil, fmt.Errorf("SharedRoster(%s) scan: %w", matchID, err)
		}
		if gt.Valid && gt.String != "" {
			out[xuid] = gt.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SharedRoster(%s) rows: %w", matchID, err)
	}
	return out, nil
}

// participantsForMatch complete `out` avec les xuids du match, la reference `shots_fired`,
// l EQUIPE de chacun, et le TABLEAU DE L API que le registre d identite consomme pour departager
// un siege partage (`out.Participants`, cf. identities.go et identity_registry_scoreboard.go).
//
// ⚠ `shots_fired` NULL N EST PAS ZERO. Une colonne nulle veut dire « l API n a pas donne le
// nombre de tirs » ; zero veut dire « l API dit qu il n a pas tire ». La porte de publication
// traite les deux DIFFEREMMENT (refus faute de reference d un cote, verdict de l autre), donc la
// lecture ne doit surtout pas les confondre : une valeur nulle n entre pas dans la table.
//
// L INSTANT D ARRIVEE PASSE PAR LE FRAGMENT TIMEZONE CANONIQUE (regle n 8 du depot) — jamais
// `start_time` brut — et par LA MEME FORMULE que `platform/duckdb/replay_facts_repo.go`
// (`playerFacts`) : les deux producteurs du registre (cuisson, collecteur) doivent caler
// l instant d arrivee sur le MEME zero, sans quoi la fenetre que le tableau tranche glisserait
// d un producteur a l autre.
func (r *SharedRoster) participantsForMatch(ctx context.Context, matchID string, out *MatchIdentities) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("SharedRoster: db nil")
	}
	start := analysis.SQLStartTimeCanonical("mr")
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.xuid, p.shots_fired, p.team_id,
		       COALESCE(p.joined_in_progress, FALSE),
		       CAST(epoch_ms(p.first_joined_time) - epoch_ms(`+start+`) AS BIGINT)
		FROM match_participants p
		JOIN match_registry mr ON mr.match_id = p.match_id
		WHERE p.match_id = ? AND p.xuid IS NOT NULL AND p.xuid <> ''
		ORDER BY p.xuid
	`, matchID)
	if err != nil {
		return fmt.Errorf("SharedRoster participants(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var xuid string
		var shots, team, joinMS sql.NullInt64
		var joined bool
		if err := rows.Scan(&xuid, &shots, &team, &joined, &joinMS); err != nil {
			return fmt.Errorf("SharedRoster participants(%s) scan: %w", matchID, err)
		}
		out.XUIDs = append(out.XUIDs, xuid)
		if shots.Valid {
			out.ShotsFired[xuid] = int(shots.Int64)
		}
		if team.Valid {
			out.Equipes[xuid] = int(team.Int64)
		}
		p := replay.Participant{ID: xuid, JoinedInProgress: joined}
		// L INSTANT NE VOYAGE QUE S IL EST DECLARE (meme garde que `participantsDuTableau`
		// cote cuisson) : le porter avec un zero fabriquerait une arrivee au coup d envoi, ce
		// qui donnerait TOUTES les vies du siege a l humain — exactement le defaut que ce lot
		// corrige.
		if joined && joinMS.Valid {
			ms := joinMS.Int64
			p.JoinMatchMS = &ms
		}
		out.Participants = append(out.Participants, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("SharedRoster participants(%s) rows: %w", matchID, err)
	}
	return nil
}

// KillSourceSummary : ce qu une passe multi-matchs a produit. Tous les compteurs sont des
// ENTIERS et chacun correspond a un outcome — pas de « erreurs » fourre-tout qui melangerait un
// film absent (normal) et une base injoignable (panne).
type KillSourceSummary struct {
	Total      int
	Written    int
	Deaths     int
	NoFilm     int
	NoKillFeed int
	Timeouts   int
	Errors     int
	NotSupport int
	// UnknownKey : films ECARTES parce que leur cle ecrite est absente de la table de profil
	// (lot 3.1.1). Ni un ecrit, ni une erreur : la passe s est arretee AVANT tout decodage.
	UnknownKey  int
	ElapsedTime time.Duration
}

// CollectMatches : la passe de fond, EN SERIE — le chemin de REFERENCE.
//
// ⚠ CE QUI ETAIT ECRIT ICI ETAIT PERIME, ET LE CORRIGER EST LE LOT 5.24.2 (2026-09-22). Le
// commentaire disait : « les parametres de replication du decodeur sont des globaux de paquet ;
// `killsource.Decode` serialise deja par un verrou ». **Ni l un ni l autre n existe plus.** La
// cloture M3 du chantier decodeur (ADR 0034, 2026-09-17) a DEPENSE le profil — il voyage en
// argument jusqu a `calibrate` et `runWalk` — et le dernier reglage global
// (`SetInferResyncTargets`) a ete supprime au lot E.2 du 2026-09-05. Plusieurs films PEUVENT
// donc se decoder en parallele : c est ce que fait [KillSourceCollector.CollectMatchesOuvriers],
// et `collector_ouvriers.go` porte la mesure qui le justifie et la preuve qui l atteste.
//
// CETTE BOUCLE RESTE, ET ELLE RESTE LE DEFAUT DE TOUS LES APPELANTS QUI NE DEMANDENT RIEN : le
// post-sync du serveur, `--online` et les tests. Elle est aussi le TEMOIN du test d egalite —
// c est a elle que la passe a N ouvriers doit rendre les memes lignes.
//
// Une erreur sur UN match n arrete pas la passe : elle est comptee et journalisee. Seul l arret
// de l appelant (`ctx`) interrompt — et il rend la synthese de ce qui a ete fait, pas une erreur.
func (c *KillSourceCollector) CollectMatches(ctx context.Context, matchIDs []string) KillSourceSummary {
	start := time.Now()
	sum := KillSourceSummary{Total: len(matchIDs)}

	for _, id := range matchIDs {
		// LE BUDGET N EST PAS UNE ANNULATION DE CONTEXTE, ET C EST DELIBERE. Une premiere
		// version bornait la passe en passant un `context.WithTimeout` : a son expiration, le
		// test « depassement de delai » de CollectMatch (qui exige `ctx.Err() == nil` pour
		// distinguer l abandon du match de l arret de l appelant) devenait faux, et un arret
		// NOMINAL se comptait en `killsource_erreurs_decodage`. La surface d alerte etait
		// polluee par son propre garde-fou. Ici l arret est explicite, entre deux matchs.
		if c.budget > 0 && time.Since(start) >= c.budget {
			observability.AddInt(metricBudget, 1)
			slog.InfoContext(ctx, "killsource: budget de passe epuise — le solde repart au cycle suivant",
				"budget", c.budget, "traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		if c.arretDouxDemande() {
			// ARRET DOUX (lot 5.24.4) : le film en cours est deja fini — on ne prend pas le
			// suivant. La boucle en serie et la passe a ouvriers s arretent au meme endroit.
			slog.InfoContext(ctx, "killsource: arret demande — la passe s arrete entre deux films",
				"traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "killsource: passe interrompue par l appelant",
				"traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		debut := time.Now()
		if c.observateur != nil {
			c.observateur.FilmDemarre(id, debut)
		}
		outcome, deaths, err := c.CollectMatch(ctx, id)
		ev := EvenementDeFilm{MatchID: id, Outcome: outcome, Morts: deaths,
			Duree: time.Since(debut), Err: err}
		if err == nil {
			// LES MARQUEURS DE FILM DU REGISTRE (cf. registry_flags.go). Ils etaient poses par
			// l etape 1.55 jusqu au 2026-09-01 ; cette passe telecharge le meme film, au meme
			// moment, donc elle sait ce qu ils affirment. Ecrire ici et pas dans `collect` :
			// seule cette boucle distingue une erreur (rien a affirmer) d un outcome.
			c.marquerFilm(ctx, id, outcome, deaths)
		}
		if c.observateur != nil {
			c.observateur.FilmFini(ev)
		}
		// UN SEUL CHEMIN DE COMPTAGE POUR LES DEUX PASSES (`comptabiliserFilm`, lot 5.24.2) :
		// deux copies du meme `switch` diraient tot ou tard deux totaux differents de la meme
		// passe, et c est exactement le genre d ecart qu on ne voit pas.
		comptabiliserFilm(&sum, ev)
	}

	sum.ElapsedTime = time.Since(start)
	slog.InfoContext(ctx, "killsource: passe terminee",
		"total", sum.Total, "ecrits", sum.Written, "films_absents", sum.NoFilm,
		"sans_killfeed", sum.NoKillFeed, "abandons_delai", sum.Timeouts,
		"erreurs", sum.Errors, "capability_absente", sum.NotSupport,
		"ecartes_cle_inconnue", sum.UnknownKey,
		"duration", sum.ElapsedTime)
	return sum
}

// AvecAnnuaireDePasse branche l ANNUAIRE DE PASSE : `v_gamertag_lookup` lue UNE FOIS, au premier
// match, au lieu d etre jointe par match (lot 5.24.2, 2026-09-22).
//
// # LA MESURE QUI L EXIGE
//
// D1 du lot 5.12 (§ 4 du plan) avait recense le defaut sans le corriger : `gamertagsForMatch`
// joint la vue canonique d identite PAR MATCH, et le filtre `mp.match_id = ?` ne peut RIEN
// pousser dans une vue faite de trois legs agreges en FULL OUTER JOIN — elle est donc
// materialisee entierement, une fois par film. Mesure 5.24.1 : **49 a 60 ms par match, constants
// (independants du film)** sur un banc de 180 000 lignes, et « des secondes » sur la base de
// production, qui en porte plusieurs millions. Sur 1 612 films c est de la minute au quart
// d heure — et depuis que la passe tourne a N ouvriers cette lecture est SERIALISEE derriere la
// porte de la base : elle devient le plafond de la passe, quel que soit le nombre d ouvriers.
//
// # CE QU IL CHANGE, ET CE QU IL NE CHANGE PAS
//
// La SOURCE DES NOMS RESTE `v_gamertag_lookup` — meme vue, meme cascade, meme repli. Ce qui
// change est le nombre d evaluations. La selection des participants reste une lecture par match,
// mais elle porte sur `match_participants` seule, qui pousse son predicat.
//
// L EQUIVALENCE EST EXACTE SUR LA FORME : la jointure d origine etait un INNER JOIN qui ne
// gardait que les noms non vides ; la vue ne rend JAMAIS un nom vide (son dernier repli est le
// libelle masque « Joueur #### »), et tout xuid de `match_participants` est dans la vue par son
// leg `mp`. Un participant nomme le reste, un participant sans xuid est exclu des deux cotes.
// L annuaire fait meme MIEUX sur un point : si la vue rendait deux lignes pour un xuid, la
// jointure retenait l une des deux au hasard de l ordre de sortie de DuckDB, la ou l annuaire
// retient la premiere d un `ORDER BY` (cf. [requeteAnnuaireDesNoms]).
//
// # LE SEUL ECART POSSIBLE EST NOMME, ET IL VA DANS LE BON SENS
//
// L instantane est pris au premier match ; la passe ecrit ensuite dans `match_kill_events`, que
// la vue relit (son leg 4). Un nom ajoute par la passe pourrait donc, sans annuaire, etre relu
// par un match suivant. Les noms que la passe ecrit viennent de [MatchIdentities.Resoudre], donc
// de l annuaire lui-meme — SAUF UN : quand le film ne porte aucun gamertag pour un joueur, le
// decodeur ecrit `xuid:NNN` et `Resoudre` garde cette forme brute si le xuid n est pas au roster
// du match. La jointure par match pourrait alors, sur un match suivant, servir `xuid:NNN` comme
// nom d affichage (le leg 4 prend le `MAX` des noms du kill-feed, et `x` est haut) — c est-a-dire
// exactement le « xuid brut a l affichage » que `v_gamertag_lookup` existe pour empecher.
// L annuaire ne peut pas le faire. L ecart est donc REEL, RARE, et il retire un faux nom ; il est
// verifie sur les films du cache par `TestAnnuaireDePasse_MemesIdentitesQueLaJointure`.
//
// ⚠ UN COLLECTEUR = UNE PASSE, comme pour le credit : l instantane vit aussi longtemps que le
// `SharedRoster`. Le brancher sur un roster garde vivant des heures (le post-sync du serveur, le
// chemin live) servirait des noms vieux de ces heures — d ou le fait que ce soit un reglage
// EXPLICITE, et que seul le backfill le pose.
func (r *SharedRoster) AvecAnnuaireDePasse() *SharedRoster {
	if r != nil {
		r.chargeur = &chargeurDAnnuaire{}
	}
	return r
}

// requeteParticipantsNommables : les xuids du match, et eux seuls. Le predicat porte sur une
// TABLE (pas une vue agregee), donc DuckDB le pousse.
const requeteParticipantsNommables = `
	SELECT DISTINCT mp.xuid
	FROM match_participants mp
	WHERE mp.match_id = ? AND mp.xuid IS NOT NULL AND mp.xuid <> ''`

// gamertagsParLAnnuaire rend `xuid -> gamertag` pour les participants du match, en lisant les
// noms dans l annuaire de passe au lieu de joindre la vue.
func (r *SharedRoster) gamertagsParLAnnuaire(ctx context.Context, matchID string) (map[string]string, error) {
	noms, err := r.chargeur.annuaireDe(ctx, r.db, "films")
	if err != nil {
		return nil, fmt.Errorf("SharedRoster(%s): %w", matchID, err)
	}
	rows, err := r.db.QueryContext(ctx, requeteParticipantsNommables, matchID)
	if err != nil {
		return nil, fmt.Errorf("SharedRoster(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string, 16)
	for rows.Next() {
		var xuid string
		if err := rows.Scan(&xuid); err != nil {
			return nil, fmt.Errorf("SharedRoster(%s) scan: %w", matchID, err)
		}
		// MEME PORTE QUE LA JOINTURE : un nom vide n entre pas. La vue n en rend pas, mais la
		// garde reste — elle est le contrat de la table, pas une precaution sur la vue du jour.
		if nom := noms.nom(xuid); nom != "" {
			out[xuid] = nom
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SharedRoster(%s) rows: %w", matchID, err)
	}
	return out, nil
}
