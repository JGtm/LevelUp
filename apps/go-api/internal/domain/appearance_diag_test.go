package domain

import "testing"

func TestDiagnoseServiceTag(t *testing.T) {
	present := DiagnoseServiceTag("JGTM")
	if present.Verdict != AppearanceVerdictOK || present.Detail != AppearanceDetailServiceTagPresent || present.ServedFrom != AppearanceServedFromLive {
		t.Fatalf("present: %+v, attendu ok/service_tag_present/live", present)
	}
	absent := DiagnoseServiceTag("   ")
	if absent.Verdict != AppearanceVerdictTransient || absent.Detail != AppearanceDetailNoServiceTag || absent.ServedFrom != AppearanceServedFromCarry {
		t.Fatalf("absent: %+v, attendu transient/no_service_tag/carry", absent)
	}
}
