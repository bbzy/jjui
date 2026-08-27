package jj

import (
	"encoding/json"
	"strings"
)

const (
	moveBookmarkTemplate = `name ++ ";" ++ if(remote, "remote", ".") ++ ";" ++ present ++ ";" ++ tracked ++ ";" ++ conflict ++ ";" ++ if(normal_target, normal_target.contained_in("%s"), false) ++ ";" ++ if(normal_target, normal_target.commit_id().shortest(1), "") ++ "\n"`
	allBookmarkTemplate  = `name ++ ";" ++ if(remote, remote, ".") ++ ";" ++ present ++ ";" ++ tracked ++ ";" ++ conflict ++ ";" ++ false ++ ";" ++ if(normal_target, normal_target.commit_id().shortest(1), "") ++ "\n"`
	pendingBookmarkDeletionTemplate = `if(remote && present && tracked && !tracking_present && !conflict, json(name) ++ "\t" ++ json(remote) ++ "\t" ++ json(normal_target.change_id()) ++ "\t" ++ json(normal_target.commit_id()) ++ "\n")`
)

type BookmarkRemote struct {
	Remote   string
	CommitId string
	Tracked  bool
	Present  bool
}

type Bookmark struct {
	Name      string
	Local     *BookmarkRemote
	Remotes   []BookmarkRemote
	Conflict  bool
	Backwards bool
}

type PendingBookmarkDeletion struct {
	Name     string
	Remote   string
	ChangeId string
	CommitId string
}

func (b Bookmark) IsDeletable() bool {
	return b.Local != nil && b.Local.Present
}

func (b Bookmark) IsTrackable() bool {
	return b.Local != nil && b.Local.Present && len(b.Remotes) == 0
}

func (b Bookmark) IsDeleted() bool {
	return b.Local != nil && !b.Local.Present
}

func (b Bookmark) HasTrackedRemote(remote string) bool {
	for _, r := range b.Remotes {
		if r.Remote == remote && r.Tracked {
			return true
		}
	}
	return false
}

func ParseBookmarkListOutput(output string) []Bookmark {
	lines := strings.Split(output, "\n")
	bookmarkMap := make(map[string]*Bookmark)
	var orderedNames []string

	for _, b := range lines {
		parts := strings.Split(b, ";")
		if len(parts) != 7 {
			continue
		}

		name := parts[0]
		name = strings.Trim(name, "\"")
		remoteName := parts[1]
		present := parts[2] == "true"
		tracked := parts[3] == "true"
		conflict := parts[4] == "true"
		backwards := parts[5] == "true"
		commitId := parts[6]

		if remoteName == "git" {
			continue
		}

		bookmark, exists := bookmarkMap[name]
		if !exists {
			bookmark = &Bookmark{
				Name:      name,
				Conflict:  conflict,
				Backwards: backwards,
			}
			bookmarkMap[name] = bookmark
			orderedNames = append(orderedNames, name)
		}

		if remoteName == "." {
			bookmark.Local = &BookmarkRemote{
				Remote:   ".",
				CommitId: commitId,
				Tracked:  tracked,
				Present:  present,
			}
		} else {
			remote := BookmarkRemote{
				Remote:   remoteName,
				Tracked:  tracked,
				CommitId: commitId,
				Present:  present,
			}
			if remoteName == "origin" {
				bookmark.Remotes = append([]BookmarkRemote{remote}, bookmark.Remotes...)
			} else {
				bookmark.Remotes = append(bookmark.Remotes, remote)
			}
		}
	}

	if len(orderedNames) == 0 {
		return nil
	}

	bookmarks := make([]Bookmark, len(orderedNames))
	for i, name := range orderedNames {
		bookmarks[i] = *bookmarkMap[name]
	}
	return bookmarks
}

func ParsePendingBookmarkDeletions(output string) []PendingBookmarkDeletion {
	lines := strings.Split(output, "\n")
	deletions := make([]PendingBookmarkDeletion, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			continue
		}
		var deletion PendingBookmarkDeletion
		fields := []*string{&deletion.Name, &deletion.Remote, &deletion.ChangeId, &deletion.CommitId}
		valid := true
		for i, field := range fields {
			if err := json.Unmarshal([]byte(parts[i]), field); err != nil {
				valid = false
				break
			}
		}
		if valid {
			deletions = append(deletions, deletion)
		}
	}
	return deletions
}
