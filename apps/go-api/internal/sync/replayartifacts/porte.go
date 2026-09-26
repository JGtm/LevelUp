package replayartifacts

// porte.go — LA PORTE DE CAPABILITY DE L'ÉTAPE POST-SYNC, en un seul exemplaire.
//
// # POURQUOI CE FICHIER EXISTE
//
// Quatre familles de cette étape ouvrent la même porte, et la lisaient en quatre exemplaires
// littéralement identiques à trois chaînes près : `titreProduitDesArtefacts` (l'artefact
// lui-même), `capabilityUsageArmee`, `capabilityBombeArmee` et la quatrième, arrivée avec les
// prises nettes le 2026-09-13. La règle du dépôt est explicite (CLAUDE.md n°6) : à la
// TROISIÈME copie, on centralise ET on pose un garde-rail — une factorisation sans garde-rail
// re-diverge.
//
// # CE QUE LA PORTE DOIT DISTINGUER, ET QUE TROIS DES QUATRE COPIES DISTINGUAIENT DÉJÀ
//
//	TOML ILLISIBLE          un INCIDENT. WARN, et l'appelant compte le lot en échecs — sans
//	                        quoi une configuration cassée passerait pour un titre qui ne
//	                        produit rien.
//	CLÉ ABSENTE             une CONFIGURATION DE TITRE. Pas un défaut : le titre ne déclare
//	                        pas cette famille, il n'y a rien à produire.
//	CLÉ PRÉSENTE            la porte est ouverte.
//
// Le niveau de journal du REFUS est laissé à l'appelant, parce qu'il n'est pas le même
// partout : la porte de l'artefact parle une fois par titre et par process (elle est franchie
// à CHAQUE cycle, même sur un titre sans décodeur), les portes des dérivations ne sont
// atteintes que si le cycle a cuit quelque chose et restent donc en DEBUG.
//
// # LE NIVEAU EST UNE FONCTION, PAS UNE VALEUR, ET C'EST NÉCESSAIRE
//
// `titreProduitDesArtefacts` le calcule par `premierRefusDeCeTitre`, qui a un EFFET DE BORD :
// il marque le titre comme déjà vu. L'évaluer avant de savoir si la clé est présente
// consommerait ce « premier refus » sur un titre qui, lui, produit — et le vrai premier refus
// d'un autre titre passerait alors en DEBUG. Le niveau ne s'évalue donc que dans la branche
// « clé absente ».

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games"
)

// porteCapability lit les capabilities du titre et dit si la famille `cle` est armée.
//
// `libelle` nomme la famille en clair dans le journal (« résumé d'usage », « prises nettes ») :
// une ligne de log qui ne dit que la clé technique oblige le lecteur à la traduire.
// `niveauRefus` peut être nil — le refus est alors journalisé en DEBUG.
//
// Rend `incident` vrai UNIQUEMENT quand le TOML est illisible : c'est le seul cas où
// l'appelant doit compter des échecs.
func porteCapability(
	ctx context.Context, d Deps, cle games.CapabilityKey, libelle string,
	niveauRefus func() slog.Level,
) (armee, incident bool) {
	caps, err := games.LoadCapabilityMap(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: "+libelle+" — capabilities illisibles, rien n'est produit",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "capability", string(cle), "err", err)
		return false, true
	}
	if !caps.Has(cle) {
		niveau := slog.LevelDebug
		if niveauRefus != nil {
			niveau = niveauRefus()
		}
		slog.Log(ctx, niveau, "post-sync: "+libelle+" — titre sans la capability, rien à produire",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "capability", string(cle))
		return false, false
	}
	return true, false
}

// echecFauteDeWriter enregistre au bilan qu'un lot de passes n'a pas été persisté faute de
// writer shared.
//
// TROISIÈME COPIE CENTRALISÉE (revue du 2026-09-13) : `echecBombe`, `echecPositions` et
// `echecPrisesNettes` étaient le même corps à trois types de tranche près. Sans cette trace,
// la marque de dérivation se poserait sur un match dont RIEN n'a été écrit (constat C1 de la
// revue A-R1) — c'est la raison d'être de la fonction, pas un détail de journalisation.
func echecFauteDeWriter(b *bilanDerivations, matchIDs []string) {
	b.writerIndisponible()
	for _, id := range matchIDs {
		b.echec(id)
	}
}
