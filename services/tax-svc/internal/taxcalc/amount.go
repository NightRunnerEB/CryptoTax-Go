package taxcalc

import (
	"fmt"
	"math/big"
	"strings"
)

const decimalScale = 18

type Amount struct {
	r *big.Rat
}

func Zero() Amount {
	return Amount{r: new(big.Rat)}
}

func NewInt(v int64) Amount {
	return Amount{r: big.NewRat(v, 1)}
}

func Parse(raw string) (Amount, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Zero(), fmt.Errorf("empty decimal")
	}

	r := new(big.Rat)
	if _, ok := r.SetString(raw); !ok {
		return Zero(), fmt.Errorf("invalid decimal: %q", raw)
	}
	return Amount{r: r}, nil
}

func MustParse(raw string) Amount {
	a, err := Parse(raw)
	if err != nil {
		panic(err)
	}
	return a
}

func (a Amount) Add(b Amount) Amount {
	return Amount{r: new(big.Rat).Add(a.rat(), b.rat())}
}

func (a Amount) Sub(b Amount) Amount {
	return Amount{r: new(big.Rat).Sub(a.rat(), b.rat())}
}

func (a Amount) Mul(b Amount) Amount {
	return Amount{r: new(big.Rat).Mul(a.rat(), b.rat())}
}

func (a Amount) Div(b Amount) Amount {
	return Amount{r: new(big.Rat).Quo(a.rat(), b.rat())}
}

func (a Amount) Abs() Amount {
	if a.Cmp(Zero()) >= 0 {
		return a
	}
	return Amount{r: new(big.Rat).Neg(a.rat())}
}

func (a Amount) Cmp(b Amount) int {
	return a.rat().Cmp(b.rat())
}

func (a Amount) IsZero() bool {
	return a.rat().Sign() == 0
}

func (a Amount) IsNegative() bool {
	return a.rat().Sign() < 0
}

func (a Amount) String() string {
	if a.rat().Sign() == 0 {
		return "0"
	}
	s := a.rat().FloatString(decimalScale)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func (a Amount) rat() *big.Rat {
	if a.r == nil {
		return new(big.Rat)
	}
	return a.r
}

func Min(a, b Amount) Amount {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}
