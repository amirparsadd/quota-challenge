package regionsvc

import (
	"fmt"

	"quota-challenge/internal/capacity"
)

type ResourceType struct {
	Name       string
	Unit       string
	MinAmount  int64
	StepAmount int64
	MaxAmount  *int64
	Discrete   bool
}

func ValidateAmount(rt ResourceType, amount int64) error {
	if amount < rt.MinAmount {
		return fmt.Errorf("%w: %s below min %d", capacity.ErrValidation, rt.Name, rt.MinAmount)
	}
	if rt.MaxAmount != nil && amount > *rt.MaxAmount {
		return fmt.Errorf("%w: %s above max", capacity.ErrValidation, rt.Name)
	}
	if rt.StepAmount > 0 && amount%rt.StepAmount != 0 {
		return fmt.Errorf("%w: %s step %d", capacity.ErrValidation, rt.Name, rt.StepAmount)
	}
	if rt.Discrete && amount != 1 {
		return fmt.Errorf("%w: %s must be 1", capacity.ErrValidation, rt.Name)
	}
	return nil
}