package replayartifacts

// padtiers_preparation.go — CE QUE LA FAMILLE DES NIVEAUX D'ARMES FAIT AVANT D'ECRIRE : porte de
// capability, references du titre, identite des matchs (segment de LECTURE), projection.
//
// # LA PANNE QUE CE FICHIER FERME (lot L4.1 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26)
//
// Du 2026-09-14 au 2026-09-27, AUCUN lot n'etait projete au fil de l'eau. Deux defauts empiles :
//
//  1. `DerivationsDeps` ne portait pas de segment de lecture : la vue [Deps] rendue aux familles
//     avait `WithRead == nil` sur les trois chemins (post-sync, rattrapage, depot d'ouvrier), et
//     la lecture des identites rendait « illisible » a chaque passe. Journalise en WARN (30
//     occurrences du 17 au 19/09), jamais compte au bilan : la marque de derivation se posait
//     quand meme, et le rattrapage ne reprenait jamais ces matchs.
//  2. Latent derriere le premier : la lecture avait lieu APRES les familles qui ecrivent. Le
//     segment d'ecriture de la passe est memoise et tenu jusqu'a la fin de [Deriver]
//     ([writerUnique]) ; en B-swap, un `Provider.Get` ouvert pendant qu'il est tenu attend le
//     retour en lecture seule que ce meme segment empeche, jusqu'a `ErrSwapTimeout`.
//
// D'ou la forme : [Deriver] appelle [preparerNiveauxDArmes] AVANT la premiere famille qui ecrit,
// et [ecrireNiveauxDArmes] a la place qu'occupait la famille dans l'ordre des ecritures.
//
// # IDENTITE INDISPONIBLE = MATCH NON MARQUE
//
// Un match projetable dont l'identite ne se lit pas est un ECHEC au bilan : sa marque ne se pose
// pas, et le rattrapage le rejouera (les familles append-only supersedent, le T0 a sa garde).
// C'est la difference avec une REFERENCE manquante — installation incomplete de CETTE famille,
// permanente : un echec bouclerait le rattrapage a jamais (correctif de revue du 2026-09-14,
// inchange, cf. [referencesDuTitre]).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
)

// preparationNiveaux : les passes projetees AVANT tout segment d'ecriture, pretes a ecrire.
type preparationNiveaux struct {
	titre string
	prets []passeNiveauxPrete
}

// preparerNiveauxDArmes fait tout ce qui precede l'ecriture des niveaux d'armes. Best-effort :
// aucun echec ne remonte, aucun ne se tait ; un match qu'elle n'a pas pu projeter faute
// d'identite est inscrit au bilan (pas de marque).
func preparerNiveauxDArmes(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) preparationNiveaux {
	titre := ctxkeys.TitleSlug(ctx)
	prep := preparationNiveaux{titre: titre}
	if len(lus) == 0 {
		return prep
	}
	armee, incident := capabiliteNiveauxArmee(ctx, d)
	if !armee {
		if incident {
			observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(len(lus)))
		}
		return prep
	}
	ref, reg, ok := referencesDuTitre(ctx, d, titre, len(lus))
	if !ok {
		return prep
	}
	projetables := lotProjetable(ctx, lus)
	if len(projetables) == 0 {
		return prep
	}
	identites, lisibles := identitesDuLot(ctx, d, b, projetables)
	prep.prets = projeterNiveauxDuLot(ctx, lisibles, ref, reg, identites)
	return prep
}

// lotProjetable garde les documents qui PEUVENT donner une passe ([projetableNiveaux]). Les
// autres (artefact anterieur au schema 30, film sans humain) n'ont rien a ecrire : ce n'est pas
// un defaut, ils se marquent comme une derivation jouee.
func lotProjetable(ctx context.Context, lus []artefactLu) []artefactLu {
	out := make([]artefactLu, 0, len(lus))
	for _, a := range lus {
		if !projetableNiveaux(a.doc) {
			slog.DebugContext(ctx, "post-sync: niveaux d'armes — artefact sans prise nommable",
				"match_id", a.matchID, "schema", a.doc.SchemaVersion)
			continue
		}
		out = append(out, a)
	}
	return out
}

// referencesDuTitre charge la reference des emplacements et les regles du titre. Rend
// (nil, nil, false) quand l une des deux manque — cette famille s abstient SEULE, sans toucher
// au bilan des autres.
func referencesDuTitre(ctx context.Context, d Deps, titre string, n int) (
	*ReferenceEmplacements, *mappings.RegulationSet, bool,
) {
	ref, err := ChargerReferenceEmplacements(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: niveaux d'armes — reference des emplacements illisible, "+
			"cette famille s'abstient (les autres gardent leur marque)",
			"titleSlug", d.TitleSlug, "err", err)
		observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(n))
		return nil, nil, false
	}
	reg, err := ReglesDepartsAleatoires(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: niveaux d'armes — regulation.toml illisible, "+
			"cette famille s'abstient (les autres gardent leur marque)",
			"titleSlug", d.TitleSlug, "err", err)
		observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(n))
		return nil, nil, false
	}
	return ref, reg, true
}

// identitesDuLot lit map_id et pair_name du lot par un SEGMENT DE LECTURE court, et rend les
// artefacts dont l'identite est connue.
//
// ON NE DEVINE PAS : un match dont le registre ne rend pas la ligne n'a pas d'identite, et le
// projeter sans elle lui preterait un mode regulier (un niveau « base » ecrit sur des Fiesta).
// Il n'est pas projete ET il est inscrit au bilan, pour que sa marque ne se pose pas.
func identitesDuLot(
	ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu,
) (map[string]IdentiteMatchNiveaux, []artefactLu) {
	if d.WithRead == nil {
		slog.ErrorContext(ctx, "post-sync: niveaux d'armes — aucun segment de lecture cable sur ce "+
			"chemin, identites de match illisibles : lot NON projete, NON marque",
			"titleSlug", d.TitleSlug, "matchs", len(lus))
		echecSansIdentite(ctx, b, lus)
		return nil, nil
	}
	ids := matchIDsDuLot(lus)
	var out map[string]IdentiteMatchNiveaux
	var lecture error
	lu := false
	d.WithRead(ctx, "niveaux d'armes", func(sharedDB *sql.DB) {
		lu = true
		out, lecture = IdentitesDesMatchs(ctx, sharedDB, ids)
	})
	if !lu || lecture != nil {
		// Segment non ouvert (le porteur du lease a deja dit pourquoi) ou requete en echec.
		slog.ErrorContext(ctx, "post-sync: niveaux d'armes — identites de match illisibles : lot "+
			"NON projete, NON marque (le rattrapage le rejouera)",
			"titleSlug", d.TitleSlug, "matchs", len(lus), "segment_ouvert", lu, "err", lecture)
		echecSansIdentite(ctx, b, lus)
		return nil, nil
	}
	gardes := make([]artefactLu, 0, len(lus))
	var manquants []artefactLu
	for _, a := range lus {
		if _, connu := out[a.matchID]; connu {
			gardes = append(gardes, a)
			continue
		}
		manquants = append(manquants, a)
	}
	if len(manquants) > 0 {
		slog.ErrorContext(ctx, "post-sync: niveaux d'armes — matchs absents du registre : NON "+
			"projetes, NON marques", "titleSlug", d.TitleSlug, "matchs", matchIDsDuLot(manquants))
		echecSansIdentite(ctx, b, manquants)
	}
	return out, gardes
}

// echecSansIdentite compte ces matchs en echec et les inscrit au bilan : sans cette trace, la
// marque de derivation se poserait sur un match dont les niveaux n'ont jamais ete ecrits.
func echecSansIdentite(ctx context.Context, b *bilanDerivations, lus []artefactLu) {
	observability.AddIntT(ctxkeys.TitleSlug(ctx), CompteurNiveauxArmesEchecs, int64(len(lus)))
	b.echecLot(lus)
}

// matchIDsDuLot rend les identifiants d un lot d artefacts lus.
func matchIDsDuLot(lus []artefactLu) []string {
	ids := make([]string, 0, len(lus))
	for i := range lus {
		ids = append(ids, lus[i].matchID)
	}
	return ids
}

// IdentitesDesMatchs lit `map_id` et `pair_name` pour un lot de matchs.
//
// EXPORTEE pour le backfill. Requete indexee sur la cle primaire du registre : quelques
// millisecondes sur un lot de cycle.
func IdentitesDesMatchs(ctx context.Context, db *sql.DB, ids []string) (map[string]IdentiteMatchNiveaux, error) {
	out := make(map[string]IdentiteMatchNiveaux, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := `SELECT match_id, COALESCE(map_id, ''), COALESCE(pair_name, '')
	      FROM match_registry WHERE match_id IN (?` + strings.Repeat(",?", len(ids)-1) + `)`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("niveaux d'armes: lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, mapID, pair string
		if err := rows.Scan(&id, &mapID, &pair); err != nil {
			return nil, fmt.Errorf("niveaux d'armes: scan match_registry: %w", err)
		}
		out[id] = IdentiteMatchNiveaux{MapID: mapID, PairName: pair}
	}
	return out, rows.Err()
}
