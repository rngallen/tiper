package response

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestDefaultWriteMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method, want string
		status       int
	}{
		{fiber.MethodGet, "", 200},
		{fiber.MethodPost, "Updated successfully", 200},
		{fiber.MethodPost, "Created successfully", 201},
		{fiber.MethodPut, "Updated successfully", 200},
		{fiber.MethodPatch, "Updated successfully", 200},
		{fiber.MethodDelete, "Deleted successfully", 200},
	}
	for _, tc := range cases {
		if got := defaultWriteMessage(tc.method, tc.status); got != tc.want {
			t.Fatalf("%s %d: got %q want %q", tc.method, tc.status, got, tc.want)
		}
	}
}

