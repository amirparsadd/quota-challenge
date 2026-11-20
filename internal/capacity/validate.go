package capacity

import "fmt"

// ValidateAmount checks catalog rules for a single resource amount.
func (rt ResourceType) ValidateAmount(amount int64) error {
	if amount < rt.MinAmount {
		return fmt.Errorf("%w: %s below min %d", ErrValidation, rt.Name, rt.MinAmount)
	}
	if rt.MaxAmount != nil && amount > *rt.MaxAmount {
		return fmt.Errorf("%w: %s above max", ErrValidation, rt.Name)
	}
	if rt.StepAmount > 0 && amount%rt.StepAmount != 0 {
		return fmt.Errorf("%w: %s step %d", ErrValidation, rt.Name, rt.StepAmount)
	}
	if rt.Discrete && amount != 1 {
		return fmt.Errorf("%w: %s must be 1", ErrValidation, rt.Name)
	}
	return nil
}