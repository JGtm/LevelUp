package migration

// steps_shared_kill_events.go — table `match_kill_events` : UNE LIGNE PAR MORT, et LES DEUX
// VERITES a cote l une de l autre. Remplacante annoncee de `killer_victim_pairs`.
//
// ─── POURQUOI UNE NOUVELLE TABLE ET PAS UN ALTER ──────────────────────────────────────────
//
// `killer_victim_pairs` ne sait pas si elle est un JOURNAL ou un CUMUL. Son en-tete dit
// « agregee, donc pas de PRIMARY KEY (les analytics font SUM(kill_count)) » ; la mesure dit
// l inverse : `kill_count` vaut 1 sur les 248 566 lignes de la prod, `SUM(kill_count)` est donc
// un `COUNT(*)` deguise. C est deja un journal, avec un en-tete qui ment.
//
// Le cout de ce mensonge est chiffre : l absence de PK autorise le flux primaire a INSERT sans
// supprimer pendant que la completion fait DELETE-then-INSERT — 46,8 % de la table est du
// doublon exact (248 566 lignes pour 132 313 cles distinctes, 423 matchs sur 1 325 touches) et
// les agregats carriere sont gonfles d un facteur 1,879 en moyenne, jusqu a 3,5x sur un duo.
//
// On ne repare pas ca par un ALTER. Une PK posee apres coup n arrive pas sur une table existante
// (`CREATE TABLE IF NOT EXISTS` ne l ajoute jamais — piege documente CLAUDE.md n6), et une PK
// naturelle sur une table dont on veut reecrire les lignes ramenerait l UPSERT, donc le bug ART.
// Table neuve, append-only.
//
// ─── LA GRANULARITE : UNE LIGNE PAR MORT ──────────────────────────────────────────────────
//
// L agregat se retrouve toujours par `GROUP BY` ; l inverse est impossible. Et la table porte
// desormais ce qu une paire agregee ne peut pas porter : l instant, l assistant, la source du
// degat, les parts de degats, la divergence, la portee de la mesure.
//
// Il n y a donc PAS de colonne `kill_count`. Un lecteur qui faisait `SUM(kill_count)` fait
// `COUNT(*)` — c est exactement ce qu il calculait deja sans le savoir.
//
// ─── LES DEUX VERITES, EN BASE AUSSI ──────────────────────────────────────────────────────
//
//	feed_killer_*     LE CREDIT      qui le jeu credite au kill-feed. Statistiques officielles.
//	source_tag        LA SOURCE      d ou vient le degat fatal, lu dans le dead-state de la
//	                                 VICTIME. Jamais l arme tenue, jamais le credit.
//
// Aucune des deux ne corrige l autre. Il n existe volontairement AUCUNE colonne « vrai tueur »
// ni « arme corrigee » : ce serait presenter une verite comme l amendement de l autre.
// `diverges` signale les cas ou elles ne designent pas le meme responsable (une roquette tiree
// trop pres d un mur) — la divergence EST l information.
//
// ─── LA SOURCE EST FACULTATIVE — ET C EST LA MEME DOCTRINE, PAS UN RELACHEMENT ────────────
//
// `source_tag` / `source_category` / `diverges` sont NULLABLES. NULL = « source non mesuree ».
//
// LA MESURE QUI TRANCHE : 1 908 matchs au registre, 1 325 porteurs de paires tueur -> victime,
// 949 films en cache. Les films Theater EXPIRENT cote serveur : le manque n est pas un retard,
// il est DEFINITIF. Une table qui exigerait un tag de source refuserait au moins 28 % des matchs
// — ils perdraient leur journal des morts pour cause d une mesure qu on ne peut plus faire.
//
// L ARGUMENT DE FOND, QUI VAUT MEME SI LE CHIFFRE BOUGE : ce fichier proclame que les deux
// verites sont A EGALITE. Rendre `feed_killer_gamertag` nullable et `source_tag` NOT NULL ferait
// de la source la CONDITION D EXISTENCE d une mort — la hierarchie que la doctrine interdit,
// ecrite dans le schema.
//
// CONSEQUENCE ASSUMEE — `match_kill_events` A DEUX PRODUCTEURS, pas un :
//
//	killsource (film)     les deux verites. read_path 'marche' | 'scan'.
//	highlight_events      LE CREDIT SEUL — c est deja la source de `killer_victim_pairs`.
//	                      source_tag/source_category/diverges NULL, assist_known FALSE.
//	                      read_path 'highlight-events', read_origin 'credit-seul'.
//
// Les deux se distinguent par la PROVENANCE (`read_path` / `read_origin` / `decoder_rev`), pas
// par la nullite d une colonne. ⚠ COROLLAIRE : la vue `_latest` retient LA DERNIERE PASSE par
// match — si les deux producteurs ecrivent le meme match, LE DERNIER GAGNE ENTIEREMENT. Un
// producteur credit-seul qui repasserait apres le decodeur de film EFFACERAIT la source de la
// lecture. C est une contrainte d ORDONNANCEMENT pour le brancheur, pas une contrainte de schema.
//
// ─── CE QU ON STOCKE, ET CE QU ON REFUSE DE STOCKER ───────────────────────────────────────
//
// LA REGLE : on ne stocke JAMAIS une resolution qui peut S AMELIORER ; on stocke toujours une
// mesure qui ne peut plus changer.
//
//	source_tag        STOCKE       identifiant 32 bits, ne depend d aucune table de nommage.
//	nom de l arme     PAS STOCKE   206 des 468 entrees du catalogue sortent « Autres » aujourd hui.
//	                               Figer « Autres » figerait a jamais ce qu on ignorait ce jour-la.
//	                               Le catalogue est embarque en Go (63 Ko) : le re-generer
//	                               re-etiquette TOUT l historique sans toucher une ligne de cette
//	                               table. Le chiffre qui tranche : re-seeder 468 entrees, contre
//	                               re-decoder 1 325 films a 8-30 s piece (3 a 11 heures).
//	source_category   STOCKE       enum de 10 valeurs GELEE dans le format de film (tir a la tete,
//	                               assassinat, ...). Elle ne peut pas s ameliorer.
//	read_path/origin  STOCKE       la PORTEE de la mesure. Une mesure porte sur ce qu elle a
//	publishable                    mesure : la portee s ecrit AVEC le resultat, pas a cote.
//
// ─── L ASSISTANT : TROIS ETATS, PAS DEUX ──────────────────────────────────────────────────
//
//	assist_known = FALSE                              ON NE SAIT PAS (aucun kill-event attache)
//	assist_known = TRUE  + assist_gamertag NULL       mesure : PAS d assistant
//	assist_known = TRUE  + assist_gamertag renseigne  l assistant, nomme
//
// Confondre les deux premiers fabriquerait des faits « 0 assist » jamais observes : 62 a 78 morts
// par film ont un « pas d assistant » MESURE, 3 a 7 n ont aucun kill-event attache.
//
// UN SEUL ASSISTANT — ET C EST UNE HYPOTHESE SOUS SURVEILLANCE. Le film n en encode qu un
// (deserialiseur lineaire sans boucle, deux largeurs d enregistrement seulement). Donc des
// colonnes simples, pas de LIST DuckDB : le cout d une LIST se paie a CHAQUE lecture pour
// representer une cardinalite mesuree a 1 sur 100 % DES KILL-EVENTS ATTACHES — c est LE
// denominateur, et il n a jamais ete « la population » : la grammaire ne lit qu UN champ
// d assistant par enregistrement (RE_LOG 7ter.76). Mais une hypothese de schema n est legitime
// que si une mesure la surveille : `assist_extra_count` compte les assistants distincts
// SUPPLEMENTAIRES vus sur une mort.
//
// ⚠ LE GARDE-FOU A BOUGE, ET LE DECLENCHEUR EST RECRIT (2026-08-03). Ce commentaire annoncait
// « 0 partout aujourd hui ; s il bouge, migrer vers une table fille ». La mesure du 2026-08-03 sur
// la base Halo Infinite dit : **5 lignes a 1, sur 5 matchs distincts, pour 124 694 lignes servies
// (0,004 %)** ; aucune ligne au-dela de 1. Le declencheur litteral est donc franchi, et pourtant
// l hypothese de schema TIENT — parce que ce que ce compteur mesure n est pas « une mort porte
// deux assistants » mais « DEUX KILL-EVENTS ATTACHES a la meme mort nomment des assistants
// differents » (cf. `killsource.Assist.Extra`, qui delimite lui-meme cette portee). Un seuil a
// zero sur une quantite d appariement, et non sur une cardinalite du format, ne pouvait rester
// vrai indefiniment.
//
// LE DECLENCHEUR MESURABLE QUI LE REMPLACE — migrer vers une table fille des que L UN des deux
// est atteint :
//
//	(a) UNE SEULE ligne a `assist_extra_count >= 2`. Deux surplus sur une meme mort ne
//	    s expliquent plus par un doublon d appariement : le format porterait bien plusieurs
//	    assistants, et les colonnes simples deviendraient fausses ;
//	(b) la part des lignes a `>= 1` depasse 0,1 % de `match_kill_events_latest` (125 lignes a
//	    l echelle actuelle, soit 25 fois la mesure). Au-dela, ce n est plus une anecdote
//	    d appariement et il faut instruire.
//
// Requete de surveillance :
//
//	SELECT COUNT(*) FILTER (WHERE assist_extra_count >= 2) AS declencheur_a,
//	       COUNT(*) FILTER (WHERE assist_extra_count >= 1) AS lignes,
//	       COUNT(*)                                        AS total
//	FROM match_kill_events_latest;
//
// La donnee etant DERIVEE de chunks en cache, cette migration se fera par un redecodage.
//
// ─── LES PARTS DE DEGATS, ET LE PIEGE QUI VA AVEC ─────────────────────────────────────────
//
// Le kill-event porte deux pourcentages entiers adjacents : la part du TUEUR puis celle de
// l ASSISTANT. Somme == 99 sur 71,8 % des couples propres, jamais 100, jamais 101 : signature du
// double arrondi par defaut de deux parts complementaires. En solo, le premier vaut 100.
//
// RESERVE A GARDER EN TETE PARTOUT OU CES DEUX COLONNES SONT LUES : le chemin de donnees entre
// le kill-event du film et `KillerPercentageDamageDone` / `AssistantPercentageDamageDone` N EST
// PAS DEMONTRE. Quatre jambes convergentes — adjacence et ordre dans le modele, type entier
// confirme, somme == 99, collision avec une capture Cheat Engine — PAS une chaine d appels.
//
// ⚠ AUCUN PLAFOND A 100, NI EN BASE NI AU PERSISTER. 1,7 % des kill-events attaches a de vraies
// morts NOMMEES portent une valeur superieure, jusqu a 228. Une premiere version refusait ces
// lignes au motif qu une valeur hors 0..100 « signalait une lecture au mauvais endroit » : c est
// l interpretation qui plafonnait la donnee, pas la donnee qui confirmait l interpretation.
//
// ⚠ PIEGE : SANS ASSISTANT, LE SECOND BLOC PORTE UNE CONSTANTE PAR FILM QUI NE SIGNIFIE RIEN —
// et elle vaut 20 sur certains films, un pourcentage parfaitement credible. L ABSENCE
// D ASSISTANT SE LIT DANS `assist_gamertag`/`assist_known`, JAMAIS DANS `assist_damage_pct`.
// Le persister REFUSE d ecrire un `assist_damage_pct` sans assistant nomme : le piege est rendu
// impossible a l ecriture plutot que documente et subi.
//
// LE RESIDU (`99 - tueur - assistant`) n est PAS stocke : il est CALCULE PAR LA VUE `_latest`,
// sous le nom `damage_pct_residual`. Il peut etre NEGATIF, il n est pas borne, et il ne dit rien
// de QUI a inflige le reste.
//
// ─── APPEND-ONLY, ET LA PASSE COMME UNITE DE GENERATION ───────────────────────────────────
//
// PK technique `id`, colonne `written_at`, INSERT purs (ADR 0026/0030). Mais la vue `_latest` ne
// retient PAS « la derniere ligne par mort » : elle retient LA DERNIERE PASSE DE DECODAGE PAR
// MATCH (`decode_pass`). C est la difference qui compte : l unite de production est le MATCH
// ENTIER (un decodage rend toute sa liste de morts). Un `_latest` ligne par ligne melangerait
// deux passes — si une passe B publie 95 morts la ou A en publiait 99, les 4 lignes de A
// survivraient et pollueraient. `decode_pass` est genere une fois par passe cote persister ; la
// vue est donc exacte meme si deux passes partageaient la meme microseconde.
//
// ─── LE SORT DE `killer_victim_pairs` : ELLE RESTE ────────────────────────────────────────
//
// Elle n est NI supprimee NI touchee par cette migration. 8 lecteurs de production la lisent
// (5 en cumul `SUM(kill_count)`, 2 en journal, 1 en sonde de presence `NOT EXISTS`). Le chemin de
// retrait, dans cet ordre :
//
//	1. DEUX collecteurs remplissent `match_kill_events` : celui du film (les deux verites) ET
//	   celui de `highlight_events` (le credit seul) — sans le second, la table couvre 949 matchs
//	   sur 1 325 et le retrait est une perte seche ;
//	2. les 5 lecteurs « cumul » basculent sur `COUNT(*)` depuis `match_kill_events_latest` ;
//	3. les 2 lecteurs « journal » (match-view Q20, penalite de depart LUSR v2) basculent — ils y
//	   gagnent l assistant, la source du degat et les morts de BOT, que l ancienne table ne sait
//	   pas representer (0 ligne `bid(...)` en prod, cote tueur comme victime) ;
//	4. la sonde de presence `internal/sync/events_replay.go` bascule ;
//	5. `internal/ops/seed_demo.go` : `killer_victim_pairs` figure dans la liste des tables COPIEES
//	   dans la base de demonstration. Un remplacement qui oublierait cette ligne livrerait une
//	   demo dont les duels sont vides — panne silencieuse, visible seulement a l ecran. La VUE ne
//	   se copie pas : la demo doit rejouer les migrations (elle le fait deja) ;
//	6. `cmd/rebuild_mp/main.go` : ce reconstructeur CTAS capture puis recree `dependentViews`, qui
//	   contient `v_killer_victim_full`. Le jour ou l on reconstruira une table portant
//	   `match_kill_events_latest`, la vue doit entrer dans cette liste. Outil `//go:build ignore`,
//	   donc AUCUN test ne le protege : ce commentaire est le seul filet ;
//	7. ALORS SEULEMENT `DROP TABLE killer_victim_pairs`, dans une migration dediee.
//
// ─── DEUX ARBITRAGES TRANCHES, A NE PAS REOUVRIR AU MOMENT DE LA BASCULE ──────────────────
//
// (A) LES BOTS RESTENT EXCLUS DES AGREGATS CARRIERE. La nouvelle table sait les representer,
// l ancienne non (0 ligne `bid(...)` en prod). Un compteur carriere qui bougerait le jour d une
// bascule TECHNIQUE serait une mauvaise surprise. Les bots sont DANS la table — c est le journal
// — et ce sont LES LECTEURS CARRIERE qui les filtrent. ⚠ Q26 ne filtre PAS les bots aujourd hui,
// Q27 si : c est le piege de la bascule, et ces filtres `NOT LIKE 'bid(%'` sont aujourd hui des
// NO-OP (la table source n a aucune ligne de bot) donc jamais observes. Les tester AVANT.
//
// (B) LE BTB RESTE INCLUS DANS LES CUMULS, INTERDIT LIGNE PAR LIGNE. C est exactement ce que
// `publishable = FALSE` veut dire : les lignes sont justes EN AGREGAT et fausses INDIVIDUELLEMENT
// (marge de bijection nulle). Un lecteur de cumul ne filtre pas sur `publishable` ; un lecteur qui
// affiche UNE mort nommee — kill feed, duel, timeline — exige `publishable = TRUE`.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_kill_events_v1",
		TargetDB:    TargetShared,
		Description: "Table append-only match_kill_events (1 ligne par mort, credit + source de degat) + vue _latest par passe de decodage — remplacante de killer_victim_pairs, qui reste en place",
		ApplySchema: applyMatchKillEvents,
	})
}

// applyMatchKillEvents cree la table, son index et sa vue. Idempotente (`IF NOT EXISTS` /
// `OR REPLACE`) : la table etant NET-NEUVE, le piege « CREATE TABLE IF NOT EXISTS n ajoute jamais
// une PK a une table existante » ne s applique pas ici.
//
// Relocation : ce step appartient fonctionnellement a Halo Infinite (les films Halo 5 ont un
// autre format) et rejoindra `internal/games/halo_infinite/migrations/` avec le lot de la voie B
// (ADR 0025). Consequence assumee : la
// table est creee VIDE dans le shared des autres titres. Le nom reste dans `canonicalOrder` le
// jour de la relocation.
func applyMatchKillEvents(db *sql.DB) error {
	return EnsureMatchKillEvents(db)
}

// EnsureMatchKillEvents cree la table et sa vue `_latest` si elles manquent. Idempotente.
//
// EXPORTEE depuis le 2026-08-02 pour la meme raison qui avait fait remonter
// `killer_victim_pairs` dans le schema de base : `v_gamertag_lookup` lit desormais
// `match_kill_events_latest`, et DuckDB BIND les vues a leur creation — la table doit donc
// exister AVANT le resolveur d identite, sur une base neuve comme au boot. Les deux appelants
// hors migrations sont `sync.EnsureSharedSchema` et [ApplyResolutionViews].
func EnsureMatchKillEvents(db *sql.DB) error {
	if err := execScript(db, ddlMatchKillEvents); err != nil {
		return err
	}
	return execScript(db, ddlMatchKillEventsLatest)
}

// ddlMatchKillEvents : la table + son unique index.
//
// UN SEUL INDEX, et c est un choix argumente. DuckDB est colonnaire : un index ART ne sert que
// les acces ponctuels et les contraintes ; les agregats (« mes rencontres carriere », « les morts
// par arme ») font un scan de toute facon, que zone-maps et dictionnaires rendent deja rapide sur
// ~250 k lignes. Le seul acces ponctuel reel est « les morts de CE match » (vue match). Chaque
// index en plus coute a l INSERT et elargit la surface ART le jour ou quelqu un ecrirait un
// DELETE. On en pose donc un, celui qui sert la vue `_latest` et la vue match.
const ddlMatchKillEvents = `
	CREATE SEQUENCE IF NOT EXISTS match_kill_events_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_kill_events (
		-- identite technique (append-only : PK non naturelle, ADR 0026)
		id                   BIGINT PRIMARY KEY DEFAULT nextval('match_kill_events_id_seq'),
		match_id             VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe de decodage d UN film. C est l unite de
		-- generation. La vue _latest retient une passe ENTIERE, jamais un melange.
		decode_pass          VARCHAR   NOT NULL,
		-- decoder_rev : version du decodeur qui a produit la passe. Sert a savoir QUELS matchs
		-- redecoder apres un changement de decodeur, au lieu de tout redecoder.
		decoder_rev          VARCHAR   NOT NULL,
		written_at           TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		-- publishable : la passe autorisait-elle la publication LIGNE PAR LIGNE ? FAUX = les
		-- lignes sont justes en AGREGAT et fausses individuellement (marge de bijection nulle,
		-- cas BTB) ou la sante etait en alerte. La PORTEE, ecrite AVEC le resultat.
		publishable          BOOLEAN   NOT NULL,

		time_ms              INTEGER   NOT NULL,

		-- la victime. victim_xuid NULL = BOT (un bot n a pas de XUID) ou nom non resolu par
		-- l appelant. Le film ne rend que des NOMS : la resolution en xuid est la
		-- responsabilite du collecteur, et son echec est une donnee, pas une erreur.
		victim_gamertag      VARCHAR   NOT NULL,
		victim_xuid          VARCHAR,

		-- ── VERITE 1 : LE CREDIT (kill-feed) ────────────────────────────────────────────
		feed_killer_gamertag VARCHAR,
		feed_killer_xuid     VARCHAR,
		-- feed_present : le kill-feed porte-t-il cette MORT ? FAUX pour une victime BOT — le
		-- kill-feed du film est humain-seul. ⚠ feed_killer_gamertag reste RENSEIGNE dans ce
		-- cas : le KILL y est, c est la MORT qui n y est pas. Cas symetrique (read_origin
		-- 'tueur-bot') : le tueur est un bot, donc feed_killer_xuid est NULL avec un
		-- feed_killer_gamertag renseigne — un bot n a pas de XUID.
		feed_present         BOOLEAN   NOT NULL,
		-- assistant : attribut de la verite KILL-FEED, pas une troisieme verite.
		assist_gamertag      VARCHAR,
		assist_xuid          VARCHAR,
		-- assist_known : FAUX = ON NE SAIT PAS. Jamais « pas d assistant » (cf. en-tete).
		assist_known         BOOLEAN   NOT NULL,
		-- assist_index : indice de replication BRUT. Conserve parce que le nom de l assistant,
		-- lui, passe par une bijection propre a la passe, et l indice est la quantite a citer si
		-- un nom surprend. (Le tueur, lui, vient du kill-feed avec son XUID : pas de bijection.)
		assist_index         SMALLINT,
		assist_rejected      VARCHAR,
		-- assist_extra_count : LE GARDE-FOU de l hypothese « un seul assistant ». Mesure du
		-- 2026-08-03 : 5 lignes a 1 sur 124 694, aucune au-dela. Declencheur de migration vers
		-- une table fille : une ligne a >= 2, OU la part des lignes a >= 1 au-dela de 0,1 %
		-- (cf. en-tete du fichier).
		assist_extra_count   UTINYINT  NOT NULL DEFAULT 0,

		-- ── VERITE 2 : LA SOURCE DU DEGAT FATAL ─────────────────────────────────────────
		-- source_tag : identifiant jpt! 32 bits. LA quantite qui ne depend d aucune table de
		-- nommage. Le nom se resout A LA LECTURE — jamais stocke. NULLABLE : NULL = SOURCE NON
		-- MESUREE (film absent/expire, ou producteur kill-feed). Ce n est PAS une tolerance,
		-- c est la doctrine appliquee au schema.
		source_tag           UINTEGER,
		-- source_category : identifiant moteur du modificateur de degat (None, Headshot,
		-- SilentMelee, ...). Enum GELEE par le format de film, donc stockable sans risque de
		-- peremption. ⚠ elle ne distingue PAS grenade et melee ordinaire (les deux sortent
		-- None) : c est source_tag qui les porte. Voyage AVEC source_tag : les deux NULL ou les
		-- deux renseignes (tenu par le persister).
		source_category      VARCHAR,

		-- parts de degats, en pourcentage entier. NULL = non mesure.
		-- ⚠ assist_damage_pct DOIT etre NULL quand il n y a pas d assistant nomme : sans
		-- assistant le champ porte une constante par film qui ne veut rien dire (elle vaut 20
		-- sur certains films). Contrainte tenue par le persister.
		-- ⚠ CES COLONNES NE SONT PAS BORNEES A 100. 1,7 % des kill-events attaches a de vraies
		-- morts nommees portent une valeur superieure, jusqu a 228. Un plafond jetterait de la
		-- donnee reelle pour proteger une interpretation.
		killer_damage_pct    UTINYINT,
		assist_damage_pct    UTINYINT,

		-- diverges : la source appartient a la VICTIME alors que le feed credite un autre
		-- joueur. LES DEUX SONT VRAIS. NULLABLE, et pour une raison de doctrine : sans source
		-- mesuree la divergence est INDEFINISSABLE. Ecrire FALSE la ferait passer pour
		-- « mesure : pas de divergence », c est-a-dire une mesure d absence fabriquee a partir
		-- d une absence de mesure. NULL = non mesurable.
		diverges             BOOLEAN,

		-- portee de la lecture : 'marche' (98,2 % de couples exacts) ou 'scan' (78,4 %).
		-- origine, QUATRE valeurs cote decodeur de film : 'credit-concordant' | 'source-victime'
		-- | 'bot' (la VICTIME est un bot) | 'tueur-bot' (le TUEUR est un bot). Le producteur
		-- credit-seul, lui, ecrit 'credit-seul'. A PONDERER par le lecteur.
		read_path            VARCHAR   NOT NULL,
		read_origin          VARCHAR   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_kill_events_match
		ON match_kill_events(match_id, written_at);
`

// ddlMatchKillEventsLatest : LE SEUL CHEMIN DE LECTURE AUTORISE (ADR 0026 — une lecture brute
// sert des lignes perimees, ici les lignes d une passe de decodage precedente).
//
// Deux choses s y jouent :
//
//  1. la selection de LA DERNIERE PASSE par match (jamais un melange de passes) ;
//  2. `damage_pct_residual` — un RESTE ARITHMETIQUE, et rien de plus.
//
// ─── LE RESIDU : CE QU IL EST, CE QU IL N EST PAS ─────────────────────────────────────────
//
// Il a d abord porte le nom `unattributed_damage_pct` — « part de degats que personne ne se voit
// crediter ». CE NOM A ETE RETIRE : aucune mesure ne porte cette interpretation. Nommer le reste
// « non attribue » lui preterait un sens (« quelqu un a inflige ces degats sans etre credite »)
// qui ajouterait une interpretation A une lecture qui en est deja une.
//
// LE NOM RETENU NE PROMET DONC RIEN : `damage_pct_residual` = total empirique applicable moins
// les parts lues. IL EST GARDE, malgre la reserve, pour une raison operationnelle : la formule
// depend du cas et le CHOIX DU TOTAL est le piege (100 sans assistant, 99 avec). Chaque lecteur
// qui la recopierait aurait sa propre chance de se tromper ; ici il y en a une seule copie,
// testee. Et c est la quantite par laquelle se manifesterait un troisieme contributeur de degats.
//
// CE QUE LE CALCUL IGNORE, ASSUME :
//   - les totaux 99/100 sont EMPIRIQUES (99 sur 22 367 des 31 204 kills assistes de 892 films),
//     pas une regle du format. Ils n expliquent pas le reste, ils le cadrent ;
//   - il PEUT ETRE NEGATIF, et il n est pas borne : 1,7 % des kill-events portent une part > 100
//     (jusqu a 228). Un CASE qui ramenerait ces valeurs a 0 cacherait exactement la population
//     qui contredit l interpretation. On la laisse sortir ;
//   - il ne dit rien de QUI a inflige le reste, ni meme qu il y ait eu un « qui ».
//
// LES QUATRE CAS OU IL VAUT NULL, ET POURQUOI (« NULL n est jamais zero ») :
//
//	killer_damage_pct IS NULL   rien de mesure du cote tueur.
//	NOT assist_known            on ne sait pas s il y avait un assistant, donc on ne sait pas
//	                            QUEL TOTAL appliquer (99 ou 100) : le residu n est pas calculable.
//	assistant nomme, part NULL  un terme de la soustraction manque. C EST LE DEFAUT QUE LA
//	                            PREMIERE VERSION AVAIT : elle branchait sur « la part d assistant
//	                            est absente » au lieu de « il n y a pas d assistant », donc un
//	                            assistant NOMME dont la part n a pas ete lue tombait dans le cas
//	                            solo (total 100) au lieu de 99 — residu faux de 1, silencieux.
//	                            COALESCE(assist_damage_pct, 0) serait la meme faute : traiter une
//	                            absence de mesure comme un zero mesure.
const ddlMatchKillEventsLatest = `
	CREATE OR REPLACE VIEW match_kill_events_latest AS
	SELECT
		e.*,
		CASE
			WHEN e.killer_damage_pct IS NULL THEN NULL
			WHEN NOT e.assist_known THEN NULL
			WHEN e.assist_gamertag IS NULL
				THEN 100 - CAST(e.killer_damage_pct AS SMALLINT)
			WHEN e.assist_damage_pct IS NULL THEN NULL
			ELSE 99 - CAST(e.killer_damage_pct AS SMALLINT)
			        - CAST(e.assist_damage_pct AS SMALLINT)
		END AS damage_pct_residual
	FROM match_kill_events AS e
	QUALIFY e.decode_pass = FIRST_VALUE(e.decode_pass) OVER (
		PARTITION BY e.match_id ORDER BY e.written_at DESC, e.id DESC
	);
`
