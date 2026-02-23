package service

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"

	ingestModel "crisisecho/internal/apps/ingest/model"
	postModel   "crisisecho/internal/apps/post/model"
	postService "crisisecho/internal/apps/post/service"
	"crisisecho/internal/geo"
)

// IngestService defines the public contract for the Kafka ingest domain.
type IngestService interface {
	ConsumeAndRoute(ctx context.Context) error
	ReplayBatch(ctx context.Context, dataset string, ratePerSec int) error
}

type ingestService struct {
	brokers     []string
	groupID     string
	postService postService.PostService
}

// NewIngestService constructs an IngestService.
// brokers is a slice of Kafka broker addresses (e.g. ["localhost:9092"]).
// groupID is the Kafka consumer group ID.
func NewIngestService(brokers []string, groupID string, postSvc postService.PostService) IngestService {
	return &ingestService{
		brokers:     brokers,
		groupID:     groupID,
		postService: postSvc,
	}
}

// aivenTLSConfig builds a *tls.Config from KAFKA_SSL_* environment variables.
// Returns nil if KAFKA_SSL_CA_FILE is not set (plain-text mode).
func aivenTLSConfig() (*tls.Config, error) {
	caFile   := os.Getenv("KAFKA_SSL_CA_FILE")
	certFile := os.Getenv("KAFKA_SSL_CERT_FILE")
	keyFile  := os.Getenv("KAFKA_SSL_KEY_FILE")
	if caFile == "" {
		return nil, nil
	}
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("aivenTLSConfig: read CA cert %q: %w", caFile, err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	tlsCfg := &tls.Config{RootCAs: pool}
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("aivenTLSConfig: load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

// ConsumeAndRoute runs a sarama consumer group that subscribes to all three Kafka
// topics and routes each message to PostService.CreateRawPost by source.
// Runs until ctx is cancelled (graceful shutdown).
func (s *ingestService) ConsumeAndRoute(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	if tlsCfg, err := aivenTLSConfig(); err != nil {
		return fmt.Errorf("IngestService.ConsumeAndRoute: tls: %w", err)
	} else if tlsCfg != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsCfg
	}

	group, err := sarama.NewConsumerGroup(s.brokers, s.groupID, config)
	if err != nil {
		return fmt.Errorf("IngestService.ConsumeAndRoute: new consumer group: %w", err)
	}
	defer group.Close()

	topics := []string{
		ingestModel.TopicSocialRaw,
		ingestModel.TopicOfficialAlerts,
		ingestModel.TopicNewsFeed,
	}

	handler := &consumerGroupHandler{postService: s.postService}

	for {
		if err := group.Consume(ctx, topics, handler); err != nil {
			log.Printf("IngestService.ConsumeAndRoute: consume error: %v", err)
		}
		if ctx.Err() != nil {
			return nil // context cancelled — graceful shutdown
		}
	}
}

// ReplayBatch reads a newline-delimited JSON file and re-produces each line to Kafka
// at ratePerSec messages per second. Useful for replaying historical datasets in testing.
// Each line must be a valid JSON object with at least "source" and "payload" fields.
func (s *ingestService) ReplayBatch(ctx context.Context, dataset string, ratePerSec int) error {
	f, err := os.Open(dataset)
	if err != nil {
		return fmt.Errorf("IngestService.ReplayBatch: open %q: %w", dataset, err)
	}
	defer f.Close()

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Producer.Return.Successes = true

	if tlsCfg, err := aivenTLSConfig(); err != nil {
		return fmt.Errorf("IngestService.ReplayBatch: tls: %w", err)
	} else if tlsCfg != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsCfg
	}

	producer, err := sarama.NewSyncProducer(s.brokers, config)
	if err != nil {
		return fmt.Errorf("IngestService.ReplayBatch: new producer: %w", err)
	}
	defer producer.Close()

	interval := time.Second / time.Duration(ratePerSec)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		var km ingestModel.KafkaMessage
		if err := json.Unmarshal(line, &km); err != nil {
			log.Printf("IngestService.ReplayBatch: skip malformed line: %v", err)
			continue
		}

		topic := km.Topic
		if topic == "" {
			topic = ingestModel.TopicSocialRaw
		}

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(line),
		}
		if _, _, err := producer.SendMessage(msg); err != nil {
			log.Printf("IngestService.ReplayBatch: send error: %v", err)
		}
		time.Sleep(interval)
	}

	return scanner.Err()
}

// ─── sarama ConsumerGroupHandler ─────────────────────────────────────────────

type consumerGroupHandler struct {
	postService postService.PostService
}

// Setup is called at the start of a new consumer group session.
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

// Cleanup is called at the end of a consumer group session.
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim processes messages from a single assigned partition.
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.routeMessage(session.Context(), msg); err != nil {
				log.Printf("IngestService.ConsumeClaim: route error (non-critical): %v", err)
			}
			session.MarkMessage(msg, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

// routeMessage deserializes a Kafka message envelope and calls PostService.CreateRawPost.
func (h *consumerGroupHandler) routeMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var km ingestModel.KafkaMessage
	if err := json.Unmarshal(msg.Value, &km); err != nil {
		return fmt.Errorf("routeMessage: unmarshal KafkaMessage: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(km.Payload, &payload); err != nil {
		return fmt.Errorf("routeMessage: unmarshal payload: %w", err)
	}

	rawPost := &postModel.RawPost{
		Source:    km.Source,
		Timestamp: km.ReceivedAt,
		Location:  geo.GeoJSONPoint{Type: "Point", Coordinates: [2]float64{0, 0}},
		ImageURLs: []string{},
		Metadata:  payload,
	}

	// Extract well-known fields from the payload.
	if text, ok := payload["text"].(string); ok {
		rawPost.Text = text
	}
	if postID, ok := payload["post_id"].(string); ok {
		rawPost.PostID = postID
	}
	if user, ok := payload["user"].(string); ok {
		rawPost.User = user
	}
	if url, ok := payload["url"].(string); ok {
		rawPost.URL = url
	}
	if crisisType, ok := payload["crisis_type"].(string); ok {
		rawPost.CrisisType = crisisType
	}

	if err := h.postService.CreateRawPost(ctx, km.Source, rawPost); err != nil {
		return fmt.Errorf("routeMessage: CreateRawPost: %w", err)
	}
	return nil
}
