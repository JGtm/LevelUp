// Command chatter-build — la table des répliques des Spartans, bâtie depuis le jeu installé.
//
// CE QU'ELLE PRODUIT. Pour chaque son de réplique parlée que les films jouent : son texte dans chaque langue de
// `config/locales.toml` et sa voix, une variante par son (le film ne dit pas laquelle a joué ; texte et voix
// viennent de la même, voir choix.go). Le fichier versé est
// `static/sounds/halo_infinite/repliques/repliques.json`, à côté des voix encodées `<langue>/<son>.ogg`
// (livrer.go) ; le rejeu les lit à l'ouverture d'un film qui porte des répliques nommées.
//
// POURQUOI PLUSIEURS ETAPES. Les modules du jeu pèsent plusieurs gigaoctets (0,6 Go pour les tags `snd!`, 7 Go
// pour les banques) : chaque étape est un processus, et les étapes s'échangent des fichiers JSON.
//
//	rejeux   -rejeux <dossier>                             sons joués par les films (aucun jeu requis)
//	textes   -deploy <racine> -sons s.json                 tags `snd!` -> variantes et textes    [jeu, tag gamefiles]
//	banques  -deploy <racine> -textes t.json               banques -> médias de chaque évènement [jeu, tag gamefiles]
//	choix    -sons s.json -textes t.json -banques b.json   une variante parlée par son (aucun jeu requis)
//	medias   -deploy <racine> -choix c.json -out <dossier> extrait les `.wem` choisis            [jeu, tag gamefiles]
//	voix     -src <dossier .wem> -out <dossier .wav>       décode (vgmstream) et met en stéréo   (aucun jeu requis)
//	         puis : go run ./cmd/sounds-encode -src <dossier .wav> -dst <dossier .ogg>
//	livrer   -choix c.json -voix fr=<dossier .ogg>[,...]   écrit la table et copie les voix       (aucun jeu requis)
//
// LES VOIX ANGLAISES. Le jeu installé n'a l'anglais qu'en écran d'accueil : les voix de Spartan anglaises sont à
// recevoir d'un tiers (leurs paquets `.pck`, ou des médias nommés comme le dit `medias`). Déposer leurs `.ogg`
// et relancer `livrer -voix fr=...,en=...` suffit : le client suit `audio` et n'annonce que ce qui existe.
//
// ENTREE ET SORTIE DES ETAPES sous `-travail` (défaut : le dossier courant) : sons.json, textes.json,
// banques.json, choix.json, wem/<langue>/, wav/<langue>/, ogg/<langue>/. Rien de cela n'est versé : seuls
// `repliques.json` et les `.ogg` le sont, aucun fichier brut du jeu n'entre dans le dépôt.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
)

const (
	moduleTags    = "any/globals/globals-rtx-new.module"
	moduleBanques = "pc/globals/globals-rtx-new.module"
	dossierVerse  = "static/sounds/halo_infinite/repliques"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := executer(ctx, os.Args[1:], log); err != nil {
		log.ErrorContext(ctx, "chatter-build: échec", "err", err)
		os.Exit(1)
	}
}

// commande : une étape de la chaîne, avec les options qu'elle lit.
type commande struct {
	nom string
	run func(ctx context.Context, o *options, log *slog.Logger) error
}

// options : l'union des options de toutes les étapes ; chaque étape lit les siennes.
type options struct {
	rejeux, deploy, sons, textes, banques, choix string
	travail, src, out, vgmstream, voix, verse    string
}

func executer(ctx context.Context, args []string, log *slog.Logger) error {
	cmds := []commande{
		{"rejeux", etapeRejeux}, {"textes", etapeTextes}, {"banques", etapeBanques},
		{"choix", etapeChoix}, {"medias", etapeMedias}, {"voix", etapeVoix}, {"livrer", etapeLivrer},
	}
	if len(args) == 0 {
		return fmt.Errorf("étape attendue parmi %s", nomsDesEtapes(cmds))
	}
	for _, c := range cmds {
		if c.nom != args[0] {
			continue
		}
		fs := flag.NewFlagSet(c.nom, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		o := &options{}
		fs.StringVar(&o.rejeux, "rejeux", "", "dossier des documents de rejeu (étape rejeux)")
		fs.StringVar(&o.deploy, "deploy", "", "racine `deploy` du jeu (défaut : détectée, tag gamefiles)")
		fs.StringVar(&o.sons, "sons", "", "sons.json (défaut : <travail>/sons.json)")
		fs.StringVar(&o.textes, "textes", "", "textes.json (défaut : <travail>/textes.json)")
		fs.StringVar(&o.banques, "banques", "", "banques.json (défaut : <travail>/banques.json)")
		fs.StringVar(&o.choix, "choix", "", "choix.json (défaut : <travail>/choix.json)")
		fs.StringVar(&o.travail, "travail", ".", "dossier de travail des fichiers intermédiaires")
		fs.StringVar(&o.src, "src", "", "dossier source (étape voix)")
		fs.StringVar(&o.out, "out", "", "sortie de l'étape (défaut : un fichier ou dossier sous <travail>)")
		fs.StringVar(&o.vgmstream, "vgmstream", "vgmstream-cli", "chemin de vgmstream-cli (étape voix)")
		fs.StringVar(&o.voix, "voix", "", "langue=dossier des .ogg encodés, séparés par des virgules (étape livrer)")
		fs.StringVar(&o.verse, "verse", "", "dossier versé (défaut : "+dossierVerse+" du dépôt)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("%s : %w", c.nom, err)
		}
		return c.run(ctx, o, log)
	}
	return fmt.Errorf("étape %q inconnue ; attendue parmi %s", args[0], nomsDesEtapes(cmds))
}

func nomsDesEtapes(cmds []commande) string {
	noms := make([]string, len(cmds))
	for i, c := range cmds {
		noms[i] = c.nom
	}
	return strings.Join(noms, ", ")
}

// parDefaut rend `valeur`, ou le fichier `nom` du dossier de travail quand elle est vide.
func (o *options) parDefaut(valeur, nom string) string {
	if valeur != "" {
		return valeur
	}
	return filepath.Join(o.travail, nom)
}

func etapeRejeux(_ context.Context, o *options, log *slog.Logger) error {
	if o.rejeux == "" {
		return errors.New("-rejeux : dossier des documents de rejeu requis")
	}
	res, err := lireSonsJoues(o.rejeux, log)
	if err != nil {
		return err
	}
	log.Info("chatter-build: sons joués", "films", res.Films, "sons", len(res.Sons))
	return ecrireJSON(o.parDefaut(o.out, "sons.json"), res)
}

func etapeTextes(_ context.Context, o *options, log *slog.Logger) error {
	var joues SonsJoues
	if err := lireJSON(o.parDefaut(o.sons, "sons.json"), &joues); err != nil {
		return err
	}
	langues, err := languesDeLApp()
	if err != nil {
		return err
	}
	racine, err := racineDuJeu(o.deploy)
	if err != nil {
		return err
	}
	mod, err := ouvrirModule(filepath.Join(racine, filepath.FromSlash(moduleTags)))
	if err != nil {
		return err
	}
	defer func() { _ = mod.Close() }()
	voulus := map[uint32]bool{}
	for id := range joues.Sons {
		if v, ok := parseHex(id); ok {
			voulus[v] = true
		}
	}
	res, err := extraireTextes(mod, voulus, langues, log)
	if err != nil {
		return err
	}
	return ecrireJSON(o.parDefaut(o.out, "textes.json"), res)
}

func etapeBanques(_ context.Context, o *options, log *slog.Logger) error {
	var textes []SonTextes
	if err := lireJSON(o.parDefaut(o.textes, "textes.json"), &textes); err != nil {
		return err
	}
	langues, err := languesDeLApp()
	if err != nil {
		return err
	}
	racine, err := racineDuJeu(o.deploy)
	if err != nil {
		return err
	}
	mod, err := ouvrirModule(filepath.Join(racine, filepath.FromSlash(moduleBanques)))
	if err != nil {
		return err
	}
	defer func() { _ = mod.Close() }()
	res, err := extraireBanques(mod, textes, langues, racineSons(racine), log)
	if err != nil {
		return err
	}
	return ecrireJSON(o.parDefaut(o.out, "banques.json"), res)
}

func etapeChoix(_ context.Context, o *options, log *slog.Logger) error {
	var (
		joues   SonsJoues
		textes  []SonTextes
		banques []SonBanque
	)
	for _, l := range []struct {
		chemin string
		cible  any
	}{{o.parDefaut(o.sons, "sons.json"), &joues}, {o.parDefaut(o.textes, "textes.json"), &textes}, {o.parDefaut(o.banques, "banques.json"), &banques}} {
		if err := lireJSON(l.chemin, l.cible); err != nil {
			return err
		}
	}
	langues, err := languesDeLApp()
	if err != nil {
		return err
	}
	choix, bilan := choisir(joues, textes, banques, langues)
	log.Info("chatter-build: choix", "joues", bilan.Joues, "parles", bilan.Parles, "sans_texte", bilan.SansTexte,
		"sans_variante", bilan.SansVariant, "sans_banque", bilan.SansBanque, "avec_voix", bilan.AvecVoix)
	return ecrireJSON(o.parDefaut(o.choix, "choix.json"), choix)
}

func etapeMedias(_ context.Context, o *options, log *slog.Logger) error {
	var choix []Choix
	if err := lireJSON(o.parDefaut(o.choix, "choix.json"), &choix); err != nil {
		return err
	}
	langues, err := languesDeLApp()
	if err != nil {
		return err
	}
	racine, err := racineDuJeu(o.deploy)
	if err != nil {
		return err
	}
	dest := o.out
	if dest == "" {
		dest = filepath.Join(o.travail, "wem")
	}
	emb, manquants, err := extrairePaquets(choix, langues, racineSons(racine), dest, log)
	if err != nil {
		return err
	}
	if manquants > 0 {
		log.Warn("chatter-build: médias introuvables (paquet de voix non installé : à recevoir)", "medias", manquants)
	}
	if len(emb) == 0 {
		return nil
	}
	mod, err := ouvrirModule(filepath.Join(racine, filepath.FromSlash(moduleBanques)))
	if err != nil {
		return err
	}
	defer func() { _ = mod.Close() }()
	_, err = extraireEmbarques(mod, emb, dest, log)
	return err
}

func etapeVoix(ctx context.Context, o *options, log *slog.Logger) error {
	if o.src == "" || o.out == "" {
		return errors.New("-src (dossier de .wem) et -out (dossier de .wav) requis")
	}
	n, err := convertirVoix(ctx, vgmstream{Exe: o.vgmstream}, mettreEnStereo, o.src, o.out, log)
	log.Info("chatter-build: voix décodées", "fichiers", n)
	return err
}

func etapeLivrer(_ context.Context, o *options, log *slog.Logger) error {
	var choix []Choix
	if err := lireJSON(o.parDefaut(o.choix, "choix.json"), &choix); err != nil {
		return err
	}
	langues, err := languesDeLApp()
	if err != nil {
		return err
	}
	voix, err := lireSourcesVoix(o.voix)
	if err != nil {
		return err
	}
	dossier := o.verse
	if dossier == "" {
		racine, err := title.FindRepoRoot()
		if err != nil {
			return err
		}
		dossier = filepath.Join(racine, filepath.FromSlash(dossierVerse))
	}
	t, err := livrer(choix, voix, langues, dossier, log)
	if err != nil {
		return err
	}
	if err := verifierVoix(t, langues, dossier); err != nil {
		return err
	}
	log.Info("chatter-build: table livrée", "repliques", len(t.Sons), "dossier", dossier)
	return nil
}

// lireSourcesVoix lit `fr=dossier,en=dossier`.
func lireSourcesVoix(s string) (sourcesVoix, error) {
	out := sourcesVoix{}
	for _, part := range strings.Split(s, ",") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		code, dossier, ok := strings.Cut(part, "=")
		if !ok || code == "" || dossier == "" {
			return nil, fmt.Errorf("-voix : %q, attendu langue=dossier", part)
		}
		out[strings.TrimSpace(code)] = strings.TrimSpace(dossier)
	}
	return out, nil
}
