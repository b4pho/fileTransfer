package sftp

import (
	"encoding/base64"
	"errors"
	clientConfig "fileTransfer/configuration"
	"fileTransfer/terminal"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func getKnownhostFilename() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
}

func getPrivateKeyFilename() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
}

func addHostnameKey(host string, pubKey ssh.PublicKey) error {
	knownhostFilename := getKnownhostFilename()

	f, err := os.OpenFile(knownhostFilename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error while opening SSH knownhosts file (%s): %v", knownhostFilename, err)
	}
	defer f.Close()

	hostname := knownhosts.Normalize(host)
	hostname = knownhosts.HashHostname(hostname)
	_, err = f.WriteString(knownhosts.Line([]string{hostname}, pubKey) + "\n")
	return err
}

/*
// TODO: should I also add IP-based known_host key?
func addIPAddressKey(remote net.Addr, pubKey ssh.PublicKey) error {
	knownhostFilename := getKnownhostFilename()

	f, err := os.OpenFile(knownhostFilename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error while opening SSH knownhosts file (%s): %v", knownhostFilename, err)
	}
	defer f.Close()

	ipAddress := knownhosts.Normalize(remote.String())
	ipAddress = knownhosts.HashHostname(ipAddress)
	log.Println(ipAddress, knownhosts.Normalize(remote.String()), ipAddress)
	_, err = f.WriteString(knownhosts.Line([]string{ipAddress}, pubKey) + "\n")
	return err
}
*/

func createKnownHosts() (*string, error) {
	knownhostFilename := getKnownhostFilename()
	f, err := os.OpenFile(knownhostFilename, os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("error while opening new SSH knownhosts file(%s): %v", knownhostFilename, err)
	}
	f.Close()
	return &knownhostFilename, nil
}

func checkKnownHosts() (ssh.HostKeyCallback, error) {
	knownhostFilename, err := createKnownHosts()
	if err != nil {
		return nil, err
	}
	kh, err := knownhosts.New(*knownhostFilename)
	if err != nil {
		return nil, fmt.Errorf("error while checking SSH knownhosts file(%s): %v", *knownhostFilename, err)
	}
	return kh, nil
}

func isLocalhost(address net.Addr) bool {
	var ip string
	switch addr := address.(type) {
	case *net.UDPAddr:
		ip = addr.IP.String()
	case *net.TCPAddr:
		ip = addr.IP.String()
	}
	if ip == "127.0.0.1" || ip == "::1" {
		return true
	}
	return false
}

func Connect(sftpConfig clientConfig.Configuration, password string) (*ssh.Client, error) {
	var keyErr *knownhosts.KeyError

	privateKeyFilename := getPrivateKeyFilename()
	privateKey, err := ioutil.ReadFile(privateKeyFilename)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: sftpConfig.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
			if isLocalhost(remote) {
				log.Print("Skipping SSH known_hosts verification for localhost")
				return nil
			}
			kh, err := checkKnownHosts()
			if err != nil {
				log.Fatal(err)
			}
			hErr := kh(host, remote, pubKey)
			base64PubKey := base64.StdEncoding.EncodeToString(pubKey.Marshal())
			if errors.As(hErr, &keyErr) && len(keyErr.Want) > 0 {
				log.Printf("WARNING: %s is not a key of %s, either a MiTM attack or %s has reconfigured the host pub key.", base64PubKey, host, host)
				return keyErr
			} else if errors.As(hErr, &keyErr) && len(keyErr.Want) == 0 {
				log.Printf("WARNING: %s is not a trusted host", host)
				promptMessage := fmt.Sprintf("Do you want to add the host to known_host file?\n%s\n", base64PubKey)
				if terminal.Confirm(promptMessage) {
					return addHostnameKey(host, pubKey)
				} else {
					log.Fatal("Operation cancelled.")
				}
			}
			return nil
		}),
	}
	hostAndPort := fmt.Sprintf("%s:%d", sftpConfig.Hostname, sftpConfig.Port)
	conn, err := ssh.Dial("tcp", hostAndPort, config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
