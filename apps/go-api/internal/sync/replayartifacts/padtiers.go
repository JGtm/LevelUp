package replayartifacts

// padtiers.go — LES PRISES DE SOCLE VENTILEES PAR NIVEAU D'ARME, PERSISTEES AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Il LIT les socles de l'artefact range (`weaponPads`, `padPickups`, `loadouts`), les croise a
// la REFERENCE DES EMPLACEMENTS de la carte (`map_weapon_pads.json`, la MEME jointure que
// `replay.BuildMapWeaponPads` — un socle du match confirme un emplacement a moins d'un metre),
// applique la regle des niveaux (`analysis/weapontier`) et transporte le resultat vers
// `persist.PadTiersPersister`.
//
// # POURQUOI CETTE PASSE EXISTE, ALORS QUE LA VUE MATCH SAIT DEJA LE FAIRE
//
// La vue match resout le niveau A LA REQUETE : elle a l'artefact entier sous la main. Les
// pages d'AGREGAT (Sessions, Escouade, Timeseries) ne l'ont pas — elles lisent
// `match_usage_players_latest`, ou les prises sont deja comptees PAR ARME et ou l'identite du
// socle a disparu. Le niveau ne s'en deduit pas : il lui faut la POSITION du socle. D'ou cette
// projection, au grain (match, joueur, niveau, arme).
//
// # ELLE NE RE-CUIT RIEN, ET C'EST LA RAISON DU MOTIF
//
// Meme doctrine que `flaggrabsnet.go` : la source est l'artefact DEJA RANGE. Aucune recuisson
// n'est necessaire — et aucune n'est declenchee : cuire des artefacts en lot est INTERDIT
// (bombe RAM, verrou `filmproc.AcquireSolo`).
//
// # LA FORME, ET LA SEULE DIFFERENCE AVEC LES PRISES NETTES
//
//	SUR DISQUE, PAS LE BLOB   le document est deja lu et deserialise par [Deriver].
//	VIA `internal/persist`    INSERT-only (ADR 0019/0026/0030).
//	SEGMENT WRITER COURT      acquis APRES toute cuisson, relache aussitot.
//
// LA DIFFERENCE : la projection a besoin de DEUX CHOSES QUI VIVENT EN BASE — le `map_id` du
// match (quelle carte croiser) et son `pair_name` (le mode est-il a departs aleatoires). Les
// prises nettes, elles, se projettent du seul document. Le segment writer sert donc AUSSI a
// lire ces deux colonnes de `match_registry`, juste avant d'ecrire. Le segment reste COURT :
// une requete indexee sur quelques dizaines d'identifiants, puis des projections qui sont du
// calcul pur sur des documents deja en memoire. Aucun decodage n'a lieu sous le writer.
//
// # DEUX PORTES, ET ELLES NE PESENT PAS PAREIL
//
//	`film.weapon_tiers`                        « ce titre sait lire ces socles » — sans elle,
//	                                           RIEN n'est produit ;
//	`[weapon_tiers].random_start_mode_tokens` « voici mes modes sans arme de base » — son
//	                                           absence n'eteint rien, elle fait seulement
//	                                           qu'aucun mode n'est tenu pour aleatoire.
//
// # LES SILENCES ORDINAIRES, ET ILS NE SE COMPTENT PAS EN DEFAUTS
//
//	artefact anterieur au schema 30   `padPickups[].xuid` n'existe pas : aucune prise n'est
//	                                  nommee, et une passe de zeros AFFIRMERAIT que personne
//	                                  n'a rien pris. DEBUG, rien d'ecrit ;
//	film sans aucun humain au roster  rien a attribuer, DEBUG ;
//	carte hors reference              la passe est ECRITE quand meme, `pads_confirmed = 0` :
//	                                  « des socles, mais pas de niveaux » est une mesure, et
//	                                  c'est elle qui fait dire a l'ecran « niveaux non
//	                                  etablis » plutot que de tout ranger sous « non classe ».

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/weapontier"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// PadTiersMinSchema est le schema d'artefact minimal exploitable : `padPickups[].xuid` — le
// ramasseur NOMME — arrive au schema 30. En deca, aucune prise n'est attribuable, et une passe
// de lignes a zero affirmerait « personne n'a pris de socle » la ou la verite est « le film ne
// le disait pas encore ».
const PadTiersMinSchema = 30

// passeNiveauxPrete : une projection reussie, prete a persister.
type passeNiveauxPrete struct {
	matchID string
	batch   persist.PadTiersBatch
}

// IdentiteMatchNiveaux : ce que la BASE apporte a la projection d'UN match.
type IdentiteMatchNiveaux struct {
	// MapID : la carte du match (asset UGC). Vide = carte inconnue, la passe s'ecrit avec
	// `pads_confirmed = 0`.
	MapID string
	// PairName : le mode joue, BRUT (« Super Fiesta:Slayer »). C'est le titre qui dit, par son
	// `regulation.toml`, lequel de ces prefixes est a departs aleatoires.
	PairName string
}

// ReferenceEmplacements est la reference des emplacements de socle d'un titre, chargee UNE
// FOIS par passe. Un alias nomme plutot que le type nu : le catalogue est le SEUL etat partage
// entre les projections d'un lot, et il doit se lire comme tel.
type ReferenceEmplacements = replay.MapWeaponPadsCatalog

// ChargerReferenceEmplacements lit la reference versionnee des emplacements du titre, overlay
// des cartes rattrapees compris (`LoadMapWeaponPadsMerged`).
//
// EXPORTEE parce que le backfill (`cmd/levelup`) en a besoin AUSSI, et qu'une seconde lecture
// du meme fichier ailleurs ferait deux verites de la meme reference.
func ChargerReferenceEmplacements(repoRoot, titleSlug string) (*ReferenceEmplacements, error) {
	res := titlePkg.NewPathResolver(repoRoot)
	return replay.LoadMapWeaponPadsMerged(
		res.MapWeaponPadsPath(titleSlug), res.MapWeaponPadsOverlayPath(titleSlug))
}

// ProjeterNiveauxDArmes tire d'UN document range la passe a ecrire.
//
// Rend une passe VIDE (MatchID vide) quand il n'y a RIEN a ecrire : artefact trop ancien pour
// porter un ramasseur nomme, ou film sans aucun humain au roster. Ni l'un ni l'autre n'est un
// defaut.
//
// EXPORTEE pour la meme raison que [ChargerReferenceEmplacements] : le backfill rejoue
// EXACTEMENT cette projection sur les artefacts deja ranges. Deux projections de la meme regle
// divergeraient au premier changement.
func ProjeterNiveauxDArmes(
	matchID string, doc *replay.ReplayDocument,
	ref *ReferenceEmplacements, id IdentiteMatchNiveaux, randomStarts bool,
) persist.PadTiersBatch {
	if doc == nil || doc.SchemaVersion < PadTiersMinSchema {
		return persist.PadTiersBatch{}
	}
	humains := humainsDuRoster(doc)
	if len(humains) == 0 {
		return persist.PadTiersBatch{}
	}
	// LA MEME JOINTURE QUE LA VUE MATCH, littéralement : un socle du match confirme un
	// emplacement de la carte a moins d'un metre, le plus proche l'emporte et est ensuite pris.
	var cross *replay.MapWeaponPads
	if ref != nil && id.MapID != "" {
		if entry, err := ref.Lookup(id.MapID); err == nil {
			cross = replay.BuildMapWeaponPads(entry, doc.WeaponPads)
		}
	}
	// LA TRADUCTION VERS LE PAQUET PUR SE FAIT ICI, et c'est la frontiere d'ADR 0012 :
	// `internal/analysis/weapontier` ne connait aucun format de film, c'est la couche du titre
	// qui lui donne ses quatre listes elementaires.
	m := weapontier.NewMatch(soclesPour(doc.WeaponPads), emplacementsPour(cross),
		departsPour(doc.Loadouts), prisesPour(doc), randomStarts)
	out := persist.PadTiersBatch{
		MatchID:      matchID,
		PadsTotal:    len(doc.WeaponPads),
		RandomStarts: randomStarts,
	}
	if cross != nil {
		out.PadsConfirmed = len(cross.Pads)
	}
	out.Rows = lignesDeNiveaux(doc, m, humains)
	return out
}

// humainsDuRoster rend les xuid des joueurs HUMAINS du film, tries.
//
// LES BOTS N'EN SONT PAS : un bot n'a pas de xuid (`RosterEntry.Bot`, XUID vide), et la table
// est clef par xuid. Leur absence n'est pas un zero, c'est une identite que la base ne porte
// pas — meme regle que `match_flag_grabs_net`.
func humainsDuRoster(doc *replay.ReplayDocument) []string {
	vus := make(map[string]bool, len(doc.Roster))
	out := make([]string, 0, len(doc.Roster))
	for _, r := range doc.Roster {
		if r.XUID == "" || r.Bot || vus[r.XUID] {
			continue
		}
		vus[r.XUID] = true
		out = append(out, r.XUID)
	}
	sort.Strings(out)
	return out
}

// lignesDeNiveaux ventile les prises NOMMEES par (joueur, niveau, famille d'arme), puis
// complete le roster a zero.
//
// LE ZERO EST UNE MESURE, PAS UN REMPLISSAGE : sur un match dont les socles ont ete LUS, « ce
// joueur n'a pris aucun socle » est un fait ; sans sa ligne, il serait indistinguable d'un
// joueur d'un match sans film — les deux se liraient « non mesure ».
func lignesDeNiveaux(doc *replay.ReplayDocument, m weapontier.Match, humains []string) []persist.PadTierRow {
	type cle struct{ xuid, tier, famille string }
	compte := map[cle]int{}
	connus := make(map[string]bool, len(humains))
	for _, x := range humains {
		connus[x] = true
	}
	for i := range doc.PadPickups {
		p := &doc.PadPickups[i]
		if p.XUID == nil || *p.XUID == "" || !connus[*p.XUID] {
			continue
		}
		if p.Pad < 0 || p.Pad >= len(doc.WeaponPads) {
			continue
		}
		arme := doc.WeaponPads[p.Pad].Weapon
		// LA CLE NORMALISEE, JAMAIS LA FORME VERBATIM : deux artefacts peuvent ecrire la meme
		// famille sous deux conventions, et des cles verbatim la couperaient en deux (meme
		// regle que `pad_pickups_json`). Une famille que la normalisation refuse — un socle de
		// BONUS, dont l'identite est un nom canonique — garde sa forme telle quelle.
		famille := arme
		if k, ok := replay.PadWeaponFamilyKey(arme); ok {
			famille = k
		}
		compte[cle{*p.XUID, string(m.TierOf(p.Pad, arme)), famille}]++
	}
	rows := make([]persist.PadTierRow, 0, len(compte)+len(humains))
	servis := make(map[string]bool, len(humains))
	for c, n := range compte {
		rows = append(rows, persist.PadTierRow{
			XUID: c.xuid, Tier: niveauEnBase(c.tier), WeaponFamily: c.famille, Pickups: n,
		})
		servis[c.xuid] = true
	}
	for _, x := range humains {
		if !servis[x] {
			rows = append(rows, persist.PadTierRow{XUID: x, Tier: persist.PadTierNoPickup})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].XUID != rows[j].XUID {
			return rows[i].XUID < rows[j].XUID
		}
		if rows[i].Tier != rows[j].Tier {
			return rows[i].Tier < rows[j].Tier
		}
		return rows[i].WeaponFamily < rows[j].WeaponFamily
	})
	return rows
}

// niveauEnBase traduit le vocabulaire du paquet d'analyse vers celui de la TABLE.
//
// LES DEUX VOCABULAIRES SONT DISTINCTS A DESSEIN : `weapontier.Tier` est une valeur de calcul,
// les constantes `persist.PadTier*` sont un CONTRAT DE DONNEES (elles sont ecrites sur disque
// et relues par le backfill des mois plus tard). Les confondre ferait qu'un renommage interne
// du paquet d'analyse changerait silencieusement le contenu de la base.
func niveauEnBase(t string) string {
	switch weapontier.Tier(t) {
	case weapontier.TierBase:
		return persist.PadTierBase
	case weapontier.TierGround:
		return persist.PadTierGround
	case weapontier.TierPower:
		return persist.PadTierPower
	case weapontier.TierPowerup:
		return persist.PadTierPowerup
	default:
		return persist.PadTierUnclassified
	}
}

// capabiliteNiveauxArmee dit si le titre declare `film.weapon_tiers`, et DIT POURQUOI quand la
// reponse est non — meme contrat que `capabilitePrisesNettesArmee`.
func capabiliteNiveauxArmee(ctx context.Context, d Deps) (armee, incident bool) {
	return porteCapability(ctx, d, games.CapFilmWeaponTiers, "niveaux d'armes", nil)
}

// ReglesDepartsAleatoires lit les CATEGORIES de mode a departs aleatoires declarees par le
// titre. Liste vide = le titre n en declare aucune, ce qui n eteint rien.
//
// EXPORTEE pour le backfill, meme raison que les deux fonctions ci-dessus.
func ReglesDepartsAleatoires(repoRoot, titleSlug string) (*mappings.RegulationSet, error) {
	return mappings.LoadRegulationFromFile(filepath.Join(
		titlePkg.NewPathResolver(repoRoot).TitleMappingsDir(titleSlug), "regulation.toml"))
}

// DepartsAleatoires dit si le mode d un match distribue des equipements de debut de vie TIRES
// AU SORT.
//
// LA REGLE VIENT DU TITRE, et elle est cherchee dans le `pair_name` ENTIER — jamais sur son
// prefixe, jamais sur la categorie resolue. Mesure du 2026-09-14 : « Slayer:Arena Super
// Fiesta » a pour prefixe « Slayer » et pour categorie « Other », « BTB:Fiesta CTF » a pour
// categorie « BTB » ; aucune des deux lectures ne les reconnait, et ce sont les formes les plus
// nombreuses du registre. Detail et mesure : `mappings.RegulationSet.HasRandomStarts`.
//
// EXPORTEE : le backfill, le fil de l eau et le service de rejeu doivent tous les trois lire la
// MEME regle. Une seconde resolution ailleurs divergerait au premier format nouveau.
func DepartsAleatoires(reg *mappings.RegulationSet, pairName string) bool {
	return reg.HasRandomStarts(pairName)
}

// persisterNiveauxDArmes projette puis ecrit les niveaux d'armes des artefacts ranges du lot.
// Best-effort de bout en bout : aucun echec ne remonte au cycle, aucun ne se tait.
//
// ─── CETTE FAMILLE S ABSTIENT SEULE, ELLE NE PRIVE JAMAIS LES AUTRES DE LEUR MARQUE ───────
//
// Correctif de revue du 2026-09-14. La version precedente appelait b.echecLot(lus) des que SA
// reference de cartes ou SON regulation.toml etait illisible : le lot entier passait alors pour
// non derive, et les QUATRE autres familles — deja ECRITES — n etaient jamais marquees. Le
// rattrapage rejouait le meme lot indefiniment, et la fixture d integration
// TestRun_SelectionDeCuissonVide_RattrapeQuandMeme le prouvait en rouge.
//
// La regle est desormais celle de toute famille a reference : elle se tait, elle compte son
// echec, et elle laisse le lot suivre son cours. Une reference manquante est une installation
// incomplete de CETTE famille, jamais un defaut des autres.
func persisterNiveauxDArmes(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) {
	if len(lus) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	armee, incident := capabiliteNiveauxArmee(ctx, d)
	if !armee {
		if incident {
			observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(len(lus)))
		}
		return
	}
	ref, reg, ok := referencesDuTitre(ctx, d, titre, len(lus))
	if !ok {
		return
	}
	// L IDENTITE DES MATCHS SE LIT AVANT LE WRITER, par un segment de LECTURE (correctif de
	// revue) : la projection doit se faire hors du lease d ecriture, comme l en-tete de ce
	// fichier le promet et comme flaggrabsnet.go le fait.
	identites, ok := identitesDuLot(ctx, d, lus)
	if !ok {
		// IDENTITE ILLISIBLE = LOT NON PROJETE. Projeter avec un pair_name vide ferait rendre
		// "departs non aleatoires" a tous les matchs, donc ecrire un niveau "base" sur des
		// Fiesta. Le cycle suivant reessaiera ; ces matchs gardent la marque des autres
		// familles et n auront simplement pas encore de niveaux.
		slog.WarnContext(ctx, "post-sync: niveaux d'armes — identites de match illisibles, "+
			"lot NON projete (un pair_name vide ecrirait un niveau de base sur des Fiesta)",
			"titleSlug", d.TitleSlug, "matchs", len(lus))
		observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(len(lus)))
		return
	}
	prets := projeterNiveauxDuLot(ctx, lus, ref, reg, identites)
	if len(prets) == 0 {
		return
	}
	ecrireNiveauxDArmes(ctx, d, b, titre, prets)
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

// identitesDuLot lit map_id et pair_name du lot par un SEGMENT DE LECTURE court.
//
// Rend (nil, false) quand la lecture est impossible OU incomplete : un match dont le registre
// ne rend pas la ligne n a pas d identite, et le projeter sans elle reviendrait a lui preter un
// mode regulier. Sans segment de lecture cable, meme verdict — on ne devine pas.
func identitesDuLot(ctx context.Context, d Deps, lus []artefactLu) (map[string]IdentiteMatchNiveaux, bool) {
	if d.WithRead == nil {
		return nil, false
	}
	ids := matchIDsDuLot(lus)
	var out map[string]IdentiteMatchNiveaux
	var lecture error
	d.WithRead(ctx, "niveaux d'armes", func(sharedDB *sql.DB) {
		out, lecture = IdentitesDesMatchs(ctx, sharedDB, ids)
	})
	if lecture != nil || out == nil {
		return nil, false
	}
	for _, id := range ids {
		if _, connu := out[id]; !connu {
			return nil, false
		}
	}
	return out, true
}

// ecrireNiveauxDArmes acquiert le writer et ecrit les passes DEJA projetees.
//
// LE SEGMENT NE CONTIENT QUE DES ECRITURES : la lecture des identites et la projection ont eu
// lieu avant (cf. persisterNiveauxDArmes) — c est le motif de flaggrabsnet.go, et l en-tete de
// ce fichier le promet.
func ecrireNiveauxDArmes(
	ctx context.Context, d Deps, b *bilanDerivations, titre string, prets []passeNiveauxPrete,
) {
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: niveaux d'armes NON persistes (aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(len(prets)))
		echecNiveauxDArmes(b, prets)
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, niveaux d'armes non persistes",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(len(prets)))
		echecNiveauxDArmes(b, prets)
		return
	}
	defer release()
	p := persist.NewPadTiersPersister(db)
	ecrits, echecs := 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: ecriture des niveaux d'armes echouee",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			b.echec(prets[i].matchID)
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurNiveauxArmesEcrits, int64(ecrits))
	observability.AddIntT(titre, CompteurNiveauxArmesEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: niveaux d'armes persistes",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}

// projeterNiveauxDuLot projette tous les documents du lot. Rend les passes NON VIDES.
func projeterNiveauxDuLot(
	ctx context.Context, lus []artefactLu, ref *ReferenceEmplacements,
	reg *mappings.RegulationSet, identites map[string]IdentiteMatchNiveaux,
) []passeNiveauxPrete {
	prets := make([]passeNiveauxPrete, 0, len(lus))
	for _, a := range lus {
		id := identites[a.matchID]
		batch := ProjeterNiveauxDArmes(a.matchID, a.doc, ref, id, DepartsAleatoires(reg, id.PairName))
		if batch.MatchID == "" {
			slog.DebugContext(ctx, "post-sync: niveaux d'armes — artefact sans prise nommable",
				"match_id", a.matchID, "schema", a.doc.SchemaVersion)
			continue
		}
		prets = append(prets, passeNiveauxPrete{matchID: a.matchID, batch: batch})
	}
	return prets
}

// echecNiveauxDArmes enregistre au bilan que ces passes n ont pas ete persistees faute de
// writer : sans cette trace, la marque de derivation se poserait sur un match dont RIEN n a ete
// ecrit.
func echecNiveauxDArmes(b *bilanDerivations, prets []passeNiveauxPrete) {
	ids := make([]string, 0, len(prets))
	for i := range prets {
		ids = append(ids, prets[i].matchID)
	}
	echecFauteDeWriter(b, ids)
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

// soclesPour traduit les socles du film vers le vocabulaire du paquet de niveaux. L INDEX EST
// L IDENTITE : la tranche garde l ordre de `weaponPads`, celui que `padPickups[].pad` cite.
func soclesPour(pads []replay.WeaponPad) []weapontier.Pad {
	out := make([]weapontier.Pad, len(pads))
	for i := range pads {
		out[i] = weapontier.Pad{Weapon: pads[i].Weapon}
	}
	return out
}

// emplacementsPour traduit le calque des emplacements CONFIRMES. Nil (carte hors reference) rend
// une tranche vide : aucun socle n est confirme, tout sera « non classe ».
func emplacementsPour(cross *replay.MapWeaponPads) []weapontier.Spot {
	if cross == nil {
		return nil
	}
	out := make([]weapontier.Spot, 0, len(cross.Pads))
	for _, s := range cross.Pads {
		out = append(out, weapontier.Spot{Pad: s.Pad, Family: s.Family})
	}
	return out
}

// departsPour traduit le canal `loadouts` DANS SON ORDRE — le paquet de niveaux ne retient que
// la premiere emission de chaque slot, et c est cet ordre qui le lui permet.
func departsPour(loadouts []replay.Loadout) []weapontier.Spawn {
	out := make([]weapontier.Spawn, len(loadouts))
	for i := range loadouts {
		out[i] = weapontier.Spawn{Slot: loadouts[i].Slot, T: loadouts[i].T, Weapons: loadouts[i].W}
	}
	return out
}
