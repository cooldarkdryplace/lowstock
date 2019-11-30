package lowstock

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStoredUserCanBeRead(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "lowstock_test_users.db")
	defer os.Remove(dbFile)

	db, err := NewBoltStorage(dbFile)
	if err != nil {
		t.Fatalf("Failed to init DB: %s", err)
	}
	defer db.Close()

	expectedUser := User{}

	ctx := context.Background()
	if err := db.SaveUser(ctx, expectedUser); err != nil {
		t.Errorf("Failed to save user: %s", err)
	}

	actualUser, err := db.User(ctx, expectedUser.EtsyUserID)
	if err != nil {
		t.Errorf("Failed to retrieve user: %s", err)
	}

	if diff := cmp.Diff(actualUser, expectedUser); diff != "" {
		t.Errorf("Users are different:\n%s", diff)
	}
}

func TestStoredTokenDetailsCanBeRead(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "lowstock_test_users.db")
	defer os.Remove(dbFile)

	db, err := NewBoltStorage(dbFile)
	if err != nil {
		t.Fatalf("Failed to init DB: %s", err)
	}
	defer db.Close()

	expectedDetails := TokenDetails{}

	ctx := context.Background()
	if err := db.SaveTokenDetails(ctx, expectedDetails); err != nil {
		t.Errorf("Failed to save token details: %s", err)
	}

	actualDetails, err := db.TokenDetails(ctx, expectedDetails.ID)
	if err != nil {
		t.Errorf("Failed to retrieve token details: %s", err)
	}

	if diff := cmp.Diff(actualDetails, expectedDetails); diff != "" {
		t.Errorf("Token details are different:\n%s", diff)
	}
}
