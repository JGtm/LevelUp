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

	"levelup/go-api/internal/games"
)

// titreProduitDesArtefacts dit si le titre déclare `film.replay_artifact`, et DIT POURQUOI
// quand la réponse est non.
//
// Deux « non » distincts, comme pour les gates jumeaux du paquet (capabilityUsageArmee,
// capabilityBombeArmee) : un TOML illisible est un INCIDENT (WARN — la configuration du titre
// est cassée, quelqu'un doit le voir), une clé absente est une CONFIGURATION DE TITRE. Cette
// dernière est journalisée en INFO, une seule ligne par cycle : contrairement aux gates des
// projections — qui ne parlent que d'un dérivé — celle-ci éteint l'étape ENTIÈRE, et « le
// rejeu ne tourne pas pour ce titre » ne doit pas se lire au même niveau que du bavardage de
// boucle. Elle reste bornée : Run n'est appelée qu'une fois par cycle de sync et par joueur.
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
		slog.InfoContext(ctx, "post-sync: rejeu 2D — titre sans la capability, aucune production",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug,
			"capability", string(games.CapFilmReplayArtifact))
		return false
	}
	return true
}
