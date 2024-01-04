package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsConnection struct {
	HostName     string
	Port         string
	UseJetstream bool
	connOpts     []nats.Option
	jetOpts      []jetstream.JetStreamOpt
	Conn         *nats.Conn
	JS           jetstream.JetStream
}

func DefaultNatsConnection() *NatsConnection {
	return &NatsConnection{
		HostName:     "127.0.0.1",
		Port:         "4222",
		UseJetstream: true,
		connOpts:     []nats.Option{},
		jetOpts:      []jetstream.JetStreamOpt{},
	}
}

type NatsConnectionOption func(*NatsConnection)

func WithHostName(hostname string) NatsConnectionOption {
	return func(nc *NatsConnection) {
		nc.HostName = hostname
	}
}

func WithPort(port string) NatsConnectionOption {
	return func(nc *NatsConnection) {
		nc.Port = port
	}
}

func WithoutJetstream(nc *NatsConnection) {
	nc.UseJetstream = false
}

func WithClientName(name string) NatsConnectionOption {
	return func(nc *NatsConnection) {
		nc.connOpts = append(nc.connOpts, nats.Name(name))
	}
}

func WithToken(token string) NatsConnectionOption {
	return func(nc *NatsConnection) {
		if token != "" {
			nc.connOpts = append(nc.connOpts, nats.Token(token))
		}
	}
}

func WithConnectionOption(opt nats.Option) NatsConnectionOption {
	return func(nc *NatsConnection) {
		nc.connOpts = append(nc.connOpts, opt)
	}
}

func WithJetstreamnOption(opt jetstream.JetStreamOpt) NatsConnectionOption {
	return func(nc *NatsConnection) {
		nc.jetOpts = append(nc.jetOpts, opt)
	}
}

func New(optFuncs ...NatsConnectionOption) *NatsConnection {
	n := DefaultNatsConnection()
	for _, optFn := range optFuncs {
		optFn(n)
	}
	return n
}

func (n *NatsConnection) Connect() error {
	var err error

	n.Conn, err = nats.Connect(fmt.Sprintf("nats://%s:%s", n.HostName, n.Port), n.connOpts...)
	if err != nil {
		return err
	}

	if n.UseJetstream {
		n.JS, err = jetstream.New(n.Conn, n.jetOpts...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *NatsConnection) Close() {
	if n.Conn != nil {
		n.Conn.Drain()
	}
}

func (n *NatsConnection) Publish(ctx context.Context, subject string, payload []byte, opts ...jetstream.PublishOpt) {
	if n.UseJetstream {
		n.JS.PublishAsync(subject, payload, opts...)
		// n.JS.Publish(ctx, subject, payload, opts...)
	} else {
		n.Conn.Publish(subject, payload)
	}
}

func (n *NatsConnection) Subscribe(subject string) (sub *nats.Subscription, ch chan *nats.Msg, err error) {
	if !n.UseJetstream {
		ch = make(chan *nats.Msg, 64)
		sub, err = n.Conn.ChanSubscribe(subject, ch)
		return sub, ch, err
	}
	return nil, nil, errors.New("jetstream in use, you should use Stream instead")
}

func (n *NatsConnection) Stream(ctx context.Context, subject string) (stream jetstream.Stream, consumer jetstream.Consumer, err error) {
	if n.UseJetstream {
		stream, err = n.JS.CreateStream(ctx, jetstream.StreamConfig{
			Name:     strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(subject, ".", "_"), "*", "any"), ">", "arrow"),
			Subjects: []string{subject},
		})
		if err != nil {
			return nil, nil, err
		}

		consumer, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{})
		if err != nil {
			return nil, nil, err
		}

		return stream, consumer, nil
	}
	return nil, nil, errors.New("jetstream not in use, you should use Subscribe instead")
}
