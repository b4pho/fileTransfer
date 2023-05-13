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

func listFiles(client *sftp.Client, remoteDir string) ([]files.FileData, error) {
	remoteFiles, err := client.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("unable to list remote dir: %v", err)
	}

	var listedFiles []files.FileData
	for _, f := range remoteFiles {
		if !f.IsDir() {
			listedFiles = append(listedFiles, files.FileData{
				RelativePath: f.Name(),
				AbsolutePath: filepath.Join(remoteDir, f.Name()),
				Size:         f.Size(),
				ModTime:      f.ModTime(),
			})
		} else {
			remoteFiles2, err := listFiles(client, filepath.Join(remoteDir, f.Name()))
			if err != nil {
				return nil, err
			}
			listedFiles = append(listedFiles, remoteFiles2...)
		}
	}

	return listedFiles, nil
}

func copyFileToLocal(client *sftp.Client, localFile string, remoteFile files.FileData) (int64, error) {
	destinationFile, err := os.Create(localFile)
	if err != nil {
		return 0, fmt.Errorf("cannot open local file (%s): %v", localFile, err)
	}
	defer destinationFile.Close()

	sourceFile, err := client.Open(remoteFile.AbsolutePath)
	if err != nil {
		return 0, fmt.Errorf("cannot open remote file (%s): %v", remoteFile.AbsolutePath, err)
	}
	bytes, err := io.Copy(destinationFile, sourceFile)
	if err != nil {
		return 0, fmt.Errorf("cannot copy remote file (%s -> %s): %v", remoteFile.AbsolutePath, localFile, err)
	}

	err = destinationFile.Sync()
	if err != nil {
		return 0, fmt.Errorf("cannot sync local file (%s): %v", localFile, err)
	}
	if _, err := os.Stat(localFile); err != nil {
		return 0, fmt.Errorf("cannot verify local file %s: %v", localFile, err)
	}
	return bytes, nil
}

func downloadFile(conn *ssh.Client, localFilename string, remoteFile files.FileData) (err error) {
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to instantiate new SFTP client: %v", err)
	}
	defer client.Close()

	localFilePath, _ := filepath.Split(localFilename)
	if err := os.MkdirAll(localFilePath, os.ModePerm); err != nil {
		return fmt.Errorf("cannot create folder(s) (%s): %v", localFilePath, err)
	}
	copiedBytes, err := copyFileToLocal(client, localFilename, remoteFile)
	if err != nil {
		return fmt.Errorf("cannot copy remote file (%s) to local (%s): %v", remoteFile.AbsolutePath, localFilename, err)
	}
	log.Printf("transfered file: %s ---> %s [%d bytes copied]\n", remoteFile.AbsolutePath, localFilename, copiedBytes)

	return nil
}

func getAllRemoteFiles(conn *ssh.Client, sftpConfig clientConfig.Configuration) ([]files.FileData, error) {
	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	serverFolder := filepath.Join("/", sftpConfig.ServerFolder)
	files, err := listFiles(client, serverFolder)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func Clone(conn *ssh.Client, sftpConfig clientConfig.Configuration) error {
	remoteFiles, err := getAllRemoteFiles(conn, sftpConfig)
	if err != nil {
		return fmt.Errorf("cannot list all remote files: %v", err)
	}

	currentDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %v", err)
	}
	serverFolder := filepath.Join("/", filepath.Clean(sftpConfig.ServerFolder))

	var wg sync.WaitGroup
	limitGuard := make(chan struct{}, sftpConfig.MaxConnections)

	for _, remoteFile := range remoteFiles {
		limitGuard <- struct{}{}
		wg.Add(1)

		go func(remoteFile files.FileData) {
			defer wg.Done()
			relativePath, err := filepath.Rel(serverFolder, remoteFile.AbsolutePath)
			if err != nil {
				log.Fatal(err)
			}
			clientPath := filepath.Join(currentDirectory, relativePath)
			err = downloadFile(conn, clientPath, remoteFile)
			if err != nil {
				log.Fatal(err)
			}
			<-limitGuard
		}(remoteFile)
	}
	wg.Wait()
	return nil
}
