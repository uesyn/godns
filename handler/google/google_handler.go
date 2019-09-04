package google

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/TimothyYe/godns"

	"golang.org/x/net/proxy"
)

var (
	// GoogleUrl the API address for Google Domains
	GoogleUrl = "https://domains.google.com/nic/update"
)

// Handler struct
type Handler struct {
	Configuration *godns.Settings
}

// SetConfiguration pass dns settings and store it to handler instance
func (handler *Handler) SetConfiguration(conf *godns.Settings) {
	handler.Configuration = conf
}

// DomainLoop the main logic loop
func (handler *Handler) DomainLoop(domain *godns.Domain, panicChan chan<- godns.Domain) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered in %v: %v\n", err, debug.Stack())
			panicChan <- *domain
		}
	}()

	for {
		currentIP, err := godns.GetCurrentIP(handler.Configuration)

		if err != nil {
			log.Println("get_currentIP:", err)
			continue
		}
		log.Println("currentIP is:", currentIP)

		for _, subDomain := range domain.SubDomains {
			var resolvedIP string
			fqdn := subDomain + "." + domain.DomainName
			resolvedIPs, err := net.LookupHost(fqdn)
			if err != nil {
				log.Printf("couldn't lookup %s\n", fqdn)
			}

			if resolvedIPs != nil && len(resolvedIPs) == 1 {
				resolvedIP = resolvedIPs[0]
			}

			if currentIP == resolvedIP {
				log.Println("skip due to current IP == resolved IP")
				continue
			}

			log.Printf("%s.%s Start to update record IP...\n", subDomain, domain.DomainName)
			handler.UpdateIP(domain.DomainName, subDomain, currentIP)

			// Send mail notification if notify is enabled
			if handler.Configuration.Notify.Enabled {
				log.Print("Sending notification to:", handler.Configuration.Notify.SendTo)
				if err := godns.SendNotify(handler.Configuration, fmt.Sprintf("%s.%s", subDomain, domain.DomainName), currentIP); err != nil {
					log.Println("Failed to send notificaiton")
				}
			}
		}

		// Sleep with interval
		log.Printf("Going to sleep, will start next checking in %d seconds...\r\n", handler.Configuration.Interval)
		time.Sleep(time.Second * time.Duration(handler.Configuration.Interval))
	}

}

// UpdateIP update subdomain with current IP
func (handler *Handler) UpdateIP(domain, subDomain, currentIP string) {
	u, err := url.Parse(GoogleUrl)
	if err != nil {
		log.Fatalln("Failed to parse URL: ", GoogleUrl)
	}

	q := u.Query()
	q.Set("hostname", fmt.Sprintf("%s.%s", subDomain, domain))
	q.Set("myip", currentIP)
	u.RawQuery = q.Encode()
	u.User = url.UserPassword(handler.Configuration.Email, handler.Configuration.Password)

	client := &http.Client{}

	if handler.Configuration.Socks5Proxy != "" {
		log.Println("use socks5 proxy:" + handler.Configuration.Socks5Proxy)
		dialer, err := proxy.SOCKS5("tcp", handler.Configuration.Socks5Proxy, nil, proxy.Direct)
		if err != nil {
			log.Println("can't connect to the proxy:", err)
			return
		}

		httpTransport := &http.Transport{}
		client.Transport = httpTransport
		httpTransport.Dial = dialer.Dial
	}

	req, _ := http.NewRequest("GET", u.String(), nil)
	resp, err := client.Do(req)

	if err != nil {
		log.Println("Request error...")
		log.Println("Err:", err.Error())
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			log.Println("Update IP success:", string(body))
		} else {
			log.Println("Update IP failed:", string(body))
		}
	}
}
