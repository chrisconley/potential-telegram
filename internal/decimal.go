package internal

import (
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

type Decimal struct {
	value apd.Decimal
}

func NewDecimal(s string) (Decimal, error) {
	var d apd.Decimal
	_, _, err := d.SetString(s)
	if err != nil {
		return Decimal{}, fmt.Errorf("invalid decimal: %w", err)
	}
	return Decimal{value: d}, nil
}

func NewDecimalFromInt64(i int64) Decimal {
	var d apd.Decimal
	d.SetInt64(i)
	return Decimal{value: d}
}

func (d Decimal) String() string {
	return d.value.String()
}

func (d Decimal) IsZero() bool {
	return d.value.IsZero()
}

func (d Decimal) Cmp(other Decimal) int {
	return d.value.Cmp(&other.value)
}

// Add returns the sum of d and other.
func (d Decimal) Add(other Decimal) Decimal {
	var result apd.Decimal
	ctx := apd.BaseContext.WithPrecision(34)
	ctx.Add(&result, &d.value, &other.value)
	return Decimal{value: result}
}

// Mul returns the product of d and other.
func (d Decimal) Mul(other Decimal) Decimal {
	var result apd.Decimal
	ctx := apd.BaseContext.WithPrecision(34)
	ctx.Mul(&result, &d.value, &other.value)
	return Decimal{value: result}
}

// Div returns the quotient of d divided by other.
func (d Decimal) Div(other Decimal) Decimal {
	var result apd.Decimal
	ctx := apd.BaseContext.WithPrecision(34)
	ctx.Quo(&result, &d.value, &other.value)
	return Decimal{value: result}
}
