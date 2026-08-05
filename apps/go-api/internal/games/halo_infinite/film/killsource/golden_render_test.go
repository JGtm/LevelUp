package killsource

// golden_render_test.go — LE RENDU DES FICHIERS FIGES.
//
// REGLE D ECRITURE, ET ELLE EST LA RAISON D ETRE DE CE FICHIER : chaque chiffre sort ACCOMPAGNE
// DE LA PHRASE QUI LE QUALIFIE. Un golden qui ne porterait que des nombres validerait n importe
// quoi — c est exactement la faute que le chantier a payee d une section entiere de journal, et
// [TestGoldenPhrasesJustes] la rend impossible en verifiant la presence des phrases elles-memes.

import (
	"fmt"
	"sort"
	"strings"
)

// enteteGolden : la recette de regeneration, dans le fichier lui-meme. Un fichier fige qui ne dit
// pas comment il a ete produit devient indechiffrable des que son auteur n est plus la.
const enteteGolden = `# GOLDEN killsource — %s
#
# Fige la sortie du decodeur de la SOURCE DE DEGAT. Ne PAS editer a la main.
#
# REGENERATION (les fixtures ne sont pas versionnees : 107 Mo pour les quatre films) :
#   1. les films de reference sont des repertoires chunk_NN.bin sous data/cache/film_chunks/ ;
#      ils se retelechargent avec  go run ./cmd/fetch_film_chunks/
#   2. KILLSOURCE_FIXTURES=<racine des films> \
#        go test ./internal/games/halo_infinite/film/killsource/ -run Golden -update
#
# CHAQUE CHIFFRE PORTE LE NOM DE SON DENOMINATEUR. Un taux nu ne veut rien dire ici : trois
# denominateurs coexistent hors ligne et ne donnent pas le meme resultat.
`

// rendreFilm : la sortie figee d un film.
func rendreFilm(film string, res *Result, neg negControle) string {
	var b strings.Builder
	fmt.Fprintf(&b, enteteGolden, "film "+film)
	sectionCouverture(&b, res.Coverage)
	sectionAncres(&b, film, res)
	sectionAutoInfligees(&b, res)
	sectionMortsParUnBot(&b, res)
	sectionControleNegatif(&b, neg)
	sectionVoies(&b, res)
	sectionSante(&b, res)
	return b.String()
}

// sectionCouverture : LES DENOMINATEURS, NOMMES, un par ligne.
func sectionCouverture(b *strings.Builder, c Coverage) {
	fmt.Fprint(b, "\n## COUVERTURE — chaque taux porte le nom de son denominateur\n")
	fmt.Fprintf(b, "%d / %d couples REELS (couples reconstruits moins les couples FABRIQUES)\n",
		c.Covered, c.RealPairs)
	fmt.Fprintf(b, "%d / %d couples reconstruits par la reconstruction de kill-feed\n",
		c.Covered, c.ReconstructedPairs)
	fmt.Fprintf(b, "%d / %d morts du KILL-FEED (il ne contient AUCUNE mort de bot)\n",
		c.Covered, c.FeedDeaths)
	fmt.Fprintf(b, "%d couple(s) FABRIQUE(S) retire(s) du denominateur : la vraie victime est un bot\n",
		c.GhostPairs)
	fmt.Fprintf(b, "%d mort(s) de BOT decodee(s) — POPULATION NEUVE, jamais au denominateur ci-dessus\n",
		c.BotDeaths)
	fmt.Fprintf(b, "%d mort(s) INFLIGEE(S) PAR un bot — SECONDE POPULATION NEUVE, jamais au denominateur\n",
		c.BotKillerDeaths)
	fmt.Fprint(b, "   ci-dessus non plus : le feed porte la MORT mais aucun kill en face\n")
	fmt.Fprintf(b, "%d kill(s) et %d mort(s) au kill-feed brut · %d couple(s) au MEME instant\n",
		c.FeedKills, c.FeedDeaths, c.SameInstantPairs)
}

// sectionAncres : LES ANCRES DE VERITE TERRAIN, TAGS COMPRIS. Le filet le plus important.
//
// Elles sont ecrites AVEC LEUR TAG et avec l etiquette publiee : un compte peut rester juste
// pendant qu une etiquette devient fausse, et c est precisement ce que ce bloc attrape.
func sectionAncres(b *strings.Builder, film string, res *Result) {
	as := anchors[film]
	fmt.Fprintf(b, "\n## ANCRES DE VERITE TERRAIN — %d confrontation(s) confirmee(s) en mode Theater\n", len(as))
	fmt.Fprint(b, "# format : instant  tag  divergence  etiquette publiee  nature  voie de lecture\n")
	parSeconde := map[string][]Kill{}
	for _, k := range res.Kills {
		parSeconde[clock(k.TimeMS)] = append(parSeconde[clock(k.TimeMS)], k)
	}
	for _, a := range as {
		fmt.Fprintf(b, "%s  %08x  divergence=%s  %s\n",
			a.at, a.tag, ouiNonGolden(a.diverges), ligneAncre(parSeconde[a.at], a))
	}
}

// ligneAncre : la ligne publiee qui porte l ancre. DEUX MORTS PEUVENT TOMBER DANS LA MEME
// SECONDE — on cherche donc celle qui porte le tag, on ne suppose pas qu il n y en a qu une.
func ligneAncre(lignes []Kill, a anchor2) string {
	for _, k := range lignes {
		if k.Source.Tag == a.tag && k.Diverges == a.diverges {
			return fmt.Sprintf("%q  %s  %s", k.Source.Display, k.Source.Class, k.Read.Path)
		}
	}
	if len(lignes) == 0 {
		return "AUCUNE LIGNE PUBLIEE A CETTE SECONDE"
	}
	got := make([]string, 0, len(lignes))
	for _, k := range lignes {
		got = append(got, fmt.Sprintf("%08x(divergence=%s)", k.Source.Tag, ouiNonGolden(k.Diverges)))
	}
	return "ANCRE NON RETROUVEE — la seconde porte " + strings.Join(got, " ")
}

// sectionAutoInfligees : LES MORTS DONT LA SOURCE APPARTIENT A LA VICTIME.
//
// C est ici que la doctrine des deux verites se verifie : le credit et la source ne designent pas
// le meme responsable, et LES DEUX SONT VRAIS. Confirme 8/8 en mode Theater (RE_LOG 7ter.63).
// Le credit est ecrit A COTE du tag, parce que c est leur COUPLE qui est la mesure.
func sectionAutoInfligees(b *strings.Builder, res *Result) {
	var lignes []Kill
	for _, k := range res.Kills {
		if k.Diverges {
			lignes = append(lignes, k)
		}
	}
	fmt.Fprintf(b, "\n## MORTS DONT LA SOURCE APPARTIENT A LA VICTIME — %d ligne(s)\n", len(lignes))
	fmt.Fprint(b, "# le kill-feed credite un AUTRE joueur : les deux verites sont vraies et divergent\n")
	fmt.Fprint(b, "# format : instant  victime  tag  etiquette  CREDIT DU JEU  voie  origine\n")
	for _, k := range lignes {
		fmt.Fprintf(b, "%s  %s  %08x  %q  credit=%s  %s  %s\n",
			clock(k.TimeMS), k.Victim, k.Source.Tag, k.Source.Display,
			k.Feed.Killer, k.Read.Path, k.Read.Origin)
	}
}

// sectionMortsParUnBot : LA POPULATION NEUVE, LIGNE PAR LIGNE (RE_LOG 7ter.79).
//
// Elle est ecrite EN ENTIER, avec tag, categorie, voie et assistant, parce qu elle est la seule du
// paquet a n avoir AUCUNE ancre Theater : personne ne peut rejouer en Theatre la mort d un humain
// tue par un bot en la designant par son couple, le kill-feed du jeu ne l attribuant a aucun
// gamertag. Ce que ces quatre lignes ont a la place, c est une COHERENCE INTERNE verifiable ici :
// la categorie et le tag se corroborent sans qu aucun des deux n ait servi a choisir l autre.
func sectionMortsParUnBot(b *strings.Builder, res *Result) {
	var lignes []Kill
	for _, k := range res.Kills {
		if k.Read.Origin == OriginBotKiller {
			lignes = append(lignes, k)
		}
	}
	fmt.Fprintf(b, "\n## MORTS INFLIGEES PAR UN BOT — %d ligne(s)\n", len(lignes))
	fmt.Fprint(b, "# le feed porte la MORT, pas le kill : le tueur vient du roster de replication\n")
	fmt.Fprint(b, "# format : instant  victime  tueur  tag  etiquette  categorie  voie  assistant\n")
	for _, k := range lignes {
		fmt.Fprintf(b, "%s  %s  %s  %08x  %q  %s  %s  assistant=%s\n",
			clock(k.TimeMS), k.Victim, k.Feed.Killer, k.Source.Tag, k.Source.Display,
			k.Source.Category.Name(), k.Read.Path, assistantGolden(k.Assist))
	}
}

// assistantGolden : les TROIS etats de l assistant, jamais deux. Un golden qui ecrirait "" pour
// << on ne sait pas >> et pour << pas d assistant >> effacerait la distinction que tout le schema
// protege.
func assistantGolden(a Assist) string {
	switch {
	case !a.Known:
		return "NON MESURE"
	case a.Name == "" && a.Rejected != "":
		return "REFUSE (" + a.Rejected + ")"
	case a.Name == "":
		return "AUCUN (mesure)"
	default:
		return a.Name
	}
}

// sectionControleNegatif : le scan invente-t-il des morts ?
func sectionControleNegatif(b *strings.Builder, n negControle) {
	fmt.Fprint(b, "\n## CONTROLE NEGATIF — le scan n invente pas de morts\n")
	fmt.Fprint(b, "# population ou la cible est ABSENTE par construction : les morts sont toutes\n")
	fmt.Fprint(b, "# dans les paquets A EVENTS, donc tout candidat trouve ici est un faux positif.\n")
	fmt.Fprintf(b, "%d paquet(s) SANS evenement · %d bits balayes · %d candidat(s) accepte(s)\n",
		n.paquets, n.bits, n.candidats)
}

// sectionVoies : le cout par voie, et les deux invariants de l hybride.
func sectionVoies(b *strings.Builder, res *Result) {
	s := res.Stats
	fmt.Fprint(b, "\n## COUT PAR VOIE DE LECTURE — ventilation, PAS deux precisions comparables\n")
	fmt.Fprintf(b, "marche   : %d propose(s), %d apparie(s) au couple exact, %d publiee(s)\n",
		s.Walk.Population, s.Walk.Matched, s.Walk.Published)
	fmt.Fprintf(b, "scan     : %d propose(s), %d apparie(s) au couple exact, %d publiee(s)\n",
		s.Scan.Population, s.Scan.Matched, s.Scan.Published)
	fmt.Fprintf(b, "marche, source appartenant a la victime : %d propose(s), %d publiee(s)\n",
		s.SelfWalk.Population, s.SelfWalk.Published)
	fmt.Fprintf(b, "scan,   source appartenant a la victime : %d propose(s), %d publiee(s)\n",
		s.SelfScan.Population, s.SelfScan.Published)
	fmt.Fprintf(b, "morts de BOT : %d propose(s), %d publiee(s)\n", s.Bot.Population, s.Bot.Published)
	fmt.Fprintf(b, "morts infligees PAR un bot : %d propose(s), %d apparie(s), %d publiee(s)\n",
		s.BotKiller.Population, s.BotKiller.Matched, s.BotKiller.Published)
	fmt.Fprintf(b, "%d enregistrement(s) lu(s) par LES DEUX voies · ACCORD %d · DESACCORD %d\n",
		s.Redundant, s.Agree, s.Disagree)
	fmt.Fprintf(b, "DESACCORD doit valoir 0 : sinon l hybride n est plus une PREFERENCE mais un ARBITRAGE\n")
	fmt.Fprintf(b, "%d mort(s) a plusieurs candidats en concurrence · %d enregistrement(s) sans position\n",
		s.MultiCandidate, s.NoBit)
	fmt.Fprintf(b, "%d / %d paquet(s) a evenements localise(s)\n", s.PacketsLocated, s.PacketsWithEvents)
	fmt.Fprintf(b, "calibration resolue automatiquement : %s\n", res.Calibration)
}

// sectionSante : le verdict de domaine et les compteurs.
func sectionSante(b *strings.Builder, res *Result) {
	h := res.Health
	fmt.Fprint(b, "\n## SANTE — un verdict porte sur le DOMAINE, pas sur les etiquettes publiees\n")
	fmt.Fprintf(b, "verdict : %s\n", h.Verdict())
	for _, a := range h.Alerts() {
		fmt.Fprintf(b, "ALERTE : %s\n", a)
	}
	fmt.Fprintf(b, "%d candidat(s) consulte(s) · %d publie(s) · %d inexplique(s) sur les candidats\n",
		h.Candidates, h.Published, h.UnexplainedTotal())
	fmt.Fprintf(b, "ventilation : couple sans equivalent %d · source-victime non retenue %d · indice de bot %d\n",
		h.UnexplainedPair, h.UnexplainedSelf, h.UnexplainedBotIdx)
	fmt.Fprintf(b, "hors roster : %d — COMPTEUR SEPARE, il ne se compte pas sur les candidats et n entre\n",
		h.OutOfRoster)
	fmt.Fprint(b, "   donc PAS dans le taux d inexpliques (un numerateur qui deborde son denominateur\n")
	fmt.Fprint(b, "   ne veut rien dire)\n")
	fmt.Fprintf(b, "tags hors catalogue vus par la MARCHE : %d — compteur principal de table perimee\n",
		h.TagOutOfCatalogueWalk)
	fmt.Fprint(b, "   PORTEE : bruit MESURE nul sur 5 films, et AVEUGLE a un tag servi par le seul SCAN\n")
	fmt.Fprintf(b, "publication ligne par ligne : %s (marge de bijection %d)\n",
		autoriseeGolden(res.LineByLinePublishable()), res.BijectionMargin)
	fmt.Fprintf(b, "roster : %d nom(s) dont %d humain(s)%s\n",
		len(res.Roster.Names), res.Roster.Humans, botsGolden(res.Roster))
}

func botsGolden(r Roster) string {
	if len(r.Bots) == 0 {
		return ", aucun bot declare par le film"
	}
	parts := make([]string, 0, len(r.Bots))
	for _, b := range r.Bots {
		parts = append(parts, fmt.Sprintf("%s (slot %d, bid %d)", b.Name, b.Slot, b.BotID))
	}
	sort.Strings(parts)
	return " · bots declares : " + strings.Join(parts, ", ")
}

func ouiNonGolden(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

func autoriseeGolden(b bool) string {
	if b {
		return "AUTORISEE"
	}
	return "REFUSEE (agregat seulement)"
}
