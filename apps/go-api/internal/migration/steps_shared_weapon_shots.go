package migration

// steps_shared_weapon_shots.go — table `match_weapon_shots` : LA VENTILATION DES TIRS PAR ARME.
// Grain `match x joueur x arme`, UNE SEULE colonne de mesure : le nombre de tirs decodes.
//
// ─── CE QUE CETTE TABLE APPORTE, ET CE QU ELLE N APPORTE PAS ──────────────────────────────
//
// LE TOTAL PAR JOUEUR N APPORTE RIEN DE NEUF : `match_participants.shots_fired` le donne deja,
// et il le donne mieux (c est la reference). LA SEULE INFORMATION NOUVELLE EST LA VENTILATION —
// combien de tirs avec QUELLE arme. C est pour ca que le grain descend jusqu a l arme et que la
// table n a pas de colonne « total » : le total est un `SUM`, et il n a jamais ete l objet.
//
// ─── UNE SEULE COLONNE DE MESURE, ET C EST UN REFUS, PAS UN MANQUE ────────────────────────
//
//	shots_decoded    STOCKE       un fire-event est un tir, EXACTEMENT, sans coefficient
//	                              (loi k = 1 : mediane 0.994 sur 3 129 observations de mode
//	                              standard, 84.0 % dans une tolerance de 10 % ; controle negatif
//	                              par permutation intra-film 0 sur 500 tirages).
//	touches          REFUSE       le champ `HitLikely` du scanner annonce 75-79 % de touches
//	                              quand la precision reelle mediane vaut 0.446 : sur un joueur,
//	                              314 « touches » pour 130 reelles. Publier une colonne de
//	                              touches DEPUIS CE CHAMP serait publier un chiffre faux d un
//	                              facteur ~2. LA QUESTION N EST PAS FERMEE POUR AUTANT : le bit
//	                              `eventStart+106` polarite 0 bat le taux constant d un facteur
//	                              1.8 hors echantillon sur 1 556 joueurs — mais il vaut ZERO sur
//	                              les armes a projectile lent et les populations ne sont pas
//	                              reconciliees. Rien n est publiable aujourd hui ; la colonne
//	                              n est pas interdite pour toujours (7ter.80 (8)(9)).
//	tete / corps     HORS PORTEE  jamais mesure.
//	coefficient      REFUSE       un tableau de coefficients par arme, appris sur des films
//	  par arme                    disjoints, fait DEUX FOIS PIRE que k = 1 sur l erreur du joueur
//	                              typique (0.0450 contre 0.0216). C est de l absorption de
//	                              residu, pas un modele.
//
// ─── L IDENTIFIANT D ARME : DEUX UNITES QUI NE SE MELANGENT JAMAIS ────────────────────────
//
//	weapon_id        ICI          identifiant filmshell 64 bits, celui de
//	                              `metadata.weapon_labels.weapon_id` (UBIGINT, 39 armes).
//	                              Il vient du fire-event, a `eventStart+40` sur 64 bits.
//	source_tag       AILLEURS     le tag `jpt!` 32 bits du dead-state, colonne de
//	                              `match_kill_events`. AUTRE ESPACE, AUTRE LARGEUR.
//
// Les confondre produirait des jointures vides et silencieuses. Le nom de l arme n est PAS
// stocke, pour la meme raison que dans `match_kill_events` : il se resout a la lecture
// (`metadata.weapon_labels`), et re-etiqueter un referentiel ne doit jamais exiger de redecoder
// 949 films.
//
// ─── LA PORTE DE PUBLICATION, ET CE QU ELLE COUTE ─────────────────────────────────────────
//
// Une ventilation n est publiable que si le TOTAL decode du joueur tombe a +-10 % de son
// `shots_fired`. CETTE PORTE EXIGE LA REFERENCE API. Aucun detecteur offline ne la remplace :
// le taux de pas du compteur de tirs correle a r = 0.15 avec l erreur, et une porte sur le
// NOMBRE D ARMES ne gagne que +3.8 points en jetant 46.6 % des observations. On assume donc une
// porte qui a besoin de l API, et on ECRIT SON RESULTAT ici — le lecteur ne la recalcule pas.
//
// ⚠ `publishable = FALSE` NE VEUT PAS DIRE LA MEME CHOSE QUE DANS `match_kill_events`. La-bas
// il veut dire « juste en agregat, faux ligne par ligne ». ICI IL VEUT DIRE : NE PAS UTILISER.
// Une ventilation hors tolerance est fausse EN AGREGAT AUSSI — c est le total qui est faux. Un
// lecteur de cumul ne peut pas « rattraper » par la somme ce que la porte vient de refuser.
//
// ─── TROIS RESERVES QUI VOYAGENT AVEC LA DONNEE ───────────────────────────────────────────
//
//  1. FIESTA ET GRAND FORMAT NE SONT PAS LIVRABLES : 6.9 % et 38.8 % des joueurs seulement
//     tombent dans la tolerance, et 5.1 % du grand format SUR-attribue de plus du double.
//     Pire : 168 observations ou l API dit `shots_fired = 0` EXACTEMENT et ou le decodeur
//     attribue 32 090 tirs (zero cas en mode standard). Ce n est pas une perte, c est une
//     ERREUR D ATTRIBUTION. La porte les refuse une par une — il n y a donc PAS de colonne
//     « mode » ici : c est la mesure qui tranche, pas une liste de modes tenue a la main.
//  2. LES ARMES RARES ET LOURDES SONT SOUS-ESTIMEES (Skewer : 5 204 events sur 2 214 414).
//     Aucune population « arme dominante » n existe pour les calibrer, donc aucun correctif
//     n est applique — un correctif non calibrable serait une invention.
//  3. UN DEFICIT INEXPLIQUE SUBSISTE — MAIS PAS AVEC LES CHIFFRES QUI ONT CIRCULE. Les valeurs
//     « MA40 0.928 · Sidekick 0.925 contre BR75 0.981 · Bandit 1.012 » sont IRREPRODUCTIBLES et
//     NE DOIVENT PLUS ETRE CITEES. Mesure courante du meme pipeline (mode standard, ratio par
//     joueur) : MA40 0.971 · Mk51 Sidekick 1.004 · BR75 1.007 · Bandit Evo 1.007. CE QUI SURVIT
//     A DEUX LOTS INDEPENDANTS : le MA40 a un deficit, plus grand que celui du BR75. CE QUI EST
//     CONTESTE : le deficit du Sidekick — a ne publier ni comme present ni comme absent.
//     Cf. `.ai/GUIDE_WEAPON_SHOTS.md` §3, RE_LOG 7ter.80 (3)(6) et 7ter.81 (9).
//
// ─── APPEND-ONLY, MEME UNITE DE GENERATION QUE `match_kill_events` ────────────────────────
//
// PK technique `id`, `written_at`, INSERT purs (ADR 0026/0030). La vue `_latest` retient LA
// DERNIERE PASSE PAR MATCH (`decode_pass`), jamais la derniere ligne par cle : l unite de
// production est le FILM ENTIER. Un `_latest` ligne par ligne laisserait survivre les armes
// d une passe precedente que la nouvelle ne decode plus — une ventilation qui n a jamais existe.
//
// ─── L ABSENCE DE LIGNE : CE QU ELLE DIT, ET CE QU ELLE NE DIT PAS ────────────────────────
//
// Pas de ligne `(match, joueur, arme)` = AUCUN fire-event decode pour cette arme. Ce n est une
// mesure de zero QUE si la ligne du joueur est `publishable` : sur un joueur dont la porte a
// echoue, l absence ne dit rien du tout. Une absence de mesure n est jamais une mesure d absence.
//
// ⚠ LIMITE STRUCTURELLE, ECRITE PLUTOT QUE SUBIE : le verdict de la porte vit sur les LIGNES du
// joueur. Un joueur dont AUCUNE arme n est decodee n a donc AUCUNE ligne — et son echec est
// INVISIBLE ici, indistinguable d un joueur qui n a pas tire. Le cas symetrique (l API annonce
// des tirs, le decodeur n en trouve aucun) est donc SILENCIEUX dans cette table. Le compter
// exige de croiser avec `match_participants` sur les joueurs du match. Ce n est pas reparable a
// ce grain : une ligne « zero tir » serait une mesure fabriquee.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_weapon_shots_v1",
		TargetDB:    TargetShared,
		Description: "Table append-only match_weapon_shots (match x joueur x arme, tirs decodes du film) + vue _latest par passe de decodage",
		ApplySchema: applyMatchWeaponShots,
	})
}

// applyMatchWeaponShots cree la table, son index et sa vue. Idempotente. La table etant
// NET-NEUVE, le piege « CREATE TABLE IF NOT EXISTS n ajoute jamais une PK a une table
// existante » ne s applique pas.
//
// Relocation : ce step appartient fonctionnellement a Halo Infinite (les films Halo 5 ont un
// autre format), et rejoindra `internal/games/halo_infinite/migrations/` avec le lot de la
// voie B (ADR 0025) — comme `shared_match_kill_events_v1` et `shared_weapon_kills_v3`.
func applyMatchWeaponShots(db *sql.DB) error {
	if err := execScript(db, ddlMatchWeaponShots); err != nil {
		return err
	}
	return execScript(db, ddlMatchWeaponShotsLatest)
}

// ddlMatchWeaponShots : la table + son unique index.
//
// UN SEUL INDEX, meme argument que `match_kill_events` : DuckDB est colonnaire, un index ART ne
// sert que les acces ponctuels, et le seul acces ponctuel reel est « les tirs de CE match ».
// Chaque index en plus coute a l INSERT et elargit la surface ART.
const ddlMatchWeaponShots = `
	CREATE SEQUENCE IF NOT EXISTS match_weapon_shots_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_weapon_shots (
		-- identite technique (append-only : PK non naturelle, ADR 0026)
		id                BIGINT    PRIMARY KEY DEFAULT nextval('match_weapon_shots_id_seq'),
		match_id          VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe de decodage d UN film. Unite de generation.
		decode_pass       VARCHAR   NOT NULL,
		-- decoder_rev : version du decodeur. Sert a savoir QUELS matchs redecoder apres un
		-- changement, au lieu de tout redecoder.
		decoder_rev       VARCHAR   NOT NULL,
		written_at        TIMESTAMP NOT NULL DEFAULT now(),

		-- ── LE JOUEUR ───────────────────────────────────────────────────────────────────
		-- player_index : indice de replication BRUT lu dans le fire-event, sur 5 bits
		-- (eventStart+31). C est LA quantite qui ne depend d aucune resolution : elle est
		-- conservee pour qu un nom qui surprend puisse etre confronte a l indice.
		player_index      SMALLINT  NOT NULL,
		-- player_xuid : NULL = BOT (un bot n a pas de xuid) ou indice non rattache au roster.
		-- La resolution indice -> xuid est la responsabilite du COLLECTEUR, et son echec est
		-- une donnee, pas une erreur.
		player_xuid       VARCHAR,

		-- ── LA MESURE, ET RIEN D AUTRE ──────────────────────────────────────────────────
		-- weapon_id : identifiant filmshell 64 bits, joignable a metadata.weapon_labels.
		-- ⚠ CE N EST PAS le tag jpt! 32 bits de match_kill_events.source_tag. Deux espaces
		-- distincts qui ne se joignent jamais l un a l autre.
		weapon_id         UBIGINT   NOT NULL,
		-- shots_decoded : nombre de fire-events decodes. UN FIRE-EVENT EST UN TIR, exactement,
		-- sans coefficient. Aucune colonne de touches : le champ HitLikely du decodeur annonce
		-- 75-79 % la ou la precision reelle mediane vaut 0.446.
		shots_decoded     INTEGER   NOT NULL,

		-- ── LA PORTEE, ECRITE AVEC LE RESULTAT ──────────────────────────────────────────
		-- publishable : le TOTAL decode du joueur tombait-il a +-10 % de son shots_fired ?
		-- ⚠ FALSE veut dire NE PAS UTILISER — pas « juste en agregat ». Une ventilation hors
		-- tolerance est fausse en agregat aussi.
		publishable       BOOLEAN   NOT NULL,
		-- gate_reason : le verdict de la porte, en toutes lettres. Toujours renseigne, y
		-- compris quand la porte passe : une portee qui ne s ecrit que sur les echecs laisse
		-- croire que le succes n a pas de portee.
		gate_reason       VARCHAR   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_weapon_shots_match
		ON match_weapon_shots(match_id, written_at);
`

// ddlMatchWeaponShotsLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026 — une lecture brute
// sert les lignes d une passe de decodage precedente).
//
// La vue NE FILTRE PAS sur `publishable` : filtrer ici rendrait le refus invisible, et un
// lecteur ne saurait pas distinguer « aucun tir decode » de « decodage refuse ». Le filtre
// appartient au lecteur, et il est OBLIGATOIRE pour tout usage de la ventilation.
const ddlMatchWeaponShotsLatest = `
	CREATE OR REPLACE VIEW match_weapon_shots_latest AS
	SELECT s.*
	FROM match_weapon_shots AS s
	QUALIFY s.decode_pass = FIRST_VALUE(s.decode_pass) OVER (
		PARTITION BY s.match_id ORDER BY s.written_at DESC, s.id DESC
	);
`
