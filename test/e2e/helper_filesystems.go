package e2e

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"gotest.tools/assert"
)

func CreateNewContext() string {
	directory, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatalf("Error: Failed to create new context %+v", err)
	}
	log.Printf("Created dir: %s\n", directory)
	return directory
}

func DeleteDir(dir string) {
	log.Printf("Deleting dir: %s\n", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("Error: Failed to Delete dir %+v", err)
	}

}

// Chdir change current working dir
func Chdir(dir string) {
	log.Printf("Setting current dir to: %s\n", dir)
	err := os.Chdir(dir)
	if err != nil {
		log.Fatalf("Error: Failed to Change dir %+v", err)
	}
}

// Getwd retruns current working dir

func Getwd() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error: Failed to get pwd %+v", err)
	}
	log.Printf("Current working dir: %s\n", dir)
	return dir
}

// CopyExample copies an example from tests/e2e/examples/<exampleName> into targetDir
func CopyExample(exampleName string, targetDir string, t *testing.T) {
	// filename of this file
	_, filename, _, _ := runtime.Caller(0)
	// path to the examples directory
	examplesDir := filepath.Join(filepath.Dir(filename), "..", "examples")

	src := filepath.Join(examplesDir, exampleName)
	info, err := os.Stat(src)
	assert.NilError(t, err)

	err = copyDir(src, targetDir, info)
	assert.NilError(t, err)
}

// ReplaceString replaces oldString with newString in text file
func ReplaceString(filename string, oldString string, newString string, t *testing.T) {
	fmt.Fprintf(os.Stdout, "Replacing \"%s\" with \"%s\" in %s\n", oldString, newString, filename)
	f, err := ioutil.ReadFile(filename)
	assert.NilError(t, err)

	newContent := strings.Replace(string(f), oldString, newString, 1)

	err = ioutil.WriteFile(filename, []byte(newContent), 0644)
	assert.NilError(t, err)
}

// copyDir copy one directory to the other
// this function is called recursively info should start as os.Stat(src)
func copyDir(src string, dst string, info os.FileInfo) error {

	if info.IsDir() {
		files, err := ioutil.ReadDir(src)
		if err != nil {
			return err
		}

		for _, file := range files {
			dsrt := filepath.Join(src, file.Name())
			ddst := filepath.Join(dst, file.Name())
			if err := copyDir(dsrt, ddst, file); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	dFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dFile.Close()

	sFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sFile.Close()

	if err = os.Chmod(dFile.Name(), info.Mode()); err != nil {
		return err
	}

	_, err = io.Copy(dFile, sFile)
	return err
}

// func hasElement(s interface{}, elem interface{}) bool {
// 	arrV := reflect.ValueOf(s)

// 	if arrV.Kind() == reflect.Slice {
// 		for i := 0; i < arrV.Len(); i++ {
// 			if arrV.Index(i).Interface() == elem {
// 				return true
// 			}
// 		}
// 	}

// 	return false
// }

// func containsString(sl []string, v string) bool {
// 	for _, vv := range sl {
// 		x := strings.Join(strings.Fields(vv), "")
// 		y := strings.Join(strings.Fields(v), "")
// 		if x == y {
// 			return true
// 		}
// 	}
// 	return false
// }

func Process(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}

func ProcessString(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)
	if err != nil {
		panic(err)
	}
	return Process(tmpl, vars)
}
