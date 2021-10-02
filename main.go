//A collection of functions I use very often
//
// This is a convenient place to store all the functions that I use in a lot of programs.  They were useful for me, so they might be useful for you too.
package goof

//go:generate go mod init github.com/donomii/goof
//go:generate go mod tidy

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func Shell(cmd string) string {
	var result string
	switch runtime.GOOS {
	case "linux":
		log.Println("Starting ", cmd)
		result = Command("/bin/sh", []string{"-c", cmd})
		result = result + Command("cmd", []string{"/c", cmd})
	case "windows":
		cmdArray := []string{"/c", cmd}
		log.Println("Starting cmd", cmdArray)
		result = Command("c:\\Windows\\System32\\cmd.exe", cmdArray)
	case "darwin":
		result = result + Command("/bin/sh", []string{"-c", cmd})
	default:
		log.Println("unsupported platform when trying to run application")
	}
	return result
}

// Return an array of integers from min to max, so you can range over them

func Seq(min, max int) []int {
	size := max - min + 1
	if size < 1 {
		return []int{}
	}
	a := make([]int, size)
	for i := range a {
		a[i] = min + i
	}
	return a
}

//Opens a file or stdin (if filename is "").  Can open compressed files, and can decompress stdin.
//Compression is "bz2" or "gz".  Pass "" as a filename to read stdin.
func OpenInput(filename string, compression string) io.Reader {
	var f *os.File
	var err error

	var inReader io.Reader
	if filename == "" {
		f = os.Stdin
	} else {
		f, err = os.Open(filename)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		//defer f.Close()
	}

	inReader = bufio.NewReader(f)

	if (strings.HasSuffix(filename, "gz") || compression == "gz") && (!strings.HasSuffix(filename, "bz2")) {
		inReader, err = gzip.NewReader(f)
		if err != nil {
			log.Fatalf("Error ungzipping file: %v", err)
		}
	}

	if strings.HasSuffix(filename, "bz2") || compression == "bz2" {
		inReader = bzip2.NewReader(f)
	}

	return inReader
}

func AbsFloat32(x float32) float32 {
	if x < 0.0 {
		return x * -1.0
	}
	return x
}

func AbsFloat64(x float64) float64 {
	if x < 0.0 {
		return x * -1.0
	}
	return x
}

func AbsInt(x int) int {
	if x < 0.0 {
		return x * -1.0
	}
	return x
}

func OpenBufferedInput(filename string, compression string) *bufio.Reader {
	return bufio.NewReaderSize(OpenInput(filename, compression), 134217728)
}

//Takes a (c-style) filehandle, and returns go queues that let you write to and read from that handle
//Bytes will be read from the wrapped handle and written to the channels as quickly as possible, but there are no guarantees on speed or how many bytes
//are delivered per message in the channel.  This routine does no buffering, however the wrapped process can use buffers, so you still might not get prompt
//delivery of your data.  In general, most programs will use line buffering unless you can force them not to.
//
//Channel length is the buffer length of the go pipes
func WrapHandle(fileHandle uintptr, channel_length int) (chan []byte, chan []byte) {
	stdinQ := make(chan []byte, channel_length)
	stdoutQ := make(chan []byte, channel_length)

	pty := os.NewFile(fileHandle, "WrappedShell")

	go func() {
		for {
			data := <-stdinQ
			if len(data) != 0 {
				//log.Println("sent to process:", []byte(data))
				pty.Write(data)
			}
		}
	}()

	go func() {
		var data []byte = make([]byte, 1024*1024)
		for {
			count, _ := pty.Read(data)
			//if err != nil {
			//log.Fatal(err)
			//}

			if count > 0 {
				//log.Printf("read %v bytes from pty: %v,%v\n", count, string(data[:count]), []byte(data[:count]))
				//log.Println("read from process:", data)
				stdoutQ <- data[:count]
			}

		}
	}()

	return stdinQ, stdoutQ
}

//Starts a program, in the background, and returns three Go pipes of type (chan []byte), which are connected to the process's STDIN, STDOUT and STDERR.
//Bytes will be read from the wrapped program and written to the channels as quickly as possible, but there are no guarantees on speed or how many bytes
//are delivered per message in the channel.  This routine does no buffering, however the wrapped process can use buffers, so you still might not get prompt
//delivery of your data.  In general, most programs will use line buffering unless you can force them not to.
//
//Channel length is the buffer length of the go pipes
func WrapProc(pathToProgram string, channel_length int) (chan []byte, chan []byte, chan []byte) {
	cmd := exec.Command(pathToProgram)

	return WrapCmd(cmd, channel_length)
}

//Starts a program, in the background, and returns three Go pipes of type (chan []byte), which are connected to the process's STDIN, STDOUT and STDERR.
//Bytes will be read from the wrapped program and written to the channels as quickly as possible, but there are no guarantees on speed or how many bytes
//are delivered per message in the channel.  This routine does no buffering, however the wrapped process can use buffers, so you still might not get prompt
//delivery of your data.  In general, most programs will use line buffering unless you can force them not to.
//
//Channel length is the buffer length of the go pipes
func WrapCmd(cmd *exec.Cmd, channel_length int) (chan []byte, chan []byte, chan []byte) {

	stdinQ := make(chan []byte, channel_length)
	stdoutQ := make(chan []byte, channel_length)
	stderrQ := make(chan []byte, channel_length)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not open pipe: %v\n", err))
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not open pipe: %v\n", err))
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not open pipe: %v\n", err))
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(fmt.Sprintf("Could not start command: %v\n", err))
	}

	go func() {
		for {
			data := <-stdinQ
			if len(data) != 0 {
				//log.Println("sent to process:", []byte(data))
				stdin.Write(data)
			}
			//time.Sleep(10 * time.Millisecond)
		}
	}()
	rdout := bufio.NewReader(out)
	go func() {
		var data []byte = make([]byte, 1024*1024)
		for {

			if rdout.Buffered() > 0 {
				//log.Printf("%v characters ready to read from stdout:", rdout.Buffered())
			}

			count, _ := rdout.Read(data)
			/*if err != nil {
				log.Fatal(fmt.Sprintf("Could not read from process: %v.  %v\n", cmd.Path, err))
			}*/
			if count > 0 {
				//log.Printf("read %v bytes from process: %v,%v\n", count, string(data[:count]), []byte(data[:count]))

				//log.Println("read from process:", data)
				stdoutQ <- data[:count]
			}

			time.Sleep(10 * time.Millisecond) //FIXME why does this loop not block?

		}
	}()
	rderr := bufio.NewReader(errPipe)
	go func() {
		var data []byte = make([]byte, 1024)
		for {

			if rdout.Buffered() > 0 {
				//log.Printf("%v characters ready to read from stderr:", rderr.Buffered())
			}

			count, _ := rderr.Read(data)
			/*if err != nil {
				log.Fatal(fmt.Sprintf("Could not read from process: %v.  %v\n", cmd.Path, err))
			}*/
			if count > 0 {
				//log.Println("read from process:", data)
				stderrQ <- data[:count]
			}

			time.Sleep(10 * time.Millisecond) //FIXME why does this loop not block?

		}
	}()
	return stdinQ, stdoutQ, stderrQ
}

func getBody(response *http.Response, url string) ([]byte, bool) {
	var bodyText []byte
	var err error
	if response == nil {
		log.Println("Null pointer instead of respose")
		return bodyText, false
	}
	if response.Body != nil {
		defer response.Body.Close()
		bodyText, err = ioutil.ReadAll(response.Body)
	} else {
		if response.Request.Method != "PUT" {
			//No body in response to a put request
		} else {
			panic("Nil response to request")

		}
	}

	if err != nil {
		return bodyText, false
	}
	//log.Println("Response code:", response.StatusCode)
	if response.StatusCode > 399 {
		log.Printf("Unrecoverable error during http request(%v)!  Server responded with: %v, %v(%v)", url, response.StatusCode, bodyText, string(bodyText))
		panic(fmt.Sprintf("Unrecoverable error during http request(%v)!  Server responded with: %v, %v(%v)", url, response.StatusCode, bodyText, string(bodyText)))
	} else {
		//log.Printf("Result code %v is less than 400, call was probably successful", response.StatusCode)
	}
	if response.StatusCode == 200 {
		//log.Println("Status 200, call successful, returning true")
		return bodyText, true
	}
	log.Println("Call failed due to non 200 error code:", response)
	return bodyText, false

}

func SimpleGet(path string) ([]byte, error) {
	tr := &http.Transport{
		MaxIdleConns:        2,
		IdleConnTimeout:     10 * time.Second,
		DisableCompression:  true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := http.Client{Transport: tr}
	req, err := http.NewRequest("GET", path, nil)
	response, err := client.Do(req)
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	if err != nil {
		log.Println(err)
		return []byte{}, err
	}
	bodyText, ok := getBody(response, path)
	if !ok {
		log.Println("getHttp failed for", path)
		return bodyText, errors.New("getHttp failed")
	}

	return bodyText, nil
}

//Returns the directory this executable is in
func ExecutablePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}

//Delete a trailing /n, if it exists
func Chomp(s string) string {
	return strings.TrimSuffix(s, "\n")
}

//Return contents of file.  It's just ioutil.ReadFile, but with only one return value, and instead it will panic on any error
func CatFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

//Does this file or directory exist?
func Exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		//fmt.Println(path, "Exists")
		return true
	} else {
		return false
	}
}

//Write text at the end of a file.  Note that the file is opened and closed on each call, so it's not a good choice for logging..
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
	//out := []string{".."}
	out := []string{}
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		out = append(out, f.Name())
	}
	return out
}

//Is this path a directory?  Return error if error occurs (e.g. does not exist)
func IsDirr(pth string) (bool, error) {
	fi, err := os.Stat(pth)
	if err != nil {
		return false, err
	}

	return fi.Mode().IsDir(), nil
}

//Is this path a directory?  Any error results in false
func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f != nil && f.IsDir()
}

//Calculate the MD5sum of a file
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

//Run a command, wait for it to finish and then return stdout
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

//Run a command.  The first element is the path to the executable, the rest are program arguments.  Returns stdout
func QC(strs []string) string {
	cmd := exec.Command(strs[0], strs[1:]...)
	return QuickCommand(cmd)
}

//Run a command in an interactive shell.  If there isn't a terminal associated with this program, one should be opened for you.
//
//The current STDIN/OUT/ERR will be provided to the child process
func QuickCommandInteractive(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

//Run a command in an interactive shell.  If there isn't a terminal associated with this program, one should be opened for you.  The first element is the path to the executable, the rest are program arguments.  Returns stdout
func QCI(strs []string) {
	cmd := exec.Command(strs[0], strs[1:]...)
	QuickCommandInteractive(cmd)
}

//Run a command.  The first arg is the path to the executable, the rest are program args.  Returns stdout and stderr mixed together
func Command(cmd string, args []string) string {
	//log.Printf("Running '%v', '%v'", cmd, args)
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

func SplitPath(path string) []string {
	return regexp.MustCompile("\\\\|/").Split(path, -1)

}

func Clamp(a, min, max int) int {
	if a < min {
		a = min
	}
	if a > max {
		a = max
	}
	return a
}

//Searches a string to see if any lines in it match search
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

func HomeDirectory() string {
	user, _ := user.Current()
	hDir := user.HomeDir
	return hDir
}

//Searches a list of strings, return any that match search.  Case insensitive
func ListGrep(search string, strs []string) []string {
	var out = []string{}
	for _, v := range strs {
		if strings.Index(strings.ToLower(v), strings.ToLower(search)) > -1 {
			out = append(out, v)
		}
	}
	return out
}

//Searches a list of strings, return any that don't match search.  Case insensitive
func ListGrepInv(search string, strs []string) []string {
	var out = []string{}
	for _, v := range strs {
		if strings.Index(strings.ToLower(v), strings.ToLower(search)) == -1 {
			out = append(out, v)
		}
	}
	return out
}

//ASCII id -> string
func ToCharStr(i int) string {
	r := rune('A' - 1 + i)
	return fmt.Sprintf("%v", r)
}

//ASCII id -> char
func ToChar(i int) rune {
	return rune('a' + i)
}

//Build a path to a config file, from the default config location
func ConfigFilePath(filename string) string {
	user, _ := user.Current()
	hDir := user.HomeDir
	confPath := hDir + "/" + filename
	return confPath
}

//Attempt to read config from filename.  If filename does not exist, write default_config to the file and parse that data.
func ReadOrMakeConfig(filename string, default_config string) map[string]interface{} {
	var err error
	var data []byte
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Could not read config file, writing new file and returning default values(%v)", err)
		data = []byte(default_config)
		err := ioutil.WriteFile(filename, []byte(default_config), 0644)
		if err != nil {
			log.Printf("Could not write new config file, returning default values(%v)", err)
		}
	}
	var f interface{}
	err = json.Unmarshal([]byte(data), &f)
	if err != nil {
		log.Printf("Could not parse config file: %v", err)
	}
	return f.(map[string]interface{})
}

// Find a string value in the config and return it
func ConfString(f map[string]interface{}, key string, default_value string) string {
	val, err := f[key]
	if err {
		log.Printf("key '%v' not found in config file!", key)
		return default_value
	}
	return val.(string)
}

// Find an int value in the config and return it
func ConfInt(f map[string]interface{}, key string, default_value int) int {
	val, err := f[key]
	if err {
		log.Printf("key '%v' not found in config file!", key)
		return default_value
	}
	return val.(int)
}

// Find a bool value in the config and return it
func ConfBool(f map[string]interface{}, key string, default_value bool) bool {
	val, err := f[key]
	if err {
		log.Printf("key '%v' not found in config file!", key)
		return default_value
	}
	return val.(bool)
}

// Find a Float64 value in the config and return it
func ConfFloat64(f map[string]interface{}, key string, default_value float64) float64 {
	val, err := f[key]
	if err {
		log.Printf("key '%v' not found in config file!", key)
		return default_value
	}
	return val.(float64)
}

func WriteMacAgentStart(appName string) {

	execPath, _ := os.Executable()
	user, _ := user.Current()
	hDir := user.HomeDir

	ioutil.WriteFile(hDir+"/Library/LaunchAgents/`+appName+`.plist", []byte(Make_agent_plist(appName, execPath)), 0644)

}

func Make_agent_plist(appName, appPath string) string {
	template := ` <?xml version="1.0" encoding="UTF-8"?>
 <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0. dtd">
 <plist version="1.0">
 <dict>
     <key>Label</key>
     <string>` + appName + `</string>
     <key>Program</key>
     <string>` + appPath + `</string>
     <key>ProgramArguments</key>
     <array>
         <string>` + appPath + `</string>
         <string>Public</string>
     </array>
     <key>RunAtLoad</key>
     <true/>
 </dict>
 </plist>`

	return template

}
