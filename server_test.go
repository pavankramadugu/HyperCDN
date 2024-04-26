package main

import (
	"context"
	"net"
	"testing"

	"github.com/go-redis/redis"
	"google.golang.org/grpc"

	"pavankramadugu.hypercdn/m/cache"
)

func TestCacheServer(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	grpcServer := grpc.NewServer()
	cache.RegisterCacheServiceServer(grpcServer, &cacheServer{
		redisClient: redisClient,
	})

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	defer grpcServer.Stop()

	conn, err := grpc.Dial(":50052", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	cacheClient := cache.NewCacheServiceClient(conn)

	// Test Set
	setReq := &cache.SetRequest{
		UserId:     "user1",
		Key:        "key1",
		Value:      []byte("value1"),
		Expiration: 60,
	}
	setResp, err := cacheClient.Set(context.Background(), setReq)
	if err != nil {
		t.Errorf("Failed to set cache: %v", err)
	}
	if !setResp.Success {
		t.Error("Set cache failed")
	}

	// Test Get
	getReq := &cache.GetRequest{
		UserId: "user1",
		Key:    "key1",
	}
	getResp, err := cacheClient.Get(context.Background(), getReq)
	if err != nil {
		t.Errorf("Failed to get cache: %v", err)
	}
	if string(getResp.Value) != "value1" {
		t.Errorf("Unexpected cache value. Expected: value1, Got: %s", string(getResp.Value))
	}

	// Test Delete
	deleteReq := &cache.DeleteRequest{
		UserId: "user1",
		Key:    "key1",
	}
	deleteResp, err := cacheClient.Delete(context.Background(), deleteReq)
	if err != nil {
		t.Errorf("Failed to delete cache: %v", err)
	}
	if !deleteResp.Success {
		t.Error("Delete cache failed")
	}
}
