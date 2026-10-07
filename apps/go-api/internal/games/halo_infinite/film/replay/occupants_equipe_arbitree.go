package replay

// occupants_equipe_arbitree.go — UN BOT DONT L'ENTITE ET LA DECLARATION DISENT DEUX EQUIPES : LA
// FEUILLE DE MATCH TRANCHE.
//
// # LA REGLE (decision utilisateur du 2026-10-07)
//
// Le film ecrit l'equipe d'un bot deux fois : le designateur de son entite `ti=9` et l'equipe de son
// entree BOT_METADATA. Quand les deux s'accordent, rien ne change ([occupants.equipeDe]). Quand elles
// se contredisent, le film ne sait pas laquelle croire, et l'API sait qui est dans quelle equipe :
// l'equipe publiee est celle de la ligne de la feuille de match du bot (`match_participants.team_id`),
// jointe par son `bid(N.0)` (`RosterEntry.Bid`, la forme meme de la feuille).
//
// # LA CORRESPONDANCE ENTRE LES DEUX ESPACES
//
// La valeur de la feuille se compare au designateur du film SANS TRADUCTION : c'est la convention
// du controle des equipes ([teamPublication.controler], `coverage.teams.{accord, contradiction}`),
// et la mesure qui la fonde est au plan `.ai/PLAN_REJEU_DERNIERS_CORRECTIFS_2026-10-07.md` (K1.2).
//
// # CE QUI NE S'ARBITRE PAS
//
// Un bot que la feuille ne porte pas (pas de `bid`, ligne absente, equipe inconnue) garde l'equipe
// de son entite, comme avant la regle : l'ecart reste compte (`rejeu_bots_equipe_sans_arbitre`) et
// journalise en ERREUR. Rien ne se devine.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// contradictionDEquipe : un bot dont l'entite et la declaration disent deux equipes, et ce que la
// feuille de match en dit (nil : elle ne le connait pas).
type contradictionDEquipe struct {
	nom, bid         string
	entite, declaree int
	base             *int
}

// arbitrerParLaBase rend l'equipe d'un bot dont l'entite (`entite`) contredit la declaration
// (`declaree`) : celle de la feuille de match quand elle porte son `bid`, celle de l'entite sinon.
// La contradiction se compte et se garde pour le journal dans les deux cas.
func (o *occupants) arbitrerParLaBase(e RosterEntry, entite, declaree int, base map[string]int) *int {
	o.equipesContreDeclaration++
	c := contradictionDEquipe{nom: e.Name, bid: e.Bid, entite: entite, declaree: declaree}
	if t, ok := equipeDeLaFeuille(e.Bid, base); ok {
		c.base = &t
		o.contradictions = append(o.contradictions, c)
		return &t
	}
	o.contradictions = append(o.contradictions, c)
	return &entite
}

// equipeDeLaFeuille rend l'equipe que la feuille de match donne a un identifiant. Un identifiant vide
// ne joint rien, et une equipe negative est « inconnue » (meme sentinelle que la feuille, -1).
func equipeDeLaFeuille(id string, base map[string]int) (int, bool) {
	if id == "" {
		return 0, false
	}
	t, ok := base[id]
	return t, ok && t >= 0
}

// metriqueEquipesContreDeclaration : le compteur expvar des bots dont l'entite `ti=9` et l'entree
// BOT_METADATA disent deux equipes, arbitres ou non (cf. [occupants.equipeDe]).
const metriqueEquipesContreDeclaration = "rejeu_bots_equipe_contre_declaration"

// metriqueEquipesSansArbitre : parmi eux, ceux que la feuille de match ne connait pas — l'entite est
// publiee.
const metriqueEquipesSansArbitre = "rejeu_bots_equipe_sans_arbitre"

// journaliserLesEquipesDeclarees dit d'ou viennent les equipes des bots que la liaison a posees :
// combien de leur seule declaration BOT_METADATA (une lecture du film) ; puis chaque contradiction
// entite / declaration, au compteur, en avertissement quand la feuille de match l'a tranchee, en
// ERREUR (et au second compteur) quand elle ne connait pas le bot et que l'entite est publiee.
func journaliserLesEquipesDeclarees(ctx context.Context, matchID string, occ occupants) {
	if occ.equipesParDeclaration > 0 {
		slog.InfoContext(ctx, "rejeu : equipe de bot(s) lue dans leur declaration BOT_METADATA, faute d'entite ti=9",
			"match_id", matchID, "bots", occ.equipesParDeclaration)
	}
	if occ.equipesContreDeclaration > 0 {
		observability.AddInt(metriqueEquipesContreDeclaration, int64(occ.equipesContreDeclaration))
	}
	for _, c := range occ.contradictions {
		if c.base != nil {
			slog.WarnContext(ctx, "rejeu : l'entite ti=9 d'un bot et son entree BOT_METADATA disent deux equipes — "+
				"l'equipe de la feuille de match est publiee", "match_id", matchID, "bot", c.nom, "bid", c.bid,
				"entite", c.entite, "declaration", c.declaree, "feuille", *c.base)
			continue
		}
		observability.AddInt(metriqueEquipesSansArbitre, 1)
		slog.ErrorContext(ctx, "rejeu : l'entite ti=9 d'un bot et son entree BOT_METADATA disent deux equipes, "+
			"et la feuille de match ne connait pas ce bot — l'entite est publiee, l'ecart est compte",
			"match_id", matchID, "bot", c.nom, "bid", c.bid, "entite", c.entite, "declaration", c.declaree)
	}
}
