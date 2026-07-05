package rules

import (
	"testing"
	"time"
)

func TestIsCampaignActive(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	today := now.Format("2006-01-02")

	if !isCampaignActive(FreightRule{CampaignStart: yesterday, CampaignEnd: tomorrow}) {
		t.Fatal("expected campaign active when within start/end dates")
	}
	if isCampaignActive(FreightRule{CampaignStart: tomorrow, CampaignEnd: ""}) {
		t.Fatal("expected campaign inactive when start date is in the future")
	}
	if isCampaignActive(FreightRule{CampaignStart: "", CampaignEnd: yesterday}) {
		t.Fatal("expected campaign inactive when end date is in the past")
	}
	if !isCampaignActive(FreightRule{CampaignStart: "", CampaignEnd: today}) {
		t.Fatal("expected campaign active on the end date")
	}
	if !isCampaignActive(FreightRule{CampaignStart: "bad-date", CampaignEnd: "bad-date"}) {
		t.Fatal("expected campaign to remain active when date fields are invalid")
	}
}
