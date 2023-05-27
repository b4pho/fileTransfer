package ftp

import (
	clientConfig "fileTransfer/configuration"
	"fmt"
	"log"
	"os"

	"path/filepath"

	"github.com/secsy/goftp"

	files "fileTransfer/filesystem"

	"path"

	"sync/atomic"

	"strings"
)

func listFiles(client *goftp.Client, ftpConfig clientConfig.Configuration) ([]files.FileData, error) {
	list := []files.FileData{}
	rootFolder := ftpConfig.ServerFolder

	// It seems that forward slashes are also used on Windows FTP servers BTW
	if !strings.HasPrefix(rootFolder, string(os.PathSeparator)) {
		rootFolder = string(os.PathSeparator) + rootFolder
	}

	err := walk(client, rootFolder, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			// no permissions is okay, keep walking
			if err.(goftp.Error).Code() == 550 {
				return nil
			}
			return err
		}

		if !info.IsDir() {
			filedata := files.FileData{
				AbsolutePath: fullPath,
				RelativePath: fullPath,
				Size:         info.Size(),
				ModTime:      info.ModTime(),
				IsDeleted:    false,
			}
			list = append(list, filedata)
		}

		return nil
	})
	return list, err
}

func walk(client *goftp.Client, root string, walkFn filepath.WalkFunc) (ret error) {
	dirsToCheck := make(chan string, 100)

	var workCount int32 = 1
	dirsToCheck <- root

	for dir := range dirsToCheck {
		go func(dir string) {
			files, err := client.ReadDir(dir)

			if err != nil {
				if err = walkFn(dir, nil, err); err != nil && err != filepath.SkipDir {
					ret = err
					close(dirsToCheck)
					return
				}
			}

			for _, file := range files {
				if err = walkFn(path.Join(dir, file.Name()), file, nil); err != nil {
					if file.IsDir() && err == filepath.SkipDir {
						continue
					}
					ret = err
					close(dirsToCheck)
					return
				}

				if file.IsDir() {
					atomic.AddInt32(&workCount, 1)
					dirsToCheck <- path.Join(dir, file.Name())
				}
			}

			atomic.AddInt32(&workCount, -1)
			if workCount == 0 {
				close(dirsToCheck)
			}
		}(dir)
	}

	return ret
}

func Clone(conn *goftp.Client, ftpConfig clientConfig.Configuration) error {
	remoteFiles, err := listFiles(conn, ftpConfig)
	if err != nil {
		return fmt.Errorf("cannot list all remote files: %v", err)
	}

	currentDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %v", err)
	}
	serverFolder := filepath.Join("/", filepath.Clean(ftpConfig.ServerFolder))

	for _, remoteFile := range remoteFiles {
		relativePath, err := filepath.Rel(serverFolder, remoteFile.AbsolutePath)
		if err != nil {
			log.Fatal(err)
		}
		clientPath := filepath.Join(currentDirectory, relativePath)
		err = downloadFile(conn, clientPath, remoteFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func downloadFile(conn *goftp.Client, localFilename string, remoteFile files.FileData) error {
	localFilePath, _ := filepath.Split(localFilename)
	if err := os.MkdirAll(localFilePath, os.ModePerm); err != nil {
		return fmt.Errorf("cannot create folder(s) (%s): %v", localFilePath, err)
	}

	localFile, err := os.Create(localFilename)
	if err != nil {
		return err
	}

	err = conn.Retrieve(remoteFile.AbsolutePath, localFile)
	if err != nil {
		return err
	}

	return nil
}
