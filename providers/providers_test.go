package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	u "github.com/ipfs/go-ipfs-util"
	peer "github.com/libp2p/go-libp2p-peer"
)

func TestProviderManager(t *testing.T) {
	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())
	a := cid.NewCidV0(u.Hash([]byte("test")))
	p.AddProvider(ctx, a, peer.ID("testingprovider"))
	resp := p.GetProviders(ctx, a)
	if len(resp) != 1 {
		t.Fatal("Could not retrieve provider.")
	}
	p.proc.Close()
}

func TestProvidersDatastore(t *testing.T) {
	old := lruCacheSize
	lruCacheSize = 10
	defer func() { lruCacheSize = old }()

	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())
	defer p.proc.Close()

	friend := peer.ID("friend")
	var cids []*cid.Cid
	for i := 0; i < 100; i++ {
		c := cid.NewCidV0(u.Hash([]byte(fmt.Sprint(i))))
		cids = append(cids, c)
		p.AddProvider(ctx, c, friend)
	}

	for _, c := range cids {
		resp := p.GetProviders(ctx, c)
		if len(resp) != 1 {
			t.Fatal("Could not retrieve provider.")
		}
		if resp[0] != friend {
			t.Fatal("expected provider to be 'friend'")
		}
	}
}

func TestProvidersSerialization(t *testing.T) {
	dstore := ds.NewMapDatastore()

	k := cid.NewCidV0(u.Hash(([]byte("my key!"))))
	p1 := peer.ID("peer one")
	p2 := peer.ID("peer two")
	pt1 := time.Now()
	pt2 := pt1.Add(time.Hour)

	err := writeProviderEntry(dstore, k, p1, pt1)
	if err != nil {
		t.Fatal(err)
	}

	err = writeProviderEntry(dstore, k, p2, pt2)
	if err != nil {
		t.Fatal(err)
	}

	pset, err := loadProvSet(dstore, k)
	if err != nil {
		t.Fatal(err)
	}

	lt1, ok := pset.set[p1]
	if !ok {
		t.Fatal("failed to load set correctly")
	}

	if pt1 != lt1 {
		t.Fatal("time wasnt serialized correctly")
	}

	lt2, ok := pset.set[p2]
	if !ok {
		t.Fatal("failed to load set correctly")
	}

	if pt2 != lt2 {
		t.Fatal("time wasnt serialized correctly")
	}
}

func TestProvidesExpire(t *testing.T) {
	pval := ProviderValidity
	cleanup := defaultCleanupInterval
	ProviderValidity = time.Second / 2
	defaultCleanupInterval = time.Second / 2
	defer func() {
		ProviderValidity = pval
		defaultCleanupInterval = cleanup
	}()

	ctx := context.Background()
	mid := peer.ID("testing")
	p := NewProviderManager(ctx, mid, ds.NewMapDatastore())

	peers := []peer.ID{"a", "b"}
	var cids []*cid.Cid
	for i := 0; i < 10; i++ {
		c := cid.NewCidV0(u.Hash([]byte(fmt.Sprint(i))))
		cids = append(cids, c)
		p.AddProvider(ctx, c, peers[0])
		p.AddProvider(ctx, c, peers[1])
	}

	for i := 0; i < 10; i++ {
		out := p.GetProviders(ctx, cids[i])
		if len(out) != 2 {
			t.Fatal("expected providers to still be there")
		}
	}

	time.Sleep(time.Second)
	for i := 0; i < 10; i++ {
		out := p.GetProviders(ctx, cids[i])
		if len(out) > 0 {
			t.Fatal("expected providers to be cleaned up, got: ", out)
		}
	}

	if p.providers.Len() != 0 {
		t.Fatal("providers map not cleaned up")
	}

	allprovs, err := p.getAllProvKeys()
	if err != nil {
		t.Fatal(err)
	}

	if len(allprovs) != 0 {
		t.Fatal("expected everything to be cleaned out of the datastore")
	}
}
