package etsy

import (
	"testing"
	"time"

	"github.com/cooldarkdryplace/lowstock"

	"github.com/google/go-cmp/cmp"
)

func TestToUpdate(t *testing.T) {
	var (
		listingID int64 = 1
		state           = "test state"
		userID    int64 = 121314
		quantity  int64 = 100
		title           = "test title"
		sku             = []string{"SKU#1"}
		createdAt       = time.Now().Add(-2 * time.Hour).Unix()
		updatedAt       = time.Now().Unix()
		shopName        = "TestShop"
	)

	expectedUpdate := lowstock.Update{
		ListingID:       listingID,
		State:           state,
		Title:           title,
		ShopName:        shopName,
		UserID:          userID,
		Quantity:        quantity,
		CreationTSZ:     createdAt,
		LastModifiedTSZ: updatedAt,
	}

	info := listingInfo{
		ListingID:       listingID,
		State:           state,
		UserID:          userID,
		Quantity:        quantity,
		Title:           title,
		SKU:             sku,
		CreationTSZ:     createdAt,
		LastModifiedTSZ: updatedAt,
		Shop:            shop{Name: shopName},
	}

	actualUpdate := toLowstockUpdate(info)

	if diff := cmp.Diff(expectedUpdate, actualUpdate); diff != "" {
		t.Errorf("Updates do not match:\n%s", diff)
	}
}

func TestToUpdates(t *testing.T) {
	expectedUpdates := []lowstock.Update{
		lowstock.Update{ListingID: 1},
		lowstock.Update{ListingID: 2},
		lowstock.Update{ListingID: 3},
	}

	infos := []listingInfo{
		listingInfo{ListingID: 1},
		listingInfo{ListingID: 2},
		listingInfo{ListingID: 3},
	}

	actualUpdates := toLowstockUpdates(infos)

	if diff := cmp.Diff(expectedUpdates, actualUpdates); diff != "" {
		t.Errorf("Updates do not match:\n%s", diff)
	}
}
