package handler

import (
	"context"

	"gpioblink.com/x/karaoke-demon-clean/application"
)

type Request struct {
	action string
	params []string
}

type HandlerFunc func(context.Context, application.MusicService, Request)
type HandlerFuncWithResponse func(context.Context, application.MusicService, Request) string

func NewRequest(action string, params []string) *Request {
	return &Request{
		action: action,
		params: params,
	}
}
