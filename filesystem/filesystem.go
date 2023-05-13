package filesystem

import (
	"fileTransfer/configuration"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type FileData struct {
	AbsolutePath string    `yaml:"absolute-path"`
	RelativePath string    `yaml:"relative-path"`
	Size         int64     `yaml:"size"`
	ModTime      time.Time `yaml:"modified"`
	IsDeleted    bool      `yaml:"deleted"`
}

const fileSystemFilename = ".sftp.files"

var ignoredFiles []string = []string{configuration.Filename, fileSystemFilename}

func shouldIgnoreFile(filename string) bool {
	shouldInclude := true
	for _, ignoredFile := range ignoredFiles {
		if filepath.Base(filename) == ignoredFile {
			shouldInclude = false
			break
		}
	}
	return shouldInclude
}

type Filesystem map[string]FileData

func New() (Filesystem, error) {
	directoryPath := "." // Always check files in current directory
	newfs := Filesystem{}
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if shouldIgnoreFile(absPath) && !info.IsDir() {
			fileItem := FileData{absPath, path, info.Size(), info.ModTime(), false}
			newfs[absPath] = fileItem
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// check if old "filesystem" file was created and find deleted files
	previousFilesystem, err := readFilesystemFile()
	if err == nil {
		deletedFiles := previousFilesystem.Remove(newfs)
		for filename, fileData := range deletedFiles {
			fileData.IsDeleted = true
			fileData.ModTime = time.Now()
			newfs[filename] = fileData
		}
	}
	err = newfs.storeFile()
	if err != nil {
		return nil, fmt.Errorf("cannot store filesystem changes: %v", err)
	}
	return newfs, nil
}

func (fs Filesystem) Add(other Filesystem) Filesystem {
	newfs := Filesystem{}
	for filename, filedata := range fs {
		latestFileData := filedata
		otherFileData, ok := other[filename]
		if ok && otherFileData.ModTime.After(filedata.ModTime) {
			latestFileData = otherFileData
		}
		newfs[filename] = latestFileData
	}
	for filename, filedata := range other {
		latestFileData := filedata
		otherFileData, ok := fs[filename]
		if ok && otherFileData.ModTime.After(filedata.ModTime) {
			latestFileData = otherFileData
		}
		newfs[filename] = latestFileData
	}
	return newfs
}

func (fs Filesystem) Remove(other Filesystem) Filesystem {
	newfs := Filesystem{}
	for filename, filedata := range fs {
		_, ok := other[filename]
		if !ok {
			newfs[filename] = filedata
		}
	}
	return newfs
}

func (fs Filesystem) List(updatedSince string) ([]FileData, error) {
	lastUpdate, err := time.Parse(time.RFC3339, updatedSince)
	if err != nil {
		return nil, err
	}
	filesList := []FileData{}
	for _, fileData := range fs {
		if fileData.ModTime.After(lastUpdate) && shouldIgnoreFile(fileData.AbsolutePath) {
			filesList = append(filesList, fileData)
		}
	}
	return filesList, nil
}

// This method removes "deleted" file entries previously stored
func (fs Filesystem) Clean() error {
	toBeDeleted := []string{}
	for filename, fileData := range fs {
		if fileData.IsDeleted {
			toBeDeleted = append(toBeDeleted, filename)
		}
	}
	for _, filename := range toBeDeleted {
		delete(fs, filename)
	}
	return fs.storeFile()
}

func (fs Filesystem) storeFile() error {
	yamlData, err := yaml.Marshal(fs)
	if err != nil {
		return err
	}
	return os.WriteFile(fileSystemFilename, yamlData, 0644)
}

func readFilesystemFile() (*Filesystem, error) {
	content, err := os.ReadFile(fileSystemFilename)
	if err != nil {
		return nil, err
	}
	var filesystem Filesystem
	err = yaml.Unmarshal(content, &filesystem)
	if err != nil {
		return nil, err
	}
	return &filesystem, nil
}
