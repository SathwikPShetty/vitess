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

func (v *YearMonth) Hash(id sqltypes.Value) ([]byte, error) {
	return v.computeKSID(id)
}

// func (v *YearMonth) Hash(id sqltypes.Value) ([]byte, error) {

// 	ksid, err := v.computeKSID(id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	fmt.Printf(
// 		"YEARMONTH INPUT=%s OUTPUT=%08b\n",
// 		id.ToString(),
// 		ksid[0],
// 	)

// 	return ksid, nil
// }

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

	var t time.Time
	var err error

	// MySQL DATETIME(6)
	t, err = time.Parse(
		"2006-01-02 15:04:05.999999",
		ts,
	)

	if err != nil {

		// MySQL DATETIME
		t, err = time.Parse(
			"2006-01-02 15:04:05",
			ts,
		)

		if err != nil {

			// MySQL DATE
			t, err = time.Parse(
				"2006-01-02",
				ts,
			)

			if err != nil {
				return nil, fmt.Errorf(
					"invalid datetime format: %v",
					err,
				)
			}
		}
	}

	// Month offset from Jan 2025
	yearMonth := (t.Year()-2025)*12 + int(t.Month()) - 1

	if yearMonth < 0 {
		return nil, fmt.Errorf(
			"year must be >= 2025",
		)
	}

	// 1-byte KSID
	return []byte{
		byte(yearMonth),
	}, nil
}

// func testYearMonth() {

// 	v := &YearMonth{}

// 	tests := []string{
// 		"2025-01-01 00:00:00",
// 		"2025-02-01 00:00:00",
// 		"2025-03-27 10:30:47",
// 		"2025-12-31 23:59:59",
// 		"2026-01-01 00:00:00",
// 		"2027-06-15 12:00:00",
// 		"2030-01-01 00:00:00",
// 		"2025-03-27 10:30:47.123456",
// 	}

// 	fmt.Println("==== YEARMONTH TEST ====")

// 	for _, ts := range tests {

// 		id := sqltypes.NewVarChar(ts)

// 		ksid, err := v.Hash(id)

// 		if err != nil {

// 			fmt.Printf(
// 				"INPUT=%s ERROR=%v\n",
// 				ts,
// 				err,
// 			)

// 			continue
// 		}

// 		value := uint8(ksid[0])

// 		fmt.Printf(
// 			"INPUT=%s KSID=%v MONTH_INDEX=%d BINARY=%08b\n",
// 			ts,
// 			ksid,
// 			value,
// 			value,
// 		)
// 	}

// 	fmt.Println("========================")
// }

func init() {
	Register("yearmonth", NewYearMonth)

	// testYearMonth()
}
