package convert

import (
	"fmt"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/lookup"
)

func populateCerts(resourceLookup lookup.ServerResourceLookup, lbService *client.Service, defaultCert string, certs []string) error {
	if defaultCert != "" {
		certId, err := findCertByName(resourceLookup, defaultCert)
		if err != nil {
			return err
		}
		lbService.LbConfig.DefaultCertificateId = certId
	}

	certIds := []string{}
	for _, certName := range certs {
		certId, err := findCertByName(resourceLookup, certName)
		if err != nil {
			return err
		}
		certIds = append(certIds, certId)
	}
	lbService.LbConfig.CertificateIds = certIds

	return nil
}

func findCertByName(resourceLookup lookup.ServerResourceLookup, name string) (string, error) {
	cert, err := resourceLookup.Cert(name)
	if err != nil {
		return "", err
	}

	if cert == nil {
		return "", fmt.Errorf("Failed to find certificate %s", name)
	}

	return cert.Id, nil
}
