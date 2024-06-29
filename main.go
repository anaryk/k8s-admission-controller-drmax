package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	azurewrapper "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/azure"
	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/certificatecache"
	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"
	mutating "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/webhook/mutation"
	validating "dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/webhook/validation"
	"github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhprometheus "github.com/slok/kubewebhook/v2/pkg/metrics/prometheus"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	"k8s.io/client-go/kubernetes"
)

const (
	gracePeriod = 3 * time.Second
	minReps     = 1
	maxReps     = 12
)

type Main struct {
	flags        *Flags
	logger       kwhlog.Logger
	stopC        chan struct{}
	keyVaultName string
}

// Run will run the main program.
func (m *Main) Run() error {
	// Create services.
	promReg := prometheus.NewRegistry()
	metricsRec, err := kwhprometheus.NewRecorder(kwhprometheus.RecorderConfig{Registry: promReg})
	if err != nil {
		return fmt.Errorf("could not create prometheus recorder: %w", err)
	}

	// Create webhooks

	//Cert order mutating webhook
	certOrderMutator, err := mutating.CertOrderMutateWebhook(m.logger)
	if err != nil {
		return err
	}
	certOrderMutator = kwhwebhook.NewMeasuredWebhook(metricsRec, certOrderMutator)
	certOrderWebHook, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: certOrderMutator, Logger: m.logger})
	if err != nil {
		return err
	}

	//Ingress certs mutating webhook
	ingressCertsMutator, err := mutating.IngressCertsMutateWebhook(m.logger)
	if err != nil {
		return err
	}
	ingressCertsMutator = kwhwebhook.NewMeasuredWebhook(metricsRec, ingressCertsMutator)
	ingressCertsWebHook, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: ingressCertsMutator, Logger: m.logger})
	if err != nil {
		return err
	}

	//Certificate cache mutating webhook
	certificateCacheMutator, err := mutating.CertificateCacheMutateWebhook(m.logger, m.keyVaultName)
	if err != nil {
		return err
	}
	certificateCacheMutator = kwhwebhook.NewMeasuredWebhook(metricsRec, certificateCacheMutator)
	certificateCacheWebHook, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: certificateCacheMutator, Logger: m.logger})
	if err != nil {
		return err
	}

	//Deployment validation webhook (not used) only as exampel for feature development
	vdw, err := validating.NewDeploymentWebhook(minReps, maxReps, m.logger)
	if err != nil {
		return err
	}
	vdw = kwhwebhook.NewMeasuredWebhook(metricsRec, vdw)
	vdwh, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: vdw, Logger: m.logger})
	if err != nil {
		return err
	}

	// Create the servers and set them listenig.
	errC := make(chan error)

	// Serve webhooks.
	// TODO: Move to it's own service.
	go func() {

		m.logger.Infof("webhooks listening on %s...", m.flags.ListenAddress)
		mux := http.NewServeMux()
		mux.Handle("/webhooks/mutating/certorder", certOrderWebHook)
		mux.Handle("/webhooks/mutating/certificatecache", certificateCacheWebHook)
		mux.Handle("/webhooks/mutating/ingresscerts", ingressCertsWebHook)
		mux.Handle("/webhooks/validating/deployment", vdwh)
		errC <- http.ListenAndServeTLS(
			m.flags.ListenAddress,
			m.flags.CertFile,
			m.flags.KeyFile,
			mux,
		)
	}()

	metricsHandler := promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})
	go func() {
		m.logger.Infof("metrics listening on %s...", m.flags.MetricsListenAddress)
		errC <- http.ListenAndServe(m.flags.MetricsListenAddress, metricsHandler)
	}()

	defer m.stop()

	sigC := m.createSignalChan()
	select {
	case err := <-errC:
		if err != nil {
			m.logger.Errorf("error received: %s", err)
			return err
		}
		m.logger.Infof("app finished successfuly")
	case s := <-sigC:
		m.logger.Infof("signal %s received", s)
		return nil
	}

	return nil
}

func (m *Main) stop() {
	m.logger.Infof("stopping everything, waiting %s...", gracePeriod)

	close(m.stopC)

	// Stop everything and let them time to stop.
	time.Sleep(gracePeriod)
}

func (m *Main) createSignalChan() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	return c
}

func main() {
	m := Main{
		flags:        NewFlags(),
		stopC:        make(chan struct{}),
		keyVaultName: os.Getenv("KEY_VAULT_NAME"),
	}

	logrusLogEntry := logrus.NewEntry(logrus.New())
	if m.flags.Debug {
		logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	} else {
		logrusLogEntry.Logger.SetLevel(logrus.InfoLevel)
	}
	m.logger = kwhlogrus.NewLogrus(logrusLogEntry)

	// Initialize Kubernetes client
	k8sClient, err := k8s.PrepareLocalKubeconfigK8SClient()
	if err != nil {
		m.logger.Errorf("Failed to create Kubernetes client: %v", err)
	}

	// Initialize Kubernetes clientset
	k8sClientSet, err := kubernetes.NewForConfig(k8sClient)
	if err != nil {
		m.logger.Errorf("Failed to create Kubernetes clientset: %v", err)
	}

	// Initialize Key Vault client
	keyVaultClient, err := azurewrapper.NewKeyVaultClient(m.keyVaultName)
	if err != nil {
		m.logger.Errorf("Failed to create Key Vault client: %v", err)
	}
	certManagerClient, err := versioned.NewForConfig(k8sClient)
	if err != nil {
		m.logger.Errorf("Failed to create cert-manager client: %v", err)
	}

	ccm := certificatecache.NewCertificateCacheManager(k8sClientSet, keyVaultClient, certManagerClient, m.logger)

	// Initialize cron
	c := cron.New()

	// Add CheckAndCacheCertificates job to run every 10 minutes
	_, err = c.AddFunc("@every 10m", func() {
		m.logger.Infof("Running CheckAndCacheCertificates() ")
		err := ccm.CheckAndCacheCertificates()
		if err != nil {
			log.Printf("Failed to check and cache certificates: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to add CheckAndCacheCertificates cron job: %v", err)
	}

	// Add CleanupExpiringCertificates job to run every 4 hours
	_, err = c.AddFunc("@every 4h", func() {
		m.logger.Infof("Running CleanupExpiringCertificates() ")
		err := ccm.CleanupExpiringCertificates()
		if err != nil {
			log.Printf("Failed to cleanup expiring certificates: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to add CleanupExpiringCertificates cron job: %v", err)
	}

	c.Start()

	err = m.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	os.Exit(0)

}
