package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
)

type createBody struct {
	Product    string `json:"product"`
	TTLSeconds *int64 `json:"ttl_seconds"`
	Items      []struct {
		Resource string `json:"resource"`
		Amount   int64  `json:"amount"`
	} `json:"items"`
}

type confirmBody struct {
	State string `json:"state"`
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	region := r.PathValue("region")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.ListServices(ctx, &regionquotav1.ListServicesRequest{CustomerId: cid})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeListServicesResponse(resp))
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	region := r.PathValue("region")
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	cid, ok := customerID(r)
	if !ok || idem == "" {
		WriteProblem(w, http.StatusBadRequest, "missing-headers", "X-Customer-ID and Idempotency-Key required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteProblem(w, http.StatusBadRequest, "invalid-body", err.Error())
		return
	}
	var cb createBody
	if err := json.Unmarshal(body, &cb); err != nil {
		WriteProblem(w, http.StatusBadRequest, "invalid-json", err.Error())
		return
	}
	items := make([]*regionquotav1.ResourceLine, len(cb.Items))
	for i, it := range cb.Items {
		items[i] = &regionquotav1.ResourceLine{Resource: it.Resource, Amount: it.Amount}
	}
	req := &regionquotav1.ReserveRequest{
		CustomerId: cid, IdempotencyKey: idem, Product: cb.Product, Items: items,
	}
	if cb.TTLSeconds != nil {
		req.TtlSeconds = cb.TTLSeconds
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.Reserve(ctx, req)
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	if resp.GetRejectionReplay() {
		w.Header().Set("Idempotent-Replayed", "true")
		status := int(resp.GetHttpStatus())
		if status == 0 {
			status = http.StatusConflict
		}
		WriteRawProblem(w, status, []byte(resp.GetProblemJson()))
		return
	}
	sid := resp.Service.GetId()
	w.Header().Set("Location", "/v1/regions/"+region+"/services/"+sid)
	if resp.Replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	writeJSON(w, http.StatusCreated, encodeServiceResponse(&regionquotav1.ServiceResponse{Service: resp.Service}))
}

func (s *Server) listServiceEvents(w http.ResponseWriter, r *http.Request) {
	region, id := r.PathValue("region"), r.PathValue("id")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.ListServiceEvents(ctx, &regionquotav1.ListServiceEventsRequest{
		CustomerId: cid, ServiceId: id,
	})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeListServiceEventsResponse(resp))
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	region, id := r.PathValue("region"), r.PathValue("id")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.GetService(ctx, &regionquotav1.GetServiceRequest{CustomerId: cid, ServiceId: id})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeServiceResponse(resp))
}

func (s *Server) confirmService(w http.ResponseWriter, r *http.Request) {
	region, id := r.PathValue("region"), r.PathValue("id")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	raw, _ := io.ReadAll(r.Body)
	var patch confirmBody
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &patch); err != nil {
			WriteProblem(w, http.StatusBadRequest, "invalid-json", err.Error())
			return
		}
		if patch.State != "" && patch.State != "active" {
			WriteProblem(w, http.StatusUnprocessableEntity, "validation-error", "only state:active is supported")
			return
		}
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.Confirm(ctx, &regionquotav1.ConfirmRequest{CustomerId: cid, ServiceId: id})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeServiceResponse(resp))
}

func (s *Server) cancelService(w http.ResponseWriter, r *http.Request) {
	region, id := r.PathValue("region"), r.PathValue("id")
	cid, ok := customerID(r)
	if !ok {
		WriteProblem(w, http.StatusUnauthorized, "missing-customer", "X-Customer-ID required")
		return
	}
	b, ok := s.backend(region)
	if !ok {
		WriteProblem(w, http.StatusNotFound, "unknown-region", region)
		return
	}
	ctx, cancel := s.rpcContext(r.Context())
	defer cancel()
	resp, err := b.Cancel(ctx, &regionquotav1.CancelRequest{CustomerId: cid, ServiceId: id})
	if err != nil {
		WriteGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, encodeServiceResponse(resp))
}