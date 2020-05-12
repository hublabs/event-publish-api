package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pangpanglabs/goutils/behaviorlog"
	"github.com/pangpanglabs/goutils/echomiddleware"
	"github.com/pangpanglabs/goutils/kafka"
)

func main() {
	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	o2oProducer, err := kafka.NewProducer(kafkaBrokers, "o2o", func(c *sarama.Config) {
		c.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
		c.Producer.Compression = sarama.CompressionGZIP     // Compress messages
		c.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	})
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	e.POST("/o2o/:status", func(c echo.Context) error {
		var payload interface{}
		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		status := c.Param("status")
		if status == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "'status' is required.",
			})
		}
		behaviorlogContext := behaviorlog.FromCtx(c.Request().Context())
		if err := o2oProducer.Send(map[string]interface{}{
			"status":    status,
			"requestId": behaviorlogContext.RequestID,
			"authToken": behaviorlogContext.AuthToken,
			"payload":   payload,
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]bool{
			"success": true,
		})
	})
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(echomiddleware.BehaviorLogger("event-publish-api", kafka.Config{
		Brokers: kafkaBrokers,
		Topic:   "behaviorlog",
	}))

	if err := e.Start(":8000"); err != nil {
		log.Println(err)
	}
}
