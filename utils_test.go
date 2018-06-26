package slackbot

import (
	"testing"

	"github.com/flw-cn/slack"
	"github.com/stretchr/testify/assert"
)

func TestStripDirectMention(t *testing.T) {
	assert := assert.New(t)

	pairs := []string{
		"a message", "a message",
		"    space", "    space",
		"<@USOMEUSER> I 😍 u", "I 😍 u",
		"<@USOMEUSER>       space", "space",
		"<@USOMEUSER>:  abc", "abc",
		"<@USOMEUSER>: you <@USOMEUSER>", "you <@USOMEUSER>",
		"space    ", "space    ",
	}

	for i := 0; i < len(pairs); {
		assert.Equal(pairs[i+1], StripDirectMention(pairs[i]))
		i += 2
	}
}

func TestIsDirectMessage(t *testing.T) {
	// TODO:
}

func TestIsDirectMention(t *testing.T) {
	// TODO:
}

func TestWhoMentioned(t *testing.T) {
	assert := assert.New(t)
	msg := &slack.MessageEvent{}

	msg.Text = "<@U123456> hi there <@UABCDEF>"
	assert.Equal([]string{"U123456", "UABCDEF"}, WhoMentioned(msg))

	msg.Text = "<@> <@DD <@U123456> hi there <@UABCDEF> LL"
	assert.Equal([]string{"U123456", "UABCDEF"}, WhoMentioned(msg))
}

func TestIsMention(t *testing.T) {
	assert := assert.New(t)
	msg := &slack.MessageEvent{}

	msg.Text = "<@U123456> hi there <@UABCDEF>"
	assert.True(IsMention(msg))

	msg.Text = "this is something"
	assert.False(IsMention(msg))
}

func TestIsMentioned(t *testing.T) {
	assert := assert.New(t)
	msg := &slack.MessageEvent{}

	msg.Text = "<@U123456> hi there <@UABCDEF>"
	assert.True(IsMentioned(msg, "UABCDEF"))

	msg.Text = "<@U123456> hi there <@UABCDEF>"
	assert.False(IsMentioned(msg, "UABCDE"))

	msg.Text = "<@U123456> hi there <@UABCDEF>"
	assert.False(IsMentioned(msg, "UXXXXXX"))

	msg.Text = "this is something"
	assert.False(IsMentioned(msg, "UAAAAAA"))
}
