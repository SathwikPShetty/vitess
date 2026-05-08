// go/vt/vtgate/vindexes/yearmonth.go
package vindexes

import (
	"context"
	"fmt"
	"time"

	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/key"
)

type YearMonth struct {
	name string
}

var (
	_ SingleColumn = (*YearMonth)(nil)
	_ Hashing      = (*YearMonth)(nil)
)

func NewYearMonth(name string, m map[string]string) (Vindex, error) {
	return &YearMonth{name: name}, nil
}

func (v *YearMonth) String() string {
	return v.name
}

func (v *YearMonth) Cost() int {
	return 1
}

func (v *YearMonth) IsUnique() bool {
	return true
}

func (v *YearMonth) NeedsVCursor() bool {
	return false
}

// func (v *YearMonth) Hash(id sqltypes.Value) ([]byte, error) {
// 	return v.computeKSID(id)
// }

func (v *YearMonth) Hash(id sqltypes.Value) ([]byte, error) {
	ksid, err := v.computeKSID(id)
	if err != nil {
		return nil, err
	}

	fmt.Printf(
		"YEARMONTH INPUT=%s OUTPUT=%08b\n",
		id.ToString(),
		ksid[0],
	)

	return ksid, nil
}

func (v *YearMonth) Verify(
	ctx context.Context,
	vcursor VCursor,
	ids []sqltypes.Value,
	ksids [][]byte,
) ([]bool, error) {
	out := make([]bool, len(ids))

	for i, id := range ids {
		ksid, err := v.computeKSID(id)
		if err != nil {
			return nil, err
		}
		out[i] = string(ksid) == string(ksids[i])
	}

	return out, nil
}

func (v *YearMonth) Map(
	ctx context.Context,
	vcursor VCursor,
	ids []sqltypes.Value,
) ([]key.ShardDestination, error) {
	destinations := make([]key.ShardDestination, len(ids))

	for i, id := range ids {
		ksid, err := v.computeKSID(id)
		if err != nil {
			return nil, err
		}
		destinations[i] = key.DestinationKeyspaceID(ksid)
	}

	return destinations, nil
}

func (v *YearMonth) computeKSID(id sqltypes.Value) ([]byte, error) {
	ts := id.ToString()

	// Example input:
	// 2024-10-01T12:00:00Z

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// fallback for plain date
		t, err = time.Parse("2006-01-02", ts)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format: %v", err)
		}
	}

	year := (t.Year()-2025)*12 + int(t.Month()) - 1

	// upper 4 bits = year
	// lower 4 bits = month

	value := byte(year)

	return []byte{value}, nil
}

func testYearMonth() {
	v := &YearMonth{}

	id := sqltypes.NewVarChar("2026-02-15T12:00:00Z")

	ksid, _ := v.Hash(id)

	fmt.Printf("TEST YEARMONTH = %08b\n", ksid[0])
}

func init() {
	Register("yearmonth", NewYearMonth)

	testYearMonth()
}
