package replay

// identity_registry_film_table.go — LA TABLE DU FILM EST LE LIEN DIRECT (lot 1.6.1).
//
// # LA DOCTRINE, ET L'ORDRE QU'ELLE IMPOSE
//
// « L'index c'est l'index » (decision utilisateur du 2026-09-07) : quand le film ECRIT le lien
// `index <-> xuid <-> gamertag`, c'est lui qu'on lit, a 100 %, et aucune autre voie ne le
// remplace. Ce fichier compose les deux lectures du film dans un ORDRE FIXE :
//
//  1. LA TABLE DU FILM (`chunk_00`, lot 1.5) pose les sieges qu'elle porte — voie `film_table` ;
//  2. LA LECTURE DES CHUNKS DE REPLICATION (`player_index.go`) COMPLETE, pour les seuls xuids
//     dont la table est MUETTE — voie `PlayerIndexTable`, repli NOMME et COMPTE ;
//  3. ce que ni l'une ni l'autre ne porte n'entre pas : rien ne s'invente.
//
// # LE REPLI SE DECLENCHE SUR UN DIAGNOSTIC, JAMAIS SUR UN DESACCORD (D14 b)
//
// Le diagnostic est « ce xuid n'a pas de siege dans la table du film », et il a une population
// MESUREE : la table du film est celle du DEBUT du film, donc un joueur qui rejoint en cours de
// partie n'y figure pas (mesure du 2026-09-14, huit builds : 0 a 5 par film, 13 au total). Un
// DESACCORD, lui, ne declenche rien : le film fait foi, l'ecart se COMPTE
// (`contradiction`) et le relecteur le lit. Mesure du 2026-09-14 : 125 accords, 0 contradiction.
//
// # LE CONTROLE, ET CE QU'IL EST
//
// La lecture des chunks est aussi le CONTROLE de la table du film, et c'est bien « la table de
// la base » au sens du plan : son roster d'entree vient de la feuille de match completee par le
// fil des morts (`rosterOf(deaths, opt.RosterXUIDs)`, film_scan.go) — sans la base, elle ne voit
// que les joueurs qui meurent. Trois comptes, publies dans `coverage.identity.filmTable` :
//
//	accord         le controle donne le MEME index que la table du film ;
//	contradiction  il en donne un AUTRE — la table du film est conservee, l'ecart est compte ;
//	silence        le controle ne porte pas ce xuid (il ne l'a pas cherche, ou ne l'a pas trouve).
//
// # LE VACANT INTERCALE EST UNE CONTRADICTION, PAS UNE LECTURE (decouverte D3 (1.5))
//
// `PlayerSlot.FilmIndex` est le rang ABSOLU dans la table de 32, vacants compris. Tant qu'aucun
// vacant n'est INTERCALE, ce rang EST le `player_index` de production (76 films sur 76). Quand un
// vacant s'intercale, les deux lectures divergent et AUCUN oracle ne les departage : les 13 films
// du cache concernes n'ont aucun document de rejeu (mesure du 2026-09-14). Le lien direct ne
// s'affirme donc PAS sur ces films — la table entiere passe en repli, et le refus est nomme
// (`FilmPlayerTable.Lue`). Le jour ou un tel film aura un document, la question se tranchera sur
// piece ; d'ici la, on se tait plutot que de choisir.

import (
	"log/slog"

	"levelup/go-api/internal/games/canonical"
)

// filmTableLinks est la table d'index EFFECTIVE et la provenance de chacun de ses liens.
type filmTableLinks struct {
	// table est ce que le registre emploie partout : la composition « film d'abord, chunks en
	// complement ». Elle n'est JAMAIS plus pauvre que la lecture des chunks seule.
	table PlayerIndexTable
	// voie dit, par xuid, laquelle des deux lectures a pose le lien.
	voie map[uint64]canonical.LinkMethod
	// noms porte le gamertag que LE FILM ecrit, par xuid. Vide pour un xuid sans siege.
	noms map[uint64]string
	// couverture est ce que la section publie.
	couverture canonical.FilmTableCounts
}

// composerTableDIndex compose les deux lectures dans l'ordre de la doctrine et rend la table
// effective, la voie de chaque lien et la couverture.
func composerTableDIndex(in IdentityInput) filmTableLinks {
	out := filmTableLinks{
		table: PlayerIndexTable{
			ByXUID:        make(map[uint64]int, len(in.PlayerIndices.ByXUID)),
			Readings:      in.PlayerIndices.Readings,
			Disagreements: in.PlayerIndices.Disagreements,
		},
		voie: map[uint64]canonical.LinkMethod{},
		noms: map[uint64]string{},
		couverture: canonical.FilmTableCounts{
			Read: in.FilmTable.Lue(), Refusal: string(in.FilmTable.Refusal),
			Seats: in.FilmTable.Occupied,
		},
	}
	if in.FilmTable.InterleavedVacant && in.FilmTable.Refusal == FilmTableRead {
		// Une table LUE dont le rang est ambigu : le refus n'est pas une erreur de lecture, il
		// est une abstention, et il se nomme comme tel.
		out.couverture.Refusal = string(canonical.FilmTableInterleavedVacant)
	}
	if out.couverture.Read {
		poserSiegesDuFilm(&out, in)
	}
	completerParLesChunks(&out, in)
	return out
}

// poserSiegesDuFilm pose les liens que la table du film porte, et les confronte au controle.
func poserSiegesDuFilm(out *filmTableLinks, in IdentityInput) {
	for _, s := range in.FilmTable.Seats {
		if s.XUID == 0 {
			continue
		}
		out.table.ByXUID[s.XUID] = s.FilmIndex
		out.voie[s.XUID] = canonical.MethodFilmPlayerTable
		out.couverture.Direct++
		if s.Gamertag != "" {
			out.noms[s.XUID] = s.Gamertag
		}
		switch pi, connu := in.PlayerIndices.ByXUID[s.XUID]; {
		case !connu:
			out.couverture.Silence++
		case pi == s.FilmIndex:
			out.couverture.Accord++
		default:
			out.couverture.Contradiction++
		}
	}
}

// completerParLesChunks ajoute les xuids dont la table du film est MUETTE. C'est le repli, et il
// ne se declenche que sur ce diagnostic — jamais sur un desaccord.
func completerParLesChunks(out *filmTableLinks, in IdentityInput) {
	for x, pi := range in.PlayerIndices.ByXUID {
		if _, deja := out.table.ByXUID[x]; deja {
			continue
		}
		out.table.ByXUID[x] = pi
		out.voie[x] = canonical.MethodPlayerIndexTable
		out.couverture.Fallback++
	}
}

// alarmerSurLaTableDuFilm journalise ce que la composition a refuse et ce qu'elle a contredit.
// Un refus ou une contradiction se disent AVANT toute degradation (regle n° 3 du depot).
func (l filmTableLinks) alarmerSurLaTableDuFilm(matchID string) {
	c := l.couverture
	slog.Info("rejeu : table du film composee au registre d'identite", "match_id", matchID,
		"lue", c.Read, "refus", c.Refusal, "sieges", c.Seats, "direct", c.Direct,
		"repli", c.Fallback, "accord", c.Accord, "contradiction", c.Contradiction,
		"silence", c.Silence)
	if c.Contradiction > 0 {
		slog.Warn("rejeu : la lecture des chunks CONTREDIT la table du film sur des index — la "+
			"table du film fait foi, l'ecart est compte",
			"match_id", matchID, "contradictions", c.Contradiction, "accords", c.Accord)
	}
	if !c.Read {
		slog.Warn("rejeu : table du film NON EMPLOYEE — le lien index <-> xuid retombe "+
			"entierement sur la lecture des chunks de replication",
			"match_id", matchID, "refus", c.Refusal, "liens", c.Fallback)
	}
}

// TableDIndex rend la table d'index EFFECTIVE du registre — celle que tout lecteur doit employer.
// Le producteur du document en tire son roster ; deux tables du meme film divergeraient.
func (r IdentityRegistry) TableDIndex() PlayerIndexTable { return r.filmTable.table }

// NomsDuFilm rend le gamertag que LA TABLE DU FILM ecrit, par xuid. Vide quand la table n'a pas
// ete employee.
func (r IdentityRegistry) NomsDuFilm() map[uint64]string { return r.filmTable.noms }

// CouvertureTableDuFilm rend ce que la composition publie.
func (r IdentityRegistry) CouvertureTableDuFilm() canonical.FilmTableCounts {
	return r.filmTable.couverture
}

// voieDuLienDIndex rend la voie qui a pose le lien de ce xuid. [canonical.MethodPlayerIndexTable]
// par defaut : c'est la voie historique, et un xuid sans entree ne vient d'ailleurs que d'une
// table composee hors de ce fichier — ce que la construction interdit.
func (l filmTableLinks) voieDuLienDIndex(xuid uint64) canonical.LinkMethod {
	if m, ok := l.voie[xuid]; ok {
		return m
	}
	return canonical.MethodPlayerIndexTable
}
