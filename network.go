package goof

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

/*
package main

import (
	"log"
	"time"

	"github.com/donomii/goof"
)

func main() {
	go func() {
		time.Sleep(10 * time.Second)
		goof.AdvertiseMDNS(80, "test._workstation._tcp", "local", "test server", []string{"lalala"}, 120, false)
	}()
	c := goof.StartMDNSscan("_services._dns-sd._udp", "local", -1)
	goof.ScanMDNS(c, "_workstation._tcp", "local", -1)
	goof.ScanMDNS(c, "_udisks-ssh._tcp", "local", -1)
	goof.ScanMDNS(c, "_ssh._tcp", "local", -1)
	goof.ScanMDNS(c, "_tcp", "local", -1)
	goof.ScanMDNS(c, "_udp", "local", -1)
	for x := range c {
		log.Println(x)
	}
}
*/

func WrappedTraceroute(target string) []string {
	out := []string{}
	raw := QC([]string{"traceroute", "-n", "-m", "3", "-q", "1", "-P", "icmp", "8.8.8.8"})
	hops := Grep("ms", raw)
	for _, l := range strings.Split(hops, "\n") {
		bits := strings.Split(l, "  ")
		if len(bits) > 1 {
			ip := bits[1]
			ip = strings.Trim(ip, " \t\r\n")
			fmt.Printf("IP '%v'\n", ip)
			out = append(out, ip)
		}
	}
	return out
}

//Try to find the network interface that is connected to the internet, and get its IP address
//This does not find the IP of your firewall or WAN connection, just the IP on the network that
//you are directly connected to
func ExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func AllIps() []string {
	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	out := []string{}
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			fmt.Println("IPv4: ", ipv4)
			out = append(out, fmt.Sprintf("%v", ipv4))
		}
	}
	return out
}

//Attempt to connect to PORT on every IP address in your class C network
func ScanHosts(timeout, port int, outch chan string) {
	ScanHostsRec(timeout, port, 0, outch)
}

//Attempt to connect to PORT on every IP address in your class C network
func ScanHostsRec(timeout, port, elapsed int, outch chan string) {
	if timeout > 0 && elapsed > timeout {
		//panic("Cannot find any server!")
		return
	}
	ip, err := ExternalIP()
	if err == nil {

		log.Printf("Found base IP number: %v\n", ip)
		//log.Printf("Using timeout: %v\n", timeout)
		ip_chunks := strings.Split(ip, ".")
		classC := strings.Join(ip_chunks[:3], ".")
		//log.Printf("IP: %v\n", classC)

		for jj := 1; jj < 255; jj++ {
			j := 0 + jj
			go func() {
				testIP := fmt.Sprintf("%v.%v", classC, j)
				connectString := fmt.Sprintf("http://%v:%v/", testIP, port)
				resp, err := http.Get(connectString)
				if err == nil {
					if resp.StatusCode < 300 {
						log.Printf("Found server at: %v\n", testIP)
						outch <- connectString
					}
				}

			}()
		}

		time.Sleep(5 * time.Second)
	} else {
		fmt.Println("NO NETWORK")
		log.Println("NO NETWORK")
	}
	//fmt.Println("Finished scan")
	ScanHostsRec(timeout, port, elapsed+1000, outch)
}


//search e.g. "_workstation._tcp" or _services._dns-sd._udp
//domain e.g. "local"
//waitTime e.g. -1
func ScanMDNSCallback(found func(*zeroconf.ServiceEntry) bool, search, domain string, waitTime int) context.Context {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)

	var ctx context.Context
	var cancel context.CancelFunc
	if waitTime > -1 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*time.Duration(waitTime))
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}
	go func(results chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Printf("ScanMDNS found: %+v\n", entry)
			contin :=found( entry)
			if !contin {return }
			if ctx.Err != nil {
				return
			}
		}
		log.Println("ScanMDNS: No more entries.")
	}(entries)
	err = resolver.Browse(ctx, search, domain, entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	return ctx
}



//search e.g. "_workstation._tcp" or _services._dns-sd._udp
//domain e.g. "local"
//waitTime e.g. -1
func ScanMDNS(found chan *zeroconf.ServiceEntry, search, domain string, waitTime int) {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Printf("ScanMDNS found: %+v\n", entry)
			found <- entry
		}
		log.Println("ScanMDNS: No more entries.")
	}(entries)
	var ctx context.Context
	var cancel context.CancelFunc
	if waitTime > -1 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*time.Duration(waitTime))
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}
	err = resolver.Browse(ctx, search, domain, entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
}

//search e.g. "_workstation._tcp" or _services._dns-sd._udp
//domain e.g. "local"
//waitTime e.g. 10
func StartMDNSscan(search, domain string, waitTime int) chan *zeroconf.ServiceEntry {
	found := make(chan *zeroconf.ServiceEntry, 100)
	go ScanMDNS(found, search, domain, waitTime)
	return found
}

//Attempt to get the primary network address
func GetOutboundIP() (localAddr net.IP) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	stuff := conn.LocalAddr().(*net.UDPAddr)
	localAddr = stuff.IP

	return
}

//service e.g. "_workstation._tcp"
//domain e.g. "local"
//serverPort: e.g. 80
//name: e.g. "Totally awesome server"
//payload: will be delivered verbatim to the client
//sleepTime:  Time between advertisements
//logAdvertisement: log each time we advertise
func AdvertiseMDNS(serverPort int, service, domain, name string, payload []string, sleepTime int, logAdvertisement bool) {
	for {

		if logAdvertisement {
			log.Printf("Published service - Name: %v, Type: %v, Domain: %v, Port: %v\n", name, service, domain, serverPort)
		}
		server, err := zeroconf.Register(name, service, domain, serverPort, payload, nil)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Duration(sleepTime) * time.Second)
		server.Shutdown()
	}
}

//Register programPath with Windows UAC so it can listen on network ports.  programName is a descriptive name
func OpenFirewall(programPath, programName string) {
	ioutil.WriteFile("firewall.bat", firewallbat(programPath, programName), 0644) //FIXME temp filenames
	QC([]string{"firewall.bat"})
	go func() {
		time.Sleep(30 * time.Second)
		os.Remove("firewall.bat")
	}()
}

func firewallbat(programPath, programName string) []byte {
	return []byte(strings.Join([]string{
		`@echo off
	
:: BatchGotAdmin
:-------------------------------------
REM  --> Check for permissions
    IF "%PROCESSOR_ARCHITECTURE%" EQU "amd64" (
>nul 2>&1 "%SYSTEMROOT%\SysWOW64\cacls.exe" "%SYSTEMROOT%\SysWOW64\config\system"
) ELSE (
>nul 2>&1 "%SYSTEMROOT%\system32\cacls.exe" "%SYSTEMROOT%\system32\config\system"
)

REM --> If error flag set, we do not have admin.
if '%errorlevel%' NEQ '0' (
    echo Requesting administrative privileges...
    goto UACPrompt
) else ( goto gotAdmin )

:UACPrompt
    echo Set UAC = CreateObject^("Shell.Application"^) > "%temp%\getadmin.vbs"
    set params= %*
    echo UAC.ShellExecute "cmd.exe", "/c ""%~s0"" %params:"=""%", "", "runas", 1 >> "%temp%\getadmin.vbs"

    "%temp%\getadmin.vbs"
    del "%temp%\getadmin.vbs"
    exit /B

:gotAdmin
    pushd "%CD%"
    CD /D "%~dp0"
:--------------------------------------
netsh firewall add allowedprogram %cd%\`,
		programPath,
		` "`,
		programName,
		`" ENABLE
`,
	}, ""))
}
