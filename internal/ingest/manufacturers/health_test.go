package manufacturers

import (
	"testing"

	"github.com/zinackes/forza-open-api/internal/health"
)

// hasRule indique si une règle de violation est présente, et sa sévérité.
func hasRule(vs []health.Violation, rule string) (health.Severity, bool) {
	for _, v := range vs {
		if v.Rule == rule {
			return v.Severity, true
		}
	}
	return "", false
}

func TestCheckRunHealthy(t *testing.T) {
	res := Result{
		Game: "fh6", CategoryFound: true, Members: 86, PagesFetched: 86,
		Manufacturers: 86, CountryResolved: 84, NoOrigin: 2, MatchedToCars: 80,
		UnknownOrigins: map[string]int{},
	}
	if vs := CheckRun(res); len(vs) != 0 {
		t.Fatalf("run sain : %d violations, want 0 (%+v)", len(vs), vs)
	}
}

func TestCheckRunNoCategory(t *testing.T) {
	vs := CheckRun(Result{Game: "fh6", CategoryFound: false, UnknownOrigins: map[string]int{}})
	if len(vs) != 1 {
		t.Fatalf("catégorie absente : %d violations, want 1", len(vs))
	}
	if sev, ok := hasRule(vs, "no_category"); !ok || sev != health.SevWarning {
		t.Errorf("no_category attendu en warning, got (%v,%v)", sev, ok)
	}
}

func TestCheckRunNoPages(t *testing.T) {
	res := Result{Game: "fh6", CategoryFound: true, Members: 86, PagesFetched: 0, UnknownOrigins: map[string]int{}}
	if sev, ok := hasRule(CheckRun(res), "no_pages"); !ok || sev != health.SevAnomaly {
		t.Errorf("no_pages attendu en anomalie, got (%v,%v)", sev, ok)
	}
}

func TestCheckRunNoManufacturers(t *testing.T) {
	res := Result{Game: "fh6", CategoryFound: true, Members: 86, PagesFetched: 86, Manufacturers: 0, UnknownOrigins: map[string]int{}}
	if sev, ok := hasRule(CheckRun(res), "no_manufacturers"); !ok || sev != health.SevAnomaly {
		t.Errorf("no_manufacturers attendu en anomalie, got (%v,%v)", sev, ok)
	}
}

func TestCheckRunNoCountryResolved(t *testing.T) {
	res := Result{Game: "fh6", CategoryFound: true, Members: 86, PagesFetched: 86, Manufacturers: 86, CountryResolved: 0, MatchedToCars: 80, UnknownOrigins: map[string]int{}}
	if sev, ok := hasRule(CheckRun(res), "no_country_resolved"); !ok || sev != health.SevAnomaly {
		t.Errorf("no_country_resolved attendu en anomalie, got (%v,%v)", sev, ok)
	}
}

func TestCheckRunManyUnknownOrigins(t *testing.T) {
	res := Result{
		Game: "fh6", CategoryFound: true, Members: 20, PagesFetched: 20,
		Manufacturers: 20, CountryResolved: 5, MatchedToCars: 18,
		UnknownOrigins: map[string]int{"origin:foo": 5, "origin:bar": 3},
	}
	if sev, ok := hasRule(CheckRun(res), "many_unknown_origins"); !ok || sev != health.SevWarning {
		t.Errorf("many_unknown_origins attendu en warning, got (%v,%v)", sev, ok)
	}
}

func TestCheckRunNoCarMatch(t *testing.T) {
	res := Result{
		Game: "fh6", CategoryFound: true, Members: 86, PagesFetched: 86,
		Manufacturers: 86, CountryResolved: 84, MatchedToCars: 0,
		UnknownOrigins: map[string]int{},
	}
	if sev, ok := hasRule(CheckRun(res), "no_car_match"); !ok || sev != health.SevWarning {
		t.Errorf("no_car_match attendu en warning, got (%v,%v)", sev, ok)
	}
}
