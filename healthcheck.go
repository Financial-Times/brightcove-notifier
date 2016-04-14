package main

import (
	"fmt"
	"net/http"

	"github.com/Financial-Times/go-fthealth"
)

func (bn brightcoveNotifier) health() func(w http.ResponseWriter, r *http.Request) {
	return fthealth.HandlerParallel("Dependent services healthcheck", "Checks if all the dependent services are reachable and healthy.", bn.cmsNotifierReachable(), bn.brightcoveAPIReachable())
}

func (bn brightcoveNotifier) gtg(w http.ResponseWriter, r *http.Request) {
	healthChecks := []func() error{bn.checkCmsNotifierGTG, bn.checkBrightcoveAPIReachable}

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
		Checker:          bn.checkCmsNotifierGTG,
	}
}

func (bn brightcoveNotifier) checkCmsNotifierGTG() error {
	req, err := http.NewRequest("GET", bn.cmsNotifierConf.addr+"/__gtg", nil)
	if err != nil {
		return err
	}
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
	req, err := http.NewRequest("GET", bn.brightcoveConf.hcAddr, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+bn.brightcoveConf.accessToken)

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer cleanupResp(resp)

	switch resp.StatusCode {
	case 401:
		err = bn.renewAccessToken()
		if err != nil {
			err = fmt.Errorf("Video publishing won't work. Renewing access token failure: [%v].", err)
			return err
		}
		return bn.checkBrightcoveAPIReachable()
	case 200:
		return nil
	default:
		return fmt.Errorf("Invalid statusCode received: [%d]", resp.StatusCode)
	}
}
