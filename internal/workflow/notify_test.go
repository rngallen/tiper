package workflow

import (
	"testing"

	"dfms/apps/models"
	"dfms/pkg/types"
)

func TestMergeOutcomeRecipients_CreatorComplete(t *testing.T) {
	creator := models.User{ID: 1, Email: "init@example.com"}
	fyi := []models.User{{ID: 2, Email: "watch@example.com"}}
	users := mergeOutcomeRecipients(
		types.NotifyComplete, types.AmendCreator,
		&creator, nil, fyi, nil,
	)
	if len(users) != 2 {
		t.Fatalf("users: %d", len(users))
	}
	if users[0].ID != 1 || users[1].ID != 2 {
		t.Fatalf("order/ids: %+v", users)
	}
}

func TestMergeOutcomeRecipients_PoolCompleteIncludesPool(t *testing.T) {
	creator := models.User{ID: 1, Email: "init@example.com"}
	pool := []models.User{
		{ID: 3, Email: "a@example.com"},
		{ID: 1, Email: "init@example.com"},
	}
	users := mergeOutcomeRecipients(
		types.NotifyComplete, types.AmendPool,
		&creator, pool, nil, nil,
	)
	if len(users) != 2 {
		t.Fatalf("users: %+v", users)
	}
}

func TestMergeOutcomeRecipients_SubmitIsFYIOnly(t *testing.T) {
	creator := models.User{ID: 1, Email: "init@example.com"}
	pool := []models.User{{ID: 3, Email: "a@example.com"}}
	fyi := []models.User{{ID: 2, Email: "watch@example.com"}, {ID: 9, Email: "op@example.com"}}
	exclude := []models.User{{ID: 9, Email: "op@example.com"}}
	users := mergeOutcomeRecipients(
		types.NotifySubmit, types.AmendPool,
		&creator, pool, fyi, exclude,
	)
	if len(users) != 1 || users[0].ID != 2 {
		t.Fatalf("submit FYI after exclude: %+v", users)
	}
}

func TestMergeOutcomeRecipients_SubmitCreatorModeNoWatchers(t *testing.T) {
	creator := models.User{ID: 1, Email: "init@example.com"}
	users := mergeOutcomeRecipients(
		types.NotifySubmit, types.AmendCreator,
		&creator, nil, nil, nil,
	)
	if len(users) != 0 {
		t.Fatalf("creator submit with no FYI: users=%v", users)
	}
}
