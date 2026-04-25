package main

import "testing"

func TestIsItemJSON_ItemWithCommonData(t *testing.T) {
	raw := `{"CommonData":{"Quality":"Legendary","Type":"ArmorCoating"}}`
	if !isItemJSON(raw) {
		t.Error("attendu true pour JSON item avec CommonData")
	}
}

func TestIsItemJSON_TrackWithRanks(t *testing.T) {
	raw := `{"Ranks":[{"Rank":1}],"XpPerRank":1000}`
	if isItemJSON(raw) {
		t.Error("attendu false pour JSON Reward Track avec Ranks[]")
	}
}

func TestIsItemJSON_MissingCommonData(t *testing.T) {
	raw := `{"Name":"test"}`
	if isItemJSON(raw) {
		t.Error("attendu false pour JSON sans CommonData")
	}
}

func TestIsItemJSON_InvalidJSON(t *testing.T) {
	if isItemJSON("{not valid json}") {
		t.Error("attendu false pour JSON invalide")
	}
}

func TestIsItemJSON_EmptyString(t *testing.T) {
	if isItemJSON("") {
		t.Error("attendu false pour string vide")
	}
}
