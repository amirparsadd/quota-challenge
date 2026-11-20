package regionsvc

import "quota-challenge/internal/region/port"

type Service struct {
	repo port.QuotaRepository
	cfg  Config
}

func New(repo port.QuotaRepository) *Service {
	return NewWithConfig(repo, DefaultConfig())
}

func NewWithConfig(repo port.QuotaRepository, cfg Config) *Service {
	if cfg == (Config{}) {
		cfg = DefaultConfig()
	}
	return &Service{repo: repo, cfg: cfg}
}