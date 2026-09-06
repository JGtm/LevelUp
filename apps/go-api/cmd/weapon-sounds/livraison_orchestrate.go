package main

// livraison_orchestrate.go — le point d'entree du mode `livrer` : charge les cinq fichiers
// de `_donnees`, decide l'ordre de traitement, livre chaque dossier candidat, ecrit
// `weaponSoundVariations.ts` et le rapport de console — port du corps principal de
// livraison.py (la boucle d'assemblage et la generation du fichier TS).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// livrer est le mode `livrer`. Voir l'en-tete de livraison.go pour le contrat complet.
func livrer(donneesDir, sonsRacine, depotCible string) error {
	if sonsRacine == "" {
		sonsRacine = filepath.Dir(filepath.Clean(donneesDir))
	}
	if depotCible == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("livraison: racine du depot introuvable (passer -depot) : %w", err)
		}
		depotCible = root
	}
	d, err := livraisonCharger(donneesDir)
	if err != nil {
		return err
	}
	ordre := livraisonOrdre(livraisonCandidats(d), d.Manifeste)

	cible := filepath.Join(depotCible, "static", "sounds", "halo_infinite")
	if err := os.MkdirAll(cible, 0o755); err != nil {
		return err
	}

	// ECRITURE ATOMIQUE DU LOT. Le script Python vidait la cible de ses `hinf_*.wav` AVANT de
	// produire quoi que ce soit : toute erreur en cours de route (source illisible, vote mal
	// forme, disque plein) laissait la cible A MOITIE VIDEE, avec un `weaponSoundVariations.ts`
	// jamais reecrit — un depot casse pour un accident de donnees (constat C7 de la revue R1).
	// Tout est donc produit d'abord dans un repertoire d'attente, VOISIN de la cible pour que
	// la publication se fasse par renommage sur le meme volume ; rien n'est efface tant que le
	// lot n'est pas complet.
	attente, err := os.MkdirTemp(filepath.Dir(cible), ".livrer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(attente)

	ch := livraisonChemins{
		SonsRacine: sonsRacine,
		Cible:      attente,
		Tmp:        filepath.Join(attente, "_rendu.wav"),
	}
	e := &livraisonEtat{Livres: map[string]string{}, Variations: map[string]*livraisonVariationOut{}}
	for _, dossier := range ordre {
		if err := livraisonTraiterUnDossier(dossier, ch, d, e); err != nil {
			return err
		}
	}

	if err := livraisonPublier(attente, cible); err != nil {
		return err
	}
	tsvar := filepath.Join(depotCible, "apps", "web", "src", "features", "match-replay", "weaponSoundVariations.ts")
	if err := livraisonEcrireTS(tsvar, e.Variations); err != nil {
		return err
	}
	for _, l := range e.Lignes {
		fmt.Println(l)
	}
	return livraisonRapportFinal(cible, e.Variations)
}

// livraisonPublier fait passer le lot complet du repertoire d'attente a la cible, PUIS retire
// les `hinf_*.wav` que le nouveau lot ne remplace pas.
//
// L'ORDRE EST L'INVERSE DE CELUI DU SCRIPT PYTHON, ET C'EST VOULU : deplacer d'abord fait
// passer la cible de l'ancien lot a « ancien PLUS nouveau » puis au nouveau seul, sans jamais
// la vider. L'etat final est le meme — miroir strict du PERIMETRE ARMES : les fichiers
// d'evenements du pack utilisateur ne portent pas le prefixe et ne sont jamais touches.
func livraisonPublier(attente, cible string) error {
	entrees, err := os.ReadDir(attente)
	if err != nil {
		return err
	}
	publies := map[string]bool{}
	for _, en := range entrees {
		n := en.Name()
		if !livraisonEstFichierArme(n) {
			continue
		}
		if err := os.Rename(filepath.Join(attente, n), filepath.Join(cible, n)); err != nil {
			return err
		}
		publies[n] = true
	}
	return livraisonNettoyerArmes(cible, publies)
}

// livraisonTraiterUnDossier livre (ou refuse) UN dossier candidat, dans l'ordre de
// `livraisonOrdre` — port du corps de boucle principal de livraison.py.
func livraisonTraiterUnDossier(dossier string, ch livraisonChemins, d *livraisonDonnees, e *livraisonEtat) error {
	a := d.Manifeste[dossier]
	cle := *a.Cle
	if _, deja := e.Livres[cle]; deja {
		e.Lignes = append(e.Lignes, fmt.Sprintf("  variante servie par la base : %-34s (%s)", dossier, cle))
		return nil
	}
	choix, err := livraisonChoixDossier(dossier, ch.SonsRacine, d.Coups, d.Votes)
	if err != nil {
		return err
	}
	if choix.Source == "" {
		e.Lignes = append(e.Lignes, "  SANS FICHIER "+dossier)
		return nil
	}
	dst := filepath.Join(ch.Cible, cle+".wav")
	if strings.HasPrefix(choix.Source, "__RENDRE__") {
		if err := livraisonRendreEtTronquer(dossier, choix.Source, dst, ch, d); err != nil {
			return err
		}
	} else {
		srcP := filepath.Join(ch.SonsRacine, filepath.FromSlash(choix.Source))
		if _, statErr := os.Stat(srcP); statErr != nil {
			e.Lignes = append(e.Lignes, fmt.Sprintf("  INTROUVABLE %s -> %s", dossier, choix.Source))
			return nil
		}
		if err := livraisonTronquer(srcP, dst, livraisonDureeLivreeS); err != nil {
			return err
		}
	}
	e.Livres[cle] = dossier
	suffixe := ""
	if v := livraisonVariationDe(dossier, choix.EvHex, d.Lot1, d.Lot2); v != nil {
		e.Variations[cle] = v
		suffixe = "  +variation"
	}
	e.Lignes = append(e.Lignes, fmt.Sprintf("  %-34s %-20s -> %s.wav%s", dossier, choix.Groupe, cle, suffixe))
	return nil
}

// livraisonRendreEtTronquer rend l'evenement designe par le sentinel "__RENDRE__<hex>" dans
// un fichier temporaire, puis le tronque vers sa destination finale — le fichier temporaire
// est toujours efface, meme si la troncature echoue.
func livraisonRendreEtTronquer(dossier, source, dst string, ch livraisonChemins, d *livraisonDonnees) error {
	idEvent, err := livraisonParseHex32(strings.TrimPrefix(source, "__RENDRE__"))
	if err != nil {
		return fmt.Errorf("livraison: evenement a rendre illisible (%s): %w", source, err)
	}
	if err := livraisonRendreEvent(dossier, idEvent, ch.Tmp, ch.SonsRacine, d.Lot1); err != nil {
		return err
	}
	defer os.Remove(ch.Tmp)
	return livraisonTronquer(ch.Tmp, dst, livraisonDureeLivreeS)
}

// livraisonEstFichierArme : le PERIMETRE du miroir, et rien d'autre. Les sons d'evenements du
// pack utilisateur (melee_kill, camo_*, overshield_*...) ne portent pas ce prefixe.
func livraisonEstFichierArme(nom string) bool {
	return strings.HasPrefix(nom, "hinf_") && strings.HasSuffix(nom, ".wav")
}

// livraisonNettoyerArmes efface les `hinf_*.wav` de la cible qui ne font pas partie du lot
// qu'on vient de publier — MIROIR, PERIMETRE ARMES UNIQUEMENT.
func livraisonNettoyerArmes(cible string, publies map[string]bool) error {
	entries, err := os.ReadDir(cible)
	if err != nil {
		return err
	}
	for _, e := range entries {
		n := e.Name()
		if livraisonEstFichierArme(n) && !publies[n] {
			if err := os.Remove(filepath.Join(cible, n)); err != nil {
				return err
			}
		}
	}
	return nil
}

// livraisonRapportFinal imprime le compte et le poids des armes livrees, et confirme que les
// fichiers d'evenements du pack sont INTACTS — port des trois derniers print() de
// livraison.py.
func livraisonRapportFinal(cible string, variations map[string]*livraisonVariationOut) error {
	entries, err := os.ReadDir(cible)
	if err != nil {
		return err
	}
	var nArmes, nAutres int
	var octetsArmes int64
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "hinf_") {
			nArmes++
			if info, ierr := e.Info(); ierr == nil {
				octetsArmes += info.Size()
			}
			continue
		}
		nAutres++
	}
	fmt.Printf("\n%d fichiers d'armes livres, %d variations ; %d fichiers d'evenements du pack INTACTS\n",
		nArmes, len(variations), nAutres)
	fmt.Printf("taille armes : %.1f Mo\n", float64(octetsArmes)/1e6)
	return nil
}

// --- chargement des cinq fichiers _donnees ---

func livraisonCharger(donneesDir string) (*livraisonDonnees, error) {
	lot1, err := livraisonChargerLot1(filepath.Join(donneesDir, "lot1.json"))
	if err != nil {
		return nil, err
	}
	lot2, err := livraisonChargerLot2(filepath.Join(donneesDir, "lot2.json"))
	if err != nil {
		return nil, err
	}
	manifesteRoot, err := livraisonChargerJSONInto[livraisonManifesteRoot](filepath.Join(donneesDir, "manifeste.json"))
	if err != nil {
		return nil, err
	}
	coups, err := livraisonChargerJSONInto[map[string]livraisonCoupsEntree](filepath.Join(donneesDir, "coups.json"))
	if err != nil {
		return nil, err
	}
	votesRoot, err := livraisonChargerJSONInto[livraisonVotesRoot](filepath.Join(donneesDir, "votes-final.json"))
	if err != nil {
		return nil, err
	}
	manifesteByDossier, ordreDossiers := livraisonIndexManifeste(manifesteRoot.Armes)
	return &livraisonDonnees{
		Lot1: lot1, Lot2: lot2,
		Manifeste: manifesteByDossier, OrdreManifeste: ordreDossiers,
		Coups: coups, Votes: votesRoot.Votes,
	}, nil
}

func livraisonChargerLot1(chemin string) (map[string]livraisonLot1Arme, error) {
	root, err := livraisonChargerJSONInto[livraisonLot1Root](chemin)
	if err != nil {
		return nil, err
	}
	out := map[string]livraisonLot1Arme{}
	for _, a := range root.Armes {
		out[joliDossier(a.PCK)] = a // dernier gagne, comme {joli(pck): a for a in armes}
	}
	return out, nil
}

func livraisonChargerLot2(chemin string) (map[string]livraisonLot2Arme, error) {
	arr, err := livraisonChargerJSONInto[[]livraisonLot2Arme](chemin)
	if err != nil {
		return nil, err
	}
	out := map[string]livraisonLot2Arme{}
	for _, a := range arr {
		out[joliDossier(a.PCK)] = a
	}
	return out, nil
}

// livraisonIndexManifeste construit la table dossier -> arme et l'ordre d'apparition
// deduplique (premiere occurrence conserve la position, derniere gagne la valeur — meme
// comportement qu'un dict comprehension Python sur des cles dupliquees).
func livraisonIndexManifeste(armes []livraisonManifesteArme) (map[string]livraisonManifesteArme, []string) {
	byDossier := map[string]livraisonManifesteArme{}
	var ordre []string
	vus := map[string]bool{}
	for _, a := range armes {
		if !vus[a.Dossier] {
			ordre = append(ordre, a.Dossier)
			vus[a.Dossier] = true
		}
		byDossier[a.Dossier] = a
	}
	return byDossier, ordre
}

// livraisonChargerJSONInto lit et decode un fichier JSON dans le type demande. Les champs
// typés json.Number (livraisonPlage) gardent le TEXTE numerique d'origine sans qu'il soit
// necessaire d'armer UseNumber() sur un decodeur — ce comportement est celui du type lui
// meme, pas une option globale du decodage.
func livraisonChargerJSONInto[T any](chemin string) (T, error) {
	var cible T
	b, err := os.ReadFile(chemin)
	if err != nil {
		return cible, fmt.Errorf("livraison: lecture %s: %w", chemin, err)
	}
	if err := json.Unmarshal(b, &cible); err != nil {
		return cible, fmt.Errorf("livraison: analyse %s: %w", chemin, err)
	}
	return cible, nil
}

// --- generation de weaponSoundVariations.ts ---

// livraisonTSTemplate reproduit le gabarit de livraison.py, seul l'en-tete "GENERE PAR"
// change (mode Go plutot que le script Python hors depot, cf. journal du lot v2 G.3).
const livraisonTSTemplate = "/**\n" +
	" * weaponSoundVariations.ts — GENERE PAR `weapon-sounds -mode livrer` (cmd/weapon-sounds,\n" +
	" * recette `.ai/V7.5/RECETTE_SONS_ARMES.md`), NE PAS EDITER A LA MAIN : toute reprise\n" +
	" * rejoue la recette et reecrit ce fichier avec les sons.\n" +
	" *\n" +
	" * Les fourchettes RANGED extraites des banks Wwise du jeu, par stem d'arme : ce que le\n" +
	" * moteur du jeu fait varier a CHAQUE lecture (volume en dB, hauteur en centiemes).\n" +
	" * Une arme absente d'ici se joue pure — c'est le cas nominal, pas une erreur.\n" +
	" */\n" +
	"import type { SoundVariation } from './weaponSoundLogic'\n" +
	"\n" +
	"export const WEAPON_SOUND_VARIATIONS: Readonly<Record<string, SoundVariation>> = {\n" +
	"%s\n" +
	"}\n"

func livraisonEcrireTS(chemin string, variations map[string]*livraisonVariationOut) error {
	cles := make([]string, 0, len(variations))
	for c := range variations {
		cles = append(cles, c)
	}
	sort.Strings(cles)
	var lignes []string
	for _, cle := range cles {
		v := variations[cle]
		var champs []string
		if v.VolumeDB != nil {
			champs = append(champs, "volume_db: "+livraisonTsPlage(*v.VolumeDB))
		}
		if v.PitchCents != nil {
			champs = append(champs, "pitch_cents: "+livraisonTsPlage(*v.PitchCents))
		}
		lignes = append(lignes, fmt.Sprintf("  %s: { %s },", cle, strings.Join(champs, ", ")))
	}
	contenu := fmt.Sprintf(livraisonTSTemplate, strings.Join(lignes, "\n"))
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		return err
	}
	return os.WriteFile(chemin, []byte(contenu), 0o644)
}

func livraisonTsPlage(p livraisonPlage) string {
	return fmt.Sprintf("{ bas: %s, haut: %s }", livraisonFormatNombrePy(p.Bas), livraisonFormatNombrePy(p.Haut))
}

// livraisonFormatNombrePy formate un json.Number comme le ferait Python "%s" % valeur : un
// litteral SANS point ni exposant reste un entier tel quel (un entier JSON est un int
// Python) ; sinon, la representation la plus courte qui redonne la meme valeur (str() d'un
// float Python).
func livraisonFormatNombrePy(n json.Number) string {
	s := string(n)
	if !strings.ContainsAny(s, ".eE") {
		return s
	}
	v, err := n.Float64()
	if err != nil {
		return s
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
