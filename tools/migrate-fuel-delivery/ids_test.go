package main

import (
	"testing"

	"dfms/apps/models"
)

func TestIdByDjango_Guards(t *testing.T) {
	resetIDCache()
	if idByDjango(nil, &models.Customer{}, 0) != 0 {
		t.Fatal("zero django id")
	}
	if idByDjango(nil, &models.Customer{}, 42) != 0 {
		t.Fatal("nil dest")
	}
	stampDjangoID(nil, &models.Customer{}, 1, 42)
	if userByDjango(nil, 7, 0) != 7 {
		t.Fatal("fallback admin")
	}
}

func TestIdByDjango_CacheHitWithoutDest(t *testing.T) {
	resetIDCache()
	rememberDjango(&models.Customer{}, 42, 99)
	if idByDjango(nil, &models.Customer{}, 42) != 99 {
		t.Fatal("cached django id")
	}
	if idByDjango(nil, &models.Product{}, 42) != 0 {
		t.Fatal("other model")
	}
}

func TestIlrAndTruckCache(t *testing.T) {
	resetIDCache()
	putILR(10, 3, 4)
	row, ok := ilrFields(10)
	if !ok || row.ProductID != 3 || row.StockStatusID != 4 {
		t.Fatalf("ilr %v %v", row, ok)
	}
	id, ok := ilrProduct(10)
	if !ok || id != 3 {
		t.Fatal("ilr product")
	}
	putTruck(models.Truck{ID: 8, PlateNumber: "T100", Trailer: "A", TrailerTwo: "B", DjangoID: 22})
	if idByDjango(nil, &models.Truck{}, 22) != 8 {
		t.Fatal("truck django")
	}
	plates, ok := truckPlates(8)
	if !ok || plates.PlateNumber != "T100" {
		t.Fatal("truck plates")
	}
}

func TestFirstStatusNilDest(t *testing.T) {
	if firstStatus(nil) != 0 {
		t.Fatal("nil dest")
	}
}

func TestTankKey(t *testing.T) {
	if tankKey(0, "T100", 1) != "0:T100:1" {
		t.Fatal("null truck")
	}
	if tankKey(8, "T100", 2) == tankKey(0, "T100", 2) {
		t.Fatal("linked vs retired tank")
	}
	if derefUint(nil) != 0 {
		t.Fatal("nil truck id")
	}
	v := uint(9)
	if derefUint(&v) != 9 {
		t.Fatal("set truck id")
	}
}

func TestLookupProduct_UsesSnap(t *testing.T) {
	resetIDCache()
	copyCache.mu.Lock()
	copyCache.productsDjango[7] = namedRow{ID: 3, Code: "1001", Name: "PMS"}
	copyCache.productsCode["1002"] = namedRow{ID: 4, Code: "1002", Name: "AGO"}
	copyCache.mu.Unlock()
	id, code, name := lookupProduct(nil, 7, "x", "y")
	if id != 3 || code != "1001" || name != "PMS" {
		t.Fatalf("django snap %d %s %s", id, code, name)
	}
	id, code, name = lookupProduct(nil, 0, "1002", "diesel")
	if id != 4 || code != "1002" || name != "AGO" {
		t.Fatalf("code snap %d %s %s", id, code, name)
	}
}
