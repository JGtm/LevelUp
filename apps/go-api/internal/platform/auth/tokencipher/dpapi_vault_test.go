//go:build windows

package tokencipher

import (
	"bytes"
	"testing"
)

func TestDPAPI_ScelleDifferentDuClairEtAllerRetour(t *testing.T) {
	c, err := NewDPAPI()
	if err != nil {
		t.Fatal(err)
	}
	if c.Name() != NameDPAPI {
		t.Fatalf("nom = %q", c.Name())
	}
	clair := []byte("refresh-token-0.AXkA-secret")
	sealed, err := c.Seal(clair)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(sealed, clair) || bytes.Contains(sealed, clair) {
		t.Fatal("le scellé ne doit pas contenir le clair")
	}
	out, err := c.Open(sealed)
	if err != nil || !bytes.Equal(out, clair) {
		t.Fatalf("Open = %q, %v", out, err)
	}
}

func TestDPAPI_OuvertureRefuseUnScelleAltere(t *testing.T) {
	c, _ := NewDPAPI()
	sealed, err := c.Seal([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	sealed[len(sealed)/2] ^= 0xff
	if _, err := c.Open(sealed); err == nil {
		t.Fatal("un scellé altéré ne doit pas s'ouvrir")
	}
	if _, err := c.Open(nil); err == nil {
		t.Fatal("un scellé vide ne doit pas s'ouvrir")
	}
	if _, err := c.Open([]byte("pas un scellé dpapi")); err == nil {
		t.Fatal("des octets arbitraires ne doivent pas s'ouvrir")
	}
}

// TestDPAPI_LEntropieLieLeScelle : un scellé ne s'ouvre pas avec une autre entropie que celle du
// dépôt (c'est ce qui le lie à LevelUp parmi les programmes du même compte).
func TestDPAPI_LEntropieLieLeScelle(t *testing.T) {
	c, _ := NewDPAPI()
	sealed, err := c.Seal([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	saved := dpapiEntropy
	defer func() { dpapiEntropy = saved }()
	dpapiEntropy = []byte("autre entropie")
	if _, err := c.Open(sealed); err == nil {
		t.Fatal("une autre entropie ne doit pas ouvrir le scellé")
	}
}
