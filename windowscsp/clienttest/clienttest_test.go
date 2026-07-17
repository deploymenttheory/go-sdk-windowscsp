package clienttest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

func TestInMemoryCRUD(t *testing.T) {
	m := New()
	ctx := context.Background()

	if _, err := m.Get(ctx, "./missing"); !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("Get missing = %v", err)
	}

	if err := m.Add(ctx, "./Vendor/MSFT/Demo/X", client.Int(5)); err != nil {
		t.Fatal(err)
	}
	v, err := m.Get(ctx, "./Vendor/MSFT/Demo/X")
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := v.Int(); n != 5 {
		t.Fatalf("value = %+v", v)
	}

	if err := m.Replace(ctx, "./Vendor/MSFT/Demo/X", client.Int(7)); err != nil {
		t.Fatal(err)
	}
	v, _ = m.Get(ctx, "./Vendor/MSFT/Demo/X")
	if n, _ := v.Int(); n != 7 {
		t.Fatalf("after replace = %+v", v)
	}

	if err := m.Delete(ctx, "./Vendor/MSFT/Demo/X"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(ctx, "./Vendor/MSFT/Demo/X"); !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("after delete = %v", err)
	}
	if err := m.Delete(ctx, "./Vendor/MSFT/Demo/X"); !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("double delete = %v", err)
	}
}

func TestInMemoryListAndSubtreeDelete(t *testing.T) {
	m := New()
	ctx := context.Background()
	m.Seed("./Vendor/MSFT/VPNv2/P1/Server", client.Chr("a"))
	m.Seed("./Vendor/MSFT/VPNv2/P1/Port", client.Int(443))
	m.Seed("./Vendor/MSFT/VPNv2/P2/Server", client.Chr("b"))

	names, err := m.List(ctx, "./Vendor/MSFT/VPNv2")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"P1", "P2"}) {
		t.Fatalf("List = %v", names)
	}

	if err := m.Delete(ctx, "./Vendor/MSFT/VPNv2/P1"); err != nil {
		t.Fatal(err)
	}
	names, _ = m.List(ctx, "./Vendor/MSFT/VPNv2")
	if !reflect.DeepEqual(names, []string{"P2"}) {
		t.Fatalf("List after subtree delete = %v", names)
	}
}

func TestInMemoryExec(t *testing.T) {
	m := New()
	if err := m.Exec(context.Background(), "./Vendor/MSFT/Reboot/RebootNow", client.Null()); err != nil {
		t.Fatal(err)
	}
	if len(m.Executed) != 1 || m.Executed[0].URI != "./Vendor/MSFT/Reboot/RebootNow" {
		t.Fatalf("Executed = %+v", m.Executed)
	}
}
