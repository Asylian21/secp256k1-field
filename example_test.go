package field_test

import (
	"fmt"

	field "github.com/Asylian21/secp256k1-field"
)

// ExampleVal demonstrates a multiply in the field. A result must be normalized
// before it is serialized or compared.
func ExampleVal() {
	var a, b, r field.Val
	a.SetInt(7)
	b.SetInt(6)

	r.Mul2(&a, &b) // r = a*b mod p

	fmt.Println(r.String())
	// Output: 000000000000000000000000000000000000000000000000000000000000002a
}
