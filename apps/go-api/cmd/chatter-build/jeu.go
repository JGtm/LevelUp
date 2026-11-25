package main

// jeu.go — LES ETAPES QUI LISENT UN MODULE DU JEU : `textes` (module des tags), `banques` (module des
// banques) et `medias` (paquets de voix + banques).
//
// Elles travaillent sur l'interface `sourceModule`, que `*himodule.Module` satisfait : les tests leur donnent
// un module fabriqué. L'ouverture d'un vrai module, elle, passe par `ouvrirModule` (jeu_gamefiles.go) et ne
// se compile que sous le tag `gamefiles`.

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JGtm/halo-infinite-files-reader/himodule"
)

// sourceModule : ce que les étapes demandent à un module. `*himodule.Module` y répond.
type sourceModule interface {
	Files(group string) []himodule.File
	Extract(f himodule.File) ([]byte, error)
	ResourceBlob(f himodule.File) ([]byte, error)
	Close() error
}

// Les groupes de tags que la chaîne lit.
const (
	groupeSon    = "snd!"
	groupeListe  = "uslg"
	groupeBanque = "sbnk"
)

func hex8(v uint32) string { return fmt.Sprintf("%08x", v) }

func parseHex(s string) (uint32, bool) {
	v, err := strconv.ParseUint(s, 16, 32)
	return uint32(v), err == nil
}

// extraireTextes lit, pour chaque son voulu, son tag `snd!` : la banque et la liste de chaînes qu'il désigne,
// puis ses variantes (évènement, clé de texte) et le texte de chaque variante dans chaque langue.
//
// Une variante est un couple de mots consécutifs à deux places d'écart dans le tag : l'évènement, puis, deux
// mots plus loin, une clé qui existe dans la liste de chaînes (mesuré sur 543 sons : une variante par
// évènement, jamais de clé orpheline). Un son dont la liste de chaînes manque est rendu sans variante.
func extraireTextes(mod sourceModule, voulus map[uint32]bool, langues []langueApp, log *slog.Logger) ([]SonTextes, error) {
	listes := map[uint32]himodule.File{}
	for _, f := range mod.Files(groupeListe) {
		listes[f.GlobalID] = f
	}
	tables := map[uint32][]map[uint32]string{}
	tableDe := func(id uint32) ([]map[uint32]string, error) {
		if t, ok := tables[id]; ok {
			return t, nil
		}
		f, ok := listes[id]
		if !ok {
			tables[id] = nil
			return nil, nil
		}
		blob, err := mod.ResourceBlob(f)
		if err != nil {
			return nil, fmt.Errorf("liste de chaînes %s : %w", hex8(id), err)
		}
		var t []map[uint32]string
		for _, sf := range sousFichiersUcsh(blob) {
			t = append(t, tableUcsh(sf))
		}
		tables[id] = t
		return t, nil
	}

	var res []SonTextes
	for _, f := range mod.Files(groupeSon) {
		if !voulus[f.GlobalID] {
			continue
		}
		d, err := mod.Extract(f)
		if err != nil {
			log.Error("chatter-build: tag de son illisible", "son", hex8(f.GlobalID), "err", err)
			continue
		}
		s := SonTextes{Son: hex8(f.GlobalID)}
		groupes, ids := dependancesSnd(d)
		var idListe uint32
		for i, g := range groupes {
			switch g {
			case groupeBanque:
				s.Banque = hex8(ids[i])
			case groupeListe:
				s.Liste = hex8(ids[i])
				idListe = ids[i]
			}
		}
		if s.Liste != "" {
			t, err := tableDe(idListe)
			if err != nil {
				return nil, err
			}
			s.Variantes = variantesDeSon(motsSnd(d, len(groupes)), t, langues)
		}
		res = append(res, s)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Son < res[j].Son })
	log.Info("chatter-build: textes lus", "sons", len(res), "demandes", len(voulus), "listes", len(tables))
	return res, nil
}

// variantesDeSon extrait les couples (évènement, clé de texte) des mots d'un tag de son, et leur texte dans
// chaque langue. `tables` est une table de chaînes par langue du jeu, dans l'ordre des blobs.
func variantesDeSon(mots []uint32, tables []map[uint32]string, langues []langueApp) []VarianteTexte {
	if len(tables) == 0 {
		return nil
	}
	const ecartEvenement = 2
	vus := map[string]bool{}
	var out []VarianteTexte
	for i := ecartEvenement; i < len(mots); i++ {
		if _, ok := tables[0][mots[i]]; !ok {
			continue
		}
		v := VarianteTexte{Evenement: hex8(mots[i-ecartEvenement]), Cle: hex8(mots[i]), Textes: map[string]string{}}
		if vus[v.Evenement+"/"+v.Cle] {
			continue
		}
		vus[v.Evenement+"/"+v.Cle] = true
		for _, l := range langues {
			if l.Jeu.Blob < len(tables) {
				if t := nettoyerTexte(tables[l.Jeu.Blob][mots[i]]); t != "" {
					v.Textes[l.Code] = t
				}
			}
		}
		out = append(out, v)
	}
	return out
}

// nettoyerTexte ramène les retours à la ligne et tabulations à une espace et retire les blancs de bord. Les
// espaces insécables de la typographie française (avant « ! », « ? », « : ») sont conservées : ce sont elles
// qui empêchent une ponctuation de passer seule à la ligne.
func nettoyerTexte(s string) string {
	return strings.TrimSpace(sautsDeLigne.ReplaceAllString(s, " "))
}

var sautsDeLigne = regexp.MustCompile(`[\t\r\n]+`)

// racineSons : le dossier `Sound/win` d'une installation, frère de `deploy`.
func racineSons(racineDeploy string) string {
	return filepath.Join(filepath.Dir(racineDeploy), "Sound", "win")
}

// indexPaquets rend, pour un dossier de voix, l'identifiant de média -> nom du paquet qui le contient. Seul
// l'en-tête de chaque paquet est lu.
func indexPaquets(dossier string, log *slog.Logger) (map[uint32]string, error) {
	chemins, err := filepath.Glob(filepath.Join(dossier, "*.pck"))
	if err != nil {
		return nil, err
	}
	idx := map[uint32]string{}
	for _, c := range chemins {
		ents, err := lirePckFichier(c)
		if err != nil {
			log.Error("chatter-build: paquet illisible", "paquet", filepath.Base(c), "err", err)
			continue
		}
		for _, e := range ents {
			if _, vu := idx[e.ID]; !vu {
				idx[e.ID] = filepath.Base(c)
			}
		}
	}
	return idx, nil
}

// extraireBanques rend, pour chaque son, les médias de chaque évènement dans chaque langue : l'évènement est
// descendu dans la banque du son jusqu'aux médias, qui se retrouvent dans les paquets de la langue ou, à
// défaut, dans les médias embarqués de la banque.
func extraireBanques(mod sourceModule, sons []SonTextes, langues []langueApp, racineSon string, log *slog.Logger) ([]SonBanque, error) {
	paquets := map[string]map[uint32]string{}
	for _, l := range langues {
		idx, err := indexPaquets(filepath.Join(racineSon, l.Jeu.Dossier), log)
		if err != nil {
			return nil, err
		}
		paquets[l.Code] = idx
		log.Info("chatter-build: paquets indexés", "langue", l.Code, "medias", len(idx))
	}
	parBanque := map[uint32][]int{}
	for i, s := range sons {
		if id, ok := parseHex(s.Banque); ok {
			parBanque[id] = append(parBanque[id], i)
		}
	}
	res := make([]SonBanque, len(sons))
	for i, s := range sons {
		res[i] = SonBanque{Son: s.Son, Absente: true}
	}
	for _, f := range mod.Files(groupeBanque) {
		idxSons, ok := parBanque[f.GlobalID]
		if !ok {
			continue
		}
		d, err := mod.Extract(f)
		if err != nil {
			log.Error("chatter-build: banque illisible", "banque", hex8(f.GlobalID), "err", err)
			continue
		}
		banques := separerBanques(d)
		for _, i := range idxSons {
			res[i] = banqueDuSon(sons[i], banques, langues, paquets)
		}
	}
	absentes := 0
	for _, r := range res {
		if r.Absente {
			absentes++
		}
	}
	log.Info("chatter-build: banques lues", "sons", len(res), "sans_banque", absentes)
	return res, nil
}

// banqueDuSon décrit les évènements d'un son d'après les banques de ses langues.
func banqueDuSon(s SonTextes, banques []banqueLangue, langues []langueApp, paquets map[string]map[uint32]string) SonBanque {
	r := SonBanque{Son: s.Son, Absente: len(banques) == 0}
	embarques := make([]map[uint32]bool, len(banques))
	for i, b := range banques {
		embarques[i] = map[uint32]bool{}
		for _, m := range mediasEmbarques(b) {
			embarques[i][m.ID] = true
		}
	}
	for _, v := range s.Variantes {
		ev, ok := parseHex(v.Evenement)
		if !ok {
			continue
		}
		e := EvenementBanque{Evenement: v.Evenement, Langues: map[string][]MediaVoix{}}
		for _, l := range langues {
			for i, b := range banques {
				if b.Langue != fnv1(l.Jeu.Dossier) {
					continue
				}
				if o, ok := b.Objets[ev]; !ok || o.Type != hircEvenement {
					continue
				}
				for _, m := range mediasDeLEvenement(b.Objets, ev) {
					media := MediaVoix{ID: hex8(m), Paquet: paquets[l.Code][m]}
					if media.Paquet == "" && embarques[i][m] {
						media.Embarque = true
					}
					e.Langues[l.Code] = append(e.Langues[l.Code], media)
				}
			}
		}
		r.Evenements = append(r.Evenements, e)
	}
	return r
}

// extrairePaquets écrit dans dest/<langue>/<id>.wem les médias de `choix` qui vivent dans un paquet de voix, et
// rend ceux qui sont embarqués dans une banque (à extraire avec le module des banques) et ceux qui manquent.
func extrairePaquets(choix []Choix, langues []langueApp, racineSon, dest string, log *slog.Logger) (embarques []embarque, manquants int, err error) {
	for _, l := range langues {
		var (
			paquets   = map[string][]MediaVoix{}
			demandes  = 0
			recuperes = 0
		)
		for _, c := range choix {
			m, ok := c.Voix[l.Code]
			if !ok {
				continue
			}
			demandes++
			switch {
			case m.Paquet != "":
				paquets[m.Paquet] = append(paquets[m.Paquet], m)
			case m.Embarque:
				embarques = append(embarques, embarque{Langue: l, Son: c.Son, Banque: c.Banque, Media: m})
				recuperes++
			default:
				manquants++
			}
		}
		dossier := filepath.Join(dest, l.Code)
		if err = os.MkdirAll(dossier, 0o755); err != nil {
			return nil, 0, err
		}
		noms := make([]string, 0, len(paquets))
		for p := range paquets {
			noms = append(noms, p)
		}
		sort.Strings(noms)
		for _, p := range noms {
			n, e := extraireDUnPaquet(filepath.Join(racineSon, l.Jeu.Dossier, p), paquets[p], dossier)
			if e != nil {
				return nil, 0, e
			}
			recuperes += n
		}
		log.Info("chatter-build: médias extraits", "langue", l.Code, "demandes", demandes, "extraits_ou_embarques", recuperes)
	}
	return embarques, manquants, nil
}

// embarque : un média à lire dans la banque de son son.
type embarque struct {
	Langue langueApp
	Son    string
	Banque string
	Media  MediaVoix
}

// extraireDUnPaquet écrit dans `dossier` les médias demandés d'un paquet, un fichier `<id>.wem` chacun.
func extraireDUnPaquet(chemin string, medias []MediaVoix, dossier string) (int, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	ents, err := lirePck(f, st.Size())
	if err != nil {
		return 0, fmt.Errorf("%s : %w", filepath.Base(chemin), err)
	}
	parID := make(map[uint32]entreePck, len(ents))
	for _, e := range ents {
		parID[e.ID] = e
	}
	n := 0
	for _, m := range medias {
		id, ok := parseHex(m.ID)
		if !ok {
			return n, fmt.Errorf("identifiant de média %q illisible", m.ID)
		}
		e, ok := parID[id]
		if !ok {
			return n, fmt.Errorf("média %s absent de %s", m.ID, filepath.Base(chemin))
		}
		buf := make([]byte, e.Taille)
		if _, err := f.ReadAt(buf, e.Offset); err != nil {
			return n, fmt.Errorf("média %s de %s : %w", m.ID, filepath.Base(chemin), err)
		}
		if err := os.WriteFile(filepath.Join(dossier, m.ID+".wem"), buf, 0o644); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// extraireEmbarques écrit, pour chaque média embarqué demandé, son `.wem` lu dans la banque de son son.
func extraireEmbarques(mod sourceModule, demandes []embarque, dest string, log *slog.Logger) (int, error) {
	parBanque := map[uint32][]embarque{}
	for _, d := range demandes {
		if id, ok := parseHex(d.Banque); ok {
			parBanque[id] = append(parBanque[id], d)
		}
	}
	n := 0
	for _, f := range mod.Files(groupeBanque) {
		ds, ok := parBanque[f.GlobalID]
		if !ok {
			continue
		}
		data, err := mod.Extract(f)
		if err != nil {
			return n, fmt.Errorf("banque %s : %w", hex8(f.GlobalID), err)
		}
		banques := separerBanques(data)
		for _, d := range ds {
			id, _ := parseHex(d.Media.ID)
			octets, err := mediaEmbarqueDe(banques, fnv1(d.Langue.Jeu.Dossier), id)
			if err != nil {
				return n, fmt.Errorf("son %s : %w", d.Son, err)
			}
			if err := os.WriteFile(filepath.Join(dest, d.Langue.Code, d.Media.ID+".wem"), octets, 0o644); err != nil {
				return n, err
			}
			n++
		}
	}
	log.Info("chatter-build: médias embarqués extraits", "fichiers", n, "demandes", len(demandes))
	return n, nil
}

// mediaEmbarqueDe cherche un média embarqué dans la banque de la langue `langue`.
func mediaEmbarqueDe(banques []banqueLangue, langue, id uint32) ([]byte, error) {
	for _, b := range banques {
		if b.Langue != langue {
			continue
		}
		for _, m := range mediasEmbarques(b) {
			if m.ID == id {
				return m.Octet, nil
			}
		}
	}
	return nil, errors.New("média embarqué " + hex8(id) + " introuvable dans la banque de la langue")
}
