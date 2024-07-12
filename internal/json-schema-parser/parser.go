package jsonschemaparser

import gj "github.com/sighupio/go-jsonschema/pkg/schemas"

type Parser interface {
	Parse() error
}

type BaseParser struct {
	Input string
}

func NewBaseParser(input string) *BaseParser {
	return &BaseParser{
		Input: input,
	}
}

func (p *BaseParser) Parse() (*gj.Schema, error) {
	s, err := gj.FromJSONFile(p.Input)
	if err != nil {
		return nil, err
	}

	return s, nil
}
