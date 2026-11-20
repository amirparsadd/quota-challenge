package capacity

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAmount(t *testing.T) {
	rt := ResourceType{Name: "cpu", MinAmount: 1, StepAmount: 2, Discrete: false}
	require.NoError(t, rt.ValidateAmount(2))
	require.Error(t, rt.ValidateAmount(1))
	require.True(t, errors.Is(rt.ValidateAmount(1), ErrValidation))
}

func TestValidateDiscretePublicIP(t *testing.T) {
	rt := ResourceType{Name: "public_ip", MinAmount: 1, StepAmount: 1, Discrete: true}
	require.NoError(t, rt.ValidateAmount(1))
	require.Error(t, rt.ValidateAmount(2))
}