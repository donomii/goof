package goof

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Exists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	} else {
		return false
	}
}

func AppendStringToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}

//List all files in a directory, and recursively in its subdirectories
func LslR(dir string) []string {
	out := []string{}
	walkHandler := func(path string, info os.FileInfo, err error) error {
		out = append(out, path)
		return nil
	}
	filepath.Walk(dir, walkHandler)
	return out
}

//List all files in a directory
func Ls(dir string) []string {
	out := []string{".."}
	files, _ := ioutil.ReadDir(".")
	for _, f := range files {
		out = append(out, f.Name())
	}
	return out
}

func IsDirr(pth string) (bool, error) {
	fi, err := os.Stat(pth)
	if err != nil {
		return false, err
	}

	return fi.Mode().IsDir(), nil
}

func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f != nil && f.IsDir()
}

func Hash_file_md5(filePath string) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	//Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	//Tell the program to call the following function when the current function returns
	defer file.Close()

	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil

}

func QuickCommand(cmd *exec.Cmd) string {
	in := strings.NewReader("")
	cmd.Stdin = in
	var out bytes.Buffer
	cmd.Stdout = &out
	var err bytes.Buffer
	cmd.Stderr = &err
	cmd.Run()
	//fmt.Printf("Command result: %v\n", res)
	ret := out.String()
	//fmt.Println(ret)
	return ret
}

func QC(strs []string) string {
	cmd := exec.Command(strs[0], strs[1:]...)
	return QuickCommand(cmd)
}

func QuickCommandInteractive(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func QCI(strs []string) {
	cmd := exec.Command(strs[0], strs[1:]...)
	QuickCommandInteractive(cmd)
}

func Command(cmd string, args []string) string {
	log.Printf("Running '%v', '%v'", cmd, args)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		//fmt.Fprintf(os.Stderr, "IO> %v\n", string(out))
		//fmt.Fprintf(os.Stderr, "E> %v\n", err)
		//os.Exit(1)
	}
	if string(out) != "" {
		//fmt.Fprintf(os.Stderr, "O> %v\n\n", string(out))
	}
	return string(out)
}

func Grep(search, str string) string {
	var out string
	strs := strings.Split(str, "\n")
	for _, v := range strs {
		if strings.Index(v, search) > -1 {
			out = out + v + "\n"
		}
	}
	return out
}

func ListGrep(search string, strs []string) []string {
	var out = []string{}
	for _, v := range strs {
		if strings.Index(strings.ToLower(v), strings.ToLower(search)) > -1 {
			out = append(out, v)
		}
	}
	return out
}

func ListGrepInv(search string, strs []string) []string {
	var out = []string{}
	for _, v := range strs {
		if strings.Index(strings.ToLower(v), strings.ToLower(search)) == -1 {
			out = append(out, v)
		}
	}
	return out
}

func ToCharStr(i int) string {
	return string('A' - 1 + i)
}

func ToChar(i int) rune {
	return rune('a' + i)
}
