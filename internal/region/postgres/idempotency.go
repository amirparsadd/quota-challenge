package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"quota-challenge/internal/capacity"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Store) GetIdempotency(ctx context.Context, customerID, key string) (*capacity.IdempotencyRecord, error) {
	var row capacity.IdempotencyRecord
	var body []byte
	err := s.pool.QueryRow(ctx, `
		SELECT status, request_hash, lease_expires_at, lease_token, response_status, response_body
		FROM idempotency_keys WHERE customer_id = $1 AND key = $2`, customerID, key).
		Scan(&row.Status, &row.RequestHash, &row.LeaseExpires, &row.LeaseToken, &row.ResponseStatus, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	row.ResponseBody = body
	return &row, err
}

func (s *Store) ClaimIdempotency(ctx context.Context, customerID, key, hash string, lease time.Duration) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	token := uuid.New()
	exp := time.Now().UTC().Add(lease)

	var status, existingHash string
	var leaseExp *time.Time
	var leaseTok *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT status, request_hash, lease_expires_at, lease_token
		FROM idempotency_keys WHERE customer_id = $1 AND key = $2 FOR UPDATE`,
		customerID, key).Scan(&status, &existingHash, &leaseExp, &leaseTok)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO idempotency_keys (key, customer_id, request_hash, status, lease_expires_at, lease_token)
			VALUES ($1, $2, $3, 'in_progress', $4, $5)`,
			key, customerID, hash, exp, token)
		if err != nil {
			if isUniqueViolation(err) {
				return uuid.Nil, capacity.ErrIdempotencyConflict
			}
			return uuid.Nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return uuid.Nil, err
		}
		return token, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	if existingHash != hash {
		return uuid.Nil, capacity.ErrIdempotencyMismatch
	}
	if status == string(capacity.IdempotencyCompleted) || status == string(capacity.IdempotencyRejected) {
		return uuid.Nil, capacity.ErrIdempotencyConflict
	}
	if status == "in_progress" {
		if leaseExp != nil && time.Now().UTC().Before(*leaseExp) && leaseTok != nil {
			return uuid.Nil, capacity.ErrIdempotencyConflict
		}
		_, err = tx.Exec(ctx, `
			UPDATE idempotency_keys SET status = 'in_progress', lease_expires_at = $3, lease_token = $4, updated_at = now()
			WHERE customer_id = $1 AND key = $2`, customerID, key, exp, token)
		if err != nil {
			return uuid.Nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return uuid.Nil, err
		}
		return token, nil
	}
	return uuid.Nil, capacity.ErrIdempotencyConflict
}

func (s *Store) CompleteIdempotency(ctx context.Context, customerID, key string, token uuid.UUID, status string, httpStatus int, body any) error {
	b, _ := json.Marshal(body)
	tag, err := s.pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = $3, response_status = $4, response_body = $5, updated_at = now()
		WHERE customer_id = $1 AND key = $2 AND lease_token = $6 AND status = 'in_progress'`,
		customerID, key, status, httpStatus, b, token)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return capacity.ErrIdempotencyConflict
	}
	return nil
}

func (s *Store) CompleteRejected(ctx context.Context, customerID, key string, token uuid.UUID, problem any) error {
	return s.CompleteIdempotency(ctx, customerID, key, token, string(capacity.IdempotencyRejected), 409, problem)
}

func (s *Store) IdempotencyReplay(row *capacity.IdempotencyRecord) (*capacity.Service, int, error) {
	if row.ResponseBody == nil {
		return nil, 0, capacity.ErrNotFound
	}
	var wrapper struct {
		Service capacity.Service `json:"service"`
	}
	if err := json.Unmarshal(row.ResponseBody, &wrapper); err != nil {
		return nil, 0, err
	}
	status := 201
	if row.ResponseStatus != nil {
		status = *row.ResponseStatus
	}
	return &wrapper.Service, status, nil
}

// IdempotencyHTTPReplay returns stored HTTP status and body for completed or rejected keys.
func (s *Store) IdempotencyHTTPReplay(row *capacity.IdempotencyRecord) (int, []byte, error) {
	if row.ResponseBody == nil {
		return 0, nil, capacity.ErrNotFound
	}
	status := 409
	if row.ResponseStatus != nil {
		status = *row.ResponseStatus
	}
	body := append([]byte(nil), row.ResponseBody...)
	return status, body, nil
}

func (s *Store) SaveCompletedReserve(ctx context.Context, customerID, key string, token uuid.UUID, svc *capacity.Service) error {
	body := map[string]any{"service": svc}
	return s.CompleteIdempotency(ctx, customerID, key, token, string(capacity.IdempotencyCompleted), 201, body)
}