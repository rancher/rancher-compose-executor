package composinator

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	v3 "github.com/rancher/go-rancher/v3"
)

const (
	cattleExportListenPort = "CATTLE_EXPORT_LISTEN_PORT"
)

var rancherClient *v3.RancherClient

type convertOptions struct {
	StackID string `json:"stackId,omitempty" yaml:"stackId,omitempty"`
	Format  string `json:"format,omitempty" yaml:"format,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		d, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body from request", http.StatusBadRequest)
			return
		}
		var input convertOptions
		if err := json.Unmarshal(d, &input); err != nil {
			http.Error(w, err.Error()+": can't unmarshall body from request", http.StatusBadRequest)
			return
		}
		convert(w, rancherClient, input)
	}
}

func getRancherClient() (*v3.RancherClient, error) {
	apiURL := os.Getenv("CATTLE_URL")
	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	apiClient, err := v3.NewRancherClient(&v3.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       apiURL,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return nil, err
	}
	return apiClient, nil
}

func StartServer() error {
	rc, err := getRancherClient()
	if err != nil {
		return err
	}
	rancherClient = rc
	router := mux.NewRouter()
	router.HandleFunc("/convert", handler).Methods("POST")
	listenPort := os.Getenv(cattleExportListenPort)
	if listenPort == "" {
		listenPort = "8099"
	}
	logrus.Infof("starting composinator on %v", listenPort)
	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%s", listenPort), router)
}
