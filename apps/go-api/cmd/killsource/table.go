package main

// table.go — LA TABLE DES MORTS, LISIBLE.
//
// PARTI PRIS D AFFICHAGE, et il decoule directement de la doctrine : les deux verites occupent
// deux colonnes de MEME POIDS VISUEL, separees par le marqueur de divergence. Il n existe pas de
// colonne << vrai tueur >> ni << arme corrigee >> : ce serait presenter une verite comme
// l amendement de l autre.
//
// LES DENOMINATEURS SONT ECRITS EN TOUTES LETTRES sous la table. Un taux nu ne veut rien dire —
// trois denominateurs coexistent offline, ils ne donnent pas le meme chiffre, et les confondre est
// la faute que le chantier a payee d une section entiere de journal.

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// marqueurDivergence : le signe qui dit que les deux verites ne designent pas le meme responsable.
const marqueurDivergence = "<>"

func afficherTable(r *rapport, o options) error {
	enteteFilm(r)
	nDiv := corpsTable(r, o)
	legende(r, nDiv, o)
	blocCouverture(r.result.Coverage)
	blocPublication(r.result)
	return nil
}

// enteteFilm : de quoi on parle, et avec quelle table de nommage. La date du catalogue accompagne
// toujours la sortie : une table de tags se perime en silence, et une sortie sans date ne se
// verifie pas.
func enteteFilm(r *rapport) {
	p := killsource.CatalogueProvenance()
	fmt.Printf("FILM %s — %d mort(s) publiee(s) en %s\n",
		r.film, len(r.result.Kills), r.duree.Round(1e8))
	fmt.Printf("catalogue embarque : %d identifiants de source (genere le %s), etiquettes du %s\n",
		killsource.CatalogueSize(), p.IDsDate, p.LabelsDate)
	fmt.Printf("calibration retenue automatiquement : %s\n\n", r.result.Calibration)
}

// corpsTable : la table elle-meme. Rend le nombre de lignes ou les deux verites divergent.
func corpsTable(r *rapport, o options) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// `assistant` et `parts` sont TOUJOURS visibles, pas reserves au mode detaille : ce sont deux
	// faits de premier ordre, et l un d eux — les parts de degats — n existe NULLE PART dans le
	// jeu. Les cacher derriere une option revenait a decoder pour personne.
	fmt.Fprint(w, "temps\tvictime\tSOURCE DU DEGAT FATAL\tnature\tCREDIT DU JEU\tassistant\tparts\t")
	if o.full {
		fmt.Fprint(w, "tag\tlecture\treserve\t")
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, "-----\t-------\t---------------------\t------\t-------------\t---------\t-----\t")
	if o.full {
		fmt.Fprint(w, "---\t-------\t-------\t")
	}
	fmt.Fprintln(w)

	tronque := 0
	for i, k := range r.result.Kills {
		if o.limit > 0 && i >= o.limit {
			tronque = len(r.result.Kills) - o.limit
			break
		}
		ligneTable(w, k, o)
	}
	// LA NOTE DE TRONCATURE SORT DU TABLEAU, jamais dedans : une cellule longue elargirait la
	// colonne des noms de joueurs sur toute la table et la rendrait illisible.
	_ = w.Flush()
	if tronque > 0 {
		fmt.Printf("...   %d ligne(s) de plus — retirer -limite pour tout voir.\n", tronque)
	}
	// LE COMPTE DES DIVERGENCES PORTE SUR LE FILM ENTIER, jamais sur l affichage : une legende qui
	// compterait les lignes visibles dirait quelque chose de faux des qu on tronque.
	return compte(r.result.Kills, func(k killsource.Kill) bool { return k.Diverges })
}

// ligneTable : une mort. La source et le credit sont deux colonnes, jamais une seule.
func ligneTable(w *tabwriter.Writer, k killsource.Kill, o options) {
	credit := k.Feed.Killer
	if !k.Feed.Present {
		credit = k.Feed.Killer + "  (mort absente du kill-feed)"
	}
	// LE SYMETRIQUE, ET IL FAUT LE DIRE AUSSI : ici c est la mort qui est au feed et le KILL qui
	// n y est pas. Le nom du tueur vient du roster de replication.
	if k.Read.Origin == killsource.OriginBotKiller {
		credit = k.Feed.Killer + "  (kill absent du kill-feed)"
	}
	if k.Diverges {
		credit += "  " + marqueurDivergence
	}
	source := k.Source.Display
	if c := categorieFR(k.Source.Category); c != "" {
		source += " · " + c
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t",
		mmss(k.TimeMS), k.Victim, source, natureFR(k.Source.Class), credit,
		assistantCourt(k.Assist), partsCourtes(k))
	if o.full {
		fmt.Fprintf(w, "%08x\t%s\t%s\t", k.Source.Tag, voieFR(k.Read.Path), statutFR(k.Source.Status))
	}
	fmt.Fprintln(w)
}

// assistantCourt : les TROIS etats en une cellule etroite, et ils restent distincts.
//
//	?          on ne sait pas — aucun kill-event attache a cette mort
//	—          pas d assistant, et c est une MESURE
//	(refuse)   un assistant etait declare, il est ecarte
//	<nom>      l assistant
//
// La distinction entre `?` et `—` est la raison d etre du champ : les confondre publierait un fait
// jamais observe. La legende sous la table les rappelle.
func assistantCourt(a killsource.Assist) string {
	switch {
	case !a.Known:
		return "?"
	case a.Rejected != "":
		return "(refuse)"
	case a.Name == "":
		return "—"
	}
	if a.Extra > 0 {
		return a.Name + " (+)"
	}
	return a.Name
}

// partsCourtes : `tueur/assistant` en pourcentage entier. `?` = non mesure, JAMAIS zero.
//
// Aucun plafond n est applique : une valeur au-dessus de 100 est une donnee (degat excedentaire),
// pas une lecture ratee, et l ecraser reviendrait a cacher ce qu on ne comprend pas encore.
func partsCourtes(k killsource.Kill) string {
	part := func(d killsource.DamageShare) string {
		if !d.Known {
			return "?"
		}
		return strconv.Itoa(d.Pct)
	}
	// Sans assistant NOMME, la part de l assistant n a pas de sens : le champ y porte une
	// constante par film. On affiche un tiret plutot qu un nombre credible mais faux.
	if k.Assist.Known && k.Assist.Name == "" {
		return part(k.KillerDamage) + "/—"
	}
	return part(k.KillerDamage) + "/" + part(k.AssistDamage)
}

// reserves : les reserves DISTINCTES rencontrees, dans l ordre d apparition.
//
// ELLES SORTENT DU TABLEAU, ET C EST UN CHOIX DE LISIBILITE, PAS UN ESCAMOTAGE : le texte d une
// reserve fait deux lignes et il est le MEME sur des dizaines de morts. Le mettre en cellule
// ecrase toutes les autres colonnes ; le mettre en legende le rend lisible et le garde entier.
func reserves(ks []killsource.Kill) []string {
	vu := map[string]bool{}
	var out []string
	for _, k := range ks {
		if k.Source.Reserve == "" || vu[k.Source.Reserve] {
			continue
		}
		vu[k.Source.Reserve] = true
		out = append(out, k.Source.Reserve)
	}
	return out
}

// legende : ce qu il faut avoir lu pour interpreter la table sans se tromper.
func legende(r *rapport, nDiv int, o options) {
	fmt.Println()
	if nDiv > 0 {
		fmt.Printf("%s  LES DEUX VERITES DIVERGENT sur %d ligne(s). La source du degat appartenait a la\n",
			marqueurDivergence, nDiv)
		fmt.Println("    VICTIME (roquette tiree trop pres, baril lance trop pres, chute) et le jeu credite")
		fmt.Println("    un autre joueur. LES DEUX SONT VRAIES : le credit est ce que le jeu affiche, la")
		fmt.Println("    source est d ou vient le degat. Confirme 8/8 en mode Theater.")
	}
	if n := compte(r.result.Kills, func(k killsource.Kill) bool { return !k.Source.Named }); n > 0 {
		fmt.Printf("\n<< Autres >> sur %d ligne(s) : aucun nom propre publiable pour cette source. La NATURE\n", n)
		fmt.Println("    reste juste et reste affichee ; c est le nom qui manque, pas la lecture.")
	}
	if !o.full {
		fmt.Println("\nAjouter -tout pour voir l identifiant brut de la source, la voie de lecture et le statut.")
		return
	}
	rs := reserves(r.result.Kills)
	if len(rs) == 0 {
		return
	}
	fmt.Println("\nCE QUE LES ETIQUETTES NE DISENT PAS — les reserves rencontrees, en entier :")
	for _, s := range rs {
		fmt.Printf("  - %s\n", s)
	}
	fmt.Println("\nCOLONNE << lecture >> : deux voies lisent le MEME champ, au meme bit quand elles")
	fmt.Println("  repondent toutes les deux. La sequentielle apparie 98.2 % de ses candidats au")
	fmt.Println("  couple exact du kill-feed, le balayage 78.4 % — a PONDERER, pas a interpreter.")
}

// blocCouverture : LES DENOMINATEURS, NOMMES. Trois se calculent hors ligne ; le quatrieme (les
// morts de l API) exige une source externe et n est donc PAS affiche ici — l annoncer sans
// pouvoir le calculer serait exactement l ambiguite que la doctrine interdit.
func blocCouverture(c killsource.Coverage) {
	fmt.Println("\nCOUVERTURE — chaque taux porte le nom de son denominateur")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %d / %d\tcouples REELS (couples reconstruits moins les couples fabriques)\t%s\t\n",
		c.Covered, c.RealPairs, pct(c.Covered, c.RealPairs))
	fmt.Fprintf(w, "  %d / %d\tcouples reconstruits par la reconstruction de kill-feed\t%s\t\n",
		c.Covered, c.ReconstructedPairs, pct(c.Covered, c.ReconstructedPairs))
	fmt.Fprintf(w, "  %d / %d\tmorts du KILL-FEED (il ne contient AUCUNE mort de bot)\t%s\t\n",
		c.Covered, c.FeedDeaths, pct(c.Covered, c.FeedDeaths))
	_ = w.Flush()
	if c.GhostPairs > 0 {
		fmt.Printf("  %d couple(s) FABRIQUE(S) retire(s) du denominateur : la vraie victime est un bot,\n", c.GhostPairs)
		fmt.Println("     la reconstruction avait pris la victime du voisin. Ce ne sont pas des morts manquees.")
	}
	if c.BotDeaths > 0 {
		fmt.Printf("  + %d mort(s) de BOT decodee(s) — POPULATION NEUVE, jamais au denominateur ci-dessus\n", c.BotDeaths)
		fmt.Println("     (le kill-feed du film est humain-seul : un bot n a pas de XUID).")
	}
	if c.BotKillerDeaths > 0 {
		fmt.Printf("  + %d mort(s) INFLIGEE(S) PAR un bot — SECONDE POPULATION NEUVE, jamais au\n", c.BotKillerDeaths)
		fmt.Println("     denominateur ci-dessus non plus : le feed porte la MORT, mais aucun kill en face.")
		fmt.Println("     Le nom du tueur vient du roster de replication, pas du kill-feed.")
	}
	fmt.Println("  Le quatrieme denominateur — les morts de l API, la seule reference complete — exige")
	fmt.Println("  une source externe : il n est pas calculable ici et n est donc pas affiche.")
}

// blocPublication : ce que le consommateur a le droit de faire de cette sortie.
func blocPublication(res *killsource.Result) {
	fmt.Println("\nCE QUE CETTE SORTIE AUTORISE")
	if res.LineByLinePublishable() {
		fmt.Printf("  publication LIGNE PAR LIGNE : autorisee (marge de bijection %d, sante %s)\n",
			res.BijectionMargin, res.Health.Verdict())
		return
	}
	fmt.Printf("  publication LIGNE PAR LIGNE : REFUSEE — agregat seulement (marge de bijection %d, sante %s)\n",
		res.BijectionMargin, res.Health.Verdict())
	if res.BijectionMargin <= 0 {
		fmt.Println("     marge nulle : au moins deux joueurs sont interchangeables, donc les attributions")
		fmt.Println("     individuelles sont fausses meme si l agregat est juste. C est le cas du BTB.")
	}
	for _, a := range res.Health.Alerts() {
		fmt.Printf("     ALERTE : %s\n", a)
	}
}

// compte : combien de morts verifient un predicat.
func compte(ks []killsource.Kill, ok func(killsource.Kill) bool) int {
	n := 0
	for _, k := range ks {
		if ok(k) {
			n++
		}
	}
	return n
}
