package grpcserver

import (
	"google.golang.org/grpc"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
	"quota-challenge/internal/region/regionsvc"
)

const callerGateway = "quota-gateway"

type Server struct {
	regionquotav1.UnimplementedCapacityInternalServer
	regionquotav1.UnimplementedCapacityAdminServer

	svc      *regionsvc.Service
	regionID string
}

func Register(s *grpc.Server, svc *regionsvc.Service, regionID string) {
	impl := &Server{svc: svc, regionID: regionID}
	regionquotav1.RegisterCapacityInternalServer(s, impl)
	regionquotav1.RegisterCapacityAdminServer(s, impl)
}