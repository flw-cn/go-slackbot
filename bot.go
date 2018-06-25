package slackbot

import (
	"context"
	"fmt"
	"time"

	"github.com/flw-cn/slack"
	log "github.com/sirupsen/logrus"
)

const (
	// WithTyping sends a message with typing indicator
	WithTyping bool = true
	// WithoutTyping sends a message without typing indicator
	WithoutTyping bool = false

	maxTypingSleepMs time.Duration = time.Millisecond * 2000
)

var botUserID string

// BotOption is a functional option for configuring the bot
type BotOption func(*Bot) error

// WithLogger sets the logger to use
func WithLogger(l *log.Logger) BotOption {
	return func(b *Bot) error {
		b.SetLogger(l)
		return nil
	}
}

// WithClient sets a custom slack client to use
func WithClient(c *slack.Client) BotOption {
	return func(b *Bot) error {
		b.Client = c
		return nil
	}
}

// NewWithOpts creates a new bot with options
func NewWithOpts(opts ...BotOption) (*Bot, error) {
	b := &Bot{}
	for _, opt := range opts {
		err := opt(b)
		if err != nil {
			return nil, err
		}
	}
	if b.logger == nil {
		b.logger = log.New()
	}
	return b, nil
}

// New constructs a new Bot using the slackToken to authorize against the Slack service.
func New(slackToken string) *Bot {
	fmt.Println("New is deprecated. Please migrate to using NewWithOpts")
	b, err := NewWithOpts(WithClient(slack.New(slackToken)))
	if err != nil {
		b.logger.Panicf("error creating bot: %s", err.Error())
	}
	return b
}

// NewWithLogger constructs a new Bot using the slackToken and custom logger instance provided
func NewWithLogger(slackToken string, l *log.Logger) *Bot {
	fmt.Println("NewWithLogger is deprecated. Please migrate to using NewWithOpts")
	b, err := NewWithOpts(WithLogger(l), WithClient(slack.New(slackToken)))
	if err != nil {
		b.logger.Panicf("error creating bot: %s", err.Error())
	}
	return b
}

// Bot is a bot
type Bot struct {
	SimpleRouter
	// Routes to be matched, in order.
	routes []*Route
	// unhandledEventsHandlers are event handlers for unknown events
	unhandledEventsHandlers []EventHandler
	// channelJoinEventsHandlers are event handlers for channel join events
	channelJoinEventsHandlers []ChannelJoinHandler
	// Slack UserID of the bot UserID
	botUserID string
	// logger instance
	logger *log.Logger
	// Slack API
	Client              *slack.Client
	RTM                 *slack.RTM
	TypingDelayModifier float64 // percentage increase or decrease to typing *delay*. 0 = 2ms per character, 4 = 10ms per, -0.5 = 1ms per. Max delay is 2000ms regardless.
}

// Run listens for incoming slack RTM events, matching them to an appropriate handler.
func (b *Bot) Run(rtmopts bool, quitCh <-chan bool) {
	if quitCh == nil {
		quitCh = make(chan bool)
	}

	options := slack.RTMOptions{}
	options.UseRTMStart = rtmopts

	b.RTM = b.Client.NewRTMWithOptions(&options)

	// slack.SetLogger(b.logger)
	go b.RTM.ManageConnection()
	for {
		select {
		case msg := <-b.RTM.IncomingEvents:
			ctx := AddBotToContext(context.Background(), b)
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				log.Printf("Connected: %#v\n", ev.Info.User)
				b.setBotID(ev.Info.User.ID)
				botUserID = ev.Info.User.ID
			case *slack.MessageEvent:
				ctx = AddMessageToContext(ctx, ev)
				var match RouteMatch
				if matched, newCtx := b.Match(ctx, &match); matched && match.Handler != nil {
					match.Handler(newCtx)
				}
			case *slack.ChannelJoinedEvent:
				if len(b.channelJoinEventsHandlers) > 0 {
					for _, h := range b.channelJoinEventsHandlers {
						var handler ChannelJoinMatch
						handler.Handler = h
						go handler.Handle(ctx, b, &ev.Channel)
					}
				}
			case *slack.GroupJoinedEvent:
				if len(b.channelJoinEventsHandlers) > 0 {
					for _, h := range b.channelJoinEventsHandlers {
						var handler ChannelJoinMatch
						handler.Handler = h
						go handler.Handle(ctx, b, &ev.Channel)
					}
				}
			case *slack.RTMError:
				b.logger.Print(ev.Error())
			case *slack.InvalidAuthEvent:
				b.logger.Print("Invalid credentials")
			default:
				if len(b.unhandledEventsHandlers) > 0 {
					for _, h := range b.unhandledEventsHandlers {
						var handler EventMatch
						handler.Handler = h
						go handler.Handle(ctx, b, &msg)
					}
				}
			}
		case <-quitCh:
			log.Debugf("Quit event received.")
			return
		}
	}
}

// SetLogger sets the bot's logger to a custom one
func (b *Bot) SetLogger(l *log.Logger) {
	b.logger = l
}

// OnUnhandledEvent handles any events not already handled
func (b *Bot) OnUnhandledEvent(h EventHandler) {
	b.unhandledEventsHandlers = append(b.unhandledEventsHandlers, h)
}

// OnChannelJoin handles ChannelJoin events
func (b *Bot) OnChannelJoin(h ChannelJoinHandler) {
	b.channelJoinEventsHandlers = append(b.channelJoinEventsHandlers, h)
}

// Reply replies to a message event with a simple message.
func (b *Bot) Reply(evt *slack.MessageEvent, msg string, typing bool) {
	if typing {
		b.Type(evt, msg)
	}
	b.RTM.SendMessage(b.RTM.NewOutgoingMessage(msg, evt.Channel))
}

// ReplyWithAttachments replys to a message event with a Slack Attachments message.
func (b *Bot) ReplyWithAttachments(evt *slack.MessageEvent, attachments []slack.Attachment, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = attachments

	b.Client.PostMessage(evt.Msg.Channel, "", params)
}

// ReplyInThread
func (b *Bot) ReplyInThread(evt *slack.MessageEvent, msg string, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}

	if evt.ThreadTimestamp == "" {
		params.ThreadTimestamp = evt.Timestamp
	} else {
		params.ThreadTimestamp = evt.ThreadTimestamp
	}

	b.Client.PostMessage(evt.Msg.Channel, msg, params)
}

// ReplyInThreadWithAttachments replys to a message event inside a thread with a Slack Attachments message.
func (b *Bot) ReplyInThreadWithAttachments(evt *slack.MessageEvent, attachments []slack.Attachment, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = attachments

	if evt.ThreadTimestamp == "" {
		params.ThreadTimestamp = evt.Timestamp
	} else {
		params.ThreadTimestamp = evt.ThreadTimestamp
	}

	b.Client.PostMessage(evt.Msg.Channel, "", params)
}

// Type sends a typing message and simulates delay (max 2000ms) based on message size.
func (b *Bot) Type(evt *slack.MessageEvent, msg interface{}) {
	msgLen := msgLen(msg)

	sleepDuration := time.Duration(float64(time.Minute*time.Duration(msgLen)/30000) * (1 + b.TypingDelayModifier))
	if sleepDuration > maxTypingSleepMs {
		sleepDuration = maxTypingSleepMs
	}

	b.RTM.SendMessage(b.RTM.NewTypingMessage(evt.Channel))
	time.Sleep(sleepDuration)
}

// BotUserID fetches the Bot's user ID.
func (b *Bot) BotUserID() string {
	return b.botUserID
}

func (b *Bot) setBotID(ID string) {
	b.botUserID = ID
	b.SimpleRouter.SetBotID(ID)

	for _, route := range b.routes {
		route.setBotID(ID)
	}
}

// msgLen gets length of message and attachment messages. Unsupported types return 0.
func msgLen(msg interface{}) (msgLen int) {
	switch m := msg.(type) {
	case string:
		msgLen = len(m)
	case []slack.Attachment:
		msgLen = len(fmt.Sprintf("%#v", m))
	}
	return
}

// Stop stops the bot
func (b *Bot) Stop() {
	_ = b.RTM.Disconnect()
}
