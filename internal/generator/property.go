package generator

import "fmt"

type Property struct {
	Type *interface{} `json:"type"`

	Description *string `json:"description"`

	Required *bool `json:"required"`
	Limits

	Ref *string `json:"$ref"`

	Enum      *[]interface{} `json:"enum"`
	EnumNames *[]string      `json:"enumNames"`

	// type == array
	Items *Property `json:"items"`

	// type == object
	PatternProperties *map[string]map[string]string `json:"patternProperties"`
	Properties        *map[string]Property          `json:"properties"`
	AllOf             *[]Property                   `json:"allOf"`
	OneOf             *[]Property                   `json:"oneOf"`
}

type Limits struct {
	Minimum   interface{} `json:"minimum"`
	Maximum   interface{} `json:"maximum"`
	MinLength *int        `json:"minLength"`
	MaxLength *int        `json:"maxLength"`
	MinItems  *int        `json:"minItems"`
	MaxItems  *int        `json:"maxItems"`
	Format    *string     `json:"format"`
	Default   interface{} `json:"default"`
}

func (l *Limits) Add(newL Limits) {
	if newL.Minimum != nil {
		l.Minimum = newL.Minimum
	}
	if newL.Maximum != nil {
		l.Maximum = newL.Maximum
	}
	if newL.MinLength != nil {
		l.MinLength = newL.MinLength
	}
	if newL.MaxLength != nil {
		l.MaxLength = newL.MaxLength
	}
	if newL.MinItems != nil {
		l.MinItems = newL.MinItems
	}
	if newL.MaxItems != nil {
		l.MaxItems = newL.MaxItems
	}
	if newL.Format != nil {
		l.Format = newL.Format
	}
	if newL.Default != nil {
		l.Default = newL.Default
	}
}

func (l Limits) gen(nestingLvl int) (gen string) {
	tabs := getTabs(nestingLvl)

	if l.Default != nil {
		gen += fmt.Sprintf("%s//  Default: %v\n", tabs, l.Default)
	}
	if l.Format != nil {
		gen += fmt.Sprintf("%s//  Format: %s\n", tabs, *l.Format)
	}
	if l.MinItems != nil {
		gen += fmt.Sprintf("%s//  MinItems: %d\n", tabs, *l.MinItems)
	}
	if l.MaxItems != nil {
		gen += fmt.Sprintf("%s//  MaxItems: %d\n", tabs, *l.MaxItems)
	}
	if l.Minimum != nil {
		gen += fmt.Sprintf("%s//  Minimum: %v\n", tabs, l.Minimum)
	}
	if l.Maximum != nil {
		gen += fmt.Sprintf("%s//  Maximum: %v\n", tabs, l.Maximum)
	}
	if l.MinLength != nil {
		gen += fmt.Sprintf("%s//  MinLength: %d\n", tabs, *l.MinLength)
	}
	if l.MaxLength != nil {
		gen += fmt.Sprintf("%s//  MaxLength: %d\n", tabs, *l.MaxLength)
	}

	return
}

type Namer interface {
	GetName() string
}

type Coder interface {
	GetCode() int
}

type Genner interface {
	Gen() (gen string)
}

type GennerWithTest interface {
	Genner
	TestGen() (testGen string)
}

type NameGennerWithTest interface {
	Namer
	GennerWithTest
}
