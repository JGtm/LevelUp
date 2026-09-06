package replayartifacts

// capability.go — LA PORTE DE PRODUCTION DE L'ÉTAPE, en tête de Run.
//
// # POURQUOI ELLE EXISTE
//
// Jusqu'au 2026-09-05, TOUTE la chaîne du rejeu (le pont disque, la mise en file chez
// l'ouvrier, le rattrapage du catalogue de cartes, la cuisson) tournait pour n'importe quel
// titre : aucune clé de capability ne la gouvernait (registre
// `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`, constats D1 et D3). La seule sonde de titre
// arrivait APRÈS la mise en file et APRÈS le rattrapage des cartes, et c'était un
// `replaybuild.NewBuilder` — une dégradation par ABSENCE DE DONNÉE (« ce titre n'a pas de
// catalogue de bornes »), pas une déclaration d'intention. Un titre sans décodeur de film y
// entrait donc quand même : il téléchargeait des films, remplissait la file, et n'échouait
// qu'au bout.
//
// La décision utilisateur du 2026-09-05 tranche : `film.replay_artifact` gouverne la
// PRODUCTION — pas de clé, pas de cuisson — et l'affichage suit (pas de fichier, pas de page).
// Cette porte est donc la PREMIÈRE chose que fait `Run`, avant `selectionnerLeTravail`,
// avant `rattraperCartesAbsentes` et avant `enqueueAll`.
//
// # POURQUOI ELLE N'EST PAS DANS artifacts.go
//
// Même découpage par responsabilité que le reste du paquet (usage.go, bombstats.go, t0film.go
// portent chacun leur gate et leur projection) : `artifacts.go` décide QUOI faire, ce
// fichier-ci dit SI le titre a le droit d'en faire quoi que ce soit.

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/games"
)

// titreProduitDesArtefacts dit si le titre déclare `film.replay_artifact`, et DIT POURQUOI
// quand la réponse est non.
//
// Deux « non » distincts, comme pour les gates jumeaux du paquet (capabilityUsageArmee,
// capabilityBombeArmee) : un TOML illisible est un INCIDENT (WARN — la configuration du titre
// est cassée, quelqu'un doit le voir, à CHAQUE fois), une clé absente est une CONFIGURATION
// DE TITRE.
//
// # POURQUOI LA CLÉ ABSENTE SE DIT UNE FOIS EN INFO, PUIS EN DEBUG
//
// Cette porte n'éteint pas un dérivé, elle éteint l'ÉTAPE ENTIÈRE : « le rejeu ne tourne pas
// pour ce titre » mérite d'être vu une fois, pas noyé dans du DEBUG. Mais le répéter à chaque
// cycle serait du bruit permanent sur un état qui ne changera jamais — halo_5 est un titre
// ACTIF qui n'aura pas de décodeur de film, et `Run` est appelée à chaque cycle de sync et
// par joueur (observation C7 de la revue adversariale du 2026-09-06). Les gates jumeaux ne
// posent pas ce problème : ils ne sont atteints QUE si le cycle a effectivement cuit quelque
// chose, donc sur un titre sans film ils ne parlent jamais.
//
// D'où le compromis : INFO au PREMIER refus de chaque titre dans la vie du process, DEBUG
// ensuite. Un redémarrage rend une ligne INFO par titre — assez pour que l'état soit lisible
// au démarrage, jamais assez pour saturer.
//
// Le TOML est relu à chaque cycle, comme pour les deux autres gates : c'est une petite lecture
// de fichier, et elle suit les règles vivantes sans redémarrage.
func titreProduitDesArtefacts(ctx context.Context, d Deps) bool {
	caps, err := games.LoadCapabilityMap(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D non produit — capabilities illisibles",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "err", err)
		return false
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		niveau := slog.LevelDebug
		if premierRefusDeCeTitre(d.TitleSlug) {
			niveau = slog.LevelInfo
		}
		slog.Log(ctx, niveau, "post-sync: rejeu 2D — titre sans la capability, aucune production",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug,
			"capability", string(games.CapFilmReplayArtifact))
		return false
	}
	return true
}

// refusDejaDits mémorise les titres dont le refus a déjà été journalisé en INFO, pour la vie
// du process. Une map sous mutex, et non un sync.Once par titre : le nombre de titres est
// borné par la configuration (deux aujourd'hui), et Run peut être appelée depuis les cycles
// de plusieurs joueurs.
var refusDejaDits = struct {
	mu  sync.Mutex
	vus map[string]bool
}{vus: map[string]bool{}}

// premierRefusDeCeTitre rend vrai UNE SEULE FOIS par titre et par process, et marque au
// passage. Voir l'en-tête pour le pourquoi du changement de niveau.
func premierRefusDeCeTitre(slug string) bool {
	refusDejaDits.mu.Lock()
	defer refusDejaDits.mu.Unlock()
	if refusDejaDits.vus[slug] {
		return false
	}
	refusDejaDits.vus[slug] = true
	return true
}

// oublierRefusDits remet le mémo à zéro. RÉSERVÉ AUX TESTS : sans lui, l'ordre d'exécution
// des tests du paquet déciderait du niveau de la ligne attendue.
func oublierRefusDits() {
	refusDejaDits.mu.Lock()
	defer refusDejaDits.mu.Unlock()
	refusDejaDits.vus = map[string]bool{}
}
