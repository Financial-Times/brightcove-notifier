package main

import (
	"fmt"
	"net/http"

	"github.com/Financial-Times/go-fthealth"
)

func (bn brightcoveNotifier) health() func(w http.ResponseWriter, r *http.Request) {
	return fthealth.HandlerParallel("Dependent services healthcheck", "Checks if all the dependent services are reachable and healthy.",
		bn.cmsNotifierReachable(), bn.brightcoveAPIReachable(), bn.brightcoveAPIRenewingAccessTokenWorks())
}

func (bn brightcoveNotifier) gtg(w http.ResponseWriter, r *http.Request) {
	healthChecks := []func() error{bn.checkCmsNotifierHealth, bn.checkBrightcoveAPIReachable, bn.checkAccessTokenIsValid}

	for _, hCheck := range healthChecks {
		if err := hCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
}

func (bn brightcoveNotifier) cmsNotifierReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Notifications about newly modified/published videos will not reach UPP stack.",
		Name:             "CMS Notifier Reachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/brightcove-notifier-runbook",
		Severity:         1,
		TechnicalSummary: "CMS Notifier is not reachable/healthy",
		Checker:          bn.checkCmsNotifierHealth,
	}
}

func (bn brightcoveNotifier) checkCmsNotifierHealth() error {
	req, err := http.NewRequest("GET", bn.cmsNotifierConf.addr+"/__health", nil)
	if err != nil {
		return err
	}
	if (bn.cmsNotifierConf.hostHeader != "") {
		req.Header.Add("Host", bn.cmsNotifierConf.hostHeader)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", bn.cmsNotifierConf.auth)

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("Unhealthy status code received: [%d]", resp.StatusCode)
	}
	return nil
}

func (bn brightcoveNotifier) brightcoveAPIReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Video models of newly modified/published videos could not be fetched.",
		Name:             "Brightcove API Reachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/brightcove-notifier-runbook",
		Severity:         1,
		TechnicalSummary: "Brightcove API is not reachable/healthy",
		Checker:          bn.checkBrightcoveAPIReachable,
	}
}

func (bn brightcoveNotifier) checkBrightcoveAPIReachable() error {
	req, err := http.NewRequest("GET", bn.brightcoveConf.addr+bn.brightcoveConf.accountID+"/counts/videos", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+bn.brightcoveConf.accessToken)

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 && resp.StatusCode != 401 {
		return fmt.Errorf("Invalid status code received: [%d]", resp.StatusCode)
	}
	return nil
}

// This check tests the highly unlikely scenario of Brightcove OAuth API sending us invalid access tokens
func (bn brightcoveNotifier) brightcoveAPIRenewingAccessTokenWorks() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Video models of newly modified/published videos could not be fetched.",
		Name:             "Brightcove API credentials are valid",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/brightcove-notifier-runbook",
		Severity:         1,
		TechnicalSummary: "Brightcove API returns invalid access token.",
		Checker:          bn.checkAccessTokenIsValid,
	}
}

type brightcoveAPIAccessTokenHealthCheck struct {
	bn    brightcoveNotifier
	calls int
}

func (bn brightcoveNotifier) checkAccessTokenIsValid() error {
	hc := &brightcoveAPIAccessTokenHealthCheck{bn, 0}
	return hc.checkBrightcoveAPIReturnsValidAccessToken()
}

func (hc brightcoveAPIAccessTokenHealthCheck) checkBrightcoveAPIReturnsValidAccessToken() error {
	if hc.calls == 2 {
		return fmt.Errorf("Video publishing won't work. Access token is not valid.")
	}
	req, err := http.NewRequest("GET", hc.bn.brightcoveConf.addr+hc.bn.brightcoveConf.accountID+"/counts/videos", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+hc.bn.brightcoveConf.accessToken)

	resp, err := hc.bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)

	switch resp.StatusCode {
	case 401:
		infoLogger.Println("Renewing access token.")
		err = hc.bn.renewAccessToken()
		if err != nil {
			err = fmt.Errorf("Video publishing won't work. Renewing access token failure: [%v].", err)
			warnLogger.Println(err)
			return err
		}
		hc.calls++
		return hc.checkBrightcoveAPIReturnsValidAccessToken()
	case 200:
		return nil
	default:
		return fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}
}
