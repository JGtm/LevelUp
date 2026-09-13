package replayartifacts

// flaggrabsnet.go — LES PRISES DE DRAPEAU, BRUTES ET NETTES, PERSISTEES AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Il LIT le calque de drapeau de l'artefact range (`flagCarries[].spans`), applique la regle
// du jonglage (`objectiveevents.NetFlagGrabs`) avec la fenetre declaree par le titre, et
// transporte le resultat vers `persist.FlagGrabsNetPersister`.
//
// # POURQUOI LA MESURE SE FAIT ICI, ET PAS A LA CUISSON
//
// C'est l'inverse du choix de `bombstats.go`, et la difference est justifiee dans
// `replay/flag_grabs_net_bridge.go` : le document porte DEJA la chronologie exacte du portage,
// a une grille de 100 ms, la ou la fenetre vaut 1,5 s. Rien n'est reconstruit, et surtout : le
// parc entier devient lisible SANS RECUISSON — les artefacts publient `flagCarries` depuis le
// schema 14, et la cuisson en lot est interdite.
//
// # LA FORME, CALQUEE SUR bombstats.go
//
//	SUR DISQUE, PAS LE BLOB   le document est deja lu et deserialise par [Deriver].
//	PROJETER PUIS ECRIRE      toutes les projections se font AVANT d'acquerir le writer.
//	SEGMENT WRITER COURT      acquis APRES toute cuisson, relache aussitot.
//	VIA `internal/persist`    INSERT-only (ADR 0019/0026/0030).
//
// # DEUX PORTES, ET ELLES DISENT DEUX CHOSES DIFFERENTES
//
//	`film.flag_grabs_net`                  « ce titre sait lire ce calque »
//	`[flag_grabs_net].flag_juggle_window_s`  « voici ma regle de jonglage »
//
// L'absence de l'une OU de l'autre donne le meme silence propre : aucune ligne ecrite,
// grandeur non mesuree — jamais des zeros. Un titre qui declare la capability sans la fenetre
// est une configuration INCOMPLETE, et elle se journalise en WARN : ce n'est pas un choix,
// c'est un oubli.
//
// # LES SILENCES ORDINAIRES, ET ILS NE SE COMPTENT PAS EN DEFAUTS
//
//	film qui n'est pas du CTF          l'immense majorite du parc (63 artefacts sur 76 au
//	                                   releve du 2026-09-13) : aucun calque exploitable,
//	                                   DEBUG ;
//	film CTF sans aucune prise nommee  le pont n'a nomme personne : rien a ecrire, DEBUG.

import (
	"context"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// passePrisesNettesPrete : une projection réussie, prête à persister.
type passePrisesNettesPrete struct {
	matchID string
	batch   persist.FlagGrabsNetBatch
}

// FenetreJonglage lit la fenêtre de jonglage déclarée par le titre dans son
// `regulation.toml`. Rend (0, false) quand le titre ne la déclare pas OU quand le fichier est
// illisible — dans les deux cas la grandeur n'est pas publiée, et `err` distingue l'oubli de
// configuration (err non nil) du choix assumé (err nil).
//
// EXPORTÉE parce que le backfill (`cmd/levelup`) en a besoin AUSSI, et qu'une seconde lecture
// du même fichier ailleurs ferait deux vérités de la même règle.
func FenetreJonglage(repoRoot, titleSlug string) (time.Duration, bool, error) {
	path := filepath.Join(repoRoot, "config", "titles", titleSlug, "mappings", "regulation.toml")
	set, err := mappings.LoadRegulationFromFile(path)
	if err != nil {
		return 0, false, err
	}
	w, ok := set.FlagJuggleWindow()
	return w, ok, nil
}

// ProjeterPrisesNettes tire d'UN document rangé la passe à écrire. Rend une passe VIDE
// (MatchID vide) quand il n'y a RIEN à écrire — document sans calque de drapeau exploitable
// (le cas NORMAL de tout film hors CTF), ou film de CTF dont aucune prise n'a de porteur
// nommé. Ni l'un ni l'autre n'est un défaut.
//
// EXPORTÉE pour la même raison que [FenetreJonglage] : le backfill rejoue EXACTEMENT cette
// projection sur les artefacts déjà rangés.
func ProjeterPrisesNettes(matchID string, doc *replay.ReplayDocument, window time.Duration) persist.FlagGrabsNetBatch {
	tracks, openings, ok := replay.FlagTracksOf(doc)
	if !ok {
		return persist.FlagGrabsNetBatch{}
	}
	res := objectiveevents.NetFlagGrabs(tracks, window)
	if !res.Measured || len(res.Players) == 0 {
		// AUCUN PORTAGE NOMME : la passe ne prouve pas qu'elle a su LIRE le calque, donc elle
		// n'ecrit rien — et surtout pas un roster entier a zero, qui affirmerait « personne
		// n'a pris de drapeau » sur un film ou le pont n'a simplement nomme personne.
		return persist.FlagGrabsNetBatch{}
	}
	out := persist.FlagGrabsNetBatch{MatchID: matchID, WindowMS: res.WindowMS, Openings: openings}
	out.Players = completerRosterAZero(doc, res.Players)
	return out
}

// completerRosterAZero rend les lignes de la passe : celles qu'on a mesurees, PLUS une ligne
// A ZERO pour chaque humain du roster que la mesure ne nomme pas.
//
// POURQUOI CE ZERO EST UNE MESURE, ET PAS UN REMPLISSAGE. Sur un match dont le calque a ete
// LU, « ce joueur n'a jamais touche le drapeau » est un fait ; sans sa ligne, il serait
// indistinguable d'un joueur d'un match sans film — les deux rendraient « non mesure » a
// l'ecran. La ligne a zero est ce qui separe les deux.
//
// LES BOTS N'EN ONT PAS : un bot n'a pas de xuid (`RosterEntry.Bot`, XUID vide), et la table
// est clef par xuid. Leur absence n'est pas un zero, c'est une identite que la base ne porte
// pas — exactement ce que dit le schema de `match_bomb_stats`.
func completerRosterAZero(doc *replay.ReplayDocument, mesures []objectiveevents.FlagGrabsNetPlayer) []persist.FlagGrabsNetRow {
	out := make([]persist.FlagGrabsNetRow, 0, len(mesures)+len(doc.Roster))
	vus := make(map[string]bool, len(mesures))
	for _, p := range mesures {
		out = append(out, persist.FlagGrabsNetRow{XUID: p.XUID, Raw: p.Raw, Net: p.Net})
		vus[p.XUID] = true
	}
	for _, r := range doc.Roster {
		if r.XUID == "" || vus[r.XUID] {
			continue
		}
		vus[r.XUID] = true
		out = append(out, persist.FlagGrabsNetRow{XUID: r.XUID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	return out
}

// capabilitePrisesNettesArmee dit si le titre déclare `film.flag_grabs_net`, et DIT POURQUOI
// quand la réponse est non — même contrat que capabilityBombeArmee : un TOML illisible est un
// INCIDENT (WARN + compteur d'échecs), une clé absente est une configuration de titre (DEBUG).
func capabilitePrisesNettesArmee(ctx context.Context, d Deps) (armee, incident bool) {
	return porteCapability(ctx, d, games.CapFilmFlagGrabsNet, "prises nettes", nil)
}

// regleJonglageArmee résout la fenêtre du titre et dit, comme ci-dessus, si son absence est un
// choix (DEBUG) ou un incident (WARN + compteur).
func regleJonglageArmee(ctx context.Context, d Deps) (window time.Duration, armee, incident bool) {
	w, ok, err := FenetreJonglage(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: prises nettes non produites — regulation.toml illisible",
			"titleSlug", d.TitleSlug, "err", err)
		return 0, false, true
	}
	if !ok {
		slog.DebugContext(ctx, "post-sync: prises nettes — titre sans fenêtre de jonglage déclarée",
			"titleSlug", d.TitleSlug)
		return 0, false, false
	}
	return w, true, false
}

// persisterPrisesNettes projette puis écrit les prises de drapeau des artefacts rangés du lot.
// Best-effort de bout en bout : aucun échec ne remonte au cycle, aucun ne se tait.
func persisterPrisesNettes(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) {
	if len(lus) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	window, armee, incident := portesPrisesNettes(ctx, d)
	if !armee {
		if incident {
			// Configuration illisible : un DÉFAUT, compté comme tel — et aucun de ces matchs
			// n'est marqué dérivé (même règle que les stats d'Assaut).
			observability.AddIntT(titre, CompteurPrisesNettesEchecs, int64(len(lus)))
			b.echecLot(lus)
		}
		return
	}
	prets := projeterPrisesNettesDuLot(ctx, lus, window)
	if len(prets) == 0 {
		// AUCUN film de CTF dans le lot est le cas NORMAL et majoritaire.
		return
	}
	ecrirePrisesNettes(ctx, d, b, titre, prets)
}

// portesPrisesNettes franchit les DEUX portes (capability, puis règle) et rend la fenêtre.
func portesPrisesNettes(ctx context.Context, d Deps) (window time.Duration, armee, incident bool) {
	if ok, inc := capabilitePrisesNettesArmee(ctx, d); !ok {
		return 0, false, inc
	}
	w, ok, inc := regleJonglageArmee(ctx, d)
	if !ok {
		return 0, false, inc
	}
	return w, true, false
}

// ecrirePrisesNettes acquiert le writer, écrit les passes, et journalise le bilan.
func ecrirePrisesNettes(
	ctx context.Context, d Deps, b *bilanDerivations, titre string, prets []passePrisesNettesPrete,
) {
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: prises nettes NON persistées (aucun writer shared câblé sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurPrisesNettesEchecs, int64(len(prets)))
		echecPrisesNettes(b, prets)
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, prises nettes non persistées",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurPrisesNettesEchecs, int64(len(prets)))
		echecPrisesNettes(b, prets)
		return
	}
	defer release()
	p := persist.NewFlagGrabsNetPersister(db)
	ecrits, echecs := 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: écriture des prises nettes échouée",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			b.echec(prets[i].matchID)
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurPrisesNettesEcrites, int64(ecrits))
	observability.AddIntT(titre, CompteurPrisesNettesEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: prises nettes persistées",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}

// echecPrisesNettes enregistre au bilan que ces passes n'ont pas été persistées faute de
// writer : sans cette trace, la marque de dérivation se poserait sur un match dont RIEN n'a
// été écrit.
func echecPrisesNettes(b *bilanDerivations, prets []passePrisesNettesPrete) {
	ids := make([]string, 0, len(prets))
	for i := range prets {
		ids = append(ids, prets[i].matchID)
	}
	echecFauteDeWriter(b, ids)
}

// projeterPrisesNettesDuLot projette tous les documents du lot, AVANT tout writer. Rend les
// passes NON VIDES.
func projeterPrisesNettesDuLot(ctx context.Context, lus []artefactLu, window time.Duration) []passePrisesNettesPrete {
	prets := make([]passePrisesNettesPrete, 0, len(lus))
	for _, a := range lus {
		batch := ProjeterPrisesNettes(a.matchID, a.doc, window)
		if batch.MatchID == "" {
			slog.DebugContext(ctx, "post-sync: prises nettes — artefact sans calque de drapeau exploitable",
				"match_id", a.matchID)
			continue
		}
		prets = append(prets, passePrisesNettesPrete{matchID: a.matchID, batch: batch})
	}
	return prets
}
