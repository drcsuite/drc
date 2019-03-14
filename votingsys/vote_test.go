package votingsys

import (
	"fmt"
	"testing"
)

func TestEstimateScale(t *testing.T) {
	fmt.Println(EstimateScale(200, 190))
}
