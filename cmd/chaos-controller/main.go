package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "tcp://emqx.observability:1883"
	}

	namespace := os.Getenv("TARGET_NAMESPACE")
	if namespace == "" {
		namespace = "hardware-sim"
	}

	// 1. Initialize Kubernetes Client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create kubernetes client: %v", err)
	}

	// 2. Initialize MQTT Client
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	opts.SetClientID("chaos-controller")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT: %v", token.Error())
	}
	defer client.Disconnect(250)

	log.Printf("Chaos Controller started. Targeting namespace: %s", namespace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	// Dynamic chaos loop
	for {
		// Randomize interval between 15s and 45s (Average 30s)
		interval := time.Duration(15+rand.Intn(31)) * time.Second
		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			injectChaos(ctx, clientset, client, namespace)
		}
	}
}

func injectChaos(ctx context.Context, k8s *kubernetes.Clientset, mqttClient mqtt.Client, namespace string) {
	pods, err := k8s.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=sensor-fleet",
	})
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		return
	}

	if len(pods.Items) == 0 {
		log.Println("No sensor pods found to target.")
		return
	}

	// Target a random pod
	targetPod := pods.Items[rand.Intn(len(pods.Items))]

	// Randomize Spike Parameters
	durationSec := 10 + rand.Intn(21) // 10s to 30s
	intensity := []string{"low", "medium", "high"}[rand.Intn(3)]

	log.Printf("Injecting Chaos into %s: Intensity=%s, Duration=%ds", targetPod.Name, intensity, durationSec)

	topic := fmt.Sprintf("sensors/%s/chaos", targetPod.Name)
	payload := fmt.Sprintf(`{"command": "spike", "duration": "%ds", "intensity": "%s"}`, durationSec, intensity)

	token := mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error publishing chaos command to %s: %v", targetPod.Name, token.Error())
	}
}
