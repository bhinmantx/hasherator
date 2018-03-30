package hasherator

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tideland/golib/logger"
)

type AssetsDir struct {
	Map           map[string]string
	noHashDirList []string
	noCopyDirList []string
}

func (a *AssetsDir) Run(sourcePath, workingPath string, noHashDirs []string) error {

	err := a.RunWithTrimPath(sourcePath, workingPath, noHashDirs, "")
	if err != nil {
		return err
	}
	return nil
}

func (a *AssetsDir) RunWithTrimPath(sourcePath, workingPath string, noHashDirs []string, trimPath string) error {

	err := a.RunWithTrimPathAndIgnore(sourcePath, workingPath, noHashDirs, []string{}, trimPath, false)
	if err != nil {
		return err
	}

	return nil
}

func (a *AssetsDir) RunWithTrimPathAndIgnore(sourcePath, workingPath string, noHashDirs []string, noCopyDirs []string, trimPath string, redundant_copy bool) error {

	a.noHashDirList = noHashDirs
	a.noCopyDirList = noCopyDirs
	//err := makeAMapOfDirectoryPathsWeHate(noHashDirs)

	a.Map = map[string]string{}

	//Original code used RemoveAll but this also deleted the directory!	err := os.RemoveAll(workingPath)

	err := RemoveContents(workingPath)

	if err != nil {
		return fmt.Errorf("failed to remove working directory prior to copy: " + err.Error())
	}
	logger.Criticalf("here is a.noCopyDirList %v", a.noCopyDirList)
	err = a.recursiveHashAndCopy(sourcePath, workingPath, trimPath, redundant_copy)
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

//Redundant copy allows for all files to be transfered so it "just works" in case certain assets
//Aren't inside the hasher
func (a *AssetsDir) recursiveHashAndCopy(dirPath, runtimePath string, trimPath string, redundant_copy bool) error {
	var err error

	assetDirs, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading directory: %s", err)
	}

	for _, fileEntry := range assetDirs {

		entryName := fileEntry.Name()
		//We're not going to even copy the contents of noCopyDirs
		if fileEntry.IsDir() {
			if stringInSlice(entryName, a.noCopyDirList) {
				return err
			}
			//This is to make certain the correct directory separators are used.
			newPath := filepath.Join(runtimePath, fileEntry.Name())
			err := os.MkdirAll(newPath, 0777)
			if err != nil {
				panic("failed to make directory: " + err.Error())
			}

			err = a.recursiveHashAndCopy(filepath.Join(dirPath, fileEntry.Name()), newPath, trimPath, redundant_copy)
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

			//Searching for an excluded foldername. Has a bug where duplicate foldernames in a nested directory
			//result in a no-hash
			for _, excluded := range a.noHashDirList {
				if stringInSlice(excluded, dir) {
					noHash = true
				}
			}

			if !noHash {
				h := md5.Sum(file)
				hash = fmt.Sprintf("-%x", string(h[:16]))
			}

			err = copyFile(filepath.Join(dirPath, fileEntry.Name()), fmt.Sprintf("%s%s%s%s", filepath.Join(runtimePath, entryName), hash, dot, fileExtension))
			//temporary test for transitional states:
			if !noHash && redundant_copy {
				err = copyFile(filepath.Join(dirPath, fileEntry.Name()), fmt.Sprintf("%s%s%s", filepath.Join(runtimePath, entryName), dot, fileExtension))
			}
			if err != nil {
				return fmt.Errorf("failed to return file: " + err.Error())
			}
			//Just by filename:
			a.Map[fileEntry.Name()] = fmt.Sprintf("%s%s%s%s", entryName, hash, dot, fileExtension)
			//This supports the use of the full path
			//need to make sure the filepath key is set for web path type slashes
			hashfilepathname := fmt.Sprintf("%s%s%s%s", filepath.Join(dirPath, entryName), hash, dot, fileExtension)
			hashfilepathname = strings.Replace(hashfilepathname, string(filepath.Separator), "/", -1)
			filepathname := filepath.Join(dirPath, fileEntry.Name())
			filepathname = strings.Replace(filepathname, string(filepath.Separator), "/", -1)
			//here is where you could clip the part of the directory structure you want
			filepathname = trimDirectoryPath(filepathname, trimPath)
			hashfilepathname = trimDirectoryPath(hashfilepathname, trimPath)
			a.Map[filepathname] = hashfilepathname
		}
	}
	return nil
}

//This allows for the trimming of a path down to the relative directory structure, so that a folder doesn't need to be at the
//root of your web app's directory
func trimDirectoryPath(path string, trim string) (trimmed_string string) {
	trimmed_string = strings.Replace(path, trim, "", -1)
	//To avoid double slashes
	trimmed_string = strings.Replace(trimmed_string, "//", "/", -1)
	return
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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func makeAMapOfDirectoryPathsWeHate(noHashDirs []string) (err error) {
	for _, file := range noHashDirs {
		files, err := ioutil.ReadDir(file)

		if err != nil {
			logger.Errorf("failed in the read: %s", err.Error())
		}

		for _, f := range files {
			if f.IsDir() {
				fmt.Println(f.Name())
			}
		}
		logger.Debugf("thats a loop for %s", file)
	}
	return
}
