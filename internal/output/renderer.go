package output

import "github.com/elefantephp/elefante/internal/model"

type Result struct {
	Payload any
	Text    string
}

type Renderer interface {
	Started() error
	Result(Result) error
	Error(*model.Error) error
	Completed(model.Exit) error
}
