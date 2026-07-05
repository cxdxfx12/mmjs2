package freight

import (
	"testing"
	"yunfei/internal/excel"
	"yunfei/internal/rules"
)

func TestCalcAvgWeightMarkupFast_RoundModes(t *testing.T) {
	rows := []excel.RowData{{Customer: "Acme", Weight: 1.1}, {Customer: "Acme", Weight: 1.2}}
	customerRules := map[string]*rules.AvgWeightRule{
		"Acme": {
			ScopeType:    "customer",
			CustomerName: "Acme",
			BaseWeight:   1.0,
			WeightLimit:  0,
			StepWeight:   0.1,
			StepPrice:    1.0,
			RoundMode:    "ceil",
			IsEnabled:    1,
		},
	}

	results, total := CalcAvgWeightMarkupFast(rows, customerRules, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if total != 4.0 {
		t.Fatalf("expected total markup 4.00, got %.2f", total)
	}
	if results[0].Steps != 2 {
		t.Fatalf("expected 2 steps, got %d", results[0].Steps)
	}
	if results[0].PerItemMarkup != 2.0 {
		t.Fatalf("expected per item markup 2.00, got %.2f", results[0].PerItemMarkup)
	}
}

func TestCalcAvgWeightMarkupFast_FloorRoundMode(t *testing.T) {
	rows := []excel.RowData{{Customer: "Acme", Weight: 1.15}, {Customer: "Acme", Weight: 1.15}}
	customerRules := map[string]*rules.AvgWeightRule{
		"Acme": {
			ScopeType:    "customer",
			CustomerName: "Acme",
			BaseWeight:   1.0,
			WeightLimit:  0,
			StepWeight:   0.1,
			StepPrice:    1.0,
			RoundMode:    "floor",
			IsEnabled:    1,
		},
	}

	results, total := CalcAvgWeightMarkupFast(rows, customerRules, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if total != 2.0 {
		t.Fatalf("expected total markup 2.00 with floor rounding, got %.2f", total)
	}
	if results[0].Steps != 1 {
		t.Fatalf("expected 1 step with floor rounding, got %d", results[0].Steps)
	}
}

func TestCalcAvgWeightMarkupFast_MaxMarkupCap(t *testing.T) {
	rows := []excel.RowData{{Customer: "Acme", Weight: 2.0}, {Customer: "Acme", Weight: 2.0}}
	customerRules := map[string]*rules.AvgWeightRule{
		"Acme": {
			ScopeType:    "customer",
			CustomerName: "Acme",
			BaseWeight:   1.0,
			WeightLimit:  0,
			StepWeight:   0.1,
			StepPrice:    1.0,
			MaxMarkup:    3.0,
			RoundMode:    "ceil",
			IsEnabled:    1,
		},
	}

	results, total := CalcAvgWeightMarkupFast(rows, customerRules, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].PerItemMarkup != 3.0 {
		t.Fatalf("expected per item cap 3.00, got %.2f", results[0].PerItemMarkup)
	}
	if total != 6.0 {
		t.Fatalf("expected total markup 6.00 with cap, got %.2f", total)
	}
}
