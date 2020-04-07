package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	resticCheckExitCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "restic_check_exit_code",
			Help: "restic check command exit code result.",
		},
		[]string{"prefix"},
	)
)

func resticCheck() {
	// override restic executable path
	resticExecPath := os.Getenv("RESTIC_BINARY_PATH")
	if resticExecPath == "" {
		resticExecPath = "/usr/local/bin/restic"
	}

	godotenv.Load("/etc/default/restic")
	godotenv.Load("/etc/default/restic_exporter")

	prefixes := strings.Split(os.Getenv("RESTIC_CHECK_PREFIXES"), ",")

	go func() {
		for {
			for _, prefix := range prefixes {
				os.Setenv(
					"RESTIC_REPOSITORY",
					fmt.Sprintf("%s/%s", os.Getenv("RESTIC_REPOSITORY_BUCKET"), prefix))

				// 5 attempts before failed state return
				for i := 0; i < 5; i++ {
					cmd := exec.Command(resticExecPath, "check")
					err := cmd.Run()
					if err != nil {
						log.Printf("check failed: %s\n", err)
					} else {
						resticCheckExitCode.WithLabelValues(prefix).Set(0)
						break
					}

					if i < 4 {
						time.Sleep(5 * time.Second)
					} else {
						resticCheckExitCode.WithLabelValues(prefix).Set(1)
					}
				}
			}

			time.Sleep(30 * time.Second)
		}
	}()
}

func init() {
	prometheus.MustRegister(resticCheckExitCode)
}

func main() {
	resticCheck()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("exporter started")
	http.ListenAndServe(":9707", nil)
}
