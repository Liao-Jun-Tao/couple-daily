// Package authz encodes the locked author rule: only the author may delete.
// Anniversary PATCH is creator-only as well, including reminder fields.
package authz

// AuthorOnly reports whether actorID is the author of a resource.
// Posts, comments, check-ins, anniversaries, and favorites use it for delete.
// Anniversary updates use it too. Likes are cancelled only by the user who created them.
func AuthorOnly(actorID, authorID string) bool {
	return actorID != "" && actorID == authorID
}
