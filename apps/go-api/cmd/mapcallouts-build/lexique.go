package main

// lexique.go — LA PRODUCTION DU LEXIQUE DES NOMS DE LIEU : string_id -> libellé joueur EN/FR.
//
// POURQUOI UN SECOND FICHIER DE RÉFÉRENCE, à côté de callouts_i18n.csv.
//
// `callouts_i18n.csv` est indexé par (carte, volumeIndex) : c'est la table des zones des
// 22 cartes INTÉGRÉES, et elle reste la source de la passe native. Une carte Forge n'a ni
// module ni indice de volume — son `map.mvar` ne porte que le StringId du lieu — et son
// vocabulaire dépasse celui des cartes intégrées. D'où ce fichier-ci, plat et keyé par
// string_id : ce sont des mots du dictionnaire du jeu, pas des zones d'une carte donnée.
//
// SA SOURCE : les listes de chaînes localisées du jeu (`uslg`), décodées par
// internal/himap.LexiqueLieux — voir uslg.go pour le format. Il se régénère par
// `mapcallouts-build --lexique` et EXIGE le jeu installé. Sa LECTURE est dans
// `mapcatalog.ChargerLexique`, partagée avec le serveur : elle n'exige rien.
//
// SA FIABILITÉ EST VÉRIFIÉE, PAS POSTULÉE : le lexique reproduit les string_id de
// callouts_i18n.csv avec un texte EN et FR identique au caractère près (lexique_test.go, hors
// ligne, et uslg_gamefiles_test.go sur les fichiers du jeu). La fusion ci-dessous REFUSE
// d'ailleurs toute divergence plutôt que d'en trancher une au hasard : un faux nom de zone est
// pire qu'un nom absent.

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/mapcatalog"
)

// nomLexique : le fichier, à côté du catalogue, dans reference/ du titre.
const nomLexique = "callouts_lexique.csv"

// cheminLexique rend le chemin du lexique pour un catalogue donné.
func cheminLexique(outPath string) string {
	return filepath.Join(filepath.Dir(outPath), nomLexique)
}

// ecritLexique sérialise le lexique, trié par string_id (diff stable d'une extraction à
// l'autre). Une entrée dont un des deux textes manque est ÉCARTÉE : publier une moitié de
// couple ferait une zone nommée dans une langue et muette dans l'autre.
func ecritLexique(path string, lex map[uint32]himap.LibelleLieu) (int, int, error) {
	ids := make([]uint32, 0, len(lex))
	ecartes := 0
	for id, l := range lex {
		if l.EN == "" || l.FR == "" {
			ecartes++
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, 0, err
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = ';'
	w.UseCRLF = false
	if err := w.Write(mapcatalog.ColonnesLexique); err != nil {
		return 0, 0, err
	}
	for _, id := range ids {
		l := lex[id]
		if err := w.Write([]string{fmt.Sprintf("0x%08X", id), l.EN, l.FR}); err != nil {
			return 0, 0, err
		}
	}
	w.Flush()
	return len(ids), ecartes, w.Error()
}

// fusionneLexique complète l'index par string_id du CSV avec le lexique.
//
// Le CSV reste PRIORITAIRE en cas d'égalité de clé — c'est la table figée qui a servi à
// valider le catalogue natif. Une DIVERGENCE de texte est une erreur : elle signifierait
// que le lexique décrit un autre jeu que le CSV (mise à jour, décodeur cassé), et deux
// noms concurrents pour une même zone ne se départagent pas au hasard.
func fusionneLexique(base, lex mapcatalog.Lexique) (mapcatalog.Lexique, int, error) {
	out := mapcatalog.Lexique{}
	for k, v := range base {
		out[k] = v
	}
	ajouts := 0
	for id, l := range lex {
		vu, deja := out[id]
		if !deja {
			out[id] = l
			ajouts++
			continue
		}
		if vu != l {
			return nil, 0, fmt.Errorf("string_id %08x : le CSV dit (%q, %q), le lexique dit (%q, %q) — "+
				"lexique périmé ou décodeur cassé, rien n'est publié", id, vu.EN, vu.FR, l.EN, l.FR)
		}
	}
	return out, ajouts, nil
}
