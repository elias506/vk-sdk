package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Parameter struct {
	Property
	Name string `json:"name"`
}

type MethodJSON struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	AccessTokenType []string            `json:"access_token_type"`
	Parameters      []Parameter         `json:"parameters"`
	Responses       map[string]Property `json:"responses"`
	Errors          []Property          `json:"errors"`
}

type MethodsFile struct {
	Methods []MethodJSON `json:"methods"`
}

func GenerateMethods(w, wTest io.Writer, methodsRaw []byte) {
	var file MethodsFile

	if err := json.Unmarshal(methodsRaw, &file); err != nil {
		panic(err.Error())
	}

	genners := make([]GennerWithTest, 0, len(file.Methods))

	for _, mJSON := range file.Methods {
		genners = append(genners, parseMethodGenner(mJSON)...)
	}

	writeStartFile(w, "vk_sdk", "", "context", "net/url")
	writeStartFile(wTest, "vk_sdk", "",
		"context", "encoding/json", "errors", "github.com/stretchr/testify/assert", "github.com/stretchr/testify/require", "net/url", "testing")

	for _, g := range genners {
		fmt.Fprint(w, g.Gen())
		fmt.Fprint(wTest, g.TestGen())
	}
}

type ParamNameNestedGenner interface {
	NameNestedGenner
	Param() Param
}

type Param struct {
	Name              string
	Type              string
	HasCustomType     bool
	IsRequired        bool
	ArrayNestingLevel int
	Limits            Limits
}

func (t SimpleType) Param() (p Param) {
	p.Name = t.Name
	p.ArrayNestingLevel = t.ArrayNestingLevel
	p.IsRequired = t.IsRequired
	p.Type = t.Type
	p.Limits = t.Limits

	return
}

func (e Enum) Param() (p Param) {
	p.Name = e.Name
	p.Type = e.ValuesType
	p.HasCustomType = true
	p.ArrayNestingLevel = e.ArrayNestingLevel
	p.IsRequired = e.IsRequired
	p.Limits = e.Limits

	return
}

func parseParamNestedGenner(param Parameter, arrayNestingLvl int) ParamNameNestedGenner {
	if param.Ref != nil {
		t := parseSimpleType(param.Name, param.Property, arrayNestingLvl)
		t.withoutJSON = true
		return t
	}

	if param.Items != nil {
		// TODO: научиться прокидывать
		param.Property.Items.Limits.Add(param.Property.Limits)
		param.Property = *param.Property.Items
		return parseParamNestedGenner(param, arrayNestingLvl+1)
	}

	if param.Enum != nil {
		e := parseEnum(param.Name, param.Property, arrayNestingLvl)
		e.withoutJSON = true
		return e
	}

	t := parseSimpleType(param.Name, param.Property, arrayNestingLvl)
	t.withoutJSON = true
	return t
}

type Method struct {
	Name         string
	VKDevName    string
	Description  string
	AccessTokens []string
	ErrorRefs    []string

	GenRequest  bool
	RequestName string
	FullName    string

	Params      []ParamNameNestedGenner
	ResponseRef *string
	SetFields   []SetField
}

type SetField struct {
	Name  string
	Value string
}

func parseMethodGenner(mJSON MethodJSON) []GennerWithTest {
	var m Method

	m.Name = mJSON.Name
	m.VKDevName = mJSON.Name
	m.Description = mJSON.Description
	m.AccessTokens = mJSON.AccessTokenType

	// parse params
	m.Params = make([]ParamNameNestedGenner, 0, len(mJSON.Parameters))
	for _, parameter := range mJSON.Parameters {
		m.Params = append(m.Params, parseParamNestedGenner(parameter, 0))
	}

	// parse error references
	m.ErrorRefs = make([]string, 0, len(mJSON.Errors))
	for _, e := range mJSON.Errors {
		m.ErrorRefs = append(m.ErrorRefs, getRefName(*e.Ref))
	}

	// parse responses
	additionalMethods := m.parseResponses(mJSON.Responses)

	return append([]GennerWithTest{m}, additionalMethods...)
}

func (m *Method) parseResponses(responses map[string]Property) []GennerWithTest {
	m.FullName = getFullMethodName(m.Name)
	m.RequestName = m.FullName + "_Request"
	m.GenRequest = true

	if response, ok := responses["response"]; ok {
		ref := getRefName(*response.Ref)
		m.ResponseRef = &ref
	}

	if _, ok := responses["multiResponse"]; ok {
		return nil
	}

	if responseIntegerProp, ok := responses["responseInteger"]; ok {
		responseInteger := getRefName(*responseIntegerProp.Ref)
		responseArray := getRefName(*responses["responseArray"].Ref)

		countersMethod := Method{
			Name:         m.Name,
			VKDevName:    m.VKDevName,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   true,
			RequestName:  m.FullName + "Counters_Request",
			FullName:     m.FullName + "Counters",
			Params:       m.Params,
			ResponseRef:  &responseArray,
			SetFields:    nil,
		}

		m.ResponseRef = &responseInteger
		m.removeParam("counters")

		countersMethod.removeParam("user_id")
		countersMethod.removeParam("counter")
		countersMethod.removeParam("increment")

		notSecureMethod := Method{
			Name:         "setCounter",
			VKDevName:    m.VKDevName,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   true,
			RequestName:  m.FullName + "NotSecure_Request",
			FullName:     m.FullName + "NotSecure",
			Params:       m.Params,
			ResponseRef:  &responseInteger,
			SetFields:    nil,
		}

		notSecureMethod.removeParam("user_id")

		return []GennerWithTest{countersMethod, notSecureMethod}
	}

	if userIdsResponseProp, ok := responses["userIdsResponse"]; ok {
		userIdsResponse := getRefName(*userIdsResponseProp.Ref)

		if userIdsExtendedResponseProp, ok := responses["userIds_Extended_Response"]; ok {

			userIdsMethod := Method{
				Name:         m.Name,
				VKDevName:    m.VKDevName,
				Description:  m.Description,
				AccessTokens: m.AccessTokens,
				ErrorRefs:    m.ErrorRefs,
				GenRequest:   true,
				RequestName:  m.FullName + "UserIDs_Request",
				FullName:     m.FullName + "UserIDs",
				Params:       m.Params,
				ResponseRef:  &userIdsResponse,
				SetFields: []SetField{
					{
						Name:  "extended",
						Value: "0",
					},
				},
			}
			userIdsMethod.removeParam("extended")
			userIdsMethod.removeParam("user_id")

			userIdsExtendedResponse := getRefName(*userIdsExtendedResponseProp.Ref)
			userIdsExtendedMethod := Method{
				Name:         m.Name,
				VKDevName:    m.VKDevName,
				Description:  m.Description,
				AccessTokens: m.AccessTokens,
				ErrorRefs:    m.ErrorRefs,
				GenRequest:   false,
				RequestName:  userIdsMethod.RequestName,
				FullName:     m.FullName + "ExtendedUserIDs",
				Params:       userIdsMethod.Params,
				ResponseRef:  &userIdsExtendedResponse,
				SetFields: []SetField{
					{
						Name:  "extended",
						Value: "1",
					},
				},
			}

			m.removeParam("extended")
			m.removeParam("user_ids")
			m.SetFields = []SetField{
				{
					Name:  "extended",
					Value: "0",
				},
			}

			extendedResponse := getRefName(*responses["extendedResponse"].Ref)
			extendedMethod := Method{
				Name:         m.Name,
				VKDevName:    m.VKDevName,
				Description:  m.Description,
				AccessTokens: m.AccessTokens,
				ErrorRefs:    m.ErrorRefs,
				GenRequest:   false,
				RequestName:  m.RequestName,
				FullName:     m.FullName + "Extended",
				Params:       m.Params,
				ResponseRef:  &extendedResponse,
				SetFields: []SetField{
					{
						Name:  "extended",
						Value: "1",
					},
				},
			}
			return []GennerWithTest{extendedMethod, userIdsMethod, userIdsExtendedMethod}
		}

		userIdsMethod := Method{
			Name:         m.Name,
			VKDevName:    m.VKDevName,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   true,
			RequestName:  m.FullName + "UserIDs_Request",
			FullName:     m.FullName + "UserIDs",
			Params:       m.Params,
			ResponseRef:  &userIdsResponse,
			SetFields:    nil,
		}
		userIdsMethod.removeParam("peer_id")

		m.removeParam("peer_ids")

		return []GennerWithTest{userIdsMethod}
	}

	if targetUidsResponseProp, ok := responses["targetUidsResponse"]; ok {
		targetUidsResponse := getRefName(*targetUidsResponseProp.Ref)

		targetUidsMethod := Method{
			Name:         m.Name,
			VKDevName:    m.VKDevName,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   true,
			RequestName:  m.FullName + "TargetUIDs_Request",
			FullName:     m.FullName + "TargetUIDs",
			Params:       m.Params,
			ResponseRef:  &targetUidsResponse,
			SetFields:    nil,
		}
		targetUidsMethod.removeParam("target_uid")

		m.removeParam("target_uids")

		return []GennerWithTest{targetUidsMethod}
	}

	if onlineMobileResponseProp, ok := responses["onlineMobileResponse"]; ok {
		m.removeParam("online_mobile")
		m.SetFields = []SetField{
			{
				Name:  "online_mobile",
				Value: "0",
			},
		}

		onlineMobileResponse := getRefName(*onlineMobileResponseProp.Ref)

		onlineMobileMethod := Method{
			Name:         m.Name,
			VKDevName:    m.VKDevName,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   false,
			RequestName:  m.RequestName,
			FullName:     m.FullName + "OnlineMobile",
			Params:       m.Params,
			ResponseRef:  &onlineMobileResponse,
			SetFields: []SetField{
				{
					Name:  "online_mobile",
					Value: "1",
				},
			},
		}

		return []GennerWithTest{onlineMobileMethod}
	}

	if extendedResponse, ok := responses["extendedResponse"]; ok {
		m.removeParam("extended")
		m.SetFields = []SetField{
			{
				Name:  "extended",
				Value: "0",
			},
		}
		extendedRef := getRefName(*extendedResponse.Ref)
		extendedM := Method{
			Name:         m.Name,
			VKDevName:    m.Name,
			Description:  m.Description,
			AccessTokens: m.AccessTokens,
			ErrorRefs:    m.ErrorRefs,
			GenRequest:   false,
			RequestName:  m.FullName + "_Request",
			FullName:     m.FullName + "Extended",
			Params:       m.Params,
			ResponseRef:  &extendedRef,
			SetFields: []SetField{
				{
					Name:  "extended",
					Value: "1",
				},
			},
		}
		return []GennerWithTest{extendedM}
	}

	return nil
}

func (m *Method) removeParam(name string) {
	newParams := make([]ParamNameNestedGenner, 0, len(m.Params)-1)
	for _, p := range m.Params {
		if p.GetName() != name {
			newParams = append(newParams, p)
		}
	}
	m.Params = newParams
}

const docLink = "https://dev.vk.com/method/"

func (m Method) Gen() (gen string) {
	if m.GenRequest && len(m.Params) > 0 {
		gen += m.genRequest()
	}

	if m.Description != "" {
		gen += fmt.Sprintf("// %s %s\n", m.FullName, m.Description)
	} else {
		gen += fmt.Sprintf("// %s ...\n", m.FullName)
	}

	var genResp string
	if m.ResponseRef != nil {
		genResp = fmt.Sprintf("(resp %s, apiErr ApiError, err error)", getFullObjectName(*m.ResponseRef))
	} else {
		genResp = "(apiErr ApiError, err error)"
	}

	if m.AccessTokens != nil {
		gen += fmt.Sprintf("// May execute with listed access token types:\n//    [ %s ]\n", strings.Join(m.AccessTokens, ", "))
	}

	if len(m.ErrorRefs) != 0 {
		gen += fmt.Sprintf("// When executing method, may return one of global or with listed codes API errors:\n//    [ %s ]\n", buildPossibleErrors(m.ErrorRefs))
	} else {
		gen += fmt.Sprintf("// When executing method, may return one of global API errors.\n")
	}

	gen += fmt.Sprintf("//\n// %s%s\n", docLink, m.VKDevName)

	genBody := m.genBody()

	if len(m.Params) > 0 {
		gen += fmt.Sprintf("func (vk *VK) %s(ctx context.Context, req %s, options ...Option) %s {\n%s}\n\n", m.FullName, m.RequestName, genResp, genBody)
		return
	}

	gen += fmt.Sprintf("func (vk *VK) %s(ctx context.Context, options ...Option) %s {\n%s}\n\n", m.FullName, genResp, genBody)

	return
}

func (m Method) TestGen() (testGen string) {
	if m.GenRequest && len(m.Params) > 0 {
		testGen += m.testGenRequest()
	}

	testGen += m.successTestGen()
	testGen += m.apiErrorTestGen()
	testGen += m.errorTestGen()

	return
}

func (m Method) successTestGen() (testGen string) {
	testGen += fmt.Sprintf("func TestVK_%s_Success(t *testing.T) {\n", m.FullName)

	testGen += fmt.Sprintf("\t%s := make(url.Values, %d)\n",
		urlValuesName, len(m.Params)+len(m.SetFields)+2)

	if len(m.Params) > 0 {
		testGen += fmt.Sprintf("\tvar req %s\n", m.RequestName)
		testGen += fmt.Sprintf("\t%s_%s(&req)\n", testMethodName, m.RequestName)
		testGen += fmt.Sprintf("\trequire.NoError(t, req.%s(%s))\n",
			fillInValuesMethodName, urlValuesName)
	}

	for _, st := range m.SetFields {
		testGen += fmt.Sprintf("\tsetString(%s, %q, %q)\n", urlValuesName, st.Name, st.Value)
	}

	if m.ResponseRef != nil {
		resp := getFullObjectName(*m.ResponseRef)
		testGen += fmt.Sprintf("\tvar expected %s\n", resp)
		testGen += fmt.Sprintf("\t%s_%s(&expected)\n", testMethodName, resp)
		testGen += "\texpectedJSON, err := json.Marshal(expected)\n"
		testGen += "\trequire.NoError(t, err)\n"
	} else {
		testGen += fmt.Sprintf("\texpectedJSON := []byte(%q)", "{}")
	}

	testGen += "\ttoken := randString()\n"
	testGen += fmt.Sprintf("\tvk := NewVK(NewTestClient(t, token, %q, values, expectedJSON), token)\n", m.Name)

	if len(m.Params) > 0 {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background(), req)\n", m.FullName)
	} else {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background())\n", m.FullName)
	}

	testGen += "\tassert.EqualValues(t, expected, resp)\n"
	testGen += "\tassert.Nil(t, apiErr)\n"
	testGen += "\tassert.NoError(t, err)\n"

	testGen += "}\n\n"

	return
}

func (m Method) apiErrorTestGen() (testGen string) {
	testGen += fmt.Sprintf("func TestVK_%s_ApiError(t *testing.T) {\n", m.FullName)

	testGen += fmt.Sprintf("\tvar expected apiError\n")
	testGen += "\texpected.fillRandomly()\n"
	testGen += "\texpectedJSON, err := json.Marshal(expected)\n"
	testGen += "\trequire.NoError(t, err)\n"

	testGen += fmt.Sprintf("\tvk := NewVK(NewApiErrorTestClient(t, %q, expectedJSON), \"\")\n", m.Name)

	if len(m.Params) > 0 {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background(), %s{})\n", m.FullName, m.RequestName)
	} else {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background())\n", m.FullName)
	}

	testGen += "\tassert.Empty(t, resp)\n"
	testGen += "\tassert.Equal(t, &expected, apiErr)\n"
	testGen += "\tassert.NoError(t, err)\n"

	testGen += "}\n\n"

	return
}

func (m Method) errorTestGen() (testGen string) {
	testGen += fmt.Sprintf("func TestVK_%s_Error(t *testing.T) {\n", m.FullName)

	testGen += "\texpected := errors.New(randString())\n"
	testGen += "\tvk := NewVK(NewErrorTestClient(expected))\n"

	if len(m.Params) > 0 {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background(), %s{})\n", m.FullName, m.RequestName)
	} else {
		testGen += fmt.Sprintf("\tresp, apiErr, err := vk.%s(context.Background())\n", m.FullName)
	}

	testGen += "\tassert.Empty(t, resp)\n"
	testGen += "\tassert.Nil(t, apiErr)\n"
	testGen += "\tassert.ErrorIs(t, err, expected)\n"

	testGen += "}\n\n"

	return
}

const (
	urlValuesName          = "values"
	fillInValuesMethodName = "fillIn"
)

func (m Method) genBody() (body string) {
	body += fmt.Sprintf("\t%s := make(url.Values, %d+len(options))\n", urlValuesName, len(m.Params)+len(m.SetFields)+2)

	if len(m.Params) > 0 {
		body += fmt.Sprintf("\tif err = req.%s(%s); err != nil {\n\t\treturn\n\t}\n", fillInValuesMethodName, urlValuesName)
	}

	for _, f := range m.SetFields {
		body += fmt.Sprintf("\tsetString(%s, %q, %q)\n", urlValuesName, f.Name, f.Value)
	}

	body += fmt.Sprintf("\tsetOptions(%s, options)\n", urlValuesName)

	dstName := "nil"
	if m.ResponseRef != nil {
		dstName = "&resp"
	}

	body += fmt.Sprintf("\tapiErr, err = vk.doReq(%q, ctx, %s, %s)\n\treturn\n", m.Name, urlValuesName, dstName)

	return
}

func (m Method) genRequest() (gen string) {
	var fieldsGen string

	for _, pGenner := range m.Params {
		fGen, addGen := pGenner.nestedGen(1, m.FullName)
		gen += addGen
		fieldsGen += fGen
	}

	gen += fmt.Sprintf("type %s struct {\n%s}\n\n", m.RequestName, fieldsGen)

	// gen fillIn method
	gen += fmt.Sprintf("func (r %s) %s(%s url.Values) (err error) {\n", m.RequestName, fillInValuesMethodName, urlValuesName)

	for _, pGenner := range m.Params {
		gen += pGenner.Param().genInBody()
	}

	gen += "\treturn\n}\n\n"

	return
}

func (m Method) testGenRequest() (testGen string) {
	var fieldsGen string

	firstArray := true

	for _, pGenner := range m.Params {
		fGen, addGen := pGenner.nestedTestGen(m.FullName, "(*r)", &firstArray)
		testGen += addGen
		fieldsGen += fGen
	}

	testGen += fmt.Sprintf("func %s_%s(r *%s) {\n%s}\n\n", testMethodName, m.RequestName, m.RequestName, fieldsGen)

	return
}

func (p *Param) findReferenceType() {
	if !isGoType(p.Type) {
		p.HasCustomType = true
	}

	for !isGoType(p.Type) {
		refType := SimpleReferences[p.Type]
		if refType.ArrayNestingLevel > 1 {
			p.ArrayNestingLevel = refType.ArrayNestingLevel
		}
		p.Type = refType.Type
	}
}

func (p Param) genInBody() (gen string) {
	if p.IsRequired {
		if p.Limits.Format != nil && *p.Limits.Format == "json" {
			return getJSONSetter(1, "", p.Name)
		}

		p.findReferenceType()

		if p.ArrayNestingLevel == 0 {
			return getSetter(1, "", p.Name, p.Type, p.HasCustomType)
		}

		return getArraySetter(1, "", p.Name, p.Type, p.HasCustomType)
	}

	if p.Limits.Format != nil && *p.Limits.Format == "json" {
		return fmt.Sprintf("\tif r.%s != nil {\n%s\t}\n",
			getFullName(p.Name), getJSONSetter(2, "*", p.Name))
	}

	p.findReferenceType()

	if p.ArrayNestingLevel == 0 {
		return fmt.Sprintf("\tif r.%s != nil {\n%s\t}\n",
			getFullName(p.Name), getSetter(2, "*", p.Name, p.Type, p.HasCustomType))
	}

	return fmt.Sprintf("\tif r.%s != nil {\n%s\t}\n",
		getFullName(p.Name), getArraySetter(2, "*", p.Name, p.Type, p.HasCustomType))
}

func getSetter(nestingLvl int, pointer, name, t string, hasCustomType bool) string {
	tabs := getTabs(nestingLvl)

	if hasCustomType {
		switch t {
		case "string":
			return fmt.Sprintf("%ssetString(%s, %q, string(%sr.%s))\n", tabs, urlValuesName, name, pointer, getFullName(name))
		case "int":
			return fmt.Sprintf("%ssetInt(%s, %q, int(%sr.%s))\n", tabs, urlValuesName, name, pointer, getFullName(name))
		case "float64":
			return fmt.Sprintf("%ssetFloat(%s, %q, float64(%sr.%s))\n", tabs, urlValuesName, name, pointer, getFullName(name))
		case "bool":
			return fmt.Sprintf("%ssetBool(%s, %q, bool(%sr.%s))\n", tabs, urlValuesName, name, pointer, getFullName(name))
		}
	}

	switch t {
	case "string":
		return fmt.Sprintf("%ssetString(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "int":
		return fmt.Sprintf("%ssetInt(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "float64":
		return fmt.Sprintf("%ssetFloat(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "bool":
		return fmt.Sprintf("%ssetBool(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	}

	panic(t)
}

func getArraySetter(nestingLvl int, pointer, name, t string, hasCustomType bool) string {
	tabs := getTabs(nestingLvl)

	if hasCustomType {
		var setter string
		switch t {
		case "string":
			setter = fmt.Sprintf("%ssetStrings(%s, %q, vs)\n", tabs, urlValuesName, name)
		case "int":
			setter = fmt.Sprintf("%ssetInts(%s, %q, vs)\n", tabs, urlValuesName, name)
		case "float64":
			setter = fmt.Sprintf("%ssetFloats(%s, %q, vs)\n", tabs, urlValuesName, name)
		case "bool":
			setter = fmt.Sprintf("%ssetBools(%s, %q, vs)\n", tabs, urlValuesName, name)
		}

		return fmt.Sprintf("%svs := make([]%s, len(%sr.%s))\n%sfor i, v := range %sr.%s {\n\t%svs[i] = %s(v)\n%s}\n%s",
			tabs, t, pointer, getFullName(name), tabs, pointer, getFullName(name), tabs, t, tabs, setter)
	}

	switch t {
	case "string":
		return fmt.Sprintf("%ssetStrings(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "int":
		return fmt.Sprintf("%ssetInts(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "float64":
		return fmt.Sprintf("%ssetFloats(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	case "bool":
		return fmt.Sprintf("%ssetBools(%s, %q, %sr.%s)\n", tabs, urlValuesName, name, pointer, getFullName(name))
	}

	panic(t)
}

func getJSONSetter(nestingLvl int, pointer, name string) string {
	tabs := getTabs(nestingLvl)

	return fmt.Sprintf("%sif err = setJSON(%s, %q, %sr.%s); err != nil {\n\t%sreturn\n%s}\n",
		tabs, urlValuesName, name, pointer, getFullName(name), tabs, tabs)
}
