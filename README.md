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
TODO

###Register notify endpoint with Brightcove Notification API
TODO

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
