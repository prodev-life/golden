package fsys

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func DoesDirExists(dir string) bool {
	file, err := os.Open(dir)

	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func DoesFileExists(filename string) bool {
	file, err := os.Open(filename)

	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func GetDirs(dir string) ([]string, error) {
	files := make([]string, 0, 10)
	f, err := os.Open(dir)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		if !file.IsDir() {
			continue
		}

		files = append(files, filepath.Join(dir, file.Name()))
	}

	return files, nil
}

func GetFiles(dir string) ([]string, error) {
	files := make([]string, 0, 10)
	f, err := os.Open(dir)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		if file.IsDir() {
			continue
		}

		files = append(files, file.Name())
	}

	return files, nil
}

func GetAllFilesRecursive(dir string) ([]string, error) {
	files := make([]string, 0, 10)
	bfsList := []string{dir}
	for ;len(bfsList) != 0; {
		dir = bfsList[0]
		bfsList = bfsList[1:]
		f, err := os.Open(dir)
		if err != nil {
			continue
		}

		fileInfos, err := f.Readdir(-1)
		_ = f.Close()

		if err != nil {
			return nil, err
		}

		for _, file := range fileInfos {
			if file.IsDir() {
				bfsList = append(bfsList, filepath.Join(dir, file.Name()))
				continue
			}
			files = append(files, filepath.Join(dir, file.Name()))
		}
	}

	return files, nil
}

func MustGetWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func EnsurePathExists(path string) {
	if path == "" {
		panic("EnsurePathExists: path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(fmt.Errorf("EnsurePathExists: os.Stat: %s", err.Error()))
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			panic(fmt.Errorf("EnsurePathExists: os.MkdirAll: %s: %s", err, path))
		}
		return
	}

	if !info.IsDir() {
		panic(fmt.Errorf("EnsurePathExists: %s is a file", path))
	}
}

func EnsureDirExists(path string) {
	if path == "" {
		panic("EnsureDirExists: path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(fmt.Errorf("EnsureDirExists: os.Stat: %s", err.Error()))
		}
		if err := os.Mkdir(path, 0755); err != nil {
			panic(fmt.Errorf("EnsureDirExists: os.Mkdir: %s: %s", err.Error(), path))
		}
		return
	}

	if !info.IsDir() {
		panic(fmt.Errorf("EnsureDirExists: %s is a file", path))
	}
}

// WriteToFile will attempt to copy all bytes from io.Reader to a file created
// at path. If path already exists, it will be overriden. In case io.Copy
// errors out, file will be removed.
func WriteToFile(from io.Reader, path string) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()
	_, err = io.Copy(file, from)
	if err != nil {
		os.Remove(path)
	}
	return
}

func IsDir(file *os.File) (answer bool, err error) {
	info, err := file.Stat()
	if err != nil {
		return
	}
	return info.IsDir(), nil
}

func FileSize(path string) (size int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	info, err := file.Stat()
	if err != nil {
		return
	}
	return info.Size(), nil
}