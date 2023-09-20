//go:build e2e
// +build e2e

package trigger_update_so_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "cloudevent-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                  = fmt.Sprintf("%s-ns", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	cloudEventName             = fmt.Sprintf("%s-ce", testName)
	cloudEventHTTPReceiverName = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName  = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL   = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, namespace)
)

type templateData struct {
	TestNamespace              string
	ScaledObject               string
	ClientName                 string
	CloudEventName             string
	CloudEventHTTPReceiverName string
	CloudEventHTTPServiceName  string
	CloudEventHTTPServiceURL   string
}

const (
	cloudEventTemplate = `
  apiVersion: keda.sh/v1alpha1
  kind: CloudEvent
  metadata:
    name: {{.CloudEventName}}
    namespace: {{.TestNamespace}}
  spec:
    clusterName: cluster-sample
    eventHandlers:
    - type: cloud-event-http
      name: cloud-event-http-sample
      metadata:
        endPoint: {{.CloudEventHTTPServiceURL}}
  `

	cloudEventHTTPServiceTemplate = `
  apiVersion: v1
  kind: Service
  metadata:
    name: {{.CloudEventHTTPServiceName}}
    namespace: {{.TestNamespace}}
  spec:
    type: ClusterIP
    ports:
    - protocol: TCP
      port: 8899
      targetPort: 8899
    selector:
      app: {{.CloudEventHTTPReceiverName}}
  `

	cloudEventHTTPReceiverTemplate = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      deploy: {{.CloudEventHTTPReceiverName}}
    name: {{.CloudEventHTTPReceiverName}}
    namespace: {{.TestNamespace}}
  spec:
    selector:
      matchLabels:
        app: {{.CloudEventHTTPReceiverName}}
    replicas: 1
    template:
      metadata:
        labels:
          app: {{.CloudEventHTTPReceiverName}}
      spec:
        containers:
        - name: httpreceiver
          image: docker.io/spiritzhou/cloudeventhttp:v2
          ports:
          - containerPort: 8899
          resources:
            requests:
              cpu: "200m"
            limits:
              cpu: "500m"
  `

	scaledObjectErrTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: test
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'pod=testWorkloadDeploymentName'
        value: '1'
        activationValue: '3'
`

	clientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.ClientName}}
    image: curlimages/curl
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`
)

func TestScaledObjectGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, namespace, 5, 20), "all pods should be running")

	testErrCloudEventEmitValue(t, kc, data)

	DeleteKubernetesResources(t, namespace, data, templates)
}

// tests basic scaling with one trigger based on metrics
func testErrCloudEventEmitValue(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- test emitting cloudevent about scaledobject err---")
	KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	// recreate database to clear it
	out, _, _ := ExecCommandOnSpecificPod(t, clientName, namespace, fmt.Sprintf("curl -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectCheckFailed"))

	assert.NotNil(t, out)

	cloudEvent := make(map[string]interface{})
	err := json.Unmarshal([]byte(out), &cloudEvent)
	assert.Nil(t, err)
	assert.Equal(t, cloudEvent["data"].(map[string]interface{})["message"], "ScaledObject doesn't have correct scaleTargetRef specification")
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              namespace,
			ScaledObject:               scaledObjectName,
			ClientName:                 clientName,
			CloudEventName:             cloudEventName,
			CloudEventHTTPReceiverName: cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:  cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:   cloudEventHTTPServiceURL,
		}, []Template{
			{Name: "cloudEventTemplate", Config: cloudEventTemplate},
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
		}
}