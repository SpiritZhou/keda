package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	amqp "github.com/rabbitmq/amqp091-go"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var rabbitMQAnonymizePattern *regexp.Regexp

func init() {
	rabbitMQAnonymizePattern = regexp.MustCompile(`([^ \/:]+):([^\/:]+)\@`)
}

const (
	rabbitQueueLengthMetricName            = "queueLength"
	rabbitModeTriggerConfigName            = "mode"
	rabbitValueTriggerConfigName           = "value"
	rabbitActivationValueTriggerConfigName = "activationValue"
	rabbitModeQueueLength                  = "QueueLength"
	rabbitModeMessageRate                  = "MessageRate"
	defaultRabbitMQQueueLength             = 20
	rabbitMetricType                       = "External"
	rabbitRootVhostPath                    = "/%2F"
	rmqTLSEnable                           = "enable"
)

const (
	httpProtocol    = "http"
	amqpProtocol    = "amqp"
	autoProtocol    = "auto"
	defaultProtocol = autoProtocol
)

const (
	sumOperation     = "sum"
	avgOperation     = "avg"
	maxOperation     = "max"
	defaultOperation = sumOperation
)

type rabbitMQScaler struct {
	metricType v2.MetricTargetType
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
	azureOAuth *azure.ADWorkloadIdentityTokenProvider
	logger     logr.Logger
}

type rabbitMQMetadata struct {
	triggerIndex          int           // scaler index
	connectionName        string        // name used for the AMQP connection
	QueueName             string        `keda:"name=queueName, order=triggerMetadata"`
	QueueLength           int           `keda:"name=queueLength, order=triggerMetadata, optional"`
	Mode                  string        `keda:"name=mode, order=triggerMetadata, enum=QueueLength;MessageRate, optional"`                     // QueueLength or MessageRate
	Value                 float64       `keda:"name=value, order=triggerMetadata, optional"`                                                  // trigger value (queue length or publish/sec. rate)
	ActivationValue       float64       `keda:"name=activationValue, order=triggerMetadata, default=0"`                                       // activation value
	Host                  string        `keda:"name=host, order=triggerMetadata;resolvedEnv"`                                                 // connection string for either HTTP or AMQP protocol
	Protocol              string        `keda:"name=protocol, order=triggerMetadata;authParams, enum=auto;http;amqp, default=auto, optional"` // either http or amqp protocol
	VhostName             string        `keda:"name=vhostName, order=triggerMetadata, optional"`                                              // override the vhost from the connection info
	UseRegex              bool          `keda:"name=useRegex, order=triggerMetadata, enum=true;false, default=false, optional"`               // specify if the queueName contains a rexeg
	ExcludeUnacknowledged bool          `keda:"name=excludeUnacknowledged, order=triggerMetadata, enum=true;false, default=false, optional"`  // specify if the QueueLength value should exclude Unacknowledged messages (Ready messages only)
	PageSize              int64         `keda:"name=pageSize, order=triggerMetadata, default=100, optional"`                                  // specify the page size if useRegex is enabled
	Operation             string        `keda:"name=operation, order=triggerMetadata, enum=sum;max;avg, default=sum, optional"`               // specify the operation to apply in case of multiples queues
	Timeout               time.Duration `keda:"name=timeout, order=triggerMetadata, optional"`                                                // custom http timeout for a specific trigger

	// TLS
	EnableTLS   bool
	Ca          string `keda:"name=ca, order=authParams, optional"`
	Cert        string `keda:"name=cert, order=authParams, optional"`
	Key         string `keda:"name=key, order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
	Tls         string `keda:"name=tls, order=authParams, enum=enable;disable, default=disable, optional"`
	UnsafeSsl   bool   `keda:"name=unsafeSsl, order=triggerMetadata, enum=true;false, default=false, optional"`

	// token provider for azure AD
	workloadIdentityClientID      string
	workloadIdentityTenantID      string
	workloadIdentityAuthorityHost string
	WorkloadIdentityResource      string `keda:"name=workloadIdentityResource, order=authParams, optional"`
}

func (r *rabbitMQMetadata) Validate() error {
	if r.PageSize < 1 {
		return fmt.Errorf("pageSize should be 1 or greater than 1")
	}

	r.EnableTLS = r.Tls == rmqTLSEnable

	certGiven := r.Cert != ""
	keyGiven := r.Key != ""
	if certGiven != keyGiven {
		return fmt.Errorf("both key and cert must be provided")
	}

	if r.Protocol == autoProtocol {
		parsedURL, err := url.Parse(r.Host)
		if err != nil {
			return fmt.Errorf("can't parse host to find protocol: %w", err)
		}
		switch parsedURL.Scheme {
		case "amqp", "amqps":
			r.Protocol = amqpProtocol
		case "http", "https":
			r.Protocol = httpProtocol
		default:
			return fmt.Errorf("unknown host URL scheme `%s`", parsedURL.Scheme)
		}
	}

	if r.Protocol == amqpProtocol && r.WorkloadIdentityResource != "" {
		return fmt.Errorf("workload identity is not supported for amqp protocol currently")
	}

	if r.UseRegex && r.Protocol != httpProtocol {
		return fmt.Errorf("configure only useRegex with http protocol")
	}

	if r.ExcludeUnacknowledged && r.Protocol != httpProtocol {
		return fmt.Errorf("configure ExcludeUnacknowledged=true with http protocol only")
	}

	_, err := parseTrigger(r)
	if err != nil {
		return fmt.Errorf("unable to parse trigger: %w", err)
	}
	// Resolve timeout
	if err := resolveTimeout(r); err != nil {
		return err
	}
	return nil
}

type queueInfo struct {
	Messages               int         `json:"messages"`
	MessagesReady          int         `json:"messages_ready"`
	MessagesUnacknowledged int         `json:"messages_unacknowledged"`
	MessageStat            messageStat `json:"message_stats"`
	Name                   string      `json:"name"`
}

type regexQueueInfo struct {
	Queues     []queueInfo `json:"items"`
	TotalPages int         `json:"page_count"`
}

type messageStat struct {
	PublishDetail publishDetail `json:"publish_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	s := &rabbitMQScaler{}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	s.metricType = metricType

	s.logger = InitializeLogger(config, "rabbitmq_scaler")

	meta, err := parseRabbitMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %w", err)
	}
	s.metadata = meta
	s.httpClient = kedautil.CreateHTTPClient(meta.Timeout, meta.UnsafeSsl)

	if meta.EnableTLS {
		tlsConfig, tlsErr := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
		if tlsErr != nil {
			return nil, tlsErr
		}
		s.httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
	}

	if meta.Protocol == amqpProtocol {
		// Override vhost if requested.
		host := meta.Host
		if meta.VhostName != "" {
			hostURI, err := amqp.ParseURI(host)
			if err != nil {
				return nil, fmt.Errorf("error parsing rabbitmq connection string: %w", err)
			}
			hostURI.Vhost = meta.VhostName
			host = hostURI.String()
		}

		conn, ch, err := getConnectionAndChannel(host, meta)
		if err != nil {
			return nil, fmt.Errorf("error establishing rabbitmq connection: %w", err)
		}
		s.connection = conn
		s.channel = ch
	}

	return s, nil
}

func resolveTimeout(meta *rabbitMQMetadata) error {
	if meta.Timeout != 0 {
		if meta.Protocol == amqpProtocol {
			return fmt.Errorf("amqp protocol doesn't support custom timeouts")
		}
		if meta.Timeout <= 0 {
			return fmt.Errorf("timeout must be greater than 0")
		}
		meta.Timeout = time.Duration(meta.Timeout) * time.Millisecond
	}
	return nil
}

func parseRabbitMQMetadata(config *scalersconfig.ScalerConfig) (*rabbitMQMetadata, error) {
	meta := rabbitMQMetadata{}
	if err := config.TypedConfig(&meta); err != nil {
		return &meta, fmt.Errorf("error parsing rabbitMQ metadata: %w", err)
	}
	meta.triggerIndex = config.TriggerIndex
	meta.connectionName = connectionName(config)

	if config.PodIdentity.Provider == v1alpha1.PodIdentityProviderAzureWorkload {
		if meta.WorkloadIdentityResource != "" {
			meta.workloadIdentityClientID = config.PodIdentity.GetIdentityID()
			meta.workloadIdentityTenantID = config.PodIdentity.GetIdentityTenantID()
		}
	}

	if meta.Timeout == 0 {
		meta.Timeout = config.GlobalHTTPTimeout
	}

	return &meta, nil
}

func parseTrigger(meta *rabbitMQMetadata) (*rabbitMQMetadata, error) {
	if meta.QueueLength == 0 && meta.Mode == "" && math.Dim(meta.Value, 0) < 0.00001 {
		return meta, nil
	}

	// Only allow one of `queueLength` or `mode`/`value`
	if meta.QueueLength != 0 && (meta.Mode != "" || math.Dim(meta.Value, 0) > 0.00001) {
		return nil, fmt.Errorf("queueLength is deprecated; configure only %s and %s", rabbitModeTriggerConfigName, rabbitValueTriggerConfigName)
	}

	// Parse deprecated `queueLength` value
	if meta.QueueLength != 0 {
		meta.Mode = rabbitModeQueueLength
		meta.Value = float64(meta.QueueLength)

		return meta, nil
	}

	if meta.Mode == "" {
		return nil, fmt.Errorf("%s must be specified", rabbitModeTriggerConfigName)
	}
	if math.Dim(meta.Value, 0) < 0.00001 {
		return nil, fmt.Errorf("%s must be specified", rabbitValueTriggerConfigName)
	}

	// Resolve trigger mode
	switch meta.Mode {
	case rabbitModeQueueLength:
		meta.Mode = rabbitModeQueueLength
	case rabbitModeMessageRate:
		meta.Mode = rabbitModeMessageRate
	default:
		return nil, fmt.Errorf("trigger mode %s must be one of %s, %s", meta.Mode, rabbitModeQueueLength, rabbitModeMessageRate)
	}

	if meta.Mode == rabbitModeMessageRate && meta.Protocol != httpProtocol {
		return nil, fmt.Errorf("protocol %s not supported; must be http to use mode %s", meta.Protocol, rabbitModeMessageRate)
	}

	return meta, nil
}

// getConnectionAndChannel returns an amqp connection. If enableTLS is true tls connection is made using
// the given ceClient cert, ceClient key,and CA certificate. If clientKeyPassword is not empty the provided password will be used to
// decrypt the given key. If enableTLS is disabled then amqp connection will be created without tls.
func getConnectionAndChannel(host string, meta *rabbitMQMetadata) (*amqp.Connection, *amqp.Channel, error) {
	amqpConfig := amqp.Config{
		Properties: amqp.Table{
			"connection_name": meta.connectionName,
		},
	}

	if meta.EnableTLS {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.Ca, meta.UnsafeSsl)
		if err != nil {
			return nil, nil, err
		}

		amqpConfig.TLSClientConfig = tlsConfig
	}

	conn, err := amqp.DialConfig(host, amqpConfig)
	if err != nil {
		return nil, nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}

	return conn, channel, nil
}

// Close disposes of RabbitMQ connections
func (s *rabbitMQScaler) Close(context.Context) error {
	if s.connection != nil {
		err := s.connection.Close()
		if err != nil {
			s.logger.Error(err, "Error closing rabbitmq connection")
			return err
		}
	}
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *rabbitMQScaler) getQueueStatus(ctx context.Context) (int64, float64, error) {
	if s.metadata.Protocol == httpProtocol {
		info, err := s.getQueueInfoViaHTTP(ctx)
		if err != nil {
			return -1, -1, err
		}

		if s.metadata.ExcludeUnacknowledged {
			// messages count includes only ready
			return int64(info.MessagesReady), info.MessageStat.PublishDetail.Rate, nil
		}
		// messages count includes count of ready and unack-ed
		return int64(info.Messages), info.MessageStat.PublishDetail.Rate, nil
	}

	// QueueDeclarePassive assumes that the queue exists and fails if it doesn't
	items, err := s.channel.QueueDeclarePassive(s.metadata.QueueName, false, false, false, false, amqp.Table{})
	if err != nil {
		return -1, -1, err
	}

	return int64(items.Messages), 0, nil
}

func getJSON(ctx context.Context, s *rabbitMQScaler, url string) (queueInfo, error) {
	var result queueInfo

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return result, err
	}

	if s.metadata.WorkloadIdentityResource != "" {
		if s.azureOAuth == nil {
			s.azureOAuth = azure.NewAzureADWorkloadIdentityTokenProvider(ctx, s.metadata.workloadIdentityClientID, s.metadata.workloadIdentityTenantID, s.metadata.workloadIdentityAuthorityHost, s.metadata.WorkloadIdentityResource)
		}

		err = s.azureOAuth.Refresh()
		if err != nil {
			return result, err
		}

		request.Header.Set("Authorization", "Bearer "+s.azureOAuth.OAuthToken())
	}

	r, err := s.httpClient.Do(request)
	if err != nil {
		return result, err
	}

	defer r.Body.Close()

	if r.StatusCode == 200 {
		if s.metadata.UseRegex {
			var queues regexQueueInfo
			err = json.NewDecoder(r.Body).Decode(&queues)
			if err != nil {
				return queueInfo{}, err
			}
			if queues.TotalPages > 1 {
				return queueInfo{}, fmt.Errorf("regex matches more queues than can be recovered at once")
			}
			result, err := getComposedQueue(s, queues.Queues)
			return result, err
		}

		err = json.NewDecoder(r.Body).Decode(&result)
		return result, err
	}

	body, _ := io.ReadAll(r.Body)
	return result, fmt.Errorf("error requesting rabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
}

func getVhostAndPathFromURL(rawPath, vhostName string) (resolvedVhostPath, resolvedPath string) {
	pathParts := strings.Split(rawPath, "/")
	resolvedVhostPath = "/" + pathParts[len(pathParts)-1]
	resolvedPath = path.Join(pathParts[:len(pathParts)-1]...)

	if len(resolvedPath) > 0 {
		resolvedPath = "/" + resolvedPath
	}
	if vhostName != "" {
		resolvedVhostPath = "/" + url.QueryEscape(vhostName)
	}
	if resolvedVhostPath == "" || resolvedVhostPath == "/" || resolvedVhostPath == "//" {
		resolvedVhostPath = rabbitRootVhostPath
	}

	return
}

func (s *rabbitMQScaler) getQueueInfoViaHTTP(ctx context.Context) (*queueInfo, error) {
	parsedURL, err := url.Parse(s.metadata.Host)

	if err != nil {
		return nil, err
	}

	vhost, subpaths := getVhostAndPathFromURL(parsedURL.Path, s.metadata.VhostName)
	parsedURL.Path = subpaths

	var getQueueInfoManagementURI string
	if s.metadata.UseRegex {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s?page=1&use_regex=true&pagination=false&name=%s&page_size=%d", parsedURL.String(), vhost, url.QueryEscape(s.metadata.QueueName), s.metadata.PageSize)
	} else {
		getQueueInfoManagementURI = fmt.Sprintf("%s/api/queues%s/%s", parsedURL.String(), vhost, url.QueryEscape(s.metadata.QueueName))
	}

	var info queueInfo
	info, err = getJSON(ctx, s, getQueueInfoManagementURI)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("rabbitmq-%s", url.QueryEscape(s.metadata.QueueName)))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: rabbitMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	messages, publishRate, err := s.getQueueStatus(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, s.anonymizeRabbitMQError(err)
	}

	var metric external_metrics.ExternalMetricValue
	var isActive bool
	if s.metadata.Mode == rabbitModeQueueLength {
		metric = GenerateMetricInMili(metricName, float64(messages))
		isActive = float64(messages) > s.metadata.ActivationValue
	} else {
		metric = GenerateMetricInMili(metricName, publishRate)
		isActive = publishRate > s.metadata.ActivationValue || float64(messages) > s.metadata.ActivationValue
	}

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func getComposedQueue(s *rabbitMQScaler, q []queueInfo) (queueInfo, error) {
	var queue = queueInfo{}
	queue.Name = "composed-queue"
	queue.MessagesUnacknowledged = 0
	if len(q) > 0 {
		switch s.metadata.Operation {
		case sumOperation:
			sumMessages, sumReady, sumRate := getSum(q)
			queue.Messages = sumMessages
			queue.MessagesReady = sumReady
			queue.MessageStat.PublishDetail.Rate = sumRate
		case avgOperation:
			avgMessages, avgReady, avgRate := getAverage(q)
			queue.Messages = avgMessages
			queue.MessagesReady = avgReady
			queue.MessageStat.PublishDetail.Rate = avgRate
		case maxOperation:
			maxMessages, maxReady, maxRate := getMaximum(q)
			queue.Messages = maxMessages
			queue.MessagesReady = maxReady
			queue.MessageStat.PublishDetail.Rate = maxRate
		default:
			return queue, fmt.Errorf("operation mode %s must be one of %s, %s, %s", s.metadata.Operation, sumOperation, avgOperation, maxOperation)
		}
	} else {
		queue.Messages = 0
		queue.MessageStat.PublishDetail.Rate = 0
	}

	return queue, nil
}

func getSum(q []queueInfo) (int, int, float64) {
	var sumMessages int
	var sumMessagesReady int
	var sumRate float64
	for _, value := range q {
		sumMessages += value.Messages
		sumMessagesReady += value.MessagesReady
		sumRate += value.MessageStat.PublishDetail.Rate
	}
	return sumMessages, sumMessagesReady, sumRate
}

func getAverage(q []queueInfo) (int, int, float64) {
	sumMessages, sumReady, sumRate := getSum(q)
	length := len(q)
	return sumMessages / length, sumReady / length, sumRate / float64(length)
}

func getMaximum(q []queueInfo) (int, int, float64) {
	var maxMessages int
	var maxReady int
	var maxRate float64
	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessagesReady > maxReady {
			maxReady = value.MessagesReady
		}
		if value.MessageStat.PublishDetail.Rate > maxRate {
			maxRate = value.MessageStat.PublishDetail.Rate
		}
	}
	return maxMessages, maxReady, maxRate
}

// Mask host for log purposes
func (s *rabbitMQScaler) anonymizeRabbitMQError(err error) error {
	errorMessage := fmt.Sprintf("error inspecting rabbitMQ: %s", err)
	return fmt.Errorf(rabbitMQAnonymizePattern.ReplaceAllString(errorMessage, "user:password@"))
}

// connectionName is used to provide a deterministic AMQP connection name when
// connecting to RabbitMQ
func connectionName(config *scalersconfig.ScalerConfig) string {
	return fmt.Sprintf("keda-%s-%s", config.ScalableObjectNamespace, config.ScalableObjectName)
}
