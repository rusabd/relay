package api

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gin-gonic/gin"
	"github.com/rusabd/relay/pkg/db"
)

type Result struct {
	Data []interface{} `json:"data"`
	Next string        `json:"next"`
}

func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // process request

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.WithFields(logrus.Fields{
			"status":     status,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"latency":    latency,
			"user-agent": c.Request.UserAgent(),
		}).Info("request completed")
	}
}

func SetupRouter(storage *db.MongoDBRelay) error {

	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(&logrus.TextFormatter{})

	r := gin.New()
	r.Use(LoggerMiddleware(log))
	r.Use(gin.Recovery())

	r.POST("/v1/qs/:namespace/:key", func(c *gin.Context) {
		namespace := c.Param("namespace")
		key := c.Param("key")
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			log.Errorf("Failed to bind JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		if namespace == "" || key == "" {
			log.Error("Namespace or key is empty")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Namespace and key are required"})
			return
		}
		version, err := storage.Set(namespace, key, data)
		if err != nil {
			log.Errorf("Failed to set data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set value"})
			return
		}

		encodedVersion, err := encodeVersion(version)
		if err != nil {
			log.Errorf("Failed to encode version: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode version"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"next": encodedVersion})
	})

	r.GET("/v1/qs/:namespace/:key", func(c *gin.Context) {
		namespace := c.Param("namespace")
		key := c.Param("key")
		version := c.Query("next")
		var versionData primitive.ObjectID
		if version != "" {
			decodedVersion, err := decodeVersion(version)
			if err != nil {
				log.Errorf("Failed to decode version: %v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version"})
				return
			}
			versionData = decodedVersion
		} else {
			versionData = primitive.NilObjectID
		}
		if namespace == "" || key == "" {
			log.Error("Namespace or key is empty")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Namespace and key are required"})
			return
		}
		data, err := storage.Get(namespace, key, versionData)
		if err != nil {
			log.Errorf("Failed to get data: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
			return
		}
		allData := make([]interface{}, len(data))
		for i, item := range data {
			allData[i] = item["value"]
			versionData = item["_id"].(primitive.ObjectID)
		}
		next, err := encodeVersion(versionData)
		if err != nil {
			log.Errorf("Failed to encode version: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode version"})
			return
		}
		c.JSON(http.StatusOK, Result{
			Data: allData,
			Next: next,
		})
	})
	r.Run(":8082")
	return nil
}

func encodeVersion(version primitive.ObjectID) (string, error) {
	encoded := hex.EncodeToString(version[:])
	return encoded, nil
}

func decodeVersion(encoded string) (primitive.ObjectID, error) {
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return primitive.NilObjectID, err
	}
	if len(decoded) != 12 {
		return primitive.NilObjectID, fmt.Errorf("invalid version length: %d", len(decoded))
	}
	var version primitive.ObjectID
	copy(version[:], decoded)
	return version, nil
}
