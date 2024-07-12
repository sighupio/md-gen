package mdgen

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	gj "github.com/sighupio/go-jsonschema/pkg/schemas"
	jsonschemaparser "github.com/sighupio/md-gen/internal/json-schema-parser"
	"github.com/sirupsen/logrus"
)

type Generator interface {
	Generate() ([]byte, error)
}

type BaseGenerator struct {
	Output  string
	Sources map[string]*gj.Schema
	RootDir string
}

func NewBaseGenerator(output string, schema *gj.Schema, rootDir string) *BaseGenerator {
	sources := make(map[string]*gj.Schema)

	sources["root"] = schema

	return &BaseGenerator{
		Output:  output,
		Sources: sources,
		RootDir: rootDir,
	}
}

type Element struct {
	Key                   string
	VanillaKey            string
	ParentKey             *string
	FromRef               bool
	FromItems             bool
	DescriptionFromParent string
	MaxItemsFromParent    int
	MinItemsFromParent    int
	El                    *gj.Type
	Source                string
}

func (g *BaseGenerator) Generate() ([]byte, error) {
	genOut := ""

	stack := make([]*Element, 0)

	if len(g.Sources["root"].Type) == 0 {
		return nil, errors.New("no type found in the schema")
	}

	if g.Sources["root"].Type[0] != "object" {
		return nil, errors.New("only object types are supported")
	}

	t := (*gj.Type)(g.Sources["root"].ObjectAsType)

	root := &Element{
		Key:    "",
		El:     t,
		Source: "root",
	}

	stack = append(stack, root)

	for len(stack) > 0 {
		p := stack[len(stack)-1]

		stack = stack[:len(stack)-1]

		elements := make([]*Element, 0)

		if p.ParentKey != nil && !p.FromRef && !p.FromItems {
			genOut += "## " + p.Key + "\n\n"
		}

		if p.El.Properties != nil {
			genOut, elements = g.handleObject(genOut, p)
		}

		if p.El.Items != nil {
			stack = append(stack, &Element{
				Key:                   p.Key,
				ParentKey:             p.ParentKey,
				El:                    p.El.Items,
				FromItems:             true,
				DescriptionFromParent: p.El.Description,
				MaxItemsFromParent:    p.El.MaxItems,
				MinItemsFromParent:    p.El.MinItems,
				Source:                p.Source,
			})

			continue
		}

		if p.El.Ref != "" {
			var pRef *gj.Type

			pSource := p.Source

			logrus.Debugf("Found ref '%s'", p.El.Ref)

			rt, err := gj.GetRefType(p.El.Ref)
			if err != nil {
				return nil, err
			}

			if rt != "file" {
				return nil, errors.New("only file refs are supported")
			}

			// handle from definitions
			if strings.HasPrefix(p.El.Ref, "#/$defs/") {
				rf := strings.TrimPrefix(p.El.Ref, "#/$defs/")

				pRef = g.Sources[p.Source].Definitions[rf]

				if pRef == nil {
					return nil, errors.New("ref not found in definitions")
				}
			}

			// handle from external refs
			if strings.HasPrefix(p.El.Ref, ".") {
				refPath := p.El.Ref

				if !filepath.IsAbs(p.El.Ref) {
					oldWd, err := os.Getwd()
					if err != nil {
						return nil, err
					}

					err = os.Chdir(g.RootDir)
					if err != nil {
						return nil, err
					}

					refPath, err = filepath.Abs(p.El.Ref)
					if err != nil {
						return nil, err
					}

					err = os.Chdir(oldWd)
					if err != nil {
						return nil, err
					}
				}

				parser := jsonschemaparser.NewBaseParser(refPath)

				refSource, err := parser.Parse()
				if err != nil {
					return nil, err
				}

				pRef = (*gj.Type)(refSource.ObjectAsType)

				g.Sources[p.El.Ref] = refSource

				pSource = p.El.Ref

				elo := &Element{
					Key:       p.Key,
					ParentKey: p.ParentKey,
					El:        pRef,
					FromRef:   true,
					Source:    pSource,
				}

				logrus.Debugf("adding element from file ref %+v", elo)
			}

			if pRef != nil {
				el := &Element{
					Key:       p.Key,
					ParentKey: p.ParentKey,
					El:        pRef,
					FromRef:   true,
					Source:    pSource,
				}

				if p.FromItems && pRef.Description == "" && p.DescriptionFromParent != "" {
					el.DescriptionFromParent = p.DescriptionFromParent
				}

				if pRef.Description == "" && p.El.Description != "" && !p.FromItems {
					el.DescriptionFromParent = p.El.Description
				}

				if p.FromItems && pRef.MaxItems == 0 && p.MaxItemsFromParent != 0 {
					el.MaxItemsFromParent = p.MaxItemsFromParent
				}

				if p.FromItems && pRef.MinItems == 0 && p.MinItemsFromParent != 0 {
					el.MinItemsFromParent = p.MinItemsFromParent
				}

				stack = append(stack, el)

				continue
			}
		}

		if p.El.Description != "" {
			genOut += "### Description\n\n"
			genOut += p.El.Description + "\n\n"
		} else if p.DescriptionFromParent != "" {
			genOut += "### Description\n\n"
			genOut += p.DescriptionFromParent + "\n\n"
		}

		if p.El.Enum != nil || p.El.Pattern != "" || p.El.MinItems != 0 || p.El.MaxItems != 0 || p.El.MinLength != 0 || p.El.MaxLength != 0 ||
			p.MaxItemsFromParent != 0 || p.MinItemsFromParent != 0 {
			genOut += "### Constraints\n\n"
		}

		if p.El.MaxLength != 0 {
			genOut += "**maximum length**: the maximum number of characters for this string is: `" + strconv.Itoa(p.El.MaxLength) + "`\n\n"
		}

		if p.El.MinLength != 0 {
			genOut += "**minimum length**: the minimum number of characters for this string is: `" + strconv.Itoa(p.El.MinLength) + "`\n\n"
		}

		if p.El.MaxItems != 0 {
			genOut += "**maximum number of items**: the maximum number of items for this array is: `" + strconv.Itoa(p.El.MaxItems) + "`\n\n"
		} else if p.MaxItemsFromParent != 0 {
			genOut += "**maximum number of items**: the maximum number of items for this array is: `" + strconv.Itoa(p.MaxItemsFromParent) + "`\n\n"
		}

		if p.El.MinItems != 0 {
			genOut += "**minimum number of items**: the minimum number of items for this array is: `" + strconv.Itoa(p.El.MinItems) + "`\n\n"
		} else if p.MinItemsFromParent != 0 {
			genOut += "**minimum number of items**: the minimum number of items for this array is: `" + strconv.Itoa(p.MinItemsFromParent) + "`\n\n"
		}

		if p.El.Enum != nil {
			genOut = g.handleEnum(genOut, p)
		}

		if p.El.Pattern != "" {
			genOut += "**pattern**: the string must match the following regular expression:\n\n"

			genOut += "```regexp\n"

			genOut += p.El.Pattern + "\n"

			genOut += "```\n\n"

			s := strings.ReplaceAll(p.El.Pattern, "/", "\\/")

			s = strings.ReplaceAll(s, "(", "\\(")

			s = strings.ReplaceAll(s, ")", "\\)")

			s = strings.ReplaceAll(s, "+", "%2B")

			genOut += "[try pattern](https://regexr.com/?expression=" + s + ")\n\n"
		}

		for i := range elements {
			logrus.Debugf("Adding property '%s' to stack", elements[i].Key)

			stack = append(stack, elements[len(elements)-1-i])
		}
	}

	return []byte(genOut), nil
}

func (g *BaseGenerator) handleEnum(out string, p *Element) string {
	maxValueLen := 0

	values := make([]string, 0)

	if p.El.Enum != nil {
		out += "**enum**: the value of this property must be equal to one of the following values:\n\n"

		for _, v := range p.El.Enum {
			s := v.(string)

			s = "`\"" + s + "\"`"

			if len(s) > maxValueLen {
				maxValueLen = len(s)
			}

			values = append(values, s)
		}

		out += "| Value "

		for i := 0; i < maxValueLen-len(" Value "); i++ {
			out += " "
		}

		out += "|\n"

		out += "|:----"

		for i := 0; i < maxValueLen-len(":----"); i++ {
			out += "-"
		}

		out += "|\n"

		for _, v := range values {
			out += "|" + v

			for i := 0; i < maxValueLen-len(v); i++ {
				out += " "
			}

			out += "|\n"
		}

		out += "\n"
	}

	return out
}

func (g *BaseGenerator) handleObject(out string, p *Element) (string, []*Element) {
	elements := make([]*Element, 0)

	maxTitleLen := 0
	maxTypeLen := 0
	maxRequiredLen := 0

	properties := make([]*Property, 0)

	if p.El.Properties != nil {
		if p.ParentKey != nil {
			out += "### Properties\n\n"
		} else {
			out += "## Properties\n\n"
		}

		required := p.El.Required

		for i, prop := range p.El.Properties {
			if i == "if" {
				continue
			}

			elements = append(elements, &Element{
				Key:        p.Key + "." + i,
				VanillaKey: i,
				ParentKey:  &p.Key,
				El:         prop,
				Source:     p.Source,
			})
		}

		slices.SortStableFunc(elements, func(i, j *Element) int {
			return strings.Compare(i.Key, j.Key)
		})

		for _, el := range elements {
			t := "object"

			if strings.HasPrefix(el.El.Ref, "#/$defs/") {
				rf := strings.TrimPrefix(el.El.Ref, "#/$defs/")

				pRef := g.Sources[p.Source].Definitions[rf]

				logrus.Debugf("Found ref on object handling %+v", pRef)

				if pRef != nil {
					if len(pRef.Type) > 0 {
						t = pRef.Type[0]
					}
				}
			}

			if len(el.El.Type) > 0 {
				t = el.El.Type[0]
			}

			r := "Required"

			if !slices.Contains(required, el.VanillaKey) {
				r = "Optional"
			}

			link := strings.ReplaceAll(p.Key, ".", "") + el.VanillaKey

			titleStr := " [" + el.VanillaKey + "](#" + strings.ToLower(link) + ") "
			typeStr := " `" + t + "` "
			requiredStr := " " + r + " "

			if len(titleStr) > maxTitleLen {
				maxTitleLen = len(titleStr)
			}

			if len(typeStr) > maxTypeLen {
				maxTypeLen = len(typeStr)
			}

			if len(requiredStr) > maxRequiredLen {
				maxRequiredLen = len(requiredStr)
			}

			properties = append(properties, &Property{
				Title:    titleStr,
				Type:     typeStr,
				Required: requiredStr,
			})
		}

		out += "| Property "

		for i := 0; i < maxTitleLen-len(" Property "); i++ {
			out += " "
		}

		out += "| Type "

		for i := 0; i < maxTypeLen-len(" Type "); i++ {
			out += " "
		}

		out += "| Required "

		for i := 0; i < maxRequiredLen-len(" Required "); i++ {
			out += " "
		}

		out += "|\n"

		out += "|:----"

		for i := 0; i < maxTitleLen-len(":----"); i++ {
			out += "-"
		}

		out += "|:----"

		for i := 0; i < maxTypeLen-len(":----"); i++ {
			out += "-"
		}

		out += "|:----"

		for i := 0; i < maxRequiredLen-len(":----"); i++ {
			out += "-"
		}

		out += "|\n"

		for _, prop := range properties {
			out += "|" + prop.Title

			for i := 0; i < maxTitleLen-len(prop.Title); i++ {
				out += " "
			}

			out += "|" + prop.Type

			for i := 0; i < maxTypeLen-len(prop.Type); i++ {
				out += " "
			}

			out += "|" + prop.Required

			for i := 0; i < maxRequiredLen-len(prop.Required); i++ {
				out += " "
			}

			out += "|\n"
		}

		out += "\n"
	}

	return out, elements
}

type Property struct {
	Title    string
	Type     string
	Required string
}
