// Package authz encodes the locked deletion rule: only the author may delete.
package authz

// AuthorOnly reports whether actorID may delete a resource created by authorID.
// Posts, comments, check-ins, anniversaries, and favorites all use this rule.
// Likes are cancelled only by the user who created them.
func AuthorOnly(actorID, authorID string) bool {
	return actorID != "" && actorID == authorID
}
