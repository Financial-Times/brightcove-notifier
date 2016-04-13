package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRenewAccessToken_NewTokenIsSavedOnModel(t *testing.T) {
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
		t.Fatal("Expected new access token to be available on brightcove model.\nExpected: [%s].\nActual: [%s]", nextAccToken, bn.brightcoveConf.accessToken)
	}
}

func buildTestAccessTokenResponse(accToken string) string {
	return fmt.Sprintf(`{"access_token": "%s","token_type": "Bearer","expires_in": 300}`, accToken)
}
