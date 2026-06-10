package cars_test

import (
	"testing"

	"github.com/zinackes/forza-open-api/internal/health"
	"github.com/zinackes/forza-open-api/internal/ingest/cars"
)

func anomalyRules(vs []health.Violation) map[string]bool {
	out := map[string]bool{}
	for _, v := range vs {
		if v.Severity == health.SevAnomaly {
			out[v.Rule] = true
		}
	}
	return out
}

// TestCheckRunHealthyCatalogue : un catalogue plein, presque tout normalisé, sans
// anomalie bloquante → run sain.
func TestCheckRunHealthyCatalogue(t *testing.T) {
	rep := cars.Report{Rows: 700, Imported: 695, Distinct: 690}
	if vs := cars.CheckRun(rep, cars.ValidationReport{}, "fh6"); len(vs) != 0 {
		t.Errorf("CheckRun (catalogue sain) = %+v, want 0 violation", vs)
	}
}

// TestCheckRunFlagsEmptyList : page liste vide (template {{CarListStats…}} renommé)
// → anomalie zero_rows.
func TestCheckRunFlagsEmptyList(t *testing.T) {
	vs := cars.CheckRun(cars.Report{Rows: 0}, cars.ValidationReport{}, "fh6")
	if !anomalyRules(vs)["zero_rows"] {
		t.Errorf("CheckRun (0 ligne) = %+v, want anomalie zero_rows", vs)
	}
}

// TestCheckRunFlagsCollapse : count effondré + fort taux de lignes écartées
// (changement de structure du template) → count_floor + skip_ratio.
func TestCheckRunFlagsCollapse(t *testing.T) {
	rep := cars.Report{Rows: 100, Imported: 40, Distinct: 38}
	got := anomalyRules(cars.CheckRun(rep, cars.ValidationReport{}, "fh6"))
	for _, want := range []string{"count_floor", "skip_ratio"} {
		if !got[want] {
			t.Errorf("anomalie %q absente, got %v", want, got)
		}
	}
}

// TestCheckRunFlagsBlocking : une entrée violant une contrainte DB (détectée par
// Validate) remonte en anomalie blocking.
func TestCheckRunFlagsBlocking(t *testing.T) {
	rep := cars.Report{Rows: 700, Imported: 700, Distinct: 700}
	vrep := cars.ValidationReport{Anomalies: []cars.Anomaly{
		{CarID: "x", Severity: cars.SevBlocking, Rule: "pi_range", Detail: "pi=42 hors [100,999]"},
	}}
	if !anomalyRules(cars.CheckRun(rep, vrep, "fh6"))["blocking"] {
		t.Errorf("CheckRun (1 bloquante) = want anomalie blocking")
	}
}
