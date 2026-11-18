package mapfondsidecar

import "testing"

// Le nom des fichiers du relief se fabrique et se relit par le même couple de fonctions ; les
// données et les autres images (fond, plan) n'en sont pas une image de relief.
func TestNommageRelief(t *testing.T) {
	for _, ext := range []string{".png", ".webp"} {
		nom := NomImageRelief("ridgeline", ext)
		if nom != "ridgeline.relief"+ext {
			t.Errorf("NomImageRelief = %q", nom)
		}
		if cle, ok := CleDeImageRelief(nom); !ok || cle != "ridgeline" {
			t.Errorf("CleDeImageRelief(%q) = %q, %v", nom, cle, ok)
		}
	}
	if got := NomDonneesRelief("ridgeline"); got != "ridgeline.relief.json.gz" {
		t.Errorf("NomDonneesRelief = %q", got)
	}
	// Une clé de carte Forge (UUID, tirets, points possibles) se relit entière.
	if cle, ok := CleDeImageRelief(NomImageRelief("1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48", ".webp")); !ok || cle != "1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48" {
		t.Errorf("clé Forge = %q, %v", cle, ok)
	}
	for _, nom := range []string{"ridgeline.png", "ridgeline.webp", "ridgeline.plan.webp", "ridgeline.relief.json.gz",
		".relief.png", "ridgeline.relief.gif", "ridgeline.relief", "ridgeline.reliefs.png", "ridgeline.relief.relief.txt"} {
		if cle, ok := CleDeImageRelief(nom); ok {
			t.Errorf("CleDeImageRelief(%q) = %q, attendu aucune clé", nom, cle)
		}
	}
	// Le relief n'est pas pris pour une image plan, ni le plan pour un relief.
	if _, ok := CleDeImagePlan(NomImageRelief("ridgeline", ".webp")); ok {
		t.Error("une image de relief est prise pour une image plan")
	}
	if _, ok := CleDeImageRelief(NomImagePlan("ridgeline", ".webp")); ok {
		t.Error("une image plan est prise pour une image de relief")
	}
	for nom, attendu := range map[string]bool{
		"ridgeline.relief.json.gz": true, "ridgeline.json": false, ".relief.json": false, "ridgeline.relief.json": false,
		"ridgeline.relief.webp": false, "ridgeline.plan.json": false,
	} {
		if EstDonneesRelief(nom) != attendu {
			t.Errorf("EstDonneesRelief(%q) = %v, attendu %v", nom, !attendu, attendu)
		}
	}
	if !EstDonneesRelief(NomDonneesRelief("ridgeline")) {
		t.Error("NomDonneesRelief n'est pas reconnu par EstDonneesRelief")
	}
}
