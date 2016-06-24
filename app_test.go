package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRenewAccessToken_HappyScenario_NewTokenIsSavedOnModel(t *testing.T) {
	currentAccToken := "AIMofDb6D0wOG8JLGTU0Uahl8ckx6yfTdTO7OHeI-tZ4lSqQaSE2sh3K8gb9sSK7uzGMPVSU-RQilr_5chv5-n-XsVgHG05BBnHdUW08jN5Wu0NaR-AOuIpM0cT-dyemA5HiSwsty0EsczI3oi9LE5m_lqjPYjfozOu-gWJbeGU8IM1IzcVvSSzUOCIhNkPVIkRkdYNSwkP0yC0b8QYIyI89oQdFAi4VI1-jaqvZtvWueixUUJ-xkCQxpHdQsR6pWtZIWxlfrZQOq4CjfjQJSf7lz1CWsXlEHsxEr3kwC8UvXZsyTsMhRlltsAxBHtfAyNzhJunFgiuVFlo_Yk0jzI4xVBRQfE7iPLdRJlsSVKh2_bcUy5wXdfM"
	nextAccToken := "AIMofDZb0Z2SbUCHPuy-VKFhVO3aW5tZVRuUyDJDxsNsLfn7GgXnDYQE0GLMy5s2YPsoi-wlNUlJteKD5WzRzqWmHrUpS6tb6jjKxiTjoa2KHccUxd0HY5OoqbP3qW5IFyoRC517IY4kQW2RvuHsGPHfNerJoPbA7sz5iZYhkJ6vEhUgbb2Sus_peENtCwmXb4nexUzYlUCvRjI6GJnfzDCwRPLGMa2xmSxjeWkJfBjAd3BijJvyiWEFbeyFGg0YDqIH5rczgGVO1A1ZmOtQTVQoF_p9SykM8xhdm6mwJVn-M7H2a5gp2UONxafDqmcCpmRVJ-ahOqeZTlfP6zVN8g1zLdNKQIz1gaxNv2R0gyoCre0lfbDJj-8"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, buildTestAccessTokenResponse(nextAccToken))
	}))
	defer ts.Close()

	bn := &brightcoveNotifier{
		brightcoveConf: &brightcoveConfig{
			oauthAddr:   ts.URL,
			accessToken: currentAccToken,
		},
		client: &http.Client{},
	}

	err := bn.renewAccessToken()
	if err != nil {
		t.Fatalf("[%v]", err)
	}

	if bn.brightcoveConf.accessToken != nextAccToken {
		t.Fatalf("Expected new access token to be available on brightcove model.\nExpected: [%s].\nActual: [%s]", nextAccToken, bn.brightcoveConf.accessToken)
	}
}

func TestRenewAccessToken_InvalidResponseStatusCode_ErrorIsReturned(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	bn := &brightcoveNotifier{
		brightcoveConf: &brightcoveConfig{
			oauthAddr: ts.URL,
		},
		client: &http.Client{},
	}
	err := bn.renewAccessToken()
	if err == nil {
		t.Fatal("Expected error.")
	}
}

func TestRenewAccessToken_AccessTokenFieldIsMissing_ErrorIsReturned(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"token_type": "Bearer", "expires_in": 300)`)
	}))
	defer ts.Close()

	bn := &brightcoveNotifier{
		brightcoveConf: &brightcoveConfig{
			oauthAddr: ts.URL,
		},
		client: &http.Client{},
	}
	err := bn.renewAccessToken()
	if err == nil {
		t.Fatal("Expected error.")
	}
}

func TestRenewAccessToken_AccessTokenFieldIsEmpty_ErrorIsReturned(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, buildTestAccessTokenResponse(""))
	}))
	defer ts.Close()

	bn := &brightcoveNotifier{
		brightcoveConf: &brightcoveConfig{
			oauthAddr: ts.URL,
		},
		client: &http.Client{},
	}
	err := bn.renewAccessToken()
	if err == nil {
		t.Fatal("Expected error.")
	}
}

func buildTestAccessTokenResponse(accToken string) string {
	return fmt.Sprintf(`{"access_token": "%s","token_type": "Bearer","expires_in": 300}`, accToken)
}

func TestFwdVideo_RequestContainsXOriginSystemHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Origin-System-Id") == "" {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	bn := &brightcoveNotifier{
		cmsNotifierConf: &cmsNotifierConfig{
			addr: ts.URL,
		},
		client: &http.Client{},
	}

	video := make(map[string]interface{})
	err := bn.fwdVideo(video, "tid_test")
	if err != nil {
		t.Fatalf("Expected success. Received: [%v]", err)
	}
}

func TestAddUPPRequiredFields_IDExists_ValidUUIDIsAddedToThePayload(t *testing.T) {
	video := make(map[string]interface{})
	video["id"] = "4492075574001"
	err := addUPPRequiredFields(video)
	if err != nil {
		t.Fatalf("[%v]", err)
	}
	if uuid, present := video["uuid"]; !present || uuid == "" {
		t.Fatalf("Expected valid uuid to be found in the map. Actual map: [%v]", video)
	}
}

func TestAddUPPRequiredFields_IDDoesNotExists_ErrorIsReturned(t *testing.T) {
	video := make(map[string]interface{})
	video["name"] = "foobar"
	err := addUPPRequiredFields(video)
	if err == nil {
		t.Fatal("Expected failure")
	}
}

func TestAddUPPRequiredFields_TypeIsAdded(t *testing.T) {
	video := make(map[string]interface{})
	video["id"] = "4492075574001"
	err := addUPPRequiredFields(video)
	if err != nil {
		t.Fatalf("[%v]", err)
	}
	if typ, present := video["type"]; !present || typ != "video" {
		t.Fatalf("Expected 'type' field to be set. Actual value: [%v]", typ)
	}
}

func TestHandleNotification_Integration_Return200StatusCode(t *testing.T) {
	accID := "775205503001"
	videoID := "4020894387001"
	bn := &brightcoveNotifier{
		client: &http.Client{},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchPath := fmt.Sprintf("/accounts/%s/videos/%s", accID, videoID)
		switch r.URL.Path {
		case "/notify":
			bn.handleNotification(w, r)
		case fetchPath:
			_, err := w.Write([]byte(buildTestVideoModel(accID, videoID)))
			if err != nil {
				warnLogger.Printf("Could not write response: [%v]", err)
			}
		case "/cms-notifier/notify":
			//do nothing, just return 200
		default:
			w.WriteHeader(http.StatusNotFound)
		}

	}))
	bn.brightcoveConf = &brightcoveConfig{
		addr: ts.URL + "/accounts/",
	}
	bn.cmsNotifierConf = &cmsNotifierConfig{
		addr: ts.URL + "/cms-notifier",
	}

	res, err := http.Post(ts.URL+"/notify", "application/json", bytes.NewReader([]byte(buildTestVideoEvent(accID, videoID))))
	if err != nil {
		t.Fatalf("[%v]", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("Expected success. Received status code: [%d]", res.StatusCode)
	}
}

func TestHandleNotification_Integration_VideoModelWithUUIDReachesCMSNotifier(t *testing.T) {
	accID := "775205503001"
	videoID := "4020894387001"
	testVideoModel := buildTestVideoModel(accID, videoID)
	bn := &brightcoveNotifier{
		client: &http.Client{},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchPath := fmt.Sprintf("/accounts/%s/videos/%s", accID, videoID)
		switch r.URL.Path {
		case "/notify":
			bn.handleNotification(w, r)
		case fetchPath:
			_, err := w.Write([]byte(testVideoModel))
			if err != nil {
				warnLogger.Printf("Could not write response: [%v]", err)
			}
		case "/cms-notifier/notify":
			err := receivedVideoModelMatchesFetchedVideoAndUUIDIsPresent(w, r, []byte(testVideoModel))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(err.Error()))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	bn.brightcoveConf = &brightcoveConfig{
		addr: ts.URL + "/accounts/",
	}
	bn.cmsNotifierConf = &cmsNotifierConfig{
		addr: ts.URL + "/cms-notifier",
	}

	res, err := http.Post(ts.URL+"/notify", "application/json", bytes.NewReader([]byte(buildTestVideoEvent(accID, videoID))))
	if err != nil {
		t.Fatalf("[%v]", err)
	}

	if res.StatusCode != 200 {
		msgBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("[%v]", err)
		}
		t.Fatalf("Expected success. Received status code: [%d]. Response body: [%s]", res.StatusCode, string(msgBytes))
	}
}

func TestFetchVideo_404VideoNotFound_VideoIdAndNotFoundMessageIsPresent(t *testing.T) {
	ts := mockBrightcoveServer(`[{ "error_code": "RESOURCE_NOT_FOUND" }]`)
	bn := &brightcoveNotifier{
		client: &http.Client{},
		brightcoveConf: &brightcoveConfig{
			addr: ts.URL,
		},
	}
	videoID := "4020894387001"
	v, err := bn.fetchVideo(videoEvent{Video: videoID}, "tid_test")
	if err != nil {
		t.Fatalf("Expected success. Received error: [%v]", err)
	}
	if v["id"] != videoID || v["error_code"] != "RESOURCE_NOT_FOUND" {
		t.Fatalf("Unexpected id or error_code. Found: [%#v]", v)
	}
}

func TestFetchVideo_404ButUnmarshallableResponse_ErrorIsReturned(t *testing.T) {
	ts := mockBrightcoveServer(`[ "error_code", "RESOURCE_NOT_FOUND" ]`)
	bn := &brightcoveNotifier{
		client: &http.Client{},
		brightcoveConf: &brightcoveConfig{
			addr: ts.URL,
		},
	}

	videoID := "4020894387001"
	_, err := bn.fetchVideo(videoEvent{Video: videoID}, "tid_test")
	if err == nil {
		t.Fatalf("Expected failure")
	}
}

func TestFetchVideo_404ButUnexpectedResponse_ErrorIsReturned(t *testing.T) {
	ts := mockBrightcoveServer(`[]`)
	bn := &brightcoveNotifier{
		client: &http.Client{},
		brightcoveConf: &brightcoveConfig{
			addr: ts.URL,
		},
	}

	videoID := "4020894387001"
	_, err := bn.fetchVideo(videoEvent{Video: videoID}, "tid_test")
	if err == nil {
		t.Fatalf("Expected failure")
	}
}

func mockBrightcoveServer(mockVideoResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(mockVideoResponse))
		if err != nil {
			warnLogger.Printf("Can't write response: [%v]", err)
		}
	}))
}

func receivedVideoModelMatchesFetchedVideoAndUUIDIsPresent(w http.ResponseWriter, r *http.Request, fetchedVideoModel []byte) error {
	var received video
	err := json.NewDecoder(r.Body).Decode(&received)
	if err != nil {
		return err
	}

	var fetched video
	err = json.Unmarshal(fetchedVideoModel, &fetched)
	if err != nil {
		return err
	}

	for k, v := range fetched {
		if !reflect.DeepEqual(received[k], v) {
			return fmt.Errorf("Discrepancy found: Fetched value [%v] differs from received [%v]", v, received[k])
		}
	}

	if uuid, present := received["uuid"]; !present || uuid == "" {
		return fmt.Errorf("UUID is missing or is empty")
	}

	return nil
}

func buildTestVideoEvent(accID, videoID string) string {
	return fmt.Sprintf(`{"timestamp":1423840514446,"account_id":"%s","event":"video-change","video":"%s","version":26}`, accID, videoID)
}

func buildTestVideoModel(accID, videoID string) string {
	return fmt.Sprintf(
		`{
		    "account_id": "%s",
		    "complete": true,
		    "created_at": "2015-09-17T16:08:37.108Z",
		    "cue_points": [],
		    "custom_fields": {},
		    "description": null,
		    "digital_master_id": "4492154733001",
		    "duration": 155573,
		    "economics": "AD_SUPPORTED",
		    "folder_id": null,
		    "geo": null,
		    "id": "%s",
		    "images": {},
		    "link": null,
		    "long_description": null,
		    "name": "sea_marvels.mp4",
		    "reference_id": null,
		    "schedule": null,
		    "sharing": null,
		    "state": "ACTIVE",
		    "tags": [],
		    "text_tracks": [],
		    "updated_at": "2015-09-17T17:41:20.782Z"
		}`, accID, videoID)
}
