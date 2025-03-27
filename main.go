package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const VERSION = "1.0.0"

var (
	hostMappings = map[string]string{
		"cache.ott.ystenlive.itv.cmvideo.cn": "",
		"cache.ott.bestlive.itv.cmvideo.cn":  "",
		"cache.ott.wasulive.itv.cmvideo.cn":  "",
		"cache.ott.fifalive.itv.cmvideo.cn":  "",
		"cache.ott.hnbblive.itv.cmvideo.cn":  "",
	}

	iparray [16 * 1024]string
	ipno    = 0
)

func getHTTPResponse(requestURL string, testip string) (string, string, error) {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	// 自定义resolver
	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			for originalHost := range hostMappings {
				if strings.Contains(address, originalHost) {
					if testip != "" {
						newAddress := strings.Replace(address, originalHost, testip, 1)
						fmt.Println(address + "->" + newAddress)
						address = newAddress
					}
				}
			}
			return dialer.DialContext(ctx, network, address)
		},
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: resolver.Dial,
		},

		// //Disable automatic redirect
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// 	return http.ErrUseLastResponse
		// },
	}

	fmt.Println("RequestURL: " + requestURL)
	resp, err := client.Get(requestURL)

	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", http.ErrServerClosed
	}

	redirectURL := resp.Header.Get("Location")
	if redirectURL == "" {
		redirectURL = requestURL
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return "", "", err
	}

	return body, redirectURL, nil
}

func readResponseBody(resp *http.Response) (string, error) {
	var builder strings.Builder
	_, err := io.Copy(&builder, resp.Body)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func dotest(myid string, cdn string, testip string) {
	startUrl := "http://gslbserv.itv.cmvideo.cn:80/" + myid + "/1.m3u8?channel-id=" + cdn + "&Contentid=" + myid + "&livemode=1&stbId=003803ff00010060180758b42d777238"
	_, _, err := getHTTPResponse(startUrl, testip)
	if err != nil {
		fmt.Println(testip, "failed")
	} else {
		fmt.Println(testip, "success")
	}
}

//This method parses the filter file
func ParseFilterFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(f)
	for {
		line, _, err := reader.ReadLine()
		if nil != err {
			break
		}
		str := strings.TrimSpace(string(line))
		if str != "" {
			iparray[ipno] = str
			ipno++
		}

		if ipno >= 16*1024 {
			break
		}
	}

	fmt.Println("loaded", ipno, "ips")
	return nil
}

func main() {
	fmt.Println("VERSION:", VERSION)
	fmt.Println("itvtester <ipFilePath> [myid] [cdn]")

	arg_num := len(os.Args)

	if arg_num < 2 {
		return
	}

	ipFilePath := os.Args[1]

	myid := "5000000004000002226"
	cdn := "bestzb"

	if arg_num >= 4 {
		myid = os.Args[2]
		cdn = os.Args[3]
	}

	err := ParseFilterFile(ipFilePath)

	if err != nil {
		fmt.Println("Fail to load ip file", err)
		return
	}

	for i := 0; i < ipno; i++ {
		dotest(myid, cdn, iparray[i])
		time.Sleep(1 * time.Second)
	}
}
