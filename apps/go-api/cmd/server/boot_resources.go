// boot_resources.go — le paquet de ressources au démarrage (ADR 0045, amendement L24).
//
// Le moteur embarque l'empreinte de SON paquet (resourcesFingerprint) ; le paquet installé la porte dans
// resources.manifest.json. Au démarrage, avant que le serveur n'ouvre un fichier de ressources :
//
//   - un paquet téléchargé pendant une exécution précédente est activé s'il est celui du moteur, et une
//     bascule interrompue par un arrêt brutal est défaite (resourcepack.Settle) ;
//   - hébergé (capacité ResourcePackFatal) : une empreinte différente refuse le démarrage, comme une
//     migration fatale (état `resources_incompatible`) ;
//   - desktop (capacité UpdateApp) : une empreinte différente démarre quand même sur le paquet présent,
//     journalise un avertissement et télécharge le bon en arrière-plan (porte du mode jeu) ; un paquet
//     ABSENT est téléchargé tout de suite, et son échec refuse le démarrage ;
//   - sans empreinte embarquée (développement, tests) : aucun contrôle.
//
// Le téléchargement lui-même vit dans internal/resourcepack ; ce fichier ne fait que le brancher.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/netguard"
	"levelup/go-api/internal/resourcepack"
)

const (
	// resourcesReleaseBase : la racine des publications où vit le paquet de chaque version.
	resourcesReleaseBase = "https://github.com/JGtm/LevelUp/releases/download"
	// resourcesSurface nomme la sortie réseau du paquet dans le journal de netguard.
	resourcesSurface = "resources_pack"
	// stdinCommandResources : la commande de la coquille qui demande de préparer un paquet.
	stdinCommandResources = "resources"
	// resourcesDialTimeout borne l'établissement de la connexion et l'attente des en-têtes.
	resourcesDialTimeout = 60 * time.Second
	// resourcesMaxRedirects borne les redirections d'un téléchargement de publication.
	resourcesMaxRedirects = 5
)

// bootResourcePack est l'étape de démarrage décrite en tête de fichier. Elle suit la prise du verrou
// d'instance (aucun autre processus ne tient les fichiers du paquet) et précède l'outillage et le registre
// des titres, qui lisent config/ et le paquet.
func bootResourcePack(st *bootState) {
	expected := resourcesFingerprint
	if expected == "" {
		return
	}
	caps := st.cfg.Capabilities
	if caps.UpdateApp {
		st.resources = newResourceManager(st, expected)
		resourcepack.SetDefault(st.resources)
	}
	root := st.cfg.Roots().Resources
	out := settleResources(st, root, expected)
	err := verdictError(out.Verdict)
	switch resourceActionFor(caps, out.Verdict) {
	case resourceOK:
		slog.InfoContext(st.ctx, "resources: paquet conforme au moteur", "fingerprint", expected)
	case resourceRefuse:
		exitAfterRefusal(st, bootFailureResourcesIncompatible, err)
	case resourceDownloadNow:
		downloadMissingPack(st, out.Verdict, err)
	case resourceCatchUp:
		slog.WarnContext(st.ctx, "resources: paquet différent de celui du moteur, démarrage sur le paquet présent",
			"err", err, "state", out.Verdict.State)
		st.resourcesCatchUp = true
	default:
		slog.WarnContext(st.ctx, "resources: paquet différent de celui du moteur (aucun refus hors hébergé et desktop)",
			"err", err)
	}
}

// settleResources fait l'étape de démarrage du paquet ; avec un gestionnaire, il relit le paquet installé après
// l'activation (l'état publié dit alors le paquet en place).
func settleResources(st *bootState, root, expected string) resourcepack.Outcome {
	if st.resources != nil {
		return st.resources.Settle(st.ctx)
	}
	return resourcepack.Settle(st.ctx, root, expected)
}

// resourceAction : ce que le démarrage fait d'un verdict de contrôle.
type resourceAction int

const (
	// resourceOK : le paquet est celui du moteur.
	resourceOK resourceAction = iota
	// resourceRefuse : le démarrage est refusé (hébergé).
	resourceRefuse
	// resourceDownloadNow : paquet absent sur le bureau, à télécharger avant de servir.
	resourceDownloadNow
	// resourceCatchUp : paquet différent sur le bureau, le bon se télécharge en arrière-plan.
	resourceCatchUp
	// resourceWarn : paquet différent hors hébergé et desktop (développement) : journal seul.
	resourceWarn
)

// resourceActionFor décide, d'après les capacités du profil, ce que vaut un verdict.
func resourceActionFor(caps config.DeploymentCapabilities, v resourcepack.Verdict) resourceAction {
	switch {
	case v.State == resourcepack.StateMatch || v.State == resourcepack.StateUnchecked:
		return resourceOK
	case caps.ResourcePackFatal:
		return resourceRefuse
	case caps.UpdateApp && v.State == resourcepack.StateAbsent:
		return resourceDownloadNow
	case caps.UpdateApp:
		return resourceCatchUp
	default:
		return resourceWarn
	}
}

// downloadMissingPack télécharge, avant de servir quoi que ce soit, le paquet absent du bureau ; son échec refuse
// le démarrage. Un paquet différent ou illisible se remplace en arrière-plan (launchResources).
func downloadMissingPack(st *bootState, v resourcepack.Verdict, cause error) {
	slog.WarnContext(st.ctx, "resources: aucun paquet installé, téléchargement avant le démarrage", "err", cause)
	if err := st.resources.PrepareNow(st.ctx, st.resources.OwnDescriptor()); err != nil {
		exitAfterRefusal(st, bootFailureResourcesIncompatible, err)
	}
	out := settleResources(st, st.cfg.Roots().Resources, v.Expected)
	if out.Verdict.State != resourcepack.StateMatch {
		exitAfterRefusal(st, bootFailureResourcesIncompatible, verdictError(out.Verdict))
	}
}

// verdictError dit en une erreur ce qui sépare le paquet installé de celui du moteur.
func verdictError(v resourcepack.Verdict) error {
	switch v.State {
	case resourcepack.StateAbsent:
		return fmt.Errorf("aucun manifeste de ressources (%s) : le moteur attend le paquet %s",
			resourcepack.ManifestFileName, v.Expected)
	case resourcepack.StateInvalid:
		return fmt.Errorf("manifeste de ressources illisible : %w", v.Err)
	default:
		return fmt.Errorf("le paquet installé (%s) n'est pas celui du moteur (%s)", v.Active, v.Expected)
	}
}

// newResourceManager assemble le gestionnaire du profil desktop : télécharge par une sortie gardée de
// netguard (hôtes de publication seulement, https), attend la fin d'une partie, annonce chaque changement
// d'état à la coquille.
func newResourceManager(st *bootState, expected string) *resourcepack.Manager {
	cfg := resourcepack.Config{
		Root:        st.cfg.Roots().Resources,
		Expected:    expected,
		Version:     st.cfg.AppVersion,
		ReleaseBase: resourcesReleaseBase,
		Downloader:  newResourcesDownloader(),
		Gate: func(ctx context.Context, onWait func()) error {
			return gameproc.Default().WaitHeavyAllowed(ctx, onWait)
		},
		OnStatus: st.events.Resources,
	}
	if key, err := resourcepack.ParsePublicKey(st.cfg.UpdatePublicKey); err == nil {
		cfg.Key = &key
	} else {
		slog.WarnContext(st.ctx, "resources: clé publique de mise à jour absente ou illisible, aucun paquet ne sera téléchargé",
			"err", err)
	}
	return resourcepack.NewManager(cfg)
}

// newResourcesDownloader rend le téléchargeur du paquet : une sortie gardée par netguard, hôtes de publication
// seulement. Variable pour que les tests de démarrage servent le paquet depuis un serveur local.
var newResourcesDownloader = func() resourcepack.Downloader {
	return resourcepack.Downloader{Doer: netguard.Doer{Surface: resourcesSurface, Next: newResourcesClient()}}
}

// newResourcesClient rend le client des téléchargements : pas de délai global (une archive de plusieurs
// dizaines de Mo sur une connexion lente), mais un délai de connexion et d'en-têtes, et des redirections
// bornées qui ne quittent jamais les hôtes de publication.
func newResourcesClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = resourcesDialTimeout
	tr.TLSHandshakeTimeout = resourcesDialTimeout
	return &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= resourcesMaxRedirects {
				return errors.New("trop de redirections")
			}
			if !resourcepack.DefaultAllowURL(req.URL) {
				return fmt.Errorf("redirection refusée vers %s", req.URL.Redacted())
			}
			return nil
		},
	}
}

// launchResources branche le gestionnaire sur le serveur démarré : la commande `resources` de la stdin de la
// coquille et, quand le démarrage a eu lieu sur un paquet différent, le téléchargement du bon.
func launchResources(st *bootState) {
	m := st.resources
	if m == nil {
		return
	}
	// Les préparations suivent le cycle de vie du serveur : l'arrêt les annule puis les attend avant la fermeture des bases.
	m.Bind(st.schedulerCtx, func(fn func()) {
		if st.schedulerCtx.Err() != nil {
			return // arrêt en cours : plus de préparation
		}
		st.schedulerWG.Add(1)
		go func() {
			defer st.schedulerWG.Done()
			fn()
		}()
	})
	st.bootSrv.commands.HandleArg(stdinCommandResources, func(arg string) { startResourcesFromCommand(st.schedulerCtx, m, arg) })
	if st.resourcesCatchUp {
		if err := m.Start(m.OwnDescriptor()); err != nil {
			slog.WarnContext(st.ctx, "resources: téléchargement du paquet du moteur non lancé", "err", err)
		}
	}
}

// startResourcesFromCommand lance la préparation du paquet que décrit l'argument de la commande. Un
// descripteur illisible ou invalide est journalisé et ignoré : la coquille attend alors en vain et son délai
// tranche, rien n'est installé.
func startResourcesFromCommand(ctx context.Context, m *resourcepack.Manager, arg string) {
	var desc resourcepack.Descriptor
	if err := json.Unmarshal([]byte(arg), &desc); err != nil {
		slog.ErrorContext(ctx, "resources: commande avec un descripteur illisible", "err", err)
		return
	}
	if err := desc.Validate(); err != nil {
		slog.ErrorContext(ctx, "resources: commande avec un descripteur invalide", "err", err)
		return
	}
	switch err := m.Start(desc); {
	case err == nil:
		slog.InfoContext(ctx, "resources: préparation demandée", "fingerprint", desc.Fingerprint)
	case errors.Is(err, resourcepack.ErrBusy):
		slog.InfoContext(ctx, "resources: ce paquet est déjà en préparation, ses événements suivront", "fingerprint", desc.Fingerprint)
	default:
		slog.WarnContext(ctx, "resources: préparation non lancée", "err", err, "fingerprint", desc.Fingerprint)
	}
}
