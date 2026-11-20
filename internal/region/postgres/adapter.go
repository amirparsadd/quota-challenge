package postgres

import "quota-challenge/internal/region/port"

// Store implements the region quota outbound port via PostgreSQL.
var _ port.QuotaRepository = (*Store)(nil)