package goof

import (
	"io"
	"os"
	"strings"

	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os/exec"
)

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
		if strings.Index(v, search) == 0 {
			out = out + v + "\n"
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
