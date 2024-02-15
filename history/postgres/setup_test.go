package postgres_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/ZutrixPog/dispatcher/history/postgres"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

const (
	image       = "postgres"
	version     = "latest"
	poolMaxWait = 120 * time.Second
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "latest",
		Env: []string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=test",
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: "5432"},
			},
		},
	}
	container, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(m, pool, container)

	pool.MaxWait = poolMaxWait

	port := container.GetPort("5432/tcp")

	if err := pool.Retry(func() error {
		url := fmt.Sprintf("host=localhost port=%s user=test dbname=test password=test sslmode=disable", port)
		db, err = postgres.InitDB(url, &gorm.Config{})
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
		c, _ := db.DB()
		err := c.Close()
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
