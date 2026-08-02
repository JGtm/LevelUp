package killcollector

// identities.go — L IDENTITE D UN JOUEUR DANS UN FILM, ET LA SEULE REGLE QUI LA RESOUT.
//
// Ce fichier existe pour une raison mesuree : la regle de resolution a ete FAUSSE pendant une
// passe entiere de backfill (16 908 morts ecrites, 10 avec un xuid de victime), et une regle
// qui se trompe en silence doit vivre a UN seul endroit, sous son propre en-tete.

import (
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// MatchIdentities : ce que la passe demande a la base sur les participants d un match.
//
// LES CHAMPS NE SE DEDUISENT PAS LES UNS DES AUTRES, et c est le piege qu ils evitent :
// `ShotsFired` est ABSENTE quand la colonne est NULL (« pas de reference »), alors que le joueur,
// lui, EXISTE. Deriver la liste des xuids des cles de la reference perdrait silencieusement ces
// joueurs-la — leurs tirs seraient decodes et jamais attribues.
type MatchIdentities struct {
	// ParXUID : `xuid -> gamertag`, par la vue canonique `v_gamertag_lookup`.
	ParXUID map[string]string
	// ParNom : `gamertag -> xuid`. Les noms AMBIGUS (deux participants homonymes) en sont
	// ABSENTS : ecrire les morts d un joueur sous le xuid d un autre serait pire que rien.
	ParNom map[string]string
	// XUIDs : tous les participants porteurs d un xuid, dans un ordre stable.
	XUIDs []string
	// ShotsFired : la reference de l API, par xuid. Une entree absente veut dire « aucune
	// reference » — la porte de publication REFUSE alors, elle ne suppose pas.
	ShotsFired map[string]int
}

// Resoudre : LE nom que le film donne devient un xuid et un gamertag. UNE SEULE COPIE DE CETTE
// REGLE EXISTE, et c est deliberé — elle a deja coute une passe entiere de backfill.
//
// LE FILM DONNE DEUX FORMES DE NOM, et les confondre est silencieux :
//
//	"Chocoboflor"           un GAMERTAG, tel que le kill-feed du film le porte. Il se resout
//	                        contre le roster du match, et un nom inconnu reste sans xuid.
//	"xuid:2535469190789936" LE XUID LUI-MEME, ecrit par le decodeur quand le film ne porte
//	                        aucun gamertag pour ce joueur (cf. killsource.XUIDNamePrefix).
//	                        C est l identite la PLUS FORTE, et la chercher dans une table de
//	                        gamertags ne rend evidemment rien.
//
// Mesure du defaut, le 2026-08-01 : traiter la seconde forme comme un gamertag a produit
// 16 908 morts dont **10** portaient un xuid de victime. Le nom d affichage, lui, repasse par la
// vue canonique — sinon la table stockerait `xuid:2535...` comme pseudo, exactement l « xuid brut
// a l affichage » que `v_gamertag_lookup` existe pour empecher.
func (m MatchIdentities) Resoudre(nom string) (xuid, gamertag string) {
	if reste, ok := strings.CutPrefix(nom, killsource.XUIDNamePrefix); ok {
		if estDecimal(reste) {
			if gt := m.ParXUID[reste]; gt != "" {
				return reste, gt
			}
			return reste, nom // xuid connu, nom inconnu : on garde la forme brute, honnete
		}
		return "", nom
	}
	return m.ParNom[nom], nom
}

// estDecimal : un xuid est une suite de chiffres, et rien d autre.
func estDecimal(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
