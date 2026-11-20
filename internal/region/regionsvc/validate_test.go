package regionsvc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"quota-challenge/internal/capacity"
)

func TestValidateAmount(t *testing.T) {
	rt := ResourceType{Name: "cpu", MinAmount: 1, StepAmount: 2, Discrete: false}
	require.NoError(t, ValidateAmount(rt, 2))
	require.Error(t, ValidateAmount(rt, 1))
	require.True(t, errors.Is(ValidateAmount(rt, 1), capacity.ErrValidation))
}

func TestValidateDiscretePublicIP(t *testing.T) {
	rt := ResourceType{Name: "public_ip", MinAmount: 1, StepAmount: 1, Discrete: true}
	require.NoError(t, ValidateAmount(rt, 1))
	require.Error(t, ValidateAmount(rt, 2))
}