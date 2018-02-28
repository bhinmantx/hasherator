package hasherator

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type AssetsDir struct {
	Map           map[string]string
	noHashDirList []string
}

func (a *AssetsDir) Run(sourcePath, workingPath string, noHashDirs []string) error {

	a.noHashDirList = noHashDirs

	a.Map = map[string]string{}

	//Original code used RemoveAll but this also deleted the directory!	err := os.RemoveAll(workingPath)

	err := RemoveContents(workingPath)

	if err != nil {
		return fmt.Errorf("failed to remove working directory prior to copy: " + err.Error())
	}

	err = a.recursiveHashAndCopy(sourcePath, workingPath)
	if err != nil {
		return err
	}

	return nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AssetsDir) recursiveHashAndCopy(dirPath, runtimePath string) error {
	var err error

	assetDirs, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading directory: %s", err)
	}

	for _, fileEntry := range assetDirs {

		entryName := fileEntry.Name()

		if fileEntry.IsDir() {
			//This is to make certain the correct directory separators are used.
			newPath := filepath.Join(runtimePath, fileEntry.Name())
			err := os.MkdirAll(newPath, 0777)
			if err != nil {
				panic("failed to make directory: " + err.Error())
			}

			err = a.recursiveHashAndCopy(filepath.Join(dirPath, fileEntry.Name()), newPath)
			if err != nil {
				return err
			}

		} else {

			var fileExtension string
			var dot string
			if strings.Contains(entryName, ".") {
				fileExtension = entryName[strings.LastIndex(entryName, ".")+1:]
				entryName = entryName[:strings.LastIndex(entryName, ".")]
				dot = "."
			}

			file, err := ioutil.ReadFile(filepath.Join(dirPath, fileEntry.Name()))

			if err != nil {
				return fmt.Errorf("failed to read file: " + err.Error())
			}

			var hash string
			var noHash bool

			dir := strings.Split(runtimePath, string(os.PathSeparator))
			for _, noDir := range a.noHashDirList {
				if noDir == dir[len(dir)-1] {
					noHash = true
				}
			}

			if !noHash {
				h := md5.Sum(file)
				hash = fmt.Sprintf("-%x", string(h[:16]))
			}

			err = copyFile(filepath.Join(dirPath, fileEntry.Name()), fmt.Sprintf("%s%s%s%s", filepath.Join(runtimePath, entryName), hash, dot, fileExtension))
			if err != nil {
				return fmt.Errorf("failed to return file: " + err.Error())
			}

			a.Map[fileEntry.Name()] = fmt.Sprintf("%s%s%s%s", entryName, hash, dot, fileExtension)
			a.Map[filepath.Join(dirPath, fileEntry.Name())] = fmt.Sprintf("%s%s%s%s", filepath.Join(runtimePath, entryName), hash, dot, fileExtension)
		}
	}
	return nil
}

func copyFile(src, dst string) error {

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)

	return err
}
