#UPP Brightcove Notifier

[![Circle CI](https://circleci.com/gh/Financial-Times/brightcove-notifier/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/brightcove-notifier/tree/master)

Receives Brightcove video notification events. Fetches video model, then creates UPP publish event and posts it to the CMS-Notifier.

##Build & Run the binary

```bash
export BRIGHTCOVE_AUTH="Basic YjY..."
export BRIGHTCOVE_ACCOUNT_ID="47628783001"
export CMS_NOTIFIER="https://pub-xp-up.ft.com/__cms-notifier"
export CMS_NOTIFIER_HOST_HEADER="cms-notifier"
export CMS_NOTIFIER_AUTH="Basic dXB..."
go build; ./brightcove-notifier

```

Look for the auth values in LastPass' UPP Shared Folder.

##Endpoints

* /notify

POST endpoint (registered with Brightcove CMS Notifications API)
* /force-notify/{videoID}

POST endpoint (useful for forcing video model publishes)
* /__health

GET endpoint (FT standard)
* /__gtg

GET endpoint (FT standard)


##Testing

###Locally

videoEvent.json
```json
{
	"timestamp": 1423840514446,
	"account_id": "421252784301",
	"event": "video-change",
	"video": "4144892532001",
	"version": 26
}
```

request
```bash
curl localhost:8080/notify -X POST -d @videoEvent.json -i
```

###Heroku
Sometimes it's handy to test against the Brightcove Notifications API directly.
To create a public endpoint the most easy way is to deploy your app in Heroku. 
Assuming you have an account create the Procfile and run godep:

1. ```echo "web: brightcove-notifier" > Procfile```

1. ```godep save ./...``` 

1. Commit your changes, then push to heroku

Now you have a public endpoint, you can use the FT Development account to register this endpoint with Brightcove Notifications API. Once you've done with testing, please unregister your endpoint from Brightcove.

##Brightcove Integration

###Obtain Client Credentials
```
curl -i -XPOST \
   -H"Authorization:Basic ..." \
   -H"Content-Type:application/x-www-form-urlencoded" \
   -d"grant_type=client_credentials" \
   "https://oauth.brightcove.com/v3/access_token?client_id=2221711291001"
```

you get an access token that you can use for a few minutes:

```
{
  "access_token": "AK5366ogmVGB-eix...",
  "token_type": "Bearer",
  "expires_in": 300
}
```

###Register notify endpoint with Brightcove Notification API

Let's say your client_id is 2221711291001. You could check your subscriptions:

```
curl -i \
  -H "Authorization:Bearer AK5366ogmVGB-eix..." \
  https://cms.api.brightcove.com/v1/accounts/2221711291001/subscriptions
```

It will give you a response like:

```
[
  {
    "service_account": "2221711291001",
    "id": "5e831b0d-9d45-43b1-9464-86421e0feb4d",
    "events": "video-change",
    "endpoint": "https://pub-xp-up.ft.com/notification/brightcove/metadata"
  },
  {
    "service_account": "2221711291001",
    "id": "c456f1c2-f682-4a56-a276-93eab2075e87",
    "events": "video-change",
    "endpoint": "https://pub-xp-up.ft.com/notification/brightcove/content"
  },
  {
    "service_account": "2221711291001",
    "id": "7492891d-f012-46af-b571-060ae180bfbe",
    "events": "video-change",
    "endpoint": "https://brightcove-notifier-up-test.ft.com/notify"
  }
]
```

Then you can create a new subscription:

```
curl -i -XPOST \
  -H "Authorization:Bearer AK5366oD..." \
  -H"Content-Type:application/json" \
  -d'{"endpoint":"https://pub-pre-prod-up.ft.com/notification/brightcove/content", "events":["video-change"]}' \
  "https://cms.api.brightcove.com/v1/accounts/2221711291001/subscriptions"
```

```
{
  "service_account": "2221711291001",
  "id": "a35aa9c7-cefa-40dd-9222-e7216bccfa13",
  "events": [
    "video-change"
  ],
  "endpoint": "https://pub-pre-prod-up.ft.com/notification/brightcove/content"
}
```

Check your subscriptions again.

###Integration points

List of used Brightcove API endpoints (please keep this list updated):

1. [Get Video by ID](http://docs.brightcove.com/en/video-cloud/cms-api/references/cms-api/versions/v1/index.html#api-videoGroup-Get_Video_by_ID_or_Reference_ID)
1. [Get Video Count - used in healthcheck](http://docs.brightcove.com/en/video-cloud/cms-api/references/cms-api/versions/v1/index.html#api-videoGroup-Get_Video_Count)
1. [Creating notification subscription - called once, manually, before deployment](http://docs.brightcove.com/en/video-cloud/cms-api/references/cms-api/versions/v1/index.html#api-notificationGroup-Create_Subscription)
1. [OAuth API Renewing Access Token](http://docs.brightcove.com/en/video-cloud/oauth-api/reference/versions/v4/index.html)

##Reference

Read more on Brightcove API:

1. [CMS Notifications API Guide](http://docs.brightcove.com/en/video-cloud/media-management/guides/notifications.html#cmsAPI)
1. [CMS Notifications API Reference](http://docs.brightcove.com/en/video-cloud/cms-api/references/cms-api/versions/v1/index.html#api-notificationGroup)
1. [CMS Video API Reference](http://docs.brightcove.com/en/video-cloud/cms-api/references/cms-api/versions/v1/index.html#api-videoGroup)
1. [OAuth API Overview](http://docs.brightcove.com/en/video-cloud/oauth-api/getting-started/oauth-api-overview.html)
1. [OAuth API Getting Client Credentials](http://docs.brightcove.com/en/video-cloud/oauth-api/guides/get-client-credentials.html)
1. [OAuth API Getting Access Token](http://docs.brightcove.com/en/video-cloud/oauth-api/guides/get-token.html)
