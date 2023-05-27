package protocols

import (
	clientConfig "fileTransfer/configuration"
	"fileTransfer/protocols/ftp"
	"fileTransfer/protocols/sftp"

	"errors"

	"github.com/secsy/goftp"
	"golang.org/x/crypto/ssh"
)

type ProtocolClient interface {
	Close() error
}

func Connect(config clientConfig.Configuration, password string) (ProtocolClient, error) {
	switch config.Protocol {
	case "SFTP":
		return sftp.Connect(config, password)
	case "FTP", "FTPS-IMPLICIT", "FTPS-EXPLICIT":
		return ftp.Connect(config, password)
	default:
		return nil, raiseUnexpectedProtocolError(config)
	}
}

func Clone(conn ProtocolClient, config clientConfig.Configuration) error {
	switch config.Protocol {
	case "SFTP":
		return sftp.Clone(conn.(*ssh.Client), config)
	case "FTP", "FTPS-IMPLICIT", "FTPS-EXPLICIT":
		return ftp.Clone(conn.(*goftp.Client), config)
	default:
		return raiseUnexpectedProtocolError(config)
	}
}

func PushChanges(conn ProtocolClient, config clientConfig.Configuration) error {
	switch config.Protocol {
	case "SFTP":
		return sftp.PushChanges(conn.(*ssh.Client), config)
	case "FTP", "FTPS-IMPLICIT", "FTPS-EXPLICIT":
		return ftp.PushChanges(conn.(*goftp.Client), config)
	default:
		return raiseUnexpectedProtocolError(config)
	}
}

func raiseUnexpectedProtocolError(config clientConfig.Configuration) error {
	return errors.New("Unexpected protocol: " + config.Protocol)
}
