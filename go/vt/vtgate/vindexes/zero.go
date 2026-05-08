// go/vt/vtgate/vindexes/zero.go
package vindexes

import (
	"context"
	"fmt"

	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/key"
)

type Zero struct {
	name string
}

var (
	_ SingleColumn = (*Zero)(nil)
	_ Hashing      = (*Zero)(nil)
)

func NewZero(name string, m map[string]string) (Vindex, error) {
	return &Zero{name: name}, nil
}

func (v *Zero) String() string {
	return v.name
}

func (v *Zero) Cost() int {
	return 1
}

func (v *Zero) IsUnique() bool {
	return true
}

func (v *Zero) NeedsVCursor() bool {
	return false
}

// func (v *Zero) Hash(id sqltypes.Value) ([]byte, error) {
// 	return []byte{0x00}, nil
// }

func (v *Zero) Hash(id sqltypes.Value) ([]byte, error) {
	ksid := []byte{0x00}

	fmt.Printf(
		"ZERO INPUT=%s OUTPUT=%08b\n",
		id.ToString(),
		ksid[0],
	)

	return ksid, nil
}

func (v *Zero) Verify(
	ctx context.Context,
	vcursor VCursor,
	ids []sqltypes.Value,
	ksids [][]byte,
) ([]bool, error) {
	out := make([]bool, len(ids))

	for i := range ids {
		out[i] = len(ksids[i]) == 1 && ksids[i][0] == 0x00
	}

	return out, nil
}

func (v *Zero) Map(
	ctx context.Context,
	vcursor VCursor,
	ids []sqltypes.Value,
) ([]key.ShardDestination, error) {
	destinations := make([]key.ShardDestination, len(ids))

	for i := range ids {
		ksid := []byte{0x00}

		destinations[i] = key.DestinationKeyspaceID(ksid)
	}

	return destinations, nil
}

func testZero() {
	v := &Zero{}

	id := sqltypes.NewVarChar("abc")

	ksid, _ := v.Hash(id)

	fmt.Printf("TEST ZERO = %08b\n", ksid[0])
}

func init() {
	Register("zero", NewZero)

	testZero()
}
