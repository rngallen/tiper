package orders

import (
	"context"
	"testing"
)

func TestCreateCompSchema(t *testing.T) {
	s := createCompSchema{IloID: "  abc  ", BadgeID: " badge "}
	s.Sanitize()
	if s.IloID != "abc" || s.BadgeID != "badge" {
		t.Fatalf("sanitize: %+v", s)
	}
	if err := s.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	empty := createCompSchema{}
	if err := empty.Validate(context.Background()); err == nil {
		t.Fatal("expected ILO required")
	}
}

func TestSaveCompLineSeals(t *testing.T) {
	ln := saveCompLineSchema{ID: "01HZX", TopSeal: " aa ", DipSeal: "aa", BottomSeal: "bb"}
	ln.Sanitize()
	if ln.TopSeal != "AA" || ln.DipSeal != "AA" {
		t.Fatalf("seals not uppercased: %+v", ln)
	}
	if err := ln.Validate(context.Background()); err == nil {
		t.Fatal("expected duplicate seal error")
	}
	ln.DipSeal = "CC"
	if err := ln.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAmendmentKind(t *testing.T) {
	s := createAmendmentSchema{Kind: "nope", IloID: "01HZX"}
	if err := s.Validate(context.Background()); err == nil {
		t.Fatal("expected unknown kind")
	}
	s.Kind = "extend"
	if err := s.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}
