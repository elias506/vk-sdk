package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type ObjectsFile struct {
	Definitions map[string]Property
}

func GenerateObjects(w, testW io.Writer, objectsRaw []byte) {
	var file ObjectsFile

	if err := json.Unmarshal(objectsRaw, &file); err != nil {
		panic(err.Error())
	}

	nameGenners := make([]NameGennerWithTest, 0, len(file.Definitions))

	for name, prop := range file.Definitions {
		obj := parseObjectNameGenner(name, prop, 0)
		if obj != nil {
			nameGenners = append(nameGenners, obj)
		}
	}

	writeStartFile(w, "vk_sdk", "", "encoding/json")
	writeStartFile(testW, "vk_sdk", "", "encoding/json")

	fmt.Fprint(w, "// suppress unused package warning\nvar _ *json.RawMessage\n\n")
	fmt.Fprint(testW, "// suppress unused package warning\nvar _ *json.RawMessage\n\n")

	sort.SliceStable(nameGenners, func(i, j int) bool {
		return nameGenners[i].GetName() < nameGenners[j].GetName()
	})

	for _, genner := range nameGenners {
		fmt.Fprint(w, genner.Gen())
		fmt.Fprint(testW, genner.TestGen())
	}
}

var SimpleReferences = make(map[string]RefType)

type RefType struct {
	ArrayNestingLevel int
	Type              string
}

type NestedGenner interface {
	nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string)
	nestedGen(nestingLvl int, objName string) (nestedGen string, additionalGen string)
}

type NameNestedGenner interface {
	Namer
	NestedGenner
}

const testMethodName = "fillRandomly"

func parseObjectNameGenner(name string, prop Property, arrayNestingLvl int) NameGennerWithTest {
	if prop.Ref != nil {
		t := parseSimpleType(name, prop, arrayNestingLvl)
		SimpleReferences[t.Name] = RefType{
			ArrayNestingLevel: arrayNestingLvl,
			Type:              t.Type,
		}
		return t
	}

	if prop.Items != nil {
		prop.Limits.Add(prop.Items.Limits)
		prop.Items.Description = prop.Description
		prop.Items.Required = prop.Required
		return parseObjectNameGenner(name, *prop.Items, arrayNestingLvl+1)
	}

	if prop.Enum != nil {
		e := parseEnum(name, prop, arrayNestingLvl)
		SimpleReferences[e.Name] = RefType{
			ArrayNestingLevel: arrayNestingLvl,
			Type:              e.ValuesType,
		}
		return e
	}

	if prop.AllOf != nil {
		return parseAllOf(name, prop, arrayNestingLvl)
	}

	if prop.OneOf != nil {
		return parseOneOf(name, prop, arrayNestingLvl)
	}

	if prop.Properties != nil {
		return parseObject(name, prop, arrayNestingLvl)
	}

	if prop.PatternProperties != nil {
		return parsePatternProperties(name, prop, arrayNestingLvl)
	}

	t := parseSimpleType(name, prop, arrayNestingLvl)
	SimpleReferences[t.Name] = RefType{
		ArrayNestingLevel: arrayNestingLvl,
		Type:              t.Type,
	}
	return t
}

func parseNameNestedGenner(name string, prop Property, arrayNestingLvl int) NameNestedGenner {
	if prop.Ref != nil {
		return parseSimpleType(name, prop, arrayNestingLvl)
	}

	if prop.Items != nil {
		prop.Limits.Add(prop.Items.Limits)
		prop.Items.Description = prop.Description
		prop.Items.Required = prop.Required
		return parseNameNestedGenner(name, *prop.Items, arrayNestingLvl+1)
	}

	if prop.Enum != nil {
		return parseEnum(name, prop, arrayNestingLvl)
	}

	if prop.AllOf != nil {
		return parseAllOf(name, prop, arrayNestingLvl)
	}

	if prop.OneOf != nil {
		return parseOneOf(name, prop, arrayNestingLvl)
	}

	if prop.Properties != nil {
		return parseObject(name, prop, arrayNestingLvl)
	}

	if prop.PatternProperties != nil {
		return parsePatternProperties(name, prop, arrayNestingLvl)
	}

	return parseSimpleType(name, prop, arrayNestingLvl)
}

type Object struct {
	Name              string
	Description       string
	Fields            []NameNestedGenner
	ArrayNestingLevel int

	// fields for nested Object
	IsRequired bool
}

func (o Object) GetName() string {
	return o.Name
}

func parseObject(name string, prop Property, arrayNestingLvl int) (o Object) {
	o.Name = name
	o.ArrayNestingLevel = arrayNestingLvl

	if prop.Description != nil {
		o.Description = *prop.Description
	}

	if prop.Required != nil {
		o.IsRequired = *prop.Required
	}

	if prop.Properties == nil {
		panic(name)
	}

	o.Fields = make([]NameNestedGenner, 0, len(*prop.Properties))

	for fieldName, fieldProp := range *prop.Properties {
		o.Fields = append(o.Fields, parseNameNestedGenner(fieldName, fieldProp, 0))
	}

	return
}

func (o Object) Gen() (gen string) {
	genName := getFullObjectName(o.Name)

	sort.SliceStable(o.Fields, func(i, j int) bool {
		return o.Fields[i].GetName() < o.Fields[j].GetName()
	})

	var fieldsGen string

	for _, nestedGenner := range o.Fields {
		fGen, addGen := nestedGenner.nestedGen(1, genName)
		fieldsGen += fGen
		gen += addGen
	}

	// write comment
	if o.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", genName, o.Description)
	}

	if _, easySkipped := easyJSONBlackList[o.Name]; easySkipped {
		gen += "//easyjson:skip\n"
	}

	gen += fmt.Sprintf("type %s %sstruct {\n%s}\n\n", genName, getArrayBrackets(o.ArrayNestingLevel), fieldsGen)

	return
}

func (o Object) TestGen() (testGen string) {
	genName := getFullObjectName(o.Name)

	sort.SliceStable(o.Fields, func(i, j int) bool {
		return o.Fields[i].GetName() < o.Fields[j].GetName()
	})

	var fieldsGen, additionalGen string

	firstArray := true
	for _, nestedGenner := range o.Fields {
		fGen, addGen := nestedGenner.nestedTestGen(genName, "(*o)", &firstArray)
		fieldsGen += fGen
		additionalGen += addGen
	}

	testGen += additionalGen

	testGen += fmt.Sprintf("func %s_%s(o *%s) {\n%s}\n\n", testMethodName, genName, genName, fieldsGen)

	return
}

func (o Object) nestedGen(nestingLvl int, objName string) (nestedGen, additionalGen string) {
	genName := getFullName(o.Name)
	newObjName := objName + "_" + genName
	if o.Name == "" {
		newObjName = objName
		nestingLvl--
	}

	sort.SliceStable(o.Fields, func(i, j int) bool {
		return o.Fields[i].GetName() < o.Fields[j].GetName()
	})

	var fieldsGen string

	for _, nestedGenner := range o.Fields {
		fGen, addGen := nestedGenner.nestedGen(nestingLvl+1, newObjName)
		fieldsGen += fGen
		additionalGen += addGen
	}

	tabs := getTabs(nestingLvl)

	// write comment
	if o.Description != "" {
		nestedGen += fmt.Sprintf("%s// %s\n", tabs, o.Description)
	}

	if o.Name == "" {
		// Unknown object
		// Like in AllOf or OneOf
		nestedGen += fieldsGen
		return
	}

	var preType, omitempty string
	if !o.IsRequired {
		omitempty = ",omitempty"
		preType = "*"
	}
	preType += getArrayBrackets(o.ArrayNestingLevel)

	genJSON := fmt.Sprintf("`json:%q`", o.Name+omitempty)

	nestedGen += fmt.Sprintf("%s%s %sstruct {\n%s%s} %s\n",
		tabs, genName, preType, fieldsGen, tabs, genJSON)

	return
}

func (o Object) nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string) {
	genName := getFullObjectName(o.Name)

	ref := refName + "." + genName
	newObjName := objName + "_" + genName

	if o.Name == "" {
		newObjName = objName
		ref = refName
	}

	sort.SliceStable(o.Fields, func(i, j int) bool {
		return o.Fields[i].GetName() < o.Fields[j].GetName()
	})

	if !o.IsRequired && o.Name != "" {
		var nestedGen string

		for _, nestedGenner := range o.Fields {
			nGen, _ := nestedGenner.nestedGen(2, objName+"_"+genName)
			nestedGen += nGen
		}

		testGen += fmt.Sprintf("\t %s = new(struct {\n%s\t})\n", ref, nestedGen)
	}

	for _, nestedGenner := range o.Fields {
		fGen, addGen := nestedGenner.nestedTestGen(newObjName, ref, firstArray)
		testGen += fGen
		additionalGen += addGen
	}

	return
}

type AllOf struct {
	Name              string
	Description       string
	Fields            []NestedGenner
	ArrayNestingLevel int

	// fields for nested AllOf
	IsRequired bool
}

func (ao AllOf) GetName() string {
	return ao.Name
}

func parseAllOf(name string, prop Property, arrayNestingLvl int) (ao AllOf) {
	ao.Name = name
	ao.ArrayNestingLevel = arrayNestingLvl

	if prop.Description != nil {
		ao.Description = *prop.Description
	}

	if prop.Required != nil {
		ao.IsRequired = *prop.Required
	}

	if prop.AllOf == nil {
		panic(name)
	}

	ao.Fields = make([]NestedGenner, 0, len(*prop.AllOf))

	for _, allOfProp := range *prop.AllOf {
		required := true
		allOfProp.Required = &required
		ao.Fields = append(ao.Fields, parseNameNestedGenner("", allOfProp, 0))
	}

	return
}

func (ao AllOf) Gen() (gen string) {
	genName := getFullObjectName(ao.Name)

	var fieldsGen string

	for _, nestedGenner := range ao.Fields {
		fGen, addGen := nestedGenner.nestedGen(1, genName)
		fieldsGen += fGen
		gen += addGen
	}

	// write comment
	if ao.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", genName, ao.Description)
	}

	if _, easySkipped := easyJSONBlackList[ao.Name]; easySkipped {
		gen += "//easyjson:skip\n"
	}

	gen += fmt.Sprintf("type %s %sstruct {\n%s}\n\n", genName, getArrayBrackets(ao.ArrayNestingLevel), fieldsGen)

	return
}

func (ao AllOf) TestGen() (testGen string) {
	genName := getFullObjectName(ao.Name)

	var fieldsGen, additionalGen string

	firstArray := true
	for _, nestedGenner := range ao.Fields {
		fGen, addGen := nestedGenner.nestedTestGen(genName, "(*o)", &firstArray)
		fieldsGen += fGen
		additionalGen += addGen
	}

	testGen += additionalGen

	testGen += fmt.Sprintf("func %s_%s(o *%s) {\n%s}\n\n", testMethodName, genName, genName, fieldsGen)

	return
}

func (ao AllOf) nestedGen(nestingLvl int, objName string) (nestedGen, additionalGen string) {
	genName := getFullName(ao.Name)

	var fieldsGen string

	for _, nestedGenner := range ao.Fields {
		fGen, addGen := nestedGenner.nestedGen(nestingLvl, objName+"_"+genName)
		fieldsGen += fGen
		additionalGen += addGen
	}

	tabs := getTabs(nestingLvl)

	// write comment
	if ao.Description != "" {
		nestedGen += fmt.Sprintf("%s// %s\n", tabs, ao.Description)
	}

	if ao.Name == "" {
		// Unknown object
		// Like in AllOf or OneOf
		nestedGen += fieldsGen
		return
	}

	var preType, omitempty string
	if !ao.IsRequired {
		omitempty = ",omitempty"
		preType = "*"
	}
	preType += getArrayBrackets(ao.ArrayNestingLevel)

	genJSON := fmt.Sprintf("`json:%q`", ao.Name+omitempty)

	nestedGen += fmt.Sprintf("%s%s %sstruct {\n%s%s} %s\n",
		tabs, genName, preType, fieldsGen, tabs, genJSON)

	return
}

func (ao AllOf) nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string) {
	panic("here")
}

type OneOf struct {
	Name              string
	Description       string
	Fields            []NestedGenner
	ArrayNestingLevel int

	// fields for nested OneOf
	IsRequired bool
	isNested   bool
}

func (of OneOf) GetName() string {
	return of.Name
}

func parseOneOf(name string, prop Property, arrayNestingLvl int) (of OneOf) {
	of.Name = name
	of.ArrayNestingLevel = arrayNestingLvl

	if prop.Description != nil {
		of.Description = *prop.Description
	}

	if prop.Required != nil {
		of.IsRequired = *prop.Required
	}

	if prop.OneOf == nil {
		panic(name)
	}

	of.Fields = make([]NestedGenner, 0, len(*prop.OneOf))

	for _, oneOfProp := range *prop.OneOf {
		of.Fields = append(of.Fields, parseNameNestedGenner("", oneOfProp, 0))
	}

	return
}

func (of OneOf) Gen() (gen string) {
	var genName string

	if of.isNested {
		genName = of.Name
	} else {
		genName = getFullObjectName(of.Name)
	}

	// write comment
	if of.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", genName, of.Description)
	}

	// write main object
	gen += "//easyjson:skip\n"
	gen += fmt.Sprintf("type %s struct{\n", genName)
	gen += "\traw []byte\n"
	gen += "}\n\n"

	// write marshal/unmarshaler method
	gen += of.genMarshaler()
	gen += of.genUnmarshaler()

	// write json map getter
	gen += of.genRawGetter()

	return
}

func (of OneOf) genMarshaler() (gen string) {
	var genName string

	if of.isNested {
		genName = of.Name
	} else {
		genName = getFullObjectName(of.Name)
	}

	gen += fmt.Sprintf("func (o *%s) MarshalJSON() ([]byte, error) {\n", genName)

	gen += fmt.Sprintf("\treturn o.raw, nil\n")
	gen += "}\n\n"

	return
}

func (of OneOf) genUnmarshaler() (gen string) {
	var genName string

	if of.isNested {
		genName = of.Name
	} else {
		genName = getFullObjectName(of.Name)
	}

	gen += fmt.Sprintf("func (o *%s) UnmarshalJSON(body []byte) (err error) {\n", genName)

	gen += "\to.raw = body\n"
	//gen += "\treturn json.Unmarshal(body, &o.raw)\n"
	gen += "\treturn nil\n"

	gen += "}\n\n"

	return
}

func (of OneOf) genRawGetter() (gen string) {
	var genName string

	if of.isNested {
		genName = of.Name
	} else {
		genName = getFullObjectName(of.Name)
	}

	gen += fmt.Sprintf("func (o %s) Raw() []byte {\n", genName)
	gen += fmt.Sprintf("\treturn o.raw\n")
	gen += "}\n\n"

	return
}

func (of OneOf) TestGen() (testGen string) {
	var genName string

	if of.isNested {
		genName = of.Name
	} else {
		genName = getFullObjectName(of.Name)
	}

	testGen += fmt.Sprintf("func %s_%s(o *%s) {\n", testMethodName, genName, genName)
	testGen += fmt.Sprintf("\tvar rawJSON []byte\n")
	testGen += fmt.Sprintf("\tswitch randIntn(%d) {\n", len(of.Fields))

	for i, f := range of.Fields {
		testGen += fmt.Sprintf("\tcase %d:\n", i)
		t := f.(SimpleType)
		fGenName := getFullObjectName(t.Type)

		if t.ArrayNestingLevel == 0 {

			if isGoType(t.Type) {
				testGen += fmt.Sprintf("\t\tr := %s\n", getRandSetter(t.Type))
				testGen += "\t\trawJSON, _ = json.Marshal(r)\n"
				continue
			}

			testGen += fmt.Sprintf("\t\tr := new(%s)\n", fGenName)
			testGen += fmt.Sprintf("\t\t%s_%s(r)\n", testMethodName, fGenName)
			testGen += "\t\trawJSON, _ = json.Marshal(*r)\n"
			continue
		}

		testGen += "\t\tl0 := randIntn(maxArrayLength + 1)\n"

		testGen += fmt.Sprintf("\t\tr := make(%s%s, l0)\n", getArrayBrackets(t.ArrayNestingLevel), fGenName)

		var tabs string
		var endBrackets string

		for j := 0; j < t.ArrayNestingLevel; j++ {
			tabs = getTabs(j + 2)
			testGen += fmt.Sprintf("%sfor i%d := 0; i%d < l%d; i%d++ {\n",
				tabs, j, j, j, j)

			if j+1 < t.ArrayNestingLevel {
				testGen += fmt.Sprintf("\t\t%sl%d = randIntn(maxArrayLength + 1)\n", tabs, j+1)

				testGen += fmt.Sprintf("\t\t%s(*r)[i%d] = make(%s%s, l%d)\n",
					tabs, j, getArrayBrackets(t.ArrayNestingLevel-j-1), fGenName, j+1)
			}

			endBrackets = tabs + "}\n" + endBrackets
		}

		tabs += "\t"

		brackets := ""
		for j := 0; j < t.ArrayNestingLevel; j++ {
			brackets += fmt.Sprintf("[i%d]", j)
		}

		testGen += fmt.Sprintf("%s%s_%s(&(r%s))\n",
			tabs, testMethodName, fGenName, brackets)

		testGen += endBrackets

		testGen += "\t\trawJSON, _ = json.Marshal(r)\n"
	}

	testGen += "\t}\n"

	testGen += "\to.raw = rawJSON\n"

	testGen += "}\n\n"

	return
}

func (of OneOf) nestedGen(tabsCount int, objName string) (gen, additionalGen string) {
	genName := getFullName(of.Name)

	newStructType := objName + "_" + genName

	mainOneOf := OneOf{
		Name:              newStructType,
		Description:       "",
		Fields:            of.Fields,
		ArrayNestingLevel: of.ArrayNestingLevel,
		isNested:          true,
	}

	additionalGen += mainOneOf.Gen()

	tabs := getTabs(tabsCount)

	// write comment
	if of.Description != "" {
		gen += fmt.Sprintf("%s// %s\n", tabs, of.Description)
	}

	var preType, omitempty string
	if !of.IsRequired {
		omitempty = ",omitempty"
		preType = "*"
	}
	preType += getArrayBrackets(of.ArrayNestingLevel)

	genJSON := fmt.Sprintf("`json:%q`", of.Name+omitempty)

	// write main object
	gen += fmt.Sprintf("%s%s %s%s %s\n", tabs, genName, preType, newStructType, genJSON)

	return
}

func (of OneOf) nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string) {

	genName := getFullName(of.Name)

	newStructType := objName + "_" + genName

	mainOneOf := OneOf{
		Name:              newStructType,
		Description:       "",
		Fields:            of.Fields,
		ArrayNestingLevel: of.ArrayNestingLevel,
		isNested:          true,
	}

	additionalGen += mainOneOf.TestGen()

	ref := "&"

	if !of.IsRequired {
		testGen += fmt.Sprintf("\t%s.%s = new(%s%s)\n",
			refName, genName, getArrayBrackets(of.ArrayNestingLevel), newStructType)
		ref = ""
	}

	if of.ArrayNestingLevel == 0 {
		testGen += fmt.Sprintf("\t%s_%s(%s%s.%s)\n", testMethodName, newStructType, ref, refName, genName)
		return
	}

	equal := "="
	if *firstArray {
		equal = ":="
		*firstArray = false
	}

	testGen += fmt.Sprintf("\tl0 %s randIntn(maxArrayLength + 1)\n", equal)

	testGen += fmt.Sprintf("\t%s.%s = make(%s%s, l0)\n", refName, genName, getArrayBrackets(of.ArrayNestingLevel), newStructType)

	var tabs string
	var endBrackets string

	for i := 0; i < of.ArrayNestingLevel; i++ {
		tabs = getTabs(i + 1)
		testGen += fmt.Sprintf("%sfor i%d := 0; i%d < l%d; i%d++ {\n",
			tabs, i, i, i, i)

		if i+1 < of.ArrayNestingLevel {
			testGen += fmt.Sprintf("\t%sl%d := randIntn(maxArrayLength + 1)\n", tabs, i+1)

			testGen += fmt.Sprintf("\t%s(*o)[i%d] = make(%s%s, l%d)\n",
				tabs, i, getArrayBrackets(of.ArrayNestingLevel-i-1), newStructType, i+1)
		}

		endBrackets = tabs + "}\n" + endBrackets
	}

	tabs += "\t"

	testGen += fmt.Sprintf("%s%s.%s", tabs, refName, genName)
	for i := 0; i < of.ArrayNestingLevel; i++ {
		testGen += fmt.Sprintf("[i%d]", i)
	}

	testGen += fmt.Sprintf(".%s_%s(&%s.%s)\n", testMethodName, newStructType, refName, genName)

	testGen += endBrackets
	testGen += "}\n\n"

	return
}

type SimpleType struct {
	Name              string
	Description       string
	Type              string
	Limits            Limits
	ArrayNestingLevel int
	IsRequired        bool

	withoutJSON bool
	isNesting   bool
}

func (t SimpleType) GetName() string {
	return t.Name
}

func parseSimpleType(name string, prop Property, arrayNestingLvl int) (t SimpleType) {
	t.Name = name
	t.ArrayNestingLevel = arrayNestingLvl
	t.Limits = prop.Limits

	if prop.Description != nil {
		t.Description = *prop.Description
	}

	if prop.Required != nil {
		t.IsRequired = *prop.Required
	}

	if prop.Ref != nil {
		t.Type = getRefName(*prop.Ref)
		return
	}

	if prop.Type != nil {
		if *prop.Type == "object" {
			panic(name)
		}

		if _, ok := (*prop.Type).([]interface{}); ok {
			t.Type = "string"
		} else {
			t.Type = getSimpleType((*prop.Type).(string))
		}
		return
	}

	panic(name)
}

func (t SimpleType) Gen() (gen string) {
	var genName string

	if t.isNesting {
		genName = t.Name
	} else {
		genName = getFullObjectName(t.Name)
	}

	// write comment
	if t.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", genName, t.Description)
	}
	gen += t.Limits.gen(1)

	if _, easySkipped := easyJSONBlackList[t.Name]; easySkipped {
		gen += "//easyjson:skip\n"
	}

	// write main object
	gen += fmt.Sprintf("type %s %s%s\n\n", genName, getArrayBrackets(t.ArrayNestingLevel), getFullObjectName(t.Type))

	return
}

func (t SimpleType) TestGen() (testGen string) {
	var genName string

	if t.isNesting {
		genName = t.Name
	} else {
		genName = getFullObjectName(t.Name)
	}

	testGen += fmt.Sprintf("func %s_%s(o *%s) {\n", testMethodName, genName, genName)

	if t.ArrayNestingLevel == 0 {
		if isGoType(t.Type) {
			testGen += fmt.Sprintf("\t*o = %s(%s)\n}\n\n", genName, getRandSetter(t.Type))
			return
		}

		testGen += fmt.Sprintf("\tr := %s(*o)\n", getFullObjectName(t.Type))
		testGen += fmt.Sprintf("\t%s_%s(&r)\n", testMethodName, getFullObjectName(t.Type))
		testGen += fmt.Sprintf("\t*o = %s(r)\n}\n\n", genName)
		return
	}

	for i := 0; i < t.ArrayNestingLevel; i++ {
		testGen += fmt.Sprintf("\tl%d := randIntn(maxArrayLength + 1)\n", i)
	}

	testGen += fmt.Sprintf("\t*o = make(%s%s, l0)\n", getArrayBrackets(t.ArrayNestingLevel), getFullObjectName(t.Type))

	var tabs string
	var endBrackets string

	for i := 0; i < t.ArrayNestingLevel; i++ {
		tabs = getTabs(i + 1)
		testGen += fmt.Sprintf("%sfor i%d := 0; i%d < l%d; i%d++ {\n",
			tabs, i, i, i, i)

		if i+1 < t.ArrayNestingLevel {
			testGen += fmt.Sprintf("%s(*o)[i%d] = make(%s%s, l%d)\n",
				tabs, i, getArrayBrackets(t.ArrayNestingLevel-i), getFullObjectName(t.Type), i+1)
		}

		endBrackets += tabs + "}\n"
	}

	tabs += "\t"

	brackets := ""
	for i := 0; i < t.ArrayNestingLevel; i++ {
		brackets += fmt.Sprintf("[i%d]", i)
	}

	if isGoType(t.Type) {
		testGen += fmt.Sprintf("%s(*o)%s = %s\n", tabs, brackets, getRandSetter(t.Type))
	} else {
		testGen += fmt.Sprintf("%s%s_%s(&(*o)%s)\n",
			tabs, testMethodName, getFullObjectName(t.Type), brackets)
	}

	testGen += endBrackets
	testGen += "}\n\n"
	return
}

func (t SimpleType) nestedGen(nestingLvl int, objName string) (gen, _ string) {
	tabs := getTabs(nestingLvl)
	genType := getFullObjectName(t.Type)
	genName := getFullName(t.Name)

	if t.Name == "" {
		if !t.IsRequired {
			// OneOf style
			if t.ArrayNestingLevel > 0 {
				genName = strings.ReplaceAll(getArrayBrackets(t.ArrayNestingLevel), "[]", "Array") + genType
			} else {
				genName = upFirstAny(genType)
			}
		}
		t.withoutJSON = true
	}

	var preType, omitempty string
	if !t.IsRequired {
		omitempty = ",omitempty"
		preType = "*"
	}
	preType += getArrayBrackets(t.ArrayNestingLevel)

	if genName == "2faRequired" {
		genName = "TwoFaRequired"
	}

	// write comment
	if t.Description != "" {
		gen += fmt.Sprintf("%s// %s\n", tabs, t.Description)
	}
	gen += t.Limits.gen(nestingLvl)

	// write nested object
	if genName != "" {
		gen += fmt.Sprintf("%s%s %s%s", tabs, genName, preType, genType)
	} else {
		gen += fmt.Sprintf("%s%s%s", tabs, preType, genType)
	}

	if !t.withoutJSON {
		gen += fmt.Sprintf(" `json:%q`", t.Name+omitempty)
	}

	gen += "\n"
	return
}

func (t SimpleType) nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string) {
	genName := getFullName(t.Name)
	genType := getFullObjectName(t.Type)

	if t.Name == "" {
		genName = upFirstAny(getFullObjectName(t.Type))
		if !t.IsRequired {
			// OneOf style
			if t.ArrayNestingLevel > 0 {
				genName = strings.ReplaceAll(getArrayBrackets(t.ArrayNestingLevel), "[]", "Array") + genType
			}
		}
	}

	if genName == "2faRequired" {
		genName = "TwoFaRequired"
	}

	pointer := ""
	if !t.IsRequired {
		pointer = "*"
	}

	ref := "&"
	if !t.IsRequired {
		testGen += fmt.Sprintf("\t%s.%s = new(%s%s)\n",
			refName, genName, getArrayBrackets(t.ArrayNestingLevel), genType)
		ref = ""
	}

	if t.ArrayNestingLevel == 0 {
		if isGoType(t.Type) {
			testGen += fmt.Sprintf("\t%s%s.%s = %s\n", pointer, refName, genName, getRandSetter(t.Type))
			return
		}

		// TODO: костыль
		if t.Type == "map[string]base_bool_int" {
			return
		}

		if getFullObjectName(t.Type) == objName {
			testGen += "\t//"
		} else {
			testGen += "\t"
		}

		testGen += fmt.Sprintf("%s_%s(%s%s.%s)\n", testMethodName, genType, ref, refName, genName)
		return
	}

	equal := "="

	if *firstArray {
		equal = ":="
		*firstArray = false
	}
	testGen += fmt.Sprintf("\tl0 %s randIntn(maxArrayLength + 1)\n", equal)

	testGen += fmt.Sprintf("\t%s%s.%s = make(%s%s, l0)\n", pointer, refName, genName, getArrayBrackets(t.ArrayNestingLevel), genType)

	var tabs string
	var endBrackets string

	for i := 0; i < t.ArrayNestingLevel; i++ {
		tabs = getTabs(i + 1)
		testGen += fmt.Sprintf("%sfor i%d := 0; i%d < l%d; i%d++ {\n",
			tabs, i, i, i, i)

		if i+1 < t.ArrayNestingLevel {
			testGen += fmt.Sprintf("\t%sl%d := randIntn(maxArrayLength + 1)\n", tabs, i+1)

			testGen += fmt.Sprintf("\t%s(%s%s.%s)[i%d] = make(%s%s, l%d)\n",
				tabs, pointer, refName, genName, i, getArrayBrackets(t.ArrayNestingLevel-i-1), genType, i+1)
		}

		endBrackets = tabs + "}\n" + endBrackets
	}

	tabs += "\t"

	brackets := ""
	for i := 0; i < t.ArrayNestingLevel; i++ {
		brackets += fmt.Sprintf("[i%d]", i)
	}

	if isGoType(t.Type) {
		name := fmt.Sprintf("*%s.%s", refName, genName)
		if t.IsRequired {
			name = fmt.Sprintf("%s.%s", refName, genName)
		}
		testGen += fmt.Sprintf("%s(%s)%s = %s\n", tabs, name, brackets, getRandSetter(t.Type))
	} else {
		if getFullObjectName(t.Type) == objName {
			testGen += tabs + "//"
		} else {
			testGen += tabs
		}
		testGen += fmt.Sprintf("%s_%s(&(%s%s.%s)%s)\n",
			testMethodName, genType, pointer, refName, genName, brackets)
	}

	testGen += endBrackets

	return
}

type Enum struct {
	Name        string
	Description string
	ValuesType  string
	EnumValues  []interface{}
	EnumNames   []string
	Limits      Limits

	IsRequired        bool
	ArrayNestingLevel int

	isNested    bool // global or nested
	withoutJSON bool
}

func (e Enum) GetName() string {
	return e.Name
}

func parseEnum(name string, prop Property, arrayNestingLvl int) (e Enum) {
	e.Name = name
	e.ArrayNestingLevel = arrayNestingLvl
	e.Limits = prop.Limits

	if prop.Description != nil {
		e.Description = *prop.Description
	}

	if prop.Required != nil {
		e.IsRequired = *prop.Required
	}

	if prop.Enum == nil {
		panic(name)
	}

	e.EnumValues = *prop.Enum

	if prop.EnumNames != nil {
		e.EnumNames = *prop.EnumNames
	}

	// TODO: костыль
	if vType := fmt.Sprintf("%T", e.EnumValues[0]); vType == "float64" {
		e.ValuesType = "int"
	} else {
		e.ValuesType = vType
	}

	return
}

func (e Enum) Gen() (gen string) {
	var genName string

	if e.isNested {
		genName = e.Name
	} else {
		genName = getFullObjectName(e.Name)
	}

	// write comment
	if e.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", genName, e.Description)
	}

	if _, easySkipped := easyJSONBlackList[e.Name]; easySkipped {
		gen += "//easyjson:skip\n"
	}

	// write main object
	gen += fmt.Sprintf("type %s %s\n\n", genName, e.ValuesType)

	consts := make([]string, 0, len(e.EnumValues))

	for i, v := range e.EnumValues {
		var suffix string
		if len(e.EnumNames) == 0 {
			suffix = fmt.Sprintf("%v", v)
		} else {
			suffix = editEnumSpace(e.EnumNames[i])
		}

		var genValue string
		if e.ValuesType == "string" {
			genValue = fmt.Sprintf("%q", v)
		} else {
			genValue = fmt.Sprintf("%v", v)
		}

		c := fmt.Sprintf("\t%s_%s %s = %s", genName, getFullName(suffix), genName, genValue)

		consts = append(consts, c)
	}

	genConsts := strings.Join(consts, "\n")

	gen += fmt.Sprintf("const (\n%s\n)\n\n", genConsts)

	return
}

func (e Enum) TestGen() (testGen string) {
	var genName string

	if e.isNested {
		genName = e.Name
	} else {
		genName = getFullObjectName(e.Name)
	}

	testGen += fmt.Sprintf("func %s_%s(o *%s) {\n\tswitch randIntn(%d) {\n", testMethodName, genName, genName, len(e.EnumValues))

	for i, v := range e.EnumValues {
		var genV string
		if e.ValuesType == "string" {
			genV = fmt.Sprintf("%q", v)
		} else {
			genV = fmt.Sprintf("%v", v)
		}
		testGen += fmt.Sprintf("\tcase %d:\n\t\t*o = %s\n", i, genV)
	}

	testGen += "\t}\n}\n\n"

	return
}

func (e Enum) nestedGen(nestingLvl int, objName string) (nestedGen, additionalGen string) {
	genName := getFullName(e.Name)

	newStructType := objName + "_" + genName

	mainEnum := Enum{
		Name:        newStructType,
		Description: "",
		ValuesType:  e.ValuesType,
		EnumValues:  e.EnumValues,
		EnumNames:   e.EnumNames,
		isNested:    true,
	}

	additionalGen = mainEnum.Gen()

	// generate nested enum field
	tabs := getTabs(nestingLvl)

	// write comment
	if e.Description != "" {
		nestedGen += fmt.Sprintf("%s// %s\n", tabs, e.Description)
	}
	nestedGen += e.Limits.gen(nestingLvl)

	var preType, omitempty string
	if !e.IsRequired {
		omitempty = ",omitempty"
		preType = "*"
	}
	preType += getArrayBrackets(e.ArrayNestingLevel)

	// write nested enum
	nestedGen += fmt.Sprintf("%s%s %s%s", tabs, genName, preType, newStructType)

	if !e.withoutJSON {
		nestedGen += fmt.Sprintf("`json:%q`", e.Name+omitempty)
	}

	nestedGen += "\n"

	return
}

func (e Enum) nestedTestGen(objName, refName string, firstArray *bool) (testGen, additionalGen string) {
	genName := getFullName(e.Name)

	newStructType := objName + "_" + genName

	mainEnum := Enum{
		Name:        newStructType,
		Description: "",
		ValuesType:  e.ValuesType,
		EnumValues:  e.EnumValues,
		EnumNames:   e.EnumNames,
		isNested:    true,
	}

	additionalGen += mainEnum.TestGen()

	ref := "&"
	if !e.IsRequired {
		testGen += fmt.Sprintf("\t%s.%s = new(%s%s)\n",
			refName, genName, getArrayBrackets(e.ArrayNestingLevel), newStructType)
		ref = ""
	}

	if e.ArrayNestingLevel == 0 {
		testGen += fmt.Sprintf("\t%s_%s(%s%s.%s)\n", testMethodName, newStructType, ref, refName, genName)
		return
	}

	pointer := ""
	if !e.IsRequired {
		pointer = "*"
	}

	equal := "="
	if *firstArray {
		equal = ":="
		*firstArray = false
	}

	testGen += fmt.Sprintf("\tl0 %s randIntn(maxArrayLength + 1)\n", equal)

	testGen += fmt.Sprintf("\t%s%s.%s = make(%s%s, l0)\n", pointer, refName, genName, getArrayBrackets(e.ArrayNestingLevel), newStructType)

	var tabs string
	var endBrackets string

	for i := 0; i < e.ArrayNestingLevel; i++ {
		tabs = getTabs(i + 1)
		testGen += fmt.Sprintf("%sfor i%d := 0; i%d < l%d; i%d++ {\n",
			tabs, i, i, i, i)

		if i+1 < e.ArrayNestingLevel {
			testGen += fmt.Sprintf("\t%sl%d = randIntn(maxArrayLength + 1)\n", tabs, i+1)

			testGen += fmt.Sprintf("\t%s(*o)[i%d] = make(%s%s, l%d)\n",
				tabs, i, getArrayBrackets(e.ArrayNestingLevel-i-1), newStructType, i+1)
		}

		endBrackets = tabs + "}\n" + endBrackets
	}

	tabs += "\t"

	brackets := ""
	for i := 0; i < e.ArrayNestingLevel; i++ {
		brackets += fmt.Sprintf("[i%d]", i)
	}

	testGen += fmt.Sprintf("%s%s_%s(&(%s%s.%s)%s)\n",
		tabs, testMethodName, newStructType, pointer, refName, genName, brackets)

	testGen += endBrackets

	return
}

func parsePatternProperties(name string, prop Property, arrayNestingLvl int) (t SimpleType) {
	t.Name = name
	t.ArrayNestingLevel = arrayNestingLvl
	t.Limits = prop.Limits

	if prop.Description != nil {
		t.Description = *prop.Description
	}

	if prop.Required != nil {
		t.IsRequired = *prop.Required
	}

	if prop.PatternProperties == nil {
		panic(name)
	}

	ref := (*prop.PatternProperties)["^[0-9]+$"]["$ref"]

	t.Type = "map[string]" + getRefName(ref)

	return
}

func getRandSetter(t string) (s string) {
	switch t {
	case "string":
		s += "randString()"
	case "int":
		s += "randInt()"
	case "float64":
		s += "randFloat()"
	case "bool":
		s += "randBool()"
	}

	return
}
