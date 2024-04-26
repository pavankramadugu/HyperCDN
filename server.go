// server.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"google.golang.org/grpc"

	"pavankramadugu.hypercdn/m/cache"
)

type cacheServer struct {
	cache.UnimplementedCacheServiceServer
	redisClient *redis.Client
}

func (s *cacheServer) Get(ctx context.Context, req *cache.GetRequest) (*cache.GetResponse, error) {
	key := fmt.Sprintf("%s:%s", req.UserId, req.Key)
	value, err := s.redisClient.Get(key).Bytes()
	if err != nil {
		return nil, err
	}
	return &cache.GetResponse{Value: value}, nil
}

func (s *cacheServer) Set(ctx context.Context, req *cache.SetRequest) (*cache.SetResponse, error) {
	key := fmt.Sprintf("%s:%s", req.UserId, req.Key)
	err := s.redisClient.Set(key, req.Value, time.Duration(req.Expiration)*time.Second).Err()
	if err != nil {
		return nil, err
	}
	return &cache.SetResponse{Success: true}, nil
}

func (s *cacheServer) Delete(ctx context.Context, req *cache.DeleteRequest) (*cache.DeleteResponse, error) {
	key := fmt.Sprintf("%s:%s", req.UserId, req.Key)
	err := s.redisClient.Del(key).Err()
	if err != nil {
		return nil, err
	}
	return &cache.DeleteResponse{Success: true}, nil
}

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	grpcServer := grpc.NewServer()
	cache.RegisterCacheServiceServer(grpcServer, &cacheServer{
		redisClient: redisClient,
	})

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	cacheClient := cache.NewCacheServiceClient(conn)

	router := gin.Default()

	router.GET("/cache/:userId/:key", func(c *gin.Context) {
		userId := c.Param("userId")
		key := c.Param("key")
		resp, err := cacheClient.Get(context.Background(), &cache.GetRequest{UserId: userId, Key: key})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Data(200, "application/octet-stream", resp.Value)
	})

	router.POST("/cache", func(c *gin.Context) {
		var req cache.SetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		_, err := cacheClient.Set(context.Background(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(200)
	})

	router.DELETE("/cache/:userId/:key", func(c *gin.Context) {
		userId := c.Param("userId")
		key := c.Param("key")
		_, err := cacheClient.Delete(context.Background(), &cache.DeleteRequest{UserId: userId, Key: key})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(200)
	})

	err = router.Run(":8080")
	if err != nil {
		return
	}
}
