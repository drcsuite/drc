package memdb

import (
	"testing"

	"github.com/drcsuite/drc/goleveldb/leveldb/testutil"
)

func TestMemDB(t *testing.T) {
	testutil.RunSuite(t, "MemDB Suite")
}
