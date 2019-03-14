package leveldb

import (
	"testing"

	"github.com/drcsuite/drc/goleveldb/leveldb/testutil"
)

func TestLevelDB(t *testing.T) {
	testutil.RunSuite(t, "LevelDB Suite")
}
