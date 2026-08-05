package main

// sortie_json.go — LA MEME CHOSE, EN JSON.
//
// POURQUOI UN SCHEMA DEFINI ICI ET PAS DES BALISES `json` DANS LE PAQUET : le paquet publie des
// TYPES GO, pas un format d echange. Lui coller des balises figerait un contrat de serialisation
// que personne n a demande, et melangerait la couche d affichage a la couche de decodage. Le jour
// ou une API expose ces donnees, elle definira SON schema — comme celui-ci le fait.
//
// LE SCHEMA REPRODUIT LA DOCTRINE : `credit` et `source` sont deux objets freres, `divergence` est
// un booleen a part, et les denominateurs sont des CHAMPS NOMMES, jamais des taux precalcules. Un
// consommateur qui veut un pourcentage le calcule ; il ne peut pas se tromper de denominateur
// puisque chacun porte son nom.

import (
	"encoding/json"
	"os"

	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// sortieJSON : la racine du document.
type sortieJSON struct {
	Film        string          `json:"film"`
	Calibration string          `json:"calibration"`
	Catalogue   catalogueJSON   `json:"catalogue"`
	Couverture  couvertureJSON  `json:"couverture"`
	Sante       santeJSON       `json:"sante"`
	Publication publicationJSON `json:"publication"`
	Morts       []mortJSON      `json:"morts"`
}

type catalogueJSON struct {
	Identifiants   int    `json:"identifiants"`
	GenereLe       string `json:"genere_le"`
	EtiquettesDu   string `json:"etiquettes_du"`
	FichiersDuJeu  bool   `json:"lit_des_fichiers_du_jeu"`
	NombreLibelles int    `json:"nombre_libelles"`
}

// couvertureJSON : LES DENOMINATEURS, chacun sous son nom. Aucun ratio n est serialise.
type couvertureJSON struct {
	Couvertes             int    `json:"morts_couvertes"`
	CouplesReels          int    `json:"denominateur_couples_reels"`
	CouplesReconstruits   int    `json:"denominateur_couples_reconstruits"`
	MortsDuKillFeed       int    `json:"denominateur_morts_du_kill_feed"`
	CouplesFabriques      int    `json:"couples_fabriques_retires"`
	MortsDeBot            int    `json:"morts_de_bot_population_neuve"`
	MortsParUnBot         int    `json:"morts_infligees_par_un_bot_population_neuve"`
	MortsDeLAPI           *int   `json:"denominateur_morts_de_l_api"`
	NoteMortsDeLAPI       string `json:"note_morts_de_l_api"`
	CouplesMemeInstant    int    `json:"couples_au_meme_instant"`
	KillsDuKillFeed       int    `json:"kills_du_kill_feed"`
	NoteCouplesFabriques  string `json:"note_couples_fabriques"`
	NoteMortsDeBotNeuvete string `json:"note_morts_de_bot"`
	NoteMortsParUnBot     string `json:"note_morts_infligees_par_un_bot"`
}

type santeJSON struct {
	Verdict            string              `json:"verdict"`
	Alertes            []string            `json:"alertes"`
	TauxInexpliques    float64             `json:"taux_inexpliques"`
	TauxCouverture     float64             `json:"taux_couverture"`
	Compteurs          []compteurJSON      `json:"compteurs_expvar"`
	GateParVoie        map[string]gateJSON `json:"gate_par_voie"`
	Concordance        concordanceJSON     `json:"concordance"`
	SondeCatalogue     *sondeJSON          `json:"sonde_catalogue_relachee"`
	NoteSondeCatalogue string              `json:"note_sonde_catalogue"`
}

type compteurJSON struct {
	Nom     string `json:"nom"`
	Valeur  int64  `json:"valeur"`
	Alerte  bool   `json:"peut_declencher_une_alerte"`
	Comment string `json:"remarque,omitempty"`
}

type gateJSON struct {
	Population int     `json:"population"`
	Apparies   int     `json:"apparies_au_couple_exact"`
	Publiees   int     `json:"lignes_publiees"`
	Taux       float64 `json:"taux"`
}

type concordanceJSON struct {
	Redondants          int    `json:"enregistrements_lus_par_les_deux_voies"`
	Accord              int    `json:"accord"`
	Desaccord           int    `json:"desaccord"`
	MortsPlusieursCands int    `json:"morts_a_plusieurs_candidats"`
	Note                string `json:"note"`
}

type sondeJSON struct {
	Candidats      int      `json:"candidats"`
	HorsCatalogue  int      `json:"hors_catalogue"`
	Apparies       int      `json:"apparies"`
	NonCouvertes   int      `json:"morts_non_couvertes"`
	TagsConcernees []string `json:"tags"`
}

type publicationJSON struct {
	LigneParLigneAutorisee bool   `json:"ligne_par_ligne_autorisee"`
	MargeDeBijection       int    `json:"marge_de_bijection"`
	Motif                  string `json:"motif,omitempty"`
}

// mortJSON : UNE MORT, DEUX VERITES FRERES. Ni l une ni l autre n est imbriquee dans l autre.
//
// `assistant` et `parts_de_degats` sont des ATTRIBUTS DE LA VERITE << CREDIT >>, pas une troisieme
// verite : ils decrivent QUI a aide et DANS QUELLE PROPORTION, c est-a-dire la meme question que le
// credit, en plus fin.
type mortJSON struct {
	TempsMS       int         `json:"temps_ms"`
	Temps         string      `json:"temps"`
	Victime       string      `json:"victime"`
	Credit        creditJSON  `json:"credit_du_jeu"`
	Assistant     assistJSON  `json:"assistant"`
	PartsDeDegats partsJSON   `json:"parts_de_degats"`
	Source        sourceJSON  `json:"source_du_degat_fatal"`
	Divergence    bool        `json:"divergence"`
	Lecture       lectureJSON `json:"lecture"`
}

type creditJSON struct {
	Joueur          string `json:"joueur"`
	PresentAuFeed   bool   `json:"present_au_kill_feed"`
	NoteAbsenceFeed string `json:"note,omitempty"`
}

// assistJSON : L ASSISTANT, ET SES TROIS ETATS — jamais deux.
//
// `connu` FAUX ne veut PAS dire << pas d assistant >> : il veut dire QU ON NE SAIT PAS, parce
// qu aucun kill-event n a ete attache a cette mort. Confondre les deux publierait des faits jamais
// observes, et c est la raison d etre de ce champ. Les trois etats se lisent ainsi :
//
//	connu = false                 -> ON NE SAIT PAS
//	connu = true, joueur = ""     -> PAS D ASSISTANT, et c est une MESURE
//	connu = true, joueur = "X"    -> X a assiste
//
// `indice_replication` est la quantite qui ne depend d AUCUNE bijection : a citer quand un nom
// surprend. `assistants_en_surplus` est le garde-fou de l hypothese << un seul assistant >> ; il
// vaut zero sur toute la serie de reference, et le jour ou il bouge, il faut une table fille.
type assistJSON struct {
	Connu             bool   `json:"connu"`
	Joueur            string `json:"joueur,omitempty"`
	IndiceReplication int    `json:"indice_replication"`
	Refus             string `json:"refus,omitempty"`
	Surplus           int    `json:"assistants_en_surplus"`
	Note              string `json:"note"`
}

// partsJSON : LES DEUX PARTS DE DEGATS, en pourcentage entier.
//
// C est une information que LE JEU N AFFICHE NULLE PART — ni en partie, ni au Theater. Elle repose
// sur quatre jambes convergentes : l executable nomme les deux quantites cote a cote, leur somme
// vaut 99 avec un assistant, le premier vaut 100 sans, et une capture memoire independante tombe
// sur la meme constante arbitraire.
//
// LA RESERVE << CONFRONTABLE PAR AUCUNE VERITE TERRAIN >> A ETE ECRITE ICI PUIS REFUTEE LE MEME
// JOUR (2026-07-31), par l utilisateur. Elle confondait deux choses :
//   - LA VALEUR EXACTE reste inconfrontable — personne ne peut lire << 61 >> a l ecran ;
//   - MAIS L ORDRE ET L ECART SE CONFRONTENT PAR LE DEROULE DU COMBAT, qui est observable.
//
// LE CAS QUI L A ETABLI, film `78919882` a 10:38 : le decodeur publie JGtm tue par Senor Pepper au
// S7 Sniper, assiste par Only1kenobi1701, parts **1/61**. Releve au Theater : *<< il y a eu un duel
// au corps a corps entre JGtm et Only1kenobi1701, et Senor Pepper a fini JGtm d un tir de sniper
// dans la tete >>*. L assistant qui fait tout le travail, le credite qui arrive et acheve : c est
// exactement ce que 1/61 raconte.
//
// CONSEQUENCE DE METHODE, ET ELLE DEPASSE CE CHAMP : une quantite que le jeu n affiche pas n est
// pas pour autant invalidable. Le DEROULE d une action est une reference terrain a part entiere —
// il ne donne pas le nombre, il donne son ORDRE. Ne plus ecrire << infalsifiable >> la ou il faut
// lire << la valeur exacte est infalsifiable >>.
//
// DEUX PIEGES, tous deux mesures :
//   - `null` VEUT DIRE << NON MESURE >>, JAMAIS ZERO. La part de l assistant est `null` des qu il
//     n y a pas d assistant nomme : le champ y porte alors une CONSTANTE PAR FILM qui ne dit rien
//     du kill, et qui vaut 20 sur certains films — un pourcentage parfaitement credible.
//   - AUCUN PLAFOND A 100 n est impose : 1.7 % des morts nommees portent une valeur superieure,
//     jusqu a 228. Le degat excedentaire est l hypothese naturelle ; elle n est pas etablie, mais
//     une valeur > 100 est une DONNEE, pas une lecture ratee.
type partsJSON struct {
	Tueur     *int   `json:"part_du_tueur_pct"`
	Assistant *int   `json:"part_de_l_assistant_pct"`
	NonNommee *int   `json:"part_non_attribuee_pct"`
	Note      string `json:"note"`
}

type sourceJSON struct {
	Tag        string `json:"tag"`
	Etiquette  string `json:"etiquette"`
	NomPropre  bool   `json:"nom_propre"`
	Nature     string `json:"nature"`
	Statut     string `json:"statut"`
	Detail     string `json:"detail,omitempty"`
	Reserve    string `json:"reserve,omitempty"`
	Categorie  string `json:"categorie,omitempty"`
	NatureBrut string `json:"nature_identifiant"`
}

type lectureJSON struct {
	Voie          string `json:"voie"`
	Origine       string `json:"origine"`
	Multiplicite  int    `json:"multiplicite"`
	NotePonderer  string `json:"note_ponderation"`
	VoieIdentifie string `json:"voie_identifiant"`
}

func afficherJSON(r *rapport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(construireJSON(r))
}

func construireJSON(r *rapport) sortieJSON {
	res := r.result
	p := killsource.CatalogueProvenance()
	out := sortieJSON{
		Film:        r.film,
		Calibration: res.Calibration,
		Catalogue: catalogueJSON{
			Identifiants: killsource.CatalogueSize(), GenereLe: p.IDsDate,
			EtiquettesDu: p.LabelsDate, FichiersDuJeu: false, NombreLibelles: p.LabelCount,
		},
		Couverture:  couvertureDeJSON(res.Coverage),
		Sante:       santeDeJSON(res),
		Publication: publicationDeJSON(res),
		Morts:       make([]mortJSON, 0, len(res.Kills)),
	}
	for _, k := range res.Kills {
		out.Morts = append(out.Morts, mortDeJSON(k))
	}
	return out
}

// assistDeJSON : les trois etats, explicites. La note DIT lequel, pour qu un lecteur presse ne
// puisse pas confondre << pas d assistant >> avec << on ne sait pas >>.
func assistDeJSON(a killsource.Assist) assistJSON {
	j := assistJSON{
		Connu: a.Known, Joueur: a.Name, IndiceReplication: a.Index,
		Refus: a.Rejected, Surplus: a.Extra,
	}
	switch {
	case !a.Known:
		j.Note = "ON NE SAIT PAS : aucun kill-event n a ete attache a cette mort. Ce n est PAS " +
			"<< pas d assistant >>"
	case a.Rejected != "":
		j.Note = "un assistant etait declare mais il est REFUSE (" + a.Rejected + ") ; la mort " +
			"est publiee sans assistant"
	case a.Name == "":
		j.Note = "PAS D ASSISTANT, et c est une MESURE : le kill-event est lu et son champ " +
			"assistant est absent"
	default:
		j.Note = "assistant lu dans le kill-event attache a cette mort"
	}
	if a.Extra > 0 {
		j.Note = joindre(j.Note, "ATTENTION : des assistants EN SURPLUS ont ete observes sur "+
			"cette mort — l hypothese << un seul assistant >> ne tient plus ici")
	}
	return j
}

// partsDeJSON : les deux parts, et le residu.
//
// LE RESIDU DEPEND DU CAS, et c est pour cela qu il est calcule ICI et pas chez le lecteur : le
// total applicable vaut 100 quand le tueur est seul, mais 99 des qu un assistant est nomme — deux
// troncatures d entiers y perdent un point. Un consommateur qui recopierait la formule aurait une
// chance sur deux de se tromper de constante.
func partsDeJSON(k killsource.Kill) partsJSON {
	j := partsJSON{Note: "null = NON MESURE, jamais zero. Aucun plafond a 100 n est impose."}
	if k.KillerDamage.Known {
		v := k.KillerDamage.Pct
		j.Tueur = &v
	}
	if k.AssistDamage.Known {
		v := k.AssistDamage.Pct
		j.Assistant = &v
	}
	// Le residu n a de sens que si la part du tueur est mesuree : sans elle il n y a rien a
	// soustraire. Et il n est calcule que lorsque les DEUX parts requises sont la.
	if j.Tueur != nil {
		switch {
		case j.Assistant != nil:
			r := 99 - *j.Tueur - *j.Assistant
			j.NonNommee = &r
			j.Note = joindre(j.Note, "total applicable 99 (un assistant nomme) : deux troncatures "+
				"d entiers perdent un point")
		case k.Assist.Known && k.Assist.Name == "":
			r := 100 - *j.Tueur
			j.NonNommee = &r
			j.Note = joindre(j.Note, "total applicable 100 (aucun assistant, MESURE)")
		default:
			j.Note = joindre(j.Note, "residu non calcule : on ignore s il y avait un assistant, "+
				"donc on ignore si le total applicable est 99 ou 100")
		}
	}
	return j
}

func couvertureDeJSON(c killsource.Coverage) couvertureJSON {
	return couvertureJSON{
		Couvertes: c.Covered, CouplesReels: c.RealPairs,
		CouplesReconstruits: c.ReconstructedPairs, MortsDuKillFeed: c.FeedDeaths,
		CouplesFabriques: c.GhostPairs, MortsDeBot: c.BotDeaths,
		MortsParUnBot:      c.BotKillerDeaths,
		CouplesMemeInstant: c.SameInstantPairs, KillsDuKillFeed: c.FeedKills,
		MortsDeLAPI: nil,
		NoteMortsDeLAPI: "le quatrieme denominateur — les morts de l API, la seule reference " +
			"complete — exige une source externe et n est pas calculable hors ligne",
		NoteCouplesFabriques: "retires du denominateur : la vraie victime est un bot, la " +
			"reconstruction avait pris la victime du voisin — ce ne sont pas des morts manquees",
		NoteMortsDeBotNeuvete: "population NEUVE : le kill-feed du film est humain-seul, ces " +
			"morts n ont jamais ete au denominateur et ne s y additionnent pas",
		NoteMortsParUnBot: "SECONDE population NEUVE, symetrique de la precedente : le kill-feed " +
			"porte la MORT (la victime est humaine) mais aucun kill en face (le bot n a pas de " +
			"XUID). Le nom du tueur vient du roster de replication. Ces morts ne s additionnent " +
			"a aucun des trois denominateurs ci-dessus",
	}
}

func mortDeJSON(k killsource.Kill) mortJSON {
	m := mortJSON{
		TempsMS: k.TimeMS, Temps: mmss(k.TimeMS), Victime: k.Victim,
		Credit: creditJSON{Joueur: k.Feed.Killer, PresentAuFeed: k.Feed.Present},
		Source: sourceJSON{
			Tag: hex8(k.Source.Tag), Etiquette: k.Source.Display, NomPropre: k.Source.Named,
			Nature: natureFR(k.Source.Class), NatureBrut: string(k.Source.Class),
			Statut: string(k.Source.Status), Detail: k.Source.Detail,
			Reserve: k.Source.Reserve, Categorie: k.Source.Category.Name(),
		},
		Assistant:     assistDeJSON(k.Assist),
		PartsDeDegats: partsDeJSON(k),
		Divergence:    k.Diverges,
		Lecture: lectureJSON{
			Voie: voieFR(k.Read.Path), VoieIdentifie: string(k.Read.Path),
			Origine: origineFR(k.Read.Origin), Multiplicite: k.Read.Multiplicity,
			NotePonderer: "la voie sequentielle apparie 98.2 % de ses candidats au couple exact, " +
				"le balayage 78.4 % — ventilation du cout, pas deux precisions comparables",
		},
	}
	if !k.Feed.Present {
		m.Credit.NoteAbsenceFeed = "mort absente du kill-feed (victime bot) : la victime vient du " +
			"roster de replication, le tueur vient bien du kill-feed"
	}
	if k.Read.Origin == killsource.OriginBotKiller {
		m.Credit.NoteAbsenceFeed = "kill absent du kill-feed (TUEUR bot) : la mort y est bien, " +
			"c est le kill qui manque — le nom du tueur vient du roster de replication"
	}
	if k.Diverges {
		m.Source.Reserve = joindre(m.Source.Reserve,
			"la source appartenait a la VICTIME ; le credit du jeu designe un autre joueur, "+
				"les deux sont vrais")
	}
	return m
}

func joindre(a, b string) string {
	if a == "" {
		return b
	}
	return a + " · " + b
}

func hex8(t uint32) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = digits[t&0xf]
		t >>= 4
	}
	return string(out)
}
