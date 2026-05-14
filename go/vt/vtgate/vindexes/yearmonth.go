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
	// NULL partition_key is a data-quality bug at the source. Reject loudly
	// rather than silently routing the row to a catch-all shard, so the
	// caller fixes the dump / app code instead of accumulating orphan rows
	// in pmin that no query can locate by date.
	if id.IsNull() {
		return nil, fmt.Errorf(
			"yearmonth: partition_key is NULL",
		)
	}

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

	// Month offset from Jan 2025. One byte (0..255) covers Jan 2025 .. Dec 2045.
	yearMonth := (t.Year()-2025)*12 + int(t.Month()) - 1

	// Pre-2025 → route to pmin (catch-all for legacy data). The pmin shard
	// (-0005 keyrange) owns byte values 0x00..0x04, so emitting 0x00 here
	// lands the row inside pmin's range. This handles legitimate historical
	// data that predates the partitioning epoch.
	if yearMonth < 0 {
		return []byte{0x00}, nil
	}

	// Post-Dec-2045 (offset > 255): reject explicitly. We could clamp to
	// 0xFF (pmax) but that conflates "far future" with "currently active
	// future months", which makes sliding-window operations ambiguous.
	// If we ever need to support dates past 2045 we should widen the byte
	// to a 2-byte month offset rather than overload pmax.
	if yearMonth > 0xFF {
		return nil, fmt.Errorf(
			"yearmonth: %04d-%02d is beyond the supported epoch (Jan 2025 .. Dec 2045); offset=%d > 255",
			t.Year(), t.Month(), yearMonth,
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
