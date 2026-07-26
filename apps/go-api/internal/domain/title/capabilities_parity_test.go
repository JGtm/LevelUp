package title

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

// capabilities_parity_test.go — garde-rail des capabilities TITLE-LEVEL
// (clés plates : "waypoint_match_url", "team_mmr" ; type Capability de registry.go).
// NE COUVRE PAS les capabilities DATA-LEVEL (games.CapabilityKey, clés pointées
// "match.objective.stats", déclarées en capabilities.toml) : deux systèmes distincts
// qui partagent des noms de constantes (games.CapWeaponAccuracy vs CapWeaponAccuracy) —
// cf. .ai/DIAG_WAYPOINT_COLUMN_INFINITE.md §4.4.
//
// Topologie RÉELLE des listes maintenues à la main (mesurée le 2026-07-26) :
//
//	(1) const Cap* (registry.go)              — source de vérité, 21 constantes
//	(2) knownCapabilities (config_loader.go)  — validation des title.toml
//	(3) TITLE_CAPABILITIES (capabilities.ts)  — miroir front, type TitleCapability
//	(4) listes Capabilities des descripteurs  — Go en dur (Infinite) + title.toml (autres)
//
// (1)<->(3) est DÉJÀ couvert par TestCapabilitiesGoTSMirror (capabilities_ts_mirror_test.go).
// Ce fichier ferme les liens restants :
//
//   - (1)<->(2) : un Cap* absent de knownCapabilities fait REJETER tout title.toml
//     qui le déclare — et LoadTitlesIntoRegistry n'enregistre alors PAS le titre du
//     tout (le titre entier disparaît du switcher). C'est le trou qui aurait mordu
//     si waypoint_match_url avait été ajouté côté Halo 5 sans toucher au Go.
//   - front -> (1) : tout littéral passé à un gate d'affichage doit être une
//     capability Go réelle (défense en profondeur derrière le typage TitleCapability,
//     qu'un `as TitleCapability` ou un littéral hors call-site typé contourne).
//   - (1) -> (4) : une capability que PLUS AUCUN titre public ne déclare est morte
//     (gate toujours false → feature invisible partout).
//   - (1) -> consommateurs : une constante que plus personne ne lit est du code mort
//     (anti-pattern n°1 du CLAUDE.md).
//
// Les extracteurs vivent dans capabilities_parity_scan_test.go : aucune liste n'est
// recopiée ici, elles sont toutes lues dans les sources.

// ---------------------------------------------------------------------------
// (1) <-> (2) : constantes Cap* <-> set de validation knownCapabilities
// ---------------------------------------------------------------------------

// TestCapabilitiesKnownSetMirrorsConstants — parité EXACTE entre les constantes
// Cap* de registry.go et les clés du set knownCapabilities (config_loader.go).
//
// Sens 1 (const -> set) : le trou réel. Une capability absente du set fait échouer
// la validation de tout title.toml qui la déclare ("capability inconnue") et le
// titre n'est PAS enregistré — perte silencieuse d'un titre entier au boot.
//
// Sens 2 (set -> const) : détecte une constante Cap* déclarée AILLEURS que dans
// registry.go. Elle serait invisible pour l'extracteur du miroir TS (qui ne parse
// que registry.go) : le garde-rail (1)<->(3) deviendrait aveugle sur elle.
func TestCapabilitiesKnownSetMirrorsConstants(t *testing.T) {
	root := capabilityRepoRoot(t)
	goCaps := extractGoCapabilities(t, capabilityRegistryPath(root))
	if len(goCaps) == 0 {
		t.Fatalf("aucune constante Cap* extraite de registry.go — parseur cassé ou fichier restructuré")
	}
	constNames := make(map[string]string, len(goCaps)) // nom -> valeur
	for value, name := range goCaps {
		constNames[name] = value
	}

	loaderPath := filepath.Join(root, "apps", "go-api", "internal", "domain", "title", "config_loader.go")
	known := extractKnownCapabilityKeys(t, loaderPath)
	if len(known) == 0 {
		t.Fatalf("aucune clé extraite de knownCapabilities (%s) — map renommée/restructurée ? "+
			"Le garde-rail passerait à vide : corriger l'extracteur AVANT de livrer", loaderPath)
	}

	for _, name := range sortedKeys(constNames) {
		if known[name] {
			continue
		}
		t.Errorf("constante %s (%q) absente de knownCapabilities (config_loader.go) — "+
			"tout config/titles/*/title.toml qui déclare %q sera REJETÉ à la validation "+
			"(« capability inconnue ») et le TITRE ENTIER ne sera pas enregistré. "+
			"Ajouter %s au set knownCapabilities.", name, constNames[name], constNames[name], name)
	}
	for _, name := range sortedKeys(known) {
		if _, ok := constNames[name]; !ok {
			t.Errorf("knownCapabilities contient la clé %s, qui n'est PAS une constante Cap* de "+
				"registry.go — constante déclarée dans un autre fichier du package ? "+
				"La déplacer dans registry.go (sinon le miroir Go<->TS, qui ne parse que "+
				"registry.go, devient aveugle sur elle).", name)
		}
	}
}

// ---------------------------------------------------------------------------
// front -> (1) : littéraux de gating côté apps/web
// ---------------------------------------------------------------------------

// TestCapabilityLiteralsInFrontAreDeclaredInGo — tout littéral passé à un gate
// d'affichage (useCapability / useCapabilityStrict / hasCapabilityIn / prop
// `capability`) doit correspondre à une constante Cap* Go.
//
// Le typage TitleCapability couvre déjà le cas nominal ; ce scan couvre ce que le
// typage ne voit pas (cast `as TitleCapability`, littéral recopié dans un endroit
// non typé) et surtout le cas où TITLE_CAPABILITIES lui-même dériverait — un gate
// sur une capability que le serveur n'émet jamais est TOUJOURS false : la feature
// disparaît de l'UI sans la moindre erreur.
func TestCapabilityLiteralsInFrontAreDeclaredInGo(t *testing.T) {
	root := capabilityRepoRoot(t)
	goCaps := extractGoCapabilities(t, capabilityRegistryPath(root))
	if len(goCaps) == 0 {
		t.Fatalf("aucune constante Cap* extraite de registry.go — parseur cassé")
	}

	literals := frontCapabilityLiterals(t, filepath.Join(root, "apps", "web", "src"))
	const minDistinctLiterals = 12 // 17 mesurés le 2026-07-26 — garde anti-scan-à-vide
	if len(literals) < minDistinctLiterals {
		t.Fatalf("seulement %d capabilities distinctes trouvées côté front (< %d attendues) — "+
			"le scan ne mord plus (arborescence apps/web déplacée, ou gates renommés) : "+
			"corriger l'extracteur, ne pas baisser le seuil", len(literals), minDistinctLiterals)
	}

	for _, value := range sortedKeys(literals) {
		if _, ok := goCaps[value]; ok {
			continue
		}
		t.Errorf("le front gate sur %q, qui n'est AUCUNE constante Cap* de registry.go "+
			"(gate toujours false → feature invisible) — occurrences :\n    %s",
			value, strings.Join(literals[value], "\n    "))
	}
}

// ---------------------------------------------------------------------------
// (1) -> (4) et (1) -> consommateurs : capabilities orphelines
// ---------------------------------------------------------------------------

// orphanCapabilityAllowlist — capabilities Go volontairement NON déclarées par un
// titre public, ou NON consommées. Chaque entrée est datée + justifiée (règle
// CLAUDE.md n°11). VIDE au 2026-07-26 : les 21 capabilities sont toutes déclarées
// par Halo Infinite ou Halo 5, et toutes lues par au moins un consommateur.
var orphanCapabilityAllowlist = map[string]string{}

// TestCapabilitiesGrantedByAPublicTitle — toute capability déclarée côté Go doit
// être accordée par AU MOINS UN titre public (built-in Infinite ou config/titles/
// */title.toml). Sinon le gate est toujours false : la feature est morte partout,
// et sa documentation dérive sans que rien ne l'exerce (règle CLAUDE.md n°11 :
// pas de feature livrée OFF « pour plus tard »).
//
// Les titres INTERNES (fixtures, synthetic_title_b) ne comptent PAS : une
// capability que seule une fixture déclare n'existe pour aucun utilisateur.
func TestCapabilitiesGrantedByAPublicTitle(t *testing.T) {
	root := capabilityRepoRoot(t)
	goCaps := extractGoCapabilities(t, capabilityRegistryPath(root))
	if len(goCaps) == 0 {
		t.Fatalf("aucune constante Cap* extraite de registry.go — parseur cassé")
	}

	reg := NewRegistryFromConfig(root, slog.New(slog.NewTextHandler(io.Discard, nil)))
	public := reg.PublicTitles()
	if len(public) < 2 {
		t.Fatalf("%d titre(s) public(s) chargé(s) depuis %s/config/titles — découverte à vide "+
			"(le test passerait sans rien vérifier) : vérifier la racine du dépôt", len(public), root)
	}

	granted := make(map[Capability][]string)
	for _, d := range public {
		for _, c := range d.Capabilities {
			granted[c] = append(granted[c], d.Slug)
		}
	}

	for _, value := range sortedKeys(goCaps) {
		if len(granted[Capability(value)]) > 0 {
			continue
		}
		if reason, ok := orphanCapabilityAllowlist[value]; ok {
			t.Logf("capability %q non accordée — exception documentée : %s", value, reason)
			continue
		}
		t.Errorf("la capability %s (%q) n'est déclarée par AUCUN titre public "+
			"(ni le descripteur built-in de registry.go, ni un config/titles/*/title.toml) — "+
			"gate toujours false, feature morte : l'accorder à un titre, supprimer la "+
			"constante, ou documenter une exception datée dans orphanCapabilityAllowlist.",
			goCaps[value], value)
	}
}

// TestCapabilitiesReferencedByAConsumer — toute capability doit être LUE quelque
// part : côté Go (référence qualifiée `title.Cap*` hors du package title lui-même)
// ou côté front (littéral de gating). Une constante déclarée, mirroirée en TS,
// validée, accordée… et lue par personne est du code mort déguisé en feature
// (anti-pattern « dead code museum »).
func TestCapabilitiesReferencedByAConsumer(t *testing.T) {
	root := capabilityRepoRoot(t)
	goCaps := extractGoCapabilities(t, capabilityRegistryPath(root))
	if len(goCaps) == 0 {
		t.Fatalf("aucune constante Cap* extraite de registry.go — parseur cassé")
	}

	goRefs := goCapabilityReferences(t, filepath.Join(root, "apps", "go-api"))
	const minGoRefs = 8 // 16 mesurés le 2026-07-26 — garde anti-scan-à-vide
	if len(goRefs) < minGoRefs {
		t.Fatalf("seulement %d constantes Cap* référencées côté Go (< %d attendues) — "+
			"le scan ne mord plus : corriger l'extracteur, ne pas baisser le seuil", len(goRefs), minGoRefs)
	}
	frontRefs := frontCapabilityLiterals(t, filepath.Join(root, "apps", "web", "src"))

	for _, value := range sortedKeys(goCaps) {
		name := goCaps[value]
		if goRefs[name] != "" || len(frontRefs[value]) > 0 {
			continue
		}
		if reason, ok := orphanCapabilityAllowlist[value]; ok {
			t.Logf("capability %q sans consommateur — exception documentée : %s", value, reason)
			continue
		}
		t.Errorf("la constante %s (%q) n'est lue par AUCUN consommateur : ni référence Go "+
			"qualifiée hors du package title (HasCapability, RequireCapability, ...), ni "+
			"littéral de gating côté apps/web. Constante morte : la brancher, la supprimer "+
			"(avec son entrée knownCapabilities et son miroir TS), ou documenter une "+
			"exception datée dans orphanCapabilityAllowlist.", name, value)
	}
}

// TestOrphanCapabilityAllowlistIsCurrent — hygiène de l'allowlist (leçon VF-6 :
// une entrée périmée est un trou latent). Chaque entrée doit désigner une
// capability Go réelle ET rester effectivement orpheline.
func TestOrphanCapabilityAllowlistIsCurrent(t *testing.T) {
	root := capabilityRepoRoot(t)
	goCaps := extractGoCapabilities(t, capabilityRegistryPath(root))

	reg := NewRegistryFromConfig(root, slog.New(slog.NewTextHandler(io.Discard, nil)))
	granted := make(map[Capability]bool)
	for _, d := range reg.PublicTitles() {
		for _, c := range d.Capabilities {
			granted[c] = true
		}
	}
	goRefs := goCapabilityReferences(t, filepath.Join(root, "apps", "go-api"))
	frontRefs := frontCapabilityLiterals(t, filepath.Join(root, "apps", "web", "src"))

	for _, value := range sortedKeys(orphanCapabilityAllowlist) {
		name, isCap := goCaps[value]
		if !isCap {
			t.Errorf("orphanCapabilityAllowlist contient %q (%s) mais aucune constante Cap* de "+
				"registry.go ne porte cette valeur — entrée périmée, la retirer",
				value, orphanCapabilityAllowlist[value])
			continue
		}
		if granted[Capability(value)] && (goRefs[name] != "" || len(frontRefs[value]) > 0) {
			t.Errorf("orphanCapabilityAllowlist contient %q (%s) mais elle est désormais accordée "+
				"par un titre ET consommée — exception périmée, la retirer (allowlist décroissante)",
				value, orphanCapabilityAllowlist[value])
		}
	}
}
