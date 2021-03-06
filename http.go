package main

import (
	"fmt"
	"regexp"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"strconv"
	"time"
	"github.com/prometheus/common/log"
)

func extractErrorRate(reader io.Reader, config HTTPProbe) int {
	var re = regexp.MustCompile(`(\d+)]]$`)
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Errorf("Error reading HTTP body: %s", err)
		return 0
	}
	var str = string(body)
	matches := re.FindStringSubmatch(str)
	value, err := strconv.Atoi(matches[1])
	if err == nil {
		return value
	}
	return 0
}

func printRespBody(reader io.Reader) string {
	body, err:= ioutil.ReadAll(reader)
	if err != nil {
		return "Error reading HTTP body"
	}
	var str = string(body)
	return str
}
func probeHTTP(target string, w http.ResponseWriter, module Module) (success bool) {
	config := module.HTTP

	client := &http.Client{
		Timeout: module.Timeout,
	}
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	log.Infof(timestamp)
 	requestURL := config.Prefix + target + "/stats/"
	log.Infof(requestURL)
	log.Infof("URL should be https://sentry.io/api/0/projects/screenscape-networks/%s", target)
	log.Infof("Changing this back to normal now that we have proper slugs")
	request, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Errorf("Error creating request for target %s: %s", target, err)
		return
	}

	for key, value := range config.Headers {
		if strings.Title(key) == "Host" {
			request.Host = value
			continue
		}
		request.Header.Set(key, value)
	}

	resp, err := client.Do(request)
	// Err won't be nil if redirects were turned off. See https://github.com/golang/go/issues/3795
	if err != nil && resp == nil {
		log.Warnf("Error for HTTP request to %s: %s", target, err)
	} else {
		status := strconv.Itoa(resp.StatusCode)
		log.Infof(status)
		defer resp.Body.Close()
		length := strconv.Itoa(len(config.ValidStatusCodes))
		log.Infof(length)
		if len(config.ValidStatusCodes) != 0 {
			log.Infof("Attempting to loop through valid codes")
			log.Infof(strconv.Itoa(resp.StatusCode))
			for _, code := range config.ValidStatusCodes {
				log.Infof(strconv.Itoa(code))
				if resp.StatusCode == code {
					success = true
					break
				}
			}
		} else if 200 <= resp.StatusCode && resp.StatusCode < 300 {
			success = true
		}
		if success {
			fmt.Fprintf(w, "probe_sentry_error_received %d\n", extractErrorRate(resp.Body, config))
		}
	}
	if resp == nil {
		resp = &http.Response{}
	}

	fmt.Fprintf(w, "probe_sentry_status_code %d\n", resp.StatusCode)
	fmt.Fprintf(w, "probe_sentry_content_length %d\n", resp.ContentLength)

	return
}
