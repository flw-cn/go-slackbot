package slackbot

import (
	"regexp"
	"strings"

	"github.com/flw-cn/slack"
)

// StripDirectMention removes a leading mention (aka direct mention) from a message string
func StripDirectMention(text string) string {
	r, rErr := regexp.Compile(`(^<@[a-zA-Z0-9]+>[\:]*[\s]*)?(.*)`)
	if rErr != nil {
		return ""
	}
	return r.FindStringSubmatch(text)[2]
}

// IsDirectMessage returns true if this message is in a direct message conversation
func IsDirectMessage(rtm *slack.RTM, evt *slack.MessageEvent) bool {
	return strings.HasPrefix(evt.Channel, "D")
}

// IsDirectMention returns true is message is a Direct Mention that mentions a specific user.
// A direct mention is a mention at the very beginning of the message
func IsDirectMention(rtm *slack.RTM, evt *slack.MessageEvent, userID string) bool {
	r, err := regexp.Compile(`^\s*<@` + userID + `>`)
	if err != nil {
		return false
	}

	return r.MatchString(evt.Text)
}

// IsMessage returns true if this message is in a channel/group conversation
func IsMessage(rtm *slack.RTM, evt *slack.MessageEvent) bool {
	ch := rtm.GetInfo().GetChannelByID(evt.Channel)
	gp := rtm.GetInfo().GetGroupByID(evt.Channel)
	return ch != nil || gp != nil
}

// IsMentioned returns true if this message contains a mention of a specific user
func IsMentioned(evt *slack.MessageEvent, userID string) bool {
	userIDs := WhoMentioned(evt)
	for _, u := range userIDs {
		if u == userID {
			return true
		}
	}
	return false
}

// IsMention returns true the message contains a mention
func IsMention(evt *slack.MessageEvent) bool {
	r, rErr := regexp.Compile(`<@(U[a-zA-Z0-9]+)>`)
	if rErr != nil {
		return false
	}
	results := r.FindAllStringSubmatch(evt.Text, -1)
	return len(results) > 0
}

// IsFileShared returns true the message has `file_shared` type
func IsFileShared(evt *slack.MessageEvent) bool {
	// FIXME: return evt.Type == "file_shared"
	return evt.SubType == "file_share"
}

// WhoMentioned returns a list of userIDs mentioned in the message
func WhoMentioned(evt *slack.MessageEvent) []string {
	r, rErr := regexp.Compile(`<@(U[a-zA-Z0-9]+)>`)
	if rErr != nil {
		return []string{}
	}
	results := r.FindAllStringSubmatch(evt.Text, -1)
	matches := make([]string, len(results))
	for i, r := range results { // nolint: gosimple
		matches[i] = r[1]
	}
	return matches
}

func namedRegexpParse(message string, exp *regexp.Regexp) (bool, map[string]string) {
	md := make(map[string]string)
	allMatches := exp.FindStringSubmatch(message)
	if len(allMatches) == 0 {
		return false, md
	}
	keys := exp.SubexpNames()
	if len(keys) != 0 {
		for i, name := range keys {
			if i != 0 {
				md[name] = allMatches[i]
			}
		}
	}
	return true, md
}
