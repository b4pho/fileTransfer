package ftp

import (
	"errors"
	clientConfig "fileTransfer/configuration"
	"fmt"
	"os"
	"time"

	"crypto/tls"

	"github.com/secsy/goftp"
)

func Connect(ftpConfig clientConfig.Configuration, password string) (*goftp.Client, error) {
	tlsConfig := tls.Config{
		InsecureSkipVerify:     true,
		ServerName:             ftpConfig.Hostname,
		MaxVersion:             tls.VersionTLS12,
		ClientAuth:             tls.RequestClientCert,
		SessionTicketsDisabled: false,
		ClientSessionCache:     tls.NewLRUClientSessionCache(0),
	}
	logger := os.Stderr
	if !ftpConfig.DebugMode {
		logger = nil
	}
	config := goftp.Config{
		User:               ftpConfig.Username,
		Password:           password,
		Timeout:            10 * time.Second,
		Logger:             logger,
		ConnectionsPerHost: ftpConfig.MaxConnections,
	}
	switch ftpConfig.Protocol {
	case "FTPS-IMPLICIT":
		config.TLSMode = goftp.TLSImplicit
		config.TLSConfig = &tlsConfig
	case "FTP-EXPLICIT":
		config.TLSMode = goftp.TLSExplicit
		config.TLSConfig = &tlsConfig
	case "FTP":
	default:
		return nil, errors.New("Unexpected protocol type: " + ftpConfig.Protocol)
	}
	hostnameAndPort := fmt.Sprintf("%s:%d", ftpConfig.Hostname, ftpConfig.Port)
	client, err := goftp.DialConfig(config, hostnameAndPort)
	if err != nil {
		return nil, fmt.Errorf("unable to connect: %v", err)
	}
	return client, nil
}
