package ftp

import (
	clientConfig "fileTransfer/configuration"
	"log"
	"os"
	"strings"

	"path/filepath"

	"github.com/secsy/goftp"

	files "fileTransfer/filesystem"
)

func PushChanges(conn *goftp.Client, ftpConfig clientConfig.Configuration) error {
	fs, err := files.CreateAndStoreFileList()
	if err != nil {
		return err
	}
	filesList, err := fs.List(ftpConfig.LastUpdateDate)
	if err != nil {
		return err
	}
	for _, localFile := range filesList {
		destinationFilename := filepath.Join("/", filepath.Clean(ftpConfig.ServerFolder), localFile.RelativePath)
		err = uploadFile(conn, localFile, destinationFilename)
		if err != nil {
			log.Fatal(err)
		}
	}
	ftpConfig.UpdateTime()
	ftpConfig.Store()
	fs.Clean()
	return nil
}

func remoteMkdirAll(conn *goftp.Client, path string) {
	path, _ = filepath.Split(path)
	folders := strings.Split(path, string(os.PathSeparator))
	incrementalPath := string(os.PathSeparator)
	for _, folder := range folders {
		incrementalPath = filepath.Join(incrementalPath, folder)
		if incrementalPath == string(os.PathSeparator) || folder == "" {
			continue
		}
		// NOTE: ingnoring errors as I they are rarely helpful
		conn.Mkdir(incrementalPath)
	}
}

func uploadFile(conn *goftp.Client, localFileData files.FileData, remoteFilename string) error {
	localFile, err := os.Open(localFileData.AbsolutePath)
	if err != nil {
		return err
	}

	// Check if path DOESN'T exist "try" to create it.
	remoteMkdirAll(conn, remoteFilename)
	err = conn.Store(remoteFilename, localFile)
	if err != nil {
		return err
	}

	return nil
}
