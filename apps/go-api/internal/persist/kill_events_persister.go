// Package persist — kill_events_persister.go : ecriture INSERT-ONLY d une PASSE DE DECODAGE de
// film dans `shared.match_kill_events` (1 ligne par MORT, les deux verites cote a cote).
//
// POURQUOI UN PERSISTER DEDIE, ET PAS `SharedPersister` : `SharedPersister.Persist` est un no-op
// si `batch.Shared.Match == nil` et un skip si le match existe deja dans `match_registry`. Or une
// passe de decodage arrive TOUJOURS sur un match deja insere — le film n est pas pret au sync
// primaire, il arrive un cycle plus tard. `SharedPersister` est donc structurellement inutilisable
// ici, exactement comme pour `EventsCompletionPersister` (meme raison, meme precedent).
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun DELETE, aucun UPDATE, aucun ON CONFLICT.
// La table est append-only ; « remplacer » une passe consiste a en ecrire une nouvelle, et c est
// la vue `match_kill_events_latest` qui ne rend que la DERNIERE PASSE par match. Ce persister n a
// donc rien a faire figurer dans l allowlist de `internal/sync/no_art_patterns_test.go` — il
// n emet aucun des motifs a risque, et c est la vraie preuve que la conception tient.
//
// LA SOURCE DU DEGAT EST FACULTATIVE, ET C EST UNE DECISION DE CONCEPTION, PAS UNE TOLERANCE.
// Ce persister n est PAS le seul producteur prevu de `match_kill_events` : le chemin
// `highlight_events` (celui qui alimente `killer_victim_pairs` aujourd hui) en est un SECOND, et
// il ne mesure QUE le credit. 1 908 matchs au registre, 1 325 porteurs de paires, 949 films en
// cache — et les films Theater expirent cote serveur, donc le manque est DEFINITIF. Exiger une
// source reviendrait a refuser le journal tueur -> victime d au moins 28 % des matchs, c est-a-dire
// a faire de la source la CONDITION D EXISTENCE d une mort. Les deux verites sont a EGALITE :
// `SourceTag == 0` se dit donc « source non mesuree » et s ecrit NULL.
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb (comme tous les
// persisters shared). `txBeginner` accepte aussi bien *sql.DB qu un LeasedWriter.

package persist

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// KillEventInsert : UNE MORT, et les deux reponses a « qu est-ce qui l a tuee ».
//
// Les deux sous-ensembles de champs sont COMPLETS chacun de son cote et aucun ne se derive de
// l autre : un consommateur qui ne veut que le credit doit pouvoir ignorer entierement la source,
// et inversement. Il n existe volontairement aucun champ « vrai tueur » ni « arme corrigee ».
//
// Le decodeur ne rend que des NOMS (le film ne porte pas de XUID cote replication). La resolution
// nom -> xuid est la responsabilite du COLLECTEUR, contre le roster du match. Un xuid vide n est
// pas une erreur : c est le cas normal d une victime BOT, et le cas honnete d un nom que le
// roster n a pas su rattacher.
type KillEventInsert struct {
	// TimeMS : instant de la mort, tel que le kill-feed le date.
	TimeMS int

	// VictimGamertag : requis. Une mort sans victime n est pas une mort.
	VictimGamertag string
	// VictimXUID : vide = BOT, ou nom non resolu. Colonne NULL.
	VictimXUID string

	// ── VERITE 1 : LE CREDIT (kill-feed) ──────────────────────────────────────────────────
	FeedKillerGamertag string
	FeedKillerXUID     string
	// FeedPresent : le kill-feed porte-t-il cette MORT ? Faux pour une victime bot.
	// ⚠ PIEGE : `FeedPresent == false` avec un `FeedKillerGamertag` RENSEIGNE est le cas
	// NORMAL d une mort de bot — le KILL est au feed, c est la MORT qui n y est pas. Ne jamais
	// « corriger » l un par l autre. Cas symetrique (`ReadOrigin == "tueur-bot"`) : le TUEUR
	// est un bot, donc `FeedKillerGamertag` renseigne SANS `FeedKillerXUID` — un bot n a pas
	// de XUID, et cette combinaison est ACCEPTEE (elle est meme la seule correcte).
	FeedPresent bool

	// AssistGamertag / AssistXUID : l assistant declare par le kill-event. Vide = aucun, OU
	// refuse (voir AssistRejected). Lire AssistKnown avant de conclure.
	AssistGamertag string
	AssistXUID     string
	// AssistKnown : un kill-event a-t-il ete attache a cette mort ? FAUX veut dire QU ON NE
	// SAIT PAS, jamais « pas d assistant ».
	AssistKnown bool
	// AssistIndex : indice de replication BRUT, nil quand le champ est absent (le decodeur
	// rend -1 dans ce cas — ecrire -1 en base fabriquerait un indice).
	AssistIndex *int
	// AssistRejected : motif de refus, vide s il n y en a pas.
	AssistRejected string
	// AssistExtra : nombre d assistants distincts SUPPLEMENTAIRES observes sur cette mort.
	// LE GARDE-FOU de l hypothese « un seul assistant ». Vaut 0 partout aujourd hui.
	AssistExtra int

	// ── VERITE 2 : LA SOURCE DU DEGAT FATAL ───────────────────────────────────────────────
	// SourceTag : identifiant jpt! 32 bits. FACULTATIF — 0 veut dire « source non mesuree »
	// (film absent ou expire, ou producteur qui ne lit que le kill-feed) et s ecrit NULL.
	// 0 n est pas un identifiant du catalogue : il n y a donc pas d ambiguite.
	SourceTag uint32
	// SourceCategory : identifiant moteur du modificateur de degat ("None", "Headshot",
	// "SilentMelee", ...). Voyage AVEC SourceTag : les deux viennent de la meme lecture du
	// dead-state. Renseigner l un sans l autre est refuse. « Aucun modificateur » se dit
	// "None", pas "" — "" veut dire « non mesure », comme SourceTag == 0.
	SourceCategory string

	// KillerDamagePct / AssistDamagePct : parts de degats en pourcentage entier, nil = non
	// mesure. ⚠ AssistDamagePct DOIT etre nil sans assistant nomme : sans assistant, le champ
	// porte une constante par film qui ne signifie rien (elle vaut 20 sur certains films, un
	// pourcentage parfaitement credible). Le persister le REFUSE.
	//
	// ⚠ AUCUN PLAFOND A 100 : 1,7 % des kill-events attaches a de vraies morts nommees portent
	// une valeur superieure, jusqu a 228. Refuser ces lignes jetterait de la donnee reelle sous
	// pretexte qu elle contredit une interpretation (« c est un pourcentage ») qui est, elle,
	// une lecture a quatre jambes convergentes et non une chaine d appels prouvee.
	KillerDamagePct *uint8
	AssistDamagePct *uint8

	// Diverges : la source appartient a la victime alors que le feed credite un autre joueur.
	// N a de sens QUE si la source a ete mesuree : sans source, la colonne est ecrite NULL
	// (« divergence non mesurable »), jamais FALSE — une absence de mesure n est pas une mesure
	// d absence. Declarer Diverges avec SourceTag == 0 est refuse.
	Diverges bool

	// ReadPath / ReadOrigin : LA PORTEE de la ligne. Requis tous les deux — une mesure porte
	// sur ce qu elle a mesure, et sa portee s ecrit AVEC elle.
	ReadPath   string
	ReadOrigin string
}

// KillSourceBatch : LE RESULTAT D UNE PASSE DE DECODAGE D UN FILM.
//
// L unite de production est le MATCH ENTIER : un decodage rend toute la liste des morts. C est
// pour ca que la vue `_latest` retient une passe ENTIERE et jamais un melange de passes.
type KillSourceBatch struct {
	// MatchID : porte ici et pas deduit de `Shared.Match` — une passe arrive sur un match deja
	// insere, donc `Shared.Match` est nil sur ce chemin.
	MatchID string `json:"match_id"`
	// DecoderRev : version du decodeur. Requis : le jour ou le decodage change, c est la seule
	// facon de savoir QUELS matchs redecoder au lieu de tous les redecoder.
	DecoderRev string `json:"decoder_rev"`
	// Publishable : la passe autorisait-elle la publication LIGNE PAR LIGNE ? Faux = agregat
	// seulement (marge de bijection nulle, ou sante en alerte). Ecrit sur CHAQUE ligne.
	Publishable bool `json:"publishable"`
	// Deaths : les morts, dans l ordre rendu par le decodeur.
	Deaths []KillEventInsert `json:"deaths,omitempty"`
}

// KillSourcePersister ecrit une passe de decodage dans shared_matches_v2.duckdb.
type KillSourcePersister struct {
	db txBeginner
}

// NewKillSourcePersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewKillSourcePersister(db txBeginner) *KillSourcePersister {
	return &KillSourcePersister{db: db}
}

// Persist ecrit le sous-batch `batch.Shared.KillSource` s il existe. No-op sinon.
//
// C est le chemin BatchBuilder ; le chemin direct (completion tardive d un film, backfill) passe
// par [KillSourcePersister.PersistPass].
func (p *KillSourcePersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: KillSourcePersister.Persist: batch nil")
	}
	if batch.Shared.KillSource == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.KillSource)
}

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
//
// Toutes les lignes d une passe portent le MEME `decode_pass` et le MEME `written_at` — c est ce
// qui rend la vue `_latest` capable de retenir une generation entiere plutot qu un melange.
//
// Une passe VIDE n est pas une erreur mais n est pas non plus anodine : elle est ignoree (aucune
// ligne ecrite, donc la vue continue de servir la passe precedente) et LOGGUEE. Ecrire zero ligne
// serait indistinguable d un match sans mort.
func (p *KillSourcePersister) PersistPass(ctx context.Context, in KillSourceBatch) error {
	if err := validateKillSourceBatch(in); err != nil {
		return err
	}
	if len(in.Deaths) == 0 {
		slog.WarnContext(ctx, "persist: passe killsource vide, aucune ligne ecrite",
			"match_id", in.MatchID, "decoder_rev", in.DecoderRev)
		return nil
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", in.MatchID, err)
	}

	extra, err := p.insertKillPass(ctx, in, pass, time.Now().UTC())
	if err != nil {
		return err
	}

	// Le garde-fou de l hypothese « un seul assistant » : il vit dans la donnee
	// (assist_extra_count) ET remonte ici. Silencieux tant qu il vaut 0.
	if extra > 0 {
		slog.WarnContext(ctx, "persist: morts portant plus d un assistant distinct — "+
			"l hypothese de schema « un seul assistant » est en defaut, migrer vers une table fille",
			"match_id", in.MatchID, "assist_extra_total", extra, "decoder_rev", in.DecoderRev)
	}
	return nil
}

// insertKillPass ouvre la transaction et ecrit toutes les lignes. Retourne le total
// d assistants supplementaires observes sur la passe.
func (p *KillSourcePersister) insertKillPass(
	ctx context.Context, in KillSourceBatch, pass string, now time.Time,
) (int, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("persist: BeginTx kill_events %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	stmt, err := tx.PrepareContext(ctx, insertKillEventSQL)
	if err != nil {
		return 0, fmt.Errorf("persist: prepare match_kill_events %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	extra := 0
	for i := range in.Deaths {
		d := &in.Deaths[i]
		if _, err := stmt.ExecContext(ctx, killEventArgs(in, d, pass, now)...); err != nil {
			return 0, fmt.Errorf("persist: INSERT match_kill_events %s@%d: %w", in.MatchID, d.TimeMS, err)
		}
		extra += d.AssistExtra
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("persist: Commit match_kill_events %s: %w", in.MatchID, err)
	}
	return extra, nil
}

const insertKillEventSQL = `
	INSERT INTO match_kill_events (
		match_id, decode_pass, decoder_rev, written_at, publishable,
		time_ms, victim_gamertag, victim_xuid,
		feed_killer_gamertag, feed_killer_xuid, feed_present,
		assist_gamertag, assist_xuid, assist_known, assist_index,
		assist_rejected, assist_extra_count,
		source_tag, source_category,
		killer_damage_pct, assist_damage_pct,
		diverges, read_path, read_origin
	) VALUES (
		?, ?, ?, ?, ?,
		?, ?, ?,
		?, ?, ?,
		?, ?, ?, ?,
		?, CAST(? AS UTINYINT),
		CAST(? AS UINTEGER), ?,
		CAST(? AS UTINYINT), CAST(? AS UTINYINT),
		?, ?, ?
	)`

// newDecodePassID tire l identifiant de generation d une passe.
//
// POURQUOI PAS `newBatchID()` : celui-ci fait `_, _ = rand.Read(...)`. Ailleurs un identifiant
// degrade ne coute qu un nom de fichier WAL moins joli ; ICI c est la cle de generation de la vue
// `_latest`. Si le tirage echouait silencieusement, toutes les passes porteraient la meme valeur
// et la vue rendrait TOUTES les passes de chaque match — exactement le melange qu elle existe pour
// interdire, et sans aucun symptome visible. Probabilite faible, consequence maximale : on refuse
// d ecrire plutot que de publier une passe non distinguable. Le cout d un refus est un redecodage.
func newDecodePassID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("tirage de decode_pass impossible, passe non ecrite "+
			"(un decode_pass non unique ferait rendre a la vue _latest un melange de passes): %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// killEventArgs serialise une mort. Les champs texte vides deviennent NULL : « pas de valeur » et
// « chaine vide » ne doivent pas se confondre en base. Idem pour la source : `SourceTag == 0` est
// « non mesuree », donc NULL sur les trois colonnes qui en dependent (tag, categorie, divergence).
func killEventArgs(in KillSourceBatch, d *KillEventInsert, pass string, now time.Time) []any {
	sourceKnown := d.SourceTag != 0
	return []any{
		in.MatchID, pass, in.DecoderRev, now, in.Publishable,
		d.TimeMS, d.VictimGamertag, nullIfEmpty(d.VictimXUID),
		nullIfEmpty(d.FeedKillerGamertag), nullIfEmpty(d.FeedKillerXUID), d.FeedPresent,
		nullIfEmpty(d.AssistGamertag), nullIfEmpty(d.AssistXUID), d.AssistKnown, nullIfNilInt(d.AssistIndex),
		nullIfEmpty(d.AssistRejected), int64(d.AssistExtra),
		nullUnless(sourceKnown, int64(d.SourceTag)), nullUnless(sourceKnown, d.SourceCategory),
		nullIfNilPct(d.KillerDamagePct), nullIfNilPct(d.AssistDamagePct),
		nullUnless(sourceKnown, d.Diverges), d.ReadPath, d.ReadOrigin,
	}
}

// validateKillSourceBatch : ce que le persister REFUSE au niveau de la passe. La validation passe
// AVANT la transaction : un refus ne laisse aucune ligne derriere lui.
func validateKillSourceBatch(in KillSourceBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: KillSourceBatch.MatchID vide")
	}
	if in.DecoderRev == "" {
		return fmt.Errorf("persist: KillSourceBatch.DecoderRev vide (%s) — "+
			"une passe doit dire quel decodeur l a produite", in.MatchID)
	}
	for i := range in.Deaths {
		if err := validateKillEvent(&in.Deaths[i]); err != nil {
			return fmt.Errorf("persist: %s mort #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validateKillEvent : ce que le persister REFUSE au niveau de la ligne, cote MORT et cote SOURCE.
// Les refus propres a l assistant vivent dans [validateKillEventAssist] — meme contrat, decoupe
// pour rester sous le plafond de complexite du depot (12).
//
// CE QUI EST EXIGE : une victime, un instant, et une PORTEE. Rien d autre. En particulier PAS la
// source du degat : la rendre obligatoire ferait d elle la condition d existence d une mort, et
// exclurait de la table les ~28 % de matchs dont le film a expire cote serveur (cf. en-tete).
//
// CE QUI EST REFUSE, ET POURQUOI :
//
//	(1) une mort sans victime ni instant, ou une ligne sans portee ;
//	(2) une DEMI-SOURCE : tag sans categorie ou categorie sans tag. Les deux sortent de la meme
//	    lecture du dead-state — l une sans l autre signale une lecture ratee, pas une mesure ;
//	(3) une DIVERGENCE DECLAREE SANS SOURCE : `Diverges` compare les deux verites, il est
//	    indefinissable quand l une des deux n a pas ete mesuree ;
//	(4) les trois etats de l assistant, et le piege des pourcentages (voir la fonction dediee).
//
// CE QUI N EST PAS REFUSE, ET DOIT ETRE ACCEPTE : `VictimXUID` vide (mort de BOT),
// `FeedPresent = false` avec un `FeedKillerGamertag` renseigne (le KILL est au feed, la MORT non),
// et le cas symetrique `FeedKillerGamertag` renseigne SANS `FeedKillerXUID` (le TUEUR est un bot,
// `ReadOrigin == "tueur-bot"` — un bot n a pas de XUID).
//
// CE QUI N EST PLUS REFUSE : une part de degats > 100. Le plafond existait au motif qu une valeur
// hors 0..100 « signalait une lecture au mauvais endroit » — c est faux : 1,7 % des kill-events
// attaches a de vraies morts NOMMEES en portent une, jusqu a 228. Le refus jetait silencieusement
// de la donnee reelle pour proteger une interpretation. Cout supplementaire : la validation
// passant avant la transaction, UNE SEULE ligne a 228 faisait echouer la passe ENTIERE.
func validateKillEvent(d *KillEventInsert) error {
	if d.VictimGamertag == "" {
		return errors.New("VictimGamertag vide (une mort sans victime n est pas une mort)")
	}
	if d.TimeMS < 0 {
		return fmt.Errorf("TimeMS negatif (%d)", d.TimeMS)
	}
	if d.ReadPath == "" || d.ReadOrigin == "" {
		return errors.New("ReadPath/ReadOrigin vide (la portee s ecrit AVEC le resultat)")
	}
	if (d.SourceTag == 0) != (d.SourceCategory == "") {
		return fmt.Errorf("source a moitie renseignee (tag=%d categorie=%q) — tag et categorie "+
			"sortent de la meme lecture du dead-state, l un sans l autre est une lecture ratee, "+
			"pas une mesure. Source non mesuree = les DEUX vides", d.SourceTag, d.SourceCategory)
	}
	if d.Diverges && d.SourceTag == 0 {
		return errors.New("divergence declaree (Diverges=true) sans source mesuree " +
			"(la divergence compare les deux verites, elle est indefinissable sans l une d elles)")
	}
	return validateKillEventAssist(d)
}

// validateKillEventAssist : les refus propres a l assistant.
//
//	(a) l assistant a TROIS etats — on refuse de nommer un assistant tout en declarant qu on ne
//	    sait pas (`AssistKnown == false`), ce qui detruirait la distinction ;
//	(b) LE PIEGE DES POURCENTAGES — sans assistant nomme, le second bloc porte une constante par
//	    film qui ne signifie rien (elle vaut 20 sur certains films). On refuse de l ecrire plutot
//	    que de le documenter et de le subir. CE REFUS-LA RESTE : il ne plafonne rien, il exige
//	    seulement qu une part d assistant ait un assistant.
func validateKillEventAssist(d *KillEventInsert) error {
	if !d.AssistKnown && (d.AssistGamertag != "" || d.AssistXUID != "") {
		return errors.New("assistant nomme alors que AssistKnown=false " +
			"(« on ne sait pas » et « pas d assistant » ne se confondent pas)")
	}
	if d.AssistExtra < 0 {
		return fmt.Errorf("AssistExtra negatif (%d)", d.AssistExtra)
	}
	if d.AssistDamagePct != nil && d.AssistGamertag == "" {
		return errors.New("AssistDamagePct renseigne sans assistant nomme " +
			"(sans assistant le champ porte une constante par film, il ne veut rien dire)")
	}
	return nil
}

// ─── Helpers de serialisation ──────────────────────────────────────────────────────────────

// nullIfEmpty : "" -> NULL. « Pas de valeur » et « chaine vide » sont deux choses differentes.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullUnless : NULL des que la mesure dont la valeur depend n a pas eu lieu. Sert les trois
// colonnes de la verite SOURCE : sans lecture du dead-state, `diverges = FALSE` serait une mesure
// d absence fabriquee a partir d une absence de mesure.
func nullUnless(measured bool, v any) any {
	if !measured {
		return nil
	}
	return v
}

// nullIfNilInt : nil -> NULL. Sert `assist_index`, dont le -1 du decodeur ne doit JAMAIS entrer
// en base (il fabriquerait un indice de replication qui n existe pas).
func nullIfNilInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// nullIfNilPct : nil -> NULL. « Non mesure » n est jamais zero.
func nullIfNilPct(p *uint8) any {
	if p == nil {
		return nil
	}
	return int64(*p)
}
