package capacity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestServiceConfirmable(t *testing.T) {
	future := time.Now().UTC().Add(time.Minute)
	s := Service{ID: uuid.New(), State: StateReserved, ExpiresAt: &future}
	require.NoError(t, s.Confirmable(time.Now().UTC()))

	past := time.Now().UTC().Add(-time.Minute)
	expired := Service{State: StateReserved, ExpiresAt: &past}
	require.ErrorIs(t, expired.Confirmable(time.Now().UTC()), ErrReservationExpired)

	active := Service{State: StateActive}
	require.ErrorIs(t, active.Confirmable(time.Now().UTC()), ErrInvalidState)
}