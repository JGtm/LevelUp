package replayartifacts

// vehicletakes.go — LA RESSOURCE VEHICULES DE L'EMPRISE, PERSISTEE AU FIL DE L'EAU (plan
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.2, decisions D1, D8, D9, D10).
//
// # CE QUE CE FICHIER FAIT
//
// Il LIT le calque vehicules de l'artefact range (`replay.ProjectVehicleTakes`, L7.1), apparie les
// frags de classe engin du match aux episodes de leur tueur (`replay.PairVehicleFrags`, D9), et
// transporte le resultat vers `persist.VehicleTakesPersister`.
//
// # LA FORME, CALQUEE SUR padtiers.go (et c'est voulu)
//
//	SUR DISQUE, PAS LE BLOB   le document est deja lu et deserialise par [Deriver] ;
//	UNE LECTURE EN BASE       les frags de mort se lisent par un segment de LECTURE court, AVANT
//	                          la premiere famille qui ecrit (padtiers_preparation.go : la panne du
//	                          2026-09-14 au 2026-09-27 et pourquoi l'ordre compte) ;
//	VIA `internal/persist`    INSERT-only (ADR 0019/0026/0030), une passe par match.
//
// # UNE SEULE DEFINITION DE « FRAG DE CLASSE ENGIN » (D5)
//
// Source de degat mesuree -> cle de registre (`KillSourceClassifier`, le classificateur de la
// Repartition des frags) -> classe (`weapons.ClassesByKey`, le registre statique dont
// `metadata.weapons` est la copie seedee) -> `domain.IsEngineFragClass`. Une source sans cle de
// registre (les ecrasements en tete) n'a pas de classe et sort du compte.
//
// # LES SILENCES, ET CE QUI S'ECRIT QUAND MEME
//
//	artefact sans occupation lue (schema < 67, calque non balaye)   passe « non mesure » (D8) :
//	                                                                UNE ligne `match`, raison dite ;
//	match sans evenement de mort a source mesuree                   prises et temps ecrits, frags
//	                                                                « non lus » (le rendement est
//	                                                                non mesure, pas nul) ;
//	titre sans `film.vehicle_usage`                                 rien, DEBUG.
//
// UNE LECTURE DE FRAGS EN ECHEC (segment de lecture indisponible, requete refusee) est un DEFAUT :
// les matchs mesures du lot ne sont ni ecrits ni marques, le rattrapage les rejouera.
//
// # POURQUOI `DerivationsRev` N'A PAS ETE MONTEE
//
// Meme choix que les familles `padtiers` et `flaggrabsnet` (ajoutees apres la revision
// `derivations-2026-09-06`, qui n'a pas bouge) : monter la revision ferait rejouer TOUTES les
// derivations (dont les positions, les plus volumineuses) de tout le parc, cinq artefacts par
// cycle, pour une seule famille neuve. Le parc existant passe par la commande de rattrapage de la
// famille (`levelup backfill-vehicle-takes`), qui ne redecode rien ; les artefacts a venir passent
// par le fil de l'eau. Une correction ulterieure de la PROJECTION se rejoue par la meme commande.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/killcollector"
)

// Raisons propres a ce paquet d'un appariement non lu (`persist.VehicleTakesBatch.FragsReason`).
// Identifiants STABLES. Les raisons du projecteur pur sont les constantes `VehicleFrags*` du paquet replay.
const (
	// VehicleFragsNoClassifier : le titre n'a pas de classificateur de source de degat.
	VehicleFragsNoClassifier = "no_classifier"
	// VehicleFragsNotMeasured : les prises elles-memes ne sont pas mesurees (D8), aucun frag a apparier.
	VehicleFragsNotMeasured = "takes_not_measured"
)

// VehicleFragsLecture : les frags de classe engin d'UN match, tels que la base les a rendus.
// Read faux = non lus, Reason dit pourquoi ; jamais une liste vide qui se lirait comme un zero.
type VehicleFragsLecture struct {
	Read   bool
	Reason string
	Frags  []replay.VehicleFragRef
}

// passeVehiculesPrete : une projection reussie, prete a persister.
type passeVehiculesPrete struct {
	matchID string
	batch   persist.VehicleTakesBatch
}

// preparationVehicules : les passes projetees AVANT tout segment d'ecriture.
type preparationVehicules struct {
	titre string
	prets []passeVehiculesPrete
}

// ClassifierDEngins rend le classificateur de source de degat du titre, ou nil. EXPORTE pour le
// backfill, qui doit lire les frags comme le fil de l'eau.
func ClassifierDEngins(repoRoot, titleSlug string) port.KillSourceClassifier {
	return killcollector.ClassifierPourTitre(repoRoot, titleSlug)
}

// LireFragsDEngin lit, pour un lot de matchs, les frags de classe engin de `match_kill_events_latest`
// (tueur du kill-feed, source de degat MESUREE). Rend une lecture par match demande : un match sans
// aucun evenement a source mesuree est « non lu » (`replay.VehicleFragsNoSource`), jamais zero.
//
// EXPORTEE pour le backfill. Requete indexee par match : quelques millisecondes sur un lot de cycle.
func LireFragsDEngin(
	ctx context.Context, db *sql.DB, classifier port.KillSourceClassifier, ids []string,
) (map[string]VehicleFragsLecture, error) {
	out := make(map[string]VehicleFragsLecture, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	if classifier == nil {
		for _, id := range ids {
			out[id] = VehicleFragsLecture{Reason: VehicleFragsNoClassifier}
		}
		return out, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx, `SELECT match_id, time_ms, feed_killer_xuid, source_tag
		FROM match_kill_events_latest
		WHERE match_id IN (?`+strings.Repeat(",?", len(ids)-1)+`) AND source_tag IS NOT NULL`, args...)
	if err != nil {
		return nil, fmt.Errorf("vehicules: lecture des frags de mort: %w", err)
	}
	defer func() { _ = rows.Close() }()
	classes := weapons.ClassesByKey()
	lus := map[string]bool{}
	frags := map[string][]replay.VehicleFragRef{}
	for rows.Next() {
		var id string
		var timeMS int
		var killer sql.NullString
		var tag uint32
		if err := rows.Scan(&id, &timeMS, &killer, &tag); err != nil {
			return nil, fmt.Errorf("vehicules: scan des frags de mort: %w", err)
		}
		lus[id] = true
		if !killer.Valid || killer.String == "" {
			continue // un tueur bot : pas de xuid, aucun episode nomme ne peut le couvrir
		}
		if key, ok := classifier.KillSourceRegistryKey(tag); ok && domain.IsEngineFragClass(classes[key]) {
			frags[id] = append(frags[id], replay.VehicleFragRef{XUID: killer.String, TimeMS: timeMS})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vehicules: lignes des frags de mort: %w", err)
	}
	for _, id := range ids {
		if lus[id] {
			out[id] = VehicleFragsLecture{Read: true, Frags: frags[id]}
		} else {
			out[id] = VehicleFragsLecture{Reason: replay.VehicleFragsNoSource}
		}
	}
	return out, nil
}

// ProjeterPrisesDeVehicules tire d'UN document range la passe a ecrire. Rend TOUJOURS une passe
// (MatchID non vide) : « non mesure » et « zero mesure » s'ecrivent, c'est ce qui les distingue.
//
// EXPORTEE : le backfill rejoue EXACTEMENT cette projection sur les artefacts deja ranges.
func ProjeterPrisesDeVehicules(matchID string, doc *replay.ReplayDocument, frags VehicleFragsLecture) persist.VehicleTakesBatch {
	rep := replay.ProjectVehicleTakes(doc)
	out := persist.VehicleTakesBatch{
		MatchID: matchID, Measured: rep.Measured, Reason: rep.Reason,
		EpisodesRead: rep.Coverage.Episodes, EpisodesNoXUID: rep.Coverage.EpisodesNoXUID,
		EpisodesNoCamp: rep.Coverage.EpisodesNoCamp,
	}
	if doc != nil {
		out.DocSchema = doc.SchemaVersion
	}
	if !rep.Measured {
		out.FragsReason = VehicleFragsNotMeasured
		return out
	}
	cov := replay.VehicleFragsCoverage{Reason: frags.Reason}
	if frags.Read {
		cov = replay.PairVehicleFrags(doc, &rep, frags.Frags, true)
	}
	out.FragsRead, out.FragsReason = cov.Read, cov.Reason
	out.FragsTotal, out.FragsUnmatched = cov.Total, cov.Unmatched
	if !out.FragsRead && out.FragsReason == "" {
		out.FragsReason = replay.VehicleFragsNoSource
	}
	out.Rows = make([]persist.VehicleTakeRow, 0, len(rep.Rows))
	for _, r := range rep.Rows {
		out.Rows = append(out.Rows, persist.VehicleTakeRow{
			Camp: r.Camp, XUID: r.XUID, Family: r.Family, Takes: r.Takes, AboardMS: r.AboardMS,
			Episodes: r.Episodes, ProximityEpisodes: r.ProximityEpisodes, Frags: r.Frags,
		})
	}
	return out
}

// capabiliteVehiculesArmee dit si le titre declare `film.vehicle_usage`, et DIT POURQUOI quand la
// reponse est non (meme contrat que `capabiliteNiveauxArmee`).
func capabiliteVehiculesArmee(ctx context.Context, d Deps) (armee, incident bool) {
	return porteCapability(ctx, d, games.CapFilmVehicleUsage, "vehicules", nil)
}

// preparerPrisesDeVehicules fait tout ce qui precede l'ecriture : porte de capability, lecture des
// frags (segment de LECTURE court), projection. Best-effort : aucun echec ne remonte, aucun ne se
// tait ; un match mesure dont les frags n'ont pas pu etre lus est inscrit au bilan (pas de marque).
func preparerPrisesDeVehicules(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) preparationVehicules {
	titre := ctxkeys.TitleSlug(ctx)
	prep := preparationVehicules{titre: titre}
	if len(lus) == 0 {
		return prep
	}
	armee, incident := capabiliteVehiculesArmee(ctx, d)
	if !armee {
		if incident {
			observability.AddIntT(titre, CompteurPrisesVehiculesEchecs, int64(len(lus)))
			b.echecLot(lus)
		}
		return prep
	}
	mesures := make(map[string]bool, len(lus))
	var aLire []artefactLu
	for _, a := range lus {
		if replay.ProjectVehicleTakes(a.doc).Measured {
			mesures[a.matchID] = true
			aLire = append(aLire, a)
		}
	}
	lectures, ok := lireFragsDuLot(ctx, d, b, titre, aLire)
	for _, a := range lus {
		if mesures[a.matchID] && !ok {
			continue // frags illisibles : ce match mesure n'est ni projete ni marque
		}
		batch := ProjeterPrisesDeVehicules(a.matchID, a.doc, lectures[a.matchID])
		if !batch.Measured {
			observability.AddIntT(titre, CompteurPrisesVehiculesNonMesurees, 1)
		}
		prep.prets = append(prep.prets, passeVehiculesPrete{matchID: a.matchID, batch: batch})
	}
	return prep
}

// lireFragsDuLot lit les frags du lot par un segment de LECTURE court. ok faux = la lecture a
// echoue : les matchs mesures sont inscrits en echec au bilan et comptes.
func lireFragsDuLot(
	ctx context.Context, d Deps, b *bilanDerivations, titre string, mesures []artefactLu,
) (map[string]VehicleFragsLecture, bool) {
	if len(mesures) == 0 {
		return nil, true
	}
	echec := func(motif string, err error) (map[string]VehicleFragsLecture, bool) {
		slog.ErrorContext(ctx, "post-sync: vehicules — frags de mort illisibles : matchs mesures NON "+
			"projetes, NON marques (le rattrapage les rejouera)",
			"titleSlug", d.TitleSlug, "matchs", len(mesures), "motif", motif, "err", err)
		observability.AddIntT(titre, CompteurPrisesVehiculesEchecs, int64(len(mesures)))
		b.echecLot(mesures)
		return nil, false
	}
	if d.WithRead == nil {
		return echec("aucun segment de lecture cable", nil)
	}
	classifier := ClassifierDEngins(d.RepoRoot, d.TitleSlug)
	ids := matchIDsDuLot(mesures)
	var out map[string]VehicleFragsLecture
	var lecture error
	ouvert := false
	d.WithRead(ctx, "vehicules", func(sharedDB *sql.DB) {
		ouvert = true
		out, lecture = LireFragsDEngin(ctx, sharedDB, classifier, ids)
	})
	if !ouvert || lecture != nil {
		return echec(fmt.Sprintf("segment_ouvert=%v", ouvert), lecture)
	}
	return out, true
}

// ecrirePrisesDeVehicules acquiert le writer et ecrit les passes DEJA projetees.
func ecrirePrisesDeVehicules(ctx context.Context, d Deps, b *bilanDerivations, prep preparationVehicules) {
	titre, prets := prep.titre, prep.prets
	if len(prets) == 0 {
		return
	}
	ids := make([]string, 0, len(prets))
	for i := range prets {
		ids = append(ids, prets[i].matchID)
	}
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: vehicules NON persistes (aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurPrisesVehiculesEchecs, int64(len(prets)))
		echecFauteDeWriter(b, ids)
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, vehicules non persistes",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurPrisesVehiculesEchecs, int64(len(prets)))
		echecFauteDeWriter(b, ids)
		return
	}
	defer release()
	p := persist.NewVehicleTakesPersister(db)
	ecrits, echecs := 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: ecriture des vehicules echouee",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			b.echec(prets[i].matchID)
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurPrisesVehiculesEcrites, int64(ecrits))
	observability.AddIntT(titre, CompteurPrisesVehiculesEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: vehicules persistes",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}
