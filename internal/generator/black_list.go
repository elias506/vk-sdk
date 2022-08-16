package generator

type skipReason string

var easyJSONBlackList = map[string]skipReason{
	"notifications_notification_parent": "Ambiguous reference 'Date'",
	"newsfeed_item_wallpost":            "Ambiguous reference 'Date'",
}
