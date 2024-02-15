package redis_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/ory/dockertest"
)

var (
	client *redis.Client
)

const (
	image       = "redis"
	version     = "latest"
	poolMaxWait = 30 * time.Second
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if err := pool.Client.Ping(); err != nil {
		panic("Couldn't connect to docker")
	}

	opts := dockertest.RunOptions{
		Repository: image,
		Tag:        version,
	}
	container, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	container.Expire(30)

	handleInterrupt(m, pool, container)

	pool.MaxWait = poolMaxWait

	port := container.GetPort("6379/tcp")

	if err := pool.Retry(func() error {
		hostAndPort := fmt.Sprintf("localhost:%s", port)
		client = redis.NewClient(&redis.Options{
			Addr: hostAndPort,
			DB:   0,
		})
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)

	defer func() {
		client.Close()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()
}

func handleInterrupt(m *testing.M, pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}
