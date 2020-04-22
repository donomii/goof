package goof

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

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

func ScanHosts(timeout, port int, outch chan string) string {
	if timeout > 3000 {
		//panic("Cannot find any server!")
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
	return ScanHosts(timeout+1000, port, outch)

}

//search e.g. "_workstation._tcp"
//domain e.g. "local"
//waitTime e.g. 10
func ScanMDNS(found chan []string, search, domain string, waitTime int) {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Printf("%+v\n", entry)
			found <- entry.Text
		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(waitTime))
	defer cancel()
	err = resolver.Browse(ctx, search, domain, entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
}

//search e.g. "_workstation._tcp"
//domain e.g. "local"
//waitTime e.g. 10
func StartMDNSscan(search, domain string, waitTime int) chan []string {
	found := make(chan []string, 0)
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

//search e.g. "_workstation._tcp"
//domain e.g. "local"
//waitTime e.g. 10
//serverPort: e.g. 80
//name: e.g. "Totally awesome server"
//payload: will be delivered verbatim to the client
func AdvertiseMDNS(serverPort int, service, domain, name string, payload []string) {
	for {
		server, err := zeroconf.Register(name, service, domain, serverPort, payload, nil)
		if err != nil {
			panic(err)
		}

		if false {
			log.Println("Published service:")
			log.Println("- Name:", name)
			log.Println("- Type:", service)
			log.Println("- Domain:", domain)
			log.Println("- Port:", serverPort)

			log.Println("Advertising")
		}
		time.Sleep(5 * time.Second)
		server.Shutdown()
	}
}
