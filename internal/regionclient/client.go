package regionclient

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	regionquotav1 "quota-challenge/api/gen/regionquota/v1"
	"quota-challenge/internal/obslog"
)

type Client struct {
	conn   *grpc.ClientConn
	Region string
	API    regionquotav1.CapacityInternalClient
}

func Dial(ctx context.Context, region, addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(obslog.UnaryClientInterceptor(region)),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		Region: region,
		API:    regionquotav1.NewCapacityInternalClient(conn),
	}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}