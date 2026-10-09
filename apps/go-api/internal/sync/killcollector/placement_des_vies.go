package killcollector

// placement_des_vies.go — LA TROISIEME PROJECTION DE LA PASSE DE POSITIONS : le placement et le
// rendement de chaque vie (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2).
//
// # POURQUOI ICI (decision V1 du plan)
//
// Decision utilisateur du 2026-09-07 : « les donnees d'un match en base sont completes au sync ;
// seul le rejeu peut attendre la cuisson ». Le placement d'une vie est un fait d'isolement : il
// se calcule sur le MEME materiau que `match_lives` et `match_death_context` (registre et
// positions de `buildPositionRows`), par la fonction pure `replay.PlacementDesVies`.
//
// # CE QUI S'AJOUTE A LA PASSE, ET SEULEMENT SOUS LA GARDE DE MODE
//
// Les instants ou le joueur PORTE l'objectif sont exclus de la mesure. Ce fait n'existait qu'a la
// cuisson ; `replay.PortagesAuSync` le lit sur le film deja ouvert, par les lectures et
// l'assembleur de la cuisson (voie (b), surcout accepte par l'utilisateur le 2026-09-28). La
// garde de mode est DANS l'entree : hors drapeau, crane, bombe et VIP, rien n'est lu.
//
// # ELLE N'EST JAMAIS BLOQUANTE
//
// Meme doctrine que les faits d'isolement : elle vient APRES les vies, ecrit sous son PROPRE
// lease court, et son echec se journalise et se compte sans jamais remonter — ni les morts, ni
// les positions, ni les vies ne dependent d'elle.

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs de sante du placement des vies (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricPlacementMatches   = "killsource_placement_matchs_couverts"
	metricPlacementLives     = "killsource_placement_vies_ecrites"
	metricPlacementNonMesure = "killsource_placement_vies_non_mesurees"
	// metricPlacementFragsHorsVie : frags recevables qu'aucune vie du tueur ne couvre — avant sa
	// premiere vie, ou tueur sans vie nommee. Comptes (decision V3), jamais rattaches.
	metricPlacementFragsHorsVie = "killsource_placement_frags_hors_vie"
	metricPlacementWriteFail    = "killsource_placement_erreurs_ecriture"
	metricPlacementSansPortee   = "killsource_placement_matchs_sans_portee"
	// metricPlacementPontRefuse : le pont slot->xuid n'est pas publiable — les lignes s'ecrivent,
	// chaque vie entiere « non situee », rien n'est mesure (decision V1 amendee le 2026-09-29,
	// lot V2b : sans ligne, le rattrapage re-selectionnerait le match a chaque passe).
	metricPlacementPontRefuse = "killsource_placement_pont_non_publiable"
	// metricPlacementPorteursLus : matchs dont la garde de mode a fait PAYER une lecture de
	// porteurs (le surcout accepte). metricPlacementSansCalage : le document assemble n'avait pas
	// de calage d'horloge, aucun portage n'a pu etre date.
	metricPlacementPorteursLus = "killsource_placement_porteurs_lus"
	metricPlacementSansCalage  = "killsource_placement_porteurs_sans_calage"
)

// PlacementRev — la revision du placement des vies, ecrite dans `decoder_rev` de CHAQUE ligne de
// `match_life_placement`.
//
// DISTINCTE D'[IsolationDecoderRev] : le contenu de `match_lives` ne change pas quand la mesure
// d'une vie change, et l'inverse. Le rattrapage (`matchsAJour`, cmd_backfill_killsource_selection.go)
// exige une passe a CETTE revision pour tout match qui a des vies ; le backlog automatique du
// post-sync ne la lit pas (decision V12 : un deploiement ne relance aucun redecodage de lui-meme).
//
// ELLE MONTE QUAND LES PORTAGES LUS CHANGENT, et non seulement la mesure : `carrier_ms` vient des
// calques de porteur (drapeau, crane, couronne, bombe) que [replay.PortagesAuSync] relit. Les socles
// du drapeau se lisent aussi dans le film (`replay/flag_film_bases.go`) ; le portage de la bombe
// suit les changements d'arme tenue, que la grammaire lit sur la marche des trames, l'ancrage
// derriere elle (grammar `grammar-2026-10-06.4`) — les portages, donc les durees portees, en
// dependent.
const PlacementRev = "placement-2026-10-06-v1"

// PorteeDuRadar rend la portee du radar d'une variante (`game_variant_name` tel que la base le
// porte), en metres. `connue` faux = variante absente de la table : la ligne s'ecrit sans portee
// (`radar_m` et `beyond_ms` NULL). Une portee connue est strictement positive.
type PorteeDuRadar func(variante string) (metres float64, connue bool)

// depsDuPlacement : ce que le placement demande en plus du materiau de la passe. Tout est
// optionnel : sans libelles ni objectifs, le calque du drapeau degrade comme a la cuisson ; sans
// portee, les lignes s'ecrivent sans part hors radar.
type depsDuPlacement struct {
	libelles  replay.LabelCatalog
	objectifs *replay.MapObjectivesCatalog
	portee    PorteeDuRadar
}

// AvecPorteeDuRadar injecte la portee du radar par variante. Chainable. En production, c'est
// [KillSourceCollector.AvecCapture] qui l'appelle, avec la portee que
// [CaptureDepuisCatalogue] a chargee (plan Emprise vies, lot V2b).
func (c *KillSourceCollector) AvecPorteeDuRadar(p PorteeDuRadar) *KillSourceCollector {
	c.placement.portee = p
	return c
}

// projeterPlacementDesVies ecrit `match_life_placement` a partir de ce que la passe de positions
// a deja lu. Appelee APRES les vies, et seulement si elles sont ecrites (cf. l'appel dans
// `ecrireLesDeuxPasses`). `journal` est la passe FUSIONNEE du journal : ses morts et sa
// publiabilite.
func (c *KillSourceCollector) projeterPlacementDesVies(
	ctx context.Context, matchID string, mat materiauDIsolement, ids MatchIdentities,
	journal persist.KillSourceBatch,
) {
	portages, lus := c.portagesDuMatch(ctx, matchID, mat, ids)
	radar := c.placement.porteeDe(ids.Variante)
	vies, bilan := replay.PlacementDesVies(replay.EntreePlacement{
		Positions: mat.positions, Registre: mat.registre, Equipes: equipesNumeriques(ids.Equipes),
		Journal: journalDuPlacement(journal), Portages: portages, RadarM: radar,
	})
	rows, nonMesurees := toPlacementRows(vies)
	if err := c.writePlacement(ctx, persist.LifePlacementBatch{
		MatchID: matchID, DecoderRev: PlacementRev, Rows: rows,
	}); err != nil {
		observability.AddInt(metricPlacementWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: placement — ecriture echouee (morts, positions et vies "+
			"restent ecrits)", "match_id", matchID, "err", err)
		return
	}
	publierLePlacement(ctx, matchID, bilanDuPlacement{
		vies: len(rows), nonMesurees: nonMesurees, radar: radar, frags: bilan, porteurs: lus,
	})
}

// portagesDuMatch lit les intervalles de port d'objectif, sous la garde de mode de l'entree.
//
// PONT NON PUBLIABLE : AUCUNE LECTURE PAYEE. `replay.PlacementDesVies` classe alors chaque instant
// « non situe », portage compris (un portage s'attribue par le meme pont) : le lire ne changerait
// aucune ligne.
func (c *KillSourceCollector) portagesDuMatch(
	ctx context.Context, matchID string, mat materiauDIsolement, ids MatchIdentities,
) (map[uint64][]replay.IntervalleDePort, replay.BilanPortages) {
	if !mat.registre.PontPubliable() {
		return nil, replay.BilanPortages{}
	}
	portages, lus := replay.PortagesAuSync(ctx, c.placement.entreeDesPorteurs(ctx, matchID, mat, ids))
	// LES REPLIS DE LA LECTURE DES PORTEURS (revue finale P1-b, 2026-10-02) rejoignent le compteur
	// de la passe, publie a la sortie du film (`publierReplisDeLaPasse`). UNE fois, ici, le seul
	// appelant : leurs sources (pose des largeurs, calques, consultations) ne sont comptees par
	// aucun autre etage du collecteur.
	replisDeLaPasse(ctx).Cumuler(lus.Replis.Rapport())
	return portages, lus
}

// entreeDesPorteurs assemble ce que le collecteur donne a `replay.PortagesAuSync`. PURE hors du
// catalogue deja charge : c'est la COUTURE par laquelle un test pince ce que la production
// transmet — meme raison qu'[entreeDuRegistre]. La feuille du match, par exemple, ne sert au pont
// par manche que sur certains films : son retrait laisserait verts les temoins qui n'en ont pas
// besoin.
func (d depsDuPlacement) entreeDesPorteurs(
	ctx context.Context, matchID string, mat materiauDIsolement, ids MatchIdentities,
) replay.EntreePorteursAuSync {
	return replay.EntreePorteursAuSync{
		MatchID: matchID, Film: mat.film, Contexte: mat.contexte, Carte: mat.carte,
		Variante: ids.Variante, ProfilDeBalayage: mat.profil, Identite: mat.identite,
		Lignes: ids.Feuille, Equipes: ids.Equipes, Socles: d.soclesDe(ctx, matchID, ids.CarteID),
		Libelles: d.libelles,
	}
}

// soclesDe rend les socles de drapeau de la carte (catalogue d'objectifs, par `map_id` — la
// MEME cle et la MEME projection que la cuisson). Absence = socles lus dans le film par le calque,
// comme a la cuisson : journalisee en Debug, le calque reste lu.
func (d depsDuPlacement) soclesDe(ctx context.Context, matchID, carteID string) []replay.FlagSpawn {
	if d.objectifs == nil || carteID == "" {
		return nil
	}
	entree, err := d.objectifs.Lookup(carteID)
	if err != nil {
		slog.DebugContext(ctx, "killsource: placement — carte hors catalogue d'objectifs, socles "+
			"de drapeau a lire dans le film", "match_id", matchID, "map_id", carteID)
		return nil
	}
	return entree.SoclesDeDrapeau()
}

// porteeDe rend la portee du radar de la variante, ou nil si elle est inconnue.
func (d depsDuPlacement) porteeDe(variante string) *float64 {
	if d.portee == nil {
		return nil
	}
	metres, connue := d.portee(variante)
	if !connue {
		return nil
	}
	return &metres
}

// journalDuPlacement traduit la passe fusionnee du journal : un xuid illisible (bot, nom non
// resolu) devient 0 — « non resolu » pour `replay.PlacementDesVies`. Le TUEUR est celui du fil
// (`FeedKillerXUID`), le meme que celui des positions de kill.
func journalDuPlacement(b persist.KillSourceBatch) []replay.FragDuJournal {
	out := make([]replay.FragDuJournal, 0, len(b.Deaths))
	for i := range b.Deaths {
		d := &b.Deaths[i]
		tueur, _ := parseXUID(d.FeedKillerXUID)
		victime, _ := parseXUID(d.VictimXUID)
		out = append(out, replay.FragDuJournal{
			TueurXUID: tueur, VictimeXUID: victime, TempsMS: int64(d.TimeMS), Publiable: b.Publishable,
		})
	}
	return out
}

// toPlacementRows traduit les mesures pures en lignes ecrivables, et compte les vies non
// mesurees (mediane nulle).
func toPlacementRows(vies []replay.PlacementVie) ([]persist.LifePlacementInsert, int) {
	out := make([]persist.LifePlacementInsert, 0, len(vies))
	nonMesurees := 0
	for i := range vies {
		v := &vies[i]
		if v.MedianeM == nil {
			nonMesurees++
		}
		out = append(out, persist.LifePlacementInsert{
			XUID: strconv.FormatUint(v.XUID, 10), StartMS: v.DebutMS, EndMS: v.FinMS,
			DurationMS: v.DureeMS, MeasuredMS: v.MesureMS, MedianM: v.MedianeM,
			BeyondMS: v.HorsRadarMS, RadarM: v.RadarM, CarrierMS: v.PorteurMS,
			TeamDownMS: v.EquipeATerreMS, UnplacedMS: v.NonSitueMS,
			TeammateUnplacedMS: v.CoequipierNonSitueMS, Kills: v.Frags,
		})
	}
	return out, nonMesurees
}

// writePlacement : l'ecriture, sous son PROPRE lease court — meme raison que writeIsolationFacts
// (le lease RW de shared est la ressource la plus disputee du process, ADR 0013).
func (c *KillSourceCollector) writePlacement(ctx context.Context, batch persist.LifePlacementBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", batch.MatchID, err)
	}
	defer release()
	return persist.NewLifePlacementPersister(db).PersistPass(ctx, batch)
}

// bilanDuPlacement : ce qu'une passe de placement publie.
type bilanDuPlacement struct {
	vies, nonMesurees int
	radar             *float64
	frags             replay.BilanPlacement
	porteurs          replay.BilanPortages
}

// publierLePlacement : les compteurs de sante (ADR 0009) et la trace de la passe.
func publierLePlacement(ctx context.Context, matchID string, b bilanDuPlacement) {
	horsVie := b.frags.FragsAvantLaPremiereVie + b.frags.FragsSansVie
	observability.AddInt(metricPlacementMatches, 1)
	observability.AddInt(metricPlacementLives, int64(b.vies))
	observability.AddInt(metricPlacementNonMesure, int64(b.nonMesurees))
	observability.AddInt(metricPlacementFragsHorsVie, int64(horsVie))
	if b.radar == nil {
		observability.AddInt(metricPlacementSansPortee, 1)
	}
	if b.frags.PontNonPubliable {
		observability.AddInt(metricPlacementPontRefuse, 1)
	}
	l := b.porteurs.Lectures
	if l.Statborg || l.Equipes || l.ObjetsDuMonde || l.ArmesTenues {
		observability.AddInt(metricPlacementPorteursLus, 1)
	}
	if b.porteurs.SansCalage {
		observability.AddInt(metricPlacementSansCalage, 1)
	}
	slog.InfoContext(ctx, "killsource: placement des vies ecrit",
		"match_id", matchID, "vies", b.vies, "vies_non_mesurees", b.nonMesurees,
		"portee_connue", b.radar != nil, "pont_non_publiable", b.frags.PontNonPubliable,
		"frags_rattaches", b.frags.FragsRattaches,
		"frags_avant_premiere_vie", b.frags.FragsAvantLaPremiereVie,
		"frags_sans_vie", b.frags.FragsSansVie, "frags_non_publiables", b.frags.FragsNonPubliables,
		"frags_tueur_inconnu", b.frags.FragsTueurInconnu, "frags_camp_inconnu", b.frags.FragsCampInconnu,
		"trahisons", b.frags.Trahisons, "porteurs_gardes", fmt.Sprintf("%+v", b.porteurs.Gardes),
		"porteurs_lectures", fmt.Sprintf("%+v", l), "portages", b.porteurs.Intervalles,
		"portages_sans_calage", b.porteurs.SansCalage, "drapeau_ouverts", b.porteurs.DrapeauOuverts,
		"porteurs_xuid_illisibles", b.porteurs.XUIDIllisibles)
}

// profilCalibre rend le profil que `killsource` a calibre sur le film, ou nil sans resultat de
// decodage. `replay.PortagesAuSync` le pose sur le contexte avant ses lectures, dans l'ordre de
// la cuisson (`poserProfilPuisCarte`).
func profilCalibre(res *decfilm.Result) *decfilm.ProfilDeBalayage {
	if res == nil {
		return nil
	}
	return &res.ProfilCalibre
}
