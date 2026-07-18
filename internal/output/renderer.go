package output

import "github.com/elefantephp/elefante/internal/model"

type Result struct {
	Payload any
	Text    string
}

type Fact struct {
	Payload any
	Text    string
}

type Diagnostic struct {
	Payload model.Diagnostic
	Text    string
}

type Plan struct {
	Payload model.Plan
	Text    string
}

type Renderer interface {
	Started() error
	Fact(Fact) error
	Diagnostic(Diagnostic) error
	Plan(Plan) error
	Result(Result) error
	Error(*model.Error) error
	Completed(model.Exit) error
}
