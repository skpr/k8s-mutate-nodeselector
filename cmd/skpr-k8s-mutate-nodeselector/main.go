// Package main is the entrypoint for our program.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/christgf/env"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/skpr/k8s-mutate-nodeselector/internal/mutator"
)

var (
	cmdLong = `
		Run the mutating webhook server`

	cmdExample = `
        # Using flags.
        k8s-mutate-nodeselector --cert=/path/to/my/certificate --key=/path/to/my/key		

        # Using environment variables
        export K8S_MUTATE_NODESELECTOR_CERT=/path/to/my/certificate
        export K8S_MUTATE_NODESELECTOR_KEY=/path/to/my/key
        k8s-mutate-nodeselector`
)

// Options for this application.
type Options struct {
	Port         string
	KubeConfig   string
	CertFilePath string
	KeyFilePath  string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "k8s-mutate-nodeselector",
		Short:   "Run the mutating webhook server",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

			logger.Info("Starting server")

			config, err := clientcmd.BuildConfigFromFlags("", o.KubeConfig)
			if err != nil {
				panic(err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err)
			}

			clientset.CoreV1().Namespaces()

			mux := http.NewServeMux()

			handler := mutator.NewHandler(logger, clientset.CoreV1().Namespaces())

			mux.HandleFunc("/mutate", handler.Handle)

			s := &http.Server{
				Addr:           o.Port,
				Handler:        mux,
				ReadTimeout:    10 * time.Second,
				WriteTimeout:   10 * time.Second,
				MaxHeaderBytes: 1 << 20, // 1048576
			}

			err = s.ListenAndServeTLS(o.CertFilePath, o.KeyFilePath)
			if err != nil {
				panic(err)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&o.Port, "port", env.String("K8S_MUTATE_NODESELECTOR_PORT", ":8443"), "Port which this webserver will receive requests")
	cmd.PersistentFlags().StringVar(&o.KubeConfig, "kubeconfig", env.String("K8S_MUTATE_NODESELECTOR_KUBECONFIG", ""), "Path to the kubeconfig file")
	cmd.PersistentFlags().StringVar(&o.CertFilePath, "cert", env.String("K8S_MUTATE_NODESELECTOR_CERT", ""), "Path to the certificate file")
	cmd.PersistentFlags().StringVar(&o.KeyFilePath, "key", env.String("K8S_MUTATE_NODESELECTOR_KEY", ""), "Path to the key file")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
