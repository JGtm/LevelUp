package persist

// lives_mates_persister.go — la table `match_death_mates` : l'etat et la position de chaque
// coequipier de la victime a l'instant d'une mort du journal (plan Tactique coequipiers, L1).
//
// Elle s'ecrit par [LivesPersister.PersistPass], dans la MEME transaction et sous le MEME
// `decode_pass` que `match_lives` et `match_death_context` : trois tables d'un meme decodage,
// une seule passe. INSERT purs, aucun UPDATE/DELETE/ON CONFLICT (ADR 0019/0026) ; lecture par la
// vue `match_death_mates_latest` uniquement.

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DeathMateInsert — un coequipier de la victime a l'instant d'une mort. La cle de la mort
// (VictimXUID, TimeMS) est celle de `match_kill_events` et de `match_death_context`.
//
// X, Y et DistanceM sont renseignes ENSEMBLE si et seulement si State vaut
// [EtatCoequipierVisible] : seule une position repliquee autorise une coordonnee.
type DeathMateInsert struct {
	VictimXUID string   `json:"victim_xuid"`
	TimeMS     int64    `json:"time_ms"`
	MateXUID   string   `json:"mate_xuid"`
	State      string   `json:"state"`
	X          *float64 `json:"x,omitempty"`
	Y          *float64 `json:"y,omitempty"`
	DistanceM  *float64 `json:"distance_m,omitempty"`
}

// Les trois etats d'un coequipier. Ils DOUBLENT les constantes `Etat*` de
// `games/halo_infinite/film/replay/death_context.go` (`persist` n'importe pas un paquet
// d'analyse pour valider une colonne) ; `archlint/no_life_cause_divergence_test.go` interdit aux
// deux jeux de diverger.
const (
	EtatCoequipierVisible   = "visible"
	EtatCoequipierEnAttente = "waiting"
	EtatCoequipierHorsDeVue = "out_of_sight"
)

// insertDeathMateRows ecrit les coequipiers des morts. INSERT pur, table append-only.
func insertDeathMateRows(ctx context.Context, tx *sql.Tx, in LivesBatch, pass string, now time.Time) error {
	for i := range in.Mates {
		m := &in.Mates[i]
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_death_mates (
				match_id, decode_pass, decoder_rev, written_at,
				victim_xuid, time_ms, mate_xuid, state, x, y, distance_m
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.MatchID, pass, in.DecoderRev, now,
			m.VictimXUID, m.TimeMS, m.MateXUID, m.State, nullable(m.X), nullable(m.Y), nullable(m.DistanceM))
		if err != nil {
			return fmt.Errorf("persist: INSERT match_death_mates %s/%s/%d/%s: %w",
				in.MatchID, m.VictimXUID, m.TimeMS, m.MateXUID, err)
		}
	}
	return nil
}

// nullable rend la valeur pointee, ou nil (NULL SQL) pour un pointeur nil.
func nullable(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// cleDeMort identifie une mort du journal.
type cleDeMort struct {
	victime string
	tempsMS int64
}

// bilanDesCoequipiers : ce que les lignes d'une mort disent, a confronter a son contexte.
type bilanDesCoequipiers struct {
	total, visibles, enAttente, horsDeVue int
	plusProche                            *float64
}

// validateMates refuse un lot de coequipiers qui contredirait `match_death_context`.
//
// LES REFUS SONT DES ERREURS, PAS DES CORRECTIONS : une ligne de coequipier qui ne retombe pas sur
// les comptes de sa mort est le symptome d'un producteur casse, et la corriger ici publierait une
// donnee inventee. Pour chaque mort : somme des lignes = `teammates_total`, comptes par etat =
// colonnes, distance du plus proche visible = `nearest_teammate_m`. Une mort a `teammates_total`
// > 0 sans aucune ligne est refusee (somme nulle contre total non nul).
//
// CONTRAT ASSUME : UN REFUS FAIT PERDRE LA PASSE ENTIERE (`PersistPass` valide tout AVANT d'ecrire) —
// donc aussi `match_lives` et `match_death_context` du match pour cette passe, pas seulement les
// coequipiers. Un producteur incoherent ne publie rien : la vue `_latest` garde la passe precedente.
func validateMates(in LivesBatch) error {
	lignes := make(map[cleDeMort]*bilanDesCoequipiers, len(in.Contexts))
	for i := range in.Mates {
		m := &in.Mates[i]
		if err := validateMate(m, in.MatchID, i); err != nil {
			return err
		}
		bilanDe(lignes, cleDeMort{m.VictimXUID, m.TimeMS}).ajouter(m)
	}
	// Deux contextes de MEME cle (doublon du journal) recevraient chacun l'ensemble des lignes de
	// la cle : on confronte donc la somme des contextes de la cle a la somme des lignes.
	annonce := make(map[cleDeMort]*bilanDesCoequipiers, len(in.Contexts))
	for i := range in.Contexts {
		c := &in.Contexts[i]
		a := bilanDe(annonce, cleDeMort{c.VictimXUID, c.TimeMS})
		a.total += c.TeammatesTotal
		a.visibles += c.TeammatesVisible
		a.enAttente += c.TeammatesWaiting
		a.horsDeVue += c.TeammatesOutOfSight
		if c.NearestTeammateM != nil {
			a.plusProche = c.NearestTeammateM
		}
	}
	for k := range lignes {
		if _, connue := annonce[k]; !connue {
			return fmt.Errorf("persist: match_death_mates %s: des coequipiers pour la mort %s a %d ms "+
				"qui n'a aucun contexte dans la meme passe", in.MatchID, k.victime, k.tempsMS)
		}
	}
	for k, a := range annonce {
		if err := confronter(in.MatchID, k, *a, *bilanDe(lignes, k)); err != nil {
			return err
		}
	}
	return nil
}

// bilanDe rend le bilan de la cle, cree a la demande.
func bilanDe(par map[cleDeMort]*bilanDesCoequipiers, k cleDeMort) *bilanDesCoequipiers {
	b := par[k]
	if b == nil {
		b = &bilanDesCoequipiers{}
		par[k] = b
	}
	return b
}

// ajouter compte une ligne de coequipier.
func (b *bilanDesCoequipiers) ajouter(m *DeathMateInsert) {
	b.total++
	switch m.State {
	case EtatCoequipierVisible:
		b.visibles++
		if b.plusProche == nil || *m.DistanceM < *b.plusProche {
			d := *m.DistanceM
			b.plusProche = &d
		}
	case EtatCoequipierEnAttente:
		b.enAttente++
	case EtatCoequipierHorsDeVue:
		b.horsDeVue++
	}
}

// confronter compare ce que le contexte annonce a ce que les lignes portent.
func confronter(matchID string, k cleDeMort, ctx, lignes bilanDesCoequipiers) error {
	if lignes.total != ctx.total || lignes.visibles != ctx.visibles ||
		lignes.enAttente != ctx.enAttente || lignes.horsDeVue != ctx.horsDeVue {
		return fmt.Errorf("persist: match_death_mates %s mort %s a %d ms: lignes total/visible/"+
			"waiting/out_of_sight = %d/%d/%d/%d, contexte = %d/%d/%d/%d — un coequipier manquant "+
			"ou en trop se cacherait dans l'ecart", matchID, k.victime, k.tempsMS,
			lignes.total, lignes.visibles, lignes.enAttente, lignes.horsDeVue,
			ctx.total, ctx.visibles, ctx.enAttente, ctx.horsDeVue)
	}
	if (lignes.plusProche == nil) != (ctx.plusProche == nil) ||
		(lignes.plusProche != nil && *lignes.plusProche != *ctx.plusProche) {
		return fmt.Errorf("persist: match_death_mates %s mort %s a %d ms: plus proche visible des "+
			"lignes (%v) != nearest_teammate_m du contexte (%v)", matchID, k.victime, k.tempsMS,
			lignes.plusProche, ctx.plusProche)
	}
	return nil
}

// validateMate refuse une ligne de coequipier incoherente avec son etat.
func validateMate(m *DeathMateInsert, matchID string, i int) error {
	if m.VictimXUID == "" || m.MateXUID == "" {
		return fmt.Errorf("persist: match_death_mates %s ligne #%d: victim_xuid ou mate_xuid vide", matchID, i)
	}
	switch m.State {
	case EtatCoequipierVisible, EtatCoequipierEnAttente, EtatCoequipierHorsDeVue:
	default:
		return fmt.Errorf("persist: match_death_mates %s ligne #%d: state %q inconnu", matchID, i, m.State)
	}
	avecPosition := m.X != nil && m.Y != nil && m.DistanceM != nil
	sansPosition := m.X == nil && m.Y == nil && m.DistanceM == nil
	if (m.State == EtatCoequipierVisible && !avecPosition) || (m.State != EtatCoequipierVisible && !sansPosition) {
		return fmt.Errorf("persist: match_death_mates %s ligne #%d: etat %q et position/distance "+
			"incoherents — seule une position repliquee autorise une coordonnee, et un visible en porte une",
			matchID, i, m.State)
	}
	if m.DistanceM != nil && *m.DistanceM < 0 {
		return fmt.Errorf("persist: match_death_mates %s ligne #%d: distance negative (%v)", matchID, i, *m.DistanceM)
	}
	return nil
}
