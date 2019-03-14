package table

import (
	"testing"

	"github.com/drcsuite/drc/goleveldb/leveldb/testutil"
)

func TestTable(t *testing.T) {
	testutil.RunSuite(t, "Table Suite")
}
