package auth

import (
	"context"
	"testing"
)

func TestPhoneNumberRejectsLetters(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		phone   string
		wantErr bool
	}{
		{"", false},
		{"255711223344", false},
		{"abc", true},
		{"255-711-223", true},
		{"+255711223344", true},
		{"0255711223344", true},
		{"123", true}, // too short
	}
	for _, tc := range cases {
		p := profileInSchema{PhoneNumber: tc.phone}
		err := p.Validate(ctx)
		if tc.wantErr && err == nil {
			t.Fatalf("phone %q: expected error", tc.phone)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("phone %q: unexpected error: %v", tc.phone, err)
		}
	}
}

func TestUserCreatePhoneNested(t *testing.T) {
	ctx := context.Background()
	u := userCreateSchema{
		Email: "someone@example.com",
	}
	u.FirstName = "Jane"
	u.LastName = "Doe"
	u.PhoneNumber = "not-a-phone"
	if err := u.Validate(ctx); err == nil {
		t.Fatal("expected nested phone validation to fail")
	}
}
