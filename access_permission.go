package vk_sdk

// AccessPermission define possibility of using token for one or another data section.
// E.g., to send private message token should be obtained with messages scope.
//
// Sum of their bit masks in AuthorizeRequest.Scope parameter while obtaining an access token.
//
// Each method's required permissions are described on the method's page.
// Note that some methods does not require specified permissions but cannot be called without access token.
// You can use no scope while obtaining a token in this case.
//
// https://dev.vk.com/reference/access-rights
type AccessPermission int

const (
	// UserPermissionNotify User allowed sending notifications to him/her (for Flash/iFrame apps).
	UserPermissionNotify AccessPermission = 1 << 0
	// UserPermissionFriends Access to friends.
	UserPermissionFriends AccessPermission = 1 << 1
	// UserPermissionPhotos Access to photos.
	UserPermissionPhotos AccessPermission = 1 << 2
	// UserPermissionAudio Access to video.
	UserPermissionAudio AccessPermission = 1 << 3
	// UserPermissionVideo Access to video.
	UserPermissionVideo AccessPermission = 1 << 4
	// UserPermissionStories Access to stories.
	UserPermissionStories AccessPermission = 1 << 6
	// UserPermissionPages Access to wiki pages.
	UserPermissionPages AccessPermission = 1 << 7
	// UserPermissionUnknown256 Addition of link to the application in the left menu.
	UserPermissionUnknown256 AccessPermission = 1 << 8
	// UserPermissionStatus Access to user status.
	UserPermissionStatus AccessPermission = 1 << 10
	// UserPermissionNotes Access to notes.
	UserPermissionNotes AccessPermission = 1 << 11
	// UserPermissionMessages (for Standalone applications) Access to advanced methods for messaging.
	UserPermissionMessages AccessPermission = 1 << 12
	// UserPermissionWall Access to standard and advanced methods for the wall.
	// Note that this access permission is unavailable for sites (it is ignored at attempt of authorization).
	UserPermissionWall AccessPermission = 1 << 13
	// UserPermissionAds Access to advanced methods for Ads API. (VK.Ads_{...} methods)
	UserPermissionAds AccessPermission = 1 << 15
	// UserPermissionOffline Access to API at any time (you will receive expires_in = 0 in this case).
	UserPermissionOffline AccessPermission = 1 << 16
	// UserPermissionDocs Access to docs.
	UserPermissionDocs AccessPermission = 1 << 17
	// UserPermissionGroups Access to user groups.
	UserPermissionGroups AccessPermission = 1 << 18
	// UserPermissionNotifications Access to notifications about answers to the user.
	UserPermissionNotifications AccessPermission = 1 << 19
	// UserPermissionStats Access to statistics of user groups and applications where he/she is an administrator.
	UserPermissionStats AccessPermission = 1 << 20
	// UserPermissionEmail Access to user email.
	UserPermissionEmail AccessPermission = 1 << 22
	// UserPermissionMarket Access to market.
	UserPermissionMarket AccessPermission = 1 << 27
	// GroupPermissionStories Access to stories.
	GroupPermissionStories AccessPermission = 1 << 0
	// GroupPermissionPhotos Access to photos.
	GroupPermissionPhotos AccessPermission = 1 << 2
	// GroupPermissionAppWidget Access to group app widget (https://dev.vk.com/api/community-apps-widgets/getting-started).
	// This permission can only be requested using the showGroupSettingsBox Client API method (https://vk.com/dev/clientapi).
	GroupPermissionAppWidget AccessPermission = 1 << 6
	// GroupPermissionMessages Access to messages.
	GroupPermissionMessages AccessPermission = 1 << 12
	// GroupPermissionDocs Access to community management.
	GroupPermissionDocs AccessPermission = 1 << 17
	// GroupPermissionManage Access to community management.
	GroupPermissionManage AccessPermission = 1 << 18
)

// InScope check is given AccessPermissions in scope.
func InScope(scope int, permissions ...AccessPermission) bool {
	for _, p := range permissions {
		if int(p) != scope&int(p) {
			return false
		}
	}

	return true
}
