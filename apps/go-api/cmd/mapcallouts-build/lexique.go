package main

// lexique.go — LE LEXIQUE DES NOMS DE LIEU : string_id -> libellé joueur EN/FR.
//
// POURQUOI UN SECOND FICHIER DE RÉFÉRENCE, à côté de callouts_i18n.csv.
//
// `callouts_i18n.csv` est indexé par (carte, volumeIndex) : c'est la table des zones des
// 22 cartes INTÉGRÉES, et elle reste la source de la passe native. Une carte Forge n'a ni
// module ni indice de volume — son `map.mvar` ne porte que le StringId du lieu — et son
// vocabulaire dépasse celui des cartes intégrées : 266 string_id employés sur la rotation
// du 2026-08-27, dont 66 seulement figuraient au CSV (25 %). Les 200 autres n'ont PAS de
// ligne à écrire dans un fichier indexé par carte : ce sont des mots du dictionnaire du
// jeu, pas des zones d'une carte donnée. D'où ce fichier-ci, plat et keyé par string_id.
//
// SA SOURCE : les listes de chaînes localisées du jeu (`uslg`), décodées par
// internal/himap.LexiqueLieux — voir uslg.go pour le format. Il se régénère par
// `mapcallouts-build --lexique` et EXIGE le jeu installé ; sa LECTURE, elle, n'exige rien
// (c'est de la donnée versionnée, comme map_structure ou map_objectives).
//
// SA FIABILITÉ EST VÉRIFIÉE, PAS POSTULÉE : le lexique reproduit les 463 string_id de
// callouts_i18n.csv avec un texte EN et FR identique au caractère près (0 absent,
// 0 divergence — lexique_test.go, hors ligne, et uslg_gamefiles_test.go sur les fichiers du
// jeu). La fusion ci-dessous REFUSE d'ailleurs toute divergence plutôt que d'en trancher
// une au hasard : un faux nom de zone est pire qu'un nom absent.

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// nomLexique : le fichier, à côté du catalogue, dans reference/ du titre.
const nomLexique = "callouts_lexique.csv"

// colonnesLexique : l'en-tête attendu (un fichier réordonné doit échouer).
var colonnesLexique = []string{"string_id", "en", "fr"}

// cheminLexique rend le chemin du lexique pour un catalogue donné.
func cheminLexique(outPath string) string {
	return filepath.Join(filepath.Dir(outPath), nomLexique)
}

// chargeLexique lit le lexique versionné. Son absence n'est PAS une erreur silencieuse :
// l'appelant décide (la passe Forge se contentera alors du seul CSV), mais elle est
// journalisée — un lexique manquant fait retomber la couverture Forge à 25 %.
func chargeLexique(path string) (libellesParStringID, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("lexique invalide (%s) : %w", path, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("lexique vide (%s)", path)
	}
	head := rows[0]
	if len(head) > 0 {
		head[0] = strings.TrimPrefix(head[0], bomUTF8)
	}
	if len(head) != len(colonnesLexique) {
		return nil, fmt.Errorf("lexique : %d colonnes, attendu %d", len(head), len(colonnesLexique))
	}
	for i, c := range colonnesLexique {
		if head[i] != c {
			return nil, fmt.Errorf("lexique : colonne %d = %q, attendu %q", i, head[i], c)
		}
	}
	out := libellesParStringID{}
	for n, row := range rows[1:] {
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[0], "0x"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("lexique ligne %d : string_id %q : %w", n+2, row[0], err)
		}
		if row[1] == "" || row[2] == "" {
			return nil, fmt.Errorf("lexique ligne %d (%s) : libellé vide", n+2, row[0])
		}
		if vu, deja := out[uint32(sid)]; deja && (vu.en != row[1] || vu.fr != row[2]) {
			return nil, fmt.Errorf("lexique ligne %d : string_id %08x porte deux libellés", n+2, sid)
		}
		out[uint32(sid)] = libelle{en: row[1], fr: row[2], stringID: uint32(sid)}
	}
	return out, nil
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
	if err := w.Write(colonnesLexique); err != nil {
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
func fusionneLexique(base, lex libellesParStringID) (libellesParStringID, int, error) {
	out := libellesParStringID{}
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
		if vu.en != l.en || vu.fr != l.fr {
			return nil, 0, fmt.Errorf("string_id %08x : le CSV dit (%q, %q), le lexique dit (%q, %q) — "+
				"lexique périmé ou décodeur cassé, rien n'est publié", id, vu.en, vu.fr, l.en, l.fr)
		}
	}
	return out, ajouts, nil
}

// bomUTF8 : la marque d ordre des octets qu un tableur ajoute en tete d un CSV.
const bomUTF8 = "\uFEFF"
