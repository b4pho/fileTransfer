package sftp

import (
	clientConfig "fileTransfer/configuration"
	files "fileTransfer/filesystem"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func copyFileToRemote(client *sftp.Client, remoteFilename string, localFile files.FileData) (int64, error) {
	destinationFile, err := client.Create(remoteFilename)
	if err != nil {
		return 0, fmt.Errorf("cannot create remote file (%s): %v", remoteFilename, err)
	}
	defer destinationFile.Close()

	sourceFile, err := os.Open(localFile.AbsolutePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open local file (%s): %v", localFile.AbsolutePath, err)
	}

	bytes, err := io.Copy(destinationFile, sourceFile)
	if err != nil {
		return 0, fmt.Errorf("cannot copy local file (%s -> %s): %v", localFile.AbsolutePath, remoteFilename, err)
	}
	err = destinationFile.Sync()
	if err != nil {
		return 0, fmt.Errorf("cannot sync remote file (%s): %v", localFile.AbsolutePath, err)
	}
	_, err = client.Lstat(remoteFilename)
	if err != nil {
		return 0, fmt.Errorf("cannot verify remote file %s: %v", remoteFilename, err)
	}
	return bytes, nil
}

func uploadFile(conn *ssh.Client, localFile files.FileData, destinationFilename string) error {
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to instantiate new SFTP client: %v", err)
	}
	defer client.Close()

	destinationDirectory, _ := filepath.Split(destinationFilename)
	err = client.MkdirAll(destinationDirectory)
	if err != nil {
		return fmt.Errorf("cannot create folder(s) (%s): %v", destinationDirectory, err)
	}
	if localFile.IsDeleted {
		err = client.Remove(destinationFilename)
		if err != nil {
			log.Printf("skipping file deletion. File '%s' not found\n", destinationFilename)
		} else {
			log.Printf("deleted file: %s ---> %s\n", localFile.AbsolutePath, destinationFilename)
		}
	} else {
		copiedBytes, err := copyFileToRemote(client, destinationFilename, localFile)
		if err != nil {
			return fmt.Errorf("cannot copy local file (%s) to remote (%s): %v", localFile.AbsolutePath, destinationFilename, err)
		}
		log.Printf("transfered file: %s ---> %s [%d bytes copied]\n", localFile.AbsolutePath, destinationFilename, copiedBytes)
	}
	return nil
}

func PushChanges(conn *ssh.Client, sftpConfig clientConfig.Configuration) error {
	fs, err := files.CreateAndStoreFileList()
	if err != nil {
		return err
	}
	filesList, err := fs.List(sftpConfig.LastUpdateDate)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	limitGuard := make(chan struct{}, sftpConfig.MaxConnections)
	for _, localFile := range filesList {
		limitGuard <- struct{}{}
		wg.Add(1)
		go func(localFile files.FileData) {
			defer wg.Done()
			destinationFilename := filepath.Join("/", filepath.Clean(sftpConfig.ServerFolder), localFile.RelativePath)
			err = uploadFile(conn, localFile, destinationFilename)
			if err != nil {
				log.Fatal(err)
			}
			<-limitGuard
		}(localFile)
	}
	wg.Wait()
	sftpConfig.UpdateTime()
	sftpConfig.Store()
	fs.Clean()
	return nil
}
