package domain

import (
	"fmt"
	"strings"
)

type TaxPolicy struct {
	TreatCryptoCryptoAsDisposal bool            `json:"treat_crypto_crypto_as_disposal"`
	CostBasisMethod             CostBasisMethod `json:"cost_basis_method"`
}

type CostBasisMethod string

const (
	FIFO CostBasisMethod = "FIFO"
	LIFO CostBasisMethod = "LIFO"
	AVG  CostBasisMethod = "AVG"
)

func (m CostBasisMethod) Normalize() CostBasisMethod {
	return CostBasisMethod(strings.ToUpper(strings.TrimSpace(string(m))))
}

func (m CostBasisMethod) Validate() error {
	switch m {
	case FIFO, LIFO, AVG:
		return nil
	default:
		return fmt.Errorf("unsupported cost basis method: %s", m)
	}
}

func (p TaxPolicy) Normalize() TaxPolicy {
	p.CostBasisMethod = p.CostBasisMethod.Normalize()
	if p.CostBasisMethod == "" {
		p.CostBasisMethod = FIFO
	}
	return p
}

func (p TaxPolicy) Validate() error {
	return p.CostBasisMethod.Validate()
}
